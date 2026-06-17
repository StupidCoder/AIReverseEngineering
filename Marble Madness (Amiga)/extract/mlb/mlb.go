// Package mlb decodes a Marble Madness course's .mlb ("map library") file and
// assembles the playfield tilemap into an image. The whole file is one
// ByteRun1/PackBits stream; unpacked, its header holds the four tile bitplane
// offsets and the tilemap offset, the 16-colour course palette sits at +0x16,
// and the tilemap is a row-major stream of big-endian tile-index words 36 tiles
// (288 px) wide — the game scrolls vertically so width is fixed and height
// varies (Marble_Madness.md Part IV §3). Tiles are 8x8, 4 bitplanes (16
// colours), even/odd tiles byte-interleaved within 16-byte groups.
package mlb

import (
	"image"
	"image/color"
)

// CourseW is the fixed tilemap width in tiles (the blitter's 72-byte row).
const CourseW = 36

// cycleInitial maps the palette slots the runtime colour-cycling system drives
// to a representative mid-cycle colour, so a static render of a fully
// cycle-driven slot (left black in the .mlb, e.g. Beginner's ice pit) still
// shows the hazard/ice colour it reads as in motion (Part IV §3).
var cycleInitial = map[int]uint16{11: 0x0F88, 12: 0x0F88, 13: 0x088F, 14: 0x088F}

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

// palette reads the 16-colour playfield palette from the unpacked buffer at
// +0x16 (big-endian $0RGB words), restoring cycle-driven black slots.
func palette(buf []byte) color.Palette {
	n4 := func(x uint16) uint8 { return uint8(x) * 17 }
	rgb := func(w uint16) color.RGBA {
		return color.RGBA{n4((w >> 8) & 0xF), n4((w >> 4) & 0xF), n4(w & 0xF), 0xFF}
	}
	p := make(color.Palette, 16)
	for i := 0; i < 16; i++ {
		o := 0x16 + 2*i
		var w uint16
		if o+1 < len(buf) {
			w = uint16(buf[o])<<8 | uint16(buf[o+1])
		}
		if w == 0 {
			if init, ok := cycleInitial[i]; ok {
				w = init
			}
		}
		p[i] = rgb(w)
	}
	return p
}

// blitTile draws 8x8 tile idx at (px,py): tile row r is the byte at
// plane[(idx>>1)*16 + (idx&1) + 2*r] of the unpacked buffer.
func blitTile(img *image.Paletted, buf []byte, planes [4]int, idx, px, py int) {
	b0 := (idx>>1)*16 + (idx & 1)
	for r := 0; r < 8; r++ {
		var bp [4]byte
		for p := 0; p < 4; p++ {
			if o := planes[p] + b0 + 2*r; o >= 0 && o < len(buf) {
				bp[p] = buf[o]
			}
		}
		for x := 0; x < 8; x++ {
			bit := uint(7 - x)
			v := bp[0]>>bit&1 | (bp[1]>>bit&1)<<1 | (bp[2]>>bit&1)<<2 | (bp[3]>>bit&1)<<3
			img.SetColorIndex(px+x, py+r, v)
		}
	}
}

// RenderCourse decodes a .mlb file and returns the assembled course image (8 px
// per tile, CourseW tiles wide) and its height in tiles.
func RenderCourse(file []byte) (*image.Paletted, int) {
	buf := unpackByteRun1(file)
	lw := func(o int) int {
		if o+4 > len(buf) {
			return 0
		}
		return int(buf[o])<<24 | int(buf[o+1])<<16 | int(buf[o+2])<<8 | int(buf[o+3])
	}
	planes := [4]int{lw(2), lw(6), lw(10), lw(14)}
	tmOff := lw(0x12)
	tilemap := buf[tmOff:]
	h := (len(tilemap) / 2) / CourseW
	img := image.NewPaletted(image.Rect(0, 0, CourseW*8, h*8), palette(buf))
	for ty := 0; ty < h; ty++ {
		for tx := 0; tx < CourseW; tx++ {
			o := (ty*CourseW + tx) * 2
			idx := int(tilemap[o])<<8 | int(tilemap[o+1])
			blitTile(img, buf, planes, idx, tx*8, ty*8)
		}
	}
	return img, h
}
