package main

import (
	"os"
	"testing"

	"fortapoc/extract/fastload"
	"stupidcoder.com/c64tools/cbmtape"
	"stupidcoder.com/c64tools/tap"
)

// TestFortApocalypse verifies extraction of the real tape image end to end.
// Skipped when the image is not present.
func TestFortApocalypse(t *testing.T) {
	raw, err := os.ReadFile("../Fort_Apocalypse.tap")
	if err != nil {
		t.Skip("Fort_Apocalypse.tap not available:", err)
	}
	img, err := tap.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	blocks := cbmtape.ScanBlocks(img.Pulses)
	if len(blocks) != 4 {
		t.Fatalf("got %d kernal blocks, want 4 (header+data, two copies each)", len(blocks))
	}
	for i, b := range blocks {
		if !b.ChecksumOK {
			t.Errorf("kernal block %d: checksum failed", i)
		}
	}
	h, err := cbmtape.ParseHeader(blocks[0].Payload)
	if err != nil {
		t.Fatal(err)
	}
	if h.Type != 1 || h.Name[:4] != "FORT" {
		t.Errorf("header: type=%d name=%q", h.Type, h.Name)
	}
	// The loader's IRQ handler hidden in the header starts PHA TYA PHA LDA $DC05.
	if h.Extra[0] != 0x48 || h.Extra[1] != 0x98 || h.Extra[3] != 0xAD {
		t.Errorf("header extra does not contain the loader IRQ handler: % X", h.Extra[:6])
	}
	// The data block is the BASIC stub: link, line 0, SYS token.
	d := blocks[2].Payload
	if d[2] != 0 || d[3] != 0 || d[4] != 0x9E {
		t.Errorf("data block is not a '0 SYS' BASIC stub: % X", d[:8])
	}

	res := fastload.Decode(img.Pulses, blocks[3].EndPulse)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if !res.Terminated {
		t.Error("fastload stream did not terminate cleanly")
	}
	if len(res.Records) != 84 {
		t.Errorf("got %d fastload records, want 84", len(res.Records))
	}
	for _, r := range res.Records {
		if !r.ChecksumOK {
			t.Errorf("fastload page $%02X: checksum failed", r.Page)
		}
	}
	wantRanges := []struct {
		start uint16
		size  int
	}{
		{0x7000, 0x4900}, // main game, $7000-$B8FF
		{0xE000, 0x0700}, // stage-2 code, $E000-$E6FF
		{0xEE00, 0x0400}, // $EE00-$F1FF
	}
	rgs := res.Ranges()
	if len(rgs) != len(wantRanges) {
		t.Fatalf("got %d memory ranges, want %d", len(rgs), len(wantRanges))
	}
	for i, w := range wantRanges {
		if rgs[i].Start != w.start || len(rgs[i].Data) != w.size {
			t.Errorf("range %d: $%04X len $%04X, want $%04X len $%04X",
				i, rgs[i].Start, len(rgs[i].Data), w.start, w.size)
		}
	}
	// The first fastloaded bytes at $E000 are JMP $E008.
	if res.Memory[0xE000] != 0x4C || res.Memory[0xE002] != 0xE0 {
		t.Errorf("$E000 is not JMP $E0xx: %02X %02X %02X",
			res.Memory[0xE000], res.Memory[0xE001], res.Memory[0xE002])
	}
}
