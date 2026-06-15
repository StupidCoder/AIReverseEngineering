// disz80 linearly disassembles a raw Z80 binary (e.g. one bank of a Sega
// Master System / Game Gear cartridge).
//
// Because the Z80 address space is 16-bit but a cartridge ROM is larger, you
// disassemble a slice of the file and tell the tool where it is mapped:
//
//	disz80 [-off FILEOFF] [-len N] [-base ADDR] rom.gg
//
// -off/-len select the byte range in the file (hex); -base is the Z80 address the
// first selected byte is mapped to (hex, default 0). For example, ROM bank 2 paged
// into slot 2 ($8000) is  -off 0x8000 -len 0x4000 -base 0x8000.
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"stupidcoder.com/tools/z80"
)

func hx(s string) (int, error) {
	s = strings.TrimPrefix(strings.TrimPrefix(s, "$"), "0x")
	v, err := strconv.ParseUint(s, 16, 32)
	return int(v), err
}

func main() {
	offF := flag.String("off", "0", "file offset to start at (hex)")
	lenF := flag.String("len", "", "number of bytes (hex, default: to end of file)")
	baseF := flag.String("base", "0", "Z80 address the first selected byte maps to (hex)")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: disz80 [-off FILEOFF] [-len N] [-base ADDR] rom")
		os.Exit(2)
	}
	raw, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "disz80:", err)
		os.Exit(1)
	}
	off, err := hx(*offF)
	if err != nil || off < 0 || off > len(raw) {
		fmt.Fprintf(os.Stderr, "disz80: bad -off (file is %d bytes)\n", len(raw))
		os.Exit(2)
	}
	n := len(raw) - off
	if *lenF != "" {
		if n, err = hx(*lenF); err != nil || n < 0 || off+n > len(raw) {
			fmt.Fprintf(os.Stderr, "disz80: bad -len (file is %d bytes)\n", len(raw))
			os.Exit(2)
		}
	}
	base, err := hx(*baseF)
	if err != nil {
		fmt.Fprintln(os.Stderr, "disz80: bad -base")
		os.Exit(2)
	}
	for _, l := range z80.Disassemble(raw[off:off+n], uint16(base)) {
		fmt.Println(l)
	}
}
