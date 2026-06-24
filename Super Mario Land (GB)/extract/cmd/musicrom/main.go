// musicrom renders Super Mario Land's music and sound to MP3. The SML sound engine (bank 3,
// entered from the timer ISR at $7FF0 -> $6762; see Super_Mario_Land.md Part VI) is a
// sophisticated multi-channel SFX/sequence driver; rather than reimplement it byte for byte
// we run the real engine in our Game Boy emulator (tools/gameboy) and capture the writes it
// makes to the APU registers ($FF10-$FF3F), then synthesise those through our DMG APU
// emulator (tools/gameboy/apu.go) — the same "decode the structure, render the raw hardware
// output" split this project uses for graphics. ffmpeg (libmp3lame) encodes the MP3.
//
// A song is selected by writing its id to the music request slot $DFE8 and letting the engine
// play; we capture a fixed window and trim to a whole number of seconds.
//
//	go run ./cmd/musicrom [-rom PATH] [-o DIR]
package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"stupidcoder.com/tools/gameboy"
)

// songs to render: each music id (written to the song-request slot $DFE8) and an output
// name. Id $07 is the overworld/main theme (verified by its melody); the rest keep id-based
// names — they are clearly distinct songs but naming every one needs a listen-comparison.
var songs = []struct {
	id   byte
	name string
}{
	{0x01, "music-01"}, {0x02, "music-02"}, {0x03, "music-03"}, {0x04, "music-04"},
	{0x05, "music-05"}, {0x06, "music-06"}, {0x07, "overworld"}, {0x08, "music-08"},
	{0x09, "music-09"}, {0x0A, "music-10"}, {0x0B, "music-11"}, {0x0C, "music-12"},
}

func main() {
	rom := flagStr("-rom", "../Super Mario Land (World).gb")
	out := flagStr("-o", "../rendered/music")
	data, err := os.ReadFile(rom)
	ck(err)
	ck(os.MkdirAll(out, 0o755))

	for _, s := range songs {
		pcm := renderSong(data, s.id, 16.0)
		wav := filepath.Join(out, s.name+".wav")
		writeWAV(wav, pcm)
		mp3 := filepath.Join(out, s.name+".mp3")
		c := exec.Command("ffmpeg", "-y", "-loglevel", "error", "-i", wav,
			"-c:a", "libmp3lame", "-b:a", "96k", "-ac", "1", mp3)
		if e := c.Run(); e != nil {
			fmt.Printf("  ffmpeg failed for %s: %v\n", s.name, e)
			continue
		}
		os.Remove(wav)
		fi, _ := os.Stat(mp3)
		fmt.Printf("%-12s id $%02X -> %s (%d KB)\n", s.name, s.id, s.name+".mp3", fi.Size()/1024)
	}
}

// renderSong boots into gameplay, triggers song `id`, and captures APU writes for `secs`.
func renderSong(rom []byte, id byte, secs float64) []int16 {
	m := gameboy.NewMachine(rom)
	m.RunFrames(120)
	for f := 0; f < 6; f++ {
		m.Buttons = gameboy.BtnStart
		m.RunFrame()
	}
	m.Buttons = 0
	m.RunFrames(60) // settle into gameplay

	// Select the song: write its id to the request slot and let the engine switch to it.
	for f := 0; f < 4; f++ {
		m.Write(0xDFE8, id)
		m.RunFrame()
	}

	// Seed the APU with the current register state (master volume/panning, wave RAM and the
	// channel config were set before our capture window, so the renderer must start from
	// them). The trigger bit (7) of the freq-hi registers is cleared so the seed doesn't
	// spuriously re-trigger a note; the song's own writes do the triggering.
	var events []gameboy.RegWrite
	for reg := uint16(0xFF10); reg <= 0xFF3F; reg++ {
		v := m.Read(reg)
		if reg == 0xFF14 || reg == 0xFF19 || reg == 0xFF1E || reg == 0xFF23 {
			v &= 0x7F
		}
		events = append(events, gameboy.RegWrite{Cycle: 0, Reg: reg, Val: v})
	}

	// Capture APU writes from now on.
	base := m.Cycles
	m.OnWrite = func(pc, addr uint16, v byte) {
		if addr >= 0xFF10 && addr <= 0xFF3F {
			events = append(events, gameboy.RegWrite{Cycle: m.Cycles - base, Reg: addr, Val: v})
		}
	}
	frames := int(secs * 59.7275)
	for f := 0; f < frames; f++ {
		m.RunFrame()
	}
	total := m.Cycles - base

	apu := gameboy.NewAPU()
	return normalize(apu.Render(events, total))
}

// normalize peak-scales the PCM to ~90% full scale so every track is comfortably audible.
func normalize(pcm []int16) []int16 {
	peak := 1
	for _, s := range pcm {
		if v := int(s); v > peak {
			peak = v
		} else if -v > peak {
			peak = -v
		}
	}
	g := 29500.0 / float64(peak)
	for i, s := range pcm {
		v := float64(s) * g
		if v > 32767 {
			v = 32767
		} else if v < -32768 {
			v = -32768
		}
		pcm[i] = int16(v)
	}
	return pcm
}

func writeWAV(path string, pcm []int16) {
	f, err := os.Create(path)
	ck(err)
	defer f.Close()
	n := len(pcm)
	dataLen := n * 2
	f.Write([]byte("RIFF"))
	binary.Write(f, binary.LittleEndian, uint32(36+dataLen))
	f.Write([]byte("WAVEfmt "))
	binary.Write(f, binary.LittleEndian, uint32(16))
	binary.Write(f, binary.LittleEndian, uint16(1))                  // PCM
	binary.Write(f, binary.LittleEndian, uint16(1))                  // mono
	binary.Write(f, binary.LittleEndian, uint32(gameboy.APURate))    // sample rate
	binary.Write(f, binary.LittleEndian, uint32(gameboy.APURate*2))  // byte rate
	binary.Write(f, binary.LittleEndian, uint16(2))                  // block align
	binary.Write(f, binary.LittleEndian, uint16(16))                 // bits
	f.Write([]byte("data"))
	binary.Write(f, binary.LittleEndian, uint32(dataLen))
	binary.Write(f, binary.LittleEndian, pcm)
}

func flagStr(name, def string) string {
	for i, a := range os.Args {
		if a == name && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	return def
}
func ck(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "musicrom:", err)
		os.Exit(1)
	}
}
