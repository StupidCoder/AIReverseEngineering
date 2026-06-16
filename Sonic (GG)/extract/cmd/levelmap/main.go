// levelmap renders every level in the game, reconstructed ENTIRELY from the cartridge.
// It reads the level-resource descriptor table (bank 5 $5600), and for each of the 18
// acts (6 zones x 3) everything comes from ROM:
//
//   - the block-index map: decomp.LoadMapRLE (the $0A73 codec) -> a (4096/stride) x stride
//     grid of block indices (stride from the descriptor; small stride = a vertical level);
//   - the tiles: each block index -> a 4x4 grid of tiles via the block tile table (file
//     $10000 + word), each tile being one of 256 patterns decompressed by decomp.Decompress
//     (the $0406 codec) from the tile set at file $30000 + word;
//   - the palette: the BG palette index (descriptor +29) resolved through the bank-8 $7400
//     offset table to 16 colours (romPalette), exactly as load_palette $0586 does.
//
// The render is cropped to the level width (descriptor right bound $D26F / 32) x the full
// grid height; columns past the width are off-level storage padding (no content trimming).
//
// To stay honest it ALSO boots the Game Gear machine model into each distinct (tile set,
// palette) combination and checks the from-ROM tiles + palette against the loader's live
// VRAM/CRAM: they match except for a handful of animated water tiles and a 3-colour
// palette cycle, which a static frame cannot represent.
//
// Output: rendered/level_<zone>_act<N>.png (full) + _overview.png (1/4) for every act.
//
// Usage: levelmap <rom.gg> <outdir>
package main

import (
	"bytes"
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
	blockBase = 0x10000 // block tile tables live at file $10000 + descriptor word (banks 4/5)
	tileBase  = 0x30000 // compressed tile sets live at file $30000 + descriptor word (banks $0C-$0F)
	palTable  = 0x23400 // bank 8 $7400: per-index palette offset table ($0586/$05C4)
)

// zoneNames are the six zones of Sonic 1 (8-bit), in act order.
var zoneNames = []string{"greenhills", "bridge", "jungle", "labyrinth", "scrapbrain", "skybase"}

// Act is one level, with everything needed to render it located statically in the ROM.
type Act struct {
	num      int    // 0..17 (the value forced into $D238)
	name     string // <zone>_act<N>
	mapFile  int    // file offset of the compressed block-index map
	mapLen   int    // compressed length
	widthBlk int    // played width in blocks (right scroll bound $D26F / 32)
	stride   int    // map row stride = columns (map height = 4096/stride)
	blkTable int    // file offset of the 16-byte-per-block tile table
	tileFile int    // file offset of the compressed tile set (block tiles index into it)
	bgPal    int    // BG palette index ($D22C, descriptor +29)
}

// parseActs reads the descriptor table and decodes all 18 acts. The map address, block-
// table word and tile-set word are bank-relative; because the source windows are
// contiguous in the file, the file offset is the fixed bank base plus the word.
func parseActs(rom []byte) []Act {
	w := func(o int) int { return int(rom[o]) | int(rom[o+1])<<8 }
	var acts []Act
	for i := 0; i < 18; i++ {
		d := descTable + w(descTable+i*2)
		acts = append(acts, Act{
			num:      i,
			name:     fmt.Sprintf("%s_act%d", zoneNames[i/3], i%3+1),
			mapFile:  0x14000 + w(d+15),   // map address is offset-from-$14000
			mapLen:   w(d + 17),           // compressed length (BC)
			widthBlk: w(d+7) / 32,         // right scroll bound $D26F / 32 px-per-block
			stride:   w(d + 1),            // map stride (256/128/64/.. = number of columns)
			blkTable: blockBase + w(d+19), // block tile table = $10000 + word
			tileFile: tileBase + w(d+21),  // compressed tile set = $30000 + word
			bgPal:    int(rom[d+29]),      // BG palette index -> $D22C (load_palette $0586)
		})
	}
	return acts
}

// animFrame describes a runtime-animated tile group: the first tile slot it occupies, the
// ROM file offset of a frame's data, and how many 32-byte tiles it covers. These slots are
// EMPTY in the base tile set ($0406); the game's animation update ($15FF) copies a frame
// into them each cycle. Loading one frame statically makes them visible (rings, water, ...).
type animFrame struct{ vramTile, fileOff, nTiles int }

// ringAnim is the rings/flowers animation (tiles 252-255), shared by every zone (the
// animation update copies its frame from a fixed bank-11 source for all of them).
var ringAnim = animFrame{252, 0x2F73D, 4}

// zoneAnims is the per-zone animation data, traced from the $15FF update (sources in bank
// 11). The update is hardcoded per zone. Green Hills (0) additionally animates water; the
// other zones' extra animations (only the rings run during the idle intro probed so far)
// are not yet captured.
var zoneAnims = map[int][]animFrame{
	0: {{12, 0x2FA3D, 4}}, // Green Hills water (tiles 12-15)
}

// applyAnimFrame copies one frame of each animated tile group into the tile set, replacing
// the blank base slots so the rings/flowers/water show up in the still render.
func applyAnimFrame(rom, tiles []byte, zone int) {
	apply := func(a animFrame) {
		n := a.nTiles * 32
		if a.fileOff+n <= len(rom) && a.vramTile*32+n <= len(tiles) {
			copy(tiles[a.vramTile*32:], rom[a.fileOff:a.fileOff+n])
		}
	}
	apply(ringAnim)
	for _, a := range zoneAnims[zone] {
		apply(a)
	}
}

// romPalette resolves a BG palette index to its 16 colours, read straight from ROM the
// way load_palette ($0586/$05C4) does: a per-index offset table at bank 8 $7400 gives the
// offset of the 32-byte (16-colour) palette within that same bank.
func romPalette(rom []byte, idx int) color.Palette {
	off := int(rom[palTable+idx*2]) | int(rom[palTable+idx*2+1])<<8
	p := palTable + off
	return gamegear.Palette(rom[p : p+32])
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: levelmap <rom.gg> <outdir>")
		os.Exit(2)
	}
	rom, err := os.ReadFile(os.Args[1])
	chk(err)
	outdir := os.Args[2]
	chk(os.MkdirAll(outdir, 0o755))

	acts := parseActs(rom)
	// Validation cache: the real loader's VRAM tiles + CRAM palette, loaded via the oracle
	// once per distinct (tile set, palette) combination and checked against the from-ROM
	// data below. Keying on both matters because Sky Base reuses one tile set across acts
	// with different palettes.
	type live struct {
		vram []byte
		cram color.Palette
	}
	type key struct{ tile, pal int }
	vcache := map[key]live{}
	for _, a := range acts {
		// --- everything from ROM ---
		mp := decomp.LoadMapRLE(rom, a.mapFile, a.mapLen) // block-index map
		tiles := decomp.Decompress(rom, a.tileFile)       // 256 BG tiles ($0406 codec)
		pal := romPalette(rom, a.bgPal)                   // 16 BG colours
		applyAnimFrame(rom, tiles, a.num/3)               // fill the runtime-animated tile slots

		// --- validate against the oracle (load each distinct (tile set, palette) once) ---
		k := key{a.tileFile, a.bgPal}
		v, ok := vcache[k]
		if !ok {
			vt, vp := loadLevel(rom, a.num)
			v = live{vt, vp}
			vcache[k] = v
		}
		// The static decode is frame 0; the oracle has run animation (cycling water tiles +
		// a rotating palette), so count matches rather than demand byte-equality. A high
		// match (only a handful of animated tiles/colours differ) means the decode is right.
		tilesSame := matchTiles(tiles, v.vram)
		palSame := matchColors(pal, v.cram, 16)

		// --- render, all from ROM. The map is a (4096/stride) x stride grid; small stride =
		// a tall vertical level. block(row,col) = map[row*stride + col]. $D26F is the max
		// camera position (the screen's top-left), so the level is visible up to one screen
		// (160 px = 5 blocks) PAST it; the level width is therefore $D26F/32 + 5 columns. The
		// full grid height (4096/stride) already spans the camera's vertical range + a screen. ---
		const screenBlk = 5 // GG visible width 160 px / 32 px-per-block
		cols := clampi(a.widthBlk+screenBlk, 1, a.stride)
		rows := 4096 / a.stride
		img := renderMap(rom, mp, tiles, pal, a.stride, cols, rows, a.blkTable)
		writePNG(filepath.Join(outdir, "level_"+a.name+".png"), img)
		writePNG(filepath.Join(outdir, "level_"+a.name+"_overview.png"), downscale(img, 4))
		fmt.Printf("%-18s %3dx%-3d blocks (%dx%d px)  validate vs oracle: tiles %d/256, palette %d/16 match (rest = animation)\n",
			a.name, cols, rows, cols*32, rows*32, tilesSame, palSame)
	}
}

// matchTiles counts how many of the 256 8x8 tiles (32 bytes each) in the from-ROM set are
// byte-identical to the oracle's VRAM.
func matchTiles(rom, vram []byte) int {
	n := 0
	for t := 0; t*32+32 <= len(rom) && t*32+32 <= len(vram); t++ {
		if bytes.Equal(rom[t*32:t*32+32], vram[t*32:t*32+32]) {
			n++
		}
	}
	return n
}

// matchColors counts how many of the first n palette colours match.
func matchColors(a, b color.Palette, n int) int {
	m := 0
	for i := 0; i < n && i < len(a) && i < len(b); i++ {
		if a[i] == b[i] {
			m++
		}
	}
	return m
}

// loadLevel boots the machine into act `num` (forcing $D238 through the level load) and
// returns the VRAM tile data and CRAM palette the real loader produced — used only to
// validate the from-ROM graphics. The tiles slice is a copy.
func loadLevel(rom []byte, num int) ([]byte, color.Palette) {
	m := gamegear.NewMachine(rom)
	captured := func() bool { return m.Captured }
	m.CapturePC = 0x0A73
	m.CapLo, m.CapHi, m.CapOutBase = 0x0A73, 0x0AA2, 0xC000 // snapshot $C000 at the RET
	for i := 0; i < 700; i++ {
		m.RunFrame()
	}
	// Press Start and force the act number every frame until the map decompresses.
	for round := 0; round < 40 && !captured(); round++ {
		m.Pad00 = 0x7F
		m.Write(0xD238, byte(num))
		for i := 0; i < 8; i++ {
			m.RunFrame()
			m.Write(0xD238, byte(num))
		}
		m.Pad00 = 0xFF
		for k := 0; k < 242 && !captured(); k++ {
			m.Write(0xD238, byte(num))
			m.RunFrame()
		}
	}
	// Let the level come up so the palette fades in — but DO NOT scroll (no Right+Jump):
	// scrolling runs the tile streamer, which overwrites VRAM and breaks the byte-for-byte
	// comparison against the static tile set. The level intro holds still, so VRAM stays as
	// the freshly decompressed tiles while the palette fade completes.
	for i := 0; i < 200; i++ {
		m.RunFrame()
	}
	tiles := make([]byte, len(m.VDP.VRAM))
	copy(tiles, m.VDP.VRAM[:])
	return tiles, gamegear.Palette(m.VDP.CRAM[:])
}

// renderMap paints a decoded block-index map into a full-resolution image. The 4096-byte
// map is a (4096/stride) x stride grid: block(row,col) = romMap[row*stride + col]. The
// drawn extent is cols (the level width) x rows (the full grid height); columns beyond the
// width are off-level storage padding, cropped here. Each block index expands to a 4x4
// grid of 8x8 tiles via the block tile table: tile(r,c) = rom[blkTable + idx*16 + r*4 + c].
func renderMap(rom, romMap, tiles []byte, pal color.Palette, stride, cols, rows, blkTable int) *image.Paletted {
	img := image.NewPaletted(image.Rect(0, 0, cols*4*8, rows*4*8), pal)
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			def := blkTable + int(romMap[row*stride+col])*16
			for r := 0; r < 4; r++ {
				for c := 0; c < 4; c++ {
					t := gamegear.DecodeTile(tiles[int(rom[def+r*4+c])*32:])
					ox, oy := (col*4+c)*8, (row*4+r)*8
					for y := 0; y < 8; y++ {
						for x := 0; x < 8; x++ {
							img.SetColorIndex(ox+x, oy+y, t[y][x])
						}
					}
				}
			}
		}
	}
	return img
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

// downscale box-averages a paletted image by integer factor n into an RGBA image.
func downscale(src *image.Paletted, n int) *image.RGBA {
	b := src.Bounds()
	ow, oh := b.Dx()/n, b.Dy()/n
	dst := image.NewRGBA(image.Rect(0, 0, ow, oh))
	for oy := 0; oy < oh; oy++ {
		for ox := 0; ox < ow; ox++ {
			var rs, gs, bs uint32
			for dy := 0; dy < n; dy++ {
				for dx := 0; dx < n; dx++ {
					r, g, bl, _ := src.At(b.Min.X+ox*n+dx, b.Min.Y+oy*n+dy).RGBA()
					rs += r >> 8
					gs += g >> 8
					bs += bl >> 8
				}
			}
			d := uint32(n * n)
			dst.Set(ox, oy, color.RGBA{uint8(rs / d), uint8(gs / d), uint8(bs / d), 0xFF})
		}
	}
	return dst
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
