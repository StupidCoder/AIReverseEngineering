// webexport serializes the decoded Sonic levels for the static web viewer (see
// site/PLAN.md). Everything is reconstructed from the cartridge by the same decode
// path as cmd/levelmap, then written as small block-indexed JSON plus a per-(tileset,
// palette) tile-atlas PNG:
//
//	meta.json            zone names, the 18-act index, animation cadence
//	shapes.json          the 48 collision height profiles ($3E7A) + angles ($3978)
//	atlas_<k>.png        256 tiles (8x8) at a zone palette, 16 wide; reused across acts
//	act<NN>.json         per act: block map, block->tile table, block->collision shape,
//	                     palette, spawn, objects, atlas reference
//
// The client expands blocks -> tiles into a tilemap. Output defaults to
// site/public/sonic (relative to the repo root).
//
// Usage: webexport <rom.gg> [outdir]
package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"sonicgg/extract/decomp"

	"stupidcoder.com/tools/gamegear"
)

const (
	descTable = 0x15600 // bank 5 $5600: 18 word-pointers -> per-act descriptors
	blockBase = 0x10000 // block tile tables: file $10000 + descriptor word
	tileBase  = 0x30000 // compressed tile sets: file $30000 + descriptor word
	palTable  = 0x23400 // bank 8 $7400: per-index palette offset table
	objTable  = 0x15600 // object tables: $15600 + descriptor word +30
	attrPtrs  = 0x343D  // per-zone block-attribute table pointers (bit7 priority, bits0-5 shape)
	shapeTbl  = 0x3E7A  // floor height-profile pointer table (shape*2 -> 32-byte profile)
	angleTbl  = 0x3978  // per-shape surface angle (signed)
	numShapes = 48      // collision shapes $00-$2F (the table ends where profile data begins)
)

var zoneNames = []string{"Green Hills", "Bridge", "Jungle", "Labyrinth", "Scrap Brain", "Sky Base", "Special Stage"}

// Bonus/special stages: descriptor indices 28-35 ($1C-$23), reached when an act is
// cleared with >=50 rings (goal handler $61F8 -> $D282=4 -> sets bonus flag IY+7 bit0;
// $D239 counts from $1C and selects the next descriptor each time). All 8 share one zone-6
// tilemap/tileset/palette; they differ in spawn, camera bounds and object (ring) layout.
const bonusFirst, bonusCount = 28, 8

// objNames maps the object type bytes we have identified (Part V §1) to display names.
var objNames = map[byte]string{
	0x00: "Sonic", 0x01: "bonus", 0x02: "bonus", 0x03: "bonus", 0x04: "shield",
	0x06: "emerald", 0x07: "goal", 0x08: "crab", 0x09: "swing platform",
	0x0E: "bird", 0x0F: "moving platform", 0x10: "beetle", 0x12: "world 1 boss",
	0x25: "capsule", 0x26: "fish", 0x2C: "world 3 boss", 0x2D: "porcupine",
	0x48: "world 2 boss", 0x49: "world 4 boss", 0x4E: "seesaw",
	0x50: "scroll lock", 0x51: "checkpoint", 0x13: "teleporter",
	0x21: "bumper", 0x52: "continue", // special-stage objects ($52 = Continue powerup)
	// (rings are not objects: they are baked into the block map as $79-$7B, like normal zones)
}

type Act struct {
	num      int
	zone     int
	name     string
	mapFile  int
	mapLen   int
	widthBlk int
	stride   int
	blkTable int
	tileFile int
	bgPal    int
	spawnX   int // descriptor +13
	spawnY   int // descriptor +14 - 1 (verified against the oracle's pristine spawn)
}

func w(rom []byte, o int) int { return int(rom[o]) | int(rom[o+1])<<8 }

func parseActs(rom []byte) []Act {
	var acts []Act
	for i := 0; i < 18; i++ {
		d := descTable + w(rom, descTable+i*2)
		acts = append(acts, Act{
			num: i, zone: i / 3,
			name:     fmt.Sprintf("%s Act %d", zoneNames[i/3], i%3+1),
			mapFile:  0x14000 + w(rom, d+15),
			mapLen:   w(rom, d+17),
			widthBlk: w(rom, d+7) / 32,
			stride:   w(rom, d+1),
			blkTable: blockBase + w(rom, d+19),
			tileFile: tileBase + w(rom, d+21),
			bgPal:    int(rom[d+29]),
			spawnX:   int(rom[d+13]),
			spawnY:   int(rom[d+14]) - 1,
		})
	}
	// Bonus/special stages (zone 6): same descriptor layout, one shared map.
	for n := 0; n < bonusCount; n++ {
		i := bonusFirst + n
		d := descTable + w(rom, descTable+i*2)
		acts = append(acts, Act{
			num: i, zone: 6,
			name:     fmt.Sprintf("Special Stage %d", n+1),
			mapFile:  0x14000 + w(rom, d+15),
			mapLen:   w(rom, d+17),
			widthBlk: w(rom, d+7) / 32,
			stride:   w(rom, d+1),
			blkTable: blockBase + w(rom, d+19),
			tileFile: tileBase + w(rom, d+21),
			bgPal:    int(rom[d+29]),
			spawnX:   int(rom[d+13]),
			spawnY:   int(rom[d+14]) - 1,
		})
	}
	return acts
}

func romPalette(rom []byte, idx int) color.Palette {
	off := w(rom, palTable+idx*2)
	return gamegear.Palette(rom[palTable+off : palTable+off+32])
}

// objectTable reads a level's [type, blockX, blockY] placements straight from ROM.
type Obj struct {
	Type byte   `json:"type"`
	Bx   int    `json:"bx"`
	By   int    `json:"by"`
	Name string `json:"name"`
}

func objectTable(rom []byte, act int) []Obj {
	d := descTable + w(rom, descTable+act*2)
	t := objTable + w(rom, d+30)
	count := int(rom[t])
	objs := make([]Obj, 0, count)
	for i := 0; i < count; i++ {
		p := t + 1 + i*3
		typ := rom[p]
		objs = append(objs, Obj{typ, int(rom[p+1]), int(rom[p+2]), objNames[typ]})
	}
	return objs
}

// blockShapes returns block index -> collision shape (0-47) for a zone, from the $343D
// per-zone attribute table (low 6 bits of each block's attribute byte). Home-config
// addresses map 1:1 to file offsets, so the pointer is read directly.
func blockShapes(rom []byte, zone int) []int {
	p := w(rom, attrPtrs+zone*2)
	out := make([]int, 256)
	for b := 0; b < 256; b++ {
		out[b] = int(rom[p+b]) & 0x3F
	}
	return out
}

// animFrame is a runtime-animated tile group (see cmd/levelmap).
type animFrame struct{ vramTile, fileOff, nTiles int }

var ringAnim = animFrame{252, 0x2F73D, 4}
var zoneAnims = map[int][]animFrame{0: {{12, 0x2FA3D, 4}}}

func applyAnimFrame(rom, tiles []byte, zone int) {
	apply := func(a animFrame) {
		n := a.nTiles * 32
		if a.fileOff+n <= len(rom) && a.vramTile*32+n <= len(tiles) {
			copy(tiles[a.vramTile*32:], rom[a.fileOff:a.fileOff+n])
		}
	}
	// Tiles 252-255 are empty in every zone's base set (including the special stage); the
	// engine fills them with the spinning ring frames at runtime. The special stage's
	// rectangular ring fields are blocks $79-$7B (= those tiles), so it needs this too.
	apply(ringAnim)
	for _, a := range zoneAnims[zone] {
		apply(a)
	}
}

// Animated tile sources (bank 11, contiguous): the rings spin through 6 frames, the
// Green Hills flowers through 2, each frame being 4 tiles (16x16). Frame 0 is the one
// already baked into the base tile set; frames 1+ are appended to the atlas.
const (
	ringSrc      = 0x2F73D
	ringFrames   = 6
	flowerSrc    = 0x2FA3D
	flowerFrames = 2
)

// AnimGroup describes one animated tile group: per frame, the 4 atlas tile indices that
// replace the group's base tiles (frame 0 = the base tiles themselves).
type AnimGroup struct {
	Name   string  `json:"name"`
	Frames [][]int `json:"frames"`
}

func drawTile(img *image.RGBA, pal color.Palette, data []byte, atlasIdx int) {
	t := gamegear.DecodeTile(data)
	ox, oy := (atlasIdx%16)*8, (atlasIdx/16)*8
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(ox+x, oy+y, pal[t[y][x]])
		}
	}
}

// renderAtlas paints the 256 base tiles into a 16-wide RGBA atlas, then appends the
// animation frames (frames 1+) below them, returning the per-group frame->atlas-index
// metadata. Atlas height is 18 rows (the appended frames fit in rows 16-17).
func renderAtlas(rom, tiles []byte, pal color.Palette, zone int) (*image.RGBA, []AnimGroup) {
	const cols, rows = 16, 18
	img := image.NewRGBA(image.Rect(0, 0, cols*8, rows*8))
	for ti := 0; ti < 256; ti++ {
		drawTile(img, pal, tiles[ti*32:], ti)
	}
	next := 256
	appendGroup := func(name string, base []int, src, nframes int) AnimGroup {
		g := AnimGroup{Name: name, Frames: [][]int{base}} // frame 0 = base tiles
		for f := 1; f < nframes; f++ {
			idx := []int{next, next + 1, next + 2, next + 3}
			for i := 0; i < 4; i++ {
				drawTile(img, pal, rom[src+f*128+i*32:], idx[i])
			}
			g.Frames = append(g.Frames, idx)
			next += 4
		}
		return g
	}
	groups := []AnimGroup{appendGroup("rings", []int{252, 253, 254, 255}, ringSrc, ringFrames)}
	if zone == 0 {
		groups = append(groups, appendGroup("flowers", []int{12, 13, 14, 15}, flowerSrc, flowerFrames))
	}
	return img, groups
}

// PaletteCycle is a runtime BG-palette rotation (e.g. the Green Hills water/waterfall,
// which cycles colours 10-12 through three blues every ~10 frames — a palette effect, not
// a tile animation). Steps[0] is aligned to the static palette (= the atlas colours).
type PaletteCycle struct {
	Slots        []int      `json:"slots"`        // BG palette slots that cycle
	Steps        [][]string `json:"steps"`        // per step: hex colour for each slot
	PeriodFrames int        `json:"periodFrames"` // game frames per step
	Tiles        []int      `json:"tiles"`        // tile indices that actually USE a cycling slot
}

// cyclingTiles lists the tile indices (0-255) whose pixels use one of the cycling palette
// slots. The cycle must be limited to these — other tiles may share a colour with the
// cycling slots at rest (e.g. sky in Bridge) but use a different, static palette index.
func cyclingTiles(tiles []byte, slots []int) []int {
	slot := map[int]bool{}
	for _, s := range slots {
		slot[s] = true
	}
	var out []int
	for t := 0; t < 256; t++ {
		px := gamegear.DecodeTile(tiles[t*32:])
		found := false
		for y := 0; y < 8 && !found; y++ {
			for x := 0; x < 8; x++ {
				if slot[int(px[y][x])] {
					found = true
					break
				}
			}
		}
		if found {
			out = append(out, t)
		}
	}
	return out
}

// capturePaletteCycle boots the act and watches CRAM for a BG-palette cycle, returning the
// cycling slots + per-step colours (step 0 = the static palette) + period, or nil. It is
// the one oracle-assisted part of the export (the cycle is driven at runtime).
func capturePaletteCycle(rom []byte, act int, staticPal color.Palette) *PaletteCycle {
	m := gamegear.NewMachine(rom)
	m.CapturePC = 0x0A73
	for i := 0; i < 700; i++ {
		m.RunFrame()
	}
	for r := 0; r < 40 && !m.Captured; r++ {
		m.Pad00 = 0x7F
		m.Write(0xD238, byte(act))
		for i := 0; i < 8; i++ {
			m.RunFrame()
			m.Write(0xD238, byte(act))
		}
		m.Pad00 = 0xFF
		for k := 0; k < 242 && !m.Captured; k++ {
			m.Write(0xD238, byte(act))
			m.RunFrame()
		}
	}
	for i := 0; i < 200; i++ { // let the palette fade finish
		m.RunFrame()
	}
	const N = 120
	rec := make([][]string, N)
	for f := 0; f < N; f++ {
		m.RunFrame()
		rec[f] = paletteHex(gamegear.Palette(m.VDP.CRAM[:32]))
	}
	eq := func(a, b []string) bool {
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}
	// Frames where the palette changes; the period is the gap BETWEEN changes (the first
	// change is just the remainder of the step the recording started in).
	var changes []int
	for f := 1; f < N; f++ {
		if !eq(rec[f], rec[f-1]) {
			changes = append(changes, f)
		}
	}
	if len(changes) < 2 {
		return nil
	}
	P := changes[1] - changes[0]
	// The distinct states in cycle order: the state at the start, then after each change.
	runVals := [][]string{rec[0]}
	for _, cf := range changes {
		runVals = append(runVals, rec[cf])
	}
	L := 0
	for cand := 1; cand <= len(runVals)/2; cand++ {
		ok := true
		for i := 0; i+cand < len(runVals); i++ {
			if !eq(runVals[i], runVals[i+cand]) {
				ok = false
				break
			}
		}
		if ok {
			L = cand
			break
		}
	}
	if L < 2 {
		return nil
	}
	steps := runVals[:L]
	var slots []int
	for i := 0; i < 16; i++ {
		for k := 1; k < len(steps); k++ {
			if steps[k][i] != steps[0][i] {
				slots = append(slots, i)
				break
			}
		}
	}
	if len(slots) == 0 {
		return nil
	}
	sp := paletteHex(staticPal)
	rot := 0
	for k := 0; k < len(steps); k++ {
		match := true
		for _, s := range slots {
			if steps[k][s] != sp[s] {
				match = false
				break
			}
		}
		if match {
			rot = k
			break
		}
	}
	pc := &PaletteCycle{Slots: slots, PeriodFrames: P}
	for k := 0; k < len(steps); k++ {
		st := steps[(rot+k)%len(steps)]
		row := make([]string, len(slots))
		for j, s := range slots {
			row[j] = st[s]
		}
		pc.Steps = append(pc.Steps, row)
	}
	return pc
}

// JSON shapes ----------------------------------------------------------------

type Meta struct {
	Zones []string   `json:"zones"`
	Acts  []ActIndex `json:"acts"`
	Anim  AnimMeta   `json:"anim"`
}
type ActIndex struct {
	File  string `json:"file"`
	Zone  int    `json:"zone"`
	Name  string `json:"name"`
	Atlas string `json:"atlas"`
}
type AnimMeta struct {
	FramesPerTick int `json:"framesPerTick"`
}

type Shapes struct {
	Count    int     `json:"count"`
	Profiles [][]int `json:"profiles"` // 48 x 32 signed heights; -128 = no surface
	Angles   []int   `json:"angles"`   // 48 signed angles
}

type ActFile struct {
	Zone         int           `json:"zone"`
	Act          int           `json:"act"`
	Name         string        `json:"name"`
	Atlas        string        `json:"atlas"`
	TileSize     int           `json:"tileSize"`
	Stride       int           `json:"stride"`
	WidthBlocks  int           `json:"widthBlocks"`
	HeightBlocks int           `json:"heightBlocks"`
	Palette      []string      `json:"palette"`
	BlockTiles   [][]int       `json:"blockTiles"` // block -> 16 tile indices (4x4)
	BlockShape   []int         `json:"blockShape"` // block -> collision shape 0-47
	Blocks       []int         `json:"blocks"`     // block-index map, row-major, len = w*h
	Spawn        [2]int        `json:"spawn"`      // [bx, by]
	Objects      []Obj         `json:"objects"`
	Anim         []AnimGroup   `json:"anim"`                   // animated tile groups (atlas indices per frame)
	PaletteCycle *PaletteCycle `json:"paletteCycle,omitempty"` // runtime BG-palette rotation (water/waterfall)
	Water        *Water        `json:"water,omitempty"`        // Labyrinth flooded-act underwater split (Part V §3)
	Music        string        `json:"music,omitempty"`        // background-music track name (descriptor +36 -> id)
}

// musicTrack returns the act's background-music track name. The music id is descriptor byte
// +36 (the loader $1A66 stores it to $D2F7 and plays it via RST $18); the id indexes the
// $4716 song table. Acts map to ids 0-5 by zone, with two Sky Base acts reusing Scrap Brain's
// theme and the special stage on id 16 (Part VI). cmd/musicrom bakes these by the same names.
func musicTrack(rom []byte, act int) string {
	d := descTable + w(rom, descTable+act*2)
	names := map[byte]string{0: "greenhills", 1: "bridge", 2: "jungle", 3: "labyrinth",
		4: "scrapbrain", 5: "skybase", 16: "special"}
	return names[rom[d+36]]
}

// Water describes a Labyrinth act's underwater split. The engine raster-swaps the BG
// palette at the water-line scanline (IRQ line interrupt, palette from bank 0 $0216) and
// runs slower physics below it. For the viewer: blocks whose top is at or below LineY use
// the static underwater palette (no cycle); above it is the normal surface palette + cycle.
type Water struct {
	LineY   int      `json:"lineY"`   // water surface world-Y (px); >= this is underwater
	Palette []string `json:"palette"` // 16 underwater BG colours, static (bank 0 $0216)
}

// underwaterPalette returns the 16 static underwater BG colours the IRQ line-split writes
// to CRAM from the bank-0 table at file $0216 (Part V §3).
func underwaterPalette(rom []byte) []string {
	pal := gamegear.Palette(rom[0x0216 : 0x0216+32])
	return paletteHex(pal)
}

// waterLine returns the water surface world-Y (px) for a Labyrinth act, or -1 if the act
// has no water. The water surface is object type $40 (the first object placed); its block-Y
// times 32 is the surface line (loader $185D arms the split only for acts 9-11).
func waterLine(rom []byte, act int) int {
	if act < 9 || act > 11 {
		return -1
	}
	for _, o := range objectTable(rom, act) {
		if o.Type == 0x40 {
			return o.By * 32
		}
	}
	return -1
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: webexport <rom.gg> [outdir]")
		os.Exit(2)
	}
	rom, err := os.ReadFile(os.Args[1])
	chk(err)
	outdir := "site/public/sonic"
	if len(os.Args) > 2 {
		outdir = os.Args[2]
	}
	chk(os.MkdirAll(outdir, 0o755))

	// shapes.json (global)
	sh := Shapes{Count: numShapes}
	for s := 0; s < numShapes; s++ {
		p := w(rom, shapeTbl+s*2)
		prof := make([]int, 32)
		for c := 0; c < 32; c++ {
			prof[c] = int(int8(rom[p+c])) // signed; -128 ($80) = no surface
		}
		sh.Profiles = append(sh.Profiles, prof)
		sh.Angles = append(sh.Angles, int(int8(rom[angleTbl+s])))
	}
	writeJSON(filepath.Join(outdir, "shapes.json"), sh)

	acts := parseActs(rom)
	// Dedup atlases by (tileFile, bgPal): same tiles + palette -> one PNG.
	atlasName := map[[2]int]string{}
	atlasAnim := map[string][]AnimGroup{}
	cycleCache := map[int]*PaletteCycle{} // by bgPal: the BG-palette cycle (oracle-captured)
	meta := Meta{Zones: zoneNames, Anim: AnimMeta{FramesPerTick: 10}}

	const screenBlk = 5
	for _, a := range acts {
		tiles := decomp.Decompress(rom, a.tileFile)
		applyAnimFrame(rom, tiles, a.zone)
		pal := romPalette(rom, a.bgPal)

		// atlas (deduped)
		key := [2]int{a.tileFile, a.bgPal}
		atlas, ok := atlasName[key]
		if !ok {
			atlas = fmt.Sprintf("atlas_%d.png", len(atlasName))
			atlasName[key] = atlas
			img, groups := renderAtlas(rom, tiles, pal, a.zone)
			atlasAnim[atlas] = groups
			writePNG(filepath.Join(outdir, atlas), img)
		}

		// map + geometry
		mp := decomp.LoadMapRLE(rom, a.mapFile, a.mapLen)
		cols := clampi(a.widthBlk+screenBlk, 1, a.stride)
		rows := 4096 / a.stride
		blocks := make([]int, cols*rows)
		for r := 0; r < rows; r++ {
			for c := 0; c < cols; c++ {
				blocks[r*cols+c] = int(mp[r*a.stride+c])
			}
		}

		// block tile table (256 blocks x 16 tiles) + collision shapes
		bt := make([][]int, 256)
		for b := 0; b < 256; b++ {
			row := make([]int, 16)
			for i := 0; i < 16; i++ {
				row[i] = int(rom[a.blkTable+b*16+i])
			}
			bt[b] = row
		}

		// Sonic's spawn is not in the object table; the loader places him from the spawn
		// pointer. It is stored in the descriptor as (+13, +14-1) = (blockX, blockY),
		// verified block-exact against the oracle's pristine slot-0 capture (objprobe).
		spawn := [2]int{a.spawnX, a.spawnY}
		objs := objectTable(rom, a.num)

		actNo := a.num%3 + 1
		if a.zone == 6 {
			actNo = a.num - bonusFirst + 1 // Special Stage round number
		}
		af := ActFile{
			Zone: a.zone, Act: actNo, Name: a.name, Atlas: atlas,
			TileSize: 8, Stride: a.stride, WidthBlocks: cols, HeightBlocks: rows,
			Palette:    paletteHex(pal),
			BlockTiles: bt, BlockShape: blockShapes(rom, a.zone),
			Blocks: blocks, Spawn: spawn, Objects: objs,
			Anim: atlasAnim[atlas], Music: musicTrack(rom, a.num),
		}
		// The palette cycle is oracle-captured by booting the act; only do it for the real
		// zones (the bonus stages can't be reached by forcing $D238 and have no water/lava
		// cycle anyway).
		var pc *PaletteCycle
		if a.zone < 6 {
			var seen bool
			pc, seen = cycleCache[a.bgPal]
			if !seen {
				pc = capturePaletteCycle(rom, a.num, pal)
				cycleCache[a.bgPal] = pc
			}
		}
		if pc != nil {
			actPC := *pc // per-act copy: the cycling tiles depend on this act's tile set
			actPC.Tiles = cyclingTiles(tiles, pc.Slots)
			af.PaletteCycle = &actPC
		}
		if ly := waterLine(rom, a.num); ly >= 0 {
			af.Water = &Water{LineY: ly, Palette: underwaterPalette(rom)}
		}
		file := fmt.Sprintf("act%02d.json", a.num+1)
		writeJSON(filepath.Join(outdir, file), af)
		meta.Acts = append(meta.Acts, ActIndex{File: file, Zone: a.zone, Name: a.name, Atlas: atlas})
		fmt.Printf("%-16s %3dx%-3d blocks  atlas %s  %d objects\n", a.name, cols, rows, atlas, len(objs))
	}
	writeJSON(filepath.Join(outdir, "meta.json"), meta)
	fmt.Printf("wrote %d acts, %d atlases, shapes(%d) + meta to %s\n", len(acts), len(atlasName), numShapes, outdir)
}

func paletteHex(p color.Palette) []string {
	out := make([]string, len(p))
	for i, c := range p {
		r, g, b, _ := c.RGBA()
		out[i] = fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
	}
	return out
}

func clampi(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func writeJSON(path string, v any) {
	b, err := json.Marshal(v)
	chk(err)
	chk(os.WriteFile(path, b, 0o644))
}

func writePNG(path string, img image.Image) {
	f, err := os.Create(path)
	chk(err)
	defer f.Close()
	chk(png.Encode(f, img))
}

func chk(e error) {
	if e != nil {
		panic(e)
	}
}
