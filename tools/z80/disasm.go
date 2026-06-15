// Package z80 is a small, generic Zilog Z80 toolkit shared across the projects in
// this repository: a table-driven disassembler (this file + decode.go) that mirrors
// the mos6502 and m68k packages. It operates on raw Z80 code and makes no
// assumptions about where that code came from (cartridge banking, the Game Gear
// memory map, etc. are the caller's concern).
//
// The Z80 has a 64 KB address space and instructions of one to four bytes. The
// opcode space is reached through prefixes — CB (rotate/shift, BIT/RES/SET),
// ED (block ops, 16-bit arithmetic, I/O), and DD/FD (which remap HL→IX/IY and
// (HL)→(IX+d)/(IY+d), including the awkward DD CB d op displacement-first form).
// decode.go documents the regular x/y/z/p/q bit-field decoding it all rests on.
package z80

import (
	"fmt"
	"strings"
)

// Disassemble renders a code blob loaded at base, one line per instruction, in
// "ADDR: BYTES  MNEMONIC operands" form (hex). It is a thin linear driver over
// Decode; undefined opcodes and truncated reads appear as ".byte".
func Disassemble(code []byte, base uint16) []string {
	var out []string
	for pc := 0; pc < len(code); {
		in := Decode(code[pc:], base+uint16(pc))
		raw := make([]string, 0, in.Len)
		for i := 0; i < in.Len && pc+i < len(code); i++ {
			raw = append(raw, fmt.Sprintf("%02X", code[pc+i]))
		}
		out = append(out, fmt.Sprintf("$%04X: %-12s %s", base+uint16(pc), strings.Join(raw, " "), in.Text))
		if in.Len <= 0 {
			break
		}
		pc += in.Len
	}
	return out
}
