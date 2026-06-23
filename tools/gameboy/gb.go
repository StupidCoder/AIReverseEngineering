package gameboy

import (
	"image"
	"image/color"
)

// Graphics decoders for the Game Boy's fixed hardware formats — the 2bpp tile, the
// DMG palette registers, and background-map composition — mirroring gamegear/gg.go.
// These are reusable by any DMG game; game-specific things (where a game keeps its
// assets, any compression) live in that game's own extract module.

// shades is the monochrome DMG ramp, lightest (index 0) to darkest (index 3). The
// Game Boy LCD is monochrome; the green tint is the panel, so a neutral grey ramp is
// the faithful choice.
var shades = [4]uint8{0xFF, 0xAA, 0x55, 0x00}

// DMGPalette turns a palette register (BGP $FF47 / OBP0 $FF48 / OBP1 $FF49) into a
// 4-colour palette. The register packs four 2-bit shade numbers: bits 1-0 select the
// colour drawn for pixel value 0, bits 3-2 for value 1, and so on.
func DMGPalette(reg byte) color.Palette {
	p := make(color.Palette, 4)
	for i := 0; i < 4; i++ {
		s := shades[(reg>>(2*i))&3]
		p[i] = color.RGBA{s, s, s, 0xFF}
	}
	return p
}

// GreyPalette is the identity 4-grey palette (pixel value == shade index), for
// rendering raw tile data without a specific palette register.
func GreyPalette() color.Palette {
	p := make(color.Palette, 4)
	for i := 0; i < 4; i++ {
		p[i] = color.RGBA{shades[i], shades[i], shades[i], 0xFF}
	}
	return p
}

// DecodeTile decodes one 16-byte 2bpp tile into an 8×8 grid of pixel values (0..3).
// A tile is 8 rows of 2 bytes; row y's two bytes are the low and high bitplane, and
// pixel x's value is bit (7-x) of each: low-plane bit + high-plane bit<<1.
func DecodeTile(b []byte) [8][8]uint8 {
	var t [8][8]uint8
	for y := 0; y < 8 && y*2+1 < len(b); y++ {
		lo, hi := b[y*2], b[y*2+1]
		for x := 0; x < 8; x++ {
			s := uint(7 - x)
			t[y][x] = (lo >> s & 1) | (hi>>s&1)<<1
		}
	}
	return t
}

// TileSheet renders nTiles consecutive tiles from data in a cols-wide grid with a
// 1-pixel gap, using the given 4-colour palette.
func TileSheet(data []byte, pal color.Palette, nTiles, cols int) *image.Paletted {
	rows := (nTiles + cols - 1) / cols
	img := image.NewPaletted(image.Rect(0, 0, cols*9+1, rows*9+1), pal)
	for i := 0; i < nTiles; i++ {
		if (i+1)*16 > len(data) {
			break
		}
		t := DecodeTile(data[i*16:])
		ox, oy := (i%cols)*9+1, (i/cols)*9+1
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				img.SetColorIndex(ox+x, oy+y, t[y][x])
			}
		}
	}
	return img
}

// tileOffset resolves a background tile index to a byte offset into VRAM (which
// starts at $8000), honouring LCDC bit 4: 1 = unsigned indices from $8000, 0 = signed
// indices from $9000 (the −128..127 window into $8800-$97FF).
func tileOffset(lcdc byte, idx byte) int {
	if lcdc&0x10 != 0 {
		return int(idx) * 16 // $8000 + idx*16
	}
	return 0x1000 + int(int8(idx))*16 // $9000 + signed(idx)*16
}

// RenderBGMap composes the full 256×256 background (32×32 tiles) from VRAM. mapHi
// selects the tile-map base ($9C00 if true, else $9800); lcdc drives the tile-data
// addressing; pal is the palette (e.g. DMGPalette of BGP).
func RenderBGMap(vram []byte, lcdc byte, mapHi bool, pal color.Palette) *image.Paletted {
	mapBase := 0x1800 // $9800
	if mapHi {
		mapBase = 0x1C00 // $9C00
	}
	img := image.NewPaletted(image.Rect(0, 0, 256, 256), pal)
	for ty := 0; ty < 32; ty++ {
		for tx := 0; tx < 32; tx++ {
			idx := vram[mapBase+ty*32+tx]
			off := tileOffset(lcdc, idx)
			if off+16 > len(vram) {
				continue
			}
			t := DecodeTile(vram[off:])
			for y := 0; y < 8; y++ {
				for x := 0; x < 8; x++ {
					img.SetColorIndex(tx*8+x, ty*8+y, t[y][x])
				}
			}
		}
	}
	return img
}

// RenderScreen composites the authentic 160×144 viewport: the scrolled background
// plus the 8×8 sprites, as the LCD would show it. It applies the BG and object
// palette registers per pixel (sprite pixel value 0 is transparent), so it is the
// "what you see" render. The window layer is not composited, and the BG is drawn with
// a single SCX/SCY, so a game that uses a mid-frame STAT scroll split (as SML does to
// pin its status bar) is only reproduced faithfully when the split is inactive — e.g.
// when the playfield scroll is 0.
func RenderScreen(vram, oam []byte, lcdc, scx, scy, bgp, obp0, obp1 byte) *image.Paletted {
	img := image.NewPaletted(image.Rect(0, 0, 160, 144), GreyPalette())
	mapBase := 0x1800
	if lcdc&0x08 != 0 {
		mapBase = 0x1C00
	}
	shade := func(reg byte, v uint8) uint8 { return (reg >> (2 * v)) & 3 }

	// Background.
	if lcdc&0x01 != 0 {
		for y := 0; y < 144; y++ {
			for x := 0; x < 160; x++ {
				bx, by := (x+int(scx))&0xFF, (y+int(scy))&0xFF
				idx := vram[mapBase+(by/8)*32+bx/8]
				off := tileOffset(lcdc, idx)
				t := DecodeTile(vram[off:])
				img.SetColorIndex(x, y, shade(bgp, t[by%8][bx%8]))
			}
		}
	}
	// Sprites (8×8). Drawn back-to-front so lower-priority OAM entries lose; pixel
	// value 0 is transparent. BG-priority sprites (attr bit 7) are skipped over
	// non-zero BG — approximated here by always drawing on top, which is enough for
	// a verification shot.
	for i := len(oam)/4 - 1; i >= 0; i-- {
		s := DecodeOAM(oam[i*4:])[0]
		reg := obp0
		if s.Palette1 {
			reg = obp1
		}
		t := DecodeTile(vram[s.Tile*16:])
		for ty := 0; ty < 8; ty++ {
			for tx := 0; tx < 8; tx++ {
				sx, sy := tx, ty
				if s.FlipX {
					sx = 7 - tx
				}
				if s.FlipY {
					sy = 7 - ty
				}
				v := t[sy][sx]
				if v == 0 {
					continue // transparent
				}
				px, py := s.X+tx, s.Y+ty
				if px < 0 || px >= 160 || py < 0 || py >= 144 {
					continue
				}
				img.SetColorIndex(px, py, shade(reg, v))
			}
		}
	}
	return img
}

// Sprite is one decoded OAM entry (4 bytes: Y, X, tile, attributes). Screen position
// is Y-16, X-8. Attribute bits: 7 priority, 6 Y-flip, 5 X-flip, 4 DMG palette (OBP0/1).
type Sprite struct {
	Y, X, Tile         int
	Prio, FlipY, FlipX bool
	Palette1           bool
}

// DecodeOAM unpacks the 40 sprite entries from a 160-byte OAM buffer.
func DecodeOAM(oam []byte) []Sprite {
	var out []Sprite
	for i := 0; i+3 < len(oam) && i < 160; i += 4 {
		a := oam[i+3]
		out = append(out, Sprite{
			Y: int(oam[i]) - 16, X: int(oam[i+1]) - 8, Tile: int(oam[i+2]),
			Prio: a&0x80 != 0, FlipY: a&0x40 != 0, FlipX: a&0x20 != 0, Palette1: a&0x10 != 0,
		})
	}
	return out
}
