// Super Mario Land level viewer — a tilemap rebuilt from the cartridge: each level is
// a row-major grid of 8×8 background tile indices (decoded by extract/level from the
// ROM), drawn from the world's 256-tile atlas. Drag to pan, scroll to zoom. The
// collision and object layers will hang off this later.
import { Application, Container, Graphics, Sprite, Texture, Rectangle } from 'pixi.js';
import { composeTilemap } from '../tilemap-compose.js';

const DATA = 'public/sml/';
const TILE = 8;
const NATIVE_H = 144; // the Game Boy screen height — the default vertical framing
const ZOOM_STEP = Math.pow(1.15, 0.25);

// Object/enemy marker palette, grouped by the ROM's object-type id ($401A placement
// list, decoded by extract/level). The exact per-type sprite isn't drawn (each type has
// bespoke handler graphics); the colour bands keep related types visually distinct.
const OBJ_COLORS = [
  0xff4040, 0xff8c1a, 0xffd21a, 0x46e05a, 0x33c6ff, 0x8a6dff, 0xff5ed0, 0xc0c0c0,
];
const objColor = (type) => OBJ_COLORS[type % OBJ_COLORS.length];

// Solidity is decided purely by the background tile id: Mario's foot check ($17B3) treats
// id >= $60 as floor and the enemy checks ($2B7B..) use [$5F,$F0); we use the shared range.
const isSolid = (id) => id >= 0x60 && id < 0xF0;

export class SMLViewer {
  constructor(viewportEl, hudEl) {
    this.el = viewportEl;
    this.hud = hudEl;
    this.app = new Application();
    this.world = new Container();
    this.zoom = 1; this.minZoom = 0.05; this.maxZoom = 16;
    this.layer = null;
    this.collisionLayer = null;
    this.objLayer = null;
    this.src = null;
    this._texMode = 'nearest';
    this._showObjects = true;
    this._showCollision = false;
  }

  // Display-layer toggle (studio adapter drives this).
  setLayer(id, on) {
    if (id === 'objects') {
      this._showObjects = on;
      if (this.objLayer) this.objLayer.visible = on;
    } else if (id === 'collision') {
      this._showCollision = on;
      if (this.collisionLayer) this.collisionLayer.visible = on;
    }
  }

  async init() {
    await this.app.init({ background: 0x0a0e16, antialias: false, resizeTo: this.el, preserveDrawingBuffer: true });
    this.app.canvas.style.imageRendering = 'pixelated';
    this.el.appendChild(this.app.canvas);
    this.app.stage.addChild(this.world);
    this._wireCamera();
    const meta = await fetch(DATA + 'meta.json').then((r) => r.json());
    return meta.levels;
  }

  _loadImage(src) {
    return new Promise((res, rej) => {
      const i = new Image();
      i.onload = () => res(i); i.onerror = rej; i.src = src;
    });
  }

  async loadLevel(meta) {
    this.name = meta.name;
    const level = await fetch(DATA + meta.file).then((r) => r.json());
    const atlas = await this._loadImage(DATA + level.atlas);
    if (this.layer) { this.world.removeChild(this.layer); this.layer.destroy({ children: true }); }
    const { container, src } = composeTilemap(atlas, level.cells, level.width, level.height, { tileSize: TILE });
    this.layer = container;
    this.src = src;
    this.world.addChild(this.layer);
    this._buildCollision(level);
    await this._buildObjects(level);
    this.levelW = level.width * TILE;
    this.levelH = level.height * TILE;
    this._fitDefault();
    const nObj = (level.objects || []).length;
    if (this.hud) this.hud.textContent = `${this.name} · ${level.width}×${level.height} tiles · ${nObj} objects`;
  }

  // Collision overlay: every solid 8x8 tile (the ones Mario and enemies stand on, decided
  // by isSolid on the tile id) is filled semi-transparent red. Adjacent solid cells in a row
  // are merged into one rect to keep the geometry light.
  _buildCollision(level) {
    if (this.collisionLayer) { this.world.removeChild(this.collisionLayer); this.collisionLayer.destroy({ children: true }); }
    const g = new Graphics();
    const W = level.width, H = level.height, cells = level.cells;
    for (let r = 0; r < H; r++) {
      let runStart = -1;
      const flush = (xEnd) => { if (runStart >= 0) { g.rect(runStart * TILE, r * TILE, (xEnd - runStart) * TILE, TILE); runStart = -1; } };
      for (let x = 0; x < W; x++) {
        if (isSolid(cells[r * W + x])) { if (runStart < 0) runStart = x; }
        else flush(x);
      }
      flush(W);
    }
    g.fill({ color: 0xff2020, alpha: 0.45 });
    this.collisionLayer = g;
    this.collisionLayer.visible = this._showCollision;
    this.world.addChild(this.collisionLayer);
  }

  // Object/enemy overlay: known types are drawn as their real metasprite (sliced from the
  // per-world object-icon atlas the exporter composites from ROM); unknown types fall back
  // to a type-coloured marker box. The two share one container so the layer toggles together.
  async _buildObjects(level) {
    if (this.objLayer) { this.world.removeChild(this.objLayer); this.objLayer.destroy({ children: true }); }
    const objects = level.objects || [];
    const layer = new Container();
    const g = new Graphics();
    layer.addChild(g);

    // Load this world's object-icon atlas once.
    const cell = level.objCell || 24;
    const [orgX, orgY] = level.objOrigin || [12, 12];
    const types = level.objTypes || {};
    let iconSrc = null;
    if (level.objAtlas) {
      const img = await this._loadImage(DATA + level.objAtlas);
      iconSrc = Texture.from(img).source;
      iconSrc.scaleMode = 'nearest';
    }

    for (const o of objects) {
      const idx = types[o.type];
      if (iconSrc && idx !== undefined) {
        // place the icon so its metasprite origin (objOrigin in the cell) lands on the
        // object's map-tile origin (col,row) — same as the offline render.
        const tex = new Texture({ source: iconSrc, frame: new Rectangle(0, idx * cell, cell, cell) });
        const sp = new Sprite(tex);
        sp.position.set(o.col * TILE - orgX, o.row * TILE - orgY);
        layer.addChild(sp);
        continue;
      }
      const x = o.col * TILE, y = o.row * TILE;
      g.rect(x + 0.5, y + 0.5, TILE - 1, TILE - 1).fill({ color: objColor(o.type), alpha: 0.55 });
      g.rect(x + 0.5, y + 0.5, TILE - 1, TILE - 1).stroke({ width: o.hard ? 1 : 0.5, color: o.hard ? 0xffffff : 0x101010, alpha: 0.9 });
    }

    // Mario at his fixed start position (his sprite top-left is composited at the cell
    // origin, so blit the same way as the objects: anchor px minus the cell origin).
    if (iconSrc && level.player) {
      const tex = new Texture({ source: iconSrc, frame: new Rectangle(0, level.player.icon * cell, cell, cell) });
      const sp = new Sprite(tex);
      sp.position.set(level.player.x - orgX, level.player.y - orgY);
      layer.addChild(sp);
    }

    this.objLayer = layer;
    this.objLayer.visible = this._showObjects;
    this.world.addChild(this.objLayer);
  }

  // --- camera (shared pattern with the Marble/Turrican viewers) -----------
  _fitDefault() {
    const W = this.app.screen.width, H = this.app.screen.height;
    // Frame the Game Boy's screen height (one screenful tall), start at the left edge.
    const z = H / NATIVE_H;
    this.minZoom = Math.min((W / this.levelW) * 0.9, z);
    this.maxZoom = Math.max((W / 160) * 6, z);
    this.zoom = Math.max(this.minZoom, Math.min(this.maxZoom, z));
    this.world.position.set(8, (H - this.levelH * this.zoom) / 2);
    this._apply();
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
    const mode = this.zoom < 1 ? 'linear' : 'nearest';
    if (mode !== this._texMode && this.src) { this._texMode = mode; this.src.scaleMode = mode; }
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
