// Sonic level viewer — PixiJS v8, no build step.
//
// The level is drawn as an actual tilemap: each distinct block index (0..255) is baked
// once to a 32x32 canvas from the tile atlas, and the map is one sprite per block cell
// referencing that cached texture (≤ 4096 sprites, all sharing ≤ 256 textures, so PixiJS
// batches them — whole-map zoom is cheap). Two further layers, toggled from the control
// bar, overlay the collision height-profiles and the object placements.

import { Application, Container, Sprite, Texture, Graphics, Text, Rectangle } from 'pixi.js';
import { MapCamera } from '../shared/camera.js';

const TILE = 8;
const BLOCK = 32;       // 4x4 tiles
const DATA = 'public/sonic/';
const GG_W = 160;                       // Game Gear visible width (px); max zoom-in = GG 1:1
const GG_H = 144;                       // Game Gear visible height (px); default view fits this
const ZOOM_STEP = Math.pow(1.15, 0.25); // per wheel notch — a quarter of the old 1.15 in log space

const hexToRgb = (h) => { const n = parseInt(h.slice(1), 16); return [(n >> 16) & 255, (n >> 8) & 255, n & 255]; };

const OBJ_COLORS = {     // category tint by object name
  default: 0xaaaaaa,
  enemy: 0xff3838, item: 0xff9020, platform: 0x29d46e, boss: 0xc83cff, ctrl: 0xffe000,
};
const OBJ_CAT = {
  crab: 'enemy', beetle: 'enemy', fish: 'enemy', porcupine: 'enemy', bird: 'enemy',
  bonus: 'item', shield: 'item', emerald: 'item', goal: 'item', capsule: 'item', ring: 'item',
  'swing platform': 'platform', 'moving platform': 'platform', seesaw: 'platform',
  'world 1 boss': 'boss', 'world 2 boss': 'boss', 'world 3 boss': 'boss', 'world 4 boss': 'boss',
  checkpoint: 'ctrl', 'bg animator': 'ctrl',
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
    this._texMode = 'nearest';
    this.atlasCache = new Map();   // atlas name -> HTMLImageElement
    this.shapes = null;
    this.level = null;
  }

  async init() {
    await this.app.init({ background: 0x000000, antialias: false, resizeTo: this.el, preserveDrawingBuffer: true });
    this.el.appendChild(this.app.canvas);
    this.world.addChild(this.tileLayer, this.collisionLayer, this.objectLayer);
    this.app.stage.addChild(this.world);
    this.shapes = await fetch(DATA + 'shapes.json').then((r) => r.json());
    await this._loadSprites();
    this.cam = new MapCamera(this, { onApply: () => {
      this._updateTexFilter();
      if (this.hud) this.hud.textContent = `${this.levelW / BLOCK}x${this.levelH / BLOCK} blocks`;
    } });
    this.cam.wirePointer();
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

  // Background-cell animators (object type $50, Part V §1): each repaints its own
  // 16x32 strip with a 4-phase block sequence — the growing flowers / sea twinkle.
  // One sprite per placement in the tile layer; phase textures baked from the atlas,
  // advanced with the engine's own per-phase frame counts.
  _buildCellAnims(level) {
    this.cellAnims = [];
    for (const ca of level.cellAnims || []) {
      const texs = ca.phases.map((ph) => {
        const cv = document.createElement('canvas');
        cv.width = 2 * TILE; cv.height = 4 * TILE;
        const ctx = cv.getContext('2d');
        ctx.imageSmoothingEnabled = false;
        ph.tiles.forEach((tile, i) => {
          const sx = (tile % 16) * TILE, sy = ((tile / 16) | 0) * TILE;
          ctx.drawImage(this.atlasImg, sx, sy, TILE, TILE, (i % 2) * TILE, ((i / 2) | 0) * TILE, TILE, TILE);
        });
        const tx = Texture.from(cv);
        tx.source.scaleMode = 'nearest';
        return tx;
      });
      const s = new Sprite(texs[0]);
      s.x = ca.tx * TILE; s.y = ca.ty * TILE;
      this.tileLayer.addChild(s);
      this.cellAnims.push({ sprite: s, texs, durs: ca.phases.map((p) => Math.max(1, p.frames)), idx: 0, acc: 0 });
    }
  }

  _advanceAnim() {
    if (!this.animOn) return;
    const dt = this.app.ticker.deltaMS;
    const df = dt * 60 / 1000; // elapsed engine frames (60 Hz)
    // background-cell animators: per-phase durations
    for (const c of this.cellAnims || []) {
      c.acc += df;
      while (c.acc >= c.durs[c.idx]) {
        c.acc -= c.durs[c.idx];
        c.idx = (c.idx + 1) % c.texs.length;
        c.sprite.texture = c.texs[c.idx];
      }
    }
    // animated object sprites: (texture, hold) steps (enemy walks, Sonic idle/bored)
    for (const o of this.animObjs || []) {
      o.acc += df;
      while (o.acc >= o.steps[o.idx].frames) {
        o.acc -= o.steps[o.idx].frames;
        o.idx = (o.idx + 1) % o.steps.length;
        o.sprite.texture = o.steps[o.idx].tex;
      }
    }
    // moving platforms: per-frame (dx,dy) offsets from the placement
    for (const o of this.pathObjs || []) {
      o.t = (o.t + df) % o.path.length;
      const [dx, dy] = o.path[o.t | 0];
      o.sprite.x = o.baseX + dx;
      o.sprite.y = o.baseY + dy;
    }
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
    this._buildCellAnims(level);

    this._buildCollision(level);
    this.zoneSprites = await this._loadZoneSprites(level.zone);
    this._buildObjects(level);
    this._setMusicTrack(level.music);
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

  // --- music --------------------------------------------------------------
  // Per-act background music, synthesized from the ROM sound-driver data (extract/cmd/
  // musicrom). The track name comes from the act's descriptor (+36 music id); one <audio>
  // follows it, toggled by the Music checkbox.
  _setMusicTrack(track) {
    if (!track) { if (this.audio) this.audio.pause(); return; }
    const src = DATA + 'music/' + track + '.mp3';
    if (!this.audio) {
      this.audio = new Audio();
      this.audio.loop = true;
      this.audio.volume = 0.55;
    }
    if (this._musicSrc !== src) {
      this._musicSrc = src;
      this.audio.src = src;
    }
    if (this.musicOn) this.audio.play().catch(() => {});
    else this.audio.pause();
  }

  // Sprites extracted straight from the ROM (cmd/spriterip): every placed object type's
  // idle metasprite (Sonic = type 00, his standing frame from the bank-8 stream source),
  // rendered per zone with that zone's sprite tile set + palette. The index maps
  // zone -> type(hex) -> true; the PNGs live at sprites/<zone>/<hex>.png.
  async _loadSprites() {
    this.spriteTex = {};
    this.zoneSpriteTex = {};   // zone -> { typeNumber -> Texture }, lazily loaded
    this.spriteIndex = {};
    const load = (src) => new Promise((res, rej) => {
      const i = new Image();
      i.onload = () => res(i); i.onerror = rej; i.src = src;
    });
    const tex = async (src) => {
      const tx = Texture.from(await load(src));
      tx.source.scaleMode = 'nearest';
      return tx;
    };
    try { this.spriteIndex = await fetch(DATA + 'sprites/index.json').then((r) => r.json()); } catch { /* optional */ }
    // per-type movement paths (moving platforms): per-frame [dx,dy] offsets from the
    // placement, sampled from the handlers' own tables (swing arc $682E, the
    // horizontal platform's 160px triangle).
    this.spritePaths = (this.spriteIndex && this.spriteIndex.paths) || {};
  }

  // Lazily load (and cache) every object sprite for one zone.
  async _loadZoneSprites(zone) {
    if (this.zoneSpriteTex[zone]) return this.zoneSpriteTex[zone];
    const out = {};
    const types = this.spriteIndex[zone] || {};
    await Promise.all(Object.keys(types).map(async (hex) => {
      try {
        const base = Texture.from(await new Promise((res, rej) => {
          const i = new Image();
          i.onload = () => res(i); i.onerror = rej;
          i.src = DATA + 'sprites/' + zone + '/' + hex + '.png';
        }));
        base.source.scaleMode = 'nearest';
        // The PNG is a horizontal strip of 48x48 frames; the index carries the
        // per-frame durations (engine frames) extracted from the handler ($7C75
        // sequences / the pickup blink).
        const meta = types[hex];
        const n = (meta && meta.f) || 1;
        const texs = [];
        for (let i = 0; i < n; i++) {
          texs.push(new Texture({ source: base.source, frame: new Rectangle(i * 48, 0, 48, 48) }));
        }
        // play steps: an explicit sequence (meta.s: Sonic's idle -> bored program), or
        // the strip frames in order with their durations.
        let steps = null;
        if (meta && meta.s) {
          steps = meta.s.map(([f, d]) => ({ tex: texs[f], frames: Math.max(1, d) }));
        } else if (n > 1) {
          steps = texs.map((t, i) => ({ tex: t, frames: Math.max(1, (meta.d || [])[i] || 1) }));
        }
        out[parseInt(hex, 16)] = { texs, steps, path: this.spritePaths[hex] };
      } catch { /* skip */ }
    }));
    this.zoneSpriteTex[zone] = out;
    return out;
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
    // Each sprite PNG is the full 48x48 metasprite grid; the engine draws that grid with
    // its top-left at the object's world position ($2CD4 tail: screen = world - camera,
    // then $2F07 walks the grid from there). Each object's (x, y) is its engine REST
    // position: spawn + the pickup handlers' one-time adjust ($6089) + the shared floor
    // snap ($2CD4), reimplemented in objplace and verified live by cmd/objsettle. So the
    // sprite goes at (x, y) directly — the grid's transparent padding does the rest.
    this.animObjs = [];
    this.pathObjs = [];
    const sprite = (spr, x, y) => {
      const s = new Sprite(spr.steps ? spr.steps[0].tex : spr.texs[0]);
      s.x = x;
      s.y = y;
      this.objectLayer.addChild(s);
      if (spr.steps) {
        this.animObjs.push({ sprite: s, steps: spr.steps, idx: 0, acc: 0 });
      }
      if (spr.path) {
        this.pathObjs.push({ sprite: s, baseX: x, baseY: y, path: spr.path, t: 0 });
      }
    };
    // Each placed object draws its ROM-extracted sprite (this zone's set), animated when
    // the handler's $7C75 sequence gave multiple frames; types without an extractable
    // metasprite (invisible triggers, own-gfx loaders) fall back to a marker.
    const zoneSprites = this.zoneSprites || {};
    for (const o of level.objects) {
      if (o.type === 0x50) continue; // bg animators: their effect runs in the tile layer
      const spr = zoneSprites[o.type];
      if (spr) { sprite(spr, o.x, o.y); continue; }
      const cat = OBJ_CAT[o.name] || 'default';
      mk(o.bx, o.by, BLOCK, BLOCK, OBJ_COLORS[cat], o.name || '?' + o.type.toString(16));
    }
    // Sonic: type-00 sprite at his rest position — the spawn (descriptor +13/+14)
    // dropped to the first floor line, exactly where he stands when the fade-in ends.
    // In the acts where he spawns over a pit (spawnPx[2] = 0) this is the spawn itself,
    // mid-air: he really does fall into those levels.
    const [sx, sy] = level.spawnPx || [level.spawn[0] * BLOCK, level.spawn[1] * BLOCK];
    if (zoneSprites[0]) {
      sprite(zoneSprites[0], sx, sy);
    } else {
      mk(sx / BLOCK, sy / BLOCK, 16, 32, 0x3cb4ff, 'SONIC');
    }
    // (Sonic stays on his standing frame: his animation system streams tiles per pose
    // from bank 8 — decoding the walk/idle sequences is Part V follow-up work.)
  }

  setLayer(name, on) {
    if (name === 'music') {
      this.musicOn = on;
      if (this.level) this._setMusicTrack(this.level.music);
    }
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
        // reset the stepped subsystems too, so animation-off is a deterministic state
        for (const c of this.cellAnims || []) { c.idx = 0; c.acc = 0; c.sprite.texture = c.texs[0]; }
        for (const o of this.animObjs || []) { o.idx = 0; o.acc = 0; o.sprite.texture = o.steps[0].tex; }
        for (const o of this.pathObjs || []) { o.t = 0; o.sprite.x = o.baseX + o.path[0][0]; o.sprite.y = o.baseY + o.path[0][1]; }
      }
    }
  }

  // --- camera (shared MapCamera; only the fit + texture-filter policy live here) ----
  _fitDefault(level) {
    const W = this.app.screen.width, H = this.app.screen.height;
    this.cam.minZoom = Math.min(W / this.levelW, H / this.levelH) * 0.95;
    this.cam.maxZoom = W / GG_W;                                  // GG 1:1 — never magnify past the original viewport
    this.cam.zoom = Math.min(this.cam.maxZoom, Math.max(this.cam.minZoom, H / GG_H)); // start showing the full GG screen height (144px)
    // centre on Sonic's spawn
    const [sx, sy] = level.spawn;
    this.cam.panTo(sx * BLOCK + 8, sy * BLOCK + 16);
    this.cam.apply();
  }

  // Crisp nearest-neighbour when magnifying (zoom >= 1), but linear + mipmaps when minifying
  // (zoomed out) so the downscaled tiles don't moiré/shimmer.
  _updateTexFilter() {
    const mode = this.cam.zoom < 1 ? 'linear' : 'nearest';
    if (mode === this._texMode) return;
    this._texMode = mode;
    for (const idx in this.blockTex) this.blockTex[idx].source.scaleMode = mode;
    if (this.blockTexUW) for (const idx in this.blockTexUW) this.blockTexUW[idx].source.scaleMode = mode;
  }
}
