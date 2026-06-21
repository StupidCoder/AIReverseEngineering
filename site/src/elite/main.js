// Elite ship viewer entry: build the viewer, then a horizontally scrollable
// strip of thumbnails (rendered client-side from the same blueprint geometry,
// so each thumbnail matches the opening pose of the main viewport). Clicking a
// thumbnail loads that ship.
import { ShipViewer } from './viewer.js';

const THUMB = 128; // render resolution; displayed smaller via CSS

const viewer = new ShipViewer(
  document.getElementById('viewport'),
  document.getElementById('hud'),
);

const ships = await viewer.init();

const strip = document.getElementById('ships');
const buttons = [];
ships.forEach((ship, i) => {
  const btn = document.createElement('button');
  btn.className = 'thumb';
  btn.title = `${ship.name} (type ${ship.type})`;

  const canvas = document.createElement('canvas');
  canvas.width = THUMB;
  canvas.height = THUMB;

  const label = document.createElement('span');
  label.className = 'thumb-label';
  label.textContent = ship.name;

  btn.appendChild(canvas);
  btn.appendChild(label);
  btn.addEventListener('click', () => select(i));
  strip.appendChild(btn);
  buttons.push(btn);

  // Render the thumbnail once the strip is in the DOM.
  viewer.renderThumbnail(i, canvas, THUMB);
});

function select(i) {
  viewer.loadShip(i);
  buttons.forEach((b, j) => b.classList.toggle('active', j === i));
  buttons[i].scrollIntoView({ behavior: 'smooth', inline: 'nearest', block: 'nearest' });
}

// "Old school" effects — four independent toggles so each can be judged alone.
const FX = { crt: 'fx-crt', lowRes: 'fx-lowres', flicker: 'fx-flicker', lowFps: 'fx-lowfps' };
for (const [name, id] of Object.entries(FX)) {
  const el = document.getElementById(id);
  if (el) el.addEventListener('change', () => viewer.setEffect(name, el.checked));
}

// Open with the Cobra Mk III (the player's ship and the cover star) if present.
const startIndex = Math.max(0, ships.findIndex((s) => s.type === 11));
select(startIndex);
