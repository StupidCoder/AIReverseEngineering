// Super Mario Land level viewer — a tilemap rebuilt from the cartridge: each level is
// a row-major grid of 8×8 background tile indices (decoded by extract/level from the
// ROM), drawn from the world's 256-tile atlas. Drag to pan, scroll to zoom. The
// collision and object layers will hang off this later.
import { Application, Container, Graphics, Sprite, Texture, Rectangle } from 'pixi.js';
import { composeTilemap } from '../tilemap-compose.js';
import { MapCamera } from '../shared/camera.js';

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
    this.cam = new MapCamera(this, { onApply: () => {
      const mode = this.cam.zoom < 1 ? 'linear' : 'nearest';
      if (mode !== this._texMode && this.src) { this._texMode = mode; this.src.scaleMode = mode; }
    } });
    this.cam.wirePointer();
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
    this.cam.minZoom = Math.min((W / this.levelW) * 0.9, z);
    this.cam.maxZoom = Math.max((W / 160) * 6, z);
    this.cam.zoom = Math.max(this.cam.minZoom, Math.min(this.cam.maxZoom, z));
    this.world.position.set(8, (H - this.levelH * this.cam.zoom) / 2);
    this.cam.apply();
  }
}
