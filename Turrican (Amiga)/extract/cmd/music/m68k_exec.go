package main

import "fmt"

// step decodes and executes one instruction at the PC. It covers the 68000 subset the
// Turrican sound driver uses; anything else panics with the opcode so it can be added.

func (m *m68k) step() {
	m.insn++
	if m.insn > 50_000_000 {
		panic("instruction runaway — likely a driver infinite loop")
	}
	op := m.fetch16()
	switch op >> 12 {
	case 0x0:
		m.grp0(op)
	case 0x1:
		m.move(op, 1)
	case 0x2:
		m.move(op, 4)
	case 0x3:
		m.move(op, 2)
	case 0x4:
		m.grp4(op)
	case 0x5:
		m.grp5(op)
	case 0x6:
		m.grpBranch(op)
	case 0x7:
		m.d[(op>>9)&7] = uint32(int32(int8(uint8(op)))) // MOVEQ
		m.setNZ(m.d[(op>>9)&7], 4)
	case 0x8:
		m.grp8(op)
	case 0x9:
		m.grpAddSub(op, false) // SUB
	case 0xB:
		m.grpB(op)
	case 0xC:
		m.grpC(op)
	case 0xD:
		m.grpAddSub(op, true) // ADD
	case 0xE:
		m.grpShift(op)
	default:
		panic(fmt.Sprintf("unimplemented opcode %04X @%06X", op, m.pc-2))
	}
}

func opSize(op uint16) int { // size field bits 6-7: 0=byte,1=word,2=long
	switch (op >> 6) & 3 {
	case 0:
		return 1
	case 1:
		return 2
	}
	return 4
}

// ---- arithmetic helpers ----

func (m *m68k) aluAdd(dst, src uint32, size int, withX bool) uint32 {
	bits := uint(size * 8)
	mask := sizeMask(size)
	x := uint32(0)
	if withX && m.getf(flagX) {
		x = 1
	}
	res := (dst & mask) + (src & mask) + x
	r := res & mask
	carry := res&(1<<bits) != 0
	sm := uint32(1) << (bits - 1)
	v := (dst^src)&sm == 0 && (dst^r)&sm != 0
	m.setf(flagC, carry)
	m.setf(flagX, carry)
	m.setf(flagN, r&sm != 0)
	m.setf(flagZ, r == 0)
	m.setf(flagV, v)
	return r
}

func (m *m68k) aluSub(dst, src uint32, size int, withX bool) uint32 {
	bits := uint(size * 8)
	mask := sizeMask(size)
	x := uint32(0)
	if withX && m.getf(flagX) {
		x = 1
	}
	res := (dst & mask) - (src & mask) - x
	r := res & mask
	borrow := (src&mask)+x > (dst & mask)
	sm := uint32(1) << (bits - 1)
	v := (dst^src)&sm != 0 && (dst^r)&sm != 0
	m.setf(flagC, borrow)
	m.setf(flagX, borrow)
	m.setf(flagN, r&sm != 0)
	m.setf(flagZ, r == 0)
	m.setf(flagV, v)
	return r
}

// aluCmp is SUB without storing and without touching X.
func (m *m68k) aluCmp(dst, src uint32, size int) {
	x := m.getf(flagX)
	m.aluSub(dst, src, size, false)
	m.setf(flagX, x)
}

// ---- group 0: immediate ops + bit ops ----

func (m *m68k) grp0(op uint16) {
	if op&0x0100 != 0 || (op>>8)&0xF == 8 {
		// bit ops: BTST/BCHG/BCLR/BSET (dynamic via Dn, or static via #imm when (op>>8)==8)
		m.bitOp(op)
		return
	}
	size := opSize(op)
	imm := m.immFetch(size)
	e := m.decodeEA(int((op>>3)&7), int(op&7), size)
	switch (op >> 9) & 7 {
	case 0: // ORI
		r := m.eaLoad(e, size) | imm
		m.eaStore(e, size, r)
		m.setNZ(r, size)
	case 1: // ANDI
		r := m.eaLoad(e, size) & imm
		m.eaStore(e, size, r)
		m.setNZ(r, size)
	case 2: // SUBI
		r := m.aluSub(m.eaLoad(e, size), imm, size, false)
		m.eaStore(e, size, r)
	case 3: // ADDI
		r := m.aluAdd(m.eaLoad(e, size), imm, size, false)
		m.eaStore(e, size, r)
	case 5: // EORI
		r := m.eaLoad(e, size) ^ imm
		m.eaStore(e, size, r)
		m.setNZ(r, size)
	case 6: // CMPI
		m.aluCmp(m.eaLoad(e, size), imm, size)
	default:
		panic(fmt.Sprintf("grp0 %04X @%06X", op, m.pc-2))
	}
}

func (m *m68k) immFetch(size int) uint32 {
	if size == 4 {
		return m.fetch32()
	}
	w := m.fetch16()
	if size == 1 {
		return uint32(w & 0xFF)
	}
	return uint32(w)
}

func (m *m68k) bitOp(op uint16) {
	var bit uint32
	if (op>>8)&0xF == 8 { // static: bit number is immediate
		bit = uint32(m.fetch16())
	} else { // dynamic: bit number in Dn
		bit = m.d[(op>>9)&7]
	}
	mode := int((op >> 3) & 7)
	e := m.decodeEA(mode, int(op&7), 4)
	size := 4
	if mode != 0 { // memory: byte operand, bit mod 8
		size = 1
		bit &= 7
	} else {
		bit &= 31
	}
	val := m.eaLoad(e, size)
	m.setf(flagZ, val&(1<<bit) == 0)
	switch (op >> 6) & 3 {
	case 0: // BTST — no change
	case 1: // BCHG
		m.eaStore(e, size, val^(1<<bit))
	case 2: // BCLR
		m.eaStore(e, size, val&^(1<<bit))
	case 3: // BSET
		m.eaStore(e, size, val|(1<<bit))
	}
}

// ---- MOVE / MOVEA ----

func (m *m68k) move(op uint16, size int) {
	src := m.decodeEA(int((op>>3)&7), int(op&7), size)
	v := m.eaLoad(src, size)
	dmode := int((op >> 6) & 7)
	dreg := int((op >> 9) & 7)
	if dmode == 1 { // MOVEA
		if size == 2 {
			m.a[dreg] = uint32(int32(int16(v)))
		} else {
			m.a[dreg] = v
		}
		return
	}
	dst := m.decodeEA(dmode, dreg, size)
	m.eaStore(dst, size, v)
	m.setNZ(v, size)
}

// ---- group 4: misc ----

func (m *m68k) grp4(op uint16) {
	switch {
	case op == 0x4E75: // RTS
		m.pc = m.rd32(m.a[7])
		m.a[7] += 4
	case op == 0x4E71: // NOP
	case op == 0x4E73: // RTE
		m.sr = m.rd16(m.a[7])
		m.a[7] += 2
		m.pc = m.rd32(m.a[7])
		m.a[7] += 4
	case op&0xFFC0 == 0x4EC0: // JMP
		e := m.decodeEA(int((op>>3)&7), int(op&7), 4)
		m.pc = e.addr
	case op&0xFFC0 == 0x4E80: // JSR
		e := m.decodeEA(int((op>>3)&7), int(op&7), 4)
		m.a[7] -= 4
		m.wr32(m.a[7], m.pc)
		m.pc = e.addr
	case op&0xF1C0 == 0x41C0: // LEA
		e := m.decodeEA(int((op>>3)&7), int(op&7), 4)
		m.a[(op>>9)&7] = e.addr
	case op&0xFFC0 == 0x44C0: // MOVE to CCR
		e := m.decodeEA(int((op>>3)&7), int(op&7), 2)
		m.sr = m.sr&0xFF00 | uint16(m.eaLoad(e, 2))&0x1F
	case op&0xFFC0 == 0x46C0: // MOVE to SR
		e := m.decodeEA(int((op>>3)&7), int(op&7), 2)
		m.sr = uint16(m.eaLoad(e, 2))
	case op&0xFFC0 == 0x40C0: // MOVE from SR
		e := m.decodeEA(int((op>>3)&7), int(op&7), 2)
		m.eaStore(e, 2, uint32(m.sr))
	case op&0xFF00 == 0x4200: // CLR
		size := opSize(op)
		e := m.decodeEA(int((op>>3)&7), int(op&7), size)
		m.eaStore(e, size, 0)
		m.setNZ(0, size)
	case op&0xFF00 == 0x4400: // NEG
		size := opSize(op)
		e := m.decodeEA(int((op>>3)&7), int(op&7), size)
		r := m.aluSub(0, m.eaLoad(e, size), size, false)
		m.eaStore(e, size, r)
	case op&0xFF00 == 0x4600: // NOT
		size := opSize(op)
		e := m.decodeEA(int((op>>3)&7), int(op&7), size)
		r := ^m.eaLoad(e, size) & sizeMask(size)
		m.eaStore(e, size, r)
		m.setNZ(r, size)
	case op&0xFF00 == 0x4A00: // TST
		size := opSize(op)
		e := m.decodeEA(int((op>>3)&7), int(op&7), size)
		m.setNZ(m.eaLoad(e, size), size)
	case op&0xFFB8 == 0x4880: // EXT
		reg := int(op & 7)
		if op&0x40 != 0 { // EXT.l (word->long)
			m.d[reg] = uint32(int32(int16(m.d[reg])))
			m.setNZ(m.d[reg], 4)
		} else { // EXT.w (byte->word)
			v := uint32(int32(int8(uint8(m.d[reg])))) & 0xFFFF
			m.d[reg] = m.d[reg]&0xFFFF0000 | v
			m.setNZ(v, 2)
		}
	case op&0xFFF8 == 0x4840: // SWAP
		reg := int(op & 7)
		m.d[reg] = m.d[reg]<<16 | m.d[reg]>>16
		m.setNZ(m.d[reg], 4)
	case op&0xFFC0 == 0x4840: // PEA
		e := m.decodeEA(int((op>>3)&7), int(op&7), 4)
		m.a[7] -= 4
		m.wr32(m.a[7], e.addr)
	case op&0xFB80 == 0x4880: // MOVEM
		m.movem(op)
	default:
		panic(fmt.Sprintf("grp4 %04X @%06X", op, m.pc-2))
	}
}

func (m *m68k) movem(op uint16) {
	dir := (op >> 10) & 1 // 0 = reg->mem, 1 = mem->reg
	sz := 2
	if op&0x40 != 0 {
		sz = 4
	}
	mask := m.fetch16()
	mode := int((op >> 3) & 7)
	reg := int(op & 7)
	if dir == 0 && mode == 4 { // reg->mem, predecrement
		addr := m.a[reg]
		for i := 0; i < 16; i++ {
			if mask&(1<<uint(i)) != 0 {
				addr -= uint32(sz)
				var val uint32
				if i < 8 {
					val = m.a[7-i]
				} else {
					val = m.d[15-i]
				}
				if sz == 4 {
					m.wr32(addr, val)
				} else {
					m.wr16(addr, uint16(val))
				}
			}
		}
		m.a[reg] = addr
		return
	}
	if dir == 1 && mode == 3 { // mem->reg, postincrement
		addr := m.a[reg]
		for i := 0; i < 16; i++ {
			if mask&(1<<uint(i)) != 0 {
				var val uint32
				if sz == 4 {
					val = m.rd32(addr)
				} else {
					val = uint32(int32(int16(m.rd16(addr))))
				}
				if i < 8 {
					m.d[i] = val
				} else {
					m.a[i-8] = val
				}
				addr += uint32(sz)
			}
		}
		m.a[reg] = addr
		return
	}
	// control modes: ascending addresses, order D0..A7
	e := m.decodeEA(mode, reg, sz)
	addr := e.addr
	for i := 0; i < 16; i++ {
		if mask&(1<<uint(i)) != 0 {
			if dir == 0 {
				var val uint32
				if i < 8 {
					val = m.d[i]
				} else {
					val = m.a[i-8]
				}
				if sz == 4 {
					m.wr32(addr, val)
				} else {
					m.wr16(addr, uint16(val))
				}
			} else {
				var val uint32
				if sz == 4 {
					val = m.rd32(addr)
				} else {
					val = uint32(int32(int16(m.rd16(addr))))
				}
				if i < 8 {
					m.d[i] = val
				} else {
					m.a[i-8] = val
				}
			}
			addr += uint32(sz)
		}
	}
}

// ---- group 5: ADDQ/SUBQ/Scc/DBcc ----

func (m *m68k) grp5(op uint16) {
	if (op>>6)&3 == 3 { // Scc or DBcc
		if (op>>3)&7 == 1 { // DBcc
			cc := (op >> 8) & 0xF
			reg := int(op & 7)
			disp := int32(int16(m.fetch16()))
			base := m.pc - 2
			if !m.testCC(cc) {
				lo := uint16(m.d[reg]) - 1
				m.d[reg] = m.d[reg]&0xFFFF0000 | uint32(lo)
				if lo != 0xFFFF {
					m.pc = uint32(int32(base) + disp)
				}
			}
			return
		}
		// Scc
		cc := (op >> 8) & 0xF
		e := m.decodeEA(int((op>>3)&7), int(op&7), 1)
		if m.testCC(cc) {
			m.eaStore(e, 1, 0xFF)
		} else {
			m.eaStore(e, 1, 0)
		}
		return
	}
	size := opSize(op)
	data := uint32((op >> 9) & 7)
	if data == 0 {
		data = 8
	}
	mode := int((op >> 3) & 7)
	e := m.decodeEA(mode, int(op&7), size)
	if op&0x0100 == 0 { // ADDQ
		if mode == 1 { // An: full add, no flags
			m.a[e.idx] += data
			return
		}
		m.eaStore(e, size, m.aluAdd(m.eaLoad(e, size), data, size, false))
	} else { // SUBQ
		if mode == 1 {
			m.a[e.idx] -= data
			return
		}
		m.eaStore(e, size, m.aluSub(m.eaLoad(e, size), data, size, false))
	}
}

// ---- group 6: Bcc / BSR / BRA ----

func (m *m68k) grpBranch(op uint16) {
	cc := (op >> 8) & 0xF
	disp := int32(int8(uint8(op)))
	base := m.pc
	if uint8(op) == 0 {
		disp = int32(int16(m.fetch16()))
	} else if uint8(op) == 0xFF {
		disp = int32(m.fetch32())
	}
	target := uint32(int32(base) + disp)
	switch cc {
	case 0: // BRA
		m.pc = target
	case 1: // BSR
		m.a[7] -= 4
		m.wr32(m.a[7], m.pc)
		m.pc = target
	default:
		if m.testCC(cc) {
			m.pc = target
		}
	}
}

func (m *m68k) testCC(cc uint16) bool {
	c, v, z, n := m.getf(flagC), m.getf(flagV), m.getf(flagZ), m.getf(flagN)
	switch cc {
	case 0x0:
		return true
	case 0x1:
		return false
	case 0x2:
		return !c && !z // HI
	case 0x3:
		return c || z // LS
	case 0x4:
		return !c // CC
	case 0x5:
		return c // CS
	case 0x6:
		return !z // NE
	case 0x7:
		return z // EQ
	case 0x8:
		return !v // VC
	case 0x9:
		return v // VS
	case 0xA:
		return !n // PL
	case 0xB:
		return n // MI
	case 0xC:
		return n == v // GE
	case 0xD:
		return n != v // LT
	case 0xE:
		return !z && n == v // GT
	case 0xF:
		return z || n != v // LE
	}
	return false
}

// ---- group 8: OR / DIVU / DIVS ----

func (m *m68k) grp8(op uint16) {
	if (op>>6)&3 == 3 { // DIVU/DIVS
		e := m.decodeEA(int((op>>3)&7), int(op&7), 2)
		divisor := m.eaLoad(e, 2)
		reg := (op >> 9) & 7
		if divisor == 0 {
			return // division by zero: trap ignored
		}
		dividend := m.d[reg]
		if op&0x0100 != 0 { // DIVS
			q := int32(dividend) / int32(int16(divisor))
			r := int32(dividend) % int32(int16(divisor))
			m.d[reg] = uint32(uint16(r))<<16 | uint32(uint16(q))
		} else { // DIVU
			q := dividend / divisor
			r := dividend % divisor
			m.d[reg] = (r&0xFFFF)<<16 | (q & 0xFFFF)
		}
		return
	}
	size := opSize(op)
	reg := (op >> 9) & 7
	e := m.decodeEA(int((op>>3)&7), int(op&7), size)
	if op&0x0100 == 0 { // OR <ea>,Dn
		r := (m.d[reg] | m.eaLoad(e, size)) & sizeMask(size)
		m.d[reg] = m.d[reg]&^sizeMask(size) | r
		m.setNZ(r, size)
	} else { // OR Dn,<ea>
		r := (m.eaLoad(e, size) | m.d[reg]) & sizeMask(size)
		m.eaStore(e, size, r)
		m.setNZ(r, size)
	}
}

// ---- group 9/D: SUB/ADD (incl. SUBA/ADDA) ----

func (m *m68k) grpAddSub(op uint16, isAdd bool) {
	if (op>>6)&3 == 3 { // SUBA/ADDA (word/long to An)
		size := 2
		if op&0x0100 != 0 {
			size = 4
		}
		e := m.decodeEA(int((op>>3)&7), int(op&7), size)
		v := m.eaLoad(e, size)
		if size == 2 {
			v = uint32(int32(int16(v)))
		}
		reg := (op >> 9) & 7
		if isAdd {
			m.a[reg] += v
		} else {
			m.a[reg] -= v
		}
		return
	}
	size := opSize(op)
	reg := (op >> 9) & 7
	e := m.decodeEA(int((op>>3)&7), int(op&7), size)
	if op&0x0100 == 0 { // <ea> op Dn -> Dn
		var r uint32
		if isAdd {
			r = m.aluAdd(m.d[reg], m.eaLoad(e, size), size, false)
		} else {
			r = m.aluSub(m.d[reg], m.eaLoad(e, size), size, false)
		}
		m.d[reg] = m.d[reg]&^sizeMask(size) | r&sizeMask(size)
	} else { // Dn op <ea> -> <ea>
		var r uint32
		if isAdd {
			r = m.aluAdd(m.eaLoad(e, size), m.d[reg], size, false)
		} else {
			r = m.aluSub(m.eaLoad(e, size), m.d[reg], size, false)
		}
		m.eaStore(e, size, r)
	}
}

// ---- group B: CMP / CMPA / EOR ----

func (m *m68k) grpB(op uint16) {
	if (op>>6)&3 == 3 { // CMPA
		size := 2
		if op&0x0100 != 0 {
			size = 4
		}
		e := m.decodeEA(int((op>>3)&7), int(op&7), size)
		v := m.eaLoad(e, size)
		if size == 2 {
			v = uint32(int32(int16(v)))
		}
		m.aluCmp(m.a[(op>>9)&7], v, 4)
		return
	}
	size := opSize(op)
	reg := (op >> 9) & 7
	e := m.decodeEA(int((op>>3)&7), int(op&7), size)
	if op&0x0100 == 0 { // CMP <ea>,Dn
		m.aluCmp(m.d[reg], m.eaLoad(e, size), size)
	} else { // EOR Dn,<ea>
		r := (m.eaLoad(e, size) ^ m.d[reg]) & sizeMask(size)
		m.eaStore(e, size, r)
		m.setNZ(r, size)
	}
}

// ---- group C: AND / MULU / MULS / EXG ----

func (m *m68k) grpC(op uint16) {
	if (op>>6)&3 == 3 { // MULU/MULS
		e := m.decodeEA(int((op>>3)&7), int(op&7), 2)
		src := m.eaLoad(e, 2)
		reg := (op >> 9) & 7
		if op&0x0100 != 0 { // MULS
			m.d[reg] = uint32(int32(int16(m.d[reg])) * int32(int16(src)))
		} else { // MULU
			m.d[reg] = (m.d[reg] & 0xFFFF) * (src & 0xFFFF)
		}
		m.setNZ(m.d[reg], 4)
		return
	}
	if op&0x0130 == 0x0100 { // EXG
		rx := (op >> 9) & 7
		ry := op & 7
		switch (op >> 3) & 0x1F {
		case 0x08: // EXG Dx,Dy
			m.d[rx], m.d[ry] = m.d[ry], m.d[rx]
		case 0x09: // EXG Ax,Ay
			m.a[rx], m.a[ry] = m.a[ry], m.a[rx]
		case 0x11: // EXG Dx,Ay
			m.d[rx], m.a[ry] = m.a[ry], m.d[rx]
		}
		return
	}
	size := opSize(op)
	reg := (op >> 9) & 7
	e := m.decodeEA(int((op>>3)&7), int(op&7), size)
	if op&0x0100 == 0 { // AND <ea>,Dn
		r := (m.d[reg] & m.eaLoad(e, size)) & sizeMask(size)
		m.d[reg] = m.d[reg]&^sizeMask(size) | r
		m.setNZ(r, size)
	} else { // AND Dn,<ea>
		r := (m.eaLoad(e, size) & m.d[reg]) & sizeMask(size)
		m.eaStore(e, size, r)
		m.setNZ(r, size)
	}
}

// ---- group E: shifts / rotates ----

func (m *m68k) grpShift(op uint16) {
	if (op>>6)&3 == 3 { // memory shift by 1
		e := m.decodeEA(int((op>>3)&7), int(op&7), 2)
		v := m.eaLoad(e, 2)
		v = m.doShift(op, (op>>8)&0xFF&0x9|uint16(boolToInt(op&0x100 != 0)), v, 1, 2)
		m.eaStore(e, 2, v)
		return
	}
	size := opSize(op)
	reg := int(op & 7)
	var cnt int
	if op&0x20 != 0 { // count in register
		cnt = int(m.d[(op>>9)&7] & 63)
	} else {
		cnt = int((op >> 9) & 7)
		if cnt == 0 {
			cnt = 8
		}
	}
	typ := (op >> 3) & 3 // 0=AS,1=LS,2=ROX,3=RO
	left := op&0x0100 != 0
	r := m.doShiftReg(typ, left, m.d[reg]&sizeMask(size), cnt, size)
	m.d[reg] = m.d[reg]&^sizeMask(size) | r&sizeMask(size)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// doShift handles the memory-shift-by-1 forms (only LSL/LSR/ASL/ASR needed).
func (m *m68k) doShift(op, _ uint16, v uint32, cnt, size int) uint32 {
	typ := (op >> 9) & 3
	left := op&0x0100 != 0
	return m.doShiftReg(typ, left, v, cnt, size)
}

func (m *m68k) doShiftReg(typ uint16, left bool, v uint32, cnt, size int) uint32 {
	bits := uint(size * 8)
	mask := sizeMask(size)
	sm := uint32(1) << (bits - 1)
	v &= mask
	var lastC bool
	for i := 0; i < cnt; i++ {
		switch typ {
		case 0: // arithmetic
			if left {
				lastC = v&sm != 0
				v = (v << 1) & mask
			} else {
				lastC = v&1 != 0
				v = (v >> 1) | (v & sm) // keep sign
			}
		case 1: // logical
			if left {
				lastC = v&sm != 0
				v = (v << 1) & mask
			} else {
				lastC = v&1 != 0
				v >>= 1
			}
		case 3: // rotate (no extend)
			if left {
				lastC = v&sm != 0
				v = ((v << 1) | (v >> (bits - 1))) & mask
			} else {
				lastC = v&1 != 0
				v = ((v >> 1) | (v << (bits - 1))) & mask
			}
		case 2: // rotate with extend
			xin := uint32(0)
			if m.getf(flagX) {
				xin = 1
			}
			if left {
				lastC = v&sm != 0
				v = ((v << 1) | xin) & mask
			} else {
				lastC = v&1 != 0
				v = (v >> 1) | (xin << (bits - 1))
			}
			m.setf(flagX, lastC)
		}
	}
	if cnt > 0 {
		m.setf(flagC, lastC)
		if typ != 3 {
			m.setf(flagX, lastC)
		}
	} else {
		m.setf(flagC, false)
	}
	m.setf(flagN, v&sm != 0)
	m.setf(flagZ, v&mask == 0)
	m.setf(flagV, false)
	return v & mask
}
