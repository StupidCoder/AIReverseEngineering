// spawntrace instruments the oracle to locate Super Mario Land's enemy/object
// placement data. It boots the game into a level, holds Right so the camera
// scrolls, and watches the 10 active object slots ($D100-$D19F). Whenever a
// slot's type byte ($D1x0) transitions from $FF (empty) to a real type, it
// records the spawning PC and a ring buffer of the most recent banked-ROM reads
// — which reveals the per-level spawn table and the routine that walks it.
//
// This is a GUIDE only: it tells us which routine/table to disassemble; the
// actual data is then decoded from the ROM (never scraped from this run).
//
//	go run ./cmd/spawntrace [-rom PATH] [-id NN] [-frames N]
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"stupidcoder.com/tools/gameboy"
)

type romRead struct {
	pc   uint16
	bank int
	addr uint16
}

func main() {
	rom := flag.String("rom", "../Super Mario Land (World).gb", "ROM path")
	id := flag.String("id", "", "level id (hex, e.g. 11); empty = play default 1-1")
	frames := flag.Int("frames", 600, "gameplay frames to run while holding Right")
	flag.Parse()

	data, err := os.ReadFile(*rom)
	if err != nil {
		fmt.Fprintln(os.Stderr, "spawntrace:", err)
		os.Exit(1)
	}

	m := gameboy.NewMachine(data)

	// Ring buffer of recent banked-ROM reads.
	const ringSize = 12
	ring := make([]romRead, 0, ringSize)
	m.OnROMRead = func(pc uint16, bank int, addr uint16) {
		if len(ring) < ringSize {
			ring = append(ring, romRead{pc, bank, addr})
		} else {
			copy(ring, ring[1:])
			ring[ringSize-1] = romRead{pc, bank, addr}
		}
	}

	// Shadow of the 10 slot type bytes; report FF->type transitions.
	var slotType [10]byte
	for i := range slotType {
		slotType[i] = 0xFF
	}
	tracing := false
	m.OnWrite = func(pc, addr uint16, v byte) {
		if !tracing || addr < 0xD100 || addr > 0xD19F || addr&0x0F != 0 {
			return
		}
		slot := int((addr - 0xD100) >> 4)
		prev := slotType[slot]
		slotType[slot] = v
		if prev == 0xFF && v != 0xFF {
			fmt.Printf("\nSPAWN slot %d type=$%02X  by PC=$%04X (bank %d)  frame ~%d\n",
				slot, v, pc, m.ROMBank(), curFrame)
			fmt.Printf("  recent banked reads (oldest first):\n")
			for _, r := range ring {
				fmt.Printf("    PC=$%04X  bank %d  $%04X\n", r.pc, r.bank, r.addr)
			}
		}
	}

	// Boot to the title, press Start to begin play.
	m.RunFrames(80)
	for f := 0; f < 6; f++ {
		m.Buttons = gameboy.BtnStart
		m.RunFrame()
	}
	m.Buttons = 0

	// Optional warp to another level id.
	if *id != "" {
		idv, _ := strconv.ParseUint(*id, 16, 8)
		for f := 0; f < 40; f++ {
			m.Write(0xFFB4, byte(idv))
			m.RunFrame()
		}
	} else {
		m.RunFrames(40)
	}

	// Reset slot shadow to current state, then start tracing and hold Right.
	for s := 0; s < 10; s++ {
		slotType[s] = m.Read(uint16(0xD100 + s*0x10))
	}
	tracing = true
	m.Buttons = gameboy.BtnRight
	for curFrame = 0; curFrame < *frames; curFrame++ {
		m.RunFrame()
	}
	fmt.Println("\ndone.")
}

var curFrame int
