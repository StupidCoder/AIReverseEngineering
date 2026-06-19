package main

// player.go — a reimplementation of Turrican's in-game TFMX player (the $1BB00
// sound driver), enough to render a song to PCM. It mirrors the driver's three
// layers: the song/trackstep sequencer, the pattern reader, and the per-voice macro
// VM, all driving a 4-channel Paula mixer. Offsets and opcode semantics were taken
// from the disassembly (disasm/overlay_1bb00.asm); see Turrican.md Part V §7.
//
// mdat layout: pattern ptr table @ +$400, macro ptr table @ +$600, trackstep table
// @ +$800 (8 channel-words/entry; a word's bit15 = off, pattern=(w>>8)&$7F,
// transpose=w&$FF; first word $EFFE = a command step). Samples are raw signed 8-bit.

import (
	"encoding/binary"
	"fmt"
	"os"
)

var dbg = os.Getenv("DBG") != ""

const (
	patTable   = 0x400
	macTable   = 0x600
	trackTable = 0x800
	paulaClock = 3546895 // PAL Paula sample clock (Hz)
	// stdTickHz is the game's default music tick rate. The driver is CIA-timed: the
	// in-game config installs CIA-B timer-B with the standard divisor $40, so the rate
	// is 709379 / (($1C00/$40)<<8) = 709379/$7000 = 24.74 Hz. Songs that carry no in-song
	// $EFFE tempo command (every world module) run at this rate — verified against the
	// real driver in FS-UAE: its tempo fields read $50=$51=$40 while playing world 0.
	// (Defaulting to 50 Hz played those songs ~2x too fast.)
	stdTickHz = 709379.0 / float64((0x1C00/0x40)<<8)
)

// periodTab is the standard Amiga period table the driver uses (overlay $1CF6E):
// 5 octaves, the top two clamped. note index 0..59.
var periodTab = []int{
	1710, 1614, 1524, 1438, 1357, 1281, 1209, 1141, 1077, 1017, 960, 908,
	856, 810, 764, 720, 680, 642, 606, 571, 539, 509, 480, 454,
	428, 404, 381, 360, 340, 320, 303, 286, 270, 254, 240, 227,
	214, 202, 191, 180, 170, 160, 151, 143, 135, 127, 120, 113,
	214, 202, 191, 180, 170, 160, 151, 143, 135, 127, 120, 113,
}

// track is one of the 8 sequencer tracks. The driver's seq_step ($1BBC2) walks all
// 8 each step; each track reads its own pattern and, on a note, routes it to the
// Paula voice named by the note's byte2 (& 3) — tracks and voices are decoupled (8→4).
type track struct {
	patData   int  // mdat offset of the current pattern ($28; 0 = none)
	patPos    int  // entry index into the pattern ($68)
	wait      int  // note-delay counter ($6A)
	transpose int8 // low byte of the trackstep word ($49)
	active    bool // a pattern is assigned and running
	loopCnt   int  // pattern-loop counter ($4A); -1 = none
}

type voice struct {
	// macro VM state
	macro   int  // mdat offset of the current macro (0 = none)
	macPos  int  // command index
	macWait int  // ticks to wait before the next macro command ($42)
	keyOn   bool // voice is sounding ($2)
	note    int  // current note ($11)
	addNote int  // per-channel note offset from $F5/portamento
	detune  int  // macro period detune
	// Paula "registers": the macro sets these; Paula reloads them when the current
	// sample finishes (the loop). cmd $19 points them at a 2-byte silence so a
	// one-shot plays once then loops silence (the note ends naturally).
	regStart  int // smpl offset ($B0 / AUDxLC)
	regLen    int // length in bytes ($D0*2 / AUDxLEN)
	basePer   int // base period ($A0)
	period    int // current period after effects
	vol       int // volume 0..64 ($60)
	dma       bool
	retrigger bool // a fresh DMA start (cmd $01) was requested this tick
	// effects
	vibOn                                  bool
	vibDepth, vibSpeed, vibPos, vibSign    int
	portOn                                 bool
	portTarget, portRate                   int
	envOn                                  bool
	envTarget, envSpeed, envCnt, envReload int
	// macro loop (cmd $05): byte1 = count (0 = infinite), byte2-3 = target macPos.
	// Wrapped around cmd $11 (advance sample) this is the wavetable scan / sustain.
	mLoopOn  bool
	mLoopInf bool
	mLoopCnt int
}

type paulaCh struct {
	data               []int8  // sample bank (signed)
	curStart, curLen   int     // the sample window currently playing
	loopStart, loopLen int     // reloaded into cur when cur finishes (Paula's loop)
	period             int
	vol                int
	pos                float64 // fractional read position within [curStart,curStart+curLen)
	playing            bool
}

type player struct {
	mdat []byte
	smpl []int8
	tr   [8]track
	v    [4]voice
	p    [4]paulaCh
	// song state
	trackPos, trackStart, trackEnd int
	speed, speedCnt                int
	tickHz                         float64
	songOver                       bool
	trackAdvanced                  bool // a track's $F0 advanced the trackstep this row
	speedOverride                  int  // row speed to force (-1 = derive from the song table)
	loopedAt                       int // tick at which the song looped (-1 until then)
	tick                           int
	Trace                          [][8]int // per-tick [per0,vol0,per1,vol1,...] when tracing
	tracing                        bool
}

func be16(b []byte, o int) int { return int(binary.BigEndian.Uint16(b[o:])) }
func be32(b []byte, o int) int { return int(binary.BigEndian.Uint32(b[o:])) }

func newPlayer(mdat, smpl []byte) *player {
	s := make([]int8, len(smpl))
	for i, b := range smpl {
		s[i] = int8(b)
	}
	return &player{mdat: mdat, smpl: s, tickHz: stdTickHz, speedOverride: -1, loopedAt: -1}
}

// start prepares song number n.
func (p *player) start(n int) {
	p.trackStart = be16(p.mdat, 0x100+n*2)
	p.trackEnd = be16(p.mdat, 0x140+n*2)
	// The song table's "tempo" word is the driver's $6 row-speed register, but the
	// api_play setup ($1C8FA) reinterprets large values: tempo > $1F is a CIA-timer
	// divisor and the real row speed is forced to 5 (the actual tick rate then comes
	// from the in-song $EFFE/sub4 command); $10..$1F is an intro mode (row speed = t-$10);
	// <= $F is the row speed directly. Without this, songs 1/2 (tempo $78/$A0) crawl.
	t := be16(p.mdat, 0x180+n*2)
	switch {
	case t > 0x1F:
		p.speed = 5
	case t > 0xF:
		p.speed = t - 0x10
	default:
		p.speed = t
	}
	// The in-game (world) modules play through the resident driver; with this synth's
	// macro-envelope timing their note durations land right at a row speed of 1 (rows
	// every 2 ticks) — verified by ear against world 0's theme. renderAll/-mod set this.
	if p.speedOverride >= 0 {
		p.speed = p.speedOverride
	}
	if s := os.Getenv("TSPEED"); s != "" { // test override for row speed
		fmt.Sscanf(s, "%d", &p.speed)
	}
	p.trackPos = p.trackStart
	p.speedCnt = 0
	for i := range p.tr {
		p.tr[i] = track{loopCnt: -1}
	}
	for i := range p.v {
		p.v[i] = voice{}
	}
	p.processTrack()
}

// processTrack reads the current trackstep, following command steps until it lands on
// a normal one (which assigns patterns to the 8 tracks). A trackstep word with bit15
// set leaves that track running its previous pattern (no reset).
func (p *player) processTrack() {
	for guard := 0; guard < 256; guard++ {
		o := trackTable + p.trackPos*16
		if o+16 > len(p.mdat) {
			p.songOver = true
			return
		}
		if be16(p.mdat, o) == 0xEFFE {
			if p.trackCommand(be16(p.mdat, o+2), o) {
				return // command set up a jump/stop; don't auto-advance
			}
			p.advanceTrack()
			continue
		}
		for t := 0; t < 8; t++ {
			w := be16(p.mdat, o+t*2)
			if w&0x8000 != 0 {
				continue // keep this track's previous pattern running
			}
			pn := (w >> 8) & 0x7F
			p.tr[t].patData = be32(p.mdat, patTable+pn*4)
			p.tr[t].patPos = 0
			p.tr[t].wait = 0
			p.tr[t].active = p.tr[t].patData != 0
			p.tr[t].transpose = int8(w & 0xFF)
			p.tr[t].loopCnt = -1
			if dbg {
				fmt.Fprintf(os.Stderr, "trackstep $%X: trk%d = pat $%X (data $%X) transpose %d\n", p.trackPos, t, pn, p.tr[t].patData, int8(w&0xFF))
			}
		}
		return
	}
}

// trackCommand handles an $EFFE command step. Returns true if it set song position
// itself (caller must not advance).
func (p *player) trackCommand(sub, o int) bool {
	switch sub {
	case 4: // word2 = row-speed (kept = song tempo, verified vs the real driver's work
		// struct $6=3); word3 sets the CIA timer B that drives sound_tick — that's the
		// real tick rate (the music is CIA-timed, not vblank-timed).
		if t := be16(p.mdat, o+6) & 0x1FF; t != 0 {
			reload := (0x1C00 / t) << 8 // CIA timer B reload value
			if reload > 0 {
				hz := 709379.0 / float64(reload) // PAL E-clock / reload
				if hz >= 12 && hz <= 120 {       // ignore implausible values
					p.tickHz = hz
				}
			}
		}
		return false
	case 0: // loop song to the position in word2
		p.trackPos = be16(p.mdat, o+2)
		if p.loopedAt < 0 {
			p.loopedAt = p.tick
		}
		p.processTrack()
		return true
	}
	return false
}

func (p *player) advanceTrack() {
	if p.trackPos >= p.trackEnd {
		p.trackPos = p.trackStart
		if p.loopedAt < 0 {
			p.loopedAt = p.tick
		}
	} else {
		p.trackPos++
	}
}

// stepTick advances the player one driver tick: the pattern sequencer (every
// speed+1 ticks) and then the per-voice macro VM.
func (p *player) stepTick() {
	if p.speedCnt == 0 {
		p.speedCnt = p.speed
		p.seqStep()
	} else {
		p.speedCnt--
	}
	for ch := 0; ch < 4; ch++ {
		p.runMacro(ch)
		p.applyEffects(ch)
		p.toPaula(ch)
	}
	if p.tracing {
		var row [8]int
		for ch := 0; ch < 4; ch++ {
			if p.p[ch].playing {
				row[ch*2] = p.p[ch].period
				row[ch*2+1] = p.p[ch].vol
			}
		}
		p.Trace = append(p.Trace, row)
		if dbg {
			fmt.Fprintf(os.Stderr, "t%d:", p.tick)
			for c := 0; c < 4; c++ {
				v := &p.v[c]
				fmt.Fprintf(os.Stderr, " ch%d[off$%X per%d vol%d]", c, v.regStart, v.period, v.vol)
			}
			fmt.Fprintln(os.Stderr)
		}
	}
	p.tick++
}

// seqStep runs the sequencer for one row: it walks all 8 tracks (seq_step $1BBC2),
// each reading its own pattern. A track's $F0 end advances the global trackstep, which
// reloads every track's pattern — so restart the walk when that happens (mirrors the
// driver's $A-flag resync).
func (p *player) seqStep() {
	for restart := 0; restart < 8; restart++ {
		p.trackAdvanced = false
		for t := 0; t < 8; t++ {
			p.readTrack(t)
			if p.trackAdvanced {
				break
			}
		}
		if !p.trackAdvanced {
			return
		}
	}
}

// readTrack runs one sequencer step for track t (driver process_track $1BC54). Unlike a
// naive one-entry-per-row reader, the driver reads *multiple* pattern entries per step
// and only stops on certain ones — so a musical "row" can key several voices at once and
// carry its own duration. Each note routes to the Paula voice named by its byte2 (& 3).
//
// Per entry (with tb = (byte0 + transpose) & $FF, the driver's post-transpose decision
// byte): a $F0-$FF command dispatches (see below); otherwise it's a note — key the voice,
// then CONTINUE reading if tb < $7F or tb >= $C0, else STOP this step (tb in [$7F,$C0) is
// a row terminator). A note with byte0 in [$7F,$C0) also sets the track's note-duration
// (wait) from byte3; $F3 sets it from byte1. tb >= $BF means a hold/portamento note
// (change pitch, no macro/sample retrigger); tb < $BF is a fresh note (retrigger).
func (p *player) readTrack(t int) {
	tr := &p.tr[t]
	if tr.wait > 0 { // $6A: still holding the previous step's notes
		tr.wait--
		return
	}
	for guard := 0; tr.active && guard < 512; guard++ {
		o := tr.patData + tr.patPos*4
		if o+4 > len(p.mdat) {
			tr.active = false
			return
		}
		b0, b1, b2, b3 := p.mdat[o], p.mdat[o+1], p.mdat[o+2], p.mdat[o+3]

		if b0 >= 0xF0 {
			switch b0 {
			case 0xF0: // end -> advance the trackstep (reloads every track), restart walk
				p.advanceTrack()
				p.processTrack()
				p.trackAdvanced = true
				return
			case 0xF1: // pattern loop: count b1, target (b2<<8|b3); continue reading
				if tr.loopCnt < 0 {
					tr.loopCnt = int(b1)
				}
				if tr.loopCnt == 0 {
					tr.loopCnt = -1
					tr.patPos++
				} else {
					if b1 != 0xFF {
						tr.loopCnt--
					}
					tr.patPos = int(b2)<<8 | int(b3)
				}
				continue
			case 0xF2: // jump to pattern b1 at (b2<<8|b3); continue reading
				tr.patData = be32(p.mdat, patTable+(int(b1)&0x7F)*4)
				tr.patPos = int(b2)<<8 | int(b3)
				tr.active = tr.patData != 0
				continue
			case 0xF3: // set the step's note duration (track wait) from b1, then stop
				tr.wait = int(b1)
				tr.patPos++
				return
			case 0xF4: // disable/stop the track
				tr.patPos++
				return
			default: // F5/F6/F7 (and others): per-voice effect-note — continue (effects
				// are approximated at the macro level for now)
				tr.patPos++
				continue
			}
		}

		// a note
		ch := int(b2) & 3
		tb := (int(b0) + int(tr.transpose)) & 0xFF
		note := int(b0&0x3F) + int(tr.transpose)
		if tb >= 0xBF { // hold / portamento: pitch only, no retrigger
			v := &p.v[ch]
			v.note = note
			v.basePer = p.noteToPeriod(note)
			if !v.portOn {
				v.period = v.basePer
			}
		} else {
			p.trigger(ch, note, int(b1), 0, false)
		}
		tr.patPos++
		if b0 >= 0x7F && b0 < 0xC0 { // this note carries the step's duration in byte3
			tr.wait = int(b3)
		}
		if tb >= 0x7F && tb < 0xC0 { // row terminator
			return
		}
		// otherwise keep reading more entries this same step
	}
}

// trigger starts a note on voice ch: install the macro and key on.
func (p *player) trigger(ch, note, macroNum, detune int, porta bool) {
	v := &p.v[ch]
	v.note = note
	// The pattern's byte3 goes to the driver's $23 field (not the period finetune
	// $22, which a macro sets), so it must NOT detune the note. Adding it shifted
	// every note's pitch by up to ±127.
	v.detune = 0
	_ = detune
	v.macro = be32(p.mdat, macTable+(macroNum&0x7F)*4)
	v.macPos = 0
	v.macWait = 0
	v.keyOn = true
	v.addNote = 0
	v.mLoopOn = false
	if porta {
		// keep current period as portamento source (approximate)
		v.portOn = true
		v.portTarget = p.noteToPeriod(note) + detune
	} else {
		v.vibOn, v.portOn, v.envOn = false, false, false
	}
}

func (p *player) noteToPeriod(note int) int {
	if note < 0 {
		note = 0
	}
	if note >= len(periodTab) {
		note = len(periodTab) - 1
	}
	return periodTab[note]
}

// runMacro steps voice ch's instrument macro until it waits or stops.
func (p *player) runMacro(ch int) {
	v := &p.v[ch]
	if v.macWait > 0 {
		v.macWait--
		return
	}
	if v.macro == 0 || !v.keyOn {
		return
	}
	for guard := 0; guard < 256; guard++ {
		o := v.macro + v.macPos*4
		if o+4 > len(p.mdat) {
			v.keyOn = false
			return
		}
		cmd := p.mdat[o]
		b1 := p.mdat[o+1]
		w := int(p.mdat[o+2])<<8 | int(p.mdat[o+3]) // byte2-3 word
		off := int(p.mdat[o])<<24 | int(b1)<<16 | w // full 24-bit operand (cmd byte ignored)
		off &= 0x00FFFFFF
		v.macPos++
		stop := false
		switch cmd {
		case 0x00: // DMA off + reset effects
			v.dma = false
			v.vibOn, v.envOn = false, false
		case 0x01: // DMA on — (re)start the current registers as the playing sample
			v.dma = true
			v.retrigger = true
		case 0x02: // set sample start = smpl + offset24
			v.regStart = off
		case 0x03: // set sample length (words)
			v.regLen = w * 2
		case 0x04: // wait w ticks
			v.macWait = w
			stop = true
		case 0x05: // macro loop: byte1 = count (0 = infinite), byte2-3 = target macPos.
			// Loops the body (which advances the sample via $11) for the scan/sustain.
			if !v.mLoopOn {
				v.mLoopOn = true
				v.mLoopInf = b1 == 0
				v.mLoopCnt = int(b1)
			}
			if v.mLoopInf || v.mLoopCnt > 0 {
				if !v.mLoopInf {
					v.mLoopCnt--
				}
				v.macPos = w
			} else {
				v.mLoopOn = false
			}
		case 0x13: // DMA off (the macro turns the channel off, e.g. to restart it)
			v.dma = false
		case 0x1A: // key off + 1-tick wait
			stop = true
		case 0x1C: // note-split: if byte1 >= note keep going, else jump to macPos byte2-3
			// (selects a low/high instrument macro; driver $1C310)
			if int(b1) < v.note {
				v.macPos = w
			}
		case 0x06: // set new macro (instrument)
			v.macro = be32(p.mdat, macTable+(int(b1)&0x7F)*4)
			v.macPos = 0
		case 0x07: // stop the MACRO (the voice keeps playing its current loop)
			v.macro = 0
			stop = true
		case 0x19: // loop point -> a 2-byte silence at smpl[0]: a one-shot plays once
			v.regStart = 0
			v.regLen = 2
		case 0x08, 0x09: // play note; advance 1 tick. $08 = byte1 is a SIGNED relative
			// transpose added to the pattern note; $09 = byte1 is an absolute note.
			base := int(b1)
			if cmd == 0x08 {
				base = int(int8(b1)) + v.note
			}
			v.basePer = p.noteToPeriod(base+v.addNote) + v.detune
			if !v.portOn {
				v.period = v.basePer
			}
			stop = true
		case 0x0A: // set period directly
			v.basePer = w
			if !v.portOn {
				v.period = w
			}
		case 0x0B: // portamento on (rate w)
			v.portOn = true
			v.portRate = w
			v.portTarget = v.basePer
		case 0x0C: // vibrato on (speed b1, depth b3)
			v.vibOn = true
			v.vibSpeed = int(b1)
			v.vibDepth = int(p.mdat[o+3])
			v.vibPos = 0
			v.vibSign = 1
		case 0x0D, 0x0E: // set volume = b3
			v.vol = int(p.mdat[o+3]) & 0x7F
			if v.vol > 64 {
				v.vol = 64
			}
		case 0x0F: // volume envelope: speed b1, target b3
			v.envOn = true
			v.envSpeed = int(b1) & 0x7F
			v.envReload = int(b1) & 0x7F
			v.envCnt = 0
			v.envTarget = int(p.mdat[o+3]) & 0x7F
		case 0x10: // reset effects
			v.vibOn, v.envOn, v.portOn = false, false, false
		case 0x11: // add a SIGNED delta to the sample start (drives the wavetable scan)
			v.regStart += int(int16(w))
		case 0x12: // add a signed delta to the sample length
			v.regLen += int(int16(w)) * 2
		case 0x18: // set loop point: advance the sample start by w bytes and shrink the
			// length by w — the loop reg becomes the sustain portion (the note holds)
			v.regStart += w
			v.regLen -= w
			if v.regLen < 2 {
				v.regLen = 2
			}
		case 0x14: // wait for DMA / sustain marker — no-op (keep stepping)
		default:
			// 0x13-0x22 (loop/wave-shaper/etc.) — ignore, keep stepping
		}
		if stop {
			return
		}
	}
}

// applyEffects runs the per-tick vibrato / portamento / envelope.
func (p *player) applyEffects(ch int) {
	v := &p.v[ch]
	per := v.basePer
	if v.portOn {
		if v.period < v.portTarget {
			v.period += v.portRate
			if v.period > v.portTarget {
				v.period = v.portTarget
			}
		} else if v.period > v.portTarget {
			v.period -= v.portRate
			if v.period < v.portTarget {
				v.period = v.portTarget
			}
		}
		per = v.period
	}
	if v.vibOn {
		v.vibPos += v.vibSpeed
		if v.vibPos >= v.vibDepth {
			v.vibPos = v.vibDepth
			v.vibSign = -1
		} else if v.vibPos <= -v.vibDepth {
			v.vibPos = -v.vibDepth
			v.vibSign = 1
		}
		_ = v.vibSign
		per += (v.vibPos >> 4)
	}
	v.period = per
	if v.envOn {
		if v.envCnt > 0 {
			v.envCnt--
		} else {
			v.envCnt = v.envReload
			if v.vol < v.envTarget {
				v.vol++
			} else if v.vol > v.envTarget {
				v.vol--
			} else {
				v.envOn = false
			}
		}
	}
}

// toPaula pushes the voice's registers onto its Paula channel. The voice keeps
// playing across macro-end; sound stops only when the sample becomes the silent
// loop (set by cmd $19) or DMA is left off.
func (p *player) toPaula(ch int) {
	v := &p.v[ch]
	pc := &p.p[ch]
	pc.data = p.smpl
	clamp := func(s, l int) (int, int) {
		if s < 0 {
			s = 0
		}
		if s+l > len(p.smpl) {
			l = len(p.smpl) - s
		}
		if l < 0 {
			l = 0
		}
		return s, l
	}
	pc.loopStart, pc.loopLen = clamp(v.regStart, v.regLen)
	if v.retrigger {
		pc.curStart, pc.curLen = pc.loopStart, pc.loopLen
		pc.pos = 0
		pc.playing = pc.curLen > 0
		v.retrigger = false
	}
	// A transient DMA-off (cmd $13/$00, immediately followed by $01) must NOT silence
	// the channel — real Paula holds the current sample. The note ends only when the
	// macro repoints the loop at the 2-byte silence (cmd $19) or the volume hits 0.
	pc.period = v.period
	if pc.period < 100 {
		pc.period = 100
	}
	pc.vol = v.vol
}

// render produces interleaved stereo float32 PCM at sampleRate, for nSeconds.
func (p *player) render(sampleRate, nSeconds int) []float32 {
	return p.renderN(sampleRate, nSeconds, false)
}

// renderN renders up to maxSeconds; if stopAtLoop, it ends one full pass after the
// song's trackstep loops back (p.loopedAt), with a small minimum so very short loops
// still give something to listen to.
func (p *player) renderN(sampleRate, maxSeconds int, stopAtLoop bool) []float32 {
	total := sampleRate * maxSeconds
	minSamples := sampleRate * 20
	out := make([]float32, total*2)
	samplesPerTick := float64(sampleRate) / p.tickHz
	var acc float64
	n := 0
	for i := 0; i < total; i++ {
		if acc <= 0 {
			p.stepTick()
			acc += float64(sampleRate) / p.tickHz
			_ = samplesPerTick
			if stopAtLoop && p.loopedAt >= 0 && p.tick > p.loopedAt && i >= minSamples {
				break
			}
		}
		acc--
		n = i + 1
		var l, r float64
		for ch := 0; ch < 4; ch++ {
			pc := &p.p[ch]
			if !pc.playing || pc.curLen <= 0 {
				continue
			}
			s := float64(pc.data[pc.curStart+int(pc.pos)]) / 128.0 * float64(pc.vol) / 64.0
			step := float64(paulaClock) / float64(pc.period) / float64(sampleRate)
			pc.pos += step
			// When the current sample finishes, Paula reloads the (loop) registers.
			for pc.curLen > 0 && int(pc.pos) >= pc.curLen {
				pc.pos -= float64(pc.curLen)
				pc.curStart, pc.curLen = pc.loopStart, pc.loopLen
			}
			if ch == 0 || ch == 3 {
				l += s
			} else {
				r += s
			}
		}
		out[i*2] = float32(clamp(l * 0.5))
		out[i*2+1] = float32(clamp(r * 0.5))
	}
	return out[:n*2]
}

func clamp(v float64) float64 {
	if v > 1 {
		return 1
	}
	if v < -1 {
		return -1
	}
	return v
}
