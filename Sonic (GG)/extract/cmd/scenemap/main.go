// scenemap statically decodes and renders the type-2 attract screen of Sonic the
// Hedgehog (Game Gear) — the screen loaded by the routine traced at $0C7A — without
// running the game. Everything here comes from reading that loader: it decompresses
// background tiles from (bank $0C, $171A), then loads the name table from a stored
// RLE map in bank 5 (two layers at $6C6D and $6DC3, the engine's $0502 codec), and
// uses background palette index $16. This tool reproduces exactly those steps to see
// what scenes 9..17 display — a test of how far pure tracing gets us.
//
// Usage: scenemap <rom.gg> <outdir>
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

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: scenemap <rom.gg> <outdir>")
		os.Exit(2)
	}
	rom, err := os.ReadFile(os.Args[1])
	chk(err)
	outdir := os.Args[2]
	chk(os.MkdirAll(outdir, 0o755))

	// Background tiles: $0C7A does CALL $0406 with A=$0C, HL=$171A -> VRAM $0000.
	tileSrc := decomp.SourceOffset(0x0C, 0x171A)
	tiles := decomp.Decompress(rom, tileSrc)
	fmt.Printf("BG tiles: file $%05X -> %d bytes = %d tiles\n", tileSrc, len(tiles), len(tiles)/32)

	// Name table: two $0502 RLE layers from bank 5, both to VRAM $3800.
	//   $0CAA  HL=$6C6D BC=$0156 $D20F=$10   (priority base)
	//   $0CBD  HL=$6DC3 BC=$0198 $D20F=$00   (overlay)
	// Bank 5 in slot 1: z80 $6C6D -> file 5*$4000 + ($6C6D-$4000).
	const bank5 = 5 * 0x4000
	l1 := decomp.LoadRLE(rom, bank5+(0x6C6D-0x4000), 0x0156, 0x10)
	l2 := decomp.LoadRLE(rom, bank5+(0x6DC3-0x4000), 0x0198, 0x00)
	fmt.Printf("name table: layer1 %d cells (hi $10), layer2 %d cells (hi $00)\n", len(l1)/2, len(l2)/2)

	// Compose the 32x28 name table: layer 1 from cell 0, then layer 2 overwrites
	// from cell 0 (both $0502 calls reset the VRAM address to $3800).
	nt := make([]byte, 32*28*2)
	copy(nt, l1)
	copy(nt, l2)

	// Palette: $0AAB loads a black start (index $16) then fades toward the real
	// targets — background index $0C and sprite index $0D (the $0B3F fade loader).
	pal := bankPalette(rom, 0x0C, 0x0D)

	full := gamegear.RenderNameTable(nt, tiles, 32, 28, pal)
	writePNG(filepath.Join(outdir, "worldmap.screen.png"), scale(full, 2))
	gg := full.SubImage(image.Rect(48, 24, 48+160, 24+144)).(*image.Paletted)
	writePNG(filepath.Join(outdir, "worldmap.gg.png"), scale(gg, 3))
	fmt.Printf("wrote worldmap.screen.png + worldmap.gg.png\n")
}

// bankPalette resolves a 32-colour palette from the bank-8 pointer table at file
// $23400 (entry = *($23400+i*2); data = $23400 + entry): the 16 background colours
// from index bg, the 16 sprite colours from index spr.
func bankPalette(rom []byte, bg, spr int) color.Palette {
	const tab = 0x23400
	read := func(i int) []byte {
		ptr := int(rom[tab+i*2]) | int(rom[tab+i*2+1])<<8
		off := tab + ptr
		fmt.Printf("palette $%02X: entry $%04X -> file $%05X\n", i, ptr, off)
		return rom[off : off+32]
	}
	cram := make([]byte, 64)
	copy(cram, read(bg))
	copy(cram[32:], read(spr))
	return gamegear.Palette(cram)
}

func scale(src *image.Paletted, n int) *image.Paletted {
	b := src.Bounds()
	out := image.NewPaletted(image.Rect(0, 0, b.Dx()*n, b.Dy()*n), src.Palette)
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			ci := src.ColorIndexAt(b.Min.X+x, b.Min.Y+y)
			for dy := 0; dy < n; dy++ {
				for dx := 0; dx < n; dx++ {
					out.SetColorIndex(x*n+dx, y*n+dy, ci)
				}
			}
		}
	}
	return out
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
