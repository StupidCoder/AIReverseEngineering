// oracleshot boots the game in the machine model, forces an act, waits a chosen number
// of frames, and composes an authentic screenshot from the live VDP state: the name
// table (with the hardware X/Y scroll applied) plus the sprite layer read from the SAT
// ($3F00 Y / $3F80 X+tile, 8x16 mode) — the ground truth for how the real game lays a
// scene out, sprite positions included, without any placement logic of ours.
//
// Usage: oracleshot <rom.gg> <out.png> [act [settleFrames]]
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"strconv"

	"stupidcoder.com/tools/gamegear"
)

func main() {
	rom, _ := os.ReadFile(os.Args[1])
	out := os.Args[2]
	act, settle := 0, 120
	if len(os.Args) > 3 {
		act, _ = strconv.Atoi(os.Args[3])
	}
	if len(os.Args) > 4 {
		settle, _ = strconv.Atoi(os.Args[4])
	}

	m := gamegear.NewMachine(rom)
	word := func(a uint16) uint16 { return uint16(m.Read(a)) | uint16(m.Read(a+1))<<8 }
	for i := 0; i < 700; i++ {
		m.RunFrame()
	}
	for round := 0; round < 40 && word(0xD26F) == 0; round++ {
		m.Pad00 = 0x7F
		for i := 0; i < 8; i++ {
			m.Write(0xD238, byte(act))
			m.RunFrame()
		}
		m.Pad00 = 0xFF
		for k := 0; k < 242 && word(0xD26F) == 0; k++ {
			m.Write(0xD238, byte(act))
			m.RunFrame()
		}
	}
	for i := 0; i < settle; i++ {
		m.RunFrame()
	}

	v := m.VDP
	pal := gamegear.Palette(v.CRAM[:])
	// background: 32x28 name table with the scroll registers applied
	bg := gamegear.RenderNameTable(v.VRAM[0x3800:], v.VRAM[:], 32, 28, pal)
	sx, sy := int(v.Regs[8]), int(v.Regs[9])
	W, H := 256, 224
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			img.Set(x, y, bg.At((x-sx+W)%W, (y+sy)%H))
		}
	}
	// sprites (8x16 mode, patterns at $2000): SAT Y at $3F00+i, X/tile at $3F80+2i.
	// A Y of $D0 ends the list; the displayed line is Y+1.
	sprPal := gamegear.Palette(v.CRAM[32:])
	for i := 0; i < 64; i++ {
		yb := v.VRAM[0x3F00+i]
		if yb == 0xD0 {
			break
		}
		x := int(v.VRAM[0x3F80+2*i])
		t := int(v.VRAM[0x3F80+2*i+1]) &^ 1
		y := int(yb) + 1
		for half := 0; half < 2; half++ {
			tile := gamegear.DecodeTile(v.VRAM[0x2000+(t+half)*32:])
			for r := 0; r < 8; r++ {
				for c := 0; c < 8; c++ {
					if v := tile[r][c]; v != 0 {
						img.Set(x+c, y+half*8+r, sprPal[v])
					}
				}
			}
		}
	}
	// crop to the GG window
	gg := image.NewRGBA(image.Rect(0, 0, 160, 144))
	for y := 0; y < 144; y++ {
		for x := 0; x < 160; x++ {
			gg.Set(x, y, img.At(x+48, y+24))
		}
	}
	f, _ := os.Create(out)
	png.Encode(f, gg)
	f.Close()
	fmt.Printf("act %d after %d frames: cam=$%04X/%04X sonic=(%d,%d)\n", act, settle,
		word(0xD254), word(0xD257),
		int(word(0xD3FD+2)), int(word(0xD3FD+5)))
}
