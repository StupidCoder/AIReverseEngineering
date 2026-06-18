// Command placements extracts Turrican's enemy placement lists from the disk —
// where each enemy is seeded in every scene of every world (Part V).
//
//	placements [-o dir] [Turrican.adf]
//
// The scroll-triggered spawner (resident `$1710`) reads, per scene, a list of
// 6-byte placement entries — `type.w, x.w, y.w` (x/y in 8-pixel units, so a tile
// is 4 units) — sorted by x, and spawns each as the camera window approaches it.
// A `$00` type ends a column's run, `$D3` ends the list. The list lives in the
// decoded scene block, indexed by a grid at the scene descriptor's `+$28`; the
// entries themselves are a contiguous sorted stream the grid points into. The
// type's low nibble selects the scene's enemy-AI handler (descriptor `+$20`
// table), which is what wires a placement to one of the extracted sprites.
//
// Output: one JSON per scene with the in-bounds placements, for a viewer object
// layer.
package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"turrican/extract/decrunch"
)

const (
	blockBase  = 0x1B980
	levelTable = 0x46A
	numWorlds  = 5
	unitsTile  = 4 // 8-pixel placement units per 32-pixel tile
)

type object struct {
	Type   int `json:"type"`   // entry type byte; low nibble = enemy-AI handler index
	X      int `json:"x"`      // tile column (x / 4)
	Y      int `json:"y"`      // tile row (y / 4)
	Sprite int `json:"sprite"` // resolved frame-table address (the enemy's sprite); 0 if unknown
}
type aiEntry struct {
	Handler int `json:"handler"` // +$20 table entry (the AI routine)
	Sprite  int `json:"sprite"`  // frame table the handler installs ($12), == a sprite sheet
}
type sceneObjects struct {
	World, Scene, Width, Height int
	AI                          []aiEntry `json:"ai"` // index by type&$F -> {handler, sprite}
	Objects                     []object
}

// frameTableOf scans an AI handler's code for `MOVE.l #imm,$12(a5)`
// (opcode 2B 7C imm32 00 12) — the frame table (sprite) it installs.
func frameTableOf(block []byte, handler, base, hi int) int {
	o := handler - base
	for i := o; i < o+260 && i+8 <= len(block); i++ {
		if block[i] == 0x2B && block[i+1] == 0x7C && block[i+6] == 0x00 && block[i+7] == 0x12 {
			ft := int(binary.BigEndian.Uint32(block[i+2:]))
			if ft >= base && ft < hi {
				return ft
			}
		}
	}
	return 0
}

func main() {
	out := flag.String("o", "rendered/placements", "output directory")
	flag.Parse()
	adfPath := flag.Arg(0)
	if adfPath == "" {
		adfPath = "Turrican.adf"
	}
	adf, err := os.ReadFile(adfPath)
	if err != nil {
		fail(err)
	}
	res, err := decrunch.Decrunch(mainBlob(adf))
	if err != nil {
		fail(err)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fail(err)
	}

	for w := 0; w < numWorlds; w++ {
		t := levelTable + w*8
		off := int(binary.BigEndian.Uint32(res.Data[t:]))
		length := int(binary.BigEndian.Uint32(res.Data[t+4:]))
		block, err := decrunch.DecrunchBlock(adf[off : off+length])
		if err != nil {
			fail(fmt.Errorf("world %d: %w", w, err))
		}
		be16 := func(a int) int { return int(binary.BigEndian.Uint16(block[a-blockBase:])) }
		be32 := func(a int) int { return int(binary.BigEndian.Uint32(block[a-blockBase:])) }
		blockHi := blockBase + len(block)

		nScenes := be16(blockBase + 0x14)
		for s := 0; s < nScenes; s++ {
			desc := be32(blockBase + 0x16 + s*4)
			width, height := be16(desc+0x04), be16(desc+0x06)
			grid := be32(desc + 0x28)

			// The grid is an array of pointers into the sorted entry stream that
			// begins immediately after it, so a grid entry always lives below the
			// lowest address it points to: scan while the cursor stays below the
			// minimum pointer seen. This pins the stream extent on narrow/tall scenes
			// (a fixed window runs off the small grid into unrelated data).
			lo, hi := 0, 0
			for a := grid; a < blockHi-4; a += 4 {
				if lo != 0 && a >= lo {
					break
				}
				p := be32(a)
				if p < blockBase || p >= blockHi {
					break
				}
				if lo == 0 || p < lo {
					lo = p
				}
				if p > hi {
					hi = p
				}
			}
			if lo == 0 {
				continue
			}

			// The scene's enemy-AI handler table (type & $F indexes it) and the
			// frame table (sprite) each handler installs.
			aiTbl := be32(desc + 0x20)
			var ai []aiEntry
			for i := 0; i < 16; i++ {
				h := be32(aiTbl + i*4)
				if h < blockBase || h >= blockHi {
					break
				}
				ai = append(ai, aiEntry{Handler: h, Sprite: frameTableOf(block, h, blockBase, blockHi)})
			}
			spriteOf := func(ty int) int {
				if n := ty & 0xF; n < len(ai) {
					return ai[n].Sprite
				}
				return 0
			}

			var objs []object
			for a := lo; a < hi+60 && a < blockHi-6; {
				ty := be16(a)
				if ty == 0 {
					a += 2
					continue
				}
				if ty&0xFF == 0xD3 {
					a += 6
					continue
				}
				x, y := be16(a+2), be16(a+4)
				if x < width*unitsTile && y < height*unitsTile {
					t := ty & 0xFF
					objs = append(objs, object{Type: t, X: x / unitsTile, Y: y / unitsTile, Sprite: spriteOf(t)})
				}
				a += 6
			}

			so := sceneObjects{World: w, Scene: s, Width: width, Height: height, AI: ai, Objects: objs}
			name := fmt.Sprintf("world%d_scene%d.json", w, s)
			b, _ := json.Marshal(so)
			if err := os.WriteFile(filepath.Join(*out, name), b, 0o644); err != nil {
				fail(err)
			}
			fmt.Printf("world %d scene %d: %d objects -> %s\n", w, s, len(objs), name)
		}
	}
}

func mainBlob(adf []byte) []byte {
	const off = 0x2C00
	return adf[off : off+int(binary.BigEndian.Uint32(adf[off:]))]
}
func fail(err error) {
	fmt.Fprintln(os.Stderr, "placements:", err)
	os.Exit(1)
}
