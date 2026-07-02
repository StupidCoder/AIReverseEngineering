// Package objplace reimplements the engine's object placement: where a placed object
// actually rests when it activates, as opposed to the raw (blockX*32, blockY*32) it is
// spawned at ($1AB3). Three pieces of the game do the moving, all reimplemented here
// from the disassembly (Sonic.md Part V):
//
//   - The per-type handler's first frame stores the hitbox: LD (IX+13),width /
//     LD (IX+14),height at the handler entry (dispatch table $24B2). The object's
//     world position (IX+2/3, IX+5/6) is the TOP-LEFT of that hitbox.
//
//   - The pickup family (bonus panels $01-$05, emerald $06, checkpoint $51, continue
//     $52 — the handlers that CALL $6089) applies a one-time spawn adjust: if the map
//     block at the object's own position is $B0 in Green Hills, (+22,+22); else X+=4.
//
//   - The shared move code $2CD4 (every object runs it after its handler, every frame)
//     probes the terrain under the hitbox's bottom edge at its horizontal centre
//     (X+width/2, Y+height) and snaps the object onto the floor line:
//     Y = blockRowBase + profile − height, setting the grounded bit (IX+24 bit 7).
//     The floor line comes from the level's collision data: per-zone block→shape at
//     ($343D+zone*2), the shape's 32-byte height profile via the pointer table $3E7A
//     ($80 = no surface in that column), and the landing bias $39DA[shape]:
//     land iff (bottom&31)+bias >= profile.
//
// Objects have no gravity in the shared code, so a non-Sonic object placed above the
// floor line stays where it spawned; Sonic's handler accelerates him downward until
// the same snap grounds him, so his rest position is the first floor line below his
// spawn (DropToFloor). All of this is verified against the live engine by cmd/objsettle.
package objplace

const (
	dispatch = 0x24B2 // per-type handler word table, types $00-$56
	attrPtrs = 0x343D // per-zone pointer to the block -> attribute (bit7 prio, bits0-5 shape) table
	floorPtr = 0x3E7A // shape*2 -> pointer to the 32-byte floor height profile
	biasTbl  = 0x39DA // per-shape landing bias: land iff (bottom&31)+bias >= profile
)

func word(rom []byte, o int) int { return int(rom[o]) | int(rom[o+1])<<8 }

// Hitbox returns the object type's collision box (width IX+13, height IX+14) by reading
// the LD (IX+13),n / LD (IX+14),n immediates at its handler's entry, and whether the
// handler stores one at all (invisible triggers like the scroll locks do not).
func Hitbox(rom []byte, typ int) (w, h int, ok bool) {
	a, end := HandlerRange(rom, typ)
	if a == 0 {
		return 0, 0, false
	}
	w, h = -1, -1
	for o := a; o+3 < end; o++ {
		if rom[o] == 0xDD && rom[o+1] == 0x36 {
			switch rom[o+2] {
			case 13:
				if w < 0 {
					w = int(rom[o+3])
				}
			case 14:
				if h < 0 {
					h = int(rom[o+3])
				}
			}
			if w >= 0 && h >= 0 {
				return w, h, true
			}
		}
	}
	return 0, 0, false
}

// HasSpawnAdjust reports whether the type's handler is in the pickup family that
// applies the one-time $6089 spawn adjust.
func HasSpawnAdjust(rom []byte, typ int) bool {
	a, end := HandlerRange(rom, typ)
	for o := a; a != 0 && o+2 < end; o++ {
		if rom[o] == 0xCD && rom[o+1] == 0x89 && rom[o+2] == 0x60 {
			return true
		}
	}
	return false
}

// HandlerRange returns the handler's z80 address (== file offset: the handlers live in
// the home banks 1/2) and the address of the next handler in the same bank, so a scan
// cannot run into a neighbour.
func HandlerRange(rom []byte, typ int) (start, end int) {
	start = word(rom, dispatch+typ*2)
	if start == 0 {
		return 0, 0
	}
	end = (start | 0x3FFF) + 1 // top of the handler's 16 KB slot
	for t := 0; t < 0x57; t++ {
		if a := word(rom, dispatch+t*2); a > start && a < end {
			end = a
		}
	}
	return start, end
}

// Level is the collision view of one act: its decoded block map plus the zone's
// block→shape table, everything the $2CD4 floor probe reads.
type Level struct {
	rom    []byte
	blocks []byte // decoded map ($0A73 output), row-major
	stride int    // columns per row ($D232)
	zone   int
}

func NewLevel(rom, blocks []byte, stride, zone int) *Level {
	return &Level{rom: rom, blocks: blocks, stride: stride, zone: zone}
}

func (l *Level) block(col, row int) int {
	if col < 0 || row < 0 || col >= l.stride || row*l.stride+col >= len(l.blocks) {
		return 0
	}
	return int(l.blocks[row*l.stride+col])
}

func (l *Level) shape(col, row int) int {
	return int(l.rom[word(l.rom, attrPtrs+l.zone*2)+l.block(col, row)]) & 0x3F
}

// floorAt returns the floor line's offset within the block at (col,row) for pixel
// column x (signed; slopes run past the block edges), or ok=false when the block has
// no surface in that column ($2D50/$2E4D: shape 0, or profile byte $80).
func (l *Level) floorAt(col, row, x int) (profile, bias int, ok bool) {
	s := l.shape(col, row)
	if s == 0 {
		return 0, 0, false
	}
	p := int(int8(l.rom[word(l.rom, floorPtr+s*2)+(x&0x1F)]))
	if p == -128 {
		return 0, 0, false
	}
	return p, int(l.rom[biasTbl+s]), true
}

// Settle reproduces the engine's placement of an object spawned at the top-left world
// position (x,y): the pickup family's $6089 adjust, then the $2CD4 bottom-edge floor
// snap. It returns the rest position and whether the object grounded. Objects have no
// gravity, so this is exactly their state on their first live frame.
func (l *Level) Settle(typ, x, y int) (rx, ry int, grounded bool) {
	w, h, ok := Hitbox(l.rom, typ)
	if !ok {
		return x, y, false
	}
	if HasSpawnAdjust(l.rom, typ) {
		if l.zone == 0 && l.block(x>>5, y>>5) == 0xB0 {
			x, y = x+22, y+22
		} else {
			x += 4
		}
	}
	bottom := y + h
	cx := x + w/2
	if p, bias, ok := l.floorAt(cx>>5, bottom>>5, cx); ok && (bottom&0x1F)+bias >= p {
		return x, (bottom &^ 0x1F) + p - h, true
	}
	return x, y, false
}

// DropToFloor is Settle for an object whose handler pulls it down (Sonic: gravity in
// $4AD0): starting from the spawn it lowers the hitbox until the $2CD4 snap grounds
// it — the first floor line at or below the initial feet — and returns the rest
// position. If no floor exists below (spawn over a pit), the spawn position itself
// comes back with grounded=false.
//
// Sonic's FIRST probe runs in his handler's initial short-box state ($55D8: 16x21;
// the paired ±11 Y adjusts at $55CE/$4CAA preserve his feet across every 21<->32 box
// switch), so his feet start at spawnY+21 — which catches a floor line inside the
// spawn block itself (Bridge 1, Labyrinth 1) that a 32-tall probe would jump past.
// He then stands up to the full 32-tall box with his feet where they landed:
// rest y = feet − 32. Oracle-verified per act by cmd/spawncheck.
const sonicSpawnFeet = 21

func (l *Level) DropToFloor(typ, x, y int) (rx, ry int, grounded bool) {
	w, h, ok := Hitbox(l.rom, typ)
	if !ok {
		return x, y, false
	}
	h0 := h
	if typ == 0 {
		h0 = sonicSpawnFeet
	}
	rows := len(l.blocks) / l.stride
	cx := x + w/2
	for bottom := y + h0; bottom < rows*32; bottom++ {
		if p, bias, ok := l.floorAt(cx>>5, bottom>>5, cx); ok && (bottom&0x1F)+bias >= p {
			return x, (bottom &^ 0x1F) + p - h, true
		}
	}
	return x, y, false
}
