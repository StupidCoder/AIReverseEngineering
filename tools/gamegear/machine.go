package gamegear

// A minimal Sega Game Gear machine model: a Z80 (from the z80 package) wired to
// 8 KB of work RAM, the Sega cartridge mapper, and enough of the 315-5124 VDP to
// run a game's boot code and capture what it draws. It is NOT a cycle-accurate
// emulator — there is no PSG, no sprite rendering, no per-pixel video. Its job is
// to be an *oracle*: run the real ROM, let it decompress tiles, build a name
// table and program CRAM through the VDP ports exactly as it does on hardware,
// then hand back VRAM/CRAM so an exact screen can be composed (see RenderNameTable).
// This is why the work belongs here and not in a game's extract: the mapper and
// VDP port protocol are Game Gear hardware, identical across cartridges.

import "stupidcoder.com/tools/z80"

// VDP holds the captured video state: 16 KB VRAM (tiles + name table + SAT),
// 32-entry CRAM (64 bytes), and the 16 registers. The fields are exported so a
// renderer can read VRAM[$3800] (name table), VRAM[0] (tiles) and CRAM directly.
type VDP struct {
	VRAM [0x4000]byte
	CRAM [0x40]byte
	Regs [16]byte

	addr    uint16 // current VRAM/CRAM auto-increment address
	code    byte   // command code: 0 VRAM read, 1 VRAM write, 2 register, 3 CRAM
	latch   byte   // first control byte (low address)
	latched bool   // a control byte is waiting for its second half
	readBuf byte   // VRAM read prefetch buffer
	status  byte   // status flags (bit7 = frame interrupt pending)
	line    byte   // current scanline, for the V-counter port ($7E)

	// Writes counts VRAM byte-writes per 1 KB region (addr>>10), a cheap way to see
	// which regions a screen actually touches — tile patterns ($0000-$37FF), the name
	// table ($3800-$3EFF), the sprite-attribute table ($3F00-$3FFF). Reset with
	// ResetWrites. CRAMWrites does the same for the 64-byte CRAM as a single counter.
	Writes     [16]uint32
	CRAMWrites uint32

	// CRAMLineLog, when non-nil, records (cramAddr, scanline) for every CRAM write — to
	// tell a vblank palette load from a mid-frame raster palette change.
	CRAMLineLog *[][2]byte
}

// ResetWrites zeroes the VRAM/CRAM write counters (call before the window of interest).
func (v *VDP) ResetWrites() {
	v.Writes = [16]uint32{}
	v.CRAMWrites = 0
}

// ActiveSprites counts SAT entries (Y table at $3F00) whose Y is not the $D0 line
// terminator or the $E0 "off-screen/hidden" value — i.e. sprites actually on screen.
func (v *VDP) ActiveSprites() int {
	n := 0
	for i := 0; i < 64; i++ {
		if y := v.VRAM[0x3F00+i]; y != 0xD0 && y != 0xE0 {
			n++
		}
	}
	return n
}

// writeControl handles a write to the control port ($BF): two bytes form a command.
func (v *VDP) writeControl(b byte) {
	if !v.latched {
		v.latch = b
		v.latched = true
		return
	}
	v.latched = false
	v.code = b >> 6
	v.addr = uint16(b&0x3F)<<8 | uint16(v.latch)
	switch v.code {
	case 2: // register write: reg = low nibble of the high byte, value = first byte
		v.Regs[b&0x0F] = v.latch
	case 0: // VRAM read: prefetch the first byte
		v.readBuf = v.VRAM[v.addr&0x3FFF]
		v.addr++
	}
}

// writeData handles a write to the data port ($BE): goes to VRAM or CRAM.
func (v *VDP) writeData(b byte) {
	v.latched = false
	if v.code == 3 {
		if v.CRAMLineLog != nil {
			*v.CRAMLineLog = append(*v.CRAMLineLog, [2]byte{byte(v.addr & 0x3F), v.line})
		}
		v.CRAM[v.addr&0x3F] = b
		v.CRAMWrites++
	} else {
		a := v.addr & 0x3FFF
		v.VRAM[a] = b
		v.Writes[a>>10]++
	}
	v.addr++
	v.readBuf = b
}

// readData handles a read from the data port: returns the prefetch buffer and
// refills it from VRAM at the auto-incrementing address.
func (v *VDP) readData() byte {
	v.latched = false
	r := v.readBuf
	v.readBuf = v.VRAM[v.addr&0x3FFF]
	v.addr++
	return r
}

// readStatus returns the status byte and clears the latch + pending flags (this is
// how the frame-interrupt handler acknowledges the VDP: IN A,($BF)).
func (v *VDP) readStatus() byte {
	v.latched = false
	s := v.status
	v.status &= 0x1F
	return s
}

// CapEntry is one observed call of CapturePC: the argument registers and the bank
// paged into slot 1 at that moment.
type CapEntry struct {
	HL, BC, DE uint16
	Slot1      int
}

// Machine is a Game Gear: CPU + RAM + cartridge + VDP, implementing z80.Bus.
type Machine struct {
	CPU *z80.CPU
	VDP VDP
	PSG PSG // SN76489 sound chip register state (fed by Out port $7F)

	rom    []byte
	nbanks int
	ram    [0x2000]byte // 8 KB work RAM, mirrored $C000-$FFFF
	slot   [3]int       // ROM bank mapped into slot 0/1/2 ($0000/$4000/$8000)

	// Injected controller state (active-low, $FF = nothing pressed). Port $00 bit 7 is
	// Start; port $DC is the D-pad/buttons (the game masks $7F). Set bits low to press.
	Pad00, PadDC byte

	// When Sample is set, RunFrame tallies the PC after every instruction into PCHist —
	// a cheap profiler to see what code the machine is actually executing.
	Sample bool
	PCHist map[uint16]int

	// RAM write watchpoint: when RAMWatchPCs != nil, every work-RAM write in
	// [RAMWatchLo,RAMWatchHi) records the CPU's PC — "which code wrote this RAM?"
	RAMWatchLo, RAMWatchHi uint16
	RAMWatchPCs            map[uint16]int

	// WriteHook, when non-nil, is called on every work-RAM write with the
	// instruction-start PC (stepPC, the address of the storing instruction itself,
	// not the post-fetch PC), the address and the value — a value-capturing debug
	// trace finer than RAMWatchPCs and correct for attribution.
	WriteHook func(pc, addr uint16, v byte)
	stepPC    uint16 // PC at the start of the instruction currently executing

	// Register capture: when CapturePC != 0, the first time execution reaches it the
	// CPU's HL and BC are saved (CapHL/CapBC) and Captured set — for reading a routine's
	// arguments (e.g. the source pointer + length at a decompressor entry).
	CapturePC    uint16
	CapHL, CapBC uint16
	CapSlot1     int
	Captured     bool

	// CapLog records (HL,BC,DE,slot1) at EVERY reach of CapturePC — for routines called
	// multiple times where we must pick the call with a particular destination (DE).
	CapLog []CapEntry

	// Return-time RAM snapshot: after CapturePC has fired, the first time PC leaves the
	// routine body [CapLo,CapHi] the 4 KB at CapOutBase is copied into CapOut (CapOutDone
	// set). This grabs a decompressor's output the instant it RETs to its caller, before
	// any other code can mutate it — regardless of which RET it exits through.
	CapLo, CapHi uint16
	CapOutBase   uint16
	CapOut       [0x1000]byte
	CapOutDone   bool

	// VRAM write watchpoint: when WatchHi > WatchLo, every VRAM write whose address
	// falls in [WatchLo,WatchHi) records the CPU's PC in WatchPCs (a histogram of how
	// many bytes each routine wrote there). It answers "which code drew this part of
	// the screen?" — the PC is the instruction after the OUT, enough to name the loop.
	WatchLo, WatchHi uint16
	WatchPCs         map[uint16]int
}

// Watch arms the VRAM write watchpoint over [lo,hi) and clears any prior hits.
func (m *Machine) Watch(lo, hi uint16) {
	m.WatchLo, m.WatchHi = lo, hi
	m.WatchPCs = map[uint16]int{}
}

// NewMachine builds a machine from a cartridge image and resets it to power-on
// state (mapper slots 0/1/2 -> banks 0/1/2, PC = $0000).
func NewMachine(rom []byte) *Machine {
	m := &Machine{rom: rom, nbanks: len(rom) / 0x4000}
	m.slot = [3]int{0, 1, 2}
	m.Pad00, m.PadDC = 0xFF, 0xFF // nothing pressed
	m.CPU = z80.NewCPU(m)
	return m
}

// Read implements z80.Bus. ROM addresses go through the shared FileOffset mapping
// (the first 1 KB is fixed to bank 0; the rest of each window follows its slot
// register); $C000-$FFFF is the 8 KB work RAM, mirrored.
func (m *Machine) Read(a uint16) byte {
	off, inRAM := FileOffset(m.slot, a)
	if inRAM {
		return m.ram[a&0x1FFF]
	}
	if off < len(m.rom) {
		return m.rom[off]
	}
	return 0xFF
}

// Write implements z80.Bus. Only RAM is writable; writes to $FFFD-$FFFF also set
// the mapper slot registers (they live in RAM and the mapper snoops them).
func (m *Machine) Write(a uint16, v byte) {
	if a < 0xC000 {
		return // ROM
	}
	if m.RAMWatchPCs != nil && a >= m.RAMWatchLo && a < m.RAMWatchHi {
		m.RAMWatchPCs[m.CPU.PC]++
	}
	if m.WriteHook != nil {
		m.WriteHook(m.stepPC, a, v)
	}
	m.ram[a&0x1FFF] = v
	switch a {
	case 0xFFFD:
		m.slot[0] = int(v) % m.nbanks
	case 0xFFFE:
		m.slot[1] = int(v) % m.nbanks
	case 0xFFFF:
		m.slot[2] = int(v) % m.nbanks
	}
}

// In implements z80.Bus port reads. $BE = VDP data, $BF = VDP status (also acks the
// frame interrupt), $7E = V-counter (the boot polls it). Ports decode on the low byte.
func (m *Machine) In(port uint16) byte {
	switch byte(port) {
	case 0xBE:
		return m.VDP.readData()
	case 0xBF:
		m.CPU.RequestIRQ(false) // reading status acknowledges the interrupt
		return m.VDP.readStatus()
	case 0x7E:
		return m.VDP.line
	case 0x7F:
		return 0
	case 0x00:
		return m.Pad00 // Start = bit 7
	case 0xDC:
		return m.PadDC // D-pad / buttons
	default:
		return 0xFF
	}
}

// Out implements z80.Bus port writes. $BE = VDP data, $BF = VDP control; the PSG
// and other ports are accepted and ignored.
func (m *Machine) Out(port uint16, v byte) {
	switch byte(port) {
	case 0xBE:
		if m.WatchPCs != nil && m.VDP.code != 3 {
			if a := m.VDP.addr & 0x3FFF; a >= m.WatchLo && a < m.WatchHi {
				m.WatchPCs[m.CPU.PC]++
			}
		}
		m.VDP.writeData(v)
	case 0xBF:
		m.VDP.writeControl(v)
	case 0x7F:
		m.PSG.Write(v)
	}
}

// RunFrame advances the machine by one ~60 Hz video frame: it steps the CPU for a
// fixed instruction budget while sweeping the V-counter across the 262 scanlines,
// then, at the start of vblank, sets the frame-interrupt flag and (if enabled in
// VDP register 1, bit 5) raises the maskable interrupt so the per-frame handler
// runs. Returns false if the CPU has fatally halted.
//
// The instruction budget is an approximation (a Game Gear runs ~3.58 MHz / 60 Hz
// ≈ 59.7k cycles per frame, very roughly ~15k instructions); it only needs to be
// large enough that the boot's per-frame work completes within a frame, which it
// comfortably is for the static screens we capture.
func (m *Machine) RunFrame() bool {
	const budget = 20000
	for i := 0; i < budget; i++ {
		m.VDP.line = byte((i * 262 / budget) & 0xFF)
		if m.VDP.line >= 192 {
			m.VDP.status |= 0x80
		}
		m.stepPC = m.CPU.PC
		m.CPU.Step()
		if m.Sample {
			m.PCHist[m.CPU.PC]++
		}
		if m.CapturePC != 0 && m.CPU.PC == m.CapturePC {
			hl := uint16(m.CPU.H)<<8 | uint16(m.CPU.L)
			bc := uint16(m.CPU.B)<<8 | uint16(m.CPU.C)
			de := uint16(m.CPU.D)<<8 | uint16(m.CPU.E)
			m.CapLog = append(m.CapLog, CapEntry{HL: hl, BC: bc, DE: de, Slot1: m.slot[1]})
			if !m.Captured {
				m.CapHL, m.CapBC, m.CapSlot1, m.Captured = hl, bc, m.slot[1], true
			}
		}
		if m.CapHi != 0 && m.Captured && !m.CapOutDone && (m.CPU.PC < m.CapLo || m.CPU.PC > m.CapHi) {
			for j := 0; j < 0x1000; j++ {
				m.CapOut[j] = m.ram[(m.CapOutBase+uint16(j))&0x1FFF]
			}
			m.CapOutDone = true
		}
		if m.CPU.Halted {
			return false
		}
	}
	// Enter vblank: flag it and request the frame interrupt if the VDP enables it.
	m.VDP.line = 192
	m.VDP.status |= 0x80
	if m.VDP.Regs[1]&0x20 != 0 {
		m.CPU.RequestIRQ(true)
	}
	// Give the interrupt handler room to run and ack (it clears the IRQ via IN $BF).
	for i := 0; i < budget/2; i++ {
		m.stepPC = m.CPU.PC
		m.CPU.Step()
		if m.Sample {
			m.PCHist[m.CPU.PC]++
		}
		if m.CPU.Halted {
			return false
		}
	}
	m.CPU.RequestIRQ(false)
	return true
}
