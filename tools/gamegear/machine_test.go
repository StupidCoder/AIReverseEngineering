package gamegear

import "testing"

// makeROM builds a 2-bank (32 KB) cartridge with code at $0000.
func makeROM(code ...byte) []byte {
	rom := make([]byte, 0x8000)
	copy(rom, code)
	return rom
}

// step runs the CPU n instructions (enough for a short test program).
func step(m *Machine, n int) {
	for i := 0; i < n; i++ {
		m.CPU.Step()
	}
}

// TestVRAMWrite drives the VDP write protocol the boot uses: set the address via
// two control-port writes (low byte, then high|$40 = write-VRAM), then stream data.
func TestVRAMWrite(t *testing.T) {
	m := NewMachine(makeROM(
		0x3E, 0x00, 0xD3, 0xBF, // LD A,$00 ; OUT ($BF),A  (addr low)
		0x3E, 0x40, 0xD3, 0xBF, // LD A,$40 ; OUT ($BF),A  (addr high | write-VRAM)
		0x3E, 0xAB, 0xD3, 0xBE, // LD A,$AB ; OUT ($BE),A  (VRAM[0] = $AB)
		0x3E, 0xCD, 0xD3, 0xBE, // LD A,$CD ; OUT ($BE),A  (VRAM[1] = $CD, auto-inc)
	))
	step(m, 8)
	if m.VDP.VRAM[0] != 0xAB || m.VDP.VRAM[1] != 0xCD {
		t.Errorf("VRAM[0,1] = %02X %02X, want AB CD", m.VDP.VRAM[0], m.VDP.VRAM[1])
	}
}

// TestVDPRegister checks a register write: control word = value, then $80|reg.
func TestVDPRegister(t *testing.T) {
	m := NewMachine(makeROM(
		0x3E, 0xE2, 0xD3, 0xBF, // LD A,$E2 ; OUT ($BF),A  (value)
		0x3E, 0x81, 0xD3, 0xBF, // LD A,$81 ; OUT ($BF),A  ($80|1 -> register 1)
	))
	step(m, 4)
	if m.VDP.Regs[1] != 0xE2 {
		t.Errorf("VDP reg1 = %02X, want E2", m.VDP.Regs[1])
	}
}

// TestCRAMWrite checks a palette (CRAM) write: address with the $C0 command.
func TestCRAMWrite(t *testing.T) {
	m := NewMachine(makeROM(
		0x3E, 0x00, 0xD3, 0xBF, // LD A,$00 ; OUT ($BF),A
		0x3E, 0xC0, 0xD3, 0xBF, // LD A,$C0 ; OUT ($BF),A  (CRAM write, addr 0)
		0x3E, 0x0F, 0xD3, 0xBE, // LD A,$0F ; OUT ($BE),A  (CRAM[0] = $0F -> pure red)
	))
	step(m, 6)
	if m.VDP.CRAM[0] != 0x0F {
		t.Errorf("CRAM[0] = %02X, want 0F", m.VDP.CRAM[0])
	}
}

// TestMapper checks the Sega cartridge mapper: writing a bank number to $FFFE
// pages that bank into slot 1 ($4000-$7FFF).
func TestMapper(t *testing.T) {
	rom := make([]byte, 0x8000) // 2 banks
	rom[0x4000] = 0x11          // first byte of bank 1
	copy(rom, []byte{
		0x3E, 0x00, 0x32, 0xFE, 0xFF, // LD A,0 ; LD ($FFFE),A  (slot1 <- bank 0)
	})
	m := NewMachine(rom)
	// By default slot 1 holds bank 1, so $4000 reads bank 1's first byte.
	if got := m.Read(0x4000); got != 0x11 {
		t.Errorf("default slot1: read $4000 = %02X, want 11 (bank 1)", got)
	}
	step(m, 2) // page bank 0 into slot 1
	if m.slot[1] != 0 {
		t.Fatalf("slot1 = %d, want 0 after LD ($FFFE),0", m.slot[1])
	}
	// Now $4000 reads bank 0's first byte = the program's first opcode ($3E).
	if got := m.Read(0x4000); got != 0x3E {
		t.Errorf("after paging: read $4000 = %02X, want 3E (bank 0)", got)
	}
}
