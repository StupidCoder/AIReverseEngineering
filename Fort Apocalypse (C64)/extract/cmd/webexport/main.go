// webexport renders the Fort Apocalypse level maps for the companion website's
// tile viewer. For each of the two levels it bakes a character atlas (the 128
// playfield chars in the level's colours, plus the extra frames the animated
// soft chars cycle through) and writes a JSON file with the 216x40 cell grid,
// the object placements, and the animation spec. A meta.json lists the levels.
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

// object marker types — colours/labels are applied by the viewer.
type jsonObj struct {
	Col  int    `json:"col"`
	Row  int    `json:"row"`
	W    int    `json:"w"`
	H    int    `json:"h"`
	Type string `json:"type"`
}

type jsonAnim struct {
	Char   int   `json:"char"`
	Period int   `json:"period"`
	Frames []int `json:"frames"` // atlas tile index per step
}

type jsonLevel struct {
	Level   int        `json:"level"`
	Width   int        `json:"width"`
	Height  int        `json:"height"`
	Atlas   string     `json:"atlas"`
	Cells   []int      `json:"cells"`
	Spawn   [2]int     `json:"spawn"`
	Objects []jsonObj  `json:"objects"`
	Drops   [][2]int   `json:"drops"`
	Anim    []jsonAnim `json:"anim"`
}

type jsonMeta struct {
	Levels []metaLevel `json:"levels"`
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

	meta := jsonMeta{}
	names := []string{"Level 1", "Level 2"}
	for level := 0; level <= 1; level++ {
		lm, err := game.LevelMap(level)
		if err != nil {
			return err
		}
		atlasName := fmt.Sprintf("atlas-L%d.png", level)
		jl := jsonLevel{Level: level, Atlas: atlasName, Drops: [][2]int{}}

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
			ja := jsonAnim{Char: int(a.Char), Period: a.Period}
			for _, fr := range a.Frames {
				ja.Frames = append(ja.Frames, addTile(fr))
			}
			jl.Anim = append(jl.Anim, ja)
		}

		pal := palette(game.MulticolorValue(level))
		if err := writeAtlas(filepath.Join(outDir, atlasName), tiles, pal); err != nil {
			return err
		}

		// Cells in render order: the wrap-seam column (stored at 255), then the
		// 215 content columns (Part IV §4).
		w := fortgfx.ContentWidth + 1
		jl.Width, jl.Height = w, fortgfx.MapHeight
		jl.Cells = make([]int, w*fortgfx.MapHeight)
		for r := 0; r < fortgfx.MapHeight; r++ {
			jl.Cells[r*w] = int(lm.Cells[r][fortgfx.MapWidth-1])
			for c := 0; c < fortgfx.ContentWidth; c++ {
				jl.Cells[r*w+1+c] = int(lm.Cells[r][c])
			}
		}

		// Objects (render coords: content column + 1 for the seam), mirroring the
		// gfxrender markers — footprints in characters.
		jl.Spawn = [2]int{lm.PlayerSpawn.Col + 1, lm.PlayerSpawn.Row}
		jl.Objects = append(jl.Objects, jsonObj{lm.PlayerSpawn.Col + 1, lm.PlayerSpawn.Row, 4, 3, "player"})
		for _, p := range lm.PrisonerSpawns {
			jl.Objects = append(jl.Objects, jsonObj{p.Col + 1, p.Row - 1, 2, 2, "prisoner"})
		}
		for _, p := range lm.TankHomes {
			jl.Objects = append(jl.Objects, jsonObj{p.Col + 1, p.Row - 1, 3, 2, "tank"})
		}
		for _, p := range lm.EnemySpawns {
			jl.Objects = append(jl.Objects, jsonObj{p.Col + 1, p.Row, 4, 3, "enemy"})
		}
		for _, p := range lm.DropPoints {
			jl.Drops = append(jl.Drops, [2]int{p.Col + 1, p.Row})
		}

		file := fmt.Sprintf("level%d.json", level)
		if err := writeJSON(filepath.Join(outDir, file), jl); err != nil {
			return err
		}
		meta.Levels = append(meta.Levels, metaLevel{Name: names[level], File: file, Atlas: atlasName})
		fmt.Printf("level %d: %dx%d cells, %d atlas tiles, %d objects, %d drops -> %s\n",
			level, w, fortgfx.MapHeight, len(tiles), len(jl.Objects), len(jl.Drops), file)
	}
	return writeJSON(filepath.Join(outDir, "meta.json"), meta)
}

// palette is the playfield's multicolor map: 00=black, 01=$D022 (per level),
// 10=white, 11=colour-RAM green (Part IV §2).
func palette(d022 byte) [4]color.RGBA {
	return [4]color.RGBA{gfx.Palette[0], gfx.Palette[d022&0x0F], gfx.Palette[1], gfx.Palette[5]}
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
