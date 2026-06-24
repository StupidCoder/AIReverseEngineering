package main

import "sort"

const dispatch = 0x24B2 // bank0: per-type word handler table ($00..$56)

// spriteRef is where a type's idle metasprite layout lives in the ROM.
type spriteRef struct {
	kind   string // "anim" (base+frame*18), "direct" (explicit ptr), "" (none/invisible)
	layout int    // file offset of the 18-byte layout (handlers are in banks 1/2 -> file = z80 addr)
	frame  int    // frame id used (for "anim")
}

// handlerAddr returns the z80 handler address for an object type, or 0 if unused.
func handlerAddr(rom []byte, t int) uint16 {
	return uint16(w(rom, dispatch+t*2))
}

// handlerBounds returns, for each handler address, the address of the next handler in
// the same bank (so a linear scan of one handler doesn't run into the next).
func handlerBounds(rom []byte) map[uint16]uint16 {
	var addrs []uint16
	for t := 0; t < 0x57; t++ {
		if a := handlerAddr(rom, t); a != 0 {
			addrs = append(addrs, a)
		}
	}
	sort.Slice(addrs, func(i, j int) bool { return addrs[i] < addrs[j] })
	end := map[uint16]uint16{}
	for i, a := range addrs {
		e := uint16(0xC000)
		bankTop := (a & 0xC000) + 0x4000 // stay within the home slot window
		if i+1 < len(addrs) && addrs[i+1] < bankTop {
			e = addrs[i+1]
		} else if bankTop < e {
			e = bankTop
		}
		end[a] = e
	}
	return end
}

// analyzeSprite scans a handler's byte range for the first (lowest-address) sprite-
// layout assignment, reading the layout pointer straight out of the handler's own
// operands. Three idioms set IX+15/16 (the metasprite pointer the draw $2F07 reads):
//
//   - CALL $7C75 (the shared animator): layout = DE_base + frameId*18. DE is loaded
//     by a nearby LD DE,nn; the frame id is the first byte of the BC animation
//     sequence when BC is a nearby immediate (else 0 = the idle pose).
//   - LD (IX+15),imm / LD (IX+16),imm: an explicit layout pointer.
//   - LD (IX+15),L / LD (IX+16),H preceded by LD HL,nn: an explicit pointer in HL.
//
// The scan is raw bytes (not a decoded stream) so it is immune to the data tables
// many handlers embed; the idioms are distinctive enough to match directly.
func analyzeSprite(rom []byte, t, zone int) spriteRef {
	a := handlerAddr(rom, t)
	if a == 0 {
		return spriteRef{}
	}
	end := int(handlerBounds(rom)[a])
	// nearestBefore finds the closest occurrence of LD <rr>,nn (opcode op, 3 bytes)
	// in the dozen bytes before pos, returning its immediate or -1.
	nearestBefore := func(op byte, pos int) int {
		for i := pos - 3; i >= pos-14 && i >= int(a); i-- {
			if rom[i] == op {
				return int(rom[i+1]) | int(rom[i+2])<<8
			}
		}
		return -1
	}
	// collectBefore returns every LD HL,nn immediate (opcode $21) in the window before
	// pos, in address order, plus whether a zone read (LD A,($D2D5) = 3A D5 D2) appears
	// in that window. Platform/zone-variant handlers load HL = the zone-0 layout first,
	// then overwrite it with the zone-1.. layouts: HL list = [zone0, zone1, .., else].
	collectBefore := func(pos int) ([]int, bool) {
		var hls []int
		zoneSel := false
		for i := pos - 22; i < pos; i++ {
			if i < int(a) {
				continue
			}
			if rom[i] == 0x21 && i+2 < pos {
				hls = append(hls, int(rom[i+1])|int(rom[i+2])<<8)
			}
			if rom[i] == 0x3A && rom[i+1] == 0xD5 && rom[i+2] == 0xD2 {
				zoneSel = true
			}
		}
		return hls, zoneSel
	}
	for off := int(a); off+1 < end && off+8 < len(rom); off++ {
		b := rom[off:]
		// CALL $7C75  (CD 75 7C)
		if b[0] == 0xCD && b[1] == 0x75 && b[2] == 0x7C {
			de := nearestBefore(0x11, off) // LD DE,base
			if de < 0 {
				continue
			}
			// Frame 0 is the layout base = the idle pose (verified on crab/beetle).
			return spriteRef{"anim", de, 0}
		}
		// LD (IX+15),imm ; LD (IX+16),imm  (DD 36 0F i0  DD 36 10 i1)
		if b[0] == 0xDD && b[1] == 0x36 && b[2] == 0x0F &&
			b[4] == 0xDD && b[5] == 0x36 && b[6] == 0x10 {
			return spriteRef{"direct", int(b[3]) | int(b[7])<<8, 0}
		}
		// LD (IX+15),L ; LD (IX+16),H  (DD 75 0F  DD 74 10) with a nearby LD HL,nn.
		// When the handler selects the layout by zone, pick this zone's pointer.
		if b[0] == 0xDD && b[1] == 0x75 && b[2] == 0x0F &&
			b[3] == 0xDD && b[4] == 0x74 && b[5] == 0x10 {
			if hls, zoneSel := collectBefore(off); len(hls) > 0 {
				ptr := hls[len(hls)-1]
				if zoneSel && len(hls) > 1 {
					if zone < len(hls) {
						ptr = hls[zone] // [zone0, zone1, .., else]; clamp below
					}
				}
				return spriteRef{"direct", ptr, 0}
			}
		}
	}
	return spriteRef{}
}
