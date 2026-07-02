// placeshot renders a strip of a level from ROM with every placed object's metasprite
// blitted at its ENGINE rest position (objplace: the pickup handlers' $6089 spawn
// adjust + the shared move code's $2CD4 floor snap), plus Sonic's standing frame at
// his dropped spawn — a by-eye check that the web viewer's placement rule puts every
// sprite exactly where the game does. Everything comes from the cartridge: block map,
// tiles, palettes, metasprite layouts (cmd/spriterip's analyzer) and Sonic's frame-0
// tiles (bank 8 3bpp).
//
// Usage: placeshot <rom.gg> <out.png> [act [firstBlock [widthBlocks]]]
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strconv"

	"sonicgg/extract/decomp"
	"sonicgg/extract/objplace"

	"stupidcoder.com/tools/gamegear"
)

const (
	descTable = 0x15600
	blockBase = 0x10000
	tileBase  = 0x30000
	palTable  = 0x23400
)

func w(rom []byte, o int) int { return int(rom[o]) | int(rom[o+1])<<8 }

func romPalette(rom []byte, idx int) color.Palette {
	off := w(rom, palTable+idx*2)
	return gamegear.Palette(rom[palTable+off : palTable+off+32])
}

func main() {
	rom, _ := os.ReadFile(os.Args[1])
	out := os.Args[2]
	act, x0, wb := 0, 0, 48
	if len(os.Args) > 3 {
		act, _ = strconv.Atoi(os.Args[3])
	}
	if len(os.Args) > 4 {
		x0, _ = strconv.Atoi(os.Args[4])
	}
	if len(os.Args) > 5 {
		wb, _ = strconv.Atoi(os.Args[5])
	}

	d := descTable + w(rom, descTable+act*2)
	zone := int(rom[d]) // desc+0 = the engine zone ($D2D5); Sky Base 3 + the teleporter interiors are zone 7
	stride := w(rom, d+1)
	mp := decomp.LoadMapRLE(rom, 0x14000+w(rom, d+15), w(rom, d+17))
	bgTiles := decomp.Decompress(rom, tileBase+w(rom, d+21))
	bgPal := romPalette(rom, int(rom[d+29]))
	blkTable := blockBase + w(rom, d+19)
	sprTiles := objplace.SpriteSheet(rom, act)
	sprPal := romPalette(rom, int(rom[d+26]))
	rows := len(mp) / stride

	img := image.NewRGBA(image.Rect(0, 0, wb*32, rows*32))
	for r := 0; r < rows; r++ {
		for c := 0; c < wb; c++ {
			idx := 0
			if x0+c < stride {
				idx = int(mp[r*stride+(x0+c)])
			}
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

	lvl := objplace.NewLevel(rom, mp, stride, zone)
	ot := descTable + w(rom, d+30)
	n := int(rom[ot])
	for k, p := 0, ot+1; k < n; k, p = k+1, p+3 {
		typ, bx, by := int(rom[p]), int(rom[p+1]), int(rom[p+2])
		x, y, _ := lvl.Settle(typ, bx*32, by*32)
		if x < x0*32-48 || x > (x0+wb)*32 {
			continue
		}
		if r := objplace.AnalyzeSprite(rom, typ, zone); r.Kind != "" && r.Layout != 0 {
			tt := objplace.ApplyIconUpload(rom, sprTiles, typ)
			blit(img, renderMeta(rom[r.Layout:r.Layout+18], tt, sprPal), x-x0*32, y)
			fmt.Printf("type $%02X spawn (%4d,%4d) rest (%4d,%4d)\n", typ, bx*32, by*32, x, y)
		}
	}
	// Sonic: standing frame (bank-8 3bpp, layout $5C1B) at the dropped spawn.
	sx, sy, _ := lvl.DropToFloor(0, int(rom[d+13])*32, int(rom[d+14])*32)
	if sx >= x0*32-48 && sx <= (x0+wb)*32 {
		blit(img, renderMeta(rom[0x5C1B:0x5C1B+18], sonicTiles(rom), sprPal), sx-x0*32, sy)
		fmt.Printf("Sonic    spawn (%4d,%4d) rest (%4d,%4d)\n", int(rom[d+13])*32, int(rom[d+14])*32, sx, sy)
	}

	f, _ := os.Create(out)
	png.Encode(f, img)
	f.Close()
}

func sonicTiles(rom []byte) []byte {
	tiles := make([]byte, 256*32)
	for t := 0; t < 8; t++ {
		for r := 0; r < 8; r++ {
			src := 0x20000 + t*24 + r*3
			dst := (0xB4+t)*32 + r*4
			copy(tiles[dst:dst+3], rom[src:src+3])
		}
	}
	return tiles
}

func blit(dst *image.RGBA, src *image.RGBA, ox, oy int) {
	b := src.Bounds()
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			if _, _, _, a := src.At(x, y).RGBA(); a != 0 {
				dst.Set(ox+x, oy+y, src.At(x, y))
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
