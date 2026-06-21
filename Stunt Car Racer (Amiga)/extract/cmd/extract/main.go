// extract pulls the program blobs off the Stunt Car Racer disk. The disk has no
// AmigaDOS filesystem (Stunt_Car_Racer.md Part I): it is read by *logical sector*
// (512 bytes; sector N = byte N*512 in the .adf), and the boot block / loader name
// fixed sector ranges. Both blobs are stored raw — there is no compression — so
// extracting them is a straight slice of the image.
//
//   loader : sectors 22..97   (offset $2C00, $9800 bytes)  — the custom track loader
//            the boot block reads and JMPs to.
//   game   : sectors 110..914 (offset $DC00, 805 sectors)  — the whole engine, which
//            the loader reads to $E700 and enters there.
//
// Usage: extract disk.adf [-out dir]   (defaults to ./extracted, beside the .adf)
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const sector = 512

// region is one named blob on the disk, given as a sector range and its run-time
// load address (the address the loader/boot places it at — used as the disasm base).
type region struct {
	name    string
	start   int    // first sector
	count   int    // sector count
	loadAt  uint32 // run-time load address
	desc    string
}

var regions = []region{
	{"loader.bin", 22, 76, 0, "custom track loader (boot reads to AllocMem'd chip RAM, JMPs to it)"},
	{"game.bin", 110, 805, 0xE700, "the game engine + data (loader reads to $E700, entry $E700)"},
}

func main() {
	out := flag.String("out", "", "output directory (default: <adf dir>/extracted)")
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: extract disk.adf [-out dir]")
		os.Exit(2)
	}
	adf, err := os.ReadFile(flag.Arg(0))
	must(err)

	dir := *out
	if dir == "" {
		dir = filepath.Join(filepath.Dir(flag.Arg(0)), "extracted")
	}
	must(os.MkdirAll(dir, 0o755))

	for _, r := range regions {
		off := r.start * sector
		n := r.count * sector
		if off+n > len(adf) {
			fmt.Fprintf(os.Stderr, "extract: %s region [%d..%d] past end of image\n", r.name, r.start, r.start+r.count)
			os.Exit(1)
		}
		p := filepath.Join(dir, r.name)
		must(os.WriteFile(p, adf[off:off+n], 0o644))
		fmt.Printf("%-10s sectors %d..%d  offset $%X  %d bytes  load $%X  — %s\n",
			r.name, r.start, r.start+r.count-1, off, n, r.loadAt, r.desc)
	}
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "extract:", err)
		os.Exit(1)
	}
}
