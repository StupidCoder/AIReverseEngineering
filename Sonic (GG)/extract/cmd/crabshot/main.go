// crabshot renders a window of Green Hills Act 1 around a chosen crab and overlays the
// crab metasprite at three candidate positions, so the correct one can be judged by eye:
//
//	G (green tint)  = the engine position: metasprite grid-top at (blockX*32, blockY*32)
//	S (no tint)     = feet on the collision ground surface (surfaceY - spriteHeight)
//	B (red tint)    = the committed "bottom-anchor": blockY*32 - spriteHeight
//
// The block map, BG tile set, palette and block tile table are decoded from ROM exactly
// as cmd/levelmap does; the crab sprite is the ROM metasprite from cmd/spriterip.
package main

import (
	"image"
	"image/color"
	"image/png"
	"os"

	"sonicgg/extract/decomp"
	"stupidcoder.com/tools/gamegear"
)

const (
	descTable = 0x15600
	blockBase = 0x10000
	tileBase  = 0x30000
	palTable  = 0x23400
	attrPtrs  = 0x343D
	shapeTbl  = 0x3E7A
)

func w(rom []byte, o int) int { return int(rom[o]) | int(rom[o+1])<<8 }

func romPalette(rom []byte, idx int) color.Palette {
	off := w(rom, palTable+idx*2)
	return gamegear.Palette(rom[palTable+off : palTable+off+32])
}

func main() {
	rom, _ := os.ReadFile(os.Args[1])
	act := 0
	d := descTable + w(rom, descTable+act*2)
	stride := w(rom, d+1)
	mp := decomp.LoadMapRLE(rom, 0x14000+w(rom, d+15), w(rom, d+17))
	bgTiles := decomp.Decompress(rom, tileBase+w(rom, d+21))
	bgPal := romPalette(rom, int(rom[d+29]))
	blkTable := blockBase + w(rom, d+19)
	// sprite sheet
	sprTiles := decomp.Decompress(rom, decomp.SourceOffset(int(rom[d+23]), uint16(w(rom, d+24))))
	sprPal := romPalette(rom, int(rom[d+26]))
	rows := len(mp) / stride

	// crab at block (42,13); window of 12 blocks around it
	cbx, cby := 42, 13
	x0 := cbx - 5
	wb := 12
	img := image.NewRGBA(image.Rect(0, 0, wb*32, rows*32))
	for r := 0; r < rows; r++ {
		for c := 0; c < wb; c++ {
			idx := int(mp[r*stride+(x0+c)])
			for tr := 0; tr < 4; tr++ {
				for tc := 0; tc < 4; tc++ {
					t := gamegear.DecodeTile(bgTiles[int(rom[blkTable+idx*16+tr*4+tc])*32:])
					for y := 0; y < 8; y++ {
						for x := 0; x < 8; x++ {
							img.Set(c*32+tc*8+x, r*32+tr*8+y, bgPal[t[y][x]])
						}
					}
				}
			}
		}
	}
	// crab sprite (layout $6704): the FULL 48px metasprite grid, blitted with its top-left
	// at the object world position (blockX*32, blockY*32) = exactly what the engine does.
	full := renderMeta(rom[0x6704:0x6704+18], sprTiles, sprPal)
	sx := (cbx - x0) * 32
	blit(img, full, sx, cby*32, color.RGBA{0, 0, 0, 0})

	// crop to a band around the action
	band := img.SubImage(image.Rect(0, (cby-5)*32, wb*32, (cby+4)*32))
	f, _ := os.Create(os.Args[2])
	png.Encode(f, band)
	f.Close()
}

func blit(dst *image.RGBA, src *image.RGBA, ox, oy int, tint color.RGBA) {
	b := src.Bounds()
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			r, g, bl, a := src.At(x, y).RGBA()
			if a != 0 {
				dst.Set(ox+x, oy+y, color.RGBA{uint8(r>>8) | tint.R, uint8(g>>8) | tint.G, uint8(bl>>8) | tint.B, 255})
			}
		}
	}
}

func renderMeta(layout, tiles []byte, pal color.Palette) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 48, 48))
	p := 0
	for row := 0; row < 3; row++ {
		if p >= len(layout) || layout[p] == 0xFF {
			break
		}
		for col := 0; col < 6; col++ {
			b := layout[p]
			p++
			if b >= 0xFE {
				continue
			}
			for half := 0; half < 2; half++ {
				ti := (int(b) + half) * 32
				if ti+32 > len(tiles) {
					continue
				}
				t := gamegear.DecodeTile(tiles[ti:])
				for y := 0; y < 8; y++ {
					for x := 0; x < 8; x++ {
						if v := t[y][x]; v != 0 {
							img.Set(col*8+x, row*16+half*8+y, pal[v])
						}
					}
				}
			}
		}
	}
	return img
}

func trim(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X, b.Min.Y
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := src.At(x, y).RGBA(); a != 0 {
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
