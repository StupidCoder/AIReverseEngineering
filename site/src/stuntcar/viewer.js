// Stunt Car Racer — track ribbon viewer. The geometry is the verified plan-view
// spine (package track, Part IV §5): the section grid plan plus, for each section, the
// exact per-rung left/right rail-height profile — the engine's vertex builder $5C0AA
// reproduced in Go and verified coordinate-exact against the original (cmd/geomoracle).
// Decoded purely from the disk and exported to tracks.json. We lay the rungs along a
// spline through the section grid cells, lift each by its rail heights (their difference
// is the real camber), and render the result as a hidden-line wireframe (invisible depth
// fill + colour LineSegments, the same technique as the Marble Madness slope viewer).
// The surface — flat, ramp, hill or hard jump edge — is whatever the profile data says;
// no heuristic decides step-vs-slope.
import * as THREE from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';

const WIDTH = 0.32; // nominal half-width of the track ribbon, in grid-cell units

export class TrackViewer {
  constructor(el) {
    this.el = el;
    const renderer = new THREE.WebGLRenderer({ antialias: true });
    renderer.setPixelRatio(Math.min(devicePixelRatio, 2));
    renderer.setClearColor(0x0a0d12, 1);
    el.appendChild(renderer.domElement);

    const scene = new THREE.Scene();
    const camera = new THREE.PerspectiveCamera(45, 1, 0.1, 100);
    const controls = new OrbitControls(camera, renderer.domElement);
    controls.enableDamping = true;
    controls.dampingFactor = 0.08;

    this.three = { renderer, scene, camera, controls, group: null };
    this._resize();
    window.addEventListener('resize', () => this._resize());

    const tick = () => {
      requestAnimationFrame(tick);
      controls.update();
      renderer.render(scene, camera);
    };
    tick();
  }

  _resize() {
    const w = this.el.clientWidth, h = this.el.clientHeight || Math.round(w * 0.62);
    const { renderer, camera } = this.three;
    renderer.setSize(w, h, false);
    camera.aspect = w / h;
    camera.updateProjectionMatrix();
  }

  // track: { name, nodes:[[x,z,type,p1,p2,attr],...] }
  show(track) {
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

    const cm = (a, b, c, d, t) => { const u = t * t, w = u * t; return 0.5 * (2 * b + (-a + c) * t + (2 * a - 5 * b + 4 * c - d) * u + (-a + 3 * b - 3 * c + d) * w); };
    const cmPt = (P, Q, R2, T, t) => ({ x: cm(P.x, Q.x, R2.x, T.x, t), z: cm(P.z, Q.z, R2.z, T.z, t) });
    const Ai = i => A[((i % n) + n) % n];

    // Walk the circuit rung by rung. Each section i spans plan param [i, i+1); rung k of
    // its profile sits at fraction k/rungs along a Catmull-Rom through the grid anchors,
    // so the curves round out exactly where the pieces carry more rungs. The rung's left
    // and right rail heights come straight from the profile (their difference is the real
    // camber — no separate banking term needed).
    const centres = [], hl = [], hr = [];
    for (let i = 0; i < n; i++) {
      const pr = track.profiles[i];
      const L = pr[0], R = pr[1], rungs = Math.min(L.length, R.length);
      for (let k = 0; k < rungs; k++) {
        const t = k / rungs;
        centres.push(cmPt(Ai(i - 1), Ai(i), Ai(i + 1), Ai(i + 2), t));
        hl.push((L[k] - minH) * HK);
        hr.push((R[k] - minH) * HK);
      }
    }
    const m = centres.length;

    // Offset each rung centre by the local normal to make the two rails, and scale into
    // grid units (centred on the plan). The normal comes from the spline tangent.
    const rings = [];
    for (let k = 0; k < m; k++) {
      const a = centres[(k - 1 + m) % m], b = centres[(k + 1) % m];
      let dx = b.x - a.x, dz = b.z - a.z;
      const len = Math.hypot(dx, dz) || 1; dx /= len; dz /= len;
      const nx = -dz, nz = dx; // left normal
      const c = centres[k];
      rings.push({
        l: { x: (c.x + nx * WIDTH - cx) * S, z: (c.z + nz * WIDTH - cz) * S },
        r: { x: (c.x - nx * WIDTH - cx) * S, z: (c.z - nz * WIDTH - cz) * S },
        hl: hl[k] * S, hr: hr[k] * S,
      });
    }
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
}

function disposeGroup(g) {
  g.traverse(o => { if (o.geometry) o.geometry.dispose(); if (o.material) o.material.dispose(); });
}
