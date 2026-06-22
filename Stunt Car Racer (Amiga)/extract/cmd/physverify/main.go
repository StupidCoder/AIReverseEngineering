// physverify checks the Go physics reimplementation (package physics) against the engine
// running on the tools/m68k core, routine by routine. For each routine it builds a
// synthetic car-state snapshot, runs the engine routine and the Go routine from the same
// bytes, and compares the resulting state. Per the project rule the oracle only verifies.
//
// Usage: physverify game.dec.bin
package main

import (
	"fmt"
	"math/rand"
	"os"

	"stuntcar/extract/physics"
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

var img []byte

// runEngine executes one engine routine at pc over a copy of mem, returning the mem after.
func runEngine(mem []byte, pc uint32, dIn map[int]uint32) ([]byte, map[int]uint32) {
	bus := &flatBus{m: make([]byte, 1<<24)}
	copy(bus.m, mem)
	c := m68k.NewCPU(bus)
	c.A[7] = stackTop - 4
	r := uint32(sentinel)
	bus.Write(c.A[7], byte(r>>24))
	bus.Write(c.A[7]+1, byte(r>>16))
	bus.Write(c.A[7]+2, byte(r>>8))
	bus.Write(c.A[7]+3, byte(r))
	for reg, v := range dIn {
		c.D[reg] = v
	}
	c.PC = pc
	for steps := 0; c.PC != sentinel; steps++ {
		if c.Halted || steps > 2_000_000 {
			fmt.Printf("engine halt/cap at $%X\n", c.PC)
			os.Exit(1)
		}
		c.Step()
	}
	out := map[int]uint32{}
	for i := 0; i < 8; i++ {
		out[i] = c.D[i]
	}
	return bus.m, out
}

// baseMem returns the loaded image as a 24-bit space (the static sin table etc. present).
func baseMem() []byte {
	mem := make([]byte, 1<<24)
	copy(mem[base:], img)
	return mem
}

func wW(mem []byte, a uint32, v int16) {
	mem[a] = byte(uint16(v) >> 8)
	mem[a+1] = byte(v)
}
func wL(mem []byte, a uint32, v int32) { wW(mem, a, int16(v>>16)); wW(mem, a+2, int16(v)) }
func rW(mem []byte, a uint32) int16    { return int16(uint16(mem[a])<<8 | uint16(mem[a+1])) }

var fails int

func checkW(name string, addr uint32, got, want []byte) {
	if rW(got, addr) != rW(want, addr) {
		fails++
		fmt.Printf("  MISMATCH %s @%X: go=%d engine=%d\n", name, addr, rW(want, addr), rW(got, addr))
	}
}

func main() {
	var err error
	img, err = os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	rng := rand.New(rand.NewSource(1))

	// --- sin/cos ($64D10/$64D08) ---
	mem := baseMem()
	gm := physics.New(img)
	sinBad, cosBad := 0, 0
	for i := 0; i < 2000; i++ {
		a := int16(rng.Intn(0x10000))
		_, es := runEngine(mem, 0x64D08, map[int]uint32{0: uint32(uint16(a))})
		if int16(uint16(es[0])) != gm.Sin(a) {
			sinBad++
		}
		_, ec := runEngine(mem, 0x64D10, map[int]uint32{0: uint32(uint16(a))})
		if int16(uint16(ec[0])) != gm.Cos(a) {
			cosBad++
		}
	}
	report("Sin $64D08", sinBad)
	report("Cos $64D10", cosBad)

	// --- integrator routines: synthetic state, compare full state block ---
	addrs := []uint32{
		physics.PosX, physics.PosY, physics.PosZ, physics.Roll, physics.Yaw, physics.Pit,
		physics.VelX, physics.VelY, physics.VelZ, physics.AmR, physics.AmP, physics.AmY,
	}
	type tc struct {
		name string
		pc   uint32
		fn   func(*physics.Mem)
	}
	cases := []tc{
		{"Force61ADC $61ADC", 0x61ADC, (*physics.Mem).Force61ADC},
		{"Torque61B26 $61B26", 0x61B26, (*physics.Mem).Torque61B26},
		{"Integrate61950 $61950", 0x61950, (*physics.Mem).Integrate61950},
	}
	for _, t := range cases {
		bad := 0
		for iter := 0; iter < 3000; iter++ {
			m := baseMem()
			// randomise the state the routine reads.
			for _, a := range []uint32{physics.VelX, physics.VelY, physics.VelZ, physics.AmR, physics.AmP, physics.AmY,
				physics.FrcX, physics.FrcY, physics.FrcZ, physics.TqR, physics.TqP, physics.TqY,
				physics.WAmR, physics.WAmY, physics.WAmP, physics.Roll, physics.Yaw, physics.Pit} {
				wW(m, a, int16(rng.Intn(0x10000)))
			}
			wL(m, physics.PosX, rng.Int31()-(1<<30))
			wL(m, physics.PosY, rng.Int31()-(1<<30))
			wL(m, physics.PosZ, rng.Int31()-(1<<30))
			m[0x1BB75] = byte(rng.Intn(256))
			m[0x1BB9A] = byte(rng.Intn(256))

			eng, _ := runEngine(m, t.pc, nil)
			gmem := physics.New(img)
			copy(gmem.B, m)
			t.fn(gmem)
			for _, a := range addrs {
				if rW(gmem.B, a) != rW(eng, a) {
					bad++
					if bad <= 3 {
						fmt.Printf("  %s @%X: go=%d eng=%d\n", t.name, a, rW(gmem.B, a), rW(eng, a))
					}
					break
				}
			}
		}
		report(t.name, bad)
	}
	if fails == 0 {
		fmt.Println("ALL OK")
	} else {
		os.Exit(1)
	}
}

func report(name string, bad int) {
	if bad == 0 {
		fmt.Printf("%-26s OK\n", name)
	} else {
		fails++
		fmt.Printf("%-26s %d FAIL\n", name, bad)
	}
}
