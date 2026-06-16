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
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"sonicgg/extract/decomp"

	"stupidcoder.com/tools/gamegear"
)

// The Green Hills (Zone 0) acts, read from the level-resource descriptor table at bank 5
// $5600. Each act's descriptor holds a map record (address, length) at offset +15; the
// loader $199D normalises the address (high byte >= $40 -> bank 6, else bank 5; +$4000)
// and decompresses it with $0A73. Crucially the descriptors for all three acts carry the
// SAME tile-table pointer ($0000 -> $D249 = $4000 = file $10000), the SAME tile-set
// pointer ($2ED5) and the SAME following palette/graphics fields — so the three acts
// share tiles, macro-blocks and palette, and differ ONLY in the block-index map. (Found
// statically; Act 1's map is the one the oracle verified byte-perfect.)
var acts = []struct {
	name     string
	mapFile  int // file offset of the compressed map
	mapLen   int // compressed length (BC)
	widthBlk int // played width in blocks (right scroll bound $D26F / 32)
}{
	{"act1", 0x17430, 0x0786, 198}, // src z80 $3430 (bank 5)
	{"act2", 0x17BB6, 0x06B4, 101}, // src z80 $3BB6 (bank 5)
	{"act3", 0x1826A, 0x033D, 80},  // src z80 $426A -> bank 6
}

const (
	mapCols   = 256
	mapRows   = 16
	blockTile = 0x10000 // file offset of the shared 16-byte-per-block tile table (bank 4)
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

	// 1. Decode every act's map straight from the cartridge — no machine involved.
	for _, a := range acts {
		mp := decomp.LoadMapRLE(rom, a.mapFile, a.mapLen)
		fmt.Printf("%s: decoded %d bytes from ROM ($%05X, src %d B -> %.2fx), played width %d blocks\n",
			a.name, len(mp), a.mapFile, a.mapLen, float64(len(mp))/float64(a.mapLen), a.widthBlk)
	}
	romMap := decomp.LoadMapRLE(rom, acts[0].mapFile, acts[0].mapLen) // act 1, for the oracle check

	// 2. Boot the oracle into Act 1 to (a) verify that decode and (b) grab the shared
	//    tile set + palette (identical across the three acts, per the descriptors).
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

	// 3. Render EACH act's full map with the (shared) real in-game tiles. The $0760
	//    terrain expander turns each block index into a 4x4 grid of 8x8 tiles: tile(r,c)
	//    is the byte at blockTile + index*16 + r*4 + c (row-major, 4 wide). blockTile is
	//    the table the loader points $D249 at: z80 $4000 read with BANK 4 in slot 1
	//    ($0760's prologue $0726 pages it) = file $10000. Attr contributes only a priority
	//    bit (no flip / no palette select), so every terrain tile uses the BG palette —
	//    the pixels come entirely from the tile index + the loaded tile set.
	for _, a := range acts {
		mp := decomp.LoadMapRLE(rom, a.mapFile, a.mapLen)
		out := filepath.Join(outdir, "level_greenhills_"+a.name+".png")
		renderMap(out, rom, mp, tiles, pal, a.widthBlk)
		chk(os.WriteFile(filepath.Join(outdir, "level_map_"+a.name+".bin"), mp, 0o644))
		fmt.Printf("%s: wrote %s (%d blocks wide)\n", a.name, filepath.Base(out), a.widthBlk)
	}
}

// renderMap paints a decoded block-index map into a full-resolution PNG, clipped to the
// act's PLAYED width (cols blocks = the camera right-scroll bound $D26F / 32). The
// decompressed map is always a fixed 16x256 grid, but only the first `cols` columns are
// reachable in-game; the rest is off-level filler. Each block index expands to a 4x4 grid
// of 8x8 tiles via the block tile table at blockTile, drawn with the tile set and palette.
func renderMap(path string, rom, romMap, tiles []byte, pal color.Palette, cols int) {
	if cols > mapCols {
		cols = mapCols
	}
	bw := cols * 4         // tiles wide  (4 tiles per block)
	const bh = mapRows * 4 // tiles tall
	img := image.NewPaletted(image.Rect(0, 0, bw*8, bh*8), pal)
	for row := 0; row < mapRows; row++ {
		for col := 0; col < cols; col++ {
			idx := int(romMap[row*mapCols+col])
			def := blockTile + idx*16
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
	writePNG(path, img)
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
