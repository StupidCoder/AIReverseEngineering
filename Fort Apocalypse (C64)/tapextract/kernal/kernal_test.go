package kernal

import (
	"testing"

	"tapextract/tap"
)

const (
	cShort  = 0x30 * 8
	cMedium = 0x42 * 8
	cLong   = 0x56 * 8
)

// encode produces the KERNAL pulse stream for one block (pilot, countdown,
// payload, checksum, end marker), the inverse of the decoder.
func encode(payload []byte, repeat bool) []tap.Pulse {
	var p []tap.Pulse
	add := func(cycles ...int) {
		for _, c := range cycles {
			p = append(p, tap.Pulse{Cycles: c})
		}
	}
	addByte := func(v byte) {
		add(cLong, cMedium) // byte marker
		parity := byte(1)
		for i := 0; i < 8; i++ {
			bit := v >> i & 1
			parity ^= bit
			if bit == 0 {
				add(cShort, cMedium)
			} else {
				add(cMedium, cShort)
			}
		}
		if parity == 0 {
			add(cShort, cMedium)
		} else {
			add(cMedium, cShort)
		}
	}
	for i := 0; i < 100; i++ { // pilot
		add(cShort)
	}
	for i := 9; i >= 1; i-- { // countdown
		v := byte(i)
		if !repeat {
			v |= 0x80
		}
		addByte(v)
	}
	var ck byte
	for _, b := range payload {
		addByte(b)
		ck ^= b
	}
	addByte(ck)
	add(cLong, cShort) // end-of-data
	return p
}

func TestRoundTrip(t *testing.T) {
	payload := []byte{0x01, 0x01, 0x08, 0xF8, 0x08}
	for i := 0; i < 60; i++ {
		payload = append(payload, byte(i*7))
	}
	pulses := append(encode(payload, false), encode(payload, true)...)
	blocks := ScanBlocks(pulses)
	if len(blocks) != 2 {
		t.Fatalf("got %d blocks, want 2", len(blocks))
	}
	for i, b := range blocks {
		if !b.ChecksumOK {
			t.Errorf("block %d: checksum failed", i)
		}
		if b.Repeat != (i == 1) {
			t.Errorf("block %d: repeat=%v", i, b.Repeat)
		}
		if len(b.Payload) != len(payload) {
			t.Fatalf("block %d: %d bytes, want %d", i, len(b.Payload), len(payload))
		}
		for j := range payload {
			if b.Payload[j] != payload[j] {
				t.Fatalf("block %d byte %d: got $%02X want $%02X", i, j, b.Payload[j], payload[j])
			}
		}
	}
}

func TestBadChecksumReported(t *testing.T) {
	pulses := encode([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, false)
	// Corrupt one payload bit: swap a short+medium (0) pair to medium+short (1).
	// Payload byte frames start after pilot(100) + countdown(9*20) pulses.
	i := 100 + 9*20 + 2 // first payload byte, first bit pair
	pulses[i], pulses[i+1] = pulses[i+1], pulses[i]
	// Fix parity by also flipping another bit's pair in the same byte.
	j := i + 2
	pulses[j], pulses[j+1] = pulses[j+1], pulses[j]
	blocks := ScanBlocks(pulses)
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks, want 1", len(blocks))
	}
	if blocks[0].ChecksumOK {
		t.Error("corrupted block passed checksum")
	}
}

func TestParseHeader(t *testing.T) {
	p := make([]byte, 192)
	p[0] = 1
	p[1], p[2] = 0x49, 0xCC
	p[3], p[4] = 0xF8, 0xCC
	copy(p[5:21], "FORT            ")
	p[21] = 0x48
	h, err := ParseHeader(p)
	if err != nil {
		t.Fatal(err)
	}
	if h.Type != 1 || h.StartAddr != 0xCC49 || h.EndAddr != 0xCCF8 {
		t.Errorf("header fields wrong: %+v", h)
	}
	if h.Name != "FORT            " || len(h.Extra) != 171 || h.Extra[0] != 0x48 {
		t.Errorf("name/extra wrong: %q len=%d", h.Name, len(h.Extra))
	}
	if _, err := ParseHeader(p[:100]); err == nil {
		t.Error("short header: expected error")
	}
}
