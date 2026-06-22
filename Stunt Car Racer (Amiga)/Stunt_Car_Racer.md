# Stunt Car Racer (Amiga) — disk format, tracks and physics

A reverse-engineering reference for `Stunt Car Racer.adf`, the Amiga release of
Geoff Crammond's *Stunt Car Racer* (1989) — a filled/wireframe-vector stunt
racing game built around an unusually advanced (for its day) rigid-body car
simulation running on elevated 3-D circuits. It is the second vector game in
this repository (after Elite) and the first whose **goal is the simulation**
rather than the static assets.

The writeup follows the same shape as the other titles, in reading order, but
the centre of gravity is the last two parts — the tracks and the physics:

* **Part I** — the disk image: the ADF container and the custom (non-AmigaDOS)
  on-disk format — enough to pull every byte off the disk;
* **Part II** — the boot chain: the boot block, the custom track loader it
  bootstraps, and how the game and its data load;
* **Part III** — the game program: the 68000 startup, the interrupt/Copper/
  blitter setup and the memory map;
* **Part IV** — **the tracks**: the vector format of the elevated circuits
  (the track sections, their 3-D geometry and connectivity) and a Go
  reimplementation that extracts and re-draws them;
* **Part V** — **the physics**: the car's rigid-body simulation — the chassis,
  suspension, wheel/ground contact, drive and damage model — reverse-engineered
  from the 68000 integrator and reimplemented in Go so a track can be *driven*.
* **Appendices** — toolchain and reproduction.

Methods: purely static analysis of the disk image, plus the 68000 toolchain in
the shared `tools/` module — the AmigaDOS reader (`tools/amiga/adf`), the
disassemblers (`tools/cmd/dis68k`, `tools/cmd/codetrace68k`) and an
instruction-level 68000 execution core (`tools/m68k`) for dynamic verification.
All addresses are 68000 addresses; sizes are `.b`/`.w`/`.l` (8/16/32-bit).
**Status: complete. Parts I–III (disk format, loader, engine architecture), Part IV (the
track geometry — footprint, surface and camber, decoded and verified coordinate-exact) and
Part V (the full rigid-body car physics, verified frame-by-frame against the original) are
all done; combined, a car can be driven over the decoded tracks.**

---

## Contents

- [Part I — The disk image](#part-i--the-disk-image)
  - [1. The ADF container](#1-the-adf-container)
  - [2. Not an AmigaDOS filesystem](#2-not-an-amigados-filesystem)
- [Part II — Boot chain and loader](#part-ii--boot-chain-and-loader)
  - [1. The boot block](#1-the-boot-block)
  - [2. The custom track loader](#2-the-custom-track-loader)
- [Part III — Game program architecture](#part-iii--game-program-architecture)
  - [1. Entry, self-check and supervisor setup](#1-entry-self-check-and-supervisor-setup-e700)
  - [2. Hardware bring-up](#2-hardware-bring-up-ed56)
  - [3. Game bootstrap and top-level loop](#3-game-bootstrap-and-top-level-loop-1ba08--5c890)
- [Part IV — The tracks (vector circuits)](#part-iv--the-tracks-vector-circuits)
  - [1. Finding the track table](#1-finding-the-track-table-the-race-setup-path)
  - [2. The record header](#2-the-record-header)
  - [3. The section stream and spine builder](#3-the-section-stream-rle-and-the-spine-builder)
  - [4. A verified Go decoder](#4-a-verified-go-decoder)
  - [5. The spine geometry and re-drawing the circuits](#5-the-spine-geometry-and-re-drawing-the-circuits)
  - [6. The plan footprint (the 16×16 track grid)](#6-the-plan-footprint-the-1616-track-grid)
  - [7. Banking and elevation](#7-banking-and-elevation)
- [Part V — The physics simulation](#part-v--the-physics-simulation)
- [Appendix A — Toolchain and reproduction](#appendix-a--toolchain-and-reproduction)

---

# Part I — The disk image

## 1. The ADF container

`Stunt Car Racer.adf` is a raw, 901,120-byte Amiga floppy image — the usual
double-density layout of **80 cylinders × 2 heads × 11 sectors × 512 bytes**
(`80 × 2 × 11 × 512 = 901,120`). A byte offset on the disk maps linearly to a
sector: `sector = offset / 512`, with no interleave at the image level. Its
identity is pinned in the repository [README](../README.md#image-files) by size
and MD5 so the analysis stays reproducible.

The image opens with a boot block whose first four bytes are the ASCII tag
`DOS\0` (`44 4F 53 00`) — the standard "bootable AmigaDOS disk" magic — followed
by the boot-block checksum and the boot code:

```
000000  44 4F 53 00 30 AD 90 C0  00 00 03 70  ...   DOS. 0...  rootblk=$370(880)
                    └ checksum ┘  └ rootblk ┘
00000C  24 49 4F FA 03 F0 2C 78  00 04 4E AE  ...   ← 68000 boot code starts here
```

So at face value it looks like an ordinary AmigaDOS disk that boots Workbench.
It is not (§2): the `DOS\0` block is just enough to be *bootable*; the boot code
ignores any filesystem and pulls the game off the disk itself (Part II).

## 2. Not an AmigaDOS filesystem

The boot block names root block 880, but block 880 is **not** a valid AmigaDOS
root header (`tools/amiga/cmd/adfdump` rejects it: *"root block is not a valid
root header"*). There is no DOS filesystem, no directory, no files — the disk is
**custom-formatted**: a flat region of game code and data that only the game's
own loader understands.

This is the same pattern as Turrican in this repository (and most commercial
Amiga games of the era): a minimal DOS-looking boot block that bootstraps a
bespoke track-loading scheme, both to fit the data densely and to resist
copying. Everything past the boot block — the loader, the engine, the track
geometry and the physics tables — has to be located by following that loader
rather than by reading a filesystem.

---

# Part II — Boot chain and loader

## 1. The boot block

The boot code (from image offset `$0C`) is a compact, self-contained track
loader. It runs while the Kickstart still has the disk inserted, with `a1` =
the boot-time IO request (an `IOStdReq` on `trackdisk.device`) and `a6` =
`ExecBase`:

```
Forbid()                                  ; JSR -$84(a6) — stop multitasking
d0 = $9800 ; d1 = MEMF_CHIP|MEMF_CLEAR     ; ($10002)
a3 = AllocMem(d0, d1)                       ; JSR -$C6(a6) — 38 KB of cleared chip RAM
io_Data($28)   = a3                         ; read destination = the buffer
io_Length($24) = $9800                      ; 38912 bytes
io_Offset($2C) = $2C00                      ; source = disk offset $2C00 (sector 22)
io_Command($1C)= 2  (CMD_READ)
DoIO()                                      ; JSR -$1C8(a6) — read it in
... (retry on io_Error) ...
io_Command = 9 ; io_Length = 0 ; DoIO()    ; motor off
Permit()                                    ; JSR -$96(a6)
JMP (a3)                                    ; enter the loaded code
```

So the boot block **reads a 38 KB blob from disk offset `$2C00` into chip RAM
and jumps to it**. That blob is the game's real loader/engine bootstrap; the
boot block itself does nothing game-specific beyond fetching it. (The boot
block also carries a short ASCII string near offset `$76` — `Prot…` — likely a
title/copyright/"protection" tag; to be transcribed.)

## 2. The custom track loader

The `$9800`-byte blob (disk sectors 22–97) is position-independent 68000 code
(every reference is `LEA …(pc)`), so it disassembles cleanly with its file
offsets as addresses. It does four things:

1. **Take over the machine.** A `MOVE.l #handler,$80 ; TRAP #0` pair jumps into
   supervisor mode at `$10`, then it kills the OS environment: `INTENA`/`INTREQ`
   cleared (`#$7FFF → $DFF09A/$9C`), `DMACON = $7C5F` (bitplane/copper/blitter/
   sprite/disk DMA + master on), a level-3 (VBlank) autovector installed at `$6C`
   pointing to its own handler (`$13A`), then `INTENA = $C020` (master + VERTB).
2. **Load stage data to `$4000`.** With `a0 = $4000` and a return address in
   `a1`, it branches to the disk reader (`$D76`) to pull an initial chunk to
   `$4000`.
3. **Show the title screen.** It unpacks a 4-bitplane image from an embedded
   table (`$D9C`) into chip RAM at `$78000` — four interleaved planes `$1F3E`
   bytes apart, `$FA0` words each (a full lo-res screen) — loads the palette
   (`$D7C → $206`) and points a Copper list at the planes (`$4A2`).
4. **Stream the game and enter it.** The load loop calls the disk reader to read
   **805 sectors starting at sector 110 into `$E700`**, retrying on error (and
   flashing the screen red and hanging if it can't); then it sets the user stack
   (`usp = $7FFFC`), the supervisor stack (`$3D80`) and `JMP $E700` — the game.

**The disk reader (`$570`).** This is a logical-sector reader, not a filesystem.
Its arguments are `d1 = start sector`, `d2 = sector count`, `a0 = destination`
(plus mode bits in `d0`/`d3`). It validates `start + count ≤ $6E0` (1760 =
80 cyl × 2 heads × 11 sectors) and converts a logical sector to a physical track
with `sector / $B` (11 sectors per track-side), then MFM-reads whole tracks (the
helpers at `$C22`/`$664`/`$9E4`/`$746`). Crucially it **only reads** — there is
no decompression — so on disk the data is stored raw, sector-aligned. (A
`"DOW"`-plus-incrementing-byte table at `$96CE` looks like a per-sector
track/format key, the same shape as Marble Madness's `sigfile`.)

### The disk map

Putting the boot block and loader together, the image is a handful of raw
regions read by sector — no filesystem, no packing:

| sectors | offset | bytes | loaded at | what |
|--------:|-------:|------:|----------:|------|
| 0 | `$0000` | 1024 | — | boot block (reads the loader, jumps to it) |
| 22–97 | `$2C00` | 38,912 | (chip) | the loader above |
| 110–914 | `$DC00` | 412,160 | `$E700` | **the whole game** (engine + tracks + physics) |

`extract/cmd/extract` slices these straight out of the `.adf`
(`extracted/loader.bin`, `extracted/game.bin`). Because nothing is compressed,
`extracted/game.bin` disassembles directly at base `$E700` — the input for
Parts III–V.

---

# Part III — Game program architecture

The game image (`game.bin`, `$E700`-based) is the 412 KB engine the loader streams
in. Its first bytes are not code that's reached by falling through — they are the
entry, a small trampoline, and an embedded Copper list — so the engine is best read
by following the three transfers out of the entry.

## 1. Entry, self-check and supervisor setup (`$E700`)

```
$E700  MOVEA.l #$E730,a0          ; a0 = start of the checksummed image
       MOVE.l  #$63BA8,d3         ; byte count ($63BA8 = 408 488)
$E70E  ADD.w   (a0)+,d0           ; 16-bit running sum over the whole image …
       SUBQ.l  #2,d3
       BNE     $E70E              ; … d0 = checksum
       MOVE.w  #$5834,d0          ; (expected value; a mismatch path exists)
$E730  MOVE.l  #$E73C,$80.l       ; install a TRAP #0 handler at vector $80…
       TRAP    #0                 ; … to drop into supervisor mode
$E73C  MOVEA.l #$EAD2,a7          ; set the supervisor stack
       JSR     $ED56              ; hardware init (below)
       JMP     $1BA08             ; main bootstrap (below)
```

So the image **checksums itself** ($63BA8 bytes summed as words), enters supervisor
mode through a self-installed `TRAP #0`, sets `a7`, and then does two things: the
hardware bring-up at `$ED56` and the game bootstrap at `$1BA08`. The bytes at
`$E74E` immediately after the trampoline are an **embedded Copper list** (bitplane
pointers `$078000…`, then the colour registers from `COLOR00` at `$180`), which
`$ED56` installs.

## 2. Hardware bring-up (`$ED56`)

`$ED56` is a textbook bare-metal Amiga take-over:

```
MOVE    #$2700,sr                 ; ints off, supervisor
MOVE.w  #$7FFF,$DFF09A / $DFF09C  ; INTENA / INTREQ: clear all
MOVE.w  #$E839,$DFF09A            ; INTENA = master|VERTB|… enable
MOVE.w  #$7CDF,$DFF096            ; DMACON  = master|bitplane|copper|blitter|sprite|disk|audio
MOVE.l  #$EEC8,$64 … #$F000,$7C   ; level-1…7 autovector table (handlers $EEC8/$EF0A/$EF1A/$EF5A/$EFC4/$EFF0/$F000)
…
MOVE.l  #$E74E,$DFF080            ; COP1LC = the embedded Copper list
MOVE.w  #$8380,$DFF096            ; DMACON: enable copper DMA
JSR     $EE8A                     ; CIA setup (timers/keyboard at $BFExxx/$BFDxxx)
;  ── anti-tamper ──
MOVEA.l #$F4B8,a0 ; MOVEA.l #$1AA4A,a2
EORI.b  #$80,(a0)+ ; CMPA.l a2,a0 ; BLT      ; XOR-$80 the range $F4B8..$1AA4A in place
JSR     $F402
RTS
```

Two details matter for the rest of the analysis:

* **The autovector table** at `$64..$7C` wires the seven 68000 interrupt levels to
  the engine's own handlers. The level-3 (VBlank, `$6C`) handler is `$EF1A`; the
  audio handler set lives alongside the sound-buffer pointers initialised just above
  (`$6A584/$6A588/$6A58C = $6A594` and `+$7D00`).
* **The XOR-`$80` decryptor** rewrites `$F4B8..$1AA4A` (≈ 46 KB) in place before the
  game runs. On disk that whole region is obfuscated, so a static disassembly of it
  is garbage until decrypted. `extract/cmd/extract` reproduces this pass and writes
  `extracted/game.dec.bin`; **all addresses in `$F4B8..$1AA4A` must be read from the
  decrypted image.** (Everything `≥ $1AA4A`, including the main bootstrap `$1BA08`
  and the top-level loop, is plaintext on disk.)

## 3. Game bootstrap and top-level loop (`$1BA08` → `$5C890`)

```
$1BA08 clear $EAD6[$80]                     ; small state table
       clear $7A01A..$7B6FA                 ; a screen / work region
       (conditional) fill palette at $E770
       JSR  $69CFC                          ; table init (per-entry loop over $6490…)
       JMP  $5C890                          ; top level
```

`$5C890` sets up the working screen and palette buffers and enters the game's outer
structure:

```
$5C890 MOVE.w #0,$23C32                     ; a mode/state word
       copy $1ECAA[256]  -> $7A91A          ; a 256-byte table (LUT)
       copy $5CF30[…]    -> $7A61A / $7A71A ; two parallel buffers (double-buffered colour/Copper)
       TST.b $64AF0 ; BNE …                 ; a "skip intro" flag
       JSR  $62D0A                          ; …
       BRA  $5C960                          ; the main loop body
```

From here the engine is a conventional VBlank-paced loop: a state word
(`$23C32`) selects the current screen (title / menu / track-select / race), the
double-buffered colour/Copper buffers (`$7A61A`/`$7A71A`) are swapped each frame,
and the level-3 handler (`$EF1A`) drives timing. The filled-vector race renderer,
the track interpreter (Part IV) and the car simulation (Part V) hang off the race
state of this loop.

*Memory map (run-time, so far):* engine code/data from `$E700`; the encrypted
`$F4B8..$1AA4A` block; sound buffers at `$6A594`; the working screen/Copper at
`$78000`+ and double-buffered colour tables at `$7A61A`/`$7A71A`; assorted state
words around `$23C32`/`$64AF0`. The track and physics tables are located in
Parts IV–V.

---

# Part IV — The tracks (vector circuits)

*The first goal.* Stunt Car Racer's circuits are short, elevated 3-D tracks
built from a sequence of **sections** — straights, banked curves, humps, jumps,
ramps and the collapsing "broken" bridge — each a piece of extruded ribbon
geometry with a height profile, joined end to end into a loop. The aim of this
part is to:

1. locate the track table on the disk and decode one section's format (its
   geometry: length, curvature, gradient/height, width, type flags, and how
   sections connect);
2. enumerate the game's built-in tracks (the league circuits);
3. reimplement the decoder in Go and **re-draw each circuit** (a 3-D wireframe/
   plan view), the way the Elite ship blueprints were re-rendered.

## 1. Finding the track table (the race-setup path)

The selected track is a single byte, **`$1CA33`** (0–7), set from the track-select
menu and printed by the name-printer `$64C3E` (which does `ASL.b #4,d1` — names are
16-byte records at **`$1EDAA`**, in track order: *LITTLE RAMP, STEPPING STONES, HUMP
BACK, BIG RAMP, SKI JUMP, DRAW BRIDGE, HIGH JUMP, ROLLER COASTER*; the AI opponents
are a parallel 14-byte table at `$1ECAA`).

Starting a race runs `$5D2CA`: `MOVE.b $1CA33,d1` then a chain
`JSR $5AE46 / $64304 / $5A794 / $696FC / $65BEC / $604B4`. The first of these,
**`$5AE46`, is the track loader**, and it is the key to the data:

```
$5AE46 MOVE.b d1,d0 ; ASL.b #1,d0 ; MOVE.b d0,d2     ; d2 = track_id * 2 (word index)
       MOVEA.l #$1F0A2,a2                            ; the track-pointer table
       MOVE.w  $0(a2,d2.w),$1BCC0                    ; word = table[id]
       MOVE.w  $1BCC0,d0
       ROL.w   #8,d0                                 ; byte-swap the word
       SUBI.w  #$B100,d0 ; ANDI.l #$FFFF,d0          ; -> a 16-bit offset
       ADDI.l  #$1EF82,d0                            ; + data base
       MOVEA.l d0,a5                                 ; a5 = this track's data stream
       MOVE.w  #0,d5
       … reads bytes through $5AE00 …
```

So a **track-pointer table of eight words at `$1F0A2`** indexes into the track data:
each word is byte-swapped, less `$B100`, added to the base `$1EF82` to give the
track's absolute address. The reader **`$5AE00`** is a plain sequential byte fetch
(`MOVE.b $0(a5,d5.w),d0 ; ADDQ #1,d5`) — the track is a **byte stream**, decoded in
order, not a fixed-size struct.

Decoding the table gives the eight tracks, stored **contiguously in track order**:

| id | track | address | bytes |
|---:|-------|--------:|------:|
| 0 | LITTLE RAMP | `$1F8E4` | 124 |
| 1 | STEPPING STONES | `$1F960` | 145 |
| 2 | HUMP BACK | `$1F9F1` | 145 |
| 3 | BIG RAMP | `$1FA82` | 142 |
| 4 | SKI JUMP | `$1FB10` | 145 |
| 5 | DRAW BRIDGE | `$1FBA1` | 213 |
| 6 | HIGH JUMP | `$1FC76` | 142 |
| 7 | ROLLER COASTER | `$1FD04` | 312 |

`extract/cmd/tracks` reimplements the `$5AE46` pointer math and writes the eight
raw streams to `extracted/tracks/<id>_<name>.bin` — the input for the format work.

## 2. The record header

Each track stream begins with a short parameter header that `$5AE46` reads first
(five bytes into `$1CA19[1..5]`, then two more into `$1BBF6..9`). The first eight
bytes of all eight tracks share a clear shape — **byte 1 always equals byte 2**:

```
LITTLE RAMP     2c 0f 0f | 25 00 05 a0 cf …
STEPPING STONES 38 2a 2a | 0e 00 0f a0 cf …
HUMP BACK       35 2e 2e | 13 40 05 60 04 …
BIG RAMP        2c 01 01 | 18 80 07 a0 c0 …
SKI JUMP        28 0f 0f | 23 40 6a aa bd …
DRAW BRIDGE     4e 2a 2a | 04 a0 11 a0 cc …
HIGH JUMP       34 1d 1d | 04 40 06 20 3f …
ROLLER COASTER  4e 00 00 | 25 00 05 a0 cf …
```

`$5AE46` reads these into `$1CA1A..$1CA1E`, which fixes their meaning:

* **byte 0 = section count** (`$1CA1A`) — the loop below runs until the section
  index reaches it. Per track: 44, 56, 53, 44, 40, 78, 52, 78 — i.e. each circuit
  is 40–78 sections long;
* **byte 1 = byte 2 = the finish/start section index** (`$1CA1B`/`$1CA1C`) — a
  duplicated value, used to wrap the lap (`$5B10C`: start from `$1CA1C+1`);
* **bytes 3–4 = the initial position seed** (`$1CA1D`/`$1CA1E`; `$1CA1E` is later
  overwritten with the total track length).

`extract/cmd/tracks` prints all of this (`sections=… finishIdx=…(dup=true) seed=…`).

## 3. The section stream (RLE) and the spine builder

After the header, the body of `$5AE46` (`$5AED0..$5B106`) walks the byte stream once,
emitting one **section** per iteration into a set of parallel arrays indexed by the
section number `d1`:

| array | per-section meaning |
|-------|---------------------|
| `$1C5EC[]` | type / flags byte |
| `$1C588[]` | parameter 1 |
| `$1C4C0[]` | parameter 2 |
| `$1C524[]` | combined attribute (`param & $7F | $1BB1B`) |
| `$1C650[]` / `$1C718[]` (word) | the section node's **X / Z** coordinate |
| `$1C7E0[]` (word) | accumulated distance along the track (`<<5`) |

**Encoding.** Each section normally costs a type byte plus 1–2 parameter bytes. Two
mechanisms compress the common case of many similar sections in a row:

* **Run-length.** If a type byte's **low nibble is `$F`**, its high nibble is a
  *run count* (`$1BB48`): the following `n` sections repeat the saved type
  (`$1BC55`) with their parameter stepped by the delta helper `$5AE0C` — which adds
  or subtracts either `$10` or `$1` depending on flag bits in `$1BB1A`. This is what
  encodes a constant curve or a constant gradient as one marker plus a step.
* **Type-nibble ≥ `$C`.** These section types take their second parameter from the
  small tables `$5B2B8`/`$5B2BA` instead of from the stream (the standard ramp/jump
  piece shapes).

**Spine layout.** As it emits each section, the loop also places it in 3-D. The
running position is `(X,Z) = ($1BBF6,$1BBF8)`, advanced per section by a helper
`$5B2D2`/`$5B2C8` that — notably — **reuses the very same handle-decode formula as
the track-pointer table** (`ROL.w #8`, `−$B100`, `+$1EF82` on a 16-bit handle in
`$1BC8C`) to follow a *section-shape* sub-table inside the track data, indexed by the
section type. So section "types" are references to reusable extruded-piece shapes,
and the per-section dx/dz come from those shape profiles rather than from a sine
table. The node coordinates are stored in `$1C650[]`/`$1C718[]`; the cumulative
distance in `$1C7E0[]`; and the grand total length in `$1CA1E` (`$1BC2A << 5`).

**Trailer.** After the sections, the stream carries a fixed 6-byte block (`$1CA2A`),
then two optional object lists guarded by counts: `$1CA2E` pairs into
`$1C8A8[]`/`$1C8C8[]` and `$1CA2F` entries into a further array — the trackside
scenery / markers.

So a track decodes to: *a header (count, finish index, seed) → an RLE section stream
→ a scenery trailer*, with each section a `(type, p1, p2)` triple where the type
indexes a shared piece-shape table that supplies the geometry.

## 4. A verified Go decoder

`extract/cmd/sections` reimplements the `$5AED0..$5B106` loop exactly — the
run-length marker, the `$5AE0C` delta step, the `≥$C` table branch and the
`bit5`/`bit6`/`bit7` flag-driven reads — and then checks itself against the data.
The whole track stream is:

```
6 (header)  +  section stream  +  6 (trailer)  +  2*trailer[4] + trailer[5] (scenery)
```

so a correct parse must consume a track's bytes **exactly**. It does, for all seven
length-bounded tracks:

```
LITTLE RAMP     44 sections  used 124/124   STEPPING STONES 56  145/145
HUMP BACK       53 sections  used 145/145   BIG RAMP        44  142/142
SKI JUMP        40 sections  used 145/145   DRAW BRIDGE     78  213/213
HIGH JUMP       52 sections  used 142/142   ROLLER COASTER  78  used 192 (last)
```

(ROLLER COASTER, the last record, is 192 bytes — the raw slice in `cmd/tracks` runs
on into padding; the parser gives its true length.) The byte-exact match across every
track confirms the read pattern is right. As a spot check, LITTLE RAMP's first ten
sections are a single `$A0` run: a fresh `type $A0, p1 $CF, p2 $6A` section, then one
marker byte `$9F` (run = 9) followed by nine `p2` bytes, with `p1` stepping `−$10`
each section — exactly the bytes on disk.

What remains for the circuits is geometric, not structural — covered in §5.

## 5. The spine geometry, and re-drawing the circuits

The section loop also lays the track out in plan view as it parses. Per section,
given its decoded `(type, p1, p2, attr)`, the per-section setup `$5FE56`/`$5FEFA`
resolves the geometry from three data tables — all reached by the same
byte-swap/`−$B100`/`+$1EF82` handle decode as the track-pointer table:

* **`$1EF82 + ((type & $F) << 8)`** → a per-type *piece-shape* table. Its layout
  yields the section's **segment length** (`$1BB6A = $1BB97/2 − 1`, where
  `$1BB97 = shape[shape[0]]`) and related step counts.
* **`$1EFA2[p2*2]`** → the **X-offset shape** handle (`$1BC8C`); **`$1EFA2[attr*2]`**
  → the **Z-offset shape** handle (`$1BC90`). `p2` also sets the sign mode `$1BB79`.

The shape readers `$5B2D2` (X) and `$5B2C8` (Z) return the piece's offset at a given
distance along it, and the loop walks the spine by

```
node.X = cur.X − shapeX(0) ;  cur.X = node.X + shapeX(len)
node.Z = cur.Z − shapeZ(0) ;  cur.Z = node.Z + shapeZ(len)
```

storing each node in `$1C650[]` (X) / `$1C718[]` (Z) and accumulating the heading in
`$1BC2A`. So a straight piece (where `p2 == attr`, common in the runs) advances X and
Z equally — the world axes are the 45°-rotated `(x±z)` pair — and curves bend them
apart.

**The Go reimplementation (`extract/track`).** `package track` walks this spine
purely in Go, reading only the game's data tables — it does *not* run any 68000 code.
The handle math has two byte-width subtleties worth recording (both were initial
mis-reads, caught because the format is shared with the 6502/Z80 ports and so cannot
depend on any 68000 quirk):

* the per-type shape table is a **word table at `$1EF82 + nib*2`** (an earlier
  reading saw a phantom `ASL.l #7` — actually the low word of the
  `MOVEA.l #$1EF82` immediately before it; there is no shift);
* the X/Z offset-shape handle index is `ASL.b #1` on the param, i.e.
  **`(param*2) & $FF`** — a byte multiply that wraps, not a 16-bit one.

`extract/cmd/spineoracle` executes the real `$5AE46` on the `tools/m68k` core (it
takes only `d1` = track id and self-initialises) and reads the `$1C650`/`$1C718`
arrays back out; `extract/cmd/spineverify` checks `package track` against it.
**All eight tracks match coordinate-exact** — the Go decoder is the source of truth,
the oracle only confirms it.

**These arrays are not (yet) the world centre-line.** Plotted directly, `$1C650`/
`$1C718` form thin slivers (plan aspect 6:1 to 23:1 — Ski Jump is almost a straight
line), which are *not* the shapes of the real circuits. Tracing the renderer shows
why: the per-section vertex builder `$5C0AA` returns `shape_offset + $1BC0E/$1BC10`,
where `$1BC0E/$1BC10` are exactly `$1C650[i]`/`$1C718[i]` — so these arrays are the
section *base positions*, but the renderer (`$5A1A6`) also reads **two** offset shapes
per vertex (`a4` from `p2`, `a5` from `attr`, alternating), and rotates everything by a
per-section **heading** (`$1BBF2 = $1BC30 − $1BC4A`, with `$1BC4A = type & $C0` the
section's quadrant and `p1` the intra-quadrant fine direction via `$5FF94`). Strong
evidence that one of `$1C650`/`$1C718` is the **vertical** (the `a4`/`a5` pair reads as
horizontal/vertical cross-sections), so the raw plot is closer to a side profile than
a plan.

So `$1C650/$1C718` are per-piece *extents*, not the plan. The plan footprint is held
elsewhere — and reading the renderer, not guessing, finds it (§6).

## 6. The plan footprint (the 16×16 track grid)

The engine renders the track by walking from the player's section outward, and to find
the next section it consults a lookup table. `$5FE04` is that lookup: it forms a byte
index `(y << 4) | x` from a grid coordinate and reads the **256-entry table `$1C280`**,
returning the section sitting in that cell (or `$FF` for empty). That table is built by
`$64304`:

```
for each section i:  $1C280[ p1[i] ] = i        ; p1 indexes the 16×16 grid
```

So **`p1` is the section's cell on a 16×16 grid** — its low nibble the grid X, its high
nibble the grid Y. That is the track's plan footprint, read straight from the data:

```
node[i].planX = p1 & $0F
node[i].planY = p1 >> 4
```

No heading accumulation, no quadrant transforms, no fitting — and it is unambiguous:
**every consecutive section is an adjacent grid cell** (0 non-adjacent steps across all
eight tracks), so the plans are smooth closed loops with no sharp corners and no
spurious self-crossings. The footprints are the real circuits — *Little Ramp* a
diamond, *Stepping Stones* a rectangle, and *Draw Bridge* a loop with the two inner
prongs that are literally its drawbridge:

```
.##..........##.        .##....########.
#..#........#..#        #..#..#........#
#..#........#..#        #..#..#........#   Roller Coaster (right):
#..#........S..#        #..#..#........S   an outer lap with an
#..#........#..#  ...   #..#..#........#   inner peninsula the
#...########...#        #...##.........#   track doubles back along
#..............#        #..............#
.##############.        .##############.
  Draw Bridge             Roller Coaster
```

`package track` sets `PlanX/PlanY` from `p1`; `cmd/trackjson` exports them; the viewer
draws them. (An earlier attempt reconstructed the plan by accumulating a per-section
heading from `p1`'s nibbles and a `type` quadrant — it was wrong: it forced closure with
a fitted quadrant table and produced sharp corners and false crossings. Reading
`$5FE04`/`$64304` replaced it with the actual grid.)

## 7. Banking and elevation

The game's pre-race **preview** (the "steer to rotate view" screen) draws each circuit
as an *elevated ribbon on support columns*, which makes the two vertical quantities
plain: the ribbon **banks** on the curves, and it **rises and falls** (the ramps, the
Roller-Coaster hills) on columns of varying height. Both come, cleanly, from the
loader's two per-section extent arrays.

Tracing the two vertex builders settles which is which:

* `$5C6C4` reads the section's per-type **piece-shape** and rotates *both* its
  components by the section's orientation (its four quadrant cases are a 2-D rotation).
  A quantity that rotates with heading is **in-plane** — so this builds the plan
  footprint, not the height.
* `$5C0AA` reads the two extent arrays `$1C650` (from `p2`) and `$1C718` (from `attr`),
  adds them **un-rotated** (and `>>5`), and *those* are the heights of the ribbon's
  **left and right rails** — a vertical doesn't turn with the heading.

So the two rail heights give both quantities directly:

```
elevation = ($1C650 + $1C718) / 2      (the surface height)
banking   =  $1C650 − $1C718           (the camber: rail-height difference)
```

On a straight the section's `p2 == attr`, the two rails are level, banking is zero;
on a curve they diverge and the road tilts. The elevation profile **matches the
preview screenshots**: Big Ramp is flat with a single ramp, Stepping Stones gently
undulates, Roller Coaster is a run of hills — and every circuit's height **closes**
over the lap (Hump Back exactly). (An earlier draft mis-used `$1C650 − $1C718` as the
height; its bumps fell on the corners, where roads bank — caught and corrected.)

**Smooth or stepped — solved, exactly.** A long search for an explicit "step here" flag
(`$65D3C`, the `type` bits, the `p2` sign, `attr`, `a0[1]`, the height magnitude) found
nothing — *because there is no such flag*. The surface is not one height per section at
all. `$5C0AA` is a **per-rung** builder: it is called for vertex index `d1 = 0,1,2,…`
across the section, and each call adds the section's base accumulator (`$1C650` for the
left rail, `$1C718` for the right) to a **profile value** read from the `p2`/`attr`
cross-section shape, then `>>5`. So every section carries a short array of rail heights
*along its length* — the in-game surface itself:

```
rung count   = shape[shape[0]] of the per-TYPE piece-shape   ($1BB97; counts both rails)
left rail[k]  = ($1C650 + a4-profile[k]) >> 5                  (even vertex 2k,  a4 from p2)
right rail[k] = ($1C718 + a5-profile[k]) >> 5                  (odd  vertex 2k+1, a5 from attr)
read mode     = nibble-packed if p2 ≥ 0, two bytes/entry if p2 < 0   ($1BB79)
```

The step-vs-slope distinction is simply **what those profile values do**. A smooth run is
a drivable slope; a sudden jump in consecutive rungs is a hard edge. Big Ramp's takeoff
section reads `[80 85 90 95 100 105 110 115 35]` — a clean climb to the lip (115) then a
hard drop into the gap (35); the next piece resumes at 35 and the landing ramp rises again
`[35 105 100 …]`. That is **three jumps, six steep lips**, present in the data, with no
heuristic. The Stepping Stones read `[118 36 36 …]` (a flat stone with a square gap), the
Ski Jump's launch is a `704 → 389` drop mid-section, and the Roller Coaster is a long
continuous `281 → 925` climb. No magnitude rule, no extremum test — the profile *is* the
surface.

This is verified the strict way: `cmd/geomoracle` runs the engine's own `$5FE56`+`$5C0AA`
on the m68k core and the Go reimplementation in `package track` (`railHeight`/
`railProfile`) reproduces every rung of every section of all eight tracks **coordinate-
exact**.

### The plan outline (`$5C6C4`)

The other half of each piece is its **plan footprint** — the `(x,z)` shape of the rung
strip seen from above. The draw loop (`$65CDC`) reads it with `$5C6C4`, which fetches two
consecutive 16-bit little-endian values at byte offset `d2` in the per-type piece-shape
and adds the section base, rotating by quadrant. The vertex pairs are a flat array:

```
plan vertex k = LE16 pair at  a0[0] + 7 + 4k     (even k = left rail, odd = right rail)
                d2 = 2*d1 + $1BB91 ,  $1BB91 = a0[0]+7        (the draw loop's index math)
```

Read out, these are exactly the piece's local outline (`+z` forward, `x` lateral): the
straight piece (`nib 0`) is two parallel rails of width 384 marching `z = 0,256,…,2048`;
the curve pieces (`nib 6`/`7`) are mirror-image arcs; ramps carry their own profiles. The
left/right rails are independent, so the rail **width** (and any asymmetry through a bend)
is exact, not a nominal constant. `cmd/planoracle` drives `$5C6C4` in the canonical frame
(zero base, zero quadrant) and confirms `package track`'s `planProfile` matches it
**exactly** on every vertex of all eight tracks.

The renderer itself only ever builds *view-space* points (it walks outward from the car,
`$1BB22 = dir*8 + cameraPos`), so there is no clean absolute top-down accumulator to lift.
But there doesn't need to be: the section's **absolute** placement is the verified 16×16
grid (`p1`), exactly as the engine uses it (`$5FE04`), and that closes every circuit. So
the viewer similarity-fits each section's exact local outline onto its grid-anchor segment
— mapping the outline's local centreline start→`A[i]` and end→`A[i+1]` with a rotation and
uniform scale (one complex multiply). Straights stay straight, curve pieces carry their
real arc, consecutive sections share the anchor (the ribbon is continuous), and every rung
is lifted by its exact left/right rail heights. `cmd/trackjson` exports both the height
profiles and the plan outlines; the viewer needs **no spline and no nominal width** — the
geometry is entirely the engine's own. **Part IV is complete: the track geometry —
footprint, surface and camber — is decoded purely from the disk in Go and verified
coordinate-exact against the original.**

*Part V — the physics.*

---

# Part V — The physics simulation

*The headline goal — done.* The car is simulated as a sprung rigid body, not a point,
which is what gave the game its reputation: the chassis pitches and rolls on its
suspension, the wheels gain and lose contact over crests and on landings, too hard a
landing damages the car, and airtime/handling depend on speed and the track gradient.
The **entire** per-frame simulation has been reverse-engineered from the 68000 code and
reimplemented in Go (`extract/physics`), **verified coordinate-exact against the original
frame by frame** on the `tools/m68k` core (`cmd/physverify`): every sub-routine matches on
3000 random states each, and the whole frame matches bit-for-bit over 60 consecutive
frames on three tracks. The reimplementation operates directly on the game's 24-bit memory
image, so it can be checked address-for-address.

### 1. The car state and the frame

One physics frame is `$6185C`, called from the race loop (`$5D42C`) just before the
renderer. The model is **semi-implicit Euler in fixed point** with a `0.93` (`= $EE/256`)
damping factor applied at *both* integration stages — that constant is the drag that keeps
the sim stable. The car-state block lives at `$1BCxx`:

```
position  (16.16)  X $1BCD8   Y/height $1BCDC (clamped <= $3E8)   Z $1BCE0
angles    (16-bit) roll $1BCE4   yaw $1BCE6   pitch $1BCE8
velocity           $1BCEA / $1BCEC / $1BCEE
angular momentum   roll $1BCF0   pitch $1BCF2   yaw $1BCF4   (body frame)
```

Forces are summed in the **body frame**, rotated into the world, and integrated. The frame
pipeline is, in order: build the orientation matrix (`$61368`); compute the wheel contact
offsets (`$618CE`); **sample the track surface** under each wheel (`$5C1D0`); compute the
chassis contact-point heights (`$61B70`); rotate velocity into the body frame (`$6158C`)
and gravity into it (`$615E6`); run the **suspension** (`$61BCC`); and — only when grounded
(`$1BB72`) — the **drive/tyre/steer** block (`$620B8`, `$61012`, `$61618`, `$621F4`,
`$62138`, `$61B26`, `$61672`); then integrate force→velocity (`$61ADC`) and
velocity→position (`$61950`).

### 2. Frame transforms

`$61368` builds the 3×3 chassis orientation matrix at `$1C230` from the three Euler angles
(`$64D08` = sin, `$64D10` = cos, via a quarter-wave table at `$1CA42`): it seeds the slots
with sin/cos of each angle, cascades the `MULS/SWAP` products into the composite rotation,
and forms the cross terms. Everything is then shuttled through it: world velocity → body
(`$6158C`), body force → world (`$61618`), body torque → world angular rate (`$61672`), all
driven by a small index table at `$1EC46` that selects which matrix slot each term uses.
**Gravity** enters via `$615E6`: a fixed world-down vector (magnitude `$13D` = 317) is
re-expressed in the tilted body frame each tick, so it always pulls world-down whatever the
car's attitude.

### 3. The suspension — and the surface

The car has a **three-point suspension** (`$61BCC`). Each point computes a spring
compression

```
compression = trackSurfaceY - chassisContactY - restLength      (clamped to [-$300, $1400])
force       = compression + 1.078 * (compression - prevCompression)   ($6180E: spring + damper)
```

The chassis contact heights (`$1BC94/98/9C`) come from the car height tilted by sin(roll)
and sin(pitch) (`$61B70`). The track surface heights (`$1BCA4/A8/AC`) are the **direct
coupling to Part IV**: `$5C1D0` locates the car's section, computes where each wheel sits
across the rung strip, samples the four surrounding rung-corner heights with the *same*
`$5C0AA` builder that produces the rail heights in Part IV, and bilinearly interpolates them
(`$5C554`) — so the springs ride the exact decoded track surface, and a wheel running off
the lateral edge (`$1BB65`) is detected and tapered (`$5C808`).

The three spring forces combine (`$61F42`) into a **net lift** (`$1BD38`, the average), a
**roll torque** (`3 * (left - right)`, clamped) and a **pitch torque** (front − rear); the
**on-ground flag** `$1BB7E` is set if any spring is engaged. The net lift is projected onto
the body axes by the road slope (`$622DC`) into the tyre **loads** `$1BD40/42/44`.

**Damage.** When a spring force exceeds a tolerance (`$1BB01 << 8 + $700`) and stays there
for `$63CE2` frames, damage accumulates into `$1BB4F/50/51` (0–255), with a separate
bottoming counter (`$1BB7D`) for hard slams — the "land too hard and you wreck the car"
mechanic, reproduced exactly.

### 4. Drive, grip and steering

The longitudinal drive force (`$620B8`) comes from the throttle/gear, decays on wheelspin,
and is clamped to the available **grip** (`$621DA`): grip = tyre-load × 2 *only when
grounded* — zero in the air, and load-sensitive on the ground. The lateral tyre force
(`$6217A`) is likewise grip-limited, setting a slide flag (`$1BBC1`) when the demand
exceeds grip — a friction-circle model. A velocity **drag** (`$621F4`) opposes motion:
stiff when grounded and gripping, speed-proportional over a deadzone when rolling free.

`$61012` is the **track-following auto-steer**: it measures the car's heading error against
the section's centreline and nudges the heading (`$1BCE6`) to follow the track, then
computes a pitch-stabilisation torque (`$1BCFE`).

### 5. A copy-protection trap inside the physics

`$61012` carries a piece of the disk **copy protection**. At `$611E8` it compares
`mem[$64AEC]` against a constant and, on mismatch, **zeroes the pitch-stabilisation
torque** — subtly degrading the handling. Both operands are obfuscated specifically so they
can't be grepped: the address is built as `$79360 - $14874`, the magic value as
`$667B379F + $36729563 = $9CEDCD02`. The value at `$64AEC` is not a settings byte — it is
written (at `$5CEFA`) by a routine (`$5CEB4`) that reads the **physical disk hardware**
(CIA-B `$BFD100` drive control, `DMACONR`/`DMACON` disk DMA), i.e. it is derived from the
disk's protection tracks. So on a genuine disk the protection writes `$9CEDCD02` and the
check passes; our extracted blob holds the unpatched `$2F76EA80`, so it fails. The
reimplementation reproduces the check exactly (matching the oracle on the same image); for
a playable sim, "inserting the genuine disk" is just storing the expected value at `$64AEC`.

### 6. Verification

`cmd/physverify` runs each engine routine and its Go twin from identical random snapshots
and compares the state — `Sin/Cos`, the integrator, the matrix and transforms, the
suspension and damage, the drive/tyre/drag, the load projection, the per-section setup, the
**surface sample over loaded tracks**, the auto-steer (both protection branches), and
finally the whole `$6185C` frame in lockstep for 60 frames. **All match exactly.** The
oracle repeatedly earned its keep — it caught a swapped sin/cos quarter-mirror, a signed
`CMPI.b #5` whose `NEG.b $80` stays `$80`, an unsigned shift on a negative handle that
mis-decoded only curve pieces, and a branch where a large heading error uses the *excess*
as the correction — all subtle enough to pass a careful read.

**Part V is complete: the full rigid-body car physics is decoded from the disk in Go and
verified coordinate-exact against the original, frame by frame.** Combined with Part IV, a
car can now be driven over the decoded tracks.

---

# Appendix A — Toolchain and reproduction

All work is reproducible from the image with the shared `tools/` module:

```sh
# Inspect the boot block / a raw region (disk offset maps 1:1 to bytes)
go run stupidcoder.com/tools/cmd/dis68k -base 0 -skip 12 "Stunt Car Racer (Amiga)/Stunt Car Racer.adf"

# Slice the disk into loader.bin, game.bin and the decrypted game.dec.bin
cd "Stunt Car Racer (Amiga)/extract" && go run ./cmd/extract "../Stunt Car Racer.adf"

# Dump the eight track byte streams (reimplements the $5AE46 pointer math)
go run ./cmd/tracks ../extracted/game.dec.bin    # -> extracted/tracks/<id>_<name>.bin

# Decode + verify the section streams (byte-exact against each track's length)
go run ./cmd/sections ../extracted/game.dec.bin [-v]

# Run the real $5AE46 on the m68k core and read out the spine (X,Z) per section
go run ./cmd/spineoracle ../extracted/game.dec.bin [trackid]   # -> extracted/spine_<id>.csv

# Verify the pure-Go spine (package track) against the oracle, coordinate-exact
go run ./cmd/spineverify ../extracted/game.dec.bin

# Export the eight decoded circuits to JSON for the web track viewer
go run ./cmd/trackjson ../extracted/game.dec.bin   # -> site/public/stuntcar/tracks.json

# Verify the Go car physics (package physics) against the engine on the m68k core:
# every sub-routine + the full $6185C frame in lockstep for 60 frames (all exact)
go run ./cmd/physverify ../extracted/game.dec.bin

# Disassemble / trace the engine. Use game.dec.bin for anything in $F4B8..$1AA4A.
go run stupidcoder.com/tools/cmd/dis68k     -base 0xE700 -start <addr> -end <addr> extracted/game.dec.bin
go run stupidcoder.com/tools/cmd/codetrace68k -base 0xE700 -entry <addr>            extracted/game.dec.bin
```

Dynamic verification uses the instruction-level 68000 core in `tools/m68k`
(`m68k.CPU` over a `Bus`), the same way the other games are checked.

The disk image is not committed (it is a copyrighted game); its size and MD5
are recorded in the repository [README](../README.md#image-files) so the exact
copy can be verified.
