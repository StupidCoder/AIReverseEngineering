package fastload

import (
	"testing"

	"tapextract/tap"
)

const (
	cZero = 0x25 * 8 // 296 cycles, bit 0
	cOne  = 0x54 * 8 // 672 cycles, bit 1
)

type stream struct{ p []tap.Pulse }

func (s *stream) bit(b byte) {
	c := cZero
	if b != 0 {
		c = cOne
	}
	s.p = append(s.p, tap.Pulse{Cycles: c})
}

func (s *stream) byte(v byte) { // LSB first
	for i := 0; i < 8; i++ {
		s.bit(v >> i & 1)
	}
}

// encode builds a complete fastloader stream for the given page records.
func encode(pages map[byte][]byte, order []byte) []tap.Pulse {
	var s stream
	for i := 0; i < 2000; i++ { // pilot: zero bits
		s.bit(0)
	}
	s.bit(1)     // pilot terminator, read as byte $80
	s.byte(0xAA) // sync
	s.byte(0x55) // key (initial checksum seed)
	for _, pg := range order {
		s.byte(pg)
		ck := pg
		for _, b := range pages[pg] {
			s.byte(b)
			ck += b
		}
		s.byte(ck)
	}
	s.byte(0x00) // end page (valid only after the $F0 record)
	return s.p
}

func page(fill byte) []byte {
	d := make([]byte, 256)
	for i := range d {
		d[i] = fill + byte(i)
	}
	return d
}

func TestRoundTrip(t *testing.T) {
	pages := map[byte][]byte{0x70: page(1), 0xF0: page(9), 0x71: page(0x40)}
	order := []byte{0x70, 0xF0, 0x71} // loading continues after the $F0 record
	res := Decode(encode(pages, order), 0)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if !res.Terminated {
		t.Error("stream not terminated")
	}
	if len(res.Records) != 3 {
		t.Fatalf("got %d records, want 3", len(res.Records))
	}
	for i, r := range res.Records {
		if r.Page != order[i] || !r.ChecksumOK {
			t.Errorf("record %d: page $%02X checksum_ok=%v", i, r.Page, r.ChecksumOK)
		}
	}
	for pg, want := range pages {
		for i, b := range want {
			if got := res.Memory[uint16(pg)<<8|uint16(i)]; got != b {
				t.Fatalf("memory $%02X%02X: got $%02X want $%02X", pg, i, got, b)
			}
		}
	}
	// $70 and $71 are contiguous, $F0 separate.
	rg := res.Ranges()
	if len(rg) != 2 || rg[0].Start != 0x7000 || len(rg[0].Data) != 512 || rg[1].Start != 0xF000 {
		t.Errorf("ranges wrong: %d ranges, first=$%04X len=%d", len(rg), rg[0].Start, len(rg[0].Data))
	}
}

func TestChecksumError(t *testing.T) {
	pulses := encode(map[byte][]byte{0xF0: page(0)}, []byte{0xF0})
	// Corrupt one data bit (a 0 to a 1) inside the page data.
	i := 2001 + 16 + 8 + 42 // pilot+term, sync+key, page byte, somewhere in data
	for pulses[i].Cycles != cZero {
		i++
	}
	pulses[i].Cycles = cOne
	res := Decode(pulses, 0)
	if res.Err == nil {
		t.Fatal("corrupted stream decoded without error")
	}
	if len(res.Records) != 1 || res.Records[0].ChecksumOK {
		t.Error("corrupted record marked checksum_ok")
	}
}

func TestPilotResync(t *testing.T) {
	// Garbage bits before the pilot must not prevent sync.
	var s stream
	for _, b := range []byte{1, 1, 0, 1, 0, 0, 1, 1, 1, 0, 1} {
		s.bit(b)
	}
	pulses := append(s.p, encode(map[byte][]byte{0xF0: page(7)}, []byte{0xF0})...)
	res := Decode(pulses, 0)
	if res.Err != nil || !res.Terminated || len(res.Records) != 1 {
		t.Fatalf("resync failed: err=%v terminated=%v records=%d", res.Err, res.Terminated, len(res.Records))
	}
}
