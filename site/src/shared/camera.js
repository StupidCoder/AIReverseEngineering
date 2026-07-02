// MapCamera — the world/zoom camera shared by every 2-D map viewer (extracted from the
// five identical per-viewer copies). Pan = move the Pixi `world` container, zoom =
// scale it about a screen point; drag/pinch/wheel input; pan clamped to the level
// rect, or wrapped horizontally for cylindrical maps (Fort Apocalypse).
//
// The viewer supplies { el, app, world } plus callbacks:
//   bounds()  -> { w, h }          level size in world px (queried live)
//   wrapX()   -> period | 0        horizontal wrap period in world px (0 = clamp)
//   onApply() -> void              after zoom/pan changes: HUD text, texture filters
//   enabled() -> bool              gate pointer input (Marble's mode switch)
//
// The per-game fit logic stays in each viewer (it sets minZoom/maxZoom/zoom and an
// initial position, then calls apply()) until the common `view` rect lands in M2 —
// after which fitView() is the single entry point.

const ZOOM_STEP = Math.pow(1.15, 0.25); // per wheel notch — a quarter of 1.15 in log space

export class MapCamera {
  constructor(viewer, opts = {}) {
    this.isMapCamera = true;
    this.el = viewer.el;
    this.app = viewer.app;
    this.world = viewer.world;
    this.bounds = opts.bounds || (() => ({ w: viewer.levelW, h: viewer.levelH }));
    this.wrapX = opts.wrapX || (() => 0);
    this.onApply = opts.onApply || (() => {});
    this.enabled = opts.enabled || (() => true);
    this.zoom = 1;
    this.minZoom = 0.05;
    this.maxZoom = 12;
  }

  // Fit the camera to frame a world-px view rect: default zoom shows the rect's
  // height (the machine's native screen), zoomable out to the whole level and in to
  // `maxNativeFactor`x the rect width. Centres the rect (or pins the top edge when
  // the rect starts at y = 0 and the level is taller than the view).
  fitView(view, { maxNativeFactor = 4, minFitFactor = 0.95 } = {}) {
    const W = this.app.screen.width, H = this.app.screen.height;
    const { w: lw, h: lh } = this.bounds();
    const v = view || { x: 0, y: 0, w: lw, h: lh };
    const z = H / v.h;
    const wrap = this.wrapX();
    const fitAll = wrap ? W / wrap : Math.min(W / lw, H / lh);
    this.minZoom = Math.min(fitAll * minFitFactor, z);
    this.maxZoom = Math.max((W / v.w) * maxNativeFactor, z);
    this.zoom = Math.max(this.minZoom, Math.min(this.maxZoom, z));
    this.panTo(v.x + v.w / 2, v.y + v.h / 2);
    this.apply();
  }

  // Centre the viewport on a world point.
  panTo(wx, wy) {
    this.world.position.set(
      this.app.screen.width / 2 - wx * this.zoom,
      this.app.screen.height / 2 - wy * this.zoom,
    );
  }

  // Pan by a screen-px drag delta.
  panBy(dx, dy) {
    this.world.position.x += dx;
    this.world.position.y += dy;
    this.clampPan();
  }

  // Map a client (CSS-px) point to app screen-px, accounting for canvas CSS scaling.
  screenPt(cx, cy) {
    const r = this.el.getBoundingClientRect();
    return {
      x: (cx - r.left) * (this.app.screen.width / r.width),
      y: (cy - r.top) * (this.app.screen.height / r.height),
    };
  }

  // Zoom by `f` about the screen point (px,py), keeping the world point under it fixed.
  zoomAt(px, py, f) {
    const wx = (px - this.world.position.x) / this.zoom;
    const wy = (py - this.world.position.y) / this.zoom;
    this.zoom = Math.min(this.maxZoom, Math.max(this.minZoom, this.zoom * f));
    this.world.position.set(px - wx * this.zoom, py - wy * this.zoom);
    this.apply();
  }

  zoomAtCenter(f) {
    this.zoomAt(this.app.screen.width / 2, this.app.screen.height / 2, f);
  }

  // Clamp the pan so the map can't leave the viewport (centred if it fits); with a
  // wrap period, the x position wraps modulo the cylinder instead (the wrap copies
  // always cover the viewport).
  clampPan() {
    const sw = this.app.screen.width, sh = this.app.screen.height;
    const { w: lw, h: lh } = this.bounds();
    const zw = lw * this.zoom, zh = lh * this.zoom;
    let { x, y } = this.world.position;
    y = zh <= sh ? (sh - zh) / 2 : Math.min(0, Math.max(sh - zh, y));
    const period = this.wrapX() * this.zoom;
    if (period > 0) x = -(((-x % period) + period) % period);
    else x = zw <= sw ? (sw - zw) / 2 : Math.min(0, Math.max(sw - zw, x));
    this.world.position.set(x, y);
  }

  apply() {
    this.world.scale.set(this.zoom);
    this.clampPan();
    this.onApply();
  }

  // Drag to pan (mouse or one finger); pinch with two fingers to zoom; wheel to zoom.
  // Tracks every active pointer so the same handlers serve mouse and multi-touch.
  wirePointer() {
    const c = this.el;
    const pts = new Map();              // pointerId -> last {x, y} in client (CSS) px
    let pinchDist = 0, pinchMid = null; // previous two-finger distance + midpoint

    c.addEventListener('pointerdown', (e) => {
      if (!this.enabled()) return;
      try { c.setPointerCapture(e.pointerId); } catch { /* no-op */ }
      pts.set(e.pointerId, { x: e.clientX, y: e.clientY });
      c.classList.add('dragging');
      if (pts.size === 2) {
        const [a, b] = [...pts.values()];
        pinchDist = Math.hypot(a.x - b.x, a.y - b.y);
        pinchMid = { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 };
      }
    });

    c.addEventListener('pointermove', (e) => {
      if (!this.enabled()) return;
      const p = pts.get(e.pointerId);
      if (!p) return;
      const dx = e.clientX - p.x, dy = e.clientY - p.y;
      p.x = e.clientX; p.y = e.clientY;
      if (pts.size >= 2) {
        // Pinch: pan by the midpoint's motion, then zoom by the distance ratio about it.
        const [a, b] = [...pts.values()];
        const dist = Math.hypot(a.x - b.x, a.y - b.y);
        const mid = { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 };
        if (pinchMid) {
          this.world.position.x += mid.x - pinchMid.x;
          this.world.position.y += mid.y - pinchMid.y;
        }
        const sp = this.screenPt(mid.x, mid.y);
        this.zoomAt(sp.x, sp.y, pinchDist > 0 ? dist / pinchDist : 1);
        pinchDist = dist; pinchMid = mid;
      } else {
        this.panBy(dx, dy);
      }
    });

    const end = (e) => {
      pts.delete(e.pointerId);
      try { c.releasePointerCapture(e.pointerId); } catch { /* no-op */ }
      if (pts.size < 2) { pinchMid = null; pinchDist = 0; }
      if (pts.size === 0) c.classList.remove('dragging');
    };
    c.addEventListener('pointerup', end);
    c.addEventListener('pointercancel', end);

    c.addEventListener('wheel', (e) => {
      if (!this.enabled()) return;
      e.preventDefault();
      const sp = this.screenPt(e.clientX, e.clientY);
      this.zoomAt(sp.x, sp.y, e.deltaY < 0 ? ZOOM_STEP : 1 / ZOOM_STEP);
    }, { passive: false });
  }
}
