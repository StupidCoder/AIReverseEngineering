// musicrom synthesises Super Mario Land's music ENTIRELY FROM THE ROM song data — a Go port
// of the bank-3 sound engine (Super_Mario_Land.md Part VI), no emulator in the render loop.
//
// Format (all bank-3 addresses):
//   - The song table $673C maps a music id to a song header; $07B7[ffe4] selects which song
//     each of the twelve levels uses, so the tracks are named by that.
//   - A song header is: master byte, a 16-bit pointer to the song's DURATION TABLE, then four
//     16-bit channel-header pointers (square1, square2, wave, noise).
//   - A channel header is an ORDER LIST of 16-bit pattern pointers; lo byte $FF is the LOOP
//     point, $00 ends the song.
//   - A pattern is a byte stream: $9D a b c = set instrument (envelope, duty/len, 3rd byte);
//     $A0-$AF = set note duration to durtable[low nibble]; $00 ends the pattern; note $01 = tie
//     (hold, no retrigger); any other byte N is a note whose pitch is the GB frequency
//     freqtable[$6F70 + N] (two bytes per semitone). The noise channel's N indexes a
//     polynomial-counter table at $7002 instead of a pitch.
//   - The engine ticks at 64 Hz (one durtable unit = 1/64 s).
//
// The decoded notes are rendered through our DMG APU (tools/gameboy/apu.go). Each channel
// loops at its $FF, so the render is trimmed to exactly one loop for a seamless file. ffmpeg
// (libmp3lame) encodes the MP3. -verify boots the real engine in the emulator and prints both
// the decoded and the engine's note streams so the port can be checked.
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"stupidcoder.com/tools/gameboy"
)

const cycPerTick = 65536 // 4194304 Hz / 64 Hz tick

var rom []byte

func b3(a int) int  { return 3*0x4000 + (a - 0x4000) }
func rb(a int) byte { return rom[b3(a)] }
func rw(a int) int  { return int(rom[b3(a)]) | int(rom[b3(a)+1])<<8 }

type instrument struct{ env, duty, x byte }

// chanEvent is one note on a channel timeline.
type chanEvent struct {
	tick, dur int
	freq      int // GB 11-bit frequency (square/wave) or noise poly byte
	tie, rest bool
	inst      instrument
}

type channel struct {
	events    []chanEvent
	loopTicks int
}

// decodeChannel walks a channel's order list and patterns into note events, stopping at the
// $FF loop marker (one full loop). isNoise selects the noise vs pitch interpretation.
func decodeChannel(hdr, durTbl int, isNoise bool) channel {
	// The order list is a run of 2-byte pattern pointers ending in an $FF entry followed by a
	// 2-byte LOOP TARGET (an address back into the list). Patterns before the target are a
	// one-shot intro; the seamless loop is the patterns from the target to the $FF. We decode
	// the WHOLE channel (so the instrument/duration set in the intro carry into the loop) and
	// then keep only the loop-body events, re-based to tick 0.
	end := hdr
	target := hdr
	for guard := 0; guard < 512; guard++ {
		if rb(end) == 0xFF {
			target = rw(end + 2)
			break
		}
		if rb(end) == 0x00 {
			break
		}
		end += 2
	}

	var all []chanEvent
	var inst instrument
	dur, tick, order, prev, loopTick := 1, 0, hdr, 0, 0
	for order < end {
		if order == target {
			loopTick = tick
		}
		pat := rw(order)
		order += 2
		for g2 := 0; g2 < 4096; g2++ {
			c := rb(pat)
			if c == 0x00 {
				break
			}
			switch {
			case c == 0x9D:
				inst = instrument{rb(pat + 1), rb(pat + 2), rb(pat + 3)}
				pat += 4
			case c >= 0xA0 && c <= 0xAF:
				dur = int(rb(durTbl + int(c&0x0F)))
				pat++
			default:
				ev := chanEvent{tick: tick, dur: dur, inst: inst}
				switch {
				case c == 0x01: // repeat previous note (retrigger same pitch)
					ev.freq = prev
				case isNoise:
					ev.freq = int(c)
					prev = ev.freq
				default:
					ev.freq = rw(0x6F70 + int(c))
					prev = ev.freq
				}
				all = append(all, ev)
				tick += dur
				pat++
			}
		}
	}
	// keep the loop body, re-based to tick 0 (instrument state carried from the intro).
	var ch channel
	for _, e := range all {
		if e.tick >= loopTick {
			e.tick -= loopTick
			ch.events = append(ch.events, e)
		}
	}
	ch.loopTicks = tick - loopTick
	return ch
}

// song decodes a music id into its four channels. A channel whose header pointer is not in
// bank 3 ($6000-$7FFF) is unused (e.g. the bonus theme has no wave channel) and stays empty.
func song(id int) [4]channel {
	hdr := rw(0x673C + (id-1)*2)
	durTbl := rw(hdr + 1)
	var chs [4]channel
	for i := 0; i < 4; i++ {
		if p := rw(hdr + 3 + i*2); p >= 0x6000 && p < 0x8000 {
			chs[i] = decodeChannel(p, durTbl, i == 3)
		}
	}
	return chs
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// regBase is the first APU register of each channel (NR_1).
var regBase = [4]uint16{0xFF11, 0xFF16, 0xFF1B, 0xFF20}

// render turns the four decoded channels into an APU register-write stream and PCM, trimmed
// to one loop (the longest channel's loop length).
func render(chs [4]channel) []int16 {
	// The channels loop at different lengths (e.g. a short noise drum under a long melody);
	// the song repeats seamlessly at the least common multiple, where they all realign. If a
	// secondary channel makes that unreasonably long, fall back to the square channels (the
	// melody/harmony that define the song) and let the others tile with a tiny seam.
	lcm := func(chans []channel) int {
		l := 1
		for _, c := range chans {
			if c.loopTicks > 0 {
				l = l / gcd(l, c.loopTicks) * c.loopTicks
			}
		}
		return l
	}
	loop := lcm(chs[:])
	if loop > 5000 {
		loop = lcm(chs[0:2]) // squares only
	}
	var ev []gameboy.RegWrite
	at := func(tick int, reg uint16, v byte) {
		ev = append(ev, gameboy.RegWrite{Cycle: int64(tick) * cycPerTick, Reg: reg, Val: v})
	}
	// power on, full panning, master volume.
	at(0, 0xFF26, 0x80)
	at(0, 0xFF25, 0xFF)
	at(0, 0xFF24, 0x77)
	at(0, 0xFF1A, 0x80) // wave DAC on

	for ci, c := range chs {
		if c.loopTicks == 0 {
			continue
		}
		base := regBase[ci]
		// repeat the channel's own loop to fill the song loop
		for rep := 0; rep*c.loopTicks < loop; rep++ {
			off := rep * c.loopTicks
			for _, e := range c.events {
				tk := e.tick + off
				if tk >= loop {
					break
				}
				e := e
				e.tick = tk
				emitNote(&ev, at, ci, base, e)
			}
		}
	}
	apu := gameboy.NewAPU()
	return normalize(apu.Render(ev, int64(loop)*cycPerTick))
}

// emitNote appends the APU register writes for one note event on channel ci.
func emitNote(ev *[]gameboy.RegWrite, at func(int, uint16, byte), ci int, base uint16, e chanEvent) {
	switch ci {
	case 0, 1: // square
		at(e.tick, base, e.inst.duty)  // NRx1 duty/length
		at(e.tick, base+1, e.inst.env) // NRx2 envelope
		at(e.tick, base+2, byte(e.freq))
		at(e.tick, base+3, byte(e.freq>>8)|0x80) // trigger
	case 2: // wave
		at(e.tick, 0xFF1C, 0x20) // volume 100%
		at(e.tick, 0xFF1D, byte(e.freq))
		at(e.tick, 0xFF1E, byte(e.freq>>8)|0x80)
	case 3: // noise
		at(e.tick, 0xFF21, e.inst.env)
		at(e.tick, 0xFF22, byte(e.freq))
		at(e.tick, 0xFF23, 0x80)
	}
}

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

// tracks: the level music (named by which levels use it via $07B7) plus the bonus theme.
var tracks = []struct {
	id   int
	name string
}{
	{0x07, "level-1-1"}, // 1-1, 1-2, 3-1
	{0x03, "level-1-3"}, // 1-3, 3-2, 3-3
	{0x08, "level-2-1"}, // 2-1, 2-2 (Muda)
	{0x06, "level-4-1"}, // 4-1, 4-2 (Chai)
	{0x05, "level-2-3"}, // 2-3, 4-3 (boss/vehicle stages)
	{0x04, "bonus"},     // pipe bonus rooms
}

func main() {
	romPath := flag.String("rom", "../Super Mario Land (World).gb", "ROM path")
	out := flag.String("o", "../rendered/music", "output dir")
	verify := flag.Int("verify", 0, "print decoded vs engine notes for this music id, then exit")
	flag.Parse()
	var err error
	rom, err = os.ReadFile(*romPath)
	ck(err)
	if *verify != 0 {
		verifyMelody(*verify)
		return
	}
	ck(os.MkdirAll(*out, 0o755))
	for _, t := range tracks {
		pcm := render(song(t.id))
		wav := filepath.Join(*out, t.name+".wav")
		writeWAV(wav, pcm)
		mp3 := filepath.Join(*out, t.name+".mp3")
		c := exec.Command("ffmpeg", "-y", "-loglevel", "error", "-i", wav,
			"-c:a", "libmp3lame", "-b:a", "96k", "-ac", "1", mp3)
		if e := c.Run(); e != nil {
			fmt.Printf("  ffmpeg failed for %s: %v\n", t.name, e)
			continue
		}
		os.Remove(wav)
		fi, _ := os.Stat(mp3)
		fmt.Printf("%-12s id $%02X -> %s (%d KB, loop %.1fs)\n", t.name, t.id, t.name+".mp3",
			fi.Size()/1024, float64(len(pcm))/gameboy.APURate)
	}
}

func writeWAV(path string, pcm []int16) {
	f, err := os.Create(path)
	ck(err)
	defer f.Close()
	dataLen := len(pcm) * 2
	f.Write([]byte("RIFF"))
	binary.Write(f, binary.LittleEndian, uint32(36+dataLen))
	f.Write([]byte("WAVEfmt "))
	binary.Write(f, binary.LittleEndian, uint32(16))
	binary.Write(f, binary.LittleEndian, uint16(1))
	binary.Write(f, binary.LittleEndian, uint16(1))
	binary.Write(f, binary.LittleEndian, uint32(gameboy.APURate))
	binary.Write(f, binary.LittleEndian, uint32(gameboy.APURate*2))
	binary.Write(f, binary.LittleEndian, uint16(2))
	binary.Write(f, binary.LittleEndian, uint16(16))
	f.Write([]byte("data"))
	binary.Write(f, binary.LittleEndian, uint32(dataLen))
	binary.Write(f, binary.LittleEndian, pcm)
}

func ck(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "musicrom:", err)
		os.Exit(1)
	}
}
