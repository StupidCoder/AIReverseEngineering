// sprites extracts the bitmaps from Marble Madness's sprite/level banks (.ilb
// course scenery, .vlb moving objects, .mlb level tiles) and writes them as PNGs.
//
// The codec was reversed from the decrypted game program (Marble_Madness.md
// Part IV §3): each bank's bitmap section is ByteRun1 / PackBits compressed (the
// same RLE as IFF ILBM bodies, engine routine $9118), and the pixels are 2
// bitplanes (4 colours), 16 px wide, row-interleaved (per row: plane-0 word then
// plane-1 word = 4 bytes). Colours are placeholder greys; the real per-course
// palette is a copper list not yet decoded.
//
// Usage: sprites <disk.adf> <outdir>
//   writes <outdir>/<bankname>.png — one tiled sheet of all cells per bank.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"stupidcoder.com/tools/amiga/adf"
)

// unpackByteRun1 decodes PackBits: signed control byte n; 0..127 -> copy n+1
// literals; -1..-127 -> repeat next byte (1-n) times; -128 -> no-op.
func unpackByteRun1(src []byte) []byte {
	var out []byte
	for i := 0; i < len(src); {
		n := int8(src[i])
		i++
		switch {
		case n >= 0:
			for k := 0; k <= int(n) && i < len(src); k++ {
				out = append(out, src[i])
				i++
			}
		case n != -128:
			if i >= len(src) {
				return out
			}
			b := src[i]
			i++
			for k := 0; k < 1-int(n); k++ {
				out = append(out, b)
			}
		}
	}
	return out
}

// planarCell renders one 16-wide, h-row, 2-plane (row-interleaved) cell starting
// at byte off in buf into a paletted image. Out-of-range bytes read as 0.
var greys = color.Palette{
	color.Gray{0x00}, color.Gray{0x55}, color.Gray{0xAA}, color.Gray{0xFF},
}

func planarCell(buf []byte, off, h int) *image.Paletted {
	img := image.NewPaletted(image.Rect(0, 0, 16, h), greys)
	for y := 0; y < h; y++ {
		base := off + y*4
		p0 := word(buf, base)
		p1 := word(buf, base+2)
		for x := 0; x < 16; x++ {
			bit := uint(15 - x)
			v := (p0>>bit)&1 | ((p1>>bit)&1)<<1
			img.SetColorIndex(x, y, uint8(v))
		}
	}
	return img
}

func word(b []byte, o int) uint16 {
	if o+1 < len(b) {
		return uint16(b[o])<<8 | uint16(b[o+1])
	}
	if o < len(b) {
		return uint16(b[o]) << 8
	}
	return 0
}

// sheet tiles cells (16 x h) cols-per-row into one image with a 1px gap.
func sheet(buf []byte, h, cols int) *image.Paletted {
	cellBytes := 4 * h
	n := (len(buf) + cellBytes - 1) / cellBytes
	rows := (n + cols - 1) / cols
	W := cols*(16+1) + 1
	H := rows*(h+1) + 1
	img := image.NewPaletted(image.Rect(0, 0, W, H), greys)
	for i := 0; i < n; i++ {
		cx := (i%cols)*(16+1) + 1
		cy := (i/cols)*(h+1) + 1
		c := planarCell(buf, i*cellBytes, h)
		for y := 0; y < h; y++ {
			for x := 0; x < 16; x++ {
				img.SetColorIndex(cx+x, cy+y, c.ColorIndexAt(x, y))
			}
		}
	}
	return img
}

// scale nearest-neighbours an image up by n for visibility.
func scale(src *image.Paletted, n int) *image.Paletted {
	b := src.Bounds()
	out := image.NewPaletted(image.Rect(0, 0, b.Dx()*n, b.Dy()*n), src.Palette)
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			ci := src.ColorIndexAt(x, y)
			for dy := 0; dy < n; dy++ {
				for dx := 0; dx < n; dx++ {
					out.SetColorIndex(x*n+dx, y*n+dy, ci)
				}
			}
		}
	}
	return out
}

func writePNG(path string, img image.Image) {
	chk(os.MkdirAll(filepath.Dir(path), 0755))
	f, err := os.Create(path)
	chk(err)
	defer f.Close()
	chk(png.Encode(f, img))
}

// bank describes how to carve a file: header length before the PackBits body,
// and the cell height in rows.
type bank struct {
	name   string
	hdrLen func(d []byte) int // bytes before the packed bitmap
	cellH  int                // rows per cell (0 = derive from descriptor byte[4])
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: sprites <disk.adf> <outdir>")
		os.Exit(2)
	}
	img, err := os.ReadFile(os.Args[1])
	chk(err)
	vol, err := adf.Open(img)
	chk(err)
	outdir := os.Args[2]

	chk(vol.Walk(func(e adf.Entry) error {
		if e.IsDir {
			return nil
		}
		base := e.Name
		ext := strings.ToLower(filepath.Ext(base))
		if ext != ".ilb" && ext != ".vlb" && ext != ".mlb" {
			return nil
		}
		d, err := vol.ReadFile(e.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", base, err)
			return nil
		}
		var hdr, h int
		switch ext {
		case ".ilb", ".vlb":
			// [flag][count:be16][$01][count*15 descriptors][packed bitmap]
			// descriptor 0 begins at byte 4; its byte[4] (= file byte 8) is the
			// cell height (33 for .ilb course cells, 17 for .vlb objects).
			count := int(d[1])<<8 | int(d[2])
			hdr = 4 + count*15
			h = int(d[8])
		case ".mlb":
			// different container (flag $01 + offset table); the tile bitmap is
			// the largest PackBits run. Best-effort: skip the 24-byte header and
			// render at the course tile height (best-effort 16).
			hdr = 24
			h = 16
		}
		if hdr >= len(d) || h <= 0 {
			fmt.Fprintf(os.Stderr, "skip %s: bad geometry hdr=%d h=%d\n", base, hdr, h)
			return nil
		}
		raw := unpackByteRun1(d[hdr:])
		out := filepath.Join(outdir, base+".png")
		writePNG(out, scale(sheet(raw, h, 16), 3))
		fmt.Printf("%-14s flag=$%02X hdr=%d packed=%d unpacked=%d  cell=16x%d -> %s\n",
			base, d[0], hdr, len(d)-hdr, len(raw), h, out)
		return nil
	}))
}

func chk(e error) {
	if e != nil {
		panic(e)
	}
}
