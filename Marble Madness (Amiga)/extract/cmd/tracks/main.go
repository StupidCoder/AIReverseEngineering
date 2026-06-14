// tracks decodes the per-course *Track files — the level/object LAYOUT (not music;
// that is *Snd). Each Track is a plain AmigaDOS hunk module that the engine
// LoadSegs at course init (load_track_data $003176); its first segment is a header
// of ten relocated pointers fanned out to the actor-system globals. Header field
// +4 ($129FC) points to the object-placement table: a list of 3-byte records
// [X][Y][type] terminated by $FF, where the engine places object `type` at screen
// (X*8+4, Y*8+4) (consumer $0124EC). This tool prints that table per course.
//
// Usage: tracks <disk.adf>
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"stupidcoder.com/tools/amiga/adf"
	"stupidcoder.com/tools/amiga/hunk"
)

// courses maps the course key to its Track filename (case-insensitive lookup).
var courses = []struct{ key, track string }{
	{"practy", "PrcTrack"}, {"beginr", "BegTrack"}, {"interm", "IntTrack"},
	{"aerial", "AerTrack"}, {"silly", "SilTrack"}, {"ultima", "UltTrack"},
}

func u32(b []byte, o uint32) uint32 {
	if int(o)+4 > len(b) {
		return 0
	}
	return uint32(b[o])<<24 | uint32(b[o+1])<<16 | uint32(b[o+2])<<8 | uint32(b[o+3])
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: tracks <disk.adf>")
		os.Exit(2)
	}
	img, err := os.ReadFile(os.Args[1])
	chk(err)
	vol, err := adf.Open(img)
	chk(err)

	// case-insensitive filename -> path
	paths := map[string]string{}
	chk(vol.Walk(func(e adf.Entry) error {
		if !e.IsDir {
			paths[strings.ToLower(e.Name)] = e.Path
		}
		return nil
	}))

	for _, c := range courses {
		p, ok := paths[strings.ToLower(c.track)]
		if !ok {
			fmt.Printf("\n== %s (%s): not found\n", c.key, c.track)
			continue
		}
		d, err := vol.ReadFile(p)
		if err != nil {
			fmt.Printf("\n== %s (%s): %v\n", c.key, c.track, err)
			continue
		}
		prog, err := hunk.Load(d, 0)
		if err != nil {
			fmt.Printf("\n== %s (%s): hunk load: %v\n", c.key, c.track, err)
			continue
		}
		im := prog.Image
		// header +4 -> placement struct; struct +0 -> the object list.
		listPtr := u32(im, u32(im, 4))
		recs := parsePlacement(im, listPtr)
		byType := map[int]int{}
		for _, r := range recs {
			byType[r[2]]++
		}
		fmt.Printf("\n== %s (%s)  %d objects placed  (list @ $%X)\n", c.key, c.track, len(recs), listPtr)
		fmt.Printf("   per-type counts: %s\n", typeHist(byType))
		fmt.Printf("   %-4s %-4s %-4s   %-9s\n", "idx", "X", "Y", "type")
		for i, r := range recs {
			fmt.Printf("   %-4d %-4d %-4d   %d   (px %d,%d)\n", i, r[0], r[1], r[2], r[0]*8+4, r[1]*8+4)
		}
	}
}

// parsePlacement reads 3-byte [X][Y][type] records until a leading $FF.
func parsePlacement(im []byte, off uint32) [][3]int {
	var out [][3]int
	for int(off)+3 <= len(im) {
		if im[off] == 0xFF {
			break
		}
		out = append(out, [3]int{int(im[off]), int(im[off+1]), int(im[off+2])})
		off += 3
	}
	return out
}

func typeHist(m map[int]int) string {
	ks := make([]int, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	parts := make([]string, len(ks))
	for i, k := range ks {
		parts[i] = fmt.Sprintf("type%d=%d", k, m[k])
	}
	return strings.Join(parts, "  ")
}

func chk(e error) {
	if e != nil {
		panic(e)
	}
}
