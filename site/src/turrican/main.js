// Turrican viewer entry: build the viewer and populate the scene selector.
import { TurricanViewer } from './viewer.js';

const viewer = new TurricanViewer(
  document.getElementById('viewport'),
  document.getElementById('hud'),
);

const meta = await viewer.init();

const sel = document.getElementById('level');
meta.levels.forEach((l, i) => {
  const o = document.createElement('option');
  o.value = String(i);
  o.textContent = l.name;
  sel.appendChild(o);
});
sel.addEventListener('change', () => viewer.loadLevel(meta.levels[+sel.value]));

const objToggle = document.getElementById('objects');
objToggle.addEventListener('change', () => viewer.setObjects(objToggle.checked));

await viewer.loadLevel(meta.levels[0]);
