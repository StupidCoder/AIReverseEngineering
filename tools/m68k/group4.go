package m68k

import (
	"fmt"
	"strings"
)

// eaFn and the small closures match the ones built inside Decode.
type eaFn func(mode, reg, size int) (text string, target uint32, known, isReg, ok bool)

// decodeGroup4 decodes the 68000 "line 4" miscellaneous opcodes: the unary
// memory operations (NEGX/CLR/NEG/NOT/NBCD/TST/TAS), MOVE to/from SR/CCR, the
// register housekeeping (SWAP/EXT/MOVEM/PEA/LEA/CHK), and the control block at
// $4E.. (TRAP/LINK/UNLK/RESET/NOP/STOP/RTE/RTS/RTR/JSR/JMP).
func decodeGroup4(op uint16, mode, reg, reg2, size int,
	ea eaFn, rd16 func() (uint16, bool), mk func(string, string) Inst, ill func() Inst) Inst {

	// LEA <ea>,An and CHK.W <ea>,Dn (precise opmode masks).
	if op&0xF1C0 == 0x41C0 {
		t, g, k, isReg, ok := ea(mode, reg, 2)
		if !ok || isReg {
			return ill()
		}
		in := mk("LEA", t+fmt.Sprintf(",a%d", reg2))
		in.Target, in.HasTarget = g, k
		return in
	}
	if op&0xF1C0 == 0x4180 {
		t, _, _, _, ok := ea(mode, reg, 1)
		if !ok {
			return ill()
		}
		return mk("CHK.W", t+fmt.Sprintf(",d%d", reg2))
	}

	// MOVE to/from the status and condition-code registers.
	switch op & 0xFFC0 {
	case 0x40C0:
		t, _, _, _, ok := ea(mode, reg, 1)
		if !ok {
			return ill()
		}
		return mk("MOVE", "sr,"+t)
	case 0x42C0:
		t, _, _, _, ok := ea(mode, reg, 1)
		if !ok {
			return ill()
		}
		return mk("MOVE", "ccr,"+t)
	case 0x44C0:
		t, _, _, _, ok := ea(mode, reg, 1)
		if !ok {
			return ill()
		}
		return mk("MOVE", t+",ccr")
	case 0x46C0:
		t, _, _, _, ok := ea(mode, reg, 1)
		if !ok {
			return ill()
		}
		return mk("MOVE", t+",sr")
	}

	// The $4E.. control block.
	if op&0xFF80 == 0x4E80 {
		// JSR / JMP <ea>.
		jmp := op&0x0040 != 0
		t, g, k, isReg, ok := ea(mode, reg, 0)
		if !ok || isReg {
			return ill()
		}
		name := "JSR"
		if jmp {
			name = "JMP"
		}
		in := mk(name, t)
		switch {
		case k && jmp:
			in.Flow, in.Target, in.HasTarget = FlowJump, g, true
		case k && !jmp:
			in.Flow, in.Target, in.HasTarget = FlowCall, g, true
		case jmp:
			in.Flow = FlowIndJump // JMP through a register/indexed EA
		default:
			in.Flow = FlowSeq // JSR through an unknown EA: continues after return
		}
		return in
	}
	if op&0xFFF0 == 0x4E40 { // TRAP #vector
		return mk("TRAP", fmt.Sprintf("#%d", op&0x000F))
	}
	if op&0xFFF8 == 0x4E50 { // LINK An,#disp
		d, ok := rd16()
		if !ok {
			return ill()
		}
		return mk("LINK", fmt.Sprintf("a%d,#%s", reg, hexDisp(int(int16(d)))))
	}
	if op&0xFFF8 == 0x4E58 { // UNLK An
		return mk("UNLK", fmt.Sprintf("a%d", reg))
	}
	if op&0xFFF0 == 0x4E60 { // MOVE An,USP / MOVE USP,An
		if op&0x0008 == 0 {
			return mk("MOVE", fmt.Sprintf("a%d,usp", reg))
		}
		return mk("MOVE", fmt.Sprintf("usp,a%d", reg))
	}
	switch op {
	case 0x4E70:
		return mk("RESET", "")
	case 0x4E71:
		return mk("NOP", "")
	case 0x4E72:
		d, ok := rd16()
		if !ok {
			return ill()
		}
		in := mk("STOP", fmt.Sprintf("#$%X", d))
		in.Flow = FlowStop
		return in
	case 0x4E73:
		in := mk("RTE", "")
		in.Flow = FlowReturn
		return in
	case 0x4E75:
		in := mk("RTS", "")
		in.Flow = FlowReturn
		return in
	case 0x4E76:
		return mk("TRAPV", "")
	case 0x4E77:
		in := mk("RTR", "")
		in.Flow = FlowReturn
		return in
	}

	// Unary memory operations and the $48/$4C register housekeeping.
	switch op & 0xFF00 {
	case 0x4000:
		return unary("NEGX", op, size, mode, reg, ea, mk, ill)
	case 0x4200:
		return unary("CLR", op, size, mode, reg, ea, mk, ill)
	case 0x4400:
		return unary("NEG", op, size, mode, reg, ea, mk, ill)
	case 0x4600:
		return unary("NOT", op, size, mode, reg, ea, mk, ill)
	case 0x4A00:
		if op == 0x4AFC {
			in := mk("ILLEGAL", "")
			in.Flow = FlowStop
			return in
		}
		if size == 3 { // TAS <ea> (byte)
			t, _, _, _, ok := ea(mode, reg, 0)
			if !ok {
				return ill()
			}
			return mk("TAS", t)
		}
		t, _, _, _, ok := ea(mode, reg, size)
		if !ok {
			return ill()
		}
		return mk("TST"+sizeSuffix(size), t)
	case 0x4800:
		switch {
		case op&0xFFC0 == 0x4800: // NBCD <ea> (byte)
			t, _, _, _, ok := ea(mode, reg, 0)
			if !ok {
				return ill()
			}
			return mk("NBCD", t)
		case op&0xFFF8 == 0x4840: // SWAP Dn
			return mk("SWAP", fmt.Sprintf("d%d", reg))
		case op&0xFFF8 == 0x4880: // EXT.W Dn
			return mk("EXT.W", fmt.Sprintf("d%d", reg))
		case op&0xFFF8 == 0x48C0: // EXT.L Dn
			return mk("EXT.L", fmt.Sprintf("d%d", reg))
		case op&0xFFC0 == 0x4840: // PEA <ea>
			t, g, k, isReg, ok := ea(mode, reg, 2)
			if !ok || isReg {
				return ill()
			}
			in := mk("PEA", t)
			in.Target, in.HasTarget = g, k
			return in
		default: // MOVEM regs,<ea> (register -> memory)
			return movem(op, false, mode, reg, ea, rd16, mk, ill)
		}
	case 0x4C00:
		if op&0xFB80 == 0x4C80 { // MOVEM <ea>,regs (memory -> register)
			return movem(op, true, mode, reg, ea, rd16, mk, ill)
		}
		return ill()
	}
	return ill()
}

func unary(name string, op uint16, size, mode, reg int, ea eaFn, mk func(string, string) Inst, ill func() Inst) Inst {
	if size == 3 {
		return ill()
	}
	t, _, _, _, ok := ea(mode, reg, size)
	if !ok {
		return ill()
	}
	return mk(name+sizeSuffix(size), t)
}

// movem decodes MOVEM in either direction. The register mask word precedes the
// effective address; in the -(An) predecrement form the mask bit order is
// reversed.
func movem(op uint16, toReg bool, mode, reg int, ea eaFn, rd16 func() (uint16, bool), mk func(string, string) Inst, ill func() Inst) Inst {
	mask, ok := rd16()
	if !ok {
		return ill()
	}
	sz := ".w"
	if op&0x0040 != 0 {
		sz = ".l"
	}
	t, _, _, _, ok := ea(mode, reg, 2)
	if !ok {
		return ill()
	}
	list := regList(mask, mode == 4) // predecrement reverses the bit order
	if toReg {
		return mk("MOVEM"+sz, t+","+list)
	}
	return mk("MOVEM"+sz, list+","+t)
}

// regList renders a MOVEM register mask as a compact d0-d7/a0-a6 list. With
// reversed set (predecrement destination), bit 0 is a7 down to bit 15 = d0.
func regList(mask uint16, reversed bool) string {
	var present [16]bool
	for i := 0; i < 16; i++ {
		bit := i
		if reversed {
			bit = 15 - i
		}
		present[i] = mask&(1<<uint(bit)) != 0
	}
	regName := func(i int) string {
		if i < 8 {
			return fmt.Sprintf("d%d", i)
		}
		return fmt.Sprintf("a%d", i-8)
	}
	var parts []string
	for i := 0; i < 16; {
		if !present[i] {
			i++
			continue
		}
		j := i
		// extend a run, but do not let it cross the d7 -> a0 boundary
		for j+1 < 16 && present[j+1] && !(j+1 == 8) {
			j++
		}
		if j == i {
			parts = append(parts, regName(i))
		} else {
			parts = append(parts, regName(i)+"-"+regName(j))
		}
		i = j + 1
	}
	if len(parts) == 0 {
		return "0"
	}
	return strings.Join(parts, "/")
}
