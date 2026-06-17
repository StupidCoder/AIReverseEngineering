// Fort Apocalypse viewer entry: build the viewer, populate the level selector,
// wire the layer toggles.
import { FortViewer } from './viewer.js';

const viewer = new FortViewer(
  document.getElementById('viewport'),
  document.getElementById('hud'),
);

const meta = await viewer.init();

const sel = document.getElementById('level');
meta.levels.forEach((l, i) => {
  const o = document.createElement('option');
  o.value = String(i); o.textContent = l.name;
  sel.appendChild(o);
});
sel.addEventListener('change', () => viewer.loadLevel(meta.levels[+sel.value]));

document.getElementById('animation').addEventListener('change', (e) => viewer.setLayer('animation', e.target.checked));
document.getElementById('objects').addEventListener('change', (e) => viewer.setLayer('objects', e.target.checked));

await viewer.loadLevel(meta.levels[0]);
