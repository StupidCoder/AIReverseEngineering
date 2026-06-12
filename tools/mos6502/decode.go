package mos6502

import "fmt"

// Flow classifies how an instruction affects control flow — the information a
// recursive-descent disassembler needs to follow every reachable path.
type Flow int

const (
	FlowSeq     Flow = iota // continues to the next instruction
	FlowBranch              // conditional branch: continues AND may go to Target
	FlowJump                // unconditional JMP abs: goes to Target, no fall-through
	FlowCall                // JSR: goes to Target, normally returns after it
	FlowReturn              // RTS / RTI: path ends
	FlowIndJump             // JMP (ind): target not statically known
	FlowStop                // BRK or an undocumented opcode: treat as data/stop
)

// Inst is one decoded instruction.
type Inst struct {
	Addr      uint16
	Len       int
	Mnem      string
	Text      string // formatted "MNEM operand"
	Flow      Flow
	Target    uint16 // branch/jump/call destination, or the operand address for abs modes
	HasTarget bool   // Target is meaningful (rel or any absolute-addressed operand)
}

// Decode decodes the instruction at absolute address pc within mem, where mem
// is indexed by address (typically the full 64 KB image). Undocumented opcodes
// and instructions that run off the end of mem decode as a one-byte ".byte".
func Decode(mem []byte, pc uint16) Inst {
	op, known := ops[mem[pc]]
	if !known {
		return Inst{Addr: pc, Len: 1, Mnem: ".byte", Text: fmt.Sprintf(".byte $%02X", mem[pc]), Flow: FlowStop}
	}
	n := modeLen[op.m]
	if int(pc)+n > len(mem) {
		return Inst{Addr: pc, Len: 1, Mnem: ".byte", Text: fmt.Sprintf(".byte $%02X", mem[pc]), Flow: FlowStop}
	}
	var b1 byte
	var w uint16
	if n >= 2 {
		b1 = mem[pc+1]
	}
	if n == 3 {
		w = uint16(mem[pc+1]) | uint16(mem[pc+2])<<8
	}
	in := Inst{Addr: pc, Len: n, Mnem: op.mn, Flow: FlowSeq}
	var arg string
	switch op.m {
	case acc:
		arg = "A"
	case imm:
		arg = fmt.Sprintf("#$%02X", b1)
	case zp:
		arg = fmt.Sprintf("$%02X", b1)
	case zpx:
		arg = fmt.Sprintf("$%02X,X", b1)
	case zpy:
		arg = fmt.Sprintf("$%02X,Y", b1)
	case izx:
		arg = fmt.Sprintf("($%02X,X)", b1)
	case izy:
		arg = fmt.Sprintf("($%02X),Y", b1)
	case rel:
		in.Target = pc + 2 + uint16(int8(b1))
		in.HasTarget = true
		arg = fmt.Sprintf("$%04X", in.Target)
	case abs:
		in.Target, in.HasTarget = w, true
		arg = fmt.Sprintf("$%04X", w)
	case abx:
		in.Target, in.HasTarget = w, true
		arg = fmt.Sprintf("$%04X,X", w)
	case aby:
		in.Target, in.HasTarget = w, true
		arg = fmt.Sprintf("$%04X,Y", w)
	case ind:
		in.Target, in.HasTarget = w, true
		arg = fmt.Sprintf("($%04X)", w)
	}
	in.Text = op.mn
	if arg != "" {
		in.Text += " " + arg
	}
	switch op.mn {
	case "BPL", "BMI", "BVC", "BVS", "BCC", "BCS", "BNE", "BEQ":
		in.Flow = FlowBranch
	case "JMP":
		if op.m == ind {
			in.Flow = FlowIndJump
		} else {
			in.Flow = FlowJump
		}
	case "JSR":
		in.Flow = FlowCall
	case "RTS", "RTI":
		in.Flow = FlowReturn
	case "BRK":
		in.Flow = FlowStop
	}
	return in
}
