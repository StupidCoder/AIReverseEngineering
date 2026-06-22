// Stunt Car Racer — track ribbon viewer. The geometry is entirely the engine's own,
// decoded purely from the disk in Go (package track, Part IV) and verified against the
// original on our m68k core. Per section we use: the 16x16 grid plan anchor; the exact
// per-rung left/right rail-HEIGHT profile (the engine's vertex builder $5C0AA, verified
// by cmd/geomoracle) — flat, ramp, hill or hard jump edge, whatever the data says, with
// no heuristic deciding step-vs-slope; and the exact local plan OUTLINE (the (x,z) vertex
// pairs $5C6C4 reads from the piece-shape, verified by cmd/planoracle) — so straights are
// straight and curve pieces carry their real arc. We similarity-fit each outline onto its
// grid-anchor segment, lift each rung by its rail heights (their difference is the real
// camber), and render a hidden-line wireframe (invisible depth fill + colour LineSegments,
// the Marble Madness slope-viewer technique).
import * as THREE from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';
import { Physics } from './physics.js';

export class TrackViewer {
  constructor(el) {
    this.el = el;
    const renderer = new THREE.WebGLRenderer({ antialias: true, preserveDrawingBuffer: true });
    renderer.setPixelRatio(Math.min(devicePixelRatio, 2));
    renderer.setClearColor(0x0a0d12, 1);
    el.appendChild(renderer.domElement);

    const scene = new THREE.Scene();
    const camera = new THREE.PerspectiveCamera(45, 1, 0.1, 100);
    const controls = new OrbitControls(camera, renderer.domElement);
    controls.enableDamping = true;
    controls.dampingFactor = 0.08;

    this.three = { renderer, scene, camera, controls, group: null };
    this.drive = null; // set while driving
    this.keys = {};
    window.addEventListener('keydown', (e) => { if (this.drive) { this.keys[e.key.toLowerCase()] = true; if ('wasd '.includes(e.key.toLowerCase())) e.preventDefault(); } });
    window.addEventListener('keyup', (e) => { this.keys[e.key.toLowerCase()] = false; });
    this._resize();
    window.addEventListener('resize', () => this._resize());

    let last = performance.now();
    const tick = (now) => {
      requestAnimationFrame(tick);
      const dt = Math.min(0.05, (now - last) / 1000); last = now;
      if (this.drive) this._driveStep(dt);
      else controls.update();
      renderer.render(scene, camera);
    };
    tick(last);
  }

  _resize() {
    const w = this.el.clientWidth, h = this.el.clientHeight || Math.round(w * 0.62);
    const { renderer, camera } = this.three;
    renderer.setSize(w, h, false);
    camera.aspect = w / h;
    camera.updateProjectionMatrix();
  }

  // track: { name, nodes:[[x,z,type,p1,p2,attr],...] }; id = 0..7
  show(track, id) {
    if (this.drive) this.exitDrive();
    this.trackId = id;
    const t = this.three;
    if (t.group) { t.scene.remove(t.group); disposeGroup(t.group); }
    const group = new THREE.Group();

    // Section plan anchors: n[0],n[1] = the section's cell on the 16x16 track grid.
    // The surface is NOT one height per section — each section carries an exact per-rung
    // rail-height profile (track.profiles[i] = [HeightL[], HeightR[]], the engine's
    // vertex builder $5C0AA reproduced exactly and verified vs the original). Those
    // profiles ARE the surface: a smooth run of values is a drivable ramp/hill, a sudden
    // jump is a hard edge (a Big Ramp jump lip, a Stepping-Stone gap). No heuristic
    // decides step-vs-slope — the data does.
    const A = track.nodes.map(nn => ({ x: nn[0], z: nn[1] }));
    const n = A.length;
    let minX = Infinity, maxX = -Infinity, minZ = Infinity, maxZ = -Infinity;
    for (const p of A) { minX = Math.min(minX, p.x); maxX = Math.max(maxX, p.x); minZ = Math.min(minZ, p.z); maxZ = Math.max(maxZ, p.z); }
    const cx = (minX + maxX) / 2, cz = (minZ + maxZ) / 2;
    const span = Math.max(maxX - minX, maxZ - minZ) || 1;
    const S = 8 / span; // fit the plan into ~8 units

    // Fixed rail-height -> grid-unit scale, shared across all tracks so relative relief
    // is honest: the Roller Coaster and Ski Jump (rail heights ~900) really do tower over
    // the gentle circuits (~250). Heights sit on the ground via the per-track minimum.
    const HK = 1 / 150;
    let minH = Infinity;
    for (const pr of track.profiles) for (const h of pr[0]) minH = Math.min(minH, h);
    for (const pr of track.profiles) for (const h of pr[1]) minH = Math.min(minH, h);

    const Ai = i => A[((i % n) + n) % n];
    const SX = p => (p.x - cx) * S, SZ = p => (p.z - cz) * S;

    // Build the ribbon by similarity-fitting each section's EXACT local plan outline
    // (track.outlines[i] = [ [[Lx,Lz]..], [[Rx,Rz]..] ], the (x,z) vertex pairs $5C6C4
    // reads from the piece-shape) onto its grid-anchor segment. We map the outline's
    // local centreline start -> A[i] and end -> A[i+1] with a rotation + uniform scale
    // (one complex multiply, w = targetChord / localChord), so straights stay straight,
    // curve pieces carry their real arc, and consecutive sections share the anchor (the
    // ribbon is continuous). The two rails come straight from the outline — their width
    // and any asymmetry are exact; heights come from the profile (their difference is the
    // real camber). No spline, no nominal width: this is the engine's own geometry.
    const rings = [];
    for (let i = 0; i < n; i++) {
      const Lp = track.outlines[i][0], Rp = track.outlines[i][1];
      const pr = track.profiles[i], HL = pr[0], HR = pr[1];
      const rungs = Math.min(Lp.length, Rp.length, HL.length, HR.length);
      if (rungs === 0) continue;
      const c0x = (Lp[0][0] + Rp[0][0]) / 2, c0z = (Lp[0][1] + Rp[0][1]) / 2;
      const cEx = (Lp[rungs - 1][0] + Rp[rungs - 1][0]) / 2, cEz = (Lp[rungs - 1][1] + Rp[rungs - 1][1]) / 2;
      const a = Ai(i), b = Ai(i + 1);
      const ux = cEx - c0x, uz = cEz - c0z;          // local chord (complex u)
      const vx = b.x - a.x, vz = b.z - a.z;          // target grid chord (complex v)
      const inv = 1 / (ux * ux + uz * uz || 1);
      const wr = (vx * ux + vz * uz) * inv;          // w = v / u : real (scale*cos)
      const wi = (vz * ux - vx * uz) * inv;          //            imag (scale*sin)
      const tf = (px, pz) => {
        const dx = px - c0x, dz = pz - c0z;
        return { x: a.x + dx * wr - dz * wi, z: a.z + dx * wi + dz * wr };
      };
      for (let j = 0; j < rungs; j++) {
        const wl = tf(Lp[j][0], Lp[j][1]), wright = tf(Rp[j][0], Rp[j][1]);
        rings.push({
          l: { x: SX(wl), z: SZ(wl) }, r: { x: SX(wright), z: SZ(wright) },
          hl: (HL[j] - minH) * HK * S, hr: (HR[j] - minH) * HK * S,
          sec: i, // the game section this ring belongs to
        });
      }
    }
    const m = rings.length;
    this.ribbon = { rings, m }; // for the drive mode
    const V = (p, y) => new THREE.Vector3(p.x, y, p.z);

    // Invisible depth fill (the ribbon surface) for hidden-line removal.
    const fpos = [];
    const quad = (a, b, c, d) => fpos.push(a.x, a.y, a.z, b.x, b.y, b.z, c.x, c.y, c.z, b.x, b.y, b.z, d.x, d.y, d.z, c.x, c.y, c.z);
    for (let k = 0; k < m; k++) {
      const a = rings[k], b = rings[(k + 1) % m];
      quad(V(a.l, a.hl), V(a.r, a.hr), V(b.l, b.hl), V(b.r, b.hr));
    }
    const fgeom = new THREE.BufferGeometry();
    fgeom.setAttribute('position', new THREE.Float32BufferAttribute(fpos, 3));
    const fill = new THREE.Mesh(fgeom, new THREE.MeshBasicMaterial({
      colorWrite: false, polygonOffset: true, polygonOffsetFactor: 1, polygonOffsetUnits: 1,
      side: THREE.DoubleSide,
    }));
    group.add(fill);

    // Wireframe: the two rails (coloured along the lap) + a rung every few rings.
    const lpos = [], lcol = [];
    const col = new THREE.Color();
    const edge = (p, q, f) => {
      col.setHSL(0.58 - 0.5 * f, 0.85, 0.55);
      lpos.push(p.x, p.y, p.z, q.x, q.y, q.z);
      lcol.push(col.r, col.g, col.b, col.r, col.g, col.b);
    };
    for (let k = 0; k < m; k++) {
      const a = rings[k], b = rings[(k + 1) % m], f = k / m;
      edge(V(a.l, a.hl), V(b.l, b.hl), f); // left rail
      edge(V(a.r, a.hr), V(b.r, b.hr), f); // right rail
      if (k % 2 === 0) edge(V(a.l, a.hl), V(a.r, a.hr), f); // rung
    }
    const lgeom = new THREE.BufferGeometry();
    lgeom.setAttribute('position', new THREE.Float32BufferAttribute(lpos, 3));
    lgeom.setAttribute('color', new THREE.Float32BufferAttribute(lcol, 3));
    group.add(new THREE.LineSegments(lgeom, new THREE.LineBasicMaterial({ vertexColors: true })));

    // Support columns down to the ground (y=0), like the game's preview.
    const cpos = [];
    for (let k = 0; k < m; k += 3) {
      const a = rings[k];
      const mx = (a.l.x + a.r.x) / 2, mz = (a.l.z + a.r.z) / 2, my = (a.hl + a.hr) / 2;
      if (my > 0.02) cpos.push(mx, my, mz, mx, 0, mz);
    }
    if (cpos.length) {
      const cg = new THREE.BufferGeometry();
      cg.setAttribute('position', new THREE.Float32BufferAttribute(cpos, 3));
      group.add(new THREE.LineSegments(cg, new THREE.LineBasicMaterial({ color: 0x6b4a3a })));
    }

    // Start/finish marker (green) at ring 0.
    const r0 = rings[0];
    const sm = new THREE.Mesh(new THREE.SphereGeometry(0.12, 12, 12), new THREE.MeshBasicMaterial({ color: 0x35d07f }));
    sm.position.set((r0.l.x + r0.r.x) / 2, (r0.hl + r0.hr) / 2, (r0.l.z + r0.r.z) / 2);
    group.add(sm);

    t.scene.add(group);
    t.group = group;

    // Frame it from a raised 3/4 angle so both the circuit plan and the elevation read.
    const cam = t.camera, ctrl = t.controls;
    ctrl.target.set(0, 0.5, 0);
    cam.position.set(2.5, 5, 8.5);
    cam.near = 0.1; cam.far = 100; cam.updateProjectionMatrix();
    ctrl.update();
  }

  // --- drive mode: run the verified physics (package physics, ported to physics.js) and
  // steer the car along the rendered ribbon with WASD. The physics provides the speed,
  // suspension bounce and roll/pitch; progress maps it onto the decoded track. ---
  async enterDrive(hud) {
    if (!this.ribbon || this.drive) return;
    const id = this.trackId;
    const [stat, init, traceR] = await Promise.all([
      fetch('public/stuntcar/phys/static.bin').then(r => r.arrayBuffer()),
      fetch(`public/stuntcar/phys/${id}.bin`).then(r => r.arrayBuffer()),
      fetch(`public/stuntcar/phys/${id}.trace.json`).then(r => r.json()),
    ]);
    // verify the JS port against the Go golden trace on a throwaway copy.
    const check = new Physics(); check.loadTrack(init, stat);
    const fail = check.selfTest(traceR);
    const phys = new Physics(); phys.loadTrack(init, stat);
    phys.B[0x1BB72] = 0x80;                       // arm the race (grounded drive block)
    phys.placeCar605B6(this.ribbon.rings[0].sec); // real start placement (local frame, posY=16)

    const car = this._buildCar();
    this.three.scene.add(car);
    this.drive = {
      phys, car, rings: this.ribbon.rings, m: this.ribbon.m,
      progress: 0, lateral: 0, throttle: 0, acc: 0, speed: 0, hud,
      verdict: fail < 0 ? `physics verified exact (${traceR.frames.length} frames)` : `selftest diverged at frame ${fail}`,
    };
    if (hud) hud.style.display = 'block';
  }

  exitDrive() {
    if (!this.drive) return;
    this.three.scene.remove(this.drive.car); disposeGroup(this.drive.car);
    if (this.drive.hud) this.drive.hud.style.display = 'none';
    this.drive = null;
    const { camera, controls } = this.three;
    controls.target.set(0, 0.5, 0); camera.position.set(2.5, 5, 8.5); controls.update();
  }

  _buildCar() {
    const g = new THREE.Group();
    const body = new THREE.Mesh(new THREE.BoxGeometry(0.34, 0.12, 0.6),
      new THREE.MeshBasicMaterial({ color: 0xe23b3b }));
    body.position.y = 0.1; g.add(body);
    const cab = new THREE.Mesh(new THREE.BoxGeometry(0.26, 0.1, 0.26),
      new THREE.MeshBasicMaterial({ color: 0xffd23b }));
    cab.position.set(0, 0.19, -0.04); g.add(cab);
    const wheelGeo = new THREE.CylinderGeometry(0.07, 0.07, 0.06, 10);
    const wheelMat = new THREE.MeshBasicMaterial({ color: 0x222428 });
    for (const [x, z] of [[-0.18, 0.2], [0.18, 0.2], [-0.18, -0.2], [0.18, -0.2]]) {
      const wm = new THREE.Mesh(wheelGeo, wheelMat);
      wm.rotation.z = Math.PI / 2; wm.position.set(x, 0.04, z); g.add(wm);
    }
    return g;
  }

  _driveStep(dt) {
    const d = this.drive, k = this.keys;
    // The original physics is FIXED-TIMESTEP, not framerate-independent: each $6185C
    // advances the sim by one tick and the constants bake the step in (the 0.93 damping,
    // the <<6/<<7 velocity->position scales). So we run it at a fixed rate decoupled from
    // the display via an accumulator -- one tick = one game frame (Amiga PAL VBlank, 50 Hz)
    // -- never scaled by the render dt. The golden-trace check is per-tick, so exactness is
    // independent of wall-clock rate; the 50 Hz only sets how fast the car feels.
    const STEP = 1 / 50; // Amiga PAL frame
    d.acc = Math.min(d.acc + dt, 0.2); // clamp so a stalled tab can't spiral
    while (d.acc >= STEP) {
      d.acc -= STEP;
      if (k['w']) d.throttle = Math.min(0x3800, d.throttle + 0x300);
      else if (k['s']) d.throttle = Math.max(-0x2000, d.throttle - 0x400);
      else d.throttle = Math.trunc(d.throttle * 0.92);
      // The exact drive/grip/drag/suspension model with the REAL render coupling: the car
      // is placed in the local track frame and the suspension samples the real surface under
      // the wheels for the section it's on (fed from the ribbon), so it rides the actual
      // ramps and bumps. Returns the throttle-responsive world speed to advance the ribbon.
      const p = ((d.progress % d.m) + d.m) % d.m;
      const sec = d.rings[Math.floor(p) % d.m].sec;
      d.speed = d.phys.driveTickCoupled(d.throttle | 0, sec);
      d.progress += d.speed * 1e-5;
      const steer = (k['d'] ? 1 : 0) - (k['a'] ? 1 : 0);
      d.lateral = Math.max(-1, Math.min(1, d.lateral * 0.86 + steer * 0.05));
    }
    this._placeCar();
    if (d.hud) {
      const dmg = Math.max(d.phys.u8(0x1BB4F), d.phys.u8(0x1BB50), d.phys.u8(0x1BB51));
      d.hud.textContent = `${d.verdict}  ·  speed ${d.speed | 0}  ·  damage ${(dmg / 255 * 100) | 0}%  ·  W/S throttle, A/D steer`;
    }
  }

  _placeCar() {
    const d = this.drive, rings = d.rings, m = d.m;
    const ctr = (r) => ({ x: (r.l.x + r.r.x) / 2, y: (r.hl + r.hr) / 2, z: (r.l.z + r.r.z) / 2 });
    const p = ((d.progress % m) + m) % m;
    const i0 = Math.floor(p), frac = p - i0;
    const a = rings[i0 % m], b = rings[(i0 + 1) % m];
    const ca = ctr(a), cb = ctr(b);
    const cx = ca.x + (cb.x - ca.x) * frac, cy = ca.y + (cb.y - ca.y) * frac, cz = ca.z + (cb.z - ca.z) * frac;
    let tx = cb.x - ca.x, tz = cb.z - ca.z; const tl = Math.hypot(tx, tz) || 1; tx /= tl; tz /= tl;
    const nx = -tz, nz = tx;
    const halfW = Math.hypot(a.r.x - a.l.x, a.r.z - a.l.z) / 2 || 0.2;
    const ox = cx + nx * d.lateral * halfW, oz = cz + nz * d.lateral * halfW;
    d.car.position.set(ox, cy + 0.06, oz);
    // bank with the road (rail-height difference) + a little into the turn; subtle pitch.
    const bank = Math.atan2(a.hr - a.hl, halfW * 2 || 1);
    const pit = d.phys.w(0x1BCE8) * (Math.PI * 2 / 65536);
    const yaw = Math.atan2(tx, tz) + d.lateral * 0.25;
    d.car.rotation.set(0, 0, 0);
    d.car.rotateY(yaw); d.car.rotateX(-pit * 0.4); d.car.rotateZ(-bank - d.lateral * 0.15);
    const cam = this.three.camera;
    cam.position.set(ox - tx * 1.7, cy + 0.95, oz - tz * 1.7);
    cam.lookAt(ox + tx * 0.6, cy + 0.18, oz + tz * 0.6);
  }
}

function disposeGroup(g) {
  g.traverse(o => { if (o.geometry) o.geometry.dispose(); if (o.material) o.material.dispose(); });
}
