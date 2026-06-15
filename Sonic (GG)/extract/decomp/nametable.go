package decomp

// LoadRLE reimplements the name-table loader the engine calls at $0502 — a small
// run-length codec distinct from the $0406 tile codec. The stream is a list of tile
// bytes: normally one literal entry each, but when a tile byte equals the one just
// written it switches to a run — the duplicate is followed by a count byte and the
// tile is emitted that many more times. An $FF tile terminates. Every emitted entry
// is the pair (tile, hi), where hi is the shared high byte the engine takes from
// $D20F (palette select + priority bits). It returns the decoded name-table entries
// as 2-byte little-endian words (tile, hi), ready to drop into a name table.
//
// fileOff is the already-resolved file offset of the source; srcLen bounds how many
// source bytes to consume (the engine passes this in BC), independent of the $FF
// terminator — decoding stops at whichever comes first.
func LoadRLE(rom []byte, fileOff, srcLen int, hi byte) []byte {
	out := []byte{}
	i := fileOff
	end := fileOff + srcLen
	prev := ^rom[i] // sentinel: the first byte can never match, so it is a literal
	for i < end && i < len(rom) {
		b := rom[i]
		switch {
		case b == prev: // run: this duplicate + the next byte (a count) repeat the tile
			i++
			if i >= len(rom) || i >= end {
				return out
			}
			n := int(rom[i])
			for k := 0; k < n; k++ {
				out = append(out, b, hi)
			}
			i++
			if i < len(rom) {
				prev = ^rom[i] // re-arm the sentinel so the next byte is a literal
			}
		case b == 0xFF: // end of stream
			return out
		default: // literal
			out = append(out, b, hi)
			prev = b
			i++
		}
	}
	return out
}
