package c64

import "testing"

// TestReadProbe verifies the read probe observes data loads (and that a fetch
// is distinguishable from a data read by addr == pc).
func TestReadProbe(t *testing.T) {
	m := New(nil, 0)
	m.Watchdog = 0 // no tape; don't let the watchdog fire

	// Program at $1000:
	//   AD 00 40   LDA $4000
	//   AE 02 40   LDX $4002
	//   00         BRK   (halts the CPU)
	prog := []byte{0xAD, 0x00, 0x40, 0xAE, 0x02, 0x40, 0x00}
	copy(m.RAM[0x1000:], prog)
	m.RAM[0x4000] = 0x11
	m.RAM[0x4002] = 0x22

	dataReads := map[uint16]bool{}
	fetches := 0
	m.SetReadProbe(func(addr, pc uint16) {
		if addr == pc {
			fetches++ // opcode fetch
		} else {
			dataReads[addr] = true
		}
	})

	m.CPU.PC = 0x1000
	m.Run(1000)

	if !dataReads[0x4000] || !dataReads[0x4002] {
		t.Errorf("probe missed data reads: got %v, want $4000 and $4002", dataReads)
	}
	if fetches == 0 {
		t.Error("probe recorded no opcode fetches")
	}
}

// TestReadProbeNilByDefault makes sure a machine with no probe still runs (the
// common loader-extraction path must not require a probe).
func TestReadProbeNilByDefault(t *testing.T) {
	m := New(nil, 0)
	m.Watchdog = 0
	copy(m.RAM[0x1000:], []byte{0xA9, 0x05, 0x00}) // LDA #$05 ; BRK
	m.CPU.PC = 0x1000
	m.Run(100)
	if m.CPU.A != 0x05 {
		t.Errorf("A = $%02X, want $05", m.CPU.A)
	}
}
