// Sonic level viewer — PixiJS v8, no build step.
//
// The level is drawn as an actual tilemap: each distinct block index (0..255) is baked
// once to a 32x32 canvas from the tile atlas, and the map is one sprite per block cell
// referencing that cached texture (≤ 4096 sprites, all sharing ≤ 256 textures, so PixiJS
// batches them — whole-map zoom is cheap). Two further layers, toggled from the control
// bar, overlay the collision height-profiles and the object placements.

import { Application, Container, Sprite, Texture, Graphics, Text } from 'pixi.js';

const TILE = 8;
const BLOCK = 32;       // 4x4 tiles
const DATA = 'public/sonic/';

const OBJ_COLORS = {     // category tint by object name
  default: 0xaaaaaa,
  enemy: 0xff3838, item: 0xff9020, platform: 0x29d46e, boss: 0xc83cff, ctrl: 0xffe000,
};
const OBJ_CAT = {
  crab: 'enemy', beetle: 'enemy', fish: 'enemy', porcupine: 'enemy', bird: 'enemy',
  bonus: 'item', shield: 'item', emerald: 'item', goal: 'item', capsule: 'item',
  'swing platform': 'platform', 'moving platform': 'platform', seesaw: 'platform',
  'world 1 boss': 'boss', 'world 2 boss': 'boss', 'world 3 boss': 'boss', 'world 4 boss': 'boss',
  checkpoint: 'ctrl', 'scroll lock': 'ctrl',
};

export class LevelViewer {
  constructor(viewportEl, hudEl) {
    this.el = viewportEl;
    this.hud = hudEl;
    this.app = new Application();
    this.world = new Container();
    this.tileLayer = new Container();
    this.collisionLayer = new Container();
    this.objectLayer = new Container();
    this.zoom = 1; this.minZoom = 0.1; this.maxZoom = 12;
    this.atlasCache = new Map();   // atlas name -> HTMLImageElement
    this.shapes = null;
    this.level = null;
  }

  async init() {
    await this.app.init({ background: 0x000000, antialias: false, resizeTo: this.el });
    this.el.appendChild(this.app.canvas);
    this.world.addChild(this.tileLayer, this.collisionLayer, this.objectLayer);
    this.app.stage.addChild(this.world);
    this.shapes = await fetch(DATA + 'shapes.json').then((r) => r.json());
    this._wireCamera();
    return fetch(DATA + 'meta.json').then((r) => r.json());
  }

  // --- data loading -------------------------------------------------------
  async _atlas(name) {
    if (this.atlasCache.has(name)) return this.atlasCache.get(name);
    const img = await new Promise((res, rej) => {
      const i = new Image();
      i.onload = () => res(i); i.onerror = rej; i.src = DATA + name;
    });
    this.atlasCache.set(name, img);
    return img;
  }

  // Bake one 32x32 texture per distinct block index used in the map.
  _bakeBlocks(level, atlasImg) {
    const used = new Set(level.blocks);
    const tex = {};
    for (const idx of used) {
      const cv = document.createElement('canvas');
      cv.width = cv.height = BLOCK;
      const ctx = cv.getContext('2d');
      ctx.imageSmoothingEnabled = false;
      const tiles = level.blockTiles[idx];
      for (let r = 0; r < 4; r++) {
        for (let c = 0; c < 4; c++) {
          const t = tiles[r * 4 + c];
          const sx = (t % 16) * TILE, sy = ((t / 16) | 0) * TILE;
          ctx.drawImage(atlasImg, sx, sy, TILE, TILE, c * TILE, r * TILE, TILE, TILE);
        }
      }
      const tx = Texture.from(cv);
      tx.source.scaleMode = 'nearest';
      tex[idx] = tx;
    }
    return tex;
  }

  async loadAct(actMeta) {
    const level = await fetch(DATA + actMeta.file).then((r) => r.json());
    const atlasImg = await this._atlas(actMeta.atlas);
    this.level = level;

    // base tilemap
    this.tileLayer.removeChildren();
    const blockTex = this._bakeBlocks(level, atlasImg);
    const { widthBlocks: W, heightBlocks: H, blocks } = level;
    for (let r = 0; r < H; r++) {
      for (let c = 0; c < W; c++) {
        const tx = blockTex[blocks[r * W + c]];
        if (!tx) continue;
        const s = new Sprite(tx);
        s.x = c * BLOCK; s.y = r * BLOCK;
        this.tileLayer.addChild(s);
      }
    }
    this.levelW = W * BLOCK; this.levelH = H * BLOCK;

    this._buildCollision(level);
    this._buildObjects(level);
    this._fitDefault(level);
    return level;
  }

  // --- collision overlay --------------------------------------------------
  _buildCollision(level) {
    this.collisionLayer.removeChildren();
    const g = new Graphics();
    const { widthBlocks: W, heightBlocks: H, blocks, blockShape } = level;
    const profiles = this.shapes.profiles;
    for (let r = 0; r < H; r++) {
      for (let c = 0; c < W; c++) {
        const shape = blockShape[blocks[r * W + c]];
        const prof = profiles[shape];
        const solid = prof.some((h) => h !== -128);
        const ox = c * BLOCK, oy = r * BLOCK;
        if (!solid) continue; // non-solid: leave clear
        // surface polyline across the 32 columns, clamped into the cell
        let started = false;
        for (let x = 0; x < BLOCK; x++) {
          const h = prof[x];
          if (h === -128) { started = false; continue; }
          const y = oy + Math.max(0, Math.min(BLOCK - 1, h));
          if (!started) { g.moveTo(ox + x, y); started = true; }
          else g.lineTo(ox + x, y);
        }
      }
    }
    g.stroke({ width: 1, color: 0xff4040, alpha: 0.85 });
    // faint tint for non-solid blocks so "you fall through here" reads at a glance
    const t = new Graphics();
    for (let r = 0; r < H; r++) {
      for (let c = 0; c < W; c++) {
        const shape = blockShape[blocks[r * W + c]];
        if (shape !== 0 && this.shapes.profiles[shape].some((h) => h !== -128)) continue;
        t.rect(c * BLOCK, r * BLOCK, BLOCK, BLOCK);
      }
    }
    t.fill({ color: 0x3a7bd5, alpha: 0.10 });
    this.collisionLayer.addChild(t, g);
    this.collisionLayer.visible = false;
  }

  // --- object markers -----------------------------------------------------
  _buildObjects(level) {
    this.objectLayer.removeChildren();
    const mk = (bx, by, w, h, color, label) => {
      const g = new Graphics();
      g.rect(bx * BLOCK, by * BLOCK, w, h).stroke({ width: 2, color });
      this.objectLayer.addChild(g);
      if (label) {
        const txt = new Text({ text: label, style: { fontFamily: 'monospace', fontSize: 9, fill: color } });
        txt.x = bx * BLOCK; txt.y = by * BLOCK - 11;
        this.objectLayer.addChild(txt);
      }
    };
    for (const o of level.objects) {
      if (o.type === 0) continue; // Sonic handled as the spawn marker
      const cat = OBJ_CAT[o.name] || 'default';
      mk(o.bx, o.by, BLOCK, BLOCK, OBJ_COLORS[cat], o.name || '?' + o.type.toString(16));
    }
    // Sonic spawn: 2x4-tile box
    const [sx, sy] = level.spawn;
    mk(sx, sy, 16, 32, 0x3cb4ff, 'SONIC');
    this.objectLayer.visible = false;
  }

  setLayer(name, on) {
    if (name === 'collision') this.collisionLayer.visible = on;
    if (name === 'objects') this.objectLayer.visible = on;
  }

  // --- camera -------------------------------------------------------------
  _fitDefault(level) {
    const W = this.app.screen.width;
    this.minZoom = Math.min(W / this.levelW, this.app.screen.height / this.levelH) * 0.95;
    this.maxZoom = Math.max(12, (W / 160) * 2);
    this.zoom = Math.min(this.maxZoom, Math.max(this.minZoom, W / 160)); // ≈ one GG screen wide
    // centre on Sonic's spawn
    const [sx, sy] = level.spawn;
    this._panTo((sx * BLOCK + 8), (sy * BLOCK + 16));
    this._apply();
  }
  _panTo(wx, wy) {
    this.world.position.set(this.app.screen.width / 2 - wx * this.zoom, this.app.screen.height / 2 - wy * this.zoom);
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
    if (this.hud) this.hud.textContent = `${(this.zoom).toFixed(2)}x  ${this.levelW / BLOCK}x${this.levelH / BLOCK} blocks`;
  }
  _wireCamera() {
    const c = this.el;
    let dragging = false, lx = 0, ly = 0;
    c.addEventListener('pointerdown', (e) => { dragging = true; lx = e.clientX; ly = e.clientY; c.classList.add('dragging'); c.setPointerCapture(e.pointerId); });
    c.addEventListener('pointermove', (e) => {
      if (!dragging) return;
      this.world.position.x += e.clientX - lx; this.world.position.y += e.clientY - ly;
      lx = e.clientX; ly = e.clientY; this._clampPan();
    });
    const end = (e) => { dragging = false; c.classList.remove('dragging'); try { c.releasePointerCapture(e.pointerId); } catch {} };
    c.addEventListener('pointerup', end);
    c.addEventListener('pointercancel', end);
    c.addEventListener('wheel', (e) => {
      e.preventDefault();
      const rect = c.getBoundingClientRect();
      const px = (e.clientX - rect.left) * (this.app.screen.width / rect.width);
      const py = (e.clientY - rect.top) * (this.app.screen.height / rect.height);
      const wx = (px - this.world.position.x) / this.zoom, wy = (py - this.world.position.y) / this.zoom;
      const f = e.deltaY < 0 ? 1.15 : 1 / 1.15;
      this.zoom = Math.min(this.maxZoom, Math.max(this.minZoom, this.zoom * f));
      this.world.scale.set(this.zoom);
      this.world.position.set(px - wx * this.zoom, py - wy * this.zoom);
      this._apply();
    }, { passive: false });
  }
}
