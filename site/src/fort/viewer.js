// Fort Apocalypse map viewer — PixiJS v8, no build step.
//
// The level is a 216x40 grid of 8x8 multicolor characters. Each of the 128
// playfield chars is baked once (from the pre-rendered atlas) into its own 8x8
// texture, and the map is one sprite per cell referencing that char's texture —
// so re-baking a char's texture animates every cell that uses it at once, the
// way the game's IRQ rewrites a soft char in place. Two control-bar toggles
// drive the soft-char animations and the object overlay.

import { Application, Container, Sprite, Texture } from 'pixi.js';

const CHAR = 8;            // character cell size (px)
const ATLAS_COLS = 16;     // atlas is 16 chars wide
const DATA = 'public/fort/';
const NATIVE_W = 320;      // C64 screen width (40 chars) — 1:1 reference
const ZOOM_STEP = Math.pow(1.15, 0.25);
const WRAP_COPIES = 3;     // cylinder is drawn as 3 copies; min zoom shows ≤2 periods

// SPMs spawn in the column band $32–$CD; buffer col = game col − 5.
const SPM_MIN = 0x32 - 5, SPM_MAX = 0xCD - 5;

// Fisher-Yates shuffle of a copy (leaves the source array untouched).
function shuffle(arr) {
  const a = arr.slice();
  for (let i = a.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [a[i], a[j]] = [a[j], a[i]];
  }
  return a;
}

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
    this.objectsOn = false;
    this.level = null;
  }

  async init() {
    await this.app.init({ background: 0x000000, antialias: false, resizeTo: this.el, preserveDrawingBuffer: true });
    this.el.appendChild(this.app.canvas);
    this.world.addChild(this.tileLayer, this.objectLayer);
    this.app.stage.addChild(this.world);
    this._wireCamera();
    this.app.ticker.add(() => this._advanceAnim());
    // Helicopter sprites (white on transparent → tinted per side when placed).
    this.chopperFwd = await this._loadTex('chopper-fwd.png');
    this.chopperSide = await this._loadTex('chopper-side.png');
    return fetch(DATA + 'meta.json').then((r) => r.json());
  }

  async _loadTex(name) {
    const tx = Texture.from(await this._loadImage(DATA + name));
    tx.source.scaleMode = 'nearest';
    return tx;
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

    this.level = level;
    this.objectLayer.removeChildren();
    if (this.objectsOn) this._placeObjects();
    this._fitDefault(level);
    return level;
  }

  _resetAnim() {
    for (const a of this.anim) { this._bakeChar(a.char, a.char); a._last = -1; }
  }

  _advanceAnim() {
    if (!this.animOn || !this.anim || !this.anim.length) return;
    this.animAccum += this.app.ticker.deltaMS;
    const step = 1000 / 50; // PAL frame — the soft-char periods are in game frames
    while (this.animAccum >= step) { this.animAccum -= step; this.animFrame++; }
    for (const a of this.anim) {
      const idx = Math.floor(this.animFrame / a.period) % a.frames.length;
      if (idx !== a._last) { this._bakeChar(a.char, a.frames[idx]); a._last = idx; }
    }
  }

  // --- objects ------------------------------------------------------------
  // Place the objects from their real characters, re-randomising each time, the
  // way the level builder seeds them: 8 prisoners chosen from the floor-with-rock
  // candidates, the 6 tanks at their fixed homes, and spmCount mines scattered
  // over empty cells. Each is decided once, then drawn on all wrap copies.
  _placeObjects() {
    this.objectLayer.removeChildren();
    const level = this.level;
    if (!level) return;

    // [{ code, col, row }] cell placements, decided once then drawn per copy.
    const place = [];
    const rnd = Math.random;
    // Prisoner: 1 wide, 2 tall — torso $49 over legs (right $3B / left $3D); the
    // paired codes ($49/$4A, $3B/$3C, $3D/$3E) are leg-phase animation frames, not
    // separate columns. Random facing per prisoner.
    for (const [c, r] of shuffle(level.prisoners || []).slice(0, 8)) {
      place.push({ code: 0x49, col: c, row: r - 1 });
      place.push({ code: rnd() < 0.5 ? 0x3B : 0x3D, col: c, row: r });
    }
    // Tank: body $6C $6D $6E with a turret ($6F left / $70 right) above the centre.
    for (const [c, r] of level.tanks || []) {
      place.push({ code: 0x6C, col: c, row: r }, { code: 0x6D, col: c + 1, row: r }, { code: 0x6E, col: c + 2, row: r });
      place.push({ code: rnd() < 0.5 ? 0x6F : 0x70, col: c + 1, row: r - 1 });
    }
    // SPM: a 2-cell craft $5B $5C at random empty positions.
    for (const [c, r] of this._spmPositions(level, level.spmCount || 13)) {
      place.push({ code: 0x5B, col: c, row: r }, { code: 0x5C, col: c + 1, row: r });
    }

    // Helicopters (sprites, not chars): the player's at its spawn (yellow,
    // facing ahead) and one enemy at a random open spot (blue, facing sideways).
    const [psx, psy] = level.spawn;
    const enemy = this._enemyPos(level);

    for (let copy = 0; copy < WRAP_COPIES; copy++) {
      const ox = copy * this.cyl;
      for (const p of place) {
        const tex = this.charTex[p.code];
        if (!tex) continue;
        const s = new Sprite(tex);
        s.x = ox + p.col * CHAR; s.y = p.row * CHAR;
        this.objectLayer.addChild(s);
      }
      if (this.chopperFwd) this._chopper(this.chopperFwd, 0xb8c76f, psx, psy, ox);
      if (enemy && this.chopperSide) this._chopper(this.chopperSide, 0x352879, enemy[0], enemy[1], ox);
    }
  }

  _chopper(tex, tint, col, row, ox) {
    const s = new Sprite(tex);
    s.tint = tint;
    s.x = ox + col * CHAR; s.y = row * CHAR;
    this.objectLayer.addChild(s);
  }

  // A random open spot (4×2 empty cells) for the enemy helicopter.
  _enemyPos(level) {
    const { width: W, height: H, cells } = level;
    const clear = (c, r) => {
      for (let dy = 0; dy < 2; dy++) for (let dx = 0; dx < 4; dx++) if (cells[(r + dy) * W + (c + dx)] !== 0) return false;
      return true;
    };
    const hi = Math.min(SPM_MAX, W - 4);
    for (let tries = 0; tries < 3000; tries++) {
      const c = SPM_MIN + Math.floor(Math.random() * (hi - SPM_MIN + 1));
      const r = Math.floor(Math.random() * (H - 1));
      if (clear(c, r)) return [c, r];
    }
    return null;
  }

  // SPM spawn: random positions whose two cells are both empty ($00), in the
  // column band, on any row — the game's "re-roll until both cells are empty".
  _spmPositions(level, n) {
    const { width: W, height: H, cells } = level;
    const out = [], used = new Set();
    const hi = Math.min(SPM_MAX, W - 2);
    for (let tries = 0; out.length < n && tries < n * 300; tries++) {
      const c = SPM_MIN + Math.floor(Math.random() * (hi - SPM_MIN + 1));
      const r = Math.floor(Math.random() * H);
      const key = r * W + c;
      if (used.has(key)) continue;
      if (cells[r * W + c] === 0 && cells[r * W + c + 1] === 0) { used.add(key); out.push([c, r]); }
    }
    return out;
  }

  setLayer(name, on) {
    if (name === 'objects') {
      this.objectsOn = on;
      this.objectLayer.visible = on;
      if (on) this._placeObjects(); // new random placement each time it's shown
    }
    if (name === 'animation') {
      this.animOn = on;
      if (!on) this._resetAnim();
    }
  }

  // --- camera (shared pattern with the Sonic viewer) ----------------------
  _fitDefault(level) {
    const W = this.app.screen.width, H = this.app.screen.height;
    // Don't zoom out past one cylinder period: the whole course is one loop, and
    // showing more would repeat the objects (e.g. all 8 prisoners twice).
    this.minZoom = W / this.cyl;
    this.maxZoom = (W / NATIVE_W) * 3;
    // Default view: the top half of the level fills the viewport vertically
    // (height = levelH/2 worth of world px), its top edge at the top, and the
    // player spawn centred horizontally.
    this.zoom = (2 * H) / this.levelH;
    const [sx] = level.spawn;
    const centreX = (sx + 2) * CHAR; // centre of the 4-char-wide copter
    this.world.position.set(W / 2 - centreX * this.zoom, 0);
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
