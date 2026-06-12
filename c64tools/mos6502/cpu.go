package mos6502

// Minimal 6502 core, sufficient to execute the tape loader code found on this
// image (all documented opcodes, binary and BCD arithmetic). Timing is
// approximate (base cycle counts, no page-cross penalties); the tape hardware
// model only needs ballpark pacing, see machine.go.

import "fmt"

type Bus interface {
	Read(addr uint16) byte
	Write(addr uint16, v byte)
}

type CPU struct {
	A, X, Y, SP         byte
	PC                  uint16
	C, Z, I, D, B, V, N bool

	Cycles uint64
	Instrs uint64
	bus    Bus

	Halted     bool
	HaltReason string
}

func NewCPU(bus Bus) *CPU {
	return &CPU{SP: 0xFD, I: true, bus: bus}
}

func (c *CPU) Halt(format string, args ...interface{}) {
	c.Halted = true
	c.HaltReason = fmt.Sprintf(format, args...)
}

func (c *CPU) read(a uint16) byte     { return c.bus.Read(a) }
func (c *CPU) write(a uint16, v byte) { c.bus.Write(a, v) }
func (c *CPU) read16(a uint16) uint16 {
	return uint16(c.read(a)) | uint16(c.read(a+1))<<8
}

func (c *CPU) fetch() byte {
	v := c.read(c.PC)
	c.PC++
	return v
}
func (c *CPU) fetch16() uint16 {
	v := c.read16(c.PC)
	c.PC += 2
	return v
}

func (c *CPU) Push(v byte) {
	c.write(0x100+uint16(c.SP), v)
	c.SP--
}
func (c *CPU) Pop() byte {
	c.SP++
	return c.read(0x100 + uint16(c.SP))
}

func (c *CPU) flags() byte {
	var p byte = 0x20
	if c.C {
		p |= 0x01
	}
	if c.Z {
		p |= 0x02
	}
	if c.I {
		p |= 0x04
	}
	if c.D {
		p |= 0x08
	}
	if c.V {
		p |= 0x40
	}
	if c.N {
		p |= 0x80
	}
	return p
}

func (c *CPU) setFlags(p byte) {
	c.C = p&0x01 != 0
	c.Z = p&0x02 != 0
	c.I = p&0x04 != 0
	c.D = p&0x08 != 0
	c.V = p&0x40 != 0
	c.N = p&0x80 != 0
}

func (c *CPU) nz(v byte) byte {
	c.Z = v == 0
	c.N = v&0x80 != 0
	return v
}

// addressing modes returning effective address
func (c *CPU) zp() uint16  { return uint16(c.fetch()) }
func (c *CPU) zpx() uint16 { return uint16(c.fetch() + c.X) }
func (c *CPU) zpy() uint16 { return uint16(c.fetch() + c.Y) }
func (c *CPU) abs() uint16 { return c.fetch16() }
func (c *CPU) abx() uint16 { return c.fetch16() + uint16(c.X) }
func (c *CPU) aby() uint16 { return c.fetch16() + uint16(c.Y) }
func (c *CPU) izx() uint16 {
	z := c.fetch() + c.X
	return uint16(c.read(uint16(z))) | uint16(c.read(uint16(z+1)))<<8
}
func (c *CPU) izy() uint16 {
	z := c.fetch()
	base := uint16(c.read(uint16(z))) | uint16(c.read(uint16(z+1)))<<8
	return base + uint16(c.Y)
}

func (c *CPU) branch(cond bool) {
	off := int8(c.fetch())
	if cond {
		c.PC = uint16(int32(c.PC) + int32(off))
		c.Cycles++
	}
}

func (c *CPU) adc(v byte) {
	if c.D {
		// decimal mode (sufficient approximation of NMOS behaviour)
		lo := int(c.A&0x0f) + int(v&0x0f)
		if c.C {
			lo++
		}
		hi := int(c.A>>4) + int(v>>4)
		if lo > 9 {
			lo += 6
			hi++
		}
		bin := int(c.A) + int(v)
		if c.C {
			bin++
		}
		c.Z = byte(bin) == 0
		c.V = (^(c.A^v)&(c.A^byte(hi<<4)))&0x80 != 0
		if hi > 9 {
			hi += 6
		}
		c.C = hi > 15
		c.A = byte(hi<<4 | lo&0x0f)
		c.N = c.A&0x80 != 0
		return
	}
	sum := int(c.A) + int(v)
	if c.C {
		sum++
	}
	r := byte(sum)
	c.V = (^(c.A^v)&(c.A^r))&0x80 != 0
	c.C = sum > 0xff
	c.A = c.nz(r)
}

func (c *CPU) sbc(v byte) {
	if c.D {
		borrow := 0
		if !c.C {
			borrow = 1
		}
		lo := int(c.A&0x0f) - int(v&0x0f) - borrow
		hi := int(c.A>>4) - int(v>>4)
		if lo < 0 {
			lo += 10
			hi--
		}
		bin := int(c.A) - int(v) - borrow
		r := byte(bin)
		c.V = ((c.A^v)&(c.A^r))&0x80 != 0
		c.C = bin >= 0
		c.Z = r == 0
		c.N = r&0x80 != 0
		if hi < 0 {
			hi += 10
		}
		c.A = byte(hi<<4 | lo&0x0f)
		return
	}
	c.adcBinary(^v)
}

func (c *CPU) adcBinary(v byte) {
	sum := int(c.A) + int(v)
	if c.C {
		sum++
	}
	r := byte(sum)
	c.V = (^(c.A^v)&(c.A^r))&0x80 != 0
	c.C = sum > 0xff
	c.A = c.nz(r)
}

func (c *CPU) cmp(reg, v byte) {
	c.C = reg >= v
	c.nz(reg - v)
}

func (c *CPU) aslM(a uint16) {
	v := c.read(a)
	c.C = v&0x80 != 0
	c.write(a, c.nz(v<<1))
}
func (c *CPU) lsrM(a uint16) {
	v := c.read(a)
	c.C = v&0x01 != 0
	c.write(a, c.nz(v>>1))
}
func (c *CPU) rolM(a uint16) {
	v := c.read(a)
	old := c.C
	c.C = v&0x80 != 0
	v <<= 1
	if old {
		v |= 1
	}
	c.write(a, c.nz(v))
}
func (c *CPU) rorM(a uint16) {
	v := c.read(a)
	old := c.C
	c.C = v&0x01 != 0
	v >>= 1
	if old {
		v |= 0x80
	}
	c.write(a, c.nz(v))
}

func (c *CPU) bit(v byte) {
	c.Z = c.A&v == 0
	c.N = v&0x80 != 0
	c.V = v&0x40 != 0
}

// Step executes one instruction.
func (c *CPU) Step() {
	if c.Halted {
		return
	}
	c.Instrs++
	c.Cycles += 4 // crude average; good enough for the tape pacing model
	op := c.fetch()
	switch op {
	case 0x00:
		c.Halt("BRK at $%04X", c.PC-1)
	case 0xEA:
		// NOP
	// --- loads/stores ---
	case 0xA9:
		c.A = c.nz(c.fetch())
	case 0xA5:
		c.A = c.nz(c.read(c.zp()))
	case 0xB5:
		c.A = c.nz(c.read(c.zpx()))
	case 0xAD:
		c.A = c.nz(c.read(c.abs()))
	case 0xBD:
		c.A = c.nz(c.read(c.abx()))
	case 0xB9:
		c.A = c.nz(c.read(c.aby()))
	case 0xA1:
		c.A = c.nz(c.read(c.izx()))
	case 0xB1:
		c.A = c.nz(c.read(c.izy()))
	case 0xA2:
		c.X = c.nz(c.fetch())
	case 0xA6:
		c.X = c.nz(c.read(c.zp()))
	case 0xB6:
		c.X = c.nz(c.read(c.zpy()))
	case 0xAE:
		c.X = c.nz(c.read(c.abs()))
	case 0xBE:
		c.X = c.nz(c.read(c.aby()))
	case 0xA0:
		c.Y = c.nz(c.fetch())
	case 0xA4:
		c.Y = c.nz(c.read(c.zp()))
	case 0xB4:
		c.Y = c.nz(c.read(c.zpx()))
	case 0xAC:
		c.Y = c.nz(c.read(c.abs()))
	case 0xBC:
		c.Y = c.nz(c.read(c.abx()))
	case 0x85:
		c.write(c.zp(), c.A)
	case 0x95:
		c.write(c.zpx(), c.A)
	case 0x8D:
		c.write(c.abs(), c.A)
	case 0x9D:
		c.write(c.abx(), c.A)
	case 0x99:
		c.write(c.aby(), c.A)
	case 0x81:
		c.write(c.izx(), c.A)
	case 0x91:
		c.write(c.izy(), c.A)
	case 0x86:
		c.write(c.zp(), c.X)
	case 0x96:
		c.write(c.zpy(), c.X)
	case 0x8E:
		c.write(c.abs(), c.X)
	case 0x84:
		c.write(c.zp(), c.Y)
	case 0x94:
		c.write(c.zpx(), c.Y)
	case 0x8C:
		c.write(c.abs(), c.Y)
	// --- transfers ---
	case 0xAA:
		c.X = c.nz(c.A)
	case 0xA8:
		c.Y = c.nz(c.A)
	case 0x8A:
		c.A = c.nz(c.X)
	case 0x98:
		c.A = c.nz(c.Y)
	case 0xBA:
		c.X = c.nz(c.SP)
	case 0x9A:
		c.SP = c.X
	// --- stack ---
	case 0x48:
		c.Push(c.A)
	case 0x68:
		c.A = c.nz(c.Pop())
	case 0x08:
		c.Push(c.flags() | 0x10)
	case 0x28:
		c.setFlags(c.Pop())
	// --- arithmetic ---
	case 0x69:
		c.adc(c.fetch())
	case 0x65:
		c.adc(c.read(c.zp()))
	case 0x75:
		c.adc(c.read(c.zpx()))
	case 0x6D:
		c.adc(c.read(c.abs()))
	case 0x7D:
		c.adc(c.read(c.abx()))
	case 0x79:
		c.adc(c.read(c.aby()))
	case 0x61:
		c.adc(c.read(c.izx()))
	case 0x71:
		c.adc(c.read(c.izy()))
	case 0xE9:
		c.sbc(c.fetch())
	case 0xE5:
		c.sbc(c.read(c.zp()))
	case 0xF5:
		c.sbc(c.read(c.zpx()))
	case 0xED:
		c.sbc(c.read(c.abs()))
	case 0xFD:
		c.sbc(c.read(c.abx()))
	case 0xF9:
		c.sbc(c.read(c.aby()))
	case 0xE1:
		c.sbc(c.read(c.izx()))
	case 0xF1:
		c.sbc(c.read(c.izy()))
	// --- logic ---
	case 0x29:
		c.A = c.nz(c.A & c.fetch())
	case 0x25:
		c.A = c.nz(c.A & c.read(c.zp()))
	case 0x35:
		c.A = c.nz(c.A & c.read(c.zpx()))
	case 0x2D:
		c.A = c.nz(c.A & c.read(c.abs()))
	case 0x3D:
		c.A = c.nz(c.A & c.read(c.abx()))
	case 0x39:
		c.A = c.nz(c.A & c.read(c.aby()))
	case 0x21:
		c.A = c.nz(c.A & c.read(c.izx()))
	case 0x31:
		c.A = c.nz(c.A & c.read(c.izy()))
	case 0x09:
		c.A = c.nz(c.A | c.fetch())
	case 0x05:
		c.A = c.nz(c.A | c.read(c.zp()))
	case 0x15:
		c.A = c.nz(c.A | c.read(c.zpx()))
	case 0x0D:
		c.A = c.nz(c.A | c.read(c.abs()))
	case 0x1D:
		c.A = c.nz(c.A | c.read(c.abx()))
	case 0x19:
		c.A = c.nz(c.A | c.read(c.aby()))
	case 0x01:
		c.A = c.nz(c.A | c.read(c.izx()))
	case 0x11:
		c.A = c.nz(c.A | c.read(c.izy()))
	case 0x49:
		c.A = c.nz(c.A ^ c.fetch())
	case 0x45:
		c.A = c.nz(c.A ^ c.read(c.zp()))
	case 0x55:
		c.A = c.nz(c.A ^ c.read(c.zpx()))
	case 0x4D:
		c.A = c.nz(c.A ^ c.read(c.abs()))
	case 0x5D:
		c.A = c.nz(c.A ^ c.read(c.abx()))
	case 0x59:
		c.A = c.nz(c.A ^ c.read(c.aby()))
	case 0x41:
		c.A = c.nz(c.A ^ c.read(c.izx()))
	case 0x51:
		c.A = c.nz(c.A ^ c.read(c.izy()))
	case 0x24:
		c.bit(c.read(c.zp()))
	case 0x2C:
		c.bit(c.read(c.abs()))
	// --- compares ---
	case 0xC9:
		c.cmp(c.A, c.fetch())
	case 0xC5:
		c.cmp(c.A, c.read(c.zp()))
	case 0xD5:
		c.cmp(c.A, c.read(c.zpx()))
	case 0xCD:
		c.cmp(c.A, c.read(c.abs()))
	case 0xDD:
		c.cmp(c.A, c.read(c.abx()))
	case 0xD9:
		c.cmp(c.A, c.read(c.aby()))
	case 0xC1:
		c.cmp(c.A, c.read(c.izx()))
	case 0xD1:
		c.cmp(c.A, c.read(c.izy()))
	case 0xE0:
		c.cmp(c.X, c.fetch())
	case 0xE4:
		c.cmp(c.X, c.read(c.zp()))
	case 0xEC:
		c.cmp(c.X, c.read(c.abs()))
	case 0xC0:
		c.cmp(c.Y, c.fetch())
	case 0xC4:
		c.cmp(c.Y, c.read(c.zp()))
	case 0xCC:
		c.cmp(c.Y, c.read(c.abs()))
	// --- inc/dec ---
	case 0xE6:
		a := c.zp()
		c.write(a, c.nz(c.read(a)+1))
	case 0xF6:
		a := c.zpx()
		c.write(a, c.nz(c.read(a)+1))
	case 0xEE:
		a := c.abs()
		c.write(a, c.nz(c.read(a)+1))
	case 0xFE:
		a := c.abx()
		c.write(a, c.nz(c.read(a)+1))
	case 0xC6:
		a := c.zp()
		c.write(a, c.nz(c.read(a)-1))
	case 0xD6:
		a := c.zpx()
		c.write(a, c.nz(c.read(a)-1))
	case 0xCE:
		a := c.abs()
		c.write(a, c.nz(c.read(a)-1))
	case 0xDE:
		a := c.abx()
		c.write(a, c.nz(c.read(a)-1))
	case 0xE8:
		c.X = c.nz(c.X + 1)
	case 0xC8:
		c.Y = c.nz(c.Y + 1)
	case 0xCA:
		c.X = c.nz(c.X - 1)
	case 0x88:
		c.Y = c.nz(c.Y - 1)
	// --- shifts ---
	case 0x0A:
		c.C = c.A&0x80 != 0
		c.A = c.nz(c.A << 1)
	case 0x06:
		c.aslM(c.zp())
	case 0x16:
		c.aslM(c.zpx())
	case 0x0E:
		c.aslM(c.abs())
	case 0x1E:
		c.aslM(c.abx())
	case 0x4A:
		c.C = c.A&0x01 != 0
		c.A = c.nz(c.A >> 1)
	case 0x46:
		c.lsrM(c.zp())
	case 0x56:
		c.lsrM(c.zpx())
	case 0x4E:
		c.lsrM(c.abs())
	case 0x5E:
		c.lsrM(c.abx())
	case 0x2A:
		old := c.C
		c.C = c.A&0x80 != 0
		c.A <<= 1
		if old {
			c.A |= 1
		}
		c.nz(c.A)
	case 0x26:
		c.rolM(c.zp())
	case 0x36:
		c.rolM(c.zpx())
	case 0x2E:
		c.rolM(c.abs())
	case 0x3E:
		c.rolM(c.abx())
	case 0x6A:
		old := c.C
		c.C = c.A&0x01 != 0
		c.A >>= 1
		if old {
			c.A |= 0x80
		}
		c.nz(c.A)
	case 0x66:
		c.rorM(c.zp())
	case 0x76:
		c.rorM(c.zpx())
	case 0x6E:
		c.rorM(c.abs())
	case 0x7E:
		c.rorM(c.abx())
	// --- jumps ---
	case 0x4C:
		c.PC = c.fetch16()
	case 0x6C:
		a := c.fetch16()
		c.PC = c.read16(a)
	case 0x20:
		a := c.fetch16()
		ret := c.PC - 1
		c.Push(byte(ret >> 8))
		c.Push(byte(ret))
		c.PC = a
	case 0x60:
		lo := c.Pop()
		hi := c.Pop()
		c.PC = (uint16(hi)<<8 | uint16(lo)) + 1
	case 0x40:
		c.setFlags(c.Pop())
		lo := c.Pop()
		hi := c.Pop()
		c.PC = uint16(hi)<<8 | uint16(lo)
	// --- branches ---
	case 0x10:
		c.branch(!c.N)
	case 0x30:
		c.branch(c.N)
	case 0x50:
		c.branch(!c.V)
	case 0x70:
		c.branch(c.V)
	case 0x90:
		c.branch(!c.C)
	case 0xB0:
		c.branch(c.C)
	case 0xD0:
		c.branch(!c.Z)
	case 0xF0:
		c.branch(c.Z)
	// --- flags ---
	case 0x18:
		c.C = false
	case 0x38:
		c.C = true
	case 0x58:
		c.I = false
	case 0x78:
		c.I = true
	case 0xB8:
		c.V = false
	case 0xD8:
		c.D = false
	case 0xF8:
		c.D = true
	default:
		c.Halt("unimplemented opcode $%02X at $%04X", op, c.PC-1)
	}
}
