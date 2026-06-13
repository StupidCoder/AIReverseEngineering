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

// pal is the 4-colour image palette for the bank currently being rendered (the
// .ilb/.vlb cells are 2 bitplanes = colours 0..3 of the course palette). It is
// reset per bank in main; greys is the fallback when no course palette is known.
var greys = color.Palette{
	color.Gray{0x00}, color.Gray{0x55}, color.Gray{0xAA}, color.Gray{0xFF},
}
var pal = greys

// mlbPalette reads a course's 16-colour playfield palette from its .mlb level
// module: 16 big-endian $0RGB words at offset 0x17 (Marble_Madness.md Part IV §3).
func mlbPalette(d []byte) color.Palette {
	p := make(color.Palette, 16)
	for i := 0; i < 16; i++ {
		o := 0x17 + 2*i
		w := uint16(0)
		if o+1 < len(d) {
			w = uint16(d[o])<<8 | uint16(d[o+1])
		}
		n4 := func(x uint16) uint8 { return uint8(x) * 17 } // 4-bit -> 8-bit
		p[i] = color.RGBA{n4((w >> 8) & 0xF), n4((w >> 4) & 0xF), n4(w & 0xF), 0xFF}
	}
	return p
}

// mlbSheet renders a .mlb course bitmap: 4 bitplanes (16 colours) whose plane
// data starts at the four big-endian offsets in the header (file bytes 0x07/
// 0x0b/0x0f/0x13, into the unpacked buffer). The .mlb is a tile-sheet format, so
// the exact width/tilemap is not yet reversed — this renders the raw planes at a
// fixed width so the real colours are visible.
func mlbSheet(up []byte, raw []byte, W int) *image.Paletted {
	off := func(o int) int {
		if o+3 < len(raw) {
			return int(raw[o])<<24 | int(raw[o+1])<<16 | int(raw[o+2])<<8 | int(raw[o+3])
		}
		return 0
	}
	planes := [4]int{off(0x07), off(0x0b), off(0x0f), off(0x13)}
	bpr := W / 8
	rows := (planes[1] - planes[0]) / bpr
	if rows < 1 || rows > 4000 {
		rows = 256
	}
	img := image.NewPaletted(image.Rect(0, 0, W, rows), pal)
	for y := 0; y < rows; y++ {
		for x := 0; x < W; x++ {
			bi := y*bpr + x/8
			bit := uint(7 - x%8)
			v := 0
			for p := 0; p < 4; p++ {
				o := planes[p] + bi
				if o < len(up) && (up[o]>>bit)&1 == 1 {
					v |= 1 << p
				}
			}
			img.SetColorIndex(x, y, uint8(v))
		}
	}
	return img
}

// planarCell renders one 16-wide, h-row, 2-plane (row-interleaved) cell starting
// at byte off in buf into a paletted image. Out-of-range bytes read as 0.
func planarCell(buf []byte, off, h int) *image.Paletted {
	img := image.NewPaletted(image.Rect(0, 0, 16, h), pal)
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
	img := image.NewPaletted(image.Rect(0, 0, W, H), pal)
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

// courseOf maps a bank's filename prefix to the course whose .mlb palette it
// uses. The shared object banks (marble, creatures) fall back to "practy".
func courseOf(base string) string {
	for _, c := range []struct{ pre, course string }{
		{"prc", "practy"}, {"practy", "practy"},
		{"beg", "beginr"}, {"beginr", "beginr"},
		{"int", "interm"}, {"interm", "interm"},
		{"aer", "aerial"}, {"aerial", "aerial"},
		{"sil", "silly"}, {"silly", "silly"}, {"slink", "silly"},
		{"ult", "ultima"}, {"ultima", "ultima"},
	} {
		if strings.HasPrefix(base, c.pre) {
			return c.course
		}
	}
	return "practy"
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

	// Pre-load each course's playfield palette from its .mlb (offset 0x17).
	palettes := map[string]color.Palette{}
	for _, c := range []string{"practy", "beginr", "interm", "aerial", "silly", "ultima"} {
		if d, err := vol.ReadFile("c/" + c + ".mlb"); err == nil {
			palettes[c] = mlbPalette(d)
		} else if d, err := vol.ReadFile(c + ".mlb"); err == nil {
			palettes[c] = mlbPalette(d)
		}
	}

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
		course := courseOf(base)
		out := filepath.Join(outdir, base+".png")

		if ext == ".mlb" {
			// The .mlb playfield is 4 bitplanes = 16 colours (its own palette at
			// 0x17). The packed tile bitmap follows the 16-colour palette at 0x37.
			pal = greys
			if p, ok := palettes[course]; ok {
				pal = p // full 16-colour playfield palette
			}
			raw := unpackByteRun1(d[0x37:])
			writePNG(out, scale(mlbSheet(raw, d, 320), 2))
			fmt.Printf("%-14s flag=$%02X unpacked=%d  4-plane 16-colour  pal=%s -> %s\n",
				base, d[0], len(raw), course, out)
			return nil
		}

		// .ilb / .vlb: [flag][count:be16][$01][count*15 descriptors][packed bitmap].
		// Descriptor 0 begins at byte 4; its byte[4] (= file byte 8) is the cell
		// height (33 for .ilb course cells, 17 for .vlb objects). These banks are 2
		// bitplanes (4 colours) -> palette colours 0..3.
		count := int(d[1])<<8 | int(d[2])
		hdr := 4 + count*15
		h := int(d[8])
		if hdr >= len(d) || h <= 0 {
			fmt.Fprintf(os.Stderr, "skip %s: bad geometry hdr=%d h=%d\n", base, hdr, h)
			return nil
		}
		pal = greys
		if p, ok := palettes[course]; ok {
			pal = p[:4]
		}
		raw := unpackByteRun1(d[hdr:])
		writePNG(out, scale(sheet(raw, h, 16), 3))
		fmt.Printf("%-14s flag=$%02X hdr=%d packed=%d unpacked=%d  cell=16x%d 2-plane  pal=%s -> %s\n",
			base, d[0], hdr, len(d)-hdr, len(raw), h, course, out)
		return nil
	}))
}

func chk(e error) {
	if e != nil {
		panic(e)
	}
}
