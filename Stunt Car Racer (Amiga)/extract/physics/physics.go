// Package physics is an independent Go reimplementation of Stunt Car Racer's car
// physics (Stunt_Car_Racer.md Part V). It operates directly on the game's 24-bit flat
// memory image at base $E700 — the same byte layout the original code uses — so the
// reimplementation can be checked address-for-address against the engine running on the
// tools/m68k core (cmd/physverify). Per the project rule this is independent Go; the
// oracle only verifies, it is never the source of shipped data.
//
// The car is a sprung rigid body integrated with semi-implicit Euler and a 0.93 (=$EE/
// 256) damping factor at both the force->velocity and velocity->position stages. This
// file holds the deterministic core: the sin/cos table lookup, the orientation-matrix
// multiply helper, and the integrator ($61ADC force->vel, $61B26 torque->angmom,
// $61950+$619E4 vel->pos and angle clamp).
package physics

const Base = 0xE700

// Mem wraps the 24-bit flat address space (game image copied at $E700). All car state
// lives in it at fixed addresses, exactly as the engine lays it out.
type Mem struct{ B []byte }

func New(b []byte) *Mem {
	m := make([]byte, 1<<24)
	copy(m[Base:], b)
	return &Mem{m}
}

// big-endian word/long access (the 68000's order).
func (m *Mem) U8(a uint32) int    { return int(m.B[a]) }
func (m *Mem) W(a uint32) int16   { return int16(uint16(m.B[a])<<8 | uint16(m.B[a+1])) }
func (m *Mem) L(a uint32) int32   { return int32(uint32(uint16(m.W(a)))<<16 | uint32(uint16(m.W(a+2)))) }
func (m *Mem) SetW(a uint32, v int16) {
	m.B[a] = byte(uint16(v) >> 8)
	m.B[a+1] = byte(v)
}
func (m *Mem) SetL(a uint32, v int32) {
	m.SetW(a, int16(v>>16))
	m.SetW(a+2, int16(v))
}

// Car-state block addresses (Part V map).
const (
	PosX = 0x1BCD8 // 32-bit (16.16); integer part = high word
	PosY = 0x1BCDC
	PosZ = 0x1BCE0
	Roll = 0x1BCE4 // 16-bit angle ($10000 = full circle)
	Yaw  = 0x1BCE6
	Pit  = 0x1BCE8
	VelX = 0x1BCEA
	VelY = 0x1BCEC
	VelZ = 0x1BCEE
	AmR  = 0x1BCF0 // angular momentum (body): roll/pitch/yaw
	AmP  = 0x1BCF2
	AmY  = 0x1BCF4
	FrcX = 0x1BCF6 // world force accumulator
	FrcY = 0x1BCF8
	FrcZ = 0x1BCFA
	TqR  = 0x1BCFC // body torque
	TqP  = 0x1BCFE
	TqY  = 0x1BD00
	WAmR = 0x1BD3A // world angular rate
	WAmY = 0x1BD3C
	WAmP = 0x1BD3E
	Mtx  = 0x1C230 // orientation matrix (words at +$4.. built by $61368)
	angLimits = 0x61AD4

	damp = 0xEE // 238/256 = 0.9297 per-frame damping
)

// mul0_93 reproduces the engine's "* $EE >> 8" damping: word in, damped word out.
// ($61950/$61ADC: MOVE.b #$EE,d2; MULS.W d2,d0; ASR.l #8,d0).
func mul0_93(v int16) int16 {
	return int16((int32(v) * damp) >> 8)
}

// Sin/Cos reproduce the engine's $64D08/$64D10: angle a (16-bit, $10000 = 2pi) via the
// quarter-wave table at $1CA42 with linear interpolation, result a signed 16-bit value
// (1.0 == $7FFF). $64D08 (selector 0) is SINE, $64D10 (selector $4000) is COSINE.
func (m *Mem) Sin(a int16) int16 { return m.sinSel(a, 0x0000) } // $64D08
func (m *Mem) Cos(a int16) int16 { return m.sinSel(a, 0x4000) } // $64D10

func (m *Mem) sinSel(a int16, d5 int) int16 {
	d0 := int(uint16(a))
	d3 := d0 & 0x3FFF
	// $64D28 BNE $64D32 skips the mirror; the fall-through ($64D2C) mirrors, so the
	// mirror is applied when the EOR result is zero.
	if (d0&0x4000)^d5 == 0 {
		d3 = ((d3 ^ 0x3FFF) + 1) & 0xFFFF
	}
	d3 = ror16(d3, 5)
	d4 := uint32(d3 & 0x3FE) // table byte offset
	tbl := uint32(0x1CA42)
	s0 := m.W(tbl + d4)          // table[idx]
	s1 := m.W(tbl + d4 + 2)      // table[idx+1]
	d6 := uint16(s0) - uint16(s1) // signed difference, used as unsigned by MULU
	frac := uint16(ror16(d3, 1) & 0xFC00)
	hi := uint16((uint32(frac) * uint32(d6)) >> 16) // MULU.W then SWAP
	d7 := int16(uint16(s0) - hi)
	d7 = int16(uint16(d7) >> 1) // LSR.w #1
	// sign: d3' = (d0 & d5) << 1 ; if (d0 EOR d3') < 0 negate
	sg := (d0 & d5) << 1
	if int16(uint16(d0^sg)) < 0 {
		d7 = -d7
	}
	return d7
}

func ror16(v, n int) int {
	v &= 0xFFFF
	return ((v >> n) | (v << (16 - n))) & 0xFFFF
}

// MtxMul reproduces $61344: returns value * matrix[$1C230 + idx*2] >> 15 (the engine's
// MULS.W ; ASL.l #1 ; SWAP). idx selects an orientation-matrix word.
func (m *Mem) MtxMul(value int16, idx int) int16 {
	d3 := m.W(Mtx + uint32(idx*2))
	p := int32(value) * int32(d3)
	p <<= 1
	return int16(p >> 16)
}

// Force61ADC: world force ($1BCF6/F8/FA) * 0.93 -> += velocity ($1BCEA/EC/EE).
func (m *Mem) Force61ADC() {
	m.SetW(VelX, m.W(VelX)+mul0_93(m.W(FrcX)))
	m.SetW(VelY, m.W(VelY)+mul0_93(m.W(FrcY)))
	m.SetW(VelZ, m.W(VelZ)+mul0_93(m.W(FrcZ)))
}

// Torque61B26: body torque ($1BCFC/FE/$1BD00) * 0.93 -> += angular momentum.
func (m *Mem) Torque61B26() {
	m.SetW(AmR, m.W(AmR)+mul0_93(m.W(TqR)))
	m.SetW(AmP, m.W(AmP)+mul0_93(m.W(TqP)))
	m.SetW(AmY, m.W(AmY)+mul0_93(m.W(TqY)))
}

// Integrate61950: velocity -> position (damped, scaled <<6/7/6), height clamp $3E8,
// then angular rate -> angles with the $619E4 limit clamp.
func (m *Mem) Integrate61950() {
	m.SetL(PosX, m.L(PosX)+int32(mul0_93(m.W(VelX)))<<6)
	m.SetL(PosY, m.L(PosY)+int32(mul0_93(m.W(VelY)))<<7)
	m.SetL(PosZ, m.L(PosZ)+int32(mul0_93(m.W(VelZ)))<<6)
	if m.W(PosY) >= 0x3E8 { // BLT skip ; else clamp the high word
		m.SetW(PosY, 0x3E8)
	}
	m.SetW(Roll, m.W(Roll)+mul0_93(m.W(WAmR)))
	d0 := mul0_93(m.W(WAmY)) // falls into $619E4 with d0 = damped yaw rate
	m.clamp619E4(d0)
}

// clamp619E4 reproduces $619E4: apply yaw/pitch rate, then clamp roll & pitch against
// the $61AD4 limit table, zeroing the matching angular momentum when a limit is hit.
func (m *Mem) clamp619E4(d0 int16) {
	m.SetW(Yaw, m.W(Yaw)+d0)
	m.SetW(Pit, m.W(Pit)+mul0_93(m.W(WAmP)))

	d2 := uint32(0)
	if int8(m.U8(0x1BB75)) < 0 && m.U8(0x1BB9A) == 0xE0 {
		d2 = 2
	}
	a0 := uint32(angLimits)
	// roll vs limits, zero AmR ($1BCF0) when clamped
	m.clampAngle(Roll, AmR, a0, d2)
	// pitch vs limits, zero AmY ($1BCF4) when clamped
	m.clampAngle(Pit, AmY, a0, d2)
}

func (m *Mem) clampAngle(ang, mom, a0, d2 uint32) {
	d3 := m.W(ang)
	var lim int16
	if d3 >= 0 {
		lim = m.W(a0 + d2) // positive limit at +0
		if uint16(lim) >= uint16(d3) {
			return // within limit (CMP d3,d0 ; BCC)
		}
	} else {
		lim = m.W(a0 + d2 + 4) // negative limit at +4
		if uint16(lim) < uint16(d3) {
			return // BCS
		}
	}
	m.SetW(ang, lim)
	// if clamped angle and momentum share sign, kill the momentum
	if (lim ^ m.W(mom)) >= 0 {
		m.SetW(mom, 0)
	}
}
