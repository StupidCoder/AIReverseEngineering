// Stunt Car Racer — the car physics, hand-ported from the verified Go reimplementation
// (Stunt Car Racer (Amiga)/extract/physics, Part V) one routine at a time. The Go is
// checked coordinate-exact against the original 68000 on the m68k core; this JS is kept
// exact by self-checking against a Go-generated golden trace (selfTest below). It runs
// directly on the game's memory image (the $1B000..$30000 slice the physics reads, loaded
// from <id>.bin, plus a few baked code-region constants), exactly as the original does.
//
// Fixed-point: i16/u16/i8 wrap to width; 16x16 products use Math.imul; >> is arithmetic,
// >>> logical, exactly mirroring the m68k word/long ops.

const i16 = (x) => (x << 16) >> 16;
const u16 = (x) => x & 0xFFFF;
const i8 = (x) => (x << 24) >> 24;
const ror16 = (v, n) => { v &= 0xFFFF; return ((v >>> n) | (v << (16 - n))) & 0xFFFF; };

// addresses (same as package physics)
const A = {
  PosX: 0x1BCD8, PosY: 0x1BCDC, PosZ: 0x1BCE0,
  Roll: 0x1BCE4, Yaw: 0x1BCE6, Pit: 0x1BCE8,
  VelX: 0x1BCEA, VelY: 0x1BCEC, VelZ: 0x1BCEE,
  AmR: 0x1BCF0, AmP: 0x1BCF2, AmY: 0x1BCF4,
  FrcX: 0x1BCF6, FrcY: 0x1BCF8, FrcZ: 0x1BCFA,
  TqR: 0x1BCFC, TqP: 0x1BCFE, TqY: 0x1BD00,
  WAmR: 0x1BD3A, WAmY: 0x1BD3C, WAmP: 0x1BD3E,
  Mtx: 0x1C230, Tmpl: 0x1EC46, Hdg: 0x1BD5A, angLimits: 0x61AD4,
  BVelL: 0x1BD2C, BVelM: 0x1BD2E, BVelV: 0x1BD30,
  GrvA: 0x1BD0E, GrvB: 0x1BD10, GrvC: 0x1BD12,
  BFrcA: 0x1BD32, BFrcB: 0x1BD34, BFrcC: 0x1BD36,
  Rest: 0x1BCA0, NetLift: 0x1BD38, RollTq: 0x1BD28, PitchTq: 0x1BD26,
  OnGround: 0x1BB7E, Bottom: 0x1BB7D, DmgEvt: 0x1BB54,
  Drive: 0x1BD2A, LoadA: 0x1BD40, LoadB: 0x1BD42, LoadC: 0x1BD44, Slip: 0x1BBC1,
  TqAppR: 0x1BCFC, TqAppY: 0x1BD00,
};
const DAMP = 0xEE, GRAV = 0x13D, DMGLIMIT = 0x63CE2;
const MAGIC = 0x9CEDCD02; // genuine-disk protection value

export class Physics {
  constructor() { this.B = new Uint8Array(0x65000); }

  // load the per-track init ($1B000..$1C800) and the shared static data ($1C800..$30000),
  // and bake the code-region constants the physics reads.
  loadTrack(perTrackBuf, staticBuf) {
    this.B.fill(0);
    this.B.set(new Uint8Array(staticBuf), 0x1C900);
    const u = new Uint8Array(perTrackBuf);
    const split = 0x1C900 - 0x1B000;
    this.B.set(u.subarray(0, split), 0x1B000);
    this.B.set(u.subarray(split), 0x1CA1A); // control bytes (override static's track-0 values)
    this.B.set([0, 0, 0, 217, 255, 39], 0x6125A);            // $6125A table
    this.B.set([44, 0, 10, 0, 211, 0, 245, 0, 48, 57, 0, 1], 0x61AD4); // $61AD4 limits
    this.B.set([0x9C, 0xED, 0xCD, 0x02], 0x64AEC);           // protection: genuine disk
  }

  // --- memory access (big-endian, like the 68000) ---
  u8(a) { return this.B[a]; }
  w(a) { return i16((this.B[a] << 8) | this.B[a + 1]); }
  setW(a, v) { this.B[a] = (v >> 8) & 0xFF; this.B[a + 1] = v & 0xFF; }
  l(a) { return ((this.B[a] << 24) | (this.B[a + 1] << 16) | (this.B[a + 2] << 8) | this.B[a + 3]) | 0; }
  setL(a, v) { this.B[a] = (v >>> 24) & 0xFF; this.B[a + 1] = (v >>> 16) & 0xFF; this.B[a + 2] = (v >>> 8) & 0xFF; this.B[a + 3] = v & 0xFF; }

  mul0_93(v) { return i16((Math.imul(i16(v), DAMP)) >> 8); }

  // --- trig: $64D08 is SINE (selector 0), $64D10 is COSINE (selector $4000) ---
  sin(a) { return this.sinSel(a, 0x0000); }
  cos(a) { return this.sinSel(a, 0x4000); }
  sinSel(a, d5) {
    let d0 = a & 0xFFFF;
    let d3 = d0 & 0x3FFF;
    if (((d0 & 0x4000) ^ d5) === 0) d3 = ((d3 ^ 0x3FFF) + 1) & 0xFFFF;
    d3 = ror16(d3, 5);
    const d4 = d3 & 0x3FE;
    const s0 = this.w(0x1CA42 + d4), s1 = this.w(0x1CA42 + d4 + 2);
    const d6 = (s0 - s1) & 0xFFFF;
    const frac = ror16(d3, 1) & 0xFC00;
    const hi = ((frac * d6) >>> 16) & 0xFFFF;
    let d7 = i16((u16(s0) - hi) & 0xFFFF);
    d7 = i16(u16(d7) >>> 1);
    const sg = (d0 & d5) << 1;
    if (i16((d0 ^ sg) & 0xFFFF) < 0) d7 = -d7;
    return i16(d7);
  }

  // value * matrix[$1C230+idx*2] >> 15 ($61344)
  mtxMul(value, idx) { return i16((Math.imul(i16(value), this.w(A.Mtx + idx * 2)) << 1) >> 16); }

  // --- integrator ---
  force61ADC() {
    this.setW(A.VelX, i16(this.w(A.VelX) + this.mul0_93(this.w(A.FrcX))));
    this.setW(A.VelY, i16(this.w(A.VelY) + this.mul0_93(this.w(A.FrcY))));
    this.setW(A.VelZ, i16(this.w(A.VelZ) + this.mul0_93(this.w(A.FrcZ))));
  }
  torque61B26() {
    this.setW(A.AmR, i16(this.w(A.AmR) + this.mul0_93(this.w(A.TqR))));
    this.setW(A.AmP, i16(this.w(A.AmP) + this.mul0_93(this.w(A.TqP))));
    this.setW(A.AmY, i16(this.w(A.AmY) + this.mul0_93(this.w(A.TqY))));
  }
  integrate61950() {
    this.setL(A.PosX, (this.l(A.PosX) + (this.mul0_93(this.w(A.VelX)) << 6)) | 0);
    this.setL(A.PosY, (this.l(A.PosY) + (this.mul0_93(this.w(A.VelY)) << 7)) | 0);
    this.setL(A.PosZ, (this.l(A.PosZ) + (this.mul0_93(this.w(A.VelZ)) << 6)) | 0);
    if (this.w(A.PosY) >= 0x3E8) this.setW(A.PosY, 0x3E8);
    this.setW(A.Roll, i16(this.w(A.Roll) + this.mul0_93(this.w(A.WAmR))));
    this.clamp619E4(this.mul0_93(this.w(A.WAmY)));
  }
  clamp619E4(d0) {
    this.setW(A.Yaw, i16(this.w(A.Yaw) + d0));
    this.setW(A.Pit, i16(this.w(A.Pit) + this.mul0_93(this.w(A.WAmP))));
    let d2 = 0;
    if (i8(this.u8(0x1BB75)) < 0 && this.u8(0x1BB9A) === 0xE0) d2 = 2;
    this.clampAngle(A.Roll, A.AmR, A.angLimits, d2);
    this.clampAngle(A.Pit, A.AmY, A.angLimits, d2);
  }
  clampAngle(ang, mom, a0, d2) {
    const d3 = this.w(ang);
    let lim;
    if (d3 >= 0) { lim = this.w(a0 + d2); if (u16(lim) >= u16(d3)) return; }
    else { lim = this.w(a0 + d2 + 4); if (u16(lim) < u16(d3)) return; }
    this.setW(ang, lim);
    if ((i16(lim ^ this.w(mom))) >= 0) this.setW(mom, 0);
  }

  // --- orientation matrix + transforms ---
  mt(off) { return this.w(A.Mtx + off); }
  smt(off, v) { this.setW(A.Mtx + off, v); }
  prod(off, d5) { this.smt(off, i16((Math.imul(this.mt(off), d5) << 1) >> 16)); }
  matrix61368() {
    const sy = this.sin(this.w(A.Yaw));
    for (const o of [0x4, 0xC, 0xE, 0x14, 0x16]) this.smt(o, sy);
    const cy = this.cos(this.w(A.Yaw));
    for (const o of [0x6, 0x10, 0x12, 0x18, 0x1A]) this.smt(o, cy);
    const yh = i16(this.w(A.Yaw) - this.w(A.Hdg));
    const sh = this.sin(yh); for (const o of [0x34, 0x42, 0x44]) this.smt(o, sh);
    const ch = this.cos(yh); for (const o of [0x38, 0x3E, 0x46]) this.smt(o, ch);
    this.smt(0x8, this.sin(this.w(A.Roll)));
    const cr = this.cos(this.w(A.Roll)); for (const o of [0xA, 0x1C, 0x1E]) this.smt(o, cr);
    this.smt(0x22, this.cos(this.w(A.Pit)));
    this.smt(0x20, this.sin(this.w(A.Pit)));
    let d5 = this.mt(0x8);
    for (let o = 0xC; o <= 0x12; o += 2) this.prod(o, d5);
    for (let o = 0x34; o <= 0x38; o += 4) this.prod(o, d5);
    this.smt(0x0, this.mt(0xC)); this.smt(0x2, this.mt(0x10));
    d5 = this.mt(0xA);
    for (let o = 0x4; o <= 0x6; o += 2) this.prod(o, d5);
    for (let o = 0x44; o <= 0x46; o += 2) this.prod(o, d5);
    d5 = this.mt(0x20);
    for (let o = 0xC; o <= 0x1C; o += 4) this.prod(o, d5);
    for (let o = 0x34; o <= 0x38; o += 4) this.prod(o, d5);
    d5 = this.mt(0x22);
    for (let o = 0xE; o <= 0x1E; o += 4) this.prod(o, d5);
    for (let o = 0x3E; o <= 0x42; o += 4) this.prod(o, d5);
    this.smt(0x28, i16(this.mt(0x18) - this.mt(0xE)));
    this.smt(0x2A, i16(-this.mt(0x12) - this.mt(0x14)));
    this.smt(0x2C, i16(this.mt(0x1A) + this.mt(0xC)));
    this.smt(0x2E, i16(this.mt(0x10) - this.mt(0x16)));
    this.smt(0x30, i16(-this.mt(0x1C)));
    this.smt(0x24, i16(-this.mt(0x20)));
  }
  tmpl(off) { return this.u8(A.Tmpl + off); }
  velToBody6158C() {
    for (let d2 = 2; ; d2 -= 2) {
      let d5 = 0;
      d5 = i16(d5 + this.mtxMul(this.w(A.VelX), this.tmpl(d2 + 0)));
      d5 = i16(d5 + this.mtxMul(this.w(A.VelY), this.tmpl(d2 + 3)));
      d5 = i16(d5 + this.mtxMul(this.w(A.VelZ), this.tmpl(d2 + 6)));
      this.setW(A.BVelL + (d2 << 1), d5);
      if (d2 === 0) break;
    }
  }
  gravToBody615E6() {
    this.setW(A.GrvB, this.mtxMul(-GRAV, 0xF));
    this.setW(A.GrvC, this.mtxMul(-GRAV, 0x4));
    this.setW(A.GrvA, this.mtxMul(GRAV, 0xE));
  }
  forceToWorld61618() {
    for (let d2 = 2; ; d2 -= 1) {
      let d5 = 0;
      d5 = i16(d5 + this.mtxMul(this.w(A.BFrcA), this.tmpl(d2 + 0x9)));
      d5 = i16(d5 + this.mtxMul(this.w(A.BFrcB), this.tmpl(d2 + 0xC)));
      d5 = i16(d5 + this.mtxMul(this.w(A.BFrcC), this.tmpl(d2 + 0xF)));
      this.setW(A.FrcX + (d2 << 1), d5);
      if (d2 === 0) break;
    }
  }
  torqueToWorld61672() {
    for (let d2 = 1; ; d2 -= 1) {
      let d5 = 0;
      d5 = i16(d5 + this.mtxMul(this.w(A.AmR), this.tmpl(d2 + 0x12)));
      d5 = i16(d5 + this.mtxMul(this.w(A.AmP), this.tmpl(d2 + 0x14)));
      this.setW(A.WAmR + (d2 << 1), d5);
      if (d2 === 0) break;
    }
    this.setW(A.WAmP, i16(this.mtxMul(this.w(A.WAmY), 0x4) + this.w(A.AmY)));
  }

  // --- track-surface sample ---
  interp5C554() {
    const along = this.u8(0x1BC4D);
    const left = (Math.imul(i16(this.w(0x1BC04) - this.w(0x1BC02)), along) + (this.w(0x1BC02) << 8)) | 0;
    let d0 = (Math.imul(i16(this.w(0x1BC08) - this.w(0x1BC06)), along) + (this.w(0x1BC06) << 8)) | 0;
    const across = this.u8(0x1BC41);
    d0 = (d0 - left) | 0;
    let d4 = d0 < 0 ? -d0 : d0;
    const big = (d4 >>> 0) >= 0x8000;
    if (big) d0 = d0 >> 3;
    let p;
    if (i16(d0) < 0) { const wv = u16(-i16(d0)); p = (Math.imul(wv, across) & ~0xFF) | 0; p = (-p) | 0; }
    else { p = Math.imul(u16(i16(d0)), across) | 0; }
    if (big) p = p << 3;
    p = p >> 8;
    p = (p + left) | 0;
    this.setL(0x1BB18, p);
  }
  corners618CE() {
    let d4 = i16((this.w(0x1C26E) >> 1) - (this.w(0x1C264) >> 1));
    let d5 = i16((this.w(0x1C268) >> 1) - (this.w(0x1C272) >> 1));
    const d0 = this.w(0x1C274) >> 5, d3 = this.w(0x1C276) >> 5;
    d4 = d4 >> 5; d5 = d5 >> 5;
    this.setW(0x1BD06, i16(-d0)); this.setW(0x1BD0C, i16(-d3));
    this.setW(0x1BD02, i16(d0 - d4)); this.setW(0x1BD04, i16(d0 + d4));
    this.setW(0x1BD08, i16(d3 - d5)); this.setW(0x1BD0A, i16(d3 + d5));
  }

  // --- suspension ---
  spring6180E(delta, travel) { return i16(u16((Math.imul(i16(delta), 0x114) >> 8)) + u16(travel)); }
  spring(surf, car, comp, travel, prev, force, dmg) {
    let d0 = (this.l(surf) - this.l(car) - this.l(A.Rest)) | 0;
    this.setL(comp, d0);
    if (d0 >= 0) { if (d0 >= 0x1400) d0 = 0x1400; } else if (d0 < -0x300) d0 = -0x300;
    this.setW(travel, i16(d0));
    const d6 = i16(d0);
    const f = this.spring6180E(i16(i16(d0) - this.w(prev)), d6);
    if (f < 0) { this.setW(force, 0); this.B[0x1BB56] = 0; this.setW(prev, this.w(travel)); return; }
    const d4 = this.w(force);
    this.setW(force, f);
    if (f >= 0x400 && d4 < 0x200) this.B[A.Bottom] = (this.B[A.Bottom] + 1) & 0xFF;
    let d = i16(f - (this.u8(0x1BB01) << 8));
    if (d < 0 || d < 0x700) { this.B[0x1BB56] = 0; this.setW(prev, this.w(travel)); return; }
    if (d >= this.w(0x1BC3A)) this.setW(0x1BC3A, d);
    d = i16(d - 0x600);
    if (i8(this.u8(0x1BBCD)) >= 0) {
      this.B[0x1BB56] = (this.u8(0x1BB56) + 1) & 0xFF;
      if (i8(this.u8(0x1BB56)) < i8(this.u8(DMGLIMIT))) {
        let sev = u16(d) >>> 8;
        sev = (sev + (sev >> 1)) & 0xFF;
        let n = sev + this.u8(dmg); if (n > 0xFF) n = 0xFF;
        this.B[dmg] = n & 0xFF; this.B[A.DmgEvt] = 0x80;
      }
    }
    if (this.w(force) >= 0x1200) this.setW(force, 0x11FF);
    this.setW(prev, this.w(travel));
  }
  contactHeights61B70() {
    const sr = this.sin(this.w(A.Roll));
    this.setW(0x1BBF6, sr);
    const d0 = (this.sin(this.w(A.Pit)) << 3) | 0;
    const d3 = (sr << 4) | 0;
    this.setL(0x1BC9C, (this.l(A.PosY) - d3) >> 8);
    const d4 = (this.l(A.PosY) + d3) | 0;
    this.setL(0x1BC98, (d4 - d0) >> 8);
    this.setL(0x1BC94, (d4 + d0) >> 8);
  }
  suspension61BCC() {
    this.B[A.Bottom] = 0; this.B[0x1BC3A] = 0;
    this.spring(0x1BCA4, 0x1BC94, 0x1BCB0, 0x1BD14, 0x1BD1A, 0x1BD20, 0x1BB4F);
    this.spring(0x1BCA8, 0x1BC98, 0x1BCB4, 0x1BD16, 0x1BD1C, 0x1BD22, 0x1BB50);
    this.spring(0x1BCAC, 0x1BC9C, 0x1BCB8, 0x1BD18, 0x1BD1E, 0x1BD24, 0x1BB51);
    let d0 = i16((this.w(0x1BD20) + this.w(0x1BD22)) >> 1);
    this.setW(0x1BBF6, d0);
    d0 = i16((d0 + this.w(0x1BD24)) >> 1);
    this.setW(A.NetLift, d0);
    let dd = i16(this.w(0x1BD20) - this.w(0x1BD22));
    let t = i16((dd << 1) + dd);
    if (t < 0) t = -t; if (t >= 0x1000) t = 0x1000; if (dd < 0) t = -t;
    this.setW(A.RollTq, i16(t));
    this.setW(A.PitchTq, i16(this.w(0x1BBF6) - this.w(0x1BD24)));
    const og = this.u8(A.NetLift) | this.u8(A.NetLift + 1);
    this.B[A.OnGround] = og & 0xFF;
    if (og !== 0) return;
    if (this.u8(0x1BBDF) !== 0) return;
    let d3 = i16(-0x80);
    const roll = this.w(A.Roll);
    if (roll >= 0) { if (roll >= 0x1000) d3 = i16(-0x100); }
    else {
      switch (this.u8(0x1CA33)) {
        case 7: d3 = i16(-0x80); break;
        case 4: d3 = i16(-0x8); break;
        default: return;
      }
    }
    d3 = i16(d3 - this.w(A.PitchTq));
    if (d3 >= 0) return;
    const c = this.u8(0x1BCF0);
    if (i8(c) >= 0 || c === 0xFF) this.setW(A.PitchTq, d3);
  }

  // --- drive / tyre ---
  grip621DA() { return this.u8(A.OnGround) === 0 ? 0 : i16(this.w(A.LoadB) << 1); }
  lateralTire6217A() {
    const d4 = i16(this.w(A.GrvA) + this.w(A.LoadA));
    let d3 = i16(d4 - this.w(A.BVelL)); if (d3 < 0) d3 = -d3;
    const g = this.grip621DA();
    if (u16(d3) < u16(g)) { this.setW(A.BFrcA, i16(this.w(A.LoadA) - this.w(A.BVelL))); this.B[A.Slip] = 0; return; }
    let gg = g; if (this.w(A.BVelL) < 0) gg = -g;
    this.setW(A.BFrcA, i16(d4 - gg)); this.B[A.Slip] = 0x80;
  }
  drive620B8() {
    this.setW(A.BFrcB, i16(this.w(A.GrvB) + this.w(A.LoadB)));
    const d0b = this.u8(A.Drive) | this.u8(A.BVelV);
    if (i8(d0b) >= 0 && this.u8(0x1BD2B) !== 0) this.setW(A.Drive, i16(this.w(A.Drive) - (d0b & 0xFF)));
    let d3 = this.w(A.Drive); if (d3 < 0) d3 = -d3;
    const g = this.grip621DA();
    if (u16(d3) >= u16(g)) { this.setW(A.Drive, this.w(A.Drive) >= 0 ? g : i16(-g)); }
    this.setW(A.BFrcC, i16(this.w(A.Drive) + this.w(A.LoadC) + this.w(A.GrvC)));
    this.lateralTire6217A();
  }
  torqueApply62138() {
    let d0 = i16(this.w(A.PitchTq) - (this.w(A.AmR) >> 4));
    if (this.u8(A.OnGround) !== 0) d0 = i16(d0 + (this.w(A.BFrcC) >> 2));
    this.setW(A.TqAppR, d0);
    this.setW(A.TqAppY, i16(this.w(A.RollTq) - (this.w(A.AmY) >> 4)));
  }
  drag621F4() {
    const absw = (v) => v < 0 ? i16(-v) : v;
    let d7 = 1, d0 = 0x6000, handled = false;
    if (this.u8(A.OnGround) !== 0) {
      let s = this.u8(0x1BD46); if (i8(s) < 0) s ^= 0xFF;
      if ((s & 0xFF) >= 3 || i8(this.u8(0x1BB9C)) < 0) handled = true;
      else if (this.u8(0x1BCA2) !== 0) { d7 = 3; handled = true; }
    }
    let low = false;
    if (!handled) { if (this.u8(0x1BBDF) === 0) low = true; else d7 = 3; }
    if (low) {
      d0 = absw(this.w(0x1BD2C));
      let v = absw(this.w(0x1BD2E)); if (v > d0) d0 = v;
      v = absw(this.w(0x1BD30)); if (v > d0) d0 = v;
      d7 = 5;
      if (i8(this.u8(0x1BBC7)) < 0 && i8(this.u8(0x1BBB8)) >= 0) {
        if (u16(d0) >= 0xA00) d0 = i16(u16(d0) - 0xA00); else d0 = 0;
      }
    }
    const apply = (va, fa) => {
      const hi = i16((Math.imul(this.w(va), i16(d0)) >> 16)) >> d7;
      this.setW(fa, i16(this.w(fa) - hi));
    };
    apply(A.VelX, A.FrcX); apply(A.VelY, A.FrcY); apply(A.VelZ, A.FrcZ);
  }

  // --- load projection ($622DC) ---
  slope62424(d0w) {
    let d0 = i16(d0w); if (d0 < 0) d0 = -d0;
    let d1 = 0xFF; if (d0 < 0x100) d1 = d0 & 0xFF;
    this.B[0x1BB2B] = d1;
    this.B[0x1BB2D] = this.B[0x1EECA + (d1 >> 1)];
  }
  frac612A2(d0) {
    const d3 = ((this.u8(0x1BB1A) & 0xFF) * (d0 & 0xFF)) | 0;
    this.B[0x1BB1B] = d3 & 0xFF;
    return (d3 >> 8) & 0xFFFF;
  }
  loadProject622DC() {
    this.setW(0x1BD4A, 0);
    let d0 = ((((this.l(0x1BCB0) + this.l(0x1BCB4)) >> 1) - this.l(0x1BCB8)) | 0) >> 4;
    this.setW(0x1BD4C, i16(u16(i16(d0)) ^ 0x8000));
    this.slope62424(i16(d0));
    this.B[0x1BB2C] = this.B[0x1BB2D];
    this.B[0x1BD52] = this.B[0x1BB2B];
    d0 = (this.l(0x1BCB0) - this.l(0x1BCB4)) >> 3;
    this.setW(0x1BD48, i16(d0));
    this.slope62424(i16(d0));
    this.B[0x1BB1A] = this.B[0x1BB2C];
    this.B[0x1BD50] = this.frac612A2(this.u8(0x1BB2D)) & 0xFF;
    this.B[0x1BD4E] = this.frac612A2(this.u8(0x1BB2B)) & 0xFF;
    const proj = () => {
      let d3 = this.u8(0x1BB1A) & 0xFF;
      if (i8(this.u8(0x1BBBB)) < 0) d3 = (0 - d3) & 0xFFFF;
      d3 = (d3 << 7) & 0xFFFF;
      return i16((Math.imul(this.w(A.NetLift), i16(d3)) << 1) >> 16);
    };
    this.B[0x1BB1A] = this.B[0x1BD4E]; this.B[0x1BBBB] = this.B[0x1BD48]; this.setW(A.LoadA, proj());
    this.B[0x1BB1A] = this.B[0x1BD50]; this.B[0x1BBBB] = this.B[0x1BD4A]; this.setW(A.LoadB, proj());
    this.B[0x1BB1A] = this.B[0x1BD52]; this.B[0x1BBBB] = this.B[0x1BD4C]; this.setW(A.LoadC, proj());
  }

  // --- sound + tail housekeeping ---
  sound60FBE() {
    let d0 = this.w(0x1BD30); if (d0 < 0) d0 = -d0;
    this.setW(0x1BD5C, d0);
    if (this.u8(A.OnGround) === 0) { this.setW(0x1BC62, i16(this.w(0x1BC62) - i16(u16(this.w(0x1BC62)) >>> 2))); return; }
    if (d0 >= 0x800) {
      let s = (u16(d0) << 1) + 0x3000;
      if (s > 0xFFFF) s = 0xFF00;
      this.setW(0x1BC62, i16(u16(s)));
    } else this.setW(0x1BC62, i16(d0 << 3));
  }
  tail63E2E() {
    if (this.u8(0x63EE0) !== 0) this.B[0x63EE0] = (this.B[0x63EE0] - 1) & 0xFF;
    if (this.u8(0x1BB46) === 0) return;
    this.B[0x1BB46] = 0;
    let d0 = this.w(0x1BBEE) - this.w(0x1BD58); if (d0 < 0) d0 = 0;
    this.setW(0x1BBEE, i16(d0));
    const imp = this.w(0x1BD56) >> 4;
    this.setW(0x1BD76, i16(this.w(0x1BD76) - imp));
    this.setW(0x1BD78, i16(this.w(0x1BD78) - imp));
    this.setW(0x1BD7A, i16(this.w(0x1BD7A) - imp));
    this.setW(0x1BD40, i16(this.w(0x1BD40) + this.w(0x1BD54)));
    this.setW(0x1BD42, i16(this.w(0x1BD42) + this.w(0x1BD56)));
    this.setW(0x1BD44, i16(this.w(0x1BD44) + this.w(0x1BD58)));
    this.setW(0x1BD54, 0); this.setW(0x1BD56, 0); this.setW(0x1BD58, 0);
    if (this.u8(0x63EE0) === 0) this.B[0x63EE0] = 5;
  }

  // --- track helpers (ports of physics/track.go) ---
  handlePhys(wv) { return ((((wv << 8 | wv >>> 8) & 0xFFFF) - 0xB100) & 0xFFFF) + 0x1EF82; }
  setup5FE56(sec) {
    const p2 = this.u8(0x1C4C0 + sec);
    this.B[0x1BB79] = p2;
    this.setW(0x1BC8C, this.w(0x1EFA2 + ((p2 << 1) & 0xFF)));
    const attr = this.u8(0x1C524 + sec);
    const d2 = (attr << 1) & 0xFF;
    this.B[0x1BBDC] = (((attr >> 7) & 1) << 1) & 0xFF;
    this.setW(0x1BC90, this.w(0x1EFA2 + d2));
    this.setW(0x1BC0E, this.w(0x1C650 + sec * 2));
    this.setW(0x1BC10, this.w(0x1C718 + sec * 2));
    const typ = this.u8(0x1C5EC + sec);
    this.B[0x1BC4A] = typ & 0xC0;
    this.B[0x1BC32] = ((typ & 0x10) << 3) & 0xFF;
    const nib = typ & 0x0F;
    this.B[0x1BB86] = nib;
    this.setW(0x1BCBC, this.w(0x1EF82 + nib * 2));
    const a0 = this.handlePhys(u16(this.w(0x1BCBC)));
    this.B[0x1BB4D] = this.u8(a0 + 1);
    const off = this.u8(a0);
    const cnt = this.u8(a0 + off);
    let d2i = off + 1;
    this.B[0x1BB97] = cnt;
    this.B[0x1BB59] = (cnt << 1) & 0xFF;
    const cm2 = (cnt - 2) & 0xFF;
    this.B[0x1BB98] = cm2;
    this.B[0x1BB5A] = (cm2 << 1) & 0xFF;
    this.B[0x1BB6A] = (((cnt >> 1) - 1)) & 0xFF;
    let v = this.u8(a0 + d2i); d2i++;
    v = ((v >> 1) | ((v & 1) << 7)) & 0xFF;
    this.B[0x1BC44] = v & 0x80;
    this.B[0x1BB7B] = this.u8(a0 + d2i); d2i++;
    this.B[0x1BBD9] = this.u8(a0 + d2i); d2i += 3;
    this.B[0x1BBD4] = this.u8(a0 + d2i);
  }
  railHeight5C0AA(a4, a5, d1) {
    const p2 = this.u8(0x1BB79);
    const b650 = this.w(0x1BC0E), b718 = this.w(0x1BC10);
    let d2 = d1;
    if (i8(p2) >= 0) {
      const odd = d2 & 1; d2 >>= 1;
      let d0;
      if (odd) { const v = this.u8(a5 + d2); d0 = (((v << 1) & 0xE0) | ((v & 0xF) << 8)) + b718; }
      else { const v = this.u8(a4 + d2); d0 = (((v << 1) & 0xE0) | ((v & 0xF) << 8)) + b650; }
      return i16(d0 & 0xFFFF) >> 5;
    }
    d2 &= ~1;
    if (d1 & 1) { const d3 = this.u8(a5 + d2 + 1); const d0 = ((this.u8(a5 + d2) & 0x7F) << 8 | d3) + b718; return i16(d0 & 0xFFFF) >> 5; }
    const d3 = this.u8(a4 + d2 + 1); const d0 = ((this.u8(a4 + d2) & 0x7F) << 8 | d3) + b650; return i16(d0 & 0xFFFF) >> 5;
  }
  secAdvance5C51A() { let d1 = this.u8(0x1BB85) + 1; if (d1 >= this.u8(0x1CA1A)) d1 = 0; this.B[0x1BB85] = d1 & 0xFF; }
  secRetreat5C538() { let d1 = i8(this.u8(0x1BB85)) - 1; if (d1 < 0) d1 = i8(this.u8(0x1CA1A)) - 1; this.B[0x1BB85] = d1 & 0xFF; }
  edgeCross5C484() {
    let dc;
    if (i8((this.u8(0x1BC40) ^ this.u8(0x1BC32)) & 0xFF) >= 0) {
      this.secAdvance5C51A(); this.setup5FE56(this.u8(0x1BB85)); dc = i8(this.u8(0x1BC32)) < 0;
    } else {
      this.secRetreat5C538(); this.setup5FE56(this.u8(0x1BB85)); dc = i8(this.u8(0x1BC32)) >= 0;
    }
    if (dc) { this.B[0x1BBA3] = (this.u8(0x1BB97) - 4) & 0xFF; if (i8(this.u8(0x1BC40)) < 0) return; }
    else { this.B[0x1BBA3] = 0; if (i8(this.u8(0x1BC40)) >= 0) return; }
    let vv = (256 - this.u8(0x1BC41)) & 0xFF; this.B[0x1BC41] = vv !== 0 ? vv : 0xFF;
    vv = (256 - this.u8(0x1BC4D)) & 0xFF; this.B[0x1BC4D] = vv !== 0 ? vv : 0xFF;
  }
  offTrack5C872() { this.setL(0x1BB18, 0x1000); this.B[0x1BB9A] = ((this.u8(0x1BB9A) >> 1) | 0x80) & 0xFF; }
  offEdge5C808() {
    const v = this.w(0x1BC22);
    let d0;
    if (v < 0) d0 = -v; else { d0 = 0x180 - v; if (d0 < 0) d0 = -d0; }
    if (d0 > 0x30) { this.offTrack5C872(); return; }
    d0 = (d0 & 0xFF) << 4;
    const d3 = (this.l(0x1BB18) - d0 - 0x100) | 0;
    if (d3 < 0x1000) { this.offTrack5C872(); return; }
    this.setL(0x1BB18, d3);
    if (((this.u8(0x1BC22) ^ this.u8(0x1BC32)) & 0x80) !== 0) this.B[0x1BBDA] = 0x80; else this.B[0x1BBDA] = 0x40;
  }
  store5C5F2(d1) {
    this.interp5C554();
    const off = d1 << 1;
    if ((this.u8(0x1BB65) & 0x80) !== 0) { this.B[0x1BB65] &= ~0x80; this.offEdge5C808(); }
    else this.B[0x1BB65] &= ~0x80;
    if (i8(this.u8(0x1BD5C)) >= 0x0A) { this.setL(0x1BCA4 + off, this.l(0x1BB18)); return; }
    let r = this.u8(0x1BCE4); if (i8(r) < 0) r = (256 - r) & 0xFF;
    if (i8(r & 0xFF) > 5) { this.setL(0x1BCA4 + off, this.l(0x1BB18)); return; }
    const a = this.l(0x1BCA4 + off) >>> 0, b = this.l(0x1BB18) >>> 0;
    const sum = (a + b) >>> 0;
    const carry = (a + b) > 0xFFFFFFFF ? 1 : 0;
    this.setL(0x1BCA4 + off, ((carry << 31) | (sum >>> 1)) | 0);
  }
  mulSwap(d0, d3) { return i16((Math.imul(i16(d0), i16(d3)) << 1) >> 16); }
  surface5C1D0() {
    let sec = this.u8(0x1BB1C);
    this.B[0x1BB85] = sec;
    this.setup5FE56(sec);
    this.B[0x1BB9A] = 0;
    let d1 = 4;
    for (;;) {
      this.B[0x1BBF9] = d1 & 0xFF;
      if (this.u8(0x1BB1C) !== this.u8(0x1BB85)) {
        const s = this.u8(0x1BB1C); this.B[0x1BB85] = s; this.setup5FE56(s); d1 = this.u8(0x1BBF9);
      }
      this.B[0x1BB1A] = this.u8(0x1BB7B);
      const pos = ((this.w(0x1BD02 + d1) >> 4) + this.w(0x1BC5E)) | 0;
      let d0;
      if ((u16(pos) >>> 0) >= 0x180) {
        this.B[0x1BB65] |= 0x80; this.setW(0x1BC22, i16(pos));
        d0 = i16(pos) < 0 ? 0 : 0xFF;
      } else {
        let a = i16(pos); if (a < 0) a = -a;
        d0 = this.mulSwap(a, (this.u8(0x1BB1A) << 7) & 0x7FFF);
        if (d0 >= 0x100) d0 = this.u8(0xFF);
      }
      this.B[0x1BC4D] = d0 & 0xFF;
      if (i8(this.u8(0x1BC32)) < 0) d0 = (d0 ^ 0xFF) & 0xFF;
      if (d1 === 4) this.B[0x1BBA1] = d0 & 0xFF;
      this.B[0x1BB1A] = this.u8(0x1BBD9);
      const ac = (this.mulSwap(this.w(0x1BD08 + d1) >> 3, (this.u8(0x1BB1A) << 7) & 0x7FFF) + this.w(0x1BB10)) | 0;
      this.setW(0x1BC40, i16(ac));
      const e = (this.u8(0x1BC40) << 1) & 0xFF;
      this.B[0x1BBA3] = e;
      if (i8(e) < 0 || i8(e) >= i8(this.u8(0x1BB98))) this.edgeCross5C484();
      const a4 = this.handlePhys(u16(this.w(0x1BC8C)));
      const a5 = this.handlePhys(u16(this.w(0x1BC90)));
      if (i8(this.u8(0x1BC32)) < 0) {
        const vi = (this.u8(0x1BB97) - this.u8(0x1BBA3) - 4) & 0xFF;
        this.setW(0x1BC08, this.railHeight5C0AA(a4, a5, vi));
        this.setW(0x1BC06, this.railHeight5C0AA(a4, a5, (vi + 1) & 0xFF));
        this.setW(0x1BC04, this.railHeight5C0AA(a4, a5, (vi + 2) & 0xFF));
        this.setW(0x1BC02, this.railHeight5C0AA(a4, a5, (vi + 3) & 0xFF));
      } else {
        const vi = this.u8(0x1BBA3);
        this.setW(0x1BC02, this.railHeight5C0AA(a4, a5, vi));
        this.setW(0x1BC04, this.railHeight5C0AA(a4, a5, (vi + 1) & 0xFF));
        this.setW(0x1BC06, this.railHeight5C0AA(a4, a5, (vi + 2) & 0xFF));
        this.setW(0x1BC08, this.railHeight5C0AA(a4, a5, (vi + 3) & 0xFF));
      }
      this.store5C5F2(this.u8(0x1BBF9));
      d1 = i8(this.u8(0x1BBF9)) - 2;
      if (d1 < 0) break;
    }
  }
  sectionLocate61012() {
    let d0 = 0, d4 = 0, dd0 = 0;
    const sec = this.u8(0x1BB1C);
    this.B[0x1BB85] = sec; this.setup5FE56(sec);
    const d3w = this.w(0x1BC32);
    d4 = i16(u16(this.w(0x1BD5A) - this.w(A.Yaw)) ^ u16(d3w));
    let d2 = 0;
    if (i8(this.u8(0x1BB4D)) < 0) { d2 += 2; if (i16(u16(this.w(0x1BC44)) ^ u16(d3w)) < 0) d2 += 2; }
    d4 = i16(d4 + this.w(0x6125A + d2));
    d0 = d4; if (i16(d0) < 0) d0 = -i16(d0);
    this.setW(0x1BC2A, i16(d0)); this.setW(0x1BBF6, i16(d4));
    if (u16(d0) >= 0x800) d0 = 0x7FFF; else d0 = i16(i16(d0) << 4);
    this.setW(0x1BC3C, i16(d0));
    if (((this.u8(0x1BB6A) - this.u8(0x1BB0A)) & 0xFF) < 2) { this.secAdvance5C51A(); this.setup5FE56(this.u8(0x1BB85)); }
    this.B[0x1BB5D] = (this.u8(0x1BC44) ^ this.u8(0x1BC32)) & 0xFF;
    let label;
    if (this.u8(0x1BBC6) === 0) label = 'branch3C';
    else {
      this.B[0x1BB1B] = (this.u8(0x1BBC6) ^ this.u8(0x1BBF6)) & 0xFF;
      if (i8(this.u8(0x1BB4D)) < 0) {
        if (i8((this.u8(0x1BBC6) ^ this.u8(0x1BB5D)) & 0xFF) < 0) {
          this.B[0x1BBC6] = this.u8(0x1BB5D); dd0 = (this.u8(0x1BBD4) - 0x23) & 0xFF; label = 'label21C';
        } else { dd0 = (this.u8(0x1BBD4) + 0x2D) & 0xFF; label = 'label126'; }
      } else { dd0 = this.u8(0x1BBD4); label = 'label126'; }
    }
    for (;;) {
      if (label === 'label126') {
        if (i8(this.u8(0x1BB1B)) >= 0) dd0 = (dd0 + this.u8(0x1BC3C)) & 0xFF;
        label = 'label21C';
      } else if (label === 'branch3C') {
        d4 = 0;
        if (i8(this.u8(0x1BB4D)) < 0) { this.B[0x1BBC6] = this.u8(0x1BB5D); dd0 = this.u8(0x1BBD4); label = 'label21C'; }
        else label = 'label160';
      } else if (label === 'label160') {
        this.B[0x1BBC6] = this.u8(0x1BBF6);
        let d2b = this.w(0x1BC2A) & 0xFF;
        let dv = this.w(0x1BC2A);
        let toLabel1C2 = false;
        if (this.u8(0x1BC2A) !== 0) {
          dv = i16(dv - 0x1E00);
          if (i16(dv) >= 0) { dd0 = dv; label = 'label1C2'; toLabel1C2 = true; }
          else d2b = 0xFF;
        }
        if (!toLabel1C2) {
          this.B[0x1BB1A] = d2b & 0xFF;
          let v = this.w(0x1BD30); if (i16(v) < 0) v = -i16(v);
          v = i16(v + 0xA00); if (i16(v) < 0) v = 0x7F00;
          const d3b = (this.u8(0x1BB1A) << 7) & 0x7FFF;
          v = i16((Math.imul(i16(v), i16(d3b)) << 1) >> 16);
          v = u16(v) >>> 7; if ((v & 0xFF) === 0) v++;
          dd0 = v; label = 'label1C2';
        }
      } else if (label === 'label1C2') {
        if (i8(this.u8(0x1BBF6)) < 0) dd0 = i16(-i16(dd0));
        this.setW(A.Yaw, i16(this.w(A.Yaw) + i16(dd0)));
        d4 = i16(d4 - this.w(A.AmP));
        if ((this.l(0x64AEC) >>> 0) !== (MAGIC >>> 0) || this.u8(A.OnGround) === 0) d4 = 0;
        this.setW(A.TqP, i16(d4)); return;
      } else if (label === 'label21C') {
        this.B[0x1BB1A] = dd0 & 0xFF;
        let v = i16((Math.imul(this.w(0x1BD30), i16((this.u8(0x1BB1A) << 7) & 0x7FFF)) << 1) >> 16);
        if (i8(this.u8(0x1BBC6)) < 0) v = i16(-i16(v));
        d4 = i16(v) >> 3;
        if ((this.u8(0x1BC2A) & 0xFF) < 0x1E) {
          d4 = i16(d4 - this.w(A.AmP));
          if ((this.l(0x64AEC) >>> 0) !== (MAGIC >>> 0) || this.u8(A.OnGround) === 0) d4 = 0;
          this.setW(A.TqP, i16(d4)); return;
        }
        label = 'label160';
      }
    }
  }

  // --- the full frame ($6185C) ---
  frame6185C() {
    this.matrix61368();
    this.corners618CE();
    this.surface5C1D0();
    this.contactHeights61B70();
    this.velToBody6158C();
    this.sound60FBE();
    this.gravToBody615E6();
    this.suspension61BCC();
    this.loadProject622DC();
    this.setW(0x1BD46, this.w(0x1BD44));
    this.tail63E2E();
    if (this.u8(0x620B6) !== 0) this.B[0x620B6] = (this.B[0x620B6] - 1) & 0xFF;
    if (this.u8(0x1BB72) !== 0) {
      this.drive620B8();
      this.sectionLocate61012();
      this.forceToWorld61618();
      this.drag621F4();
      this.torqueApply62138();
      this.torque61B26();
      this.torqueToWorld61672();
    }
    this.force61ADC();
    this.integrate61950();
  }

  // self-check against the Go golden trace; returns the first mismatching frame or -1.
  selfTest(trace) {
    const L = ['Drive', 'PosX', 'PosY', 'PosZ', 'Roll', 'Yaw', 'Pit', 'VelX', 'VelY', 'VelZ'];
    for (let f = 0; f < trace.frames.length; f++) {
      const fr = trace.frames[f];
      this.setW(A.Drive, fr[0]);
      this.frame6185C();
      const got = [fr[0], this.l(A.PosX), this.l(A.PosY), this.l(A.PosZ),
        this.w(A.Roll), this.w(A.Yaw), this.w(A.Pit), this.w(A.VelX), this.w(A.VelY), this.w(A.VelZ)];
      for (let k = 1; k < got.length; k++) {
        if ((got[k] | 0) !== (fr[k] | 0)) {
          console.warn(`physics selfTest: frame ${f} ${L[k]} js=${got[k]} go=${fr[k]}`);
          return f;
        }
      }
    }
    return -1;
  }
}
