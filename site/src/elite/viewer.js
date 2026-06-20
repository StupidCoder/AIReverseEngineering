// Elite ship viewer: renders one decoded wireframe blueprint with three.js and
// reproduces the game's own hidden-surface removal. The ship sits at the origin
// and the camera orbits it (OrbitControls), so model space is world space. Each
// frame we back-face test every face by the sign of dot(normal, eye - faceCenter)
// — the face is visible when it points toward the eye from where the face sits —
// and draw an edge only when one of its two adjacent faces is visible (Elite.md
// Part IV §1). Using the face's own centre (not the ship's) is what correctly
// hides faces on the far side; the eye is kept outside the hull so faces never
// flip abruptly.
import * as THREE from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';

const WHITE = 0xffffff;
const FACE_NONE = 15; // edge face nibble sentinel: no face this side → always drawn

// "Old school" authentic flicker. The C64 draws ships by XOR-plotting their line
// heap into a single, live bitmap (no double buffer): to move a ship it re-draws
// the old lines to erase them, reprojects, then draws the new lines. That
// erase→reproject→redraw runs in the game loop, out of sync with the 50 Hz beam,
// so the beam keeps catching individual lines during the brief window they are
// erased — the signature Elite shimmer. We reproduce it by giving every edge its
// own erase-window phase and, each displayed frame, hiding the edges whose window
// the "beam" is currently sampling. FLICKER_DUTY ≈ the fraction of a redraw cycle
// a line spends erased; FLICKER_RATE drifts the sampling phase so the off-set
// changes (and beats) every frame rather than sitting still.
const FLICKER_DUTY = 0.17;
const FLICKER_RATE = 0.37;

// ShipMesh holds one ship's geometry and the per-frame visible-edge buffer.
// Flat typed arrays keep the HSR loop tight; verts/normals stay in model space
// (the ship never moves — the camera orbits). Elite's +Y is up, matching
// three.js; X right, Z toward the viewer (the offline montage's handedness).
class ShipMesh {
  constructor(ship) {
    this.radius = ship.radius || 1;

    this.verts = new Float32Array(ship.verts.length * 3);
    for (let i = 0; i < ship.verts.length; i++) {
      this.verts[i * 3] = ship.verts[i][0];
      this.verts[i * 3 + 1] = ship.verts[i][1];
      this.verts[i * 3 + 2] = ship.verts[i][2];
    }

    this.edges = ship.edges; // [v1, v2, faceA, faceB]
    // A stable per-edge phase in [0,1) for the authentic flicker: each line gets
    // its own slot in the erase/redraw cycle, scattered so the blinking reads as
    // shimmer rather than a clean wipe.
    this.edgePhase = new Float32Array(this.edges.length);
    for (let i = 0; i < this.edges.length; i++) {
      const e = this.edges[i];
      const h = Math.sin((e[0] + 1) * 12.9898 + (e[1] + 1) * 78.233 + i * 0.613) * 43758.5453;
      this.edgePhase[i] = h - Math.floor(h);
    }
    this.faceN = new Float32Array(ship.faces.length * 3); // outward normal per face
    this.faceC = new Float32Array(ship.faces.length * 3); // a point on each face
    for (let i = 0; i < ship.faces.length; i++) {
      this.faceN[i * 3] = ship.faces[i][0];
      this.faceN[i * 3 + 1] = ship.faces[i][1];
      this.faceN[i * 3 + 2] = ship.faces[i][2];
      this.faceC[i * 3] = ship.faceC[i][0];
      this.faceC[i * 3 + 1] = ship.faceC[i][1];
      this.faceC[i * 3 + 2] = ship.faceC[i][2];
    }
    this.faceVis = new Uint8Array(ship.faces.length);

    // One LineSegments whose position buffer we refill each frame with only the
    // currently-visible edges; drawRange caps it to what we wrote.
    const positions = new Float32Array(this.edges.length * 6);
    const geom = new THREE.BufferGeometry();
    geom.setAttribute('position', new THREE.BufferAttribute(positions, 3));
    geom.setDrawRange(0, 0);
    this.geom = geom;
    this.positions = positions;
    this.object = new THREE.LineSegments(geom, new THREE.LineBasicMaterial({ color: WHITE }));
    this.object.frustumCulled = false;
  }

  // updateForCamera rebuilds the visible-edge list for a camera at camPos
  // (THREE.Vector3). A face is visible when its outward normal points toward the
  // eye from the face's own position: dot(normal, camPos - faceCenter) > 0.
  // Testing from the face centre (not the origin) is what correctly culls faces
  // on the far side instead of leaving them showing "below the horizon".
  // flickerPhase < 0 disables the effect; otherwise it is the current sampling
  // phase, and an edge is dropped this frame when the "beam" falls inside its
  // erase window — reproducing the C64's single-buffer XOR redraw flicker.
  updateForCamera(camPos, flickerPhase = -1) {
    const { verts, faceN, faceC, faceVis, edges, edgePhase } = this;
    for (let i = 0; i < faceVis.length; i++) {
      const dot = faceN[i * 3] * (camPos.x - faceC[i * 3])
        + faceN[i * 3 + 1] * (camPos.y - faceC[i * 3 + 1])
        + faceN[i * 3 + 2] * (camPos.z - faceC[i * 3 + 2]);
      faceVis[i] = dot > 0 ? 1 : 0;
    }
    const pos = this.positions;
    const flicker = flickerPhase >= 0;
    let n = 0;
    for (let ei = 0; ei < edges.length; ei++) {
      const e = edges[ei];
      const fa = e[2], fb = e[3];
      const vis = fa === FACE_NONE || fb === FACE_NONE || faceVis[fa] || faceVis[fb];
      if (!vis) continue;
      if (flicker) {
        let d = edgePhase[ei] - flickerPhase;
        d -= Math.floor(d);
        if (d < FLICKER_DUTY) continue; // this line is mid-erase right now → blink off
      }
      const a = e[0] * 3, b = e[1] * 3;
      pos[n++] = verts[a]; pos[n++] = verts[a + 1]; pos[n++] = verts[a + 2];
      pos[n++] = verts[b]; pos[n++] = verts[b + 1]; pos[n++] = verts[b + 2];
    }
    this.geom.setDrawRange(0, n / 3);
    this.geom.attributes.position.needsUpdate = true;
  }

  dispose() {
    this.geom.dispose();
    this.object.material.dispose();
  }
}

// A pleasant 3/4 viewing direction (normalized), shared by the main camera's
// starting pose and the thumbnails, so a thumbnail previews the opening view.
const VIEW_DIR = new THREE.Vector3(0.55, 0.42, 1).normalize();

// fitDistance returns a camera distance that frames a ship of the given radius.
function fitDistance(radius, fovDeg) {
  return (radius * 1.6) / Math.sin((fovDeg * Math.PI) / 360);
}

// makeStarfield returns a Points cloud of dim dots scattered on a large sphere.
// It lives in world space (the ship is fixed; the camera orbits), so the stars
// wheel around with the ship. sizeAttenuation:false keeps them a constant pixel
// size, so zooming the ship doesn't change the stars.
function makeStarfield(count, radius) {
  const pos = new Float32Array(count * 3);
  for (let i = 0; i < count; i++) {
    const u = Math.random() * 2 - 1;
    const t = Math.random() * Math.PI * 2;
    const r = Math.sqrt(1 - u * u);
    pos[i * 3] = Math.cos(t) * r * radius;
    pos[i * 3 + 1] = u * radius;
    pos[i * 3 + 2] = Math.sin(t) * r * radius;
  }
  const geom = new THREE.BufferGeometry();
  geom.setAttribute('position', new THREE.BufferAttribute(pos, 3));
  const mat = new THREE.PointsMaterial({ color: 0xdfe6f2, size: 2, sizeAttenuation: false });
  const pts = new THREE.Points(geom, mat);
  pts.frustumCulled = false;
  return pts;
}

export class ShipViewer {
  constructor(viewport, hud) {
    this.viewport = viewport;
    this.hud = hud;
    this.ships = [];
    this.current = null;
    this.oldSchool = false; // "old school" CRT/flicker effects (start: line flicker)
    this.flickerPhase = 0;
  }

  setOldSchool(on) { this.oldSchool = on; }

  async init() {
    const res = await fetch('public/elite/ships.json');
    const doc = await res.json();
    this.ships = doc.ships;

    const fov = 45;
    this.renderer = new THREE.WebGLRenderer({ antialias: true });
    this.renderer.setPixelRatio(Math.min(devicePixelRatio, 2));
    this.viewport.appendChild(this.renderer.domElement);
    this.scene = new THREE.Scene();
    this.scene.background = new THREE.Color(0x000000);
    this.camera = new THREE.PerspectiveCamera(fov, 1, 0.1, 200000);

    // Stars on a sphere well beyond any ship; a calm vector-graphics backdrop.
    this.scene.add(makeStarfield(500, 60000));

    this.controls = new OrbitControls(this.camera, this.renderer.domElement);
    this.controls.enableDamping = true;
    this.controls.dampingFactor = 0.08;
    this.controls.enablePan = false;
    this.controls.rotateSpeed = 0.9;
    this.controls.zoomSpeed = 4.0;
    this.controls.autoRotate = true;
    this.controls.autoRotateSpeed = 1.1;
    // Once the user grabs the ship, stop the idle spin for good.
    this.controls.addEventListener('start', () => { this.controls.autoRotate = false; });

    this._resize();
    new ResizeObserver(() => this._resize()).observe(this.viewport);

    const tick = () => {
      this.controls.update();
      if (this.oldSchool) this.flickerPhase = (this.flickerPhase + FLICKER_RATE) % 1;
      if (this.current) {
        this.current.updateForCamera(this.camera.position, this.oldSchool ? this.flickerPhase : -1);
      }
      this.renderer.render(this.scene, this.camera);
      requestAnimationFrame(tick);
    };
    requestAnimationFrame(tick);
    return this.ships;
  }

  _resize() {
    const w = this.viewport.clientWidth, h = this.viewport.clientHeight;
    if (!w || !h) return;
    this.renderer.setSize(w, h, false);
    this.camera.aspect = w / h;
    this.camera.updateProjectionMatrix();
  }

  loadShip(index) {
    const ship = this.ships[index];
    if (!ship) return;
    if (this.current) {
      this.scene.remove(this.current.object);
      this.current.dispose();
    }
    const mesh = new ShipMesh(ship);
    this.scene.add(mesh.object);
    this.current = mesh;
    this.currentIndex = index;

    const dist = fitDistance(mesh.radius, this.camera.fov);
    this.camera.position.copy(VIEW_DIR).multiplyScalar(dist);
    this.controls.target.set(0, 0, 0);
    // Keep the eye just outside the bounding sphere: with the per-face test,
    // entering the hull is what makes faces flip abruptly. At this distance the
    // ship already overfills the view, so it's still close enough to inspect.
    this.controls.minDistance = mesh.radius * 1.1;
    this.controls.maxDistance = dist * 3;
    this.controls.autoRotate = true;
    this.controls.update();

    if (this.hud) {
      this.hud.textContent =
        `${ship.name}  ·  type ${ship.type}  ·  ${ship.verts.length} verts  ${ship.edges.length} edges  ${ship.faces.length} faces`;
    }
  }

  // renderThumbnail draws one ship at the shared 3/4 view into a 2D canvas,
  // using a single throwaway WebGL renderer for every thumbnail (so the page
  // never holds more than two GL contexts). HSR is applied for that fixed eye.
  renderThumbnail(index, canvas2d, size) {
    if (!this._thumbRenderer) {
      this._thumbRenderer = new THREE.WebGLRenderer({ antialias: true, alpha: false, preserveDrawingBuffer: true });
      this._thumbRenderer.setPixelRatio(1);
      this._thumbRenderer.setSize(size, size, false);
      this._thumbScene = new THREE.Scene();
      this._thumbScene.background = new THREE.Color(0x000000);
      this._thumbCam = new THREE.PerspectiveCamera(45, 1, 0.1, 200000);
    }
    const mesh = new ShipMesh(this.ships[index]);
    const dist = fitDistance(mesh.radius, this._thumbCam.fov);
    this._thumbCam.position.copy(VIEW_DIR).multiplyScalar(dist);
    this._thumbCam.lookAt(0, 0, 0);
    mesh.updateForCamera(this._thumbCam.position);
    this._thumbScene.add(mesh.object);
    this._thumbRenderer.render(this._thumbScene, this._thumbCam);
    canvas2d.getContext('2d').drawImage(this._thumbRenderer.domElement, 0, 0, size, size);
    this._thumbScene.remove(mesh.object);
    mesh.dispose();
  }
}
