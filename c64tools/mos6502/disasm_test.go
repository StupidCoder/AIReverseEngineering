package mos6502

import "testing"

func TestDisassemble(t *testing.T) {
	code := []byte{
		0xA9, 0x93, // LDA #$93
		0x20, 0xD2, 0xFF, // JSR $FFD2
		0x90, 0x02, // BCC +2
		0x66, 0xA9, // ROR $A9
		0xFF, // undocumented
	}
	want := []string{
		"$080D: A9 93     LDA #$93",
		"$080F: 20 D2 FF  JSR $FFD2",
		"$0812: 90 02     BCC $0816",
		"$0814: 66 A9     ROR $A9",
		"$0816: FF        .byte $FF",
	}
	got := Disassemble(code, 0x080D)
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d:\n got %q\nwant %q", i, got[i], want[i])
		}
	}
}
