// webexport renders the Fort Apocalypse level maps in the Studio's common level
// format (site/FORMAT.md). For each of the two levels it bakes a character atlas
// (the 128 playfield chars in the level's colours, plus the extra frames the
// animated soft chars cycle through) and writes a JSON file with the cell grid,
// the soft-char tile animations, and the randomized object POOLS (the game seeds
// prisoners/tanks/mines at load; the viewer re-rolls them from the exported
// candidate lists). A meta.json lists the levels; the playfield is a horizontal
// cylinder (meta wrap "x").
//
// Usage: webexport [-prg FORT-fast-7000.prg] [-o dir]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"

	"fortapoc/extract/fortgfx"
	"stupidcoder.com/tools/c64/gfx"
)

type jsonAnim struct {
	Tiles        []int   `json:"tiles"`
	Frames       [][]int `json:"frames"` // atlas tile per step (one slot: the char itself)
	PeriodFrames int     `json:"periodFrames"`
}

type jsonStamp struct {
	DX   int `json:"dx"`
	DY   int `json:"dy"`
	Tile int `json:"tile"`
}

type jsonVariant struct {
	Stamps []jsonStamp `json:"stamps,omitempty"`
	Sprite string      `json:"sprite,omitempty"`
	Tint   string      `json:"tint,omitempty"`
}

// objectPools: the game seeds prisoners/tanks/mines at level load — the viewer
// re-rolls `count` picks from the exported candidate list each time the objects
// layer is shown, choosing a random variant per placement (leg/turret facing).
type jsonPool struct {
	Count      int           `json:"count"`
	Candidates [][2]int      `json:"candidates"` // world px
	Variants   []jsonVariant `json:"variants"`
}

type jsonGrid struct {
	TileSize    int    `json:"tileSize"`
	Atlas       string `json:"atlas"`
	AtlasCols   int    `json:"atlasCols"`
	AtlasGutter int    `json:"atlasGutter"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Cells       []int  `json:"cells"`
}

type jsonRect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type jsonSpawn struct {
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Sprite string `json:"sprite"`
	Tint   string `json:"tint"`
}

type jsonLevel struct {
	Format      int        `json:"format"`
	Name        string     `json:"name"`
	Grid        jsonGrid   `json:"grid"`
	View        jsonRect   `json:"view"`
	Spawn       jsonSpawn  `json:"spawn"`
	TileAnims   []jsonAnim `json:"tileAnims"`
	ObjectPools []jsonPool `json:"objectPools"`
}

type metaLevel struct {
	Name  string `json:"name"`
	File  string `json:"file"`
	Atlas string `json:"atlas"`
}

func main() {
	prg := flag.String("prg", "../extracted/FORT-fast-7000.prg", "game file ($7000)")
	outDir := flag.String("o", "../../site/public/fort", "output directory")
	flag.Parse()
	if err := run(*prg, *outDir); err != nil {
		fmt.Fprintln(os.Stderr, "webexport:", err)
		os.Exit(1)
	}
}

func run(prgPath, outDir string) error {
	game, err := fortgfx.LoadGame(prgPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	cs := game.PlayfieldCharset()
	anim := game.SoftCharAnim()

	// Helicopter sprites (shared by both levels): the level-flight pose for the
	// player and a banked pose for the enemy, white on transparent so the viewer
	// can tint them (yellow / blue). The poses come in tilt order full-left ..
	// level .. full-right; [0] is the sprite block of each pose's first rotor frame.
	poses := game.HelicopterPoses()
	shapes := game.SpriteShapes()
	if err := os.MkdirAll(filepath.Join(outDir, "sprites"), 0o755); err != nil {
		return err
	}
	if err := gfx.WritePNG(filepath.Join(outDir, "sprites", "chopper-fwd.png"),
		chopperImg(shapes[poses[len(poses)/2][0]-1])); err != nil {
		return err
	}
	if err := gfx.WritePNG(filepath.Join(outDir, "sprites", "chopper-side.png"),
		chopperImg(shapes[poses[0][0]-1])); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outDir, "sprites", "index.json"), map[string]any{
		"chopper-fwd":  map[string]any{"src": "sprites/chopper-fwd.png", "frames": [][4]int{{0, 0, 32, 18}}},
		"chopper-side": map[string]any{"src": "sprites/chopper-side.png", "frames": [][4]int{{0, 0, 32, 18}}},
	}); err != nil {
		return err
	}

	var metas []metaLevel
	names := []string{"Vaults of Draconis", "Crystalline Caves"}
	for level := 0; level <= 1; level++ {
		lm, err := game.LevelMap(level)
		if err != nil {
			return err
		}
		atlasName := fmt.Sprintf("atlas-L%d.png", level)
		jl := jsonLevel{Format: 1, Name: names[level]}
		jl.Grid = jsonGrid{TileSize: 8, Atlas: atlasName, AtlasCols: 16}

		// Atlas tiles: the 128 base chars at fixed indices 0..127, then any extra
		// frame bitmaps the animations need, appended and de-duplicated.
		tiles := make([][8]byte, 128)
		for ch := 0; ch < 128; ch++ {
			copy(tiles[ch][:], cs[ch*8:])
		}
		idxOf := map[[8]byte]int{}
		for i := 127; i >= 0; i-- {
			idxOf[tiles[i]] = i
		}
		addTile := func(b [8]byte) int {
			if i, ok := idxOf[b]; ok {
				return i
			}
			i := len(tiles)
			tiles = append(tiles, b)
			idxOf[b] = i
			return i
		}
		for _, a := range anim {
			ja := jsonAnim{Tiles: []int{int(a.Char)}, PeriodFrames: a.Period}
			for _, fr := range a.Frames {
				ja.Frames = append(ja.Frames, []int{addTile(fr)})
			}
			jl.TileAnims = append(jl.TileAnims, ja)
		}

		pal := palette(game.MulticolorValue(level))
		if err := writeAtlas(filepath.Join(outDir, atlasName), tiles, pal); err != nil {
			return err
		}

		// Cells: the 215 content columns (Part IV §4). The playfield is a
		// cylinder — column 214 joins back to column 0 — so the stored wrap-seam
		// column (a duplicate of column 0) is dropped; the viewer wraps instead.
		w := fortgfx.ContentWidth
		jl.Grid.Width, jl.Grid.Height = w, fortgfx.MapHeight
		jl.Grid.Cells = make([]int, w*fortgfx.MapHeight)
		for r := 0; r < fortgfx.MapHeight; r++ {
			for c := 0; c < fortgfx.ContentWidth; c++ {
				jl.Grid.Cells[r*w+c] = int(lm.Cells[r][c])
			}
		}
		cellAt := func(c, r int) int { return jl.Grid.Cells[r*w+c] }

		// Player spawn (the yellow helicopter, tinted white sprite) + initial view:
		// one C64 screen (320x200) with the spawn centred horizontally, top row up.
		sx, sy := lm.PlayerSpawn.Col*8, lm.PlayerSpawn.Row*8
		jl.Spawn = jsonSpawn{X: sx, Y: sy, Sprite: "chopper-fwd", Tint: "#b8c76f"}
		jl.View = jsonRect{X: sx + 16 - 160, Y: 0, W: 320, H: 200}

		// Object pools — the game's load-time seeding rules, as candidates the
		// viewer re-rolls (Part V; previously hardcoded in the viewer):
		//  - prisoners: 8 of the $90A4 floor-with-rock candidates; torso $49 over
		//    legs $3B (facing right) or $3D (left), random per prisoner.
		//  - tanks: every fixed home; body $6C $6D $6E, turret $6F/$70 random.
		//  - SPMs: spmCount mines ($5B $5C) at empty 2-cell spots in the column
		//    band $2D..$C8 (the game re-rolls until both cells are empty).
		//  - one enemy helicopter at a random 4x2-clear spot in the same band.
		var prisoners [][2]int
		for _, p := range lm.PrisonerSpawns {
			prisoners = append(prisoners, [2]int{p.Col * 8, p.Row * 8})
		}
		jl.ObjectPools = append(jl.ObjectPools, jsonPool{
			Count: 8, Candidates: prisoners,
			Variants: []jsonVariant{
				{Stamps: []jsonStamp{{0, -8, 0x49}, {0, 0, 0x3B}}},
				{Stamps: []jsonStamp{{0, -8, 0x49}, {0, 0, 0x3D}}},
			},
		})
		var tanks [][2]int
		for _, p := range lm.TankHomes {
			tanks = append(tanks, [2]int{p.Col * 8, p.Row * 8})
		}
		jl.ObjectPools = append(jl.ObjectPools, jsonPool{
			Count: len(tanks), Candidates: tanks,
			Variants: []jsonVariant{
				{Stamps: []jsonStamp{{0, 0, 0x6C}, {8, 0, 0x6D}, {16, 0, 0x6E}, {8, -8, 0x6F}}},
				{Stamps: []jsonStamp{{0, 0, 0x6C}, {8, 0, 0x6D}, {16, 0, 0x6E}, {8, -8, 0x70}}},
			},
		})
		const bandMin, bandMax = 0x32 - 5, 0xCD - 5
		var spmSpots [][2]int
		for r := 0; r < fortgfx.MapHeight; r++ {
			for c := bandMin; c <= bandMax && c+1 < w; c++ {
				if cellAt(c, r) == 0 && cellAt(c+1, r) == 0 {
					spmSpots = append(spmSpots, [2]int{c * 8, r * 8})
				}
			}
		}
		jl.ObjectPools = append(jl.ObjectPools, jsonPool{
			Count: 13, Candidates: spmSpots, // base difficulty (13 / 26 / 39 by variant)
			Variants: []jsonVariant{
				{Stamps: []jsonStamp{{0, 0, 0x5B}, {8, 0, 0x5C}}},
			},
		})
		var heliSpots [][2]int
		for r := 0; r+1 < fortgfx.MapHeight; r++ {
			for c := bandMin; c <= bandMax && c+3 < w; c++ {
				clear := true
				for dy := 0; dy < 2 && clear; dy++ {
					for dx := 0; dx < 4; dx++ {
						if cellAt(c+dx, r+dy) != 0 {
							clear = false
							break
						}
					}
				}
				if clear {
					heliSpots = append(heliSpots, [2]int{c * 8, r * 8})
				}
			}
		}
		jl.ObjectPools = append(jl.ObjectPools, jsonPool{
			Count: 1, Candidates: heliSpots,
			Variants: []jsonVariant{
				{Sprite: "chopper-side", Tint: "#352879"},
			},
		})

		file := fmt.Sprintf("level%d.json", level)
		if err := writeJSON(filepath.Join(outDir, file), jl); err != nil {
			return err
		}
		metas = append(metas, metaLevel{Name: names[level], File: file, Atlas: atlasName})
		fmt.Printf("level %d: %dx%d cells, %d atlas tiles; %d prisoner cands, %d tanks, %d SPM spots, %d heli spots -> %s\n",
			level, w, fortgfx.MapHeight, len(tiles), len(prisoners), len(tanks), len(spmSpots), len(heliSpots), file)
	}
	return writeJSON(filepath.Join(outDir, "meta.json"), map[string]any{
		"format": 1, "game": "fort",
		"native": map[string]int{"w": 320, "h": 200},
		"tickHz": 50,
		"wrap":   "x",
		"levels": metas,
	})
}

// palette is the playfield's multicolor map: 00=black, 01=$D022 (per level),
// 10=white, 11=colour-RAM green (Part IV §2).
func palette(d022 byte) [4]color.RGBA {
	return [4]color.RGBA{gfx.Palette[0], gfx.Palette[d022&0x0F], gfx.Palette[1], gfx.Palette[5]}
}

// chopperImg renders one helicopter sprite block (the [left][right][$00] rows
// the game expands, 16x18 used pixels) into a 32x18 RGBA image — white where set,
// transparent elsewhere, X-expanded (the sprites are drawn double-wide in game).
func chopperImg(block []byte) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 32, 18))
	white := color.RGBA{255, 255, 255, 255}
	for r := 0; r < 18; r++ {
		for bi := 0; bi < 2; bi++ {
			b := block[r*3+bi]
			for bit := 0; bit < 8; bit++ {
				if b&(0x80>>bit) != 0 {
					x := (bi*8 + bit) * 2
					img.SetRGBA(x, r, white)
					img.SetRGBA(x+1, r, white)
				}
			}
		}
	}
	return img
}

// writeAtlas renders the tiles as a 16-wide grid of 8x8 multicolor chars.
func writeAtlas(path string, tiles [][8]byte, pal [4]color.RGBA) error {
	const cols = 16
	rows := (len(tiles) + cols - 1) / cols
	img := image.NewRGBA(image.Rect(0, 0, cols*8, rows*8))
	for i, t := range tiles {
		gfx.DrawChar(img, t[:], (i%cols)*8, (i/cols)*8, 1, pal)
	}
	return gfx.WritePNG(path, img)
}

func writeJSON(path string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
