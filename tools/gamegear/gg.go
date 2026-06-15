// Package gamegear decodes the standard Sega Game Gear VDP graphics formats —
// the 4-bitplane tile, the 12-bit CRAM palette, and name-table composition — into
// Go images. These are fixed *hardware* formats and so are reusable by any Game
// Gear (and, for the tiles, Master System) game. Game-specific things — compression
// schemes, where a particular game keeps its assets — live in that game's own
// extract module, not here.
package gamegear

import (
	"image"
	"image/color"
)

// Palette decodes Game Gear CRAM bytes into an RGBA palette. Each colour is two
// little-endian bytes holding 4 bits per channel, BGR order: 0000BBBB GGGGRRRR.
func Palette(cram []byte) color.Palette {
	n := len(cram) / 2
	p := make(color.Palette, n)
	for i := 0; i < n; i++ {
		w := uint16(cram[2*i]) | uint16(cram[2*i+1])<<8
		ch := func(v uint16) uint8 { return uint8(v&0xF) * 0x11 } // 4-bit -> 8-bit
		p[i] = color.RGBA{ch(w), ch(w >> 4), ch(w >> 8), 0xFF}
	}
	return p
}

// DecodeTile decodes one 32-byte 4-bitplane tile into an 8×8 grid of colour
// indices (0..15). The VDP stores a tile as 8 rows of 4 bytes; row y's four bytes
// are bitplanes 0..3, and pixel x's index is bit (7-x) of each plane (low plane =
// bit 0).
func DecodeTile(b []byte) [8][8]uint8 {
	var t [8][8]uint8
	for y := 0; y < 8 && y*4+3 < len(b); y++ {
		p0, p1, p2, p3 := b[y*4], b[y*4+1], b[y*4+2], b[y*4+3]
		for x := 0; x < 8; x++ {
			s := uint(7 - x)
			t[y][x] = (p0>>s&1) | (p1>>s&1)<<1 | (p2>>s&1)<<2 | (p3>>s&1)<<3
		}
	}
	return t
}

// TileSheet renders nTiles consecutive tiles from data in a cols-wide grid with a
// 1-pixel gap between tiles (so individual tiles are legible).
func TileSheet(data []byte, pal color.Palette, nTiles, cols int) *image.Paletted {
	rows := (nTiles + cols - 1) / cols
	img := image.NewPaletted(image.Rect(0, 0, cols*9+1, rows*9+1), pal)
	for i := 0; i < nTiles; i++ {
		if (i+1)*32 > len(data) {
			break
		}
		t := DecodeTile(data[i*32:])
		ox, oy := (i%cols)*9+1, (i/cols)*9+1
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				img.SetColorIndex(ox+x, oy+y, t[y][x])
			}
		}
	}
	return img
}

// NameTableEntry unpacks a 2-byte Game Gear name-table word: the tile number and
// the per-tile flags (horizontal/vertical flip, the sprite/BG palette select, and
// the priority bit).
type NameTableEntry struct {
	Tile           int
	FlipH, FlipV   bool
	Palette1, Prio bool
}

func DecodeNameEntry(lo, hi byte) NameTableEntry {
	return NameTableEntry{
		Tile:     int(lo) | int(hi&1)<<8,
		FlipH:    hi&0x02 != 0,
		FlipV:    hi&0x04 != 0,
		Palette1: hi&0x08 != 0,
		Prio:     hi&0x10 != 0,
	}
}

// RenderNameTable composes a wxh-tile screen from a name table (2-byte entries),
// the tile pattern data, and the two 16-colour palettes (background indices 0..15,
// sprite palette indices 16..31). It honours per-tile flip and palette select.
func RenderNameTable(nt, tiles []byte, w, h int, pal color.Palette) *image.Paletted {
	img := image.NewPaletted(image.Rect(0, 0, w*8, h*8), pal)
	for ty := 0; ty < h; ty++ {
		for tx := 0; tx < w; tx++ {
			o := (ty*w + tx) * 2
			if o+1 >= len(nt) {
				continue
			}
			e := DecodeNameEntry(nt[o], nt[o+1])
			if (e.Tile+1)*32 > len(tiles) {
				continue
			}
			t := DecodeTile(tiles[e.Tile*32:])
			base := uint8(0)
			if e.Palette1 {
				base = 16
			}
			for y := 0; y < 8; y++ {
				for x := 0; x < 8; x++ {
					sx, sy := x, y
					if e.FlipH {
						sx = 7 - x
					}
					if e.FlipV {
						sy = 7 - y
					}
					img.SetColorIndex(tx*8+x, ty*8+y, base+t[sy][sx])
				}
			}
		}
	}
	return img
}
