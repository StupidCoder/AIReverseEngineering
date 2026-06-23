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
	"image/png"
	"os"
	"strconv"

	"stupidcoder.com/tools/gameboy"
	"supermarioland/extract/level"
)

func main() {
	rom := flag.String("rom", "../Super Mario Land (World).gb", "ROM path")
	bank := flag.Int("bank", 2, "ROM bank holding the level data")
	p1s := flag.String("p1", "6192", "screen-order table address (hex)")
	start := flag.Int("start", 3, "first main-path screen index (0=lead-in, 1/2=bonus rooms)")
	out := flag.String("o", "../rendered/level-1-1-map.png", "output PNG")
	flag.Parse()
	p1v, _ := strconv.ParseUint(*p1s, 16, 16)

	data, err := os.ReadFile(*rom)
	if err != nil {
		fmt.Fprintln(os.Stderr, "levelmap:", err)
		os.Exit(1)
	}

	// Decode the map from the ROM.
	cols := level.DecodeLevelAt(data, *bank, uint16(p1v), *start)
	fmt.Printf("decoded %d columns (%d tiles wide x 16 tall)\n", len(cols), len(cols))

	// Tile graphics + palette: run the oracle to the level so VRAM holds this world's
	// tiles (used only to draw the picture).
	m := gameboy.NewMachine(data)
	m.RunFrames(80)
	for f := 0; f < 6; f++ {
		m.Buttons = gameboy.BtnStart
		m.RunFrame()
	}
	m.Buttons = 0
	m.RunFrames(60)
	vram := m.VRAM()
	lcdc := m.Read(0xFF40)
	pal := gameboy.DMGPalette(m.Read(0xFF47))

	img := image.NewPaletted(image.Rect(0, 0, len(cols)*8, 16*8), pal)
	for x, col := range cols {
		for row := 0; row < 16; row++ {
			off := tileOffset(lcdc, col[row])
			t := gameboy.DecodeTile(vram[off:])
			for py := 0; py < 8; py++ {
				for px := 0; px < 8; px++ {
					img.SetColorIndex(x*8+px, row*8+py, t[py][px])
				}
			}
		}
	}
	save(*out, img, pal)
	fmt.Println("wrote", *out)
}

// tileOffset mirrors the BG tile addressing (LCDC bit 4: $8000 unsigned vs signed $8800).
func tileOffset(lcdc, idx byte) int {
	if lcdc&0x10 != 0 {
		return int(idx) * 16
	}
	return 0x1000 + int(int8(idx))*16
}

func save(path string, img image.Image, pal color.Palette) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "levelmap:", err)
		os.Exit(1)
	}
	defer f.Close()
	png.Encode(f, img)
}
