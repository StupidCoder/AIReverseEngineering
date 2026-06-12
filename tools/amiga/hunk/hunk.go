// Package hunk loads an AmigaDOS "hunk" object/executable into a flat,
// relocated memory image so it can be disassembled. A hunk file is the format
// produced by the Amiga linker and brought in by dos.library LoadSeg: a header
// listing the segment sizes, then a sequence of CODE/DATA/BSS segments each
// optionally followed by 32-bit relocation tables. This loader lays the
// segments out contiguously from a chosen base, applies the relocations, and
// returns the flat image plus each segment's base — the input the dis68k /
// codetrace68k tools expect.
package hunk

import (
	"encoding/binary"
	"fmt"
)

// Hunk block type IDs.
const (
	hunkName         = 0x3E8
	hunkCode         = 0x3E9
	hunkData         = 0x3EA
	hunkBSS          = 0x3EB
	hunkReloc32      = 0x3EC
	hunkSymbol       = 0x3F0
	hunkDebug        = 0x3F1
	hunkEnd          = 0x3F2
	hunkHeader       = 0x3F3
	hunkReloc32Short = 0x3FC
)

// Segment is one loaded hunk.
type Segment struct {
	Kind string // "CODE", "DATA" or "BSS"
	Base uint32 // load address in the flat image
	Size int    // bytes
}

// Program is a loaded hunk file.
type Program struct {
	Base     uint32    // load address of the first segment
	Segments []Segment // in file order
	Image    []byte    // the flat, relocated image (all segments concatenated)
}

type reader struct {
	b   []byte
	pos int
	err error
}

func (r *reader) u32() uint32 {
	if r.err != nil || r.pos+4 > len(r.b) {
		r.err = fmt.Errorf("hunk: unexpected end of file at %d", r.pos)
		return 0
	}
	v := binary.BigEndian.Uint32(r.b[r.pos:])
	r.pos += 4
	return v
}

func (r *reader) u16() uint16 {
	if r.err != nil || r.pos+2 > len(r.b) {
		r.err = fmt.Errorf("hunk: unexpected end of file at %d", r.pos)
		return 0
	}
	v := binary.BigEndian.Uint16(r.b[r.pos:])
	r.pos += 2
	return v
}

func (r *reader) skip(n int) { r.pos += n }
func (r *reader) align4()    { r.pos = (r.pos + 3) &^ 3 }

type seg struct {
	kind   string
	data   []byte
	relocs []reloc
}

type reloc struct {
	target  int // segment number the offsets point into
	offsets []uint32
}

// Load parses a hunk file and returns it loaded contiguously from base.
func Load(data []byte, base uint32) (*Program, error) {
	r := &reader{b: data}
	if r.u32() != hunkHeader {
		return nil, fmt.Errorf("hunk: not a HUNK_HEADER")
	}
	for { // resident library names (usually none)
		n := r.u32()
		if r.err != nil {
			return nil, r.err
		}
		if n == 0 {
			break
		}
		r.skip(int(n) * 4)
	}
	tableSize := r.u32()
	first := r.u32()
	last := r.u32()
	if r.err != nil {
		return nil, r.err
	}
	if last < first || last-first+1 != tableSize || tableSize > 4096 {
		return nil, fmt.Errorf("hunk: implausible header (table=%d first=%d last=%d)", tableSize, first, last)
	}
	for i := uint32(0); i < tableSize; i++ { // hunk sizes (with memory flags)
		s := r.u32()
		if s>>30 == 3 {
			r.u32() // explicit memory-attribute longword
		}
	}
	if r.err != nil {
		return nil, r.err
	}

	var segs []*seg
	cur := -1
	for r.pos < len(data) {
		t := r.u32()
		if r.err != nil {
			return nil, r.err
		}
		switch t {
		case hunkCode, hunkData:
			n := int(r.u32()) * 4
			kind := "CODE"
			if t == hunkData {
				kind = "DATA"
			}
			segs = append(segs, &seg{kind: kind, data: append([]byte(nil), r.b[r.pos:r.pos+n]...)})
			r.skip(n)
			cur = len(segs) - 1
		case hunkBSS:
			n := int(r.u32()) * 4
			segs = append(segs, &seg{kind: "BSS", data: make([]byte, n)})
			cur = len(segs) - 1
		case hunkReloc32:
			for {
				c := r.u32()
				if c == 0 {
					break
				}
				rl := reloc{target: int(r.u32()), offsets: make([]uint32, c)}
				for i := range rl.offsets {
					rl.offsets[i] = r.u32()
				}
				segs[cur].relocs = append(segs[cur].relocs, rl)
			}
		case hunkReloc32Short:
			for {
				c := int(r.u16())
				if c == 0 {
					break
				}
				rl := reloc{target: int(r.u16()), offsets: make([]uint32, c)}
				for i := range rl.offsets {
					rl.offsets[i] = uint32(r.u16())
				}
				segs[cur].relocs = append(segs[cur].relocs, rl)
			}
			r.align4()
		case hunkSymbol:
			for {
				n := r.u32()
				if n == 0 {
					break
				}
				r.skip(int(n)*4 + 4) // name + value
			}
		case hunkDebug, hunkName:
			r.skip(int(r.u32()) * 4)
		case hunkEnd:
			// segment terminator
		default:
			return nil, fmt.Errorf("hunk: unknown block $%X at %d", t, r.pos-4)
		}
		if r.err != nil {
			return nil, r.err
		}
	}
	return assemble(segs, base)
}

// assemble lays the segments out from base and applies the relocations.
func assemble(segs []*seg, base uint32) (*Program, error) {
	p := &Program{Base: base}
	addr := base
	for _, s := range segs {
		p.Segments = append(p.Segments, Segment{Kind: s.kind, Base: addr, Size: len(s.data)})
		addr += uint32(len(s.data))
	}
	for _, s := range segs {
		for _, rl := range s.relocs {
			if rl.target >= len(p.Segments) {
				return nil, fmt.Errorf("hunk: relocation targets hunk %d of %d", rl.target, len(p.Segments))
			}
			tgt := p.Segments[rl.target].Base
			for _, off := range rl.offsets {
				if int(off)+4 > len(s.data) {
					return nil, fmt.Errorf("hunk: relocation offset %d out of range", off)
				}
				v := binary.BigEndian.Uint32(s.data[off:]) + tgt
				binary.BigEndian.PutUint32(s.data[off:], v)
			}
		}
		p.Image = append(p.Image, s.data...)
	}
	return p, nil
}
