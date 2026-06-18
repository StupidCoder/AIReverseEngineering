// webexport renders Turrican's disk-streamed level maps for the companion
// website's tile viewer. For each world it writes a tile atlas PNG (the world's
// 32x32 tiles in its palette) and, for each scene in that world, a JSON file with
// the column-major map flattened to a row-major cell grid. A meta.json lists every
// scene. The viewer resolves the flip flag (cell >= ntiles -> tile cell-128,
// horizontally flipped) just like extract/cmd/map.
//
// It also emits the object layer: a per-world object atlas (objatlas<w>.png) holding
// the first animation frame of every sprite the scenes' objects use, and per scene
// an `objects` list (each object's pixel position and a sprite index into the
// scene's `objSprites` rects). Placement comes straight off the disk via the
// extract/scene package (the scroll-triggered spawner's grid + entry lists); the
// sprite each object uses is resolved type -> AI handler -> the frame table it
// installs (see Turrican.md Part V §6).
//
// Usage: webexport [-o dir] [Turrican.adf]
package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"

	"turrican/extract/scene"
)

const (
	blockBase = 0x1B980
	tileSide   = 32
	tileBytes  = tileSide * 4 * (tileSide / 8) // 512
	atlasCols  = 16
	tileGutter = 1                       // extruded border per tile (kills atlas bleed at fractional zoom)
	atlasCell  = tileSide + 2*tileGutter // 34
	objAtlasW  = 256                     // object-atlas shelf width in pixels
)

type objSprite struct {
	X int `json:"x"` // sprite's rect inside the world object atlas
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}
type objInst struct {
	X int `json:"x"` // top-left pixel position on the map
	Y int `json:"y"`
	S int `json:"s"` // index into the scene's objSprites
}

type point struct {
	X int `json:"x"`
	Y int `json:"y"`
}
type rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type jsonLevel struct {
	World      int         `json:"world"`
	Scene      int         `json:"scene"`
	Width      int         `json:"width"`
	Height     int         `json:"height"`
	NTiles     int         `json:"ntiles"`
	Atlas      string      `json:"atlas"`
	Cells      []int       `json:"cells"` // row-major, raw map bytes (>=ntiles = flipped tile-128)
	Spawn      point       `json:"spawn"` // player spawn, world pixels
	View       rect        `json:"view"`  // the Amiga on-screen viewport at the spawn, world pixels
	ObjAtlas   string      `json:"objAtlas,omitempty"`
	ObjSprites []objSprite `json:"objSprites,omitempty"`
	Objects    []objInst   `json:"objects,omitempty"`
}

// amigaView is Turrican's visible playfield in pixels (one screen).
const amigaViewW, amigaViewH = 320, 256

type metaLevel struct {
	Name  string `json:"name"`
	File  string `json:"file"`
	Atlas string `json:"atlas"`
}

// spriteKey identifies a frame table within a world (resident and scene tables share
// no addresses, but the flag keeps the mapping explicit).
type spriteKey struct {
	ft       int
	resident bool
}

func main() {
	out := flag.String("o", "site/public/turrican", "output directory")
	flag.Parse()
	adfPath := flag.Arg(0)
	if adfPath == "" {
		adfPath = "Turrican (Amiga)/Turrican.adf"
	}
	if err := run(adfPath, *out); err != nil {
		fmt.Fprintln(os.Stderr, "webexport:", err)
		os.Exit(1)
	}
}

func run(adfPath, outDir string) error {
	adf, err := os.ReadFile(adfPath)
	if err != nil {
		return err
	}
	game, err := scene.Load(adf)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	var meta []metaLevel
	for w := 0; w < scene.NumWorlds; w++ {
		block := game.Block(w).Data
		be32 := func(o int) int { return int(binary.BigEndian.Uint32(block[o:])) }
		at := func(addr int) int { return addr - blockBase }

		pal := readPalette(block, at(be32(0x08)))
		tableOff := at(be32(0x00))
		nTiles := be32(tableOff) / 4

		atlasName := fmt.Sprintf("atlas%d.png", w)
		if err := writeAtlas(filepath.Join(outDir, atlasName), block, tableOff, nTiles, pal); err != nil {
			return err
		}

		scenes := game.Scenes(w)

		// Collect every sprite the world's objects use, then pack each one's first
		// frame into the world object atlas.
		used := map[spriteKey]bool{}
		for _, sc := range scenes {
			for _, o := range sc.Objects {
				if o.FT != 0 {
					used[spriteKey{o.FT, o.Resident}] = true
				}
			}
		}
		objAtlasName := fmt.Sprintf("objatlas%d.png", w)
		rectOf, err := writeObjAtlas(filepath.Join(outDir, objAtlasName), game, w, used)
		if err != nil {
			return err
		}

		for _, sc := range scenes {
			descOff := at(be32(0x16 + sc.Index*4))
			mapOff := at(be32(descOff + 0x00))
			if sc.Width <= 0 || sc.Height <= 0 || mapOff+sc.Width*sc.Height > len(block) {
				return fmt.Errorf("world %d scene %d: bad map %dx%d", w, sc.Index, sc.Width, sc.Height)
			}
			cells := make([]int, sc.Width*sc.Height)
			for col := 0; col < sc.Width; col++ {
				for row := 0; row < sc.Height; row++ {
					cells[row*sc.Width+col] = int(block[mapOff+col*sc.Height+row]) // col-major -> row-major
				}
			}

			// Index each object into a per-scene objSprites rect list.
			var sprites []objSprite
			spriteIdx := map[spriteKey]int{}
			var objects []objInst
			for _, o := range sc.Objects {
				r, ok := rectOf[spriteKey{o.FT, o.Resident}]
				if !ok {
					continue // no sprite resolved (unhandled type) — skip
				}
				k := spriteKey{o.FT, o.Resident}
				si, seen := spriteIdx[k]
				if !seen {
					si = len(sprites)
					spriteIdx[k] = si
					sprites = append(sprites, r)
				}
				objects = append(objects, objInst{X: o.X, Y: o.Y, S: si})
			}

			file := fmt.Sprintf("world%d_scene%d.json", w, sc.Index)
			lvl := jsonLevel{
				World: w, Scene: sc.Index, Width: sc.Width, Height: sc.Height,
				NTiles: nTiles, Atlas: atlasName, Cells: cells,
				Spawn: point{sc.SpawnX, sc.SpawnY},
				View:  rect{sc.CamX, sc.CamY, amigaViewW, amigaViewH},
			}
			if len(objects) > 0 {
				lvl.ObjAtlas = objAtlasName
				lvl.ObjSprites = sprites
				lvl.Objects = objects
			}
			if err := writeJSON(filepath.Join(outDir, file), lvl); err != nil {
				return err
			}
			meta = append(meta, metaLevel{
				Name: fmt.Sprintf("World %d · Scene %d", w+1, sc.Index+1),
				File: file, Atlas: atlasName,
			})
			fmt.Printf("world %d scene %d: %dx%d, %d tiles, %d objects -> %s\n", w, sc.Index, sc.Width, sc.Height, nTiles, len(objects), file)
		}
	}
	return writeJSON(filepath.Join(outDir, "meta.json"), map[string]any{"levels": meta})
}

// writeObjAtlas shelf-packs the first frame of each used sprite into one paletted PNG
// (index 0 transparent) and returns each sprite's rect in it.
func writeObjAtlas(path string, game *scene.Game, w int, used map[spriteKey]bool) (map[spriteKey]objSprite, error) {
	// Deterministic order: residents first, then by address.
	keys := make([]spriteKey, 0, len(used))
	for k := range used {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].resident != keys[j].resident {
			return keys[i].resident
		}
		return keys[i].ft < keys[j].ft
	})

	rectOf := map[spriteKey]objSprite{}
	type placed struct {
		sp   scene.Space
		f    scene.Frame
		x, y int
	}
	var ps []placed
	cx, cy, shelfH, atlasH := 0, 0, 0, 0
	for _, k := range keys {
		sp := game.Space(w, k.resident)
		lo, hi := game.GfxRange(w, k.resident)
		f, ok := scene.FirstFrame(sp, k.ft, lo, hi)
		if !ok {
			continue
		}
		pw := f.DW * 16
		if cx+pw > objAtlasW {
			cy += shelfH
			cx, shelfH = 0, 0
		}
		ps = append(ps, placed{sp: sp, f: f, x: cx, y: cy})
		rectOf[k] = objSprite{X: cx, Y: cy, W: pw, H: f.H}
		cx += pw
		if f.H > shelfH {
			shelfH = f.H
		}
		if cy+shelfH > atlasH {
			atlasH = cy + shelfH
		}
	}
	if atlasH == 0 {
		return rectOf, nil
	}
	img := image.NewPaletted(image.Rect(0, 0, objAtlasW, atlasH), game.WorldPalette(w, true))
	for _, p := range ps {
		scene.DrawFirstFrame(img, p.sp, p.f, p.x, p.y)
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return rectOf, png.Encode(f, img)
}

// writeAtlas packs the world's 32x32 tiles into a grid PNG. Each tile sits in a
// (tileSide+2*tileGutter)-pixel cell whose 1-pixel border duplicates the tile's edge
// pixels (extrusion): when the viewer samples a sub-pixel past a tile's edge at
// fractional zoom, it hits the tile's own colour instead of bleeding the neighbour.
func writeAtlas(path string, block []byte, tableOff, nTiles int, pal color.Palette) error {
	rows := (nTiles + atlasCols - 1) / atlasCols
	img := image.NewPaletted(image.Rect(0, 0, atlasCols*atlasCell, rows*atlasCell), pal)
	clamp := func(v int) int {
		if v < 0 {
			return 0
		}
		if v >= tileSide {
			return tileSide - 1
		}
		return v
	}
	for n := 0; n < nTiles; n++ {
		off := tableOff + int(binary.BigEndian.Uint32(block[tableOff+n*4:]))
		if off+tileBytes > len(block) {
			break
		}
		var t [tileSide][tileSide]uint8
		for y := 0; y < tileSide; y++ {
			var planes [4]uint32
			for p := 0; p < 4; p++ {
				planes[p] = binary.BigEndian.Uint32(block[off+(y*4+p)*4:])
			}
			for x := 0; x < tileSide; x++ {
				var v uint8
				for p := 0; p < 4; p++ {
					v |= uint8((planes[p]>>(31-uint(x)))&1) << uint(p)
				}
				t[y][x] = v
			}
		}
		ox, oy := (n%atlasCols)*atlasCell, (n/atlasCols)*atlasCell
		for y := -tileGutter; y < tileSide+tileGutter; y++ {
			for x := -tileGutter; x < tileSide+tileGutter; x++ {
				img.SetColorIndex(ox+tileGutter+x, oy+tileGutter+y, t[clamp(y)][clamp(x)])
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func readPalette(block []byte, off int) color.Palette {
	pal := make(color.Palette, 16)
	for i := range pal {
		c := binary.BigEndian.Uint16(block[off+i*2:])
		pal[i] = color.RGBA{R: uint8((c>>8)&0xF) * 17, G: uint8((c>>4)&0xF) * 17, B: uint8(c&0xF) * 17, A: 255}
	}
	return pal
}

func writeJSON(path string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
