// Package level reimplements Super Mario Land's background-map decoder (the routine
// at $218F in the ROM) so a level's map can be extracted straight from the cartridge,
// not pieced together from the oracle's RAM.
//
// The format (all pointers are bank-relative into the $4000-$7FFF window):
//
//	$4000[ffe4]                -> P1, a per-level table of block pointers
//	P1[block]                  -> P2, a block = 20 columns of RLE data, read in order
//	P1 is terminated by an entry whose low byte is $FF.
//
// A column is 16 tiles tall and built from runs:
//
//	run byte rrrrcccc : start at row r (high nibble); cccc tiles follow (count, 0 = 16)
//	  each of the next `count` bytes is a tile placed in consecutive rows, EXCEPT
//	  $FD <tile> : fill the rest of this run with one tile
//	$FE : end of the column (the next column resumes after it)
//
// Tiles are 8x8 background tile indices (special ids like $70/$80 are normal tiles
// that the engine also hangs behaviour on; for the map they are just indices). The
// column buffer starts blank ($2C, the space tile), as the ROM routine initialises it.
package level

const blankTile = 0x2C

// bankByte reads ROM address addr with the given bank paged into $4000-$7FFF.
func bankByte(rom []byte, bank int, addr uint16) byte {
	var off int
	if addr < 0x4000 {
		off = int(addr)
	} else {
		off = bank*0x4000 + int(addr) - 0x4000
	}
	if off >= 0 && off < len(rom) {
		return rom[off]
	}
	return 0xFF
}

func bankWord(rom []byte, bank int, addr uint16) uint16 {
	return uint16(bankByte(rom, bank, addr)) | uint16(bankByte(rom, bank, addr+1))<<8
}

// DecodeColumn decodes one column starting at ptr and returns the 16 tile rows plus
// the pointer just past the column's $FE terminator.
func DecodeColumn(rom []byte, bank int, ptr uint16) (col [16]byte, next uint16) {
	for i := range col {
		col[i] = blankTile
	}
	for {
		run := bankByte(rom, bank, ptr)
		ptr++
		if run == 0xFE { // end of column
			return col, ptr
		}
		row := int(run >> 4)
		count := int(run & 0x0F)
		if count == 0 {
			count = 16
		}
		for i := 0; i < count; {
			tile := bankByte(rom, bank, ptr)
			ptr++
			if tile == 0xFD { // fill the rest of the run with one tile
				fill := bankByte(rom, bank, ptr)
				ptr++
				for ; i < count; i++ {
					if row+i < 16 {
						col[row+i] = fill
					}
				}
				break
			}
			if row+i < 16 {
				col[row+i] = tile
			}
			i++
		}
	}
}

// DecodeLevel decodes a level selected by ffe4 (the index into the $4000 page-table
// pointer table in the given bank), reimplementing the ROM's $218F path:
// $4000[ffe4] -> P1, then DecodeLevelAt.
func DecodeLevel(rom []byte, bank int, ffe4 byte) [][16]byte {
	p1 := bankWord(rom, bank, 0x4000+uint16(ffe4)*2)
	return DecodeLevelAt(rom, bank, p1)
}

// DecodeLevelAt decodes every column of a level given the page-pointer table p1
// (a list of 16-bit page-data pointers, terminated by an entry whose low byte is
// $FF). Each page is 20 columns of RLE read sequentially; bank is paged into
// $4000-$7FFF. Returns columns[x][row], 16 rows tall.
func DecodeLevelAt(rom []byte, bank int, p1 uint16) [][16]byte {
	const maxPages = 256
	var cols [][16]byte
	for pg := 0; pg < maxPages; pg++ {
		entryAddr := p1 + uint16(pg)*2
		if bankByte(rom, bank, entryAddr) == 0xFF { // table terminator
			break
		}
		ptr := bankWord(rom, bank, entryAddr)
		for c := 0; c < 20; c++ { // each page is 20 columns
			col, next := DecodeColumn(rom, bank, ptr)
			cols = append(cols, col)
			ptr = next
		}
	}
	return cols
}
