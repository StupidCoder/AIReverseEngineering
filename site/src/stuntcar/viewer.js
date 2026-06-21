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

    // Nodes -> centre-line points. n[0],n[1] = grid plan cell; n[2] = elevation
    // (height above the track reference). Scale elevation into grid-cell units.
    const EY = 1 / 900; // elevation units (~±1600) -> a few grid cells
    const pts = track.nodes.map(n => ({ x: n[0], z: n[1], y: n[2] * EY }));
    const n = pts.length;
    let minX = Infinity, maxX = -Infinity, minZ = Infinity, maxZ = -Infinity;
    for (const p of pts) { minX = Math.min(minX, p.x); maxX = Math.max(maxX, p.x); minZ = Math.min(minZ, p.z); maxZ = Math.max(maxZ, p.z); }
    const cx = (minX + maxX) / 2, cz = (minZ + maxZ) / 2;
    const span = Math.max(maxX - minX, maxZ - minZ) || 1;
    const S = 8 / span; // fit into ~8 units

    // Left/right ribbon edges, perpendicular to the local direction (closed loop).
    const left = [], right = [];
    for (let i = 0; i < n; i++) {
      const a = pts[(i - 1 + n) % n], b = pts[(i + 1) % n], c = pts[i];
      let dx = b.x - a.x, dz = b.z - a.z;
      const len = Math.hypot(dx, dz) || 1; dx /= len; dz /= len;
      const nx = -dz, nz = dx; // left normal
      left.push(new THREE.Vector3((c.x + nx * WIDTH - cx) * S, c.y * S, (c.z + nz * WIDTH - cz) * S));
      right.push(new THREE.Vector3((c.x - nx * WIDTH - cx) * S, c.y * S, (c.z - nz * WIDTH - cz) * S));
    }

    // Invisible depth fill (the ribbon surface) for hidden-line removal.
    const fpos = [];
    for (let i = 0; i < n; i++) {
      const j = (i + 1) % n;
      const a = left[i], b = right[i], c = left[j], d = right[j];
      fpos.push(a.x, a.y, a.z, b.x, b.y, b.z, c.x, c.y, c.z);
      fpos.push(b.x, b.y, b.z, d.x, d.y, d.z, c.x, c.y, c.z);
    }
    const fgeom = new THREE.BufferGeometry();
    fgeom.setAttribute('position', new THREE.Float32BufferAttribute(fpos, 3));
    const fill = new THREE.Mesh(fgeom, new THREE.MeshBasicMaterial({
      colorWrite: false, polygonOffset: true, polygonOffsetFactor: 1, polygonOffsetUnits: 1,
      side: THREE.DoubleSide,
    }));
    group.add(fill);

    // Wireframe: the two rails (closed) + a rung at every section, coloured along
    // the lap so direction reads.
    const lpos = [], lcol = [];
    const col = new THREE.Color();
    const edge = (p, q, f) => {
      col.setHSL(0.58 - 0.5 * f, 0.85, 0.55);
      lpos.push(p.x, p.y, p.z, q.x, q.y, q.z);
      lcol.push(col.r, col.g, col.b, col.r, col.g, col.b);
    };
    for (let i = 0; i < n; i++) {
      const j = (i + 1) % n, f = i / n;
      edge(left[i], left[j], f);
      edge(right[i], right[j], f);
      edge(left[i], right[i], f); // rung
    }
    const lgeom = new THREE.BufferGeometry();
    lgeom.setAttribute('position', new THREE.Float32BufferAttribute(lpos, 3));
    lgeom.setAttribute('color', new THREE.Float32BufferAttribute(lcol, 3));
    group.add(new THREE.LineSegments(lgeom, new THREE.LineBasicMaterial({ vertexColors: true })));

    // Start/finish marker (green) at section 0.
    const sm = new THREE.Mesh(new THREE.SphereGeometry(0.12, 12, 12), new THREE.MeshBasicMaterial({ color: 0x35d07f }));
    sm.position.copy(left[0]).add(right[0]).multiplyScalar(0.5);
    group.add(sm);

    t.scene.add(group);
    t.group = group;

    // Frame it from a high, near-top-down angle so the (flat) circuit reads as a plan
    // rather than edge-on. Pull back to fit the longest axis of the ribbon.
    const cam = t.camera, ctrl = t.controls;
    ctrl.target.set(0, 0, 0);
    cam.position.set(0.5, 6.5, 8);
    cam.near = 0.1; cam.far = 100; cam.updateProjectionMatrix();
    ctrl.update();
  }
}

function disposeGroup(g) {
  g.traverse(o => { if (o.geometry) o.geometry.dispose(); if (o.material) o.material.dispose(); });
}
