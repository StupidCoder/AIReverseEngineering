// This file adds an instruction-level Motorola 68000 execution core to the m68k
// package — the executable counterpart to the disassembler, mirroring the shape
// of mos6502's CPU. Memory access goes through the Bus interface (big-endian
// byte access), so a machine model supplies the address map and can install
// PC hooks the same way the c64 model wraps the 6502 core.
//
// Scope: a "minimal core" — MOVE/MOVEQ/MOVEA, LEA/PEA, the common ALU ops
// (ADD/SUB/CMP/AND/OR/EOR and their immediate and quick forms), the shift and
// rotate family, the branch/jump/call/return and DBcc control flow, MOVEM,
// LINK/UNLK, EXT/SWAP, TST/CLR and NOP — all with correct X/N/Z/V/C condition
// codes. Anything outside that set halts the CPU with the offending opcode, so
// gaps are obvious and easy to fill on demand.
package m68k

import "fmt"

// Bus is the memory interface seen by the CPU: flat byte-addressed memory.
// The CPU composes big-endian words and longs from it.
type Bus interface {
	Read(addr uint32) byte
	Write(addr uint32, v byte)
}

// CPU is a 68000 execution core. Registers are exported so a machine model and
// tests can seed and inspect them.
type CPU struct {
	D             [8]uint32 // data registers
	A             [8]uint32 // address registers; A[7] is the active stack pointer
	PC            uint32
	X, N, Z, V, C bool // condition codes

	bus        Bus
	Halted     bool
	HaltReason string
}

// NewCPU returns a CPU bound to bus.
func NewCPU(bus Bus) *CPU { return &CPU{bus: bus} }

// Halt stops execution and records why (an unimplemented opcode, a bad EA, …).
func (c *CPU) Halt(format string, args ...interface{}) {
	c.Halted = true
	c.HaltReason = fmt.Sprintf(format, args...)
}

// --- bus access (big-endian) ---

func (c *CPU) read8(a uint32) uint32 { return uint32(c.bus.Read(a)) }
func (c *CPU) read16(a uint32) uint32 {
	return uint32(c.bus.Read(a))<<8 | uint32(c.bus.Read(a+1))
}
func (c *CPU) read32(a uint32) uint32 {
	return c.read16(a)<<16 | c.read16(a+2)
}
func (c *CPU) write8(a, v uint32) { c.bus.Write(a, byte(v)) }
func (c *CPU) write16(a, v uint32) {
	c.bus.Write(a, byte(v>>8))
	c.bus.Write(a+1, byte(v))
}
func (c *CPU) write32(a, v uint32) {
	c.write16(a, v>>16)
	c.write16(a+2, v)
}

func (c *CPU) readSize(a uint32, size int) uint32 {
	switch size {
	case 0:
		return c.read8(a)
	case 1:
		return c.read16(a)
	default:
		return c.read32(a)
	}
}
func (c *CPU) writeSize(a, v uint32, size int) {
	switch size {
	case 0:
		c.write8(a, v)
	case 1:
		c.write16(a, v)
	default:
		c.write32(a, v)
	}
}

func (c *CPU) fetch16() uint32 { v := c.read16(c.PC); c.PC += 2; return v }
func (c *CPU) fetch32() uint32 { v := c.read32(c.PC); c.PC += 4; return v }

func (c *CPU) push32(v uint32) { c.A[7] -= 4; c.write32(c.A[7], v) }
func (c *CPU) pop32() uint32   { v := c.read32(c.A[7]); c.A[7] += 4; return v }

// CCR assembles the condition codes into the low byte of the status register.
func (c *CPU) CCR() byte {
	var r byte
	for i, f := range []bool{c.C, c.V, c.Z, c.N, c.X} {
		if f {
			r |= 1 << uint(i)
		}
	}
	return r
}

// setCCR writes the condition codes from the low byte of the status register.
func (c *CPU) setCCR(v byte) {
	c.C = v&0x01 != 0
	c.V = v&0x02 != 0
	c.Z = v&0x04 != 0
	c.N = v&0x08 != 0
	c.X = v&0x10 != 0
}

// --- size helpers ---

func sizeBytes(s int) uint32 { return []uint32{1, 2, 4}[s] }
func sizeBits(s int) int     { return []int{8, 16, 32}[s] }
func sizeMask(s int) uint32  { return []uint32{0xFF, 0xFFFF, 0xFFFFFFFF}[s] }
func signBit(v uint32, s int) bool {
	return v&(1<<uint(sizeBits(s)-1)) != 0
}
func signExtend(v uint32, s int) uint32 {
	switch s {
	case 0:
		return uint32(int32(int8(byte(v))))
	case 1:
		return uint32(int32(int16(uint16(v))))
	default:
		return v
	}
}

// --- effective addresses ---

// operand is a resolved effective address. Post-increment and pre-decrement
// side effects are applied at resolve time, so a read-modify-write uses one
// stable address.
type operand struct {
	kind int // 0 = Dn, 1 = An, 2 = memory, 3 = immediate
	reg  int
	addr uint32
	imm  uint32
}

func (c *CPU) incr(reg, size int) uint32 {
	if reg == 7 && size == 0 {
		return 2 // keep the stack pointer even on byte access
	}
	return sizeBytes(size)
}

// indexEA reads a brief extension word and adds d8 + scaled index register.
func (c *CPU) indexEA(base uint32) uint32 {
	ext := c.fetch16()
	disp := uint32(int32(int8(byte(ext))))
	ireg := (ext >> 12) & 7
	var idx uint32
	if ext&0x8000 != 0 {
		idx = c.A[ireg]
	} else {
		idx = c.D[ireg]
	}
	if ext&0x0800 == 0 { // word index: sign-extend the low 16 bits
		idx = uint32(int32(int16(uint16(idx))))
	}
	return base + disp + idx
}

func (c *CPU) resolveEA(mode, reg, size int) operand {
	switch mode {
	case 0:
		return operand{kind: 0, reg: reg}
	case 1:
		return operand{kind: 1, reg: reg}
	case 2:
		return operand{kind: 2, addr: c.A[reg]}
	case 3:
		a := c.A[reg]
		c.A[reg] += c.incr(reg, size)
		return operand{kind: 2, addr: a}
	case 4:
		c.A[reg] -= c.incr(reg, size)
		return operand{kind: 2, addr: c.A[reg]}
	case 5:
		d := uint32(int32(int16(uint16(c.fetch16()))))
		return operand{kind: 2, addr: c.A[reg] + d}
	case 6:
		return operand{kind: 2, addr: c.indexEA(c.A[reg])}
	case 7:
		switch reg {
		case 0:
			return operand{kind: 2, addr: uint32(int32(int16(uint16(c.fetch16()))))}
		case 1:
			return operand{kind: 2, addr: c.fetch32()}
		case 2:
			pc := c.PC
			d := uint32(int32(int16(uint16(c.fetch16()))))
			return operand{kind: 2, addr: pc + d}
		case 3:
			return operand{kind: 2, addr: c.indexEA(c.PC)}
		case 4:
			if size == 2 {
				return operand{kind: 3, imm: c.fetch32()}
			}
			v := c.fetch16()
			if size == 0 {
				v &= 0xFF
			}
			return operand{kind: 3, imm: v}
		}
	}
	c.Halt("bad effective address mode %d/%d at $%06X", mode, reg, c.PC)
	return operand{kind: 3}
}

func (c *CPU) load(o operand, size int) uint32 {
	switch o.kind {
	case 0:
		return c.D[o.reg] & sizeMask(size)
	case 1:
		return c.A[o.reg] & sizeMask(size)
	case 3:
		return o.imm & sizeMask(size)
	default:
		return c.readSize(o.addr, size)
	}
}

func (c *CPU) store(o operand, size int, v uint32) {
	switch o.kind {
	case 0:
		m := sizeMask(size)
		c.D[o.reg] = (c.D[o.reg] &^ m) | (v & m)
	case 1:
		c.A[o.reg] = signExtend(v, size) // address registers are written full-width
	case 3:
		c.Halt("write to immediate operand at $%06X", c.PC)
	default:
		c.writeSize(o.addr, v, size)
	}
}

// --- flag helpers ---

func (c *CPU) setNZ(v uint32, size int) {
	c.N = signBit(v, size)
	c.Z = v&sizeMask(size) == 0
}
func (c *CPU) setLogic(v uint32, size int) {
	c.setNZ(v, size)
	c.V, c.C = false, false
}

func (c *CPU) add(d, s uint32, size int) uint32 {
	mask := sizeMask(size)
	full := uint64(d&mask) + uint64(s&mask)
	r := uint32(full) & mask
	sm, dm, rm := signBit(s, size), signBit(d, size), signBit(r, size)
	c.C = full>>uint(sizeBits(size)) != 0
	c.V = sm == dm && rm != sm
	c.X = c.C
	c.setNZ(r, size)
	return r
}

func (c *CPU) sub(d, s uint32, size int) uint32 {
	mask := sizeMask(size)
	r := (d - s) & mask
	sm, dm, rm := signBit(s, size), signBit(d, size), signBit(r, size)
	c.C = s&mask > d&mask
	c.V = dm != sm && dm != rm
	c.X = c.C
	c.setNZ(r, size)
	return r
}

// bcd does packed-BCD add (ABCD) or subtract (SBCD) of two bytes with the X flag,
// setting X/C/N/Z per the 68000 (Z is only cleared, for multi-precision chains).
func (c *CPU) bcd(d, s uint32, isAdd bool) uint32 {
	x := uint32(0)
	if c.X {
		x = 1
	}
	var res uint32
	var carry bool
	if isAdd {
		res = (s & 0x0F) + (d & 0x0F) + x
		if res > 9 {
			res += 6
		}
		res += (s & 0xF0) + (d & 0xF0)
		carry = res > 0x99
		if carry {
			res -= 0xA0
		}
	} else {
		res = (d & 0x0F) - (s & 0x0F) - x
		borrowLo := int32(res) < 0
		if borrowLo {
			res -= 6
		}
		res += (d & 0xF0) - (s & 0xF0)
		carry = int32(res)&0x100 != 0 || res > 0xFF
		if carry {
			res -= 0x60
		}
	}
	res &= 0xFF
	c.X, c.C = carry, carry
	c.N = res&0x80 != 0
	if res != 0 {
		c.Z = false
	}
	return res
}

// cmp sets flags like sub but leaves X (and the destination) unchanged.
func (c *CPU) cmp(d, s uint32, size int) {
	x := c.X
	c.sub(d, s, size)
	c.X = x
}

func (c *CPU) cond(cc int) bool {
	switch cc {
	case 0:
		return true
	case 1:
		return false
	case 2:
		return !c.C && !c.Z
	case 3:
		return c.C || c.Z
	case 4:
		return !c.C
	case 5:
		return c.C
	case 6:
		return !c.Z
	case 7:
		return c.Z
	case 8:
		return !c.V
	case 9:
		return c.V
	case 10:
		return !c.N
	case 11:
		return c.N
	case 12:
		return c.N == c.V
	case 13:
		return c.N != c.V
	case 14:
		return !c.Z && c.N == c.V
	default:
		return c.Z || c.N != c.V
	}
}
