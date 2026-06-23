// levelmap decodes a Super Mario Land level map straight from the ROM (the level
// package reimplements the game's $218F decoder) and renders it to a PNG. The tile
// graphics are taken from a brief oracle run only to draw the verification image —
// the map itself is decoded from the cartridge bytes, not read from the oracle's RAM.
//
//	go run ./cmd/levelmap [-rom PATH] [-bank N] [-p1 HEX] [-o FILE]
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strconv"

	"stupidcoder.com/tools/gameboy"
	"supermarioland/extract/level"
)

func main() {
	rom := flag.String("rom", "../Super Mario Land (World).gb", "ROM path")
	id := flag.String("id", "", "level id, e.g. 11 for 1-1 or 43 for 4-3 (selects bank+table automatically)")
	bank := flag.Int("bank", 2, "ROM bank holding the level data (when -id is not used)")
	p1s := flag.String("p1", "6192", "screen-order table address (hex; when -id is not used)")
	start := flag.Int("start", 3, "first main-path screen index (0=lead-in, 1/2=bonus rooms)")
	out := flag.String("o", "../rendered/level-1-1-map.png", "output PNG")
	flag.Parse()

	data, err := os.ReadFile(*rom)
	if err != nil {
		fmt.Fprintln(os.Stderr, "levelmap:", err)
		os.Exit(1)
	}

	// Decode the map from the ROM, either by level id (master lookup) or bank+table.
	var cols [][16]byte
	if *id != "" {
		idv, _ := strconv.ParseUint(*id, 16, 8)
		var p1 uint16
		cols, *bank, p1 = level.DecodeLevelByID(data, byte(idv))
		fmt.Printf("level $%s: bank %d, order table $%04X\n", *id, *bank, p1)
	} else {
		p1v, _ := strconv.ParseUint(*p1s, 16, 16)
		cols = level.DecodeLevelAt(data, *bank, uint16(p1v), *start)
	}
	fmt.Printf("decoded %d columns (%d tiles wide x 16 tall)\n", len(cols), len(cols))

	// Tile graphics + palette: run the oracle to a level and read back VRAM (used only
	// to draw the picture; the map itself is decoded from the ROM above). When a level
	// id is given, force $FFB4 so the loader pages this level's world bank and copies
	// that world's tile set — the only thing taken from the oracle here is the tiles.
	m := gameboy.NewMachine(data)
	m.RunFrames(80)
	for f := 0; f < 6; f++ {
		m.Buttons = gameboy.BtnStart
		m.RunFrame()
	}
	m.Buttons = 0
	if *id != "" {
		idv, _ := strconv.ParseUint(*id, 16, 8)
		for f := 0; f < 40; f++ {
			m.Write(0xFFB4, byte(idv))
			m.RunFrame()
		}
	} else {
		m.RunFrames(60)
	}
	vram := m.VRAM()
	lcdc := m.Read(0xFF40)
	pal := gameboy.DMGPalette(m.Read(0xFF47))

	pimg := image.NewPaletted(image.Rect(0, 0, len(cols)*8, 16*8), pal)
	for x, col := range cols {
		for row := 0; row < 16; row++ {
			off := tileOffset(lcdc, col[row])
			t := gameboy.DecodeTile(vram[off:])
			for py := 0; py < 8; py++ {
				for px := 0; px < 8; px++ {
					pimg.SetColorIndex(x*8+px, row*8+py, t[py][px])
				}
			}
		}
	}

	// Overlay the object/enemy placements (decoded from the ROM, not the oracle) as
	// red boxes at their map-tile positions, to verify the placement decoder visually.
	img := image.NewRGBA(pimg.Bounds())
	draw.Draw(img, img.Bounds(), pimg, image.Point{}, draw.Src)
	if *id != "" {
		idv, _ := strconv.ParseUint(*id, 16, 8)
		objs := level.DecodeObjectsByID(data, byte(idv))
		fmt.Printf("decoded %d objects\n", len(objs))
		for _, o := range objs {
			c := color.RGBA{0xff, 0x30, 0x30, 0xff}
			if o.Hard {
				c = color.RGBA{0xff, 0xc0, 0x20, 0xff} // amber = second-quest flag
			}
			drawBox(img, o.Col*8, o.Row*8, 8, 8, c)
		}
	}
	save(*out, img)
	fmt.Println("wrote", *out)
}

// drawBox outlines a w*h box at (x,y) in colour c.
func drawBox(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for i := 0; i < w; i++ {
		img.Set(x+i, y, c)
		img.Set(x+i, y+h-1, c)
	}
	for i := 0; i < h; i++ {
		img.Set(x, y+i, c)
		img.Set(x+w-1, y+i, c)
	}
}

// tileOffset mirrors the BG tile addressing (LCDC bit 4: $8000 unsigned vs signed $8800).
func tileOffset(lcdc, idx byte) int {
	if lcdc&0x10 != 0 {
		return int(idx) * 16
	}
	return 0x1000 + int(int8(idx))*16
}

func save(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "levelmap:", err)
		os.Exit(1)
	}
	defer f.Close()
	png.Encode(f, img)
}
