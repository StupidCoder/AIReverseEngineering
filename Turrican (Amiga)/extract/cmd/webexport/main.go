// webexport renders Turrican's disk-streamed level maps for the companion
// website's tile viewer. For each world it writes a tile atlas PNG (the world's
// 32x32 tiles in its palette) and, for each scene in that world, a JSON file with
// the column-major map flattened to a row-major cell grid. A meta.json lists every
// scene. The viewer resolves the flip flag (cell >= ntiles -> tile cell-128,
// horizontally flipped) just like extract/cmd/map.
//
// It also emits the enemy object layer: a per-world object atlas (objatlas<w>.png)
// holding the first animation frame of every enemy sprite the world's AI handlers
// install, and per scene an `objects` list (each enemy's pixel position and a sprite
// index into the scene's `objSprites` rects). Placement comes straight off the disk
// (the scroll-triggered spawner's lists); the sprite each placement uses is resolved
// type -> +$20 AI handler -> the frame table it installs (see Turrican.md Part V §6).
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

	"turrican/extract/decrunch"
)

const (
	blockBase  = 0x1B980
	levelTable = 0x46A
	numWorlds  = 5
	tileSide   = 32
	tileBytes  = tileSide * 4 * (tileSide / 8) // 512
	atlasCols  = 16
	unitsTile  = 4   // 8-pixel placement units per 32-pixel tile
	objAtlasW  = 256 // object-atlas shelf width in pixels
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

type jsonLevel struct {
	World      int         `json:"world"`
	Scene      int         `json:"scene"`
	Width      int         `json:"width"`
	Height     int         `json:"height"`
	NTiles     int         `json:"ntiles"`
	Atlas      string      `json:"atlas"`
	Cells      []int       `json:"cells"` // row-major, raw map bytes (>=ntiles = flipped tile-128)
	ObjAtlas   string      `json:"objAtlas,omitempty"`
	ObjSprites []objSprite `json:"objSprites,omitempty"`
	Objects    []objInst   `json:"objects,omitempty"`
}

type metaLevel struct {
	Name  string `json:"name"`
	File  string `json:"file"`
	Atlas string `json:"atlas"`
}

// rawObj is a placement before sprite atlasing: pixel position + frame-table addr.
type rawObj struct {
	px, py, ft int
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
	res, err := decrunch.Decrunch(mainBlob(adf))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	var meta []metaLevel
	for w := 0; w < numWorlds; w++ {
		t := levelTable + w*8
		off := int(binary.BigEndian.Uint32(res.Data[t:]))
		length := int(binary.BigEndian.Uint32(res.Data[t+4:]))
		block, err := decrunch.DecrunchBlock(adf[off : off+length])
		if err != nil {
			return fmt.Errorf("world %d: %w", w, err)
		}

		be16 := func(o int) int { return int(binary.BigEndian.Uint16(block[o:])) }
		be32 := func(o int) int { return int(binary.BigEndian.Uint32(block[o:])) }
		at := func(addr int) int { return addr - blockBase }
		blockHi := blockBase + len(block)

		pal := readPalette(block, at(be32(0x08)))
		tableOff := at(be32(0x00))
		nTiles := be32(tableOff) / 4

		atlasName := fmt.Sprintf("atlas%d.png", w)
		if err := writeAtlas(filepath.Join(outDir, atlasName), block, tableOff, nTiles, pal); err != nil {
			return err
		}

		nScenes := be16(0x14) // header+$14 high word = scene count

		// Pass 1: parse every scene's placements off the disk, collecting the set
		// of sprite frame tables the world references so we can build one atlas.
		rawByScene := make([][]rawObj, nScenes)
		ftSet := map[int]bool{}
		var ftOrder []int
		for s := 0; s < nScenes; s++ {
			objs := sceneObjects(block, be16, be32, at, blockHi, s)
			rawByScene[s] = objs
			for _, o := range objs {
				if o.ft != 0 && !ftSet[o.ft] {
					ftSet[o.ft] = true
					ftOrder = append(ftOrder, o.ft)
				}
			}
		}

		// Render each referenced sprite's first frame into the world object atlas
		// (shelf-packed); rectOf maps a frame table to its atlas rect.
		objAtlasName := fmt.Sprintf("objatlas%d.png", w)
		rectOf, err := writeObjAtlas(filepath.Join(outDir, objAtlasName), block, at, blockHi, ftOrder, objPalette(block, at(be32(0x08))))
		if err != nil {
			return err
		}

		for s := 0; s < nScenes; s++ {
			descOff := at(be32(0x16 + s*4))
			mapOff := at(be32(descOff + 0x00))
			width := be16(descOff + 0x04)
			height := be16(descOff + 0x06)
			if width <= 0 || height <= 0 || mapOff+width*height > len(block) {
				return fmt.Errorf("world %d scene %d: bad map %dx%d", w, s, width, height)
			}
			cells := make([]int, width*height)
			for col := 0; col < width; col++ {
				for row := 0; row < height; row++ {
					cells[row*width+col] = int(block[mapOff+col*height+row]) // col-major -> row-major
				}
			}

			// Build the scene's object list, keeping only in-bounds placements and
			// indexing each into a per-scene objSprites rect list.
			var sprites []objSprite
			spriteIdx := map[int]int{}
			var objects []objInst
			for _, o := range rawByScene[s] {
				r, ok := rectOf[o.ft]
				if !ok {
					continue
				}
				if o.px < 0 || o.py < 0 || o.px >= width*tileSide || o.py >= height*tileSide {
					continue
				}
				si, seen := spriteIdx[o.ft]
				if !seen {
					si = len(sprites)
					spriteIdx[o.ft] = si
					sprites = append(sprites, r)
				}
				objects = append(objects, objInst{X: o.px, Y: o.py, S: si})
			}

			file := fmt.Sprintf("world%d_scene%d.json", w, s)
			lvl := jsonLevel{
				World: w, Scene: s, Width: width, Height: height,
				NTiles: nTiles, Atlas: atlasName, Cells: cells,
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
				Name: fmt.Sprintf("World %d · Scene %d", w+1, s+1),
				File: file, Atlas: atlasName,
			})
			fmt.Printf("world %d scene %d: %dx%d, %d tiles, %d objects -> %s\n", w, s, width, height, nTiles, len(objects), file)
		}
	}
	return writeJSON(filepath.Join(outDir, "meta.json"), map[string]any{"levels": meta})
}

// sceneObjects reads scene s's placement list off the disk and resolves each entry
// to a pixel position and the frame table its enemy-AI handler installs.
func sceneObjects(block []byte, be16, be32, at func(int) int, blockHi, s int) []rawObj {
	descOff := at(be32(0x16 + s*4))
	width := be16(descOff + 0x04)
	height := be16(descOff + 0x06)
	grid := be32(descOff + 0x28)

	// The grid is an array of pointers into the sorted entry stream that begins
	// immediately after it, so a grid entry always lives below the lowest address it
	// points to: scan while the cursor is still below the minimum pointer seen. This
	// is what pins the stream extent on narrow/tall scenes (a fixed window would run
	// off the small grid into unrelated block data and grossly over-read).
	lo, hi := 0, 0
	for a := grid; at(a)+4 <= len(block); a += 4 {
		if lo != 0 && a >= lo {
			break
		}
		p := be32(at(a))
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
		return nil
	}

	// type & $F indexes the scene's +$20 AI-handler table; each handler installs a
	// frame table via MOVE.l #ft,$12(a5).
	aiTbl := be32(descOff + 0x20)
	var ai []int
	for i := 0; i < 16; i++ {
		if at(aiTbl+i*4)+4 > len(block) {
			break
		}
		h := be32(at(aiTbl + i*4))
		if h < blockBase || h >= blockHi {
			break
		}
		ai = append(ai, frameTableOf(block, h, blockBase, blockHi))
	}

	var out []rawObj
	for a := lo; a < hi+60 && at(a)+6 <= len(block); {
		ty := be16(at(a))
		if ty == 0 {
			a += 2
			continue
		}
		if ty&0xFF == 0xD3 {
			a += 6
			continue
		}
		x, y := be16(at(a+2)), be16(at(a+4))
		if x < width*unitsTile && y < height*unitsTile {
			ft := 0
			if n := ty & 0xF; n < len(ai) {
				ft = ai[n]
			}
			out = append(out, rawObj{px: x * 8, py: y * 8, ft: ft}) // 8-px placement units -> pixels
		}
		a += 6
	}
	return out
}

// frameTableOf scans an AI handler for MOVE.l #ft,$12(a5) (2B 7C imm32 00 12).
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

// writeObjAtlas shelf-packs the first frame of each frame table into one paletted
// PNG (index 0 transparent) and returns each frame table's rect in it.
func writeObjAtlas(path string, block []byte, at func(int) int, blockHi int, fts []int, pal color.Palette) (map[int]objSprite, error) {
	rectOf := map[int]objSprite{}
	type placed struct {
		ft, bm, w, h int // w in pixels
		x, y         int
	}
	var ps []placed
	cx, cy, shelfH, atlasH := 0, 0, 0, 0
	for _, ft := range fts {
		bm, h, dw, ok := firstFrame(block, ft, at, blockHi)
		if !ok {
			continue
		}
		pw := dw * 16
		if cx+pw > objAtlasW {
			cy += shelfH
			cx, shelfH = 0, 0
		}
		ps = append(ps, placed{ft: ft, bm: bm, w: dw, h: h, x: cx, y: cy})
		rectOf[ft] = objSprite{X: cx, Y: cy, W: pw, H: h}
		cx += pw
		if h > shelfH {
			shelfH = h
		}
		if cy+shelfH > atlasH {
			atlasH = cy + shelfH
		}
	}
	if atlasH == 0 {
		return rectOf, nil // no sprites this world
	}
	img := image.NewPaletted(image.Rect(0, 0, objAtlasW, atlasH), pal)
	for _, p := range ps {
		drawFirstFrame(img, block, at, p.bm, p.w, p.h, p.x, p.y)
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return rectOf, png.Encode(f, img)
}

// firstFrame decodes the first BOB descriptor of a frame table: bitmap addr, height
// and data width in words (one less than BLTSIZE's, the cookie-cut extra word).
func firstFrame(block []byte, ft int, at func(int) int, blockHi int) (bm, h, dw int, ok bool) {
	if at(ft)+4 > len(block) || at(ft) < 0 {
		return 0, 0, 0, false
	}
	p := int(binary.BigEndian.Uint32(block[at(ft):]))
	if p < blockBase || at(p)+14 > len(block) {
		return 0, 0, 0, false
	}
	bm = int(binary.BigEndian.Uint32(block[at(p):]))
	bs := int(binary.BigEndian.Uint16(block[at(p)+0xA:]))
	h, w := bs>>6, bs&0x3F
	dw = w - 1
	if bm < blockBase || bm >= blockHi || dw < 1 || h < 1 || h > 96 || at(bm)+4*h*dw*2 > len(block) {
		return 0, 0, 0, false
	}
	return bm, h, dw, true
}

// drawFirstFrame decodes one 4-bitplane plane-major BOB into img at (ox,oy).
func drawFirstFrame(img *image.Paletted, block []byte, at func(int) int, bm, dw, h, ox, oy int) {
	bpr := dw * 2
	planeSize := h * bpr
	for y := 0; y < h; y++ {
		for x := 0; x < dw*16; x++ {
			var v uint8
			for p := 0; p < 4; p++ {
				a := at(bm) + p*planeSize + y*bpr + x/8
				if a >= 0 && a < len(block) && block[a]&(0x80>>(x%8)) != 0 {
					v |= 1 << uint(p)
				}
			}
			if v != 0 { // colour 0 transparent
				img.SetColorIndex(ox+x, oy+y, v)
			}
		}
	}
}

func writeAtlas(path string, block []byte, tableOff, nTiles int, pal color.Palette) error {
	rows := (nTiles + atlasCols - 1) / atlasCols
	img := image.NewPaletted(image.Rect(0, 0, atlasCols*tileSide, rows*tileSide), pal)
	for n := 0; n < nTiles; n++ {
		off := tableOff + int(binary.BigEndian.Uint32(block[tableOff+n*4:]))
		if off+tileBytes > len(block) {
			break
		}
		ox, oy := (n%atlasCols)*tileSide, (n/atlasCols)*tileSide
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
				img.SetColorIndex(ox+x, oy+y, v)
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

// objPalette is the world palette with colour 0 transparent (sprites cookie-cut it).
func objPalette(block []byte, off int) color.Palette {
	pal := readPalette(block, off)
	pal[0] = color.RGBA{0, 0, 0, 0}
	return pal
}

func writeJSON(path string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func mainBlob(adf []byte) []byte {
	const off = 0x2C00
	n := int(binary.BigEndian.Uint32(adf[off:]))
	return adf[off : off+n]
}
