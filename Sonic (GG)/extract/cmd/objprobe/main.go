// objprobe decodes a level's object placement, both statically from ROM and (to confirm
// it) from the running machine.
//
// The level loader ($1A80/$1AB3) builds an object array at RAM $D3FD: 32 records of 26
// bytes, type at +0, world position X=u16(+2), Y=u16(+5) (= placement block coordinate
// x32). Record 0 is Sonic, spawned from a pointer at ($D217). The placements come from a
// per-act table in bank 5 at $15600 + word(descriptor+30): a count byte followed by
// 3-byte [type, blockX, blockY] entries. objprobe reads that table straight from ROM,
// then boots the act and checks Sonic's spawn + the live object array against it.
//
// Usage: objprobe <rom.gg> [act]
package main

import (
	"fmt"
	"os"
	"strconv"

	"stupidcoder.com/tools/gamegear"
)

const (
	descTable = 0x15600
	objBase   = 0xD3FD
	objSize   = 0x1A
	objMax    = 32
)

type obj struct {
	typ    byte
	bx, by int
}

// objectTable reads the per-act object placement table straight from ROM.
func objectTable(rom []byte, act int) []obj {
	w := func(o int) int { return int(rom[o]) | int(rom[o+1])<<8 }
	d := descTable + w(descTable+act*2)
	t := descTable + w(d+30) // object table = $15600 + descriptor word +30
	count := int(rom[t])
	var objs []obj
	for i := 0; i < count; i++ {
		p := t + 1 + i*3
		objs = append(objs, obj{rom[p], int(rom[p+1]), int(rom[p+2])})
	}
	return objs
}

func main() {
	rom, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	act := 0
	if len(os.Args) > 2 {
		act, _ = strconv.Atoi(os.Args[2])
	}

	objs := objectTable(rom, act)
	fmt.Printf("act %d: object placement table (ROM, %d objects):\n", act, len(objs))
	fmt.Printf("  #   type   blockX  blockY    worldX  worldY\n")
	for i, o := range objs {
		fmt.Printf("  %2d  $%02X    %5d   %5d    %6d  %6d\n", i, o.typ, o.bx, o.by, o.bx*32, o.by*32)
	}

	// Boot the act and read Sonic's spawn + the live array to confirm the decode.
	m := gamegear.NewMachine(rom)
	u16 := func(a uint16) int { return int(m.Read(a)) | int(m.Read(a+1))<<8 }
	m.CapturePC = 0x0A73
	for i := 0; i < 700; i++ {
		m.RunFrame()
	}
	for round := 0; round < 40 && !m.Captured; round++ {
		m.Pad00 = 0x7F
		m.Write(0xD238, byte(act))
		for i := 0; i < 8; i++ {
			m.RunFrame()
			m.Write(0xD238, byte(act))
		}
		m.Pad00 = 0xFF
		for k := 0; k < 242 && !m.Captured; k++ {
			m.Write(0xD238, byte(act))
			m.RunFrame()
		}
	}
	// Capture the spawn the INSTANT the object array is populated (slot 0 = Sonic's
	// placed position), before any physics -- in some levels (Labyrinth) Sonic falls
	// immediately, and the model has no collision, so a later read is garbage.
	spawnX, spawnY := 0, 0
	for i := 0; i < 400 && spawnX == 0 && spawnY == 0; i++ {
		m.RunFrame()
		spawnX, spawnY = u16(objBase+2), u16(objBase+5)
	}
	fmt.Printf("\nSonic slot-0 spawn (pristine): block (%d,%d) = world (%d,%d)\n",
		spawnX/32, spawnY/32, spawnX, spawnY)
	match := 0
	for i := 0; i < objMax; i++ {
		b := uint16(objBase + i*objSize)
		if t := m.Read(b); t != 0 && t != 0xFF {
			lx, ly := u16(b+2)/32, u16(b+5)/32
			for _, o := range objs {
				if o.typ == t && o.bx == lx && o.by == ly {
					match++
					break
				}
			}
		}
	}
	fmt.Printf("  %d/%d ROM-table objects found unmodified in the live array\n", match, len(objs))
}
