# site/

Interactive companion website for the reverse-engineering write-ups (see [PLAN.md](PLAN.md)).

It is a **no-build static site**: plain ES modules with an [import map](index.html) that pulls
PixiJS (v8) from a CDN, so there is nothing to install or compile. 3D viewers use three.js
the same way (Elite ships, Stunt Car Racer tracks).

## Run locally

Serve the folder with any static server (the import map + `fetch()` need HTTP, not `file://`):

```sh
cd site
python3 -m http.server 8000
# open http://localhost:8000/
```

## Layout

- `index.html` — the **Studio**: a single full-screen front-end for all games. A floating menu
  picks the game and asset; the selected viewer renders full-bleed, with display-layer toggles,
  a music player, an optional CRT filter, and a technical-details panel. This is the whole site.
- `src/studio/` — the Studio shell (`main.js`, `info-content.js`, `crt.js`, `camera.js`).
- `src/shared/` — the **shared 2-D level viewer** every tilemap game runs on (Sonic, Fort,
  Turrican, Marble's map view, SML): camera, tilemap renderer (sliced / baked / block
  strategies), overlay layers, the animation runner and palette effects. It consumes the
  **common level format** specified in [FORMAT.md](FORMAT.md); per-game specifics live in a
  small `src/<game>/config.js`.
- `src/elite/`, `src/stuntcar/`, `src/marble/slopes.js` — the three.js 3-D viewers (wireframe
  ships, track ribbons, the slope height-mesh), outside the tilemap format.
- `public/marble/` — per-course `<course>.png` + `meta.json`. Regenerate from the disk with:

  ```sh
  cd "Marble Madness (Amiga)/extract"
  go run ./cmd/webexport
  ```
- `public/fort/` — per-level JSON (`level0/1.json`, `meta.json`) + `atlas-L0/1.png`.
  Regenerate (after extracting `FORT-fast-7000.prg` from the tape) with:

  ```sh
  cd "Fort Apocalypse (C64)/extract"
  go run ./cmd/webexport
  ```
- `public/turrican/` — per-world `atlas<N>.png` tile sheets + per-scene `world<W>_scene<S>.json`
  (row-major tile-index `cells`, `ntiles` for the flip threshold) + `meta.json`. Regenerate
  from the disk image with:

  ```sh
  go run turrican/extract/cmd/webexport   # run from the repo root
  ```
- `public/elite/ships.json` — decoded ship blueprints (vertices, edges, face normals).
  Regenerate from the extracted engine block with:

  ```sh
  cd "Elite (C64)/extract"
  go run ./cmd/webexport
  ```
- `public/sonic/` — exported data: `meta.json`, `shapes.json`, the `atlas_*.png` tile
  atlases, and `act01.json … act18.json`. Regenerate from the ROM with:

  ```sh
  cd "Sonic (GG)/extract"
  go run ./cmd/webexport "../Sonic The Hedgehog (Japan, USA).gg" "../../site/public/sonic"
  ```

## Deploy

Pushed to GitHub Pages by [`.github/workflows/pages.yml`](../.github/workflows/pages.yml),
which uploads this folder as the Pages artifact (no build step). Set the repository's Pages
source to **GitHub Actions**.

## Sonic level viewer

The map is a real tilemap rebuilt from the cartridge — each 32×32 block is baked once from
the zone's tile atlas, then placed as one sprite per cell (so PixiJS batches the whole level
and zoom is cheap). Drag to pan, scroll to zoom. Toggle layers:

- **Animation** — the rings (6 frames) and Green Hills flowers (2 frames) cycle at the
  Game Gear's cadence by re-baking only the animated block textures.
- **Collision shapes** — each block's surface height-profile (red) over the real tiles;
  non-solid blocks tinted blue (where Sonic falls through).
- **Objects** — enemies/items/bosses and Sonic's spawn marker.

> Note: the frontend was written in an environment without a browser or JS runtime, so it is
> verified at the data-contract level (the exporter output is checked against the
> `cmd/levelmap` render pixel-for-pixel, and the JSON/atlas indices are validated) but not
> yet run in a browser — please try it and report anything that needs fixing.
