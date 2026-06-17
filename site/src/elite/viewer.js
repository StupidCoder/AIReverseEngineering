// Elite ship viewer: renders one decoded wireframe blueprint with three.js and
// reproduces the game's own hidden-line removal. The ship sits at the origin and
// the camera orbits it (OrbitControls), so model space is world space. Each
// frame we test every face against the current camera position and draw an edge
// only when at least one of its two bordering faces points toward the eye —
// exactly the back-face test the C64 game runs (Elite.md Part IV §1).
import * as THREE from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';

const WHITE = 0xffffff;

// ShipMesh holds one ship's geometry plus the per-frame visible-edge buffer.
// The blueprint is flat typed arrays for a tight HSR loop; verts/normals are
// kept in model space and never transformed (the ship stays at the origin).
class ShipMesh {
  constructor(ship) {
    this.ship = ship;
    this.radius = ship.radius || 1;

    this.verts = new Float32Array(ship.verts.length * 3);
    for (let i = 0; i < ship.verts.length; i++) {
      const v = ship.verts[i];
      // Elite's vertical axis is +Y up, matching three.js; X right, Z toward the
      // viewer — same handedness the offline montage renderer uses.
      this.verts[i * 3] = v[0];
      this.verts[i * 3 + 1] = v[1];
      this.verts[i * 3 + 2] = v[2];
    }

    this.edges = ship.edges; // [v1, v2, faceA, faceB]
    this.faceN = new Float32Array(ship.faces.length * 3); // outward normal per face
    for (let i = 0; i < ship.faces.length; i++) {
      const f = ship.faces[i];
      this.faceN[i * 3] = f[0];
      this.faceN[i * 3 + 1] = f[1];
      this.faceN[i * 3 + 2] = f[2];
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

  // updateForCamera rebuilds the visible-edge list for an eye looking from
  // camPos (THREE.Vector3) toward the model centre at the origin. A face is
  // visible when its outward normal points toward the eye. We test the normal
  // against the eye *direction* (camPos, the model is centred on the orbit
  // target) rather than the eye *position*: for a distant eye the two agree —
  // the regime the game itself draws in — but the direction test is independent
  // of zoom, so a grazing face stays visible as you dolly in instead of popping
  // out when the eye crosses its plane. Returns the number of edges drawn.
  updateForCamera(camPos) {
    const { verts, faceN, faceVis } = this;
    for (let i = 0; i < faceVis.length; i++) {
      const dot = faceN[i * 3] * camPos.x
        + faceN[i * 3 + 1] * camPos.y
        + faceN[i * 3 + 2] * camPos.z;
      faceVis[i] = dot > 0 ? 1 : 0;
    }
    const pos = this.positions;
    let n = 0;
    for (const e of this.edges) {
      if (!(faceVis[e[2]] || faceVis[e[3]])) continue;
      const a = e[0] * 3, b = e[1] * 3;
      pos[n++] = verts[a]; pos[n++] = verts[a + 1]; pos[n++] = verts[a + 2];
      pos[n++] = verts[b]; pos[n++] = verts[b + 1]; pos[n++] = verts[b + 2];
    }
    this.geom.setDrawRange(0, n / 3);
    this.geom.attributes.position.needsUpdate = true;
    return n / 6;
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

export class ShipViewer {
  constructor(viewport, hud) {
    this.viewport = viewport;
    this.hud = hud;
    this.ships = [];
    this.current = null;
  }

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
      if (this.current) this.current.updateForCamera(this.camera.position);
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
    this.controls.minDistance = mesh.radius * 0.4;
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
