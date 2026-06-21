// spineverify checks the independent Go spine reimplementation (package track)
// against the m68k oracle's output (extracted/spine_<id>.csv from cmd/spineoracle).
// Run cmd/spineoracle first to produce the CSVs.
//
// Usage: spineverify game.dec.bin
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"stuntcar/extract/track"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: spineverify game.dec.bin")
		os.Exit(2)
	}
	img, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "spineverify:", err)
		os.Exit(1)
	}
	im := track.New(img)
	allOK := true
	for id := 0; id < 8; id++ {
		t := im.Spine(id)
		ref, err := loadCSV(fmt.Sprintf("../extracted/spine_%d.csv", id))
		if err != nil {
			fmt.Printf("track %d: no oracle csv (%v)\n", id, err)
			allOK = false
			continue
		}
		mism, first := 0, -1
		for i, n := range t.Nodes {
			if i >= len(ref) {
				break
			}
			if int(n.X) != ref[i][0] || int(n.Z) != ref[i][1] {
				mism++
				if first < 0 {
					first = i
				}
			}
		}
		status := "MATCH"
		if mism != 0 {
			status, allOK = fmt.Sprintf("%d/%d differ (first sec %d: go=%d,%d oracle=%d,%d)",
				mism, len(t.Nodes), first, t.Nodes[first].X, t.Nodes[first].Z, ref[first][0], ref[first][1]), false
		}
		fmt.Printf("track %d: %2d sections  %s\n", id, len(t.Nodes), status)
	}
	if !allOK {
		os.Exit(1)
	}
}

func loadCSV(path string) ([][2]int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out [][2]int
	sc := bufio.NewScanner(f)
	sc.Scan() // header
	for sc.Scan() {
		p := strings.Split(sc.Text(), ",")
		if len(p) != 3 {
			continue
		}
		x, _ := strconv.Atoi(p[1])
		z, _ := strconv.Atoi(p[2])
		out = append(out, [2]int{x, z})
	}
	return out, nil
}
