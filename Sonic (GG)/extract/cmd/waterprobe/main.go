// waterprobe boots each Labyrinth act in the oracle and reads the live water-level
// state the engine computes: $D2DD (water surface world-Y, set each frame by the
// type-$40 water-surface object handler bank2 $8D59), $D2DC (water-line screen
// scanline = waterY - cameraY, used by the IRQ raster split that swaps in the
// underwater BG palette at $0216), and $D257 (camera Y). Confirms the static finding.
package main

import (
	"fmt"
	"os"

	"stupidcoder.com/tools/gamegear"
)

func main() {
	rom, _ := os.ReadFile(os.Args[1])
	for _, act := range []int{9, 10, 11} {
		m := gamegear.NewMachine(rom)
		for i := 0; i < 700; i++ {
			m.RunFrame()
		}
		for round := 0; round < 40; round++ {
			m.Pad00 = 0x7F
			m.Write(0xD238, byte(act))
			for i := 0; i < 8; i++ {
				m.RunFrame()
				m.Write(0xD238, byte(act))
			}
			m.Pad00 = 0xFF
			for k := 0; k < 242; k++ {
				m.Write(0xD238, byte(act))
				m.RunFrame()
			}
		}
		for i := 0; i < 120; i++ {
			m.RunFrame()
		}
		w := func(a uint16) int { return int(m.Read(a)) | int(m.Read(a+1))<<8 }
		fmt.Printf("act %2d (Labyrinth %d): D2DD(waterY)=%d ($%04X) D2DC(scanline)=$%02X D257(camY)=%d sonicY($D402)=%d\n",
			act, act-8, w(0xD2DD), w(0xD2DD), m.Read(0xD2DC), w(0xD257), w(0xD402))
	}
}
