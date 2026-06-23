// Package gameboy is a minimal Game Boy (DMG) machine model that drives the sm83
// CPU core as an emulation oracle, mirroring tools/gamegear for the Game Gear. It
// implements the cartridge mapper (MBC1), the memory map (VRAM/WRAM/OAM/HRAM/IO),
// and enough of the hardware — the timer and the LCD scanline counter, with their
// interrupts — to run a real ROM through its boot and per-frame loop. After running
// some frames the caller reads back VRAM/OAM to see exactly what the game drew, the
// same idea as the Game Gear oracle.
//
// It is not cycle-perfect: the PPU is modelled only as an advancing LY/​mode counter
// that raises the VBlank/STAT/timer interrupts on schedule (no pixel pipeline), which
// is what a static-analysis oracle needs — let the game's own code run and observe
// the memory it produces.
package gameboy

import "stupidcoder.com/tools/sm83"

// I/O register addresses we model specially (the rest are plain bytes in io[]).
const (
	regJOYP = 0xFF00
	regDIV  = 0xFF04
	regTIMA = 0xFF05
	regTMA  = 0xFF06
	regTAC  = 0xFF07
	regIF   = 0xFF0F
	regLCDC = 0xFF40
	regSTAT = 0xFF41
	regLY   = 0xFF44
	regLYC  = 0xFF45
	regDMA  = 0xFF46
	regIE   = 0xFFFF
)

// Button bits for Machine.Buttons (1 = pressed). The Game Boy joypad is active-low
// in hardware; this model inverts it for you.
const (
	BtnRight = 1 << iota
	BtnLeft
	BtnUp
	BtnDown
	BtnA
	BtnB
	BtnSelect
	BtnStart
)

const (
	dotsPerLine    = 456
	linesPerFrame  = 154
	cyclesPerFrame = dotsPerLine * linesPerFrame // 70224
)

// Machine is a DMG: CPU + cartridge (MBC1) + RAM + a timer and LCD line counter,
// implementing sm83.Bus.
type Machine struct {
	CPU *sm83.CPU

	rom    []byte
	nbanks int

	// MBC1 state.
	romBank   int // bank mapped into $4000-$7FFF (never 0)
	ramBank   int // upper bank bits / RAM bank
	ramEnable bool
	mode      byte // 0 = ROM banking, 1 = RAM banking

	vram   [0x2000]byte // $8000-$9FFF
	wram   [0x2000]byte // $C000-$DFFF (+ echo $E000-$FDFF)
	oam    [0xA0]byte   // $FE00-$FE9F
	hram   [0x7F]byte   // $FF80-$FFFE
	io     [0x80]byte   // $FF00-$FF7F
	extram [0x8000]byte // cartridge RAM (none on SML, but supported)
	ie     byte         // $FFFF

	// timer / LCD timing accumulators (T-cycles).
	divCounter  int // 16-bit; DIV is its high byte
	timaCounter int
	lcdDot      int

	// Buttons is the injected joypad state (see Btn* bits; 1 = pressed).
	Buttons byte

	// Debug hooks (opt-in), in the spirit of the Game Gear oracle.
	Sample bool           // RunFrame tallies the PC after each instruction into PCHist
	PCHist map[uint16]int // PC histogram (a cheap profiler)

	WatchLo, WatchHi uint16         // VRAM write watch over [lo,hi): which PC wrote it
	WatchPCs         map[uint16]int // histogram of writes per storing PC

	// OnWrite, when set, is called on every memory write with the CPU PC (post-fetch),
	// the address and the value — a general probe for "which code wrote this?".
	OnWrite func(pc, addr uint16, v byte)
	// OnROMRead, when set, is called on every read from the $4000-$7FFF bank window
	// with the PC, the bank, and the address — to locate banked data being consumed.
	OnROMRead func(pc uint16, bank int, addr uint16)
}

// NewMachine builds a machine from a cartridge image and resets it to the DMG
// post-boot state (the CPU enters at $0100; LCD on, banks 0/1).
func NewMachine(rom []byte) *Machine {
	m := &Machine{rom: rom, nbanks: len(rom) / 0x4000, romBank: 1}
	if m.nbanks == 0 {
		m.nbanks = 1
	}
	m.io[regLCDC-0xFF00] = 0x91 // LCD on, BG on (post-boot)
	m.io[regJOYP-0xFF00] = 0x3F // no buttons selected
	m.CPU = sm83.NewCPU(m)
	return m
}

// Watch arms the VRAM-write watchpoint over [lo,hi) and clears prior hits.
func (m *Machine) Watch(lo, hi uint16) {
	m.WatchLo, m.WatchHi = lo, hi
	m.WatchPCs = map[uint16]int{}
}

// VRAM / OAM / WRAM expose the captured memory for read-back after running.
func (m *Machine) VRAM() []byte { return m.vram[:] }
func (m *Machine) OAM() []byte  { return m.oam[:] }
func (m *Machine) WRAM() []byte { return m.wram[:] }

// ROMBank returns the bank currently paged into $4000-$7FFF.
func (m *Machine) ROMBank() int { return m.romBank }

// --- sm83.Bus --------------------------------------------------------------

func (m *Machine) Read(a uint16) byte {
	switch {
	case a < 0x4000:
		return m.rom[int(a)]
	case a < 0x8000:
		if m.OnROMRead != nil {
			m.OnROMRead(m.CPU.PC, m.romBank, a)
		}
		off := m.romBank*0x4000 + int(a-0x4000)
		if off < len(m.rom) {
			return m.rom[off]
		}
		return 0xFF
	case a < 0xA000:
		return m.vram[a-0x8000]
	case a < 0xC000:
		if m.ramEnable {
			return m.extram[m.ramOff(a)]
		}
		return 0xFF
	case a < 0xE000:
		return m.wram[a-0xC000]
	case a < 0xFE00:
		return m.wram[a-0xE000] // echo RAM
	case a < 0xFEA0:
		return m.oam[a-0xFE00]
	case a < 0xFF00:
		return 0xFF // unusable
	case a < 0xFF80:
		return m.readIO(a)
	case a < 0xFFFF:
		return m.hram[a-0xFF80]
	default:
		return m.ie
	}
}

func (m *Machine) Write(a uint16, v byte) {
	if m.OnWrite != nil {
		m.OnWrite(m.CPU.PC, a, v)
	}
	switch {
	case a < 0x8000:
		m.mbcWrite(a, v)
	case a < 0xA000:
		if m.WatchPCs != nil && a >= m.WatchLo && a < m.WatchHi {
			m.WatchPCs[m.CPU.PC]++
		}
		m.vram[a-0x8000] = v
	case a < 0xC000:
		if m.ramEnable {
			m.extram[m.ramOff(a)] = v
		}
	case a < 0xE000:
		m.wram[a-0xC000] = v
	case a < 0xFE00:
		m.wram[a-0xE000] = v // echo
	case a < 0xFEA0:
		m.oam[a-0xFE00] = v
	case a < 0xFF00:
		// unusable
	case a < 0xFF80:
		m.writeIO(a, v)
	case a < 0xFFFF:
		m.hram[a-0xFF80] = v
	default:
		m.ie = v
	}
}

// ramOff maps an $A000-$BFFF address to a cartridge-RAM offset (RAM-banking mode
// selects an 8 KB bank).
func (m *Machine) ramOff(a uint16) int {
	bank := 0
	if m.mode == 1 {
		bank = m.ramBank
	}
	return bank*0x2000 + int(a-0xA000)
}

// mbcWrite handles a write into ROM space — an MBC1 control register.
func (m *Machine) mbcWrite(a uint16, v byte) {
	switch {
	case a < 0x2000: // RAM enable
		m.ramEnable = v&0x0F == 0x0A
	case a < 0x4000: // ROM bank (low 5 bits)
		lo := int(v & 0x1F)
		if lo == 0 {
			lo = 1 // the MBC1 bank-0→1 translation
		}
		m.romBank = (m.romBank&0x60 | lo) % m.nbanks
	case a < 0x6000: // RAM bank / ROM bank high bits
		m.ramBank = int(v & 3)
		m.romBank = (m.romBank&0x1F | int(v&3)<<5) % m.nbanks
		if m.romBank == 0 {
			m.romBank = 1
		}
	default: // banking mode
		m.mode = v & 1
	}
}

func (m *Machine) readIO(a uint16) byte {
	switch a {
	case regJOYP:
		return m.joyp()
	default:
		return m.io[a-0xFF00]
	}
}

func (m *Machine) writeIO(a uint16, v byte) {
	switch a {
	case regDIV: // any write resets the divider
		m.divCounter = 0
		m.io[regDIV-0xFF00] = 0
	case regDMA: // OAM DMA: copy $XX00-$XX9F into OAM
		src := uint16(v) << 8
		for i := uint16(0); i < 0xA0; i++ {
			m.oam[i] = m.Read(src + i)
		}
		m.io[regDMA-0xFF00] = v
	case regLY: // read-only
	default:
		m.io[a-0xFF00] = v
	}
}

// joyp resolves the joypad register from the selected line and Buttons (active-low).
func (m *Machine) joyp() byte {
	sel := m.io[regJOYP-0xFF00] & 0x30
	out := byte(0x0F)
	if sel&0x10 == 0 { // direction keys selected
		out &= ^(m.Buttons & 0x0F)
	}
	if sel&0x20 == 0 { // action buttons selected
		out &= ^(m.Buttons >> 4)
	}
	return 0xC0 | sel | (out & 0x0F)
}

// --- timing ----------------------------------------------------------------

// tick advances the timer and the LCD line counter by cyc T-cycles, raising the
// timer/VBlank/STAT interrupt request bits as their schedules come due.
func (m *Machine) tick(cyc int) {
	// DIV: high byte of a free-running 16-bit counter.
	m.divCounter = (m.divCounter + cyc) & 0xFFFF
	m.io[regDIV-0xFF00] = byte(m.divCounter >> 8)

	// TIMA: increments at the TAC-selected rate; overflow reloads TMA + IRQ.
	if tac := m.io[regTAC-0xFF00]; tac&0x04 != 0 {
		period := [4]int{1024, 16, 64, 256}[tac&3]
		m.timaCounter += cyc
		for m.timaCounter >= period {
			m.timaCounter -= period
			t := m.io[regTIMA-0xFF00] + 1
			if t == 0 {
				t = m.io[regTMA-0xFF00]
				m.reqInt(0x04) // timer interrupt
			}
			m.io[regTIMA-0xFF00] = t
		}
	}

	// LCD: advance LY one line per 456 dots when the LCD is on.
	if m.io[regLCDC-0xFF00]&0x80 == 0 {
		m.lcdDot, m.io[regLY-0xFF00] = 0, 0
		return
	}
	m.lcdDot += cyc
	for m.lcdDot >= dotsPerLine {
		m.lcdDot -= dotsPerLine
		ly := m.io[regLY-0xFF00] + 1
		if ly >= linesPerFrame {
			ly = 0
		}
		m.io[regLY-0xFF00] = ly
		stat := m.io[regSTAT-0xFF00]
		if ly == 144 {
			m.reqInt(0x01) // VBlank
			if stat&0x10 != 0 {
				m.reqInt(0x02) // mode-1 STAT
			}
		}
		// LYC=LY coincidence
		if ly == m.io[regLYC-0xFF00] {
			m.io[regSTAT-0xFF00] |= 0x04
			if stat&0x40 != 0 {
				m.reqInt(0x02)
			}
		} else {
			m.io[regSTAT-0xFF00] &^= 0x04
		}
	}
}

func (m *Machine) reqInt(bit byte) { m.io[regIF-0xFF00] |= bit }

// --- running ---------------------------------------------------------------

// Step runs one CPU instruction and advances time by its cycles; returns the cycles
// (0 if the CPU has hit a fatal Halt).
func (m *Machine) Step() int {
	if m.CPU.Halted {
		return 0
	}
	pc := m.CPU.PC
	cyc := m.CPU.Step()
	m.tick(cyc)
	if m.Sample {
		if m.PCHist == nil {
			m.PCHist = map[uint16]int{}
		}
		m.PCHist[pc]++
	}
	return cyc
}

// RunFrame runs ~one video frame (70224 T-cycles) of instructions. It stops early
// if the CPU hits a fatal Halt (an unimplemented/illegal opcode); returns false then.
func (m *Machine) RunFrame() bool {
	for c := 0; c < cyclesPerFrame; {
		cyc := m.Step()
		if cyc == 0 || m.CPU.Halted {
			return false
		}
		c += cyc
	}
	return true
}

// RunFrames runs n frames (or until a fatal Halt). Returns the number completed.
func (m *Machine) RunFrames(n int) int {
	for i := 0; i < n; i++ {
		if !m.RunFrame() {
			return i
		}
	}
	return n
}
