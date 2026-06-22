// physoracle runs the car physics master $6185C on the tools/m68k core, frame by frame,
// over a properly initialised race state, and dumps the car-state block. It is the
// ground-truth verifier for the Go reimplementation of the physics (Part V) — per the
// project rule the oracle only verifies; it is never the source of shipped data.
//
// Setup: run the real race-init chain ($5AE46 track load, $64304 grid, $5A794 start
// section, $696FC config, $604B4 car placement), set the input ($1BB47) and gear
// ($1BB57), then call $6185C N times.
//
// Usage: physoracle game.dec.bin [trackid] [frames] [inputhex]
package main

import (
	"fmt"
	"os"
	"strconv"

	"stupidcoder.com/tools/m68k"
)

const (
	base     = 0xE700
	sentinel = 0xFFFFFE
	stackTop = 0x300000
)

type flatBus struct{ m []byte }

func (b *flatBus) Read(a uint32) byte     { return b.m[a&0xFFFFFF] }
func (b *flatBus) Write(a uint32, v byte) { b.m[a&0xFFFFFF] = v }

var bus *flatBus
var c *m68k.CPU

func call(pc uint32, regs map[int]uint32) {
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
		if c.Halted {
			fmt.Printf("HALT at $%X: %s\n", c.PC, c.HaltReason)
			os.Exit(1)
		}
		if steps > 50_000_000 {
			fmt.Printf("step cap at $%X\n", c.PC)
			os.Exit(1)
		}
		c.Step()
	}
}

func rw(a uint32) int16  { return int16(uint16(bus.Read(a))<<8 | uint16(bus.Read(a+1))) }
func rl(a uint32) int32  { return int32(uint32(rw(a))<<16 | uint32(uint16(rw(a+2)))) }
func wb(a uint32, v byte) { bus.Write(a, v) }

// state dumps the car-state block.
func dumpState(tag string) {
	fmt.Printf("[%s] pos(%d,%d,%d) ang roll=%d yaw=%d pit=%d  vel(%d,%d,%d) angmom(r=%d p=%d y=%d) onGround=%02x dmg=%d/%d\n",
		tag,
		rl(0x1BCD8), rl(0x1BCDC), rl(0x1BCE0),
		rw(0x1BCE4), rw(0x1BCE6), rw(0x1BCE8),
		rw(0x1BCEA), rw(0x1BCEC), rw(0x1BCEE),
		rw(0x1BCF0), rw(0x1BCF2), rw(0x1BCF4),
		bus.Read(0x1BB7E), bus.Read(0x1BB4F), bus.Read(0x1BB51))
}

func main() {
	img, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "physoracle:", err)
		os.Exit(1)
	}
	id := 1
	frames := 8
	input := 0
	if len(os.Args) >= 3 {
		id, _ = strconv.Atoi(os.Args[2])
	}
	if len(os.Args) >= 4 {
		frames, _ = strconv.Atoi(os.Args[3])
	}
	if len(os.Args) >= 5 {
		v, _ := strconv.ParseInt(os.Args[4], 16, 32)
		input = int(v)
	}

	bus = &flatBus{m: make([]byte, 1<<24)}
	copy(bus.m[base:], img)
	c = m68k.NewCPU(bus)

	// race-init chain (mirrors $5D2CA): track -> grid -> start section -> config -> render -> car.
	step := func(name string, pc uint32, regs map[int]uint32) {
		call(pc, regs)
		_ = name
	}
	wb(0x1CA33, byte(id))
	step("loadtrack", 0x5AE46, map[int]uint32{1: uint32(id)})
	step("grid", 0x64304, nil)
	step("startsec", 0x5A794, nil)
	step("config", 0x696FC, nil)

	// gear 1, chosen input held each frame.
	wb(0x1BB57, 1)

	// Place the car manually (replicates $604B4 without its render sub-call $64E18,
	// which needs graphics state the bare physics doesn't): per-gear start tables.
	g := uint32(1 & 3)
	wb(0x1BCD8, bus.Read(0x60552+g)) // posX (high byte of 32-bit)
	wb(0x1BCE0, bus.Read(0x60556+g)) // posZ
	wb(0x1BCE6, bus.Read(0x6055A+g)) // yaw
	wb(0x1BCDC, 0x03)                // posY high word = $03F0
	wb(0x1BCDD, 0xF0)
	wb(0x1BCC8, 0x00)
	bus.Write(0x1BC42, 0x07)
	bus.Write(0x1BC43, 0x00) // $1BC42 = $0700
	wb(0x1BB68, 0x80)
	dumpState("init")
	for f := 0; f < frames; f++ {
		wb(0x1BB47, byte(input))
		call(0x6185C, nil)
		dumpState(fmt.Sprintf("f%d", f))
	}
}
