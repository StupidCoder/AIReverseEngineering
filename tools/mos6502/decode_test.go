package mos6502

import "testing"

func TestDecodeFlow(t *testing.T) {
	mem := make([]byte, 0x10000)
	put := func(a int, bs ...byte) {
		for i, b := range bs {
			mem[a+i] = b
		}
	}
	put(0x1000, 0x20, 0x34, 0x12) // JSR $1234
	put(0x1003, 0xD0, 0x05)       // BNE $100A
	put(0x1005, 0x4C, 0x00, 0x20) // JMP $2000
	put(0x1008, 0x6C, 0xFE, 0xFF) // JMP ($FFFE)
	put(0x100B, 0x60)             // RTS
	put(0x100C, 0xA9, 0x01)       // LDA #$01

	cases := []struct {
		a      uint16
		mn     string
		flow   Flow
		target uint16
		length int
	}{
		{0x1000, "JSR", FlowCall, 0x1234, 3},
		{0x1003, "BNE", FlowBranch, 0x100A, 2},
		{0x1005, "JMP", FlowJump, 0x2000, 3},
		{0x1008, "JMP", FlowIndJump, 0xFFFE, 3},
		{0x100B, "RTS", FlowReturn, 0, 1},
		{0x100C, "LDA", FlowSeq, 0, 2},
	}
	for _, c := range cases {
		in := Decode(mem, c.a)
		if in.Mnem != c.mn || in.Flow != c.flow || in.Len != c.length {
			t.Errorf("$%04X: got %s/%d/len%d, want %s/%d/len%d", c.a, in.Mnem, in.Flow, in.Len, c.mn, c.flow, c.length)
		}
		if c.flow == FlowCall || c.flow == FlowBranch || c.flow == FlowJump {
			if !in.HasTarget || in.Target != c.target {
				t.Errorf("$%04X: target $%04X, want $%04X", c.a, in.Target, c.target)
			}
		}
	}
}
