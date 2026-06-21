// spineoracle executes the engine's real track loader $5AE46 on the tools/m68k
// CPU core, over a flat memory image holding the decrypted game at $E700, and reads
// out the spine arrays it computes ($1C650 = X, $1C718 = Z, one signed word per
// section). It is a *guide and verifier* for the Go reimplementation of the track
// geometry (Stunt_Car_Racer.md Part IV) — per the project rule, the reimplementation
// is independent Go and the oracle only confirms it, never the source of shipped data.
//
// $5AE46 takes only d1 = track id (0..7); it self-initialises the rest from the track
// header. We give it a stack and a sentinel return address and step until it returns.
//
// Usage: spineoracle game.dec.bin [trackid]   (default: all eight)
package main

import (
	"fmt"
	"os"
	"strconv"

	"stupidcoder.com/tools/m68k"
)

const (
	base     = 0xE700
	loader   = 0x5AE46
	cA650    = 0x1C650 // per-section X (word)
	cA718    = 0x1C718 // per-section Z (word)
	cCount   = 0x1CA1A // section count (byte) set by the loader from the header
	sentinel = 0xFFFFFE
	stackTop = 0x300000
)

// flatBus is a 24-bit flat address space; custom-chip writes ($DFFxxx) just land in
// RAM harmlessly since the loader's geometry path never reads them back.
type flatBus struct{ m []byte }

func (b *flatBus) Read(a uint32) byte     { return b.m[a&0xFFFFFF] }
func (b *flatBus) Write(a uint32, v byte) { b.m[a&0xFFFFFF] = v }

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: spineoracle game.dec.bin [trackid]")
		os.Exit(2)
	}
	img, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "spineoracle:", err)
		os.Exit(1)
	}
	ids := []int{0, 1, 2, 3, 4, 5, 6, 7}
	if len(os.Args) >= 3 {
		n, _ := strconv.Atoi(os.Args[2])
		ids = []int{n}
	}
	for _, id := range ids {
		runTrack(img, id)
	}
}

func runTrack(img []byte, id int) {
	bus := &flatBus{m: make([]byte, 1<<24)}
	copy(bus.m[base:], img)
	c := m68k.NewCPU(bus)
	c.A[7] = stackTop
	// push sentinel return address so the loader's final RTS lands on it
	c.A[7] -= 4
	ret := uint32(sentinel)
	bus.Write(c.A[7], byte(ret>>24))
	bus.Write(c.A[7]+1, byte(ret>>16))
	bus.Write(c.A[7]+2, byte(ret>>8))
	bus.Write(c.A[7]+3, byte(ret))
	c.PC = loader
	c.D[1] = uint32(id)

	steps := 0
	var segLens []byte // value written to $1BB6A at $5FF5A, per section
	for c.PC != sentinel {
		if c.Halted {
			fmt.Printf("track %d: HALTED at $%X — %s\n", id, c.PC, c.HaltReason)
			return
		}
		if steps > 20_000_000 {
			fmt.Printf("track %d: step cap (pc=$%X)\n", id, c.PC)
			return
		}
		if c.PC == 0x5FF5A { // MOVE.b d0,$1BB6A — d0 is the segment length
			segLens = append(segLens, byte(c.D[0]))
		}
		c.Step()
		steps++
	}
	if len(os.Getenv("SEGLENS")) > 0 {
		fmt.Printf("track %d segLens(%d): %v\n", id, len(segLens), segLens)
	}

	n := int(bus.Read(cCount))
	rd := func(addr uint32) int16 { return int16(uint16(bus.Read(addr))<<8 | uint16(bus.Read(addr+1))) }
	f, _ := os.Create(fmt.Sprintf("../extracted/spine_%d.csv", id))
	defer f.Close()
	fmt.Fprintln(f, "sec,x,z")
	for i := 0; i < n; i++ {
		fmt.Fprintf(f, "%d,%d,%d\n", i, rd(uint32(cA650+2*i)), rd(uint32(cA718+2*i)))
	}
	fmt.Printf("track %d: %d sections, %d steps -> spine_%d.csv\n", id, n, steps, id)
}
