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

	// --- stateful routines: randomise the inputs they read, compare the outputs ---
	matrixSlots := []uint32{}
	for o := uint32(0); o <= 0x46; o += 2 {
		matrixSlots = append(matrixSlots, physics.Mtx+o)
	}
	stateBlock := []uint32{
		physics.PosX, physics.PosY, physics.PosZ, physics.Roll, physics.Yaw, physics.Pit,
		physics.VelX, physics.VelY, physics.VelZ, physics.AmR, physics.AmP, physics.AmY,
	}
	type tc struct {
		name        string
		pc          uint32
		fn          func(*physics.Mem)
		seed, check []uint32
	}
	intSeed := []uint32{physics.VelX, physics.VelY, physics.VelZ, physics.AmR, physics.AmP, physics.AmY,
		physics.FrcX, physics.FrcY, physics.FrcZ, physics.TqR, physics.TqP, physics.TqY,
		physics.WAmR, physics.WAmY, physics.WAmP, physics.Roll, physics.Yaw, physics.Pit}
	cases := []tc{
		{"Force61ADC $61ADC", 0x61ADC, (*physics.Mem).Force61ADC, intSeed, stateBlock},
		{"Torque61B26 $61B26", 0x61B26, (*physics.Mem).Torque61B26, intSeed, stateBlock},
		{"Integrate61950 $61950", 0x61950, (*physics.Mem).Integrate61950, intSeed, stateBlock},
		{"Matrix61368 $61368", 0x61368, (*physics.Mem).Matrix61368,
			[]uint32{physics.Roll, physics.Yaw, physics.Pit, physics.Hdg}, matrixSlots},
		{"VelToBody $6158C", 0x6158C, (*physics.Mem).VelToBody6158C,
			append(append([]uint32{}, matrixSlots...), physics.VelX, physics.VelY, physics.VelZ),
			[]uint32{physics.BVelL, physics.BVelV}},
		{"GravToBody $615E6", 0x615E6, (*physics.Mem).GravToBody615E6,
			matrixSlots, []uint32{physics.GrvA, physics.GrvB, physics.GrvC}},
		{"ForceToWorld $61618", 0x61618, (*physics.Mem).ForceToWorld61618,
			append(append([]uint32{}, matrixSlots...), physics.BFrcA, physics.BFrcB, physics.BFrcC),
			[]uint32{physics.FrcX, physics.FrcY, physics.FrcZ}},
		{"TorqueToWorld $61672", 0x61672, (*physics.Mem).TorqueToWorld61672,
			append(append([]uint32{}, matrixSlots...), physics.AmR, physics.AmP, physics.AmY, physics.WAmY),
			[]uint32{physics.WAmR, physics.WAmY, physics.WAmP}},
		{"Interp $5C554", 0x5C554, (*physics.Mem).Interp5C554,
			[]uint32{0x1BC02, 0x1BC04, 0x1BC06, 0x1BC08, 0x1BC40, 0x1BC4C},
			[]uint32{0x1BB18, 0x1BB1A}},
		{"Corners $618CE", 0x618CE, (*physics.Mem).Corners618CE,
			[]uint32{0x1C264, 0x1C268, 0x1C26E, 0x1C272, 0x1C274, 0x1C276},
			[]uint32{0x1BD02, 0x1BD04, 0x1BD06, 0x1BD08, 0x1BD0A, 0x1BD0C}},
		{"ContactHeights $61B70", 0x61B70, (*physics.Mem).ContactHeights61B70,
			[]uint32{physics.Roll, physics.Pit},
			[]uint32{0x1BC94, 0x1BC96, 0x1BC98, 0x1BC9A, 0x1BC9C, 0x1BC9E, 0x1BBF6}},
		{"Suspension $61BCC", 0x61BCC, (*physics.Mem).Suspension61BCC,
			[]uint32{physics.Spr0Prev, physics.Spr1Prev, physics.Spr2Prev,
				physics.Spr0Force, physics.Spr1Force, physics.Spr2Force,
				0x1BB01, 0x1BBCD, 0x1BB56, 0x1BC3A, physics.Spr0Dmg, physics.Spr1Dmg, physics.Spr2Dmg,
				0x1BBDF, physics.Roll, 0x1BCF0, 0x1CA33},
			[]uint32{physics.Spr0Force, physics.Spr1Force, physics.Spr2Force,
				physics.Spr0Travel, physics.Spr1Travel, physics.Spr2Travel,
				physics.Spr0Prev, physics.Spr1Prev, physics.Spr2Prev,
				0x1BCB0, 0x1BCB2, 0x1BCB4, 0x1BCB6, 0x1BCB8, 0x1BCBA,
				physics.Spr0Dmg, physics.Spr2Dmg, 0x1BB56,
				physics.NetLift, 0x1BBF6, physics.RollTq, physics.OnGround, physics.Bottom}},
		// NB $1BD26 (pitch torque) is finalised by $5B32E (steering) in $61BCC's tail; it
		// is verified with the control routines, not here.
		{"LateralTire $6217A", 0x6217A, (*physics.Mem).LateralTire6217A,
			[]uint32{physics.GrvA, physics.LoadA, physics.BVelL, physics.LoadB, physics.OnGround},
			[]uint32{physics.BFrcA, physics.Slip}},
		{"TorqueApply $62138", 0x62138, (*physics.Mem).TorqueApply62138,
			[]uint32{physics.PitchTq, physics.RollTq, physics.AmR, physics.AmY, physics.BFrcC, physics.OnGround},
			[]uint32{physics.TqAppR, physics.TqAppY}},
		{"Drive $620B8", 0x620B8, (*physics.Mem).Drive620B8,
			[]uint32{physics.GrvB, physics.GrvC, physics.GrvA, physics.LoadA, physics.LoadB, physics.LoadC,
				physics.Drive, physics.BVelV, 0x1BD2B, physics.BVelL, physics.OnGround},
			[]uint32{physics.BFrcB, physics.BFrcC, physics.Drive, physics.BFrcA, physics.Slip}},
	}
	for _, t := range cases {
		bad := 0
		for iter := 0; iter < 3000; iter++ {
			m := baseMem()
			for _, a := range t.seed {
				wW(m, a, int16(rng.Intn(0x10000)))
			}
			// give the integrator some 32-bit positions and the $619E4 flag bytes too.
			if t.pc == 0x61950 {
				wL(m, physics.PosX, rng.Int31()-(1<<30))
				wL(m, physics.PosY, rng.Int31()-(1<<30))
				wL(m, physics.PosZ, rng.Int31()-(1<<30))
				m[0x1BB75] = byte(rng.Intn(256))
				m[0x1BB9A] = byte(rng.Intn(256))
			}
			if t.pc == 0x61B70 {
				wL(m, physics.PosY, rng.Int31()-(1<<30))
			}
			if t.pc == 0x61BCC {
				// 32-bit surface / chassis / rest heights, in a range that exercises the
				// clamp boundaries and the damage thresholds.
				r32 := func() int32 { return int32(rng.Intn(0x8000) - 0x4000) }
				for _, a := range []uint32{physics.Spr0Surf, physics.Spr1Surf, physics.Spr2Surf,
					physics.Spr0Car, physics.Spr1Car, physics.Spr2Car} {
					wL(m, a, r32())
				}
				wL(m, physics.Rest, int32(rng.Intn(0x2000)))
			}
			eng, _ := runEngine(m, t.pc, nil)
			gmem := physics.New(img)
			copy(gmem.B, m)
			t.fn(gmem)
			for _, a := range t.check {
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
	// --- $5FE56 per-section setup: needs a loaded track ---
	setBytes := []uint32{0x1BB79, 0x1BBDC, 0x1BC4A, 0x1BC32, 0x1BB86, 0x1BB4D, 0x1BB97,
		0x1BB59, 0x1BB98, 0x1BB5A, 0x1BB6A, 0x1BC44, 0x1BB7B, 0x1BBD9, 0x1BBD4,
		0x1BC8C, 0x1BC8D, 0x1BC90, 0x1BC91, 0x1BC0E, 0x1BC0F, 0x1BC10, 0x1BC11, 0x1BCBC, 0x1BCBD}
	for _, id := range []int{1, 3, 7} {
		m := baseMem()
		m[0x1CA33] = byte(id)
		loaded, _ := runEngine(m, 0x5AE46, map[int]uint32{1: uint32(id)})
		n := int(loaded[0x1CA1A])
		bad := 0
		for sec := 0; sec < n; sec++ {
			eng, _ := runEngine(loaded, 0x5FE56, map[int]uint32{1: uint32(sec)})
			gm := physics.New(img)
			copy(gm.B, loaded)
			gm.Setup5FE56(sec)
			for _, a := range setBytes {
				if gm.B[a] != eng[a] {
					bad++
					if bad <= 3 {
						fmt.Printf("  Setup5FE56 t%d sec%d @%X: go=%02x eng=%02x | 1BCBC go=%02x%02x eng=%02x%02x  type=%02x\n",
							id, sec, a, gm.B[a], eng[a], gm.B[0x1BCBC], gm.B[0x1BCBD], eng[0x1BCBC], eng[0x1BCBD], eng[0x1C5EC+uint32(sec)])
					}
					break
				}
			}
		}
		report(fmt.Sprintf("Setup5FE56 track %d", id), bad)
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
