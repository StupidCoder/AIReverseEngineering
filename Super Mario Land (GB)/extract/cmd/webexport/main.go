// webexport decodes every Super Mario Land level map from the ROM and writes the
// assets the Studio viewer loads: a per-world 256-tile atlas PNG, a per-level
// tile-index JSON (row-major cells), and a meta.json index. The maps are decoded from
// the cartridge (extract/level); the tile graphics come from a short oracle run nudged
// into each world (the oracle supplies only the pixels, not the map).
//
//	go run ./cmd/webexport [-rom PATH] [-o DIR]
package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"

	"stupidcoder.com/tools/gameboy"
	"supermarioland/extract/level"
)

const (
	tile  = 8
	gut   = 1            // 1px extruded gutter so tiles don't bleed at fractional zoom
	cell  = tile + 2*gut // 10
	cols  = 16           // atlas is 16 tiles wide (256 tiles total)
	nTile = 256
)

type levelMeta struct {
	World  int    `json:"world"`
	Level  int    `json:"level"`
	Name   string `json:"name"`
	File   string `json:"file"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func main() {
	rom := stringFlag("-rom", "../Super Mario Land (World).gb")
	out := stringFlag("-o", "../../site/public/sml")

	data, err := os.ReadFile(rom)
	must(err)
	must(os.MkdirAll(out, 0o755))

	var metas []levelMeta
	for world := 1; world <= 4; world++ {
		// Tiles: warp the oracle into this world and read back VRAM + palettes.
		vram, lcdc, bgp := worldTiles(data, byte(world))
		obp0 := objPalette(data, byte(world))
		saveAtlas(filepath.Join(out, fmt.Sprintf("world%d.png", world)), vram, lcdc, bgp)
		// Object sprite icons: one composited metasprite per known type, in this world's
		// OBJ tiles, packed into an atlas the viewer blits at each placement.
		objAtlas := fmt.Sprintf("world%d-obj.png", world)
		objTypes, marioIcon := saveObjIcons(data, filepath.Join(out, objAtlas), vram, obp0)

		for lv := 1; lv <= 3; lv++ {
			id := byte(world<<4 | lv)
			cols2, _, _ := level.DecodeLevelByID(data, id)
			w := len(cols2)
			cells := make([]int, w*16) // row-major: cells[r*w + x]
			for x, col := range cols2 {
				for r := 0; r < 16; r++ {
					cells[r*w+x] = int(col[r])
				}
			}
			// Object/enemy placements (decoded from the ROM list at $401A[ffe4]).
			objs := level.DecodeObjectsByID(data, id)
			ojson := make([]map[string]any, len(objs))
			for i, o := range objs {
				ojson[i] = map[string]any{
					"col": o.Col, "row": o.Row, "type": o.Type, "hard": o.Hard, "fineX": o.FineX,
				}
			}
			name := fmt.Sprintf("%d-%d", world, lv)
			file := "level-" + name + ".json"
			writeJSON(filepath.Join(out, file), map[string]any{
				"world": world, "level": lv,
				"width": w, "height": 16,
				"atlas":     fmt.Sprintf("world%d.png", world),
				"cells":     cells,
				"objects":   ojson,
				"objAtlas":  objAtlas,
				"objCell":   objCell,
				"objOrigin": objOrigin, // sprite-origin offset within a cell (x,y)
				"objTypes":  objTypes,  // type id -> icon index in the atlas (row of cells)
				// Mario's fixed start (his sprite top-left, in map pixels) + its atlas icon.
				// The engine spawns him at screen (50,134); minus the 16px HUD that is map px.
				"player": map[string]any{"x": 35, "y": 96, "icon": marioIcon},
			})
			metas = append(metas, levelMeta{world, lv, name, file, w, 16})
		}
	}
	writeJSON(filepath.Join(out, "meta.json"), map[string]any{"levels": metas})
	fmt.Printf("wrote %d levels + 4 world atlases to %s\n", len(metas), out)
}

// Object-icon atlas geometry: each known type's metasprite is composited into one
// objCell-square cell (stacked vertically). objOrigin is the cell pixel where the
// metasprite's cursor origin (0,0) sits, so the viewer can line an icon up with a
// placement: it blits the cell so objOrigin lands on the object's world anchor. The cell
// must hold the largest metasprite: across all base frames the tiles span DX -8..+32,
// DY -24..+16, so a 40x40 cell with the origin at (8,24) fits every sprite without clipping.
const objCell = 40

var objOrigin = [2]int{8, 24}

// marioFrame is Mario's idle standing sprite: a 2x2 of OBJ tiles, each {tile, dx, dy}
// from the sprite top-left (read from the running game's OAM at spawn).
var marioFrame = []level.Sprite{{0x00, 0, 0}, {0x01, 8, 0}, {0x10, 0, 8}, {0x11, 8, 8}}

// saveObjIcons composites one metasprite per known object type into a vertical atlas (in
// this world's OBJ tiles + sprite palette), plus one extra cell for Mario's idle sprite at
// the end. It returns the type id -> icon index map and Mario's icon index. Types not in
// level.TypeFrame are omitted (the viewer falls back to a marker for them).
func saveObjIcons(rom []byte, path string, vram []byte, obp0 byte) (map[string]int, int) {
	// Stable order so the atlas is deterministic.
	typeFrame := level.TypeFrames(rom)
	var types []int
	for t := range typeFrame {
		types = append(types, int(t))
	}
	sort.Ints(types)

	marioIdx := len(types) // Mario gets the last cell
	img := image.NewNRGBA(image.Rect(0, 0, objCell, objCell*(len(types)+1)))
	stamp := func(s level.Sprite, ox, oy int) {
		tl := gameboy.DecodeTile(vram[int(s.Tile)*16:]) // 8x8 OBJ tile
		for py := 0; py < 8; py++ {
			for px := 0; px < 8; px++ {
				if v := tl[py][px]; v != 0 { // OBJ colour 0 = transparent
					g := []uint8{0xff, 0xaa, 0x55, 0x00}[(obp0>>(2*v))&3]
					img.Set(ox+s.DX+px, oy+s.DY+py, color.NRGBA{g, g, g, 0xff})
				}
			}
		}
	}
	out := map[string]int{}
	for i, t := range types {
		out[fmt.Sprintf("%d", t)] = i
		for _, s := range level.DecodeMetasprite(rom, int(typeFrame[byte(t)])) {
			stamp(s, objOrigin[0], i*objCell+objOrigin[1])
		}
	}
	for _, s := range marioFrame { // Mario's 2x2, top-left at the cell origin
		stamp(s, objOrigin[0], marioIdx*objCell+objOrigin[1])
	}
	f, err := os.Create(path)
	must(err)
	defer f.Close()
	must(png.Encode(f, img))
	return out, marioIdx
}

// objPalette warps the oracle into `world` and returns its OBP0 sprite palette register.
func objPalette(rom []byte, world byte) byte {
	m := gameboy.NewMachine(rom)
	m.RunFrames(80)
	for f := 0; f < 6; f++ {
		m.Buttons = gameboy.BtnStart
		m.RunFrame()
	}
	m.Buttons = 0
	id := byte(world<<4 | 1)
	for f := 0; f < 40; f++ {
		m.Write(0xFFB4, id)
		m.RunFrame()
	}
	return m.Read(0xFF48)
}

// worldTiles boots the ROM, nudges it into `world` by forcing the level id through the
// load window, and returns the VRAM, LCDC and BG palette register.
func worldTiles(rom []byte, world byte) (vram []byte, lcdc, bgp byte) {
	m := gameboy.NewMachine(rom)
	m.RunFrames(80)
	for f := 0; f < 6; f++ {
		m.Buttons = gameboy.BtnStart
		m.RunFrame()
	}
	m.Buttons = 0
	id := byte(world<<4 | 1) // world's first level
	for f := 0; f < 40; f++ {
		m.Write(0xFFB4, id)
		m.RunFrame()
	}
	return m.VRAM(), m.Read(0xFF40), m.Read(0xFF47)
}

// saveAtlas writes the 256 background tiles (indexed by the map's tile value, signed
// $8800 addressing) into a 16-wide atlas with a 1px extruded gutter, in the BG palette.
func saveAtlas(path string, vram []byte, lcdc, bgp byte) {
	rows := nTile / cols
	img := image.NewPaletted(image.Rect(0, 0, cols*cell, rows*cell), gameboy.GreyPalette())
	shade := func(v uint8) uint8 { return (bgp >> (2 * v)) & 3 }
	for n := 0; n < nTile; n++ {
		t := gameboy.DecodeTile(vram[tileOffset(lcdc, byte(n)):])
		ox, oy := (n%cols)*cell+gut, (n/cols)*cell+gut
		for y := -gut; y < tile+gut; y++ {
			for x := -gut; x < tile+gut; x++ {
				sx, sy := clamp(x), clamp(y) // extrude edges into the gutter
				img.SetColorIndex(ox+x, oy+y, shade(t[sy][sx]))
			}
		}
	}
	f, err := os.Create(path)
	must(err)
	defer f.Close()
	must(png.Encode(f, img))
}

func clamp(v int) int {
	if v < 0 {
		return 0
	}
	if v >= tile {
		return tile - 1
	}
	return v
}

// tileOffset mirrors the BG tile addressing (LCDC bit 4: $8000 unsigned vs signed $8800).
func tileOffset(lcdc, idx byte) int {
	if lcdc&0x10 != 0 {
		return int(idx) * 16
	}
	return 0x1000 + int(int8(idx))*16
}

func writeJSON(path string, v any) {
	b, err := json.Marshal(v)
	must(err)
	must(os.WriteFile(path, b, 0o644))
}

func stringFlag(name, def string) string {
	for i, a := range os.Args {
		if a == name && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	return def
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "webexport:", err)
		os.Exit(1)
	}
}
