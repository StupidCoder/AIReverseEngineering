// shiprender draws Elite's wireframe ships from their decoded blueprints
// (Elite.md Part IV §1) and writes a rotating animated PNG per ship, plus a
// static montage for review. Lines are white on black; hidden edges are
// removed exactly as the game does — an edge is drawn only when at least one
// of the two faces beside it points towards the camera (its surface normal has
// a positive component along the view axis).
//
// Usage: shiprender [-extracted dir] [-o dir] [-size n] [-frames n]
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"

	"elite/extract/shipmodel"
	"stupidcoder.com/c64tools/gfx"
)

var (
	white = color.RGBA{255, 255, 255, 255}
	black = color.RGBA{0, 0, 0, 255}
)

// startAngle puts the first animation frame (the poster shown by non-animating
// viewers) at a 3/4 view rather than nose-on.
const startAngle = 0.9

func main() {
	extracted := flag.String("extracted", "../extracted", "directory of extracted files")
	outDir := flag.String("o", "../rendered", "output directory")
	size := flag.Int("size", 96, "frame size in pixels")
	frames := flag.Int("frames", 36, "animation frames (full turn)")
	still := flag.String("still", "", "debug: render one frame to still.png, as type:frame")
	flag.Parse()
	if *still != "" {
		if err := renderStill(*extracted, *outDir, *size, *frames, *still); err != nil {
			fmt.Fprintln(os.Stderr, "shiprender:", err)
			os.Exit(1)
		}
		return
	}
	if err := run(*extracted, *outDir, *size, *frames); err != nil {
		fmt.Fprintln(os.Stderr, "shiprender:", err)
		os.Exit(1)
	}
}

// renderStill renders a single ship/frame to rendered/still.png for inspection.
func renderStill(extracted, outDir string, size, frames int, spec string) error {
	var typ, frame int
	if _, err := fmt.Sscanf(spec, "%d:%d", &typ, &frame); err != nil {
		return fmt.Errorf("bad -still %q, want type:frame", spec)
	}
	mem, err := shipmodel.LoadEngine(extracted)
	if err != nil {
		return err
	}
	s, err := shipmodel.Parse(mem, typ)
	if err != nil {
		return err
	}
	ang := startAngle + 2*math.Pi*float64(frame)/float64(frames)
	return gfx.WritePNG(filepath.Join(outDir, "still.png"), renderFrame(s, ang, size))
}

func run(extracted, outDir string, size, frames int) error {
	mem, err := shipmodel.LoadEngine(extracted)
	if err != nil {
		return err
	}
	ships := shipmodel.ParseAll(mem)
	shipsDir := filepath.Join(outDir, "ships")
	if err := os.MkdirAll(shipsDir, 0o755); err != nil {
		return err
	}
	for _, s := range ships {
		seq := make([]*image.RGBA, frames)
		for f := 0; f < frames; f++ {
			ang := startAngle + 2*math.Pi*float64(f)/float64(frames)
			seq[f] = renderFrame(s, ang, size)
		}
		p := filepath.Join(shipsDir, fmt.Sprintf("ship-%02d.png", s.Type))
		if err := gfx.WriteAPNG(p, seq, 4, 100); err != nil {
			return err
		}
	}
	montage := makeMontage(ships, 64)
	if err := gfx.WritePNG(filepath.Join(outDir, "ships-montage.png"), montage); err != nil {
		return err
	}
	fmt.Printf("rendered %d ships -> %s/ship-NN.png + ships-montage.png\n", len(ships), shipsDir)
	return nil
}

// renderFrame draws one ship rotated by ang radians about its up (y) axis.
func renderFrame(s *shipmodel.Ship, ang float64, size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for i := range img.Pix {
		img.Pix[i] = 0
	}
	for x := 0; x < size; x++ {
		for y := 0; y < size; y++ {
			img.SetRGBA(x, y, black)
		}
	}
	sin, cos := math.Sincos(ang)

	// model radius for framing
	var r float64
	for _, v := range s.Vertices {
		d := math.Sqrt(float64(v.X*v.X + v.Y*v.Y + v.Z*v.Z))
		if d > r {
			r = d
		}
	}
	if r == 0 {
		return img
	}
	cam := 2.6 * r                          // camera distance along +z
	focal := 0.42 * float64(size) * cam / r // so radius r at z=0 fills ~0.42*size
	cx, cy := float64(size)/2, float64(size)/2

	// View transform: spin about the up (y) axis by ang, then tilt the camera
	// down by a fixed elevation so the ship is shown in 3/4 view rather than
	// straight down its nose axis. The spin is what animates.
	const pitch = 0.42 // camera elevation (radians)
	ps, pc := math.Sincos(pitch)
	rot := func(x, y, z int) (float64, float64, float64) {
		fx, fy, fz := float64(x), float64(y), float64(z)
		x1, y1, z1 := fx*cos+fz*sin, fy, -fx*sin+fz*cos
		return x1, y1*pc - z1*ps, y1*ps + z1*pc
	}
	// project rotated vertices, keeping their rotated 3-D coordinates for the
	// hidden-surface test (camera/eye at (0,0,cam) looking toward -z).
	type pt struct{ x, y int }
	type v3 struct{ x, y, z float64 }
	proj := make([]pt, len(s.Vertices))
	rv := make([]v3, len(s.Vertices))
	for i, v := range s.Vertices {
		rx, ry, rz := rot(v.X, v.Y, v.Z)
		rv[i] = v3{rx, ry, rz}
		proj[i] = pt{int(cx + rx*focal/(cam-rz)), int(cy - ry*focal/(cam-rz))}
	}
	// pick any vertex lying on each face (edges name the faces they border, and
	// both endpoints lie on both bordering faces) to place the face in space.
	faceVert := make([]int, len(s.Faces))
	for i := range faceVert {
		faceVert[i] = -1
	}
	for _, e := range s.Edges {
		if faceVert[e.FaceA] < 0 {
			faceVert[e.FaceA] = e.V1
		}
		if faceVert[e.FaceB] < 0 {
			faceVert[e.FaceB] = e.V1
		}
	}
	// Hidden-surface removal (perspective-correct back-face test): a face is
	// visible when the eye lies on its outward side, dot(normal, eye-P) > 0,
	// for a point P on the face. Using the eye *position* (not a fixed view
	// direction) makes faces appear exactly at the silhouette.
	vis := make([]bool, len(s.Faces))
	for i, f := range s.Faces {
		nx, ny, nz := rot(f.NX, f.NY, f.NZ)
		var px, py, pz float64
		if faceVert[i] >= 0 {
			px, py, pz = rv[faceVert[i]].x, rv[faceVert[i]].y, rv[faceVert[i]].z
		}
		vis[i] = nx*(-px)+ny*(-py)+nz*(cam-pz) > 0
	}
	// draw each edge whose adjacent face(s) are visible
	for _, e := range s.Edges {
		if !(vis[e.FaceA] || vis[e.FaceB]) {
			continue
		}
		gfx.Line(img, proj[e.V1].x, proj[e.V1].y, proj[e.V2].x, proj[e.V2].y, white)
	}
	return img
}

// makeMontage tiles a 3/4 view of every ship into one grid image for review.
func makeMontage(ships []*shipmodel.Ship, cell int) *image.RGBA {
	cols := 8
	rows := (len(ships) + cols - 1) / cols
	img := image.NewRGBA(image.Rect(0, 0, cols*cell, rows*cell))
	for x := 0; x < img.Bounds().Dx(); x++ {
		for y := 0; y < img.Bounds().Dy(); y++ {
			img.SetRGBA(x, y, black)
		}
	}
	for i, s := range ships {
		frame := renderFrame(s, 0.6, cell)
		ox, oy := (i%cols)*cell, (i/cols)*cell
		for x := 0; x < cell; x++ {
			for y := 0; y < cell; y++ {
				img.SetRGBA(ox+x, oy+y, frame.RGBAAt(x, y))
			}
		}
	}
	return img
}
