// Sonic the Hedgehog (Game Gear) — configuration for the shared 2-D level viewer
// (site/FORMAT.md). The richest data set: block-indirected tilemap, ring/flower tile
// animations, the $50 background cell animators, animated object sprites with
// movement paths (index.json steps/durations/path), height-profile collision
// (shapes.json) and the palette effects (water/lava cycle + the Labyrinth underwater
// split). Exported by "Sonic (GG)/extract/cmd/webexport" + cmd/spriterip.
const CAT = {
  crab: 'enemy', beetle: 'enemy', fish: 'enemy', porcupine: 'enemy', bird: 'enemy',
  bonus: 'item', shield: 'item', emerald: 'item', goal: 'item', capsule: 'item',
  'swing platform': 'platform', 'moving platform': 'platform', seesaw: 'platform',
  'bobbing platform': 'platform', 'floating log': 'platform',
  'world 1 boss': 'boss', 'world 2 boss': 'boss', 'world 3 boss': 'boss', 'world 4 boss': 'boss',
  checkpoint: 'ctrl', 'bg animator': 'ctrl',
};

export default {
  base: 'public/sonic/',
  maxNativeFactor: 1, // GG 1:1 — never magnify past the original viewport
  markerCat: (o) => CAT[o.name] || 'default',
  hud: (level) => `${level.grid.width}x${level.grid.height} blocks`,
};
