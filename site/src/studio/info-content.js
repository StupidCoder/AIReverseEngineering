// Technical-manual text for the info panel, one entry per game per tab. The prose is derived
// from each game's Markdown write-up but rewritten in a neutral reference style: the
// reverse-engineering narrative and history in the source docs are dropped, leaving a
// description of how the shipped game works.
//
// The five tabs fold the Markdown's parts into reader-facing sections:
//   loader   -> Parts I & II  (the disk/tape image and the boot/loader chain)
//   engine   -> Part III      (the game program's architecture / main loop)
//   graphics -> Part IV       (graphics and data formats)
//   music    -> Part VI       (the music engine and tracks)
//   gameplay -> Part V        (game mechanics)
//
// Content is filled in over subsequent passes. INFO_CONTENT[gameId][tabId] is an HTML string
// (rendered inside .info-doc); a missing entry shows a "not written yet" placeholder.

export const INFO_TABS = [
  { id: 'loader', label: 'Image & Loader' },
  { id: 'engine', label: 'Game Engine' },
  { id: 'graphics', label: 'Graphics' },
  { id: 'music', label: 'Music' },
  { id: 'gameplay', label: 'Gameplay' },
];

export const INFO_CONTENT = {
  sonic: {},
  fort: {
    loader: `
<div class="info-eyebrow">Fort Apocalypse · Image &amp; Loader</div>
<p>Fort Apocalypse ships on a single cassette, preserved as a <strong>TAP image</strong>: a
recording of the raw signal the C64 reads off tape. Loading runs in two phases — a short
bootstrap in the ROM's standard tape format, then a custom high-speed loader that streams the
rest of the game in while a U.S. Gold loading screen plays.</p>

<h2>The tape image</h2>
<p>A TAP file stores the cassette signal as a list of pulse lengths, one byte each. After a
20-byte header (the magic string <code>C64-TAPE-RAW</code>, a version byte and a little-endian
data length), every non-zero byte is a single pulse of <code>n &times; 8</code> clock cycles —
985,248&nbsp;Hz on PAL. A zero byte marks a pause and is followed by a 24-bit cycle count. Only a
handful of distinct pulse widths carry data: roughly 300 and 670 cycles for the fastloader's 0 and
1 bits, and three medium widths used by the KERNAL bootstrap.</p>
<p>Three regions sit back to back, separated by pauses: a KERNAL header block, a KERNAL data block,
and the fastloader stream. The two KERNAL blocks are each recorded twice for reliability.</p>

<h3>The KERNAL bootstrap</h3>
<p>The first two blocks use the C64's built-in ROM tape format. Each bit is a <em>pair</em> of
pulses (short+medium = 0, medium+short = 1); a byte is a marker pair, eight data bits LSB-first,
and an odd parity bit. Each record carries a pilot tone, a nine-byte countdown sync sequence, the
payload, and an XOR checksum. Two records load:</p>
<ul>
  <li>a <strong>header block</strong> announcing a relocatable BASIC program named <code>FORT</code>.
  Its payload is nominally the 16-character filename — but the bytes after the name are not padding.
  The KERNAL copies the whole header into the cassette buffer at <code>$033C</code>, which quietly
  plants machine code at <strong><code>$0351–$03F5</code></strong>: the fastloader's interrupt
  handler.</li>
  <li>a <strong>data block</strong> loaded to <code>$0801</code> — a one-line BASIC program,
  <code>SYS 2061</code>, followed by the loader-setup code at <code>$080D</code>.</li>
</ul>
<p>So <code>LOAD"",1</code> then <code>RUN</code> runs the BASIC stub, which <code>SYS</code>es into
the setup routine — and the real loader is already resident in the tape buffer, smuggled in
disguised as a filename.</p>

<h3>The NOVALOAD fastloader</h3>
<p>The bulk of the game arrives through a custom turbo loader (it names itself <strong>NOVALOAD</strong>,
serial D100701, on screen) that reads <strong>one bit per pulse</strong> rather than two. CIA timer A
is latched and force-reloaded on every cassette edge; the interrupt handler reads the timer's high
byte and treats a short pulse as 0, a long pulse as 1. Bits are rotated in with <code>ROR</code>, so
the first pulse of a byte is its least significant bit. The shift register starts at <code>$7F</code>,
and any run of eight-or-more zero bits ending in a one bit reads back as the pilot byte
<code>$80</code> — which is how the decoder self-synchronises without an explicit reset.</p>
<p>The stream is a pilot tone, a sync byte (<code>$AA</code>), a key byte (<code>$55</code>), then
<strong>84 records</strong>, each a page number, 256 data bytes, and a checksum
(<code>page + sum of bytes, mod 256</code>). Every record loads to <code>page &lt;&lt; 8</code>, so
pages may arrive in any order and gaps are harmless. One record carries page <code>$F0</code>, which
arms "end mode"; after it, a page number of <code>$00</code> ends the load. The pages come in two
groups: first the stage-2 loading screen (<code>$E000–$E6FF</code> and <code>$EE00–$F1FF</code>),
then the main game (<code>$7000–$B8FF</code>) streamed in behind it.</p>

<h2>The boot chain</h2>
<p>End to end, control flows:</p>
<ol>
  <li><code>SYS 2061</code> runs the loader setup at <code>$080D</code>.</li>
  <li>Setup banks out the BASIC and KERNAL ROMs, points the CPU's IRQ vector at the planted handler
  at <code>$0351</code>, arms a CIA FLAG interrupt that fires once per tape pulse, and busy-waits
  until page <code>$F0</code> has arrived.</li>
  <li>With the loading screen now in memory, it calls stage 2 at <code>$E000</code> while the
  interrupt keeps streaming the game in the background.</li>
  <li>On success, stage 2 fades the music, banks the ROMs back in, and jumps to the game's
  initialisation at <code>$8600</code>.</li>
</ol>

<h3>Loader setup ($080D)</h3>
<p>Besides redirecting interrupts, setup clears the screen and prints the filename and the
<code>NOVALOAD D100701</code> banner, primes the SID for the loading-noise effect (each loaded byte
is also written to a SID register), and lays out its zero-page state: a store pointer, a page
offset, the checksum seed, and a status byte (loading / done / error). It also aims the BASIC text
pointer at a planted <code>:RUN</code> token sequence — a decoy that makes a memory snapshot look
like a harmless return to BASIC.</p>

<h3>The interrupt handler ($0351)</h3>
<p>The handler runs once per tape pulse. After demodulating the bit and assembling a byte, a
self-modifying branch offset dispatches a small state machine: search for the pilot, match the sync
byte, verify the checksum, read a page number (<code>$F0</code> arming end mode, <code>$00</code>
afterwards completing the load), or store 256 data bytes and accumulate the checksum. A bad checksum
sets the error status and halts the load.</p>

<h2>The loading screen</h2>
<p>While the game streams in, stage 2 paints the U.S. Gold loading screen and runs a scroller and
three-voice music. The screen is drawn from a compact <strong>display script</strong> — border and
background colours, then runs of screen codes with a single escape byte for newlines, colour changes
and end-of-script. It includes the three-digit "BLOCKS TO LOAD" counter that the tape interrupt
decrements as each page arrives. The scrolling message is stored <strong>reversed</strong> and read
backwards through a self-modified pointer.</p>

<h3>A tune that is also a program</h3>
<p>The music is not merely audio: it is a small bytecode the player interprets. Commands play a note
for a duration, set the read pointer (to loop the tune), or — the notable one — copy the next
<em>n</em> stream bytes to <strong>any address in memory</strong>, implemented by patching the
operand of a store instruction. The tune uses that copy command, on its very first tick, to rewrite
the machine itself:</p>
<ul>
  <li>it redirects the KERNAL NMI vector so that RUN/STOP–RESTORE becomes a clean no-op during play;</li>
  <li>it re-initialises the SID and some player variables;</li>
  <li>and, crucially, it overwrites the loader's epilogue at <strong><code>$03F5</code></strong> with
  <code>JMP $8600</code> — the jump that actually starts the game.</li>
</ul>
<p>As loaded from tape, that epilogue ends in an innocuous <code>RTS</code> followed by the decoy
<code>:RUN</code> bytes. The real entry address appears nowhere in the code; it exists only as data
inside the music stream, and only the act of playing the tune assembles it.</p>

<h3>Error handling</h3>
<p>On a clean load, stage 2 fades the volume over a few seconds, clears the SID, restores the ROMs,
and takes the patched jump into the game. If a tape error stalls the loader — detected as a frozen
byte counter — stage 2 instead <strong>wipes all RAM except its own page and jumps through the reset
vector</strong>, a response that is as much anti-tamper as error recovery.</p>
`,
    engine: `
<div class="info-eyebrow">Fort Apocalypse · Game Engine</div>
<p>Once loaded, Fort Apocalypse is an almost entirely <strong>interrupt-driven</strong> program. A
brief setup routine builds the world in memory and arms a raster interrupt, then deliberately parks
the processor in a tight infinite loop — every frame of the game is produced by the raster handlers
and a main loop that they release.</p>

<h2>Initialization ($8600)</h2>
<p>Entry at <code>$8600</code> jumps straight to the init routine at <code>$8927</code>. It runs once:</p>
<ul>
  <li><code>SEI</code>, <code>CLD</code>, clear zero page, and set <code>$01 = $2E</code> — <strong>BASIC
  ROM banked out, KERNAL left in</strong>, so the game's own code in the <code>$A000–$B8FF</code> region
  is called directly underneath where BASIC used to be.</li>
  <li>Point the VIC at bank 1 (<code>$4000–$7FFF</code>) through CIA2, with the screen at
  <code>$4400</code>.</li>
  <li>Reset the SID — and set voice 3 to noise, whose output at <code>$D41B</code> becomes the game's
  <strong>random-number source</strong>.</li>
  <li>Zero <code>$0380–$6FFF</code>, then build both character sets and expand all sprite shapes
  (see Graphics).</li>
  <li>Draw the HUD frame and title text with a double-width font renderer: each glyph is drawn as
  character <code>n</code> alongside character <code>n+$20</code>.</li>
  <li>Install the title raster interrupt at line <code>$F9</code> and finish with
  <code>$8A9F: JMP $8A9F</code> — a one-instruction halt. Everything after this point happens inside
  interrupts.</li>
</ul>

<h2>The raster architecture</h2>
<p>The display is split into two horizontal bands, each served by its own interrupt handler that
reprograms the VIC mid-frame and chains to the next split:</p>
<ul>
  <li><strong>Line <code>$F9</code> — the HUD handler (<code>$9BD4</code>).</strong> Selects the HUD
  character set (<code>$D018 = $14</code>, charset <code>$5000</code>), sets the scroll registers,
  latches the sprite-collision registers, increments the frame counter, reads keyboard and joystick,
  updates the player sprite, bullets and the enemy sprite, drives sound, and schedules the next
  interrupt for line <code>$76</code>.</li>
  <li><strong>Line <code>$76</code> — the playfield handler (<code>$AE19</code>).</strong> Selects the
  playfield character set (<code>$D018 = $16</code>, charset <code>$5800</code>), applies fine
  scrolling, sets the per-level colours, runs the in-place charset animations, copies the scrolling
  playfield window, applies SID effects, and schedules the next interrupt back at line
  <code>$F9</code>.</li>
</ul>
<p>The consequence of the split is that screen rows 0–6 (the HUD and scanner) and rows 7–24 (the
playfield) are drawn from <strong>two different character sets</strong>, swapped partway down every
frame.</p>

<h2>The main loop and game state</h2>
<p>The main game loop lives at <code>$8BB1</code>. It is entered from the title interrupt by a
stack-resetting jump the moment fire is pressed, and from then on it waits for the frame counter to
change, runs the per-frame logic chain — the object engines, zone checks and state dispatch — and
loops. Because it is gated on the frame counter, the loop runs in lock-step with the raster handlers
that drive the screen.</p>
<p>A single byte at <code>$9D</code> holds the overall game state and selects what that chain does:
<code>1</code> title / attract, <code>9</code> demo game, <code>3</code> new game, <code>4</code> "get
ready", <code>5</code> life lost, <code>2</code> playing, <code>6</code> game over and debrief,
<code>7</code> a transition lock, and <code>$0A</code> the cavern teleport.</p>

<h2>Memory layout in play</h2>
<p>With the ROMs banked the way they are, the 64&nbsp;KB address space is densely packed. Zero page
holds the live state — game state, frame counter, the camera position, the player block and a set of
pointers. The VIC's bank 1 contains the screen at <code>$4400</code>, the sprite shape blocks at
<code>$4000</code> (blocks 1–14 are the enemy helicopter's animation frames), and the two character
sets at <code>$5000</code> (HUD) and <code>$5800</code> (playfield).</p>
<p>The current level is held as a <strong>decompressed map</strong> from <code>$0503</code> — 40 rows
of one page each — beside a soft <strong>scanner bitmap</strong> that backs the radar display, and
small per-object coordinate and state tables for the char-based actors (tanks, prisoners, mines). The
loaded game file itself occupies <code>$7000–$B8FF</code>: the two level maps and their RLE-packed
scanner bitmaps, the HUD screen image, the packed sprite shapes, then the bulk of the code and its
data tables, and finally the raw character-set data. The stage-2 loader and loading screen are left
as dead remnants higher in memory, never referenced again.</p>
`,
    graphics: `
<div class="info-eyebrow">Fort Apocalypse · Graphics</div>
<p>Fort Apocalypse is a character-mapped game: the playfield is built from an 8&times;8 tile set, the
moving actors are a mix of hardware sprites and animated characters, and the levels are stored as
compressed grids of screen codes. None of the data is encrypted — the only transformations applied
to it are a simple run-length scheme, the sprite packing, and a <code>$7F</code> mask on map bytes.</p>

<h2>Compression</h2>
<p>A single decompressor at <code>$8CDB</code> serves all level data. It reads a byte; if that value
appears in the active <strong>run-table</strong>, the following byte is a repeat count (with
<code>0</code> meaning 256) and the value is emitted that many times; otherwise the byte is a single
literal. Two run-tables pick which values are eligible to repeat — one for terrain, a smaller one
(<code>$00 $55 $AA $FF</code>) for the scanner bitmap — so there are no escape codes at all. Every
decompressed byte is masked with <code>AND #$7F</code>, which keeps all map codes below
<code>$80</code>.</p>

<h2>Character sets</h2>
<p>Both character sets are built at init from uncompressed data, copied in overlapping 256-byte
strips. They are swapped mid-frame by the raster handlers, so the HUD and the playfield draw from
different sets.</p>
<h3>HUD set ($5000)</h3>
<p>Selected by <code>$D018 = $14</code> for screen rows 0–6. It holds the score font and the HUD
furniture. Its high characters are left as <strong>soft characters</strong> into which the radar
window is rendered at runtime.</p>
<h3>Playfield set ($5800)</h3>
<p>Selected by <code>$D018 = $16</code> for rows 7–24. It holds the terrain glyphs — 8&times;8
multicolor dither patterns, including the mountain-slope, flat-dither and solid-block tiles. The low
characters <code>$00–$20</code> are reserved as <strong>soft characters animated in place</strong> by
the playfield interrupt: the energy barriers cycle between a stored pattern and blank on a timer; the
laser-grid segments each flip on or off independently and are re-rolled periodically; a four-character
group lights one member per phase to rotate; the explosion character and the fort core are masked
against the SID noise register (<code>$D41B</code>) every frame for a live flicker; the reactor-gate
walls pick one of two solid forms per life; and the missile-exhaust rows are noise-flickered each
frame. The same alphabet glyphs that form the double-width HUD font also serve as object graphics —
distinct glyph ranges are the prisoners, the self-propelled mines, and the tanks and their missiles.</p>

<h2>Sprites</h2>
<p>Fourteen sprite shapes are stored in a <strong>packed column format</strong>: 36 bytes per shape,
arranged as two 18-byte pixel columns (the left column's rows, then the right column's), located by a
pointer table. Init expands each shape into a 64-byte VIC sprite block, laying out <code>[left][right][pad]</code>
per row. The sprites are hi-res — no sprite multicolor — and the player and enemy sprites are
horizontally expanded.</p>
<p>Both helicopters, player and enemy, draw from <strong>one shared animation table</strong> of 18
entries indexed by bank/tilt: seven banking poses &times; two rotor frames, with the level-flight pose
covering three tilt steps. The player toggles its rotor frame every frame; the enemy every fourth
frame. The two bullet sprites are built at runtime from a nine-byte dot pattern — one block carries
the pattern twice for angled shots, the other once for straight-down shots.</p>

<h2>The level maps</h2>
<p>Each level's terrain is decompressed from a per-level source into a buffer at <code>$0503</code> —
one 256-byte page per map row, 40 rows. The map bytes <strong>are screen character codes directly</strong>,
with no tile-index indirection. Two placeholder codes are resolved after decompression: one is replaced
by a random pick from three cave-rock glyphs and another by a different trio, driven by the SID noise,
which gives the cave rock its mottled texture. The two levels are <em>Vaults of Draconis</em> (the
surface, with fuel depots and the landing pad) and <em>Crystalline Caves</em> (the Kralthan fortress,
with its central shaft and a large field of destructible rock).</p>
<h3>A cylindrical world</h3>
<p>The 256-byte rows are wider than the visible playfield. Columns 0–214 hold the 215 columns of level
content, columns 215–254 are padding, and <strong>column 255 is a copy of column 0</strong>. The world
is a horizontal cylinder: the camera column wraps around, and at the wrap point the right edge of the
screen displays that stored copy of the leftmost column, so the world's left edge meets its right edge
without a seam.</p>
<h3>Scrolling</h3>
<p>When the camera advances a full character — or every 8 frames regardless, so that map-embedded
objects keep animating — the engine rewrites the source operands of an unrolled copy loop and
block-copies <strong>16 rows &times; 40 columns</strong> from the map buffer straight to the screen.
Sub-character movement between copies is done with hardware fine-scroll. Because moving objects write
themselves <em>into the map buffer</em>, this periodic re-copy doubles as their on-screen update.</p>

<h2>The scanner</h2>
<p>The radar is backed by a second compressed stream that decompresses per level into a 1600-byte soft
bitmap — the whole map as a 320&times;40-pixel image (40 chars &times; 5 rows). The HUD rows are a
prebuilt screen image whose scanner window is made of soft characters; each frame a 12&times;3-character
window of the bitmap, following the camera, is copied into those characters' definitions. Blips are
XOR-plotted through a pixel-pair mask table — the player every frame, the enemy helicopter and the tank
bases blinking.</p>

<h2>The HUD</h2>
<p>The status display shows the score (six BCD digits), a bonus that counts down during play and is set
to 9999 when the fort is destroyed, the fuel gauge (four BCD digits), the "MEN TO RESCUE" count, and a
message row for flashing texts such as "LOW ON FUEL". The digits are drawn with leading-zero blanking.</p>
`,
    gameplay: `
<div class="info-eyebrow">Fort Apocalypse · Gameplay</div>
<p>Fort Apocalypse is a rescue-and-destroy game: you pilot the Rocket Copter through a surface and a
fortress of caverns, lift out trapped men and blow the enemy's reactor core, against tanks, mines,
homing missiles and a hunting enemy helicopter. Almost every interaction in the game follows from one
unusual rule about what counts as solid.</p>

<h2>The collision model</h2>
<ul>
  <li><strong>Solidity is defined by pixels, not tables.</strong> The core test takes the character
  drawn under an actor and scans its eight charset bytes; any non-zero byte is a hit. So blanking a
  character's definition makes every cell drawn with it non-solid <em>at once</em> — the basis for all
  the dynamic walls and barriers below.</li>
  <li><strong>Character-based actors carry their own collision.</strong> Tanks, mines, missiles and
  prisoners draw themselves into the map buffer (saving the background underneath) and react to the
  character codes they find around them.</li>
  <li><strong>Hardware sprites use the VIC collision latches</strong>, read once per frame —
  sprite-to-sprite and sprite-to-background.</li>
  <li><strong>Bullets bridge the two worlds.</strong> They fly as sprites but stamp an explosion
  character (<code>$20</code>) into the map on impact, and the character actors die from touching it.</li>
</ul>

<h2>The player — Rocket Copter</h2>
<p>Left and right build a <strong>bank</strong> that steers the copter, aims its gun and indexes the
sprite shape so it visibly tilts; up and down move it directly; and gravity pulls it down at a rate set
by the gravity option. The camera keeps the copter within a horizontal band and scrolls the cylindrical
world beneath it. (The title attract mode flies the copter by replaying a recorded joystick sequence.)</p>
<p>Contact with terrain is fatal <em>unless</em> the cell is a legal landing surface — the landing pad,
a fuel depot, the walkway floor, or a prisoner — in which case the copter bounces gently and the spot
becomes the <strong>respawn checkpoint</strong>. Setting down on a fuel depot refuels, the depot draining
visibly as it does. Fuel falls slowly in flight; at zero the engine sputters and "LOW ON FUEL" flashes.
A crash — from enemy or enemy-bullet contact, or a hard landing on an empty tank — sends the copter into
a flashing fall and costs a life; running out of lives ends the game. Brief grace timers protect the
moments just after spawning or teleporting.</p>

<h2>Bullets</h2>
<p>The gun fires from the nose along the current bank angle — from full-left, through level (which fires
<em>straight down</em>), to full-right — using the same bank-to-trajectory mapping as the enemy's gun.
Two impacts are special: the reactor core on level 1 triggers the <strong>fort-destruction sequence</strong>
(an expanding explosion, sixteen colour flashes, a 9999 bonus), and destructible rock is permanently
cleared. Every other hit stamps the explosion character into the cell, and what follows depends on the
victim. Against plain terrain the explosion lingers a few frames, then the original character is restored.
Against an object — a mine, tank, missile or prisoner — the bullet is freed at once and the object's own
engine finds the explosion in its cell, dies, and restores its background. So a direct hit kills any of
them; the sole exception is the enemy helicopter, a sprite that dies through the collision latch instead.
A consequence worth noting: prisoners can be shot, by you or by the enemy.</p>

<h2>The enemy helicopter</h2>
<p>Only one is active at a time. After a delay it spawns at a random patrol point — but never within
roughly 34 columns or 8 rows of the player, so it cannot materialise on top of you. It then hunts by
<strong>pure per-axis pursuit</strong>: each tick it steps one cell toward the player horizontally and
then vertically, with no pathfinding, testing the cells ahead so it only advances into clear corridor.
It banks into its motion — which in turn aims its shots — and fires periodically while on screen. It
cannot chase you across the cylinder's wrap. Off-camera it keeps hunting in map coordinates, with only
its sprite and gun going live once it is back in view, and a watchdog quietly resets it to a fresh patrol
point if it spends too long stuck off-screen while you are underground. Its only exits are death — flying
into terrain, or being hit by a player bullet — after which it explodes and waits to respawn. Its
climbing is notably erratic, incidentally: an easter-egg signature left in the binary overwrote one
opcode in its upward-probe routine, so its ceiling checks read a garbage column and it often stalls or
clips going up.</p>

<h2>Tanks, missiles and mines</h2>
<p>These are the character-based enemies. <strong>Tanks</strong> — six per level — are three body cells
plus a turret that always aims at the player; they patrol horizontally, reverse at obstacles, and
respawn at fixed home positions once cleared. Each tank can launch one <strong>homing missile</strong>
when the player passes within range above it: the missile flies in its facing direction, steering toward
the player's row, and falls once its fuel runs out, the player slips behind it, or it leaves its column
range — detonating on anything solid. <strong>Self-Propelled Mines</strong> (the manual's name for the
small drones) patrol the corridors in numbers set by difficulty; they spawn at random empty cells, fly
horizontally and reverse at obstacles, and do not respawn once destroyed until the next level. All three
die the same way — an explosion character or a missile in their cell — and because a missile's own
character kills the mines, tanks and prisoners it passes through, missiles can be lured into the other
enemies.</p>

<h2>Prisoners — "men to rescue"</h2>
<p>Eight per level, placed wherever the level builder finds a floor cell with rock directly above. Each
runs back and forth along its walkway. Flying into one within a few cells rescues him: he boards, the
rescued count rises, and the on-screen tally is reprinted. He can also be killed — by shooting away the
floor beneath him, or by an explosion or missile — so a stray shot, yours or the enemy's, can kill the
very men you need. Either way he leaves the "men to rescue" count, and <strong>both level exits stay
locked until that count reaches zero</strong>.</p>

<h2>The dynamic fortress</h2>
<p>None of the fortress's walls, gates and hazards use object slots. The map cells never change; their
character glyphs are <strong>redefined at runtime</strong>, and because solidity is pixel-based,
redefining one glyph opens or closes every cell drawn with it simultaneously. This drives:</p>
<ul>
  <li><strong>Reactor gate walls</strong> — two gates on level 1; at each life one is filled solid and
  the other left passable, chosen at random, so the safe route changes every life. Destroying the core
  opens both for the escape.</li>
  <li><strong>Sweeping walls</strong> — a band of four glyphs of which exactly one is solid at a time,
  advancing in phase so a wall section appears to march along the corridor. Its direction is reversed by
  every shot you fire, anywhere on the map.</li>
  <li><strong>Laser grids</strong> — four glyphs re-rolled every couple of seconds, each independently
  lit or dark at even odds; a lit segment is lethal, a dark one open air. There is no pattern to learn —
  passage is a gamble.</li>
  <li><strong>Energy barriers</strong> — two interleaved groups that are blank except for a brief lethal
  flash each cycle, the two groups flashing in alternation. On level 0 they form diagonal "scissor gates"
  across the cavern passages; on level 1 they are rails and shaft columns. Destroying the fort forces
  them permanently blank.</li>
</ul>
<p>The barriers double as the level-0 transport system. Flying into a lit barrier on level 0 from beneath
a scissor gate does not kill — it <strong>teleports</strong> the copter to one of four random cavern drop
points, each beside another gate, with a grace flag so the arrival cannot crash. On level 1 a barrier
always kills; there the hazard of the gates is the rock around the funnel, not the barrier itself. (Some
walls also carry a purely cosmetic shimmer that never affects collision.)</p>

<h2>Difficulty</h2>
<p>Three options on the title screen tune a run: <strong>Gravity Skill</strong> (how fast the copter
sinks), <strong>Pilot Skill</strong> (the speed of the enemy helicopter, tanks, missiles, barriers and
sweeping walls, plus the number of active mines — 13, 26 or 39), and <strong>Robo Pilots</strong> (three,
five or seven lives).</p>

<h2>Progression and rank</h2>
<p>Two playfields loop with rising difficulty. Clear and rescue the surface — <em>Vaults of Draconis</em>
— then land on the bottom-centre pad and sink through the floor into the fortress, <em>Crystalline
Caves</em>; there, rescue the men and shoot the reactor core, then fly out the top opening. The third pass
is the surface again, harder, and landing back on the base deck ends the mission. The debrief tallies
rescued men and bonuses into a <strong>rank from 0 to 15</strong>, shown as one of four bird names —
Sparrow, Condor, Hawk, Eagle — and a class number from 4 up to 1, with Eagle Class 1 at the very top.</p>
`,
  },
  turrican: {},
  marble: {},
  stuntcar: {},
  elite: {},
};

// HTML for a game/tab, or null if nothing has been written for it yet.
export function infoHtml(gameId, tabId) {
  const game = INFO_CONTENT[gameId];
  return (game && game[tabId]) || null;
}
