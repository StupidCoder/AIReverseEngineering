// Marble Madness course viewer — PixiJS (tilemap) + three.js (slopes), no build.
//
// Two views of the same course share one viewport:
//  • Tilemap — the playfield image decoded from the .mlb, drawn with the shared
//    drag-to-pan / scroll-to-zoom camera (PixiJS).
//  • Slopes — the static slope field (the height the marble rolls on, decoded
//    from the Track file) as a 3-D height-mesh you drag to rotate (three.js).
// A toggle switches engines; only the active canvas is shown and handles input.

import { Application, Container, Sprite, Texture } from 'pixi.js';
import * as THREE from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';

const DATA = 'public/marble/';
const NATIVE_W = 288;                    // the playfield is 288 px (36 tiles) wide
const ZOOM_STEP = Math.pow(1.15, 0.25);
const HEIGHT_SCALE = 0.15;               // slope-mesh vertical exaggeration (per tile unit)

// heightRamp maps t in [0,1] to blue(low)..white(high), matching the offline
// region renderer; returns r,g,b in 0..1 for three.js vertex colours.
const RAMP = [
  [0.0, 30, 40, 120], [0.3, 40, 140, 150], [0.55, 60, 170, 80], [0.78, 220, 205, 70], [1.0, 250, 250, 250],
];
function heightRamp(t) {
  t = Math.max(0, Math.min(1, t));
  for (let i = 0; i < RAMP.length - 1; i++) {
    if (t <= RAMP[i + 1][0]) {
      const a = RAMP[i], b = RAMP[i + 1];
      const f = (t - a[0]) / (b[0] - a[0] + 1e-9);
      return [(a[1] + (b[1] - a[1]) * f) / 255, (a[2] + (b[2] - a[2]) * f) / 255, (a[3] + (b[3] - a[3]) * f) / 255];
    }
  }
  return [250 / 255, 250 / 255, 250 / 255];
}

export class MarbleViewer {
  constructor(viewportEl, hudEl) {
    this.el = viewportEl;
    this.hud = hudEl;
    this.mode = 'tilemap';
    this.app = new Application();
    this.world = new Container();
    this.zoom = 1; this.minZoom = 0.1; this.maxZoom = 12;
    this._texMode = 'nearest';
    this.sprite = null;
    this.three = null;
  }

  async init() {
    await this.app.init({ background: 0x000000, antialias: false, resizeTo: this.el });
    this.app.canvas.classList.add('mm-pixi');
    this.el.appendChild(this.app.canvas);
    this.app.stage.addChild(this.world);
    this._wireCamera();
    return fetch(DATA + 'meta.json').then((r) => r.json());
  }

  _loadImage(src) {
    return new Promise((res, rej) => {
      const i = new Image();
      i.onload = () => res(i); i.onerror = rej; i.src = src;
    });
  }

  async loadLevel(metaLevel) {
    this.name = metaLevel.name;
    // Tilemap sprite.
    const img = await this._loadImage(DATA + metaLevel.file);
    const tx = Texture.from(img);
    tx.source.autoGenerateMipmaps = true;
    tx.source.scaleMode = this._texMode;
    if (this.sprite) { this.world.removeChild(this.sprite); this.sprite.destroy(); }
    this.sprite = new Sprite(tx);
    this.world.addChild(this.sprite);
    this.tex = tx;
    this.levelW = img.width; this.levelH = img.height;
    this._fitDefault();
    // Slope field (lazy: only meshed when the 3-D view is active).
    this.slope = await fetch(DATA + metaLevel.slope).then((r) => r.json());
    if (this.three && this.mode === 'slopes') this._buildMesh();
    this._setHud();
  }

  // --- mode switching -----------------------------------------------------
  setMode(mode) {
    this.mode = mode;
    if (mode === 'slopes') {
      if (!this.three) this._initThree();
      this._buildMesh();
      this.app.canvas.style.display = 'none';
      this.three.renderer.domElement.style.display = 'block';
      this._resizeThree();
    } else {
      if (this.three) this.three.renderer.domElement.style.display = 'none';
      this.app.canvas.style.display = 'block';
    }
    this._setHud();
  }

  _setHud() {
    if (!this.hud) return;
    this.hud.textContent = this.mode === 'slopes'
      ? `${this.name} · slope field · drag to rotate`
      : `${this.name} · ${this.levelW}x${this.levelH}`;
  }

  // --- three.js slope view ------------------------------------------------
  _initThree() {
    const renderer = new THREE.WebGLRenderer({ antialias: true });
    renderer.setPixelRatio(Math.min(devicePixelRatio, 2));
    renderer.domElement.classList.add('mm-three');
    this.el.appendChild(renderer.domElement);
    const scene = new THREE.Scene();
    scene.background = new THREE.Color(0x0a0e16);
    const camera = new THREE.PerspectiveCamera(45, 1, 0.1, 100000);
    const controls = new OrbitControls(camera, renderer.domElement);
    controls.enableDamping = true;
    controls.dampingFactor = 0.08;
    controls.rotateSpeed = 0.9;
    controls.zoomSpeed = 1.2;
    scene.add(new THREE.AmbientLight(0xffffff, 0.55));
    const dir = new THREE.DirectionalLight(0xffffff, 0.9);
    dir.position.set(0.6, 1, 0.4);
    scene.add(dir);
    this.three = { renderer, scene, camera, controls, mesh: null };
    new ResizeObserver(() => this._resizeThree()).observe(this.el);
    const tick = () => {
      if (this.mode === 'slopes' && this.three) {
        this.three.controls.update();
        this.three.renderer.render(this.three.scene, this.three.camera);
      }
      requestAnimationFrame(tick);
    };
    requestAnimationFrame(tick);
  }

  _resizeThree() {
    if (!this.three) return;
    const w = this.el.clientWidth, h = this.el.clientHeight;
    if (!w || !h) return;
    this.three.renderer.setSize(w, h, false);
    this.three.camera.aspect = w / h;
    this.three.camera.updateProjectionMatrix();
  }

  // Build the height mesh: a vertex per rolling-surface tile, quads between the
  // four-neighbour groups that all exist (pits leave holes). Coloured by height.
  _buildMesh() {
    const t = this.three;
    if (t.mesh) { t.scene.remove(t.mesh); t.mesh.geometry.dispose(); t.mesh.material.dispose(); }
    const s = this.slope;
    const { w, h, heights } = s;
    const range = Math.max(1, s.hi - s.lo);
    const cx = (w - 1) / 2, cz = (h - 1) / 2;
    const vidx = new Int32Array(w * h).fill(-1);
    const pos = [], col = [];
    for (let gy = 0; gy < h; gy++) {
      for (let gx = 0; gx < w; gx++) {
        const v = heights[gy * w + gx];
        if (v <= 0) continue;
        const hv = v - 1; // height above lo
        vidx[gy * w + gx] = pos.length / 3;
        pos.push(gx - cx, hv * HEIGHT_SCALE, gy - cz);
        const c = heightRamp(hv / range);
        col.push(c[0], c[1], c[2]);
      }
    }
    const idx = [];
    const at = (gx, gy) => vidx[gy * w + gx];
    for (let gy = 0; gy < h - 1; gy++) {
      for (let gx = 0; gx < w - 1; gx++) {
        const a = at(gx, gy), b = at(gx + 1, gy), c = at(gx, gy + 1), d = at(gx + 1, gy + 1);
        if (a >= 0 && b >= 0 && c >= 0 && d >= 0) idx.push(a, c, b, b, c, d);
      }
    }
    const geom = new THREE.BufferGeometry();
    geom.setAttribute('position', new THREE.Float32BufferAttribute(pos, 3));
    geom.setAttribute('color', new THREE.Float32BufferAttribute(col, 3));
    geom.setIndex(idx);
    geom.computeVertexNormals();
    const mat = new THREE.MeshStandardMaterial({ vertexColors: true, roughness: 0.9, metalness: 0, side: THREE.DoubleSide });
    t.mesh = new THREE.Mesh(geom, mat);
    t.scene.add(t.mesh);

    // Frame the mesh from a 3/4 elevated angle.
    geom.computeBoundingBox();
    const bb = geom.boundingBox, ctr = new THREE.Vector3();
    bb.getCenter(ctr);
    const span = Math.max(w, h);
    t.controls.target.copy(ctr);
    t.camera.position.set(ctr.x + span * 0.9, ctr.y + span * 0.8, ctr.z + span * 0.9);
    t.camera.near = span / 100; t.camera.far = span * 20;
    t.camera.updateProjectionMatrix();
    t.controls.minDistance = span * 0.25;
    t.controls.maxDistance = span * 4;
    t.controls.update();
  }

  // --- PixiJS tilemap camera (shared pattern; only active in tilemap mode) -
  _fitDefault() {
    const W = this.app.screen.width, H = this.app.screen.height;
    this.minZoom = Math.min(W / this.levelW, H / this.levelH) * 0.95;
    this.maxZoom = (W / NATIVE_W) * 3;
    this.zoom = Math.max(this.minZoom, Math.min(this.maxZoom, (W / this.levelW) * 0.98));
    this._panTo(this.levelW / 2, 0);
    this._apply();
  }
  _panTo(wx, wy) {
    this.world.position.set(this.app.screen.width / 2 - wx * this.zoom, this.app.screen.height / 2 - wy * this.zoom);
  }
  _screenPt(cx, cy) {
    const r = this.el.getBoundingClientRect();
    return { x: (cx - r.left) * (this.app.screen.width / r.width), y: (cy - r.top) * (this.app.screen.height / r.height) };
  }
  _zoomAt(px, py, f) {
    const wx = (px - this.world.position.x) / this.zoom, wy = (py - this.world.position.y) / this.zoom;
    this.zoom = Math.min(this.maxZoom, Math.max(this.minZoom, this.zoom * f));
    this.world.position.set(px - wx * this.zoom, py - wy * this.zoom);
    this._apply();
  }
  _clampPan() {
    const sw = this.app.screen.width, sh = this.app.screen.height;
    const lw = this.levelW * this.zoom, lh = this.levelH * this.zoom;
    let { x, y } = this.world.position;
    x = lw <= sw ? (sw - lw) / 2 : Math.min(0, Math.max(sw - lw, x));
    y = lh <= sh ? (sh - lh) / 2 : Math.min(0, Math.max(sh - lh, y));
    this.world.position.set(x, y);
  }
  _apply() {
    this.world.scale.set(this.zoom);
    this._clampPan();
    this._updateTexFilter();
  }
  _updateTexFilter() {
    const mode = this.zoom < 1 ? 'linear' : 'nearest';
    if (mode === this._texMode) return;
    this._texMode = mode;
    if (this.tex) this.tex.source.scaleMode = mode;
  }
  _wireCamera() {
    const c = this.el;
    const pts = new Map();
    let pinchDist = 0, pinchMid = null;
    c.addEventListener('pointerdown', (e) => {
      if (this.mode !== 'tilemap') return;
      try { c.setPointerCapture(e.pointerId); } catch {}
      pts.set(e.pointerId, { x: e.clientX, y: e.clientY });
      c.classList.add('dragging');
      if (pts.size === 2) {
        const [a, b] = [...pts.values()];
        pinchDist = Math.hypot(a.x - b.x, a.y - b.y);
        pinchMid = { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 };
      }
    });
    c.addEventListener('pointermove', (e) => {
      if (this.mode !== 'tilemap') return;
      const p = pts.get(e.pointerId);
      if (!p) return;
      const dx = e.clientX - p.x, dy = e.clientY - p.y;
      p.x = e.clientX; p.y = e.clientY;
      if (pts.size >= 2) {
        const [a, b] = [...pts.values()];
        const dist = Math.hypot(a.x - b.x, a.y - b.y);
        const mid = { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 };
        if (pinchMid) { this.world.position.x += mid.x - pinchMid.x; this.world.position.y += mid.y - pinchMid.y; }
        const sp = this._screenPt(mid.x, mid.y);
        this._zoomAt(sp.x, sp.y, pinchDist > 0 ? dist / pinchDist : 1);
        pinchDist = dist; pinchMid = mid;
      } else {
        this.world.position.x += dx; this.world.position.y += dy;
        this._clampPan();
      }
    });
    const end = (e) => {
      pts.delete(e.pointerId);
      try { c.releasePointerCapture(e.pointerId); } catch {}
      if (pts.size < 2) { pinchMid = null; pinchDist = 0; }
      if (pts.size === 0) c.classList.remove('dragging');
    };
    c.addEventListener('pointerup', end);
    c.addEventListener('pointercancel', end);
    c.addEventListener('wheel', (e) => {
      if (this.mode !== 'tilemap') return;
      e.preventDefault();
      const sp = this._screenPt(e.clientX, e.clientY);
      this._zoomAt(sp.x, sp.y, e.deltaY < 0 ? ZOOM_STEP : 1 / ZOOM_STEP);
    }, { passive: false });
  }
}
