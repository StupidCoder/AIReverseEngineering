package m68k

// Step executes one instruction. Unimplemented opcodes halt the CPU.
func (c *CPU) Step() {
	if c.Halted {
		return
	}
	start := c.PC
	op := uint16(c.fetch16())
	switch op >> 12 {
	case 0x0:
		c.execImmediate(op)
	case 0x1, 0x2, 0x3:
		c.execMove(op)
	case 0x4:
		c.execGroup4(op)
	case 0x5:
		c.execGroup5(op, start)
	case 0x6:
		c.execBranch(op, start)
	case 0x7:
		c.execMoveq(op)
	case 0x8:
		c.execLogic(op, '|')
	case 0x9:
		c.execAddSub(op, false)
	case 0xB:
		c.execCmpEor(op)
	case 0xC:
		c.execLogic(op, '&')
	case 0xD:
		c.execAddSub(op, true)
	case 0xE:
		c.execShift(op)
	default:
		c.Halt("unimplemented opcode $%04X at $%06X", op, start)
	}
}

func fields(op uint16) (mode, reg, reg2, size int) {
	return int(op>>3) & 7, int(op) & 7, int(op>>9) & 7, int(op>>6) & 3
}

func b2i(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

func (c *CPU) regVal(i int) uint32 {
	if i < 8 {
		return c.D[i]
	}
	return c.A[i-8]
}
func (c *CPU) setReg(i int, v uint32) {
	if i < 8 {
		c.D[i] = v
	} else {
		c.A[i-8] = v
	}
}

// --- MOVE / MOVEA / MOVEQ ---

var moveSize = map[uint16]int{1: 0, 3: 1, 2: 2}

func (c *CPU) execMove(op uint16) {
	size := moveSize[op>>12]
	mode, reg, reg2, _ := fields(op)
	src := c.resolveEA(mode, reg, size)
	v := c.load(src, size)
	dmode := int(op>>6) & 7
	if dmode == 1 { // MOVEA: sign-extend to long, no flags
		c.A[reg2] = signExtend(v, size)
		return
	}
	dst := c.resolveEA(dmode, reg2, size)
	c.store(dst, size, v)
	c.setLogic(v, size)
}

func (c *CPU) execMoveq(op uint16) {
	if op&0x0100 != 0 {
		c.Halt("illegal MOVEQ $%04X", op)
		return
	}
	v := signExtend(uint32(byte(op)), 0)
	c.D[int(op>>9)&7] = v
	c.setLogic(v, 2)
}

// --- immediate ALU group (ORI/ANDI/SUBI/ADDI/EORI/CMPI) ---

func (c *CPU) execImmediate(op uint16) {
	mode, reg, reg2, size := fields(op)
	// Bit operations (BTST/BCHG/BCLR/BSET) live in group 0:
	//   static  0000 1000 ttmmmrrr  (bit number in the immediate word)
	//   dynamic 0000 rrr1 ttmmmrrr  (bit number in Dn), mode!=001 (else MOVEP)
	static := op&0xFF00 == 0x0800
	dynamic := op&0x0100 != 0 && mode != 1
	if static || dynamic {
		var bit uint32
		if static {
			bit = c.fetch16()
		} else {
			bit = c.D[reg2]
		}
		which := (op >> 6) & 3 // 0=BTST 1=BCHG 2=BCLR 3=BSET
		if mode == 0 {         // register: 32-bit, bit mod 32
			bit &= 31
			d := c.D[reg]
			c.Z = d&(1<<bit) == 0
			switch which {
			case 1:
				c.D[reg] = d ^ (1 << bit)
			case 2:
				c.D[reg] = d &^ (1 << bit)
			case 3:
				c.D[reg] = d | (1 << bit)
			}
		} else { // memory: byte, bit mod 8
			bit &= 7
			ea := c.resolveEA(mode, reg, 0)
			d := c.load(ea, 0)
			c.Z = d&(1<<bit) == 0
			switch which {
			case 1:
				c.store(ea, 0, d^(1<<bit))
			case 2:
				c.store(ea, 0, d&^(1<<bit))
			case 3:
				c.store(ea, 0, d|(1<<bit))
			}
		}
		return
	}
	if op&0x0100 != 0 || size == 3 {
		c.Halt("unimplemented bit/movep opcode $%04X at $%06X", op, c.PC-2)
		return
	}
	names := map[int]byte{0: '|', 1: '&', 2: '-', 3: '+', 5: '^', 6: 'c'}
	kind, ok := names[reg2]
	if !ok {
		c.Halt("unimplemented opcode $%04X at $%06X", op, c.PC-2)
		return
	}
	var imm uint32
	if size == 2 {
		imm = c.fetch32()
	} else {
		imm = c.fetch16()
		if size == 0 {
			imm &= 0xFF
		}
	}
	dst := c.resolveEA(mode, reg, size)
	d := c.load(dst, size)
	switch kind {
	case '|':
		r := d | imm
		c.store(dst, size, r)
		c.setLogic(r, size)
	case '&':
		r := d & imm
		c.store(dst, size, r)
		c.setLogic(r, size)
	case '^':
		r := d ^ imm
		c.store(dst, size, r)
		c.setLogic(r, size)
	case '+':
		c.store(dst, size, c.add(d, imm, size))
	case '-':
		c.store(dst, size, c.sub(d, imm, size))
	case 'c': // CMPI
		c.cmp(d, imm, size)
	}
}

// --- ADD/ADDA/ADDQ and SUB/SUBA/SUBQ register/EA forms ---

func (c *CPU) execAddSub(op uint16, isAdd bool) {
	mode, reg, reg2, size := fields(op)
	opm := int(op>>6) & 7
	if opm == 3 || opm == 7 { // ADDA/SUBA
		sz := 1
		if opm == 7 {
			sz = 2
		}
		src := c.resolveEA(mode, reg, sz)
		v := signExtend(c.load(src, sz), sz)
		if isAdd {
			c.A[reg2] += v
		} else {
			c.A[reg2] -= v
		}
		return
	}
	if op&0x0130 == 0x0100 { // ADDX/SUBX
		c.Halt("unimplemented ADDX/SUBX $%04X at $%06X", op, c.PC-2)
		return
	}
	ea := c.resolveEA(mode, reg, size)
	dnOp := operand{kind: 0, reg: reg2}
	if op&0x0100 != 0 { // <ea> = <ea> op Dn
		var r uint32
		if isAdd {
			r = c.add(c.load(ea, size), c.D[reg2], size)
		} else {
			r = c.sub(c.load(ea, size), c.D[reg2], size)
		}
		c.store(ea, size, r)
	} else { // Dn = Dn op <ea>
		var r uint32
		if isAdd {
			r = c.add(c.D[reg2], c.load(ea, size), size)
		} else {
			r = c.sub(c.D[reg2], c.load(ea, size), size)
		}
		c.store(dnOp, size, r)
	}
}

// --- AND/OR (and the MUL/DIV/EXG slots we don't implement) ---

func (c *CPU) execLogic(op uint16, kind byte) {
	mode, reg, reg2, size := fields(op)
	opm := int(op>>6) & 7
	if opm == 3 || opm == 7 { // MUL/DIV
		c.Halt("unimplemented MUL/DIV $%04X at $%06X", op, c.PC-2)
		return
	}
	if op&0x0130 == 0x0100 && (op&0x00C0) != 0x00C0 { // ABCD/SBCD/EXG region
		if kind == '&' { // EXG lives in the AND ($Cxxx) group
			switch (op >> 3) & 0x1F {
			case 0x08: // EXG Dx,Dy
				c.D[reg2], c.D[reg] = c.D[reg], c.D[reg2]
				return
			case 0x09: // EXG Ax,Ay
				c.A[reg2], c.A[reg] = c.A[reg], c.A[reg2]
				return
			case 0x11: // EXG Dx,Ay
				c.D[reg2], c.A[reg] = c.A[reg], c.D[reg2]
				return
			}
		}
		c.Halt("unimplemented opcode $%04X at $%06X", op, c.PC-2)
		return
	}
	ea := c.resolveEA(mode, reg, size)
	apply := func(a, b uint32) uint32 {
		if kind == '&' {
			return a & b
		}
		return a | b
	}
	if op&0x0100 != 0 { // <ea> = <ea> op Dn
		r := apply(c.load(ea, size), c.D[reg2])
		c.store(ea, size, r)
		c.setLogic(r, size)
	} else { // Dn = Dn op <ea>
		r := apply(c.D[reg2], c.load(ea, size))
		c.store(operand{kind: 0, reg: reg2}, size, r)
		c.setLogic(r, size)
	}
}

// --- CMP/CMPA/EOR ---

func (c *CPU) execCmpEor(op uint16) {
	mode, reg, reg2, size := fields(op)
	opm := int(op>>6) & 7
	if opm == 3 || opm == 7 { // CMPA
		sz := 1
		if opm == 7 {
			sz = 2
		}
		src := signExtend(c.load(c.resolveEA(mode, reg, sz), sz), sz)
		c.cmp(c.A[reg2], src, 2)
		return
	}
	if op&0x0100 != 0 { // EOR Dn,<ea>  (or CMPM, which we skip)
		if mode == 1 {
			c.Halt("unimplemented CMPM $%04X at $%06X", op, c.PC-2)
			return
		}
		ea := c.resolveEA(mode, reg, size)
		r := c.load(ea, size) ^ c.D[reg2]
		c.store(ea, size, r)
		c.setLogic(r, size)
		return
	}
	// CMP <ea>,Dn
	ea := c.resolveEA(mode, reg, size)
	c.cmp(c.D[reg2], c.load(ea, size), size)
}

// --- shifts and rotates (register forms) ---

func (c *CPU) execShift(op uint16) {
	mode, reg, reg2, size := fields(op)
	left := op&0x0100 != 0
	if size == 3 {
		// memory shift by one
		typ := int(op>>9) & 3
		ea := c.resolveEA(mode, reg, 1)
		v := c.doShift(c.load(ea, 1), 1, 1, typ, left)
		c.store(ea, 1, v)
		return
	}
	typ := int(op>>3) & 3
	var count int
	if op&0x0020 != 0 {
		count = int(c.D[reg2] & 63)
	} else if reg2 == 0 {
		count = 8
	} else {
		count = reg2
	}
	c.D[reg] = (c.D[reg] &^ sizeMask(size)) | c.doShift(c.D[reg], count, size, typ, left)
}

func (c *CPU) doShift(val uint32, count, size, typ int, left bool) uint32 {
	mask, bits := sizeMask(size), sizeBits(size)
	val &= mask
	c.V = false
	last := false
	switch typ {
	case 0: // arithmetic
		if left {
			msb0 := signBit(val, size)
			for i := 0; i < count; i++ {
				last = val&(1<<uint(bits-1)) != 0
				val = (val << 1) & mask
				if signBit(val, size) != msb0 {
					c.V = true
				}
			}
		} else {
			for i := 0; i < count; i++ {
				last = val&1 != 0
				val = (val >> 1) | (val & (1 << uint(bits-1)))
			}
		}
	case 1: // logical
		if left {
			for i := 0; i < count; i++ {
				last = val&(1<<uint(bits-1)) != 0
				val = (val << 1) & mask
			}
		} else {
			for i := 0; i < count; i++ {
				last = val&1 != 0
				val >>= 1
			}
		}
	case 3: // rotate without X
		for i := 0; i < count; i++ {
			if left {
				last = val&(1<<uint(bits-1)) != 0
				val = ((val << 1) | b2i(last)) & mask
			} else {
				last = val&1 != 0
				val = ((val >> 1) | (b2i(last) << uint(bits-1))) & mask
			}
		}
	case 2: // rotate through X
		x := c.X
		for i := 0; i < count; i++ {
			if left {
				last = val&(1<<uint(bits-1)) != 0
				val = ((val << 1) | b2i(x)) & mask
			} else {
				last = val&1 != 0
				val = ((val >> 1) | (b2i(x) << uint(bits-1))) & mask
			}
			x = last
		}
		if count == 0 {
			last = c.X
		}
		c.X = last
	}
	switch typ {
	case 0, 1: // ASx/LSx: X and C take the last bit out (X unchanged on count 0)
		if count > 0 {
			c.C, c.X = last, last
		} else {
			c.C = false
		}
	case 3: // ROx: C from last bit; count 0 clears C
		c.C = count > 0 && last
	case 2: // ROXx: C mirrors X
		c.C = last
	}
	c.setNZ(val, size)
	return val
}

// --- ADDQ/SUBQ, DBcc (and Scc, which we skip) ---

func (c *CPU) execGroup5(op uint16, start uint32) {
	mode, reg, reg2, size := fields(op)
	if size == 3 {
		if mode == 1 { // DBcc Dn,disp
			cc := int(op>>8) & 0xF
			disp := uint32(int32(int16(uint16(c.fetch16()))))
			target := start + 2 + disp
			if c.cond(cc) {
				return // condition met: fall through
			}
			dw := (c.D[reg] - 1) & 0xFFFF
			c.D[reg] = (c.D[reg] &^ 0xFFFF) | dw
			if dw != 0xFFFF {
				c.PC = target
			}
			return
		}
		c.Halt("unimplemented Scc $%04X at $%06X", op, start)
		return
	}
	data := reg2
	if data == 0 {
		data = 8
	}
	sub := op&0x0100 != 0
	ea := c.resolveEA(mode, reg, size)
	if ea.kind == 1 { // ADDQ/SUBQ to An: full 32-bit, no flags
		if sub {
			c.A[ea.reg] -= uint32(data)
		} else {
			c.A[ea.reg] += uint32(data)
		}
		return
	}
	d := c.load(ea, size)
	if sub {
		c.store(ea, size, c.sub(d, uint32(data), size))
	} else {
		c.store(ea, size, c.add(d, uint32(data), size))
	}
}

// --- Bcc / BRA / BSR ---

func (c *CPU) execBranch(op uint16, start uint32) {
	cc := int(op>>8) & 0xF
	base := start + 2
	var target uint32
	if byte(op) == 0 {
		target = base + uint32(int32(int16(uint16(c.fetch16()))))
	} else {
		target = base + uint32(int32(int8(byte(op))))
	}
	switch cc {
	case 0: // BRA
		c.PC = target
	case 1: // BSR
		c.push32(c.PC)
		c.PC = target
	default:
		if c.cond(cc) {
			c.PC = target
		}
	}
}

// --- line-4 miscellaneous (the runnable subset) ---

func (c *CPU) execGroup4(op uint16) {
	mode, reg, reg2, size := fields(op)
	start := c.PC - 2
	switch {
	case op == 0x4E71: // NOP
		return
	case op == 0x4E75: // RTS
		c.PC = c.pop32()
		return
	case op&0xFFF8 == 0x4E50: // LINK An,#disp
		d := uint32(int32(int16(uint16(c.fetch16()))))
		c.push32(c.A[reg])
		c.A[reg] = c.A[7]
		c.A[7] += d
		return
	case op&0xFFF8 == 0x4E58: // UNLK An
		c.A[7] = c.A[reg]
		c.A[reg] = c.pop32()
		return
	case op&0xFFC0 == 0x4E80: // JSR <ea>
		ea := c.resolveEA(mode, reg, 0)
		c.push32(c.PC)
		c.PC = ea.addr
		return
	case op&0xFFC0 == 0x4EC0: // JMP <ea>
		ea := c.resolveEA(mode, reg, 0)
		c.PC = ea.addr
		return
	case op&0xF1C0 == 0x41C0: // LEA <ea>,An
		c.A[reg2] = c.resolveEA(mode, reg, 2).addr
		return
	case op&0xFFF8 == 0x4840: // SWAP Dn
		v := c.D[reg]
		v = v>>16 | v<<16
		c.D[reg] = v
		c.setLogic(v, 2)
		return
	case op&0xFFF8 == 0x4880 && mode == 0: // EXT.W Dn
		v := (c.D[reg] &^ 0xFFFF) | (signExtend(c.D[reg], 0) & 0xFFFF)
		c.D[reg] = v
		c.setLogic(v, 1)
		return
	case op&0xFFF8 == 0x48C0 && mode == 0: // EXT.L Dn
		c.D[reg] = signExtend(c.D[reg], 1)
		c.setLogic(c.D[reg], 2)
		return
	case op&0xFFC0 == 0x4840: // PEA <ea>
		c.push32(c.resolveEA(mode, reg, 2).addr)
		return
	case op&0xFB80 == 0x4880: // MOVEM
		c.movem(op)
		return
	case op&0xFF00 == 0x4600 && size != 3: // NOT
		ea := c.resolveEA(mode, reg, size)
		r := ^c.load(ea, size)
		c.store(ea, size, r)
		c.setLogic(r, size)
		return
	case op&0xFF00 == 0x4400 && size != 3: // NEG
		ea := c.resolveEA(mode, reg, size)
		c.store(ea, size, c.sub(0, c.load(ea, size), size))
		return
	case op&0xFF00 == 0x4000 && size != 3: // NEGX
		ea := c.resolveEA(mode, reg, size)
		x := uint32(0)
		if c.X {
			x = 1
		}
		c.store(ea, size, c.sub(0, c.load(ea, size)+x, size))
		return
	case op&0xFF00 == 0x4200 && size != 3: // CLR
		ea := c.resolveEA(mode, reg, size)
		c.store(ea, size, 0)
		c.N, c.Z, c.V, c.C = false, true, false, false
		return
	case op&0xFF00 == 0x4A00 && size != 3: // TST
		ea := c.resolveEA(mode, reg, size)
		c.setLogic(c.load(ea, size), size)
		return
	}
	c.Halt("unimplemented opcode $%04X at $%06X", op, start)
}

// movem transfers a register list to or from memory.
func (c *CPU) movem(op uint16) {
	toReg := op&0x0400 != 0
	long := op&0x0040 != 0
	mode, reg, _, _ := fields(op)
	mask := c.fetch16()
	step := uint32(2)
	if long {
		step = 4
	}

	rd := func(a uint32) uint32 {
		if long {
			return c.read32(a)
		}
		return uint32(int32(int16(uint16(c.read16(a))))) // word -> sign-extended long
	}
	wr := func(a, v uint32) {
		if long {
			c.write32(a, v)
		} else {
			c.write16(a, v&0xFFFF)
		}
	}

	switch {
	case !toReg && mode == 4: // registers -> -(An), reversed bit order
		a := c.A[reg]
		for i := 0; i < 16; i++ {
			if mask&(1<<uint(i)) == 0 {
				continue
			}
			a -= step
			ri := 15 - i // bit 0 = A7 … bit 15 = D0
			wr(a, c.regVal(ri))
		}
		c.A[reg] = a
	case toReg && mode == 3: // (An)+ -> registers
		a := c.A[reg]
		for i := 0; i < 16; i++ {
			if mask&(1<<uint(i)) == 0 {
				continue
			}
			c.setReg(i, rd(a))
			a += step
		}
		c.A[reg] = a
	default: // control mode, ascending
		a := c.resolveEA(mode, reg, 1).addr
		for i := 0; i < 16; i++ {
			if mask&(1<<uint(i)) == 0 {
				continue
			}
			if toReg {
				c.setReg(i, rd(a))
			} else {
				wr(a, c.regVal(i))
			}
			a += step
		}
	}
}
