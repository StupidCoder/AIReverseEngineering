// hunkload loads an AmigaDOS hunk object/executable, prints its segment map,
// and (with -o) writes the flat, relocated image — ready to feed to dis68k or
// codetrace68k with the printed base address of the CODE segment.
//
// Usage: hunkload [-base HEX] [-o image.bin] file
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"stupidcoder.com/tools/amiga/hunk"
)

func main() {
	baseF := flag.String("base", "0", "load address of the first segment (hex)")
	out := flag.String("o", "", "write the flat relocated image here")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: hunkload [-base HEX] [-o image.bin] file")
		os.Exit(2)
	}
	base, err := strconv.ParseUint(strings.TrimPrefix(strings.TrimPrefix(*baseF, "$"), "0x"), 16, 32)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bad -base:", err)
		os.Exit(2)
	}
	data, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "hunkload:", err)
		os.Exit(1)
	}
	prog, err := hunk.Load(data, uint32(base))
	if err != nil {
		fmt.Fprintln(os.Stderr, "hunkload:", err)
		os.Exit(1)
	}
	fmt.Printf("%d segments, image %d bytes (base $%06X)\n", len(prog.Segments), len(prog.Image), prog.Base)
	for i, s := range prog.Segments {
		fmt.Printf("  hunk %2d  %-4s  $%06X..$%06X  (%d bytes)\n", i, s.Kind, s.Base, s.Base+uint32(s.Size), s.Size)
	}
	if *out != "" {
		if err := os.WriteFile(*out, prog.Image, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "hunkload:", err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s\n", *out)
	}
}
