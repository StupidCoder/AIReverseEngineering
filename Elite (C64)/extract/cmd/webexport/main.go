// webexport decodes every Elite wireframe ship blueprint (Elite.md Part IV §1)
// and writes the geometry as JSON for the three.js ship viewer on the companion
// website. Each ship carries its vertices, its edges (with the two faces beside
// each edge), and its face normals — everything the viewer needs to reproduce
// Elite's own hidden-line removal: an edge is drawn only when at least one of
// its bordering faces points toward the camera.
//
// Ship type → name is the documented Commodore 64 Elite blueprint table (XX21);
// the names live only in the manual, never in the program (Elite.md Part V §1),
// and the numbering is anchored by our own write-up's Thargoid = type $1D = 29.
//
// Usage: webexport [-extracted dir] [-o dir]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"elite/extract/shipmodel"
)

// shipNames maps each XX21 blueprint slot to the ship every Elite player knows.
// Source: the Commodore 64 Elite ship table (elite.bbcelite.com). Slots whose
// blueprint does not decode (e.g. 8 Splinter) simply never appear in the output.
var shipNames = map[int]string{
	1:  "Missile",
	2:  "Coriolis space station",
	3:  "Escape pod",
	4:  "Alloy plate",
	5:  "Cargo canister",
	6:  "Boulder",
	7:  "Asteroid",
	8:  "Splinter",
	9:  "Shuttle",
	10: "Transporter",
	11: "Cobra Mk III",
	12: "Python",
	13: "Boa",
	14: "Anaconda",
	15: "Rock hermit",
	16: "Viper",
	17: "Sidewinder",
	18: "Mamba",
	19: "Krait",
	20: "Adder",
	21: "Gecko",
	22: "Cobra Mk I",
	23: "Worm",
	24: "Cobra Mk III (pirate)",
	25: "Asp Mk II",
	26: "Python (pirate)",
	27: "Fer-de-Lance",
	28: "Moray",
	29: "Thargoid",
	30: "Thargon",
	31: "Constrictor",
	32: "Cougar",
	33: "Dodo space station",
}

// jsonShip is one ship for the three.js viewer. Edges are drawn as white lines;
// tris fill each face into the depth buffer (invisibly) so the GPU hides the
// edges behind them — robust hidden-line removal without a per-face normal test.
type jsonShip struct {
	Type   int      `json:"type"`
	Name   string   `json:"name"`
	Radius float64  `json:"radius"`
	Faces  int      `json:"faces"` // face count (for the HUD)
	Verts  [][3]int `json:"verts"`
	Edges  [][2]int `json:"edges"` // v1, v2
	Tris   [][3]int `json:"tris"`  // triangulated faces, vertex indices
}

type jsonDoc struct {
	Ships []jsonShip `json:"ships"`
}

// triangulate rebuilds each face as a fan of triangles over the model's
// vertices, used only to fill the depth buffer (so winding is irrelevant). A
// face is a flat convex polygon bounded by its edges (Elite.md Part IV §1); we
// recover the polygon by walking those edges into a vertex loop, then fan it.
// Walking the boundary needs no surface normal, which matters for the alloy
// plate — a single face with a zero normal whose four edges carry only the $F
// "always-draw" sentinel (so for a single-face model every edge bounds it).
func triangulate(s *shipmodel.Ship) [][3]int {
	var tris [][3]int
	single := len(s.Faces) == 1
	for f := range s.Faces {
		// adjacency among the vertices joined by this face's boundary edges
		adj := map[int][]int{}
		for _, e := range s.Edges {
			if single || e.FaceA == f || e.FaceB == f {
				adj[e.V1] = append(adj[e.V1], e.V2)
				adj[e.V2] = append(adj[e.V2], e.V1)
			}
		}
		if len(adj) < 3 {
			continue
		}
		// walk the loop: from any vertex, keep stepping to the neighbour we did
		// not just come from until we return to the start.
		var start int
		for v := range adj {
			start = v
			break
		}
		loop := []int{start}
		prev, cur := -1, start
		for {
			next := -1
			for _, nb := range adj[cur] {
				if nb != prev {
					next = nb
					break
				}
			}
			if next == -1 || next == start {
				break
			}
			loop = append(loop, next)
			prev, cur = cur, next
			if len(loop) > len(adj) {
				break // not a simple cycle; bail out
			}
		}
		if len(loop) < 3 {
			continue
		}
		for i := 1; i+1 < len(loop); i++ {
			tris = append(tris, [3]int{loop[0], loop[i], loop[i+1]})
		}
	}
	return tris
}

func main() {
	extracted := flag.String("extracted", "../extracted", "directory of extracted files")
	outDir := flag.String("o", "../../site/public/elite", "output directory")
	flag.Parse()
	if err := run(*extracted, *outDir); err != nil {
		fmt.Fprintln(os.Stderr, "webexport:", err)
		os.Exit(1)
	}
}

func run(extracted, outDir string) error {
	mem, err := shipmodel.LoadEngine(extracted)
	if err != nil {
		return err
	}
	ships := shipmodel.ParseAll(mem)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	doc := jsonDoc{}
	for _, s := range ships {
		js := jsonShip{Type: s.Type, Name: shipNames[s.Type]}
		if js.Name == "" {
			js.Name = fmt.Sprintf("Ship %d", s.Type)
		}
		var r float64
		for _, v := range s.Vertices {
			js.Verts = append(js.Verts, [3]int{v.X, v.Y, v.Z})
			if d := math.Hypot(math.Hypot(float64(v.X), float64(v.Y)), float64(v.Z)); d > r {
				r = d
			}
		}
		js.Radius = r
		js.Faces = len(s.Faces)
		for _, e := range s.Edges {
			js.Edges = append(js.Edges, [2]int{e.V1, e.V2})
		}
		js.Tris = triangulate(s)
		doc.Ships = append(doc.Ships, js)
	}

	out := filepath.Join(outDir, "ships.json")
	b, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %d ships -> %s\n", len(doc.Ships), out)
	return nil
}
