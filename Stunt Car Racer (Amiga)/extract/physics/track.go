package physics

// This file ports the track-facing routines the physics frame uses to read the decoded
// circuit geometry (Part IV): the per-section setup $5FE56, the vertex-height builder
// $5C0AA, the section iterators, and the surface-sample driver $5C1D0 with its helpers.
// They operate on the same memory image as the rest of package physics and are verified
// against the engine (cmd/physverify), so the physics reads the exact $5C0AA rail heights
// the renderer leaves in $1BC02-08.

// handlePhys decodes a 16-bit shape-table word to a run-time address: byte-swap, -$B100,
// +$1EF82 (the same handle math as the track loader).
func handlePhys(w int) int {
	return ((((w<<8|w>>8)&0xFFFF)-0xB100)&0xFFFF)+0x1EF82
}

// Setup5FE56 reproduces $5FE56: load all per-section state for section d1 -- the p2/attr
// cross-section shape handles ($1BC8C/$1BC90), the rail-height bases ($1BC0E/$1BC10 =
// $1C650/$1C718[sec]), the orientation/raised bits, the per-type piece-shape header
// (vertex count $1BB97 and the derived counts, the a0 flags $1BB4D/$1BC44/$1BB7B/etc.).
func (m *Mem) Setup5FE56(sec int) {
	p2 := m.U8(0x1C4C0 + uint32(sec))
	m.B[0x1BB79] = byte(p2)
	m.SetW(0x1BC8C, m.W(0x1EFA2+uint32((p2<<1)&0xFF)))

	attr := m.U8(0x1C524 + uint32(sec))
	d2 := (attr << 1) & 0xFF
	carry := (attr >> 7) & 1 // ROXL.b #1 pulls in the bit shifted out by ASL.b #1
	m.B[0x1BBDC] = byte((carry << 1) & 0xFF)
	m.SetW(0x1BC90, m.W(0x1EFA2+uint32(d2)))

	m.SetW(0x1BC0E, m.W(0x1C650+uint32(sec*2)))
	m.SetW(0x1BC10, m.W(0x1C718+uint32(sec*2)))

	typ := m.U8(0x1C5EC + uint32(sec))
	m.B[0x1BC4A] = byte(typ & 0xC0)
	m.B[0x1BC32] = byte((typ & 0x10) << 3) // bit4 -> bit7
	nib := typ & 0x0F
	m.B[0x1BB86] = byte(nib)
	m.SetW(0x1BCBC, m.W(0x1EF82+uint32(nib*2)))

	a0 := uint32(handlePhys(int(uint16(m.W(0x1BCBC)))))
	m.B[0x1BB4D] = byte(m.U8(a0 + 1))
	off := uint32(m.U8(a0))
	cnt := m.U8(a0 + off) // a0[a0[0]] = vertex count
	d2i := off + 1
	m.B[0x1BB97] = byte(cnt)
	m.B[0x1BB59] = byte((cnt << 1) & 0xFF)
	cm2 := (cnt - 2) & 0xFF
	m.B[0x1BB98] = byte(cm2)
	m.B[0x1BB5A] = byte((cm2 << 1) & 0xFF)
	m.B[0x1BB6A] = byte(((cnt >> 1) - 1) & 0xFF)

	v := m.U8(a0 + d2i) // a0[a0[0]+1]
	d2i++
	v = (v >> 1) | ((v & 1) << 7) // LSR.b #1 then ROXR.b #1 (carry back into bit7)
	m.B[0x1BC44] = byte(v & 0x80)
	m.B[0x1BB7B] = byte(m.U8(a0 + d2i)) // a0[a0[0]+2]
	d2i++
	m.B[0x1BBD9] = byte(m.U8(a0 + d2i)) // a0[a0[0]+3]
	d2i += 3
	m.B[0x1BBD4] = byte(m.U8(a0 + d2i)) // a0[a0[0]+6]
}

// railHeight5C0AA reproduces $5C0AA for one vertex index d1, given the decoded p2/attr
// cross-section shape addresses a4/a5 (the caller decodes $1BC8C/$1BC90). Even d1 reads
// the left rail (a4, base $1BC0E), odd the right (a5, base $1BC10). The read mode is set
// by p2's sign ($1BB79): nibble-packed if >=0, two bytes/entry if <0. Same routine as
// Part IV's track.railHeight, verified there; here it operates on the live image.
func (m *Mem) railHeight5C0AA(a4, a5 uint32, d1 int) int16 {
	p2 := m.U8(0x1BB79)
	b650 := int(m.W(0x1BC0E))
	b718 := int(m.W(0x1BC10))
	d2 := d1
	if int8(p2) >= 0 { // nibble-packed
		odd := d2 & 1
		d2 >>= 1
		var d0 int
		if odd != 0 {
			v := m.U8(a5 + uint32(d2))
			d0 = ((v<<1)&0xE0)|((v&0xF)<<8) + b718
		} else {
			v := m.U8(a4 + uint32(d2))
			d0 = ((v<<1)&0xE0)|((v&0xF)<<8) + b650
		}
		return int16(uint16(d0)) >> 5
	}
	d2 &^= 1 // two bytes per entry
	if d1&1 != 0 {
		d3 := m.U8(a5 + uint32(d2) + 1)
		d0 := ((m.U8(a5+uint32(d2))&0x7F)<<8 | d3) + b718
		return int16(uint16(d0)) >> 5
	}
	d3 := m.U8(a4 + uint32(d2) + 1)
	d0 := ((m.U8(a4+uint32(d2))&0x7F)<<8 | d3) + b650
	return int16(uint16(d0)) >> 5
}

// SecAdvance5C51A / SecRetreat5C538: step the current-section pointer $1BB85 forward /
// backward with wraparound over the section count $1CA1A.
func (m *Mem) SecAdvance5C51A() {
	d1 := m.U8(0x1BB85) + 1
	if d1 >= m.U8(0x1CA1A) {
		d1 = 0
	}
	m.B[0x1BB85] = byte(d1)
}

func (m *Mem) SecRetreat5C538() {
	d1 := int8(m.U8(0x1BB85)) - 1
	if d1 < 0 {
		d1 = int8(m.U8(0x1CA1A)) - 1
	}
	m.B[0x1BB85] = byte(d1)
}

// edgeCross5C484 ($5C484): when the car's across-position runs off the end of the
// current section's rung strip, step to the adjacent section ($5C51A/$5C538 + $5FE56)
// and rebase the vertex index $1BBA3, mirroring the fractions if needed.
func (m *Mem) edgeCross5C484() {
	dc := false
	if int8(byte(m.U8(0x1BC40))^byte(m.U8(0x1BC32))) >= 0 {
		m.SecAdvance5C51A()
		m.Setup5FE56(m.U8(0x1BB85))
		dc = int8(m.U8(0x1BC32)) < 0
	} else {
		m.SecRetreat5C538()
		m.Setup5FE56(m.U8(0x1BB85))
		dc = int8(m.U8(0x1BC32)) >= 0
	}
	if dc { // $5C4DC
		m.B[0x1BBA3] = byte((m.U8(0x1BB97) - 4) & 0xFF)
		if int8(m.U8(0x1BC40)) < 0 {
			return
		}
	} else { // $5C4C6
		m.B[0x1BBA3] = 0
		if int8(m.U8(0x1BC40)) >= 0 {
			return
		}
	}
	if v := byte((256 - m.U8(0x1BC41)) & 0xFF); v != 0 { // NEG.b, default $FF on zero
		m.B[0x1BC41] = v
	} else {
		m.B[0x1BC41] = 0xFF
	}
	if v := byte((256 - m.U8(0x1BC4D)) & 0xFF); v != 0 {
		m.B[0x1BC4D] = v
	} else {
		m.B[0x1BC4D] = 0xFF
	}
}

// offTrack5C872 ($5C872): the wheel is past the track edge -- pin the surface to $1000
// and set the off-track marker bit in $1BB9A.
func (m *Mem) offTrack5C872() {
	m.SetL(0x1BB18, 0x1000)
	m.B[0x1BB9A] = byte((m.U8(0x1BB9A) >> 1) | 0x80)
}

// offEdge5C808 ($5C808): near the track edge, taper the sampled surface ($1BB18) toward
// the edge or, past it, fall off (offTrack5C872); records the edge side in $1BBDA.
func (m *Mem) offEdge5C808() {
	v := int(m.W(0x1BC22))
	var d0 int
	if v < 0 {
		d0 = -v
	} else {
		d0 = 0x180 - v
		if d0 < 0 {
			d0 = -d0
		}
	}
	if d0 > 0x30 {
		m.offTrack5C872()
		return
	}
	d0 = (d0 & 0xFF) << 4
	d3 := m.L(0x1BB18) - int32(d0) - 0x100
	if d3 < 0x1000 {
		m.offTrack5C872()
		return
	}
	m.SetL(0x1BB18, d3)
	if (byte(m.U8(0x1BC22))^byte(m.U8(0x1BC32)))&0x80 != 0 {
		m.B[0x1BBDA] = 0x80
	} else {
		m.B[0x1BBDA] = 0x40
	}
}

// store5C5F2 ($5C5F2): interpolate the surface ($5C554) and store it to contact point d1
// (0/2/4 -> $1BCA4/A8/AC). Near (LOD $1BD5C>=$A) or with large roll: store directly;
// far and near-level: average with the previous value (a low-pass).
func (m *Mem) store5C5F2(d1 int) {
	m.Interp5C554()
	off := uint32(d1 << 1)
	if m.U8(0x1BB65)&0x80 != 0 { // BCLR #7,$1BB65 ; act if it was set
		m.B[0x1BB65] &^= 0x80
		m.offEdge5C808()
	} else {
		m.B[0x1BB65] &^= 0x80
	}
	if int8(m.U8(0x1BD5C)) >= 0x0A {
		m.SetL(0x1BCA4+off, m.L(0x1BB18))
		return
	}
	r := m.U8(0x1BCE4)
	if int8(r) < 0 {
		r = (256 - r) & 0xFF // NEG.b ($80 stays $80 = -128 signed)
	}
	if int8(byte(r)) > 5 { // CMPI.b #5 ; BGT is signed
		m.SetL(0x1BCA4+off, m.L(0x1BB18))
		return
	}
	a := uint32(m.L(0x1BCA4 + off))
	b := uint32(m.L(0x1BB18))
	sum := a + b
	var carry uint32
	if sum < a {
		carry = 1
	}
	m.SetL(0x1BCA4+off, int32((carry<<31)|(sum>>1))) // ROXR.l #1 after ADD.l
}

// mtxAlong is the engine's "MULS.W d3,d0 ; ASL.l #1 ; SWAP" used for the along/across
// fraction scaling in $5C1D0: d3 = (base.b << 7) with bit15 cleared.
func mulSwap(d0, d3 int) int {
	p := int32(int16(d0)) * int32(int16(d3))
	return int(int16((p << 1) >> 16))
}

// Surface5C1D0 ($5C1D0) is the per-frame surface-sample driver: for each of the three
// contact points it finds the car's section ($5FE56), computes the along/across
// fractions from the contact offset ($1BD02/$1BD08) -- flagging an off-edge in $1BB65 --
// samples the four surrounding rung-corner heights via $5C0AA into $1BC02-08, and stores
// the interpolated surface to $1BCA4/A8/AC via $5C5F2. This is where the physics reads
// the Part IV track surface.
func (m *Mem) Surface5C1D0() {
	sec := m.U8(0x1BB1C)
	m.B[0x1BB85] = byte(sec)
	m.Setup5FE56(sec)
	m.B[0x1BB9A] = 0
	d1 := 4
	for {
		m.B[0x1BBF9] = byte(d1)
		if m.U8(0x1BB1C) != m.U8(0x1BB85) {
			s := m.U8(0x1BB1C)
			m.B[0x1BB85] = byte(s)
			m.Setup5FE56(s)
			d1 = m.U8(0x1BBF9)
		}
		m.B[0x1BB1A] = byte(m.U8(0x1BB7B))
		pos := int(m.W(0x1BD02+uint32(d1)))>>4 + int(m.W(0x1BC5E))
		var d0 int
		if uint16(pos) >= 0x180 { // off-edge
			m.B[0x1BB65] |= 0x80
			m.SetW(0x1BC22, int16(pos))
			if int16(pos) < 0 {
				d0 = 0
			} else {
				d0 = 0xFF
			}
		} else {
			a := int(int16(pos))
			if a < 0 {
				a = -a
			}
			d0 = mulSwap(a, (m.U8(0x1BB1A)<<7)&0x7FFF)
			if d0 >= 0x100 {
				d0 = m.U8(0xFF) // engine's MOVE.b $FF.l clamp (0 on our flat bus)
			}
		}
		m.B[0x1BC4D] = byte(d0)
		if int8(m.U8(0x1BC32)) < 0 { // mirror along-fraction when raised
			d0 = int(byte(d0) ^ 0xFF)
		}
		if d1 == 4 {
			m.B[0x1BBA1] = byte(d0)
		}
		// across-fraction
		m.B[0x1BB1A] = byte(m.U8(0x1BBD9))
		ac := mulSwap(int(m.W(0x1BD08+uint32(d1)))>>3, (m.U8(0x1BB1A)<<7)&0x7FFF) + int(m.W(0x1BB10))
		m.SetW(0x1BC40, int16(ac))
		e := (m.U8(0x1BC40) << 1) & 0xFF
		m.B[0x1BBA3] = byte(e)
		if int8(byte(e)) < 0 || int8(byte(e)) >= int8(m.U8(0x1BB98)) {
			m.edgeCross5C484()
		}
		// sample the four corner heights
		a4 := uint32(handlePhys(int(uint16(m.W(0x1BC8C)))))
		a5 := uint32(handlePhys(int(uint16(m.W(0x1BC90)))))
		if int8(m.U8(0x1BC32)) < 0 { // raised -> reverse order
			vi := (m.U8(0x1BB97) - m.U8(0x1BBA3) - 4) & 0xFF
			m.SetW(0x1BC08, m.railHeight5C0AA(a4, a5, vi))
			m.SetW(0x1BC06, m.railHeight5C0AA(a4, a5, (vi+1)&0xFF))
			m.SetW(0x1BC04, m.railHeight5C0AA(a4, a5, (vi+2)&0xFF))
			m.SetW(0x1BC02, m.railHeight5C0AA(a4, a5, (vi+3)&0xFF))
		} else {
			vi := m.U8(0x1BBA3)
			m.SetW(0x1BC02, m.railHeight5C0AA(a4, a5, vi))
			m.SetW(0x1BC04, m.railHeight5C0AA(a4, a5, (vi+1)&0xFF))
			m.SetW(0x1BC06, m.railHeight5C0AA(a4, a5, (vi+2)&0xFF))
			m.SetW(0x1BC08, m.railHeight5C0AA(a4, a5, (vi+3)&0xFF))
		}
		m.store5C5F2(m.U8(0x1BBF9))
		d1 = int(int8(m.U8(0x1BBF9))) - 2
		if d1 < 0 {
			break
		}
	}
}
