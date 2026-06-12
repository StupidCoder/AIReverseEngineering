package m68k

import "testing"

func bytes(ws ...uint16) []byte {
	b := make([]byte, 0, len(ws)*2)
	for _, w := range ws {
		b = append(b, byte(w>>8), byte(w))
	}
	return b
}

func TestDecode(t *testing.T) {
	cases := []struct {
		name   string
		code   []byte
		addr   uint32
		text   string
		length int
		flow   Flow
		target uint32 // checked when flow has a known target
		hasTgt bool
	}{
		{"nop", bytes(0x4E71), 0, "NOP", 2, FlowSeq, 0, false},
		{"rts", bytes(0x4E75), 0, "RTS", 2, FlowReturn, 0, false},
		{"rte", bytes(0x4E73), 0, "RTE", 2, FlowReturn, 0, false},
		{"rtr", bytes(0x4E77), 0, "RTR", 2, FlowReturn, 0, false},
		{"illegal", bytes(0x4AFC), 0, "ILLEGAL", 2, FlowStop, 0, false},
		{"reset", bytes(0x4E70), 0, "RESET", 2, FlowSeq, 0, false},
		{"trapv", bytes(0x4E76), 0, "TRAPV", 2, FlowSeq, 0, false},
		{"trap15", bytes(0x4E4F), 0, "TRAP #15", 2, FlowSeq, 0, false},
		{"stop", bytes(0x4E72, 0x2700), 0, "STOP #$2700", 4, FlowStop, 0, false},
		{"moveq", bytes(0x7001), 0, "MOVEQ #$1,d0", 2, FlowSeq, 0, false},
		{"moveq neg", bytes(0x76FF), 0, "MOVEQ #$FF,d3", 2, FlowSeq, 0, false},
		{"move.l dd", bytes(0x2200), 0, "MOVE.l d0,d1", 2, FlowSeq, 0, false},
		{"move.w imm", bytes(0x303C, 0x1234), 0, "MOVE.w #$1234,d0", 4, FlowSeq, 0, false},
		{"movea.l", bytes(0x2248), 0, "MOVEA.l a0,a1", 2, FlowSeq, 0, false},
		{"lea pc", bytes(0x41FA, 0x0004), 0x1000, "LEA $001006(pc),a0", 4, FlowSeq, 0x1006, true},
		{"jmp (a0)", bytes(0x4ED0), 0, "JMP (a0)", 2, FlowIndJump, 0, false},
		{"jmp abs.l", bytes(0x4EF9, 0x1234, 0x5678), 0, "JMP $12345678.l", 6, FlowJump, 0x12345678, true},
		{"jsr abs.l", bytes(0x4EB9, 0x0000, 0x1000), 0, "JSR $1000.l", 6, FlowCall, 0x1000, true},
		{"bra.s self", bytes(0x60FE), 0x2000, "BRA $002000", 2, FlowJump, 0x2000, true},
		{"bsr.w", bytes(0x6100, 0x0010), 0x2000, "BSR $002012", 4, FlowCall, 0x2012, true},
		{"beq.s", bytes(0x6706), 0x3000, "BEQ $003008", 2, FlowBranch, 0x3008, true},
		{"dbra", bytes(0x51C8, 0xFFFC), 0x4000, "DBRA d0,$003FFE", 4, FlowBranch, 0x3FFE, true},
		{"seq", bytes(0x57C0), 0, "SEQ d0", 2, FlowSeq, 0, false},
		{"addq.l a0", bytes(0x5288), 0, "ADDQ.l #1,a0", 2, FlowSeq, 0, false},
		{"clr.w d0", bytes(0x4240), 0, "CLR.w d0", 2, FlowSeq, 0, false},
		{"tst.b d1", bytes(0x4A01), 0, "TST.b d1", 2, FlowSeq, 0, false},
		{"swap", bytes(0x4840), 0, "SWAP d0", 2, FlowSeq, 0, false},
		{"ext.l", bytes(0x48C0), 0, "EXT.L d0", 2, FlowSeq, 0, false},
		{"pea", bytes(0x4850), 0, "PEA (a0)", 2, FlowSeq, 0, false},
		{"movem push", bytes(0x48E7, 0x3002), 0, "MOVEM.l d2-d3/a6,-(a7)", 4, FlowSeq, 0, false},
		{"add.w", bytes(0xD240), 0, "ADD.w d0,d1", 2, FlowSeq, 0, false},
		{"adda.l", bytes(0xD3C8), 0, "ADDA.l a0,a1", 2, FlowSeq, 0, false},
		{"cmp.l", bytes(0xB098), 0, "CMP.l (a0)+,d0", 2, FlowSeq, 0, false},
		{"andi.w", bytes(0x0240, 0x00FF), 0, "ANDI.w #$FF,d0", 4, FlowSeq, 0, false},
		{"ori ccr", bytes(0x003C, 0x0012), 0, "ORI #$12,CCR", 4, FlowSeq, 0, false},
		{"lsl.w", bytes(0xE348), 0, "LSL.w #1,d0", 2, FlowSeq, 0, false},
		{"link", bytes(0x4E56, 0xFFFC), 0, "LINK a6,#-$4", 4, FlowSeq, 0, false},
		{"unlk", bytes(0x4E5E), 0, "UNLK a6", 2, FlowSeq, 0, false},
		{"btst dyn", bytes(0x0100), 0, "BTST.l d0,d0", 2, FlowSeq, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := Decode(c.code, c.addr)
			if in.Text != c.text {
				t.Errorf("text = %q, want %q", in.Text, c.text)
			}
			if in.Len != c.length {
				t.Errorf("len = %d, want %d", in.Len, c.length)
			}
			if in.Flow != c.flow {
				t.Errorf("flow = %d, want %d", in.Flow, c.flow)
			}
			if c.hasTgt && (!in.HasTarget || in.Target != c.target) {
				t.Errorf("target = $%X (has=%v), want $%X", in.Target, in.HasTarget, c.target)
			}
		})
	}
}

// TestLinearWalkStaysAligned checks that a stream of mixed-length instructions
// decodes back to the same byte boundaries (no desync).
func TestLinearWalkStaysAligned(t *testing.T) {
	code := bytes(
		0x303C, 0x1234, // MOVE.W #$1234,d0   (4)
		0x4EB9, 0x0000, 0x1000, // JSR $1000.l (6)
		0x4E71, // NOP                          (2)
		0x4E75, // RTS                          (2)
	)
	lines := Disassemble(code, 0x400)
	want := 4
	if len(lines) != want {
		t.Fatalf("got %d lines, want %d:\n%v", len(lines), want, lines)
	}
}
