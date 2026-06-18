// Package scene extracts Turrican's per-scene object placements off the disk,
// faithfully reproducing the scroll-triggered spawner (resident $1710).
//
// Each world is a decoded scene block loaded at $1B980; each scene's descriptor
// (table at block+$16) carries the placement data as a 2D bucket grid:
//
//	desc+$24  rowTable   — per camera-row word offsets into the grid
//	desc+$28  grid       — for each row, an array of 4-byte pointers (one per
//	                       camera column) into the sorted entry stream
//
// Many grid cells share a pointer; each distinct pointer heads a per-column run of
// 6-byte entries — `type.w, x.w, y.w` (x/y in 8-pixel units) — terminated by a
// `$00D3` word (a `$0000` word is a skipped hole). The entry type is the **high**
// byte of the type word; it selects the AI handler that drives the object and
// installs its sprite (frame table, via `MOVE.l #ft,$12(a5)`):
//
//	type < 3   resident handler table $1A60, index type-1   (engine-wide objects)
//	type >= 3  the scene descriptor's +$20 table, index type-3
//
// The spawner writes the object's world position as entry.x*8-32 / entry.y*8 (the
// camera terms cancel); that pixel position is the BOB's top-left.
package scene

import (
	"encoding/binary"
	"fmt"

	"turrican/extract/decrunch"
)

const (
	BlockBase  = 0x1B980 // a scene block's runtime load address
	levelTable = 0x46A
	NumWorlds  = 5
	residTable = 0x1A60 // resident AI handler table (entry types 1,2)
	residLo    = 0x10   // resident engine code range (for handler/frame-table addrs)
	residHi    = 0x1B780
	ResidGfxLo = 0x10000 // resident sprite bitmaps live here
	endMarker  = 0x00D3  // ends a column's entry run
)

// Space is a byte slice addressed by absolute runtime address: addr a is at
// Data[a-Base].
type Space struct {
	Data []byte
	Base int
}

func (s Space) be32(a int) int { return int(binary.BigEndian.Uint32(s.Data[a-s.Base:])) }
func (s Space) be16(a int) int { return int(binary.BigEndian.Uint16(s.Data[a-s.Base:])) }
func (s Space) Has(a, n int) bool {
	o := a - s.Base
	return o >= 0 && o+n <= len(s.Data)
}

// Object is one placed object, resolved to its sprite.
type Object struct {
	Type     int  // entry type (high byte of the type word)
	X, Y     int  // world pixel position of the BOB top-left
	Handler  int  // AI handler address (0 if unresolved)
	FT       int  // frame table the handler installs (the sprite; 0 if none)
	Resident bool // sprite lives in resident space (vs the scene block)
}

// Scene is one scene's geometry, spawn and objects.
type Scene struct {
	World, Index  int
	Width, Height int // in 32-px tiles
	// CamX/CamY: the initial camera viewport top-left in world pixels (descriptor
	// +$08/+$0A tiles, ×32). SpawnX/SpawnY: the player's spawn in world pixels —
	// the camera top-left plus the on-screen offset the descriptor stores at +$0C/+$0E
	// (which select_scene installs to $45A4 -> the player position $104/$106).
	CamX, CamY     int
	SpawnX, SpawnY int
	Objects        []Object
}

// Game holds the decoded resident image and per-world scene blocks.
type Game struct {
	Resident Space // res.Data, base 0
	blocks   [][]byte
}

// Load decrunches the main image and every world's scene block.
func Load(adf []byte) (*Game, error) {
	res, err := decrunch.Decrunch(mainBlob(adf))
	if err != nil {
		return nil, err
	}
	g := &Game{Resident: Space{Data: res.Data, Base: 0}, blocks: make([][]byte, NumWorlds)}
	for w := 0; w < NumWorlds; w++ {
		t := levelTable + w*8
		off := int(binary.BigEndian.Uint32(res.Data[t:]))
		length := int(binary.BigEndian.Uint32(res.Data[t+4:]))
		block, err := decrunch.DecrunchBlock(adf[off : off+length])
		if err != nil {
			return nil, fmt.Errorf("world %d: %w", w, err)
		}
		g.blocks[w] = block
	}
	return g, nil
}

// Block returns world w's scene block as a Space.
func (g *Game) Block(w int) Space { return Space{Data: g.blocks[w], Base: BlockBase} }

// TileCollision returns world w's per-tile collision table (block header +$04, the
// $3C1C4 section) and the tile count. Each tile has 16 bytes = a 4x4 grid of
// 8x8-block solidity (0 = passable, nonzero = solid), which select_scene's builder
// ($2994) copies into the live collision buffer the player check ($35AE) reads.
func (g *Game) TileCollision(w int) (data []byte, nTiles int) {
	blk := g.Block(w)
	nTiles = blk.be32(blk.be32(BlockBase)) / 4 // header+$00 -> tile table; [0] = table size
	coll := blk.be32(BlockBase + 0x04)
	o := coll - blk.Base
	n := nTiles * 16
	if o < 0 || o+n > len(blk.Data) {
		return nil, nTiles
	}
	return blk.Data[o : o+n], nTiles
}

// Scenes resolves every scene of world w.
func (g *Game) Scenes(w int) []Scene {
	blk := g.Block(w)
	hi := BlockBase + len(blk.Data)
	var out []Scene
	nScenes := blk.be16(BlockBase + 0x14)
	for s := 0; s < nScenes; s++ {
		if !blk.Has(BlockBase+0x16+s*4, 4) {
			break
		}
		desc := blk.be32(BlockBase + 0x16 + s*4)
		if desc < BlockBase || desc >= hi {
			continue
		}
		sc := Scene{
			World: w, Index: s,
			Width: blk.be16(desc + 0x04), Height: blk.be16(desc + 0x06),
		}
		sc.CamX, sc.CamY = blk.be16(desc+0x08)*32, blk.be16(desc+0x0A)*32
		sc.SpawnX, sc.SpawnY = sc.CamX+blk.be16(desc+0x0C), sc.CamY+blk.be16(desc+0x0E)
		sc.Objects = g.objects(w, blk, hi, desc, sc.Width, sc.Height)
		out = append(out, sc)
	}
	return out
}

// objects walks the scene's grid and resolves each placement entry.
func (g *Game) objects(w int, blk Space, hi, desc, width, height int) []Object {
	rowT := blk.be32(desc + 0x24)
	grid := blk.be32(desc + 0x28)
	aiTbl := blk.be32(desc + 0x20)
	if !blk.Has(rowT, 2) || !blk.Has(grid, 4) {
		return nil
	}

	// Collect the distinct column-pointer heads across the whole grid. rowTable
	// offsets increase monotonically; stop when they wrap (end of table).
	heads := map[int]bool{}
	prev := -1
	for r := 0; r < height+4; r++ {
		if !blk.Has(rowT+r*2, 2) {
			break
		}
		roff := blk.be16(rowT + r*2)
		if r > 0 && roff <= prev {
			break
		}
		prev = roff
		for c := 0; c < 16; c++ {
			if !blk.Has(grid+roff+c*4, 4) {
				break
			}
			p := blk.be32(grid + roff + c*4)
			if p >= BlockBase && p < hi {
				heads[p] = true
			}
		}
	}

	// Walk each column run to its $00D3 terminator, deduping entries by address.
	type ent struct{ ty, x, y int }
	seen := map[int]ent{}
	for p := range heads {
		for a := p; blk.Has(a, 6); a += 6 {
			w := blk.be16(a)
			if w == endMarker {
				break
			}
			if w == 0 {
				continue // skipped hole
			}
			seen[a] = ent{ty: int(blk.Data[a-blk.Base]), x: blk.be16(a + 2), y: blk.be16(a + 4)}
		}
	}

	var objs []Object
	for _, e := range seen {
		px, py := e.x*8-32, e.y*8
		if px < 0 || py < 0 || px >= width*32 || py >= height*32 {
			continue
		}
		o := Object{Type: e.ty, X: px, Y: py}
		o.Handler, o.Resident = g.handler(blk, hi, aiTbl, e.ty)
		if o.Handler != 0 {
			o.FT = g.frameTable(blk, hi, o.Handler, o.Resident)
		}
		objs = append(objs, o)
	}
	return objs
}

// handler resolves an entry type to its AI handler (and which space it lives in).
func (g *Game) handler(blk Space, hi, aiTbl, ty int) (addr int, resident bool) {
	if ty < 3 { // resident handler table $1A60, index ty-1
		i := ty - 1
		if i < 0 || !g.Resident.Has(residTable+i*4, 4) {
			return 0, true
		}
		h := g.Resident.be32(residTable + i*4)
		if h < residLo || h >= residHi {
			return 0, true
		}
		return h, true
	}
	i := ty - 3 // scene descriptor +$20 table, index ty-3
	if !blk.Has(aiTbl+i*4, 4) {
		return 0, false
	}
	h := blk.be32(aiTbl + i*4)
	if h < BlockBase || h >= hi {
		return 0, false
	}
	return h, false
}

// frameTable scans an AI handler for MOVE.l #ft,$12(a5) (2B 7C imm32 00 12) — the
// frame table (sprite) it installs — in the handler's own space.
func (g *Game) frameTable(blk Space, hi, handler int, resident bool) int {
	sp, ftLo, ftHi := blk, BlockBase, hi
	if resident {
		sp, ftLo, ftHi = g.Resident, residLo, residHi
	}
	o := handler - sp.Base
	for i := o; i < o+400 && i+8 <= len(sp.Data); i++ {
		if sp.Data[i] == 0x2B && sp.Data[i+1] == 0x7C && sp.Data[i+6] == 0x00 && sp.Data[i+7] == 0x12 {
			ft := int(binary.BigEndian.Uint32(sp.Data[i+2:]))
			if ft >= ftLo && ft < ftHi {
				return ft
			}
		}
	}
	return 0
}

func mainBlob(adf []byte) []byte {
	const off = 0x2C00
	return adf[off : off+int(binary.BigEndian.Uint32(adf[off:]))]
}
