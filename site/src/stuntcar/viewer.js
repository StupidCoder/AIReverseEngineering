// Stunt Car Racer — track ribbon viewer. The geometry is the verified plan-view
// spine (package track, Part IV §5): one (x,z) node per section, decoded purely in
// Go from the disk and exported to tracks.json. We build the track as a ribbon of a
// nominal width, render it as a hidden-line wireframe (invisible depth fill + colour
// LineSegments, the same technique as the Marble Madness slope viewer), and view it
// down a dimetric angle. Elevation is not yet recovered, so the ribbon is flat (y=0);
// the code carries a per-node height so it can light up once that's decoded.
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

    // Nodes -> centre-line points. n[0],n[1] = grid plan cell; n[2] = surface elevation
    // (mean of the two rail heights). Sit each track on the ground (subtract its min)
    // and scale into grid-cell units so the relief reads against the plan.
    const EY = 1 / 3600; // elevation units -> grid cells
    let minH = Infinity;
    for (const n of track.nodes) minH = Math.min(minH, n[2]);
    // y = surface height; bankY = rail-height difference (camber), applied ± per rail.
    const pts = track.nodes.map(n => ({ x: n[0], z: n[1], y: (n[2] - minH) * EY, bankY: n[3] * EY }));
    const n = pts.length;
    let minX = Infinity, maxX = -Infinity, minZ = Infinity, maxZ = -Infinity;
    for (const p of pts) { minX = Math.min(minX, p.x); maxX = Math.max(maxX, p.x); minZ = Math.min(minZ, p.z); maxZ = Math.max(maxZ, p.z); }
    const cx = (minX + maxX) / 2, cz = (minZ + maxZ) / 2;
    const span = Math.max(maxX - minX, maxZ - minZ) || 1;
    const S = 8 / span; // fit into ~8 units

    // Rail plan positions (x,z) and per-section rail heights, kept separate. The height
    // belongs to the SEGMENT, not the vertex: each section is a flat platform and the
    // changes are vertical steps at the boundaries (the Stepping Stones are literally
    // flat stones with square gaps), so we draw flat tops + vertical risers rather than
    // interpolating between node heights (which would round the steps into bumps).
    const pL = [], pR = [], hL = [], hR = [];
    for (let i = 0; i < n; i++) {
      const a = pts[(i - 1 + n) % n], b = pts[(i + 1) % n], c = pts[i];
      let dx = b.x - a.x, dz = b.z - a.z;
      const len = Math.hypot(dx, dz) || 1; dx /= len; dz /= len;
      const nx = -dz, nz = dx; // left normal
      pL.push({ x: (c.x + nx * WIDTH - cx) * S, z: (c.z + nz * WIDTH - cz) * S });
      pR.push({ x: (c.x - nx * WIDTH - cx) * S, z: (c.z - nz * WIDTH - cz) * S });
      hL.push((c.y + c.bankY * 0.5) * S);
      hR.push((c.y - c.bankY * 0.5) * S);
    }
    const V = (p, y) => new THREE.Vector3(p.x, y, p.z);

    // Smooth ramp vs hard step. The track surface is interpolated by default (so the
    // ramps and rolling hills stay smooth and drivable — Roller Coaster has +4096-per-
    // section climbs that are perfectly smooth), and only genuine features become steps:
    //  - a "platform": a prominent local extremum (a stone or a jump that rises AND
    //    falls sharply — Stepping Stones, Big Ramp's three jumps),
    //  - a "cliff": a single very large drop (Ski Jump's launch).
    // NB this is a data-pattern criterion, not a pinned flag: a small jump's drop (~990)
    // and a smooth Roller-Coaster bump (~960) are nearly equal in magnitude, so a height
    // rule can't separate them perfectly (one RC bump steps). The renderer's own sharp-
    // edge test ($65D3C) is a silhouette/crease check, not this surface decision.
    const PROM = 900, CLIFF = 5500;
    const hr = track.nodes.map(nn => nn[2]);
    const at = i => hr[((i % n) + n) % n];
    const plat = hr.map((_, i) => {
      const a = at(i - 1), b = at(i + 1), c = hr[i];
      const ext = (c >= a && c >= b) || (c <= a && c <= b);
      return ext && Math.min(Math.abs(c - a), Math.abs(c - b)) >= PROM;
    });
    const cliff = hr.map((_, i) => Math.abs(at(i + 1) - hr[i]) >= CLIFF);
    const step = i => plat[i] || plat[(i + 1) % n] || cliff[i];

    // Invisible depth fill: a sloped top across smooth segments, or a flat top + riser
    // at a step.
    const fpos = [];
    const quad = (a, b, c, d) => fpos.push(a.x, a.y, a.z, b.x, b.y, b.z, c.x, c.y, c.z, b.x, b.y, b.z, d.x, d.y, d.z, c.x, c.y, c.z);
    for (let i = 0; i < n; i++) {
      const j = (i + 1) % n;
      if (step(i)) {
        quad(V(pL[i], hL[i]), V(pR[i], hR[i]), V(pL[j], hL[i]), V(pR[j], hR[i]));   // flat top at segment i's height
        quad(V(pL[j], hL[i]), V(pR[j], hR[i]), V(pL[j], hL[j]), V(pR[j], hR[j]));   // riser to segment j's height
      } else {
        quad(V(pL[i], hL[i]), V(pR[i], hR[i]), V(pL[j], hL[j]), V(pR[j], hR[j]));   // sloped (interpolated)
      }
    }
    const fgeom = new THREE.BufferGeometry();
    fgeom.setAttribute('position', new THREE.Float32BufferAttribute(fpos, 3));
    const fill = new THREE.Mesh(fgeom, new THREE.MeshBasicMaterial({
      colorWrite: false, polygonOffset: true, polygonOffsetFactor: 1, polygonOffsetUnits: 1,
      side: THREE.DoubleSide,
    }));
    group.add(fill);

    // Wireframe: flat rails + rung per segment, and the vertical risers between them.
    const lpos = [], lcol = [];
    const col = new THREE.Color();
    const edge = (p, q, f) => {
      col.setHSL(0.58 - 0.5 * f, 0.85, 0.55);
      lpos.push(p.x, p.y, p.z, q.x, q.y, q.z);
      lcol.push(col.r, col.g, col.b, col.r, col.g, col.b);
    };
    for (let i = 0; i < n; i++) {
      const j = (i + 1) % n, f = i / n;
      edge(V(pL[i], hL[i]), V(pR[i], hR[i]), f); // rung
      if (step(i)) {
        edge(V(pL[i], hL[i]), V(pL[j], hL[i]), f); // flat left rail
        edge(V(pR[i], hR[i]), V(pR[j], hR[i]), f); // flat right rail
        edge(V(pL[j], hL[i]), V(pL[j], hL[j]), f); // left riser
        edge(V(pR[j], hR[i]), V(pR[j], hR[j]), f); // right riser
      } else {
        edge(V(pL[i], hL[i]), V(pL[j], hL[j]), f); // sloped left rail
        edge(V(pR[i], hR[i]), V(pR[j], hR[j]), f); // sloped right rail
      }
    }
    const lgeom = new THREE.BufferGeometry();
    lgeom.setAttribute('position', new THREE.Float32BufferAttribute(lpos, 3));
    lgeom.setAttribute('color', new THREE.Float32BufferAttribute(lcol, 3));
    group.add(new THREE.LineSegments(lgeom, new THREE.LineBasicMaterial({ vertexColors: true })));

    // Support columns down to the ground (y=0), like the game's preview.
    const cpos = [];
    for (let i = 0; i < n; i++) {
      const mx = (pL[i].x + pR[i].x) / 2, mz = (pL[i].z + pR[i].z) / 2, my = (hL[i] + hR[i]) / 2;
      if (my > 0.02) cpos.push(mx, my, mz, mx, 0, mz);
    }
    if (cpos.length) {
      const cg = new THREE.BufferGeometry();
      cg.setAttribute('position', new THREE.Float32BufferAttribute(cpos, 3));
      group.add(new THREE.LineSegments(cg, new THREE.LineBasicMaterial({ color: 0x6b4a3a })));
    }

    // Start/finish marker (green) at section 0.
    const sm = new THREE.Mesh(new THREE.SphereGeometry(0.12, 12, 12), new THREE.MeshBasicMaterial({ color: 0x35d07f }));
    sm.position.set((pL[0].x + pR[0].x) / 2, (hL[0] + hR[0]) / 2, (pL[0].z + pR[0].z) / 2);
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
