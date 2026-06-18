// Command sprites extracts Turrican's BOB (blitter-object) sprites and writes one
// PNG sheet per sprite, laying out all of its animation frames (Part V).
//
//	sprites [-o dir] [Turrican.adf]
//
// Two sets are written: the resident engine's shared weapon/effect sprites
// (sprite_<addr>.png), and, for each of the five worlds, that world's enemy
// sprites from its decoded scene block (world<N>_sprite_<addr>.png), drawn in the
// world's own palette.
//
// A sprite is an animation: a frame table (an array of pointers) whose entries are
// 14-byte BOB descriptors that draw_object_bob ($603A) reads:
//
//	+$0 bitmap data ptr   +$4 mask ptr   +$8 dest modulo
//	+$A BLTSIZE = height<<6 | width-in-words   +$C y-adjust   +$D flag
//
// The pixels are 4 bitplanes stored plane-major, one word narrower than BLTSIZE's
// width (the cookie-cut shift reads an extra word), so a frame is
// 4 * height * (width-1)*2 bytes, drawn through the 16-colour playfield palette
// (plane 3 doubles as the mask, so opaque pixels use colours 8-15; colour 0 is
// transparent). Frame tables are found by scanning for runs of pointers that all
// resolve to a valid descriptor.
package main

import (
	"encoding/binary"
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
	residentLo = 0x10
	residentHi = 0x1B780
	gfxLo      = 0x10000 // resident sprite bitmaps live in this region
	blockBase  = 0x1B980 // a scene block's runtime load address
	levelTable = 0x46A
	numWorlds  = 5
	sheetCols  = 8
	pad        = 2
)

// space is a byte slice addressed by absolute runtime address: addr `a` is at
// data[a-base].
type space struct {
	data []byte
	base int
}

func (s space) be32(a int) int { return int(binary.BigEndian.Uint32(s.data[a-s.base:])) }
func (s space) be16(a int) int { return int(binary.BigEndian.Uint16(s.data[a-s.base:])) }
func (s space) has(a, n int) bool {
	o := a - s.base
	return o >= 0 && o+n <= len(s.data)
}

type table struct {
	addr   int
	frames []frame
}
type frame struct{ bitmap, h, w int } // w = data width in words

func main() {
	out := flag.String("o", "rendered/sprites", "output directory")
	flag.Parse()
	adfPath := flag.Arg(0)
	if adfPath == "" {
		adfPath = "Turrican.adf"
	}
	adf, err := os.ReadFile(adfPath)
	if err != nil {
		fail(err)
	}
	res, err := decrunch.Decrunch(mainBlob(adf))
	if err != nil {
		fail(err)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fail(err)
	}

	// Resident shared sprites (weapons/effects), in world 0's palette.
	resident := space{data: res.Data, base: 0} // addr == file offset
	pal0 := worldPalette(adf, 0)
	emit(*out, "sprite_%05X.png", resident, pal0,
		findTables(resident, residentLo, residentHi, gfxLo, residentHi))

	// Per-world enemy sprites from each scene block, in the world's own palette.
	for w := 0; w < numWorlds; w++ {
		block := worldBlock(adf, res.Data, w)
		sp := space{data: block, base: blockBase}
		hi := blockBase + len(block)
		emit(*out, fmt.Sprintf("world%d_sprite_%%05X.png", w), sp, worldPalette(adf, w),
			findTables(sp, blockBase, hi, blockBase, hi))
	}
}

func emit(dir, nameFmt string, sp space, pal color.Palette, tables []table) {
	for _, t := range tables {
		name := fmt.Sprintf(nameFmt, t.addr)
		if err := writePNG(filepath.Join(dir, name), renderSheet(sp, pal, t)); err != nil {
			fail(err)
		}
		fmt.Printf("$%05X: %2d frames -> %s\n", t.addr, len(t.frames), name)
	}
}

// findTables scans [scanLo,scanHi) for runs of >=3 pointers that all resolve to a
// plausible BOB descriptor (bitmap in [gfxLo,gfxHi)).
func findTables(sp space, scanLo, scanHi, gfxLo, gfxHi int) []table {
	descAt := func(p int) (frame, bool) {
		if p < sp.base || !sp.has(p, 14) {
			return frame{}, false
		}
		bm := sp.be32(p)
		bs := sp.be16(p + 0xA)
		h, w := bs>>6, bs&0x3F
		if bm < gfxLo || bm >= gfxHi || !sp.has(bm, 4*h*(w-1)*2) || h < 4 || h > 96 || w < 2 || w > 12 {
			return frame{}, false
		}
		return frame{bitmap: bm, h: h, w: w - 1}, true
	}
	var out []table
	for a := scanLo; a < scanHi-4; {
		if f0, ok := descAt(sp.be32(a)); ok {
			frames := []frame{f0}
			j := a + 4
			for j < scanHi-4 {
				f, ok := descAt(sp.be32(j))
				if !ok {
					break
				}
				frames = append(frames, f)
				j += 4
			}
			if len(frames) >= 3 {
				out = append(out, table{addr: a, frames: frames})
				a = j
				continue
			}
		}
		a += 2 // tables are word- but not always long-aligned
	}
	return out
}

func renderSheet(sp space, pal color.Palette, t table) *image.Paletted {
	cw, ch := 0, 0
	for _, f := range t.frames {
		if f.w*16 > cw {
			cw = f.w * 16
		}
		if f.h > ch {
			ch = f.h
		}
	}
	cw += pad
	ch += pad
	rows := (len(t.frames) + sheetCols - 1) / sheetCols
	sheet := image.NewPaletted(image.Rect(0, 0, sheetCols*cw, rows*ch), pal)
	for i, f := range t.frames {
		drawBob(sheet, sp, f, (i%sheetCols)*cw, (i/sheetCols)*ch)
	}
	return sheet
}

// drawBob decodes one 4-bitplane plane-major BOB into the sheet at (ox,oy).
func drawBob(dst *image.Paletted, sp space, f frame, ox, oy int) {
	bpr := f.w * 2
	planeSize := f.h * bpr
	for y := 0; y < f.h; y++ {
		for x := 0; x < f.w*16; x++ {
			var v uint8
			for p := 0; p < 4; p++ {
				a := f.bitmap + p*planeSize + y*bpr + x/8
				if sp.has(a, 1) && sp.data[a-sp.base]&(0x80>>(x%8)) != 0 {
					v |= 1 << uint(p)
				}
			}
			if v != 0 { // colour 0 transparent
				dst.SetColorIndex(ox+x, oy+y, v)
			}
		}
	}
}

// worldBlock decodes world w's scene block from the disk.
func worldBlock(adf, img []byte, w int) []byte {
	t := levelTable + w*8
	o := int(binary.BigEndian.Uint32(img[t:]))
	n := int(binary.BigEndian.Uint32(img[t+4:]))
	block, err := decrunch.DecrunchBlock(adf[o : o+n])
	if err != nil {
		fail(err)
	}
	return block
}

// worldPalette reads world w's 16-colour playfield palette (index 0 transparent).
func worldPalette(adf []byte, w int) color.Palette {
	res, err := decrunch.Decrunch(mainBlob(adf))
	if err != nil {
		fail(err)
	}
	block := worldBlock(adf, res.Data, w)
	palOff := int(binary.BigEndian.Uint32(block[8:])) - blockBase
	pal := color.Palette{color.RGBA{0, 0, 0, 0}}
	for i := 1; i < 16; i++ {
		c := binary.BigEndian.Uint16(block[palOff+i*2:])
		pal = append(pal, color.RGBA{
			R: uint8((c>>8)&0xF) * 17, G: uint8((c>>4)&0xF) * 17, B: uint8(c&0xF) * 17, A: 255,
		})
	}
	return pal
}

func mainBlob(adf []byte) []byte {
	const off = 0x2C00
	return adf[off : off+int(binary.BigEndian.Uint32(adf[off:]))]
}
func writePNG(path string, im image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, im)
}
func fail(err error) {
	fmt.Fprintln(os.Stderr, "sprites:", err)
	os.Exit(1)
}
