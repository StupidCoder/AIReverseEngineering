// Package iff decodes IFF ILBM images — the standard Amiga bitmap format — into
// a Go image. It handles the planar BODY (uncompressed or ByteRun1/PackBits
// compressed) and the CMAP palette, which is all an ILBM screen needs.
package iff

import (
	"fmt"
	"image"
	"image/color"
)

func be16(b []byte, o int) int { return int(b[o])<<8 | int(b[o+1]) }
func be32(b []byte, o int) int {
	return int(b[o])<<24 | int(b[o+1])<<16 | int(b[o+2])<<8 | int(b[o+3])
}

// DecodeILBM decodes a FORM…ILBM image into an RGBA image using its CMAP.
func DecodeILBM(data []byte) (*image.RGBA, error) {
	if len(data) < 12 || string(data[0:4]) != "FORM" || string(data[8:12]) != "ILBM" {
		return nil, fmt.Errorf("iff: not a FORM ILBM")
	}
	var w, h, planes, masking, compression int
	var cmap [][3]byte
	var body []byte
	for o := 12; o+8 <= len(data); {
		id := string(data[o : o+4])
		sz := be32(data, o+4)
		o += 8
		if sz < 0 || o+sz > len(data) {
			sz = len(data) - o
		}
		chunk := data[o : o+sz]
		switch id {
		case "BMHD":
			if len(chunk) >= 11 {
				w, h = be16(chunk, 0), be16(chunk, 2)
				planes, masking, compression = int(chunk[8]), int(chunk[9]), int(chunk[10])
			}
		case "CMAP":
			for i := 0; i+3 <= len(chunk); i += 3 {
				cmap = append(cmap, [3]byte{chunk[i], chunk[i+1], chunk[i+2]})
			}
		case "BODY":
			body = chunk
		}
		o += sz
		if sz&1 == 1 {
			o++ // chunks are word-aligned
		}
	}
	if w == 0 || h == 0 || planes == 0 {
		return nil, fmt.Errorf("iff: missing or bad BMHD")
	}

	rowBytes := ((w + 15) / 16) * 2
	totalPlanes := planes
	if masking == 1 {
		totalPlanes++ // an interleaved mask plane per row
	}
	need := h * totalPlanes * rowBytes
	raw := body
	if compression == 1 {
		raw = unpackBits(body, need)
	}
	if len(raw) < need {
		raw = append(raw, make([]byte, need-len(raw))...)
	}

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		rowBase := y * totalPlanes * rowBytes
		for x := 0; x < w; x++ {
			idx := 0
			for p := 0; p < planes; p++ {
				b := raw[rowBase+p*rowBytes+x/8]
				if b&(1<<uint(7-(x&7))) != 0 {
					idx |= 1 << uint(p)
				}
			}
			var c [3]byte
			if idx < len(cmap) {
				c = cmap[idx]
			}
			img.SetRGBA(x, y, color.RGBA{c[0], c[1], c[2], 255})
		}
	}
	return img, nil
}

// unpackBits decompresses an IFF ByteRun1 (PackBits) stream up to limit bytes.
func unpackBits(src []byte, limit int) []byte {
	out := make([]byte, 0, limit)
	for i := 0; i < len(src) && len(out) < limit; {
		n := int(int8(src[i]))
		i++
		switch {
		case n >= 0: // copy the next n+1 bytes literally
			for k := 0; k <= n && i < len(src); k++ {
				out = append(out, src[i])
				i++
			}
		case n != -128: // replicate the next byte 1-n times
			if i < len(src) {
				b := src[i]
				i++
				for k := 0; k < 1-n; k++ {
					out = append(out, b)
				}
			}
		}
	}
	return out
}
