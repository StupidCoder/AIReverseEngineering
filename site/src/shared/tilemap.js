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
