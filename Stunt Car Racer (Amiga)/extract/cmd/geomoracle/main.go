// geomoracle verifies the Go transcription of the engine's vertex-height builder
// ($5C0AA) and the per-section vertex-count setup ($5FE56) against the real engine on
// the tools/m68k core. It runs the track loader $5AE46 (to populate the section arrays
// and the $1C650/$1C718 rail accumulators), then for each section calls $5FE56 to set up
// the per-section globals and $5C0AA for each vertex index, exactly as the in-race
// renderer $5A186 does. The output (per-section left/right rail height profiles) is the
// ground truth the Go reimplementation in package track must match. Per the project rule
// the oracle only verifies; it is never the source of shipped data.
//
// Usage: geomoracle game.dec.bin [trackid]
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
	setup    = 0x5FE56 // per-section setup: d1 = section
	vbuild   = 0x5C0AA // vertex height: d1 = vertex index; returns d0.w
	cCount   = 0x1CA1A // section count (byte)
	cVcount  = 0x1BB97 // vertices in current section (set by $5FE56)
	sentinel = 0xFFFFFE
	stackTop = 0x300000
)

type flatBus struct{ m []byte }

func (b *flatBus) Read(a uint32) byte     { return b.m[a&0xFFFFFF] }
func (b *flatBus) Write(a uint32, v byte) { b.m[a&0xFFFFFF] = v }

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: geomoracle game.dec.bin [trackid]")
		os.Exit(2)
	}
	img, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "geomoracle:", err)
		os.Exit(1)
	}
	id := 1
	if len(os.Args) >= 3 {
		id, _ = strconv.Atoi(os.Args[2])
	}

	bus := &flatBus{m: make([]byte, 1<<24)}
	copy(bus.m[base:], img)
	c := m68k.NewCPU(bus)

	call := func(pc uint32, regs map[int]uint32) uint32 {
		c.A[7] = stackTop - 4
		ret := uint32(sentinel)
		bus.Write(c.A[7], byte(ret>>24))
		bus.Write(c.A[7]+1, byte(ret>>16))
		bus.Write(c.A[7]+2, byte(ret>>8))
		bus.Write(c.A[7]+3, byte(ret))
		for r, v := range regs {
			c.D[r] = v
		}
		c.PC = pc
		for steps := 0; c.PC != sentinel; steps++ {
			if c.Halted {
				fmt.Printf("HALT at $%X: %s\n", c.PC, c.HaltReason)
				os.Exit(1)
			}
			if steps > 5_000_000 {
				fmt.Printf("step cap at $%X\n", c.PC)
				os.Exit(1)
			}
			c.Step()
		}
		return c.D[0]
	}

	// Ground truth from the independent Go reimplementation (package track).
	goTrack := track.New(img).Spine(id)
	mism := 0

	// Run the loader to populate the section arrays + rail accumulators.
	call(loader, map[int]uint32{1: uint32(id)})

	n := int(bus.Read(cCount))
	fmt.Printf("track %d: %d sections (oracle $5FE56/$5C0AA)\n", id, n)
	rdw := func(a uint32) uint16 { return uint16(bus.Read(a))<<8 | uint16(bus.Read(a+1)) }
	for sec := 0; sec < n; sec++ {
		call(setup, map[int]uint32{1: uint32(sec)})
		p2 := bus.Read(0x1C4C0 + uint32(sec))
		attr := bus.Read(0x1C524 + uint32(sec))
		typ := bus.Read(0x1C5EC + uint32(sec))
		if os.Getenv("V") != "" {
			fmt.Printf("  [sec %2d] p2=%02x attr=%02x type=%02x  $1BB79=%02x $1BC8C=%04x $1BC90=%04x base650=%d base718=%d\n",
				sec, p2, attr, typ, bus.Read(0x1BB79), rdw(0x1BC8C), rdw(0x1BC90), int16(rdw(0x1BC0E)), int16(rdw(0x1BC10)))
		}
		cnt := int(bus.Read(cVcount))
		rungs := cnt / 2
		// Replicate the renderer ($5A1D4-$5A1EE): load a4/a5 from the handle words
		// $1BC8C/$1BC90 that $5FE56 just stored ($5C0AA reads them via registers).
		hdl := func(w uint16) uint32 {
			return uint32((((uint32(w)<<8|uint32(w)>>8)&0xFFFF)-0xB100)&0xFFFF) + 0x1EF82
		}
		a4 := hdl(rdw(0x1BC8C))
		a5 := hdl(rdw(0x1BC90))
		gn := goTrack.Nodes[sec]
		if len(gn.HeightL) != rungs {
			mism++
			fmt.Printf("    COUNT MISMATCH sec %d: oracle rungs=%d go=%d\n", sec, rungs, len(gn.HeightL))
			continue
		}
		var L, R []int16
		for k := 0; k < rungs; k++ {
			c.A[4], c.A[5] = a4, a5
			ol := int(int16(uint16(call(vbuild, map[int]uint32{1: uint32(2 * k)}))))
			c.A[4], c.A[5] = a4, a5
			or := int(int16(uint16(call(vbuild, map[int]uint32{1: uint32(2*k + 1)}))))
			if gn.HeightL[k] != ol || gn.HeightR[k] != or {
				mism++
				fmt.Printf("    MISMATCH sec %d rung %d: oracle L=%d R=%d  go L=%d R=%d\n", sec, k, ol, or, gn.HeightL[k], gn.HeightR[k])
			}
			L, R = append(L, int16(ol)), append(R, int16(or))
		}
		if os.Getenv("V") != "" {
			fmt.Printf("  %2d cnt=%2d  L:%v\n              R:%v\n", sec, cnt, L, R)
		}
	}
	if mism == 0 {
		fmt.Printf("track %d: OK — all %d sections, Go transcription == oracle (coordinate-exact)\n", id, n)
	} else {
		fmt.Printf("track %d: %d MISMATCHES\n", id, mism)
		os.Exit(1)
	}
}
