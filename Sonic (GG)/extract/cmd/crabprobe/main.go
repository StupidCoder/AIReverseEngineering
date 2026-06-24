// crabprobe statically checks, for each ground enemy placed in Green Hills Act 1, where
// the level's solid ground actually is relative to the enemy's placement block — to decide
// whether the metasprite (drawn with its grid top at blockY*32) sits ON the ground, floats
// above it, or overlaps it. It decodes the block-index map, the per-zone block collision
// shapes ($343D) and the shape height profiles ($3E7A), and for each crab/beetle reports
// the placement row, the first solid row at/below it, and the surface pixel Y.
package main

import (
	"fmt"
	"os"

	"sonicgg/extract/decomp"
)

const (
	descTable = 0x15600
	attrPtrs  = 0x343D
	shapeTbl  = 0x3E7A
)

func w(rom []byte, o int) int { return int(rom[o]) | int(rom[o+1])<<8 }

func main() {
	rom, _ := os.ReadFile(os.Args[1])
	act := 0 // Green Hills Act 1
	d := descTable + w(rom, descTable+act*2)
	stride := w(rom, d+1)
	mapFile := 0x14000 + w(rom, d+15)
	mapLen := w(rom, d+17)
	zone := 0

	mp := decomp.LoadMapRLE(rom, mapFile, mapLen) // row-major, stride columns
	rows := len(mp) / stride

	// block -> collision shape (0-47); shape's profile has a surface iff some column != -128
	shapeOf := func(block int) int { return int(rom[w(rom, attrPtrs+zone*2)+block]) & 0x3F }
	surfaceHeight := func(shape, col int) int { // signed height, -128 = no surface this column
		p := shapeTbl + shape*2
		off := w(rom, p)
		return int(int8(rom[off+(col&0x1F)]))
	}
	solid := func(block int) bool {
		s := shapeOf(block)
		for c := 0; c < 32; c++ {
			if surfaceHeight(s, c) != -128 {
				return true
			}
		}
		return false
	}
	blockAt := func(bx, by int) int {
		if bx < 0 || by < 0 || bx >= stride || by >= rows {
			return 0
		}
		return int(mp[by*stride+bx])
	}

	// object table
	ot := descTable + w(rom, d+30)
	n := int(rom[ot])
	fmt.Printf("GH act1: %dx%d blocks\n", stride, rows)
	for k, p := 0, ot+1; k < n; k, p = k+1, p+3 {
		typ, bx, by := int(rom[p]), int(rom[p+1]), int(rom[p+2])
		if typ != 0x08 && typ != 0x10 && typ != 0x2D {
			continue
		}
		name := map[int]string{0x08: "crab", 0x10: "beetle", 0x2D: "porcupine"}[typ]
		// first solid row at/below the placement row
		g := -1
		for r := by; r <= by+4 && r < rows; r++ {
			if solid(blockAt(bx, r)) {
				g = r
				break
			}
		}
		// surface pixel Y in the ground block at the enemy's centre column
		surf := "-"
		if g >= 0 {
			h := surfaceHeight(shapeOf(blockAt(bx, g)), (bx*32+8)&0x1F)
			surf = fmt.Sprintf("%d", g*32+h)
		}
		fmt.Printf("%-9s block(%3d,%3d) spawnGridTopY=%d | placeRow solid=%v | groundRow=%d surfaceY=%s\n",
			name, bx, by, by*32, solid(blockAt(bx, by)), g, surf)
	}
}
