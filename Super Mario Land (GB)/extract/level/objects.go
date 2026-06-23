package level

// Object placement (enemies, moving platforms, the level-end Daisy/coin-lift, etc.)
// is a separate per-level list from the background map, decoded by the ROM's spawner
// at $2492 (init at $2453). It is NOT encoded in the column RLE — only static blocks
// ($70/$80/$5F ?-blocks & breakables) live there.
//
// The chain (reimplemented here, traced from the code, not guessed):
//
//	$401A[ffe4]  (in the world's data bank) -> L, the level's placement list
//	each entry is 3 bytes, sorted ascending by trigger column:
//	  byte0  col   : the scroll column ($C0AB) at which the object spawns. $C0AB
//	                 advances once per 16 px of scroll, so the object's map column
//	                 (8 px tiles) is col*2.
//	  byte1  pos   : bits 0-4 -> map row (packed&$1F); bits 6-7 -> fine X (sub-column).
//	  byte2  type  : bits 0-6 -> object type (indexes the $336C init table);
//	                 bit 7 -> "hard mode" flag (the object is only spawned, or spawned
//	                 differently, on a second quest; $FF9A gates it at $24E6).
//	the list is terminated by an entry whose col byte is $FF.

// Object is one placed object: Col/Row are map-tile coordinates (8x8 tiles), Type is
// the object type id, Hard is the bit-7 second-quest flag, and FineX is the 0-3
// sub-column nudge from the position byte's top two bits.
type Object struct {
	Col, Row int
	Type     byte
	Hard     bool
	FineX    int
}

// DecodeObjectsByID returns the placed objects for a level id (0x11=1-1 .. 0x43=4-3),
// using the same world->bank selection as DecodeLevelByID.
func DecodeObjectsByID(rom []byte, id byte) []Object {
	world := int(id >> 4)
	level := int(id & 0x0F)
	ffe4 := byte((world-1)*3 + (level - 1))
	return DecodeObjects(rom, worldBank[world], ffe4)
}

// --- object metasprites (the sprite graphics an object draws) ---
//
// The object sprite draw ($25B7) renders a slot's metasprite using the slot's frame field
// (slot+6) as an index into the pointer table at $2FD9 (bank-0 fixed). The pointed-at
// stream is "turtle graphics": a byte with bit7 clear moves the 8x16 cursor (low nibble:
// bit3 up / bit2 down / bit1 left / bit0 right, 8 px each) and carries the OAM attribute;
// a byte with bit7 set stamps an 8x16 OBJ sprite of that tile id at the cursor. ($FF ends.)
// SML runs sprites in 8x16 mode, so each emitted tile is a vertical pair (tile&$FE, |1).

const msTable = 0x2FD9 // object metasprite pointer table (bank-0 fixed)

// Sprite is one 8x16 OBJ sprite of a metasprite: the (even) base tile id and its pixel
// offset from the metasprite origin.
type Sprite struct {
	Tile   byte
	DX, DY int
}

// DecodeMetasprite decodes object metasprite frame `f` from the $2FD9 table into its 8x16
// sprites (reimplements the $25B7 stream walker; data is bank-0 fixed so no bank needed).
func DecodeMetasprite(rom []byte, f int) []Sprite {
	p := int(rom[msTable+f*2]) | int(rom[msTable+f*2+1])<<8
	var out []Sprite
	cx, cy := 0, 0
	for i := 0; i < 64 && p >= 0 && p < len(rom); i++ {
		b := rom[p]
		p++
		if b == 0xFF {
			break
		}
		if b&0x80 != 0 {
			out = append(out, Sprite{b & 0xFE, cx, cy})
			continue
		}
		if b&0x08 != 0 {
			cy -= 8
		}
		if b&0x04 != 0 {
			cy += 8
		}
		if b&0x02 != 0 {
			cx -= 8
		}
		if b&0x01 != 0 {
			cx += 8
		}
	}
	return out
}

// Object behaviour is script-driven: the runner at $263F looks up each object's per-type
// script via the table at $3495 (indexed by type id) and interprets it ($26AC). Script
// command $F8 xx sets the animation frame ($FFC6); a type's *base* frame is the first $F8
// in its script. So we can recover a representative metasprite frame for every type by
// reading its script — no per-level observation needed (that earlier approach only saw the
// enemies the oracle's auto-play reached). Verified to match those observations.
const (
	scriptTable = 0x3495 // per-type object script pointer table (bank-0 fixed)
	opSetFrame  = 0xF8   // script opcode: next byte = animation frame id
)

// TypeBaseFrame returns the base metasprite frame for object type `typ` by scanning its
// script for the first $F8 (set-frame) command. ok is false if the type has no valid script.
func TypeBaseFrame(rom []byte, typ byte) (frame byte, ok bool) {
	p := int(rom[scriptTable+int(typ)*2]) | int(rom[scriptTable+int(typ)*2+1])<<8
	if p < 0x3500 || p >= 0x3B00 { // outside the script block => not a real object type
		return 0, false
	}
	for i := 0; i < 96 && p+1 < len(rom); i++ {
		if rom[p] == opSetFrame {
			return rom[p+1], true
		}
		p++
	}
	return 0, false
}

// TypeFrames builds the full type -> base-frame map from the script table.
func TypeFrames(rom []byte) map[byte]byte {
	m := map[byte]byte{}
	for t := 0; t < 0x80; t++ {
		if fr, ok := TypeBaseFrame(rom, byte(t)); ok {
			m[byte(t)] = fr
		}
	}
	return m
}

// DecodeObjects decodes the placement list for global level index ffe4 (0-11) from the
// pointer table at $401A in the given bank.
func DecodeObjects(rom []byte, bank int, ffe4 byte) []Object {
	list := bankWord(rom, bank, 0x401A+uint16(ffe4)*2)
	var objs []Object
	ptr := list
	for i := 0; i < 256; i++ {
		col := bankByte(rom, bank, ptr)
		if col == 0xFF { // list terminator
			break
		}
		pos := bankByte(rom, bank, ptr+1)
		typ := bankByte(rom, bank, ptr+2)
		ptr += 3
		objs = append(objs, Object{
			Col:   int(col) * 2,
			Row:   int(pos & 0x1F),
			Type:  typ & 0x7F,
			Hard:  typ&0x80 != 0,
			FineX: int(pos&0xC0) >> 6,
		})
	}
	return objs
}
