// Package icon decodes Amiga Workbench icon files (.info) into Go images. An
// .info is a DiskObject header followed by one or two planar Image structures
// (the normal and, optionally, the highlighted/selected imagery). Icons carry
// no palette of their own — they are drawn in the Workbench screen pens — so
// this renders them with the standard 4-colour Workbench 1.x palette.
package icon

import (
	"fmt"
	"image"
	"image/color"
)

func be16(b []byte, o int) int { return int(b[o])<<8 | int(b[o+1]) }
func be32(b []byte, o int) uint32 {
	return uint32(b[o])<<24 | uint32(b[o+1])<<16 | uint32(b[o+2])<<8 | uint32(b[o+3])
}

// wbPalette is the standard Workbench 1.x four-colour palette (pens 0–3):
// blue desktop, black, white, orange.
var wbPalette = [4]color.RGBA{
	{0x00, 0x55, 0xAA, 0xFF},
	{0x00, 0x00, 0x00, 0xFF},
	{0xFF, 0xFF, 0xFF, 0xFF},
	{0xFF, 0x88, 0x00, 0xFF},
}

// DecodeInfo decodes a Workbench .info icon into its image(s): the normal
// imagery first, and the selected imagery second if present.
func DecodeInfo(data []byte) ([]*image.RGBA, error) {
	if len(data) < 78 || be16(data, 0) != 0xE310 {
		return nil, fmt.Errorf("icon: not a Workbench .info (bad magic)")
	}
	off := 78 // past the DiskObject header
	if be32(data, 66) != 0 {
		off += 56 // a DrawerData block precedes the imagery for disk/drawer icons
	}
	var out []*image.RGBA
	for off+20 <= len(data) {
		w, h, depth := be16(data, off+4), be16(data, off+6), be16(data, off+8)
		next := be32(data, off+16)
		if w <= 0 || h <= 0 || depth <= 0 || w > 512 || h > 512 || depth > 8 {
			break
		}
		rowBytes := ((w + 15) / 16) * 2
		planeLen := h * rowBytes
		start := off + 20
		if start+depth*planeLen > len(data) {
			break
		}
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				idx := 0
				for p := 0; p < depth; p++ {
					b := data[start+p*planeLen+y*rowBytes+x/8]
					if b&(1<<uint(7-(x&7))) != 0 {
						idx |= 1 << uint(p)
					}
				}
				img.SetRGBA(x, y, wbPalette[idx%len(wbPalette)])
			}
		}
		out = append(out, img)
		off = start + depth*planeLen
		if next == 0 {
			break
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("icon: no image data")
	}
	return out, nil
}
