package hunk

import (
	"encoding/binary"
	"testing"
)

// TestLoadRelocates builds a one-CODE-hunk file whose first longword is a
// self-relocation, and checks the loader fixes it up to the load base.
func TestLoadRelocates(t *testing.T) {
	var b []byte
	put := func(v uint32) { b = binary.BigEndian.AppendUint32(b, v) }
	put(hunkHeader)
	put(0)              // no resident libraries
	put(1)              // table size
	put(0)              // first hunk
	put(0)              // last hunk
	put(2)              // hunk 0 is 2 longs (8 bytes)
	put(hunkCode)       //
	put(2)              // 2 longs follow
	put(0x00000000)     // offset 0: a pointer into hunk 0 (relocated)
	put(0x4E754E75)     // offset 4: RTS / RTS, just filler
	put(hunkReloc32)    //
	put(1)              // one offset…
	put(0)              // …into hunk 0…
	put(0)              // …at offset 0
	put(0)              // end of reloc table
	put(hunkEnd)

	p, err := Load(b, 0x21000)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Segments) != 1 || p.Segments[0].Kind != "CODE" || p.Segments[0].Base != 0x21000 {
		t.Fatalf("segments = %+v", p.Segments)
	}
	if got := binary.BigEndian.Uint32(p.Image[0:4]); got != 0x21000 {
		t.Errorf("relocated pointer = $%X, want $21000", got)
	}
}

func TestLoadRejectsNonHunk(t *testing.T) {
	if _, err := Load([]byte("not a hunk file at all"), 0); err == nil {
		t.Error("accepted a non-hunk blob")
	}
}
