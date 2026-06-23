// musicrender synthesises Elite's music ENTIRELY from the decrypted game image — a
// reimplementation of the $BDDC music sequencer (Elite.md Part VI) driving the shared
// SID emulator (tools/c64/sid). No emulator capture is involved; the only inputs are the
// music command stream at $C034 and the opcode behaviour read from the disassembly.
//
// The engine is a nibble-packed bytecode: a 16-bit play pointer walks the stream, opcodes
// are 4-bit (two per byte, low nibble first), and each opcode sets SID registers (notes,
// ADSR, pulse width, waveform, filter) or steps time. A note step holds for the default
// length ($BDDB) frames; voices 2 and 3 alternate between their pitch and a +$20-detuned
// copy (5- and 6-frame cycles) — the chorused Elite sound — and the gates release two
// frames before each step ends. The player ticks once per PAL video frame (~50.12 Hz),
// matching the raster IRQ that runs it. Rendering stops at the stream's loop opcode (one
// pass). ffmpeg (libmp3lame) encodes the MP3.
package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"

	"stupidcoder.com/tools/c64/sid"
)

const (
	srate    = 44100.0
	cycFrame = 19656.0 // PAL VIC frame = 312 lines * 63 cycles
	frameHz  = sid.PAL / cycFrame
)

type player struct {
	mem  []byte
	ptr  int   // $C2/$C3 play pointer (points at the last byte read; reader pre-increments)
	d1   uint8 // nibble queue
	c6   int   // duration counter
	dur  int   // default note length ($BDDB)
	ctrl [3]uint8
	// voice 2/3 detune: base + (base+$20) copies, and the alternating cycle counters
	v2base, v2det uint16
	v3base, v3det uint16
	c7, c8        int
	c7tog, c8tog  bool
	chip          *sid.SID
	done          bool
	solo          int // -1 = all voices; 0/1/2 = only gate that voice (for diagnostics)
}

func (p *player) fetch() uint8 { p.ptr++; return p.mem[p.ptr] }

// write a SID register (offset from $D400).
func (p *player) w(reg int, v uint8) { p.chip.Write(uint8(reg), v) }

// noteOn writes the freq for a voice and retriggers its gate with the current waveform.
// gates are reg $04/$0B/$12 for voices 1/2/3 (offsets 4, 11, 18).
func (p *player) gate(vi int) {
	if p.solo >= 0 && vi != p.solo {
		return
	}
	reg := []int{4, 11, 18}[vi]
	p.w(reg, 0)          // STY $D40x (Y=0): drop gate -> retrigger edge
	p.w(reg, p.ctrl[vi]) // waveform + gate on
}

// runCommands processes opcodes until a time-step op sets c6 (or the loop op ends it).
func (p *player) runCommands() {
	for {
		if p.d1 == 0 {
			p.d1 = p.fetch()
		}
		op := p.d1 & 0x0F
		p.d1 >>= 4
		switch op {
		case 1, 2, 3: // note on one voice
			vi := int(op - 1)
			hi, lo := p.fetch(), p.fetch()
			p.setFreq(vi, hi, lo)
			p.gate(vi)
		case 4: // notes on voices 1+2
			h1, l1, h2, l2 := p.fetch(), p.fetch(), p.fetch(), p.fetch()
			p.setFreq(0, h1, l1)
			p.setFreq(1, h2, l2)
			p.gate(0)
			p.gate(1)
		case 5: // chord on all three
			d := [6]uint8{p.fetch(), p.fetch(), p.fetch(), p.fetch(), p.fetch(), p.fetch()}
			p.setFreq(0, d[0], d[1])
			p.setFreq(1, d[2], d[3])
			p.setFreq(2, d[4], d[5])
			p.gate(0)
			p.gate(1)
			p.gate(2)
		case 6: // INC $BDD7 (section counter) -- no audio effect
		case 7: // set ADSR for all three voices: AD1 AD2 AD3 SR1 SR2 SR3
			ad := [3]uint8{p.fetch(), p.fetch(), p.fetch()}
			sr := [3]uint8{p.fetch(), p.fetch(), p.fetch()}
			for vi := 0; vi < 3; vi++ {
				p.w(vi*7+5, ad[vi])
				p.w(vi*7+6, sr[vi])
			}
		case 8, 0: // step time: hold for the default length
			p.c6 = p.dur
			return
		case 15: // step time, repacking the nibble queue (the $BE54 transform)
			a := p.d1
			a = ((a << 1) | 1) & 0xFF
			a = (a << 3) & 0xFF
			p.d1 = a
			p.c6 = p.dur
			return
		case 9, 11: // loop back to the start
			p.done = true
			return
		case 10: // pulse widths: PWlo/hi for voices 1,2,3
			for vi := 0; vi < 3; vi++ {
				lo, hi := p.fetch(), p.fetch()
				p.w(vi*7+2, lo)
				p.w(vi*7+3, hi)
			}
		case 12: // default note length (tempo)
			p.dur = int(p.fetch())
		case 13: // the three voices' waveform/control bytes (applied at note-on)
			p.ctrl[0], p.ctrl[1], p.ctrl[2] = p.fetch(), p.fetch(), p.fetch()
		case 14: // filter: $D418 (vol/mode), $D417 (res/route), $D416 (cutoff hi)
			p.w(0x18, p.fetch())
			p.w(0x17, p.fetch())
			p.w(0x16, p.fetch())
		}
	}
}

func (p *player) setFreq(vi int, hi, lo uint8) {
	base := uint16(hi)<<8 | uint16(lo)
	det := base + 0x20
	switch vi {
	case 0: // voice 1: clean, no detune
		p.w(1, hi)
		p.w(0, lo)
	case 1:
		p.v2base, p.v2det = base, det
		p.w(8, hi) // $D408 freq hi
		p.w(7, lo) // $D407 freq lo
	case 2:
		p.v3base, p.v3det = base, det
		p.w(0x0F, hi) // $D40F freq hi
		p.w(0x0E, lo) // $D40E freq lo
	}
}

// perFrame is the $BFEA update: the voice-2/3 detune swap and the gate release at c6==2.
func (p *player) perFrame() {
	p.c8++
	if p.c8 == 6 {
		p.c8 = 0
		p.c8tog = !p.c8tog
		f := p.v3base
		if p.c8tog {
			f = p.v3det
		}
		p.w(0x0F, uint8(f>>8))
		p.w(0x0E, uint8(f))
	}
	p.c7++
	if p.c7 == 5 {
		p.c7 = 0
		p.c7tog = !p.c7tog
		f := p.v2base
		if p.c7tog {
			f = p.v2det
		}
		p.w(8, uint8(f>>8))
		p.w(7, uint8(f))
	}
	if p.c6 == 2 { // two frames before the step ends: release the gates
		for vi, reg := range []int{4, 11, 18} {
			p.w(reg, p.ctrl[vi]&0xFE)
		}
	}
}

// tick runs one video frame: process commands if the step counter expired, then update.
func (p *player) tick() {
	if p.c6 == 0 {
		p.runCommands()
	}
	if p.done {
		return
	}
	p.c6--
	p.perFrame()
}

func writeWAV(path string, pcm []int16) {
	f, _ := os.Create(path)
	defer f.Close()
	dl := len(pcm) * 2
	wr := func(v interface{}) { binary.Write(f, binary.LittleEndian, v) }
	f.Write([]byte("RIFF"))
	wr(uint32(36 + dl))
	f.Write([]byte("WAVEfmt "))
	wr(uint32(16))
	wr(uint16(1))
	wr(uint16(1))
	wr(uint32(srate))
	wr(uint32(srate * 2))
	wr(uint16(2))
	wr(uint16(16))
	f.Write([]byte("data"))
	wr(uint32(dl))
	wr(pcm)
}

func main() {
	in := "extracted/memory_final.bin"
	start := 0xC034
	out := "rendered/elite-music.mp3"
	maxSec := 240.0
	for i := 1; i < len(os.Args)-1; i++ {
		switch os.Args[i] {
		case "-i":
			in = os.Args[i+1]
			i++
		case "-start":
			fmt.Sscanf(os.Args[i+1], "%v", &start)
			i++
		case "-o":
			out = os.Args[i+1]
			i++
		}
	}
	mem, err := os.ReadFile(in)
	if err != nil {
		fmt.Println("read:", err)
		os.Exit(1)
	}
	solo := -1
	for i := 1; i < len(os.Args)-1; i++ {
		if os.Args[i] == "-solo" {
			fmt.Sscanf(os.Args[i+1], "%d", &solo)
		}
	}
	p := &player{
		mem:  mem,
		ptr:  start, // reader pre-increments, so the first byte read is start+1
		dur:  8,
		solo: solo,
		chip: sid.New(sid.PAL, srate),
	}
	debug := false
	for _, a := range os.Args {
		if a == "-debug" {
			debug = true
		}
	}
	samplesPerFrame := srate / frameHz
	var sAcc float64
	var pcm []int16
	maxSamples := int(maxSec * srate)
	frames := 0
	for !p.done && len(pcm) < maxSamples {
		p.tick()
		if p.done {
			break
		}
		frames++
		if debug && frames <= 70 {
			g := func(b bool) byte {
				if b {
					return '#'
				}
				return '.'
			}
			fmt.Printf("f%-3d c6=%2d  v1[%c]env=%3d  v2[%c]env=%3d  v3[%c]env=%3d\n",
				frames, p.c6,
				g(p.chip.Gate(0)), p.chip.Env(0),
				g(p.chip.Gate(1)), p.chip.Env(1),
				g(p.chip.Gate(2)), p.chip.Env(2))
		}
		sAcc += samplesPerFrame
		n := int(sAcc)
		sAcc -= float64(n)
		for i := 0; i < n; i++ {
			pcm = append(pcm, p.chip.Sample())
		}
	}
	secs := float64(len(pcm)) / srate
	fmt.Printf("rendered %d frames -> %d samples (%.1fs), loop=%v\n", frames, len(pcm), secs, p.done)

	// report raw RMS/peak (pre-normalise) for balance diagnostics
	var sumsq float64
	var rawPeak float64
	for _, s := range pcm {
		f := float64(s)
		sumsq += f * f
		if f < 0 {
			f = -f
		}
		if f > rawPeak {
			rawPeak = f
		}
	}
	rms := 0.0
	if len(pcm) > 0 {
		rms = (sumsq / float64(len(pcm)))
		rms = math.Sqrt(rms)
	}
	fmt.Printf("raw RMS=%.1f peak=%.0f (solo=%d)\n", rms, rawPeak, solo)
	if solo >= 0 {
		return // diagnostic run: don't write audio
	}

	// peak-normalise to a comfortable level (the raw mix sits low because of envelopes/rests)
	var peak int16
	for _, s := range pcm {
		if s > peak {
			peak = s
		}
		if -s > peak {
			peak = -s
		}
	}
	if peak > 0 {
		g := 29000.0 / float64(peak)
		for i, s := range pcm {
			pcm[i] = int16(float64(s) * g)
		}
	}

	os.MkdirAll(filepath.Dir(out), 0o755)
	wav := out[:len(out)-len(filepath.Ext(out))] + ".wav"
	writeWAV(wav, pcm)
	c := exec.Command("ffmpeg", "-y", "-loglevel", "error", "-i", wav,
		"-c:a", "libmp3lame", "-b:a", "96k", "-ac", "1", out)
	if err := c.Run(); err != nil {
		fmt.Println("ffmpeg:", err, "(left WAV at", wav+")")
		return
	}
	os.Remove(wav)
	fmt.Println("wrote", out)
}
