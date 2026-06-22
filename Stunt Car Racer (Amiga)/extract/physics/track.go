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
