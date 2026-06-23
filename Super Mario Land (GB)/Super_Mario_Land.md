# Super Mario Land (Game Boy) — cartridge format and game analysis

A reverse-engineering reference for `Super Mario Land (World).gb`, the 1989 Game
Boy launch title. This is the first **Game Boy** title in this repository, and the
first **Sharp LR35902** ("GBZ80" / SM83) CPU — a relative of the Z80 but, as Part I
shows, *not* the same chip. The writeup follows the same shape as the C64, Amiga and
Game Gear games, in reading order:

* **Part I** — the cartridge image: the flat ROM dump, the Game Boy memory map, the
  MBC1 bank-switching mapper, the cartridge header, and the CPU vectors;
* **Part II** — boot and initialization: the LR35902 reset sequence, the LCD/audio
  and RAM setup, and the path to the main loop;
* **Part III** — engine architecture: the main loop, the VBlank/timer interrupt
  handlers, the RAM layout and how banked resources are reached;
* **Part IV** — graphics and data formats: the 2bpp tile, tilemap, OAM-sprite and
  palette encodings, and the level and object data;
* **Part V** — game mechanics: Mario's physics, the objects, the worlds, scoring
  and progression.
* **Appendix** — toolchain and reproduction.

Methods: purely static analysis of the ROM image. **Note the toolchain gap:** the
shared `tools/z80` decoder does *not* apply here — the Game Boy CPU is the Sharp
LR35902, which shares the Z80's register names and much of its opcode map but drops
the `IX`/`IY` index registers, the alternate register set and the `IN`/`OUT` ports,
and adds Game-Boy-specific opcodes (`LDH`, `LD (C),A`, `LD (a16),A`, `STOP`, `SWAP`,
`ADD SP,e`, `LD HL,SP+e`). Decoding the engine in Parts II+ needs a new `tools`
decoder/CPU core for the LR35902 (provisionally `tools/sm83`); the opcode decodes in
this part were done by hand and are flagged as such. All addresses are CPU addresses
(16-bit, `$0000`–`$FFFF`) unless a *file offset* is called out; bytes are 8-bit.
Part I is complete; Parts II–V are stubbed.

---

## Contents

- [Part I — The cartridge image](#part-i--the-cartridge-image)
  - [1. The ROM dump](#1-the-rom-dump)
  - [2. The LR35902 address space and MBC1 bank switching](#2-the-lr35902-address-space-and-mbc1-bank-switching)
  - [3. The memory map](#3-the-memory-map)
  - [4. The cartridge header (`$0100`–`$014F`)](#4-the-cartridge-header-0100014f)
  - [5. The CPU vectors](#5-the-cpu-vectors)
  - [6. What's in each bank](#6-whats-in-each-bank)
- [Part II — Boot and initialization](#part-ii--boot-and-initialization)
- [Part III — Engine architecture](#part-iii--engine-architecture)
- [Part IV — Graphics and data formats](#part-iv--graphics-and-data-formats)
- [Part V — Game mechanics](#part-v--game-mechanics)
- [Appendix A — Toolchain and reproduction](#appendix-a--toolchain-and-reproduction)

---

# Part I — The cartridge image

A cartridge is the simplest image format in this repository: there is **no
container, no filesystem and no loader** — unlike a C64 tape (a pulse stream you have
to decode) or an Amiga disk (an AmigaDOS filesystem you have to walk). The `.gb`
file is a verbatim copy of the cartridge's mask-ROM chip: byte *N* of the file is
exactly the byte the CPU reads from the chip at ROM offset *N*. So there is nothing
to *extract*. The structure that matters is the **memory map** the console imposes
(the ROM is bigger than the CPU can address at once), a small **header** Nintendo
stamps near the front, and the fixed **CPU vectors** at the very bottom.

## 1. The ROM dump

The image is **65,536 bytes = 64 KB = 512 Kbit**, an exact power of two. It carries
**no copier header** (some circulating dumps prepend a few hundred bytes of metadata;
this one does not — the size is a clean power of two and the header lands exactly at
its canonical offset `$0100`). The exact copy this analysis is based on is pinned by
size and MD5 in the repository [README](../README.md#image-files).

Two integrity fields in the header confirm the dump is intact and untampered, and
both **verify** for this file:

- the **header checksum** at `$014D` = `$9E`. The Game Boy boot ROM computes
  `x = 0; for a in $0134..$014C: x = x − ROM[a] − 1` and refuses to run the cartridge
  unless the low byte of `x` equals `ROM[$014D]`. Computed `$9E`, stored `$9E` ✓.
- the **global checksum** at `$014E`–`$014F` = `$416B` (big-endian) = the 16-bit sum
  of every ROM byte except those two. Computed `$416B`, stored `$416B` ✓. (The boot
  ROM does *not* check this one; it is informational.)

The 48-byte **Nintendo logo** at `$0104`–`$0133` is also byte-for-byte the canonical
bitmap (`CE ED 66 66 …  BB B9 33 3E`). This is not decoration: the boot ROM scrolls
it down the screen and compares it against its own internal copy, and **locks up if
it differs** — the original anti-piracy / trademark gate. Its presence here, exact,
is a second confirmation the front of the image is genuine and unshifted.

That is the whole "format". Everything else in this part is about how the **console**
sees those 64 KB.

## 2. The LR35902 address space and MBC1 bank switching

The Game Boy's CPU is a **Sharp LR35902** with a **16-bit address bus**, so it can
only address **64 KB at a time**. This cartridge holds exactly 64 KB — so, unusually,
the whole ROM *could* fit in the address space at once. It does not: Nintendo still
fitted a **mapper chip** so the cartridge layout matches the larger games to come.

The chip is the **MBC1** (Memory Bank Controller 1; cartridge type `$01` in the
header, [§4](#4-the-cartridge-header-0100014f)). It divides the ROM into **4 banks of
16 KB** (bank *b* = file offset `b × $4000`) and maps two of them into the CPU's low
32 KB:

| CPU range | Size | Contents |
|---|---:|---|
| `$0000`–`$3FFF` | 16 KB | ROM **bank 0** — fixed, never paged |
| `$4000`–`$7FFF` | 16 KB | ROM **bank 1–3** — switchable |

The lower bank is hard-wired to bank 0 (so the header, vectors and the mapper-setup
code are always reachable); the upper window is whatever bank the program last
selected. Unlike the Game Gear's mapper (whose registers live in RAM), the MBC1 is
programmed by **writing to the ROM address space itself** — the writes never reach the
mask ROM, they are intercepted by the mapper. The control regions are:

| Write range | Register | Effect |
|---|---|---|
| `$0000`–`$1FFF` | RAM enable | `$0A` enables cartridge RAM (this cart has none) |
| `$2000`–`$3FFF` | ROM bank | low 5 bits select the bank in `$4000`–`$7FFF` |
| `$4000`–`$5FFF` | RAM bank / hi ROM | upper 2 bank bits, or RAM bank (unused here) |
| `$6000`–`$7FFF` | mode select | ROM-mode vs RAM-mode banking (unused here) |

With only four banks, just the low **two** bits of the bank register matter, and the
classic **MBC1 quirk** applies: writing `$00` to `$2000`–`$3FFF` selects bank **1**,
not bank 0 (the mapper translates a requested bank 0 to 1 for the upper window), so
the switchable window reaches banks **1, 2 and 3** and bank 0 is only ever the fixed
lower window. The game uses this constantly; for example the timer interrupt
([§5](#5-the-cpu-vectors)) does `LD A,$03 ; LD ($2000),A` to page **bank 3** into
`$4000`–`$7FFF`, runs a routine there, then restores the previous bank.

For reverse engineering this means a disassembler must be told *which bank* occupies
`$4000`–`$7FFF`; following a call *across* a bank switch is a higher-level concern for
Part II onward (and needs the LR35902 decoder noted in the intro).

## 3. The memory map

Putting the mapper together with the console's RAM, video and I/O, the LR35902 sees a
single flat 64 KB space (there are **no** I/O ports — everything is memory-mapped,
which is one of the LR35902's departures from the Z80):

| CPU range | Size | Contents |
|---|---:|---|
| `$0000`–`$3FFF` | 16 KB | ROM **bank 0** (fixed; header + vectors + core code) |
| `$4000`–`$7FFF` | 16 KB | ROM **bank 1–3** (switchable; see §2) |
| `$8000`–`$9FFF` | 8 KB | **VRAM** — tile data (`$8000`–`$97FF`) + two BG maps (`$9800`/`$9C00`) |
| `$A000`–`$BFFF` | 8 KB | cartridge RAM — **absent** on this cart (open bus) |
| `$C000`–`$DFFF` | 8 KB | **work RAM** (WRAM) |
| `$E000`–`$FDFF` | ~7.5 KB | **echo** of `$C000`–`$DDFF` (a hardware mirror) |
| `$FE00`–`$FE9F` | 160 B | **OAM** — 40 sprite entries × 4 bytes |
| `$FEA0`–`$FEFF` | 96 B | unusable |
| `$FF00`–`$FF7F` | 128 B | **I/O registers** (LCD, timer, audio, joypad, DMA, …) |
| `$FF80`–`$FFFE` | 127 B | **HRAM** (high RAM; usable while OAM-DMA runs) |
| `$FFFF` | 1 B | **IE** — interrupt-enable register |

The I/O block at `$FF00`–`$FF7F` is where the LCD controller (`$FF40` `LCDC`, `$FF41`
`STAT`, `$FF42/43` scroll, `$FF47` BG palette), the timer (`$FF05`–`$FF07`), the four
sound channels (`$FF10`–`$FF26`), the joypad (`$FF00`) and the OAM-DMA trigger
(`$FF46`) live; the reset code writes these directly (Part II). The two interrupt
latches `IF` (`$FF0F`) and `IE` (`$FFFF`) gate the five interrupt sources whose
vectors are described in §5.

## 4. The cartridge header (`$0100`–`$014F`)

Every Game Boy ROM carries an 80-byte header at a fixed offset. Decoded for this
cartridge:

```
$0100  00 C3 50 01                                  entry point: NOP ; JP $0150
$0104  CE ED 66 66 … BB B9 33 3E   (48 bytes)       Nintendo logo (canonical)
$0134  53 55 50 45 52 20 4D 41 52 49 4F 4C 41 4E 44 00   title "SUPER MARIOLAND"
$0144  00 00                                        new licensee code (unused; see $014B)
$0146  00                                           SGB flag — not Super Game Boy enhanced
$0147  01                                           cartridge type — MBC1
$0148  01                                           ROM size — 64 KiB (4 banks)
$0149  00                                           RAM size — none
$014A  00                                           destination — Japanese
$014B  01                                           old licensee — Nintendo
$014C  00                                           mask-ROM version
$014D  9E                                           header checksum (verified)
$014E  41 6B                                        global checksum (verified)
```

A few observations:

- The **entry point** `$0100` is the only code the boot ROM jumps to. It is the
  near-universal `NOP ; JP $0150` — it steps over its own four bytes (the header
  begins at `$0104`) and lands at the real init at `$0150`, which immediately
  `JP $0185` into the cold-start sequence (Part II).
- The **title** field is the older 16-byte form, `"SUPER MARIOLAND"` padded with a
  `$00`. (Later carts shortened this field to make room for the manufacturer and CGB
  flags; this 1989 ROM uses the full 16 bytes.)
- The **cartridge type** `$01` is MBC1 with **no** RAM and **no** battery — so there
  is no save memory; Super Mario Land keeps no high scores across power cycles.
- **Destination** `$00` is *Japanese* even though this dump is the "World" image: the
  same mask was sold internationally, with region only in the box and manual.
- Because the **old licensee** byte `$014B` is `$01` (not the `$33` sentinel), the
  two-byte *new* licensee field at `$0144` is unused and reads `00 00`. `$01` is
  Nintendo.

## 5. The CPU vectors

The bottom of bank 0 is fixed hardware-defined entry points. Two groups:

- **`RST` vectors** at `$00, $08, $10, $18, $20, $28, $30, $38`, 8 bytes apart — the
  single-byte `RST n` call instructions jump here.
- **Interrupt vectors** at `$40` (VBlank), `$48` (LCD STAT), `$50` (Timer), `$58`
  (Serial) and `$60` (Joypad) — the CPU pushes `PC` and jumps here when the
  corresponding enabled interrupt fires.

The bytes at those addresses (decoded by hand; the LR35902 decoder is still to be
built):

| Address | Bytes | Decoded | Role |
|---|---|---|---|
| `$0000` | `C3 85 01` | `JP $0185` | `RST $00` → cold start |
| `$0008` | `C3 85 01` | `JP $0185` | `RST $08` → cold start |
| `$0020` | `87 E1 5F 16 00 19 5E 23 56 D5 E1 E9` | jump-table dispatch | `RST $20` |
| `$0040` | `C3 60 00` | `JP $0060` | VBlank → handler at `$0060` |
| `$0048` | `C3 95 00` | `JP $0095` | LCD STAT → `$0095` |
| `$0050` | `F5 3E 03 EA 00 20 CD F0 7F …` | inline timer ISR | Timer |
| `$0060` | `F5 C5 D5 E5 CD 4F 22 …` | inline VBlank body | (the VBlank handler) |

Three of these are worth calling out now:

- **`RST $20` is a jump-table dispatcher.** Its body is `ADD A,A ; POP HL ; LD E,A ;
  LD D,$00 ; ADD HL,DE ; LD E,(HL) ; INC HL ; LD D,(HL) ; PUSH DE ; POP HL ; JP (HL)`.
  It pops the return address (which points at an inline table of 16-bit targets that
  follows the `RST $20` call), indexes it by `A × 2`, and jumps to the selected
  target — the compact "call one of N routines by index" idiom. Several `RST` slots
  are used this way as one-byte gateways rather than as eight separate hooks.
- **The timer ISR (`$0050`)** is `PUSH AF ; LD A,$03 ; LD ($2000),A ; CALL $7FF0 ;
  LDH A,($FD) ; LD ($2000),A ; POP AF ; RETI`. It pages **bank 3** in, calls a routine
  at `$7FF0` (high in that bank — the sound engine), then restores the previous bank
  number it had stashed in HRAM at `$FFFD`. The audio is thus driven from the timer,
  decoupled from the frame rate.
- **The VBlank handler lives at `$0060`** — the *joypad* vector address. Each
  interrupt vector is only 8 bytes, too small for a real handler, so `$0040` jumps to
  a body placed at `$0060`; and because that body (and the timer ISR at `$0050`,
  which overruns into `$0058`) occupy the later vector slots, **Super Mario Land does
  not use the serial or joypad interrupts** — it polls the joypad instead, as most
  Game Boy games do. The VBlank body pushes all registers and `CALL`s a chain of
  per-frame routines (`$224F`, `$1B7D`, `$1C2A`, `$FFB6`, …) detailed in Part III.

## 6. What's in each bank

The four 16 KB banks are all a dense mix of code and data (2bpp tile graphics, level
maps, tables) — none is a pure-graphics or pure-code bank, and none is padding (all
sit at 6.4–7.1 bits/byte entropy, only 3–6 % `$FF` fill):

| Bank | File offset | Notes |
|---|---|---|
| **0** | `$00000`–`$03FFF` | Fixed lower window. Header + vectors, the cold-start/init, the main loop and the per-frame interrupt handlers, plus core engine code and tables. |
| **1** | `$04000`–`$07FFF` | Switchable. Engine code + data. |
| **2** | `$08000`–`$0BFFF` | Switchable. Engine code + data. |
| **3** | `$0C000`–`$0FFFF` | Switchable. Highest entropy (≈7.1 bits) — graphics/data heavy; the **sound engine** entry the timer ISR calls (`$7FF0`, i.e. file offset `$0FFF0`) lives at the top of this bank. |

A plaintext-string scan finds essentially nothing but the header title — there is no
ASCII copyright or level-name text in the image, because the Game Boy has no text
ROM: all on-screen text is drawn from the same 2bpp tile set as the graphics, so
"strings" are tile-index sequences in the map data, not bytes you can read directly.
Pinning exactly which routines and assets sit where in banks 1–3 is the work of
Parts II–IV, once the LR35902 disassembler exists.

---

# Part II — Boot and initialization

*Stubbed.* The cold-start path `$0100 → $0150 → $0185`: `DI`, clear `IF`/`IE`, bring
up the LCD and audio registers, clear VRAM/WRAM, load the initial tiles and palette,
enable VBlank+timer interrupts, and fall into the main loop. Needs the `tools/sm83`
LR35902 decoder/CPU core (see the intro).

# Part III — Engine architecture

*Stubbed.* The main loop, the VBlank handler chain at `$0060` and the timer-driven
banked sound engine, the WRAM layout, and the cross-bank dispatch (`RST` gateways +
`LD ($2000),A` bank selects).

# Part IV — Graphics and data formats

*Stubbed.* The 2bpp tile format, the `$9800`/`$9C00` background maps, OAM sprites and
the DMG palettes; the level and object data.

# Part V — Game mechanics

*Stubbed.* Mario's physics, the objects and enemies, the twelve levels across four
worlds, scoring and progression.

---

# Appendix A — Toolchain and reproduction

Verify the image before reusing it:

```sh
md5 "Super Mario Land (GB)/Super Mario Land (World).gb"   # md5sum on Linux
# b48161623f12f86fec88320166a21fce
```

Everything in Part I was derived by static inspection of the 64 KB image (header
fields, checksum recomputation, the Nintendo-logo comparison, per-bank entropy and a
by-hand decode of the vectors). The disassembly tooling used elsewhere in this
repository does **not** apply: the Game Boy's Sharp LR35902 is not a Z80, so Parts II
onward require a new LR35902 decoder and CPU core in the shared `tools` module
(provisionally `tools/sm83`, with a `tools/gameboy` machine model as the emulation
oracle, mirroring `tools/z80` + `tools/gamegear`).
