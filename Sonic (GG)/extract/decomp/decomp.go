// Package decomp implements Sonic the Hedgehog (Game Gear)'s graphics
// decompressor — the routine the game calls at $0406 (Sonic.md Part IV). It is a
// 4-byte-unit LZ: a control bitmap selects, for each 4-byte output unit, either a
// literal (the next 4 bytes of a literal stream) or a match (a back-reference, by
// a 4-byte-aligned offset, into that same stream). A 4-byte unit is one row of a
// 4-bitplane tile, so repeated tile rows (blank rows especially) cost one bit.
//
// This is a game-specific software codec, not a Game Gear hardware format, so it
// lives with the game rather than in tools/gamegear.
package decomp

// Header is the 8-byte block header the engine reads before the streams.
type Header struct {
	MatchOff int // word1: offset (from the block start) to the match-info stream
	LitOff   int // word2: offset to the literal stream (also the back-reference base)
	Count    int // word3: number of 4-byte output units
}

// ReadHeader reads the block header at file offset off.
func ReadHeader(rom []byte, off int) Header {
	u16 := func(o int) int { return int(rom[o]) | int(rom[o+1])<<8 }
	return Header{MatchOff: u16(off + 2), LitOff: u16(off + 4), Count: u16(off + 6)}
}

// Decompress inflates the block at file offset off and returns Count*4 bytes. The
// caller passes the already-resolved file offset of the block (the engine forms it
// by normalising a (bank, address) pair so the source can span banks; see
// SourceOffset). It is defensive: a read past the end of rom stops the output.
func Decompress(rom []byte, off int) []byte {
	h := ReadHeader(rom, off)
	ctrl := off + 8       // control bitmap: 1 bit per unit
	lit := off + h.LitOff // literal stream + back-reference base
	mi := off + h.MatchOff
	litp := lit
	out := make([]byte, 0, h.Count*4)
	emit := func(p int) bool {
		if p < 0 || p+4 > len(rom) {
			return false
		}
		out = append(out, rom[p:p+4]...)
		return true
	}
	for i := 0; i < h.Count; i++ {
		if ctrl+i/8 >= len(rom) {
			break
		}
		if rom[ctrl+i/8]&(1<<(uint(i)&7)) == 0 {
			// literal: the next 4-byte unit of the literal stream
			if !emit(litp) {
				break
			}
			litp += 4
		} else {
			// match: a variable-length 4-byte-aligned offset into the literal stream
			if mi >= len(rom) {
				break
			}
			b := int(rom[mi])
			mi++
			val := b
			if b >= 0xF0 {
				if mi >= len(rom) {
					break
				}
				val = (b-0xF0)<<8 | int(rom[mi])
				mi++
			}
			if !emit(lit + val*4) {
				break
			}
		}
	}
	return out
}

// SourceOffset converts a (bank, Z80 address) source the way the engine's $0406
// prologue does — while the address is >= $4000 it subtracts $4000 and advances the
// bank, mapping the result so the data may span consecutive banks — and returns the
// flat ROM file offset of the block.
func SourceOffset(bank int, addr uint16) int {
	for addr >= 0x4000 {
		addr -= 0x4000
		bank++
	}
	return bank*0x4000 + int(addr)
}
