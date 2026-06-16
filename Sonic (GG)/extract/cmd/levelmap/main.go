// levelmap renders every level in the game as it appears, reconstructed from the
// cartridge. It reads the level-resource descriptor table (bank 5 $5600), and for each
// of the 18 acts (6 zones x 3) it:
//
//   - decodes the block-index map straight from ROM with decomp.LoadMapRLE (the $0A73
//     codec): a fixed 16-row x 256-column grid of block indices;
//   - expands each block to a 4x4 grid of 8x8 tiles via the zone's block tile table
//     (file $10000 + descriptor word), exactly as the $0760 terrain expander does;
//   - clips to the act's PLAYED width (the camera right-scroll bound $D26F / 32) — the
//     decompressed grid is always 256 wide but only the first columns are reachable.
//
// The tile graphics and palette differ per zone and are produced by the real loader, so
// for each act levelmap boots the Game Gear machine model into that exact level (forcing
// the act number $D238 during the load) and reads back VRAM tiles + CRAM palette. Act 0's
// from-ROM map decode is also asserted byte-for-byte against the live decompressor.
//
// Output: rendered/level_<zone>_act<N>.png (full) + _overview.png (1/4) for every act.
//
// Usage: levelmap <rom.gg> <outdir>
package main

import (
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
	mapCols   = 256
	mapRows   = 16
	blockBase = 0x10000 // block tile tables live at file $10000 + descriptor word (banks 4/5)
)

// zoneNames are the six zones of Sonic 1 (8-bit), in act order.
var zoneNames = []string{"greenhills", "bridge", "jungle", "labyrinth", "scrapbrain", "skybase"}

// Act is one level, with everything needed to render it located statically in the ROM.
type Act struct {
	num      int    // 0..17 (the value forced into $D238)
	name     string // <zone>_act<N>
	mapFile  int    // file offset of the compressed block-index map
	mapLen   int    // compressed length
	widthBlk int    // played width in blocks (right scroll bound / 32)
	stride   int    // map row stride = columns (level height = 4096/stride)
	blkTable int    // file offset of the 16-byte-per-block tile table
	tileset  int    // descriptor tile-set pointer (acts that share it share graphics)
}

// parseActs reads the descriptor table and decodes all 18 acts. The map address and the
// block-table word are bank-relative; because the source windows (banks 5/6/7 for maps,
// banks 4/5 for block tables) are contiguous in the file, the file offset is simply the
// fixed bank base plus the word.
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
			tileset:  w(d + 21),           // tile-set pointer (shared graphics key)
		})
	}
	return acts
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
	type gfx struct {
		tiles []byte
		pal   color.Palette
	}
	cache := map[int]gfx{} // tile set pointer -> loaded graphics (load each tile set once)
	for _, a := range acts {
		g, ok := cache[a.tileset]
		if !ok {
			loadNum := a.num // load the first act that uses this tile set
			for _, b := range acts {
				if b.tileset == a.tileset {
					loadNum = b.num
					break
				}
			}
			tiles, pal := loadLevel(rom, loadNum)
			g = gfx{tiles, pal}
			cache[a.tileset] = g
		}
		mp := decomp.LoadMapRLE(rom, a.mapFile, a.mapLen)
		// The map reshapes per act into a (4096/stride) x stride grid; stride is the number
		// of COLUMNS, so a small stride is a tall, narrow (vertical) level. block(row,col) =
		// map[row*stride + col]. Width = the played width (descriptor right bound / 32, which
		// clips off-level horizontal filler); height = trim trailing all-sky rows.
		cols := clampi(a.widthBlk, 1, a.stride)
		r0, r1 := contentBand(mp, a.stride, cols)
		img := renderMap(rom, mp, g.tiles, g.pal, a.stride, cols, r0, r1, a.blkTable)
		writePNG(filepath.Join(outdir, "level_"+a.name+".png"), img)
		writePNG(filepath.Join(outdir, "level_"+a.name+"_overview.png"), downscale(img, 4))
		fmt.Printf("%-18s stride=%3d grid %3dx%-3d -> %3d x %3d blocks  (%dx%d px)\n",
			a.name, a.stride, 4096/a.stride, a.stride, cols, r1-r0+1, cols*32, (r1-r0+1)*32)
	}
}

// loadLevel boots the machine into act `num` (forcing $D238 through the level load) and
// returns the VRAM tile data and CRAM palette the real loader produced. Only the FIRST
// act of each tile set is loaded — injecting a later act of a zone does not bring its
// graphics up cleanly, but acts in a zone share the tile set, so the first act's graphics
// are reused. The tiles slice is a copy.
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
	// Let the level come up so the tile set streams in and the palette uploads.
	m.PadDC = 0xE7
	for i := 0; i < 150; i++ {
		m.RunFrame()
	}
	tiles := make([]byte, len(m.VDP.VRAM))
	copy(tiles, m.VDP.VRAM[:])
	return tiles, gamegear.Palette(m.VDP.CRAM[:])
}

// contentBand returns the row range [first,last] to draw. The grid often has all-sky
// margins above and/or below the actual level (a vertical level fills only its lower
// rows); trim them, keeping a 2-row sky frame. Horizontal levels that already start near
// row 0 are left unchanged.
func contentBand(romMap []byte, stride, cols int) (int, int) {
	gridRows := 4096 / stride
	first, last := -1, 0
	for row := 0; row < gridRows; row++ {
		for col := 0; col < cols; col++ {
			if romMap[row*stride+col] != 0 {
				if first < 0 {
					first = row
				}
				last = row
				break
			}
		}
	}
	if first < 0 {
		return 0, 0
	}
	return clampi(first-2, 0, gridRows-1), clampi(last+2, 0, gridRows-1)
}

// renderMap paints a decoded block-index map into a full-resolution image. The 4096-byte
// map is a (4096/stride) x stride grid: block(row,col) = romMap[row*stride + col]. All
// cols wide x rows [r0..r1] is the played extent. Each block index expands to a 4x4 grid
// of 8x8 tiles via the block tile table: tile(r,c) = rom[blkTable + idx*16 + r*4 + c].
func renderMap(rom, romMap, tiles []byte, pal color.Palette, stride, cols, r0, r1, blkTable int) *image.Paletted {
	rows := r1 - r0 + 1
	img := image.NewPaletted(image.Rect(0, 0, cols*4*8, rows*4*8), pal)
	for row := r0; row <= r1; row++ {
		for col := 0; col < cols; col++ {
			def := blkTable + int(romMap[row*stride+col])*16
			for r := 0; r < 4; r++ {
				for c := 0; c < 4; c++ {
					t := gamegear.DecodeTile(tiles[int(rom[def+r*4+c])*32:])
					ox, oy := (col*4+c)*8, ((row-r0)*4+r)*8
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
