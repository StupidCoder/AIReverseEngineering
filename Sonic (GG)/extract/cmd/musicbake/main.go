// musicbake renders each zone's background music to a compressed OGG for the level viewer.
// It boots a representative act on the oracle, snapshots the SN76489 PSG once per video
// frame while the real music driver runs, synthesises the four channels (3 square + LFSR
// noise), and pipes the PCM through ffmpeg (libvorbis) to an OGG. Acts in a zone share the
// zone theme, so one track per zone + the special stage is baked; the viewer maps each act
// to its track. (v1: a fixed-length clip looped in the browser; seamless loop detection is
// a future refinement.)
package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"

	"stupidcoder.com/tools/gamegear"
)

const (
	sr     = 44100
	fps    = 60
	perFrm = sr / fps
	secs   = 30
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

// track: representative act -> output basename
var tracks = []struct {
	act  int
	name string
}{
	{0, "greenhills"}, {3, "bridge"}, {6, "jungle"},
	{9, "labyrinth"}, {12, "scrapbrain"}, {15, "skybase"}, {28, "special"},
}

func main() {
	rom, _ := os.ReadFile(os.Args[1])
	outdir := os.Args[2]
	os.MkdirAll(outdir, 0o755)
	for _, t := range tracks {
		pcm := capture(rom, t.act)
		wav := filepath.Join(outdir, t.name+".wav")
		writeWAV(wav, pcm)
		ogg := filepath.Join(outdir, t.name+".mp3")
		cmd := exec.Command("ffmpeg", "-y", "-loglevel", "error", "-i", wav,
			"-c:a", "libmp3lame", "-b:a", "64k", "-ac", "1", ogg)
		if err := cmd.Run(); err != nil {
			fmt.Printf("  ffmpeg failed for %s: %v\n", t.name, err)
			continue
		}
		os.Remove(wav)
		fi, _ := os.Stat(ogg)
		fmt.Printf("%-12s act %2d -> %s (%d KB)\n", t.name, t.act, filepath.Base(ogg), fi.Size()/1024)
	}
}

func capture(rom []byte, act int) []int16 {
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
	var nAcc, nOut float64
	pcm := make([]int16, 0, secs*fps*perFrm)
	for f := 0; f < secs*fps; f++ {
		m.Write(0xD238, byte(act))
		m.RunFrame()
		r := m.PSG.Reg
		for s := 0; s < perFrm; s++ {
			out := 0.0
			for c := 0; c < 3; c++ {
				per := float64(r[c*2])
				if per < 1 {
					per = 1
				}
				phase[c] += gamegear.PSGClock / (32 * per) / sr
				if phase[c] >= 1 {
					phase[c] -= math.Floor(phase[c])
				}
				if phase[c] < 0.5 {
					out += vol[r[c*2+1]&0x0F]
				} else {
					out -= vol[r[c*2+1]&0x0F]
				}
			}
			nr := r[6] & 0x07
			nper := []float64{0x10, 0x20, 0x40, float64(maxi(int(r[4]), 1))}[nr&3]
			nAcc += gamegear.PSGClock / (32 * nper) / sr
			for nAcc >= 1 {
				nAcc--
				bit := lfsr & 1
				fb := bit
				if nr&0x04 != 0 {
					fb = (lfsr ^ (lfsr >> 3)) & 1
				}
				lfsr = (lfsr >> 1) | (fb << 15)
				nOut = float64(int(bit)*2 - 1)
			}
			out += nOut * vol[r[7]&0x0F] * 0.6
			pcm = append(pcm, int16(math.Max(-1, math.Min(1, out/4))*30000))
		}
	}
	return pcm
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func writeWAV(path string, pcm []int16) {
	f, _ := os.Create(path)
	defer f.Close()
	dl := len(pcm) * 2
	w := func(v interface{}) { binary.Write(f, binary.LittleEndian, v) }
	f.Write([]byte("RIFF"))
	w(uint32(36 + dl))
	f.Write([]byte("WAVEfmt "))
	w(uint32(16))
	w(uint16(1))
	w(uint16(1))
	w(uint32(sr))
	w(uint32(sr * 2))
	w(uint16(2))
	w(uint16(16))
	f.Write([]byte("data"))
	w(uint32(dl))
	w(pcm)
}
