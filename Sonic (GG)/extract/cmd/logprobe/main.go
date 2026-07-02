// logprobe: verify the floating log's ($29) live rest position and bob in Jungle 1.
package main

import (
	"fmt"
	"os"

	"stupidcoder.com/tools/gamegear"
)

func main() {
	rom, _ := os.ReadFile(os.Args[1])
	m := gamegear.NewMachine(rom)
	word := func(a uint16) uint16 { return uint16(m.Read(a)) | uint16(m.Read(a+1))<<8 }
	for i := 0; i < 700; i++ {
		m.RunFrame()
	}
	for round := 0; round < 40 && word(0xD26F) == 0; round++ {
		m.Pad00 = 0x7F
		for i := 0; i < 8; i++ {
			m.Write(0xD238, 6)
			m.RunFrame()
		}
		m.Pad00 = 0xFF
		for k := 0; k < 242 && word(0xD26F) == 0; k++ {
			m.Write(0xD238, 6)
			m.RunFrame()
		}
	}
	// park Sonic near the first log (3008,288)
	for i := 0; i < 60; i++ {
		m.RunFrame()
	}
	m.Write(0xD3FD+2, byte(2900&0xFF))
	m.Write(0xD3FD+3, byte(2900>>8))
	m.Write(0xD3FD+5, 44)
	m.Write(0xD3FD+6, 1)
	var slot int = -1
	for f := 0; f < 2200; f++ {
		m.RunFrame()
		if slot < 0 {
			for i := 1; i < 26; i++ {
				b := uint16(0xD3FD + i*26)
				if m.Read(b) == 0x29 && m.Read(b+15)|m.Read(b+16) != 0 {
					slot = i
					fmt.Printf("log slot %d active at f%d\n", i, f)
				}
			}
			continue
		}
		if f%4 == 0 && f < 1900 {
			continue
		}
		b := uint16(0xD3FD + slot*26)
		fmt.Printf("f%4d log=(%d,%d) spr=%04X box=%dx%d\n", f,
			int(uint16(m.Read(b+2))|uint16(m.Read(b+3))<<8),
			int(uint16(m.Read(b+5))|uint16(m.Read(b+6))<<8),
			uint16(m.Read(b+15))|uint16(m.Read(b+16))<<8, m.Read(b+13), m.Read(b+14))
		if f > 1960 {
			break
		}
	}
}
