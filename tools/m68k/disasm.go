package m68k

import (
	"fmt"
	"strings"
)

// Disassemble renders a code blob loaded at base, one line per instruction, in
// "ADDR: WORDS  MNEMONIC operands" form (addresses and raw bytes in hex). It is
// a thin linear driver over Decode; unrecognised words appear as ".dc.w".
func Disassemble(code []byte, base uint32) []string {
	var out []string
	for pc := 0; pc < len(code); {
		in := Decode(code[pc:], base+uint32(pc))
		raw := make([]string, 0, in.Len)
		for i := 0; i < in.Len && pc+i < len(code); i++ {
			raw = append(raw, fmt.Sprintf("%02X", code[pc+i]))
		}
		out = append(out, fmt.Sprintf("$%06X: %-20s %s", base+uint32(pc), strings.Join(raw, " "), in.Text))
		if in.Len <= 0 {
			break
		}
		pc += in.Len
	}
	return out
}
