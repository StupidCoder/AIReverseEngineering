package main

import (
	"fmt"
	"os"
)

// driver.go — runs the real sound driver (via the m68k interpreter) and mixes its Paula
// output to PCM. This replaces the hand-written sequencer/macro engine: the driver's own
// code decides everything (sequencing, macros, effects, timing); we only emulate Paula.

// Driver addresses in the decoded $1BB00 overlay (the driver's public API + tick entry).
const (
	drvAPIInit   = 0x1CB62
	drvAPIConfig = 0x1C9D8
	drvAPIPlay   = 0x1CAD8
	drvSoundTick = 0x1BB78
)

type driver struct {
	cpu *m68k
	pos [4]mixChan // mixer playback state per Paula channel
}

type mixChan struct {
	start uint32  // current window start (byte address in cpu.mem)
	leng  int     // current window length in bytes
	pos   float64 // fractional read offset within the window
	play  bool
}

// newDriver loads the overlay driver code (and, for a world module, that module's mdat +
// samples) into the CPU's memory and runs api_init / set_config / play for sub-song n.
func newDriver(overlay []byte, mod tfmxModule, song int) *driver {
	cpu := newM68k()
	copy(cpu.mem[soundBase:], overlay) // driver code + constant tables (period table etc.)
	if mod.addr != mdatAddr {          // a world module: place its mdat + sample bank
		copy(cpu.mem[mod.addr:], mod.mdat)
		copy(cpu.mem[mod.smplAddr:], mod.smpl)
	}
	cpu.a[7] = 0x200000 // a stack in free RAM
	cpu.call(drvAPIInit, map[int]uint32{0: uint32(mod.addr), 1: uint32(mod.smplAddr)})
	cpu.call(drvAPIConfig, map[int]uint32{0: 0x40})
	cpu.call(drvAPIPlay, map[int]uint32{0: uint32(song)})
	return &driver{cpu: cpu}
}

func (d *driver) tickHz() float64 { return d.cpu.tickRate() }

// stepTick advances the music one driver tick (its CIA frame).
func (d *driver) stepTick() { d.cpu.call(drvSoundTick, nil) }

// mixInto renders n stereo samples for the current tick into out (interleaved L,R),
// reading the live Paula channel registers the driver just set.
func (d *driver) mixInto(out []float32, sampleRate int) {
	for ch := 0; ch < 4; ch++ {
		pc := &d.cpu.paula[ch]
		mc := &d.pos[ch]
		if pc.retrig {
			mc.start = pc.playLC
			mc.leng = int(pc.playLEN) * 2
			mc.pos = 0
			mc.play = mc.leng > 0
			pc.retrig = false
		}
	}
	for i := 0; i < len(out)/2; i++ {
		var l, r float64
		for ch := 0; ch < 4; ch++ {
			pc := &d.cpu.paula[ch]
			mc := &d.pos[ch]
			if !mc.play || !pc.dma || mc.leng <= 0 || pc.period < 100 {
				continue
			}
			s := float64(int8(d.cpu.mem[(mc.start+uint32(int(mc.pos)))&0xFFFFFF]))
			s = s / 128.0 * float64(pc.vol) / 64.0
			mc.pos += float64(paulaClock) / float64(pc.period) / float64(sampleRate)
			for mc.leng > 0 && int(mc.pos) >= mc.leng {
				mc.pos -= float64(mc.leng)
				mc.start = pc.lc // Paula reloads from the (current) registers on loop
				mc.leng = int(pc.length) * 2
			}
			if ch == 0 || ch == 3 { // Amiga stereo: 0,3 = left; 1,2 = right
				l += s
			} else {
				r += s
			}
		}
		out[i*2] = float32(clamp(l * 0.5))
		out[i*2+1] = float32(clamp(r * 0.5))
	}
}

// drvTrackPos is the driver's live trackstep (song) position word (track state $1CDF0+$4).
const drvTrackPos = 0x1CDF0 + 0x4

// renderDriver renders sub-song `song` of module `mod` by running the real driver, up to
// maxSeconds. If stopAtLoop, it stops after exactly one full pass — detected when the
// driver's trackstep position returns to where it started (the song loops back).
func renderDriver(overlay []byte, mod tfmxModule, song, sampleRate, maxSeconds int, stopAtLoop bool) ([]float32, float64) {
	d := newDriver(overlay, mod, song)
	hz := d.tickHz()
	if hz < 1 || hz > 200 {
		hz = 49.54
	}
	samplesPerTick := int(float64(sampleRate)/hz + 0.5)
	if samplesPerTick < 1 {
		samplesPerTick = 1
	}
	total := sampleRate * maxSeconds
	minTicks := int(hz * 3) // don't end before ~3s even if the start position recurs
	out := make([]float32, 0, total*2)
	buf := make([]float32, samplesPerTick*2)
	// The trackstep position only ever advances (pattern-level loops don't move it), so
	// the first time it jumps BACKWARD is the song loop point. Stop there — one full pass.
	prevPos, tickN, lastPosDbg := -1, 0, -999
	for len(out)/2 < total {
		d.stepTick()
		tickN++
		pos := int(d.cpu.rd16(drvTrackPos))
		if dbg && pos != lastPosDbg {
			fmt.Fprintf(os.Stderr, "tick %d: trackpos=%d\n", tickN, pos)
			lastPosDbg = pos
		}
		if stopAtLoop && prevPos >= 0 && pos < prevPos && tickN > minTicks {
			break // looped back; don't render the loop's first tick
		}
		prevPos = pos
		d.mixInto(buf, sampleRate)
		out = append(out, buf...)
	}
	return out, hz
}
