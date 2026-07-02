package main

import (
	"fmt"

	"sonicgg/extract/decomp"

	"stupidcoder.com/tools/gamegear"
)

// dumpCells prints my decoded map + collision lookup vs the live $C000 window for the
// cells around Sonic's spawn column.
func dumpCells(rom []byte, m *gamegear.Machine, act, col int, rows []int) {
	d := descTable + w16(rom, descTable+act*2)
	stride := w16(rom, d+1)
	zone := int(rom[d]) // desc+0 = the engine zone ($D2D5)
	blocks := decomp.LoadMapRLE(rom, 0x14000+w16(rom, d+15), w16(rom, d+17))
	attr := w16(rom, 0x343D+zone*2)
	for _, r := range rows {
		mine := int(blocks[r*stride+col])
		live := int(m.Read(uint16(0xC000 + r*stride + col)))
		shape := int(rom[attr+live]) & 0x3F
		prof := w16(rom, 0x3E7A+shape*2)
		var ps [32]int8
		for i := 0; i < 32; i++ {
			ps[i] = int8(rom[prof+i])
		}
		fmt.Printf("  row %2d col %d: mine=$%02X live=$%02X shape=$%02X bias=%d profile=%v\n",
			r, col, mine, live, shape, rom[0x39DA+shape], ps)
	}
	fmt.Printf("  live zone byte $D2D5=%d, stride word=$%04X\n", m.Read(0xD2D5), stride)
}
