package tap

import (
	"encoding/binary"
	"testing"
)

func header(version byte, dataLen int) []byte {
	h := append([]byte("C64-TAPE-RAW"), version, 0, 0, 0)
	var l [4]byte
	binary.LittleEndian.PutUint32(l[:], uint32(dataLen))
	return append(h, l[:]...)
}

func TestParsePulsesAndPauseV1(t *testing.T) {
	data := []byte{0x2F, 0x54, 0x00, 0x00, 0x10, 0x00, 0x30}
	img, err := Parse(append(header(1, len(data)), data...))
	if err != nil {
		t.Fatal(err)
	}
	want := []Pulse{
		{Cycles: 0x2F * 8, Offset: 20},
		{Cycles: 0x54 * 8, Offset: 21},
		{Cycles: 0x1000, Pause: true, Offset: 22},
		{Cycles: 0x30 * 8, Offset: 26},
	}
	if len(img.Pulses) != len(want) {
		t.Fatalf("got %d pulses, want %d", len(img.Pulses), len(want))
	}
	for i, w := range want {
		if img.Pulses[i] != w {
			t.Errorf("pulse %d: got %+v, want %+v", i, img.Pulses[i], w)
		}
	}
}

func TestParseV0Overflow(t *testing.T) {
	img, err := Parse(append(header(0, 2), 0x00, 0x10))
	if err != nil {
		t.Fatal(err)
	}
	if !img.Pulses[0].Pause || img.Pulses[0].Cycles != 256*8 {
		t.Errorf("v0 overflow decoded as %+v", img.Pulses[0])
	}
}

func TestParseErrors(t *testing.T) {
	cases := map[string][]byte{
		"short":          {1, 2, 3},
		"bad magic":      append([]byte("NOT-A-TAPEXX"), make([]byte, 8)...),
		"bad length":     append(header(1, 5), 0x2F),
		"truncated gap":  append(header(1, 2), 0x00, 0x10),
		"unsupported v9": append(header(9, 1), 0x2F),
	}
	for name, b := range cases {
		if _, err := Parse(b); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}
