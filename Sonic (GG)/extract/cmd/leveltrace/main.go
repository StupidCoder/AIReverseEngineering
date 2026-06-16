// leveltrace drives the Game Gear machine model from boot into actual gameplay —
// pressing Start to get past the title — and snapshots the live screen plus the
// scroll/map state. It is the oracle-assisted step for locating the level map:
// rather than chase reused pointers statically, run a level and read what the code
// actually does. It pins the map decompressor's arguments by capturing HL/BC/DE at
// $0A73 (CapturePC), and grabs the decompressor's $C000 output the instant it RETs
// (CapLo/CapHi) — that LOCATED the Zone 0 map source as bank 5, file $17430, len $0786,
// and gave the ground truth that verifies decomp.LoadMapRLE byte-perfect (cmd/levelmap).
//
// Holding Right+Jump (PadDC=$E7) makes the player run, so this also drives a real
// scroll: $D3FF advances and the terrain streamer (bank0 $0760/$0860) runs, which is how
// the live terrain draw was found (VRAM write watchpoint) rather than scroll_draw $3282
// (objects only).
//
// Usage: leveltrace <rom.gg> <outdir>
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sort"

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
			fmt.Printf(" %02X", m.Read(ptr+i))
		}
		fmt.Println()
	}

	run(700)             // reach the title
	m.CapturePC = 0x0A73 // grab HL (source ptr) + BC (length) at the map decompressor
	// Snapshot the decompressor's OUTPUT ($C000, 4 KB) the instant it returns to its
	// caller (PC leaves [$0A73,$0AA2]), before any other code can mutate the RAM map.
	m.CapLo, m.CapHi, m.CapOutBase = 0x0A73, 0x0AA2, 0xC000
	// Watch who fills the RAM map window during the level-load that follows.
	m.RAMWatchLo, m.RAMWatchHi = 0xC000, 0xD000
	m.RAMWatchPCs = map[uint16]int{}
	// Tap Start until we enter gameplay (a non-zero right scroll bound), then STOP
	// (further Start presses would pause / leave the level).
	for round := 0; round < 40 && word(0xD26F) == 0; round++ {
		m.Pad00 = 0x7F
		run(8)
		m.Pad00 = 0xFF
		for k := 0; k < 242 && word(0xD26F) == 0; k++ {
			run(1)
		}
		if word(0xD26F) != 0 { // level just loaded: grab the map before the intro mutates it
			clean := make([]byte, 0x1000)
			for i := range clean {
				clean[i] = m.Read(uint16(0xC000 + i))
			}
			chk(os.WriteFile(filepath.Join(outdir, "map_clean.bin"), clean, 0o644))
			run(8) // let the rest of the load finish
		}
	}
	{
		type pcc struct {
			a uint16
			n int
		}
		var h []pcc
		for a, n := range m.RAMWatchPCs {
			h = append(h, pcc{a, n})
		}
		sort.Slice(h, func(i, j int) bool { return h[i].n > h[j].n })
		fmt.Printf("RAM map ($C000) writers during level load: ")
		for i := 0; i < 12 && i < len(h); i++ {
			fmt.Printf("$%04X(%d) ", h[i].a, h[i].n)
		}
		fmt.Println()
		m.RAMWatchPCs = nil
	}
	if m.Captured {
		fmt.Printf("map decompressor $0A73: source HL=$%04X, length BC=$%04X, slot1=bank %d\n",
			m.CapHL, m.CapBC, m.CapSlot1)
	}
	if m.CapOutDone { // ground-truth decompressor output, grabbed at its RET
		chk(os.WriteFile(filepath.Join(outdir, "map_ret.bin"), m.CapOut[:], 0o644))
		fmt.Printf("dumped map_ret.bin: $C000 snapshot at $0A73 RET (decompressor output)\n")
	}
	fmt.Printf("all $0A73 calls during load (%d):\n", len(m.CapLog))
	for i, c := range m.CapLog {
		fmt.Printf("  call %2d: src HL=$%04X len BC=$%04X dest DE=$%04X slot1=bank %d\n",
			i, c.HL, c.BC, c.DE, c.Slot1)
		if c.DE == 0xC000 { // the call that fills the RAM map window — dump its exact source
			fileoff := int(c.Slot1)*0x4000 + int(c.HL-0x4000)
			if c.HL < 0x8000 && fileoff >= 0 && fileoff+int(c.BC) <= len(rom) {
				chk(os.WriteFile(filepath.Join(outdir, "map_src.bin"), rom[fileoff:fileoff+int(c.BC)], 0o644))
				fmt.Printf("           -> dumped map_src.bin (file $%05X, %d bytes)\n", fileoff, c.BC)
			}
		}
	}
	snap(fmt.Sprintf("%04d_inlevel", frame))
	dump("inlevel")
	chk(os.WriteFile(filepath.Join(outdir, "vram.bin"), m.VDP.VRAM[:], 0o644))
	chk(os.WriteFile(filepath.Join(outdir, "cram.bin"), m.VDP.CRAM[:], 0o644))
	ram := make([]byte, 0x2000)
	for i := range ram {
		ram[i] = m.Read(uint16(0xC000 + i))
	}
	chk(os.WriteFile(filepath.Join(outdir, "ram.bin"), ram, 0o644))

	// Hold Right + Jump and watch whether the scroll position EVER moves over a long run.
	m.PadDC = 0xE7 // Right (bit3) + Button 1 (bit4) low
	fmt.Printf("\nholding Right+Jump; watching scroll $D3FF / drawn $D254:\n")
	for round := 0; round < 20; round++ {
		run(100)
		fmt.Printf("  frame %4d: D3FF=$%04X D254=$%04X cam=$%04X in(D203)=$%02X\n",
			frame, word(0xD3FF), word(0xD254), word(0xD2AB), m.Read(0xD203))
		if round == 3 { // a varied surface view, before Sonic sinks into the fill
			snap(fmt.Sprintf("%04d_surface", frame))
			chk(os.WriteFile(filepath.Join(outdir, "vram_scroll.bin"), m.VDP.VRAM[:], 0o644))
			chk(os.WriteFile(filepath.Join(outdir, "cram_scroll.bin"), m.VDP.CRAM[:], 0o644))
			// Which code writes the RAM map window $C000-$D000 while scrolling?
			m.RAMWatchLo, m.RAMWatchHi = 0xC000, 0xD000
			m.RAMWatchPCs = map[uint16]int{}
			run(8)
			type pc2 struct {
				a uint16
				n int
			}
			var h []pc2
			for a, n := range m.RAMWatchPCs {
				h = append(h, pc2{a, n})
			}
			sort.Slice(h, func(i, j int) bool { return h[i].n > h[j].n })
			fmt.Printf("  RAM map ($C000) writers while scrolling: ")
			for i := 0; i < 14 && i < len(h); i++ {
				fmt.Printf("$%04X(%d) ", h[i].a, h[i].n)
			}
			fmt.Println()
			m.RAMWatchPCs = nil
		}
	}
	snap(fmt.Sprintf("%04d_right", frame))

	// Profile one frame: what code is the machine actually running in the level?
	m.PCHist = map[uint16]int{}
	m.Sample = true
	m.RunFrame()
	m.Sample = false
	type pc struct {
		a uint16
		n int
	}
	var hot []pc
	for a, n := range m.PCHist {
		hot = append(hot, pc{a, n})
	}
	sort.Slice(hot, func(i, j int) bool { return hot[i].n > hot[j].n })
	fmt.Printf("\nhottest PCs in one in-level frame (where the CPU spends time):\n")
	for i := 0; i < 16 && i < len(hot); i++ {
		fmt.Printf("  $%04X x%d\n", hot[i].a, hot[i].n)
	}
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
