package gamegear

// Game Gear / Master System cartridge address mapping. The Z80 sees a 16-bit space,
// but the cartridge is larger and is paged through the Sega mapper: three 16 KB slots
// at $0000/$4000/$8000, each showing a ROM bank, with the first 1 KB fixed to bank 0
// (so the interrupt vectors stay reachable). The same Z80 address therefore names
// different ROM bytes depending on which banks are paged — which is why disassembling
// banked code needs to know the slot configuration, not just the address.

// FileOffset maps a Z80 address to a flat ROM file offset given the banks paged into
// the three slots. inRAM is true for $C000-$FFFF (work RAM), where there is no ROM
// offset. This is the single source of truth for the mapping; Machine.Read uses it too.
func FileOffset(slots [3]int, addr uint16) (off int, inRAM bool) {
	switch {
	case addr < 0x0400: // first 1 KB is always bank 0
		return int(addr), false
	case addr < 0x4000:
		return slots[0]*0x4000 + int(addr), false
	case addr < 0x8000:
		return slots[1]*0x4000 + int(addr-0x4000), false
	case addr < 0xC000:
		return slots[2]*0x4000 + int(addr-0x8000), false
	default:
		return 0, true
	}
}

// BankView assembles the 48 KB ROM window (Z80 $0000-$BFFF) as one flat slice for the
// given slot configuration, so a linear disassembler can read it directly. The returned
// slice is 0xC000 bytes long and indexed by Z80 address; the $C000-$FFFF RAM region is
// not represented (disassemble code, not RAM). Bytes past the end of rom read as $FF.
func BankView(rom []byte, slots [3]int) []byte {
	view := make([]byte, 0xC000)
	for a := 0; a < 0xC000; a++ {
		off, _ := FileOffset(slots, uint16(a))
		if off < len(rom) {
			view[a] = rom[off]
		} else {
			view[a] = 0xFF
		}
	}
	return view
}
