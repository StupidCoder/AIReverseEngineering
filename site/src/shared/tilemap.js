// The shared tilemap renderer: one Pixi Sprite per map cell, textures shared so Pixi
// batches them. Two texture strategies, per site/FORMAT.md:
//
//  - "sliced": each cell's texture is a rect of the atlas image (static tiles —
//    Turrican, SML, Marble). Supports a 1px extruded atlas gutter and per-cell
//    horizontal flip (grid.hflipMask).
//  - "baked": each distinct tile is painted once onto its own canvas texture, and
//    tile animation repaints the canvas in place, updating every cell that shows it
//    (Fort's soft chars). Sonic's block variant (blocks indirection) lands with M4.
//
// The build owns its textures: destroy() releases them, so switching levels doesn't
// leak baked canvases.

import { Container, Rectangle, Sprite, Texture } from 'pixi.js';

// Slice an atlas image into per-tile textures honoring the extrusion gutter.
export function sliceAtlas(img, { tileSize, atlasCols = 16, gutter = 0, ntiles }) {
  const src = Texture.from(img).source;
  src.scaleMode = 'nearest';
  const cell = tileSize + 2 * gutter;
  const n = ntiles ?? atlasCols * Math.ceil(img.height / cell);
  const tiles = [];
  for (let i = 0; i < n; i++) {
    const sx = (i % atlasCols) * cell + gutter;
    const sy = ((i / atlasCols) | 0) * cell + gutter;
    tiles.push(new Texture({ source: src, frame: new Rectangle(sx, sy, tileSize, tileSize) }));
  }
  return { src, tiles };
}

// Decode a cell value into { tile, flip } per the grid's hflipMask convention.
export function cellTile(grid, v) {
  const mask = grid.hflipMask || 0;
  if (mask && (v & mask)) return { tile: v & ~mask, flip: true };
  return { tile: v, flip: false };
}

export class Tilemap {
  // strategy: "sliced" (default) | "baked"
  constructor(level, atlasImg, { strategy = 'sliced' } = {}) {
    this.level = level;
    this.grid = level.grid;
    this.container = new Container();
    this.strategy = strategy;
    this.sources = [];        // TextureSources for scale-mode switching
    this.owned = [];          // sources created per level (baked canvases) — destroyed on reload;
                              // the sliced atlas source is Pixi-cached per image and must survive
    this.baked = null;        // strategy "baked": tileId -> { canvas, tex }
    this.atlasImg = atlasImg;
    this._build();
  }

  _build() {
    const g = this.grid;
    const ts = g.tileSize;
    if (this.strategy === 'baked') this._bakeTiles();
    const sliced = this.strategy === 'sliced'
      ? sliceAtlas(this.atlasImg, { tileSize: ts, atlasCols: g.atlasCols ?? 16, gutter: g.atlasGutter ?? 0 })
      : null;
    this.sliced = sliced;
    if (sliced) this.sources.push(sliced.src);
    for (let r = 0; r < g.height; r++) {
      for (let c = 0; c < g.width; c++) {
        const { tile, flip } = cellTile(g, g.cells[r * g.width + c]);
        const tex = sliced ? sliced.tiles[tile] : (this.baked.get(tile) || this._bakeTile(tile)).tex;
        if (!tex) continue;
        const s = new Sprite(tex);
        if (flip) { s.scale.x = -1; s.x = c * ts + ts; } else { s.x = c * ts; }
        s.y = r * ts;
        this.container.addChild(s);
      }
    }
    this.widthPx = g.width * ts;
    this.heightPx = g.height * ts;
  }

  // --- baked strategy ------------------------------------------------------------
  _bakeTiles() {
    this.baked = new Map();
    const g = this.grid;
    const used = new Set();
    for (const v of g.cells) used.add(cellTile(g, v).tile);
    for (const t of used) this._bakeTile(t);
  }

  _bakeTile(tileId) {
    const ts = this.grid.tileSize;
    const cv = document.createElement('canvas');
    cv.width = cv.height = ts;
    const rec = { canvas: cv, tex: Texture.from(cv) };
    rec.tex.source.scaleMode = 'nearest';
    this.sources.push(rec.tex.source);
    this.owned.push(rec.tex.source);
    this.baked.set(tileId, rec);
    this.paintTile(tileId, tileId);
    return rec;
  }

  // Paint atlas tile `srcTile` into the canvas texture of `tileId` (tile animation:
  // every cell showing tileId updates at once).
  paintTile(tileId, srcTile) {
    const rec = this.baked && this.baked.get(tileId);
    if (!rec) return;
    const g = this.grid;
    const ts = g.tileSize;
    const cell = ts + 2 * (g.atlasGutter ?? 0);
    const cols = g.atlasCols ?? 16;
    const ctx = rec.canvas.getContext('2d');
    ctx.imageSmoothingEnabled = false;
    ctx.clearRect(0, 0, ts, ts);
    ctx.drawImage(this.atlasImg,
      (srcTile % cols) * cell + (g.atlasGutter ?? 0),
      ((srcTile / cols) | 0) * cell + (g.atlasGutter ?? 0),
      ts, ts, 0, 0, ts, ts);
    rec.tex.source.update();
  }

  // Texture for any tile id — used by object-pool stamps, which reference chars
  // that may not appear in the map (baked on demand in the baked strategy).
  tileTexture(tileId) {
    if (this.baked) return (this.baked.get(tileId) || this._bakeTile(tileId)).tex;
    return this.sliced ? this.sliced.tiles[tileId] : null;
  }

  setScaleMode(mode) {
    for (const s of this.sources) s.scaleMode = mode;
  }

  destroy() {
    this.container.destroy({ children: true });
    for (const s of this.owned) s.destroy();
    this.sources = [];
    this.owned = [];
    this.baked = null;
  }
}

// BlockTilemap — the block-indirected strategy (Sonic): cells are indices into
// blocks.tiles, each a size x size arrangement of atlas tiles baked once onto a
// canvas texture; the map is one Sprite per block cell sharing those textures.
// Tile animation and the palette effects repaint the canvases in place, updating
// every cell that shows the block. An optional waterline bakes an underwater-
// palette variant of every block for the map rows at/below the line.
export class BlockTilemap {
  // water: { row, map } — block row of the split + surface->underwater colour Map
  constructor(level, atlasImg, { water = null } = {}) {
    this.level = level;
    this.grid = level.grid;
    this.blocks = level.blocks;
    this.atlasImg = atlasImg;
    this.water = water;
    this.blockPx = this.grid.tileSize * this.blocks.size;
    this.container = new Container();
    this.sources = [];
    this.owned = [];
    this.tileOverride = new Map();   // animated tileId -> current atlas tile
    this.blockCanvas = {};
    this.blockTex = {};
    this.blockCanvasUW = {};
    this.blockTexUW = {};
    this.blocksWithTile = new Map(); // tileId -> Set of block indices
    this._texMode = 'nearest';
    this._build();
  }

  _build() {
    const g = this.grid;
    const B = this.blockPx;
    for (const idx of new Set(g.cells)) {
      const mk = () => {
        const cv = document.createElement('canvas');
        cv.width = cv.height = B;
        const tex = Texture.from(cv);
        tex.source.autoGenerateMipmaps = true;    // no moiré when zoomed far out
        tex.source.scaleMode = 'nearest';
        this.sources.push(tex.source);
        this.owned.push(tex.source);
        return { cv, tex };
      };
      const m = mk();
      this.blockCanvas[idx] = m.cv;
      this.blockTex[idx] = m.tex;
      if (this.water) {
        const u = mk();
        this.blockCanvasUW[idx] = u.cv;
        this.blockTexUW[idx] = u.tex;
      }
      this.repaintBlock(idx);
      for (const t of this.blocks.tiles[idx]) {
        if (!this.blocksWithTile.has(t)) this.blocksWithTile.set(t, new Set());
        this.blocksWithTile.get(t).add(idx);
      }
    }
    for (let r = 0; r < g.height; r++) {
      const uw = this.water && r >= this.water.row;
      for (let c = 0; c < g.width; c++) {
        const blk = g.cells[r * g.width + c];
        const tex = (uw && this.blockTexUW[blk]) ? this.blockTexUW[blk] : this.blockTex[blk];
        if (!tex) continue;
        const s = new Sprite(tex);
        s.x = c * B;
        s.y = r * B;
        this.container.addChild(s);
      }
    }
    this.widthPx = g.width * B;
    this.heightPx = g.height * B;
  }

  _paint(ctx, idx) {
    ctx.imageSmoothingEnabled = false;
    const ts = this.grid.tileSize;
    const n = this.blocks.size;
    const cols = this.grid.atlasCols ?? 16;
    const cell = ts + 2 * (this.grid.atlasGutter ?? 0);
    const tiles = this.blocks.tiles[idx];
    for (let r = 0; r < n; r++) {
      for (let c = 0; c < n; c++) {
        let tile = tiles[r * n + c];
        if (this.tileOverride.has(tile)) tile = this.tileOverride.get(tile);
        ctx.drawImage(this.atlasImg,
          (tile % cols) * cell + (this.grid.atlasGutter ?? 0),
          ((tile / cols) | 0) * cell + (this.grid.atlasGutter ?? 0),
          ts, ts, c * ts, r * ts, ts, ts);
      }
    }
  }

  repaintBlock(idx) {
    const B = this.blockPx;
    this._paint(this.blockCanvas[idx].getContext('2d'), idx);
    this.blockTex[idx].source.update();
    if (this.water && this.blockCanvasUW[idx]) {
      // underwater variant: repaint, then remap surface colours to the static
      // underwater palette (these rows never run the cycle)
      const ctx = this.blockCanvasUW[idx].getContext('2d', { willReadFrequently: true });
      this._paint(ctx, idx);
      const img = ctx.getImageData(0, 0, B, B), d = img.data;
      for (let p = 0; p < d.length; p += 4) {
        const u = this.water.map.get((d[p] << 16) | (d[p + 1] << 8) | d[p + 2]);
        if (u) { d[p] = u[0]; d[p + 1] = u[1]; d[p + 2] = u[2]; }
      }
      ctx.putImageData(img, 0, 0);
      this.blockTexUW[idx].source.update();
    }
  }

  // Tile animation entry point (same signature as Tilemap.paintTile): set the
  // tile's current frame and repaint every block that shows it.
  paintTile(tileId, atlasTile) {
    this.tileOverride.set(tileId, atlasTile);
    for (const idx of this.blocksWithTile.get(tileId) || []) this.repaintBlock(idx);
  }

  tileTexture() { return null; } // block games have no per-tile stamps

  setScaleMode(mode) {
    if (mode === this._texMode) return;
    this._texMode = mode;
    for (const s of this.sources) s.scaleMode = mode;
  }

  destroy() {
    this.container.destroy({ children: true });
    for (const s of this.owned) s.destroy();
    this.sources = [];
    this.owned = [];
  }
}
