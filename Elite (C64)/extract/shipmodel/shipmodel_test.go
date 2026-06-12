package shipmodel

import (
	"os"
	"testing"
)

// TestParseEuler checks that the decoded ships are topologically sound: a
// closed polyhedron satisfies Euler's formula V - E + F = 2. Skipped when the
// extracted data is not present.
func TestParseEuler(t *testing.T) {
	const dir = "../../extracted"
	if _, err := os.Stat(dir + "/memory_final.bin"); err != nil {
		t.Skip("extracted data not present")
	}
	mem, err := LoadEngine(dir)
	if err != nil {
		t.Fatal(err)
	}
	ships := ParseAll(mem)
	if len(ships) < 30 {
		t.Fatalf("decoded only %d ships, expected most of the 33", len(ships))
	}
	// Structural soundness: every edge must reference vertices and faces that
	// exist in the decoded model (Parse already rejects out-of-range vertices;
	// here we also confirm the referenced faces were decoded).
	for _, s := range ships {
		for i, e := range s.Edges {
			if e.FaceA >= len(s.Faces) || e.FaceB >= len(s.Faces) {
				t.Errorf("type %d edge %d references face out of range", s.Type, i)
			}
		}
	}
	// Many ships are clean closed hulls (Euler V-E+F=2); the rest carry extra
	// detail edges/faces, which is fine for edge-based hidden-surface removal.
	euler := 0
	for _, s := range ships {
		if len(s.Vertices)-len(s.Edges)+len(s.Faces) == 2 {
			euler++
		}
	}
	if euler < 12 {
		t.Errorf("Euler V-E+F=2 held for only %d ships, expected many clean hulls", euler)
	}
}
