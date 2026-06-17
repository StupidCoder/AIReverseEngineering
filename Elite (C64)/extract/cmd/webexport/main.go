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
	"sort"

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
	Type    int      `json:"type"`
	Name    string   `json:"name"`
	Radius  float64  `json:"radius"`
	Faces   int      `json:"faces"` // face count (for the HUD)
	Verts   [][3]int `json:"verts"`
	Edges   [][2]int `json:"edges"`   // v1, v2
	Tris    [][3]int `json:"tris"`    // triangulated faces, vertex indices
	TriFace []int    `json:"triFace"` // face index per triangle (for the shaded view)
}

type jsonDoc struct {
	Ships []jsonShip `json:"ships"`
}

// triangulate fills each face for the depth buffer. Every Elite face is a flat
// convex outline that may also carry interior detail vertices (low-LOD points,
// and coplanar inner shapes like an engine block — see Elite.md Part IV §1). So
// for each face we collect its candidate vertices, project them onto the face
// plane, and fan-triangulate their 2-D convex hull: the hull is the outline in
// boundary order and naturally drops the interior points (whose edges still
// draw on top of the coplanar fill). Returns the triangles and, in parallel,
// the face index each triangle belongs to (for the optional shaded view).
func triangulate(s *shipmodel.Ship) (tris [][3]int, triFace []int) {
	single := len(s.Faces) == 1
	for f := range s.Faces {
		idx := faceVerts(s, f, single)
		if len(idx) < 3 {
			continue
		}
		nx, ny, nz := faceNormal(s, f, idx)
		if nx == 0 && ny == 0 && nz == 0 {
			continue
		}
		// an orthonormal basis (u, w) in the face plane
		ux, uy, uz := perp(nx, ny, nz)
		wx, wy, wz := unit(cross(nx, ny, nz, ux, uy, uz))
		pts := make([]hullPt, len(idx))
		for i, v := range idx {
			vx, vy, vz := float64(s.Vertices[v].X), float64(s.Vertices[v].Y), float64(s.Vertices[v].Z)
			pts[i] = hullPt{x: vx*ux + vy*uy + vz*uz, y: vx*wx + vy*wy + vz*wz, vi: v}
		}
		hull := convexHull(pts)
		for i := 1; i+1 < len(hull); i++ {
			tris = append(tris, [3]int{hull[0], hull[i], hull[i+1]})
			triFace = append(triFace, f)
		}
	}
	return tris, triFace
}

// faceVerts collects the candidate vertices of face f as the union of two
// incomplete sources (each alone misses some): the endpoints of edges that
// border f, and the vertices whose own face list includes f (capped at four
// per vertex). For a single-face model (the alloy plate, whose verts/edges name
// no face) it falls back to every vertex.
func faceVerts(s *shipmodel.Ship, f int, single bool) []int {
	seen := map[int]bool{}
	var idx []int
	add := func(v int) {
		if !seen[v] {
			seen[v] = true
			idx = append(idx, v)
		}
	}
	for _, e := range s.Edges {
		if e.FaceA == f || e.FaceB == f {
			add(e.V1)
			add(e.V2)
		}
	}
	for v, vert := range s.Vertices {
		for _, vf := range vert.Faces {
			if vf == f {
				add(v)
				break
			}
		}
	}
	if len(idx) < 3 && single {
		idx = idx[:0]
		for v := range s.Vertices {
			idx = append(idx, v)
		}
	}
	return idx
}

// faceNormal returns the face's stored normal, or one computed from three
// non-collinear vertices when the stored normal is zero (the alloy plate).
func faceNormal(s *shipmodel.Ship, f int, idx []int) (float64, float64, float64) {
	nf := s.Faces[f]
	if nf.NX != 0 || nf.NY != 0 || nf.NZ != 0 {
		return float64(nf.NX), float64(nf.NY), float64(nf.NZ)
	}
	p := func(v int) (float64, float64, float64) {
		return float64(s.Vertices[v].X), float64(s.Vertices[v].Y), float64(s.Vertices[v].Z)
	}
	ax, ay, az := p(idx[0])
	for i := 1; i < len(idx); i++ {
		bx, by, bz := p(idx[i])
		for j := i + 1; j < len(idx); j++ {
			cx, cy, cz := p(idx[j])
			nx, ny, nz := cross(bx-ax, by-ay, bz-az, cx-ax, cy-ay, cz-az)
			if nx != 0 || ny != 0 || nz != 0 {
				return nx, ny, nz
			}
		}
	}
	return 0, 0, 0
}

// hullPt is a face vertex projected into the face plane, keeping its global
// vertex index.
type hullPt struct {
	x, y float64
	vi   int
}

// convexHull returns the global vertex indices on the 2-D convex hull of pts in
// boundary order (Andrew's monotone chain). Interior points are dropped.
func convexHull(pts []hullPt) []int {
	if len(pts) < 3 {
		out := make([]int, len(pts))
		for i, p := range pts {
			out[i] = p.vi
		}
		return out
	}
	sort.Slice(pts, func(i, j int) bool {
		if pts[i].x != pts[j].x {
			return pts[i].x < pts[j].x
		}
		return pts[i].y < pts[j].y
	})
	cr := func(o, a, b hullPt) float64 {
		return (a.x-o.x)*(b.y-o.y) - (a.y-o.y)*(b.x-o.x)
	}
	var h []hullPt
	for _, p := range pts { // lower hull
		for len(h) >= 2 && cr(h[len(h)-2], h[len(h)-1], p) <= 0 {
			h = h[:len(h)-1]
		}
		h = append(h, p)
	}
	lower := len(h) + 1
	for i := len(pts) - 2; i >= 0; i-- { // upper hull
		p := pts[i]
		for len(h) >= lower && cr(h[len(h)-2], h[len(h)-1], p) <= 0 {
			h = h[:len(h)-1]
		}
		h = append(h, p)
	}
	out := make([]int, len(h)-1) // last point repeats the first
	for i := 0; i < len(h)-1; i++ {
		out[i] = h[i].vi
	}
	return out
}

func cross(ax, ay, az, bx, by, bz float64) (float64, float64, float64) {
	return ay*bz - az*by, az*bx - ax*bz, ax*by - ay*bx
}

func unit(x, y, z float64) (float64, float64, float64) {
	l := math.Sqrt(x*x + y*y + z*z)
	if l == 0 {
		return 0, 0, 0
	}
	return x / l, y / l, z / l
}

// perp returns a unit vector perpendicular to (nx, ny, nz).
func perp(nx, ny, nz float64) (float64, float64, float64) {
	ax, ay, az := 1.0, 0.0, 0.0
	if math.Abs(nx) > math.Abs(ny) {
		ax, ay, az = 0.0, 1.0, 0.0
	}
	return unit(cross(nx, ny, nz, ax, ay, az))
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
		js.Tris, js.TriFace = triangulate(s)
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
