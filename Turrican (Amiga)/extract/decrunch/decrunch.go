// Package decrunch is a pure-Go reimplementation of the Turrican (Amiga, TRSI
// crack) main-part decruncher that lives at $50008 on the boot disk. No emulator
// is needed: the algorithm is reproduced from the 68000 disassembly, and the
// FS-UAE/GDB oracle is used only to verify the output (never to scrape it).
//
// The crunched blob is the main read loaded to $50000 (disk offset $2C00). Its
// layout is:
//
//	[0x000:0x004] packedLen  = $22C98 (whole blob length, big-endian long)
//	[0x004:0x008] 0
//	[0x008:0x040] bootstrap that relocates the core and copies the stream
//	[0x040:0x38C] the $34C-byte decruncher core (relocated to $7F000 at runtime)
//	[0x38C:0x39E] 18-byte parameter block:
//	                +0x00 word  unused
//	                +0x02 long  output base  = $43880
//	                +0x06 long  entry point  = $5F500
//	                +0x0A long  scratch (overwritten by escape-byte reads)
//	                +0x0E long  scratch
//	[0x39E:packedLen] the packed stream
//
// The packed stream is decoded by three successive passes, applied in this order
// (the compressor ran them in reverse: RLE, then LZ, then Huffman):
//
//	pass 1  Huffman   ($502C2)  packed stream -> LZ stream
//	pass 2  LZ77      ($5019A)  LZ stream     -> RLE stream
//	pass 3  RLE       ($500CA)  RLE stream    -> final image at $43880
//
// On the Amiga each pass decodes from the top of memory ($7EB00 downward) into
// the output buffer at $43880, then the intermediate result is relocated back to
// the top for the next pass. Here every pass is just a []byte -> []byte function.
package decrunch

import (
	"encoding/binary"
	"fmt"
)

// Result is the decrunched main part of the game.
type Result struct {
	Data  []byte // the decoded image, byte 0 == memory address Base
	Base  uint32 // load address of Data[0]   ($43880)
	Entry uint32 // game entry point          ($5F500)
}

// Decrunch decodes the crunched main-part blob (blob[0] == memory $50000).
func Decrunch(blob []byte) (*Result, error) {
	if len(blob) < 0x39E {
		return nil, fmt.Errorf("decrunch: blob too small (%d bytes)", len(blob))
	}
	packedLen := int(binary.BigEndian.Uint32(blob[0x000:]))
	base := binary.BigEndian.Uint32(blob[0x38E:])
	entry := binary.BigEndian.Uint32(blob[0x392:])
	if packedLen > len(blob) {
		return nil, fmt.Errorf("decrunch: packedLen $%X exceeds blob ($%X)", packedLen, len(blob))
	}

	img, err := passes(blob[0x39E:packedLen])
	if err != nil {
		return nil, err
	}
	return &Result{Data: img, Base: base, Entry: entry}, nil
}

// DecrunchBlock decodes an in-game data block — one loaded from the floppy and
// unpacked at runtime by huff_decode ($5F000), the same three-pass decoder as
// the $50008 main decruncher. Such a block begins with an 18-byte header (a word,
// 4 skipped bytes, then three longs that $5F000 stores into the $A4 parameter
// block) followed by the identical packed stream. The decoded bytes load at the
// block's runtime base (e.g. the $1BB00 game-code overlay = ADF[$26000:+$C268]).
func DecrunchBlock(block []byte) ([]byte, error) {
	const headerLen = 18
	if len(block) < headerLen {
		return nil, fmt.Errorf("decrunch: block too small (%d bytes)", len(block))
	}
	return passes(block[headerLen:])
}

// passes runs the three decode stages (Huffman -> LZ77 -> RLE) over a packed
// stream whose head is the Huffman pass header.
func passes(stream []byte) ([]byte, error) {
	huff, err := huffman(stream)
	if err != nil {
		return nil, fmt.Errorf("huffman pass: %w", err)
	}
	lzData, err := lz(huff)
	if err != nil {
		return nil, fmt.Errorf("lz pass: %w", err)
	}
	img, err := rle(lzData)
	if err != nil {
		return nil, fmt.Errorf("rle pass: %w", err)
	}
	return img, nil
}

// --- pass 1: Huffman ($502C2) ------------------------------------------------
//
// Header (consumed from the front of the stream):
//
//	long           decodedLen          (output length, = LZ-stream length)
//	256 bytes      symVal[256]         symbol-value table (the emitted bytes)
//	long           levels              number of code-length classes
//	levels longs   thr[levels]         first-codeword thresholds, left-justified
//	levels bytes   symBase[levels]     base index into symVal per class
//	levels bytes   codeLen[levels]     codeword bit length per class
//
// then the MSB-first bitstream. For each output byte: take the 32-bit window at
// the current bit position, pick the smallest class L with window >= thr[L],
// strip the threshold, keep the top codeLen[L] bits, and emit
// symVal[(symBase[L]+thatValue) & 0xFF]. Advance the bit position by codeLen[L].
func huffman(in []byte) ([]byte, error) {
	p := 0
	need := func(n int) error {
		if p+n > len(in) {
			return fmt.Errorf("unexpected end of stream")
		}
		return nil
	}
	if err := need(4 + 256 + 4); err != nil {
		return nil, err
	}
	decodedLen := int(binary.BigEndian.Uint32(in[p:]))
	p += 4
	symVal := in[p : p+256]
	p += 256
	levels := int(binary.BigEndian.Uint32(in[p:]))
	p += 4
	if levels <= 0 || levels > 32 {
		return nil, fmt.Errorf("implausible level count %d", levels)
	}
	if err := need(levels*4 + levels + levels); err != nil {
		return nil, err
	}
	thr := make([]uint32, levels)
	for i := range thr {
		thr[i] = binary.BigEndian.Uint32(in[p:])
		p += 4
	}
	symBase := make([]int, levels)
	for i := range symBase {
		symBase[i] = int(in[p])
		p++
	}
	codeLen := make([]int, levels)
	for i := range codeLen {
		codeLen[i] = int(in[p])
		p++
	}

	// Pad the bitstream so the 40-bit window peek never runs off the end; the
	// final symbol only consumes its own bits, so trailing zeros are harmless.
	bs := in[p:]
	stream := make([]byte, len(bs)+16)
	copy(stream, bs)

	out := make([]byte, decodedLen)
	bit := 0
	for n := 0; n < decodedLen; n++ {
		w := peek32(stream, bit)
		L := 0
		for L < levels && w < thr[L] {
			L++
		}
		if L >= levels {
			return nil, fmt.Errorf("no matching Huffman class for window %08X at byte %d", w, n)
		}
		rem := w - thr[L]
		nb := codeLen[L]
		top := rem >> (32 - uint(nb)) // top nb bits; nb==0 -> 0, nb==32 -> rem
		out[n] = symVal[(symBase[L]+int(top))&0xFF]
		bit += nb
	}
	return out, nil
}

// peek32 returns the 32-bit big-endian window starting at the given bit offset
// (MSB first), matching the 68000's (d7<<d1)|(d6>>(32-d1)) construction.
func peek32(data []byte, bitpos int) uint32 {
	bi := bitpos >> 3
	sh := uint(bitpos & 7)
	var v uint64 // 40 bits
	for i := 0; i < 5; i++ {
		v = (v << 8) | uint64(data[bi+i])
	}
	return uint32((v >> (8 - sh)) & 0xFFFFFFFF)
}

// --- pass 2: LZ77 ($5019A) ---------------------------------------------------
//
// Six escape bytes head the stream. A byte that isn't an escape is a literal.
// Each escape introduces a back-reference (copy `length` bytes from `offset`
// behind the current output position); a following 0 escapes the escape byte
// itself as a literal.
func lz(in []byte) ([]byte, error) {
	if len(in) < 6 {
		return nil, fmt.Errorf("stream too short for escape table")
	}
	esc := in[:6]
	p := 6

	var table [256]int
	for i := range table {
		table[i] = -1
	}
	for k := 0; k < 6; k++ {
		table[esc[k]] = k // later install wins, as on the 68000
	}

	out := make([]byte, 0, len(in)*3)
	rb := func() (byte, error) {
		if p >= len(in) {
			return 0, fmt.Errorf("unexpected end of stream")
		}
		b := in[p]
		p++
		return b, nil
	}

	for p < len(in) {
		c := in[p]
		p++
		k := table[c]
		if k < 0 {
			out = append(out, c) // literal
			continue
		}
		switch k {
		case 0: // length byte, 16-bit offset
			n, err := rb()
			if err != nil {
				return nil, err
			}
			if n == 0 {
				out = append(out, esc[0])
				continue
			}
			hi, err := rb()
			if err != nil {
				return nil, err
			}
			lo, err := rb()
			if err != nil {
				return nil, err
			}
			out = copyBack(out, int(hi)<<8|int(lo), int(n))
		case 1: // length byte, 8-bit offset
			n, err := rb()
			if err != nil {
				return nil, err
			}
			if n == 0 {
				out = append(out, esc[1])
				continue
			}
			off, err := rb()
			if err != nil {
				return nil, err
			}
			out = copyBack(out, int(off), int(n))
		case 2: // 8-bit offset, length 3
			b, err := rb()
			if err != nil {
				return nil, err
			}
			if b == 0 {
				out = append(out, esc[2])
				continue
			}
			out = copyBack(out, int(b), 3)
		case 3: // 16-bit offset, length 4
			hi, err := rb()
			if err != nil {
				return nil, err
			}
			if hi == 0 {
				out = append(out, esc[3])
				continue
			}
			lo, err := rb()
			if err != nil {
				return nil, err
			}
			out = copyBack(out, int(hi)<<8|int(lo), 4)
		case 4: // packed: offset=(b&0xF)+1, length=(b>>4)+3
			b, err := rb()
			if err != nil {
				return nil, err
			}
			if b == 0 {
				out = append(out, esc[4])
				continue
			}
			out = copyBack(out, int(b&0x0F)+1, int(b>>4)+3)
		case 5: // packed: offset=((b&0xF)<<8)|c, length=(b>>4)+4
			b, err := rb()
			if err != nil {
				return nil, err
			}
			if b == 0 {
				out = append(out, esc[5])
				continue
			}
			lo, err := rb()
			if err != nil {
				return nil, err
			}
			out = copyBack(out, int(b&0x0F)<<8|int(lo), int(b>>4)+4)
		}
	}
	return out, nil
}

// copyBack appends `length` bytes copied from `offset` positions behind the end
// of out. Bytes are copied one at a time so overlapping runs (offset < length)
// repeat correctly, exactly like the 68000 byte loop at $502B8.
func copyBack(out []byte, offset, length int) []byte {
	src := len(out) - offset
	for i := 0; i < length; i++ {
		out = append(out, out[src+i])
	}
	return out
}

// --- pass 3: RLE ($500CA) ----------------------------------------------------
//
// Three escape bytes head the stream. Non-escape bytes are literals.
//
//	esc0: n==0 -> 16-bit count + fill byte; n==1 -> literal esc0; else n copies of fill byte
//	esc1: n==0 -> 16-bit count of zeros;    n==1 -> literal esc1; else n zero bytes
//	esc2: b==0 -> literal esc2;             else three copies of b
func rle(in []byte) ([]byte, error) {
	if len(in) < 3 {
		return nil, fmt.Errorf("stream too short for escape table")
	}
	esc := in[:3]
	p := 3

	var table [256]int
	for i := range table {
		table[i] = -1
	}
	for k := 0; k < 3; k++ {
		table[esc[k]] = k
	}

	out := make([]byte, 0, len(in)*4)
	rb := func() (byte, error) {
		if p >= len(in) {
			return 0, fmt.Errorf("unexpected end of stream")
		}
		b := in[p]
		p++
		return b, nil
	}
	rw := func() (int, error) {
		hi, err := rb()
		if err != nil {
			return 0, err
		}
		lo, err := rb()
		if err != nil {
			return 0, err
		}
		return int(hi)<<8 | int(lo), nil
	}

	for p < len(in) {
		c := in[p]
		p++
		k := table[c]
		if k < 0 {
			out = append(out, c)
			continue
		}
		switch k {
		case 0:
			n, err := rb()
			if err != nil {
				return nil, err
			}
			switch {
			case n == 0:
				count, err := rw()
				if err != nil {
					return nil, err
				}
				fill, err := rb()
				if err != nil {
					return nil, err
				}
				for i := 0; i < count; i++ {
					out = append(out, fill)
				}
			case n == 1:
				out = append(out, esc[0])
			default:
				fill, err := rb()
				if err != nil {
					return nil, err
				}
				for i := 0; i < int(n); i++ {
					out = append(out, fill)
				}
			}
		case 1:
			n, err := rb()
			if err != nil {
				return nil, err
			}
			switch {
			case n == 0:
				count, err := rw()
				if err != nil {
					return nil, err
				}
				for i := 0; i < count; i++ {
					out = append(out, 0)
				}
			case n == 1:
				out = append(out, esc[1])
			default:
				for i := 0; i < int(n); i++ {
					out = append(out, 0)
				}
			}
		case 2:
			b, err := rb()
			if err != nil {
				return nil, err
			}
			if b == 0 {
				out = append(out, esc[2])
			} else {
				out = append(out, b, b, b)
			}
		}
	}
	return out, nil
}
