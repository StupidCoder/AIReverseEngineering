// tracks dumps the eight built-in race-track byte streams out of the (decrypted)
// game image. It reimplements the engine's own track-pointer lookup from the race
// setup routine $5AE46 (Stunt_Car_Racer.md Part IV):
//
//	id  -> word = trackPtrTable[id*2]      ; table at $1F0A2
//	    -> swap bytes (ROL.w #8)
//	    -> (swap - $B100) & $FFFF          ; 16-bit offset
//	    -> + $1EF82                        ; track-data base => absolute address
//
// The eight tracks are stored contiguously in track-table order, so a track runs
// from its address to the next-higher track address; the last (ROLLER COASTER)
// runs to the first trailing zero padding. Each record begins with 3 header bytes
// followed by the marker 25 00 05 A0 CF, then the section stream the decoder
// consumes via the sequential byte reader $5AE00.
//
// Input is the DECRYPTED image (extract writes extracted/game.dec.bin): the table
// and the data both live below $1AA4A only partially, but the addresses here
// ($1EF82+, $1F0A2) are above the encrypted range, so game.bin and game.dec.bin
// agree — either works. Usage: tracks game.dec.bin [-out dir]
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const (
	base     = 0xE700  // run-time load address of the game image
	ptrTable = 0x1F0A2 // track-pointer table (8 words)
	dataBase = 0x1EF82 // track-data base added to each decoded offset
	bias     = 0xB100  // subtracted from the byte-swapped table word
)

var names = []string{
	"little_ramp", "stepping_stones", "hump_back", "big_ramp",
	"ski_jump", "draw_bridge", "high_jump", "roller_coaster",
}

func main() {
	out := flag.String("out", "", "output directory (default: <image dir>/tracks)")
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: tracks game.dec.bin [-out dir]")
		os.Exit(2)
	}
	img, err := os.ReadFile(flag.Arg(0))
	must(err)
	at := func(addr int) int { return addr - base } // run-time addr -> file offset

	// Decode the eight track addresses exactly as $5AE46 does.
	addrs := make([]int, len(names))
	for id := range names {
		w := int(img[at(ptrTable)+2*id])<<8 | int(img[at(ptrTable)+2*id+1])
		swap := (w<<8 | w>>8) & 0xFFFF
		addrs[id] = ((swap - bias) & 0xFFFF) + dataBase
	}

	// A track runs to the next-higher track address; the last one runs to the
	// first run of >=8 zero bytes (the padding after ROLLER COASTER).
	end := func(start int) int {
		next := 1 << 30
		for _, a := range addrs {
			if a > start && a < next {
				next = a
			}
		}
		if next != 1<<30 {
			return next
		}
		for o := at(start); o+8 <= len(img); o++ {
			if isZero(img[o : o+8]) {
				return base + o
			}
		}
		return base + len(img)
	}

	dir := *out
	if dir == "" {
		dir = filepath.Join(filepath.Dir(flag.Arg(0)), "tracks")
	}
	must(os.MkdirAll(dir, 0o755))

	for id, name := range names {
		s, e := addrs[id], end(addrs[id])
		blob := img[at(s):at(e)]
		fn := fmt.Sprintf("%d_%s.bin", id, name)
		must(os.WriteFile(filepath.Join(dir, fn), blob, 0o644))
		mark := ""
		if len(blob) >= 8 && blob[3] == 0x25 && blob[4] == 0x00 && blob[5] == 0x05 {
			mark = "  hdr 25 00 05 A0 CF ok"
		}
		fmt.Printf("%-16s addr $%05X  %3d bytes  %s%s\n", name, s, len(blob), fn, mark)
	}
}

func isZero(b []byte) bool {
	for _, x := range b {
		if x != 0 {
			return false
		}
	}
	return true
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "tracks:", err)
		os.Exit(1)
	}
}
