package gamegear

// PSG models the SN76489 programmable sound generator the Game Gear uses for music and
// effects: three square-wave tone channels and one noise channel, programmed through a
// single write port ($7F). We do not run it cycle-accurately; instead we keep the latched
// register state so a renderer can snapshot the four channels once per video frame (the
// rate at which the music driver updates them) and synthesise the waveform — the standard
// way PSG logs (VGM) are rendered.
//
// Register model (8 regs): even tone regs 0/2/4 hold a 10-bit period; odd regs 1/3/5/7 hold
// a 4-bit attenuation (0 = loudest, 15 = silent); reg 6 is the 4-bit noise control.
type PSG struct {
	Reg   [8]uint16
	latch int
}

// Write decodes one byte written to the PSG port ($7F).
func (p *PSG) Write(v byte) {
	if v&0x80 != 0 { // LATCH/DATA byte: channel + type + low 4 data bits
		p.latch = int((v >> 4) & 0x07)
		if p.latch&1 == 0 && p.latch != 6 { // tone period: set low 4 bits
			p.Reg[p.latch] = (p.Reg[p.latch] & 0x3F0) | uint16(v&0x0F)
		} else { // attenuation or noise control: whole 4-bit value
			p.Reg[p.latch] = uint16(v & 0x0F)
		}
		return
	}
	// DATA byte: high 6 bits of the last-latched tone period (else re-writes the 4-bit reg)
	if p.latch&1 == 0 && p.latch != 6 {
		p.Reg[p.latch] = (uint16(v&0x3F) << 4) | (p.Reg[p.latch] & 0x0F)
	} else {
		p.Reg[p.latch] = uint16(v & 0x0F)
	}
}

// PSGClock is the SN76489 input clock on the Game Gear (NTSC colour clock).
const PSGClock = 3579545.0

// psgVolume maps a 4-bit attenuation to a linear amplitude (2 dB per step, 15 = silent).
var psgVolume = func() [16]float64 {
	var t [16]float64
	a := 1.0
	for i := 0; i < 15; i++ {
		t[i] = a
		a *= 0.79432823 // 10^(-2/20)
	}
	t[15] = 0
	return t
}()
