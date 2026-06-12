// Package shipmodel reconstructs Elite's engine memory and decodes the
// wireframe ship blueprints documented in Elite.md Part IV §1.
//
// The blueprints live in the engine block the loader relocated under the I/O
// area at $D000-$EFFF. The emulator that produces memory_final.bin cannot
// store writes to $D000-$DFFF (that range reads as I/O), so this package
// rebuilds those 4 KB from the decrypted SYS segment, exactly as the loader's
// relocation does ($5600-$65FF in the decrypted seg-3 blob -> $D000-$DFFF).
package shipmodel

import (
	"fmt"
	"os"
	"path/filepath"
)

// blueprint table and geometry constants (Elite.md Part IV §1).
const (
	tableBase  = 0xCFFE // XX21: word per ship type*2
	numTypes   = 33
	headerSize = 20
)

// Vertex is a 3-D model vertex with its level-of-detail visibility distance
// and the (up to four) faces it belongs to.
type Vertex struct {
	X, Y, Z int
	Vis     int
	Faces   [4]int
}

// Edge joins two vertices and names the two faces on either side of it.
type Edge struct {
	V1, V2       int
	FaceA, FaceB int
	Vis          int
}

// Face carries the outward surface normal used for back-face culling.
type Face struct {
	NX, NY, NZ int
	Vis        int
}

// Ship is one decoded blueprint.
type Ship struct {
	Type     int
	Addr     uint16
	Vertices []Vertex
	Edges    []Edge
	Faces    []Face
}

// LoadEngine returns the reconstructed 64 KB engine image, with the ship-data
// page range $D000-$DFFF filled in from the decrypted SYS segment.
func LoadEngine(extractedDir string) ([]byte, error) {
	mem, err := os.ReadFile(filepath.Join(extractedDir, "memory_final.bin"))
	if err != nil {
		return nil, err
	}
	if len(mem) != 0x10000 {
		return nil, fmt.Errorf("memory_final.bin is %d bytes, want 65536", len(mem))
	}
	seg, err := loadSeg3(extractedDir)
	if err != nil {
		return nil, err
	}
	copy(mem[0xD000:0xE000], seg[0x5600:0x6600])
	return mem, nil
}

// loadSeg3 loads the seg-3 SYS blob ($4000-$86CC) and decrypts it with the two
// rolling-subtraction passes the SYS handler runs (Elite.md Part III §1).
func loadSeg3(extractedDir string) ([]byte, error) {
	matches, err := filepath.Glob(filepath.Join(extractedDir, "*_seg03_4000.prg"))
	if err != nil {
		return nil, err
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("want exactly one *_seg03_4000.prg, found %d", len(matches))
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		return nil, err
	}
	seg := make([]byte, 0x10000)
	copy(seg[0x4000:], raw[2:]) // strip 2-byte load address
	decrypt(seg, 0x86, 0xCB, 0x8E, 0x54, 0x76)
	decrypt(seg, 0x76, 0x00, 0x6C, 0xFF, 0x3F)
	return seg, nil
}

// decrypt runs one descending rolling-subtraction pass: each plaintext byte is
// (cipher - key) and becomes the key for the next, walking the pointer down
// until it reaches the end page/offset.
func decrypt(seg []byte, hi, ystart, key, endY, endHi int) {
	ptr := hi << 8
	y := ystart
	k := key
	for {
		a := (int(seg[ptr+y]) - k) & 0xFF
		seg[ptr+y] = byte(a)
		k = a
		if y == 0 {
			ptr = ((ptr >> 8) - 1) << 8
		}
		y = (y - 1) & 0xFF
		if y == endY && (ptr>>8) == endHi {
			break
		}
	}
}

// BlueprintAddr returns the blueprint address for a ship type from the XX21
// table.
func BlueprintAddr(mem []byte, typ int) uint16 {
	a := tableBase + typ*2
	return uint16(mem[a]) | uint16(mem[a+1])<<8
}

// Parse decodes the blueprint for one ship type, or returns an error if the
// header does not describe a valid model (some table slots are not ships).
func Parse(mem []byte, typ int) (*Ship, error) {
	addr := BlueprintAddr(mem, typ)
	if addr < 0xD000 || int(addr)+headerSize >= 0x10000 {
		return nil, fmt.Errorf("type %d: blueprint address $%04X out of range", typ, addr)
	}
	b := mem[addr:]
	edgesOff := int(b[3])
	ne := int(b[9])
	if edgesOff <= headerSize || (edgesOff-headerSize)%6 != 0 {
		return nil, fmt.Errorf("type %d: bad edges offset %d", typ, edgesOff)
	}
	nv := (edgesOff - headerSize) / 6
	facesOff := edgesOff + ne*4
	if nv < 3 || nv > 64 || ne < 3 || ne > 96 {
		return nil, fmt.Errorf("type %d: implausible NV=%d NE=%d", typ, nv, ne)
	}

	s := &Ship{Type: typ, Addr: addr}
	maxFace := 0
	for i := 0; i < nv; i++ {
		r := b[headerSize+i*6:]
		sgn := r[3]
		v := Vertex{
			X:   signed(int(r[0]), sgn&0x80 != 0),
			Y:   signed(int(r[1]), sgn&0x40 != 0),
			Z:   signed(int(r[2]), sgn&0x20 != 0),
			Vis: int(sgn & 0x1F),
		}
		v.Faces = [4]int{int(r[4] >> 4), int(r[4] & 0xF), int(r[5] >> 4), int(r[5] & 0xF)}
		s.Vertices = append(s.Vertices, v)
	}
	for i := 0; i < ne; i++ {
		r := b[edgesOff+i*4:]
		e := Edge{Vis: int(r[0]), FaceA: int(r[1] >> 4), FaceB: int(r[1] & 0xF), V1: int(r[2]) / 4, V2: int(r[3]) / 4}
		if e.V1 >= nv || e.V2 >= nv {
			return nil, fmt.Errorf("type %d: edge %d references vertex out of range", typ, i)
		}
		if e.FaceA > maxFace {
			maxFace = e.FaceA
		}
		if e.FaceB > maxFace {
			maxFace = e.FaceB
		}
		s.Edges = append(s.Edges, e)
	}
	// The face count is not stored; it is one more than the highest face index
	// any edge refers to (every face of a closed hull bounds at least one edge).
	nf := maxFace + 1
	for i := 0; i < nf; i++ {
		r := b[facesOff+i*4:]
		sgn := r[0]
		s.Faces = append(s.Faces, Face{
			NX:  signed(int(r[1]), sgn&0x80 != 0),
			NY:  signed(int(r[2]), sgn&0x40 != 0),
			NZ:  signed(int(r[3]), sgn&0x20 != 0),
			Vis: int(sgn & 0x1F),
		})
	}
	return s, nil
}

func signed(mag int, neg bool) int {
	if neg {
		return -mag
	}
	return mag
}

// ParseAll decodes every valid ship blueprint, skipping table slots whose
// header is not a plausible model.
func ParseAll(mem []byte) []*Ship {
	var ships []*Ship
	for t := 1; t <= numTypes; t++ {
		if s, err := Parse(mem, t); err == nil {
			ships = append(ships, s)
		}
	}
	return ships
}
