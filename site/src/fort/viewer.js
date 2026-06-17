// Fort Apocalypse map viewer — PixiJS v8, no build step.
//
// The level is a 216x40 grid of 8x8 multicolor characters. Each of the 128
// playfield chars is baked once (from the pre-rendered atlas) into its own 8x8
// texture, and the map is one sprite per cell referencing that char's texture —
// so re-baking a char's texture animates every cell that uses it at once, the
// way the game's IRQ rewrites a soft char in place. Two control-bar toggles
// drive the soft-char animations and the object overlay.

import { Application, Container, Sprite, Texture, Graphics, Text } from 'pixi.js';

const CHAR = 8;            // character cell size (px)
const ATLAS_COLS = 16;     // atlas is 16 chars wide
const DATA = 'public/fort/';
const NATIVE_W = 320;      // C64 screen width (40 chars) — 1:1 reference
const ZOOM_STEP = Math.pow(1.15, 0.25);
const WRAP_COPIES = 3;     // cylinder is drawn as 3 copies; min zoom shows ≤2 periods

const OBJ = {              // marker colour + label per object type
  player: { color: 0x3cb4ff, label: 'COPTER' },
  prisoner: { color: 0xffe000, label: 'rescue' },
  tank: { color: 0xff5b5b, label: 'tank' },
  enemy: { color: 0x5be06b, label: 'heli' },
};

export class FortViewer {
  constructor(viewportEl, hudEl) {
    this.el = viewportEl;
    this.hud = hudEl;
    this.app = new Application();
    this.world = new Container();
    this.tileLayer = new Container();
    this.objectLayer = new Container();
    this.objectLayer.visible = false;
    this.zoom = 1; this.minZoom = 0.1; this.maxZoom = 12;
    this._texMode = 'nearest';
    this.animOn = true; this.animFrame = 0; this.animAccum = 0;
    this.level = null;
  }

  async init() {
    await this.app.init({ background: 0x000000, antialias: false, resizeTo: this.el });
    this.el.appendChild(this.app.canvas);
    this.world.addChild(this.tileLayer, this.objectLayer);
    this.app.stage.addChild(this.world);
    this._wireCamera();
    this.app.ticker.add(() => this._advanceAnim());
    return fetch(DATA + 'meta.json').then((r) => r.json());
  }

  _loadImage(src) {
    return new Promise((res, rej) => {
      const i = new Image();
      i.onload = () => res(i); i.onerror = rej; i.src = src;
    });
  }

  // Draw atlas tile `atlasIdx` into char `ch`'s 8x8 canvas and refresh its
  // texture (every cell sprite using that char then updates).
  _bakeChar(ch, atlasIdx) {
    const ctx = this.charCanvas[ch].getContext('2d');
    ctx.imageSmoothingEnabled = false;
    ctx.clearRect(0, 0, CHAR, CHAR);
    const sx = (atlasIdx % ATLAS_COLS) * CHAR, sy = ((atlasIdx / ATLAS_COLS) | 0) * CHAR;
    ctx.drawImage(this.atlasImg, sx, sy, CHAR, CHAR, 0, 0, CHAR, CHAR);
    if (this.charTex[ch]) this.charTex[ch].source.update();
  }

  async loadLevel(metaLevel) {
    const level = await fetch(DATA + metaLevel.file).then((r) => r.json());
    this.atlasImg = await this._loadImage(DATA + metaLevel.atlas);
    this.level = level;

    // Bake the 128 base char textures from the atlas (frame 0 = char index).
    this.charCanvas = []; this.charTex = [];
    for (let ch = 0; ch < 128; ch++) {
      const cv = document.createElement('canvas');
      cv.width = cv.height = CHAR;
      this.charCanvas[ch] = cv;
      this._bakeChar(ch, ch);
      const tx = Texture.from(cv);
      tx.source.autoGenerateMipmaps = true;
      tx.source.scaleMode = this._texMode;
      this.charTex[ch] = tx;
    }

    // Base tilemap. The playfield is a cylinder (column W-1 joins back to 0), so
    // it's drawn as WRAP_COPIES side-by-side copies one period (cyl px) apart;
    // the camera wraps horizontally across them for seamless scrolling.
    this.tileLayer.removeChildren();
    const { width: W, height: H, cells } = level;
    this.cyl = W * CHAR;
    for (let copy = 0; copy < WRAP_COPIES; copy++) {
      const ox = copy * this.cyl;
      for (let r = 0; r < H; r++) {
        for (let c = 0; c < W; c++) {
          const s = new Sprite(this.charTex[cells[r * W + c]]);
          s.x = ox + c * CHAR; s.y = r * CHAR;
          this.tileLayer.addChild(s);
        }
      }
    }
    this.levelW = this.cyl; this.levelH = H * CHAR;

    // Animation entries; reset to base if animation is currently off.
    this.anim = (level.anim || []).map((a) => ({ ...a, _last: 0 }));
    this.animFrame = 0; this.animAccum = 0;
    if (!this.animOn) this._resetAnim();

    this._buildObjects(level);
    this._fitDefault(level);
    return level;
  }

  _resetAnim() {
    for (const a of this.anim) { this._bakeChar(a.char, a.char); a._last = -1; }
  }

  _advanceAnim() {
    if (!this.animOn || !this.anim || !this.anim.length) return;
    this.animAccum += this.app.ticker.deltaMS;
    const step = 1000 / 60;
    while (this.animAccum >= step) { this.animAccum -= step; this.animFrame++; }
    for (const a of this.anim) {
      const idx = Math.floor(this.animFrame / a.period) % a.frames.length;
      if (idx !== a._last) { this._bakeChar(a.char, a.frames[idx]); a._last = idx; }
    }
  }

  // --- object markers -----------------------------------------------------
  _buildObjects(level) {
    this.objectLayer.removeChildren();
    // Markers are drawn once per cylinder copy so they wrap with the map.
    for (let copy = 0; copy < WRAP_COPIES; copy++) {
      const ox = copy * this.cyl;
      const g = new Graphics();
      for (const o of level.objects) {
        const def = OBJ[o.type] || { color: 0xaaaaaa };
        g.rect(ox + o.col * CHAR, o.row * CHAR, o.w * CHAR, o.h * CHAR).stroke({ width: 1, color: def.color });
      }
      for (const [cx, cy] of level.drops || []) {
        g.circle(ox + cx * CHAR + (4 * CHAR) / 2, cy * CHAR + (3 * CHAR) / 2, 10).stroke({ width: 1, color: 0xffffff });
      }
      this.objectLayer.addChild(g);
      for (const o of level.objects) {
        const def = OBJ[o.type];
        if (!def || !def.label || o.type !== 'player') continue;
        const t = new Text({ text: def.label, style: { fontFamily: 'monospace', fontSize: 7, fill: def.color } });
        t.x = ox + o.col * CHAR; t.y = o.row * CHAR - 9;
        this.objectLayer.addChild(t);
      }
    }
  }

  setLayer(name, on) {
    if (name === 'objects') this.objectLayer.visible = on;
    if (name === 'animation') {
      this.animOn = on;
      if (!on) this._resetAnim();
    }
  }

  // --- camera (shared pattern with the Sonic viewer) ----------------------
  _fitDefault(level) {
    const W = this.app.screen.width, H = this.app.screen.height;
    // Don't zoom out past two cylinder periods (so the 3 copies always cover the
    // view); the repetition makes the wrap visible. Fit height by default.
    this.minZoom = W / (2 * this.cyl);
    this.maxZoom = (W / NATIVE_W) * 3;
    this.zoom = Math.max(this.minZoom, Math.min(this.maxZoom, H / this.levelH));
    const [sx, sy] = level.spawn;
    this._panTo(sx * CHAR, sy * CHAR);
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
  // Vertical clamps to the map; horizontal wraps the cylinder: the camera x is
  // kept so the viewport's left edge maps into the first copy's [0, cyl) range,
  // which the three copies always cover.
  _clampPan() {
    const sh = this.app.screen.height;
    const lh = this.levelH * this.zoom;
    let { x, y } = this.world.position;
    y = lh <= sh ? (sh - lh) / 2 : Math.min(0, Math.max(sh - lh, y));
    const m = this.cyl * this.zoom;
    if (m > 0) x = -(((-x % m) + m) % m);
    this.world.position.set(x, y);
  }
  _apply() {
    this.world.scale.set(this.zoom);
    this._clampPan();
    this._updateTexFilter();
    if (this.hud) this.hud.textContent = `${this.levelW / CHAR}x${this.levelH / CHAR} chars · wraps`;
  }
  _updateTexFilter() {
    const mode = this.zoom < 1 ? 'linear' : 'nearest';
    if (mode === this._texMode) return;
    this._texMode = mode;
    if (this.charTex) for (const t of this.charTex) t.source.scaleMode = mode;
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
