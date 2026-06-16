// animprobe finds a level's animated background tiles and the code that drives them.
// It boots an act, then watches VRAM tile memory ($0000-$1FFF) across idle frames: tiles
// whose 32 bytes change frame-to-frame are the animations (rings, water, flowers, ...).
// It also arms the VDP write watchpoint over one animated tile to capture the PC that
// writes it, so the animation routine + its frame table can be traced.
//
// Usage: animprobe <rom.gg> [act]
package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"

	"stupidcoder.com/tools/gamegear"
)

func main() {
	rom, _ := os.ReadFile(os.Args[1])
	act := 0
	if len(os.Args) > 2 {
		act, _ = strconv.Atoi(os.Args[2])
	}
	m := gamegear.NewMachine(rom)
	m.CapturePC = 0x0A73
	for i := 0; i < 700; i++ {
		m.RunFrame()
	}
	for round := 0; round < 40 && !m.Captured; round++ {
		m.Pad00 = 0x7F
		m.Write(0xD238, byte(act))
		for i := 0; i < 8; i++ {
			m.RunFrame()
			m.Write(0xD238, byte(act))
		}
		m.Pad00 = 0xFF
		for k := 0; k < 242 && !m.Captured; k++ {
			m.Write(0xD238, byte(act))
			m.RunFrame()
		}
	}
	for i := 0; i < 60; i++ { // settle (palette fade etc.)
		m.RunFrame()
	}

	w := func(a uint16) int { return int(m.Read(a)) | int(m.Read(a+1))<<8 }
	bank11 := func(addr int) int { return 0x2C000 + (addr - 0x4000) } // bank 11 in slot 1
	fmt.Printf("anim ptrs: ($D28E)=$%04X(file $%05X) ($D290)=$%04X ($D292)=$%02X ($D293)=$%02X zone($D2D5)=$%02X\n",
		w(0xD28E), bank11(w(0xD28E)), w(0xD290), m.Read(0xD292), m.Read(0xD293), m.Read(0xD2D5))

	// Diff tile memory across idle frames to find which tiles animate.
	snap := func() [0x2000]byte { var b [0x2000]byte; copy(b[:], m.VDP.VRAM[:0x2000]); return b }
	prev := snap()
	changed := map[int]int{} // tile -> how many frames it changed
	for f := 0; f < 240; f++ {
		m.RunFrame()
		cur := snap()
		for t := 0; t < 256; t++ {
			for j := 0; j < 32; j++ {
				if cur[t*32+j] != prev[t*32+j] {
					changed[t]++
					break
				}
			}
		}
		prev = cur
	}
	var tiles []int
	for t := range changed {
		tiles = append(tiles, t)
	}
	sort.Ints(tiles)
	fmt.Printf("act %d animated tiles (tile: frames-changed of 240):\n", act)
	for _, t := range tiles {
		fmt.Printf("  tile %3d ($%02X): changed %d times\n", t, t, changed[t])
	}

	// Watch one animated tile region and capture the writer PC.
	if len(tiles) > 0 {
		t := tiles[len(tiles)-1] // a high tile (e.g. the rings at 252-255)
		lo := uint16(t * 32)
		m.Watch(lo, lo+32)
		for i := 0; i < 120; i++ {
			m.RunFrame()
		}
		type pc struct {
			a uint16
			n int
		}
		var h []pc
		for a, n := range m.WatchPCs {
			h = append(h, pc{a, n})
		}
		sort.Slice(h, func(i, j int) bool { return h[i].n > h[j].n })
		fmt.Printf("\nwriters of tile %d (VRAM $%04X):\n", t, lo)
		for i := 0; i < 8 && i < len(h); i++ {
			fmt.Printf("  PC $%04X x%d\n", h[i].a, h[i].n)
		}
	}
}
