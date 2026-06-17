// watershot renders Labyrinth Act 1 with the underwater split applied (rows at/below the
// water line use the bank-0 $0216 underwater palette) — a static verification of the
// viewer feature and Part V §3.
package main

import (
	"image"
	"image/color"
	"image/png"
	"os"

	"sonicgg/extract/decomp"
	"stupidcoder.com/tools/gamegear"
)

func w(rom []byte, o int) int { return int(rom[o]) | int(rom[o+1])<<8 }

func main() {
	rom, _ := os.ReadFile(os.Args[1])
	const T = 0x15600
	act := 9 // Labyrinth Act 1
	d := T + w(rom, T+act*2)
	stride := w(rom, d+1)
	tileFile := 0x30000 + w(rom, d+21)
	mapFile := 0x14000 + w(rom, d+15)
	mapLen := w(rom, d+17)
	blkTable := 0x10000 + w(rom, d+19)
	palIdx := int(rom[d+29])
	off := w(rom, 0x23400+palIdx*2)
	surf := gamegear.Palette(rom[0x23400+off : 0x23400+off+32])
	uw := gamegear.Palette(rom[0x0216 : 0x0216+32])

	tiles := decomp.Decompress(rom, tileFile)
	mp := decomp.LoadMapRLE(rom, mapFile, mapLen)
	rows := 4096 / stride
	cols := w(rom, d+7)/32 + 5
	if cols > stride {
		cols = stride
	}
	waterRow := 13 // by of the type-$40 water object
	img := image.NewRGBA(image.Rect(0, 0, cols*32, rows*32))
	for r := 0; r < rows; r++ {
		pal := surf
		if r >= waterRow {
			pal = uw
		}
		for c := 0; c < cols; c++ {
			blk := int(mp[r*stride+c])
			for ty := 0; ty < 4; ty++ {
				for tx := 0; tx < 4; tx++ {
					tile := int(rom[blkTable+blk*16+ty*4+tx])
					px := gamegear.DecodeTile(tiles[tile*32:])
					for y := 0; y < 8; y++ {
						for x := 0; x < 8; x++ {
							img.Set(c*32+tx*8+x, r*32+ty*8+y, pal[px[y][x]].(color.RGBA))
						}
					}
				}
			}
		}
	}
	f, _ := os.Create(os.Args[2])
	png.Encode(f, img)
	f.Close()
}
