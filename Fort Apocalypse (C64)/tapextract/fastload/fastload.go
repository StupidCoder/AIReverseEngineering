// Package fastload decodes the Novaload-family fastloader used on the
// Fort Apocalypse tape, as reverse engineered from the loader code hidden
// in the KERNAL header block (IRQ handler at $0351) and the BASIC stub's
// machine code at $080D.
//
// Encoding (one bit per pulse, measured with CIA1 timer A latched to $03F4):
//
//	pulse ≤ 500 cycles → bit 0      (on tape: $25 = 296 cycles)
//	pulse > 500 cycles → bit 1      (on tape: $54 = 672 cycles)
//
// Bits are shifted into a register with ROR, so the first pulse of a byte
// carries the least significant bit. The register is pre-loaded with $7F;
// the 0 that falls out of bit 0 after eight shifts marks byte completion.
// During pilot search the register is NOT reset on a mismatch, so a run of
// ≥8 zero bits followed by a single one bit reads as the pilot byte $80.
//
// Stream layout:
//
//	pilot: long run of 0-bits, terminated by one 1-bit  (reads as $80)
//	$AA   sync byte
//	$55   key byte (checked against the checksum seed, initialised to $55)
//	page records, each: [page#] [256 data bytes] [checksum]
//	      data is stored at page#<<8 .. page#<<8+255
//	      checksum = (page# + sum of data bytes) mod 256
//	the record with page# == $F0 arms end mode; after it, a page# byte
//	of $00 terminates the stream (otherwise loading continues normally)
package fastload

import (
	"fmt"
	"sort"

	"tapextract/tap"
)

// bitThreshold mirrors the loader's timer test: timer A counts down from
// $03F4; the IRQ handler reads the high byte, so a pulse longer than
// $03F4-$0200 = 500 cycles leaves the high byte ≤ $01 and is read as a 1.
const bitThreshold = 500

const (
	pilotByte = 0x80
	syncByte  = 0xAA
	seedByte  = 0x55
	endPage   = 0xF0
)

// Record is one decoded page record.
type Record struct {
	Page       byte
	Data       []byte // 256 bytes
	Checksum   byte   // as read from tape
	ChecksumOK bool
	PulseIndex int // pulse index where the record's page byte completed
}

// Result of decoding one fastloader stream.
type Result struct {
	Records    []Record
	Memory     map[uint16]byte // address -> byte, later records overwrite earlier
	Terminated bool            // saw the $00 end page after the $F0 record
	EndPulse   int             // index just past the last consumed pulse
	Err        error           // first checksum error, if any
}

// Range is a contiguous loaded memory region.
type Range struct {
	Start uint16
	Data  []byte
}

// Ranges returns the loaded memory as sorted contiguous regions.
func (r *Result) Ranges() []Range {
	addrs := make([]int, 0, len(r.Memory))
	for a := range r.Memory {
		addrs = append(addrs, int(a))
	}
	sort.Ints(addrs)
	var out []Range
	for _, a := range addrs {
		n := len(out)
		if n > 0 && int(out[n-1].Start)+len(out[n-1].Data) == a {
			out[n-1].Data = append(out[n-1].Data, r.Memory[uint16(a)])
		} else {
			out = append(out, Range{Start: uint16(a), Data: []byte{r.Memory[uint16(a)]}})
		}
	}
	return out
}

// Decode runs the loader's state machine over pulses[from:]. It mirrors the
// original 6502 code exactly, including the non-resetting shift register
// during pilot search.
func Decode(pulses []tap.Pulse, from int) *Result {
	res := &Result{Memory: make(map[uint16]byte)}

	shift := byte(0x7F) // $A9 in the original
	cksum := byte(seedByte)
	state := statePilot
	endMode := false
	var page byte
	var y int
	var cur Record

	i := from
	for ; i < len(pulses); i++ {
		p := pulses[i]
		if p.Pause {
			if state != statePilot {
				break // a pause inside a stream ends it
			}
			continue
		}
		bit := byte(0)
		if p.Cycles > bitThreshold {
			bit = 1
		}
		carryOut := shift & 1
		shift = shift>>1 | bit<<7
		if carryOut != 0 {
			continue // byte not complete
		}
		v := shift
		reset := true
		switch state {
		case statePilot:
			if v == pilotByte {
				state = stateSync
			} else {
				reset = false // keep sliding until the pilot pattern aligns
			}
		case stateSync:
			if v == syncByte {
				state = stateChecksum // first "checksum" is the $55 seed
			} else {
				state = statePilot
			}
		case stateChecksum:
			if v == cksum {
				if endMode {
					state = stateEndPage
				} else {
					state = statePage
				}
				if len(res.Records) > 0 {
					res.Records[len(res.Records)-1].ChecksumOK = true
				}
			} else {
				if len(res.Records) > 0 {
					r := &res.Records[len(res.Records)-1]
					r.Checksum = v
					res.Err = fmt.Errorf("fastload: checksum mismatch on page $%02X: tape $%02X, computed $%02X", r.Page, v, cksum)
				} else {
					res.Err = fmt.Errorf("fastload: key byte $%02X, want $%02X", v, cksum)
				}
				res.EndPulse = i + 1
				return res
			}
			if len(res.Records) > 0 {
				res.Records[len(res.Records)-1].Checksum = v
			}
		case statePage, stateEndPage:
			if state == stateEndPage && v == 0 {
				res.Terminated = true
				res.EndPulse = i + 1
				return res
			}
			if v == endPage {
				endMode = true
			}
			page = v
			cksum = v
			y = 0
			cur = Record{Page: v, Data: make([]byte, 256), PulseIndex: i}
			state = stateData
		case stateData:
			cur.Data[y] = v
			res.Memory[uint16(page)<<8|uint16(y)] = v
			cksum += v
			y++
			if y == 256 {
				res.Records = append(res.Records, cur)
				state = stateChecksum
			}
		}
		if reset {
			shift = 0x7F
		}
	}
	res.EndPulse = i
	if state != statePilot {
		res.Err = fmt.Errorf("fastload: pulse stream ended mid-block (state %d)", state)
	}
	return res
}

const (
	statePilot = iota
	stateSync
	stateChecksum
	statePage
	stateEndPage
	stateData
)
