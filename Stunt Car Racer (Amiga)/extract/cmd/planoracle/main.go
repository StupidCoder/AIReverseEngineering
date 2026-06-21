// planoracle verifies the Go extraction of the per-section plan outline (the local
// (x,z) rail vertex pairs the engine's $5C6C4 reads from the per-type piece-shape)
// against the real engine on the tools/m68k core. $5C6C4 reads two consecutive 16-bit
// LE values at byte offset d2 in the shape and adds the section base ($1BB22/$1BB26),
// rotating by quadrant $1BBF2. Driven in the canonical frame (base 0, quadrant 0) it
// returns the raw local pair, which must match track.planProfile. Per the project rule
// the oracle only verifies; it is never the source of shipped data.
//
// Usage: planoracle game.dec.bin [trackid]
package main

import (
	"fmt"
	"os"
	"strconv"

	"stuntcar/extract/track"
	"stupidcoder.com/tools/m68k"
)

const (
	base     = 0xE700
	loader   = 0x5AE46
	setup    = 0x5FE56
	c5C6C4   = 0x5C6C4
	cCount   = 0x1CA1A
	sentinel = 0xFFFFFE
	stackTop = 0x300000
)

type flatBus struct{ m []byte }

func (b *flatBus) Read(a uint32) byte     { return b.m[a&0xFFFFFF] }
func (b *flatBus) Write(a uint32, v byte) { b.m[a&0xFFFFFF] = v }

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: planoracle game.dec.bin [trackid]")
		os.Exit(2)
	}
	img, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "planoracle:", err)
		os.Exit(1)
	}
	id := 0
	if len(os.Args) >= 3 {
		id, _ = strconv.Atoi(os.Args[2])
	}

	bus := &flatBus{m: make([]byte, 1<<24)}
	copy(bus.m[base:], img)
	c := m68k.NewCPU(bus)
	wr := func(a, v uint32) { bus.Write(a, byte(v>>8)); bus.Write(a+1, byte(v)) }
	call := func(pc uint32, regs map[int]uint32) {
		c.A[7] = stackTop - 4
		r := uint32(sentinel)
		bus.Write(c.A[7], byte(r>>24))
		bus.Write(c.A[7]+1, byte(r>>16))
		bus.Write(c.A[7]+2, byte(r>>8))
		bus.Write(c.A[7]+3, byte(r))
		for reg, v := range regs {
			c.D[reg] = v
		}
		c.PC = pc
		for steps := 0; c.PC != sentinel; steps++ {
			if c.Halted || steps > 5_000_000 {
				fmt.Printf("HALT/cap at $%X\n", c.PC)
				os.Exit(1)
			}
			c.Step()
		}
	}

	goTrack := track.New(img).Spine(id)
	call(loader, map[int]uint32{1: uint32(id)})
	n := int(bus.Read(cCount))
	mism := 0
	for sec := 0; sec < n; sec++ {
		call(setup, map[int]uint32{1: uint32(sec)})
		// canonical frame: zero base, zero quadrant -> $5C6C4 returns the raw local pair.
		wr(0x1BB22, 0)
		wr(0x1BB26, 0)
		bus.Write(0x1BBF2, 0)
		gn := goTrack.Nodes[sec]
		rungs := len(gn.PlanLX)
		for j := 0; j < rungs; j++ {
			for half := 0; half < 2; half++ { // 0=left vertex 2j, 1=right vertex 2j+1
				off := gn.PlanOff + 4*(2*j+half)
				c.D[2] = uint32(off)
				call(c5C6C4, nil)
				ox := int(int16(uint16(bus.Read(0x1BBF6))<<8 | uint16(bus.Read(0x1BBF7))))
				oz := int(int16(uint16(bus.Read(0x1BBF8))<<8 | uint16(bus.Read(0x1BBF9))))
				var gx, gz int
				if half == 0 {
					gx, gz = gn.PlanLX[j], gn.PlanLZ[j]
				} else {
					gx, gz = gn.PlanRX[j], gn.PlanRZ[j]
				}
				if ox != gx || oz != gz {
					mism++
					fmt.Printf("  MISMATCH sec %d rung %d half %d: oracle (%d,%d) go (%d,%d)\n", sec, j, half, ox, oz, gx, gz)
				}
			}
		}
	}
	if mism == 0 {
		fmt.Printf("track %d: OK — plan outline (%d sections) Go == $5C6C4 oracle (exact)\n", id, n)
	} else {
		fmt.Printf("track %d: %d MISMATCHES\n", id, mism)
		os.Exit(1)
	}
}
