package z80

import "testing"

func TestDecode(t *testing.T) {
	cases := []struct {
		bytes []byte
		addr  uint16
		text  string
		len   int
		flow  Flow
	}{
		{[]byte{0x00}, 0, "NOP", 1, FlowSeq},
		{[]byte{0x3E, 0x12}, 0, "LD A,$12", 2, FlowSeq},
		{[]byte{0x21, 0x34, 0x12}, 0, "LD HL,$1234", 3, FlowSeq},
		{[]byte{0x7E}, 0, "LD A,(HL)", 1, FlowSeq},
		{[]byte{0x36, 0xFF}, 0, "LD (HL),$FF", 2, FlowSeq},
		{[]byte{0x47}, 0, "LD B,A", 1, FlowSeq},
		{[]byte{0x80}, 0, "ADD A,B", 1, FlowSeq},
		{[]byte{0xC6, 0x10}, 0, "ADD A,$10", 2, FlowSeq},
		{[]byte{0xA9}, 0, "XOR C", 1, FlowSeq},
		{[]byte{0xC3, 0x00, 0x80}, 0, "JP $8000", 3, FlowJump},
		{[]byte{0x18, 0xFE}, 0, "JR $0000", 2, FlowJump},   // to self
		{[]byte{0x28, 0x05}, 0, "JR Z,$0007", 2, FlowBranch},
		{[]byte{0x10, 0xFE}, 0, "DJNZ $0000", 2, FlowBranch},
		{[]byte{0xCA, 0x34, 0x12}, 0, "JP Z,$1234", 3, FlowBranch},
		{[]byte{0xCD, 0x34, 0x12}, 0, "CALL $1234", 3, FlowCall},
		{[]byte{0xDC, 0x34, 0x12}, 0, "CALL C,$1234", 3, FlowCall},
		{[]byte{0xC9}, 0, "RET", 1, FlowReturn},
		{[]byte{0xC8}, 0, "RET Z", 1, FlowSeq}, // conditional return: keep tracing
		{[]byte{0xE9}, 0, "JP (HL)", 1, FlowIndJump},
		{[]byte{0xF5}, 0, "PUSH AF", 1, FlowSeq},
		{[]byte{0xF1}, 0, "POP AF", 1, FlowSeq},
		{[]byte{0xC7}, 0, "RST $00", 1, FlowCall},
		{[]byte{0xFF}, 0, "RST $38", 1, FlowCall},
		{[]byte{0xD3, 0x12}, 0, "OUT ($12),A", 2, FlowSeq},
		{[]byte{0xDB, 0x12}, 0, "IN A,($12)", 2, FlowSeq},
		{[]byte{0x76}, 0, "HALT", 1, FlowSeq},
		{[]byte{0x07}, 0, "RLCA", 1, FlowSeq},
		{[]byte{0xEB}, 0, "EX DE,HL", 1, FlowSeq},
		// CB page
		{[]byte{0xCB, 0x00}, 0, "RLC B", 2, FlowSeq},
		{[]byte{0xCB, 0x46}, 0, "BIT 0,(HL)", 2, FlowSeq},
		{[]byte{0xCB, 0xFE}, 0, "SET 7,(HL)", 2, FlowSeq},
		// ED page
		{[]byte{0xED, 0xB0}, 0, "LDIR", 2, FlowSeq},
		{[]byte{0xED, 0x52}, 0, "SBC HL,DE", 2, FlowSeq},
		{[]byte{0xED, 0x43, 0x34, 0x12}, 0, "LD ($1234),BC", 4, FlowSeq},
		{[]byte{0xED, 0x4D}, 0, "RETI", 2, FlowReturn},
		{[]byte{0xED, 0x56}, 0, "IM 1", 2, FlowSeq},
		// DD/FD index prefixes
		{[]byte{0xDD, 0x21, 0x34, 0x12}, 0, "LD IX,$1234", 4, FlowSeq},
		{[]byte{0xDD, 0x7E, 0x05}, 0, "LD A,(IX+5)", 3, FlowSeq},
		{[]byte{0xFD, 0x7E, 0xFB}, 0, "LD A,(IY-5)", 3, FlowSeq},
		{[]byte{0xDD, 0x36, 0x05, 0xFF}, 0, "LD (IX+5),$FF", 4, FlowSeq},
		{[]byte{0xDD, 0x09}, 0, "ADD IX,BC", 2, FlowSeq},
		{[]byte{0xDD, 0x26, 0x12}, 0, "LD IXH,$12", 3, FlowSeq}, // undocumented
		{[]byte{0xDD, 0xE9}, 0, "JP (IX)", 2, FlowIndJump},
		// DDCB (displacement before the final opcode)
		{[]byte{0xDD, 0xCB, 0x05, 0x06}, 0, "RLC (IX+5)", 4, FlowSeq},
		{[]byte{0xDD, 0xCB, 0x05, 0x46}, 0, "BIT 0,(IX+5)", 4, FlowSeq},
	}
	for _, c := range cases {
		got := Decode(c.bytes, c.addr)
		if got.Text != c.text || got.Len != c.len || got.Flow != c.flow {
			t.Errorf("Decode(%X) = {%q len=%d flow=%d}, want {%q len=%d flow=%d}",
				c.bytes, got.Text, got.Len, got.Flow, c.text, c.len, c.flow)
		}
	}
}
