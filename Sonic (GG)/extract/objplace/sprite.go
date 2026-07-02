package objplace

import (
	"sort"

	"sonicgg/extract/decomp"
)

// SpriteRef is where a type's idle metasprite layout lives in the ROM.
type SpriteRef struct {
	Kind   string // "anim" (base+frame*18), "direct" (explicit ptr), "" (none/invisible)
	Layout int    // file offset of the 18-byte layout (handlers are in banks 1/2 -> file = z80 addr)
	Frame  int    // frame id used (for "anim")
}

// handlerAddr returns the z80 handler address for an object type, or 0 if unused.
func handlerAddr(rom []byte, t int) uint16 {
	return uint16(word(rom, dispatch+t*2))
}

// handlerBounds returns, for each handler address, the address of the next handler in
// the same bank (so a linear scan of one handler doesn't run into the next).
func handlerBounds(rom []byte) map[uint16]uint16 {
	var addrs []uint16
	for t := 0; t < 0x57; t++ {
		if a := handlerAddr(rom, t); a != 0 {
			addrs = append(addrs, a)
		}
	}
	sort.Slice(addrs, func(i, j int) bool { return addrs[i] < addrs[j] })
	end := map[uint16]uint16{}
	for i, a := range addrs {
		e := uint16(0xC000)
		bankTop := (a & 0xC000) + 0x4000 // stay within the home slot window
		if i+1 < len(addrs) && addrs[i+1] < bankTop {
			e = addrs[i+1]
		} else if bankTop < e {
			e = bankTop
		}
		end[a] = e
	}
	return end
}

// analyzeSprite scans a handler's byte range for the first (lowest-address) sprite-
// layout assignment, reading the layout pointer straight out of the handler's own
// operands. Three idioms set IX+15/16 (the metasprite pointer the draw $2F07 reads):
//
//   - CALL $7C75 (the shared animator): layout = DE_base + frameId*18. DE is loaded
//     by a nearby LD DE,nn; the frame id is the first byte of the BC animation
//     sequence when BC is a nearby immediate (else 0 = the idle pose).
//   - LD (IX+15),imm / LD (IX+16),imm: an explicit layout pointer.
//   - LD (IX+15),L / LD (IX+16),H preceded by LD HL,nn: an explicit pointer in HL.
//
// The scan is raw bytes (not a decoded stream) so it is immune to the data tables
// many handlers embed; the idioms are distinctive enough to match directly.
func AnalyzeSprite(rom []byte, t, zone int) SpriteRef {
	a := handlerAddr(rom, t)
	if a == 0 {
		return SpriteRef{}
	}
	end := int(handlerBounds(rom)[a])
	// nearestBefore finds the closest occurrence of LD <rr>,nn (opcode op, 3 bytes)
	// in the dozen bytes before pos, returning its immediate or -1.
	nearestBefore := func(op byte, pos int) int {
		for i := pos - 3; i >= pos-14 && i >= int(a); i-- {
			if rom[i] == op {
				return int(rom[i+1]) | int(rom[i+2])<<8
			}
		}
		return -1
	}
	// collectBefore returns every LD HL,nn immediate (opcode $21) in the window before
	// pos, in address order, plus whether a zone read (LD A,($D2D5) = 3A D5 D2) appears
	// in that window. Platform/zone-variant handlers load HL = the zone-0 layout first,
	// then overwrite it with the zone-1.. layouts: HL list = [zone0, zone1, .., else].
	collectBefore := func(pos int) ([]int, bool) {
		var hls []int
		zoneSel := false
		for i := pos - 22; i < pos; i++ {
			if i < int(a) {
				continue
			}
			if rom[i] == 0x21 && i+2 < pos {
				hls = append(hls, int(rom[i+1])|int(rom[i+2])<<8)
			}
			if rom[i] == 0x3A && rom[i+1] == 0xD5 && rom[i+2] == 0xD2 {
				zoneSel = true
			}
		}
		return hls, zoneSel
	}
	for off := int(a); off+1 < end && off+8 < len(rom); off++ {
		b := rom[off:]
		// CALL $7C75  (CD 75 7C)
		if b[0] == 0xCD && b[1] == 0x75 && b[2] == 0x7C {
			de := nearestBefore(0x11, off) // LD DE,base
			if de < 0 {
				continue
			}
			// Frame 0 is the layout base = the idle pose (verified on crab/beetle).
			return SpriteRef{"anim", de, 0}
		}
		// LD (IX+15),imm ; LD (IX+16),imm  (DD 36 0F i0  DD 36 10 i1)
		if b[0] == 0xDD && b[1] == 0x36 && b[2] == 0x0F &&
			b[4] == 0xDD && b[5] == 0x36 && b[6] == 0x10 {
			return SpriteRef{"direct", int(b[3]) | int(b[7])<<8, 0}
		}
		// LD (IX+15),L ; LD (IX+16),H  (DD 75 0F  DD 74 10) with a nearby LD HL,nn.
		// When the handler selects the layout by zone, pick this zone's pointer.
		if b[0] == 0xDD && b[1] == 0x75 && b[2] == 0x0F &&
			b[3] == 0xDD && b[4] == 0x74 && b[5] == 0x10 {
			if hls, zoneSel := collectBefore(off); len(hls) > 0 {
				ptr := hls[len(hls)-1]
				if zoneSel && len(hls) > 1 {
					if zone < len(hls) {
						ptr = hls[zone] // [zone0, zone1, .., else]; clamp below
					}
				}
				return SpriteRef{"direct", ptr, 0}
			}
		}
	}
	return SpriteRef{}
}

// CommonTilesFile is the COMMON sprite tile block the level loader decompresses to
// VRAM $3000 (sprite tiles $80-$BF) alongside the zone sheets ($0406 call with
// HL=$B354/DE=$3000, bank 11): HUD digits, sparkles, the item-box bottom, springs.
// Oracle-verified byte-identical to live VRAM tiles $80-$B3 ($B4-$BB are later
// overwritten by Sonic's dynamic frame stream).
const CommonTilesFile = 0x2F354

// SpriteSheet builds act i's full 256-tile sprite sheet exactly as the level loader
// lays out sprite VRAM: the zone's own sheet at tiles $00-$7F (descriptor +23/+24,
// VRAM $2000) and the common block at $80-$BF (VRAM $3000). Layout cells above the
// 128-tile zone sheet (e.g. the item box's bottom row $AA-$AF) resolve into the
// common block.
func SpriteSheet(rom []byte, act int) []byte {
	const descTable = 0x15600
	d := descTable + word(rom, descTable+act*2)
	off := decomp.SourceOffset(int(rom[d+23]), uint16(word(rom, d+24)))
	tiles := make([]byte, 256*32)
	copy(tiles, decomp.Decompress(rom, off))
	copy(tiles[0x80*32:], decomp.Decompress(rom, CommonTilesFile))
	return tiles
}

// ApplyIconUpload emulates the item handlers' lazy tile upload $0BA8 (8 bytes/frame
// from bank 5 into VRAM $2B80 = sprite tiles $5C-$5F over 16 frames): the 16x16 icon
// on the monitor's screen. The source is the LD HL,nn preceding the CALL $0BA8 in the
// handler, so each type shows its own icon (bonus $01 = file $15200, $02 = $15280,
// emerald $06 = $15480). Returns a patched copy when the handler uploads one, else
// the sheet unchanged (the slots hold the zone sheet's own tiles $5C-$5F).
func ApplyIconUpload(rom, tiles []byte, typ int) []byte {
	a, end := HandlerRange(rom, typ)
	for o := a; a != 0 && o+2 < end; o++ {
		if rom[o] == 0xCD && rom[o+1] == 0xA8 && rom[o+2] == 0x0B { // CALL $0BA8
			for j := o - 3; j >= o-16 && j >= a; j-- {
				if rom[j] == 0x21 { // LD HL,nn
					src := 0x14000 + (int(rom[j+1]) | int(rom[j+2])<<8) - 0x4000
					patched := append([]byte(nil), tiles...)
					copy(patched[0x5C*32:0x60*32], rom[src:src+128])
					return patched
				}
			}
		}
	}
	return tiles
}
