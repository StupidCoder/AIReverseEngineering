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
