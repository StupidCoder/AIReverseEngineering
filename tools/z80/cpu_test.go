package z80

import "testing"

// bus is a 64 KB flat memory plus a 64 KB I/O space, enough to exercise the core.
type bus struct {
	mem  []byte
	port [0x10000]byte
	ins  []byte // log of port writes (OUT), for the block-IO tests
}

func (b *bus) Read(a uint16) byte     { return b.mem[a] }
func (b *bus) Write(a uint16, v byte) { b.mem[a] = v }
func (b *bus) In(p uint16) byte       { return b.port[p] }
func (b *bus) Out(p uint16, v byte) {
	b.port[p] = v
	b.ins = append(b.ins, v)
}

func newCPU(code ...byte) (*CPU, *bus) {
	b := &bus{mem: make([]byte, 0x10000)}
	copy(b.mem, code)
	c := NewCPU(b)
	c.SP = 0xFFFE
	return c, b
}

// run single-steps until PC reaches stop, the CPU halts, or the budget runs out.
func run(t *testing.T, c *CPU, stop uint16) {
	t.Helper()
	for i := 0; i < 10000; i++ {
		if c.PC == stop {
			return
		}
		if c.Halted {
			t.Fatalf("halted: %s", c.HaltReason)
		}
		c.Step()
	}
	t.Fatalf("did not reach $%X (pc=$%X)", stop, c.PC)
}

func TestAddSubFlags(t *testing.T) {
	c, _ := newCPU()
	c.A = 0x7F
	c.add8(0x01, false) // 7F+01 = 80: sign set, overflow set, no carry
	if c.A != 0x80 || !c.getf(flagS) || !c.getf(flagP) || c.getf(flagC) || c.getf(flagN) {
		t.Errorf("7F+01 = %02X S=%v V=%v C=%v N=%v, want 80 S V C̄ N̄",
			c.A, c.getf(flagS), c.getf(flagP), c.getf(flagC), c.getf(flagN))
	}
	c.A = 0xFF
	c.add8(0x01, false) // FF+01 = 00: zero, carry, half-carry
	if c.A != 0x00 || !c.getf(flagZ) || !c.getf(flagC) || !c.getf(flagH) {
		t.Errorf("FF+01 = %02X Z=%v C=%v H=%v, want 00 Z C H",
			c.A, c.getf(flagZ), c.getf(flagC), c.getf(flagH))
	}
	c.A = c.sub8(0x01, false, true) // 0-1 = FF: borrow (carry), sign, N (sub8 returns the result)
	if c.A != 0xFF || !c.getf(flagC) || !c.getf(flagS) || !c.getf(flagN) {
		t.Errorf("00-01 = %02X C=%v S=%v N=%v, want FF C S N",
			c.A, c.getf(flagC), c.getf(flagS), c.getf(flagN))
	}
	c.A = 0x10
	c.sub8(0x10, false, false) // CP equal: Z set, no carry
	if !c.getf(flagZ) || c.getf(flagC) {
		t.Errorf("CP equal: Z=%v C=%v, want Z C̄", c.getf(flagZ), c.getf(flagC))
	}
}

// TestSumLoop sums 10+9+...+1 = 55 with DJNZ.
func TestSumLoop(t *testing.T) {
	c, _ := newCPU(
		0x3E, 0x00, //       LD A,0
		0x06, 0x0A, //       LD B,10
		0x80, //       loop: ADD A,B   (A += B)
		0x10, 0xFD, //       DJNZ loop
	)
	run(t, c, 7)
	// 10+9+...+1 = 55
	if c.A != 55 {
		t.Errorf("sum = %d, want 55", c.A)
	}
	if c.B != 0 {
		t.Errorf("B = %d, want 0 after DJNZ exit", c.B)
	}
}

// TestCallRet exercises CALL/RET and the stack.
func TestCallRet(t *testing.T) {
	c, _ := newCPU(
		0xCD, 0x06, 0x00, // CALL $0006
		0x76,             // HALT  (should not run; we stop before)
		0x00, 0x00,       //
		0x3E, 0x2A,       // $0006: LD A,$2A
		0xC9,             // RET
	)
	run(t, c, 3)
	if c.A != 0x2A {
		t.Errorf("A = %02X, want 2A", c.A)
	}
	if c.SP != 0xFFFE {
		t.Errorf("SP = %04X, want FFFE (balanced)", c.SP)
	}
}

// TestLDIR copies a block with LDIR.
func TestLDIR(t *testing.T) {
	c, b := newCPU(
		0x21, 0x00, 0x80, // LD HL,$8000
		0x11, 0x00, 0x90, // LD DE,$9000
		0x01, 0x04, 0x00, // LD BC,4
		0xED, 0xB0,       // LDIR
	)
	copy(b.mem[0x8000:], []byte{0xDE, 0xAD, 0xBE, 0xEF})
	run(t, c, 11)
	for i, want := range []byte{0xDE, 0xAD, 0xBE, 0xEF} {
		if got := b.mem[0x9000+i]; got != want {
			t.Errorf("dst[%d] = %02X, want %02X", i, got, want)
		}
	}
	if c.bc() != 0 {
		t.Errorf("BC = %04X, want 0", c.bc())
	}
}

// TestOUTI streams 3 bytes to a port with OTIR (the pattern the VDP fill uses).
func TestOTIR(t *testing.T) {
	c, b := newCPU(
		0x21, 0x00, 0x80, // LD HL,$8000
		0x01, 0xBE, 0x03, // LD BC,$03BE  (C=port $BE, B=count 3)
		0xED, 0xB3,       // OTIR
	)
	copy(b.mem[0x8000:], []byte{0x11, 0x22, 0x33})
	run(t, c, 8)
	if len(b.ins) != 3 || b.ins[0] != 0x11 || b.ins[2] != 0x33 {
		t.Errorf("port writes = %v, want [11 22 33]", b.ins)
	}
	if c.B != 0 {
		t.Errorf("B = %d, want 0", c.B)
	}
}

// TestIXIndexed exercises the DD prefix: LD (IX+2),A then read it back.
func TestIXIndexed(t *testing.T) {
	c, b := newCPU(
		0xDD, 0x21, 0x00, 0x80, // LD IX,$8000
		0x3E, 0x99,             // LD A,$99
		0xDD, 0x77, 0x02,       // LD (IX+2),A
		0xDD, 0x7E, 0x02,       // LD A,(IX+2)  (redundant, but checks the read path)
	)
	run(t, c, 12)
	if b.mem[0x8002] != 0x99 {
		t.Errorf("(IX+2) = %02X, want 99", b.mem[0x8002])
	}
	if c.A != 0x99 {
		t.Errorf("A = %02X, want 99", c.A)
	}
}

// TestBitSetRes exercises the CB page: SET/BIT/RES.
func TestBitSetRes(t *testing.T) {
	c, _ := newCPU(
		0x3E, 0x00,       // LD A,0
		0xCB, 0xFF,       // SET 7,A   -> A=$80
		0xCB, 0x7F,       // BIT 7,A   -> Z clear
		0xCB, 0xBF,       // RES 7,A   -> A=0
	)
	c.Step() // LD
	c.Step() // SET
	if c.A != 0x80 {
		t.Fatalf("after SET 7,A: A=%02X, want 80", c.A)
	}
	c.Step() // BIT 7,A
	if c.getf(flagZ) {
		t.Errorf("BIT 7 of $80: Z set, want clear")
	}
	c.Step() // RES
	if c.A != 0x00 {
		t.Errorf("after RES 7,A: A=%02X, want 00", c.A)
	}
}

// TestInterrupt checks IM 1 maskable-interrupt servicing (vector $0038).
func TestInterrupt(t *testing.T) {
	c, _ := newCPU(
		0xFB, // $0000: EI
		0x00, // $0001: NOP   (interrupts enable only after this)
		0x00, // $0002: NOP   (IRQ taken here)
		0x76, // $0003: HALT
	)
	c.mainSetVec(t)
	c.Step() // EI (eiPending)
	c.RequestIRQ(true)
	c.Step() // NOP: EI delay, no service yet
	if c.PC != 0x0002 {
		t.Fatalf("after EI+NOP: PC=%04X, want 0002 (no premature IRQ)", c.PC)
	}
	c.Step() // IRQ serviced -> PC = $0038
	if c.PC != 0x0038 {
		t.Errorf("IRQ not serviced: PC=%04X, want 0038", c.PC)
	}
	if c.IFF1 {
		t.Errorf("IFF1 still set after IRQ accept")
	}
}

// mainSetVec plants a RET at the $0038 vector so the handler returns cleanly.
func (c *CPU) mainSetVec(t *testing.T) {
	t.Helper()
	b := c.bus.(*bus)
	b.mem[0x0038] = 0xC9 // RET
}

// LD (IX+d),n must read the displacement d BEFORE the immediate n (regression:
// the two were swapped because Go evaluated the c.fetch() argument first).
func TestLDIndexedImmediateOrder(t *testing.T) {
	// DD 36 05 14 : LD (IX+5),$14  with IX=$C000 -> $C005 = $14, not $C014 = $05
	c, b := newCPU(0xDD, 0x36, 0x05, 0x14)
	c.IX = 0xC000
	c.Step()
	if b.mem[0xC005] != 0x14 {
		t.Fatalf("LD (IX+5),$14: want $C005=$14, got $%02X ($C014=$%02X)", b.mem[0xC005], b.mem[0xC014])
	}
	if c.PC != 4 {
		t.Fatalf("PC after LD (IX+d),n: want 4, got %d", c.PC)
	}
	// FD 36 FE 99 : LD (IY-2),$99 with IY=$D010 -> $D00E = $99
	c2, b2 := newCPU(0xFD, 0x36, 0xFE, 0x99)
	c2.IY = 0xD010
	c2.Step()
	if b2.mem[0xD00E] != 0x99 {
		t.Fatalf("LD (IY-2),$99: want $D00E=$99, got $%02X", b2.mem[0xD00E])
	}
}
