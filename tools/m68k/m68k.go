// Package m68k is a table/pattern-driven Motorola 68000 disassembler, the
// 16/32-bit counterpart to the mos6502 package. It provides the same surface:
// a Decode function returning one fully-classified instruction (length, text,
// and a Flow category for recursive-descent tracing) and a Disassemble helper
// that renders a blob line by line.
//
// The 68000 has 16-bit, word-aligned opcodes and big-endian operands, so an
// instruction is 2 to 10 bytes: the opcode word, optional instruction-immediate
// words, then the extension words of its effective-address operand(s). Decode
// reads exactly as many words as each form needs, so Inst.Len is always right
// and a linear or recursive walk stays aligned. Anything not recognised (line-A
// and line-F words, illegal encodings, embedded data) decodes as a 2-byte
// ".dc.w", which keeps word alignment.
package m68k

import "fmt"

// Flow classifies how an instruction affects control flow — the information a
// recursive-descent disassembler needs to follow every reachable path. It
// mirrors the mos6502 Flow categories.
type Flow int

const (
	FlowSeq     Flow = iota // continues to the next instruction
	FlowBranch              // conditional branch (Bcc/DBcc): continues AND may go to Target
	FlowJump                // unconditional BRA/JMP to a known Target, no fall-through
	FlowCall                // BSR/JSR to a known Target, normally returns after it
	FlowReturn              // RTS/RTE/RTR: path ends
	FlowIndJump             // JMP through a register/indirect EA: target not statically known
	FlowStop                // ILLEGAL/STOP/line-A/line-F or embedded data: treat as a stop
)

// Inst is one decoded instruction.
type Inst struct {
	Addr      uint32
	Len       int
	Mnem      string // mnemonic with size suffix, e.g. "MOVE.L"
	Text      string // formatted "MNEM operands"
	Flow      Flow
	Target    uint32 // branch/jump/call destination, or absolute operand address, when known
	HasTarget bool   // Target is meaningful
}

var condNames = [16]string{"T", "F", "HI", "LS", "CC", "CS", "NE", "EQ", "VC", "VS", "PL", "MI", "GE", "LT", "GT", "LE"}

func be16(b []byte, off int) uint16 { return uint16(b[off])<<8 | uint16(b[off+1]) }
func be32(b []byte, off int) uint32 {
	return uint32(b[off])<<24 | uint32(b[off+1])<<16 | uint32(b[off+2])<<8 | uint32(b[off+3])
}

func sizeSuffix(s int) string {
	switch s {
	case 0:
		return ".b"
	case 1:
		return ".w"
	case 2:
		return ".l"
	}
	return ""
}

// hexDisp formats a signed displacement as $1A or -$4.
func hexDisp(v int) string {
	if v < 0 {
		return fmt.Sprintf("-$%X", -v)
	}
	return fmt.Sprintf("$%X", v)
}

// briefIdx formats a brief extension word (d8, index register) for the (d8,An,Xn)
// and (d8,PC,Xn) addressing modes; base is "a3" or "pc".
func briefIdx(ext uint16, base string) string {
	d8 := int(int8(byte(ext)))
	rc := 'd'
	if ext&0x8000 != 0 {
		rc = 'a'
	}
	idx := (ext >> 12) & 7
	sz := 'w'
	if ext&0x0800 != 0 {
		sz = 'l'
	}
	return fmt.Sprintf("%s(%s,%c%d.%c)", hexDisp(d8), base, rc, idx, sz)
}

// Decode decodes the instruction whose first byte is code[0] and whose absolute
// address is addr (used for Inst.Addr and PC-relative target resolution).
func Decode(code []byte, addr uint32) Inst {
	if len(code) < 2 {
		return Inst{Addr: addr, Len: len(code), Mnem: ".dc.b", Text: ".dc.b ; truncated", Flow: FlowStop}
	}
	op := be16(code, 0)
	n := 2 // bytes consumed so far (opcode word)

	ill := func() Inst {
		return Inst{Addr: addr, Len: 2, Mnem: ".dc.w", Text: fmt.Sprintf(".dc.w $%04X", op), Flow: FlowStop}
	}
	rd16 := func() (uint16, bool) {
		if n+2 > len(code) {
			return 0, false
		}
		v := be16(code, n)
		n += 2
		return v, true
	}
	rd32 := func() (uint32, bool) {
		if n+4 > len(code) {
			return 0, false
		}
		v := be32(code, n)
		n += 4
		return v, true
	}

	// ea decodes an effective address (mode,reg) of the given size and returns
	// its assembly text, a statically-known control target (abs / PC-relative
	// without index), whether the target is known, and whether the EA is a data
	// or address register direct. ok is false if extension words run off the end.
	ea := func(mode, reg, size int) (text string, target uint32, known, isReg, ok bool) {
		switch mode {
		case 0:
			return fmt.Sprintf("d%d", reg), 0, false, true, true
		case 1:
			return fmt.Sprintf("a%d", reg), 0, false, true, true
		case 2:
			return fmt.Sprintf("(a%d)", reg), 0, false, false, true
		case 3:
			return fmt.Sprintf("(a%d)+", reg), 0, false, false, true
		case 4:
			return fmt.Sprintf("-(a%d)", reg), 0, false, false, true
		case 5:
			d, k := rd16()
			if !k {
				return "", 0, false, false, false
			}
			return fmt.Sprintf("%s(a%d)", hexDisp(int(int16(d))), reg), 0, false, false, true
		case 6:
			e, k := rd16()
			if !k {
				return "", 0, false, false, false
			}
			return briefIdx(e, fmt.Sprintf("a%d", reg)), 0, false, false, true
		case 7:
			switch reg {
			case 0: // (xxx).W
				d, k := rd16()
				if !k {
					return "", 0, false, false, false
				}
				return fmt.Sprintf("$%X.w", d), uint32(int32(int16(d))), true, false, true
			case 1: // (xxx).L
				d, k := rd32()
				if !k {
					return "", 0, false, false, false
				}
				return fmt.Sprintf("$%X.l", d), d, true, false, true
			case 2: // (d16,PC)
				base := addr + uint32(n)
				d, k := rd16()
				if !k {
					return "", 0, false, false, false
				}
				t := base + uint32(int32(int16(d)))
				return fmt.Sprintf("$%06X(pc)", t), t, true, false, true
			case 3: // (d8,PC,Xn)
				e, k := rd16()
				if !k {
					return "", 0, false, false, false
				}
				return briefIdx(e, "pc"), 0, false, false, true
			case 4: // #imm
				if size == 2 {
					d, k := rd32()
					if !k {
						return "", 0, false, false, false
					}
					return fmt.Sprintf("#$%X", d), 0, false, false, true
				}
				d, k := rd16()
				if !k {
					return "", 0, false, false, false
				}
				if size == 0 {
					d &= 0xFF
				}
				return fmt.Sprintf("#$%X", d), 0, false, false, true
			}
		}
		return "", 0, false, false, false
	}

	mode := int(op>>3) & 7
	reg := int(op) & 7
	reg2 := int(op>>9) & 7
	size := int(op>>6) & 3

	mk := func(mnem, oper string) Inst {
		in := Inst{Addr: addr, Len: n, Mnem: mnem, Flow: FlowSeq}
		if oper != "" {
			in.Text = mnem + " " + oper
		} else {
			in.Text = mnem
		}
		return in
	}

	switch op >> 12 {

	case 0x0: // immediates, bit operations, MOVEP
		if op&0x0100 != 0 { // dynamic BTST/BCHG/BCLR/BSET, or MOVEP
			if mode == 1 { // MOVEP Dx,(d16,Ay) / (d16,Ay),Dx
				d, k := rd16()
				if !k {
					return ill()
				}
				sz := ".w"
				if op&0x0040 != 0 {
					sz = ".l"
				}
				mem := fmt.Sprintf("%s(a%d)", hexDisp(int(int16(d))), reg)
				if op&0x0080 != 0 {
					return mk("MOVEP"+sz, fmt.Sprintf("d%d,%s", reg2, mem))
				}
				return mk("MOVEP"+sz, fmt.Sprintf("%s,d%d", mem, reg2))
			}
			bit := []string{"BTST", "BCHG", "BCLR", "BSET"}[size]
			sz := ".l"
			if mode != 0 {
				sz = ".b"
			}
			t, _, _, _, ok := ea(mode, reg, 0)
			if !ok {
				return ill()
			}
			return mk(bit+sz, fmt.Sprintf("d%d,%s", reg2, t))
		}
		// static immediate group
		grp := map[int]string{0: "ORI", 1: "ANDI", 2: "SUBI", 3: "ADDI", 5: "EORI", 6: "CMPI"}
		if name, isImm := grp[reg2]; isImm && size != 3 {
			// ORI/ANDI/EORI to CCR (#x,CCR) or SR (#x,SR): EA field = 0x3C
			if (name == "ORI" || name == "ANDI" || name == "EORI") && op&0x3F == 0x3C {
				d, k := rd16()
				if !k {
					return ill()
				}
				dst := "CCR"
				if size == 1 {
					dst = "SR"
				}
				return mk(name, fmt.Sprintf("#$%X,%s", d&0xFF, dst))
			}
			var imm uint32
			if size == 2 {
				v, k := rd32()
				if !k {
					return ill()
				}
				imm = v
			} else {
				v, k := rd16()
				if !k {
					return ill()
				}
				imm = uint32(v)
				if size == 0 {
					imm &= 0xFF
				}
			}
			t, _, _, _, ok := ea(mode, reg, size)
			if !ok {
				return ill()
			}
			return mk(name+sizeSuffix(size), fmt.Sprintf("#$%X,%s", imm, t))
		}
		if reg2 == 4 { // static BTST/BCHG/BCLR/BSET #bit,<ea>
			b, k := rd16()
			if !k {
				return ill()
			}
			bit := []string{"BTST", "BCHG", "BCLR", "BSET"}[size]
			sz := ".l"
			if mode != 0 {
				sz = ".b"
			}
			t, _, _, _, ok := ea(mode, reg, 0)
			if !ok {
				return ill()
			}
			return mk(bit+sz, fmt.Sprintf("#$%X,%s", b&0xFF, t))
		}
		return ill()

	case 0x1, 0x2, 0x3: // MOVE / MOVEA
		sz := map[uint16]int{1: 0, 3: 1, 2: 2}[op>>12] // B/W/L per MOVE's own size field
		src, stgt, sknown, _, ok := ea(mode, reg, sz)
		if !ok {
			return ill()
		}
		dmode := int(op>>6) & 7
		dreg := reg2
		if dmode == 1 { // MOVEA
			dst, _, _, _, ok := ea(1, dreg, sz)
			if !ok {
				return ill()
			}
			in := mk("MOVEA"+sizeSuffix(sz), src+","+dst)
			in.Target, in.HasTarget = stgt, sknown
			return in
		}
		dst, _, _, _, ok := ea(dmode, dreg, sz)
		if !ok {
			return ill()
		}
		in := mk("MOVE"+sizeSuffix(sz), src+","+dst)
		in.Target, in.HasTarget = stgt, sknown
		return in

	case 0x4: // miscellaneous
		return decodeGroup4(op, mode, reg, reg2, size, ea, rd16, mk, ill)

	case 0x5: // ADDQ/SUBQ, Scc, DBcc
		if size == 3 {
			cc := int(op>>8) & 0xF
			if mode == 1 { // DBcc Dn,disp
				base := addr + uint32(n)
				d, k := rd16()
				if !k {
					return ill()
				}
				name := "DB" + condNames[cc]
				if cc == 1 {
					name = "DBRA"
				}
				t := base + uint32(int32(int16(d)))
				in := mk(name, fmt.Sprintf("d%d,$%06X", reg, t))
				in.Flow, in.Target, in.HasTarget = FlowBranch, t, true
				return in
			}
			t, _, _, _, ok := ea(mode, reg, 0)
			if !ok {
				return ill()
			}
			return mk("S"+condNames[cc], t)
		}
		data := reg2
		if data == 0 {
			data = 8
		}
		name := "ADDQ"
		if op&0x0100 != 0 {
			name = "SUBQ"
		}
		t, _, _, _, ok := ea(mode, reg, size)
		if !ok {
			return ill()
		}
		return mk(name+sizeSuffix(size), fmt.Sprintf("#%d,%s", data, t))

	case 0x6: // Bcc / BRA / BSR
		cc := int(op>>8) & 0xF
		disp := int(int8(byte(op)))
		base := addr + 2
		var length int
		if byte(op) == 0x00 {
			d, k := rd16()
			if !k {
				return ill()
			}
			disp = int(int16(d))
			length = 4
		} else {
			length = 2
		}
		n = length
		t := base + uint32(disp)
		switch cc {
		case 0:
			in := mk("BRA", fmt.Sprintf("$%06X", t))
			in.Flow, in.Target, in.HasTarget = FlowJump, t, true
			return in
		case 1:
			in := mk("BSR", fmt.Sprintf("$%06X", t))
			in.Flow, in.Target, in.HasTarget = FlowCall, t, true
			return in
		default:
			in := mk("B"+condNames[cc], fmt.Sprintf("$%06X", t))
			in.Flow, in.Target, in.HasTarget = FlowBranch, t, true
			return in
		}

	case 0x7: // MOVEQ
		if op&0x0100 != 0 {
			return ill()
		}
		return mk("MOVEQ", fmt.Sprintf("#$%X,d%d", byte(op), reg2))

	case 0x8: // OR / DIVU / DIVS / SBCD
		if mode == 6 && size == 3 { // unused; fall through to DIV check below
		}
		if (op>>6)&7 == 3 { // DIVU.W <ea>,Dn
			t, _, _, _, ok := ea(mode, reg, 1)
			if !ok {
				return ill()
			}
			return mk("DIVU.W", t+fmt.Sprintf(",d%d", reg2))
		}
		if (op>>6)&7 == 7 { // DIVS.W <ea>,Dn
			t, _, _, _, ok := ea(mode, reg, 1)
			if !ok {
				return ill()
			}
			return mk("DIVS.W", t+fmt.Sprintf(",d%d", reg2))
		}
		if op&0x01F0 == 0x0100 { // SBCD
			return bcdReg("SBCD", op, reg, reg2, mk)
		}
		return dataAlu("OR", op, mode, reg, reg2, size, ea, mk, ill)

	case 0x9: // SUB / SUBA / SUBX
		if (op>>6)&7 == 3 || (op>>6)&7 == 7 { // SUBA.W/.L
			sz := 1
			if (op>>6)&7 == 7 {
				sz = 2
			}
			t, g, k, _, ok := ea(mode, reg, sz)
			if !ok {
				return ill()
			}
			in := mk("SUBA"+sizeSuffix(sz), t+fmt.Sprintf(",a%d", reg2))
			in.Target, in.HasTarget = g, k
			return in
		}
		if op&0x0130 == 0x0100 { // SUBX
			return xReg("SUBX", op, reg, reg2, size, mk)
		}
		return dataAlu("SUB", op, mode, reg, reg2, size, ea, mk, ill)

	case 0xB: // CMP / CMPA / CMPM / EOR
		if (op>>6)&7 == 3 || (op>>6)&7 == 7 { // CMPA.W/.L
			sz := 1
			if (op>>6)&7 == 7 {
				sz = 2
			}
			t, g, k, _, ok := ea(mode, reg, sz)
			if !ok {
				return ill()
			}
			in := mk("CMPA"+sizeSuffix(sz), t+fmt.Sprintf(",a%d", reg2))
			in.Target, in.HasTarget = g, k
			return in
		}
		if op&0x0100 != 0 { // EOR (or CMPM)
			if mode == 1 { // CMPM (Ay)+,(Ax)+
				return mk("CMPM"+sizeSuffix(size), fmt.Sprintf("(a%d)+,(a%d)+", reg, reg2))
			}
			t, _, _, _, ok := ea(mode, reg, size)
			if !ok {
				return ill()
			}
			return mk("EOR"+sizeSuffix(size), fmt.Sprintf("d%d,%s", reg2, t))
		}
		t, g, k, _, ok := ea(mode, reg, size)
		if !ok {
			return ill()
		}
		in := mk("CMP"+sizeSuffix(size), t+fmt.Sprintf(",d%d", reg2))
		in.Target, in.HasTarget = g, k
		return in

	case 0xC: // AND / MULU / MULS / ABCD / EXG
		if (op>>6)&7 == 3 {
			t, _, _, _, ok := ea(mode, reg, 1)
			if !ok {
				return ill()
			}
			return mk("MULU.W", t+fmt.Sprintf(",d%d", reg2))
		}
		if (op>>6)&7 == 7 {
			t, _, _, _, ok := ea(mode, reg, 1)
			if !ok {
				return ill()
			}
			return mk("MULS.W", t+fmt.Sprintf(",d%d", reg2))
		}
		switch op & 0x01F8 {
		case 0x0140:
			return mk("EXG", fmt.Sprintf("d%d,d%d", reg2, reg))
		case 0x0148:
			return mk("EXG", fmt.Sprintf("a%d,a%d", reg2, reg))
		case 0x0188:
			return mk("EXG", fmt.Sprintf("d%d,a%d", reg2, reg))
		}
		if op&0x01F0 == 0x0100 { // ABCD
			return bcdReg("ABCD", op, reg, reg2, mk)
		}
		return dataAlu("AND", op, mode, reg, reg2, size, ea, mk, ill)

	case 0xD: // ADD / ADDA / ADDX
		if (op>>6)&7 == 3 || (op>>6)&7 == 7 {
			sz := 1
			if (op>>6)&7 == 7 {
				sz = 2
			}
			t, g, k, _, ok := ea(mode, reg, sz)
			if !ok {
				return ill()
			}
			in := mk("ADDA"+sizeSuffix(sz), t+fmt.Sprintf(",a%d", reg2))
			in.Target, in.HasTarget = g, k
			return in
		}
		if op&0x0130 == 0x0100 { // ADDX
			return xReg("ADDX", op, reg, reg2, size, mk)
		}
		return dataAlu("ADD", op, mode, reg, reg2, size, ea, mk, ill)

	case 0xE: // shifts and rotates
		types := [4]string{"AS", "LS", "ROX", "RO"}
		dir := "R"
		if op&0x0100 != 0 {
			dir = "L"
		}
		if size == 3 { // memory shift by 1
			ty := types[(op>>9)&3]
			t, _, _, _, ok := ea(mode, reg, 1)
			if !ok {
				return ill()
			}
			return mk(ty+dir+".w", t)
		}
		ty := types[(op>>3)&3]
		var cnt string
		if op&0x0020 != 0 {
			cnt = fmt.Sprintf("d%d", reg2)
		} else {
			c := reg2
			if c == 0 {
				c = 8
			}
			cnt = fmt.Sprintf("#%d", c)
		}
		return mk(ty+dir+sizeSuffix(size), fmt.Sprintf("%s,d%d", cnt, reg))
	}
	return ill()
}

// dataAlu decodes the register/EA forms of OR, AND, ADD, SUB, CMP, EOR.
func dataAlu(name string, op uint16, mode, reg, reg2, size int,
	ea func(int, int, int) (string, uint32, bool, bool, bool),
	mk func(string, string) Inst, ill func() Inst) Inst {
	t, g, k, _, ok := ea(mode, reg, size)
	if !ok {
		return ill()
	}
	in := mk(name+sizeSuffix(size), "")
	if op&0x0100 != 0 { // Dn op <ea> -> <ea>
		in = mk(name+sizeSuffix(size), fmt.Sprintf("d%d,%s", reg2, t))
	} else { // <ea> op Dn -> Dn
		in = mk(name+sizeSuffix(size), t+fmt.Sprintf(",d%d", reg2))
		in.Target, in.HasTarget = g, k
	}
	return in
}

// bcdReg decodes ABCD/SBCD (register or -(An) predecrement form).
func bcdReg(name string, op uint16, reg, reg2 int, mk func(string, string) Inst) Inst {
	if op&0x0008 != 0 {
		return mk(name, fmt.Sprintf("-(a%d),-(a%d)", reg, reg2))
	}
	return mk(name, fmt.Sprintf("d%d,d%d", reg, reg2))
}

// xReg decodes ADDX/SUBX (register or -(An) predecrement form).
func xReg(name string, op uint16, reg, reg2, size int, mk func(string, string) Inst) Inst {
	if op&0x0008 != 0 {
		return mk(name+sizeSuffix(size), fmt.Sprintf("-(a%d),-(a%d)", reg, reg2))
	}
	return mk(name+sizeSuffix(size), fmt.Sprintf("d%d,d%d", reg, reg2))
}
