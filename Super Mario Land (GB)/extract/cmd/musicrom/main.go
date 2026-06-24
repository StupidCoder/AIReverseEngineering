// musicrom synthesises Super Mario Land's music ENTIRELY FROM THE ROM song data — a Go port
// of the bank-3 sound engine (Super_Mario_Land.md Part VI), no emulator in the render loop.
//
// Format (all bank-3 addresses):
//   - The song table $673C maps a music id to a song header; $07B7[ffe4] selects which song
//     each of the twelve levels uses, so the tracks are named by that.
//   - A song header is: master byte, a 16-bit pointer to the song's DURATION TABLE, then four
//     16-bit channel-header pointers (square1, square2, wave, noise).
//   - A channel header is an ORDER LIST of 16-bit pattern pointers ending in an $FF entry
//     followed by a 2-byte LOOP TARGET; patterns before the target are a one-shot intro.
//   - A pattern is a byte stream: $9D a b c = set voice (squares: NRx2 env, NRx1 duty, a 3rd
//     byte; WAVE: a|b<<8 is a pointer to 16 bytes of wave RAM, c is NR32 vol); $A0-$AF = set
//     note duration to durtable[low nibble]; $00 ends the pattern; note $01 = NOTE-OFF/rest
//     (the engine retriggers with a DAC-off envelope to articulate the part); any other byte N
//     is a note — pitch = freqtable[$6F70 + N] (two bytes/semitone), except the noise channel,
//     whose N indexes the envelope/poly table at $7002.
//   - The engine ticks at 64 Hz (one durtable unit = 1/64 s).
//
// The decoded notes drive our DMG APU (tools/gameboy/apu.go). Each track is rendered as its
// intro + two loop iterations with a fade-out tail. ffmpeg (libmp3lame) encodes the MP3.
// -verify boots the real engine and prints its note/envelope stream beside the port's, so the
// port can be checked channel-by-channel against ground truth.
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

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
	events    []chanEvent // the whole channel: intro then one loop, ticks from 0
	loopStart int         // tick where the looping body begins (intro plays once)
	loopLen   int         // length of the loop body in ticks
}

// decodeChannel walks a channel's order list and patterns into note events, stopping at the
// $FF loop marker (one full loop). isNoise selects the noise vs pitch interpretation.
func decodeChannel(hdr, durTbl int, isNoise bool) channel {
	// The order list is a run of 2-byte pattern pointers ending in an $FF entry followed by a
	// 2-byte LOOP TARGET (an address back into the list): the patterns before the target are a
	// one-shot intro, the rest is the looping body. We decode the whole channel and record
	// where the loop body starts so the renderer can play "intro + N loops".
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
				case c == 0x01: // note-off: the engine retriggers with a DAC-off envelope
					// ($01) to silence the channel — a rest that articulates the part.
					ev.freq = prev
					ev.rest = true
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
	return channel{events: all, loopStart: loopTick, loopLen: tick - loopTick}
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

// render plays each channel's intro once and then loops its body, for the whole song's
// intro + `loops` loop iterations, and returns peak-normalised PCM with the tail faded out.
func render(chs [4]channel, loops int) []int16 {
	// Song loop length: the LCM of the channels' loop bodies (where they realign), falling
	// back to the square channels if a secondary channel makes it unreasonably long.
	lcm := func(chans []channel) int {
		l := 1
		for _, c := range chans {
			if c.loopLen > 0 {
				l = l / gcd(l, c.loopLen) * c.loopLen
			}
		}
		return l
	}
	songLoop := lcm(chs[:])
	if songLoop > 5000 {
		songLoop = lcm(chs[0:2])
	}
	maxIntro := 0
	for _, c := range chs {
		if c.loopStart > maxIntro {
			maxIntro = c.loopStart
		}
	}
	total := maxIntro + loops*songLoop // ticks to render

	var ev []gameboy.RegWrite
	at := func(tick int, reg uint16, v byte) {
		ev = append(ev, gameboy.RegWrite{Cycle: int64(tick) * cycPerTick, Reg: reg, Val: v})
	}
	at(0, 0xFF26, 0x80) // power on, full panning, master volume, wave DAC on
	at(0, 0xFF25, 0xFF)
	at(0, 0xFF24, 0x77)
	at(0, 0xFF1A, 0x80)

	solo := os.Getenv("MUSSOLO")
	for ci, c := range chs {
		if len(c.events) == 0 || (solo != "" && solo != fmt.Sprint(ci)) {
			continue
		}
		base := regBase[ci]
		// the intro (events before loopStart) plays once; then the loop body repeats.
		for rep := 0; ; rep++ {
			off := rep * c.loopLen
			done := true
			for _, e := range c.events {
				if rep > 0 && e.tick < c.loopStart {
					continue // intro only on the first pass
				}
				tk := e.tick + off
				if tk >= total {
					continue
				}
				done = false
				e := e
				e.tick = tk
				emitNote(&ev, at, ci, base, e)
			}
			if done || c.loopLen == 0 {
				break
			}
		}
	}
	// Events were appended channel-by-channel; the APU consumes them in order, so they must be
	// globally sorted by cycle (a stable sort keeps each tick's writes in their emitted order).
	sort.SliceStable(ev, func(i, j int) bool { return ev[i].Cycle < ev[j].Cycle })

	pcm := normalize(gameboy.NewAPU().Render(ev, int64(total)*cycPerTick))
	return fadeOut(pcm, 2.5)
}

// fadeOut applies a linear fade over the final `secs` seconds.
func fadeOut(pcm []int16, secs float64) []int16 {
	n := int(secs * gameboy.APURate)
	if n > len(pcm) {
		n = len(pcm)
	}
	for i := 0; i < n; i++ {
		g := float64(n-i) / float64(n)
		pcm[len(pcm)-n+i] = int16(float64(pcm[len(pcm)-n+i]) * g)
	}
	return pcm
}

// emitNote appends the APU register writes for one note event on channel ci.
func emitNote(ev *[]gameboy.RegWrite, at func(int, uint16, byte), ci int, base uint16, e chanEvent) {
	switch ci {
	case 0, 1: // square — the $9D bytes are (env, _, duty): NRx2=env, NRx1=duty/length
		envv := e.inst.env
		if e.rest {
			envv = 0x01 // DAC off -> silence (note-off)
		}
		at(e.tick, base, e.inst.x) // NRx1 duty/length (the 3rd $9D byte)
		at(e.tick, base+1, envv)   // NRx2 envelope (the 1st $9D byte)
		at(e.tick, base+2, byte(e.freq))
		at(e.tick, base+3, byte(e.freq>>8)|0x80) // trigger
	case 2: // wave — the instrument's first two bytes are a pointer to 16 bytes of wave RAM
		if e.rest {
			at(e.tick, 0xFF1C, 0x00) // NR32 volume 0 -> mute
			return
		}
		wp := int(e.inst.env) | int(e.inst.duty)<<8
		if wp >= 0x6000 && wp < 0x8000 {
			at(e.tick, 0xFF1A, 0x00) // DAC off while loading wave RAM
			for k := 0; k < 16; k++ {
				at(e.tick, 0xFF30+uint16(k), rb(wp+k))
			}
			at(e.tick, 0xFF1A, 0x80) // DAC on
		}
		at(e.tick, 0xFF1C, e.inst.x) // NR32 volume (3rd $9D byte)
		at(e.tick, 0xFF1D, byte(e.freq))
		at(e.tick, 0xFF1E, byte(e.freq>>8)|0x80)
	case 3: // noise — the note byte indexes a 5-byte entry in the $7002 table:
		// [NR42 env, (unused), NR41 length, NR43 poly, NR44]. NR44 is $C0 = trigger +
		// length-enable, so each drum is a short percussive hit that auto-cuts (this is what
		// the engine does; without the length counter the drums droned and were far too loud).
		p := 0x7002 + e.freq
		if e.rest {
			at(e.tick, 0xFF21, 0x01) // DAC off
			return
		}
		at(e.tick, 0xFF20, rb(p+2)) // NR41 length
		at(e.tick, 0xFF21, rb(p))   // NR42 envelope
		at(e.tick, 0xFF22, rb(p+3)) // NR43 polynomial counter
		at(e.tick, 0xFF23, rb(p+4)) // NR44 trigger + length-enable ($C0)
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
		pcm := render(song(t.id), 2) // intro + 2 loops
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
		fmt.Printf("%-12s id $%02X -> %s (%d KB, length %.1fs)\n", t.name, t.id, t.name+".mp3",
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
