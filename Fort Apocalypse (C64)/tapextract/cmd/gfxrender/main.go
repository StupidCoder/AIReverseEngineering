// gfxrender renders Fort Apocalypse graphics (charsets, level maps)
// from the extracted game file into PNG images.
//
// Usage: gfxrender [-o outdir] [-scale n] [-markers] FORT-fast-7000.prg
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"tapextract/fortgfx"
)

func main() {
	outDir := flag.String("o", ".", "output directory")
	scale := flag.Int("scale", 2, "pixel scale factor")
	markers := flag.Bool("markers", false, "mark player (cyan) and enemy spawn candidates (yellow)")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: gfxrender [-o outdir] [-scale n] [-markers] FORT-fast-7000.prg")
		os.Exit(2)
	}
	if err := run(flag.Arg(0), *outDir, *scale, *markers); err != nil {
		fmt.Fprintln(os.Stderr, "gfxrender:", err)
		os.Exit(1)
	}
}

func run(path, outDir string, scale int, markers bool) error {
	game, err := fortgfx.LoadGame(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	playCS := game.PlayfieldCharset()
	hudCS := game.HUDCharset()

	p := filepath.Join(outDir, "charset-playfield.png")
	if err := fortgfx.WritePNG(p, fortgfx.RenderCharset(playCS, game.MulticolorValue(0), scale*2)); err != nil {
		return err
	}
	fmt.Println("wrote", p)
	p = filepath.Join(outDir, "charset-hud.png")
	if err := fortgfx.WritePNG(p, fortgfx.RenderCharset(hudCS, game.MulticolorValue(0), scale*2)); err != nil {
		return err
	}
	fmt.Println("wrote", p)

	// Sprites: the helicopter animation sheet (the $A320 tilt table
	// both helicopters use: 7 banking poses x 2 rotor frames — this
	// covers all 14 shapes in the file) and the two bullet blocks.
	// Sprites 0/1 are X-expanded in-game, so the sheets are rendered
	// double-wide too.
	shapes := game.SpriteShapes()
	poses := game.HelicopterPoses()
	grid := make([][][]byte, len(poses))
	for i, pair := range poses {
		grid[i] = [][]byte{shapes[pair[0]-1], shapes[pair[1]-1]}
	}
	p = filepath.Join(outDir, "sprite-anim-helicopter.png")
	if err := fortgfx.WritePNG(p, fortgfx.RenderSpriteGrid(grid, 7, scale*2, 2)); err != nil {
		return err
	}
	fmt.Printf("wrote %s (%d rows = banking poses full-left..full-right; each row: the pose's 2 rotor frames)\n", p, len(poses))

	p = filepath.Join(outDir, "sprites-bullets.png")
	if err := fortgfx.WritePNG(p, fortgfx.RenderSpriteSheet(game.BulletShapes(), 1, scale*2, 1)); err != nil {
		return err
	}
	fmt.Println("wrote", p)

	for level := 0; level <= 1; level++ {
		lm, err := game.LevelMap(level)
		if err != nil {
			return err
		}
		img := fortgfx.RenderMap(lm, playCS, game.MulticolorValue(level), scale, markers)
		p := filepath.Join(outDir, fmt.Sprintf("map-level%d.png", level))
		if err := fortgfx.WritePNG(p, img); err != nil {
			return err
		}
		fmt.Printf("wrote %s (player spawn %d,%d; %d enemy spawn candidates)\n",
			p, lm.PlayerSpawn.Col, lm.PlayerSpawn.Row, len(lm.EnemySpawns))
	}
	return nil
}
