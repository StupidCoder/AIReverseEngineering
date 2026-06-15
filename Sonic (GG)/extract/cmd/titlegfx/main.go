// titlegfx decompresses the tiles of Sonic the Hedgehog (Game Gear)'s first boot
// screen — the SEGA logo — and renders them to a PNG, to verify the decompressor
// (Sonic.md Part IV) and the Game Gear tile/palette decoders against real data.
//
// The screen loader at $1CD7 (sega_logo) decompresses the background tiles from
// (bank $0C, $FA74) to VRAM $0000 and uses background palette index $12; that
// palette lives in the pointer table in bank 8 at $7400. This tool reproduces both.
// (The game's title screen proper is a later screen loaded the same way.)
//
// Usage: titlegfx <rom.gg> <outdir>
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

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: titlegfx <rom.gg> <outdir>")
		os.Exit(2)
	}
	rom, err := os.ReadFile(os.Args[1])
	chk(err)
	outdir := os.Args[2]
	chk(os.MkdirAll(outdir, 0o755))

	// Background tiles: (bank $0C, $FA74) -> a flat file offset spanning banks.
	src := decomp.SourceOffset(0x0C, 0xFA74)
	h := decomp.ReadHeader(rom, src)
	tiles := decomp.Decompress(rom, src)
	nt := len(tiles) / 32
	fmt.Printf("BG tiles: source file $%05X (bank %d)  header{match+$%X lit+$%X count=%d}  -> %d bytes = %d tiles\n",
		src, src/0x4000, h.MatchOff, h.LitOff, h.Count, len(tiles), nt)

	// Background palette index $12 via the bank-8 pointer table at $7400
	// (file $23400): entry = *($23400 + $12*2); palette data = $23400 + entry.
	const palTab = 0x23400
	pi := 0x12
	ptr := int(rom[palTab+pi*2]) | int(rom[palTab+pi*2+1])<<8
	palOff := palTab + ptr
	pal := gamegear.Palette(rom[palOff : palOff+32]) // 16 background colours
	fmt.Printf("BG palette $%02X: table entry $%04X -> data file $%05X\n", pi, ptr, palOff)
	fmt.Printf("  colours: ")
	for i := 0; i < 16; i++ {
		w := int(rom[palOff+2*i]) | int(rom[palOff+2*i+1])<<8
		fmt.Printf("%03X ", w&0xFFF)
	}
	fmt.Println()

	sheet := scale(gamegear.TileSheet(tiles, pal, nt, 16), 3)
	out := filepath.Join(outdir, "sega.tiles.png")
	writePNG(out, sheet)
	fmt.Printf("wrote %s (%d tiles)\n", out, nt)
}

func scale(src *image.Paletted, n int) *image.Paletted {
	b := src.Bounds()
	out := image.NewPaletted(image.Rect(0, 0, b.Dx()*n, b.Dy()*n), src.Palette)
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			ci := src.ColorIndexAt(x, y)
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
