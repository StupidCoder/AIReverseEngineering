// bganim watches the name-table cells at a type-$50 object's block while the game
// idles in Green Hills Act 1, to verify the $7B29 handler repaints its own map cell
// with an animated block sequence (patterns $7BC1 -> blocks $7B99).
package main

import (
	"fmt"
	"os"

	"stupidcoder.com/tools/gamegear"
)

func main() {
	rom, _ := os.ReadFile(os.Args[1])
	m := gamegear.NewMachine(rom)
	word := func(a uint16) uint16 { return uint16(m.Read(a)) | uint16(m.Read(a+1))<<8 }
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

	fmt.Printf("cam=$%04X sonic=%d\n", word(0xD254), int(word(0xD3FD+2)))
	// The $50 at (288, 352): NT cell for world (x,y) = $3800 + ((y/8%28)*32 + x/8%32)*2.
	// Watch its 2x2 cells (4 tile entries).
	cell := func(wx, wy int) uint16 {
		return uint16(0x3800 + ((wy/8%28)*32+wx/8%32)*2)
	}
	addrs := []uint16{cell(288, 352), cell(288, 360), cell(288, 368), cell(288, 376)}
	slot17 := uint16(0xD3FD + 17*26)
	last := ""
	for f := 0; f < 400; f++ {
		m.RunFrame()
		s := fmt.Sprintf("phase=%d cnt=%2d f24=%02X req=%04X/%04X src=%04X nt:",
			m.Read(slot17+18), m.Read(slot17+17), m.Read(slot17+24),
			word(0xD2AB), word(0xD2AD), word(0xD2AF))
		for _, a := range addrs {
			s += fmt.Sprintf(" %02X%02X", m.VDP.VRAM[a], m.VDP.VRAM[a+1])
		}
		if s != last {
			fmt.Printf("frame %3d: %s\n", f, s)
			last = s
		}
	}
}
