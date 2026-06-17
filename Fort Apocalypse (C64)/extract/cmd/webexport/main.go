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

type jsonAnim struct {
	Char   int   `json:"char"`
	Period int   `json:"period"`
	Frames []int `json:"frames"` // atlas tile index per step
}

// The viewer places objects itself (re-randomised each time the layer is shown):
// it picks 8 of the prisoner candidates, scatters spmCount mines over empty cells,
// and draws each object from its real characters. So we export the candidate /
// home cells, not finished placements.
type jsonLevel struct {
	Level     int        `json:"level"`
	Width     int        `json:"width"`
	Height    int        `json:"height"`
	Atlas     string     `json:"atlas"`
	Cells     []int      `json:"cells"`
	Spawn     [2]int     `json:"spawn"`
	Anim      []jsonAnim `json:"anim"`
	Prisoners [][2]int   `json:"prisoners"` // $90A4 candidate cells (col,row); pick 8
	Tanks     [][2]int   `json:"tanks"`     // 6 fixed home cells (leftmost body cell)
	SPMCount  int        `json:"spmCount"`  // self-propelled mines to scatter (difficulty)
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
		jl := jsonLevel{Level: level, Atlas: atlasName}

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

		// Cells: the 215 content columns (Part IV §4). The playfield is a
		// cylinder — column 214 joins back to column 0 — so the stored wrap-seam
		// column (a duplicate of column 0) is dropped; the viewer wraps instead.
		w := fortgfx.ContentWidth
		jl.Width, jl.Height = w, fortgfx.MapHeight
		jl.Cells = make([]int, w*fortgfx.MapHeight)
		for r := 0; r < fortgfx.MapHeight; r++ {
			for c := 0; c < fortgfx.ContentWidth; c++ {
				jl.Cells[r*w+c] = int(lm.Cells[r][c])
			}
		}

		// Object placement data in content-column coordinates. The viewer turns
		// these into the real objects (prisoners pick 8 of the candidates, mines
		// scatter over empty cells).
		jl.Spawn = [2]int{lm.PlayerSpawn.Col, lm.PlayerSpawn.Row}
		jl.Prisoners, jl.Tanks = [][2]int{}, [][2]int{}
		for _, p := range lm.PrisonerSpawns {
			jl.Prisoners = append(jl.Prisoners, [2]int{p.Col, p.Row})
		}
		for _, p := range lm.TankHomes {
			jl.Tanks = append(jl.Tanks, [2]int{p.Col, p.Row})
		}
		jl.SPMCount = 13 // base difficulty (13 / 26 / 39 by variant)

		file := fmt.Sprintf("level%d.json", level)
		if err := writeJSON(filepath.Join(outDir, file), jl); err != nil {
			return err
		}
		meta.Levels = append(meta.Levels, metaLevel{Name: names[level], File: file, Atlas: atlasName})
		fmt.Printf("level %d: %dx%d cells, %d atlas tiles; %d prisoner candidates, %d tanks, %d SPMs -> %s\n",
			level, w, fortgfx.MapHeight, len(tiles), len(jl.Prisoners), len(jl.Tanks), jl.SPMCount, file)
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
