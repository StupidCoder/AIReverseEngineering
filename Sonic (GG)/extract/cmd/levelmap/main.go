// levelmap renders the FULL Zone 0 level (Green Hills Act 1) as it appears in the game,
// reconstructed entirely from the cartridge:
//
//   - the block-index map is decoded straight from ROM with decomp.LoadMapRLE (the $0A73
//     codec): 16 rows x 256 columns of block indices;
//   - each block index is expanded to a 4x4 grid of 8x8 tiles via the block tile table
//     at file $10000 (the table $0760 reads through $D249: tile(r,c) = $10000 + idx*16 +
//     r*4 + c, row-major 4 wide), exactly as the $0760 terrain expander does;
//   - the tile graphics and palette are the real ones the level loaded into VRAM/CRAM.
//
// To stay honest it boots the oracle into the level, snapshots the live decompressor's
// $C000 output the instant $0A73 returns, and asserts the from-ROM map decode is byte-
// identical; then it lets the level fully come up to borrow the loaded tile set + palette
// (the block tile table and map both come from ROM, only the pixels/palette are borrowed
// — the tile streamer is hard to reproduce statically). Output: level_map_tiles.png
// (8192x512, the whole level) + level_map.bin (the raw block-index map).
//
// Usage: levelmap <rom.gg> <outdir>
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"sonicgg/extract/decomp"

	"stupidcoder.com/tools/gamegear"
)

// The Zone 0 map source, located by the oracle (CapturePC at $0A73): bank 5, z80 $7430
// (= file $17430), length $0786, decompressing to 4096 bytes = 16 rows x 256 columns.
const (
	mapSrcFile = 0x17430
	mapSrcLen  = 0x0786
	mapCols    = 256
	mapRows    = 16
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: levelmap <rom.gg> <outdir>")
		os.Exit(2)
	}
	rom, err := os.ReadFile(os.Args[1])
	chk(err)
	outdir := os.Args[2]
	chk(os.MkdirAll(outdir, 0o755))

	// 1. Decode the map straight from the cartridge — no machine involved.
	romMap := decomp.LoadMapRLE(rom, mapSrcFile, mapSrcLen)
	fmt.Printf("decoded %d bytes from ROM (bank 5 $%05X, src %d bytes -> %.2fx)\n",
		len(romMap), mapSrcFile, mapSrcLen, float64(len(romMap))/float64(mapSrcLen))

	// 2. Boot the oracle into the level to (a) verify the decode and (b) grab tiles+CRAM.
	m := gamegear.NewMachine(rom)
	word := func(a uint16) uint16 { return uint16(m.Read(a)) | uint16(m.Read(a+1))<<8 }
	m.CapturePC = 0x0A73
	m.CapLo, m.CapHi, m.CapOutBase = 0x0A73, 0x0AA2, 0xC000 // snapshot $C000 at the RET
	for i := 0; i < 700; i++ {
		m.RunFrame()
	}
	for round := 0; round < 40 && word(0xD26F) == 0; round++ {
		m.Pad00 = 0x7F
		for i := 0; i < 8; i++ {
			m.RunFrame()
		}
		m.Pad00 = 0xFF
		for k := 0; k < 242 && word(0xD26F) == 0; k++ {
			m.RunFrame()
		}
	}
	for i := 0; i < 8 && !m.CapOutDone; i++ { // let the load (and the decompressor) finish
		m.RunFrame()
	}
	if !m.CapOutDone {
		fmt.Fprintln(os.Stderr, "warning: never captured the decompressor output")
	} else {
		match := 0
		first := -1
		for i := 0; i < len(romMap) && i < len(m.CapOut); i++ {
			if romMap[i] == m.CapOut[i] {
				match++
			} else if first < 0 {
				first = i
			}
		}
		if first < 0 && len(romMap) == len(m.CapOut) {
			fmt.Printf("VERIFIED: from-ROM decode == live decompressor output (%d bytes, byte-perfect)\n", len(romMap))
		} else {
			fmt.Printf("MISMATCH: %d/%d bytes; first diff at %d\n", match, len(romMap), first)
		}
	}

	// Let the level fully come up so the tile set is streamed in and the palette is
	// uploaded (a handful of frames after the map loads is too early — CRAM is still
	// blank). The ROM map decode above is already captured, so these frames don't affect
	// it. Hold Right+Jump so the tile streamer ($31BC) runs and fills the BG tile set.
	m.PadDC = 0xE7
	for i := 0; i < 150; i++ {
		m.RunFrame()
	}

	// The decompressed BG tile graphics (VRAM $0000) and the BG palette (CRAM 0..15) the
	// level loaded — the pixels the block tiles index into.
	pal := gamegear.Palette(m.VDP.CRAM[:])
	tiles := m.VDP.VRAM[:]

	// 3. Render the FULL map with the real in-game tiles. The $0760 terrain expander
	//    turns each block index into a 4x4 grid of 8x8 tiles: tile(r,c) is the byte at
	//    blockTiles + index*16 + r*4 + c (row-major, 4 wide). blockTiles is the table the
	//    loader points $D249 at: z80 $4000 read with BANK 4 in slot 1 ($0760's prologue
	//    $0726 pages it) = file $10000. Attr contributes only a priority bit (no flip /
	//    no palette select), so every terrain tile uses the BG palette — the pixels come
	//    entirely from the tile index + the loaded tile set.
	const blockTiles = 0x10000 // file offset of the 16-byte-per-block tile table (bank 4)
	const bw = mapCols * 4      // tiles wide  (4 tiles per block)
	const bh = mapRows * 4      // tiles tall
	img := image.NewPaletted(image.Rect(0, 0, bw*8, bh*8), pal)
	for row := 0; row < mapRows; row++ {
		for col := 0; col < mapCols; col++ {
			idx := int(romMap[row*mapCols+col])
			def := blockTiles + idx*16
			for r := 0; r < 4; r++ {
				for c := 0; c < 4; c++ {
					tn := int(rom[def+r*4+c])
					t := gamegear.DecodeTile(tiles[tn*32:])
					ox, oy := (col*4+c)*8, (row*4+r)*8
					for y := 0; y < 8; y++ {
						for x := 0; x < 8; x++ {
							img.SetColorIndex(ox+x, oy+y, t[y][x])
						}
					}
				}
			}
		}
	}
	writePNG(filepath.Join(outdir, "level_map_tiles.png"), img)
	fmt.Printf("wrote level_map_tiles.png (%dx%d px, real tiles)\n", bw*8, bh*8)

	// Also save the raw decoded block-index map for inspection.
	chk(os.WriteFile(filepath.Join(outdir, "level_map.bin"), romMap, 0o644))
}

func writePNG(path string, img image.Image) {
	f, err := os.Create(path)
	chk(err)
	defer f.Close()
	chk(png.Encode(f, img))
}

func chk(e error) {
	if e != nil {
		panic(e)
	}
}
