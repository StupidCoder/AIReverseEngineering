// sections reimplements Stunt Car Racer's track section-stream decoder — the loop
// at $5AED0..$5B106 in the engine (Stunt_Car_Racer.md Part IV §3) — in Go, and
// verifies it against the data: a correct parse consumes a track's bytes *exactly*,
//
//	6 (header) + section stream + 6 (trailer) + 2*trailer[4] + trailer[5] = len(track)
//
// The section stream is one (type, p1, p2, attr) record per loop, with run-length
// compression: a type byte whose low nibble is $F is a marker carrying a run count
// in its high nibble, after which that many sections repeat the saved type with p1
// stepped by the delta helper $5AE0C (+/-$10 or +/-1 per the type's bit7/bit6).
// Types whose low nibble is >= $C take p2/attr from the tables at $5B2B8/$5B2BA
// instead of from the stream.
//
// Usage: sections game.dec.bin
package main

import (
	"flag"
	"fmt"
	"os"
)

const (
	base     = 0xE700
	ptrTable = 0x1F0A2
	dataBase = 0x1EF82
	bias     = 0xB100
	tabB8    = 0x5B2B8 // p2 table for type nibble >= $C, indexed by the nibble
	tabBA    = 0x5B2BA // attr table, same indexing
)

var names = []string{
	"LITTLE RAMP", "STEPPING STONES", "HUMP BACK", "BIG RAMP",
	"SKI JUMP", "DRAW BRIDGE", "HIGH JUMP", "ROLLER COASTER",
}

type section struct{ typ, p1, p2, attr int }

func main() {
	verbose := flag.Bool("v", false, "list every section")
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sections game.dec.bin [-v]")
		os.Exit(2)
	}
	img, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "sections:", err)
		os.Exit(1)
	}
	at := func(a int) int { return a - base }

	// Track addresses via the $5AE46 pointer math (same as cmd/tracks).
	addr := func(id int) int {
		w := int(img[at(ptrTable)+2*id])<<8 | int(img[at(ptrTable)+2*id+1])
		swap := (w<<8 | w>>8) & 0xFFFF
		return ((swap - bias) & 0xFFFF) + dataBase
	}
	allOK := true
	for id, name := range names {
		start := addr(id)
		// Track end: next-higher track address, or (for the last) the byte count
		// is open — we just confirm the parse lands within the image.
		next := 1 << 30
		for j := range names {
			if a := addr(j); a > start && a < next {
				next = a
			}
		}
		blob := img[at(start):]
		if next != 1<<30 {
			blob = img[at(start):at(next)]
		}
		secs, used, tr, perr := parse(blob, img, at)
		status := "OK"
		switch {
		case perr != "":
			status, allOK = "PARSE ERR: "+perr, false
		case next != 1<<30 && used != len(blob):
			status, allOK = fmt.Sprintf("MISMATCH used=%d len=%d", used, len(blob)), false
		case next == 1<<30: // last track: just report
			status = fmt.Sprintf("used=%d (last track)", used)
		}
		fmt.Printf("%-15s $%05X  %2d sections  used %3d/%-3d bytes  trailer[%d %d scen %d/%d]  %s\n",
			name, start, len(secs), used, blobLen(blob, next), tr[0], tr[1], tr[4], tr[5], status)
		if *verbose {
			for i, s := range secs {
				fmt.Printf("    %2d: type $%02X  p1 $%02X  p2 $%02X  attr $%02X\n", i, s.typ, s.p1, s.p2, s.attr)
			}
		}
	}
	if !allOK {
		os.Exit(1)
	}
}

func blobLen(b []byte, next int) int {
	if next == 1<<30 {
		return len(b)
	}
	return len(b)
}

// parse decodes the section stream of one track blob, returning the sections, the
// number of bytes consumed (header+stream+trailer+scenery), and the 6-byte trailer.
func parse(blob, img []byte, at func(int) int) (secs []section, used int, trailer [6]int, errStr string) {
	defer func() {
		if r := recover(); r != nil {
			errStr = fmt.Sprint(r)
		}
	}()
	idx := 0
	read := func() int { b := int(blob[idx]); idx++; return b }

	count := int(blob[0])
	idx = 6 // header: count, finishIdx(x2), seed, 2 position bytes

	delta := func(v, t1A int) int {
		switch {
		case t1A&0x80 != 0 && t1A&0x40 != 0:
			return (v - 1) & 0xFF
		case t1A&0x80 != 0:
			return (v - 0x10) & 0xFF
		case t1A&0x40 != 0:
			return (v + 1) & 0xFF
		default:
			return (v + 0x10) & 0xFF
		}
	}

	run, saveType, prevParam := 0, 0, 0
	for len(secs) < count {
		var typ, t1A, p1 int
		if run != 0 {
			run--
			typ = saveType
			t1A = saveType
			if saveType&0x10 != 0 {
				t1A ^= 0xC0
			}
			p1 = delta(prevParam, t1A)
		} else {
			typ = read()
			t1A = typ
			if typ&0x0F == 0x0F { // run-length marker, not a section
				run = typ >> 4
				continue
			}
			saveType = typ
			p1 = read()
		}
		prevParam = p1

		var p2, attr int
		if nib := t1A & 0x0F; nib >= 0x0C {
			typ &= 0xF0
			p2 = int(img[at(tabB8)+nib])
			attr = int(img[at(tabBA)+nib]) & 0x7F
		} else {
			p2 = read()
			src := p2
			if t1A&0x20 == 0 {
				src = read()
			}
			attr = src & 0x7F
		}
		secs = append(secs, section{typ, p1, p2, attr})
	}

	for i := 0; i < 6; i++ {
		trailer[i] = read()
	}
	idx += 2 * trailer[4] // scenery list 1: pairs
	idx += trailer[5]     // scenery list 2: singles
	used = idx
	return
}
