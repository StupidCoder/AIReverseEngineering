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
**Status: Parts I–III done (disk format, loader, engine architecture); Part IV under
way (the eight tracks located and extracted, section grammar in progress); Part V
(physics) to follow.**

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
  - [2. The record header](#2-the-record-header-in-progress)
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

## 2. The record header (in progress)

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

Byte 0 looks like a count/length, bytes 1–2 a duplicated start/limit parameter, and
bytes 3–7 the initial position/orientation seed the simulation starts from. The
section grammar that follows (consumed by the `$5AE46` body and the post-load passes
`$64304`/`$65BEC`) is the next step: decode it into a list of typed sections
(straight / banked curve / hump / ramp / jump / broken bridge) with their
length/curvature/gradient, then reimplement that decoder in Go and re-draw each
circuit.

*Section grammar: in progress.*

---

# Part V — The physics simulation

*The headline goal.* The car is simulated as a sprung rigid body, not a point —
which is what gave the game its reputation: the chassis pitches and rolls on its
suspension, the wheels gain and lose contact over crests and on landings, too
hard a landing damages the car, and airtime/handling depend on speed and the
track gradient. The aim is to recover, from the 68000 code:

1. the **car state** (position, orientation, linear/angular velocity, per-wheel
   suspension state) and its fixed-point representation;
2. the **integrator** — per-frame forces and the update step: drive/brake,
   gravity, suspension spring/damper, wheel–ground contact and friction, and the
   damage/“boost” model;
3. a faithful **Go reimplementation** of that update, verified against the 68000
   (the `tools/m68k` core as an oracle) so that, combined with Part IV, a track
   can actually be *driven* in a reimplementation.

*Simulation: to be reverse-engineered.*

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

# Disassemble / trace the engine. Use game.dec.bin for anything in $F4B8..$1AA4A.
go run stupidcoder.com/tools/cmd/dis68k     -base 0xE700 -start <addr> -end <addr> extracted/game.dec.bin
go run stupidcoder.com/tools/cmd/codetrace68k -base 0xE700 -entry <addr>            extracted/game.dec.bin
```

Dynamic verification uses the instruction-level 68000 core in `tools/m68k`
(`m68k.CPU` over a `Bus`), the same way the other games are checked.

The disk image is not committed (it is a copyrighted game); its size and MD5
are recorded in the repository [README](../README.md#image-files) so the exact
copy can be verified.
