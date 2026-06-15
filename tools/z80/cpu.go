package z80

// A minimal but practically complete Zilog Z80 execution core, the executable
// counterpart of the decoder in decode.go (mirroring mos6502.CPU and m68k.CPU).
// Memory and I/O go through the Bus, so the caller supplies the machine model
// (RAM, cartridge mapper, VDP/PSG ports). It implements the documented instruction
// set — all the CB/ED/DD/FD prefix pages, the block move/IO ops, and IM-1
// interrupts — with documented flag behaviour; timing is not modelled.

import "fmt"

// Bus is the memory + I/O the CPU drives. Ports are the full 16-bit address the
// Z80 puts on the bus (BC for the (C) forms, A:n for the (n) forms).
type Bus interface {
	Read(addr uint16) byte
	Write(addr uint16, v byte)
	In(port uint16) byte
	Out(port uint16, v byte)
}

// Z80 flag bits (the F register).
const (
	flagC = 1 << 0 // carry
	flagN = 1 << 1 // add/subtract
	flagP = 1 << 2 // parity / overflow
	flagX = 1 << 3 // undocumented (bit 3 of result)
	flagH = 1 << 4 // half carry
	flagY = 1 << 5 // undocumented (bit 5 of result)
	flagZ = 1 << 6 // zero
	flagS = 1 << 7 // sign
)

type CPU struct {
	A, F, B, C, D, E, H, L         byte // main register set
	A2, F2, B2, C2, D2, E2, H2, L2 byte // alternate set (…')
	IX, IY, SP, PC                 uint16
	I, R                           byte
	IFF1, IFF2                     bool
	IM                             byte

	Halted     bool   // fatal stop (unimplemented opcode); inspect HaltReason
	HaltReason string
	waiting    bool // executed HALT; idling until an interrupt wakes it
	Instrs     uint64

	bus       Bus
	intReq    bool // INT line asserted (level), held until the handler acks
	eiPending bool // EI enables interrupts after the following instruction

	// per-instruction index-register substitution state (DD/FD prefix)
	idx     int    // 0 = HL, 1 = IX, 2 = IY
	usesHL  bool   // the current opcode references (HL) -> (idx+disp)
	disp    uint16 // resolved (idx+disp) address
	dispSet bool
}

func NewCPU(bus Bus) *CPU { return &CPU{bus: bus} }

// Halt stops the core (used for an unimplemented opcode), recording why.
func (c *CPU) Halt(format string, args ...interface{}) {
	c.Halted = true
	c.HaltReason = fmt.Sprintf(format, args...)
}

// RequestIRQ raises or lowers the maskable interrupt line.
func (c *CPU) RequestIRQ(b bool) { c.intReq = b }

func (c *CPU) read(a uint16) byte     { return c.bus.Read(a) }
func (c *CPU) write(a uint16, v byte) { c.bus.Write(a, v) }
func (c *CPU) read16(a uint16) uint16 { return uint16(c.read(a)) | uint16(c.read(a+1))<<8 }
func (c *CPU) write16(a, v uint16)    { c.write(a, byte(v)); c.write(a+1, byte(v>>8)) }
func (c *CPU) fetch() byte            { v := c.read(c.PC); c.PC++; return v }
func (c *CPU) fetch16() uint16        { v := c.read16(c.PC); c.PC += 2; return v }

func (c *CPU) push16(v uint16) { c.SP -= 2; c.write16(c.SP, v) }
func (c *CPU) pop16() uint16   { v := c.read16(c.SP); c.SP += 2; return v }

// --- register pairs --------------------------------------------------------

func (c *CPU) bc() uint16   { return uint16(c.B)<<8 | uint16(c.C) }
func (c *CPU) de() uint16   { return uint16(c.D)<<8 | uint16(c.E) }
func (c *CPU) hl() uint16   { return uint16(c.H)<<8 | uint16(c.L) }
func (c *CPU) af() uint16   { return uint16(c.A)<<8 | uint16(c.F) }
func (c *CPU) setBC(v uint16) { c.B, c.C = byte(v>>8), byte(v) }
func (c *CPU) setDE(v uint16) { c.D, c.E = byte(v>>8), byte(v) }
func (c *CPU) setHL(v uint16) { c.H, c.L = byte(v>>8), byte(v) }
func (c *CPU) setAF(v uint16) { c.A, c.F = byte(v>>8), byte(v) }

// idxReg returns the active 16-bit index (HL, or IX/IY under a prefix).
func (c *CPU) idxReg() uint16 {
	switch c.idx {
	case 1:
		return c.IX
	case 2:
		return c.IY
	}
	return c.hl()
}
func (c *CPU) setIdxReg(v uint16) {
	switch c.idx {
	case 1:
		c.IX = v
	case 2:
		c.IY = v
	default:
		c.setHL(v)
	}
}

// rp/rp2 read/write a 16-bit pair by index (p=0 BC,1 DE,2 HL/idx,3 SP/AF).
func (c *CPU) getRP(p int, af bool) uint16 {
	switch p {
	case 0:
		return c.bc()
	case 1:
		return c.de()
	case 2:
		return c.idxReg()
	default:
		if af {
			return c.af()
		}
		return c.SP
	}
}
func (c *CPU) setRP(p int, v uint16, af bool) {
	switch p {
	case 0:
		c.setBC(v)
	case 1:
		c.setDE(v)
	case 2:
		c.setIdxReg(v)
	default:
		if af {
			c.setAF(v)
		} else {
			c.SP = v
		}
	}
}

// --- 8-bit register access by index, honouring the DD/FD substitution ------

// memAddr resolves the (HL)/(idx+disp) address, reading the displacement once.
func (c *CPU) memAddr() uint16 {
	if c.idx == 0 {
		return c.hl()
	}
	if !c.dispSet {
		c.disp = c.idxReg() + uint16(int8(c.fetch()))
		c.dispSet = true
	}
	return c.disp
}

func (c *CPU) getR(i int) byte {
	switch i {
	case 0:
		return c.B
	case 1:
		return c.C
	case 2:
		return c.D
	case 3:
		return c.E
	case 4:
		if c.idx != 0 && !c.usesHL {
			return byte(c.idxReg() >> 8) // IXH/IYH
		}
		return c.H
	case 5:
		if c.idx != 0 && !c.usesHL {
			return byte(c.idxReg()) // IXL/IYL
		}
		return c.L
	case 6:
		return c.read(c.memAddr())
	default:
		return c.A
	}
}

func (c *CPU) setR(i int, v byte) {
	switch i {
	case 0:
		c.B = v
	case 1:
		c.C = v
	case 2:
		c.D = v
	case 3:
		c.E = v
	case 4:
		if c.idx != 0 && !c.usesHL {
			c.setIdxReg(c.idxReg()&0x00FF | uint16(v)<<8) // IXH/IYH
			return
		}
		c.H = v
	case 5:
		if c.idx != 0 && !c.usesHL {
			r := c.idxReg()&0xFF00 | uint16(v)
			c.setIdxReg(r)
			return
		}
		c.L = v
	case 6:
		c.write(c.memAddr(), v)
	default:
		c.A = v
	}
}

// --- flag helpers ----------------------------------------------------------

func (c *CPU) getf(b byte) bool { return c.F&b != 0 }
func (c *CPU) setf(b byte, on bool) {
	if on {
		c.F |= b
	} else {
		c.F &^= b
	}
}

func parity(v byte) bool { v ^= v >> 4; v ^= v >> 2; v ^= v >> 1; return v&1 == 0 }

// szxy sets S, Z and the undocumented Y/X from a result byte.
func (c *CPU) szxy(v byte) {
	c.F &^= flagS | flagZ | flagY | flagX
	c.F |= v & (flagS | flagY | flagX)
	if v == 0 {
		c.F |= flagZ
	}
}

// --- 8-bit ALU (the alu[] table: ADD ADC SUB SBC AND XOR OR CP) -------------

func (c *CPU) add8(n byte, carry bool) {
	cin := byte(0)
	if carry && c.getf(flagC) {
		cin = 1
	}
	r := uint16(c.A) + uint16(n) + uint16(cin)
	res := byte(r)
	c.F = 0
	c.szxy(res)
	c.setf(flagH, (c.A&0xF)+(n&0xF)+cin > 0xF)
	c.setf(flagC, r > 0xFF)
	c.setf(flagP, (c.A^n)&0x80 == 0 && (c.A^res)&0x80 != 0)
	c.A = res
}

func (c *CPU) sub8(n byte, carry, store bool) byte {
	cin := byte(0)
	if carry && c.getf(flagC) {
		cin = 1
	}
	r := int(c.A) - int(n) - int(cin)
	res := byte(r)
	c.F = flagN
	c.szxy(res)
	c.setf(flagH, int(c.A&0xF)-int(n&0xF)-int(cin) < 0)
	c.setf(flagC, r < 0)
	c.setf(flagP, (c.A^n)&0x80 != 0 && (c.A^res)&0x80 != 0)
	if !store { // CP: Y/X come from the operand, not the result
		c.F &^= flagY | flagX
		c.F |= n & (flagY | flagX)
	}
	return res
}

func (c *CPU) and8(n byte) { c.A &= n; c.F = flagH; c.szxy(c.A); c.setf(flagP, parity(c.A)) }
func (c *CPU) or8(n byte)  { c.A |= n; c.F = 0; c.szxy(c.A); c.setf(flagP, parity(c.A)) }
func (c *CPU) xor8(n byte) { c.A ^= n; c.F = 0; c.szxy(c.A); c.setf(flagP, parity(c.A)) }

// alu dispatches the 8-bit ALU op y (0..7) with operand n.
func (c *CPU) alu(y int, n byte) {
	switch y {
	case 0:
		c.add8(n, false)
	case 1:
		c.add8(n, true)
	case 2:
		c.A = c.sub8(n, false, true)
	case 3:
		c.A = c.sub8(n, true, true)
	case 4:
		c.and8(n)
	case 5:
		c.xor8(n)
	case 6:
		c.or8(n)
	default:
		c.sub8(n, false, false) // CP: flags only
	}
}

func (c *CPU) inc8(v byte) byte {
	r := v + 1
	c.F &^= flagS | flagZ | flagY | flagX | flagH | flagP | flagN
	c.szxy(r)
	c.setf(flagH, v&0xF == 0xF)
	c.setf(flagP, v == 0x7F)
	return r
}
func (c *CPU) dec8(v byte) byte {
	r := v - 1
	c.F = (c.F &^ (flagS | flagZ | flagY | flagX | flagH | flagP)) | flagN
	c.szxy(r)
	c.setf(flagH, v&0xF == 0)
	c.setf(flagP, v == 0x80)
	return r
}

// --- 16-bit ALU ------------------------------------------------------------

func (c *CPU) add16(a, b uint16) uint16 {
	r := uint32(a) + uint32(b)
	c.F &^= flagN | flagC | flagH | flagY | flagX
	c.setf(flagC, r > 0xFFFF)
	c.setf(flagH, (a&0xFFF)+(b&0xFFF) > 0xFFF)
	c.F |= byte(r>>8) & (flagY | flagX)
	return uint16(r)
}
func (c *CPU) adc16(a, b uint16) uint16 {
	cin := uint32(0)
	if c.getf(flagC) {
		cin = 1
	}
	r := uint32(a) + uint32(b) + cin
	res := uint16(r)
	c.F = 0
	c.setf(flagS, res&0x8000 != 0)
	c.setf(flagZ, res == 0)
	c.setf(flagC, r > 0xFFFF)
	c.setf(flagH, (a&0xFFF)+(b&0xFFF)+uint16(cin) > 0xFFF)
	c.setf(flagP, (a^b)&0x8000 == 0 && (a^res)&0x8000 != 0)
	c.F |= byte(res>>8) & (flagY | flagX)
	return res
}
func (c *CPU) sbc16(a, b uint16) uint16 {
	cin := uint32(0)
	if c.getf(flagC) {
		cin = 1
	}
	r := int32(a) - int32(b) - int32(cin)
	res := uint16(r)
	c.F = flagN
	c.setf(flagS, res&0x8000 != 0)
	c.setf(flagZ, res == 0)
	c.setf(flagC, r < 0)
	c.setf(flagH, int32(a&0xFFF)-int32(b&0xFFF)-int32(cin) < 0)
	c.setf(flagP, (a^b)&0x8000 != 0 && (a^res)&0x8000 != 0)
	c.F |= byte(res>>8) & (flagY | flagX)
	return res
}

// --- rotates / shifts (CB rot[]) -------------------------------------------

func (c *CPU) rot(y int, v byte) byte {
	var r byte
	cf := c.getf(flagC)
	switch y {
	case 0: // RLC
		r = v<<1 | v>>7
		c.setf(flagC, v&0x80 != 0)
	case 1: // RRC
		r = v>>1 | v<<7
		c.setf(flagC, v&1 != 0)
	case 2: // RL
		r = v << 1
		if cf {
			r |= 1
		}
		c.setf(flagC, v&0x80 != 0)
	case 3: // RR
		r = v >> 1
		if cf {
			r |= 0x80
		}
		c.setf(flagC, v&1 != 0)
	case 4: // SLA
		r = v << 1
		c.setf(flagC, v&0x80 != 0)
	case 5: // SRA
		r = v>>1 | v&0x80
		c.setf(flagC, v&1 != 0)
	case 6: // SLL (undocumented)
		r = v<<1 | 1
		c.setf(flagC, v&0x80 != 0)
	default: // SRL
		r = v >> 1
		c.setf(flagC, v&1 != 0)
	}
	c.F &^= flagN | flagH | flagS | flagZ | flagY | flagX | flagP
	c.szxy(r)
	c.setf(flagP, parity(r))
	return r
}

// --- the instruction step --------------------------------------------------

// Step executes one instruction (after first servicing a pending interrupt).
func (c *CPU) Step() {
	if c.Halted {
		return
	}
	// EI takes effect after the instruction following it.
	servicedEI := c.eiPending
	if c.intReq && c.IFF1 && !c.eiPending {
		c.waiting = false // an interrupt wakes a HALTed CPU
		c.IFF1, c.IFF2 = false, false
		c.push16(c.PC)
		if c.IM == 2 {
			c.PC = c.read16(uint16(c.I)<<8 | 0xFF)
		} else {
			c.PC = 0x0038
		}
		c.Instrs++
		return
	}
	if c.waiting { // HALTed: idle (the interrupt above is the only way out)
		c.Instrs++
		return
	}
	c.Instrs++
	c.idx, c.usesHL, c.dispSet = 0, false, false
	c.exec(c.fetch())
	if servicedEI {
		c.eiPending = false
	}
}

// exec runs one opcode, recursing for the CB/ED/DD/FD prefixes.
func (c *CPU) exec(op byte) {
	switch op {
	case 0xCB:
		c.execCB(c.fetch())
	case 0xED:
		c.execED(c.fetch())
	case 0xDD, 0xFD:
		c.idx = 1
		if op == 0xFD {
			c.idx = 2
		}
		nb := c.fetch()
		if nb == 0xCB {
			// DDCB: displacement precedes the final opcode.
			c.disp = c.idxReg() + uint16(int8(c.fetch()))
			c.dispSet, c.usesHL = true, true
			c.execCB(c.fetch())
			return
		}
		c.execMain(nb)
	default:
		c.execMain(op)
	}
}

func (c *CPU) execMain(op byte) {
	x, y, z := op>>6, int(op>>3)&7, int(op)&7
	p, q := y>>1, y&1
	c.usesHL = c.idx != 0 && mainUsesHL(byte(x), byte(y), byte(z))

	switch x {
	case 0:
		c.execX0(y, z, p, q)
	case 1:
		if z == 6 && y == 6 {
			c.waiting = true // HALT: idle until an interrupt fires
			return
		}
		c.setR(y, c.getR(z)) // LD r,r
	case 2:
		c.alu(y, c.getR(z))
	default:
		c.execX3(y, z, p, q)
	}
}

func (c *CPU) execX0(y, z, p, q int) {
	switch z {
	case 0:
		switch y {
		case 0: // NOP
		case 1: // EX AF,AF'
			c.A, c.A2 = c.A2, c.A
			c.F, c.F2 = c.F2, c.F
		case 2: // DJNZ d
			off := int8(c.fetch())
			c.B--
			if c.B != 0 {
				c.PC = uint16(int(c.PC) + int(off))
			}
		case 3: // JR d
			off := int8(c.fetch())
			c.PC = uint16(int(c.PC) + int(off))
		default: // JR cc,d
			off := int8(c.fetch())
			if c.cond(y - 4) {
				c.PC = uint16(int(c.PC) + int(off))
			}
		}
	case 1:
		if q == 0 { // LD rp,nn
			c.setRP(p, c.fetch16(), false)
		} else { // ADD HL,rp
			c.setIdxReg(c.add16(c.idxReg(), c.getRP(p, false)))
		}
	case 2:
		switch {
		case q == 0 && p == 0:
			c.write(c.bc(), c.A) // LD (BC),A
		case q == 0 && p == 1:
			c.write(c.de(), c.A) // LD (DE),A
		case q == 0 && p == 2:
			c.write16(c.fetch16(), c.idxReg()) // LD (nn),HL
		case q == 0 && p == 3:
			c.write(c.fetch16(), c.A) // LD (nn),A
		case q == 1 && p == 0:
			c.A = c.read(c.bc()) // LD A,(BC)
		case q == 1 && p == 1:
			c.A = c.read(c.de()) // LD A,(DE)
		case q == 1 && p == 2:
			c.setIdxReg(c.read16(c.fetch16())) // LD HL,(nn)
		default:
			c.A = c.read(c.fetch16()) // LD A,(nn)
		}
	case 3:
		if q == 0 {
			c.setRP(p, c.getRP(p, false)+1, false) // INC rp
		} else {
			c.setRP(p, c.getRP(p, false)-1, false) // DEC rp
		}
	case 4: // INC r
		c.setR(y, c.inc8(c.getR(y)))
	case 5: // DEC r
		c.setR(y, c.dec8(c.getR(y)))
	case 6: // LD r,n
		c.setR(y, c.fetch())
	default: // z==7: the accumulator/flag ops
		c.accOp(y)
	}
}

func (c *CPU) accOp(y int) {
	switch y {
	case 0: // RLCA
		c.A = c.A<<1 | c.A>>7
		c.F = c.F&^(flagN|flagH|flagC|flagY|flagX) | c.A&(flagY|flagX)
		c.setf(flagC, c.A&1 != 0)
	case 1: // RRCA
		c.setf(flagC, c.A&1 != 0)
		c.A = c.A>>1 | c.A<<7
		c.F = c.F&^(flagN|flagH|flagY|flagX) | c.A&(flagY|flagX)
	case 2: // RLA
		cf := c.getf(flagC)
		c.setf(flagC, c.A&0x80 != 0)
		c.A <<= 1
		if cf {
			c.A |= 1
		}
		c.F = c.F&^(flagN|flagH|flagY|flagX) | c.A&(flagY|flagX)
	case 3: // RRA
		cf := c.getf(flagC)
		c.setf(flagC, c.A&1 != 0)
		c.A >>= 1
		if cf {
			c.A |= 0x80
		}
		c.F = c.F&^(flagN|flagH|flagY|flagX) | c.A&(flagY|flagX)
	case 4: // DAA
		c.daa()
	case 5: // CPL
		c.A = ^c.A
		c.F |= flagN | flagH
		c.F = c.F&^(flagY|flagX) | c.A&(flagY|flagX)
	case 6: // SCF
		c.F &^= flagN | flagH
		c.F |= flagC
		c.F = c.F&^(flagY|flagX) | c.A&(flagY|flagX)
	default: // CCF
		c.setf(flagH, c.getf(flagC))
		c.F ^= flagC
		c.F &^= flagN
		c.F = c.F&^(flagY|flagX) | c.A&(flagY|flagX)
	}
}

func (c *CPU) daa() {
	a := c.A
	var add byte
	carry := c.getf(flagC)
	if c.getf(flagH) || a&0xF > 9 {
		add |= 0x06
	}
	if carry || a > 0x99 {
		add |= 0x60
		carry = true
	}
	if c.getf(flagN) {
		c.setf(flagH, c.getf(flagH) && a&0xF < 6)
		a -= add
	} else {
		c.setf(flagH, a&0xF > 9)
		a += add
	}
	c.A = a
	c.szxy(a)
	c.setf(flagP, parity(a))
	c.setf(flagC, carry)
}

func (c *CPU) execX3(y, z, p, q int) {
	switch z {
	case 0: // RET cc
		if c.cond(y) {
			c.PC = c.pop16()
		}
	case 1:
		if q == 0 {
			c.setRP(p, c.pop16(), true) // POP rp2
		} else {
			switch p {
			case 0:
				c.PC = c.pop16() // RET
			case 1: // EXX
				c.B, c.B2 = c.B2, c.B
				c.C, c.C2 = c.C2, c.C
				c.D, c.D2 = c.D2, c.D
				c.E, c.E2 = c.E2, c.E
				c.H, c.H2 = c.H2, c.H
				c.L, c.L2 = c.L2, c.L
			case 2:
				c.PC = c.idxReg() // JP (HL)
			default:
				c.SP = c.idxReg() // LD SP,HL
			}
		}
	case 2: // JP cc,nn
		a := c.fetch16()
		if c.cond(y) {
			c.PC = a
		}
	case 3:
		switch y {
		case 0:
			c.PC = c.fetch16() // JP nn
		case 2:
			c.bus.Out(uint16(c.A)<<8|uint16(c.fetch()), c.A) // OUT (n),A
		case 3:
			c.A = c.bus.In(uint16(c.A)<<8 | uint16(c.fetch())) // IN A,(n)
		case 4: // EX (SP),HL
			t := c.read16(c.SP)
			c.write16(c.SP, c.idxReg())
			c.setIdxReg(t)
		case 5: // EX DE,HL
			d, h := c.de(), c.hl()
			c.setDE(h)
			c.setHL(d)
		case 6: // DI
			c.IFF1, c.IFF2 = false, false
		default: // EI
			c.IFF1, c.IFF2 = true, true
			c.eiPending = true
		}
	case 4: // CALL cc,nn
		a := c.fetch16()
		if c.cond(y) {
			c.push16(c.PC)
			c.PC = a
		}
	case 5:
		if q == 0 {
			c.push16(c.getRP(p, true)) // PUSH rp2
		} else if p == 0 { // CALL nn
			a := c.fetch16()
			c.push16(c.PC)
			c.PC = a
		}
	case 6: // alu n
		c.alu(y, c.fetch())
	default: // RST
		c.push16(c.PC)
		c.PC = uint16(y) * 8
	}
}

// cond evaluates condition cc[y]: NZ Z NC C PO PE P M.
func (c *CPU) cond(y int) bool {
	switch y {
	case 0:
		return !c.getf(flagZ)
	case 1:
		return c.getf(flagZ)
	case 2:
		return !c.getf(flagC)
	case 3:
		return c.getf(flagC)
	case 4:
		return !c.getf(flagP)
	case 5:
		return c.getf(flagP)
	case 6:
		return !c.getf(flagS)
	default:
		return c.getf(flagS)
	}
}

func (c *CPU) execCB(op byte) {
	x, y, z := op>>6, int(op>>3)&7, int(op)&7
	// For DDCB/FDCB the operand is always (idx+disp); a non-6 z also copies the
	// result into that register (the documented behaviour stores to memory).
	get := func() byte {
		if c.idx != 0 {
			return c.read(c.disp)
		}
		return c.getR(z)
	}
	put := func(v byte) {
		if c.idx != 0 {
			c.write(c.disp, v)
			if z != 6 {
				c.setRPlain(z, v)
			}
			return
		}
		c.setR(z, v)
	}
	switch x {
	case 0:
		put(c.rot(y, get()))
	case 1: // BIT y,r
		v := get()
		c.F = c.F&flagC | flagH
		c.setf(flagZ, v&(1<<uint(y)) == 0)
		c.setf(flagP, v&(1<<uint(y)) == 0)
		c.setf(flagS, y == 7 && v&0x80 != 0)
		c.F |= v & (flagY | flagX)
	case 2: // RES
		put(get() &^ (1 << uint(y)))
	default: // SET
		put(get() | 1<<uint(y))
	}
}

// setRPlain writes an 8-bit register by index ignoring index substitution
// (used by the DDCB undocumented register copy).
func (c *CPU) setRPlain(i int, v byte) {
	switch i {
	case 0:
		c.B = v
	case 1:
		c.C = v
	case 2:
		c.D = v
	case 3:
		c.E = v
	case 4:
		c.H = v
	case 5:
		c.L = v
	case 7:
		c.A = v
	}
}

func (c *CPU) execED(op byte) {
	x, y, z := op>>6, int(op>>3)&7, int(op)&7
	p, q := y>>1, y&1
	if x == 2 {
		if z <= 3 && y >= 4 {
			// In the ED block table the operation is z (0 LD,1 CP,2 IN,3 OUT)
			// and the direction/repeat is y (4 I,5 D,6 IR,7 DR).
			c.block(z, y-4)
		}
		return
	}
	if x != 1 {
		return // NONI
	}
	switch z {
	case 0: // IN r,(C)
		v := c.bus.In(c.bc())
		if y != 6 {
			c.setRPlain(y, v)
		}
		c.F = c.F&flagC | 0
		c.szxy(v)
		c.setf(flagP, parity(v))
	case 1: // OUT (C),r
		if y == 6 {
			c.bus.Out(c.bc(), 0)
		} else {
			c.bus.Out(c.bc(), c.getRplain(y))
		}
	case 2:
		if q == 0 {
			c.setHL(c.sbc16(c.hl(), c.getRP(int(p), false))) // SBC HL,rp
		} else {
			c.setHL(c.adc16(c.hl(), c.getRP(int(p), false))) // ADC HL,rp
		}
	case 3:
		if q == 0 {
			c.write16(c.fetch16(), c.getRP(int(p), false)) // LD (nn),rp
		} else {
			c.setRP(int(p), c.read16(c.fetch16()), false) // LD rp,(nn)
		}
	case 4: // NEG
		a := c.A
		c.A = 0
		c.A = c.sub8(a, false, true)
	case 5: // RETN / RETI
		c.PC = c.pop16()
		c.IFF1 = c.IFF2
	case 6: // IM
		c.IM = []byte{0, 0, 1, 2, 0, 0, 1, 2}[y]
	default: // z==7
		switch y {
		case 0:
			c.I = c.A // LD I,A
		case 1:
			c.R = c.A // LD R,A
		case 2: // LD A,I
			c.A = c.I
			c.ldAir()
		case 3: // LD A,R
			c.A = c.R
			c.ldAir()
		case 4:
			c.rrd()
		case 5:
			c.rld()
		}
	}
}

func (c *CPU) getRplain(i int) byte {
	switch i {
	case 0:
		return c.B
	case 1:
		return c.C
	case 2:
		return c.D
	case 3:
		return c.E
	case 4:
		return c.H
	case 5:
		return c.L
	default:
		return c.A
	}
}

func (c *CPU) ldAir() {
	c.F = c.F & flagC
	c.szxy(c.A)
	c.setf(flagP, c.IFF2)
}

func (c *CPU) rrd() {
	m := c.read(c.hl())
	c.write(c.hl(), c.A<<4|m>>4)
	c.A = c.A&0xF0 | m&0x0F
	c.F = c.F & flagC
	c.szxy(c.A)
	c.setf(flagP, parity(c.A))
}
func (c *CPU) rld() {
	m := c.read(c.hl())
	c.write(c.hl(), m<<4|c.A&0x0F)
	c.A = c.A&0xF0 | m>>4
	c.F = c.F & flagC
	c.szxy(c.A)
	c.setf(flagP, parity(c.A))
}

// block runs an ED block instruction: kind 0 LD,1 CP,2 IN,3 OUT; mode 0 I,1 D,2 IR,3 DR.
func (c *CPU) block(kind, mode int) {
	inc := mode == 0 || mode == 2
	repeat := mode >= 2
	switch kind {
	case 0: // LDI/LDD/LDIR/LDDR
		v := c.read(c.hl())
		c.write(c.de(), v)
		if inc {
			c.setHL(c.hl() + 1)
			c.setDE(c.de() + 1)
		} else {
			c.setHL(c.hl() - 1)
			c.setDE(c.de() - 1)
		}
		c.setBC(c.bc() - 1)
		c.F &^= flagH | flagN | flagP
		c.setf(flagP, c.bc() != 0)
		n := v + c.A
		c.F = c.F&^(flagY|flagX) | (n&0x02)<<4 | n&flagX
		if repeat && c.bc() != 0 {
			c.PC -= 2
		}
	case 1: // CPI/CPD/CPIR/CPDR
		v := c.read(c.hl())
		res := c.A - v
		if inc {
			c.setHL(c.hl() + 1)
		} else {
			c.setHL(c.hl() - 1)
		}
		c.setBC(c.bc() - 1)
		c.F = c.F&flagC | flagN
		c.setf(flagS, res&0x80 != 0)
		c.setf(flagZ, res == 0)
		c.setf(flagH, int(c.A&0xF)-int(v&0xF) < 0)
		c.setf(flagP, c.bc() != 0)
		if repeat && c.bc() != 0 && res != 0 {
			c.PC -= 2
		}
	case 2: // INI/IND/INIR/INDR
		v := c.bus.In(c.bc())
		c.write(c.hl(), v)
		c.B--
		if inc {
			c.setHL(c.hl() + 1)
		} else {
			c.setHL(c.hl() - 1)
		}
		c.setf(flagN, true)
		c.setf(flagZ, c.B == 0)
		if repeat && c.B != 0 {
			c.PC -= 2
		}
	default: // OUTI/OUTD/OTIR/OTDR
		v := c.read(c.hl())
		c.B--
		c.bus.Out(c.bc(), v)
		if inc {
			c.setHL(c.hl() + 1)
		} else {
			c.setHL(c.hl() - 1)
		}
		c.setf(flagN, true)
		c.setf(flagZ, c.B == 0)
		if repeat && c.B != 0 {
			c.PC -= 2
		}
	}
}
