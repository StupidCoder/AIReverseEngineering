package main

import (
	"fmt"
	"math"

	"stupidcoder.com/tools/gameboy"
)

var noteNames = []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

func freqName(f int) string {
	if f <= 0 || f >= 2048 {
		return fmt.Sprintf("<%d>", f)
	}
	hz := 131072.0 / float64(2048-f)
	n := int(math.Round(12*math.Log2(hz/440))) + 69
	if n < 0 {
		return "?"
	}
	return fmt.Sprintf("%s%d", noteNames[n%12], n/12-1)
}

var chReg = [4]struct{ lo, hi, env uint16 }{
	{0xFF13, 0xFF14, 0xFF12}, {0xFF18, 0xFF19, 0xFF17},
	{0xFF1D, 0xFF1E, 0xFF1C}, {0xFF22, 0xFF23, 0xFF21},
}

// verifyMelody prints, per channel, my decoded notes/envelopes next to the engine's, so the
// port can be checked channel-by-channel against ground truth.
func verifyMelody(id int) {
	chs := song(id)

	m := gameboy.NewMachine(rom)
	m.RunFrames(120)
	for f := 0; f < 6; f++ {
		m.Buttons = gameboy.BtnStart
		m.RunFrame()
	}
	m.Buttons = 0
	m.RunFrames(120)
	for f := 0; f < 4; f++ {
		m.Write(0xDFE8, byte(id))
		m.RunFrame()
	}
	var lo, env [4]int
	var engNotes, engEnv [4][]int
	m.OnWrite = func(pc, addr uint16, v byte) {
		for c := 0; c < 4; c++ {
			switch addr {
			case chReg[c].lo:
				lo[c] = (lo[c] & 0x700) | int(v)
			case chReg[c].env:
				env[c] = int(v)
			case chReg[c].hi:
				if v&0x80 != 0 && len(engNotes[c]) < 16 {
					lo[c] = (lo[c] & 0xFF) | int(v&7)<<8
					engNotes[c] = append(engNotes[c], lo[c])
					engEnv[c] = append(engEnv[c], env[c])
				}
			}
		}
	}
	m.RunFrames(400)

	name := []string{"square1", "square2", "wave   ", "noise  "}
	for c := 0; c < 4; c++ {
		fmt.Printf("=== id $%02X ch%d (%s)  mine=%d events loopStart=%d loopLen=%d ===\n", id, c, name[c], len(chs[c].events), chs[c].loopStart, chs[c].loopLen)
		mn, me := "", ""
		for i, e := range chs[c].events {
			if i >= 16 {
				break
			}
			if c == 3 {
				mn += fmt.Sprintf("%d ", e.freq)
			} else {
				mn += freqName(e.freq) + " "
			}
			me += fmt.Sprintf("%02X ", e.inst.env)
		}
		en, ee := "", ""
		for i, f := range engNotes[c] {
			if c == 3 {
				en += fmt.Sprintf("%d ", f)
			} else {
				en += freqName(f) + " "
			}
			ee += fmt.Sprintf("%02X ", engEnv[c][i])
		}
		fmt.Println("  mine:", mn, "| env:", me)
		fmt.Println("  eng :", en, "| env:", ee)
	}
}
