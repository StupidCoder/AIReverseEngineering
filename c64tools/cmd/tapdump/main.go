// tapdump summarises a C64 TAP image: a histogram of pulse widths and the
// list of pause-delimited segments. This is the usual first step when
// reverse engineering an unknown tape — it reveals the pulse encoding (how
// many distinct widths, where the thresholds sit) and how the tape is divided
// into files/loads.
//
// Usage: tapdump file.tap
package main

import (
	"fmt"
	"os"
	"sort"

	"stupidcoder.com/c64tools/tap"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: tapdump file.tap")
		os.Exit(2)
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "tapdump:", err)
		os.Exit(1)
	}
	img, err := tap.Parse(raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tapdump:", err)
		os.Exit(1)
	}
	fmt.Printf("TAP v%d, platform $%02X, video $%02X, %d pulses\n",
		img.Version, img.Platform, img.Video, len(img.Pulses))

	// Histogram of data-pulse widths (pauses excluded).
	counts := map[int]int{}
	pauses := 0
	for _, p := range img.Pulses {
		if p.Pause {
			pauses++
			continue
		}
		counts[p.Cycles]++
	}
	widths := make([]int, 0, len(counts))
	for c := range counts {
		widths = append(widths, c)
	}
	sort.Ints(widths)
	fmt.Printf("\npulse-width histogram (%d distinct widths, %d pauses):\n", len(widths), pauses)
	for _, w := range widths {
		fmt.Printf("  %4d cycles ($%02X)  %8d\n", w, (w+4)/8, counts[w])
	}

	segs := tap.Segmentize(img.Pulses)
	fmt.Printf("\n%d pause-delimited segments:\n", len(segs))
	for i, s := range segs {
		first := img.Pulses[s.First]
		fmt.Printf("  seg %2d  pulses %7d..%-7d (%6d)  file offset $%06X\n",
			i, s.First, s.Last, s.Last-s.First+1, first.Offset)
	}
}
