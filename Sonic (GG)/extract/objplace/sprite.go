package objplace

import (
	"sort"

	"sonicgg/extract/decomp"
)

// SpriteRef is where a type's idle metasprite layout lives in the ROM.
type SpriteRef struct {
	Kind   string // "anim" (base+frame*18), "direct" (explicit ptr), "" (none/invisible)
	Layout int    // file offset of the 18-byte layout (handlers are in banks 1/2 -> file = z80 addr)
	Frame  int    // frame id used (for "anim")
}

// handlerAddr returns the z80 handler address for an object type, or 0 if unused.
func handlerAddr(rom []byte, t int) uint16 {
	return uint16(word(rom, dispatch+t*2))
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
func AnalyzeSprite(rom []byte, t, zone int) SpriteRef {
	a := handlerAddr(rom, t)
	if a == 0 {
		return SpriteRef{}
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
			return SpriteRef{"anim", de, 0}
		}
		// LD (IX+15),imm ; LD (IX+16),imm  (DD 36 0F i0  DD 36 10 i1)
		if b[0] == 0xDD && b[1] == 0x36 && b[2] == 0x0F &&
			b[4] == 0xDD && b[5] == 0x36 && b[6] == 0x10 {
			return SpriteRef{"direct", int(b[3]) | int(b[7])<<8, 0}
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
				return SpriteRef{"direct", ptr, 0}
			}
		}
	}
	return SpriteRef{}
}

// CommonTilesFile is the COMMON sprite tile block the level loader decompresses to
// VRAM $3000 (sprite tiles $80-$BF) alongside the zone sheets ($0406 call with
// HL=$B354/DE=$3000, bank 11): HUD digits, sparkles, the item-box bottom, springs.
// Oracle-verified byte-identical to live VRAM tiles $80-$B3 ($B4-$BB are later
// overwritten by Sonic's dynamic frame stream).
const CommonTilesFile = 0x2F354

// SpriteSheet builds act i's full 256-tile sprite sheet exactly as the level loader
// lays out sprite VRAM: the zone's own sheet at tiles $00-$7F (descriptor +23/+24,
// VRAM $2000) and the common block at $80-$BF (VRAM $3000). Layout cells above the
// 128-tile zone sheet (e.g. the item box's bottom row $AA-$AF) resolve into the
// common block.
func SpriteSheet(rom []byte, act int) []byte {
	const descTable = 0x15600
	d := descTable + word(rom, descTable+act*2)
	off := decomp.SourceOffset(int(rom[d+23]), uint16(word(rom, d+24)))
	tiles := make([]byte, 256*32)
	copy(tiles, decomp.Decompress(rom, off))
	copy(tiles[0x80*32:], decomp.Decompress(rom, CommonTilesFile))
	return tiles
}

// ApplyIconUpload emulates the item handlers' lazy tile upload $0BA8 (8 bytes/frame
// from bank 5 into VRAM $2B80 = sprite tiles $5C-$5F over 16 frames): the 16x16 icon
// on the monitor's screen. The source is the LD HL,nn preceding the CALL $0BA8 in the
// handler, so each type shows its own icon (bonus $01 = file $15200, $02 = $15280,
// emerald $06 = $15480). Returns a patched copy when the handler uploads one, else
// the sheet unchanged (the slots hold the zone sheet's own tiles $5C-$5F).
func ApplyIconUpload(rom, tiles []byte, typ int) []byte {
	a, end := HandlerRange(rom, typ)
	for o := a; a != 0 && o+2 < end; o++ {
		if rom[o] == 0xCD && rom[o+1] == 0xA8 && rom[o+2] == 0x0B { // CALL $0BA8
			for j := o - 3; j >= o-16 && j >= a; j-- {
				if rom[j] == 0x21 { // LD HL,nn
					src := 0x14000 + (int(rom[j+1]) | int(rom[j+2])<<8) - 0x4000
					patched := append([]byte(nil), tiles...)
					copy(patched[0x5C*32:0x60*32], rom[src:src+128])
					return patched
				}
			}
		}
	}
	return tiles
}

// AnimFrame is one step of a metasprite animation: the 18-byte layout's file offset
// and how many engine frames it is shown ($7C75 shows a step for duration+1 frames).
type AnimFrame struct {
	Layout int
	Frames int
}

// Anim returns the object type's idle/walk animation as layout+duration steps.
//
//   - Handlers driven by the shared animator $7C75 (frameBase in DE, sequence in BC:
//     (frameId, duration) byte pairs, $FF loops; layout = base + frameId*18) yield the
//     full parsed sequence of their FIRST $7C75 site (the default/idle animation).
//   - The pickup family blinks between two direct layouts on the $D224 rotor
//     ($5E17: rotor&7 < 5 shows the second): the base TV with its "static" screen
//     cell for 3 frames, the open-screen variant (the icon shows through) for 5.
//   - Anything else with a single direct layout is one static frame.
func Anim(rom []byte, typ, zone int) []AnimFrame {
	a, end := HandlerRange(rom, typ)
	if a == 0 {
		return nil
	}
	imm16 := func(op byte, pos int) int { // nearest LD rr,nn before pos
		for i := pos - 3; i >= pos-20 && i >= a; i-- {
			if rom[i] == op {
				return int(rom[i+1]) | int(rom[i+2])<<8
			}
		}
		return -1
	}
	// Shared animator: parse the (frameId, duration) sequence. The sequence pointer
	// is either a direct LD BC,nn, or — the common enemy idiom — entry 0 of a
	// per-state pointer table: LD HL,table / ADD HL,DE / LD C,(HL) / INC HL /
	// LD B,(HL) (bytes 19 4E 23 46) before the CALL. A handler with one $7C75 site
	// per facing stores the X velocity just before each — prefer the site with the
	// NEGATIVE velocity (walking left, every walker's initial direction; the
	// porcupine's first site is its facing-right walk).
	var sites []int
	for o := a; o+2 < end; o++ {
		if rom[o] == 0xCD && rom[o+1] == 0x75 && rom[o+2] == 0x7C { // CALL $7C75
			sites = append(sites, o)
		}
	}
	ordered := append([]int(nil), sites...)
	for _, o := range sites {
		for i := o - 20; i > a && i < o-3; i++ {
			// LD (IX+8/9),$FF = a negative X-velocity byte
			if rom[i] == 0xDD && rom[i+1] == 0x36 && (rom[i+2] == 8 || rom[i+2] == 9) && rom[i+3] == 0xFF {
				ordered = append([]int{o}, ordered...)
				break
			}
		}
	}
	for _, o := range ordered {
		{
			base, seq := imm16(0x11, o), imm16(0x01, o)
			if base < 0 {
				continue
			}
			if seq < 0 {
				for i := o - 24; i > a && i < o-6; i++ {
					if rom[i] == 0x21 && rom[i+3] == 0x19 && rom[i+4] == 0x4E &&
						rom[i+5] == 0x23 && rom[i+6] == 0x46 {
						tab := int(rom[i+1]) | int(rom[i+2])<<8
						seq = int(rom[tab]) | int(rom[tab+1])<<8
						break
					}
				}
			}
			if seq < 0 { // no sequence found: the idle pose alone
				return []AnimFrame{{base, 0}}
			}
			var out []AnimFrame
			for p := seq; p+1 < len(rom) && len(out) < 16; p += 2 {
				if rom[p] == 0xFF {
					break
				}
				out = append(out, AnimFrame{base + int(rom[p])*18, int(rom[p+1]) + 1})
			}
			if len(out) > 0 {
				return out
			}
			return []AnimFrame{{base, 0}}
		}
	}
	// Direct layouts: collect every LD (IX+15),imm / LD (IX+16),imm pair.
	var direct []int
	for o := a; o+7 < end; o++ {
		if rom[o] == 0xDD && rom[o+1] == 0x36 && rom[o+2] == 0x0F &&
			rom[o+4] == 0xDD && rom[o+5] == 0x36 && rom[o+6] == 0x10 {
			direct = append(direct, int(rom[o+3])|int(rom[o+7])<<8)
		}
	}
	if len(direct) >= 2 && HasSpawnAdjust(rom, typ) {
		return []AnimFrame{{direct[0], 3}, {direct[1], 5}} // the TV blink
	}
	r := AnalyzeSprite(rom, typ, zone)
	if r.Kind == "" || r.Layout == 0 {
		return nil
	}
	return []AnimFrame{{r.Layout, 0}}
}

// The pickup handlers overlay their 16x16 screen icon as two 8x16 hardware sprites
// (tiles $5C and $5E) at (X+4, Y) and (X+12, Y) — emitted via $2F5D before the
// metasprite, so the icon sits on top and the blinking screen cell shows through
// its transparent pixels ($5E5D-$5E76).
type IconOverlay struct{ X, Y, Tile int }

func IconOverlays(rom []byte, typ int) []IconOverlay {
	if !HasSpawnAdjust(rom, typ) {
		return nil
	}
	// only the types that actually upload an icon show one
	patched := ApplyIconUpload(rom, make([]byte, 256*32), typ)
	if len(patched) == 256*32 {
		blank := true
		for _, b := range patched[0x5C*32 : 0x60*32] {
			if b != 0 {
				blank = false
				break
			}
		}
		if blank {
			return nil
		}
	}
	return []IconOverlay{{4, 0, 0x5C}, {12, 0, 0x5E}}
}

// BgPhase is one phase of the type-$50 background-cell animator: the two 16x16
// blocks painted over the object's 16x32 strip (top/bottom, each 2x2 BG tiles,
// row-major) and how many frames the phase holds.
type BgPhase struct {
	Tiles  [8]int // top 2x2 then bottom 2x2
	Frames int
}

// BgAnim parses the animator's pattern table ($7BC1: 4 phases x (top block, bottom
// block, next-duration, pad)) and block defs ($7B99: 8 bytes = 2x2 name-table cells).
// The countdown decrements every second frame, so a phase lasts count*2 frames; the
// count for phase i+1 is phase i's third byte (steady state — the initial phase-0
// count of 50 only applies once at spawn).
func BgAnim(rom []byte) []BgPhase {
	const patterns, defs = 0x7BC1, 0x7B99
	blockTiles := func(id int) [4]int {
		var t [4]int
		for i := 0; i < 4; i++ {
			t[i] = int(rom[defs+id*8+i*2]) // tile bytes; attrs (odd bytes) are 0
		}
		return t
	}
	out := make([]BgPhase, 4)
	for ph := 0; ph < 4; ph++ {
		top := blockTiles(int(rom[patterns+ph*4]))
		bot := blockTiles(int(rom[patterns+ph*4+1]))
		copy(out[ph].Tiles[0:], top[:])
		copy(out[ph].Tiles[4:], bot[:])
		out[ph].Frames = int(rom[patterns+((ph+3)&3)*4+2]) * 2
	}
	return out
}

// --- Sonic's own animation system ($4E6D sequencer) --------------------------------
//
// Sonic is not animated by $7C75: his handler keeps ONE metasprite layout ($5C1B, his
// 16x32 box at the grid origin) and instead re-streams the tile GRAPHICS per pose.
// The anim id (IX+20) indexes a word table at $5C5B; the sequence is one byte per
// engine frame = the graphic frame id (tiles at bank 8 + frame*192, 3bpp — see
// $4E9A), and a byte with bit 7 set is a control: the next byte is the new cursor
// (the loop point). Anim $05 = standing (frame 0), $01 = the walk (4-9, 8 frames
// each), $02 = rolling (11-14), $0D = BORED (2x16, 1x18, then a 2/3 foot-tap loop),
// set by the idle timeout at $5379.

const (
	sonicAnimTable = 0x5C5B
	sonicGfxBase   = 0x20000 // bank 8: graphic frame id * 192 (8 tiles x 24 bytes, 3bpp)
)

// SonicSeq returns one anim id's sequence as (graphic frame, hold frames) steps plus
// the loop-back step index.
func SonicSeq(rom []byte, anim int) (steps []AnimFrame, loopStep int) {
	return perFrameSeq(rom, word(rom, sonicAnimTable+anim*2), 0xFF)
}

// perFrameSeq parses a one-byte-per-engine-frame sequence (Sonic $4E6D and the goal
// sign $63E7 share the format): each byte is the frame id for that engine frame; a
// byte >= stop is a control whose next byte is the new cursor (the loop point).
// Returns run-length-collapsed steps (frame id in Layout) and the loop step index.
func perFrameSeq(rom []byte, p int, stop byte) (steps []AnimFrame, loopStep int) {
	var perFrame []int
	loopByte := 0
	for q := p; q < p+256; q++ {
		if rom[q] >= stop {
			loopByte = int(rom[q+1])
			break
		}
		perFrame = append(perFrame, int(rom[q]))
	}
	for i := 0; i < len(perFrame); {
		j := i
		for j < len(perFrame) && perFrame[j] == perFrame[i] {
			j++
		}
		if loopByte >= i && loopByte < j {
			loopStep = len(steps)
		}
		steps = append(steps, AnimFrame{Layout: perFrame[i], Frames: j - i}) // Layout = frame id here
		i = j
	}
	return steps, loopStep
}

// SonicFrameTiles expands graphic frame f into the 256-tile sheet slots $B4-$BB his
// layout references (the same expansion the tile streamer performs).
func SonicFrameTiles(rom []byte, f int) []byte {
	tiles := make([]byte, 256*32)
	for t := 0; t < 8; t++ {
		for r := 0; r < 8; r++ {
			src := sonicGfxBase + f*192 + t*24 + r*3
			dst := (0xB4+t)*32 + r*4
			copy(tiles[dst:dst+3], rom[src:src+3])
		}
	}
	return tiles
}

// --- platform movement paths -------------------------------------------------------

// PlatformPaths samples each moving platform type's positional cycle as per-frame
// (dx,dy) offsets from its placement, exactly as the handlers compute them:
//
//   - $09 swing ($6747): anchor + the 113-pair arc table at $682E (a radius-51
//     pendulum), phase cursor +/-2 per frame ping-ponging between the ends —
//     start at the RIGHT end (phase $E0), 224 frames per period.
//   - $0F horizontal ($6DCA): X += +/-1 per frame, direction toggling every 160
//     frames, starting right — a 320-frame triangle.
//
// (Type $0B, the same platform sprite, only sinks under Sonic's weight — it has no
// idle motion, so no path.)
func PlatformPaths(rom []byte) map[string][][2]int {
	const arcTable = 0x682E
	sgn := func(b byte) int {
		if b >= 0x80 {
			return int(b) - 256
		}
		return int(b)
	}
	pair := func(i int) [2]int {
		return [2]int{sgn(rom[arcTable+i*2]), sgn(rom[arcTable+i*2+1])}
	}
	var swing [][2]int
	for i := 112; i >= 0; i-- { // right end -> left end
		swing = append(swing, pair(i))
	}
	for i := 1; i < 112; i++ { // and back
		swing = append(swing, pair(i))
	}
	var horiz [][2]int
	for t := 0; t < 320; t++ {
		off := t
		if t >= 160 {
			off = 320 - t
		}
		horiz = append(horiz, [2]int{off, 0})
	}
	return map[string][][2]int{"09": swing, "0f": horiz, "3b": BobPath(), "29": LogBobPath()}
}

// OwnGfx reports a handler that decompresses its OWN sprite graphics (the bosses:
// LD HL,src / LD DE,$2000 / LD A,bank / CALL $0406 replaces the zone sprite sheet
// while the set-piece runs) and returns the source file offset and destination
// sprite-tile slot.
func OwnGfx(rom []byte, typ int) (srcFile, dstTile int, ok bool) {
	a, end := HandlerRange(rom, typ)
	for o := a; a != 0 && o+2 < end; o++ {
		if rom[o] == 0xCD && rom[o+1] == 0x06 && rom[o+2] == 0x04 { // CALL $0406
			src, dst, bank := -1, -1, -1
			for i := o - 14; i < o; i++ {
				switch rom[i] {
				case 0x21:
					src = int(rom[i+1]) | int(rom[i+2])<<8
				case 0x11:
					dst = int(rom[i+1]) | int(rom[i+2])<<8
				case 0x3E:
					bank = int(rom[i+1])
				}
			}
			if src >= 0 && bank >= 0 && dst >= 0x2000 && dst < 0x4000 {
				return decomp.SourceOffset(bank, uint16(src)), (dst - 0x2000) / 32, true
			}
		}
	}
	return 0, 0, false
}

// HasGravity reports the per-frame Y-velocity increment idiom in the handler
// (LD L,(IX+10) / LD H,(IX+11) ... LD (IX+10),L): the object pulls itself down
// when airborne, so its rest position is the floor below the spawn.
func HasGravity(rom []byte, typ int) bool {
	a, end := HandlerRange(rom, typ)
	for i := a; a != 0 && i+24 < end; i++ {
		if rom[i] == 0xDD && rom[i+1] == 0x6E && rom[i+2] == 0x0A &&
			rom[i+3] == 0xDD && rom[i+4] == 0x66 && rom[i+5] == 0x0B {
			for j := i + 6; j < i+24; j++ {
				if rom[j] == 0xDD && rom[j+1] == 0x75 && rom[j+2] == 0x0A {
					return true
				}
			}
		}
	}
	return false
}

// NoCollide reports a handler that opts out of terrain physics on entry
// (SET 5,(IX+24) as its first instruction — platforms, floaters, effects).
func NoCollide(rom []byte, typ int) bool {
	a, _ := HandlerRange(rom, typ)
	return a != 0 && rom[a] == 0xDD && rom[a+1] == 0xCB && rom[a+2] == 0x18 && rom[a+3] == 0xEE
}

// BobPath samples the bobbing platform's (type $3B, $B4E8) vertical float: the
// Y velocity takes +/-$10 sub-px per frame — the sign flipping on an 160-frame
// phase counter ($B515, threshold ($D217)=$50) — clamped to +/-2 px/frame
// ($B53C), integrated into the 24-bit position. From the spawn it first sinks
// ~160 px (the transient), then bobs +64..+160 below the spawn with a 160-frame
// period — verified against the live engine. The exported path is one
// steady-state cycle (the viewer loops it).
func BobPath() [][2]int {
	var path [][2]int
	pos, vel := 0, 0
	for t := 0; t < 760; t++ {
		if t%160 < 80 {
			vel += 0x10
		} else {
			vel -= 0x10
		}
		if vel > 0x200 {
			vel = 0x200
		}
		if vel < -0x200 {
			vel = -0x200
		}
		pos += vel
		if t >= 600 {
			path = append(path, [2]int{0, pos >> 8})
		}
	}
	return path
}

// --- the goal sign ($07, handler $61F8) ----------------------------------------------
//
// The end-of-act sign is the one own-gfx object that is a plain metasprite: on its
// first frame it decompresses its OWN sprite sheet over VRAM $2000 ($620E: bank 9
// file $27AB8 — the sign plates and post) and switches the sprite palette to index
// $0E ($6219, via ($D22D)). Its animation uses the same one-byte-per-frame sequencer
// as Sonic ($63E7, cursor IX+18): layout = $652D + frameId*18 ($6404). The IDLE
// sequence at $64A8 spins the sign continuously — plates 0, 3 (quarter), 2 (edge),
// 4, six frames each; the post-clear sequences ($64C2..$6513) stop it on the result
// plate. The Studio viewer shows the idle spin.
const (
	goalLayoutBase = 0x652D
	goalIdleSeq    = 0x64A8
	goalPalette    = 0x0E
)

// GoalAnim returns the goal sign's animation as (layout file offset, hold frames)
// steps, plus its own sheet source and palette index. The engine idles the sign
// STATIC on the "?" plate (state sequence $64C2 = one frame) and plays the $64A8
// spin — plates 0, 3, 2, 4 at six frames each, hopping upward — while Sonic crosses
// it (oracle-verified). For the viewer the steps are the static plate held ~5
// seconds, then three spin cycles, looping — the flowers' wait-then-play pattern.
func GoalAnim(rom []byte) (steps []AnimFrame, sheetSrc, pal int) {
	spin, _ := perFrameSeq(rom, goalIdleSeq, 0xFF)
	steps = append(steps, AnimFrame{Layout: goalLayoutBase, Frames: 300})
	for cycle := 0; cycle < 3; cycle++ {
		for _, st := range spin {
			steps = append(steps, AnimFrame{Layout: goalLayoutBase + st.Layout*18, Frames: st.Frames})
		}
	}
	src, _, _ := OwnGfx(rom, 0x07)
	return steps, src, goalPalette
}

// --- the floating log ($29, handler $7EFC — Jungle's rideable "barrel") -------------
//
// No terrain contact (SET 5); box 10x16. On its first frame it lifts itself 24 px
// ($7F0E: Y += $FFE8, the same one-shot spawn-adjust pattern as the pickups), then
// bobs on the water: the handler SETS the Y velocity to +$40 sub-px (0.25 px/frame,
// down) while its counter IX+17 < 20 and -$40 while 20..39 ($7F22/$7F34, wrap at 40)
// — a gentle 5 px triangle bob. Ridden, it becomes a log-roll: Sonic steers it at
// HALF his speed (the handler adds vel/2 to the log's X and writes the log's X back
// to Sonic, $7F8F-$7FB2), a roll phase accumulates |vel| mod $900 (IX+18/19), and
// the phase's high byte picks one of THREE roll layouts through the offset table
// $8031: layouts $803A + {0, $12, $24} (a 16x16 log, zone sprite tiles). Idle it
// rests on frame 0.
const (
	logLayoutBase = 0x803A
	logFrames     = 3
)

// LogAnim returns the log's three roll layouts.
func LogAnim() []int {
	return []int{logLayoutBase, logLayoutBase + 0x12, logLayoutBase + 0x24}
}

// LogBobPath samples the 40-frame bob (offsets from the adjusted rest position).
func LogBobPath() [][2]int {
	var path [][2]int
	pos := 0
	for t := 0; t < 40; t++ {
		if t < 20 {
			pos += 0x40
		} else {
			pos -= 0x40
		}
		path = append(path, [2]int{0, pos >> 8})
	}
	return path
}

// SpawnYAdjust returns the one-shot Y offset a handler applies to its own spawn
// position in its init block (LD L,(IX+5) / LD H,(IX+6) / LD DE,nn / ADD HL,DE /
// store — the floating log lifts itself 24 px), or 0.
func SpawnYAdjust(rom []byte, typ int) int {
	a, end := HandlerRange(rom, typ)
	for i := a; a != 0 && i+14 < end; i++ {
		if rom[i] == 0xDD && rom[i+1] == 0x6E && rom[i+2] == 0x05 &&
			rom[i+3] == 0xDD && rom[i+4] == 0x66 && rom[i+5] == 0x06 &&
			rom[i+6] == 0x11 && rom[i+9] == 0x19 &&
			rom[i+10] == 0xDD && rom[i+11] == 0x75 && rom[i+12] == 0x05 {
			return int(int16(uint16(rom[i+7]) | uint16(rom[i+8])<<8))
		}
	}
	return 0
}
