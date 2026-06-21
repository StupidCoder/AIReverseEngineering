// trackjson exports the eight decoded circuits to JSON for the web level viewer.
// Geometry comes from package track (the verified pure-Go spine, Part IV §5) — the
// plan-view node X,Z per section, plus the raw section fields for reference. The
// output is written to site/public/stuntcar/tracks.json by default.
//
// Usage: trackjson game.dec.bin [-out path]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"stuntcar/extract/track"
)

var names = []string{
	"Little Ramp", "Stepping Stones", "Hump Back", "Big Ramp",
	"Ski Jump", "Draw Bridge", "High Jump", "Roller Coaster",
}

type outTrack struct {
	Name      string  `json:"name"`
	Sections  int     `json:"sections"`
	FinishIdx int     `json:"finishIdx"`
	// [planX, planY, elev, type, p1, p2, attr] per section. planX/planY = the 16x16
	// grid footprint (the circuit the viewer draws); elev = height above the track
	// reference ($1C650-$1C718).
	Nodes [][]int `json:"nodes"`
}

func main() {
	out := flag.String("out", "../../site/public/stuntcar/tracks.json", "output JSON path")
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: trackjson game.dec.bin [-out path]")
		os.Exit(2)
	}
	img, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "trackjson:", err)
		os.Exit(1)
	}
	im := track.New(img)
	var tracks []outTrack
	for id, name := range names {
		t := im.Spine(id)
		ns := make([][]int, len(t.Nodes))
		for i, n := range t.Nodes {
			ns[i] = []int{n.PlanX, n.PlanY, n.Elev, n.Type, n.P1, n.P2, n.Attr}
		}
		tracks = append(tracks, outTrack{name, t.Sections, t.FinishIdx, ns})
	}
	b, _ := json.Marshal(tracks)
	if err := os.WriteFile(*out, b, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "trackjson:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%d tracks)\n", *out, len(tracks))
}
