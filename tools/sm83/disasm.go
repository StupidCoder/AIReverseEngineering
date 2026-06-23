// Package sm83 is a small, generic disassembler for the Sharp LR35902 — the Game
// Boy's CPU (often called "GBZ80" or SM83) — shared across the projects in this
// repository, mirroring the mos6502, m68k and z80 packages. It operates on raw
// LR35902 code and makes no assumptions about where that code came from (cartridge
// banking, the Game Boy memory map, etc. are the caller's concern).
//
// The LR35902 is a relative of the Z80 but NOT the same chip, which is exactly why
// the z80 package cannot be reused here:
//
//   - no IX/IY index registers and no DD/FD/ED prefix pages — only the CB prefix
//     (rotate/shift, BIT/RES/SET) survives;
//   - no IN/OUT ports — I/O is memory-mapped and reached through the Game-Boy-only
//     high-page ops LDH ($FF00+n)/(C) (opcodes $E0/$F0/$E2/$F2);
//   - several opcodes are repurposed from the Z80: LD (HL+)/(HL-) auto-inc/dec
//     loads, LD ($nnnn),SP, LD ($nnnn),A, ADD SP,e, LD HL,SP+e, RETI, STOP, and the
//     CB-page SWAP (where the Z80 has the undocumented SLL);
//   - eleven opcodes ($D3 $DB $DD $E3 $E4 $EB $EC $ED $F4 $FC $FD) are illegal and
//     lock up the real CPU; they decode here as ".byte" with FlowStop.
//
// decode.go documents the regular x/y/z/p/q bit-field decoding it all rests on.
package sm83

import (
	"fmt"
	"strings"
)

// Disassemble renders a code blob loaded at base, one line per instruction, in
// "ADDR: BYTES  MNEMONIC operands" form (hex). It is a thin linear driver over
// Decode; illegal opcodes and truncated reads appear as ".byte".
func Disassemble(code []byte, base uint16) []string {
	var out []string
	for pc := 0; pc < len(code); {
		in := Decode(code[pc:], base+uint16(pc))
		raw := make([]string, 0, in.Len)
		for i := 0; i < in.Len && pc+i < len(code); i++ {
			raw = append(raw, fmt.Sprintf("%02X", code[pc+i]))
		}
		out = append(out, fmt.Sprintf("$%04X: %-10s %s", base+uint16(pc), strings.Join(raw, " "), in.Text))
		if in.Len <= 0 {
			break
		}
		pc += in.Len
	}
	return out
}
