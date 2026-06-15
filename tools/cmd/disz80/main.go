// disz80 linearly disassembles Z80 code from a Sega Master System / Game Gear
// cartridge. Because the Z80 address space is 16-bit but the ROM is larger and paged
// through the mapper, there are two ways to say which bytes to decode:
//
// Flat mode — disassemble a raw file slice mapped at a Z80 address:
//
//	disz80 [-off FILEOFF] [-len N] [-base ADDR] rom.gg
//
// Bank mode — disassemble a Z80 address range with a given slot configuration, so
// banked code (and its cross-references) decodes against the right banks:
//
//	disz80 -slots b0,b1,b2 -start ADDR [-end ADDR] rom.gg
//
// e.g. the bank-3 dispatcher (bank 3 paged into slot 1):
//
//	disz80 -slots 0,3,2 -start 0x4000 -end 0x4080 rom.gg
//
// All numbers are hex. In bank mode -end defaults to -start + $80.
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"stupidcoder.com/tools/gamegear"
	"stupidcoder.com/tools/z80"
)

func hx(s string) (int, error) {
	s = strings.TrimPrefix(strings.TrimPrefix(s, "$"), "0x")
	v, err := strconv.ParseUint(s, 16, 32)
	return int(v), err
}

func die(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "disz80: "+format+"\n", a...)
	os.Exit(2)
}

func main() {
	offF := flag.String("off", "0", "flat mode: file offset to start at (hex)")
	lenF := flag.String("len", "", "flat mode: number of bytes (hex, default: to end of file)")
	baseF := flag.String("base", "0", "flat mode: Z80 address the first byte maps to (hex)")
	slotsF := flag.String("slots", "", "bank mode: ROM banks in slots 0,1,2 (hex, e.g. 0,3,2)")
	startF := flag.String("start", "", "bank mode: Z80 start address (hex)")
	endF := flag.String("end", "", "bank mode: Z80 end address, exclusive (hex; default start+$80)")
	flag.Parse()
	if flag.NArg() != 1 {
		die("usage: disz80 [-off F -len N -base A | -slots b0,b1,b2 -start A -end A] rom")
	}
	raw, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		die("%v", err)
	}

	if *slotsF != "" {
		bankMode(raw, *slotsF, *startF, *endF)
		return
	}

	// Flat mode.
	off, err := hx(*offF)
	if err != nil || off < 0 || off > len(raw) {
		die("bad -off (file is %d bytes)", len(raw))
	}
	n := len(raw) - off
	if *lenF != "" {
		if n, err = hx(*lenF); err != nil || n < 0 || off+n > len(raw) {
			die("bad -len (file is %d bytes)", len(raw))
		}
	}
	base, err := hx(*baseF)
	if err != nil {
		die("bad -base")
	}
	for _, l := range z80.Disassemble(raw[off:off+n], uint16(base)) {
		fmt.Println(l)
	}
}

func bankMode(rom []byte, slotsArg, startArg, endArg string) {
	parts := strings.Split(slotsArg, ",")
	if len(parts) != 3 {
		die("-slots wants three banks, e.g. 0,3,2")
	}
	var slots [3]int
	for i, p := range parts {
		b, err := hx(strings.TrimSpace(p))
		if err != nil || b < 0 {
			die("bad bank in -slots: %q", p)
		}
		slots[i] = b
	}
	if startArg == "" {
		die("bank mode needs -start")
	}
	start, err := hx(startArg)
	if err != nil || start < 0 || start >= 0xC000 {
		die("bad -start (must be $0000-$BFFF)")
	}
	end := start + 0x80
	if endArg != "" {
		if end, err = hx(endArg); err != nil {
			die("bad -end")
		}
	}
	if end > 0xC000 {
		end = 0xC000
	}
	if end <= start {
		die("-end must be greater than -start")
	}
	view := gamegear.BankView(rom, slots)
	for _, l := range z80.Disassemble(view[start:end], uint16(start)) {
		fmt.Println(l)
	}
}
