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
// coordinate-exact vs the oracle, but a per-piece extent, not the visible plan), the
// section's plan-view grid cell (PlanX, PlanY — the actual circuit footprint), and the
// raw section fields.
type Node struct {
	X, Z         int16
	PlanX, PlanY int
	Height       int // surface elevation: ($1C650+$1C718)/2 (left+right rail heights)
	Bank         int // road camber: $1C650-$1C718 (rail height difference)
	Type, P1, P2, Attr int
	// HeightL/HeightR are the exact per-rung left/right rail heights along the section
	// (the in-game surface profile: where it is flat, ramps, or steps), one entry per
	// rung. They are the engine's vertex builder $5C0AA reproduced exactly and verified
	// coordinate-exact against the oracle (cmd/geomoracle) on all eight tracks. The
	// step-vs-slope distinction the surface needs is *in this data* — a smooth run of
	// values is a drivable slope, a sudden jump is a hard edge (e.g. a Big Ramp jump
	// lip) — so no heuristic classifier is needed.
	HeightL, HeightR []int
	// PlanLX/PlanLZ, PlanRX/PlanRZ are the exact per-rung left/right rail positions in the
	// section's LOCAL frame (+z forward, x lateral) — the (x,z) vertex pairs the engine's
	// $5C6C4 reads from the per-type piece-shape (so straights are straight, curve pieces
	// carry their real arc). Verified exact vs the engine (cmd/planoracle). PlanOff is the
	// byte offset of vertex 0's pair within the shape (a0[0]+7); the viewer places these
	// by a similarity fit onto the verified grid anchors.
	PlanLX, PlanLZ, PlanRX, PlanRZ []int
	PlanOff                        int
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

// railHeight reproduces the engine's vertex-height builder $5C0AA for one vertex
// index d1. Even d1 reads the left rail (p2 shape a4, base $1C650 = node.X), odd d1
// the right rail (attr shape a5, base $1C718 = node.Z). The shape read mode is set by
// p2's sign ($1BB79): p2>=0 nibble-packed, p2<0 two bytes per entry. Returns the
// engine's d0.w after ASR.w #5 (signed). Verified exact against the oracle.
func (im *Image) railHeight(a4, a5, base650, base718, p2, d1 int) int {
	d2 := d1
	if int8(p2) >= 0 { // BPL: nibble-packed
		odd := d2 & 1
		d2 >>= 1
		var d0 int
		if odd != 0 {
			v := im.u8(a5 + d2)
			d0 = ((v<<1)&0xE0)|((v&0xF)<<8) + base718
		} else {
			v := im.u8(a4 + d2)
			d0 = ((v<<1)&0xE0)|((v&0xF)<<8) + base650
		}
		return int(int16(uint16(d0)) >> 5)
	}
	d2 &^= 1 // two bytes per entry
	if d1&1 != 0 {
		d3 := im.u8(a5 + d2 + 1)
		d0 := ((im.u8(a5+d2)&0x7F)<<8 | d3) + base718
		return int(int16(uint16(d0)) >> 5)
	}
	d3 := im.u8(a4 + d2 + 1)
	d0 := ((im.u8(a4+d2)&0x7F)<<8 | d3) + base650
	return int(int16(uint16(d0)) >> 5)
}

// railProfile fills a node's HeightL/HeightR with the exact per-rung rail heights.
// The rung (vertex) count is the engine's $1BB97 = shape[shape[0]] of the per-type
// piece-shape (set in $5FE56); it counts both rails, so rungs = count/2. Rung k uses
// vertex indices 2k (left) and 2k+1 (right).
func (im *Image) railProfile(n *Node) {
	nib := n.Type & 0x0F
	shp := handle(im.u16(dataBase + 2*nib))
	count := im.u8(shp + im.u8(shp)) // $1BB97
	rungs := count / 2
	a4 := handle(im.u16(shapeTab + (2*n.P2)&0xFF))
	a5 := handle(im.u16(shapeTab + (2*n.Attr)&0xFF))
	bx, bz := int(n.X), int(n.Z)
	n.HeightL = make([]int, rungs)
	n.HeightR = make([]int, rungs)
	for k := 0; k < rungs; k++ {
		n.HeightL[k] = im.railHeight(a4, a5, bx, bz, n.P2, 2*k)
		n.HeightR[k] = im.railHeight(a4, a5, bx, bz, n.P2, 2*k+1)
	}
}

// s16le reads a signed 16-bit little-endian value (the shape's plan coords are LE; the
// engine builds them in $5C6C4 as byte[a+1]<<8 | byte[a]).
func (im *Image) s16le(a int) int {
	v := im.u8(a) | im.u8(a+1)<<8
	return int(int16(uint16(v)))
}

// planProfile fills a node's per-rung local plan outline from the per-type piece-shape.
// The vertex (x,z) pairs start at shape offset a0[0]+7, stride 4 bytes, vertex k at +4k
// ($5C6C4 read by the draw loop $65CDC: d2 = 2*d1 + $1BB91, $1BB91 = a0[0]+7). Even
// vertex = left rail, odd = right rail — the same interleave as the heights, so rung j
// uses vertices 2j (left) and 2j+1 (right).
func (im *Image) planProfile(n *Node) {
	nib := n.Type & 0x0F
	shp := handle(im.u16(dataBase + 2*nib))
	off := im.u8(shp)
	rungs := im.u8(shp+off) / 2
	n.PlanOff = off + 7
	n.PlanLX = make([]int, rungs)
	n.PlanLZ = make([]int, rungs)
	n.PlanRX = make([]int, rungs)
	n.PlanRZ = make([]int, rungs)
	for j := 0; j < rungs; j++ {
		l := shp + n.PlanOff + 4*(2*j)
		r := shp + n.PlanOff + 4*(2*j+1)
		n.PlanLX[j], n.PlanLZ[j] = im.s16le(l), im.s16le(l+2)
		n.PlanRX[j], n.PlanRZ[j] = im.s16le(r), im.s16le(r+2)
	}
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

	// Plan footprint: each section's cell on the 16x16 track grid. The engine's
	// section lookup ($5FE04) indexes the table $1C280 — built by $64304 as
	// $1C280[p1] = section — by (y<<4)|x, so p1's low nibble is the grid X and its
	// high nibble the grid Y. Consecutive sections are always adjacent cells.
	for i := range nodes {
		nodes[i].PlanX = nodes[i].P1 & 0x0F
		nodes[i].PlanY = nodes[i].P1 >> 4
		// The two extent arrays $1C650 (p2) and $1C718 (attr) are the ribbon's left and
		// right rail heights — the vertex builder $5C0AA adds them un-rotated (>>5),
		// unlike the heading-rotated plan footprint in $5C6C4. So their mean is the
		// surface elevation and their difference is the banking (camber). The elevation
		// profile matches the in-game preview: flat with one ramp (Big Ramp), gentle
		// (Stepping Stones), a run of hills (Roller Coaster); every track closes.
		l, r := int(nodes[i].X), int(nodes[i].Z)
		nodes[i].Height = (l + r) / 2
		nodes[i].Bank = l - r
		// Exact per-rung surface profile (the steps/slopes/jumps), verified vs oracle.
		im.railProfile(&nodes[i])
		// Exact per-rung local plan outline (straights/arcs), verified vs $5C6C4.
		im.planProfile(&nodes[i])
	}

	return Track{Sections: count, FinishIdx: off(1), Nodes: nodes}
}
