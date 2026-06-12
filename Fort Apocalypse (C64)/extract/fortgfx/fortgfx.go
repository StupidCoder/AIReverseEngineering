// Package fortgfx extracts graphics from the Fort Apocalypse game file
// (FORT-fast-7000.prg, load address $7000) and renders them as PNG
// images: the two character sets and the decompressed level maps
// (stored as 40 pages of 256 bytes; 215 content columns plus a wrap
// seam column), optionally with player/enemy spawn markers.
//
// All file offsets and algorithms mirror the game's own code (see
// GAME.md): the table-selective RLE decompressor at $8CDB, the charset
// build loop at $899C, the spawn scan at $90A4 and the player start
// tables at $910A/$9110.
package fortgfx

import (
	"fmt"
	"image"
	"image/color"
	"os"

	"stupidcoder.com/c64tools/gfx"
)

const loadAddr = 0x7000

// Game file landmarks (addresses in C64 memory).
const (
	runTableTerrain = 0x8D2B // 23 run-length-capable terrain codes
	runTableLen     = 23
	levelSrcTable   = 0x8D46 // per-level map stream addresses (words)
	colorTable      = 0x9107 // per-level multicolor ($D022) value
	spriteStartTbl  = 0x910A // per-level player sprite start X,Y
	cameraStartTbl  = 0x9110 // per-level camera start col,row
	tankHomeTbl0    = 0x911C // 6 tank home cols (levels 0/2), rows at $9122
	tankHomeTbl1    = 0x9128 // 6 tank home cols (level 1), rows at $912E
	enemyPatrolTbl  = 0x9CBA // 16 enemy-heli patrol cols, rows at $9CCA
	dropPointTbl    = 0x98DA // 4 cavern drop points: $57,$59,$56,$58,$65,$66 presets
	barrierPattern  = 0x8907 // 32 bytes: energy barrier chars 1-4
	barrierTop      = 0x891F // 8 bytes: barrier cap char 9
	waterPattern    = 0xA927 // 8 bytes: water/static chars $20/$3F
	hudCharsetSrc   = 0xB298 // raw HUD charset stream -> $500F+
	playCharsetSrc  = 0xB561 // raw playfield charset stream -> $5908+
	spritePtrTable  = 0x86F3 // 14 words: packed sprite shape locations
	bulletPattern   = 0xB0C2 // 9 bytes: bullet sprite rows ($B0B0)
	heliAnimTable   = 0xA320 // 18 bytes: tilt -> sprite block (player & enemy)
)

// NumShapes is the number of packed sprite shapes ($870F-$8906).
const NumShapes = 14

const (
	// MapWidth is the storage width: one 256-byte page per map row.
	MapWidth  = 256
	MapHeight = 40
	// ContentWidth is the real playfield width: columns 0-214 hold
	// terrain, columns 215-254 are always empty padding, and column
	// 255 is a near-duplicate of column 0 — the wrap seam shown at
	// the screen's right edge when the camera wraps ($A666/$A688:
	// camera column $D9 <-> $02). Rendered maps therefore show the
	// seam column first, then columns 0-214.
	ContentWidth = 215
)

// Game gives access to the loaded program file.
type Game struct {
	mem []byte // indexed by C64 address
}

// LoadGame reads FORT-fast-7000.prg.
func LoadGame(path string) (*Game, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(raw) < 3 {
		return nil, fmt.Errorf("fortgfx: %s: too short", path)
	}
	load := int(raw[0]) | int(raw[1])<<8
	if load != loadAddr {
		return nil, fmt.Errorf("fortgfx: %s: load address $%04X, want $7000", path, load)
	}
	mem := make([]byte, 0x10000)
	copy(mem[load:], raw[2:])
	return &Game{mem: mem}, nil
}

// PlayfieldCharset reconstructs the in-game charset at $5800 (128
// chars x 8 bytes). Chars $21+ are copied from the file as the init
// code does; the animated soft chars ($01-$11, $20, $3F) are
// synthesized in their "on" state.
func (g *Game) PlayfieldCharset() []byte {
	cs := make([]byte, 128*8)
	// init loop $899C: $B561->$5908, $B659->$5A00, $B759->$5B00
	copy(cs[0x108:0x400], g.mem[playCharsetSrc:playCharsetSrc+0x2F8])
	// $A7ED/$A830: energy barriers, chars 1-8 + cap char 9
	copy(cs[1*8:], g.mem[barrierPattern:barrierPattern+32]) // chars 1-4
	copy(cs[5*8:], g.mem[barrierPattern:barrierPattern+32]) // chars 5-8
	copy(cs[9*8:], g.mem[barrierTop:barrierTop+8])
	// $A86B: shimmer chars $0A-$0D ($55 rows when lit)
	for i := 0x0A * 8; i < 0x0E*8; i++ {
		cs[i] = 0x55
	}
	// $A8B8: rotating beacon chars $0E-$11; phase 0 lights char $0E
	for i := 0x0E * 8; i < 0x0F*8; i++ {
		cs[i] = 0x55
	}
	// $A8F3: water/static chars $20 and $3F (pattern, sans noise)
	copy(cs[0x20*8:], g.mem[waterPattern:waterPattern+8])
	copy(cs[0x3F*8:], g.mem[waterPattern:waterPattern+8])
	return cs
}

// HUDCharset reconstructs the charset at $5000 used for screen rows
// 0-6 (font, HUD furniture). The scanner soft chars $70-$7F are blank
// here; they are drawn at runtime.
func (g *Game) HUDCharset() []byte {
	cs := make([]byte, 128*8)
	// init loop $899C: $B298->$500F, $B389->$5100, $B489->$5200
	copy(cs[0x00F:0x300], g.mem[hudCharsetSrc:hudCharsetSrc+0x2F1])
	return cs
}

// Point is a map cell position (column 0-255, row 0-39).
type Point struct{ Col, Row int }

// LevelMap is one decompressed level.
type LevelMap struct {
	Level          int
	Cells          [MapHeight][MapWidth]byte // screen character codes
	PlayerSpawn    Point
	PrisonerSpawns []Point // all candidate prisoner positions ($90A4 pattern)
	TankHomes      []Point // 6 fixed tank home positions (leftmost body cell)
	EnemySpawns    []Point // unique enemy-helicopter patrol points (visual top-left)
	DropPoints     []Point // cavern teleport destinations, level 0 only (craft center)
}

// LevelMap decompresses level 0 or 1 and derives the spawn positions.
func (g *Game) LevelMap(level int) (*LevelMap, error) {
	if level < 0 || level > 1 {
		return nil, fmt.Errorf("fortgfx: level %d out of range (0-1)", level)
	}
	lm := &LevelMap{Level: level}

	// RLE decode, mirroring $8CDB with the terrain run table.
	runnable := map[byte]bool{}
	for _, b := range g.mem[runTableTerrain : runTableTerrain+runTableLen] {
		runnable[b] = true
	}
	src := int(g.mem[levelSrcTable+2*level]) | int(g.mem[levelSrcTable+2*level+1])<<8
	flat := make([]byte, 0, MapWidth*MapHeight)
	rng := uint32(0x1234567) // stand-in for the SID noise the game uses
	rand2bit := func() byte {
		for {
			rng = rng*1103515245 + 12345
			if v := byte(rng>>16) & 3; v != 0 {
				return v
			}
		}
	}
	for len(flat) < MapWidth*MapHeight {
		b := g.mem[src]
		src++
		n := 1
		if runnable[b] {
			n = int(g.mem[src])
			src++
			if n == 0 {
				n = 256
			}
		}
		v := b & 0x7F
		for i := 0; i < n; i++ {
			// post-pass $8FC2: randomized rock texture
			switch v {
			case 0x73:
				flat = append(flat, 0x61+rand2bit())
			case 0x74:
				flat = append(flat, 0x64+rand2bit())
			default:
				flat = append(flat, v)
			}
		}
	}
	for r := 0; r < MapHeight; r++ {
		copy(lm.Cells[r][:], flat[r*MapWidth:])
	}

	// Player spawn: sprite/camera start values from the $8EDD tables.
	// The marker shows the craft's *visual* position, not the game's
	// logic coordinate $69 (which sits 3 chars into the sprite):
	// hardware X = 2*(sx-$24) ($B0CB), the screen's first pixel column
	// is at hardware X 24, and the window's left column is buffer
	// camCol-1 — so the sprite's left edge is at buffer column
	// (sx-$30)/4 + camCol - 1, rounded to the nearest character.
	sx := int(g.mem[spriteStartTbl+2*level])
	sy := int(g.mem[spriteStartTbl+2*level+1])
	camCol := int(g.mem[cameraStartTbl+2*level])
	camRow := int(g.mem[cameraStartTbl+2*level+1])
	lm.PlayerSpawn = Point{Col: (sx-0x30+2)/4 + camCol - 1, Row: (sy-0x58)/8 + camRow}

	// Prisoner spawn candidates ($90A4): two $48 floor cells side by
	// side with rock $1F directly above. The level builder turns up to
	// 8 random candidates into prisoners (the $3600 tables).
	for r := 1; r < MapHeight; r++ {
		for c := 0; c < MapWidth-1; c++ {
			if lm.Cells[r][c] == 0x48 && lm.Cells[r][c+1] == 0x48 && lm.Cells[r-1][c] == 0x1F {
				lm.PrisonerSpawns = append(lm.PrisonerSpawns, Point{Col: c, Row: r})
			}
		}
	}

	// Tank homes ($8F55 tables): 6 fixed positions per level, stored in
	// game coordinates (buffer column = value - 5). The home column is
	// the leftmost cell of the 3-cell tank body.
	homeTbl := tankHomeTbl0
	if level == 1 {
		homeTbl = tankHomeTbl1
	}
	for i := 0; i < 6; i++ {
		lm.TankHomes = append(lm.TankHomes, Point{
			Col: int(g.mem[homeTbl+i]) - 5,
			Row: int(g.mem[homeTbl+6+i]),
		})
	}

	// Enemy-helicopter patrol points ($9C6B): 8 table entries per
	// level at $9CBA/$9CCA (level 1 uses the second half), with
	// duplicates for spawn-probability weighting. Stored in the
	// enemy's coordinate space; its sprite placement ($A1D6/$A1F7:
	// x = (col-cam)*4+$16, y = (row-cam)*8+$53) puts the craft's
	// visual top-left at buffer column col-7.5, row col-2.1 — rounded
	// here to (col-7, row-2).
	base := enemyPatrolTbl + 8*level
	for i := 0; i < 8; i++ {
		p := Point{
			Col: int(g.mem[base+i]) - 7,
			Row: int(g.mem[base+16+i]) - 2,
		}
		dup := false
		for _, q := range lm.EnemySpawns {
			if q == p {
				dup = true
			}
		}
		if !dup {
			lm.EnemySpawns = append(lm.EnemySpawns, p)
		}
	}

	// Cavern drop points ($9892, level 0 only): the barrier-gate
	// teleport picks one of 4 presets (fine scroll, camera, sprite
	// position) from $98DA+. Converted to the craft's visual center:
	// centre px = 2*(sx-$24) - 24 + 16 - fineX; row from sy as for
	// the player spawn, plus half the 3-row footprint.
	if level == 0 {
		for i := 0; i < 4; i++ {
			fx := int(g.mem[dropPointTbl+i]) & 7
			cc := int(g.mem[dropPointTbl+8+i])
			cr := int(g.mem[dropPointTbl+12+i])
			sx := int(g.mem[dropPointTbl+16+i])
			sy := int(g.mem[dropPointTbl+20+i])
			lm.DropPoints = append(lm.DropPoints, Point{
				Col: (2*sx-80-fx+4)/8 + cc - 1,
				Row: (sy-100+12)/8 + cr,
			})
		}
	}
	return lm, nil
}

// MulticolorValue returns the level's $D022 colour register value.
func (g *Game) MulticolorValue(level int) byte {
	return g.mem[colorTable+level] & 0x0F
}

// SpriteShapes expands the 14 packed shapes (36 bytes each: two
// 18-byte pixel columns, located via the pointer table at $86F3) into
// 63-byte VIC sprite blocks, exactly like the game's init code at
// $B044: each of the 18 used rows becomes [left][right][$00].
func (g *Game) SpriteShapes() [][]byte {
	shapes := make([][]byte, NumShapes)
	for n := range shapes {
		ptr := int(g.mem[spritePtrTable+2*n]) | int(g.mem[spritePtrTable+2*n+1])<<8
		blk := make([]byte, 63)
		for row := 0; row < 18; row++ {
			blk[row*3] = g.mem[ptr+row]
			blk[row*3+1] = g.mem[ptr+18+row]
		}
		shapes[n] = blk
	}
	return shapes
}

// BulletShapes builds the two projectile sprite blocks the game
// creates at $4800/$4840 ($B0B0): block $20 holds the 9-byte pattern
// twice (rows 0-5), block $21 once (rows 0-2).
func (g *Game) BulletShapes() [][]byte {
	pat := g.mem[bulletPattern : bulletPattern+9]
	b20 := make([]byte, 63)
	copy(b20, pat)
	copy(b20[9:], pat)
	b21 := make([]byte, 63)
	copy(b21, pat)
	return [][]byte{b20, b21}
}

// HelicopterAnim returns the 18-entry animation table at $A320 that
// maps a bank/tilt value 0-$11 to a sprite block number 1-14. Both
// the player (index = tilt $67, bit 0 = rotor phase per frame) and the
// enemy helicopter (index = tilt $71, rotor toggled every 4 frames)
// use this same table: 7 banking poses, two rotor frames each, with
// the level-flight pose 7/8 repeated for three tilt steps.
func (g *Game) HelicopterAnim() []byte {
	return g.mem[heliAnimTable : heliAnimTable+18]
}

// HelicopterPoses reduces the animation table to its distinct banking
// poses, in tilt order (full-left ... level ... full-right). Each pose
// is the pair of sprite blocks the rotor animation alternates between.
func (g *Game) HelicopterPoses() [][2]byte {
	anim := g.HelicopterAnim()
	var poses [][2]byte
	for i := 0; i+1 < len(anim); i += 2 {
		pair := [2]byte{anim[i], anim[i+1]}
		if len(poses) == 0 || poses[len(poses)-1] != pair {
			poses = append(poses, pair)
		}
	}
	return poses
}

// multicolor pixel-pair palette indices for the playfield:
// 00 = $D021 (black), 01 = $D022 (per level), 10 = $D023 (white),
// 11 = colour RAM (the playfield rows are mostly $0D = green).
func mcPalette(d022 byte) [4]color.RGBA {
	return [4]color.RGBA{
		gfx.Palette[0], gfx.Palette[d022&0x0F], gfx.Palette[1], gfx.Palette[5],
	}
}

// RenderCharset renders all 128 chars as a 16x8 grid, scale s.
func RenderCharset(charset []byte, d022 byte, s int) *image.RGBA {
	pal := mcPalette(d022)
	img := image.NewRGBA(image.Rect(0, 0, 16*8*s, 8*8*s))
	for ch := 0; ch < 128; ch++ {
		gfx.DrawChar(img, charset[ch*8:ch*8+8], (ch%16)*8*s, (ch/16)*8*s, s, pal)
	}
	return img
}

// RenderMap renders a level map at its true width (216 chars: the wrap
// seam column stored at offset 255, then content columns 0-214; the
// empty padding columns 215-254 are cropped), scale s. If markers is
// true, the player spawn is framed in cyan, every prisoner spawn
// candidate in yellow, the tank homes (body + turret row) in light
// red, and the enemy-helicopter patrol points in light green (both
// helicopter markers sized to the craft's 4x3-character footprint).
func RenderMap(lm *LevelMap, charset []byte, d022 byte, s int, markers bool) *image.RGBA {
	pal := mcPalette(d022)
	width := ContentWidth + 1 // seam column + content
	img := image.NewRGBA(image.Rect(0, 0, width*8*s, MapHeight*8*s))
	for r := 0; r < MapHeight; r++ {
		seam := int(lm.Cells[r][MapWidth-1])
		gfx.DrawChar(img, charset[seam*8:seam*8+8], 0, r*8*s, s, pal)
		for c := 0; c < ContentWidth; c++ {
			ch := int(lm.Cells[r][c])
			gfx.DrawChar(img, charset[ch*8:ch*8+8], (c+1)*8*s, r*8*s, s, pal)
		}
	}
	if markers {
		cyan, yellow, lightRed := gfx.Palette[3], gfx.Palette[7], gfx.Palette[10]
		for _, p := range lm.PrisonerSpawns {
			// the floor pattern is 2 cells wide; the prisoner himself
			// is 2 chars tall (torso drawn one row above the leg cell)
			gfx.FrameCell(img, p.Col+1, p.Row-1, 2, 2, s, yellow)
		}
		for _, p := range lm.TankHomes {
			// 3-cell body plus the turret row above
			gfx.FrameCell(img, p.Col+1, p.Row-1, 3, 2, s, lightRed)
		}
		lightGreen := gfx.Palette[13]
		for _, p := range lm.EnemySpawns {
			gfx.FrameCell(img, p.Col+1, p.Row, 4, 3, s, lightGreen)
		}
		white := gfx.Palette[1]
		for _, p := range lm.DropPoints {
			gfx.DrawCircle(img, ((p.Col+1)*8+4)*s, (p.Row*8+4)*s, 6*s, s, white)
		}
		// The helicopter's visual footprint: 16x18 used sprite pixels,
		// X-expanded = 4 chars wide, ~3 chars tall, from the left edge.
		gfx.FrameCell(img, lm.PlayerSpawn.Col+1, lm.PlayerSpawn.Row, 4, 3, s, cyan)
	}
	return img
}
