package fortgfx

import (
	"os"
	"path/filepath"
	"testing"
)

func loadTestGame(t *testing.T) *Game {
	t.Helper()
	g, err := LoadGame("../../extracted/FORT-fast-7000.prg")
	if err != nil {
		t.Skip("extracted game file not available:", err)
	}
	return g
}

func TestLevelMapDecode(t *testing.T) {
	g := loadTestGame(t)
	lm, err := g.LevelMap(0)
	if err != nil {
		t.Fatal(err)
	}
	// First map row of level 0 decompresses from "1F D7 00 28 1F 01":
	// 215 rock, 40 sky, 1 rock.
	for c := 0; c < 215; c++ {
		if lm.Cells[0][c] != 0x1F {
			t.Fatalf("row 0 col %d: got $%02X want $1F", c, lm.Cells[0][c])
		}
	}
	for c := 215; c < 255; c++ {
		if lm.Cells[0][c] != 0x00 {
			t.Fatalf("row 0 col %d: got $%02X want $00", c, lm.Cells[0][c])
		}
	}
	if lm.Cells[0][255] != 0x1F {
		t.Fatalf("row 0 col 255: got $%02X want $1F", lm.Cells[0][255])
	}
	// Randomized rock placeholders must be gone.
	for r := 0; r < MapHeight; r++ {
		for c := 0; c < MapWidth; c++ {
			if v := lm.Cells[r][c]; v == 0x73 || v == 0x74 || v >= 0x80 {
				t.Fatalf("cell %d,%d: unprocessed value $%02X", c, r, v)
			}
		}
	}
	if lm.PlayerSpawn != (Point{Col: 22, Row: 1}) {
		t.Errorf("level 0 player spawn: got %+v, want {22 1}", lm.PlayerSpawn)
	}
	if len(lm.EnemySpawns) == 0 {
		t.Error("no enemy spawn candidates found")
	}
	for _, p := range lm.EnemySpawns {
		if lm.Cells[p.Row][p.Col] != 0x48 || lm.Cells[p.Row-1][p.Col] != 0x1F {
			t.Errorf("spawn %+v does not match the $48/$1F pattern", p)
		}
	}

	lm1, err := g.LevelMap(1)
	if err != nil {
		t.Fatal(err)
	}
	// Level 1 starts at the top-center shaft entrance.
	if lm1.PlayerSpawn.Col < 0x78 || lm1.PlayerSpawn.Col > 0x88 {
		t.Errorf("level 1 player spawn col $%02X not near the shaft", lm1.PlayerSpawn.Col)
	}
	// Map structure: content in columns 0-214, columns 215-254 always
	// empty padding, column 255 = wrap seam (mostly equal to column 0).
	for _, m := range []*LevelMap{lm, lm1} {
		for r := 0; r < MapHeight; r++ {
			for c := ContentWidth; c < MapWidth-1; c++ {
				if m.Cells[r][c] != 0 {
					t.Fatalf("level %d: pad cell %d,%d not empty: $%02X", m.Level, c, r, m.Cells[r][c])
				}
			}
		}
		match := 0
		for r := 0; r < MapHeight; r++ {
			if m.Cells[r][MapWidth-1] == m.Cells[r][0] {
				match++
			}
		}
		if match < 30 {
			t.Errorf("level %d: seam column matches column 0 in only %d/40 rows", m.Level, match)
		}
	}
	if _, err := g.LevelMap(2); err == nil {
		t.Error("level 2: expected range error")
	}
}

func TestCharsets(t *testing.T) {
	g := loadTestGame(t)
	play := g.PlayfieldCharset()
	if len(play) != 1024 {
		t.Fatalf("playfield charset: %d bytes", len(play))
	}
	// Char $21 is the first file-sourced glyph ($B561 -> $5908).
	for i := 0; i < 8; i++ {
		if play[0x21*8+i] != g.mem[playCharsetSrc+i] {
			t.Fatalf("char $21 byte %d mismatch", i)
		}
	}
	// Animated chars synthesized: shimmer char $0A all $55.
	for i := 0; i < 8; i++ {
		if play[0x0A*8+i] != 0x55 {
			t.Fatalf("shimmer char $0A byte %d: $%02X", i, play[0x0A*8+i])
		}
	}
	hud := g.HUDCharset()
	if hud[0x0F] != g.mem[hudCharsetSrc] { // $B298 -> $500F
		t.Error("HUD charset offset wrong")
	}
}

func TestSprites(t *testing.T) {
	g := loadTestGame(t)
	shapes := g.SpriteShapes()
	if len(shapes) != NumShapes {
		t.Fatalf("got %d shapes, want %d", len(shapes), NumShapes)
	}
	for n, blk := range shapes {
		if len(blk) != 63 {
			t.Fatalf("shape %d: %d bytes", n, len(blk))
		}
		nonzero := false
		for row := 0; row < SpriteH; row++ {
			if blk[row*3+2] != 0 {
				t.Fatalf("shape %d row %d: third column not empty", n, row)
			}
			if row >= 18 && (blk[row*3] != 0 || blk[row*3+1] != 0) {
				t.Fatalf("shape %d row %d: data past row 17", n, row)
			}
			nonzero = nonzero || blk[row*3] != 0 || blk[row*3+1] != 0
		}
		if !nonzero {
			t.Errorf("shape %d is blank", n)
		}
	}
	// Expansion matches the packed source: shape 0's pointer, row 0.
	ptr := int(g.mem[spritePtrTable]) | int(g.mem[spritePtrTable+1])<<8
	if shapes[0][0] != g.mem[ptr] || shapes[0][1] != g.mem[ptr+18] {
		t.Error("shape 0 row 0 does not match packed data")
	}

	anim := g.HelicopterAnim()
	if len(anim) != 18 {
		t.Fatalf("anim table: %d entries", len(anim))
	}
	for i, b := range anim {
		if b < 1 || b > NumShapes {
			t.Errorf("anim entry %d: block %d out of range", i, b)
		}
	}
	// Rotor pairs: entries alternate between two blocks per pose.
	if anim[0] == anim[1] || anim[6] != 7 || anim[7] != 8 {
		t.Errorf("anim table unexpected: % X", anim)
	}
	poses := g.HelicopterPoses()
	if len(poses) != 7 {
		t.Fatalf("got %d poses, want 7 (table: % X)", len(poses), anim)
	}
	if poses[3] != [2]byte{7, 8} {
		t.Errorf("level-flight pose: %v, want {7 8}", poses[3])
	}
	for i, p := range poses {
		if p[0] == p[1] {
			t.Errorf("pose %d: rotor frames identical (%v)", i, p)
		}
	}
	grid := RenderSpriteGrid([][][]byte{{shapes[0], shapes[1]}, {shapes[2]}}, 7, 1, 1)
	if grid.Bounds().Dx() != 2*SpriteW || grid.Bounds().Dy() != 2*SpriteH {
		t.Fatalf("grid bounds %v", grid.Bounds())
	}

	bullets := g.BulletShapes()
	if len(bullets) != 2 || bullets[0][0] != 0x0C || bullets[0][9] != 0x0C || bullets[1][0] != 0x0C || bullets[1][9] != 0 {
		t.Errorf("bullet blocks wrong: % X / % X", bullets[0][:12], bullets[1][:12])
	}

	img := RenderSpriteSheet(shapes, 7, 1, 2)
	if img.Bounds().Dx() != NumShapes*SpriteW*2 || img.Bounds().Dy() != SpriteH {
		t.Fatalf("sheet bounds %v", img.Bounds())
	}
	// Shape 0 row 0 is empty -> background; some pixel of the
	// helicopter body must be the sprite colour.
	want := c64Palette[7]
	found := false
	for x := 0; x < SpriteW*2 && !found; x++ {
		for y := 0; y < SpriteH && !found; y++ {
			r, gg, b, _ := img.At(x, y).RGBA()
			if byte(r>>8) == want.R && byte(gg>>8) == want.G && byte(b>>8) == want.B {
				found = true
			}
		}
	}
	if !found {
		t.Error("no sprite-coloured pixel in frame 0")
	}
}

func TestRenderPNG(t *testing.T) {
	g := loadTestGame(t)
	lm, err := g.LevelMap(0)
	if err != nil {
		t.Fatal(err)
	}
	cs := g.PlayfieldCharset()
	img := RenderMap(lm, cs, g.MulticolorValue(0), 1, true)
	b := img.Bounds()
	if b.Dx() != (ContentWidth+1)*8 || b.Dy() != MapHeight*8 {
		t.Fatalf("map image %dx%d, want %dx%d", b.Dx(), b.Dy(), (ContentWidth+1)*8, MapHeight*8)
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "map.png")
	if err := WritePNG(p, img); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(p); err != nil || fi.Size() == 0 {
		t.Fatalf("png not written: %v", err)
	}
	cimg := RenderCharset(cs, 2, 2)
	if cimg.Bounds().Dx() != 16*8*2 || cimg.Bounds().Dy() != 8*8*2 {
		t.Fatalf("charset image bounds %v", cimg.Bounds())
	}
}
