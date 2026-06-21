// geom transcribes the engine's vertex builder $5C0AA instruction-for-instruction and
// dumps each section's piece geometry, to recover the exact per-section shape (the
// step/slope surface and the tessellation) rather than approximate it. The two outputs
// per call are a4-shape+$1C650 and a5-shape+$1C718, alternating by the vertex index's
// parity; the read mode (2-byte vs nibble-packed) is set by p2's sign ($1BB79).
package main

import (
	"fmt"
	"os"
	"strconv"

	"stuntcar/extract/track"
)

const (
	base    = 0xE700
	shapeTb = 0x1EFA2
)

var img []byte

func u8(a int) int  { return int(img[a-base]) }
func u16(a int) int { return int(img[a-base])<<8 | int(img[a-base+1]) }

// handle: byte-swap, -$B100, +$1EF82 (same decode as everywhere else).
func handle(w int) int { return ((((w<<8|w>>8)&0xFFFF)-0xB100)&0xFFFF)+0x1EF82 }

// c5C0AA is the literal transcription of $5C0AA. a4/a5 are the decoded p2/attr shape
// addresses; base650/base718 = $1C650/$1C718; p2 supplies the sign mode ($1BB79); d1 is
// the vertex index. Returns the signed coordinate ($5C0AA's d0.w after ASR.w #5).
func c5C0AA(a4, a5, base650, base718, p2, d1 int) int16 {
	d2 := d1
	if int8(p2) >= 0 { // TST.b $1BB79 ; BPL $5C0F6  (positive branch)
		carry := d2 & 1 // LSR.b #1,d2 sets X/C from bit0
		d2 >>= 1
		var d0 int
		if carry != 0 { // BCS $5C11A  (odd -> a5)
			v := u8(a5 + d2)
			d0 = ((v << 1) & 0xE0) | ((v & 0xF) << 8)
			d0 += base718
		} else {
			v := u8(a4 + d2)
			d0 = ((v << 1) & 0xE0) | ((v & 0xF) << 8)
			d0 += base650
		}
		return int16(uint16(d0)) >> 5
	}
	// negative branch ($5C0B6)
	d2 &^= 1          // BCLR.l #0,d2
	if d1&1 != 0 { // BTST.l #0,d1 ; BNE $5C0DC  (odd -> a5)
		d3 := u8(a5 + d2 + 1)
		d0 := ((u8(a5+d2) & 0x7F) << 8) | d3
		d0 += base718
		return int16(uint16(d0)) >> 5
	}
	d3 := u8(a4 + d2 + 1)
	d0 := ((u8(a4+d2) & 0x7F) << 8) | d3
	d0 += base650
	return int16(uint16(d0)) >> 5
}

func main() {
	var err error
	img, err = os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	id, _ := strconv.Atoi(os.Args[2])
	lo, hi := 0, 0
	if len(os.Args) >= 5 {
		lo, _ = strconv.Atoi(os.Args[3])
		hi, _ = strconv.Atoi(os.Args[4])
	}
	im := track.New(img)
	t := im.Spine(id)
	if hi == 0 {
		hi = len(t.Nodes)
	}
	fmt.Printf("track %d: %d sections.  L[k]=$1C650+a4-profile (left rail height), R[k]=$1C718+a5-profile (right rail height); rungs k=0..count/2-1\n", id, len(t.Nodes))
	for i := lo; i < hi && i < len(t.Nodes); i++ {
		n := t.Nodes[i]
		// Per-TYPE shape gives the rung/vertex count ($5FE56: a0=handle($1EF82+nib*2);
		// count = a0[a0[0]]). Count is vertices (L and R both), so rungs = count/2.
		nib := n.Type & 0x0F
		a0 := handle(u16(0x1EF82 + nib*2))
		off := u8(a0)
		count := u8(a0 + off)
		rungs := count / 2
		// p2/attr cross-section shapes give the height profile per rung.
		a4 := handle(u16(shapeTb + (2*n.P2)&0xFF))
		a5 := handle(u16(shapeTb + (2*n.Attr)&0xFF))
		var L, R []int16
		for k := 0; k < rungs; k++ {
			L = append(L, c5C0AA(a4, a5, int(n.X), int(n.Z), n.P2, 2*k))   // even -> left rail
			R = append(R, c5C0AA(a4, a5, int(n.X), int(n.Z), n.P2, 2*k+1)) // odd  -> right rail
		}
		fmt.Printf("  %2d t=%02x p2=%02x attr=%02x cnt=%2d  L:%v\n                              R:%v\n", i, n.Type, n.P2, n.Attr, count, L, R)
	}
}
