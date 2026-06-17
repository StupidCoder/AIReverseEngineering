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
const GG_W = 160;                       // Game Gear visible width (px); max zoom-in = GG 1:1
const ZOOM_STEP = Math.pow(1.15, 0.25); // per wheel notch — a quarter of the old 1.15 in log space

const hexToRgb = (h) => { const n = parseInt(h.slice(1), 16); return [(n >> 16) & 255, (n >> 8) & 255, n & 255]; };

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
    this.collisionLayer.visible = false; // toggled by the control bar; persists across acts
    this.objectLayer.visible = false;
    this.zoom = 1; this.minZoom = 0.1; this.maxZoom = 12;
    this._texMode = 'nearest';
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
    this.animOn = true; this.animTick = 0; this.animAccum = 0;
    this.app.ticker.add(() => this._advanceAnim());
    const meta = await fetch(DATA + 'meta.json').then((r) => r.json());
    this.framesPerTick = (meta.anim && meta.anim.framesPerTick) || 10;
    return meta;
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

  // tileFrames[srcTile] = [atlasIndex per animation frame]; built from level.anim.
  _buildAnim(level) {
    this.tileFrames = {};
    for (const g of level.anim || []) {
      for (let i = 0; i < 4; i++) {
        const src = g.frames[0][i];
        this.tileFrames[src] = g.frames.map((fr) => fr[i]);
      }
    }
  }

  // Paint block `idx`'s 4x4 tiles from the atlas into `ctx` at animation tick `t`
  // (animated tiles pick the current frame's atlas tile; others use their static index).
  _paintBlock(ctx, idx, t) {
    ctx.imageSmoothingEnabled = false;
    const tiles = this.level.blockTiles[idx];
    for (let r = 0; r < 4; r++) {
      for (let c = 0; c < 4; c++) {
        let tile = tiles[r * 4 + c];
        const fr = this.tileFrames[tile];
        if (fr) tile = fr[t % fr.length];
        const sx = (tile % 16) * TILE, sy = ((tile / 16) | 0) * TILE;
        ctx.drawImage(this.atlasImg, sx, sy, TILE, TILE, c * TILE, r * TILE, TILE, TILE);
      }
    }
  }

  // Draw block `idx` into its canvas at animation tick `t`, then flag the texture.
  _drawBlock(idx, t) {
    this._paintBlock(this.blockCanvas[idx].getContext('2d'), idx, t);
    if (this.blockTex[idx]) this.blockTex[idx].source.update();
  }

  // Underwater variant: paint the block, then remap each surface BG colour to its
  // underwater counterpart (the Labyrinth raster-split palette, Part V §3). Used for the
  // map rows at/below the water line; these never run the palette cycle (they're static).
  _drawBlockUW(idx, t) {
    const ctx = this.blockCanvasUW[idx].getContext('2d', { willReadFrequently: true });
    this._paintBlock(ctx, idx, t);
    const img = ctx.getImageData(0, 0, BLOCK, BLOCK), d = img.data;
    for (let p = 0; p < d.length; p += 4) {
      const u = this.water.map.get((d[p] << 16) | (d[p + 1] << 8) | d[p + 2]);
      if (u) { d[p] = u[0]; d[p + 1] = u[1]; d[p + 2] = u[2]; }
    }
    ctx.putImageData(img, 0, 0);
    if (this.blockTexUW[idx]) this.blockTexUW[idx].source.update();
  }

  // Bake one 32x32 texture per distinct block index used in the map (frame 0). When the act
  // is flooded (Labyrinth), also bake an underwater-palette variant of each block for the
  // map rows below the water line.
  _bakeBlocks(level) {
    this.blockCanvas = {}; this.blockTex = {}; this.animBlocks = [];
    this.blockCanvasUW = {}; this.blockTexUW = {};
    const animSet = new Set(Object.keys(this.tileFrames).map(Number));
    for (const idx of new Set(level.blocks)) {
      const cv = document.createElement('canvas');
      cv.width = cv.height = BLOCK;
      this.blockCanvas[idx] = cv;
      this._drawBlock(idx, 0);
      const tx = Texture.from(cv);
      tx.source.autoGenerateMipmaps = true;           // mipmaps: no moiré when zoomed out
      tx.source.scaleMode = this._texMode || 'nearest';
      this.blockTex[idx] = tx;
      if (this.water) {
        const cvu = document.createElement('canvas');
        cvu.width = cvu.height = BLOCK;
        this.blockCanvasUW[idx] = cvu;
        this._drawBlockUW(idx, 0);
        const txu = Texture.from(cvu);
        txu.source.autoGenerateMipmaps = true;
        txu.source.scaleMode = this._texMode || 'nearest';
        this.blockTexUW[idx] = txu;
      }
      if (level.blockTiles[idx].some((t) => animSet.has(t))) this.animBlocks.push(idx);
      // a "cycle block" contains a tile that genuinely uses a cycling palette slot (not just
      // a tile that shares the colour at rest — that's the sky/fruit, which must not blink)
      if (this.cycleColors) {
        const cells = [];
        for (let cell = 0; cell < 16; cell++) if (this.waterTiles.has(level.blockTiles[idx][cell])) cells.push(cell);
        if (cells.length) {
          this.cycleBlocks.push(idx);
          this.blockWaterCells[idx] = cells;
          this.blockClean[idx] = cv.getContext('2d').getImageData(0, 0, BLOCK, BLOCK);
        }
      }
    }
  }

  // Recolour each cycle block to palette step `s`, but only within its cycling-tile cells
  // (8x8), remapping the cycling colours from the clean step-0 pixels. Other cells stay put.
  _applyCycleStep(s) {
    for (const idx of this.cycleBlocks) {
      const clean = this.blockClean[idx].data;
      const ctx = this.blockCanvas[idx].getContext('2d');
      const out = ctx.createImageData(BLOCK, BLOCK);
      const dst = out.data;
      dst.set(clean);
      for (const cell of this.blockWaterCells[idx]) {
        const cx = (cell % 4) * 8, cy = (cell >> 2) * 8;
        for (let y = 0; y < 8; y++) {
          for (let x = 0; x < 8; x++) {
            const p = ((cy + y) * BLOCK + (cx + x)) * 4;
            const r = dst[p], g = dst[p + 1], b = dst[p + 2];
            for (let i = 0; i < this.cycleFrom.length; i++) {
              const f = this.cycleFrom[i];
              if (r === f[0] && g === f[1] && b === f[2]) { const t = this.cycleColors[s][i]; dst[p] = t[0]; dst[p + 1] = t[1]; dst[p + 2] = t[2]; break; }
            }
          }
        }
      }
      ctx.putImageData(out, 0, 0);
      this.blockTex[idx].source.update();
    }
  }

  _advanceAnim() {
    if (!this.animOn) return;
    const dt = this.app.ticker.deltaMS;
    // tile animation (rings, flowers)
    if (this.animBlocks && this.animBlocks.length) {
      this.animAccum += dt;
      const period = 1000 * (this.framesPerTick || 10) / 60;
      if (this.animAccum >= period) {
        this.animAccum = 0;
        this.animTick = (this.animTick + 1) | 0;
        for (const idx of this.animBlocks) {
          this._drawBlock(idx, this.animTick);
          if (this.water && this.blockTexUW[idx]) this._drawBlockUW(idx, this.animTick);
        }
      }
    }
    // palette cycle (water / lava)
    if (this.cycleColors && this.cycleBlocks.length) {
      this.cycleAccum += dt;
      const cperiod = 1000 * this.cyclePeriod / 60;
      if (this.cycleAccum >= cperiod) {
        this.cycleAccum = 0;
        this.cycleStep = (this.cycleStep + 1) % this.cycleColors.length;
        this._applyCycleStep(this.cycleStep);
      }
    }
  }

  async loadAct(actMeta) {
    const level = await fetch(DATA + actMeta.file).then((r) => r.json());
    this.atlasImg = await this._atlas(actMeta.atlas);
    this.level = level;
    this._buildAnim(level);
    this.animTick = 0; this.animAccum = 0;

    // palette cycle (water/lava): per-step colours for the cycling slots; step 0 = atlas.
    this.cycleColors = null; this.cycleBlocks = []; this.blockClean = {}; this.blockWaterCells = {};
    this.cycleStep = 0; this.cycleAccum = 0;
    if (level.paletteCycle) {
      this.cycleColors = level.paletteCycle.steps.map((step) => step.map(hexToRgb));
      this.cycleFrom = this.cycleColors[0];
      this.cyclePeriod = level.paletteCycle.periodFrames;
      this.waterTiles = new Set(level.paletteCycle.tiles); // tiles that actually use a cycling slot
    }

    // Labyrinth underwater split (Part V §3): below the water line the engine raster-swaps
    // the BG palette to a static underwater set. The line sits on a block-row boundary, so
    // rows at/below water.row use the underwater-baked block textures (no cycle).
    this.water = null;
    if (level.water) {
      const surf = level.palette.map(hexToRgb), uw = level.water.palette.map(hexToRgb);
      const map = new Map();
      for (let i = 0; i < uw.length; i++) map.set((surf[i][0] << 16) | (surf[i][1] << 8) | surf[i][2], uw[i]);
      this.water = { row: Math.round(level.water.lineY / BLOCK), map };
    }

    // base tilemap
    this.tileLayer.removeChildren();
    this._bakeBlocks(level);
    const { widthBlocks: W, heightBlocks: H, blocks } = level;
    for (let r = 0; r < H; r++) {
      const underwater = this.water && r >= this.water.row;
      for (let c = 0; c < W; c++) {
        const blk = blocks[r * W + c];
        const tx = (underwater && this.blockTexUW[blk]) ? this.blockTexUW[blk] : this.blockTex[blk];
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
  // Each block's collision shape is drawn as a semi-transparent red FILL of its solid
  // region: per pixel-column the surface height (signed; <0 = solid from top, >=32 or
  // -128 = no solid here) marks the top of the fill, which runs down to the block bottom.
  // Adjacent equal-height columns are merged into one rect to keep the geometry light.
  _buildCollision(level) {
    this.collisionLayer.removeChildren();
    const g = new Graphics();
    const { widthBlocks: W, heightBlocks: H, blocks, blockShape } = level;
    const profiles = this.shapes.profiles;
    for (let r = 0; r < H; r++) {
      for (let c = 0; c < W; c++) {
        const prof = profiles[blockShape[blocks[r * W + c]]];
        const ox = c * BLOCK, oy = r * BLOCK;
        let runStart = -1, runTop = 0;
        const flush = (xEnd) => { if (runStart >= 0) g.rect(ox + runStart, oy + runTop, xEnd - runStart, BLOCK - runTop); runStart = -1; };
        for (let x = 0; x < BLOCK; x++) {
          const h = prof[x];
          let top = null;
          if (h !== -128) { const s = Math.max(0, Math.min(BLOCK, h)); if (s < BLOCK) top = s; }
          if (top === null) { flush(x); }
          else if (runStart < 0) { runStart = x; runTop = top; }
          else if (top !== runTop) { flush(x); runStart = x; runTop = top; }
        }
        flush(BLOCK);
      }
    }
    g.fill({ color: 0xff2020, alpha: 0.8 });
    this.collisionLayer.addChild(g);
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
  }

  setLayer(name, on) {
    if (name === 'collision') this.collisionLayer.visible = on;
    if (name === 'objects') this.objectLayer.visible = on;
    if (name === 'animation') {
      this.animOn = on;
      if (!on) {
        if (this.animBlocks) {
          this.animTick = 0;
          for (const i of this.animBlocks) { this._drawBlock(i, 0); if (this.water && this.blockTexUW[i]) this._drawBlockUW(i, 0); }
        }
        if (this.cycleColors && this.cycleBlocks.length) { this.cycleStep = 0; this._applyCycleStep(0); }
      }
    }
  }

  // --- camera -------------------------------------------------------------
  _fitDefault(level) {
    const W = this.app.screen.width;
    this.minZoom = Math.min(W / this.levelW, this.app.screen.height / this.levelH) * 0.95;
    this.maxZoom = W / GG_W;                                  // GG 1:1 — never magnify past the original viewport
    this.zoom = Math.min(this.maxZoom, Math.max(this.minZoom, W / GG_W)); // start at the GG screen
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
    this._updateTexFilter();
    if (this.hud) this.hud.textContent = `${(this.zoom).toFixed(2)}x  ${this.levelW / BLOCK}x${this.levelH / BLOCK} blocks`;
  }

  // Crisp nearest-neighbour when magnifying (zoom >= 1), but linear + mipmaps when minifying
  // (zoomed out) so the downscaled tiles don't moiré/shimmer.
  _updateTexFilter() {
    const mode = this.zoom < 1 ? 'linear' : 'nearest';
    if (mode === this._texMode) return;
    this._texMode = mode;
    for (const idx in this.blockTex) this.blockTex[idx].source.scaleMode = mode;
    if (this.blockTexUW) for (const idx in this.blockTexUW) this.blockTexUW[idx].source.scaleMode = mode;
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
      const f = e.deltaY < 0 ? ZOOM_STEP : 1 / ZOOM_STEP;
      this.zoom = Math.min(this.maxZoom, Math.max(this.minZoom, this.zoom * f));
      this.world.scale.set(this.zoom);
      this.world.position.set(px - wx * this.zoom, py - wy * this.zoom);
      this._apply();
    }, { passive: false });
  }
}
