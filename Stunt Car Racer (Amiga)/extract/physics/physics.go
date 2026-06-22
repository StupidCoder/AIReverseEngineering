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
	Tmpl = 0x1EC46 // matrix-element template (which slot each transform term uses)
	Hdg  = 0x1BD5A // section heading (subtracted from yaw in the matrix build)

	BVelL = 0x1BD2C // body velocity (longitudinal / lateral / vertical)
	BVelM = 0x1BD2E
	BVelV = 0x1BD30
	GrvA = 0x1BD0E // gravity expressed in body frame ($615E6)
	GrvB = 0x1BD10
	GrvC = 0x1BD12
	BFrcA = 0x1BD32 // body force components (rotated to world by $61618)
	BFrcB = 0x1BD34
	BFrcC = 0x1BD36

	damp = 0xEE // 238/256 = 0.9297 per-frame damping
	grav = 0x13D // gravity constant magnitude (317); $615E6 uses -317/+317
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

// --- orientation matrix and frame transforms ---

// mt/smt read/write a matrix word by its byte offset within $1C230.
func (m *Mem) mt(off uint32) int16     { return m.W(Mtx + off) }
func (m *Mem) smt(off uint32, v int16) { m.SetW(Mtx+off, v) }

// prod is the engine's "MULS.W d5,d4 ; ASL.l #1 ; SWAP" applied in place to a matrix
// slot: m[off] = (m[off] * d5) >> 15.
func (m *Mem) prod(off uint32, d5 int16) {
	p := int32(m.mt(off)) * int32(d5)
	m.smt(off, int16((p<<1)>>16))
}

// Matrix61368 builds the chassis orientation matrix at $1C230 from the three Euler
// angles (yaw $1BCE6 less the section heading $1BD5A, roll $1BCE4, pitch $1BCE8) — a
// literal transcription of $61368: seed the slots with sin/cos of each angle, multiply
// them together into the composite rotation, then form the cross terms.
func (m *Mem) Matrix61368() {
	sy := m.Sin(m.W(Yaw))
	for _, o := range []uint32{0x4, 0xC, 0xE, 0x14, 0x16} {
		m.smt(o, sy)
	}
	cy := m.Cos(m.W(Yaw))
	for _, o := range []uint32{0x6, 0x10, 0x12, 0x18, 0x1A} {
		m.smt(o, cy)
	}
	yh := m.W(Yaw) - m.W(Hdg)
	sh := m.Sin(yh)
	for _, o := range []uint32{0x34, 0x42, 0x44} {
		m.smt(o, sh)
	}
	ch := m.Cos(yh)
	for _, o := range []uint32{0x38, 0x3E, 0x46} {
		m.smt(o, ch)
	}
	m.smt(0x8, m.Sin(m.W(Roll)))
	cr := m.Cos(m.W(Roll))
	for _, o := range []uint32{0xA, 0x1C, 0x1E} {
		m.smt(o, cr)
	}
	m.smt(0x22, m.Cos(m.W(Pit)))
	m.smt(0x20, m.Sin(m.W(Pit)))

	// in-place product cascades (each: m[d3] = m[d3]*d5 >> 15 over a slot range).
	d5 := m.mt(0x8) // sin roll
	for o := uint32(0xC); o <= 0x12; o += 2 {
		m.prod(o, d5)
	}
	for o := uint32(0x34); o <= 0x38; o += 4 {
		m.prod(o, d5)
	}
	m.smt(0x0, m.mt(0xC))
	m.smt(0x2, m.mt(0x10))
	d5 = m.mt(0xA) // cos roll
	for o := uint32(0x4); o <= 0x6; o += 2 {
		m.prod(o, d5)
	}
	for o := uint32(0x44); o <= 0x46; o += 2 {
		m.prod(o, d5)
	}
	d5 = m.mt(0x20) // sin pit
	for o := uint32(0xC); o <= 0x1C; o += 4 {
		m.prod(o, d5)
	}
	for o := uint32(0x34); o <= 0x38; o += 4 {
		m.prod(o, d5)
	}
	d5 = m.mt(0x22) // cos pit
	for o := uint32(0xE); o <= 0x1E; o += 4 {
		m.prod(o, d5)
	}
	for o := uint32(0x3E); o <= 0x42; o += 4 {
		m.prod(o, d5)
	}
	m.smt(0x28, m.mt(0x18)-m.mt(0xE))
	m.smt(0x2A, -m.mt(0x12)-m.mt(0x14))
	m.smt(0x2C, m.mt(0x1A)+m.mt(0xC))
	m.smt(0x2E, m.mt(0x10)-m.mt(0x16))
	m.smt(0x30, -m.mt(0x1C))
	m.smt(0x24, -m.mt(0x20))
}

// tmpl reads a template byte (matrix-slot selector) at $1EC46 + off.
func (m *Mem) tmpl(off uint32) int { return m.U8(Tmpl + off) }

// VelToBody6158C rotates the world velocity ($1BCEA/EC/EE) into the body frame,
// writing $1BD30 (d2=2) and $1BD2C (d2=0) — the engine's two-component form ($6158C
// steps d2 by 2). Each output sums three velocity*matrix terms via the template.
func (m *Mem) VelToBody6158C() {
	for d2 := uint32(2); ; d2 -= 2 {
		d5 := int16(0)
		d5 += m.MtxMul(m.W(VelX), m.tmpl(d2+0))
		d5 += m.MtxMul(m.W(VelY), m.tmpl(d2+3))
		d5 += m.MtxMul(m.W(VelZ), m.tmpl(d2+6))
		m.SetW(BVelL+(d2<<1), d5)
		if d2 == 0 {
			break
		}
	}
}

// GravToBody615E6 expresses the constant world-down gravity vector in the body frame
// ($1BD0E/10/12) by multiplying +-317 through three matrix slots.
func (m *Mem) GravToBody615E6() {
	m.SetW(GrvB, m.MtxMul(-grav, 0xF)) // $61338 (-317), idx $F -> $1BD10
	m.SetW(GrvC, m.MtxMul(-grav, 0x4)) // -> $1BD12
	m.SetW(GrvA, m.MtxMul(grav, 0xE))  // $61340 (+317), idx $E -> $1BD0E
}

// ForceToWorld61618 rotates the body force ($1BD32/34/36) into world force
// ($1BCF6/F8/FA); three components (d2 steps by 1).
func (m *Mem) ForceToWorld61618() {
	for d2 := uint32(2); ; d2 -= 1 {
		d5 := int16(0)
		d5 += m.MtxMul(m.W(BFrcA), m.tmpl(d2+0x9))
		d5 += m.MtxMul(m.W(BFrcB), m.tmpl(d2+0xC))
		d5 += m.MtxMul(m.W(BFrcC), m.tmpl(d2+0xF))
		m.SetW(FrcX+(d2<<1), d5)
		if d2 == 0 {
			break
		}
	}
}

// TorqueToWorld61672 rotates body angular momentum ($1BCF0/F2) into world angular rate
// ($1BD3A/3C), then forms $1BD3E from $1BD3C and the yaw momentum $1BCF4.
func (m *Mem) TorqueToWorld61672() {
	for d2 := uint32(1); ; d2 -= 1 {
		d5 := int16(0)
		d5 += m.MtxMul(m.W(AmR), m.tmpl(d2+0x12))
		d5 += m.MtxMul(m.W(AmP), m.tmpl(d2+0x14))
		m.SetW(WAmR+(d2<<1), d5)
		if d2 == 0 {
			break
		}
	}
	m.SetW(WAmP, m.MtxMul(m.W(WAmY), 0x4)+m.W(AmY))
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
