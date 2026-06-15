// leveltrace drives the Game Gear machine model from boot into actual gameplay —
// pressing Start to get past the title — and snapshots the live screen plus the
// scroll/map state. It is the oracle-assisted step for locating the level map:
// rather than chase reused pointers statically, run a level and read what $D2AF
// actually points to and which bank is paged. That LOCATED the Zone 0 map in bank 4
// ($D2AF = $7B99 = file $13B99), which is then analysed statically.
//
// Limitation: the simplified machine loads and renders the level (the screenshot is
// real Green Hills graphics) and the controller read registers ($D203 reflects the
// injected D-pad), but the player physics do not advance — driving live scrolling
// exceeds the non-cycle-accurate model's fidelity. So this captures the map's
// location and a static frame, not a streaming trace.
//
// Usage: leveltrace <rom.gg> <outdir>
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
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: leveltrace <rom.gg> <outdir>")
		os.Exit(2)
	}
	rom, err := os.ReadFile(os.Args[1])
	chk(err)
	outdir := os.Args[2]
	chk(os.MkdirAll(outdir, 0o755))

	m := gamegear.NewMachine(rom)

	word := func(a uint16) uint16 { return uint16(m.Read(a)) | uint16(m.Read(a+1))<<8 }
	snap := func(tag string) {
		pal := gamegear.Palette(m.VDP.CRAM[:])
		full := gamegear.RenderNameTable(m.VDP.VRAM[0x3800:], m.VDP.VRAM[:], 32, 28, pal)
		gg := full.SubImage(image.Rect(48, 24, 48+160, 24+144)).(*image.Paletted)
		writePNG(filepath.Join(outdir, "lvl_"+tag+".png"), scale(gg, 3))
		fmt.Printf("%-7s scene=$%02X mode=$%02X  D2AF=$%04X  cam(D2AB/D2AD)=$%04X/$%04X  "+
			"bounds(D26D/D26F)=$%04X/$%04X\n", tag, m.Read(0xD238), m.Read(0xD240),
			word(0xD2AF), word(0xD2AB), word(0xD2AD), word(0xD26D), word(0xD26F))
	}

	frame := 0
	run := func(n int) {
		for i := 0; i < n; i++ {
			m.RunFrame()
			frame++
		}
	}

	dump := func(tag string) {
		bank := m.Read(0xD22F)
		ptr := word(0xD2AF)
		fmt.Printf("%-12s in=$%02X(D203) cam=$%04X/$%04X slot1=b%-2d D2AF=$%04X bytes:",
			tag, m.Read(0xD203), word(0xD2AB), word(0xD2AD), bank, ptr)
		for i := uint16(0); i < 16; i++ {
			fmt.Printf(" %02X", m.Read(ptr + i))
		}
		fmt.Println()
	}

	run(700) // reach the title
	// Tap Start until we enter gameplay (a non-zero right scroll bound), then STOP
	// (further Start presses would pause / leave the level).
	for round := 0; round < 8 && word(0xD26F) == 0; round++ {
		m.Pad00 = 0x7F
		run(8)
		m.Pad00 = 0xFF
		run(242)
	}
	snap(fmt.Sprintf("%04d_inlevel", frame))
	dump("inlevel")

	// Now hold Right (D-pad only, no Start) and watch the camera and map pointer.
	m.PadDC = 0xF7 // Right pressed (bit 3 low)
	for round := 0; round < 12; round++ {
		run(30)
		dump(fmt.Sprintf("%04d_right", frame))
	}
	snap(fmt.Sprintf("%04d_right", frame))
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
