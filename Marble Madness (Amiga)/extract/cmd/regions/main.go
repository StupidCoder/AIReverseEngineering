// regions renders the per-course SLOPE FIELD — the static terrain the marble
// rolls on — from each course's Track file, as isometric PNGs.
//
// The slope field is the course descriptor at Track header +0 (global $9A6),
// reverse-engineered in Marble_Madness.md Part V §4. It is an array of 8-byte
// region records (descriptor +$1A = count, +$1C = table), each:
//
//	[0] x0 (s8)  [1] y0   [2] xSize  [3] ySize   (axis-aligned rect in ISO TILES)
//	[4..5] baseHeight (word)
//	[6] low5 = edge-shape selector -> a height-delta profile (table at +$20)
//	[7] low3 = slope direction 0..7 (the 4 iso diagonals x 2 fill orders)
//	[7] bit3 = flip (negate the profile)
//
// The engine routine build_region ($E158) rasterises these into a corner-height
// mesh; this tool replays that height generation — value = baseHeight ± profile,
// the profile consumed one byte per cell in $E158's exact diagonal fill order
// (reset on a $80 marker) — and plots the result in iso tile space.
//
// The wireframe also overlays the other Track layers at their TRUE (X,Y), never snapped:
// placement objects (cyan dots, Track +4), +$18 creature spawns (magenta pins) and +$20
// spawns (orange pins). The placement dots double as a calibration — course features must
// sit on the course, so their fit confirms the (X,Y) grid matches the slope mesh. (NB the
// +$18 record (X,Y) is the verified spawn position; the +$20 record (X,Y) is a secondary
// field — its spawner positions from a definition table — so those pins mark the record
// cell, not the verified creature location. The mismatch is left visible, not hidden.)
//
// Usage: regions <disk.adf> <outdir>
//
//	writes <outdir>/<course>.regions.png (iso tiles coloured by slope direction)
//	   and <outdir>/<course>.wire.png    (3-D wireframe + the Track marker layers)
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"stupidcoder.com/tools/amiga/adf"
	"stupidcoder.com/tools/amiga/hunk"
)

var courses = []struct{ key, track string }{
	{"practy", "PrcTrack"}, {"beginr", "BegTrack"}, {"interm", "IntTrack"},
	{"aerial", "AerTrack"}, {"silly", "SilTrack"}, {"ultima", "UltTrack"},
}

// $2504 direction table: the 4 iso diagonals, indexed by the record's 3-bit dir.
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

type cell struct {
	h, dir int
	sloped bool
}

// buildField replays build_region for every record into a per-tile height map.
func buildField(im []byte) (map[[2]int]cell, int, int) {
	d := u32(im, 0) // header +0 -> $9A6 descriptor
	cnt := u16(im, d+0x1A)
	tbl := u32(im, d+0x1C)
	etbl := u32(im, d+0x20)
	field := map[[2]int]cell{}
	loSlope, hiSlope := 1<<30, -(1 << 30)
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
		sloped := false
		for i := 0; i < r.xs*r.ys && i < len(seq); i++ {
			if seq[i] != 0 {
				sloped = true
				break
			}
		}
		fi := 0
		put := func(tx, ty int) {
			delta := seq[len(seq)-1]
			if fi < len(seq) {
				delta = seq[fi]
			}
			fi++
			h := r.bh + delta
			if r.flip == 1 {
				h = r.bh - delta
			}
			field[[2]int{tx, ty}] = cell{h, r.dir, sloped}
			if h > 8000 {
				if h < loSlope {
					loSlope = h
				}
				if h > hiSlope {
					hiSlope = h
				}
			}
		}
		// dir<4: outer y, inner x ; dir>=4: outer x, inner y (the two $E158 loops)
		xs := seqRange(r.x0, xEnd, dx)
		ys := seqRange(r.y0, yEnd, dy)
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
	return field, loSlope, hiSlope
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

const tileS = 12

func iso(tx, ty int) (int, int) { return (ty - tx) * tileS, (tx + ty) * tileS / 2 }

// fillDiamond draws a tile diamond centred at (cx,cy), half-width tileS.
func fillDiamond(img *image.RGBA, cx, cy int, c color.RGBA) {
	for dy := -tileS / 2; dy <= tileS/2; dy++ {
		w := tileS - 2*abs(dy)
		for dx := -w; dx <= w; dx++ {
			img.SetRGBA(cx+dx, cy+dy, c)
		}
	}
}
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
func clamp8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

var dirCol = map[int]color.RGBA{
	0: {70, 120, 220, 255}, 4: {220, 90, 70, 255}, 2: {90, 200, 90, 255},
	1: {200, 200, 80, 255}, 3: {180, 90, 200, 255}, 5: {80, 200, 200, 255},
	6: {200, 140, 70, 255}, 7: {150, 150, 150, 255},
}

// heightRamp maps t in [0,1] -> blue(low)..white(high).
func heightRamp(t float64) color.RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	stops := []struct {
		p          float64
		r, g, b    int
	}{{0, 30, 40, 120}, {0.3, 40, 140, 150}, {0.55, 60, 170, 80}, {0.78, 220, 205, 70}, {1, 250, 250, 250}}
	for i := 0; i < len(stops)-1; i++ {
		if t <= stops[i+1].p {
			f := (t - stops[i].p) / (stops[i+1].p - stops[i].p + 1e-9)
			return color.RGBA{
				clamp8(stops[i].r + int(float64(stops[i+1].r-stops[i].r)*f)),
				clamp8(stops[i].g + int(float64(stops[i+1].g-stops[i].g)*f)),
				clamp8(stops[i].b + int(float64(stops[i+1].b-stops[i].b)*f)), 255}
		}
	}
	return color.RGBA{250, 250, 250, 255}
}

// render draws the iso region map: tiles coloured by slope direction (flat dimmed).
func render(field map[[2]int]cell) *image.RGBA {
	minX, minY, maxX, maxY := 1<<30, 1<<30, -(1 << 30), -(1 << 30)
	for t := range field {
		x, y := iso(t[0], t[1])
		minX, maxX = min(minX, x), max(maxX, x)
		minY, maxY = min(minY, y), max(maxY, y)
	}
	const M = 44
	W := (maxX - minX) + 2*M + 2*tileS
	H := (maxY - minY) + 2*M + 2*tileS
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	for i := range img.Pix {
		if i%4 == 3 {
			img.Pix[i] = 255
		} else {
			img.Pix[i] = 12
		}
	}
	keys := make([][2]int, 0, len(field)) // deterministic draw order (reproducible output)
	for t := range field {
		keys = append(keys, t)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][0]+keys[i][1] != keys[j][0]+keys[j][1] {
			return keys[i][0]+keys[i][1] < keys[j][0]+keys[j][1]
		}
		return keys[i][0] < keys[j][0]
	})
	for _, t := range keys {
		c := field[t]
		x, y := iso(t[0], t[1])
		cx, cy := x-minX+M+tileS, y-minY+M+tileS
		col := dirCol[c.dir]
		if !c.sloped { // flat region -> dimmed
			col = color.RGBA{col.R/3 + 30, col.G/3 + 30, col.B/3 + 30, 255}
		}
		fillDiamond(img, cx, cy, col)
	}
	return img
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: regions <disk.adf> <outdir>")
		os.Exit(2)
	}
	img, err := os.ReadFile(os.Args[1])
	chk(err)
	vol, err := adf.Open(img)
	chk(err)
	outdir := os.Args[2]
	chk(os.MkdirAll(outdir, 0o755))

	paths := map[string]string{}
	chk(vol.Walk(func(e adf.Entry) error {
		if !e.IsDir {
			paths[strings.ToLower(e.Name)] = e.Path
		}
		return nil
	}))

	for _, c := range courses {
		p, ok := paths[strings.ToLower(c.track)]
		if !ok {
			fmt.Printf("%s: %s not found\n", c.key, c.track)
			continue
		}
		data, err := vol.ReadFile(p)
		chk(err)
		prog, err := hunk.Load(data, 0)
		chk(err)
		field, lo, hi := buildField(prog.Image)
		objects := parsePlacement(prog.Image)
		spawnsA := parseSpawns(prog.Image, 0x18)
		spawnsB := parseSpawns(prog.Image, 0x20)
		writePNG(filepath.Join(outdir, c.key+".regions.png"), render(field))
		writePNG(filepath.Join(outdir, c.key+".wire.png"), renderWire(field, lo, hi, objects, spawnsA, spawnsB))
		fmt.Printf("%s: %d tiles, height %d..%d -> %s.{regions,wire}.png\n", c.key, len(field), lo, hi, c.key)
	}
}

// --- 3-D wireframe of the height field ---------------------------------------
//
// Each tile is a vertex at (tx, ty, height); the grid edges to the (+1,0) and
// (0,+1) neighbours form the mesh. A dimetric projection lifts height up-screen
// (screenY -= z·zscale) so slopes read as the mesh bending. Quads are drawn
// far-to-near with a background-coloured fill first (hidden-line removal) then
// their bright, height-coloured edges. Rendered at 3× and box-downsampled for
// anti-aliasing — no external dependencies.

const (
	ssaa    = 3    // supersample factor
	wSX     = 13.0 // iso half-width
	wSY     = 6.5  // iso half-height
	wZScale = 2.6  // height exaggeration
	wPitDrop = 30.0
	spawnStem = 20 // marker pin height (final px)
	spawnR    = 5  // marker head radius (final px)
	spawnPad  = spawnStem + spawnR + 4 // extra top margin for the pins
)

// parseSpawns reads a creature-spawn list (Track header +$18 or +$20): the global's
// pointer -> the list -> 8-byte [X][Y][animPtr][type] records, $FF-terminated.
func parseSpawns(im []byte, hdrOff uint32) [][2]int { return parseXY(im, u32(im, u32(im, hdrOff)), 8) }

// parsePlacement reads the object-placement table (Track header +4 -> +0): 3-byte
// [X][Y][type] records, $FF-terminated.
func parsePlacement(im []byte) [][2]int { return parseXY(im, u32(im, u32(im, 4)), 3) }

// parseXY collects the [X][Y] of each stride-byte record from off until a leading $FF.
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

// dot fills a small diamond of radius r at (cx,cy).
func dot(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	for dy := -r; dy <= r; dy++ {
		w := r - abs(dy)
		for dx := -w; dx <= w; dx++ {
			px, py := cx+dx, cy+dy
			if px >= 0 && py >= 0 && px < img.Rect.Dx() && py < img.Rect.Dy() {
				img.SetRGBA(px, py, c)
			}
		}
	}
}

func renderWire(field map[[2]int]cell, lo, hi int, objects, spawnsA, spawnsB [][2]int) *image.RGBA {
	base := float64(lo)
	dz := func(c cell) float64 {
		if c.h < 8000 {
			return -wPitDrop // holes drop below the floor
		}
		return float64(c.h) - base
	}
	proj := func(tx, ty int, z float64) (float64, float64) {
		return float64(ty-tx) * wSX, float64(tx+ty)*wSY - z*wZScale
	}
	// cellAt returns the mesh cell at (x,y), or a floor-plane cell if it isn't on the
	// mesh — so markers are plotted at their TRUE (x,y), never snapped elsewhere.
	cellAt := func(x, y int) cell {
		if c, ok := field[[2]int{x, y}]; ok {
			return c
		}
		return cell{h: lo}
	}
	// bounds over the mesh AND every marker (markers can sit off the mesh)
	minX, minY, maxX, maxY := 1e18, 1e18, -1e18, -1e18
	upd := func(x, y int, c cell) {
		px, py := proj(x, y, dz(c))
		minX, maxX = math.Min(minX, px), math.Max(maxX, px)
		minY, maxY = math.Min(minY, py), math.Max(maxY, py)
	}
	for t, c := range field {
		upd(t[0], t[1], c)
	}
	for _, layer := range [][][2]int{objects, spawnsA, spawnsB} {
		for _, s := range layer {
			upd(s[0], s[1], cellAt(s[0], s[1]))
		}
	}
	const M = 30
	W := int(maxX-minX) + 2*M
	H := int(maxY-minY) + 2*M + spawnPad // extra top room for the spawn pins
	big := image.NewRGBA(image.Rect(0, 0, W*ssaa, H*ssaa))
	bg := color.RGBA{8, 10, 18, 255}
	for i := 0; i < len(big.Pix); i += 4 {
		big.Pix[i], big.Pix[i+1], big.Pix[i+2], big.Pix[i+3] = bg.R, bg.G, bg.B, 255
	}
	sp := func(tx, ty int, c cell) (int, int) {
		x, y := proj(tx, ty, dz(c))
		return int((x - minX + M) * ssaa), int((y - minY + M + spawnPad) * ssaa)
	}
	get := func(tx, ty int) (cell, bool) { c, ok := field[[2]int{tx, ty}]; return c, ok }
	lineCol := func(a, b cell) color.RGBA {
		h := (a.h + b.h) / 2
		if h < 8000 {
			return color.RGBA{90, 120, 210, 255}
		}
		c := heightRamp(float64(h-lo) / (float64(hi-lo) + 1e-9))
		// boost so even the low (dark-blue) end is clearly visible on the dark bg
		return color.RGBA{clamp8(int(c.R)*7/10 + 80), clamp8(int(c.G)*7/10 + 80), clamp8(int(c.B)*7/10 + 80), 255}
	}
	// quads sorted far (small tx+ty) -> near
	type quad struct{ tx, ty, depth int }
	var quads []quad
	for t := range field {
		if _, ok := get(t[0]+1, t[1]); !ok {
			continue
		}
		if _, ok := get(t[0], t[1]+1); !ok {
			continue
		}
		if _, ok := get(t[0]+1, t[1]+1); !ok {
			continue
		}
		quads = append(quads, quad{t[0], t[1], t[0] + t[1]})
	}
	sort.Slice(quads, func(i, j int) bool {
		if quads[i].depth != quads[j].depth {
			return quads[i].depth < quads[j].depth
		}
		return quads[i].tx < quads[j].tx
	})
	for _, q := range quads {
		c00, _ := get(q.tx, q.ty)
		c10, _ := get(q.tx+1, q.ty)
		c11, _ := get(q.tx+1, q.ty+1)
		c01, _ := get(q.tx, q.ty+1)
		x00, y00 := sp(q.tx, q.ty, c00)
		x10, y10 := sp(q.tx+1, q.ty, c10)
		x11, y11 := sp(q.tx+1, q.ty+1, c11)
		x01, y01 := sp(q.tx, q.ty+1, c01)
		// hidden-line fill (background colour) covering this quad
		fillTri(big, x00, y00, x10, y10, x11, y11, bg)
		fillTri(big, x00, y00, x11, y11, x01, y01, bg)
		// edges (all four, so the mesh's bottom/right boundary lines are drawn too)
		line(big, x00, y00, x10, y10, lineCol(c00, c10))
		line(big, x00, y00, x01, y01, lineCol(c00, c01))
		line(big, x10, y10, x11, y11, lineCol(c10, c11))
		line(big, x01, y01, x11, y11, lineCol(c01, c11))
		line(big, x00, y00, x11, y11, color.RGBA{30, 36, 54, 255}) // faint triangulation diagonal
	}
	// Markers are plotted at their TRUE (x,y); off-mesh ones sit on the floor plane (not
	// snapped) so the data is shown honestly — placement objects also double as a
	// calibration that the (x,y) grid matches the slope mesh.
	// Placement objects (the "Objects" count): small cyan dots on the terrain, first.
	for _, s := range objects {
		bx, by := sp(s[0], s[1], cellAt(s[0], s[1]))
		dot(big, bx, by, 2*ssaa, color.RGBA{90, 230, 235, 255})
	}
	// creature-spawn pins, drawn on top: +$18 magenta, +$20 orange
	pins := func(spawns [][2]int, col color.RGBA) {
		for _, s := range spawns {
			bx, by := sp(s[0], s[1], cellAt(s[0], s[1]))
			topY := by - spawnStem*ssaa
			for w := -1; w <= 1; w++ { // a few-px stem
				line(big, bx+w, by, bx+w, topY, col)
			}
			dot(big, bx, topY, spawnR*ssaa, col)
			dot(big, bx, topY, spawnR*ssaa-2, color.RGBA{255, 255, 255, 255}) // white core for contrast
			dot(big, bx, topY, spawnR*ssaa-4, col)
		}
	}
	pins(spawnsA, color.RGBA{255, 60, 210, 255}) // +$18 = magenta
	pins(spawnsB, color.RGBA{255, 170, 30, 255}) // +$20 = orange
	return downsample(big, ssaa)
}

func line(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		if x0 >= 0 && y0 >= 0 && x0 < img.Rect.Dx() && y0 < img.Rect.Dy() {
			img.SetRGBA(x0, y0, c)
		}
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func fillTri(img *image.RGBA, x0, y0, x1, y1, x2, y2 int, c color.RGBA) {
	minY := min(y0, min(y1, y2))
	maxY := max(y0, max(y1, y2))
	a := area(x0, y0, x1, y1, x2, y2)
	if a == 0 {
		return
	}
	minX := min(x0, min(x1, x2))
	maxX := max(x0, max(x1, x2))
	W := img.Rect.Dx()
	Hh := img.Rect.Dy()
	for y := max(0, minY); y <= maxY && y < Hh; y++ {
		for x := max(0, minX); x <= maxX && x < W; x++ {
			w0 := area(x1, y1, x2, y2, x, y)
			w1 := area(x2, y2, x0, y0, x, y)
			w2 := area(x0, y0, x1, y1, x, y)
			if (w0 >= 0 && w1 >= 0 && w2 >= 0) || (w0 <= 0 && w1 <= 0 && w2 <= 0) {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func area(ax, ay, bx, by, cx, cy int) int {
	return (bx-ax)*(cy-ay) - (by-ay)*(cx-ax)
}

func downsample(src *image.RGBA, n int) *image.RGBA {
	W, H := src.Rect.Dx()/n, src.Rect.Dy()/n
	dst := image.NewRGBA(image.Rect(0, 0, W, H))
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			var r, g, b int
			for dy := 0; dy < n; dy++ {
				for dx := 0; dx < n; dx++ {
					o := src.PixOffset(x*n+dx, y*n+dy)
					r += int(src.Pix[o])
					g += int(src.Pix[o+1])
					b += int(src.Pix[o+2])
				}
			}
			k := n * n
			dst.SetRGBA(x, y, color.RGBA{uint8(r / k), uint8(g / k), uint8(b / k), 255})
		}
	}
	return dst
}

func writePNG(path string, img *image.RGBA) {
	f, err := os.Create(path)
	chk(err)
	defer f.Close()
	chk(png.Encode(f, img))
}

func chk(e error) {
	if e != nil {
		panic(e)
	}
}
