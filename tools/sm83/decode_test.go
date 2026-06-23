package sm83

import "testing"

func TestDecode(t *testing.T) {
	cases := []struct {
		bytes []byte
		addr  uint16
		text  string
		len   int
		flow  Flow
	}{
		// main page basics
		{[]byte{0x00}, 0, "NOP", 1, FlowSeq},
		{[]byte{0x3E, 0x12}, 0, "LD A,$12", 2, FlowSeq},
		{[]byte{0x21, 0x34, 0x12}, 0, "LD HL,$1234", 3, FlowSeq},
		{[]byte{0x7E}, 0, "LD A,(HL)", 1, FlowSeq},
		{[]byte{0x36, 0xFF}, 0, "LD (HL),$FF", 2, FlowSeq},
		{[]byte{0x47}, 0, "LD B,A", 1, FlowSeq},
		{[]byte{0x80}, 0, "ADD A,B", 1, FlowSeq},
		{[]byte{0xC6, 0x10}, 0, "ADD A,$10", 2, FlowSeq},
		{[]byte{0xA9}, 0, "XOR C", 1, FlowSeq},
		{[]byte{0x09}, 0, "ADD HL,BC", 1, FlowSeq},
		{[]byte{0x76}, 0, "HALT", 1, FlowSeq},
		{[]byte{0x07}, 0, "RLCA", 1, FlowSeq},
		// control flow
		{[]byte{0xC3, 0x50, 0x01}, 0, "JP $0150", 3, FlowJump},
		{[]byte{0x18, 0xFE}, 0, "JR $0000", 2, FlowJump}, // to self
		{[]byte{0x28, 0x05}, 0, "JR Z,$0007", 2, FlowBranch},
		{[]byte{0x20, 0xFA}, 0, "JR NZ,$FFFC", 2, FlowBranch},
		{[]byte{0xCA, 0x34, 0x12}, 0, "JP Z,$1234", 3, FlowBranch},
		{[]byte{0xCD, 0x4F, 0x22}, 0, "CALL $224F", 3, FlowCall},
		{[]byte{0xDC, 0x34, 0x12}, 0, "CALL C,$1234", 3, FlowCall},
		{[]byte{0xC9}, 0, "RET", 1, FlowReturn},
		{[]byte{0xC8}, 0, "RET Z", 1, FlowSeq}, // conditional return: keep tracing
		{[]byte{0xD9}, 0, "RETI", 1, FlowReturn},
		{[]byte{0xE9}, 0, "JP (HL)", 1, FlowIndJump},
		{[]byte{0xF5}, 0, "PUSH AF", 1, FlowSeq},
		{[]byte{0xF1}, 0, "POP AF", 1, FlowSeq},
		{[]byte{0xC7}, 0, "RST $00", 1, FlowCall},
		{[]byte{0xFF}, 0, "RST $38", 1, FlowCall},
		{[]byte{0xEF}, 0, "RST $28", 1, FlowCall},
		// Game-Boy-specific opcodes
		{[]byte{0x08, 0x00, 0xC0}, 0, "LD ($C000),SP", 3, FlowSeq},
		{[]byte{0x10, 0x00}, 0, "STOP", 2, FlowSeq},
		{[]byte{0x22}, 0, "LD (HL+),A", 1, FlowSeq},
		{[]byte{0x2A}, 0, "LD A,(HL+)", 1, FlowSeq},
		{[]byte{0x32}, 0, "LD (HL-),A", 1, FlowSeq},
		{[]byte{0x3A}, 0, "LD A,(HL-)", 1, FlowSeq},
		{[]byte{0xE0, 0x41}, 0, "LDH ($FF41),A", 2, FlowSeq},
		{[]byte{0xF0, 0xFD}, 0, "LDH A,($FFFD)", 2, FlowSeq},
		{[]byte{0xE2}, 0, "LDH (C),A", 1, FlowSeq},
		{[]byte{0xF2}, 0, "LDH A,(C)", 1, FlowSeq},
		{[]byte{0xEA, 0x00, 0x20}, 0, "LD ($2000),A", 3, FlowSeq},
		{[]byte{0xFA, 0x34, 0x12}, 0, "LD A,($1234)", 3, FlowSeq},
		{[]byte{0xE8, 0x05}, 0, "ADD SP,+5", 2, FlowSeq},
		{[]byte{0xE8, 0xFB}, 0, "ADD SP,-5", 2, FlowSeq},
		{[]byte{0xF8, 0x08}, 0, "LD HL,SP+8", 2, FlowSeq},
		{[]byte{0xF9}, 0, "LD SP,HL", 1, FlowSeq},
		{[]byte{0xF3}, 0, "DI", 1, FlowSeq},
		{[]byte{0xFB}, 0, "EI", 1, FlowSeq},
		// CB page (note SWAP at index 6, where the Z80 has SLL)
		{[]byte{0xCB, 0x00}, 0, "RLC B", 2, FlowSeq},
		{[]byte{0xCB, 0x30}, 0, "SWAP B", 2, FlowSeq},
		{[]byte{0xCB, 0x37}, 0, "SWAP A", 2, FlowSeq},
		{[]byte{0xCB, 0x46}, 0, "BIT 0,(HL)", 2, FlowSeq},
		{[]byte{0xCB, 0xFE}, 0, "SET 7,(HL)", 2, FlowSeq},
		{[]byte{0xCB, 0x86}, 0, "RES 0,(HL)", 2, FlowSeq},
		// illegal opcodes -> .byte / FlowStop
		{[]byte{0xD3}, 0, ".byte $D3", 1, FlowStop},
		{[]byte{0xDB}, 0, ".byte $DB", 1, FlowStop},
		{[]byte{0xDD}, 0, ".byte $DD", 1, FlowStop},
		{[]byte{0xE3}, 0, ".byte $E3", 1, FlowStop},
		{[]byte{0xE4}, 0, ".byte $E4", 1, FlowStop},
		{[]byte{0xEB}, 0, ".byte $EB", 1, FlowStop},
		{[]byte{0xEC}, 0, ".byte $EC", 1, FlowStop},
		{[]byte{0xED}, 0, ".byte $ED", 1, FlowStop},
		{[]byte{0xF4}, 0, ".byte $F4", 1, FlowStop},
		{[]byte{0xFC}, 0, ".byte $FC", 1, FlowStop},
		{[]byte{0xFD}, 0, ".byte $FD", 1, FlowStop},
		// truncated read
		{[]byte{0x21, 0x00}, 0, ".byte $21", 1, FlowStop},
	}
	for _, c := range cases {
		got := Decode(c.bytes, c.addr)
		if got.Text != c.text || got.Len != c.len || got.Flow != c.flow {
			t.Errorf("Decode(%X) = {%q len=%d flow=%d}, want {%q len=%d flow=%d}",
				c.bytes, got.Text, got.Len, got.Flow, c.text, c.len, c.flow)
		}
	}
}

// TestVectors decodes Super Mario Land's RST $20 jump-table dispatcher and timer ISR
// from the real bytes documented in Part I, end to end.
func TestVectors(t *testing.T) {
	// RST $20 dispatcher at $0020.
	disp := []byte{0x87, 0xE1, 0x5F, 0x16, 0x00, 0x19, 0x5E, 0x23, 0x56, 0xD5, 0xE1, 0xE9}
	wantDisp := []string{
		"ADD A,A", "POP HL", "LD E,A", "LD D,$00", "ADD HL,DE",
		"LD E,(HL)", "INC HL", "LD D,(HL)", "PUSH DE", "POP HL", "JP (HL)",
	}
	checkSeq(t, "RST20", disp, 0x0020, wantDisp)

	// Timer ISR at $0050: page bank 3, call the sound engine, restore the bank.
	timer := []byte{0xF5, 0x3E, 0x03, 0xEA, 0x00, 0x20, 0xCD, 0xF0, 0x7F, 0xF0, 0xFD, 0xEA, 0x00, 0x20, 0xF1, 0xD9}
	wantTimer := []string{
		"PUSH AF", "LD A,$03", "LD ($2000),A", "CALL $7FF0",
		"LDH A,($FFFD)", "LD ($2000),A", "POP AF", "RETI",
	}
	checkSeq(t, "timerISR", timer, 0x0050, wantTimer)
}

func checkSeq(t *testing.T, name string, code []byte, base uint16, want []string) {
	t.Helper()
	pc := 0
	for i, w := range want {
		in := Decode(code[pc:], base+uint16(pc))
		if in.Text != w {
			t.Errorf("%s[%d] @ $%04X = %q, want %q", name, i, base+uint16(pc), in.Text, w)
		}
		pc += in.Len
	}
}
