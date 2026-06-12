// Package cbmtape decodes the standard Commodore KERNAL (ROM loader) tape
// encoding from a TAP pulse stream.
//
// Encoding summary (PAL C64, cycles):
//
//	short  ≈ 0x30*8 = 384   medium ≈ 0x42*8 = 528   long ≈ 0x56*8 = 688
//	bit 0        = short+medium pulse pair
//	bit 1        = medium+short pulse pair
//	byte marker  = long+medium
//	end-of-data  = long+short
//	byte frame   = marker, 8 data bits LSB first, odd parity bit
//
// Each record is: pilot (shorts), 9-byte countdown ($89..$81 first copy,
// $09..$01 repeat copy), payload, 1 XOR checksum byte over the payload.
package cbmtape

import (
	"fmt"

	"stupidcoder.com/c64tools/tap"
)

// Pulse classes.
const (
	pShort = iota
	pMedium
	pLong
	pOther
)

// classify maps a pulse duration to short/medium/long with generous
// tolerance; anything outside is pOther (e.g. fastloader pulses, noise).
func classify(cycles int) int {
	switch {
	case cycles >= 288 && cycles < 456:
		return pShort
	case cycles >= 456 && cycles < 608:
		return pMedium
	case cycles >= 608 && cycles < 800:
		return pLong
	default:
		return pOther
	}
}

// Block is one decoded tape record (header or data, first or repeat copy).
type Block struct {
	StartPulse int    // index into the pulse slice where the block's sync begins
	EndPulse   int    // index just past the block
	Countdown  []byte // the 9 sync bytes
	Repeat     bool   // true if this is the repeated (second) copy
	Payload    []byte // block content, checksum byte stripped
	Checksum   byte   // checksum byte as read from tape
	ChecksumOK bool
}

// Header is a parsed 192-byte KERNAL header block.
type Header struct {
	Type      byte // 1: relocatable (BASIC) PRG, 3: absolute PRG, 4: data, 5: EOT
	StartAddr uint16
	EndAddr   uint16
	Name      string // 16 chars, PETSCII
	Extra     []byte // remaining 171 bytes (loaded into the tape buffer at $0351)
}

// ParseHeader interprets a 192-byte header payload.
func ParseHeader(p []byte) (*Header, error) {
	if len(p) != 192 {
		return nil, fmt.Errorf("cbmtape: header payload is %d bytes, want 192", len(p))
	}
	return &Header{
		Type:      p[0],
		StartAddr: uint16(p[1]) | uint16(p[2])<<8,
		EndAddr:   uint16(p[3]) | uint16(p[4])<<8,
		Name:      string(p[5:21]),
		Extra:     p[21:],
	}, nil
}

// ScanBlocks decodes every KERNAL block found in pulses, skipping anything
// that does not decode (other encodings, leaders, pauses).
func ScanBlocks(pulses []tap.Pulse) []Block {
	var blocks []Block
	i := 0
	for i < len(pulses) {
		// Skip to something that could start a byte frame.
		if pulses[i].Pause || classify(pulses[i].Cycles) != pLong {
			i++
			continue
		}
		b, next, ok := decodeBlock(pulses, i)
		if ok {
			blocks = append(blocks, b)
			i = next
		} else {
			i++
		}
	}
	return blocks
}

// decodeBlock tries to decode one block whose first byte marker starts at
// pulse index i. It succeeds only if a valid 9-byte countdown follows.
func decodeBlock(pulses []tap.Pulse, i int) (Block, int, bool) {
	start := i
	var data []byte
	for {
		v, next, status := decodeByte(pulses, i)
		if status == frameEnd || status == frameBad {
			break
		}
		data = append(data, v)
		i = next
		// Bail out early on streams that do not begin with a countdown.
		if len(data) <= 9 {
			want := byte(10 - len(data)) // $09,$08,...
			if data[len(data)-1] != want && data[len(data)-1] != want|0x80 {
				return Block{}, 0, false
			}
		}
	}
	if len(data) < 11 { // countdown + at least 1 payload byte + checksum
		return Block{}, 0, false
	}
	blk := Block{
		StartPulse: start,
		EndPulse:   i,
		Countdown:  data[:9],
		Repeat:     data[0] == 0x09,
		Payload:    data[9 : len(data)-1],
		Checksum:   data[len(data)-1],
	}
	var x byte
	for _, b := range blk.Payload {
		x ^= b
	}
	blk.ChecksumOK = x == blk.Checksum
	return blk, i, true
}

const (
	frameOK = iota
	frameEnd
	frameBad
)

// decodeByte decodes one byte frame starting at pulse index i.
func decodeByte(pulses []tap.Pulse, i int) (byte, int, int) {
	cls := func(j int) int {
		if j >= len(pulses) || pulses[j].Pause {
			return pOther
		}
		return classify(pulses[j].Cycles)
	}
	// Byte marker: long+medium. Long+short means end of data.
	if cls(i) != pLong {
		return 0, i, frameBad
	}
	switch cls(i + 1) {
	case pShort:
		return 0, i + 2, frameEnd
	case pMedium:
		// fall through to data bits
	default:
		return 0, i, frameBad
	}
	i += 2
	var v byte
	var parity byte = 1 // odd parity
	for bit := 0; bit < 9; bit++ {
		a, b := cls(i), cls(i+1)
		i += 2
		var d byte
		switch {
		case a == pShort && b == pMedium:
			d = 0
		case a == pMedium && b == pShort:
			d = 1
		default:
			return 0, i, frameBad
		}
		if bit < 8 {
			v |= d << bit
			parity ^= d
		} else if d != parity {
			return 0, i, frameBad // parity error
		}
	}
	return v, i, frameOK
}
