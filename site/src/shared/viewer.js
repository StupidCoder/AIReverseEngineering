// The shared 2-D level viewer: consumes the common format (site/FORMAT.md) and
// composes the shared pieces — MapCamera, Tilemap, overlay layers, AnimRunner.
// Per-game differences live in a small config (site/src/<game>/config.js):
//
//   { base: 'public/sml/',           // data directory
//     strategy: 'sliced'|'baked',    // tile texture strategy (baked = tile anim games)
//     maxNativeFactor: 4,            // zoom-in cap, in multiples of the native view
//     markerCat(o) -> category,      // marker colour class for sprite-less objects
//     hud(level) -> string,          // HUD caption
//     hooks?: { loaded(viewer, level) }   // game-specific extension point
//   }

import { Application, Container, Sprite, Texture } from 'pixi.js';
import { MapCamera } from './camera.js';
import { GameData } from './data.js';
import { Tilemap, BlockTilemap } from './tilemap.js';
import { AnimRunner } from './anim.js';
import { buildWaterInfo, setupCycle } from './palettefx.js';
import { buildCollisionGrid, buildCollisionProfiles, buildObjects, buildPools, rng } from './layers.js';

export class LevelViewer {
  constructor(viewportEl, hudEl, config) {
    this.el = viewportEl;
    this.hud = hudEl;
    this.config = config;
    this.data = new GameData(config.base);
    this.app = new Application();
    this.world = new Container();
    this.tileLayer = new Container();
    this.collisionLayer = new Container();
    this.objectLayer = new Container();
    this.poolLayer = new Container();      // randomized placements (rebuilt per toggle)
    this.collisionLayer.visible = false;
    this._texMode = 'nearest';
    this.level = null;
    this.tilemap = null;
    this.meta = null;
  }

  async init() {
    await this.app.init({ background: 0x000000, antialias: false, resizeTo: this.el, preserveDrawingBuffer: true });
    this.el.appendChild(this.app.canvas);
    this.world.addChild(this.tileLayer, this.collisionLayer, this.objectLayer, this.poolLayer);
    this.app.stage.addChild(this.world);
    this.meta = await this.data.loadMeta();
    this.anim = new AnimRunner(this.app, this.meta.tickHz);
    this.cam = new MapCamera(this, {
      enabled: () => (this.config.camEnabled ? this.config.camEnabled() : true),
      bounds: () => ({ w: this.levelW || 1, h: this.levelH || 1 }),
      wrapX: () => (this.meta.wrap === 'x' && this.levelW ? this.levelW : 0),
      onApply: () => {
        const mode = this.cam.zoom < 1 ? 'linear' : 'nearest';
        if (mode !== this._texMode) {
          this._texMode = mode;
          if (this.tilemap) this.tilemap.setScaleMode(mode);
        }
        if (this.hud && this.level && this.config.hud) this.hud.textContent = this.config.hud(this.level);
      },
    });
    this.cam.wirePointer();
    return this.meta;
  }

  async loadLevel(entry) {
    const level = await this.data.loadLevel(entry);
    this.level = level;

    // tilemap (owns its textures; destroyed on reload). Games with block
    // indirection (level.blocks) use the block-baked strategy automatically.
    if (this.tilemap) this.tilemap.destroy();
    for (const m of this._extraMaps || []) m.destroy();
    this._extraMaps = [];
    this.tileLayer.removeChildren();
    const atlasImg = await this.data.atlasImage(level.grid.atlas);
    const mkMap = () => level.blocks
      ? new BlockTilemap(level, atlasImg, { water: buildWaterInfo(level) })
      : new Tilemap(level, atlasImg, { strategy: this.config.strategy || 'sliced' });
    this.tilemap = mkMap();
    const copies = this.meta.wrap === 'x' ? 3 : 1;
    this.tileLayer.addChild(this.tilemap.container);
    this.levelW = this.tilemap.widthPx;
    this.levelH = this.tilemap.heightPx;
    for (let i = 1; i < copies; i++) {
      const dup = mkMap();
      dup.container.x = i * this.levelW;
      this.tileLayer.addChild(dup.container);
      this._extraMaps.push(dup);
    }
    this._texMode = 'nearest';

    // animation subsystems
    this.anim.reset();
    for (const a of level.tileAnims || []) {
      this.anim.tileAnims.push({
        anim: a, acc: 0, step: 0,
        paint: (tileId, atlasTile) => {
          this.tilemap.paintTile(tileId, atlasTile);
          for (const m of this._extraMaps || []) m.paintTile(tileId, atlasTile);
        },
      });
    }
    // placement-anchored cell animators (Sonic $50): 2x4-tile strips baked from
    // the atlas, one sprite each in the tile layer, per-phase hold times
    for (const ca of level.cellAnims || []) {
      const ts = level.grid.tileSize;
      const texs = ca.phases.map((ph) => {
        const cv = document.createElement('canvas');
        cv.width = 2 * ts; cv.height = 4 * ts;
        const ctx = cv.getContext('2d');
        ctx.imageSmoothingEnabled = false;
        ph.tiles.forEach((tile, i) => {
          const cols = level.grid.atlasCols ?? 16;
          const cell = ts + 2 * (level.grid.atlasGutter ?? 0);
          ctx.drawImage(atlasImg,
            (tile % cols) * cell + (level.grid.atlasGutter ?? 0),
            ((tile / cols) | 0) * cell + (level.grid.atlasGutter ?? 0),
            ts, ts, (i % 2) * ts, ((i / 2) | 0) * ts, ts, ts);
        });
        const tx = Texture.from(cv);
        tx.source.scaleMode = 'nearest';
        return tx;
      });
      const sp = new Sprite(texs[0]);
      sp.x = ca.tx * ts;
      sp.y = ca.ty * ts;
      this.tileLayer.addChild(sp);
      this.anim.cellAnims.push({
        sprite: sp, texs,
        durs: ca.phases.map((p) => Math.max(1, p.frames)),
        idx: 0, acc: 0,
      });
    }
    // palette cycle (water/lava shimmer), an AnimRunner fx
    const cycle = setupCycle(level, this.tilemap);
    if (cycle) this.anim.fx.push(cycle);

    // collision overlay
    this.collisionLayer.removeChildren();
    if (level.collision) {
      if (level.collision.kind === 'grid') {
        this.collisionLayer.addChild(buildCollisionGrid(level));
      } else if (level.collision.kind === 'profiles') {
        const shapes = await this.data.loadShapes(level.collision.shapesFile || 'shapes.json');
        this.collisionLayer.addChild(buildCollisionProfiles(level, shapes));
      }
    }

    // objects + spawn (fixed placements)
    this.objectLayer.removeChildren();
    const { container, animObjs, pathObjs } = await buildObjects(level, this.data, {
      markerCat: this.config.markerCat || (() => 'default'),
    });
    for (let i = 0; i < copies; i++) {
      if (i === 0) { this.objectLayer.addChild(container); continue; }
      const { container: dup } = await buildObjects(level, this.data, {
        markerCat: this.config.markerCat || (() => 'default'),
      });
      dup.x = i * this.levelW;
      this.objectLayer.addChild(dup);
    }
    this.anim.spriteSteps.push(...animObjs);
    this.anim.movePaths.push(...pathObjs);

    // randomized pools (rebuilt whenever the objects layer switches on)
    await this._rebuildPools();

    this.cam.fitView(level.view, {
      maxNativeFactor: this.config.maxNativeFactor ?? 4,
      minFitFactor: this.config.minFitFactor ?? 0.95,
    });
    if (this.config.hooks?.loaded) await this.config.hooks.loaded(this, level);
    return level;
  }

  async _rebuildPools() {
    this.poolLayer.removeChildren();
    const level = this.level;
    if (!level || !(level.objectPools || []).length) return;
    const seed = typeof window !== 'undefined' ? window.__studioSeed : null;
    const random = rng(seed);
    const copies = this.meta.wrap === 'x' ? 3 : 1;
    const stampTex = (tileId) => this.tilemap.tileTexture(tileId);
    const pool = await buildPools(level, this.data, { random, stampTex });
    for (let i = 0; i < copies; i++) {
      if (i === 0) { this.poolLayer.addChild(pool); continue; }
      const dup = await buildPools(level, this.data, { random: rng(seed), stampTex });
      dup.x = i * this.levelW;
      this.poolLayer.addChild(dup);
    }
  }

  setLayer(name, on) {
    if (name === 'collision') this.collisionLayer.visible = on;
    if (name === 'objects') {
      this.objectLayer.visible = on;
      this.poolLayer.visible = on;
      if (on && (this.level?.objectPools || []).length) this._rebuildPools();
    }
    if (name === 'animation') this.anim.setEnabled(on);
  }
}
