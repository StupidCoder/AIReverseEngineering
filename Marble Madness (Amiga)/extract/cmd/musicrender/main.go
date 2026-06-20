// musicrender is a from-scratch Go reimplementation of Marble Madness's music
// player (decoded in Marble_Madness.md Part VI). It parses a course *Snd bank,
// finds the music entry (an op0 directory record), walks the per-channel note
// byte-streams of the h1 arrangement, synthesises each note Amiga-style from the
// shared h4 waveform using the ProTracker period table, mixes the voices and
// writes a WAV. No emulation: the algorithm is reimplemented from the disassembly.
//
// Note-stream format (from $1F162 / $1EE8A):
//   song = 8-byte channel slots -> 6-byte events -> a note byte-stream.
//   stream byte:  bit7 set, nibble!=0 -> set "instrument" (selects note length
//                   via the $1FA26 duration table, index = nibble);
//                 $80 (bit7, nibble 0) -> advance to the next event;
//                 $7F -> rest / note-off;
//                 $00..$6B -> a note (octave = n/12, semitone = n%12).
//   a note/rest advances the channel timer by $1FA26[instr&$F]; the per-frame
//   delta is $1FA68; a frame fires when the timer reaches <= 0.
//   pitch: period = protracker[semitone]; the octave selects the looped h4 slice
//   length (length = base << (8-octave)), classic one-waveform-many-octaves.
//
// Usage: musicrender disk.adf prcsnd [-id N] [-out song.wav] [-secs S]
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"os"

	"stupidcoder.com/tools/amiga/adf"
	"stupidcoder.com/tools/amiga/hunk"
)

const base = 0x800000

// ProTracker period table for one octave (C..B), the $1F902 table in the engine.
var period = [12]float64{428, 404, 381, 360, 339, 320, 302, 285, 269, 254, 240, 226}

// $1FA26 per-"instrument" note-length table (delta units), and the runtime delta
// $1FA68 (set by the seq-start from tempo 99). Both extracted from the engine.
var durTable = [16]uint32{65536, 6291456, 3145728, 1572864, 786432, 393216, 196608, 131072,
	4718592, 2359296, 1179648, 589824, 2097152, 1048576, 524288, 262144}

const paulaClock = 3546895.0 // PAL

type snd struct {
	img  []byte
	segs []hunk.Segment
}

func (s *snd) r16(a uint32) uint16 {
	if a < base || int(a-base)+2 > len(s.img) {
		return 0
	}
	return binary.BigEndian.Uint16(s.img[a-base:])
}
func (s *snd) r32(a uint32) uint32 {
	if a < base || int(a-base)+4 > len(s.img) {
		return 0
	}
	return binary.BigEndian.Uint32(s.img[a-base:])
}
func (s *snd) bytesAt(a uint32, n int) []byte {
	o := int(a - base)
	if a < base || o+n > len(s.img) {
		return nil
	}
	return s.img[o : o+n]
}

// envSeg is one [rate, target] segment of a volume envelope (16.16 fixed).
type envSeg struct{ rate, target int32 }

// voice is one playing note.
type voice struct {
	active bool
	sample []int8  // the looped waveform slice
	pos    float64 // fractional read position
	step   float64 // samples advanced per output sample
	vol    float64 // base level 0..1
	level  float64 // current envelope level 0..1 (updated per frame)

	// volume envelope (the engine's $21954 segment ramp): value is 16.16, 0..$10000.
	env    []envSeg
	envVal int64
	envIdx int
}

// envStep advances the envelope one sequencer frame and returns the level 0..1.
func (v *voice) envStep() float64 {
	if v.envIdx < len(v.env) {
		s := v.env[v.envIdx]
		if s.rate != 0 {
			v.envVal += int64(s.rate)
			if (s.rate > 0 && v.envVal >= int64(s.target)) || (s.rate < 0 && v.envVal <= int64(s.target)) {
				v.envVal = int64(s.target)
				v.envIdx++ // advance to the next segment (rate 0 = sustain/hold)
			}
		}
	}
	if v.envVal < 0 {
		v.envVal = 0
	}
	return float64(v.envVal) / 65536.0
}

// chanState is the per-channel sequencer cursor.
type chanState struct {
	events  uint32 // ptr to 6-byte events
	evIdx   int    // current event index
	sub     int    // byte offset within the current note-stream
	instr   int    // current "instrument" (duration index)
	timer   int64  // counts down by delta each frame
	sampleBase uint32
	v       voice
	live    bool
}

func main() {
	id := flag.Int("id", -1, "music soundID (op0 record); -1 = first op0 found")
	out := flag.String("out", "music.wav", "output WAV path")
	secs := flag.Float64("secs", 30, "seconds to render")
	rate := flag.Int("rate", 44100, "output sample rate")
	fps := flag.Float64("fps", 50, "music sequencer tick rate (Hz)")
	sbase := flag.Int("base", 1, "synth base length (words) for octave 8")
	flag.Parse()
	outRate = *rate
	synthBase = *sbase
	if flag.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "usage: musicrender disk.adf prcsnd [-id N] [-out f.wav]")
		os.Exit(2)
	}
	adfData, err := os.ReadFile(flag.Arg(0))
	must(err)
	vol, err := adf.Open(adfData)
	must(err)
	data, err := vol.ReadFile(flag.Arg(1))
	must(err)
	prog, err := hunk.Load(data, base)
	must(err)
	s := &snd{img: prog.Image, segs: prog.Segments}

	var dir uint32
	for _, sg := range prog.Segments {
		if sg.Kind == "DATA" && sg.Size > 0 {
			dir = sg.Base
			break
		}
	}
	cnt := int(s.r16(dir))
	// pick the music record (op0)
	pick := *id
	if pick < 0 {
		for i := 0; i < cnt; i++ {
			rec := dir + 2 + uint32(i)*8
			if s.r16(rec) == 0 && s.r32(rec+4) != 0 {
				pick = i
				break
			}
		}
	}
	if pick < 0 {
		fmt.Fprintln(os.Stderr, "no music (op0) record found")
		os.Exit(1)
	}
	desc := s.r32(dir + 2 + uint32(pick)*8 + 4)
	song := s.r32(desc)     // arrangement
	sub := s.r32(desc + 4)  // instrument bank
	sampBase := s.r32(sub + 4) // h4 waveform base
	volEnv = parseEnv(s, s.r32(sub+8)) // the per-note volume envelope ($1FE82 / $21954 segments)
	fmt.Printf("music id%d desc=$%X song=$%X sampleBase=$%X volEnv=%d segs\n", pick, desc, song, sampBase, len(volEnv))

	// set up channels
	var chans []*chanState
	for ch := uint32(0); ch < 8; ch++ {
		evs := s.r32(song + ch*8)
		if evs == 0 {
			continue
		}
		chans = append(chans, &chanState{events: evs, sampleBase: sampBase, live: true})
	}
	fmt.Printf("%d active channels\n", len(chans))

	delta := int64(59419) // $1FA68 runtime value (tempo 99)

	// render
	total := int(*secs * float64(*rate))
	samplesPerFrame := float64(*rate) / *fps
	buf := make([]float64, total)
	nextFrame := 0.0
	for i := 0; i < total; i++ {
		if float64(i) >= nextFrame {
			for _, c := range chans {
				tickChannel(s, c, delta)   // advance the sequencer (may trigger a new note)
				c.v.level = c.v.envStep()  // advance the volume envelope one frame
			}
			nextFrame += samplesPerFrame
		}
		var mix float64
		for _, c := range chans {
			if c.v.active {
				mix += c.v.sampleAt() * c.v.vol * c.v.level
			}
		}
		buf[i] = mix
	}
	// normalise so stacked attacks never clip
	peak := 0.0
	for _, x := range buf {
		if a := math.Abs(x); a > peak {
			peak = a
		}
	}
	if peak > 0 {
		g := 0.92 / peak
		for i := range buf {
			buf[i] *= g
		}
	}
	writeWAV(*out, buf, *rate)
	fmt.Printf("wrote %s (%.1fs @ %dHz)\n", *out, *secs, *rate)
}

// tickChannel advances one channel's sequencer by one frame.
func tickChannel(s *snd, c *chanState, delta int64) {
	c.timer -= delta
	if c.timer > 0 {
		return
	}
	// read note-stream bytes until a note or rest is emitted (or stream ends)
	for guard := 0; guard < 64; guard++ {
		ev := c.events + uint32(c.evIdx)*6
		ns := s.r32(ev + 2)
		if ns == 0 {
			c.live = false
			c.v.active = false
			return
		}
		b := s.img[ns-base+uint32(c.sub)]
		c.sub++
		if b&0x80 != 0 {
			if b&0x0F == 0 { // $80: advance to next event
				c.evIdx++
				c.sub = 0
				// loop the song if the next event is empty/terminator
				if s.r32(c.events+uint32(c.evIdx)*6+2) == 0 {
					c.evIdx = 0
					c.sub = 0
				}
				continue
			}
			c.instr = int(b & 0x0F) // set instrument (duration index)
			continue
		}
		// bit7 clear -> note or rest
		c.timer += int64(durTable[c.instr&0xF])
		if b == 0x7F { // rest / note-off
			c.v.active = false
			return
		}
		triggerNote(s, c, int(b))
		return
	}
}

// triggerNote synthesises a note from the h4 waveform.
func triggerNote(s *snd, c *chanState, note int) {
	octave := clamp(note/12, 0, 8)
	semi := note % 12
	// $1EE8A: ioa_Length (words) = base << (8-octave); the looped slice is at
	// sampleBase + lengthWords*2, lengthWords*2 bytes long. One waveform, the
	// octave picks the slice length (and thus the fundamental).
	lenWords := synthBase << (8 - octave)
	lenBytes := lenWords * 2
	if lenBytes < 2 {
		lenBytes = 2
	}
	off := lenBytes
	wav := s.bytesAt(c.sampleBase+uint32(off), lenBytes)
	if wav == nil { // fall back to the start of the waveform
		wav = s.bytesAt(c.sampleBase, lenBytes)
	}
	if wav == nil {
		return
	}
	smp := make([]int8, len(wav))
	for i, b := range wav {
		smp[i] = int8(b)
	}
	per := period[semi]
	srcRate := paulaClock / per // Paula playback rate (samples/sec) at this period
	c.v = voice{active: true, sample: smp, step: srcRate / float64(outRate), vol: 0.45, env: volEnv}
	if dump {
		pitch := srcRate / float64(len(smp))
		fmt.Fprintf(os.Stderr, "note ch=%p n=%3d oct=%d semi=%2d lenB=%d per=%.0f pitch=%.1fHz\n", c, note, octave, semi, len(smp), per, pitch)
	}
}

var dump = os.Getenv("DUMP") != ""
var synthBase = 1
var outRate = 44100
var volEnv []envSeg

// parseEnv reads the engine's volume-envelope segments ([rate:long][target:long]
// pairs, 16.16 fixed) starting at addr, stopping after a rate==0 (sustain) segment.
func parseEnv(s *snd, addr uint32) []envSeg {
	if addr == 0 {
		return nil
	}
	var segs []envSeg
	for i := 0; i < 16; i++ {
		rate := int32(s.r32(addr + uint32(i*8)))
		target := int32(s.r32(addr + uint32(i*8) + 4))
		segs = append(segs, envSeg{rate, target})
		if rate == 0 {
			break
		}
	}
	return segs
}

func (v *voice) sampleAt() float64 {
	if len(v.sample) == 0 {
		return 0
	}
	idx := int(v.pos) % len(v.sample)
	val := float64(v.sample[idx]) / 128.0
	v.pos += v.step
	return val
}

func clamp(x, lo, hi int) int {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func writeWAV(path string, buf []float64, rate int) {
	f, err := os.Create(path)
	must(err)
	defer f.Close()
	n := len(buf)
	dataLen := n * 2
	w := func(v ...interface{}) {
		for _, x := range v {
			binary.Write(f, binary.LittleEndian, x)
		}
	}
	f.WriteString("RIFF")
	w(uint32(36 + dataLen))
	f.WriteString("WAVE")
	f.WriteString("fmt ")
	w(uint32(16), uint16(1), uint16(1), uint32(rate), uint32(rate*2), uint16(2), uint16(16))
	f.WriteString("data")
	w(uint32(dataLen))
	for _, x := range buf {
		if x > 1 {
			x = 1
		}
		if x < -1 {
			x = -1
		}
		w(int16(math.Round(x * 32767)))
	}
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "musicrender:", err)
		os.Exit(1)
	}
}
