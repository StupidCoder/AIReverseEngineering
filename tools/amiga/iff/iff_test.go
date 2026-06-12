package iff

import (
	"bytes"
	"testing"
)

func TestUnpackBits(t *testing.T) {
	// ByteRun1: literal run (0x02 = copy 3), replicate run (0xFC = -4 = 5 × 0xAA),
	// literal (0x00 = copy 1).
	src := []byte{0x02, 'A', 'B', 'C', 0xFC, 0xAA, 0x00, 'Z'}
	got := unpackBits(src, 100)
	want := []byte{'A', 'B', 'C', 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 'Z'}
	if !bytes.Equal(got, want) {
		t.Errorf("unpackBits = %v, want %v", got, want)
	}
}

func TestDecodeRejectsNonILBM(t *testing.T) {
	if _, err := DecodeILBM([]byte("not an iff file")); err == nil {
		t.Error("accepted a non-ILBM blob")
	}
}
