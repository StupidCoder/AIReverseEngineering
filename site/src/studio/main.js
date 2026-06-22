// Studio front-end: an immersive presentation of the extracted game assets. The active
// game's viewer (the same per-game viewer the standalone pages use) renders full-bleed into
// #stage; a floating control window picks the game and, cascading from it, the level.
//
// Each game exposes a slightly different viewer API, so a small adapter per game normalises
// "create a viewer", "list the levels (named)", and "show level i". Viewers are created lazily
// on first visit and cached (kept mounted but hidden) so switching back is instant.
//
// A global CRT filter (crt.js) overlays all of them: a post-process that captures whichever
// viewer is active and re-renders it through a physical CRT pipeline.

import { CRT } from './crt.js';

const GAMES = [
  {
    id: 'sonic', name: 'Sonic the Hedgehog', system: 'Sega Game Gear',
    load: () => import('../sonic/viewer.js').then(m => m.LevelViewer),
    make: (V, el, hud) => new V(el, hud),
    list: async (v) => (await v.init()).acts,
    show: (v, lvl, i) => v.loadAct(lvl),
  },
  {
    id: 'fort', name: 'Fort Apocalypse', system: 'Commodore 64',
    load: () => import('../fort/viewer.js').then(m => m.FortViewer),
    make: (V, el, hud) => new V(el, hud),
    list: async (v) => (await v.init()).levels,
    show: (v, lvl, i) => v.loadLevel(lvl),
  },
  {
    id: 'turrican', name: 'Turrican', system: 'Amiga',
    load: () => import('../turrican/viewer.js').then(m => m.TurricanViewer),
    make: (V, el, hud) => new V(el, hud),
    list: async (v) => (await v.init()).levels,
    show: (v, lvl, i) => v.loadLevel(lvl),
  },
  {
    id: 'marble', name: 'Marble Madness', system: 'Amiga',
    load: () => import('../marble/viewer.js').then(m => m.MarbleViewer),
    make: (V, el, hud) => new V(el, hud),
    list: async (v) => (await v.init()).levels,
    show: (v, lvl, i) => v.loadLevel(lvl),
  },
  {
    id: 'stuntcar', name: 'Stunt Car Racer', system: 'Amiga', render: '3d',
    load: () => import('../stuntcar/viewer.js').then(m => m.TrackViewer),
    make: (V, el, hud) => new V(el),
    list: async () => (await fetch('public/stuntcar/tracks.json').then(r => r.json())),
    show: (v, lvl, i) => v.show(lvl, i),
  },
  {
    id: 'elite', name: 'Elite', system: 'Commodore 64', render: '3d',
    load: () => import('../elite/viewer.js').then(m => m.ShipViewer),
    make: (V, el, hud) => new V(el, hud),
    list: async (v) => await v.init(), // returns the ship list
    show: (v, lvl, i) => v.loadShip(i),
  },
];

const stage = document.getElementById('stage');
const hud = document.getElementById('hud');
const panel = document.getElementById('panel');
const gameList = document.getElementById('gameList');
const levelList = document.getElementById('levelList');
const titlecard = document.getElementById('titlecard');
const spinner = document.getElementById('spinner');

const mounts = new Map(); // gameId -> { game, el, viewer, levels, current }
let activeId = null;
let busy = false;

// ---- the game list (always available) ----
function buildGameList() {
  for (const game of GAMES) {
    const b = document.createElement('button');
    b.className = 'item';
    b.dataset.game = game.id;
    b.innerHTML = `<span class="name">${game.name}</span><span class="sub">${game.system}</span>`;
    b.addEventListener('click', () => selectGame(game.id));
    gameList.appendChild(b);
  }
}

function markActiveGame(id) {
  for (const b of gameList.children) b.classList.toggle('active', b.dataset.game === id);
}

// ---- level list (rebuilt per game) ----
function buildLevelList(m) {
  levelList.innerHTML = '';
  m.levels.forEach((lvl, i) => {
    const b = document.createElement('button');
    b.className = 'item' + (i === m.current ? ' active' : '');
    b.innerHTML = `<span class="name">${lvl.name || `Level ${i + 1}`}</span>`;
    b.addEventListener('click', () => selectLevel(m.game.id, i));
    levelList.appendChild(b);
  });
}

function markActiveLevel(i) {
  [...levelList.children].forEach((b, j) => {
    const on = j === i;
    b.classList.toggle('active', on);
    if (on) b.scrollIntoView({ block: 'nearest' });
  });
}

function setBusy(on) {
  busy = on;
  spinner.classList.toggle('on', on);
}

// ---- selection ----
async function selectGame(id) {
  if (busy || id === activeId) return;
  const game = GAMES.find(g => g.id === id);
  markActiveGame(id);
  setBusy(true);
  try {
    // hide the currently mounted viewer
    if (activeId && mounts.has(activeId)) mounts.get(activeId).el.style.display = 'none';

    let m = mounts.get(id);
    if (!m) {
      const el = document.createElement('div');
      el.className = 'mount';
      el.dataset.render = game.render || '2d';
      stage.appendChild(el);
      const Viewer = await game.load();
      const viewer = game.make(Viewer, el, hud);
      const levels = await game.list(viewer);
      m = { game, el, viewer, levels, current: 0 };
      mounts.set(id, m);
      await game.show(viewer, levels[0], 0);
    }
    m.el.style.display = 'block';
    activeId = id;
    buildLevelList(m);
    markActiveLevel(m.current);
    updateHud(m);
    hideTitlecard();
  } catch (err) {
    console.error('studio: failed to load', id, err);
    hud.innerHTML = `<b>${game.name}</b> — failed to load (${err.message})`;
  } finally {
    setBusy(false);
  }
}

async function selectLevel(id, i) {
  const m = mounts.get(id);
  if (!m || busy || i === m.current) return;
  setBusy(true);
  try {
    m.current = i;
    markActiveLevel(i);
    await m.game.show(m.viewer, m.levels[i], i);
    updateHud(m);
  } catch (err) {
    console.error('studio: failed to show level', id, i, err);
  } finally {
    setBusy(false);
  }
}

function updateHud(m) {
  const lvl = m.levels[m.current];
  hud.innerHTML = `<b>${m.game.name}</b> · ${m.game.system} &nbsp;—&nbsp; ${lvl.name || `Level ${m.current + 1}`}`;
}

function hideTitlecard() {
  if (titlecard.style.display === 'none') return;
  titlecard.classList.add('hidden');
  setTimeout(() => { titlecard.style.display = 'none'; }, 500); // fade out, then drop it for good
}

// ---- menu / panel ----
document.getElementById('menuBtn').addEventListener('click', () => panel.classList.toggle('open'));
document.getElementById('panelClose').addEventListener('click', () => panel.classList.remove('open'));

// make the floating window draggable by its title bar
(function makeDraggable() {
  const bar = panel.querySelector('.panel-bar');
  let sx, sy, ox, oy, dragging = false;
  bar.addEventListener('pointerdown', (e) => {
    if (e.target.closest('.panel-x')) return;
    dragging = true; bar.classList.add('dragging');
    const r = panel.getBoundingClientRect();
    panel.style.left = r.left + 'px'; panel.style.top = r.top + 'px';
    panel.style.right = 'auto'; panel.style.bottom = 'auto';
    sx = e.clientX; sy = e.clientY; ox = r.left; oy = r.top;
    bar.setPointerCapture(e.pointerId);
  });
  bar.addEventListener('pointermove', (e) => {
    if (!dragging) return;
    const nx = Math.max(6, Math.min(window.innerWidth - 60, ox + e.clientX - sx));
    const ny = Math.max(6, Math.min(window.innerHeight - 40, oy + e.clientY - sy));
    panel.style.left = nx + 'px'; panel.style.top = ny + 'px';
  });
  const end = () => { dragging = false; bar.classList.remove('dragging'); };
  bar.addEventListener('pointerup', end);
  bar.addEventListener('pointercancel', end);
})();

// ---- fullscreen ----
const fsBtn = document.getElementById('fsBtn');
const EXPAND = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 9V5a1 1 0 0 1 1-1h4M20 9V5a1 1 0 0 0-1-1h-4M4 15v4a1 1 0 0 0 1 1h4M20 15v4a1 1 0 0 1-1 1h-4"/></svg>';
const COLLAPSE = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 4v3a2 2 0 0 1-2 2H4M20 9h-3a2 2 0 0 1-2-2V4M4 15h3a2 2 0 0 1 2 2v3M15 20v-3a2 2 0 0 1 2-2h3"/></svg>';
fsBtn.innerHTML = EXPAND;
fsBtn.addEventListener('click', () => {
  if (document.fullscreenElement) document.exitFullscreen();
  else document.documentElement.requestFullscreen?.();
});
document.addEventListener('fullscreenchange', () => {
  const on = !!document.fullscreenElement;
  fsBtn.classList.toggle('on', on);
  fsBtn.innerHTML = on ? COLLAPSE : EXPAND;
});

// ---- CRT filter (global post-process over the active viewer) ----
const crt = new CRT(stage);
crt.source = () => {
  const m = activeId && mounts.get(activeId);
  if (!m) return [];
  return [...m.el.querySelectorAll('canvas')].filter(c => getComputedStyle(c).display !== 'none');
};

const CRT_KEYS = ['curvature', 'beamFocus', 'maskStrength', 'glow', 'iqBlur', 'noise', 'maskType'];
const fmtCrt = (k, v) => k === 'maskType' ? (v < 0.5 ? 'Trinitron' : 'Shadow') : v.toFixed(2);

function wireCrt() {
  const saved = JSON.parse(localStorage.getItem('studio.crt') || '{}');
  for (const k of CRT_KEYS) {
    const el = document.getElementById('c_' + k);
    const valEl = document.getElementById('v_' + k);
    if (saved.params && saved.params[k] !== undefined) crt.set(k, saved.params[k]);
    el.value = crt.params[k];
    valEl.textContent = fmtCrt(k, crt.params[k]);
    el.addEventListener('input', () => {
      const v = parseFloat(el.value);
      crt.set(k, v);
      valEl.textContent = fmtCrt(k, v);
      persistCrt();
    });
  }
  const toggle = document.getElementById('crtToggle');
  const controls = document.getElementById('crtControls');
  const apply = (on) => {
    crt.setEnabled(on);
    controls.classList.toggle('shown', on);
    persistCrt();
  };
  const forced = new URLSearchParams(location.search).get('crt');
  const startOn = (forced === '1' || (forced !== '0' && !!saved.enabled)) && crt.ok;
  toggle.checked = startOn;
  controls.classList.toggle('shown', toggle.checked);
  toggle.addEventListener('change', () => apply(toggle.checked));
  if (!crt.ok) { toggle.disabled = true; }
  if (toggle.checked) crt.setEnabled(true);
}

function persistCrt() {
  const params = {};
  for (const k of CRT_KEYS) params[k] = crt.params[k];
  localStorage.setItem('studio.crt', JSON.stringify({ enabled: crt.enabled, params }));
}

// ---- boot ----
buildGameList();
wireCrt();
panel.classList.add('open');          // start with the control window open (discoverable)
// optional deep link: ?game=sonic&level=3
const params = new URLSearchParams(location.search);
const startGame = GAMES.find(g => g.id === params.get('game')) || GAMES[0];
const startLevel = parseInt(params.get('level'), 10);
selectGame(startGame.id).then(() => {
  if (Number.isInteger(startLevel) && startLevel > 0) selectLevel(startGame.id, startLevel);
});
