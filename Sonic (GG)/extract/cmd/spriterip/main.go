// Command spriterip extracts object/enemy sprites straight from the ROM (no oracle).
//
// An object's on-screen sprite is a metasprite: a 3-row x 6-col grid of 8x16-sprite
// tile indices, walked by the sprite writer $2F07 (cell $FE = skip, a $FF at column 0
// ends the sprite; +8 px per column, +16 px per row). The grid is 18 bytes (3x6); an
// animated object selects its frame through the shared animation routine $7C75 as
// layoutBase + frameId*18. The referenced tiles live in the per-zone sprite tile set
// the level loader decompresses to VRAM $2000 (descriptor +23 bank / +24,25 addr,
// $0406 codec), coloured by the sprite palette (descriptor +26 -> bank-8 $7400 table).
//
// This first cut validates the pipeline on Green Hills: it dumps the decompressed
// sprite tile sheet and renders the crab's animation frames (layout base $6704).
package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"sort"

	"sonicgg/extract/decomp"
	"sonicgg/extract/objplace"
	"stupidcoder.com/tools/gamegear"
)

const (
	descTable = 0x15600
	palTable  = 0x23400 // bank 8 $7400 palette-offset table
)

func chk(e error) {
	if e != nil {
		panic(e)
	}
}

func w(rom []byte, o int) int { return int(rom[o]) | int(rom[o+1])<<8 }

// romPalette resolves a palette index to 16 colours via the bank-8 $7400 table.
func romPalette(rom []byte, idx int) color.Palette {
	off := w(rom, palTable+idx*2)
	p := palTable + off
	return gamegear.Palette(rom[p : p+32])
}

// spriteTiles builds act i's full sprite sheet (objplace.SpriteSheet: the zone's own
// tiles $00-$7F + the common block $80-$BF) and resolves its sprite palette.
func spriteTiles(rom []byte, act int) ([]byte, color.Palette) {
	d := descTable + w(rom, descTable+act*2)
	return objplace.SpriteSheet(rom, act), romPalette(rom, int(rom[d+26]))
}

// Sonic's sprite is not in the zone sheets: his handler keeps one layout ($5C1B, his
// 16x32 box) and re-streams the tile GRAPHICS per pose (bank 8 + frame*192, 3bpp —
// $4E9A). His animations are per-frame byte sequences in the $5C5B table ($4E6D
// sequencer); the exported strip plays the idle -> bored program: standing (anim $05,
// frame 0) held ~6 seconds like the game's idle timeout, then the bored foot-tap
// (anim $0D: 2x16, 1x18, then the 2/3 tap loop, here a few cycles before looping
// back to standing). objplace.SonicSeq/SonicFrameTiles supply both.
const sonicLayout = 0x5C1B

// sonicStrip renders Sonic's idle/bored strip: unique graphic frames plus the play
// sequence (strip-frame index, hold frames).
func sonicStrip(rom []byte, pal color.Palette) (*image.RGBA, [][2]int) {
	bored, loopStep := objplace.SonicSeq(rom, 0x0D)
	// unique graphic frames used: standing (0) + the bored frames
	gfx := []int{0}
	idxOf := map[int]int{0: 0}
	for _, st := range bored {
		if _, ok := idxOf[st.Layout]; !ok {
			idxOf[st.Layout] = len(gfx)
			gfx = append(gfx, st.Layout)
		}
	}
	strip := image.NewRGBA(image.Rect(0, 0, len(gfx)*48, 48))
	for i, f := range gfx {
		cell := renderMeta(rom[sonicLayout:sonicLayout+18], objplace.SonicFrameTiles(rom, f), pal)
		for y := 0; y < 48; y++ {
			for x := 0; x < 48; x++ {
				strip.Set(i*48+x, y, cell.At(x, y))
			}
		}
	}
	// play sequence: stand ~6s (the game's idle timeout), the bored intro, then a few
	// tap cycles before the strip loops back to standing.
	seq := [][2]int{{0, 360}}
	for _, st := range bored {
		seq = append(seq, [2]int{idxOf[st.Layout], st.Frames})
	}
	for cycle := 0; cycle < 4; cycle++ {
		for _, st := range bored[loopStep:] {
			seq = append(seq, [2]int{idxOf[st.Layout], st.Frames})
		}
	}
	return strip, seq
}

// renderMeta renders one 18-byte metasprite layout (3 rows x 6 cols of 8x16 sprites)
// into a 48x48 RGBA image; colour index 0 of the sprite palette is transparent.
func renderMeta(layout, tiles []byte, pal color.Palette) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 6*8, 3*16))
	p := 0
	for row := 0; row < 3; row++ {
		if p >= len(layout) || layout[p] == 0xFF {
			break
		}
		for col := 0; col < 6; col++ {
			b := layout[p]
			p++
			if b >= 0xFE { // $FE skip, $FF treated as skip mid-row
				continue
			}
			for half := 0; half < 2; half++ { // 8x16 sprite = tiles b (top), b+1 (bottom)
				ti := (int(b) + half) * 32
				if ti+32 > len(tiles) {
					continue
				}
				t := gamegear.DecodeTile(tiles[ti:])
				ox, oy := col*8, row*16+half*8
				for y := 0; y < 8; y++ {
					for x := 0; x < 8; x++ {
						if v := t[y][x]; v != 0 {
							img.Set(ox+x, oy+y, pal[v])
						}
					}
				}
			}
		}
	}
	return img
}

// drawCell blits one 8x16 sprite cell (tiles t, t+1) into img at (ox, oy).
func drawCell(img *image.RGBA, tiles []byte, pal color.Palette, t, ox, oy int) {
	for half := 0; half < 2; half++ {
		ti := (t + half) * 32
		if ti+32 > len(tiles) {
			continue
		}
		cell := gamegear.DecodeTile(tiles[ti:])
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				if v := cell[y][x]; v != 0 {
					img.Set(ox+x, oy+half*8+y, pal[v])
				}
			}
		}
	}
}

// renderAnimStrip renders a type's full animation as a horizontal strip of 48x48
// frames (objplace.Anim), compositing the pickup family's screen-icon overlay
// (sprites $5C/$5E at (+4,0)/(+12,0), drawn before the metasprite = on top).
func renderAnimStrip(rom, tiles []byte, pal color.Palette, typ, zone int) (*image.RGBA, []int) {
	frames := objplace.Anim(rom, typ, zone)
	if len(frames) == 0 {
		return nil, nil
	}
	icons := objplace.IconOverlays(rom, typ)
	strip := image.NewRGBA(image.Rect(0, 0, len(frames)*48, 48))
	durs := make([]int, len(frames))
	for i, f := range frames {
		if f.Layout <= 0 || f.Layout+18 > len(rom) {
			return nil, nil
		}
		cell := renderMeta(rom[f.Layout:f.Layout+18], tiles, pal)
		for _, ic := range icons {
			drawCell(cell, tiles, pal, ic.Tile, ic.X, ic.Y)
		}
		for y := 0; y < 48; y++ {
			for x := 0; x < 48; x++ {
				strip.Set(i*48+x, y, cell.At(x, y))
			}
		}
		durs[i] = f.Frames
	}
	// Collapse animations whose frames are all pixel-identical: the pickup TV's
	// layout blink alternates a screen cell that the opaque icon fully covers -
	// a sprite-per-scanline budget trick, not a visible animation.
	if len(frames) > 1 {
		same := true
		for i := 1; same && i < len(frames); i++ {
			for y := 0; same && y < 48; y++ {
				for x := 0; x < 48; x++ {
					if strip.RGBAAt(i*48+x, y) != strip.RGBAAt(x, y) {
						same = false
						break
					}
				}
			}
		}
		if same {
			one := image.NewRGBA(image.Rect(0, 0, 48, 48))
			for y := 0; y < 48; y++ {
				for x := 0; x < 48; x++ {
					one.Set(x, y, strip.At(x, y))
				}
			}
			return one, []int{0}
		}
	}
	return strip, durs
}

// goalStrip renders the goal sign's idle spin from its own graphics: the handler
// decompresses its sheet over VRAM $2000 and switches to sprite palette $0E, so the
// sign looks the same in every zone (objplace.GoalAnim).
func goalStrip(rom []byte) (*image.RGBA, [][2]int) {
	steps, src, palIdx := objplace.GoalAnim(rom)
	tiles := make([]byte, 256*32)
	copy(tiles, decomp.Decompress(rom, src))
	copy(tiles[0x80*32:], decomp.Decompress(rom, objplace.CommonTilesFile))
	pal := romPalette(rom, palIdx)
	var gfx []int
	idxOf := map[int]int{}
	for _, st := range steps {
		if _, ok := idxOf[st.Layout]; !ok {
			idxOf[st.Layout] = len(gfx)
			gfx = append(gfx, st.Layout)
		}
	}
	strip := image.NewRGBA(image.Rect(0, 0, len(gfx)*48, 48))
	for i, lay := range gfx {
		cell := renderMeta(rom[lay:lay+18], tiles, pal)
		for y := 0; y < 48; y++ {
			for x := 0; x < 48; x++ {
				strip.Set(i*48+x, y, cell.At(x, y))
			}
		}
	}
	var seq [][2]int
	for _, st := range steps {
		seq = append(seq, [2]int{idxOf[st.Layout], st.Frames})
	}
	return strip, seq
}

// trimBBox crops an RGBA to its non-transparent bounding box and returns the box's
// top-left offset (minX, minY) within the source so callers can keep the sprite's
// position relative to the original (untrimmed) metasprite grid.
func trimBBox(src *image.RGBA) (int, int, *image.RGBA) {
	b := src.Bounds()
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X, b.Min.Y
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, al := src.At(x, y).RGBA(); al != 0 {
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	if minX > maxX {
		return 0, 0, image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	out := image.NewRGBA(image.Rect(0, 0, maxX-minX+1, maxY-minY+1))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			out.Set(x-minX, y-minY, src.At(x, y))
		}
	}
	return minX, minY, out
}

// trim crops an RGBA to its non-transparent bounding box (keeps sprites compact).
func trim(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X, b.Min.Y
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, al := src.At(x, y).RGBA(); al != 0 {
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	if minX > maxX {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	out := image.NewRGBA(image.Rect(0, 0, maxX-minX+1, maxY-minY+1))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			out.Set(x-minX, y-minY, src.At(x, y))
		}
	}
	return out
}

// placedTypesByZone returns the sorted distinct object types placed in each zone's
// acts (zones 0-5 = acts z*3..z*3+2; zone 6 = the special-stage acts 28-35), read
// from each act descriptor's object table (+30).
func placedTypesByZone(rom []byte) [][]int {
	actObjTypes := func(act int) []int {
		d := descTable + w(rom, descTable+act*2)
		ot := descTable + w(rom, d+30)
		n := int(rom[ot])
		var ts []int
		for k, p := 0, ot+1; k < n; k, p = k+1, p+3 {
			ts = append(ts, int(rom[p]))
		}
		return ts
	}
	acts := func(z int) []int {
		if z == 6 {
			return []int{28, 29, 30, 31, 32, 33, 34, 35}
		}
		return []int{z * 3, z*3 + 1, z*3 + 2}
	}
	out := make([][]int, 7)
	for z := 0; z < 7; z++ {
		seen := map[int]bool{}
		for _, act := range acts(z) {
			for _, t := range actObjTypes(act) {
				if t < 0x57 && !seen[t] {
					seen[t] = true
					out[z] = append(out[z], t)
				}
			}
		}
		sort.Ints(out[z])
	}
	return out
}

// montage lays out trimmed sprites in an 8-wide grid on a neutral grey background,
// in the given type order (printed alongside), for visual verification.
func montage(cells []*image.RGBA, labels []int, pal color.Palette) *image.RGBA {
	const cw, ch, cols = 50, 50, 8
	rows := (len(cells) + cols - 1) / cols
	if rows == 0 {
		rows = 1
	}
	img := image.NewRGBA(image.Rect(0, 0, cols*cw, rows*ch))
	grey := color.RGBA{0x40, 0x40, 0x40, 0xFF}
	for y := 0; y < img.Rect.Dy(); y++ {
		for x := 0; x < img.Rect.Dx(); x++ {
			img.Set(x, y, grey)
		}
	}
	for i, c := range cells {
		t := trim(c)
		ox := (i%cols)*cw + (cw-t.Rect.Dx())/2
		oy := (i/cols)*ch + (ch-t.Rect.Dy())/2
		for y := 0; y < t.Rect.Dy(); y++ {
			for x := 0; x < t.Rect.Dx(); x++ {
				if _, _, _, al := t.At(x, y).RGBA(); al != 0 {
					img.Set(ox+x, oy+y, t.At(x, y))
				}
			}
		}
	}
	fmt.Printf("   types: ")
	for _, l := range labels {
		fmt.Printf("$%02X ", l)
	}
	fmt.Println()
	return img
}

func save(img image.Image, path string) {
	f, err := os.Create(path)
	chk(err)
	defer f.Close()
	chk(png.Encode(f, img))
}

func main() {
	rom, err := os.ReadFile(os.Args[1])
	chk(err)
	out := os.Args[2]
	chk(os.MkdirAll(out, 0o755))

	// representative act (and thus sprite sheet/palette) for each of the 7 viewer zones.
	zoneAct := []int{0, 3, 6, 9, 12, 15, 28}
	zoneName := []string{"greenhills", "bridge", "jungle", "labyrinth", "scrapbrain", "skybase", "special"}

	// index.json: zone (int) -> type (2-hex) -> true. Each PNG is the FULL 48x48 metasprite
	// grid; the viewer loads sprites/<zone>/<hex>.png and draws it with its top-left at the
	// object's world position (blockX*32, blockY*32) -- exactly where the engine draws it,
	// with the grid's transparent padding placing the visible tiles. No cropping/offsets.
	type sprMeta struct {
		F int      `json:"f"`
		D []int    `json:"d"`
		S [][2]int `json:"s,omitempty"` // explicit play sequence: (strip frame, hold frames)
	}
	index := map[string]map[string]sprMeta{}
	montages := os.Getenv("MONTAGE") != ""
	placed := placedTypesByZone(rom)
	total := 0
	for z, act := range zoneAct {
		tiles, pal := spriteTiles(rom, act)
		zdir := fmt.Sprintf("%s/%d", out, z)
		chk(os.MkdirAll(zdir, 0o755))
		index[fmt.Sprint(z)] = map[string]sprMeta{}
		var cells []*image.RGBA
		var labels []int
		// Sonic (type $00): his own ROM tiles, this zone's sprite palette.
		sonic, seq := sonicStrip(rom, pal)
		save(sonic, zdir+"/00.png")
		index[fmt.Sprint(z)]["00"] = sprMeta{F: sonic.Rect.Dx() / 48, D: []int{0}, S: seq}
		total++
		for _, t := range placed[z] {
			if t == 0 {
				continue
			}
			if t == 0x07 { // the goal sign: own gfx but a plain metasprite — render its spin
				strip, seq := goalStrip(rom)
				save(strip, fmt.Sprintf("%s/%02x.png", zdir, t))
				index[fmt.Sprint(z)]["07"] = sprMeta{F: strip.Rect.Dx() / 48, D: []int{0}, S: seq}
				total++
				continue
			}
			if t == 0x29 { // the floating log: 3 table-driven roll layouts ($803A+)
				lays := objplace.LogAnim()
				strip := image.NewRGBA(image.Rect(0, 0, len(lays)*48, 48))
				for i, lay := range lays {
					cell := renderMeta(rom[lay:lay+18], tiles, pal)
					for y := 0; y < 48; y++ {
						for x := 0; x < 48; x++ {
							strip.Set(i*48+x, y, cell.At(x, y))
						}
					}
				}
				seq := [][2]int{{0, 240}}
				for c := 0; c < 2; c++ {
					seq = append(seq, [2]int{0, 6}, [2]int{1, 6}, [2]int{2, 6})
				}
				save(strip, fmt.Sprintf("%s/%02x.png", zdir, t))
				index[fmt.Sprint(z)]["29"] = sprMeta{F: len(lays), D: []int{0}, S: seq}
				total++
				continue
			}
			r := objplace.AnalyzeSprite(rom, t, z)
			if r.Kind == "" || r.Layout == 0 || r.Layout+18 > len(rom) {
				continue
			}
			// The bosses are self-contained set-pieces: they decompress their OWN
			// graphics over the zone sprite sheet and draw multi-part sprites from a
			// bytecode script — a single extracted metasprite renders as garbage
			// (the Bridge 1 "mess in the sky" was the Egg Mobile flyby). Skip them;
			// the viewer shows its labelled marker instead.
			if _, _, own := objplace.OwnGfx(rom, t); own || t == 0x25 {
				// (the bosses draw multi-part script sprites with their own tiles, and
				// the capsule 0x25 borrows the boss's runtime tiles - marker fallback)
				continue
			}
			tt := objplace.ApplyIconUpload(rom, tiles, t)
			strip, durs := renderAnimStrip(rom, tt, pal, t, z)
			if strip == nil {
				continue
			}
			if _, _, bb := trimBBox(strip); bb.Rect.Dx() <= 1 && bb.Rect.Dy() <= 1 {
				continue // empty layout
			}
			save(strip, fmt.Sprintf("%s/%02x.png", zdir, t))
			index[fmt.Sprint(z)][fmt.Sprintf("%02x", t)] = sprMeta{F: strip.Rect.Dx() / 48, D: durs}
			total++
			if montages {
				cells = append(cells, renderMeta(rom[r.Layout:r.Layout+18], tiles, pal))
				labels = append(labels, t)
			}
		}
		if montages {
			save(montage(cells, labels, pal), fmt.Sprintf("%s/_montage_%s.png", out, zoneName[z]))
		}
		fmt.Printf("%-11s act%2d: %d sprites\n", zoneName[z], act, len(index[fmt.Sprint(z)]))
	}
	// movement paths for the moving platforms: per-frame (dx,dy) offsets from the
	// placement, sampled from the handlers' own tables (objplace.PlatformPaths).
	full := map[string]any{"paths": objplace.PlatformPaths(rom)}
	for k, v := range index {
		full[k] = v
	}
	f, err := os.Create(out + "/index.json")
	chk(err)
	chk(json.NewEncoder(f).Encode(full))
	f.Close()
	fmt.Printf("wrote %d sprites + index.json to %s\n", total, out)
}
