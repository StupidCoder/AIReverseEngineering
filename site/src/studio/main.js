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
import { KeyboardCamera } from './camera.js';
import { INFO_TABS, infoHtml } from './info-content.js';

const GAMES = [
  {
    id: 'sonic', name: 'Sonic the Hedgehog', system: 'Sega Game Gear',
    load: () => import('../sonic/viewer.js').then(m => m.LevelViewer),
    make: (V, el, hud) => new V(el, hud),
    list: async (v) => (await v.init()).acts,
    show: (v, lvl, i) => v.loadAct(lvl),
    // zone -> act accordion: "Green Hills Act 1" -> {Green Hills, Act 1}; "Special Stage 3" -> {Special Stage, Stage 3}
    group: (lvl) => {
      let mm = lvl.name.match(/^(.*?) (Act .+)$/);
      if (mm) return { section: mm[1], label: mm[2] };
      mm = lvl.name.match(/^(Special Stage) (\d+)$/);
      if (mm) return { section: mm[1], label: `Stage ${mm[2]}` };
      return { section: lvl.name, label: lvl.name };
    },
    layers: [
      { id: 'objects', label: 'Objects & enemies', default: true },
      { id: 'collision', label: 'Collision layer', default: false },
    ],
    music: async () => ['Green Hills:greenhills', 'Bridge:bridge', 'Jungle:jungle',
      'Labyrinth:labyrinth', 'Scrap Brain:scrapbrain', 'Sky Base:skybase', 'Special Stage:special']
      .map(s => { const [name, f] = s.split(':'); return { name, url: `public/sonic/music/${f}.mp3` }; }),
  },
  {
    id: 'fort', name: 'Fort Apocalypse', system: 'Commodore 64',
    load: () => import('../fort/viewer.js').then(m => m.FortViewer),
    make: (V, el, hud) => new V(el, hud),
    list: async (v) => (await v.init()).levels,
    show: (v, lvl, i) => v.loadLevel(lvl),
    layers: [{ id: 'objects', label: 'Objects & enemies', default: true }],
  },
  {
    id: 'turrican', name: 'Turrican', system: 'Amiga',
    load: () => import('../turrican/viewer.js').then(m => m.TurricanViewer),
    make: (V, el, hud) => new V(el, hud),
    list: async (v) => (await v.init()).levels,
    show: (v, lvl, i) => v.loadLevel(lvl),
    // world -> scene accordion from "World N · Scene M"
    group: (lvl) => {
      const [section, ...rest] = lvl.name.split(' · ');
      return { section, label: rest.join(' · ') || lvl.name };
    },
    layers: [
      { id: 'objects', label: 'Objects & enemies', default: true },
      { id: 'collision', label: 'Collision layer', default: false },
    ],
    music: async () => (await fetch('public/turrican/music/manifest.json').then(r => r.json()))
      .map(m => ({ name: turricanTrackName(m), url: `public/turrican/music/${m.file}` })),
  },
  {
    id: 'marble', name: 'Marble Madness', system: 'Amiga',
    load: () => import('../marble/viewer.js').then(m => m.MarbleViewer),
    make: (V, el, hud) => new V(el, hud),
    list: async (v) => (await v.init()).levels,
    show: (v, lvl, i) => v.loadLevel(lvl),
    // the Track markers only exist in the slope view, so only show the toggle there
    layers: [{ id: 'markers', label: 'Markers', default: false, when: (m) => m.leaves?.[m.currentIdx]?.name === 'Slopes' }],
    music: async () => (await fetch('public/marble/music/manifest.json').then(r => r.json()))
      .map(m => ({ name: m.course, url: `public/marble/music/${m.file}` })),
  },
  {
    id: 'sml', name: 'Super Mario Land', system: 'Nintendo Game Boy',
    load: () => import('../sml/viewer.js').then(m => m.SMLViewer),
    make: (V, el, hud) => new V(el, hud),
    list: async (v) => await v.init(), // returns the 12 level metas
    show: (v, lvl, i) => v.loadLevel(lvl),
    layers: [
      { id: 'objects', label: 'Objects & enemies', default: true },
      { id: 'collision', label: 'Collision layer', default: false },
    ],
    // world -> level accordion: "2-1" -> { World 2, 2-1 }
    group: (lvl) => ({ section: `World ${lvl.world}`, label: lvl.name }),
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
    // open on the Cobra Mk III — the iconic player ship — rather than the missile
    defaultAsset: (ships) => ships.findIndex(s => s.name === 'Cobra Mk III'),
    // the docking music (The Blue Danube), rendered from the $BDDC engine through our SID emulator
    music: async () => [{ name: 'Docking — The Blue Danube', url: 'public/elite/music/docking.mp3' }],
  },
];

// Turrican's manifest labels worlds 0-based with hex start offsets; make them readable.
function turricanTrackName(m) {
  let l = String(m.label || 'Music').replace(/world (\d+)/i, (_, n) => 'World ' + (Number(n) + 1));
  l = l.charAt(0).toUpperCase() + l.slice(1);
  if (m.start && m.start !== '0') l += ` · $${m.start}`;
  return l;
}

const stage = document.getElementById('stage');
const hud = document.getElementById('hud');
// The HUD is two stacked lines: a caption the Studio owns (game · system — asset)
// and a detail line the active viewer owns (dimensions / zoom info). They no longer
// clobber each other, so the caption stays put while the viewer updates its detail.
const hudCaption = document.createElement('div'); hudCaption.className = 'hud-caption';
const hudDetail = document.createElement('div'); hudDetail.className = 'hud-detail';
hud.append(hudCaption, hudDetail);
const panel = document.getElementById('panel');
const gameList = document.getElementById('gameList');
const assetList = document.getElementById('assetList');
const assetLabel = document.getElementById('assetLabel');
const titlecard = document.getElementById('titlecard');
const spinner = document.getElementById('spinner');

const mounts = new Map(); // gameId -> { game, el, viewer, levels, current }
let activeId = null;
let busy = false;

// ---- the game list: systems (collapsible, one open at a time) -> games ----
const SYSTEMS = [
  { full: 'Amiga', short: 'Amiga' },
  { full: 'Commodore 64', short: 'C64' },
  { full: 'Sega Game Gear', short: 'Game Gear' },
  { full: 'Nintendo Game Boy', short: 'Game Boy' },
];
const CHEVRON = '<svg class="chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 6l6 6-6 6"/></svg>';

function buildGameList() {
  gameList.innerHTML = '';
  for (const sys of SYSTEMS) {
    const games = GAMES.filter(g => g.system === sys.full);
    if (!games.length) continue;
    const group = document.createElement('div');
    group.className = 'sys-group';
    group.dataset.sys = sys.full;

    const header = document.createElement('button');
    header.className = 'item sys-header';
    header.innerHTML = `<span class="name">${sys.short}</span>${CHEVRON}`;
    header.addEventListener('click', () => toggleSystem(sys.full));
    group.appendChild(header);

    const sub = document.createElement('div');
    sub.className = 'sys-games';
    for (const game of games) {
      const b = document.createElement('button');
      b.className = 'item game-item';
      b.dataset.game = game.id;
      b.innerHTML = `<span class="name">${game.name}</span>`;
      b.addEventListener('click', () => selectGame(game.id));
      sub.appendChild(b);
    }
    group.appendChild(sub);
    gameList.appendChild(group);
  }
}

function openSystem(sysFull) {
  for (const group of gameList.querySelectorAll('.sys-group')) {
    group.classList.toggle('open', group.dataset.sys === sysFull);
  }
}

function toggleSystem(sysFull) {
  const group = gameList.querySelector(`.sys-group[data-sys="${sysFull}"]`);
  openSystem(group && group.classList.contains('open') ? null : sysFull);
}

function markActiveGame(id) {
  const game = GAMES.find(g => g.id === id);
  for (const b of gameList.querySelectorAll('.game-item')) b.classList.toggle('active', b.dataset.game === id);
  if (game) openSystem(game.system); // unfold the active game's system
}

// ---- asset list (per game). Most games are a flat list. Marble Madness is a two-level
//      accordion (course -> { Map, Slopes }); games with a `group(lvl,i)` adapter become a
//      world/zone -> act/scene accordion. Each leaf has { name, hud, run }. ----
function assetEntries(m) {
  const { game, viewer, levels } = m;
  if (game.id === 'marble') {
    return levels.map((course, ci) => ({
      name: course.name,
      children: [
        { name: 'Map', hud: `${course.name} · Map`, run: async () => { await game.show(viewer, course, ci); viewer.setMode('tilemap'); applyLayers(m); } },
        { name: 'Slopes', hud: `${course.name} · Slopes`, run: async () => { await game.show(viewer, course, ci); viewer.setMode('slopes'); applyLayers(m); } },
      ],
    }));
  }

  const leaf = (lvl, i, name) => ({
    name,
    hud: lvl.name || `Asset ${i + 1}`,
    run: async () => { await game.show(viewer, lvl, i); applyLayers(m); },
  });

  // grouped: fold the flat level list into section accordions via the adapter's group()
  if (game.group) {
    const groups = [];
    const byKey = new Map();
    levels.forEach((lvl, i) => {
      const { section, label } = game.group(lvl, i);
      let g = byKey.get(section);
      if (!g) { g = { name: section, children: [] }; byKey.set(section, g); groups.push(g); }
      g.children.push(leaf(lvl, i, label));
    });
    return groups;
  }

  return levels.map((lvl, i) => leaf(lvl, i, lvl.name || `Asset ${i + 1}`));
}

function addLeaf(m, leaf, parent) {
  const idx = m.leaves.length;
  m.leaves.push(leaf);
  const b = document.createElement('button');
  b.className = 'item asset-item';
  b.dataset.idx = idx;
  b.innerHTML = `<span class="name">${leaf.name}</span>`;
  b.addEventListener('click', () => selectAsset(m, idx));
  parent.appendChild(b);
}

function buildAssetList(m) {
  assetLabel.style.display = ''; // reveal the Asset section once a game is chosen
  assetList.innerHTML = '';
  m.leaves = [];
  for (const entry of assetEntries(m)) {
    if (entry.children) {
      const group = document.createElement('div');
      group.className = 'asset-group';
      const header = document.createElement('button');
      header.className = 'item asset-header';
      header.innerHTML = `<span class="name">${entry.name}</span>${CHEVRON}`;
      header.addEventListener('click', () => toggleAssetGroup(group));
      group.appendChild(header);
      const sub = document.createElement('div');
      sub.className = 'asset-children';
      for (const leaf of entry.children) addLeaf(m, leaf, sub);
      group.appendChild(sub);
      assetList.appendChild(group);
    } else {
      addLeaf(m, entry, assetList);
    }
  }
}

function toggleAssetGroup(group) {
  const open = group.classList.contains('open');
  for (const g of assetList.querySelectorAll('.asset-group')) g.classList.toggle('open', !open && g === group);
}

function markActiveAsset(m) {
  for (const b of assetList.querySelectorAll('.asset-item')) b.classList.toggle('active', +b.dataset.idx === m.currentIdx);
  const active = assetList.querySelector(`.asset-item[data-idx="${m.currentIdx}"]`);
  const grp = active && active.closest('.asset-group');
  for (const g of assetList.querySelectorAll('.asset-group')) g.classList.toggle('open', g === grp);
  if (active) active.scrollIntoView({ block: 'nearest' });
}

// the leaf shown on a game's first visit: the adapter's defaultAsset (a fn of the level list,
// returning a level/leaf index — leaves are built in level order), else the first leaf.
function defaultLeaf(m) {
  const i = m.game.defaultAsset ? m.game.defaultAsset(m.levels) : 0;
  return (Number.isInteger(i) && i >= 0 && i < m.leaves.length) ? i : 0;
}

// run an asset (no busy management — used for the initial selection inside selectGame).
async function runAsset(m, idx) {
  const leaf = m.leaves[idx];
  if (!leaf) return;
  m.currentIdx = idx;
  m.currentName = leaf.hud;
  markActiveAsset(m);
  buildLayerToggles(m); // some toggles are asset-specific (e.g. Marble's Markers = Slopes only)
  await leaf.run();
}

async function selectAsset(m, idx) {
  if (busy || idx === m.currentIdx) return;
  setBusy(true);
  try {
    await runAsset(m, idx);
    updateHud(m);
  } catch (err) {
    console.error('studio: failed to show asset', idx, err);
  } finally {
    setBusy(false);
  }
}

function setBusy(on) {
  busy = on;
  spinner.classList.toggle('on', on);
}

// Pause a hidden viewer's render loop (and resume the shown one) so cached viewers don't
// keep rendering in the background. Pixi via the ticker; three.js loops gate on viewer.active.
function setMountActive(m, active) {
  if (!m) return;
  const v = m.viewer;
  v.active = active;
  if (v.app && typeof v.app.stop === 'function') v.app[active ? 'start' : 'stop']();
}

// ---- selection ----
async function selectGame(id) {
  if (busy || id === activeId) return;
  const game = GAMES.find(g => g.id === id);
  markActiveGame(id);
  setBusy(true);
  try {
    // hide and pause the currently mounted viewer
    if (activeId && mounts.has(activeId)) {
      const old = mounts.get(activeId);
      old.el.style.display = 'none';
      setMountActive(old, false);
    }

    let m = mounts.get(id);
    const firstMount = !m;
    if (!m) {
      const el = document.createElement('div');
      el.className = 'mount';
      el.dataset.render = game.render || '2d';
      stage.appendChild(el);
      const Viewer = await game.load();
      const viewer = game.make(Viewer, el, hudDetail);
      const levels = await game.list(viewer);
      m = { game, el, viewer, levels, currentIdx: 0, currentName: '' };
      mounts.set(id, m);
    }
    m.el.style.display = 'block';
    setMountActive(m, true);
    activeId = id;
    buildLayerToggles(m);
    buildAssetList(m);
    if (firstMount) await runAsset(m, defaultLeaf(m)); // load the default asset on first visit
    else markActiveAsset(m);                           // returning to a cached viewer: keep its asset
    await loadGameMusic(game);
    updateHud(m);
    if (infoPanel.classList.contains('open')) renderInfo(); // refresh the details for the new game
    hideTitlecard();
  } catch (err) {
    console.error('studio: failed to load', id, err);
    hudCaption.innerHTML = `<b>${game.name}</b> — failed to load (${err.message})`;
  } finally {
    setBusy(false);
  }
}

function updateHud(m) {
  hudCaption.innerHTML = `<b>${m.game.name}</b> · ${m.game.system} &nbsp;—&nbsp; ${m.currentName}`;
}

function hideTitlecard() {
  if (titlecard.style.display === 'none') return;
  titlecard.classList.add('hidden');
  setTimeout(() => { titlecard.style.display = 'none'; }, 500); // fade out, then drop it for good
}

// ---- menu / panel ----
const menuBtn = document.getElementById('menuBtn');
function setMenu(open) {
  panel.classList.toggle('open', open);
  menuBtn.classList.toggle('hidden', open); // the panel grows out of the button; hide it while open
}
menuBtn.addEventListener('click', () => setMenu(true));
document.getElementById('panelClose').addEventListener('click', () => setMenu(false));

// ---- info panel (technical details, tabbed) ----
// A second window, top-right, that fills the space the control window leaves. Its header is a
// fixed row of tabs (folded from the games' Markdown parts); the body scrolls. Content is keyed
// by the active game and the selected tab and re-rendered whenever either changes.
const infoBtn = document.getElementById('infoBtn');
const infoPanel = document.getElementById('info');
const infoTabsEl = document.getElementById('infoTabs');
const infoBody = document.getElementById('infoBody');
let infoTab = INFO_TABS[0].id;

function buildInfoTabs() {
  infoTabsEl.innerHTML = '';
  for (const t of INFO_TABS) {
    const b = document.createElement('button');
    b.className = 'info-tab' + (t.id === infoTab ? ' active' : '');
    b.dataset.tab = t.id;
    b.textContent = t.label;
    b.addEventListener('click', () => selectInfoTab(t.id));
    infoTabsEl.appendChild(b);
  }
}

function selectInfoTab(id) {
  infoTab = id;
  for (const b of infoTabsEl.querySelectorAll('.info-tab')) b.classList.toggle('active', b.dataset.tab === id);
  renderInfo();
}

function renderInfo() {
  const game = activeId && GAMES.find(g => g.id === activeId);
  if (!game) {
    infoBody.innerHTML = `<div class="info-doc info-empty">Pick a game from the menu to read its technical notes.</div>`;
    infoBody.scrollTop = 0;
    return;
  }
  const tab = INFO_TABS.find(t => t.id === infoTab);
  const html = infoHtml(game.id, infoTab);
  infoBody.innerHTML = `<div class="info-doc">` + (html ||
    `<div class="info-eyebrow">${game.name} · ${tab.label}</div>
     <p class="info-todo">This section hasn't been written yet.</p>`) + `</div>`;
  infoBody.scrollTop = 0;
}

function setInfo(open) {
  infoPanel.classList.toggle('open', open);
  infoBtn.classList.toggle('hidden', open); // the panel grows out of the button; hide it while open
  if (open) renderInfo();
}
infoBtn.addEventListener('click', () => setInfo(true));
document.getElementById('infoClose').addEventListener('click', () => setInfo(false));
buildInfoTabs();

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

// ---- per-game display layers (e.g. Fort's objects/enemies overlay) ----
// Not universal: each game adapter declares its own `layers` ([{id,label,default}]); games without
// one show no toggles. State is per game, persisted, and re-applied every time an asset (re)loads.
const displayLayers = document.getElementById('displayLayers');
const savedLayers = JSON.parse(localStorage.getItem('studio.layers') || '{}');

function layerState(m) {
  if (!m.layerState) {
    m.layerState = {};
    const saved = savedLayers[m.game.id] || {};
    for (const l of m.game.layers || []) m.layerState[l.id] = (l.id in saved) ? saved[l.id] : !!l.default;
  }
  return m.layerState;
}

function applyLayers(m) {
  const st = layerState(m);
  for (const l of m.game.layers || []) m.viewer.setLayer(l.id, st[l.id]);
}

function persistLayers(m) {
  savedLayers[m.game.id] = { ...layerState(m) };
  localStorage.setItem('studio.layers', JSON.stringify(savedLayers));
}

function buildLayerToggles(m) {
  // a layer may be asset-specific (l.when) -- only show toggles that apply to the current asset
  const layers = (m && m.game.layers || []).filter(l => !l.when || l.when(m));
  displayLayers.innerHTML = '';
  displayLayers.style.display = layers.length ? '' : 'none';
  if (!m) return;
  const st = layerState(m);
  for (const l of layers) {
    const label = document.createElement('label');
    label.className = 'switch';
    const input = document.createElement('input');
    input.type = 'checkbox';
    input.checked = st[l.id];
    const track = document.createElement('span');
    track.className = 'track';
    track.innerHTML = '<span class="knob"></span>';
    const text = document.createElement('span');
    text.className = 'switch-label';
    text.textContent = l.label;
    label.append(input, track, text);
    input.addEventListener('change', () => {
      st[l.id] = input.checked;
      m.viewer.setLayer(l.id, input.checked);
      persistLayers(m);
    });
    displayLayers.appendChild(label);
  }
}

// ---- CRT filter (global post-process over the active viewer) ----
const crt = new CRT(stage);
crt.source = () => {
  const m = activeId && mounts.get(activeId);
  if (!m) return [];
  return [...m.el.querySelectorAll('canvas')].filter(c => getComputedStyle(c).display !== 'none');
};

const CRT_KEYS = ['curvature', 'beamFocus', 'maskStrength', 'glow', 'iqBlur', 'noise', 'maskType', 'signalLines', 'scanLines'];
const INT_CRT = new Set(['signalLines', 'scanLines']);
const fmtCrt = (k, v) => k === 'maskType' ? (v < 0.5 ? 'Trinitron' : 'Shadow') : INT_CRT.has(k) ? String(Math.round(v)) : v.toFixed(2);

function syncCrtControls() {
  for (const k of CRT_KEYS) {
    document.getElementById('c_' + k).value = crt.params[k];
    document.getElementById('v_' + k).textContent = fmtCrt(k, crt.params[k]);
  }
}

function wireCrt() {
  const saved = JSON.parse(localStorage.getItem('studio.crt') || '{}');
  for (const k of CRT_KEYS) {
    const el = document.getElementById('c_' + k);
    const valEl = document.getElementById('v_' + k);
    if (saved.params && saved.params[k] !== undefined) crt.set(k, saved.params[k]);
    el.addEventListener('input', () => {
      const v = parseFloat(el.value);
      crt.set(k, v);
      valEl.textContent = fmtCrt(k, v);
      persistCrt();
    });
  }
  syncCrtControls();
  document.getElementById('crtReset').addEventListener('click', () => {
    crt.reset();
    syncCrtControls();
    persistCrt();
  });
  const toggle = document.getElementById('crtToggle');
  const controls = document.getElementById('crtControls');
  const gear = document.getElementById('crtSettings');
  toggle.addEventListener('change', () => { crt.setEnabled(toggle.checked); persistCrt(); });
  gear.addEventListener('click', () => gear.classList.toggle('on', controls.classList.toggle('shown')));

  // enabled by default; only an explicit prior "off" (or ?crt=0) turns it off.
  const forced = new URLSearchParams(location.search).get('crt');
  const startOn = (forced === '1' || (forced !== '0' && saved.enabled !== false)) && crt.ok;
  toggle.checked = startOn;
  if (!crt.ok) toggle.disabled = true;
  if (startOn) crt.setEnabled(true);
  // the settings panel stays folded until the gear is clicked
}

function persistCrt() {
  const params = {};
  for (const k of CRT_KEYS) params[k] = crt.params[k];
  localStorage.setItem('studio.crt', JSON.stringify({ enabled: crt.enabled, params }));
}

// ---- Music player (per-game song list + transport). Switching games stops the music;
//      switching levels within a game leaves it playing. ----
const audio = new Audio();
audio.preload = 'none';
let musicTracks = [];
let playingUrl = null;
let repeat = localStorage.getItem('studio.repeat') !== '0'; // loop the current song; on by default
audio.loop = repeat;
const musicLabel = document.getElementById('musicLabel');
const musicListEl = document.getElementById('musicList');
const transport = document.getElementById('musicTransport');
const musPlay = document.getElementById('musPlay');
const musSeek = document.getElementById('musSeek');
const musTime = document.getElementById('musTime');
const PLAY_SVG = '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M7 5v14l12-7z"/></svg>';
const PAUSE_SVG = '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M7 5h3v14H7zM14 5h3v14h-3z"/></svg>';
musPlay.innerHTML = PLAY_SVG;

const fmtTime = (s) => { s = Math.max(0, s | 0); return Math.floor(s / 60) + ':' + String(s % 60).padStart(2, '0'); };

function renderMusicList() {
  musicListEl.innerHTML = '';
  musicTracks.forEach((t, i) => {
    const b = document.createElement('button');
    b.className = 'item' + (t.url === playingUrl ? ' active' : '');
    b.innerHTML = `<span class="name">${t.name}</span>`;
    b.addEventListener('click', () => playTrack(i));
    musicListEl.appendChild(b);
  });
}

function updateMusicUI() {
  const hasMusic = musicTracks.length > 0;
  const loaded = !!playingUrl;
  const show = hasMusic || loaded;
  musicLabel.style.display = show ? '' : 'none';
  musicListEl.style.display = hasMusic ? '' : 'none';
  transport.style.display = show ? 'flex' : 'none';
}

function playTrack(i) {
  const t = musicTracks[i];
  if (!t) return;
  if (playingUrl !== t.url) { audio.src = t.url; playingUrl = t.url; }
  audio.play().catch(() => {});
  renderMusicList();
  updateMusicUI();
}

function skip(d) {
  if (!musicTracks.length) return;
  let idx = musicTracks.findIndex(t => t.url === playingUrl);
  if (idx < 0) idx = d > 0 ? -1 : 0;
  playTrack((idx + d + musicTracks.length) % musicTracks.length);
}

musPlay.addEventListener('click', () => {
  if (!playingUrl) { if (musicTracks.length) playTrack(0); return; }
  if (audio.paused) audio.play().catch(() => {}); else audio.pause();
});
document.getElementById('musPrev').addEventListener('click', () => skip(-1));
document.getElementById('musNext').addEventListener('click', () => skip(1));
const musRepeat = document.getElementById('musRepeat');
function syncRepeat() {
  audio.loop = repeat;
  musRepeat.classList.toggle('on', repeat);
  musRepeat.title = repeat ? 'Repeat: on' : 'Repeat: off';
}
musRepeat.addEventListener('click', () => {
  repeat = !repeat;
  localStorage.setItem('studio.repeat', repeat ? '1' : '0');
  syncRepeat();
});
syncRepeat();
audio.addEventListener('play', () => { musPlay.innerHTML = PAUSE_SVG; musPlay.title = 'Pause'; });
audio.addEventListener('pause', () => { musPlay.innerHTML = PLAY_SVG; musPlay.title = 'Play'; });
audio.addEventListener('ended', () => skip(1));
audio.addEventListener('timeupdate', () => {
  if (audio.duration) musSeek.value = String(Math.round(audio.currentTime / audio.duration * 1000));
  musTime.textContent = fmtTime(audio.currentTime);
});
musSeek.addEventListener('input', () => { if (audio.duration) audio.currentTime = (musSeek.value / 1000) * audio.duration; });

function stopMusic() {
  audio.pause();
  try { audio.currentTime = 0; } catch {}
  playingUrl = null;
  musPlay.innerHTML = PLAY_SVG; musPlay.title = 'Play';
  musSeek.value = '0';
  musTime.textContent = '0:00';
}

async function loadGameMusic(game) {
  stopMusic(); // switching games stops the music
  try { musicTracks = game.music ? await game.music() : []; }
  catch (e) { console.error('studio: music load failed', game.id, e); musicTracks = []; }
  renderMusicList();
  updateMusicUI();
}

// ---- keyboard camera: cursor keys scroll (with momentum), +/- zoom the active viewer ----
new KeyboardCamera(() => {
  const m = activeId && mounts.get(activeId);
  return m ? m.viewer : null;
});

// ---- boot ----
buildGameList();
wireCrt();
displayLayers.style.display = 'none'; // per-game display toggles appear once a game is picked
assetLabel.style.display = 'none';    // the Asset + Music sections stay hidden until a game
updateMusicUI();                      // is picked (the splash is up meanwhile)
setMenu(true);                        // start with the control window open (discoverable)
// Keep the title card up until the user picks a game -- unless a ?game= deep link asks for
// one (?game=sonic&asset=3, asset = leaf index in the list), in which case load it straight.
const params = new URLSearchParams(location.search);
const startGame = GAMES.find(g => g.id === params.get('game'));
const startAsset = parseInt(params.get('asset') ?? params.get('level'), 10);
if (startGame) {
  selectGame(startGame.id).then(() => {
    const m = mounts.get(startGame.id);
    if (m && Number.isInteger(startAsset) && startAsset > 0 && startAsset < m.leaves.length) selectAsset(m, startAsset);
  });
}
