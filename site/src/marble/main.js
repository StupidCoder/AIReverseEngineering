// Marble Madness viewer entry: build the viewer, populate the course selector.
import { MarbleViewer } from './viewer.js';

const viewer = new MarbleViewer(
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

await viewer.loadLevel(meta.levels[0]);
