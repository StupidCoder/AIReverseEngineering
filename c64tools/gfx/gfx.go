// Package gfx provides generic C64 graphics primitives: the hardware palette,
// multicolor character rendering, hires sprite rendering, simple marker
// drawing and PNG output. Game-specific knowledge (which colour goes where,
// charset/map layout) lives in the per-game packages that call these.
package gfx

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

// Palette is the Pepto rendering of the 16 C64 colours.
var Palette = [16]color.RGBA{
	{0x00, 0x00, 0x00, 0xFF}, {0xFF, 0xFF, 0xFF, 0xFF}, {0x68, 0x37, 0x2B, 0xFF}, {0x70, 0xA4, 0xB2, 0xFF},
	{0x6F, 0x3D, 0x86, 0xFF}, {0x58, 0x8D, 0x43, 0xFF}, {0x35, 0x28, 0x79, 0xFF}, {0xB8, 0xC7, 0x6F, 0xFF},
	{0x6F, 0x4F, 0x25, 0xFF}, {0x43, 0x39, 0x00, 0xFF}, {0x9A, 0x67, 0x59, 0xFF}, {0x44, 0x44, 0x44, 0xFF},
	{0x6C, 0x6C, 0x6C, 0xFF}, {0x9A, 0xD2, 0x84, 0xFF}, {0x6C, 0x5E, 0xB5, 0xFF}, {0x95, 0x95, 0x95, 0xFF},
}

// VIC hardware sprite dimensions in pixels.
const (
	SpriteW = 24
	SpriteH = 21
)

// DrawChar renders one 8x8 multicolor character at (px,py) with pixel scale s.
// Each byte row is four colour-pair pixels indexed into pal.
func DrawChar(img *image.RGBA, glyph []byte, px, py, s int, pal [4]color.RGBA) {
	for row := 0; row < 8; row++ {
		b := glyph[row]
		for pair := 0; pair < 4; pair++ {
			c := pal[(b>>(6-2*pair))&3]
			for dy := 0; dy < s; dy++ {
				for dx := 0; dx < 2*s; dx++ {
					img.SetRGBA(px+(pair*2)*s+dx, py+row*s+dy, c)
				}
			}
		}
	}
}

// RenderSpriteSheet renders hires (1bpp) sprite frames in a single row, in the
// sprite colour colorIdx. xExpand stretches pixels horizontally (matching VIC
// X-expansion); scale s applies on top.
func RenderSpriteSheet(frames [][]byte, colorIdx byte, s, xExpand int) *image.RGBA {
	return RenderSpriteGrid([][][]byte{frames}, colorIdx, s, xExpand)
}

// RenderSpriteGrid renders rows of hires sprite frames as a grid: within a row
// frames sit side by side, rows stack top to bottom.
func RenderSpriteGrid(rows [][][]byte, colorIdx byte, s, xExpand int) *image.RGBA {
	fw := SpriteW * xExpand * s
	cols := 0
	for _, r := range rows {
		if len(r) > cols {
			cols = len(r)
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, fw*cols, SpriteH*s*len(rows)))
	c := Palette[colorIdx&0x0F]
	bg := Palette[0]
	for ri, frames := range rows {
		for i, blk := range frames {
			for row := 0; row < SpriteH; row++ {
				for bit := 0; bit < SpriteW; bit++ {
					px := bg
					if blk[row*3+bit/8]&(0x80>>(bit%8)) != 0 {
						px = c
					}
					for dy := 0; dy < s; dy++ {
						for dx := 0; dx < xExpand*s; dx++ {
							img.SetRGBA(i*fw+bit*xExpand*s+dx, (ri*SpriteH+row)*s+dy, px)
						}
					}
				}
			}
		}
	}
	return img
}

// DrawCircle draws a hollow circle of radius r px and line thickness t,
// centered at (cx,cy).
func DrawCircle(img *image.RGBA, cx, cy, r, t int, c color.RGBA) {
	inner := (r - t) * (r - t)
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			d := dx*dx + dy*dy
			if d <= r*r && d >= inner {
				img.SetRGBA(cx+dx, cy+dy, c)
			}
		}
	}
}

// FrameCell draws a rectangle around a w x h character-cell area whose top-left
// cell is (col,row), with line thickness s (one scaled pixel).
func FrameCell(img *image.RGBA, col, row, w, h, s int, c color.RGBA) {
	x0, y0 := col*8*s-s, row*8*s-s
	x1, y1 := (col+w)*8*s+s-1, (row+h)*8*s+s-1
	for t := 0; t < s; t++ {
		for x := x0; x <= x1; x++ {
			img.SetRGBA(x, y0+t, c)
			img.SetRGBA(x, y1-t, c)
		}
		for y := y0; y <= y1; y++ {
			img.SetRGBA(x0+t, y, c)
			img.SetRGBA(x1-t, y, c)
		}
	}
}

// WritePNG encodes img to path.
func WritePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return err
	}
	return f.Close()
}
