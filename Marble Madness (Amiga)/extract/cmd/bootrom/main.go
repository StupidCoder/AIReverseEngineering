// bootrom runs a Kickstart ROM image from the 68000 reset vector on the
// tools/m68k core and dumps the CPU exception/TRAP vector table that Exec's
// cold-start installs in low memory ($8..$BC).
//
// This was written to investigate c/zzz's copy protection (sub_$DAA), which
// keys the decryption table on (vector>>16)&0xFF of those vectors. On a stock
// Kickstart 1.x cold-start every vector points at one ROM handler ($00FC05B4),
// so the byte is a uniform $FC across the whole table. (The values the game's
// decoder actually sees differ — AmigaDOS/the launcher redirect the vectors at
// run time — but this confirms the ROM-page baseline.)
//
// The CPU has no hardware (CIA/custom chips read back as 0), so cold-start
// eventually wanders once it needs timers; the vector table is set well before.
//
// Usage: bootrom kick.rom [-steps N]
package main

import (
	"flag"
	"fmt"
	"os"

	"stupidcoder.com/tools/m68k"
)

// bus maps the ROM at $FC0000 (and mirrored at $F80000 for 512K images) over a
// flat 2 MB of RAM; everything else (CIA, custom registers) reads back as 0.
type bus struct {
	ram []byte
	rom []byte
}

func (b *bus) Read(a uint32) byte {
	switch {
	case a >= 0xFC0000 && int(a-0xFC0000) < len(b.rom):
		return b.rom[a-0xFC0000]
	case a >= 0xF80000 && int(a-0xF80000) < len(b.rom):
		return b.rom[a-0xF80000]
	case int(a) < len(b.ram):
		return b.ram[a]
	}
	return 0
}

func (b *bus) Write(a uint32, v byte) {
	if int(a) < len(b.ram) {
		b.ram[a] = v
	}
}

func main() {
	steps := flag.Int("steps", 5_000_000, "maximum instructions to run")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: bootrom kick.rom [-steps N]")
		os.Exit(2)
	}
	rom, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "bootrom:", err)
		os.Exit(1)
	}

	b := &bus{ram: make([]byte, 0x200000), rom: rom}
	cpu := m68k.NewCPU(b)
	// 68000 reset: SSP from $0, PC from $4 (the ROM is the reset image).
	cpu.A[7] = 0x80000
	cpu.PC = uint32(rom[4])<<24 | uint32(rom[5])<<16 | uint32(rom[6])<<8 | uint32(rom[7])
	fmt.Printf("reset PC=$%06X\n", cpu.PC)

	for i := 0; i < *steps; i++ {
		if cpu.Halted {
			fmt.Printf("stopped at $%06X after %d steps: %s\n", cpu.PC, i, cpu.HaltReason)
			break
		}
		cpu.Step()
	}

	fmt.Println("exception/TRAP vectors $8..$BC:")
	for a := uint32(8); a <= 0xBC; a += 4 {
		v := uint32(b.ram[a])<<24 | uint32(b.ram[a+1])<<16 | uint32(b.ram[a+2])<<8 | uint32(b.ram[a+3])
		fmt.Printf("  $%02X = $%08X  (>>16)&FF = $%02X\n", a, v, (v>>16)&0xFF)
	}
}
