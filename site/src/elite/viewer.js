// Elite ship viewer: renders one decoded wireframe blueprint with three.js.
// Hidden-line removal is done with the depth buffer rather than a per-face
// normal test: each face is filled into the depth buffer invisibly (no colour),
// then every edge is drawn on top with a small polygon offset, so the GPU hides
// any edge lying behind a face. This is exact at every zoom and orientation —
// no grazing-face popping — and lets the (invisible) hull occlude the background
// stars, so the ship reads as a solid object in space.
import * as THREE from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';

const WHITE = 0xffffff;

// ShipMesh builds the geometry for one ship: a depth-only fill mesh (faces) and
// a white LineSegments (edges), grouped together. Elite's +Y is up, matching
// three.js; X right, Z toward the viewer (the offline montage's handedness).
class ShipMesh {
  constructor(ship) {
    this.radius = ship.radius || 1;

    const verts = new Float32Array(ship.verts.length * 3);
    for (let i = 0; i < ship.verts.length; i++) {
      verts[i * 3] = ship.verts[i][0];
      verts[i * 3 + 1] = ship.verts[i][1];
      verts[i * 3 + 2] = ship.verts[i][2];
    }

    // Edges → line segment endpoints.
    const epos = new Float32Array(ship.edges.length * 6);
    for (let i = 0; i < ship.edges.length; i++) {
      const a = ship.edges[i][0] * 3, b = ship.edges[i][1] * 3;
      epos[i * 6] = verts[a]; epos[i * 6 + 1] = verts[a + 1]; epos[i * 6 + 2] = verts[a + 2];
      epos[i * 6 + 3] = verts[b]; epos[i * 6 + 4] = verts[b + 1]; epos[i * 6 + 5] = verts[b + 2];
    }
    const egeom = new THREE.BufferGeometry();
    egeom.setAttribute('position', new THREE.BufferAttribute(epos, 3));
    this.lines = new THREE.LineSegments(egeom, new THREE.LineBasicMaterial({ color: WHITE }));
    this.lines.frustumCulled = false;
    this.lines.renderOrder = 1;

    // Faces → depth-only fill. colorWrite:false makes it invisible but still
    // write depth; polygonOffset pushes it back a hair so coincident edges win.
    const fgeom = new THREE.BufferGeometry();
    fgeom.setAttribute('position', new THREE.BufferAttribute(verts, 3));
    fgeom.setIndex(ship.tris.flat());
    const fmat = new THREE.MeshBasicMaterial({
      colorWrite: false,
      side: THREE.DoubleSide,
      polygonOffset: true,
      polygonOffsetFactor: 1,
      polygonOffsetUnits: 1,
    });
    this.fill = new THREE.Mesh(fgeom, fmat);
    this.fill.frustumCulled = false;
    this.fill.renderOrder = 0;

    this.object = new THREE.Group();
    this.object.add(this.fill);
    this.object.add(this.lines);
  }

  dispose() {
    this.lines.geometry.dispose();
    this.lines.material.dispose();
    this.fill.geometry.dispose();
    this.fill.material.dispose();
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
    // uniform on the sphere
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
  // Draw after the ship's depth fill (renderOrder 0) and edges (1) so the hull
  // occludes any star behind it.
  pts.renderOrder = 2;
  return pts;
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

    // Stars on a sphere well beyond any ship (radius < far plane). Constant
    // pixel size, so they stay a calm backdrop at every zoom.
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
    this.controls.minDistance = mesh.radius * 0.2;
    this.controls.maxDistance = dist * 3;
    this.controls.autoRotate = true;
    this.controls.update();

    if (this.hud) {
      this.hud.textContent =
        `${ship.name}  ·  type ${ship.type}  ·  ${ship.verts.length} verts  ${ship.edges.length} edges  ${ship.faces} faces`;
    }
  }

  // renderThumbnail draws one ship at the shared 3/4 view into a 2D canvas,
  // using a single throwaway WebGL renderer for every thumbnail (so the page
  // never holds more than two GL contexts).
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
    this._thumbScene.add(mesh.object);
    this._thumbRenderer.render(this._thumbScene, this._thumbCam);
    canvas2d.getContext('2d').drawImage(this._thumbRenderer.domElement, 0, 0, size, size);
    this._thumbScene.remove(mesh.object);
    mesh.dispose();
  }
}
