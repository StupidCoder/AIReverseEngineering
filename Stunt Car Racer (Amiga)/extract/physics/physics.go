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

// --- track-surface sample ---

// Corners618CE computes the car's four contact-point lateral/longitudinal offsets
// ($1BD02/04/06 and $1BD08/0A/0C) from its projected wheel positions ($1C264/68/6E/72/
// 74/76, left by the renderer). $5C1D0 indexes these to place each contact point.
func (m *Mem) Corners618CE() {
	d4 := (m.W(0x1C26E) >> 1) - (m.W(0x1C264) >> 1)
	d5 := (m.W(0x1C268) >> 1) - (m.W(0x1C272) >> 1)
	d0 := m.W(0x1C274) >> 5
	d3 := m.W(0x1C276) >> 5
	d4 >>= 5
	d5 >>= 5
	m.SetW(0x1BD06, -d0)
	m.SetW(0x1BD0C, -d3)
	m.SetW(0x1BD02, d0-d4)
	m.SetW(0x1BD04, d0+d4)
	m.SetW(0x1BD08, d3-d5)
	m.SetW(0x1BD0A, d3+d5)
}

// Interp5C554 reproduces $5C554: bilinearly interpolate the track surface height under
// the car into $1BB18 from the four surrounding rung-corner heights ($1BC02/04 = the two
// "near" rail samples, $1BC06/08 = "far"), by the along-fraction $1BC4D and the
// across-fraction $1BC41 (both 0-255). The corner heights are the Part IV $5C0AA rail
// heights the renderer leaves in $1BC02-08. A >>3/<<3 path avoids 16-bit overflow when
// the across difference is large; the negative branch clears the product's low byte.
func (m *Mem) Interp5C554() {
	along := int32(m.U8(0x1BC4D))
	left := int32(m.W(0x1BC04)-m.W(0x1BC02))*along + int32(m.W(0x1BC02))<<8
	d0 := int32(m.W(0x1BC08)-m.W(0x1BC06))*along + int32(m.W(0x1BC06))<<8
	across := uint32(uint16(m.U8(0x1BC41)))
	d0 -= left // right - left
	d4 := d0
	if d4 < 0 {
		d4 = -d4
	}
	big := d4 >= 0x8000
	if big {
		d0 >>= 3 // ASR.l #3
	}
	var p int32
	if int16(d0) < 0 { // TST.w ; negative branch
		w := uint16(-int16(d0)) // NEG.w
		p = int32(uint32(w)*across) &^ 0xFF
		p = -p
	} else {
		p = int32(uint32(uint16(int16(d0))) * across) // MULU.W
	}
	if big {
		p <<= 3 // ASL.l #3
	}
	p >>= 8 // ASR.l #8
	p += left
	m.SetL(0x1BB18, p)
}

// --- suspension ($61BCC) ---

// Suspension addresses: the three contact points (left/right/rear). Surf = track surface
// height under the point, Car = chassis contact height, Comp = compression (Surf-Car-
// rest), Travel/Prev = clamped travel and its previous value, Force = spring+damper
// force, Dmg = the point's damage accumulator (0-255).
const (
	Rest = 0x1BCA0
	Spr0Surf, Spr0Car, Spr0Comp, Spr0Travel, Spr0Prev, Spr0Force, Spr0Dmg = 0x1BCA4, 0x1BC94, 0x1BCB0, 0x1BD14, 0x1BD1A, 0x1BD20, 0x1BB4F
	Spr1Surf, Spr1Car, Spr1Comp, Spr1Travel, Spr1Prev, Spr1Force, Spr1Dmg = 0x1BCA8, 0x1BC98, 0x1BCB4, 0x1BD16, 0x1BD1C, 0x1BD22, 0x1BB50
	Spr2Surf, Spr2Car, Spr2Comp, Spr2Travel, Spr2Prev, Spr2Force, Spr2Dmg = 0x1BCAC, 0x1BC9C, 0x1BCB8, 0x1BD18, 0x1BD1E, 0x1BD24, 0x1BB51

	NetLift = 0x1BD38 // average of the three spring forces
	RollTq  = 0x1BD28
	PitchTq = 0x1BD26
	OnGround = 0x1BB7E
	Bottom   = 0x1BB7D
	DmgEvt   = 0x1BB54
	dmgLimit = 0x63CE2 // sustained-impact frame limit (config, in image)
)

// spring6180E: $6180E spring+damper. force = (delta * $114) >> 8 + travel (word add).
func spring6180E(delta, travel int16) int16 {
	p := (int32(delta) * 0x114) >> 8
	return int16(uint16(p) + uint16(travel))
}

// spring runs one of the three identical suspension blocks: compute compression and the
// clamped spring+damper force, accumulate bottoming/damage, and clamp the force.
func (m *Mem) spring(surf, car, comp, travel, prev, force, dmg uint32) {
	d0 := m.L(surf) - m.L(car) - m.L(Rest)
	m.SetL(comp, d0)
	if d0 >= 0 {
		if d0 >= 0x1400 {
			d0 = 0x1400
		}
	} else if d0 < -0x300 {
		d0 = -0x300
	}
	m.SetW(travel, int16(d0))
	d6 := int16(d0)
	f := spring6180E(int16(d0)-m.W(prev), d6)
	if f < 0 { // negative force -> zeroed, impact counter reset
		m.SetW(force, 0)
		m.B[0x1BB56] = 0
		m.SetW(prev, m.W(travel))
		return
	}
	d4 := m.W(force) // previous force
	m.SetW(force, f)
	if f >= 0x400 && d4 < 0x200 { // hard bottoming
		m.B[Bottom]++ // ADDQ.b #1 (byte)
	}
	d := f - int16(m.U8(0x1BB01))<<8
	if d < 0 || d < 0x700 { // below damage threshold: reset impact counter
		m.B[0x1BB56] = 0
		m.SetW(prev, m.W(travel))
		return
	}
	if d >= m.W(0x1BC3A) { // track the peak
		m.SetW(0x1BC3A, d)
	}
	d -= 0x600
	if int8(m.U8(0x1BBCD)) >= 0 { // damage enabled
		m.B[0x1BB56] = byte(m.U8(0x1BB56) + 1)
		if int8(m.U8(0x1BB56)) < int8(m.U8(dmgLimit)) {
			sev := int(uint16(d) >> 8)         // LSR.w #8
			sev = (sev + (sev >> 1)) & 0xFF     // + half (byte)
			n := sev + m.U8(dmg)
			if n > 0xFF {
				n = 0xFF
			}
			m.B[dmg] = byte(n)
			m.B[DmgEvt] = 0x80
		}
	}
	if m.W(force) >= 0x1200 { // clamp force
		m.SetW(force, 0x11FF)
	}
	m.SetW(prev, m.W(travel))
}

// ContactHeights61B70 computes the chassis' three contact-point heights ($1BC94/98/9C)
// from the car height ($1BCDC) tilted by sin(roll) and sin(pitch) -- the geometry the
// springs compare against the track surface.
func (m *Mem) ContactHeights61B70() {
	sr := m.Sin(m.W(Roll))
	m.SetW(0x1BBF6, sr)
	d0 := int32(m.Sin(m.W(Pit))) << 3
	d3 := int32(sr) << 4
	m.SetL(Spr2Car, (m.L(PosY)-d3)>>8) // $1BC9C
	d4 := m.L(PosY) + d3
	m.SetL(Spr1Car, (d4-d0)>>8) // $1BC98
	m.SetL(Spr0Car, (d4+d0)>>8) // $1BC94
}

// Suspension61BCC reproduces $61BCC: the three spring blocks then the combine into net
// lift ($1BD38), roll torque ($1BD28), pitch torque ($1BD26) and the on-ground flag
// ($1BB7E), plus the airborne self-righting nudge. It omits the engine's three external
// calls ($622DC surface-slope/loads, $5B32E, $63E2E), whose outputs are disjoint from
// these; those are separate routines.
func (m *Mem) Suspension61BCC() {
	m.B[Bottom] = 0
	m.B[0x1BC3A] = 0 // MOVE.b -- clears only the high byte (matches the engine)
	m.spring(Spr0Surf, Spr0Car, Spr0Comp, Spr0Travel, Spr0Prev, Spr0Force, Spr0Dmg)
	m.spring(Spr1Surf, Spr1Car, Spr1Comp, Spr1Travel, Spr1Prev, Spr1Force, Spr1Dmg)
	m.spring(Spr2Surf, Spr2Car, Spr2Comp, Spr2Travel, Spr2Prev, Spr2Force, Spr2Dmg)

	d0 := (m.W(Spr0Force) + m.W(Spr1Force)) >> 1
	m.SetW(0x1BBF6, d0)
	d0 = (d0 + m.W(Spr2Force)) >> 1
	m.SetW(NetLift, d0)
	// (engine calls $622DC here -- surface slope + body loads; disjoint, done separately)

	// roll torque = clamp(3*(spr0-spr1), +-$1000), sign preserved
	dd := m.W(Spr0Force) - m.W(Spr1Force)
	t := dd<<1 + dd
	if t < 0 {
		t = -t
	}
	if t >= 0x1000 {
		t = 0x1000
	}
	if dd < 0 {
		t = -t
	}
	m.SetW(RollTq, t)
	// pitch torque = frontAvg - rear
	m.SetW(PitchTq, m.W(0x1BBF6)-m.W(Spr2Force))

	// on-ground flag = (high byte | low byte) of net lift
	og := m.U8(NetLift) | m.U8(NetLift+1)
	m.B[OnGround] = byte(og)
	if og != 0 {
		return // grounded: no self-righting
	}
	if m.U8(0x1BBDF) != 0 {
		return
	}
	// airborne self-righting (Ski Jump / Roller Coaster): nudge pitch torque.
	d3 := int16(-0x80)
	roll := m.W(Roll)
	if roll >= 0 {
		if roll >= 0x1000 {
			d3 = int16(-0x100) // $FF00
		}
	} else {
		switch m.U8(0x1CA33) {
		case 7:
			d3 = int16(-0x80) // d1=$F8 set in engine but d3 stays $FF80; pitch path uses d3
		case 4:
			d3 = int16(-0x8) // $FFF8
		default:
			return
		}
	}
	d3 -= m.W(PitchTq)
	if d3 >= 0 {
		return
	}
	c := m.U8(0x1BCF0)
	if int8(c) >= 0 || c == 0xFF {
		m.SetW(PitchTq, d3)
	}
}

// --- drive / tire forces ---

const (
	Drive   = 0x1BD2A // longitudinal drive (engine/wheelspin) force
	LoadA   = 0x1BD40 // body loads from the net lift (set by $622DC)
	LoadB   = 0x1BD42 // the tire-load component used for grip
	LoadC   = 0x1BD44
	Slip    = 0x1BBC1 // lateral slide flag
	TqAppR  = 0x1BCFC // applied roll torque (= TqR)
	TqAppY  = 0x1BD00 // applied yaw torque (= TqY)
)

// grip621DA: tire grip = LoadB*2 when on the ground ($1BB7E set), else 0 (no grip in
// the air). Returns the engine's d0.
func (m *Mem) grip621DA() int16 {
	if m.U8(OnGround) == 0 {
		return 0
	}
	return m.W(LoadB) << 1
}

// LateralTire6217A: the lateral tire force $1BD32 with a grip limit, plus the slide flag
// $1BBC1 (set when the demand exceeds grip).
func (m *Mem) LateralTire6217A() {
	d4 := m.W(GrvA) + m.W(LoadA)
	d3 := d4 - m.W(BVelL)
	if d3 < 0 {
		d3 = -d3
	}
	g := m.grip621DA()
	if uint16(d3) < uint16(g) { // gripping
		m.SetW(BFrcA, m.W(LoadA)-m.W(BVelL))
		m.B[Slip] = 0
		return
	}
	if m.W(BVelL) < 0 { // sliding: oppose the slide
		g = -g
	}
	m.SetW(BFrcA, d4-g)
	m.B[Slip] = 0x80
}

// Drive620B8: the longitudinal drive force. $1BD34 = gravity-x + load; manage the drive
// force $1BD2A (wheelspin decay then grip clamp); $1BD36 = drive + load + gravity-z;
// then the lateral tire force.
func (m *Mem) Drive620B8() {
	m.SetW(BFrcB, m.W(GrvB)+m.W(LoadB))
	d0b := m.U8(Drive) | m.U8(BVelV) // $1BD2A.b | $1BD30.b
	if int8(d0b) >= 0 && m.U8(0x1BD2B) != 0 {
		m.SetW(Drive, m.W(Drive)-int16(d0b&0xFF))
	}
	d3 := m.W(Drive)
	if d3 < 0 {
		d3 = -d3
	}
	g := m.grip621DA()
	if uint16(d3) >= uint16(g) { // |drive| >= grip (SUB.w no borrow): clamp to +-grip
		if m.W(Drive) >= 0 {
			m.SetW(Drive, g)
		} else {
			m.SetW(Drive, -g)
		}
	}
	m.SetW(BFrcC, m.W(Drive)+m.W(LoadC)+m.W(GrvC))
	m.LateralTire6217A()
}

// slope62424 ($62424): abs+clamp a slope value to 0..$FF ($1BB2B) and look up its
// half-angle in the table at $1EECA ($1BB2D).
func (m *Mem) slope62424(d0w int16) {
	d0 := int(d0w)
	if d0 < 0 {
		d0 = -d0
	}
	d1 := 0xFF
	if d0 < 0x100 {
		d1 = d0 & 0xFF
	}
	m.B[0x1BB2B] = byte(d1)
	m.B[0x1BB2D] = m.B[0x1EECA+uint32(d1>>1)]
}

// frac612A2 ($612A2): (d0 * $1BB1A) >> 8 (the low byte lands in $1BB1B).
func (m *Mem) frac612A2(d0 int) int {
	d3 := (m.U8(0x1BB1A) & 0xFF) * (d0 & 0xFF)
	m.B[0x1BB1B] = byte(d3)
	return (d3 >> 8) & 0xFFFF
}

// LoadProject622DC ($622DC): derive the road-surface slope under the car from the spring
// compressions ($1BCB0/B4/B8) and project the net lift ($1BD38) onto the three body axes
// -> the loads $1BD40/$1BD42/$1BD44 (the tire-load components that feed grip and drive).
func (m *Mem) LoadProject622DC() {
	m.SetW(0x1BD4A, 0)
	d0 := ((m.L(0x1BCB0)+m.L(0x1BCB4))>>1 - m.L(0x1BCB8)) >> 4
	m.SetW(0x1BD4C, int16(uint16(int16(d0))^0x8000))
	m.slope62424(int16(d0))
	m.B[0x1BB2C] = m.B[0x1BB2D]
	m.B[0x1BD52] = m.B[0x1BB2B]

	d0 = (m.L(0x1BCB0) - m.L(0x1BCB4)) >> 3
	m.SetW(0x1BD48, int16(d0))
	m.slope62424(int16(d0))
	m.B[0x1BB1A] = m.B[0x1BB2C]
	m.B[0x1BD50] = byte(m.frac612A2(m.U8(0x1BB2D)))
	m.B[0x1BD4E] = byte(m.frac612A2(m.U8(0x1BB2B)))

	// project the net lift onto each axis: load = (netLift * (+-factor << 7)) >> 15,
	// the factor from $1BB1A and the sign from $1BBBB.
	proj := func() int16 {
		d3 := m.U8(0x1BB1A) & 0xFF
		if int8(m.U8(0x1BBBB)) < 0 {
			d3 = (0 - d3) & 0xFFFF // NEG.w
		}
		d3 = (d3 << 7) & 0xFFFF
		p := int32(m.W(0x1BD38)) * int32(int16(uint16(d3)))
		return int16((p << 1) >> 16)
	}
	m.B[0x1BB1A], m.B[0x1BBBB] = m.B[0x1BD4E], m.B[0x1BD48]
	m.SetW(0x1BD40, proj())
	m.B[0x1BB1A], m.B[0x1BBBB] = m.B[0x1BD50], m.B[0x1BD4A]
	m.SetW(0x1BD42, proj())
	m.B[0x1BB1A], m.B[0x1BBBB] = m.B[0x1BD52], m.B[0x1BD4C]
	m.SetW(0x1BD44, proj())
}

// Drag621F4 ($621F4): pick a drag coefficient and shift, then subtract velocity*coef
// from the world force. On the ground and gripping (small steer/slip) the coefficient is
// a fixed $6000 with a light shift (strong, near-rigid damping); rolling free it scales
// with the fastest body-velocity component over a $A00 deadzone (speed-proportional air
// drag).
func (m *Mem) Drag621F4() {
	absw := func(v int16) int16 {
		if v < 0 {
			return int16(-int32(v))
		}
		return v
	}
	d7 := uint(1)
	d0 := int16(0x6000)
	handled := false
	if m.U8(OnGround) != 0 {
		s := m.U8(0x1BD46)
		if int8(s) < 0 {
			s ^= 0xFF
		}
		if s&0xFF >= 3 || int8(m.U8(0x1BB9C)) < 0 {
			handled = true
		} else if m.U8(0x1BCA2) != 0 {
			d7, handled = 3, true
		}
	}
	low := false
	if !handled {
		if m.U8(0x1BBDF) == 0 {
			low = true
		} else {
			d7 = 3
		}
	}
	if low {
		d0 = absw(m.W(0x1BD2C))
		if v := absw(m.W(0x1BD2E)); v > d0 {
			d0 = v
		}
		if v := absw(m.W(0x1BD30)); v > d0 {
			d0 = v
		}
		d7 = 5
		if int8(m.U8(0x1BBC7)) < 0 && int8(m.U8(0x1BBB8)) >= 0 {
			if uint16(d0) >= 0xA00 { // SUB.w ; BCC (no borrow)
				d0 = int16(uint16(d0) - 0xA00)
			} else {
				d0 = 0
			}
		}
	}
	apply := func(va, fa uint32) {
		hi := int16(int32(m.W(va))*int32(d0)>>16) >> d7 // MULS.W ; SWAP ; ASR.w d7
		m.SetW(fa, m.W(fa)-hi)
	}
	apply(VelX, FrcX)
	apply(VelY, FrcY)
	apply(VelZ, FrcZ)
}

// TorqueApply62138: form the applied roll/yaw torques from the suspension/steering
// torques and the angular momentum. $1BCFC = $1BD26 - (AmR>>4) [+ $1BD36>>2 if grounded];
// $1BD00 = $1BD28 - (AmY>>4).
func (m *Mem) TorqueApply62138() {
	d0 := m.W(PitchTq) - (m.W(AmR) >> 4)
	if m.U8(OnGround) != 0 {
		d0 += m.W(BFrcC) >> 2
	}
	m.SetW(TqAppR, d0)
	m.SetW(TqAppY, m.W(RollTq)-(m.W(AmY)>>4))
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
