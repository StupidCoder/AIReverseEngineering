// disprg disassembles a C64 .prg file (2-byte load address + data).
//
// Usage: disprg [-start addr] [-end addr] file.prg
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"stupidcoder.com/c64tools/mos6502"
)

func parseAddr(s string) (uint16, error) {
	s = strings.TrimPrefix(strings.TrimPrefix(s, "$"), "0x")
	v, err := strconv.ParseUint(s, 16, 16)
	return uint16(v), err
}

func main() {
	startF := flag.String("start", "", "start address (hex), default: load address")
	endF := flag.String("end", "", "end address (hex, exclusive), default: end of file")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: disprg [-start addr] [-end addr] file.prg")
		os.Exit(2)
	}
	raw, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(raw) < 3 {
		fmt.Fprintln(os.Stderr, "disprg: file too short")
		os.Exit(1)
	}
	load := uint16(raw[0]) | uint16(raw[1])<<8
	code := raw[2:]
	start, end := load, load+uint16(len(code))
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
	if start < load || int(start) >= int(load)+len(code) || int(end) > int(load)+len(code) || end <= start {
		fmt.Fprintf(os.Stderr, "disprg: range $%04X-$%04X outside file ($%04X-$%04X)\n",
			start, end, load, int(load)+len(code)-1)
		os.Exit(1)
	}
	for _, l := range mos6502.Disassemble(code[start-load:end-load], start) {
		fmt.Println(l)
	}
}
