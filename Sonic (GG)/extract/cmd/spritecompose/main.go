// spritecompose boots a level and composes the live SPRITE layer from the VDP sprite-
// attribute table (SAT $3F00) + sprite tiles (VRAM $2000) + sprite palette (CRAM 16-31),
// exactly as hardware would. It writes the full composed sprite layer and a crop around
// Sonic (object 0, world pos $D3FF/$D402 minus camera $D254/$D257) so his first on-screen
// frame can be lifted for the viewer.
package main

import (
	"fmt"
	"image"
	"image/color"
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
	for i := 0; i < 40; i++ { // let Sonic settle a few frames after spawn
		m.RunFrame()
	}

	pal := gamegear.Palette(m.VDP.CRAM[0x20:0x40])
	v := &m.VDP
	eightSixteen := v.Regs[1]&0x02 != 0
	img := image.NewRGBA(image.Rect(0, 0, 256, 224))
	put := func(sx, sy, tile int) {
		h := 8
		if eightSixteen {
			h = 16
			tile &= 0xFE
		}
		for py := 0; py < h; py++ {
			t := tile + py/8
			row := gamegear.DecodeTile(v.VRAM[0x2000+t*32:])
			for px := 0; px < 8; px++ {
				ci := row[py%8][px]
				if ci == 0 {
					continue // sprite colour 0 = transparent
				}
				c := pal[ci].(color.RGBA)
				if sx+px >= 0 && sx+px < 256 && sy+py >= 0 && sy+py < 224 {
					img.Set(sx+px, sy+py, c)
				}
			}
		}
	}
	n := 0
	for i := 0; i < 64; i++ {
		y := int(v.VRAM[0x3F00+i])
		if y == 0xD0 {
			break
		}
		x := int(v.VRAM[0x3F80+i*2])
		tile := int(v.VRAM[0x3F80+i*2+1])
		put(x, y+1, tile)
		n++
	}
	png.Encode(mustCreate(filepath.Join(outdir, fmt.Sprintf("spritelayer_act%02d.png", act+1))), img)

	// crop around Sonic
	rd := func(a uint16) int { return int(m.Read(a)) | int(m.Read(a+1))<<8 }
	sonX := rd(0xD3FF) - rd(0xD254)
	sonY := rd(0xD402) - rd(0xD257)
	fmt.Printf("act %d: %d sprites; 8x16=%v; Sonic screen ~(%d,%d) world(%d,%d) cam(%d,%d)\n",
		act, n, eightSixteen, sonX, sonY, rd(0xD3FF), rd(0xD402), rd(0xD254), rd(0xD257))
	// auto-bbox the non-transparent pixels in a 48x48 window around Sonic, then tight-crop
	wx0, wy0 := sonX-20, sonY-16
	minX, minY, maxX, maxY := 48, 48, -1, -1
	for yy := 0; yy < 48; yy++ {
		for xx := 0; xx < 48; xx++ {
			_, _, _, a := img.At(wx0+xx, wy0+yy).RGBA()
			if a != 0 {
				if xx < minX {
					minX = xx
				}
				if yy < minY {
					minY = yy
				}
				if xx > maxX {
					maxX = xx
				}
				if yy > maxY {
					maxY = yy
				}
			}
		}
	}
	if maxX < 0 {
		fmt.Println("  (no Sonic pixels found in window)")
		return
	}
	cw, ch := maxX-minX+1, maxY-minY+1
	crop := image.NewRGBA(image.Rect(0, 0, cw, ch))
	for yy := 0; yy < ch; yy++ {
		for xx := 0; xx < cw; xx++ {
			crop.Set(xx, yy, img.At(wx0+minX+xx, wy0+minY+yy))
		}
	}
	png.Encode(mustCreate(filepath.Join(outdir, fmt.Sprintf("sonic_act%02d.png", act+1))), crop)
	fmt.Printf("  Sonic sprite %dx%d; sprite-origin minus spawn-pixel = (%d,%d)\n",
		cw, ch, (wx0+minX)-sonX, (wy0+minY)-sonY)
}

func mustCreate(p string) *os.File { f, _ := os.Create(p); return f }
