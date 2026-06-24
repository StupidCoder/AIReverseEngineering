package gameboy

// APU models the Game Boy (DMG) sound hardware: two square channels (ch1 with a frequency
// sweep), one 32-sample wave channel, and a noise channel, mixed through a master
// volume/panning stage. It is driven by a *timed stream of register writes* captured from
// the running sound engine (registers $FF10-$FF3F), the way an accurate VGM/LSDj render
// works — the engine itself produces the music; we faithfully synthesise the chip it drives.
//
// We synthesise per output sample rather than per CPU cycle: each channel keeps a phase
// accumulator advanced by its real frequency, while the length/envelope/sweep units are
// stepped from a 512 Hz frame sequencer. This is plenty accurate for rendering the songs.

const (
	gbClock    = 4194304.0 // CPU/APU clock, Hz
	APURate    = 44100     // output sample rate
	frameSeqHz = 512.0     // frame-sequencer rate
)

// RegWrite is one captured APU register write at a CPU-cycle timestamp.
type RegWrite struct {
	Cycle int64
	Reg   uint16 // $FF10-$FF3F
	Val   byte
}

// dutyTable holds the four square-wave duty patterns (8 steps each).
var dutyTable = [4][8]byte{
	{0, 0, 0, 0, 0, 0, 0, 1}, // 12.5%
	{1, 0, 0, 0, 0, 0, 0, 1}, // 25%
	{1, 0, 0, 0, 0, 1, 1, 1}, // 50%
	{0, 1, 1, 1, 1, 1, 1, 0}, // 75%
}

type square struct {
	enabled    bool
	sweep      bool // ch1 only
	reg        [5]byte
	phase      float64
	dutyPos    int
	freq       int // 11-bit
	length     int
	lengthEn   bool
	vol        int
	envDir     int
	envPeriod  int
	envTimer   int
	swPeriod   int
	swShift    int
	swDir      int
	swTimer    int
	swEnabled  bool
	swShadow   int
}

type wave struct {
	enabled  bool
	dacOn    bool
	phase    float64
	freq     int
	length   int
	lengthEn bool
	volShift int // 0=mute,1=full,2=half,3=quarter (as a right-shift)
	ram      [16]byte
}

type noise struct {
	enabled   bool
	reg2      byte // $FF21 envelope
	lfsr      uint16
	timer     float64
	length    int
	lengthEn  bool
	vol       int
	envDir    int
	envPeriod int
	envTimer  int
	divisor   int
	shift     int
	width7    bool
}

// APU is the full sound chip.
type APU struct {
	ch1, ch2 square
	ch3      wave
	ch4      noise
	powered  bool
	panel    byte // $FF25 panning
	volL     int  // $FF24 left master volume 0-7
	volR     int
}

func NewAPU() *APU {
	a := &APU{}
	a.ch4.lfsr = 0x7FFF
	return a
}

// noiseDivisors maps the 3-bit divisor code to the base divisor.
var noiseDivisors = [8]int{8, 16, 32, 48, 64, 80, 96, 112}

func (a *APU) write(reg uint16, v byte) {
	switch {
	case reg == 0xFF26: // master power
		a.powered = v&0x80 != 0
		return
	case !a.powered && reg < 0xFF30:
		return
	}
	switch reg {
	// --- channel 1 (square + sweep) ---
	case 0xFF10:
		a.ch1.reg[0] = v
		a.ch1.swPeriod = int(v>>4) & 7
		a.ch1.swDir = int(v>>3) & 1
		a.ch1.swShift = int(v) & 7
	case 0xFF11:
		a.ch1.reg[1] = v
		a.ch1.length = 64 - int(v&0x3F)
	case 0xFF12:
		a.ch1.reg[2] = v
	case 0xFF13:
		a.ch1.reg[3] = v
		a.ch1.freq = (a.ch1.freq & 0x700) | int(v)
	case 0xFF14:
		a.ch1.reg[4] = v
		a.ch1.freq = (a.ch1.freq & 0xFF) | int(v&7)<<8
		a.ch1.lengthEn = v&0x40 != 0
		if v&0x80 != 0 {
			a.trigSquare(&a.ch1, true)
		}
	// --- channel 2 (square) ---
	case 0xFF16:
		a.ch2.reg[1] = v
		a.ch2.length = 64 - int(v&0x3F)
	case 0xFF17:
		a.ch2.reg[2] = v
	case 0xFF18:
		a.ch2.reg[3] = v
		a.ch2.freq = (a.ch2.freq & 0x700) | int(v)
	case 0xFF19:
		a.ch2.reg[4] = v
		a.ch2.freq = (a.ch2.freq & 0xFF) | int(v&7)<<8
		a.ch2.lengthEn = v&0x40 != 0
		if v&0x80 != 0 {
			a.trigSquare(&a.ch2, false)
		}
	// --- channel 3 (wave) ---
	case 0xFF1A:
		a.ch3.dacOn = v&0x80 != 0
		if !a.ch3.dacOn {
			a.ch3.enabled = false
		}
	case 0xFF1B:
		a.ch3.length = 256 - int(v)
	case 0xFF1C:
		a.ch3.volShift = int(v>>5) & 3
	case 0xFF1D:
		a.ch3.freq = (a.ch3.freq & 0x700) | int(v)
	case 0xFF1E:
		a.ch3.freq = (a.ch3.freq & 0xFF) | int(v&7)<<8
		a.ch3.lengthEn = v&0x40 != 0
		if v&0x80 != 0 && a.ch3.dacOn {
			a.ch3.enabled = true
			a.ch3.phase = 0
			if a.ch3.length == 0 {
				a.ch3.length = 256
			}
		}
	// --- channel 4 (noise) ---
	case 0xFF20:
		a.ch4.length = 64 - int(v&0x3F)
	case 0xFF21:
		a.ch4.reg2 = v
	case 0xFF22:
		a.ch4.divisor = noiseDivisors[v&7]
		a.ch4.width7 = v&8 != 0
		a.ch4.shift = int(v >> 4)
	case 0xFF23:
		a.ch4.lengthEn = v&0x40 != 0
		if v&0x80 != 0 {
			a.trigNoise(v)
		}
	// --- master ---
	case 0xFF24:
		a.volL = int(v>>4) & 7
		a.volR = int(v) & 7
	case 0xFF25:
		a.panel = v
	}
	if reg >= 0xFF30 && reg <= 0xFF3F {
		a.ch3.ram[reg-0xFF30] = v
	}
}

func (a *APU) trigSquare(c *square, sweep bool) {
	c.enabled = true
	c.phase = 0
	c.dutyPos = 0
	if c.length == 0 {
		c.length = 64
	}
	env := c.reg[2]
	c.vol = int(env >> 4)
	c.envDir = int(env>>3) & 1
	c.envPeriod = int(env) & 7
	c.envTimer = c.envPeriod
	if env&0xF8 == 0 { // DAC off
		c.enabled = false
	}
	if sweep {
		c.swShadow = c.freq
		c.swTimer = c.swPeriod
		if c.swTimer == 0 {
			c.swTimer = 8
		}
		c.swEnabled = c.swPeriod > 0 || c.swShift > 0
	}
}

func (a *APU) trigNoise(v byte) {
	c := &a.ch4
	c.enabled = true
	c.lfsr = 0x7FFF
	if c.length == 0 {
		c.length = 64
	}
	c.vol = int(c.reg2 >> 4)
	c.envDir = int(c.reg2>>3) & 1
	c.envPeriod = int(c.reg2) & 7
	c.envTimer = c.envPeriod
	if c.reg2&0xF8 == 0 {
		c.enabled = false
	}
}

// stepFrame advances the length/sweep/envelope units one 512 Hz frame-sequencer tick.
// step cycles 0-7: length on 0/2/4/6, sweep on 2/6, envelope on 7.
func (a *APU) stepFrame(step int) {
	if step%2 == 0 {
		a.clockLength()
	}
	if step == 2 || step == 6 {
		a.clockSweep()
	}
	if step == 7 {
		a.clockEnv()
	}
}

func (a *APU) clockLength() {
	if a.ch1.lengthEn && a.ch1.length > 0 {
		a.ch1.length--
		if a.ch1.length == 0 {
			a.ch1.enabled = false
		}
	}
	if a.ch2.lengthEn && a.ch2.length > 0 {
		a.ch2.length--
		if a.ch2.length == 0 {
			a.ch2.enabled = false
		}
	}
	if a.ch3.lengthEn && a.ch3.length > 0 {
		a.ch3.length--
		if a.ch3.length == 0 {
			a.ch3.enabled = false
		}
	}
	if a.ch4.lengthEn && a.ch4.length > 0 {
		a.ch4.length--
		if a.ch4.length == 0 {
			a.ch4.enabled = false
		}
	}
}

func (a *APU) clockSweep() {
	c := &a.ch1
	if !c.swEnabled || c.swPeriod == 0 {
		return
	}
	c.swTimer--
	if c.swTimer > 0 {
		return
	}
	c.swTimer = c.swPeriod
	delta := c.swShadow >> c.swShift
	if c.swDir == 1 {
		delta = -delta
	}
	nf := c.swShadow + delta
	if nf > 2047 {
		c.enabled = false
	} else if nf >= 0 && c.swShift > 0 {
		c.swShadow = nf
		c.freq = nf
	}
}

func (a *APU) clockEnv() {
	for _, c := range []*square{&a.ch1, &a.ch2} {
		if c.envPeriod == 0 {
			continue
		}
		c.envTimer--
		if c.envTimer > 0 {
			continue
		}
		c.envTimer = c.envPeriod
		if c.envDir == 1 && c.vol < 15 {
			c.vol++
		} else if c.envDir == 0 && c.vol > 0 {
			c.vol--
		}
	}
	if a.ch4.envPeriod != 0 {
		a.ch4.envTimer--
		if a.ch4.envTimer <= 0 {
			a.ch4.envTimer = a.ch4.envPeriod
			if a.ch4.envDir == 1 && a.ch4.vol < 15 {
				a.ch4.vol++
			} else if a.ch4.envDir == 0 && a.ch4.vol > 0 {
				a.ch4.vol--
			}
		}
	}
}

// channel sample outputs in [-1,1] (pre-mix, per channel), advancing phase by dt seconds.
func (c *square) sample(dt float64) float64 {
	if !c.enabled || c.freq >= 2048 {
		return 0
	}
	hz := gbClock / (4 * float64(2048-c.freq))
	c.phase += hz * dt
	for c.phase >= 1 {
		c.phase--
		c.dutyPos = (c.dutyPos + 1) & 7
	}
	duty := int(c.reg[1] >> 6)
	bit := dutyTable[duty][c.dutyPos]
	v := float64(c.vol) / 15
	if bit == 1 {
		return v
	}
	return -v
}

func (c *wave) sample(dt float64) float64 {
	if !c.enabled || !c.dacOn || c.freq >= 2048 {
		return 0
	}
	hz := gbClock / (2 * float64(2048-c.freq))
	c.phase += hz * dt
	for c.phase >= 32 {
		c.phase -= 32
	}
	idx := int(c.phase)
	b := c.ram[idx/2]
	var s byte
	if idx%2 == 0 {
		s = b >> 4
	} else {
		s = b & 0x0F
	}
	if c.volShift == 0 {
		return 0
	}
	amp := float64(s>>(c.volShift-1)) / 15 // 1=full,2=half,3=quarter
	return amp*2 - 1
}

func (c *noise) sample(dt float64) float64 {
	if !c.enabled {
		return 0
	}
	period := float64(c.divisor) * float64(int(1)<<uint(c.shift))
	if period <= 0 {
		return 0
	}
	hz := gbClock / period
	c.timer += hz * dt
	for c.timer >= 1 {
		c.timer--
		x := (c.lfsr ^ (c.lfsr >> 1)) & 1
		c.lfsr = (c.lfsr >> 1) | (x << 14)
		if c.width7 {
			c.lfsr = (c.lfsr &^ (1 << 6)) | (x << 6)
		}
	}
	v := float64(c.vol) / 15
	if c.lfsr&1 == 0 {
		return v
	}
	return -v
}

// Render plays the captured register-write stream and returns mono 16-bit PCM at APURate.
// totalCycles bounds the render length.
func (a *APU) Render(events []RegWrite, totalCycles int64) []int16 {
	dt := 1.0 / APURate
	cyclesPerSample := gbClock / APURate
	nSamples := int(float64(totalCycles) / cyclesPerSample)
	out := make([]int16, nSamples)

	frameCycles := gbClock / frameSeqHz
	var frameAcc float64
	frameStep := 0
	ei := 0

	for n := 0; n < nSamples; n++ {
		sampleCycle := int64(float64(n) * cyclesPerSample)
		for ei < len(events) && events[ei].Cycle <= sampleCycle {
			a.write(events[ei].Reg, events[ei].Val)
			ei++
		}
		// step the frame sequencer for this sample's worth of cycles
		frameAcc += cyclesPerSample
		for frameAcc >= frameCycles {
			frameAcc -= frameCycles
			a.stepFrame(frameStep)
			frameStep = (frameStep + 1) & 7
		}
		// mix channels with panning/master volume
		s1 := a.ch1.sample(dt)
		s2 := a.ch2.sample(dt)
		s3 := a.ch3.sample(dt)
		s4 := a.ch4.sample(dt)
		var l, r float64
		mix := func(s float64, bit int) {
			if a.panel&(1<<uint(bit+4)) != 0 {
				l += s
			}
			if a.panel&(1<<uint(bit)) != 0 {
				r += s
			}
		}
		mix(s1, 0)
		mix(s2, 1)
		mix(s3, 2)
		mix(s4, 3)
		l *= float64(a.volL+1) / 8
		r *= float64(a.volR+1) / 8
		mono := (l + r) * 0.5 * 0.4 // average L/R, with headroom for ~3 simultaneous channels
		if mono > 1 {
			mono = 1
		} else if mono < -1 {
			mono = -1
		}
		out[n] = int16(mono * 30000)
	}
	return out
}
