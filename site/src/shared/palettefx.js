// Palette effects (level.paletteFx, site/FORMAT.md) — currently Sonic-only, but any
// game exporting the section gets them:
//
//  - cycle: the engine rotates a few palette slots at a fixed period (water /
//    waterfall shimmer). The exporter captures the per-step colours and the set of
//    tiles that genuinely use a cycling slot; here each affected block's cycling
//    tile cells are recoloured in place by remapping the step-0 colours (ported
//    from the old Sonic viewer's _applyCycleStep).
//  - waterLine: handled at bake time by BlockTilemap (the exporter's static
//    underwater palette becomes a colour remap for rows below the line);
//    buildWaterInfo derives that remap.
//
// setupCycle returns an fx object for AnimRunner ({ tick, reset }) or null.

const hexToRgb = (h) => {
  const n = parseInt(h.slice(1), 16);
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
};

// Derive BlockTilemap's water option from paletteFx.waterLine.
export function buildWaterInfo(level) {
  const fx = level.paletteFx;
  if (!fx || !fx.waterLine) return null;
  const surf = fx.palette.map(hexToRgb);
  const uw = fx.waterLine.palette.map(hexToRgb);
  const map = new Map();
  for (let i = 0; i < uw.length; i++) {
    map.set((surf[i][0] << 16) | (surf[i][1] << 8) | surf[i][2], uw[i]);
  }
  const blockPx = level.grid.tileSize * (level.blocks ? level.blocks.size : 1);
  return { row: Math.round(fx.waterLine.y / blockPx), map };
}

export function setupCycle(level, tilemap) {
  const fx = level.paletteFx;
  if (!fx || !fx.cycle || !tilemap.blockCanvas) return null;
  const steps = fx.cycle.steps.map((step) => step.map(hexToRgb));
  const from = steps[0];
  const cycleTiles = new Set(fx.cycle.tiles);
  const B = tilemap.blockPx;
  const n = level.blocks.size;

  // Blocks containing a tile that genuinely uses a cycling slot (not just a tile
  // sharing the colour at rest — that must not blink), with their cell lists and a
  // clean step-0 snapshot to remap from.
  const blocks = [];
  for (const idx of Object.keys(tilemap.blockCanvas).map(Number)) {
    const cells = [];
    level.blocks.tiles[idx].forEach((t, cell) => { if (cycleTiles.has(t)) cells.push(cell); });
    if (!cells.length) continue;
    blocks.push({
      idx, cells,
      clean: tilemap.blockCanvas[idx].getContext('2d').getImageData(0, 0, B, B),
    });
  }
  if (!blocks.length) return null;

  const apply = (s) => {
    for (const b of blocks) {
      const ctx = tilemap.blockCanvas[b.idx].getContext('2d');
      const out = ctx.createImageData(B, B);
      out.data.set(b.clean.data);
      const d = out.data;
      for (const cell of b.cells) {
        const ts = level.grid.tileSize;
        const cx = (cell % n) * ts, cy = ((cell / n) | 0) * ts;
        for (let y = 0; y < ts; y++) {
          for (let x = 0; x < ts; x++) {
            const p = ((cy + y) * B + (cx + x)) * 4;
            const r = d[p], g = d[p + 1], bl = d[p + 2];
            for (let i = 0; i < from.length; i++) {
              const f = from[i];
              if (r === f[0] && g === f[1] && bl === f[2]) {
                const t = steps[s][i];
                d[p] = t[0]; d[p + 1] = t[1]; d[p + 2] = t[2];
                break;
              }
            }
          }
        }
      }
      ctx.putImageData(out, 0, 0);
      tilemap.blockTex[b.idx].source.update();
    }
  };

  let acc = 0, step = 0;
  return {
    tick(df) {
      acc += df;
      const period = fx.cycle.periodFrames || 30;
      let changed = false;
      while (acc >= period) { acc -= period; step = (step + 1) % steps.length; changed = true; }
      if (changed) apply(step);
    },
    reset() {
      if (step !== 0) apply(0);
      acc = 0; step = 0;
    },
  };
}
