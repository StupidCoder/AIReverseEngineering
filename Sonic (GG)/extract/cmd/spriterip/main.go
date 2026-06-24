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

// spriteTiles decompresses act i's sprite tile set (the VRAM $2000 sheet).
func spriteTiles(rom []byte, act int) ([]byte, color.Palette) {
	d := descTable + w(rom, descTable+act*2)
	bank := int(rom[d+23])
	addr := uint16(w(rom, d+24))
	off := decomp.SourceOffset(bank, addr)
	tiles := decomp.Decompress(rom, off)
	pal := romPalette(rom, int(rom[d+26]))
	return tiles, pal
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

	// index.json: zone (int) -> type (2-hex) -> {w,h}. The viewer loads
	// sprites/<zone>/<hex>.png for each placed object and draws it centred in the cell.
	index := map[string]map[string]map[string]int{}
	montages := os.Getenv("MONTAGE") != ""
	placed := placedTypesByZone(rom)
	total := 0
	for z, act := range zoneAct {
		tiles, pal := spriteTiles(rom, act)
		zdir := fmt.Sprintf("%s/%d", out, z)
		chk(os.MkdirAll(zdir, 0o755))
		index[fmt.Sprint(z)] = map[string]map[string]int{}
		var cells []*image.RGBA
		var labels []int
		for _, t := range placed[z] {
			r := analyzeSprite(rom, t, z)
			if r.kind == "" || r.layout == 0 || r.layout+18 > len(rom) {
				continue
			}
			spr := trim(renderMeta(rom[r.layout:r.layout+18], tiles, pal))
			if spr.Rect.Dx() <= 1 && spr.Rect.Dy() <= 1 {
				continue // empty layout
			}
			save(spr, fmt.Sprintf("%s/%02x.png", zdir, t))
			index[fmt.Sprint(z)][fmt.Sprintf("%02x", t)] = map[string]int{"w": spr.Rect.Dx(), "h": spr.Rect.Dy()}
			total++
			if montages {
				cells = append(cells, renderMeta(rom[r.layout:r.layout+18], tiles, pal))
				labels = append(labels, t)
			}
		}
		if montages {
			save(montage(cells, labels, pal), fmt.Sprintf("%s/_montage_%s.png", out, zoneName[z]))
		}
		fmt.Printf("%-11s act%2d: %d sprites\n", zoneName[z], act, len(index[fmt.Sprint(z)]))
	}
	f, err := os.Create(out + "/index.json")
	chk(err)
	chk(json.NewEncoder(f).Encode(index))
	f.Close()
	fmt.Printf("wrote %d sprites + index.json to %s\n", total, out)
}
