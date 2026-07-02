// spawncheck forces each act (writing $D238 during the Start-press load, the levelmap
// trick) and watches Sonic (slot 0 of $D3FD) settle, printing his live spawn and rest
// against the static prediction (descriptor +13/+14 dropped to the floor) — hunting the
// acts where the exported spawn Y looks wrong.
//
// Usage: spawncheck <rom.gg> [act...]
package main

import (
	"fmt"
	"os"
	"strconv"

	"sonicgg/extract/decomp"
	"sonicgg/extract/objplace"

	"stupidcoder.com/tools/gamegear"
)

const descTable = 0x15600

func w16(rom []byte, o int) int { return int(rom[o]) | int(rom[o+1])<<8 }

func main() {
	rom, _ := os.ReadFile(os.Args[1])
	acts := []int{0, 3, 6, 9}
	if len(os.Args) > 2 {
		acts = nil
		for _, a := range os.Args[2:] {
			n, _ := strconv.Atoi(a)
			acts = append(acts, n)
		}
	}
	for _, act := range acts {
		d := descTable + w16(rom, descTable+act*2)
		stride := w16(rom, d+1)
		blocks := decomp.LoadMapRLE(rom, 0x14000+w16(rom, d+15), w16(rom, d+17))
		lvl := objplace.NewLevel(rom, blocks, stride, int(rom[d])) // desc+0 = the engine zone ($D2D5)
		sx, sy := int(rom[d+13])*32, int(rom[d+14])*32
		px, py, g := lvl.DropToFloor(0, sx, sy)

		m := gamegear.NewMachine(rom)
		word := func(a uint16) uint16 { return uint16(m.Read(a)) | uint16(m.Read(a+1))<<8 }
		for i := 0; i < 700; i++ {
			m.RunFrame()
		}
		for round := 0; round < 40 && word(0xD26F) == 0; round++ {
			m.Pad00 = 0x7F
			for i := 0; i < 8; i++ {
				m.Write(0xD238, byte(act))
				m.RunFrame()
			}
			m.Pad00 = 0xFF
			for k := 0; k < 242 && word(0xD26F) == 0; k++ {
				m.Write(0xD238, byte(act))
				m.RunFrame()
			}
		}
		// let Sonic settle; grab his position when grounded (f24 bit7) or after 300 frames
		var lx, ly uint16
		var f24 byte
		for i := 0; i < 300; i++ {
			m.RunFrame()
			lx = uint16(m.Read(0xD3FD+2)) | uint16(m.Read(0xD3FD+3))<<8
			ly = uint16(m.Read(0xD3FD+5)) | uint16(m.Read(0xD3FD+6))<<8
			f24 = m.Read(0xD3FD + 24)
			if f24&0x80 != 0 {
				break
			}
		}
		sp := word(0xD217)
		fmt.Printf("act %2d scene=$%02X desc spawn (%d,%d)=(%4d,%4d)  pred rest (%4d,%4d) g=%-5v  LIVE (%4d,%4d) f24=%02X  ($D217)=$%04X bytes(%d,%d)\n",
			act, m.Read(0xD238), rom[d+13], rom[d+14], sx, sy, px, py, g, lx, ly, f24, sp, m.Read(sp), m.Read(sp+1))
		cx := (sx + 8) / 32
		rr := sy / 32
		dumpCells(rom, m, act, cx, []int{rr, rr + 1, rr + 2})
	}
}
