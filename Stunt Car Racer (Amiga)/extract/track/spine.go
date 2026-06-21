// Package track is an independent Go reimplementation of Stunt Car Racer's track
// section decoder and plan-view spine builder (Stunt_Car_Racer.md Part IV). It reads
// the game's own data tables out of the decrypted $E700-based image and reproduces
// the $5AE46 loop — it does not execute the original code. cmd/spineoracle (the
// m68k core running $5AE46) is the verifier this is checked against, not a data
// source. The track data is platform-independent (the game also ran on 6502/Z80),
// so the format here relies on no 68000-specific behaviour.
package track

const (
	Base     = 0xE700
	ptrTable = 0x1F0A2 // 8 track-data handles
	dataBase = 0x1EF82 // handle bias base; also the per-type shape table base
	shapeTab = 0x1EFA2 // p2/attr -> X/Z offset-shape handles
	bias     = 0xB100
)

// Image wraps the decrypted game image and maps run-time addresses to it.
type Image struct{ b []byte }

func New(img []byte) *Image { return &Image{img} }

func (im *Image) u8(a int) int  { return int(im.b[a-Base]) }
func (im *Image) u16(a int) int { return int(im.b[a-Base])<<8 | int(im.b[a-Base+1]) }

// handle decodes a 16-bit table word to a run-time address: byte-swap, -$B100, +base.
func handle(w int) int { return ((((w<<8|w>>8)&0xFFFF)-bias)&0xFFFF)+dataBase }

// Node is one section's data: the loader's internal base position (X,Z — verified
// coordinate-exact vs the oracle, but an internal layout, not the visible plan), the
// reconstructed world plan position (PlanX, PlanY — the actual circuit, see Plan), and
// the raw section fields.
type Node struct {
	X, Z         int16
	PlanX, PlanY int
	Type, P1, P2, Attr int
}

// planStep turns a section's heading (its type-quadrant + the fine direction packed in
// p1's nibbles, the $5FF94 routine) into a plan-view displacement. The intra-quadrant
// vector is (p1.lo-8, p1.hi-8); the section's quadrant (type & $C0) rotates/reflects it
// into world space. The quadrant transform was fixed by the constraint that every
// circuit must close into a loop (see Stunt_Car_Racer.md Part IV §6).
func planStep(typ, p1 int) (int, int) {
	dx, dy := (p1&0x0F)-8, (p1>>4)-8
	switch (typ >> 6) & 3 {
	case 0:
		return dx, dy
	case 1:
		return dy, dx
	case 2:
		return -dx, dy
	default:
		return -dy, dx
	}
}

// Track is a decoded circuit: its sections' plan-view nodes plus the header.
type Track struct {
	Sections  int
	FinishIdx int
	Nodes     []Node
}

// Addr returns the run-time address of track id's data via the $5AE46 pointer math.
func (im *Image) Addr(id int) int { return handle(im.u16(ptrTable + 2*id)) }

// shapeDelta reproduces $5B2D2/$5B2C8: it reads the offset-shape at the given handle
// and index and returns the signed 16-bit delta the loop subtracts/adds. The sign
// mode comes from p2 ($1BB79): p2<0 selects a 2-byte-per-entry table, p2>=0 a
// nibble-packed one. The 16-bit delta is (main<<8 | side) — the word at $1BB1A.
func (im *Image) shapeDelta(h, idx int, p2neg bool) int16 {
	a0 := handle(h)
	var main, side int
	if p2neg {
		main = im.u8(a0+2*idx) & 0x7F
		side = im.u8(a0 + 2*idx + 1)
	} else {
		v := im.u8(a0 + idx)
		main = v & 0x0F
		side = (v << 1) & 0xE0
	}
	return int16(uint16(main<<8 | side))
}

// Spine decodes track id and walks its plan-view spine, returning one node per
// section. It mirrors $5AE46: parse each (type,p1,p2,attr), resolve the per-type
// segment length and the p2/attr offset-shapes, and accumulate node/cursor.
func (im *Image) Spine(id int) Track {
	a := im.Addr(id)
	off := func(i int) int { return im.u8(a + i) }
	idx := 6 // after the 6-byte header
	count := off(0)

	// cursor X/Z init from header bytes a5[4],a5[5] (high byte = a5[5]).
	curX := int16(uint16(off(5)<<8 | off(4)))
	curZ := curX

	read := func() int { v := off(idx); idx++; return v }
	delta := func(v, t1A int) int {
		switch {
		case t1A&0x80 != 0 && t1A&0x40 != 0:
			return (v - 1) & 0xFF
		case t1A&0x80 != 0:
			return (v - 0x10) & 0xFF
		case t1A&0x40 != 0:
			return (v + 1) & 0xFF
		default:
			return (v + 0x10) & 0xFF
		}
	}

	run, saveType, prevParam := 0, 0, 0
	nodes := make([]Node, 0, count)
	for len(nodes) < count {
		var typ, t1A, p1 int
		if run != 0 {
			run--
			typ, t1A = saveType, saveType
			if saveType&0x10 != 0 {
				t1A ^= 0xC0
			}
			p1 = delta(prevParam, t1A)
		} else {
			typ = read()
			t1A = typ
			if typ&0x0F == 0x0F {
				run = typ >> 4
				continue
			}
			saveType = typ
			p1 = read()
		}
		prevParam = p1

		var p2, attr int
		if nib := t1A & 0x0F; nib >= 0x0C {
			typ &= 0xF0
			p2 = im.u8(0x5B2B8 + nib)
			attr = im.u8(0x5B2BA+nib) & 0x7F
		} else {
			p2 = read()
			src := p2
			if t1A&0x20 == 0 {
				src = read()
			}
			attr = src & 0x7F
		}

		// --- geometry ($5FE56 + spine loop) ---
		nib := typ & 0x0F
		shp := handle(im.u16(dataBase + 2*nib)) // per-type piece-shape (word table)
		bb97 := im.u8(shp + im.u8(shp))          // shape[shape[0]]
		segLen := ((bb97 >> 1) - 1) & 0xFF       // $1BB6A
		// The handle index is ASL.b #1 on the param ($5FE66/$5FE80): (param*2)&$FF.
		p2neg := p2&0x80 != 0
		xH := im.u16(shapeTab + (2*p2)&0xFF)
		zH := im.u16(shapeTab + (2*attr)&0xFF)

		nodeX := curX - im.shapeDelta(xH, 0, p2neg)
		curX = nodeX + im.shapeDelta(xH, segLen, p2neg)
		nodeZ := curZ - im.shapeDelta(zH, 0, p2neg)
		curZ = nodeZ + im.shapeDelta(zH, segLen, p2neg)
		nodes = append(nodes, Node{X: nodeX, Z: nodeZ, Type: typ, P1: p1, P2: p2, Attr: attr})
	}

	// Second pass: walk the world plan by accumulating each section's heading step.
	px, py := 0, 0
	for i := range nodes {
		nodes[i].PlanX, nodes[i].PlanY = px, py
		dx, dy := planStep(nodes[i].Type, nodes[i].P1)
		px += dx
		py += dy
	}

	return Track{Sections: count, FinishIdx: off(1), Nodes: nodes}
}
