package sm83

import "fmt"

// Flow classifies how an instruction affects control flow — the information a
// recursive-descent disassembler needs to follow every reachable path. It mirrors
// the mos6502/m68k/z80 packages so a codetrace driver can use it the same way.
type Flow int

const (
	FlowSeq     Flow = iota // continues to the next instruction
	FlowBranch              // conditional branch (JR cc / JP cc): continues AND may go to Target
	FlowJump                // unconditional JR / JP nn: goes to Target, no fall-through
	FlowCall                // CALL / CALL cc / RST: goes to Target, normally returns after it
	FlowReturn              // RET / RETI: path ends (RET cc continues, see Decode)
	FlowIndJump             // JP (HL): target not statically known
	FlowStop                // an undefined/illegal opcode or a truncated read: treat as data/stop
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
// The LR35902 uses the same regular x/y/z/p/q decoding as the Z80 for the bytes it
// shares, but only four branch conditions exist (the Z80's PO/PE/P/M slots were
// reclaimed for Game-Boy-specific opcodes), and the CB rotate group has SWAP where
// the Z80 has the undocumented SLL.
var (
	rTab   = [8]string{"B", "C", "D", "E", "H", "L", "(HL)", "A"}
	rpTab  = [4]string{"BC", "DE", "HL", "SP"}
	rp2Tab = [4]string{"BC", "DE", "HL", "AF"}
	ccTab  = [4]string{"NZ", "Z", "NC", "C"}
	aluTab = [8]string{"ADD A,", "ADC A,", "SUB ", "SBC A,", "AND ", "XOR ", "OR ", "CP "}
	rotTab = [8]string{"RLC", "RRC", "RL", "RR", "SLA", "SRA", "SWAP", "SRL"}
	accTab = [8]string{"RLCA", "RRCA", "RLA", "RRA", "DAA", "CPL", "SCF", "CCF"}
)

// Decode decodes the instruction at the start of code, which is loaded at address
// addr (code[0] is the opcode byte). An LR35902 instruction is one to three bytes,
// optionally carrying the single CB prefix (rotate/shift, BIT/RES/SET). An undefined
// (illegal) opcode or a read past the end of code decodes as a one-byte ".byte" with
// FlowStop. Note this is the Game Boy's Sharp LR35902, NOT a Z80: there are no
// DD/FD/ED prefixes and no IN/OUT ports, and the high-page LDH/LD (C) and 16-bit
// stack ops are Game-Boy-only.
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

// dec carries the decode cursor (an offset into mem).
type dec struct {
	mem  []byte
	addr uint16 // address of mem[0]
	p    int    // read cursor, an offset into mem
	bad  bool
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

func (d *dec) imm8() byte { return d.next() }
func (d *dec) imm16() uint16 {
	lo := uint16(d.next())
	hi := uint16(d.next())
	return lo | hi<<8
}

// rel reads a relative-branch displacement and returns the absolute target (address
// of the following instruction + signed displacement).
func (d *dec) rel() uint16 {
	off := int8(d.next())
	return uint16(int(d.addr) + d.p + int(off)) // p already points past the displacement byte
}

func (d *dec) run() Inst {
	op := d.next()
	if op == 0xCB {
		return d.decodeCB(d.next())
	}
	return d.decodeMain(op)
}

// decodeMain decodes a main-page opcode using the regular x/y/z/p/q bit-fields:
//
//	x = op[7:6]   y = op[5:3]   z = op[2:0]   p = y>>1   q = y&1
func (d *dec) decodeMain(op byte) Inst {
	x, y, z := op>>6, (op>>3)&7, op&7
	p, q := y>>1, y&1
	mk := func(mn, text string) Inst { return Inst{Mnem: mn, Text: text, Flow: FlowSeq} }
	bad := func() Inst { d.bad = true; return Inst{} }

	switch x {
	case 0:
		switch z {
		case 0:
			switch y {
			case 0:
				return mk("NOP", "NOP")
			case 1:
				return mk("LD", fmt.Sprintf("LD ($%04X),SP", d.imm16()))
			case 2:
				d.imm8() // STOP is two bytes: the opcode plus an ignored padding byte
				return mk("STOP", "STOP")
			case 3:
				t := d.rel()
				return Inst{Mnem: "JR", Text: fmt.Sprintf("JR $%04X", t), Flow: FlowJump, Target: t, HasTarget: true}
			default: // 4..7: JR cc,e
				t := d.rel()
				return Inst{Mnem: "JR", Text: fmt.Sprintf("JR %s,$%04X", ccTab[y-4], t), Flow: FlowBranch, Target: t, HasTarget: true}
			}
		case 1:
			if q == 0 {
				return mk("LD", fmt.Sprintf("LD %s,$%04X", rpTab[p], d.imm16()))
			}
			return mk("ADD", "ADD HL,"+rpTab[p])
		case 2:
			switch {
			case q == 0 && p == 0:
				return mk("LD", "LD (BC),A")
			case q == 0 && p == 1:
				return mk("LD", "LD (DE),A")
			case q == 0 && p == 2:
				return mk("LD", "LD (HL+),A")
			case q == 0 && p == 3:
				return mk("LD", "LD (HL-),A")
			case q == 1 && p == 0:
				return mk("LD", "LD A,(BC)")
			case q == 1 && p == 1:
				return mk("LD", "LD A,(DE)")
			case q == 1 && p == 2:
				return mk("LD", "LD A,(HL+)")
			default: // q==1 p==3
				return mk("LD", "LD A,(HL-)")
			}
		case 3:
			if q == 0 {
				return mk("INC", "INC "+rpTab[p])
			}
			return mk("DEC", "DEC "+rpTab[p])
		case 4:
			return mk("INC", "INC "+rTab[y])
		case 5:
			return mk("DEC", "DEC "+rTab[y])
		case 6:
			return mk("LD", fmt.Sprintf("LD %s,$%02X", rTab[y], d.imm8()))
		default: // z==7
			return mk(accTab[y], accTab[y])
		}
	case 1:
		if z == 6 && y == 6 {
			return mk("HALT", "HALT")
		}
		return mk("LD", fmt.Sprintf("LD %s,%s", rTab[y], rTab[z]))
	case 2:
		return mk(aluMnem(y), aluTab[y]+rTab[z])
	default: // x==3
		switch z {
		case 0:
			switch y {
			case 0, 1, 2, 3:
				return Inst{Mnem: "RET", Text: "RET " + ccTab[y], Flow: FlowSeq} // RET cc: keep tracing the fall-through
			case 4:
				return mk("LDH", fmt.Sprintf("LDH ($FF%02X),A", d.imm8())) // LD ($FF00+a8),A
			case 5:
				return mk("ADD", fmt.Sprintf("ADD SP,%+d", int8(d.imm8())))
			case 6:
				return mk("LDH", fmt.Sprintf("LDH A,($FF%02X)", d.imm8())) // LD A,($FF00+a8)
			default: // y==7
				return mk("LD", fmt.Sprintf("LD HL,SP%+d", int8(d.imm8())))
			}
		case 1:
			if q == 0 {
				return mk("POP", "POP "+rp2Tab[p])
			}
			switch p {
			case 0:
				return Inst{Mnem: "RET", Text: "RET", Flow: FlowReturn}
			case 1:
				return Inst{Mnem: "RETI", Text: "RETI", Flow: FlowReturn}
			case 2:
				return Inst{Mnem: "JP", Text: "JP (HL)", Flow: FlowIndJump}
			default: // p==3
				return mk("LD", "LD SP,HL")
			}
		case 2:
			switch y {
			case 0, 1, 2, 3:
				t := d.imm16()
				return Inst{Mnem: "JP", Text: fmt.Sprintf("JP %s,$%04X", ccTab[y], t), Flow: FlowBranch, Target: t, HasTarget: true}
			case 4:
				return mk("LDH", "LDH (C),A") // LD ($FF00+C),A
			case 5:
				return mk("LD", fmt.Sprintf("LD ($%04X),A", d.imm16()))
			case 6:
				return mk("LDH", "LDH A,(C)") // LD A,($FF00+C)
			default: // y==7
				return mk("LD", fmt.Sprintf("LD A,($%04X)", d.imm16()))
			}
		case 3:
			switch y {
			case 0:
				t := d.imm16()
				return Inst{Mnem: "JP", Text: fmt.Sprintf("JP $%04X", t), Flow: FlowJump, Target: t, HasTarget: true}
			case 6:
				return mk("DI", "DI")
			case 7:
				return mk("EI", "EI")
			default: // y==1 is CB (handled in run); y==2,3,4,5 are illegal
				return bad()
			}
		case 4:
			if y <= 3 {
				t := d.imm16()
				return Inst{Mnem: "CALL", Text: fmt.Sprintf("CALL %s,$%04X", ccTab[y], t), Flow: FlowCall, Target: t, HasTarget: true}
			}
			return bad() // y==4..7 illegal
		case 5:
			if q == 0 {
				return mk("PUSH", "PUSH "+rp2Tab[p])
			}
			if p == 0 {
				t := d.imm16()
				return Inst{Mnem: "CALL", Text: fmt.Sprintf("CALL $%04X", t), Flow: FlowCall, Target: t, HasTarget: true}
			}
			return bad() // p==1/2/3 were Z80 prefixes; illegal here
		case 6:
			return mk(aluMnem(y), aluTab[y]+fmt.Sprintf("$%02X", d.imm8()))
		default: // z==7: RST y*8
			t := uint16(y) * 8
			return Inst{Mnem: "RST", Text: fmt.Sprintf("RST $%02X", t), Flow: FlowCall, Target: t, HasTarget: true}
		}
	}
}

// decodeCB decodes a CB-page opcode: rotate/shift (RLC/RRC/RL/RR/SLA/SRA/SWAP/SRL),
// BIT, RES, SET. There is no (IX+d) form on the LR35902, so this is a plain table.
func (d *dec) decodeCB(op byte) Inst {
	x, y, z := op>>6, (op>>3)&7, op&7
	switch x {
	case 0:
		return Inst{Mnem: rotTab[y], Text: rotTab[y] + " " + rTab[z], Flow: FlowSeq}
	case 1:
		return Inst{Mnem: "BIT", Text: fmt.Sprintf("BIT %d,%s", y, rTab[z]), Flow: FlowSeq}
	case 2:
		return Inst{Mnem: "RES", Text: fmt.Sprintf("RES %d,%s", y, rTab[z]), Flow: FlowSeq}
	default:
		return Inst{Mnem: "SET", Text: fmt.Sprintf("SET %d,%s", y, rTab[z]), Flow: FlowSeq}
	}
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
