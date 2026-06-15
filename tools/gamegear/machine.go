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
		v.CRAM[v.addr&0x3F] = b
	} else {
		v.VRAM[v.addr&0x3FFF] = b
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

// Machine is a Game Gear: CPU + RAM + cartridge + VDP, implementing z80.Bus.
type Machine struct {
	CPU *z80.CPU
	VDP VDP

	rom    []byte
	nbanks int
	ram    [0x2000]byte // 8 KB work RAM, mirrored $C000-$FFFF
	slot   [3]int       // ROM bank mapped into slot 0/1/2 ($0000/$4000/$8000)
}

// NewMachine builds a machine from a cartridge image and resets it to power-on
// state (mapper slots 0/1/2 -> banks 0/1/2, PC = $0000).
func NewMachine(rom []byte) *Machine {
	m := &Machine{rom: rom, nbanks: len(rom) / 0x4000}
	m.slot = [3]int{0, 1, 2}
	m.CPU = z80.NewCPU(m)
	return m
}

func (m *Machine) bankByte(bank, off int) byte {
	a := bank*0x4000 + off
	if a < len(m.rom) {
		return m.rom[a]
	}
	return 0xFF
}

// Read implements z80.Bus. The first 1 KB is always bank 0 (the Sega mapper fixes
// it so the interrupt vectors stay reachable); the rest of each ROM window follows
// its slot register. $C000-$FFFF is the 8 KB work RAM, mirrored.
func (m *Machine) Read(a uint16) byte {
	switch {
	case a < 0x0400:
		return m.bankByte(0, int(a))
	case a < 0x4000:
		return m.bankByte(m.slot[0], int(a))
	case a < 0x8000:
		return m.bankByte(m.slot[1], int(a-0x4000))
	case a < 0xC000:
		return m.bankByte(m.slot[2], int(a-0x8000))
	default:
		return m.ram[a&0x1FFF]
	}
}

// Write implements z80.Bus. Only RAM is writable; writes to $FFFD-$FFFF also set
// the mapper slot registers (they live in RAM and the mapper snoops them).
func (m *Machine) Write(a uint16, v byte) {
	if a < 0xC000 {
		return // ROM
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
	default:
		return 0xFF
	}
}

// Out implements z80.Bus port writes. $BE = VDP data, $BF = VDP control; the PSG
// and other ports are accepted and ignored.
func (m *Machine) Out(port uint16, v byte) {
	switch byte(port) {
	case 0xBE:
		m.VDP.writeData(v)
	case 0xBF:
		m.VDP.writeControl(v)
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
		m.CPU.Step()
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
		m.CPU.Step()
		if m.CPU.Halted {
			return false
		}
	}
	m.CPU.RequestIRQ(false)
	return true
}
