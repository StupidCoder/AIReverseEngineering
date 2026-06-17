// enemyprobe boots Green Hills Act 1 and scrolls Sonic right (Right+Jump) so enemies
// come on-screen and get drawn. Each frame it scans the live object array ($D3FD: 32 x 26
// bytes, type@+0, worldX=u16@+2, worldY=u16@+5) for an active enemy; when one is on-screen
// it composes the sprite layer (SAT + VRAM sprite tiles + sprite palette) and crops a box
// around the enemy's screen position, saving one PNG per distinct enemy type.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"stupidcoder.com/tools/gamegear"
)

var enemyName = map[byte]string{0x08: "crab", 0x10: "beetle", 0x0E: "bird", 0x26: "fish", 0x2D: "porcupine", 0x0F: "platform", 0x07: "goal"}

func main() {
	rom, _ := os.ReadFile(os.Args[1])
	outdir := os.Args[2]
	os.MkdirAll(outdir, 0o755)
	m := gamegear.NewMachine(rom)
	for i := 0; i < 700; i++ {
		m.RunFrame()
	}
	for round := 0; round < 40; round++ {
		m.Pad00 = 0x7F
		m.Write(0xD238, 0)
		for i := 0; i < 8; i++ {
			m.RunFrame()
			m.Write(0xD238, 0)
		}
		m.Pad00 = 0xFF
		for k := 0; k < 242; k++ {
			m.Write(0xD238, 0)
			m.RunFrame()
		}
	}
	rd := func(a uint16) int { return int(m.Read(a)) | int(m.Read(a+1))<<8 }
	got := map[byte]bool{}
	m.PadDC = 0xE7 // hold Right (bit3) + Jump (bit4), active-low
	for f := 0; f < 4000 && len(got) < 5; f++ {
		m.RunFrame()
		camX, camY := rd(0xD254), rd(0xD257)
		for s := 0; s < 32; s++ {
			base := uint16(0xD3FD + s*0x1A)
			typ := m.Read(base)
			nm, ok := enemyName[typ]
			if !ok || got[typ] || typ == 0 {
				continue
			}
			ex, ey := rd(base+2)-camX, rd(base+5)-camY
			if ex < 116 || ex > 150 || ey < 12 || ey > 120 {
				continue
			}
			img := composeSprites(&m.VDP)
			crop := cropBox(img, ex, ey, 18)
			png.Encode(mustCreate(filepath.Join(outdir, fmt.Sprintf("enemy_%s.png", nm))), crop)
			fmt.Printf("frame %d: %s ($%02X) at screen (%d,%d)\n", f, nm, typ, ex, ey)
			got[typ] = true
		}
	}
	fmt.Printf("captured: %v\n", got)
}

func composeSprites(v *gamegear.VDP) *image.RGBA {
	pal := gamegear.Palette(v.CRAM[0x20:0x40])
	e816 := v.Regs[1]&0x02 != 0
	img := image.NewRGBA(image.Rect(0, 0, 256, 224))
	for i := 0; i < 64; i++ {
		y := int(v.VRAM[0x3F00+i])
		if y == 0xD0 {
			break
		}
		x := int(v.VRAM[0x3F80+i*2])
		tile := int(v.VRAM[0x3F80+i*2+1])
		h := 8
		if e816 {
			h = 16
			tile &= 0xFE
		}
		for py := 0; py < h; py++ {
			row := gamegear.DecodeTile(v.VRAM[0x2000+(tile+py/8)*32:])
			for px := 0; px < 8; px++ {
				if ci := row[py%8][px]; ci != 0 {
					sx, sy := x+px, y+1+py
					if sx >= 0 && sx < 256 && sy >= 0 && sy < 224 {
						img.Set(sx, sy, pal[ci].(color.RGBA))
					}
				}
			}
		}
	}
	return img
}

func cropBox(img *image.RGBA, cx, cy, r int) *image.RGBA {
	// tight-bbox the non-transparent pixels within +/-r of (cx,cy) (excludes nearby Sonic)
	minX, minY, maxX, maxY := r, r, -r-1, -r-1
	for y := -r; y < r; y++ {
		for x := -r; x < r; x++ {
			if _, _, _, a := img.At(cx+x, cy+y).RGBA(); a != 0 {
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x > maxX {
					maxX = x
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	if maxX < minX {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	out := image.NewRGBA(image.Rect(0, 0, maxX-minX+1, maxY-minY+1))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			out.Set(x-minX, y-minY, img.At(cx+x, cy+y))
		}
	}
	return out
}

func mustCreate(p string) *os.File { f, _ := os.Create(p); return f }
