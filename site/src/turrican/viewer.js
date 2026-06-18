// Turrican (Amiga) map viewer — PixiJS v8, no build step.
//
// Each scene is a grid of 32x32, 4-bitplane tiles streamed off the floppy and
// huff-decoded (see the write-up). The exporter bakes each world's tileset into
// one atlas PNG and each scene into a row-major `cells` grid; a cell value < the
// world's tile count is a tile index, and a value >= it is a horizontally flipped
// tile (index value-128). The map is one sprite per cell, drag to pan, scroll to
// zoom from the whole level down to a single tile.
//
// An optional object layer overlays the scene's objects: each placement (read off
// the disk by the scroll-triggered spawner's grid) is drawn at its pixel position
// using the first animation frame of the sprite its AI handler installs — packed
// into the world's object atlas (objSprites gives each sprite's rect, objects the
// per-object position + sprite index).

import { Application, Container, Graphics, Rectangle, Sprite, Texture } from 'pixi.js';

const TILE = 32;
const ATLAS_COLS = 16;
const ATLAS_GUTTER = 1; // each atlas tile is extruded by a 1px border (see webexport)
const ATLAS_CELL = TILE + 2 * ATLAS_GUTTER; // 34
const DATA = 'public/turrican/';
const NATIVE_W = 320; // Amiga playfield width (~10 tiles) — the 1:1 reference
const ZOOM_STEP = Math.pow(1.15, 0.25);

export class TurricanViewer {
  constructor(viewportEl, hudEl) {
    this.el = viewportEl;
    this.hud = hudEl;
    this.app = new Application();
    this.world = new Container();
    this.tileLayer = new Container();
    this.objLayer = new Container();
    this.collisionLayer = new Container(); // solid-tile overlay
    this.markerLayer = new Container();    // player spawn marker
    this.showObjects = true;
    this.showCollision = false;
    this.zoom = 1;
    this.minZoom = 0.05;
    this.maxZoom = 8;
    this.atlasTex = new Map(); // atlas name -> { source, tiles: Texture[] }
    this.objSrc = new Map();   // object-atlas name -> TextureSource
    this.level = null;
  }

  async init() {
    await this.app.init({ background: 0x101018, antialias: false, resizeTo: this.el });
    this.el.appendChild(this.app.canvas);
    this.world.addChild(this.tileLayer);
    this.world.addChild(this.objLayer);       // objects draw on top of the tiles
    this.world.addChild(this.collisionLayer); // collision overlay above those
    this.world.addChild(this.markerLayer);    // spawn marker on top of everything
    this.app.stage.addChild(this.world);
    this._wireCamera();
    return fetch(DATA + 'meta.json').then((r) => r.json());
  }

  _loadImage(src) {
    return new Promise((res, rej) => {
      const i = new Image();
      i.onload = () => res(i);
      i.onerror = rej;
      i.src = src;
    });
  }

  // Slice an atlas image into one Texture per tile (cached per atlas).
  async _atlas(name, nTiles) {
    if (this.atlasTex.has(name)) return this.atlasTex.get(name);
    const tex = Texture.from(await this._loadImage(DATA + name));
    tex.source.scaleMode = 'nearest';
    const cols = ATLAS_COLS;
    const tiles = [];
    for (let n = 0; n < nTiles; n++) {
      const sx = (n % cols) * ATLAS_CELL + ATLAS_GUTTER;
      const sy = ((n / cols) | 0) * ATLAS_CELL + ATLAS_GUTTER;
      tiles.push(new Texture({ source: tex.source, frame: new Rectangle(sx, sy, TILE, TILE) }));
    }
    const entry = { source: tex.source, tiles };
    this.atlasTex.set(name, entry);
    return entry;
  }

  // Load an object atlas's TextureSource (cached, nearest-filtered).
  async _objAtlasSource(name) {
    if (this.objSrc.has(name)) return this.objSrc.get(name);
    const tex = Texture.from(await this._loadImage(DATA + name));
    tex.source.scaleMode = 'nearest';
    this.objSrc.set(name, tex.source);
    return tex.source;
  }

  async loadLevel(metaLevel) {
    const level = await fetch(DATA + metaLevel.file).then((r) => r.json());
    this.level = level;
    const { width: W, height: H, ntiles, cells } = level;
    const atlas = await this._atlas(level.atlas, ntiles);

    this.tileLayer.removeChildren();
    for (let r = 0; r < H; r++) {
      for (let c = 0; c < W; c++) {
        const v = cells[r * W + c];
        const flip = v >= ntiles;
        let n = flip ? v - 128 : v;
        if (n < 0 || n >= ntiles) n = 0;
        const s = new Sprite(atlas.tiles[n]);
        if (flip) {
          s.scale.x = -1;
          s.x = c * TILE + TILE;
        } else {
          s.x = c * TILE;
        }
        s.y = r * TILE;
        this.tileLayer.addChild(s);
      }
    }
    // Object layer: one sprite per enemy placement, first animation frame.
    this.objLayer.removeChildren();
    if (level.objAtlas && level.objects?.length) {
      const src = await this._objAtlasSource(level.objAtlas);
      const frames = level.objSprites.map(
        (r) => new Texture({ source: src, frame: new Rectangle(r.x, r.y, r.w, r.h) }),
      );
      for (const o of level.objects) {
        const s = new Sprite(frames[o.s]);
        s.x = o.x;
        s.y = o.y;
        this.objLayer.addChild(s);
      }
    }
    this.objLayer.visible = this.showObjects;

    // Collision overlay: each tile's 4x4 grid of 8x8-block solidity (from the
    // $3C1C4 table), expanded over the map via the cell tile indices (flipped tiles
    // mirror their columns). Solid=red, special (hazard/water $7F/$D3)=blue. Built
    // as per-row run-length rectangles to stay light.
    this.collisionLayer.removeChildren();
    if (level.collision) {
      const coll = level.collision, bw = W * 4, bh = H * 4;
      const g = new Graphics();
      for (let br = 0; br < bh; br++) {
        const row = (br / 4) | 0, sr = br % 4;
        let runStart = 0, runVal = 0;
        for (let bc = 0; bc <= bw; bc++) {
          let v = 0;
          if (bc < bw) {
            const col = (bc / 4) | 0;
            let cv = cells[row * W + col];
            const flip = cv >= ntiles;
            let t = flip ? cv - 128 : cv;
            if (t < 0 || t >= ntiles) t = 0;
            const sc = flip ? 3 - (bc % 4) : bc % 4;
            v = coll[t * 16 + sr * 4 + sc];
          }
          if (v !== runVal) {
            if (runVal !== 0) {
              g.rect(runStart * 8, br * 8, (bc - runStart) * 8, 8)
                .fill({ color: runVal === 1 ? 0xff3030 : 0x3399ff, alpha: 0.45 });
            }
            runStart = bc;
            runVal = v;
          }
        }
      }
      this.collisionLayer.addChild(g);
    }
    this.collisionLayer.visible = this.showCollision;

    // Player spawn marker (a green crosshair ring at the spawn pixel).
    this.markerLayer.removeChildren();
    if (level.spawn) {
      const { x, y } = level.spawn;
      const r = 13;
      const g = new Graphics();
      g.circle(x, y, r).stroke({ width: 3, color: 0x33ff66 });
      g.moveTo(x - r - 7, y).lineTo(x + r + 7, y)
        .moveTo(x, y - r - 7).lineTo(x, y + r + 7).stroke({ width: 2, color: 0x33ff66 });
      g.circle(x, y, 2.5).fill(0x33ff66);
      this.markerLayer.addChild(g);
    }

    this.levelW = W * TILE;
    this.levelH = H * TILE;
    // Default camera: frame the Amiga on-screen view at the spawn (re-done on every
    // scene load), while still allowing zoom-out to the whole level.
    this._fitView(level.view);
    return level;
  }

  // Toggle the object overlay.
  setObjects(on) {
    this.showObjects = on;
    this.objLayer.visible = on;
  }

  // Toggle the collision (solid-tile) overlay.
  setCollision(on) {
    this.showCollision = on;
    this.collisionLayer.visible = on;
  }

  // --- camera (shared pattern with the Fort/Sonic viewers) ----------------
  // Frame the Amiga on-screen viewport (world-pixel rect `view`) centred on the
  // spawn, at 1:1-ish zoom; zoom-out still reaches the whole level.
  _fitView(view) {
    const W = this.app.screen.width, H = this.app.screen.height;
    const fitAll = Math.min(W / this.levelW, H / this.levelH);
    const v = view || { x: 0, y: 0, w: this.levelW, h: this.levelH };
    const z = Math.min(W / v.w, H / v.h); // contain the visible area in the viewport
    this.minZoom = Math.min(fitAll * 0.9, z);
    this.maxZoom = Math.max((W / NATIVE_W) * 4, z);
    this.zoom = Math.max(this.minZoom, Math.min(this.maxZoom, z));
    // Centre the view rect in the viewport.
    const cx = v.x + v.w / 2, cy = v.y + v.h / 2;
    this.world.position.set(W / 2 - cx * this.zoom, H / 2 - cy * this.zoom);
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
  // Clamp so the map can't be dragged entirely off-screen (centre if it fits).
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
    if (this.hud && this.level) {
      const n = this.level.objects?.length || 0;
      this.hud.textContent = `${this.level.width}×${this.level.height} tiles` + (n ? ` · ${n} objects` : '');
    }
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
