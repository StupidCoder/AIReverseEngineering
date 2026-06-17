# site/

Interactive companion website for the reverse-engineering write-ups (see [PLAN.md](PLAN.md)).

It is a **no-build static site**: plain ES modules with an [import map](sonic.html) that pulls
PixiJS (v8) from a CDN, so there is nothing to install or compile. The 3D pages use three.js
the same way (Elite ships now; Marble terrain later).

## Run locally

Serve the folder with any static server (the import map + `fetch()` need HTTP, not `file://`):

```sh
cd site
python3 -m http.server 8000
# open http://localhost:8000/
```

## Layout

- `index.html` — landing page.
- `sonic.html` — the Sonic level viewer.
- `elite.html` — the Elite 3D ship viewer (three.js).
- `fort.html` — the Fort Apocalypse map viewer (PixiJS).
- `marble.html` — the Marble Madness course viewer (PixiJS).
- `src/` — `style.css` and the viewer modules (`src/sonic/`, `src/elite/`, `src/fort/`, `src/marble/`).
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
