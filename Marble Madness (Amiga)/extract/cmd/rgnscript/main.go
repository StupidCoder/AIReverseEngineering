// rgnscript disassembles the per-region BYTECODE scripts of a course's dynamic
// (animated) terrain regions — the drawbridge, funnels, seesaws, sliding walls.
//
// Each course's Track file holds, at header +$14, an animation block whose first
// long heads a list of 6-byte [x][y][scriptPtr] region records ($FF-terminated).
// scriptPtr points into the same Track segment at the region's script, which the
// engine's interpreter region_script ($FD68, Marble_Madness.md Part V) walks: a
// stream of word opcodes (0..$12 into the 19-entry jump table $FD96), each opcode
// followed by its operands. The interpreter runs opcodes in a loop until a STOP op
// (op1/op3/op15/op17) ends the frame; KEYFRAME (op0) plants the ref point the marble
// rolls toward, the control-flow ops (op9 JUMP / op10 CALL / op11 RET / op12-13 LINK)
// chain scripts, and op16 MOVE drifts the ref point.
//
// This tool replays that operand grammar to print a readable disassembly of every
// region's script (or one, with -region N). It is a LINEAR disassembler over each
// script's stored byte range [scriptPtr, nextScriptPtr); jump/call targets are shown
// as absolute Track offsets, not followed.
//
// Usage: rgnscript <disk.adf> <course-key> [-region N]
//
//	course-key: practy|beginr|interm|aerial|silly|ultima
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"stupidcoder.com/tools/amiga/adf"
	"stupidcoder.com/tools/amiga/hunk"
)

func u32(b []byte, o uint32) uint32 {
	if int(o)+4 > len(b) {
		return 0
	}
	return uint32(b[o])<<24 | uint32(b[o+1])<<16 | uint32(b[o+2])<<8 | uint32(b[o+3])
}
func s16(b []byte, o uint32) int {
	if int(o)+2 > len(b) {
		return 0
	}
	v := int(b[o])<<8 | int(b[o+1])
	if v >= 0x8000 {
		v -= 0x10000
	}
	return v
}

var tracks = map[string]string{
	"practy": "PrcTrack", "beginr": "BegTrack", "interm": "IntTrack",
	"aerial": "AerTrack", "silly": "SilTrack", "ultima": "UltTrack",
}

func main() {
	region := flag.Int("region", -1, "disassemble only region index N (default: all)")
	flag.Parse()
	if flag.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: rgnscript [-region N] <disk.adf> <course-key>")
		os.Exit(2)
	}
	img, err := os.ReadFile(flag.Arg(0))
	chk(err)
	vol, err := adf.Open(img)
	chk(err)
	tr, ok := tracks[strings.ToLower(flag.Arg(1))]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown course %q\n", flag.Arg(1))
		os.Exit(2)
	}
	var path string
	chk(vol.Walk(func(e adf.Entry) error {
		if !e.IsDir && strings.EqualFold(e.Name, tr) {
			path = e.Path
		}
		return nil
	}))
	data, err := vol.ReadFile(path)
	chk(err)
	prog, err := hunk.Load(data, 0)
	chk(err)
	im := prog.Image

	// the +$14 dynamic-region list: 6-byte [x][y][scriptPtr] records, $FF-terminated.
	dyn := u32(im, u32(im, 0x14))
	type reg struct {
		idx, x, y int
		script    uint32
	}
	var regs []reg
	var ptrs []uint32
	for o, i := dyn, 0; int(o)+6 <= len(im); o, i = o+6, i+1 {
		if im[o] == 0xFF {
			break
		}
		sp := u32(im, o+2)
		regs = append(regs, reg{i, int(im[o]), int(im[o+1]), sp})
		ptrs = append(ptrs, sp)
	}
	// each script is stored up to the next script's start (sorted) — the linear end.
	sort.Slice(ptrs, func(a, b int) bool { return ptrs[a] < ptrs[b] })
	end := func(sp uint32) uint32 {
		for _, p := range ptrs {
			if p > sp {
				return p
			}
		}
		return sp + 64
	}

	fmt.Printf("%s (%s): %d dynamic regions\n", flag.Arg(1), tr, len(regs))
	for _, r := range regs {
		if *region >= 0 && r.idx != *region {
			continue
		}
		fmt.Printf("\nregion %d  trigger cell (%d,%d)  script $%05X..$%05X\n",
			r.idx, r.x, r.y, r.script, end(r.script))
		disasm(im, r.script, end(r.script))
	}
}

// disasm linearly decodes the region-script opcode stream in [pc,stop).
func disasm(im []byte, pc, stop uint32) {
	for pc < stop && int(pc)+2 <= len(im) {
		at := pc
		op := s16(im, pc) & 0xFFFF
		pc += 2
		switch op {
		case 0: // KEYFRAME refX,refY,refZ,dur,terr  (+ if dur==1: w26,w28,link long)
			refX, refY, refZ := s16(im, pc), s16(im, pc+2), s16(im, pc+4)
			dur, terr := s16(im, pc+6), s16(im, pc+8)
			pc += 10
			extra := ""
			if dur == 1 {
				link := u32(im, pc+4)
				extra = fmt.Sprintf("  w26=%d w28=%d link=$%05X", s16(im, pc), s16(im, pc+2), link)
				pc += 8
			}
			fmt.Printf("  $%05X  op0  KEYFRAME pos=(%d,%d) z=%d dur=%d terr=%d%s\n", at, refX, refY, refZ, dur, terr, extra)
		case 1:
			fmt.Printf("  $%05X  op1  STATE-STOP +$1C=%d +$21=%d\n", at, s16(im, pc), s16(im, pc+2))
			pc += 4
		case 2:
			fmt.Printf("  $%05X  op2  SPRITE +$1C=%d +$23=%d\n", at, s16(im, pc), s16(im, pc+2))
			pc += 4
		case 3:
			fmt.Printf("  $%05X  op3  STATE0-STOP +$1C=%d\n", at, s16(im, pc))
			pc += 2
		case 4:
			fmt.Printf("  $%05X  op4  LOOP-A count=%d (top=here)\n", at, s16(im, pc))
			pc += 2
		case 5:
			fmt.Printf("  $%05X  op5  NEXT-A (dbra to LOOP-A top)\n", at)
		case 6:
			fmt.Printf("  $%05X  op6  LOOP-B count=%d (top=here)\n", at, s16(im, pc))
			pc += 2
		case 7:
			fmt.Printf("  $%05X  op7  NEXT-B (dbra to LOOP-B top)\n", at)
		case 8:
			fmt.Printf("  $%05X  op8  IF-MARBLE-ON-REGION arg=%d JUMP $%05X\n", at, s16(im, pc), u32(im, pc+2))
			pc += 6
		case 9:
			fmt.Printf("  $%05X  op9  JUMP $%05X\n", at, u32(im, pc))
			pc += 4
		case 10:
			fmt.Printf("  $%05X  op10 CALL $%05X\n", at, u32(im, pc))
			pc += 4
		case 11:
			fmt.Printf("  $%05X  op11 RETURN\n", at)
		case 12:
			fmt.Printf("  $%05X  op12 LINK-46 $%05X\n", at, u32(im, pc))
			pc += 4
		case 13:
			fmt.Printf("  $%05X  op13 LINK-4A $%05X\n", at, u32(im, pc))
			pc += 4
		case 14:
			fmt.Printf("  $%05X  op14 SET-VEL vX=%d vY=%d\n", at, s16(im, pc), s16(im, pc+2))
			pc += 4
		case 15:
			fmt.Printf("  $%05X  op15 ACTIVATE-STOP\n", at)
		case 16:
			fmt.Printf("  $%05X  op16 MOVE (drift ref by vX/vY)\n", at)
		case 17:
			fmt.Printf("  $%05X  op17 FALL-STOP code=%d\n", at, s16(im, pc))
			pc += 2
		case 18:
			fmt.Printf("  $%05X  op18 IF-MARBLE-TERRAIN JUMP $%05X\n", at, u32(im, pc))
			pc += 6
		default:
			fmt.Printf("  $%05X  .word $%X  (>= $13: end/data)\n", at, op)
			return
		}
	}
}

func chk(e error) {
	if e != nil {
		panic(e)
	}
}
