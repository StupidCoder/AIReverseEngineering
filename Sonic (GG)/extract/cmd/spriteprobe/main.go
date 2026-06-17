// spriteprobe boots a level on the oracle and dumps the SPRITE tile set the loader
// decompressed to VRAM $2000 (the 256 sprite tiles 0-255), rendered with the live sprite
// palette (CRAM 16-31), as a 16x16 tile sheet. That sheet holds Sonic's animation frames
// plus the zone's enemy sprites; the ROM frame-grid data says which tiles form which frame.
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"

	"stupidcoder.com/tools/gamegear"
)

func main() {
	rom, _ := os.ReadFile(os.Args[1])
	outdir := os.Args[2]
	os.MkdirAll(outdir, 0o755)
	act := 0
	if len(os.Args) > 3 {
		act, _ = strconv.Atoi(os.Args[3])
	}
	m := gamegear.NewMachine(rom)
	for i := 0; i < 700; i++ {
		m.RunFrame()
	}
	for round := 0; round < 40; round++ {
		m.Pad00 = 0x7F
		m.Write(0xD238, byte(act))
		for i := 0; i < 8; i++ {
			m.RunFrame()
			m.Write(0xD238, byte(act))
		}
		m.Pad00 = 0xFF
		for k := 0; k < 242; k++ {
			m.Write(0xD238, byte(act))
			m.RunFrame()
		}
	}
	for i := 0; i < 120; i++ {
		m.RunFrame()
	}
	// sprite palette = CRAM entries 16-31
	pal := gamegear.Palette(m.VDP.CRAM[0x20:0x40])
	// sprite tiles 0..255 at VRAM $2000
	sheet := gamegear.TileSheet(m.VDP.VRAM[0x2000:0x4000], pal, 256, 16)
	name := filepath.Join(outdir, fmt.Sprintf("sprites_act%02d.png", act+1))
	f, _ := os.Create(name)
	// scale x3 for visibility
	png.Encode(f, scale(sheet, 3))
	f.Close()
	fmt.Printf("act %d: wrote %s (sprite palette %v)\n", act, name, palHex(m.VDP.CRAM[0x20:0x40]))
}

func palHex(cram []byte) []string {
	p := gamegear.Palette(cram)
	out := make([]string, len(p))
	for i, c := range p {
		r, g, b, _ := c.RGBA()
		out[i] = fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
	}
	return out
}

func scale(src image.Image, n int) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx()*n, b.Dy()*n))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := src.At(b.Min.X+x, b.Min.Y+y)
			for dy := 0; dy < n; dy++ {
				for dx := 0; dx < n; dx++ {
					dst.Set(x*n+dx, y*n+dy, c)
				}
			}
		}
	}
	return dst
}
