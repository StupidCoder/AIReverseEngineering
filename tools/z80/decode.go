package z80

import "fmt"

// Flow classifies how an instruction affects control flow — the information a
// recursive-descent disassembler needs to follow every reachable path. It mirrors
// the mos6502/m68k packages so codetracez80 can drive it the same way.
type Flow int

const (
	FlowSeq     Flow = iota // continues to the next instruction
	FlowBranch              // conditional branch (JR cc / JP cc / DJNZ): continues AND may go to Target
	FlowJump                // unconditional JR / JP nn: goes to Target, no fall-through
	FlowCall                // CALL / CALL cc / RST: goes to Target, normally returns after it
	FlowReturn              // RET / RETI / RETN: path ends (RET cc continues, see Decode)
	FlowIndJump             // JP (HL)/(IX)/(IY): target not statically known
	FlowStop                // an undefined opcode or a truncated read: treat as data/stop
)

// Inst is one decoded instruction.
type Inst struct {
	Addr      uint16
	Len       int
	Mnem      string // the bare mnemonic ("LD", "JP", …)
	Text      string // formatted "MNEM operands"
	Flow      Flow
	Target    uint16 // branch/call/jump destination (when HasTarget)
	HasTarget bool
}

// register / condition / operation tables, indexed by the opcode bit-fields below.
var (
	rTab    = [8]string{"B", "C", "D", "E", "H", "L", "(HL)", "A"}
	rpTab   = [4]string{"BC", "DE", "HL", "SP"}
	rp2Tab  = [4]string{"BC", "DE", "HL", "AF"}
	ccTab   = [8]string{"NZ", "Z", "NC", "C", "PO", "PE", "P", "M"}
	aluTab  = [8]string{"ADD A,", "ADC A,", "SUB ", "SBC A,", "AND ", "XOR ", "OR ", "CP "}
	rotTab  = [8]string{"RLC", "RRC", "RL", "RR", "SLA", "SRA", "SLL", "SRL"}
	imTab   = [8]string{"0", "0", "1", "2", "0", "0", "1", "2"}
	accTab  = [8]string{"RLCA", "RRCA", "RLA", "RRA", "DAA", "CPL", "SCF", "CCF"}
	blkTab  = [4][4]string{ // ED x=2 block ops: [y-4][z]
		{"LDI", "CPI", "INI", "OUTI"},
		{"LDD", "CPD", "IND", "OUTD"},
		{"LDIR", "CPIR", "INIR", "OTIR"},
		{"LDDR", "CPDR", "INDR", "OTDR"},
	}
)

// Decode decodes the instruction at the start of code, which is loaded at address
// addr (code[0] is the opcode byte). A Z80 instruction is up to four bytes and may
// carry one of the CB/ED/DD/FD prefixes; an undefined opcode or a read past the end
// of code decodes as a one-byte ".byte" with FlowStop.
func Decode(code []byte, addr uint16) Inst {
	d := &dec{mem: code, addr: addr}
	in := d.run()
	if d.bad {
		b := byte(0)
		if len(code) > 0 {
			b = code[0]
		}
		return Inst{Addr: addr, Len: 1, Mnem: ".byte", Text: fmt.Sprintf(".byte $%02X", b), Flow: FlowStop}
	}
	in.Addr = addr
	in.Len = d.p
	return in
}

// dec carries the decode cursor (an offset into mem) and the active index-register
// substitution.
type dec struct {
	mem  []byte
	addr uint16 // address of mem[0]
	p    int    // read cursor, an offset into mem

	// ix is "" for HL, or "IX"/"IY" under a DD/FD prefix. When ixUsed is set the
	// instruction references (HL), so it becomes (ix+disp) and H/L stay H/L;
	// otherwise H/L become IXH/IXL (the undocumented halves).
	ix     string
	disp   int8
	ixUsed bool
	hasD   bool // disp has been read
	bad    bool
}

// next reads one byte and advances; out-of-range marks the instruction bad.
func (d *dec) next() byte {
	if d.p >= len(d.mem) {
		d.bad = true
		return 0
	}
	b := d.mem[d.p]
	d.p++
	return b
}

func (d *dec) imm8() byte  { return d.next() }
func (d *dec) imm16() uint16 {
	lo := uint16(d.next())
	hi := uint16(d.next())
	return lo | hi<<8
}

// readDisp reads the (IX+d)/(IY+d) displacement byte for the current main-page
// instruction (it follows the opcode). For DDCB the displacement is read earlier.
func (d *dec) readDisp() {
	if !d.hasD {
		d.disp = int8(d.next())
		d.hasD = true
	}
}

// reg formats the 8-bit register operand for index i (0..7), applying the active
// index-register substitution.
func (d *dec) reg(i int) string {
	if d.ix == "" || i == 6 && !d.ixUsed {
		return rTab[i]
	}
	switch i {
	case 6:
		d.readDisp()
		return fmt.Sprintf("(%s%+d)", d.ix, d.disp)
	case 4:
		if d.ixUsed {
			return "H"
		}
		return d.ix + "H"
	case 5:
		if d.ixUsed {
			return "L"
		}
		return d.ix + "L"
	}
	return rTab[i]
}

// rp / rp2 format a 16-bit register pair (HL becomes IX/IY under a prefix).
func (d *dec) rp(i int) string {
	if i == 2 && d.ix != "" {
		return d.ix
	}
	return rpTab[i]
}
func (d *dec) rp2(i int) string {
	if i == 2 && d.ix != "" {
		return d.ix
	}
	return rp2Tab[i]
}

func (d *dec) run() Inst {
	op := d.next()
	switch op {
	case 0xCB:
		return d.decodeCB(d.next())
	case 0xED:
		return d.decodeED(d.next())
	case 0xDD, 0xFD:
		d.ix = "IX"
		if op == 0xFD {
			d.ix = "IY"
		}
		nb := d.next()
		switch nb {
		case 0xCB:
			// DDCB: the displacement comes BEFORE the final opcode byte.
			d.disp, d.hasD = int8(d.next()), true
			d.ixUsed = true
			return d.decodeCB(d.next())
		case 0xDD, 0xFD, 0xED:
			// A prefix following a prefix: the first one acts as a 1-byte no-op
			// (NONI). Report it and let the next Decode handle the real opcode.
			d.p = 1
			return Inst{Mnem: "NOP*", Text: "NOP*", Flow: FlowSeq}
		default:
			return d.decodeMain(nb)
		}
	default:
		return d.decodeMain(op)
	}
}

// decodeMain decodes an unprefixed (or DD/FD-substituted) main-page opcode using
// the regular x/y/z/p/q bit-fields:
//
//	x = op[7:6]   y = op[5:3]   z = op[2:0]   p = y>>1   q = y&1
func (d *dec) decodeMain(op byte) Inst {
	x, y, z := op>>6, (op>>3)&7, op&7
	p, q := y>>1, y&1
	// Pre-scan whether this opcode references (HL) so reg() substitutes (ix+d) and
	// leaves H/L unsubstituted (the documented behaviour under DD/FD).
	d.ixUsed = mainUsesHL(x, y, z)

	mk := func(mn, text string) Inst { return Inst{Mnem: mn, Text: text, Flow: FlowSeq} }

	switch x {
	case 0:
		switch z {
		case 0:
			switch y {
			case 0:
				return mk("NOP", "NOP")
			case 1:
				return mk("EX", "EX AF,AF'")
			case 2:
				t := d.rel()
				return Inst{Mnem: "DJNZ", Text: fmt.Sprintf("DJNZ $%04X", t), Flow: FlowBranch, Target: t, HasTarget: true}
			case 3:
				t := d.rel()
				return Inst{Mnem: "JR", Text: fmt.Sprintf("JR $%04X", t), Flow: FlowJump, Target: t, HasTarget: true}
			default: // 4..7: JR cc,d
				t := d.rel()
				return Inst{Mnem: "JR", Text: fmt.Sprintf("JR %s,$%04X", ccTab[y-4], t), Flow: FlowBranch, Target: t, HasTarget: true}
			}
		case 1:
			if q == 0 {
				return mk("LD", fmt.Sprintf("LD %s,$%04X", d.rp(int(p)), d.imm16()))
			}
			return mk("ADD", fmt.Sprintf("ADD %s,%s", d.idx("HL"), d.rp(int(p))))
		case 2:
			switch {
			case q == 0 && p == 0:
				return mk("LD", "LD (BC),A")
			case q == 0 && p == 1:
				return mk("LD", "LD (DE),A")
			case q == 0 && p == 2:
				return mk("LD", fmt.Sprintf("LD ($%04X),%s", d.imm16(), d.idx("HL")))
			case q == 0 && p == 3:
				return mk("LD", fmt.Sprintf("LD ($%04X),A", d.imm16()))
			case q == 1 && p == 0:
				return mk("LD", "LD A,(BC)")
			case q == 1 && p == 1:
				return mk("LD", "LD A,(DE)")
			case q == 1 && p == 2:
				return mk("LD", fmt.Sprintf("LD %s,($%04X)", d.idx("HL"), d.imm16()))
			default: // q==1 p==3
				return mk("LD", fmt.Sprintf("LD A,($%04X)", d.imm16()))
			}
		case 3:
			if q == 0 {
				return mk("INC", "INC "+d.rp(int(p)))
			}
			return mk("DEC", "DEC "+d.rp(int(p)))
		case 4:
			return mk("INC", "INC "+d.reg(int(y)))
		case 5:
			return mk("DEC", "DEC "+d.reg(int(y)))
		case 6:
			dst := d.reg(int(y)) // may read the (ix+d) displacement first
			return mk("LD", fmt.Sprintf("LD %s,$%02X", dst, d.imm8()))
		default: // z==7
			return mk(accTab[y], accTab[y])
		}
	case 1:
		if z == 6 && y == 6 {
			return mk("HALT", "HALT")
		}
		// LD r[y],r[z] — under DD/FD only ONE side becomes (ix+d); the other stays.
		dst := d.reg(int(y))
		src := d.reg(int(z))
		return mk("LD", fmt.Sprintf("LD %s,%s", dst, src))
	case 2:
		return mk(aluMnem(y), aluTab[y]+d.reg(int(z)))
	default: // x==3
		switch z {
		case 0:
			return Inst{Mnem: "RET", Text: "RET " + ccTab[y], Flow: FlowSeq} // RET cc: keep tracing the fall-through
		case 1:
			if q == 0 {
				return mk("POP", "POP "+d.rp2(int(p)))
			}
			switch p {
			case 0:
				return Inst{Mnem: "RET", Text: "RET", Flow: FlowReturn}
			case 1:
				return mk("EXX", "EXX")
			case 2:
				return Inst{Mnem: "JP", Text: fmt.Sprintf("JP (%s)", d.idx("HL")), Flow: FlowIndJump}
			default: // p==3
				return mk("LD", fmt.Sprintf("LD SP,%s", d.idx("HL")))
			}
		case 2:
			t := d.imm16()
			return Inst{Mnem: "JP", Text: fmt.Sprintf("JP %s,$%04X", ccTab[y], t), Flow: FlowBranch, Target: t, HasTarget: true}
		case 3:
			switch y {
			case 0:
				t := d.imm16()
				return Inst{Mnem: "JP", Text: fmt.Sprintf("JP $%04X", t), Flow: FlowJump, Target: t, HasTarget: true}
			case 2:
				return mk("OUT", fmt.Sprintf("OUT ($%02X),A", d.imm8()))
			case 3:
				return mk("IN", fmt.Sprintf("IN A,($%02X)", d.imm8()))
			case 4:
				return mk("EX", fmt.Sprintf("EX (SP),%s", d.idx("HL")))
			case 5:
				return mk("EX", "EX DE,HL")
			case 6:
				return mk("DI", "DI")
			case 7:
				return mk("EI", "EI")
			default: // y==1 is the CB prefix, handled in run(); shouldn't reach here
				d.bad = true
				return Inst{}
			}
		case 4:
			t := d.imm16()
			return Inst{Mnem: "CALL", Text: fmt.Sprintf("CALL %s,$%04X", ccTab[y], t), Flow: FlowCall, Target: t, HasTarget: true}
		case 5:
			if q == 0 {
				return mk("PUSH", "PUSH "+d.rp2(int(p)))
			}
			switch p {
			case 0:
				t := d.imm16()
				return Inst{Mnem: "CALL", Text: fmt.Sprintf("CALL $%04X", t), Flow: FlowCall, Target: t, HasTarget: true}
			default: // p==1/2/3 are DD/ED/FD prefixes, handled in run()
				d.bad = true
				return Inst{}
			}
		case 6:
			return mk(aluMnem(y), aluTab[y]+fmt.Sprintf("$%02X", d.imm8()))
		default: // z==7: RST y*8
			t := uint16(y) * 8
			return Inst{Mnem: "RST", Text: fmt.Sprintf("RST $%02X", t), Flow: FlowCall, Target: t, HasTarget: true}
		}
	}
}

// decodeCB decodes a CB-page (or DDCB/FDCB) opcode: rotate/shift, BIT, RES, SET.
func (d *dec) decodeCB(op byte) Inst {
	x, y, z := op>>6, (op>>3)&7, op&7
	// In (IX+d)/(IY+d) form the operand is the memory cell; the documented form
	// uses (ix+d), the undocumented one also copies to r[z] (shown as a comment).
	operand := rTab[z]
	if d.ix != "" {
		operand = fmt.Sprintf("(%s%+d)", d.ix, d.disp)
	}
	und := ""
	if d.ix != "" && z != 6 {
		und = " ; und: ,->" + rTab[z]
	}
	switch x {
	case 0:
		return Inst{Mnem: rotTab[y], Text: rotTab[y] + " " + operand + und, Flow: FlowSeq}
	case 1:
		return Inst{Mnem: "BIT", Text: fmt.Sprintf("BIT %d,%s", y, operand), Flow: FlowSeq}
	case 2:
		return Inst{Mnem: "RES", Text: fmt.Sprintf("RES %d,%s%s", y, operand, und), Flow: FlowSeq}
	default:
		return Inst{Mnem: "SET", Text: fmt.Sprintf("SET %d,%s%s", y, operand, und), Flow: FlowSeq}
	}
}

// decodeED decodes the ED page: block ops, 16-bit ADC/SBC, IN/OUT (C), LD (nn),rp,
// NEG, RETI/RETN, IM, RRD/RLD and the I/R register transfers.
func (d *dec) decodeED(op byte) Inst {
	x, y, z := op>>6, (op>>3)&7, op&7
	p, q := y>>1, y&1
	mk := func(mn, text string) Inst { return Inst{Mnem: mn, Text: text, Flow: FlowSeq} }
	if x == 2 {
		if z <= 3 && y >= 4 {
			m := blkTab[y-4][z]
			return mk(m, m)
		}
		return mk("NOP*", "NOP*") // ED x=2 holes are NONI no-ops
	}
	if x != 1 {
		return mk("NOP*", "NOP*") // ED x=0/3 are undefined
	}
	switch z {
	case 0:
		if y == 6 {
			return mk("IN", "IN (C)")
		}
		return mk("IN", fmt.Sprintf("IN %s,(C)", rTab[y]))
	case 1:
		if y == 6 {
			return mk("OUT", "OUT (C),0")
		}
		return mk("OUT", fmt.Sprintf("OUT (C),%s", rTab[y]))
	case 2:
		if q == 0 {
			return mk("SBC", "SBC HL,"+rpTab[p])
		}
		return mk("ADC", "ADC HL,"+rpTab[p])
	case 3:
		if q == 0 {
			return mk("LD", fmt.Sprintf("LD ($%04X),%s", d.imm16(), rpTab[p]))
		}
		return mk("LD", fmt.Sprintf("LD %s,($%04X)", rpTab[p], d.imm16()))
	case 4:
		return mk("NEG", "NEG")
	case 5:
		if y == 1 {
			return Inst{Mnem: "RETI", Text: "RETI", Flow: FlowReturn}
		}
		return Inst{Mnem: "RETN", Text: "RETN", Flow: FlowReturn}
	case 6:
		return mk("IM", "IM "+imTab[y])
	default: // z==7
		switch y {
		case 0:
			return mk("LD", "LD I,A")
		case 1:
			return mk("LD", "LD R,A")
		case 2:
			return mk("LD", "LD A,I")
		case 3:
			return mk("LD", "LD A,R")
		case 4:
			return mk("RRD", "RRD")
		case 5:
			return mk("RLD", "RLD")
		default:
			return mk("NOP*", "NOP*")
		}
	}
}

// --- small helpers ---------------------------------------------------------

// idx returns "HL" unless an index prefix is active, in which case "IX"/"IY".
func (d *dec) idx(hl string) string {
	if d.ix != "" {
		return d.ix
	}
	return hl
}

// rel reads a relative-branch displacement and returns the absolute target
// (address of the following instruction + signed displacement).
func (d *dec) rel() uint16 {
	off := int8(d.next())
	return uint16(int(d.addr) + d.p + int(off)) // p already points past the displacement byte
}

func aluMnem(y byte) string {
	switch y {
	case 0:
		return "ADD"
	case 1:
		return "ADC"
	case 2:
		return "SUB"
	case 3:
		return "SBC"
	case 4:
		return "AND"
	case 5:
		return "XOR"
	case 6:
		return "OR"
	default:
		return "CP"
	}
}

// mainUsesHL reports whether a main-page opcode references (HL) — needed so a
// DD/FD prefix turns it into (ix+d) and leaves H/L as H/L.
func mainUsesHL(x, y, z byte) bool {
	switch x {
	case 0:
		return (z == 4 || z == 5 || z == 6) && y == 6 // INC/DEC/LD (HL),n
	case 1:
		return z == 6 || y == 6 // LD r,(HL) / LD (HL),r (HALT excluded by caller)
	case 2:
		return z == 6 // ALU A,(HL)
	}
	return false
}
