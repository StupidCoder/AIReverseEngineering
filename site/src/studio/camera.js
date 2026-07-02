// Keyboard camera control for the Studio. Cursor keys scroll the active viewer with
// acceleration (speed ramps while held) and momentum (it coasts and eases out after release);
// +/- zoom. It drives each viewer family directly -- the 2-D map viewers expose a shared
// MapCamera at v.cam (pan = panBy, zoom = zoomAtCenter), the three.js viewers an OrbitControls camera
// (pan = translate camera+target, zoom = dolly) -- since synthesising pointer drags would
// trip OrbitControls' pointer capture.

const ACCEL = 1.7;     // velocity gained per frame while a key is held
const MAXV = 30;       // max pan speed (px/frame at 60fps)
const FRICTION = 0.88; // per-frame velocity retention -> momentum after release
const ZOOM_RATE = 1.02; // zoom factor per frame while +/- held

// three.js is only needed for the 3-D viewers; load it lazily (and only in the browser via
// the import map) so this module imports cleanly without three for the 2-D path and tests.
let THREE = null, _r, _u, _m;
function ensureThree() {
  if (THREE) return true;
  import('three').then(m => { THREE = m; _r = new m.Vector3(); _u = new m.Vector3(); _m = new m.Vector3(); });
  return false;
}

function threeCam(v) {
  if (v.three && v.three.camera) return v.three;
  if (v.camera && v.controls) return { camera: v.camera, controls: v.controls };
  return null;
}

function panThree(t, dx, dy) {
  if (!ensureThree()) return; // three still loading; pick up next frame
  const { camera, controls } = t;
  const el = controls.domElement;
  const h = (el && el.clientHeight) || 600;
  const w = (el && el.clientWidth) || 800;
  _r.setFromMatrixColumn(camera.matrixWorld, 0); // camera right (world)
  _u.setFromMatrixColumn(camera.matrixWorld, 1); // camera up (world)
  let kx, ky;
  if (camera.isOrthographicCamera) {
    kx = (camera.right - camera.left) / camera.zoom / w;
    ky = (camera.top - camera.bottom) / camera.zoom / h;
  } else {
    const wph = 2 * camera.position.distanceTo(controls.target) * Math.tan((camera.fov * Math.PI / 180) / 2) / h;
    kx = ky = wph;
  }
  // same drag-delta convention as the 2-D viewers: dx>0 drags the scene right (camera left)
  _m.copy(_r).multiplyScalar(-dx * kx).addScaledVector(_u, dy * ky);
  camera.position.add(_m);
  controls.target.add(_m);
  controls.update();
}

function dollyThree(t, factor) {
  const { camera, controls } = t;
  if (camera.isOrthographicCamera) {
    camera.zoom = Math.max(0.02, camera.zoom * factor);
    camera.updateProjectionMatrix();
  } else {
    const off = camera.position.clone().sub(controls.target);
    const len = Math.max(controls.minDistance || 0.01, Math.min(controls.maxDistance || Infinity, off.length() / factor));
    camera.position.copy(controls.target).add(off.setLength(len));
  }
  controls.update();
}

function mapCam(v) { return v && v.cam && v.cam.isMapCamera ? v.cam : null; }

export class KeyboardCamera {
  constructor(getActiveViewer) {
    this.getViewer = getActiveViewer;
    this.keys = { left: false, right: false, up: false, down: false, zin: false, zout: false };
    this.vx = 0; this.vy = 0;
    this._raf = null;
    this._last = 0;
    this._tick = this._tick.bind(this);
    if (typeof window !== 'undefined') {
      window.addEventListener('keydown', (e) => this._onKey(e, true));
      window.addEventListener('keyup', (e) => this._onKey(e, false));
    }
  }

  _onKey(e, down) {
    const tag = (document.activeElement && document.activeElement.tagName) || '';
    if (tag === 'INPUT' || tag === 'TEXTAREA') return; // leave sliders/seek bar to the keyboard
    let hit = true;
    switch (e.key) {
      case 'ArrowLeft': this.keys.left = down; break;
      case 'ArrowRight': this.keys.right = down; break;
      case 'ArrowUp': this.keys.up = down; break;
      case 'ArrowDown': this.keys.down = down; break;
      case '+': case '=': case 'Add': this.keys.zin = down; break;
      case '-': case '_': case 'Subtract': this.keys.zout = down; break;
      default: hit = false;
    }
    if (hit) { e.preventDefault(); this._start(); }
  }

  _start() {
    if (this._raf == null) { this._last = performance.now(); this._raf = requestAnimationFrame(this._tick); }
  }

  _tick(now) {
    const dt = Math.max(0, Math.min(40, now - this._last)) / 16.67; // ~1 at 60 fps
    this._last = now;
    const v = this.getViewer();
    if (v) this._apply(v, dt);

    const heldPan = this.keys.left || this.keys.right || this.keys.up || this.keys.down;
    const heldZoom = this.keys.zin || this.keys.zout;
    const moving = Math.abs(this.vx) > 0.06 || Math.abs(this.vy) > 0.06;
    if (heldPan || heldZoom || moving) this._raf = requestAnimationFrame(this._tick);
    else { this._raf = null; this.vx = this.vy = 0; }
  }

  _apply(v, dt) {
    const ax = (this.keys.right ? 1 : 0) - (this.keys.left ? 1 : 0);
    const ay = (this.keys.down ? 1 : 0) - (this.keys.up ? 1 : 0);
    // pan velocity is a drag delta; arrow Right scrolls right (camera right) => drag left
    this.vx = clamp(this.vx * Math.pow(FRICTION, dt) - ax * ACCEL * dt, MAXV);
    this.vy = clamp(this.vy * Math.pow(FRICTION, dt) - ay * ACCEL * dt, MAXV);
    if (Math.abs(this.vx) < 0.06) this.vx = 0;
    if (Math.abs(this.vy) < 0.06) this.vy = 0;
    if (this.vx || this.vy) this._pan(v, this.vx * dt, this.vy * dt);

    const zd = (this.keys.zin ? 1 : 0) - (this.keys.zout ? 1 : 0);
    if (zd) this._zoom(v, Math.pow(ZOOM_RATE, zd * dt));
  }

  _pan(v, dx, dy) {
    const cam = mapCam(v);
    if (cam) {
      cam.panBy(dx, dy);
    } else {
      const t = threeCam(v);
      if (t) panThree(t, dx, dy);
    }
  }

  _zoom(v, factor) {
    const cam = mapCam(v);
    if (cam) {
      cam.zoomAtCenter(factor);
    } else {
      const t = threeCam(v);
      if (t) dollyThree(t, factor);
    }
  }
}

function clamp(x, m) { return x < -m ? -m : x > m ? m : x; }
