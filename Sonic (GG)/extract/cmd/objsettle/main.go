// objsettle verifies the objplace reimplementation of the engine's object placement
// against the live game. It watches the object array ($D3FD, 32 records x 26 bytes)
// while the oracle plays Green Hills Act 1: an object is spawned at (blockX*32,
// blockY*32) but dormant (no hitbox) until the camera comes near; on activation its
// handler stores the hitbox (IX+13/14) and the shared move code $2CD4 snaps it onto
// the floor line. For every activation the live (x, y, box) is compared with the
// static prediction (objplace.Settle / DropToFloor for Sonic) — the placement rule
// the web viewer uses must reproduce the engine exactly.
//
// Usage: objsettle <rom.gg>
package main

import (
	"fmt"
	"os"

	"sonicgg/extract/decomp"
	"sonicgg/extract/objplace"

	"stupidcoder.com/tools/gamegear"
)

const (
	objArray  = 0xD3FD
	objSize   = 26
	objCount  = 32
	descTable = 0x15600
)

type slot struct {
	typ     byte
	x, y    uint16
	w, h    byte
	spr     uint16
	flags24 byte
}

func readSlots(m *gamegear.Machine) [objCount]slot {
	var s [objCount]slot
	for i := 0; i < objCount; i++ {
		b := uint16(objArray + i*objSize)
		s[i] = slot{
			typ:     m.Read(b),
			x:       uint16(m.Read(b+2)) | uint16(m.Read(b+3))<<8,
			y:       uint16(m.Read(b+5)) | uint16(m.Read(b+6))<<8,
			w:       m.Read(b + 13),
			h:       m.Read(b + 14),
			spr:     uint16(m.Read(b+15)) | uint16(m.Read(b+16))<<8,
			flags24: m.Read(b + 24),
		}
	}
	return s
}

func w16(rom []byte, o int) int { return int(rom[o]) | int(rom[o+1])<<8 }

func main() {
	rom, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	// Static predictions for act 0 (Green Hills 1) from the ROM alone.
	d := descTable + w16(rom, descTable)
	stride := w16(rom, d+1) // low byte selects the stride; GH1 = 256
	if s := stride & 0xFF; s != 0 {
		stride = s
	} else {
		stride = 256
	}
	blocks := decomp.LoadMapRLE(rom, 0x14000+w16(rom, d+15), w16(rom, d+17))
	lvl := objplace.NewLevel(rom, blocks, stride, int(rom[d])) // desc+0 = the engine zone

	type pred struct{ x, y int }
	preds := map[int]pred{} // slot index -> predicted rest position
	ot := descTable + w16(rom, d+30)
	n := int(rom[ot])
	fmt.Printf("static predictions (GH act 1, %d objects):\n", n)
	for k, p := 0, ot+1; k < n; k, p = k+1, p+3 {
		typ, bx, by := int(rom[p]), int(rom[p+1]), int(rom[p+2])
		rx, ry, g := lvl.Settle(typ, bx*32, by*32)
		preds[k+1] = pred{rx, ry} // slot 0 is Sonic; table records fill slots 1..n
		w, h, _ := objplace.Hitbox(rom, typ)
		fmt.Printf("  slot %2d type=$%02X spawn=(%4d,%4d) box=%2dx%-2d -> rest=(%4d,%4d) grounded=%v\n",
			k+1, typ, bx*32, by*32, w, h, rx, ry, g)
	}
	sx, sy := int(rom[d+13])*32, int(rom[d+14])*32
	rx, ry, g := lvl.DropToFloor(0, sx, sy)
	preds[0] = pred{rx, ry}
	fmt.Printf("  slot  0 Sonic   spawn=(%4d,%4d)           -> rest=(%4d,%4d) grounded=%v\n\n",
		sx, sy, rx, ry, g)

	// Live run: boot, Start into the level, then hold Right+Jump so the camera sweeps
	// the act and activates the objects one by one.
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
	fmt.Printf("in level: scene=$%02X cam=$%04X spawn ptr ($D217)=$%04X -> block(%d,%d)\n",
		m.Read(0xD238), word(0xD254), word(0xD217),
		m.Read(word(0xD217)), m.Read(word(0xD217)+1))

	check := func(i int, c slot, what string) {
		p, known := preds[i]
		verdict := "no prediction"
		if known {
			verdict = "MISMATCH"
			if p.x == int(c.x) && p.y == int(c.y) {
				verdict = "match"
			}
		}
		fmt.Printf("%-10s slot %2d type=$%02X live=(%4d,%4d) box=%2dx%-2d f24=%02X  pred=(%4d,%4d)  %s\n",
			what, i, c.typ, c.x, c.y, c.w, c.h, c.flags24, p.x, p.y, verdict)
	}

	prev := readSlots(m)
	seen := map[int]bool{}
	// Teleport plan: walk Sonic's world X ($D3FD+2) through each object's
	// neighbourhood so the camera (which tracks him) activates them all.
	targets := []int{}
	for k, p := 0, ot+1; k < n; k, p = k+1, p+3 {
		targets = append(targets, int(rom[p+1])*32-100)
	}
	ti := 0
	for f := 1; f <= 20000; f++ {
		if f == 300 {
			m.PadDC = 0xE7 // Right + jump
		}
		if f > 400 && f%400 == 0 && ti < len(targets) {
			x := targets[ti]
			ti++
			m.Write(0xD3FD+2, byte(x))
			m.Write(0xD3FD+3, byte(x>>8))
			m.Write(0xD3FD+5, 100) // drop him in from the sky at the new spot
			m.Write(0xD3FD+6, 0)
		}
		m.RunFrame()
		cur := readSlots(m)
		for i := range cur {
			p, c := prev[i], cur[i]
			if c.typ == 0xFF || seen[i] {
				continue
			}
			if i == 0 {
				// Sonic: check once he grounds (his spawn includes a short fall).
				if c.flags24&0x80 != 0 {
					check(0, c, "grounded")
					seen[0] = true
				}
				continue
			}
			// Others: check on activation (hitbox appears), i.e. their first live frame.
			if p.w == 0 && p.h == 0 && (c.w != 0 || c.h != 0) {
				check(i, c, "activated")
				seen[i] = true
			}
		}
		prev = cur
	}
}
