// Marble Madness course viewer — PixiJS v8, no build step.
//
// Each course is the assembled playfield tilemap, decoded from its .mlb and
// baked to one image by the exporter (the isometric look lives in the 8x8 tile
// art). The viewer shows that image with the shared drag-to-pan / scroll-to-zoom
// camera and a dropdown to switch courses. (The height field — the surface the
// marble actually rolls on — is a separate layer for later.)

import { Application, Container, Sprite, Texture } from 'pixi.js';

const DATA = 'public/marble/';
const NATIVE_W = 288;                    // the playfield is 288 px (36 tiles) wide
const ZOOM_STEP = Math.pow(1.15, 0.25);

export class MarbleViewer {
  constructor(viewportEl, hudEl) {
    this.el = viewportEl;
    this.hud = hudEl;
    this.app = new Application();
    this.world = new Container();
    this.zoom = 1; this.minZoom = 0.1; this.maxZoom = 12;
    this._texMode = 'nearest';
    this.sprite = null;
  }

  async init() {
    await this.app.init({ background: 0x000000, antialias: false, resizeTo: this.el });
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
    const img = await this._loadImage(DATA + metaLevel.file);
    const tx = Texture.from(img);
    tx.source.autoGenerateMipmaps = true;
    tx.source.scaleMode = this._texMode;
    if (this.sprite) { this.world.removeChild(this.sprite); this.sprite.destroy(); }
    this.sprite = new Sprite(tx);
    this.world.addChild(this.sprite);
    this.tex = tx;
    this.levelW = img.width; this.levelH = img.height;
    this.name = metaLevel.name;
    this._fitDefault();
  }

  // --- camera (shared pattern with the Sonic/Fort viewers) ----------------
  _fitDefault() {
    const W = this.app.screen.width, H = this.app.screen.height;
    this.minZoom = Math.min(W / this.levelW, H / this.levelH) * 0.95;
    this.maxZoom = (W / NATIVE_W) * 3;
    this.zoom = Math.max(this.minZoom, Math.min(this.maxZoom, (W / this.levelW) * 0.98)); // fit width
    this._panTo(this.levelW / 2, 0); // start at the top of the course
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
    if (this.hud) this.hud.textContent = `${this.name} · ${this.levelW}x${this.levelH}`;
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
      e.preventDefault();
      const sp = this._screenPt(e.clientX, e.clientY);
      this._zoomAt(sp.x, sp.y, e.deltaY < 0 ? ZOOM_STEP : 1 / ZOOM_STEP);
    }, { passive: false });
  }
}
