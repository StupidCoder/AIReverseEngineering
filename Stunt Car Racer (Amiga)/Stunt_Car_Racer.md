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
**Status: Part I is an initial recon; Parts II–V are stubs / in progress.**

---

## Contents

- [Part I — The disk image](#part-i--the-disk-image)
  - [1. The ADF container](#1-the-adf-container)
  - [2. Not an AmigaDOS filesystem](#2-not-an-amigados-filesystem)
- [Part II — Boot chain and loader](#part-ii--boot-chain-and-loader)
  - [1. The boot block](#1-the-boot-block)
  - [2. The custom track loader](#2-the-custom-track-loader)
- [Part III — Game program architecture](#part-iii--game-program-architecture)
- [Part IV — The tracks (vector circuits)](#part-iv--the-tracks-vector-circuits)
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

*To be analysed.* The `$9800`-byte loader at disk offset `$2C00` is the entry
point for everything else: it presumably sets up the hardware, loads the main
engine and the track data from later regions of the disk (possibly
decompressed), and hands off to the game. Disassembling it (`tools/cmd/dis68k`
/ `codetrace68k`, base = its load address) is the next step, and will establish
the on-disk map: where the engine lives, where the track geometry lives, and
any packing used.

---

# Part III — Game program architecture

*To be analysed.* The 68000 engine: startup, the Copper/blitter/interrupt setup
for the vector display (the filled-polygon horizon, track ribbon and car), the
main loop's structure (render ↔ simulate), and the memory map. Expected to be a
single-buffered or double-buffered filled-vector renderer driving Paula/sound
and reading the joystick.

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

*Format: to be reverse-engineered.*

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

# Recursive-descent trace of a located code blob (once the loader's base is known)
go run stupidcoder.com/tools/cmd/codetrace68k -base <addr> -entry <addr> blob.bin
```

Dynamic verification uses the instruction-level 68000 core in `tools/m68k`
(`m68k.CPU` over a `Bus`), the same way the other games are checked.

The disk image is not committed (it is a copyrighted game); its size and MD5
are recorded in the repository [README](../README.md#image-files) so the exact
copy can be verified.
