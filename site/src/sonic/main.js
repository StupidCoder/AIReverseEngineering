// Sonic viewer entry: build the viewer, populate the act selector, wire the controls.
import { LevelViewer } from './viewer.js';

const viewer = new LevelViewer(
  document.getElementById('viewport'),
  document.getElementById('hud'),
);

const meta = await viewer.init();

// act selector
const sel = document.getElementById('act');
meta.acts.forEach((a, i) => {
  const o = document.createElement('option');
  o.value = String(i); o.textContent = a.name;
  sel.appendChild(o);
});
sel.addEventListener('change', () => viewer.loadAct(meta.acts[+sel.value]));

// layer toggles
document.getElementById('collision').addEventListener('change', (e) => viewer.setLayer('collision', e.target.checked));
document.getElementById('objects').addEventListener('change', (e) => viewer.setLayer('objects', e.target.checked));

await viewer.loadAct(meta.acts[0]);
