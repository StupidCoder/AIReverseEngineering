// render boots Super Mario Land on the tools/gameboy oracle, lets it run to the
// title screen, and decodes what the game put in VRAM into PNGs: the raw 2bpp tile
// sheet and the composed background map. It is the Game Boy equivalent of the other
// games' webexport/render tools — proof that the hardware graphics formats decode
// correctly, straight from a real run of the ROM.
//
//	go run ./cmd/render [-rom PATH] [-frames N] [-o DIR]
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"stupidcoder.com/tools/gameboy"
)

func main() {
	rom := flag.String("rom", "../Super Mario Land (World).gb", "path to the ROM")
	frames := flag.Int("frames", 80, "frames to run before capturing")
	play := flag.Bool("play", false, "press Start and run into the first level, then capture a screen")
	outDir := flag.String("o", "../rendered", "output directory")
	flag.Parse()

	data, err := os.ReadFile(*rom)
	if err != nil {
		fmt.Fprintln(os.Stderr, "render:", err)
		os.Exit(1)
	}
	m := gameboy.NewMachine(data)
	if done := m.RunFrames(*frames); done < *frames {
		fmt.Fprintf(os.Stderr, "render: CPU halted after %d frames at $%04X\n", done, m.CPU.PC)
		os.Exit(1)
	}

	vram := m.VRAM()
	lcdc := m.Read(0xFF40)
	bgp := m.Read(0xFF47)
	fmt.Printf("after %d frames: LCDC=$%02X BGP=$%02X\n", *frames, lcdc, bgp)

	os.MkdirAll(*outDir, 0o755)
	pal := gameboy.DMGPalette(bgp)

	// The 384 tiles of VRAM tile memory ($8000-$97FF), in a 16-wide sheet.
	save(filepath.Join(*outDir, "title-tiles.png"),
		gameboy.TileSheet(vram[0:0x1800], pal, 384, 16))

	// The composed background — SML's title uses the $9800 map with signed $8800
	// tile addressing (LCDC bit 3 = 0, bit 4 = 0).
	mapHi := lcdc&0x08 != 0
	save(filepath.Join(*outDir, "title-bg.png"),
		gameboy.RenderBGMap(vram, lcdc, mapHi, pal))

	if *play {
		// Press Start to leave the title, then run into the level and capture the
		// authentic viewport (background + sprites).
		for f := 0; f < 6; f++ {
			m.Buttons = gameboy.BtnStart
			m.RunFrame()
		}
		m.Buttons = 0
		m.RunFrames(180)
		lcdc, scy, scx := m.Read(0xFF40), m.Read(0xFF42), m.Read(0xFF43)
		bgp, obp0, obp1 := m.Read(0xFF47), m.Read(0xFF48), m.Read(0xFF49)
		screen := gameboy.RenderScreen(m.VRAM(), m.OAM(), lcdc, scx, scy, bgp, obp0, obp1)
		save(filepath.Join(*outDir, "level-1-1.png"), screen)
		fmt.Printf("level: LCDC=$%02X SCX=%d SCY=%d sprites=%d -> level-1-1.png\n",
			lcdc, scx, scy, countSprites(m.OAM()))
	}

	fmt.Println("wrote PNGs to", *outDir)
}

func countSprites(oam []byte) int {
	n := 0
	for _, s := range gameboy.DecodeOAM(oam) {
		if s.Y > -16 && s.Y < 144 {
			n++
		}
	}
	return n
}

func save(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "render:", err)
		os.Exit(1)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, "render:", err)
		os.Exit(1)
	}
}
