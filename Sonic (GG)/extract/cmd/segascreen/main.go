// segascreen renders the EXACT SEGA boot screen of Sonic the Hedgehog (Game Gear)
// by running the real cartridge ROM on a Game Gear machine model (the z80 core +
// the gamegear VDP/mapper) and reading back what the boot code actually drew.
//
// Unlike titlegfx — which decompresses the logo tiles and lays them out in VRAM
// order (the tile SET) — this tool is an *oracle*: the game's own loader at $1CD7
// decompresses the tiles, programs CRAM, and builds the name table in code; we
// just let it run for a few frames and then compose the screen from the resulting
// VRAM name table ($3800) and CRAM. That gives the true tile MAP, pixel for pixel.
//
// Usage: segascreen <rom.gg> <outdir>
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"stupidcoder.com/tools/gamegear"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: segascreen <rom.gg> <outdir> [frames]")
		os.Exit(2)
	}
	rom, err := os.ReadFile(os.Args[1])
	chk(err)
	outdir := os.Args[2]
	chk(os.MkdirAll(outdir, 0o755))

	m := gamegear.NewMachine(rom)

	// Run enough frames for the boot to reach the SEGA logo and finish drawing it in.
	// The loader runs during init, but the name table is built progressively over
	// several vblanks (the logo "draws in"), so a few dozen frames are needed for the
	// complete glyph set to be laid out; 60 is comfortably past that.
	frames := 60
	if len(os.Args) > 3 {
		fmt.Sscanf(os.Args[3], "%d", &frames)
	}
	for i := 0; i < frames; i++ {
		if !m.RunFrame() {
			fmt.Printf("CPU halted at frame %d: %s\n", i, m.CPU.HaltReason)
			break
		}
	}
	fmt.Printf("ran %d frames; PC=$%04X  VDP reg0=$%02X reg1=$%02X reg2=$%02X\n",
		frames, m.CPU.PC, m.VDP.Regs[0], m.VDP.Regs[1], m.VDP.Regs[2])

	// Name table base from VDP register 2: bits 1-3 select the $0800-aligned base.
	ntBase := int(m.VDP.Regs[2]&0x0E) << 10
	fmt.Printf("name table base = $%04X (reg2=$%02X)\n", ntBase, m.VDP.Regs[2])

	// CRAM -> a 32-colour palette (BG 0-15, sprite 16-31).
	pal := gamegear.Palette(m.VDP.CRAM[:])
	fmt.Printf("CRAM BG : ")
	for i := 0; i < 16; i++ {
		w := uint16(m.VDP.CRAM[2*i]) | uint16(m.VDP.CRAM[2*i+1])<<8
		fmt.Printf("%03X ", w&0xFFF)
	}
	fmt.Printf("\nCRAM SPR: ")
	for i := 16; i < 32; i++ {
		w := uint16(m.VDP.CRAM[2*i]) | uint16(m.VDP.CRAM[2*i+1])<<8
		fmt.Printf("%03X ", w&0xFFF)
	}
	fmt.Println()

	// Compose the full 32x28 name table from VRAM tiles + the name table.
	nt := m.VDP.VRAM[ntBase:]
	tiles := m.VDP.VRAM[:]
	full := gamegear.RenderNameTable(nt, tiles, 32, 28, pal)
	writePNG(filepath.Join(outdir, "sega.screen.png"), scale(full, 2))
	fmt.Printf("wrote sega.screen.png (256x224 full name table)\n")

	// Crop the Game Gear's visible 160x144 window (centred in the 256x192 display).
	gg := full.SubImage(image.Rect(48, 24, 48+160, 24+144)).(*image.Paletted)
	writePNG(filepath.Join(outdir, "sega.gg.png"), scale(gg, 3))
	fmt.Printf("wrote sega.gg.png (160x144 Game Gear window)\n")

	// Report the logo's extent in the name table (the rows/columns that aren't the
	// $70 background fill), as evidence the screen is the code-built map, not a dump.
	minR, maxR, cells := 28, -1, 0
	for row := 0; row < 28; row++ {
		for col := 0; col < 32; col++ {
			o := ntBase + (row*32+col)*2
			if e := gamegear.DecodeNameEntry(m.VDP.VRAM[o], m.VDP.VRAM[o+1]); e.Tile != 112 {
				cells++
				if row < minR {
					minR = row
				}
				if row > maxR {
					maxR = row
				}
			}
		}
	}
	fmt.Printf("logo occupies name-table rows %d-%d, %d non-background cells\n", minR, maxR, cells)
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
