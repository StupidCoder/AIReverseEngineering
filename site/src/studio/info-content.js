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
  sonic: {
    loader: `
<div class="info-eyebrow">Sonic the Hedgehog · Image &amp; Loader</div>
<p>Sonic runs from a Game Gear cartridge — the simplest image in this collection: a verbatim copy of
the mask-ROM chip, with <strong>no container, filesystem or loader</strong>. The only structure the
console imposes is a memory map (the ROM is larger than the Z80 can address) and a small Sega header,
so there is nothing to unpack — boot is a short reset routine that brings the hardware up and hands
off to the game.</p>

<h2>The cartridge and its memory map</h2>
<p>The ROM is 256&nbsp;KB. The Z80's address bus is 16-bit, so it can only see 64&nbsp;KB at once —
a quarter of the cartridge. The ROM is divided into sixteen 16&nbsp;KB <strong>banks</strong>, and the
standard Sega mapper pages a chosen bank into one of three 16&nbsp;KB <strong>slots</strong> in the low
48&nbsp;KB; the top 16&nbsp;KB is work RAM. Four mapper registers at the very top of the address space
select which bank shows in each slot — and they physically <em>are</em> the top four bytes of RAM
(mirrored there), so a write both stores the byte and reprograms the mapper. At reset the slots default
to banks 0/1/2, which is why the engine and boot code sit in the first 48&nbsp;KB. One subtlety: the
first 1&nbsp;KB is <strong>hard-wired to bank 0</strong> and never paged, so the CPU vectors and the
mapper-setup code stay reachable however slot 0 is paged. The graphics and sound hardware are not in the
memory map at all — the Z80 reaches the VDP and PSG through I/O ports.</p>

<h2>The header and vectors</h2>
<p>Sega stamps a 16-byte <code>"TMR SEGA"</code> header near the front: a magic string, a product code
and version, and a region / ROM-size nibble (here Game Gear export, 256&nbsp;KB). On the Master System
that field gates a BIOS check; the Game Gear has no such gate, so it is informational. The Z80's fixed
entry points all sit at the bottom of the always-present bank 0 — the reset address, eight one-byte
<code>RST</code> call targets, and the interrupt vectors. Sonic routes its hottest common subroutines
through the <code>RST</code> slots (each is a one-byte call), and the maskable-interrupt vector jumps to
the per-frame VDP handler.</p>

<h2>Reset and cold start</h2>
<p>The reset code is the textbook Master System / Game Gear opening: disable interrupts, select interrupt
mode 1, busy-wait on the VDP until the raster reaches a known line, then jump to the real cold start. The
cold start re-asserts the default bank layout, clears the 8&nbsp;KB of work RAM with the classic
overlapping <code>LDIR</code> fill (stopping 16 bytes short so it does not clobber the mapper-register
mirror) and parks the stack at the top, brings the VDP up in <strong>Mode 4</strong> from a register table
(keeping a RAM shadow the interrupt handler reads back), and <strong>hides all 64 sprites</strong> by
setting their Y off-screen before the display comes on. It then calls a setup routine in another bank
through the banked-call gateway and hands off to the main entry.</p>

<h2>Cross-bank calls</h2>
<p>Because most of the game lives outside the 64&nbsp;KB the CPU can see at once, the engine reaches it
through <strong>banked-call thunks</strong> that are the <code>RST $18/$20/$28</code> vectors: each pages
a fixed bank into a slot, calls a fixed entry in it, then restores the previous bank — read back from a
RAM shadow that tracks the current slot banks so the calls can nest. <code>RST $20</code> lands in bank 3,
which holds a dispatcher the engine reaches through these one-byte calls. The main entry enables
interrupts, sets the bank shadows to the running configuration, runs the subsystem initialisers, and sets
the top-level game-mode variable — and from there control is in the main loop.</p>
`,
    engine: `
<div class="info-eyebrow">Sonic the Hedgehog · Game Engine</div>
<p>After boot, Sonic runs an attract sequence over a small <strong>state machine</strong>: a scene system
that loads the logo, the title, the world map and the levels in turn, driven by a one-byte scene counter
and a function-table dispatcher that reaches across the ROM's banks.</p>

<h2>The attract loop and scene state machine</h2>
<p>The main entry loads the SEGA logo and enters an <strong>attract loop</strong> driven by a one-byte
scene counter: each pass loads a screen, fades it in, runs it, and advances; when the counter passes the
last scene it restarts from the logo. Each scene is described by a roughly 40-byte <strong>descriptor</strong>
in a table (18 entries), which the dispatcher copies into RAM to run and maps to a screen type — the
early scenes are the title background, the later ones the world map. It only reloads the background when
the type changes, so the title stays up across its scenes while the foreground animates, and the map
likewise persists across its scenes.</p>

<h2>Title and Start</h2>
<p>While the title is up, the scene scripts animate Sonic (the finger-tapping pose) and blink
<code>PRESS START BUTTON</code>. The wait for Start is not a poll loop — it is folded into the attract
loop through two flags: the controller is read every frame, and when Start is pressed a handler raises a
<strong>launch flag</strong> and writes a target scene. On the next pass the loop sees the flag, skips the
idle wait, and uses the target scene instead of the counter — so Start jumps the sequence out of the demo
and into the post-title flow rather than letting it free-run. The same Start bit is checked during the
logo, which is why the logo and title are skippable.</p>

<h2>The world map</h2>
<p>There are <strong>two</strong> world-map screens — a different tile set and a different stored map for
each — and the engine picks between them by the level you are heading into. A countdown byte, decremented
after each level, selects the wide island for early levels and zooms in on the mountain-top goal for late
ones. It is the same island both times: the castle on the peak of the wide shot is the city that fills the
zoomed shot. On either map the route and the zone name are not in the stored map — they are a per-scene
overlay drawn by the string blitter, and the blink is that overlay repainted on a timer.</p>

<h2>Reaching a level</h2>
<p>Pressing Start does not "return into the game" — the attract loop never exits. Instead Start selects a
target scene whose script <em>itself</em> loads the level and jumps into the gameplay engine, which runs
from banked code (the gameplay engine and the level variables live in bank 1, reached through the bank-3
dispatcher). Once in play, a per-frame <strong>scroll / camera update</strong> recomputes the tile-stream
source from the camera position each frame, feeding the streamer that draws the level a column at a time.</p>
`,
    graphics: `
<div class="info-eyebrow">Sonic the Hedgehog · Graphics</div>
<p>Game Gear graphics are two layers: the fixed <strong>VDP hardware formats</strong> the data ends up in,
and the game-specific <strong>compression</strong> it is stored under in the ROM. Almost all of Sonic's
art is compressed, and levels are built from a small alphabet of reusable blocks.</p>

<h2>VDP formats</h2>
<p>These are standard Mode 4 formats. A <strong>tile</strong> is 8&times;8 pixels in four bitplanes (16
colours), 32 bytes, stored row-interleaved. The <strong>palette</strong> is 32 entries of 12-bit colour in
BGR order — 16 background, 16 sprite. The <strong>name table</strong> is the 32&times;28 background map in
VRAM; each cell is a two-byte word giving a 9-bit tile number plus per-cell flip, palette-select and
priority bits.</p>

<h2>Compression</h2>
<p>A single codec compresses almost all of Sonic's art — a <strong>four-byte-unit LZ</strong> whose unit is
one tile row (four bitplane bytes), so a repeated row (a blank row, a flat fill) costs a single bit. A
compressed block is addressed as a <code>(bank, address)</code> pair, and its prologue normalises the
address and maps two consecutive banks into the slots, so a block can span banks. The block holds a control
bitmap (one bit per output unit), a literal stream of the distinct four-byte rows in first-appearance order,
and a match-info stream of back-reference offsets; decoding walks the units, emitting either the next
literal row or a back-reference to an earlier one.</p>

<h2>The opening screens</h2>
<p>The two screens before the menu reach the same VRAM by opposite routes. The <strong>SEGA logo</strong>'s
tile map is computed in code — there is no stored map: a loop lays down a plain identity grid one vertical
column at a time, left to right, so the logo wipes in column by column behind <strong>Sonic</strong> as he
jumps across the screen. He leaps from the left, the logo drawing in behind him as he goes, then jumps back
to the left and comes to rest beside the finished logo — his parabolic arc comes from a symmetric offset
table the frame interrupt flushes to the sprite list each vblank. The <strong>title</strong> takes the other
approach, loading its name table wholesale from a stored, compressed map (a tiny RLE name-table loader runs
twice, composing a priority base layer drawn in front of sprites and a plain overlay).
<code>PRESS START BUTTON</code> is painted on afterwards by a string blitter and blinks — repainted every
time it toggles.</p>

<h2>Level maps</h2>
<p>A level is far wider than the screen, scrolls, and is built from reusable <strong>blocks</strong> rather
than per-cell tile numbers. Sonic stores it as three nested layers plus a palette:</p>
<ul>
  <li>a compressed <strong>block-index map</strong> — an RLE list of block indices that decompresses to a
  fixed 4096-byte grid in a RAM window the engine reads as it scrolls. The scenery — hills, palms, flowers,
  even the ring graphics — is part of this block map; what it does <em>not</em> contain is the object layer.</li>
  <li>a <strong>block → tiles table</strong> — each block index expands to a 4&times;4 grid of 8&times;8
  tiles (a 32&times;32-pixel chunk), 16 bytes per block. A column expander reads one map column, looks up
  each block's tiles, and uploads a column to VRAM as the camera crosses each 8-pixel boundary.</li>
  <li>the <strong>tile graphics</strong> — 256 background tiles, compressed with the same codec, plus a
  palette loaded by index through a table.</li>
</ul>
<p>Each act is driven by a 37-byte <strong>descriptor</strong> — the zone, scroll bounds, spawn block, the
offsets of the map, block table and tile set, the object table, the music id and flags. Because that table
is static, every act of a zone follows from it: the three acts of a zone share their tile set, block table
and palette, differing only in the block-index map and the width.</p>

<h2>A map that isn't always wide</h2>
<p>The decompressed map is always 4096 bytes but not always 16&times;256: the descriptor carries a
<strong>stride</strong> that sets the column count, so the grid is <code>(4096 / stride)</code> rows tall.
Most acts are wide (stride 256 → 16&times;256), but a small stride is a tall, narrow level — Jungle Act 2
is a 256-row &times; 16-column vertical waterfall climb. The grid is always 4096 cells; how they are laid
out is data-driven per act.</p>

<h2>Animated tiles</h2>
<p>Some graphics animate by swapping tile data in place. The <strong>rings</strong> (a 16&times;16 graphic
is four 8&times;8 tiles) spin through about six frames, copied from a fixed source for every zone; the Green
Hills <strong>flowers</strong> are a two-frame toggle gated on the zone. Both tile sets are empty in the base
set — a per-frame update copies a fresh frame into them each cycle. There is no generic animation table; each
animation is hardcoded.</p>

<h2>Sprites</h2>
<p>Objects — Sonic, the enemies, items — are drawn as <strong>hardware sprites</strong> (8&times;16 mode),
not from the tilemap. A per-zone sprite tile set is decompressed into VRAM, and the sprite-attribute table
lists each active sprite's position and tile; the per-object draw builds a display list in RAM that is
flushed to the attribute table each vblank. Sonic's animation is <strong>data-driven and streamed</strong>:
his pose indexes an animation table to a frame sequence (a byte stream with loop/jump controls), and when the
frame changes its 16 tiles are streamed into the sprite VRAM — so only the live frame (and its neighbours) is
resident, not a full sheet.</p>
`,
    gameplay: `
<div class="info-eyebrow">Sonic the Hedgehog · Gameplay</div>
<p>Sonic himself is just object type <code>$00</code>, and everything interactive in a level — the player,
the enemies, the platforms, the springs — runs through one <strong>object system</strong> and one master
dispatch.</p>

<h2>The object system</h2>
<p>Each act's descriptor points to an <strong>object table</strong> — a count followed by three-byte entries
(type, block X, block Y) — expanded at load into an array of fixed-size records; record 0 is Sonic, placed
from the spawn pointer. Every live object is processed once per frame by a <strong>master dispatch</strong>:
a word-pointer table indexed by the type byte selects the object's behaviour handler, which runs and then
falls through to common "apply velocity to position" code. The type also indexes a bounding-box table for
the object's size and a descriptor for its sprite class. So a single type byte ties an object to its
behaviour, its size and its sprite.</p>

<h2>Sonic's movement</h2>
<p>Sonic's handler is the largest in the game, turning the controller into motion. The pad is sampled once
per frame and is active-low (a pressed button reads <code>0</code>). Each frame the handler picks a
<strong>physics parameter set</strong> — acceleration, friction and a top-speed cap — by Sonic's state and
whether he is on the ground, then accelerates him while a direction is held and decelerates by friction when
nothing is; the resulting speed integrates into his world position. Pressing <strong>Down</strong> sets the
rolling/ball flag (and, only if he is moving, plays the roll sound — standing still and pressing Down is a
crouch); rolling selects low-acceleration, <strong>low-friction</strong> constants, so a roll holds its
momentum — you steer only a little and mostly coast, carrying farther than Sonic would slide to a stop on
foot. <strong>Jumping</strong> gives variable height: a hold timer is seeded, and while the button
stays held and the timer counts down the upward impulse keeps being added; once released or expired, gravity
is applied each frame instead.</p>

<h2>Terrain and rings</h2>
<p>Collision reads the same block map the level is built from, through a shared sampler that returns the
block at a world point (it accounts for the level's stride, so a sampled point lands on the right block
whatever the level's shape). The plain <strong>solid floor</strong> is a generic per-object routine: it
samples the block at the feet, reads a collision shape from a per-zone attribute table (shape 0 means
non-solid — Sonic passes through), and uses a per-column <strong>height profile</strong> so he lands on
slopes at the right height and angle. Special blocks are a separate interaction layer — springs, spikes and
conveyors mapped through a per-zone table to handlers (a vertical spring sets an upward velocity, a
horizontal spring an X velocity, spikes hurt, conveyors add a steady drift), each gated on which 16-pixel
sub-cell of the block Sonic is in. <strong>Rings</strong> are baked into the block map, not objects: certain
block indices mark a left/right pair of rings, and the low bits of the index say which halves still hold one.
Collecting one is a single byte-write back into the map (the graphic downgrades two → one → none), a sparkle,
and a bump to the BCD ring count — which grants an extra life every time it passes 100.</p>

<h2>Enemies and platforms</h2>
<p>Each object type is a self-contained behaviour on its record. The <strong>platforms</strong> carry Sonic
explicitly — a horizontal platform glides one pixel per frame 160&nbsp;px out and back; a swinging platform
reads an arc table to trace a semicircle below its pivot — and when Sonic stands on one, the handler glues
him on and adds the platform's per-frame motion directly to his position. The <strong>enemies</strong> share
a small script engine — a counter into a state table, one entry per state for a fixed run of frames: the crab
walks, stops and fires a projectile to each side; the beetle marches faster with no attack; the fish hides
underwater and leaps straight up; the porcupine is a slow spiky walker whose spikes are the threat. The
<strong>seesaw</strong> is simulated rather than scripted: a tilt angle with a weight that falls under
gravity, and Sonic's landing impact is transferred into the tilt — so a harder landing flings whatever is on
the other end higher, with no fixed launch height. Each <strong>boss</strong> sets itself up as a
self-contained set-piece, decompressing its own graphics and running a small bytecode script; the world-1
boss takes eight hits, scored only when Sonic touches it while rolling or jumping.</p>

<h2>Checkpoints and the camera</h2>
<p>Some "objects" are not enemies at all. A <strong>camera / scroll-lock</strong> writes the camera position
from its own position each frame, pinning or limiting how far the view scrolls. A <strong>checkpoint</strong>,
on contact, writes its own block position into a per-act respawn table — so a death returns Sonic to the
checkpoint, one block above it.</p>

<h2>Underwater</h2>
<p>Labyrinth is the only zone with water, and its acts are half-flooded, split by a horizontal water line.
The <strong>water surface is itself an object</strong> — the first one placed in each Labyrinth act, its
block-Y the water level — and each frame it writes the current surface Y (with a sine bob, so the line
ripples) and derives the scanline where the split falls. Crossing the line changes three things at once, all
driven by one underwater flag: the <strong>palette</strong> swaps via a mid-frame raster split (a line
interrupt rewrites the 16 background colours to a static underwater set below the line, restored at vblank
above it); the <strong>physics</strong> load a slower constant set (roughly quarter acceleration and gravity
and a weaker jump, so Sonic drifts down slowly); and an <strong>air timer</strong> starts that, past about 13
seconds, triggers the drowning countdown. Drowning is often assumed to be a feature only of the 16-bit
Mega Drive games, but this 8-bit version has it too.</p>

<h2>Bonus stages</h2>
<p>Clearing Act 1 or 2 of a zone with 50 or more rings sends you to a <strong>bonus stage</strong> instead of
the next act. The goal sign checks the ring count and sets a bonus flag; the next scene load runs the bonus
path, swapping in a separate bonus-stage cursor that advances each visit. The bonus stages are eight more
descriptors in the same scene table, in a seventh "zone", and they all share one tilemap, tile set, block
table and palette — what differs per round is the spawn, the camera bounds and the object layout: the rings,
a collectible, and bumpers that reverse Sonic's velocity on contact.</p>

<h2>The hidden Scrap Brain maze</h2>
<p>An invisible <strong>teleporter</strong> object converts its own position to a key, looks it up in a small
table, and launches the matched destination scene — the very mechanism Start uses on the title. Because each
teleporter's destination is hardwired to where it sits, the placements expose something the act list hides:
<strong>Scrap Brain Act 2 is not one level but seven</strong> — the listed act plus six sub-scenes reachable
only through teleporters, wired into a maze with deliberate loops so a wrong choice sends you in circles. A
second teleporter, in Sky Base, warps to a hidden fortress room holding a goal sign and a Chaos Emerald,
outside the normal act flow entirely.</p>
`,
    music: `
<div class="info-eyebrow">Sonic the Hedgehog · Music</div>
<p>The Game Gear's audio is the <strong>SN76489 PSG</strong> — three square-wave tone channels and one noise
channel, programmed through a single write port. There is no FM on the Game Gear; the music is square waves,
reprogrammed once per video frame.</p>

<h2>The sound driver</h2>
<p>A sound is started by a one-byte <code>RST</code> call with a sound id; the gateway pages in the sound
bank and indexes a song-pointer table. A song begins with a header of <strong>five relative channel
offsets</strong> — three square tones, the noise channel, and a sound-effect channel that stays idle during
music — which the loader resolves to absolute pointers in RAM. A per-frame <strong>sequencer</strong>
advances each channel and renders it to the PSG.</p>

<h2>The channel format</h2>
<p>Each channel is a byte stream with a duration counter decremented each frame; when it runs out the decoder
reads the next bytes:</p>
<ul>
  <li>a <strong>note</strong> is two bytes — an octave/note value (the pitch is a base period from a
  one-octave table, shifted down by the octave) plus a duration;</li>
  <li>a <strong>rest</strong> silences the channel for a duration;</li>
  <li>a <strong>voice</strong> command picks an instrument (a noise mode plus an ADSR envelope) — and on the
  noise channel the "notes" are voice commands, each a drum hit, which is the zone's percussion;</li>
  <li>a <strong>command</strong> byte (tempo, volume, instrument envelope, vibrato, detune, block-repeat with
  a nested loop stack, a loop-point mark and an end/loop) carries no time of its own.</li>
</ul>
<p>The loop is <strong>in the data</strong>: a loop-point mark and a loop command bracket the repeating
section, which is why each track loops seamlessly. Per frame the render forms the pitch as base + detune +
vibrato, shifted by the octave, and scales the volume with the ADSR envelope before writing the PSG.</p>

<h2>Which track plays</h2>
<p>The track for an act is its descriptor's <strong>music id</strong>, indexing the song table. The ids
resolve to about fifteen distinct tunes — the six zone themes (Sky Base acts 1 and 3 reuse Scrap Brain's),
the special-stage theme, the world map, act-clear, a shared boss theme, and assorted jingles — so, for
example, Sky Base Act 1 correctly plays the Scrap Brain music.</p>
`,
  },
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
    music: `
<div class="info-eyebrow">Fort Apocalypse · Music</div>
<p>Fort Apocalypse has no separate in-game score. Its one piece of music is the
<strong>loading-screen tune</strong> — the three-voice SID piece that plays under the U.S. Gold
loading screen while the game streams in from tape. That tune is more than music: it is a small
bytecode program whose commands also patch the machine as it plays, including writing the jump that
actually starts the game. It is described in full under <strong>Image &amp; Loader</strong>.</p>
<p>Once play begins, the SID is given over entirely to <strong>sound effects</strong>. The two raster
handlers drive the audio each frame — the copter's engine, gunfire, explosions, and the warning tones —
rather than a continuous melody, so the gameplay itself runs without a backing track.</p>
`,
  },
  turrican: {
    loader: `
<div class="info-eyebrow">Turrican · Image &amp; Loader</div>
<p>Turrican ships on a single double-density Amiga floppy that carries <strong>no filesystem</strong> —
only a boot block the Kickstart ROM will run, and behind it the whole game laid out in a private format
the loader pulls off by absolute sector. The build preserved here is a one-disk cracked release by
Tristar &amp; Red Sector, whose loader decompresses the game from a crunched image on boot.</p>

<h2>The disk image</h2>
<p>An ADF is a flat dump of the floppy's 1760 logical blocks of 512 bytes — block <em>N</em> is simply
the bytes at offset <em>N</em>&times;512. The first four bytes are the <code>"DOS\\0"</code> boot
signature, enough for the ROM to accept the disk and run its boot code, and that is the only
AmigaDOS-conformant thing on it. There is no directory: the program, graphics and levels sit in a
private layout addressed by absolute byte offset through <code>trackdisk.device</code>, never through
files. The disk falls into three regions — the boot block (blocks 0–1), a first-stage loader in plain
68000 code (blocks 2–21), and from block 22 to the end the <strong>crunched main part</strong>: the
entire game — program, graphics and levels — stored compressed.</p>

<h2>The boot block</h2>
<p>The boot block is a complete sector loader. The ROM enters it with the boot device's I/O request
ready; it reads blocks 2–9 to <code>$30000</code> and runs them (the cracker's intro), then clears the
bitplane, copper and sprite DMA. It allocates a work buffer (the largest FAST-memory chunk, or the chip
region on a 512&nbsp;KB machine), issues the main read — about 143&nbsp;KB of the crunched main part to
<code>$50000</code> — and stops the drive. It adapts to the CPU (on a 68010 or better it installs a
<code>TRAP</code> handler running a <code>MOVEC</code>, so the rest of the loader can treat the machine
as a bare 68000), seizes the machine (supervisor mode, interrupts off, stack at <code>$80000</code>),
copies a 512-byte tail routine to <code>$7F800</code> and jumps to it. The boot block never touches
<code>dos.library</code>; it drives the hardware directly.</p>

<h2>The intro and trainer</h2>
<p>Being a cracked release, blocks 2–9 are the cracker's intro: it opens <code>graphics.library</code>
and the Topaz font, allocates a chip buffer for its bitplanes, and scrolls the group's greetings and a
prompt over a copper display. Pressing the joystick button after decrunching enables the
<strong>trainer</strong> — 99 lives — and the high-score save is redirected to track 0. The game itself
appears only once the main part is decrunched.</p>

<h2>Decrunching the main part</h2>
<p>The tail at <code>$7F800</code> is the bridge from loader to game: it calls the decruncher at the head
of the crunched blob, then enters the game. The crunched main part is not one packed stream but the
output of <strong>three compressors applied in series</strong>, so unpacking runs three decoders in turn
— a Huffman bit-reader, then an LZ77 copier, then an RLE expander — each relocating its intermediate
result to the top of memory and decoding it back down. Two of the three are byte-dispatched: they build a
256-entry jump table whose default handler copies a literal and whose few escape values trigger
match/run handlers, and the loop <strong>writes each control byte to the background-colour register</strong>
(<code>$DFF180</code>) as it runs — the flickering colour bars across the screen that a cracked game
shows while it "decrunches". The result is a
214,400-byte image at <code>$43880</code>, with the game entered partway into it at <code>$5F500</code>.
The tail then applies the trainer (overwriting two longwords with branches into the cheat code) and jumps
into the decrunched game.</p>
`,
    engine: `
<div class="info-eyebrow">Turrican · Game Engine</div>
<p>The decruncher hands control to a flat image at <code>$43880</code>. The first thing the game does is
split itself in two and bring up the hardware; from there it is a <strong>vertical-blank-driven loop</strong>
with a function-pointer state machine, pulling each world's code and data off the disk as it goes.</p>

<h2>Two segments</h2>
<p>The image does not run where it is loaded. On entry the game copies the <strong>resident engine</strong>
— roughly 112&nbsp;KB — down to low memory from <code>$10</code> onward, where <code>$10–$FF</code> is the
68000 exception vector table and <code>$100</code> is the engine's internal jump table. The rest of the
image — the setup code, the interrupt handler and data — runs in place. So the program is two segments:
the relocated resident engine (the bulk of the game) and the in-place setup/ISR.</p>

<h2>Bring-up</h2>
<p>Entry reads the fire buttons to pick the trainer and option settings, waits for a press and release,
and branches into <code>game_init</code> — the hardware bring-up. It enters supervisor mode, turns all
interrupts and DMA off, then unpacks and runs several sub-modules, installs the level-3 (vertical-blank)
interrupt vector, and enables the display. The vblank interrupt bumps a frame counter, cycles the
palette, and calls the per-frame game tick.</p>

<h2>The resident engine</h2>
<p><code>game_init</code>'s last act jumps into the relocated segment through its internal jump table;
slot 0 is <code>game_start</code>, which seizes the machine, runs the OS-interface module, initialises
the object table, the map grid and the display interrupt, and falls into the main loop. The engine
re-uses the <strong>same three-pass decoder</strong> at runtime to unpack graphics and level blocks,
alongside a PowerPacker decompressor and a floppy trackloader that streams level data off the disk during
play.</p>

<h2>The streamed modules</h2>
<p>The engine does not ship complete in the resident image; three more modules stream in at startup:</p>
<ul>
  <li>the <strong>music / sound driver</strong> (see Music);</li>
  <li>a <strong>loader-sound player</strong> that installs its own vblank handler and plays the
  disk-access sound during loading;</li>
  <li>a PowerPacker-compressed <strong>OS-interface module</strong> — the engine's bridge to the system:
  it opens <code>graphics.library</code> and <code>dos.library</code>, installs a <code>TRAP</code>
  handler and saves and replaces CPU vectors for the display and disk.</li>
</ul>

<h2>The game loop</h2>
<p><code>game_start</code> falls into level setup, which clears the playfield with the blitter, installs
three triple-buffered display buffers, primes the level state and runs a chain of subsystem inits, then
drops into the game loop. Two things define its shape:</p>
<ul>
  <li><strong>Mode dispatch.</strong> A single pointer holds the current game-mode handler, called once
  per frame. Swapping it switches state — title, play, and so on — without touching the surrounding
  pipeline: the classic function-pointer state machine. It is driven by a <strong>scene system</strong>:
  a scene id indexes a descriptor table, and the descriptor's handler fields become the primary and
  secondary per-frame handlers. The descriptors are not in the resident image — they are
  <strong>streamed off the disk per world</strong>, so the states and their code change with the level.</li>
  <li><strong>Frame sync.</strong> The loop raises a flag and spins until the vblank interrupt clears it,
  locking the pipeline to the vertical blank.</li>
</ul>
<p>Around the mode call sits the fixed pipeline: blitter copies that draw the playfield and object layers
from a draw list, plus a dozen further per-frame subsystems. The engine also carries its own copy of the
sound driver on resident state, so it runs the music and the sound effects as two independent player
instances.</p>
`,
    graphics: `
<div class="info-eyebrow">Turrican · Graphics</div>
<p>The engine and its modules are only the loader and runtime; the <strong>worlds themselves are streamed
off the floppy</strong> as the game runs. Each is a self-describing block of tiles, a palette, a map, a
collision layer and sprite graphics, decoded straight into memory.</p>

<h2>Worlds streamed off disk</h2>
<p>Loading a level reads a per-world entry from a table, pulls the packed block off the disk into a buffer
just past the resident image, and decompresses it with the same three-pass decoder used at boot. Each of
the five worlds decodes to a fixed block of about 260&nbsp;KB at a known address. A small
<strong>section directory</strong> at the head of the block points at the tile data, a collision layer,
the 16-colour palette, the sprite/object graphics, and a TFMX music slot — so a single decoded block
carries everything that world needs.</p>

<h2>Tiles and palette</h2>
<p>The palette is 16 big-endian 12-bit RGB words. Tiles are reached through an offset table — a list of
longword byte-offsets, with entry 0 equal to the table's own size, so the tile count and the start of
the pixels both fall out of it. Each tile is <strong>32&times;32 pixels in four bitplanes</strong>,
interleaved per row (512 bytes), drawn through the palette. World 0 has 209 tiles, world 1 has 215, and
so on — the cave and planet surface, the machine world, and the rest.</p>

<h2>The tile map</h2>
<p>Each world holds several <strong>scenes</strong>, one per sub-map, each described by a descriptor: a
pointer to the map data, its width and height in tiles, and the scene's per-frame handlers. The map is a
<strong>column-major array of one byte per cell</strong>; a value below the tile count is a tile index,
and a value at or above it is the same tile drawn <strong>horizontally flipped</strong>. World 0's three
scenes are 137&times;51, 153&times;51 and 115&times;51 tiles, laid out back to back, while other worlds
are shaped very differently — one world opens on a tall 12&times;269 vertical shaft.</p>

<h2>Collision</h2>
<p>Solidity is not a parallel grid; it is a <strong>per-tile-type shape</strong>. A collision section gives
16 bytes per tile — a 4&times;4 grid of 8&times;8-pixel-block solidity — so each 32&times;32 tile carries
sub-tile collision. The values are not merely solid or empty: passable, solid wall or ground, solid but
reacting to shots (a hit sparks and stops), breakable or trigger (contact spawns an effect and clears the
cell), and hazard (contact drains the player's energy). At scene load the playfield builder copies each
map tile's shape into a screen-sized collision buffer, and the player check reads one byte at the player's
position at 8-pixel granularity; flipped map cells mirror their columns. This is the layer the viewer's
<strong>Collision</strong> toggle overlays.</p>

<h2>Sprites — the BOB format</h2>
<p>Enemies and effects are <strong>blitter objects</strong> cookie-cut into the back buffer: a
four-bitplane bitmap and a one-plane mask blitted through the playfield's 16-colour palette, with plane 3
doubling as the mask so opaque pixels carry colours 8–15 and colour 0 is transparent. Each is described by
a 14-byte <strong>descriptor</strong> — bitmap pointer, mask pointer, modulo, a packed size word, and a
y-adjust and flag — and a flat array of these descriptors is the animation table: an object's draw routine
picks a frame group, then the current frame within it, then draws that descriptor. The <strong>player</strong>
is the exception — he is drawn by a dedicated routine as a multi-part composite (three body parts plus the
orbiting spinning weapon), indexed by his animation state.</p>
`,
    music: `
<div class="info-eyebrow">Turrican · Music</div>
<p>Turrican's score is Chris Hülsbeck's, in his own <strong>TFMX</strong> format — Turrican is the
canonical TFMX game. The music is driven by a dedicated sound overlay, and the engine carries
<strong>two copies of the same player</strong> so the music and the sound effects run independently.</p>

<h2>The sound driver</h2>
<p>The music and sound driver is a separate module streamed off the disk at startup, decoded with the same
three-pass decoder as everything else and loaded at a fixed address. Its body opens with a
<strong>branch dispatch table</strong> — its public API, which the engine calls at fixed entry points to
start playback, initialise the player from the song and sample pointers, and set the master volume and
channel mask. Its vertical-blank entry runs the player once per frame: it processes the voices, each with
a period LFO (vibrato), a pitch slide (portamento) and a volume envelope, writing the Amiga's audio period
and volume registers, while a silence call zeros all four channel volumes. The engine keeps a second,
byte-identical copy of this player on its own state, so the music and the effects play as two independent
instances.</p>

<h2>The TFMX module</h2>
<p>The score itself is a TFMX module. It is <em>not</em> played from the per-world scene block — that
block's "TFMX-SONG" slot is an empty stub — but from the sound overlay, which carries the in-game player
and two data pointers: the song data and about 50&nbsp;KB of raw signed 8-bit PCM, the instrument samples.
The song data is a set of tables — a song table of start, end and tempo per sub-song (three real ones), a
pattern pointer table, a macro pointer table, and a trackstep table that lays out the eight channels. A
<strong>pattern</strong> is a stream of note-plus-macro entries and commands; a <strong>macro</strong> is a
stream of instrument commands (set sample, volume, period, vibrato, portamento, envelope, DMA, wait, loop)
— effectively a small instrument VM. The player runs a song sequencer and trackstep processor feeding a
pattern reader and a per-voice update with the macro VM, driving Paula's four channels.</p>
`,
    gameplay: `
<div class="info-eyebrow">Turrican · Gameplay</div>
<p>Turrican is a run-and-gun platform shooter across five large, multi-scene worlds. Crucially, the worlds
differ in their <strong>enemies and backdrops, not their mechanics</strong> — every world runs on the same
engine, object system and sound interface, bringing only its own enemy roster and scene code. The parts
that are not self-describing data — the objects, where they spawn, and how they behave — are driven by
code.</p>

<h2>The object system</h2>
<p>Active enemies and effects are a <strong>doubly-linked list</strong> of 58-byte nodes drawn from a
39-node pool; spare nodes sit on a free list. A spawn pops a free node, fills its fields and links it into
the active list; a kill unlinks it back to the pool. Each node holds its position, its current frame and
frame table, an active flag and an AI-handler pointer. Every frame the engine walks the list once calling
each node's handler, then walks it again drawing each node — cookie-cutting its sprite through its frame
table at the current frame and position. So a spawn is simply a node whose frame table points at one of
the world's sprites.</p>

<h2>Enemy behaviour and per-world code</h2>
<p>Each world's scene block carries its own code in two parts. The <strong>scene handlers</strong> run the
animated parallax backdrop and trigger ambient sounds — they only call the resident sound API, never the
spawn routines; world 1's drives the waterfall, and worlds differ here only in which backdrop they animate.
The <strong>enemy-AI handlers</strong> — six to eighteen per world — are the enemy roster: each is a
complete behaviour on an object node, setting its sprite and health, animating a loop, applying damage and,
on death, freeing the node. The per-world differences are the enemies and the backdrop, not new mechanics.</p>

<h2>Enemy placement</h2>
<p>Which enemy is seeded where is read by a <strong>scroll-triggered spawner</strong>, called twice per
frame. It builds a spawn window from the camera — the visible screen plus a margin — and spawns any entry
inside it, which is why enemies appear <em>just</em> as the screen reaches them. The layout is a 2D bucket
grid, not a flat list: a per-camera-row offset table and the grid yield, for a given camera position, a
pointer into the entry data; each distinct pointer heads a run of 6-byte entries (type, x, y in 8-pixel
units) ending at a terminator. An entry's <strong>type</strong> selects its handler in two tiers — low
types index a resident handler table (engine-wide objects like the little rotating mine), higher types
index the scene's own handler table — and the handler installs the object's sprite. So the chain from a
placement entry to a drawn enemy is <strong>type → handler → sprite</strong>.</p>

<h2>Starting position</h2>
<p>Each scene also records its initial camera tile and the player's on-screen offset, so the player spawns
at camera-plus-offset — the scene's intended starting position, which is the point the viewer frames each
scene on.</p>

<h2>Weapons</h2>
<p>Turrican's signature weapon is the <strong>spinning energy beam</strong>: holding the fire button deploys
it, and while it is active the player can sweep it through its 32 rotation angles but cannot move, before it
releases in a short burst. Its sprite is one of the shared resident sprites — the same 32-frame rotation
plus burst — rather than a per-world enemy graphic, so it is available in every world.</p>
`,
  },
  marble: {
    loader: `
<div class="info-eyebrow">Marble Madness · Image &amp; Loader</div>
<p>Marble Madness ships on a single Amiga floppy that boots through entirely <strong>stock AmigaDOS</strong> —
where a C64 tape hides a custom fastloader, here the disk is an ordinary bootable filesystem. The protection
is elsewhere: the main program is encrypted, and a from-scratch track loader reads it off the platter by
physical position.</p>

<h2>The disk and filesystem</h2>
<p>An ADF is a flat dump of the floppy's 1760 blocks of 512 bytes. The disk is a normal AmigaDOS
<strong>OFS volume</strong> named <code>MarbleMadness!</code>, with a standard boot block, a root block and
three directories holding 50 files. Almost every file is an Amiga loadable <strong>hunk object</strong> — a
relocatable code/data segment AmigaDOS brings in with <code>LoadSeg</code> — so the game is not one binary
but a launcher, a main program, and a large set of per-course overlays loaded on demand. Two files are
exceptions: the main program and a helper named <code>xxx</code> are stored <strong>encrypted</strong>,
near-random at the byte level, and decrypted at load.</p>

<h2>Booting to Workbench</h2>
<p>The boot block is the unmodified AmigaDOS boot code: it finds the resident <code>dos.library</code> and
hands off to it. DOS runs a two-line startup script — <code>LoadWb</code> then <code>endcli</code> — that
brings up Workbench (the desktop is a service of the Kickstart ROM; the disk only bundles
<code>icon.library</code> and its icon files so it can show its own window on a bare machine). The player
launches the game by double-clicking the marble icon, which <code>LoadSeg</code>s and runs the launcher.</p>

<h2>The launcher and the encrypted program</h2>
<p>The launcher is a small compiled program that, finding it was started from Workbench rather than a shell,
displays the splash screen and brings the game in. Because the main program is encrypted, that load does not
go through plain <code>LoadSeg</code> — it goes through a small decryptor named <code>zzz</code>, a custom
<strong>decrypting <code>LoadSeg</code> replacement</strong>. It reads the encrypted file, undoes a keystream
XOR, relocates the hunks, and hands the segments back. The cipher is keyed in part to machine state, so the
decryption is bound to a particular Kickstart and a booted process, not just the disk (see Game Engine).</p>

<h2>A from-scratch track loader</h2>
<p>The other encrypted file, <code>xxx</code>, is decrypted first and run as code: it is a custom floppy
<strong>track loader</strong>. It reads neither through AmigaDOS nor through <code>trackdisk.device</code>'s
normal commands — it drives the floppy hardware directly: the CIA drive-control and status ports to seek and
check readiness, and Paula's disk DMA to pull a whole raw MFM track in one burst, then MFM-decode and
validate each sector in the CPU. It reads the main program by <strong>physical track and sector position,
not by name</strong> — the filename never appears in the launcher at all. The program still exists as a real
DOS file (so the disk stays a valid bootable volume and its blocks are laid down contiguously), but it is read
by location for speed and as a copy-protection hook: a from-scratch reader can demand non-standard formatting
and bypass file-level tampering.</p>
`,
    engine: `
<div class="info-eyebrow">Marble Madness · Game Engine</div>
<p>Reaching the game's own code means getting through the encryption and the copy protection wrapped around it.
Once decrypted, the program is a stripped Amiga hunk object — almost entirely code — that drives the hardware
directly and runs as two cooperating tasks.</p>

<h2>The multi-stage load</h2>
<p>The launcher loads the game in stages over a shared control block: it decrypts the track loader
<code>xxx</code> with the decryptor <code>zzz</code> (with an empty key), runs <code>xxx</code> to read the
175&nbsp;KB main program off the disk by physical position, then mutates the key and runs <code>zzz</code>
again to decrypt the program. The track loader is the fast raw reader; the decryptor is the cipher; they
cooperate through the shared block, and the key changes between the two decrypt passes.</p>

<h2>The cipher</h2>
<p>The on-disk format is a standard AmigaDOS hunk with <strong>selective encryption</strong>: the hunk-header
magic, the block-type markers and the symbol names are left in plaintext, so the file's structure stays
legible, while the hunk sizes, relocation tables and code/data bodies are XORed with a keystream — one
keystream longword per stored longword. There is no compression; the bodies are full size and the high entropy
is the cipher. The keystream is an additive <strong>lagged-Fibonacci generator</strong> over a 55-entry table
built from a fixed seed by a multiply-hash.</p>

<h2>The copy protection</h2>
<p>The teeth are in the key setup. Before the keystream runs, the table is perturbed by folding in two pieces of
<strong>live machine state</strong>: bytes from the host's CPU exception and TRAP vector table in low memory,
and the running task's exception- and trap-handler pointers. Because those entries feed the generator, the
keystream past its first stretch depends on the vector table and the task — which is why the file's structure
decodes regardless (its keystream is drawn before the perturbed entries propagate) while the bodies scramble.
On Kickstart 1.x every exception vector points at the same ROM handler, so the relevant byte is just the ROM
page, tying the protection to the 1.x ROM layout; the handler pointers only exist once AmigaDOS has constructed
the launcher's process, so the full key is not present until the game is actually booting on the right vintage
of machine. This is why such titles are Kickstart-version-locked: the decryption key is, in effect, the ROM
page.</p>

<h2>Inside the program</h2>
<p>Decrypted, the main program is a stripped hunk load file — not merged into a few segments but <strong>347
hunks</strong> (about 115 object modules), each keeping its own code/data/BSS triple, with no symbol or debug
blocks. It is mostly code: the bitmaps and samples live in separate per-course files, so the program carries no
pixel or sample data — only the engine that drives them, and it drives them at the metal, writing the full
blitter register block and <code>DMACON</code> directly and reading the trackball ports. The small data payload
is the UI text (the course banners, "GAME OVER", the player labels), the per-course level filenames it loads at
run time, and a few lookup tables.</p>

<h2>Two cooperating tasks</h2>
<p>The running game is two contexts that talk over exec messages: a <strong>main thread</strong> (the Intuition
front-end and the game-state machine) and a separate vblank-synced <strong>"Framer" task</strong> that owns the
display refresh. There is no single linear loop — the gameplay update and the display refresh run in different
contexts. The Framer task wakes once per vertical blank, animates the cycling colours, and rebuilds the
copper/display list when a fresh world frame is ready; the main thread runs the state machine (set up the
course, then play), which each frame integrates every object and draws it.</p>
`,
    graphics: `
<div class="info-eyebrow">Marble Madness · Graphics</div>
<p>Marble Madness's graphics are <strong>blitted, not sprited</strong>: the program draws everything itself from
per-course banks of tiles and obstacle cells, scrolling a single tall course vertically. The boot screen uses
standard Amiga formats; the per-course art uses one shared RLE codec.</p>

<h2>The splash screen</h2>
<p>The title screen is a standard IFF ILBM bitmap — 320&times;200, four bitplanes (16 colours), its pixels
<strong>ByteRun1 (PackBits)</strong> compressed, with a palette and four colour-cycling ranges so parts of the
logo animate. A small boot-screen overlay loads the image and puts it up while the game streams in.</p>

<h2>Tiles</h2>
<p>Each course's floor, walls and railings — everything the marble rolls on — are a <strong>tile map</strong> in
its own file, a single PackBits stream. Unpacked, it holds a 16-colour palette, four bitplanes of tile graphics,
and a tilemap. Tiles are 8&times;8 pixels in four bitplanes; the tilemap is a row-major stream of tile-index
words, a constant <strong>36 tiles (288 px) wide</strong> — Marble Madness scrolls only vertically, so the width
is fixed and the height varies per course, from Practice's 75 rows up to Ultimate's 198. Placing each tile by
index reproduces the whole course. Four palette slots are not fixed colours but are driven at runtime by
colour-cyclers — two ramps for the hazard/lava pulse and the ice shimmer — so a static palette can't show them.
(The tilemap is only the visual surface; the physics rolls the marble on a separate height field.)</p>

<h2>Obstacle cells</h2>
<p>Each course also carries a bank of <strong>obstacle sprites</strong> — the goal flag, moving barriers,
drawbridges and the like — also one PackBits stream, holding a count and a table of cell descriptors over
contiguous planar pixel data. Each cell is one complete animation frame, in one of two layouts chosen by a type
byte: "stored" free sprites (the flag, the marble, the creatures) keep their bitplanes row-interleaved and are
drawn with a per-object colour ramp, while "composited" scenery is sequential plane blocks OR-composited into the
playfield. The moving creatures and the marble itself live in separate banks that share this container.</p>

<h2>The course layout</h2>
<p>A third per-course file holds everything else a course needs — not just object positions but all of its
gameplay data. It is a plain hunk module loaded at course init, opening with a header of relocated pointers the
engine fans out to the actor-system globals: the static slope field, a placement/feature table, the coarse-zone
partition, the animation scripts, the creature spawn lists and the actor list. (Despite the suffix on some
filenames, these are not music — the audio is in separate sound banks.)</p>
`,
    gameplay: `
<div class="info-eyebrow">Marble Madness · Gameplay</div>
<p>The marble is a real <strong>3-D simulation</strong> projected to the isometric view — rolled by a relative
trackball over a height-mapped course, with a state machine governing rolling, falling, landing and the dizzy
spin.</p>

<h2>The trackball and the marble</h2>
<p>Input is the trackball's quadrature counters, per player — a <strong>relative</strong> device, so spinning
faster pushes harder — accumulated into a roll-force. The marble is not a 2-D sprite but a <strong>point
mass</strong> with velocity and position in three dimensions; each frame the engine integrates position by
velocity and then iso-projects to the screen, so the isometric look is a projection of a real 3-D model. Exactly
three things write the marble's velocity: the scaled trackball force; friction and an octagonal speed cap
(clamped per surface — that selector is the ice/grating friction); and the surface force from the terrain.</p>

<h2>The course as a height map</h2>
<p>The terrain is two independent systems. The course itself is a <strong>static slope field</strong>: a list of
region records, each a flat isometric-tile rectangle carrying a base height, a direction and a one-dimensional
profile, rasterised at load into a <strong>corner-height mesh</strong> — a grid of cells each holding the four
corner heights of one tile. All the regions compose into one continuous 2.5-D height map, and the triangular
slope faces you see are emergent: a quad with non-coplanar corners is two triangles. Each frame the engine
samples the four mesh cells around the marble, picks which of the tile's two triangles it is over, computes the
surface gradient, and accelerates the marble down it — except on the <strong>Aerial</strong> course, which adds
instead of subtracts, giving its inverted low gravity. The walls fall out of the same mesh: a height step between
neighbouring cells becomes a side the velocity is clamped against. One height map drives both the roll and the
walls, with no per-cell terrain codes.</p>

<h2>Scripted regions</h2>
<p>A few moving or interactive surfaces — seesaws, the rail-guarded holes, the start and finish triggers, the
ball-catcher — are a separate system: a per-course list of regions whose reference point is emitted by a small
<strong>bytecode animation script</strong> (keyframes, a move opcode for a sliding slope or seesaw, sound
triggers). Each frame the engine matches a region to the marble's tile and dispatches on its terrain code: slope
codes push the marble toward the reference point, trigger codes raise wall flags that snap and bounce it.</p>

<h2>The marble's state machine</h2>
<p>All of this runs under a <strong>twelve-state machine</strong> on the marble. Three states are
player-controllable — they run input and physics — while the rest are animation or transition states that only
redraw the marble: rolling, landing after a drop, an edge reaction, falling and settling onto a surface, an
object-bump on contact with an enemy, the course-intro run, the spawn, and the hole/region capture. A notable one
is <strong>dizzy</strong>: a survivable hard hit (by another marble) or fall is <em>not</em> death — it sets a
stun flag, and the rolling state hands off to a swirl-spin that plays out and returns to rolling. Death is
running off the edge onto no terrain at all, or the hazards and the marble-munchers.</p>

<h2>Actors</h2>
<p>The moving things — the goal flag, the enemies, the munchers — are <strong>actors</strong> fed by the
course-layout data. Each frame the engine walks an array of actor records, each holding a sprite-cell pointer,
an animation-script pointer (a cell list advanced when a frame timer expires, with randomised durations) and a
position. There are <strong>no hardware sprites</strong> — everything is blitted, including the two-blit
wraparound across the course's vertical scroll seam.</p>
`,
    music: `
<div class="info-eyebrow">Marble Madness · Music</div>
<p>Each course has its own theme — in fact two per course, fourteen tunes in all — and the notable thing is how
they play. Unlike the bare-metal C64 and Turrican drivers that bang Paula's registers directly, Marble Madness
plays its music through the operating system's <strong><code>audio.device</code></strong>: it sequences the song
and, per voice, hands the OS a sample pointer, a period and a volume, letting <code>audio.device</code> perform
the actual Paula DMA. So the player is much simpler than a TFMX-class engine — no macro VM, no software mixing,
just "play this sample at this pitch and volume".</p>

<h2>Where the music lives</h2>
<p>The six per-course sound banks are ordinary AmigaDOS hunk modules of <strong>pure data</strong> — there is no
player code in the file; the player lives in the main program. Their data blocks carry the chip-memory flag (the
only memory Paula can DMA from) and hold a song header, instrument and envelope tables, and 8-bit signed PCM
sample waveforms. The song names its instruments by relocated pointer rather than by index.</p>

<h2>The sound engine</h2>
<p>The player runs as its own exec <strong>task</strong>, and the rest of the engine talks to it only by message
— a clean producer/consumer split. It is <strong>dual-clocked</strong>: an audio-reply clock advances each voice's
note list (when <code>audio.device</code> finishes a voice's sample it replies, and the player writes that
voice's next note), and a 60&nbsp;Hz timer clock ticks the per-voice pitch and volume envelopes on sustained
notes. The engine is a general sampled-sound driver — music is just a set of voices it triggers — addressed by
sound number through a per-bank directory; a course's music is one of those entries, whose event lists are the
long, looping score.</p>

<h2>The song format</h2>
<p>The music itself is a <strong>Soundtracker-style arrangement</strong>: an order table per channel of
<code>(repeat, pattern)</code> entries — play a pattern so many times, then advance, with a zero repeat looping
back to the start — over patterns that are byte-streams of note commands (a note as octave-and-semitone, a rest,
a note-length class, an end marker). Each note is voiced from the bank's <strong>single shared waveform</strong>:
the semitone indexes the standard Amiga/ProTracker period table for the fine pitch, and the octave selects the
length of the looped waveform slice — the classic one-sample-many-octaves trick. A per-note volume envelope (a
list of rate/target segments ramped one step per frame) gives each note its pluck shape rather than a flat tone.
A per-frame music tick advances the song at about 50&nbsp;Hz, emitting notes through the same voice path.</p>
`,
  },
  stuntcar: {},
  elite: {
    loader: `
<div class="info-eyebrow">Elite · Image &amp; Loader</div>
<p>Elite for the C64 loads from cassette, and the tape is as much a <strong>copy-protection device</strong>
as a loader. The game is split across a dozen tape segments in a custom turbo format with no checksums, and
the loader rewrites its own wire format between blocks — so the bytes can only be read by running the loader
the tape itself carries.</p>

<h2>The tape image</h2>
<p>A TAP file records the length of every pulse the datasette delivers. The tape opens with two ordinary
CBM ROM-format segments — a header naming the file <code>ELITE</code> and a 289-byte boot program — then
switches to a custom turbo encoding for the dozen segments that carry the game, its graphics and its data.
The ROM-format part is the bootstrap; the turbo part is everything else.</p>

<h2>The bootstrap and autostart</h2>
<p>The first two segments load through the C64's own ROM tape loader. The trick is the boot program's
<strong>load address</strong>: the KERNAL saves the IRQ vector while it uses the tape and restores it from a
fixed pair of bytes when the load finishes — and the boot file is positioned so that those two bytes become
the IRQ vector, pointing into the just-loaded code. So the first timer interrupt after "FOUND ELITE" jumps
straight into the boot code, with no <code>RUN</code> needed. The boot code fills memory, turns off the
KERNAL messages, and jumps into the turbo loader.</p>

<h2>The turbo loader</h2>
<p>From there the tape uses a custom one-bit-per-pulse encoding — a short pulse is 0, a long pulse is 1,
told apart by a CIA timer restarted on every cassette edge. A byte is nine pulses (a start bit and eight data
bits), and the stream is a pilot tone, a sync byte, then blocks <strong>back to back with no checksums and no
gaps</strong>. Each block is a four-byte header (its end and start address) followed by its data, stored
straight into memory. The store loop deliberately <em>never terminates on its own</em> — it always branches
back for another block header.</p>

<h2>A self-rewriting loader</h2>
<p>That non-terminating loop is the hook the protection hangs on. To end a load, the tape simply sends a block
whose address range <strong>covers the loop itself</strong>: as it loads, the branch instruction at the loop's
tail is overwritten, so when the block completes the loop falls through into freshly loaded code. Between
payload blocks the tape sends tiny one-to-three-byte blocks aimed at single instructions of the loader,
<strong>changing the wire format on the fly</strong> — flipping the bit order between MSB-first and LSB-first,
changing how many bits make a byte frame, growing the header with a decoy byte, and rewriting the whole loader
tail (sometimes byte-identical, a decoy that has to race the executing loop). The main payload is sent as many
small blocks with patch blocks interleaved between almost every page, so the data cannot be read off the tape
without executing the patches. It is a copy-protection scheme, not just a fastloader.</p>

<h2>The multi-stage load</h2>
<p>The chain runs through a short BASIC stub (a tape <code>LOAD</code> from inside a running BASIC program
restarts it afterwards, so a flag variable makes each line run once) into the game's own multi-load routine,
which pulls in the remaining segments — the engine, the in-game code, the colour data and the bitmap loading
picture — behind the loading screen, then jumps into the game.</p>
`,
    engine: `
<div class="info-eyebrow">Elite · Game Engine</div>
<p>Everything in the game image arrives <strong>encrypted</strong> and is unpacked in stages, then relocated —
much of the engine hidden as RAM under the I/O area — before the game brings up its own interrupt-driven
display and reaches its first frame.</p>

<h2>Layered decryption</h2>
<p>None of the game code is ever in plaintext on the tape, and even after loading it is decrypted in pieces,
at different times, with different keys. Three near-identical decryptors do the work, each a
<strong>rolling-subtraction cipher</strong>: every byte is the previous plaintext byte subtracted from the
ciphertext, so the key rolls forward as it goes, and the loop's address range is self-modified. The first
stage decrypts the loaded blob in two downward passes with different keys; a later stage decrypts the rest in
place once it has loaded.</p>

<h2>Relocation under I/O</h2>
<p>The decrypted engine relocates itself. Part of it is copied into low memory; the bulk — about 8&nbsp;KB —
is copied <strong>underneath the I/O area</strong> at <code>$D000–$EFFF</code>, hidden as RAM the routines
reach by toggling the processor port's bank bits (banking I/O out, reading or writing, then banking it back
in). The ship-model data and much of the engine live there, out of the way of the normal memory map.</p>

<h2>Hardware init and interrupts</h2>
<p>Hardware init neutralises RUN/STOP–RESTORE, banks the KERNAL and BASIC ROMs out, and points the CPU's IRQ
and NMI hardware vectors at <strong>RAM</strong> — so with no ROM in the path, interrupts dispatch straight
into the game's own handlers. The interrupt is a <strong>table-driven raster-split engine</strong>: it reads
the current split index and loads that split's VIC register values from a set of two-entry tables, giving two
splits per frame (the bitmap space view and the dashboard). When the second split completes the frame, the
handler falls through to its per-frame work — a three-voice SID player — before returning.</p>

<h2>Reaching the first frame</h2>
<p>From the init onward the game is interrupt-driven: the game start runs its one-time setup, builds the
title / commander screen, and settles into the main loop. The picture shown during the long loads is a
multicolor bitmap — the 3-D "ELITE" logo and a Cobra over a starfield — stored uncompressed and assembled
from three tape segments, displayed before the two largest loads so it masks the slowest part. Once flight
begins, two control flows cooperate: the <strong>raster interrupt</strong> only swaps VIC registers between
the bitmap and dashboard and ticks the music, with no game logic, while a <strong>foreground loop</strong>
does everything else — moving and drawing every object, spawning, combat, scoring and reading the controls.</p>
`,
    graphics: `
<div class="info-eyebrow">Elite · Graphics</div>
<p>Elite's graphics are <strong>vector, not bitmap</strong>: ships are wireframe models drawn line by line,
and the whole universe of star systems is generated from a tiny seed rather than stored. Even the text is held
as compressed tokens, not letters.</p>

<h2>Ship models</h2>
<p>Each ship is a filled-edge wireframe — a list of 3-D vertices joined by edges, with face normals used to
hide the back. The models live in the engine block hidden under the I/O area, reached through a blueprint
pointer table of 33 ship types. A blueprint is a 20-byte header followed by three packed arrays — vertices,
edges and faces — with <strong>no explicit counts</strong>: each array's length falls out of the offsets,
because every record is a fixed size. A vertex is six bytes (its three coordinate magnitudes, a byte packing
the three signs and a visibility distance, and the numbers of up to four faces it belongs to); an edge is four
bytes (a visibility distance, the two faces on either side, and its two vertex numbers); a face is four bytes
(the signed normal plus an illumination field). A face number of 15 is a <strong>sentinel</strong> meaning "no
face on this side" — an edge carrying it is always drawn, never culled, which is how flat models like the alloy
plate show their outline from any angle.</p>

<h2>Drawing a ship</h2>
<p>Drawing runs a fixed pipeline: copy the ship's state into a zero-page workspace and point at its blueprint;
rotate the model and drop or simplify it by distance (level of detail); project each vertex into view space
with a perspective divide; back-face-test each face with its normal and append every visible edge to a per-ship
<strong>line heap</strong> as endpoint records; then walk the heap drawing each as a Bresenham line into the
multicolor space-view bitmap. Because lines are plotted with EOR, the previous frame's ship is erased by drawing
its old line heap again before the new one is built — the standard flicker-free Elite redraw. Distant ships and
stars are single dots of one, two or four pixels by distance.</p>

<h2>A universe from a seed</h2>
<p>Elite's universe — hundreds of star systems, each with a name, an economy, a government and a position — is
<strong>not stored</strong> anywhere; a search of the image for names like Lave or Diso finds nothing. Every
name and attribute is generated on demand from a tiny seed. A whole galaxy is defined by a <strong>six-byte
seed</strong> (the only one stored sits in the default commander block), and a single Fibonacci-style
pseudo-random generator — a two-word lagged sum over a four-byte seed — advances it deterministically, so the
same starting seed always yields the same stream and a system's name and data are reproducible. To work on a
system, the code loads that system's seed and runs the generator. A name is one to four two-letter
<strong>digram</strong> pairs drawn from a small letter-pair table — the only stored fragment of any name —
and coordinates and the star/dust field come from the same seed and generator. The other galaxies are
transforms of the same seed, so the entire universe lives in those six bytes plus the algorithm.</p>

<h2>Text as tokens</h2>
<p>Almost every word the game prints — commodity names, government and economy descriptions, combat ranks,
mission briefings, even "GAME OVER" — is stored not as letters but as compressed, EOR-obfuscated
<strong>tokens</strong>: recursive tokens that expand to other tokens, two-letter digrams from the same table
the names use, and control codes that insert the commander's name, a generated captain's name or a number. A
raw search for any in-game phrase finds nothing, because the letters are never there in sequence.</p>
`,
    gameplay: `
<div class="info-eyebrow">Elite · Gameplay</div>
<p>Once flight begins, the game is about the <strong>other ships</strong> — how traffic appears, moves and
fights, and when it vanishes again. It runs in a foreground loop over a small, fixed set of object slots.</p>

<h2>The universe of slots</h2>
<p>The universe is at most ten object <strong>slots</strong> — the planet, the station or sun, and up to a
handful of ships — held in a slot array of ship-type bytes and a table of pointers to each slot's 37-byte
record (position, orientation, speed, AI and state flags). The update pass copies one slot's record into a
zero-page workspace, works on it there (so the same fast code serves every slot), and copies it back. The
<strong>player's ship never moves</strong>: the universe is rotated and translated around a stationary camera,
so flying forward makes everything stream toward you.</p>

<h2>Spawning</h2>
<p>New ships go through one routine that finds a free slot (ten full means nothing spawns), looks up the
blueprint, and places ordinary traffic far away so it approaches from a distance. A <strong>spawn
director</strong> runs periodically — never while docked — and makes weighted random rolls: a chance of lone
trader traffic; law-enforcement ships scaled by how many police are already present and by the player's
accumulated bounty; and pirates gated by the system's government and the commander's danger rating, as either a
single ship or a pack of up to four. So traffic is a mix of traders, police scaled to your record, and pirates
scaled to the system's lawlessness — plus the station and any escorts the AI launches itself.</p>

<h2>Movement and AI</h2>
<p>Each ship is moved forward along its nose vector, rotated by its angular velocity, then re-orthonormalised so
rounding errors do not accumulate, and handed to the wireframe renderer. Ships with hostile AI run a tactics
routine, but only <strong>every eighth frame</strong> — the expensive AI is time-sliced across ships and
frames. The station does not fly; its "AI" is a launch controller that sends an enforcement ship after a wanted
player, and mothership-class ships launch escorts the same way. Ordinary combat ships track their target
(normally the player), accelerate toward their speed limit, steer to point at or away, and — by their
aggression and the player's bounty — fire lasers or break into evasive manoeuvres; a ship that takes enough
damage can flip from attacking to fleeing.</p>

<h2>Despawning</h2>
<p>After a ship is moved, its fate is decided three ways, all funnelling into one removal routine: marked for
removal (it finished what it was doing), <strong>killed</strong> (the reward is banked first — and killing a
police ship adds to your bounty rather than your bank, which is where a clean trader becomes a fugitive), or
<strong>out of range</strong> (drifted too far on any axis — which is why traffic quietly thins out behind you).
Removal decrements the population counters and compacts the slot and pointer arrays so the list stays
contiguous.</p>

<h2>Legal status and bounty</h2>
<p>One byte holds your standing with the law, shown as <strong>Clean</strong>, <strong>Offender</strong> or
<strong>Fugitive</strong>. Every ship carries an offence value that is OR-ed into the byte when you destroy it,
so offences ratchet upward; lawful ships carry the largest value, so shooting a police ship turns a clean pilot
into a Fugitive in a single act. Each hyperspace jump halves the byte, so minor offences cool off over a few
quiet jumps. It both follows your behaviour and drives the game: a dirty record spawns more police, and past a
threshold ships engage you on sight.</p>

<h2>Combat rating, and winning</h2>
<p>Separately, every kill pays a bounty into a cumulative score that drives a nine-step <strong>combat
rating</strong> from Harmless up to Elite — it only ever climbs, and reaching the top takes thousands of kills.
Rating and legal status are independent axes: an Elite pilot can still be a Fugitive. There is <strong>no
victory state</strong> in the game at all — the only way the main loop is left is the death sequence (energy
gone under fire, a hard collision, or flying into a planet or sun), after which you resume from your last saved
commander. Elite is deliberately open-ended; "Elite" is the nominal goal, but reaching it only changes a
label.</p>

<h2>Docking</h2>
<p>Docking is the one piece of flight the game does not do for you. A legal dock requires four things at once:
approaching the station's <strong>correct face</strong>, pointing <strong>nearly straight in</strong>,
<strong>laterally lined up</strong> with the slot, and <strong>rolled</strong> to fit the rectangular opening —
all while slow enough that the contact stays under the fatal threshold. Because the station continuously
rotates, you must roll with it to keep aligned, which is the whole skill. Miss the slot at speed and you hit the
hull — the same death path as crashing into a planet. (A purchasable docking computer flies the same approach
automatically.)</p>

<h2>Thargoids</h2>
<p>The alien <strong>Thargoids</strong> sit on top of the ordinary traffic: a tough, aggressive mothership and
the small drones it launches. They arrive either through a <strong>misjump into witchspace</strong> — a random
check on a hyperspace jump that dumps you into a nest of them — or, rarely, as a random in-system attack. The
mothership keeps manufacturing fresh drones while it lives, so a drawn-out fight only grows the swarm. The neat
trick is that the drones read the same population counter that tracks live motherships: while it is non-zero
they attack, but the moment the last mothership dies the drones go inert and drift — there is no explicit
"deactivate my children" code. Kill the motherships and the swarm switches off.</p>

<h2>Missions, economy and government</h2>
<p>On top of the open-ended trading and combat is a small <strong>scripted mission chain</strong>, gated by how
many times you have docked and by specific destination systems: a briefing on docking sends you to a named
system to hunt a unique, tougher enemy or to carry stolen Thargoid plans, with a large fixed reward on return.
Each system's character comes from its <strong>seed</strong>: an economy and a government (each three bits),
with tech level and productivity derived from them and coupled — lawless systems are poor and backward,
well-governed wealthy ones high-tech. Economy drives prices through a signed per-commodity slope, so the same
good is cheap in one economy and dear in another — the whole basis of trade routes; government drives danger,
gating how freely pirates spawn; and tech level decides what equipment is for sale. So the risk and reward of
where to fly fall out of a few bytes per system.</p>
`,
    music: `
<div class="info-eyebrow">Elite · Music</div>
<p>Elite's audio is a <strong>three-voice SID</strong> score and sound effects, driven from the raster
interrupt rather than a separate sound thread.</p>

<h2>The SID player</h2>
<p>When the last screen split of a frame completes, the raster-interrupt handler falls through to a three-voice
SID player that advances the music and sound effects from a set of tables and writes the SID's registers — so
the audio is updated a fixed once per frame, underneath the foreground loop that runs the flight model. Docking
clears the SID registers to silence the music before the station menus appear.</p>
`,
  },
};

// HTML for a game/tab, or null if nothing has been written for it yet.
export function infoHtml(gameId, tabId) {
  const game = INFO_CONTENT[gameId];
  return (game && game[tabId]) || null;
}
