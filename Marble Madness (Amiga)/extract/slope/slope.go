// Package slope reconstructs a Marble Madness course's static slope field — the
// corner-height mesh the marble physically rolls on (Marble_Madness.md Part V
// §4). The course descriptor at Track header +0 (engine global $9A6) holds an
// array of 8-byte region records; the engine routine build_region ($E158)
// rasterises each into per-tile heights, which this package replays.
//
// Each region record:
//
//	[0] x0 (s8)  [1] y0   [2] xSize  [3] ySize   (axis-aligned rect in iso tiles)
//	[4..5] baseHeight (word)
//	[6] low5 = edge-shape selector -> a height-delta profile (table at +$20)
//	[7] low3 = slope direction 0..7 ; bit3 = flip (negate the profile)
package slope

// dirVec is the $2504 direction table: the 4 iso diagonals, indexed by the
// record's 3-bit dir.
var dirVec = [8][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}, {1, 1}, {-1, 1}, {-1, -1}, {1, -1}}

func u16(b []byte, o uint32) int {
	if int(o)+2 > len(b) {
		return 0
	}
	return int(b[o])<<8 | int(b[o+1])
}
func u32(b []byte, o uint32) uint32 {
	if int(o)+4 > len(b) {
		return 0
	}
	return uint32(b[o])<<24 | uint32(b[o+1])<<16 | uint32(b[o+2])<<8 | uint32(b[o+3])
}
func s8(v byte) int {
	if v > 127 {
		return int(v) - 256
	}
	return int(v)
}

type record struct{ x0, y0, xs, ys, bh, edge, dir, flip int }

// profileSeq expands an edge-shape profile to n signed deltas, resetting to the
// start on a $80 marker (build_region $E158 logic).
func profileSeq(im []byte, etbl uint32, edge, n int) []int {
	ep := u32(im, etbl+uint32(edge)*4)
	seq := make([]int, 0, n)
	for i := uint32(0); len(seq) < n; {
		if int(ep+i) >= len(im) {
			break
		}
		v := im[ep+i]
		if v == 0x80 {
			i = 0
			continue
		}
		seq = append(seq, s8(v))
		i++
		if i > 255 {
			break
		}
	}
	return seq
}

func seqRange(a, b, step int) []int {
	var out []int
	if step > 0 {
		for i := a; i <= b; i++ {
			out = append(out, i)
		}
	} else {
		for i := b; i >= a; i-- {
			out = append(out, i)
		}
	}
	return out
}

// Field is the reconstructed slope field: a height per iso-tile coordinate.
type Field struct {
	H      map[[2]int]int // (tx,ty) -> height
	Lo, Hi int            // height range over the real (rolling-surface) tiles
	MinX   int
	MinY   int
	MaxX   int
	MaxY   int
}

// realFloor is the threshold separating the marble's rolling surface (heights
// above it) from pit/sentinel cells; build_region's range is tracked only for
// heights above it (Part V §4).
const realFloor = 8000

// Build replays build_region for every record in the Track hunk image into a
// per-tile height field.
func Build(im []byte) Field {
	d := u32(im, 0) // header +0 -> $9A6 descriptor
	cnt := u16(im, d+0x1A)
	tbl := u32(im, d+0x1C)
	etbl := u32(im, d+0x20)
	f := Field{H: map[[2]int]int{}, Lo: 1 << 30, Hi: -(1 << 30), MinX: 1 << 30, MinY: 1 << 30, MaxX: -(1 << 30), MaxY: -(1 << 30)}
	for k := 0; k < cnt; k++ {
		o := tbl + uint32(k)*8
		if int(o)+8 > len(im) {
			break
		}
		r := record{s8(im[o]), int(im[o+1]), int(im[o+2]), int(im[o+3]),
			u16(im, o+4), int(im[o+6]) & 0x1F, int(im[o+7]) & 7, (int(im[o+7]) >> 3) & 1}
		dx, dy := dirVec[r.dir][0], dirVec[r.dir][1]
		xEnd, yEnd := r.x0+r.xs-1, r.y0+r.ys-1
		seq := profileSeq(im, etbl, r.edge, r.xs*r.ys+4)
		fi := 0
		put := func(tx, ty int) {
			delta := 0
			if len(seq) > 0 {
				delta = seq[len(seq)-1]
				if fi < len(seq) {
					delta = seq[fi]
				}
			}
			fi++
			h := r.bh + delta
			if r.flip == 1 {
				h = r.bh - delta
			}
			f.H[[2]int{tx, ty}] = h
			if h > realFloor {
				if h < f.Lo {
					f.Lo = h
				}
				if h > f.Hi {
					f.Hi = h
				}
			}
			if tx < f.MinX {
				f.MinX = tx
			}
			if tx > f.MaxX {
				f.MaxX = tx
			}
			if ty < f.MinY {
				f.MinY = ty
			}
			if ty > f.MaxY {
				f.MaxY = ty
			}
		}
		xs := seqRange(r.x0, xEnd, dx)
		ys := seqRange(r.y0, yEnd, dy)
		// dir<4: outer y, inner x ; dir>=4: outer x, inner y (the two $E158 loops)
		if r.dir < 4 {
			for _, ty := range ys {
				for _, tx := range xs {
					put(tx, ty)
				}
			}
		} else {
			for _, tx := range xs {
				for _, ty := range ys {
					put(tx, ty)
				}
			}
		}
	}
	return f
}

// MarkerSet is the course's Track-layer markers in iso-tile (X,Y) coordinates —
// the same overlays the offline *.wire.png draws (Marble_Madness.md Part V §4-5).
type MarkerSet struct {
	Placement [][2]int   // object-placement table (Track +4) — cyan
	Ooze      [][2]int   // +$20 ooze candidate positions — orange
	DynRegion [][2]int   // +$14 dynamic/animated terrain regions — yellow
	Marbles   [][][2]int // +$18 black-marble patrol routes (pts[0] = spawn) — magenta
	Slinkies  [][][2]int // +$1C slinky patrol routes (pts[0] = spawn) — green
}

type spawn struct {
	home [2]int
	pos  [][2]int
}

// parseSpawns reads a creature-spawn list (Track header +$18 or +$20): 8-byte
// [X][Y][animPtr:4][type][pad] records. A record with an animPtr points at the
// creature's patrol route; one without (ooze) falls back to the +$20 block's RNG
// definition table for its candidate positions.
func parseSpawns(im []byte, hdrOff uint32) []spawn {
	block := u32(im, hdrOff)
	defs := func() [][2]int {
		var d [][2]int
		for _, doff := range []uint32{0x14, 0x18, 0x1C, 0x34} {
			if dp := u32(im, block+doff); dp != 0 && int(dp)+2 <= len(im) {
				d = append(d, [2]int{int(im[dp]), int(im[dp+1])})
			}
		}
		return d
	}
	var out []spawn
	for o := u32(im, block); int(o)+8 <= len(im) && len(out) < 200; o += 8 {
		if im[o] == 0xFF {
			break
		}
		s := spawn{home: [2]int{int(im[o]), int(im[o+1])}}
		if ap := u32(im, o+2); ap != 0 && int(ap) < len(im) {
			for p := ap; int(p)+2 <= len(im) && im[p] != 0xFF && len(s.pos) < 24; p += 6 {
				s.pos = append(s.pos, [2]int{int(im[p]), int(im[p+1])})
			}
		} else {
			s.pos = defs()
		}
		out = append(out, s)
	}
	return out
}

// parseActors reads the Track +$1C actor list (the slinkies): 8-byte
// [homeX][homeY][pathPtr:4][type][pad] records whose pathPtr is an $FF-terminated
// 3-byte [X][Y][dir] waypoint route.
func parseActors(im []byte) []spawn {
	tbl := u32(im, 0x1C)
	if tbl == 0 || int(tbl)+4 > len(im) {
		return nil
	}
	rec := u32(im, tbl)
	if rec == 0 || int(rec) >= len(im) {
		return nil
	}
	var out []spawn
	for o := rec; int(o)+8 <= len(im) && len(out) < 200; o += 8 {
		if im[o] == 0xFF {
			break
		}
		s := spawn{home: [2]int{int(im[o]), int(im[o+1])}}
		if pp := u32(im, o+2); pp != 0 && int(pp) < len(im) {
			for p := pp; int(p)+3 <= len(im) && im[p] != 0xFF && len(s.pos) < 40; p += 3 {
				s.pos = append(s.pos, [2]int{int(im[p]), int(im[p+1])})
			}
		}
		out = append(out, s)
	}
	return out
}

// parseDynRegions reads the +$14 source list (6-byte [x][y][scriptPtr:4]); each
// region's position is its script's first KEYFRAME (refX,refY), else the cell.
func parseDynRegions(im []byte) [][2]int {
	block := u32(im, 0x14)
	if block == 0 || int(block)+4 > len(im) {
		return nil
	}
	var out [][2]int
	for o := u32(im, block); int(o)+6 <= len(im) && len(out) < 200; o += 6 {
		if im[o] == 0xFF {
			break
		}
		p := [2]int{int(im[o]), int(im[o+1])}
		if sp := u32(im, o+2); sp != 0 && int(sp)+10 <= len(im) && u16(im, sp) == 0 {
			p = [2]int{u16(im, sp+2), u16(im, sp+4)}
		}
		out = append(out, p)
	}
	return out
}

// parseXY collects the [X][Y] of each stride-byte record from off until $FF.
func parseXY(im []byte, off uint32, stride int) [][2]int {
	var out [][2]int
	for o := off; int(o)+stride <= len(im) && len(out) < 1000; o += uint32(stride) {
		if im[o] == 0xFF {
			break
		}
		out = append(out, [2]int{int(im[o]), int(im[o+1])})
	}
	return out
}

// Markers parses every Track-layer overlay for a course.
func Markers(im []byte) MarkerSet {
	m := MarkerSet{Placement: parseXY(im, u32(im, u32(im, 4)), 3)}
	for _, s := range parseSpawns(im, 0x18) {
		if len(s.pos) > 0 {
			m.Marbles = append(m.Marbles, s.pos)
		}
	}
	seen := map[[2]int]bool{}
	for _, s := range parseSpawns(im, 0x20) {
		for _, p := range s.pos {
			if !seen[p] {
				seen[p] = true
				m.Ooze = append(m.Ooze, p)
			}
		}
	}
	for _, s := range parseActors(im) {
		if len(s.pos) > 0 {
			m.Slinkies = append(m.Slinkies, s.pos)
		}
	}
	m.DynRegion = parseDynRegions(im)
	return m
}
