// Package powerpacker decompresses PowerPacker ("PP20") data — one of the most
// common Amiga crunchers, used by countless games, demos and intros. The decoder
// is a faithful reimplementation of the standard PP20 unpack loop (reversed here
// from Turrican's embedded decruncher at $6078C, which is a textbook copy).
//
// Format. A PP20 stream is:
//
//	[0:4]        "PP20" magic
//	[4:8]        efficiency table: four bytes, the offset bit-widths for the four
//	             match length-classes (e.g. 9,10,12,13)
//	[8:len-4]    the crunched bit data, a sequence of 32-bit big-endian longwords
//	[len-4:len]  trailer: top 24 bits = decrunched length, low byte = the number
//	             of bits to skip in the first longword read
//
// Decoding runs backward: the output is written from its end down to its start,
// and the bitstream is consumed from the last longword toward the first, taking
// bits LSB-first within each (big-endian) longword. Tokens are either literal
// runs (a count of 2-bit chunks, then that many raw bytes) or back-references
// (a 2-bit length class selecting a fixed/extended run length and an offset whose
// width comes from the efficiency table).
package powerpacker

import (
	"encoding/binary"
	"fmt"
)

// Magic is the PP20 identifier.
const Magic = "PP20"

// Decrunch decodes a complete PP20 stream (magic through trailer) and returns the
// decompressed bytes.
func Decrunch(data []byte) ([]byte, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("powerpacker: too short (%d bytes)", len(data))
	}
	if string(data[0:4]) != Magic {
		return nil, fmt.Errorf("powerpacker: bad magic %q", data[0:4])
	}
	eff := data[4:8] // efficiency table: offset bit-widths for classes 0..3

	// Trailer: read the last longword (big-endian).
	trailer := binary.BigEndian.Uint32(data[len(data)-4:])
	skip := int(trailer & 0xFF)
	outLen := int(trailer >> 8)
	out := make([]byte, outLen)
	w := outLen // write cursor, moves backward from the end

	br := &bitReader{data: data, pos: len(data) - 4}
	if err := br.refill(); err != nil { // load the last data longword
		return nil, err
	}
	if skip != 0 {
		br.bits(skip) // discard the padding bits in the first longword
	}

	for {
		if br.bit() == 0 {
			// Literal run: count = sum of 2-bit chunks (continue while ==3).
			count := 0
			for {
				c := br.bits(2)
				count += c
				if c != 3 {
					break
				}
			}
			for i := 0; i <= count; i++ {
				w--
				if w < 0 {
					return nil, fmt.Errorf("powerpacker: literal overrun")
				}
				out[w] = byte(br.bits(8))
			}
			if w == 0 {
				break
			}
		}

		// Back-reference.
		class := br.bits(2)
		runLen := class + 1 // copies runLen+1 bytes
		var offset int
		if class == 3 {
			nbits := 7
			if br.bit() == 1 {
				nbits = int(eff[3])
			}
			offset = br.bits(nbits)
			for {
				c := br.bits(3)
				runLen += c
				if c != 7 {
					break
				}
			}
		} else {
			offset = br.bits(int(eff[class]))
		}

		src := w + offset + 1
		for i := 0; i <= runLen; i++ {
			src--
			w--
			if w < 0 || src >= outLen {
				return nil, fmt.Errorf("powerpacker: match out of range (w=%d src=%d)", w, src)
			}
			out[w] = out[src]
		}
		if w == 0 {
			break
		}
	}
	if br.err != nil {
		return nil, br.err
	}
	return out, nil
}

// bitReader consumes a PP20 bitstream: 32-bit big-endian longwords taken from the
// end of the data toward the start, bits LSB-first within each longword.
type bitReader struct {
	data []byte
	pos  int    // index of the next longword to read (exclusive upper bound)
	buf  uint32 // current longword, shifting right as bits are consumed
	left int    // bits remaining in buf
	err  error
}

func (b *bitReader) refill() error {
	if b.pos < 8 { // crunched data starts at offset 8; nothing left
		b.err = fmt.Errorf("powerpacker: bitstream underflow")
		return b.err
	}
	b.pos -= 4
	b.buf = binary.BigEndian.Uint32(b.data[b.pos : b.pos+4])
	b.left = 32
	return nil
}

// bit returns the next bit (0 or 1).
func (b *bitReader) bit() int {
	v := int(b.buf & 1)
	b.buf >>= 1
	b.left--
	if b.left == 0 {
		b.refill()
	}
	return v
}

// bits returns the next n bits, MSB-first (the first bit read is the high bit).
func (b *bitReader) bits(n int) int {
	v := 0
	for i := 0; i < n; i++ {
		v = (v << 1) | b.bit()
	}
	return v
}
