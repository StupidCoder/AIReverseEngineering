// dis68k disassembles a raw Motorola 68000 code blob loaded at a base address.
//
// Unlike a C64 .prg, an Amiga binary is an AmigaDOS "hunk" file rather than a
// flat load image, so dis68k takes the bytes as-is: point it at a file (or, via
// -skip, past a hunk header) and give the load address of the first byte.
//
// Usage: dis68k [-base addr] [-skip n] [-start addr] [-end addr] file
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"stupidcoder.com/tools/m68k"
)

func parseAddr(s string) (uint32, error) {
	s = strings.TrimPrefix(strings.TrimPrefix(s, "$"), "0x")
	v, err := strconv.ParseUint(s, 16, 32)
	return uint32(v), err
}

func main() {
	baseF := flag.String("base", "0", "load address of the first byte (hex)")
	skipF := flag.Int("skip", 0, "skip this many leading bytes (e.g. past a hunk header)")
	startF := flag.String("start", "", "start address (hex), default: base")
	endF := flag.String("end", "", "end address (hex, exclusive), default: end of file")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: dis68k [-base addr] [-skip n] [-start addr] [-end addr] file")
		os.Exit(2)
	}
	raw, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "dis68k:", err)
		os.Exit(1)
	}
	base, err := parseAddr(*baseF)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bad -base:", err)
		os.Exit(2)
	}
	if *skipF < 0 || *skipF > len(raw) {
		fmt.Fprintln(os.Stderr, "dis68k: -skip out of range")
		os.Exit(1)
	}
	code := raw[*skipF:]

	start, end := base, base+uint32(len(code))
	if *startF != "" {
		if start, err = parseAddr(*startF); err != nil {
			fmt.Fprintln(os.Stderr, "bad -start:", err)
			os.Exit(2)
		}
	}
	if *endF != "" {
		if end, err = parseAddr(*endF); err != nil {
			fmt.Fprintln(os.Stderr, "bad -end:", err)
			os.Exit(2)
		}
	}
	if start < base || start >= base+uint32(len(code)) || end > base+uint32(len(code)) || end <= start {
		fmt.Fprintf(os.Stderr, "dis68k: range $%X-$%X outside loaded blob ($%X-$%X)\n",
			start, end, base, base+uint32(len(code)))
		os.Exit(1)
	}
	for _, l := range m68k.Disassemble(code[start-base:end-base], start) {
		fmt.Println(l)
	}
}
