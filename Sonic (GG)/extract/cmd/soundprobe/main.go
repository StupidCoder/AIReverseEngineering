// soundprobe boots an act and captures the PSG (SN76489) the music driver programs, then
// synthesises the four channels (3 square tones + 1 LFSR noise) by snapshotting the chip
// registers once per video frame (the driver's update rate) and rendering 1/60 s of audio
// per frame with phase-continuous oscillators. Output: a 16-bit mono WAV, plus a report of
// per-channel activity so we can confirm real music is playing in the oracle.
package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strconv"

	"stupidcoder.com/tools/gamegear"
)

const (
	sr      = 44100
	fps     = 60
	perFrm  = sr / fps // 735 samples/frame
)

var vol = func() [16]float64 {
	var t [16]float64
	a := 1.0
	for i := 0; i < 15; i++ {
		t[i] = a
		a *= 0.79432823
	}
	return t
}()

func main() {
	rom, _ := os.ReadFile(os.Args[1])
	out := os.Args[2]
	act := 0
	if len(os.Args) > 3 {
		act, _ = strconv.Atoi(os.Args[3])
	}
	seconds := 12
	m := gamegear.NewMachine(rom)
	for i := 0; i < 700; i++ {
		m.RunFrame()
	}
	for round := 0; round < 40; round++ {
		m.Pad00 = 0x7F
		m.Write(0xD238, byte(act))
		for i := 0; i < 8; i++ {
			m.RunFrame()
			m.Write(0xD238, byte(act))
		}
		m.Pad00 = 0xFF
		for k := 0; k < 242; k++ {
			m.Write(0xD238, byte(act))
			m.RunFrame()
		}
	}
	for i := 0; i < 60; i++ {
		m.RunFrame()
	}

	var phase [3]float64
	var lfsr uint16 = 0x8000
	var noiseAcc float64
	noiseOut := 1.0
	frames := seconds * fps
	pcm := make([]int16, 0, frames*perFrm)
	active := [4]int{}
	for f := 0; f < frames; f++ {
		m.Write(0xD238, byte(act)) // keep the act pinned
		m.RunFrame()
		r := m.PSG.Reg
		for s := 0; s < perFrm; s++ {
			sample := 0.0
			for c := 0; c < 3; c++ {
				per := float64(r[c*2])
				if per < 1 {
					per = 1
				}
				freq := gamegear.PSGClock / (32 * per)
				phase[c] += freq / sr
				if phase[c] >= 1 {
					phase[c] -= math.Floor(phase[c])
				}
				a := vol[r[c*2+1]&0x0F]
				if phase[c] < 0.5 {
					sample += a
				} else {
					sample -= a
				}
			}
			// noise channel
			nr := r[6] & 0x07
			var nper float64
			switch nr & 0x03 {
			case 0:
				nper = 0x10
			case 1:
				nper = 0x20
			case 2:
				nper = 0x40
			default:
				nper = float64(r[4]) // use tone2 period
			}
			if nper < 1 {
				nper = 1
			}
			nfreq := gamegear.PSGClock / (32 * nper)
			noiseAcc += nfreq / sr
			for noiseAcc >= 1 {
				noiseAcc--
				bit := lfsr & 1
				var fb uint16
				if nr&0x04 != 0 { // white
					fb = (lfsr ^ (lfsr >> 3)) & 1
				} else { // periodic
					fb = bit
				}
				lfsr = (lfsr >> 1) | (fb << 15)
				if bit == 1 {
					noiseOut = 1
				} else {
					noiseOut = -1
				}
			}
			sample += noiseOut * vol[r[7]&0x0F] * 0.6
			pcm = append(pcm, int16(math.Max(-1, math.Min(1, sample/4))*30000))
		}
		for c := 0; c < 3; c++ {
			if vol[r[c*2+1]&0x0F] > 0.01 {
				active[c]++
			}
		}
		if vol[r[7]&0x0F] > 0.01 {
			active[3]++
		}
	}
	writeWAV(out, pcm)
	fmt.Printf("act %d: %d frames; channel active-frames tone0=%d tone1=%d tone2=%d noise=%d; final periods=%v\n",
		act, frames, active[0], active[1], active[2], active[3], m.PSG.Reg)
}

func writeWAV(path string, pcm []int16) {
	f, _ := os.Create(path)
	defer f.Close()
	dataLen := len(pcm) * 2
	w := func(v interface{}) { binary.Write(f, binary.LittleEndian, v) }
	f.Write([]byte("RIFF"))
	w(uint32(36 + dataLen))
	f.Write([]byte("WAVEfmt "))
	w(uint32(16))
	w(uint16(1))     // PCM
	w(uint16(1))     // mono
	w(uint32(sr))    // sample rate
	w(uint32(sr * 2))
	w(uint16(2))
	w(uint16(16))
	f.Write([]byte("data"))
	w(uint32(dataLen))
	w(pcm)
}
