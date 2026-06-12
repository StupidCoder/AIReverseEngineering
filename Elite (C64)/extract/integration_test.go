package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// TestEliteExtraction runs the full extraction against the real tape image
// and verifies every output file byte for byte (combined hash). Skipped when
// the image is not present.
func TestEliteExtraction(t *testing.T) {
	const tapPath = "../Elite.tap"
	if _, err := os.Stat(tapPath); err != nil {
		t.Skipf("tape image not present: %v", err)
	}
	outDir := t.TempDir()
	if err := run(tapPath, outDir, 2_000_000_000); err != nil {
		t.Fatalf("run: %v", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	if len(names) != 59 {
		t.Errorf("got %d output files, want 59", len(names))
	}

	h := sha256.New()
	for _, name := range names {
		f, err := os.Open(filepath.Join(outDir, name))
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(h, "%s\n", name)
		if _, err := io.Copy(h, f); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
	got := hex.EncodeToString(h.Sum(nil))

	// Golden hash of the verified 2026-06-11 extraction (57 .prg files +
	// memory_final.bin + report.txt).
	const want = "6f0eaf92baaf0e88b2f15a7c97ed0dad27af15866de2527375f61de5ae7d272d"
	if got != want {
		t.Errorf("combined output hash = %s, want %s", got, want)
	}
}
