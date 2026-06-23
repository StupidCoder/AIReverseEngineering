// objscript disassembles a Super Mario Land object's behaviour script. Each object type
// runs a bytecode program found through the table at $3495 and interpreted by $26AC; this
// prints it with mnemonics (see Super_Mario_Land.md Part V §3). Decoded straight from ROM.
//
//	go run ./cmd/objscript [-rom PATH] [-type NN]   # NN hex; omit to dump all types
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

const scriptTable = 0x3495

// nextStart returns the smallest script-start address strictly greater than `p` (the next
// type's script), so a per-type dump stops at the boundary instead of bleeding onward.
func nextStart(rom []byte, p int) int {
	best := 0x3D00
	for t := 0; t < 0x80; t++ {
		s := word(rom, scriptTable+t*2)
		if s > p && s < best && s >= 0x3500 && s < 0x3D00 {
			best = s
		}
	}
	return best
}

// cmd describes a $F0-$FE opcode: mnemonic and whether it takes an argument byte.
var cmd = map[byte]struct {
	name string
	arg  bool
}{
	0xF0: {"face/flip", true}, 0xF1: {"spawn type", true}, 0xF2: {"set flags(C7)", true},
	0xF3: {"become type", true}, 0xF4: {"set step(C9)", true}, 0xF5: {"rnd spawn", true},
	0xF6: {"wait player", true}, 0xF7: {"spawn proj $27", false}, 0xF8: {"SET FRAME", true},
	0xF9: {"sfx DFF8", true}, 0xFA: {"sfx DFE0", true}, 0xFB: {"if near restart", true},
	0xFC: {"set Y,X=$70", true}, 0xFD: {"sfx DFE8", true}, 0xFE: {"nop", false},
}

func main() {
	rom := flag.String("rom", "../Super Mario Land (World).gb", "ROM path")
	typ := flag.String("type", "", "object type id (hex); omit to dump every type")
	flag.Parse()
	data, err := os.ReadFile(*rom)
	if err != nil {
		fmt.Fprintln(os.Stderr, "objscript:", err)
		os.Exit(1)
	}
	if *typ != "" {
		t, _ := strconv.ParseUint(*typ, 16, 8)
		dump(data, byte(t))
		return
	}
	for t := 0; t < 0x80; t++ {
		p := word(data, scriptTable+t*2)
		if p >= 0x3500 && p < 0x3D00 {
			dump(data, byte(t))
		}
	}
}

func dump(rom []byte, t byte) {
	p := word(rom, scriptTable+int(t)*2)
	end := nextStart(rom, p)
	fmt.Printf("type $%02X  script $%04X:\n", t, p)
	for i := 0; i < 64 && p < end; i++ {
		op := rom[p]
		switch {
		case op == 0xFF:
			fmt.Printf("  $%04X: FF           restart\n", p)
			return
		case op >= 0xF0:
			c := cmd[op]
			if c.arg {
				fmt.Printf("  $%04X: %02X %02X        %s $%02X\n", p, op, rom[p+1], c.name, rom[p+1])
				p += 2
			} else {
				fmt.Printf("  $%04X: %02X           %s\n", p, op, c.name)
				p++
			}
		case op >= 0xE0:
			fmt.Printf("  $%04X: %02X           coast %d frames\n", p, op, op&0x0F)
			p++
		default:
			fmt.Printf("  $%04X: %02X           move vel=$%02X\n", p, op, op)
			p++
		}
	}
}

func word(rom []byte, o int) int { return int(rom[o]) | int(rom[o+1])<<8 }
