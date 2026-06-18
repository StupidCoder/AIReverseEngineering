// Command map renders a world's tile map (a "scene") into a full playfield image
// from Turrican's disk-streamed level blocks (Part IV).
//
//	map [-world N] [-scene N] [-o out.png] [Turrican.adf]
//
// A decoded scene block holds the scene_manager directory at its head: a pointer
// table at +$16 lists the scene descriptors. Each descriptor gives the map
// pointer (+$00) and the map size in tiles (+$04 width, +$06 height). The map is
// a column-major array of one byte per cell: 0..nTiles-1 is a tile index, and a
// value >= nTiles selects a horizontally flipped tile (index = value-128). Tiles
// are 32x32, four bitplanes interleaved per row (512 bytes), drawn through the
// 16-colour palette at +$08.
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"turrican/extract/decrunch"
)

const (
	blockBase  = 0x1B980
	levelTable = 0x46A
	tileSide   = 32
	tileBytes  = tileSide * 4 * (tileSide / 8) // 512
)

func main() {
	world := flag.Int("world", 0, "world number (0..4)")
	scene := flag.Int("scene", 0, "scene/sub-map within the world")
	out := flag.String("o", "", "output PNG (default: world<W>_map<S>.png)")
	flag.Parse()
	adfPath := flag.Arg(0)
	if adfPath == "" {
		adfPath = "Turrican.adf"
	}
	if *out == "" {
		*out = fmt.Sprintf("world%d_map%d.png", *world, *scene)
	}

	adf, err := os.ReadFile(adfPath)
	if err != nil {
		fail(err)
	}
	res, err := decrunch.Decrunch(mainBlob(adf))
	if err != nil {
		fail(err)
	}
	img := res.Data

	t := levelTable + *world*8
	off := int(binary.BigEndian.Uint32(img[t:]))
	length := int(binary.BigEndian.Uint32(img[t+4:]))
	block, err := decrunch.DecrunchBlock(adf[off : off+length])
	if err != nil {
		fail(err)
	}

	at := func(addr int) int { return addr - blockBase }
	be32 := func(o int) int { return int(binary.BigEndian.Uint32(block[o:])) }
	be16 := func(o int) int { return int(binary.BigEndian.Uint16(block[o:])) }

	pal := readPalette(block, at(be32(0x08)))
	tableOff := at(be32(0x00))
	nTiles := int(binary.BigEndian.Uint32(block[tableOff:])) / 4

	// Scene descriptor: pointer table at block+$16.
	descOff := at(be32(0x16 + *scene*4))
	mapOff := at(be32(descOff + 0x00))
	w := be16(descOff + 0x04)
	h := be16(descOff + 0x06)
	if w <= 0 || h <= 0 || mapOff+w*h > len(block) {
		fail(fmt.Errorf("world %d scene %d: bad map (off=$%X %dx%d)", *world, *scene, mapOff, w, h))
	}

	// Pre-decode the tiles to index grids (and their h-flips).
	tiles := make([][]uint8, nTiles)
	for n := range tiles {
		tiles[n] = decodeTile(block, tableOff+int(binary.BigEndian.Uint32(block[tableOff+n*4:])))
	}

	out2 := image.NewPaletted(image.Rect(0, 0, w*tileSide, h*tileSide), pal)
	for col := 0; col < w; col++ {
		for row := 0; row < h; row++ {
			v := int(block[mapOff+col*h+row]) // column-major
			flip := v >= nTiles
			n := v
			if flip {
				n = v - 128
			}
			if n < 0 || n >= nTiles {
				n = 0
			}
			t := tiles[n]
			ox, oy := col*tileSide, row*tileSide
			for y := 0; y < tileSide; y++ {
				for x := 0; x < tileSide; x++ {
					sx := x
					if flip {
						sx = tileSide - 1 - x
					}
					out2.SetColorIndex(ox+x, oy+y, t[y*tileSide+sx])
				}
			}
		}
	}

	f, err := os.Create(*out)
	if err != nil {
		fail(err)
	}
	defer f.Close()
	if err := png.Encode(f, out2); err != nil {
		fail(err)
	}
	fmt.Printf("world %d scene %d: %dx%d tiles -> %s\n", *world, *scene, w, h, *out)
}

func readPalette(block []byte, off int) color.Palette {
	pal := make(color.Palette, 16)
	for i := range pal {
		c := binary.BigEndian.Uint16(block[off+i*2:])
		pal[i] = color.RGBA{
			R: uint8((c>>8)&0xF) * 17,
			G: uint8((c>>4)&0xF) * 17,
			B: uint8(c&0xF) * 17,
			A: 255,
		}
	}
	return pal
}

// decodeTile returns a 32x32 palette-index grid for one tile (4 bitplanes,
// interleaved per row).
func decodeTile(block []byte, off int) []uint8 {
	t := make([]uint8, tileSide*tileSide)
	for y := 0; y < tileSide; y++ {
		var planes [4]uint32
		for p := 0; p < 4; p++ {
			planes[p] = binary.BigEndian.Uint32(block[off+(y*4+p)*4:])
		}
		for x := 0; x < tileSide; x++ {
			var v uint8
			for p := 0; p < 4; p++ {
				v |= uint8((planes[p]>>(31-uint(x)))&1) << uint(p)
			}
			t[y*tileSide+x] = v
		}
	}
	return t
}

func mainBlob(adf []byte) []byte {
	const off = 0x2C00
	n := int(binary.BigEndian.Uint32(adf[off:]))
	return adf[off : off+n]
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "map:", err)
	os.Exit(1)
}
