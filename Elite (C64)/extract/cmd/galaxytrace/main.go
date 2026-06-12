// galaxytrace runs Elite's actual planet-name generator under emulation to
// trace how a system name is built from a seed, and whether generating a name
// also advances the seed to the next system (i.e. whether name generation is
// the system enumeration). It is a diagnostic for Elite.md Part IV §2.
//
// It runs on a flat 64 KB image of the decrypted engine (so the engine code the
// loader hid under $D000-$EFFF is present as RAM, not I/O), drives the routines
// directly, and captures the characters the name generator prints by trapping
// the character-output routine at $2F21.
package main

import (
	"fmt"
	"os"
	"strings"

	"elite/extract/shipmodel"
	"stupidcoder.com/tools/mos6502"
)

const (
	nameGen  = 0x24CB // planet-name generator (text-token handler)
	charOut  = 0x2F21 // character-output routine — trapped to capture letters
	sentinel = 0xFFF0 // fake return address marking "routine finished"
)

// ram is a flat 64 KB memory: every address is RAM (no I/O), which is what the
// relocated engine code expects when the game banks I/O out.
type ram struct{ m []byte }

func (r *ram) Read(a uint16) byte     { return r.m[a] }
func (r *ram) Write(a uint16, v byte) { r.m[a] = v }

// runName executes the name generator with the 4-byte DORND seed s2..s5 placed
// at $02..$05 and returns the printed name plus the seed bytes left afterwards.
func runName(mem []byte, s2, s3, s4, s5 byte) (string, [4]byte, int) {
	r := &ram{m: append([]byte(nil), mem...)}
	r.m[0x02], r.m[0x03], r.m[0x04], r.m[0x05] = s2, s3, s4, s5
	cpu := mos6502.NewCPU(r)
	cpu.PC = nameGen
	// push the sentinel return address (RTS at the end of the generator lands here)
	ret := uint16(sentinel - 1)
	cpu.Push(byte(ret >> 8))
	cpu.Push(byte(ret))

	var sb strings.Builder
	dornd := 0
	for steps := 0; steps < 100000 && !cpu.Halted; steps++ {
		switch cpu.PC {
		case charOut: // trap: record the character in A, then RTS
			c := cpu.A
			if c >= 0x20 && c < 0x7f {
				sb.WriteByte(c)
			}
			lo, hi := cpu.Pop(), cpu.Pop()
			cpu.PC = (uint16(hi)<<8 | uint16(lo)) + 1
			continue
		case 0x8DBB: // count DORND calls (let it run)
			dornd++
		case sentinel:
			return sb.String(), [4]byte{r.m[2], r.m[3], r.m[4], r.m[5]}, dornd
		}
		cpu.Step()
	}
	return sb.String() + "<halt>", [4]byte{r.m[2], r.m[3], r.m[4], r.m[5]}, dornd
}

func main() {
	mem, err := shipmodel.LoadEngine("../extracted")
	if err != nil {
		fmt.Fprintln(os.Stderr, "galaxytrace:", err)
		os.Exit(1)
	}

	// The galaxy-1 seed (commander block $2621) is $5A4A,$0248,$B753 = bytes
	// 4A 5A 48 02 53 B7. DORND uses 4 seed bytes ($02-$05); try the plausible
	// 4-byte windows to see which produces sensible Elite names.
	gal := []byte{0x4A, 0x5A, 0x48, 0x02, 0x53, 0xB7}
	fmt.Println("== one name per 4-byte seed window of the galaxy-1 seed ==")
	for off := 0; off+4 <= len(gal); off++ {
		s := gal[off : off+4]
		name, after, d := runName(mem, s[0], s[1], s[2], s[3])
		fmt.Printf("  seed $02-05 = %02X %02X %02X %02X -> %-10q  (%d DORND, after=%02X %02X %02X %02X)\n",
			s[0], s[1], s[2], s[3], name, d, after[0], after[1], after[2], after[3])
	}

	// Enumeration test: start from a seed and generate names repeatedly,
	// carrying the advanced seed forward each time. If the names form a varied
	// sequence, name generation is itself the system enumeration.
	fmt.Println("\n== enumeration test: 12 successive names, carrying the seed forward ==")
	cur := [4]byte{gal[0], gal[1], gal[2], gal[3]}
	for i := 0; i < 12; i++ {
		name, after, d := runName(mem, cur[0], cur[1], cur[2], cur[3])
		fmt.Printf("  %2d: %-10q  (%d DORND)  seed->%02X %02X %02X %02X\n", i, name, d, after[0], after[1], after[2], after[3])
		cur = after
	}
}
