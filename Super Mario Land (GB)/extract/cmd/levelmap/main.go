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
	collision := flag.Bool("collision", false, "tint solid tiles to verify the collision rule")
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

	img := image.NewRGBA(pimg.Bounds())
	draw.Draw(img, img.Bounds(), pimg, image.Point{}, draw.Src)

	// Optionally tint solid tiles (a tile is solid ground if its id is in [$60,$F0) —
	// Mario's foot check $17B3 uses >=$60, the enemy checks $2B7B.. use [$5F,$F0)).
	if *collision {
		for x, col := range cols {
			for row := 0; row < 16; row++ {
				if level.SolidTile(col[row]) {
					for py := 0; py < 8; py++ {
						for px := 0; px < 8; px++ {
							r, g, b, _ := img.At(x*8+px, row*8+py).RGBA()
							img.Set(x*8+px, row*8+py, color.RGBA{
								uint8(r >> 9), uint8(g>>9) + 0x40, uint8(b>>9) + 0x80, 0xff})
						}
					}
				}
			}
		}
	}

	// Overlay the object/enemy placements (decoded from the ROM, not the oracle) as
	// red boxes at their map-tile positions, to verify the placement decoder visually.
	if *id != "" {
		idv, _ := strconv.ParseUint(*id, 16, 8)
		objs := level.DecodeObjectsByID(data, byte(idv))
		obp0 := m.Read(0xFF48)
		typeFrame := level.TypeFrames(data)
		fmt.Printf("decoded %d objects\n", len(objs))
		for _, o := range objs {
			// Real metasprite if we know this type's frame; else a marker box.
			if fr, ok := typeFrame[o.Type]; ok {
				drawSprite(img, vram, level.DecodeMetasprite(data, int(fr)), obp0, o.Col*8, o.Row*8)
				continue
			}
			c := color.RGBA{0xff, 0x30, 0x30, 0xff}
			if o.Hard {
				c = color.RGBA{0xff, 0xc0, 0x20, 0xff} // amber = second-quest flag
			}
			drawBox(img, o.Col*8, o.Row*8, 8, 8, c)
		}
		// Mario at his fixed start (sprite top-left at map pixel 35,96).
		mario := []level.Sprite{{Tile: 0x00}, {Tile: 0x01, DX: 8}, {Tile: 0x10, DY: 8}, {Tile: 0x11, DX: 8, DY: 8}}
		drawSprite(img, vram, mario, m.Read(0xFF48), 35, 96)
	}
	save(*out, img)
	fmt.Println("wrote", *out)
}

// drawSprite composites an object metasprite (8x8 sprites) with its origin at map pixel
// (ox,oy) — the same point the engine sets as the object's position.
func drawSprite(img *image.RGBA, vram []byte, sprs []level.Sprite, obp0 byte, ox, oy int) {
	for _, s := range sprs {
		t := gameboy.DecodeTile(vram[int(s.Tile)*16:])
		for py := 0; py < 8; py++ {
			for px := 0; px < 8; px++ {
				v := t[py][px]
				if v == 0 {
					continue // OBJ colour 0 = transparent
				}
				sh := (obp0 >> (2 * v)) & 3
				g := []uint8{0xff, 0xaa, 0x55, 0x00}[sh]
				img.Set(ox+s.DX+px, oy+s.DY+py, color.RGBA{g, g, g, 0xff})
			}
		}
	}
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
