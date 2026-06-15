// screentrace runs Sonic the Hedgehog (Game Gear) on the Game Gear oracle for many
// frames and reports when the on-screen image or the top-level game mode changes.
// It is the analytical step before rendering a *named* screen: it shows the boot
// progress as a timeline of (frame, game-mode $D240, VRAM/CRAM hash), so we can see
// exactly when the SEGA logo gives way to the title screen and what mode drives it.
//
// No buttons are pressed (the unmapped controller ports read $FF = nothing held),
// so the game free-runs through its attract sequence on its own.
//
// Usage: screentrace <rom.gg> [frames]
package main

import (
	"fmt"
	"hash/crc32"
	"sort"
	"os"

	"stupidcoder.com/tools/gamegear"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: screentrace <rom.gg> [frames]")
		os.Exit(2)
	}
	rom, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	frames := 400
	if len(os.Args) > 2 {
		fmt.Sscanf(os.Args[2], "%d", &frames)
	}

	m := gamegear.NewMachine(rom)
	m.Watch(0x3800, 0x3F00) // watch every write into the name table

	// vramHash fingerprints the *content* of the screen — the tile patterns and the
	// name table — but NOT CRAM, so a palette fade (which rewrites CRAM every frame
	// while the picture stays put) doesn't drown out the discrete moments when the
	// game actually loads a new screen.
	vramHash := func() uint32 {
		h := crc32.NewIEEE()
		h.Write(m.VDP.VRAM[:])
		return h.Sum32()
	}
	// nameCells counts non-background ($70) name-table entries: a rough "how much is
	// on screen" gauge that distinguishes the sparse logo from the full title.
	nameCells := func() int {
		n := 0
		for i := 0x3800; i < 0x3F00; i += 2 {
			if m.VDP.VRAM[i] != 0x70 || m.VDP.VRAM[i+1] != 0 {
				n++
			}
		}
		return n
	}

	var lastHash uint32
	fmt.Printf("%-6s %-6s %-7s %-9s %s\n", "frame", "cells", "sprites", "vramhash", "event")
	for i := 0; i <= frames; i++ {
		hash := vramHash()
		if i == 0 || hash != lastHash {
			fmt.Printf("%-6d %-6d %-7d %08X  mode($D240)=$%02X\n",
				i, nameCells(), m.VDP.ActiveSprites(), hash, m.Read(0xD240))
			lastHash = hash
		}
		if !m.RunFrame() {
			fmt.Printf("CPU halted at frame %d: %s\n", i, m.CPU.HaltReason)
			break
		}
	}

	// Where did all the drawing land? Per-1KB VRAM region write totals over the run.
	fmt.Printf("\nVRAM writes by region (cumulative):\n")
	label := func(r int) string {
		switch {
		case r < 14:
			return "tile patterns"
		case r == 14:
			return "name table $3800"
		default:
			return "sprite table $3F00"
		}
	}
	for r := 0; r < 16; r++ {
		if m.VDP.Writes[r] > 0 {
			fmt.Printf("  $%04X-$%04X  %-20s %d\n", r*0x400, r*0x400+0x3FF, label(r), m.VDP.Writes[r])
		}
	}
	fmt.Printf("  CRAM writes: %d\n", m.VDP.CRAMWrites)

	// Which routines wrote the name table? PC histogram from the watchpoint, sorted.
	type pc struct {
		addr  uint16
		count int
	}
	var pcs []pc
	for a, c := range m.WatchPCs {
		pcs = append(pcs, pc{a, c})
	}
	sort.Slice(pcs, func(i, j int) bool { return pcs[i].count > pcs[j].count })
	fmt.Printf("\nname-table writers (PC just after the OUT, byte count):\n")
	for i, p := range pcs {
		if i >= 12 {
			break
		}
		fmt.Printf("  near $%04X  %d\n", p.addr, p.count)
	}
}
