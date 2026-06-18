# Turrican (Amiga) — disk format and game analysis

A reverse-engineering reference for `Turrican.adf`, the Amiga release of
Turrican (Rainbow Arts / Factor 5, 1990). This is the second Amiga title in this
repository and the writeup follows the same shape as the others, in reading
order:

* **Part I** — the disk image: the ADF container and the disk's *custom* layout.
  Unlike Marble Madness, this is **not** an AmigaDOS volume — it is a bootable
  non-DOS disk whose boot block is a hand-written sector loader, so Part I is
  about mapping the raw disk rather than walking a filesystem;
* **Part II** — the boot chain: the boot block's multi-stage load and the
  unpacking / decryption of the main program from the packed track data;
* **Part III** — the game program: the 68000 startup, the interrupt/copper
  setup and the memory map;
* **Part IV** — graphics and data formats: the tile, sprite, level and audio
  encodings;
* **Part V** — game mechanics: the player, weapons, enemies, the levels and
  progression.
* **Appendices** — toolchain and reproduction.

Methods: purely static analysis of the disk image, plus the 68000 toolchain in
the shared `tools/` module — the AmigaDOS reader (`tools/amiga/adf`), the
disassemblers (`tools/cmd/dis68k`, `tools/cmd/codetrace68k`) and the 68000
execution core (`tools/m68k`) for dynamic verification. All addresses are 68000
addresses; sizes are `.b`/`.w`/`.l` (8/16/32-bit). **Parts I–II are complete;
Parts III–V are stubs.**

---

## Contents

- [Part I — The disk image](#part-i--the-disk-image)
  - [1. The ADF container](#1-the-adf-container)
  - [2. A custom boot disk, not an AmigaDOS volume](#2-a-custom-boot-disk-not-an-amigados-volume)
  - [3. The boot block: a raw-sector loader](#3-the-boot-block-a-raw-sector-loader)
  - [4. The disk map](#4-the-disk-map)
- [Part II — Boot chain](#part-ii--boot-chain)
  - [1. A cracked release — Tristar & Red Sector](#1-a-cracked-release--tristar--red-sector)
  - [2. The boot chain at a glance](#2-the-boot-chain-at-a-glance)
  - [3. The first-stage intro (`$30000`)](#3-the-first-stage-intro-30000)
  - [4. The hand-off (`$7F800`) and the decruncher (`$50008`)](#4-the-hand-off-7f800-and-the-decruncher-50008)
  - [5. The trainer and the game entry](#5-the-trainer-and-the-game-entry)
- [Part III — Game program architecture](#part-iii--game-program-architecture)
- [Part IV — Graphics and data formats](#part-iv--graphics-and-data-formats)
- [Part V — Game mechanics](#part-v--game-mechanics)
- [Appendix A — Toolchain and reproduction](#appendix-a--toolchain-and-reproduction)

---

# Part I — The disk image

## 1. The ADF container

An ADF is the simplest possible disk image: a flat dump of the floppy's logical
blocks with no header or metadata. Turrican ships on one standard
double-density disk — **1760 blocks of 512 bytes = 901,120 bytes** — so block
*N* is simply the 512 bytes at file offset *N* × 512. The exact copy this
analysis is based on is pinned by size and MD5 in the repository
[README](../README.md#image-files).

## 2. A custom boot disk, not an AmigaDOS volume

The first four bytes are `44 4F 53 00` — the `"DOS\0"` boot-block signature — so
the Kickstart ROM will accept the disk and run its boot code. But that is as far
as AmigaDOS goes: there is **no filesystem on the disk**. The boot block's
block-8 field still carries the conventional root-block pointer (`$00000370` =
880, the standard value for a DD disk), yet block 880 is not a valid root block,
and the AmigaDOS reader rejects it:

```
$ adfdump Turrican.adf
adfdump: adf: root block is not a valid root header
```

This is the usual shape of a commercial Amiga game disk: the `"DOS\0"` signature
and a valid boot-block checksum are the *only* AmigaDOS-conformant things on it.
Everything else — the program, the graphics, the levels — is laid out in a
private format and pulled off the disk by the game's own loader, addressing the
medium by absolute byte offset through `trackdisk.device`, never through files.
(Contrast Marble Madness, whose disk is a real OFS volume — see that writeup's
Part I.)

This particular image is not the original Rainbow Arts release but a **cracked
one-disk version by Tristar & Red Sector (TRSI)** — the boot block and its loader
are the cracker's, and the game's "main part" rides the disk **crunched**
(compressed), decrunched on boot (Part II). So this Part maps the raw disk;
decoding what the loader fetches is Part II.

## 3. The boot block: a raw-sector loader

The boot block is blocks 0–1 (1024 bytes): the `"DOS\0"` tag, a checksum
(`$090B08A1` at `+4`, which the ROM verifies before it will boot), the vestigial
root pointer at `+8`, and from `+12` the boot code the ROM jumps to with the
boot device's I/O request in `A1`. That code is a complete sector loader.

It begins with a first read (`BSR $2C0`) that loads the first-stage loader and
runs it (the IOStdReq fields are `io_Length` at `+$24`, `io_Data` at `+$28`,
`io_Offset` at `+$2C`):

```
$2C2  MOVE.w #$2,$1C(a1)        ; io_Command = CMD_READ
$2C8  MOVE.l #$30000,$28(a1)    ; io_Data    = $30000
$2D0  MOVE.l #$1000,$24(a1)     ; io_Length  = $1000 (4 KB = 8 blocks)
$2D8  MOVE.l #$400,$2C(a1)      ; io_Offset  = $400  (block 2)
$2E0  JSR  -$1C8(a6)            ; DoIO  (a6 = ExecBase, -456 = DoIO)
$2E4  JSR  $30000               ; run the first stage just loaded
$2EA  MOVE.w #$1A0,$DFF096      ; DMACON: clear bitplane/copper/sprite DMA
```

So blocks 2–9 (4 KB) are read to `$30000` and `JSR $30000` runs the first-stage
loader — cleartext 68000 code (it is the crack's intro/decruncher; see Part II).
Control then returns to the main boot code, which:

1. blanks the border (`CLR.w $DFF180`) and takes `A6 = ExecBase` (`$4.w`);
2. sizes and grabs a work buffer — `AvailMem`/`AllocMem` (`-$D8`/`-$C6` on
   `ExecBase`) for the **largest FAST chunk** (`MEMF_FAST|MEMF_LARGEST`,
   `$20004`), or, on a 512 KB chip-only machine, the chip region `$80000`…
   `$100000`;
3. issues the **main read** — `io_Offset $2C00` (block 22), `io_Length $22E00`
   (143 KB = 280 blocks), `io_Data $50000` — pulling the crunched main part into
   RAM at `$50000`, then stops the drive motor (`io_Command = 9`, `TD_MOTOR`);
4. adapts to the CPU: on a 68010 or better (`BTST #1,AttnFlags`) it installs a
   `TRAP #0` handler that executes a `MOVEC` — the standard fixup so the rest of
   the loader can keep treating the machine like a bare 68000;
5. seizes the machine: `MOVE #$2000,SR` (supervisor, no interrupts), stack at
   `$80000`, copies a 512-byte tail routine to **`$7F800`** and `JMP $7F800` to
   carry on the load/decrypt.

The boot block never touches `dos.library`; it reads the disk by absolute byte
offset and drives the hardware directly. Following the `$7F800` stage and the
unpacking is Part II.

## 4. The disk map

Reading the boot block's offsets back onto the image, and confirming it with a
byte-entropy sweep, the disk falls into three regions:

| blocks | offset | entropy | contents |
|--------|--------|---------|----------|
| 0–1 | `$0`–`$400` | ~3.7 | **boot block** — the sector loader (§3) |
| 2–21 | `$400`–`$2C00` | ~3.7 | **first-stage loader** — plain 68000 (opens `MOVEM.l d0-d7/a0-a6,-(a7)`, drives `DMACON`); blocks 2–9 are loaded to `$30000`, and it carries the crack-intro text and `graphics.library`/`topaz.font` strings (Part II) |
| 22–1759 | `$2C00`–end | ~7.99 | **crunched main part** — the game program, graphics and level data, compressed; the `$22E00` main read pulls blocks 22–301 of it to `$50000` |

The low two regions are recognizable 68000 code (entropy well under 4 bits/byte);
from block 22 on the image is essentially incompressible (entropy ~7.99 of a
possible 8) — the signature of crunched (compressed) data, not a filesystem or
raw bitplanes. There is no directory to enumerate — the boot block's two reads
(§3) are the disk's entire "table of contents." Decrunching that main part back
into program is the work of Part II.

---

# Part II — Boot chain

The boot block (Part I §3) is the disk's entire bootstrap: it loads a first
stage, reads the crunched main part, seizes the machine and jumps into a tail
routine. This part follows that chain to the point where the decrunched game
runs.

## 1. A cracked release — Tristar & Red Sector

The first thing the disassembly turns up is that this is **not** the original
Rainbow Arts disk. The first-stage loader (blocks 2–9, loaded to `$30000`)
carries the crack's intro text in clear ASCII:

```
TRISTAR & RED SECTOR PRESENT:  T U R R I C A N
The 100% - One Disk - Version. !!
For The TRAINER Press Joystickbutton After DeCrunching The Mainpart.
Now You Will Have 99 Lives !!
HiScores Will Be Saved On Track 0 !
Intro Made By TRANSFORMER.   Back To The Roots !!
```

So the disk is a **TRSI** "one-disk" crack with a **trainer** (99 lives), the
high-score save redirected to track 0, and a loader/intro of the cracker's own.
Everything in this part — the intro, the decruncher, the trainer patches — is
the crack's wrapper around Turrican; the game itself only appears once the main
part is decrunched (§4). The loader also names the libraries and font it uses
for the intro display: `graphics.library` and `topaz.font`.

## 2. The boot chain at a glance

```
ROM strap
  └─ boot block ($C)                                       Part I §3
       ├─ read blocks 2–9  → $30000 ; JSR $30000           → first-stage intro (§3)
       ├─ read blocks 22–301 ($22E00) → $50000             the crunched main part
       ├─ take over (SR=$2000, sp=$80000), copy tail→$7F800
       └─ JMP $7F800                                       the hand-off (§4)
            ├─ JSR $50008                                  decrunch the main part (§4)
            ├─ patch $600CA/$600CE with BSR.W              the trainer (§5)
            └─ JMP $5F500                                  enter the decrunched game (§5)
```

## 3. The first-stage intro (`$30000`)

`JSR $30000` runs the crack intro. It is ordinary cleartext 68000:

```
$30000  MOVEM.l d0-d7/a0-a6,-(a7)
$30004  MOVE.w  #$8100,$DFF096        ; DMACON: enable bitplane DMA
$3000C  MOVE.l  $4.l,$304F4           ; stash ExecBase
$3001A  MOVE.l  #$10002,d1            ; MEMF_CHIP|MEMF_CLEAR
$30020  MOVE.l  #$2EE0,d0
$30026  JSR     -$C6(a6)              ; AllocMem $2EE0 chip  (the intro bitplanes)
        … carve the buffer into screen/scroll planes at +$7D0/+$1F40/+$2710 …
$3005A  LEA     graphics.library(pc),a1
$3005E  JSR     -$198(a6)             ; OldOpenLibrary("graphics.library")
        … set up topaz.font, a copper list and the scrolling-text display …
```

It allocates a ~12 KB chip buffer for its bitplanes, opens `graphics.library`,
puts up the scrolling TRSI greetings and the trainer prompt, and returns. (This
is decoration; the analysis does not trace the scroller instruction by
instruction.) The boot block then clears DMA (`$DFF096 = $1A0`) and proceeds to
read the main part and hand off.

## 4. The hand-off (`$7F800`) and the decruncher (`$50008`)

The 512-byte tail the boot block relocated to `$7F800` is the bridge from loader
to game:

```
$7F800  JSR  $50008                   ; decrunch the main part (returns when done)
$7F806  CLR.w $DFF180                  ; border black
$7F80C  MOVE.l #$610003DE,$600CA       ; trainer patch  (BSR.w into cheat code)
$7F816  MOVE.l #$6100F630,$600CE       ; trainer patch
$7F820  …build a small stub at $5F700…
$7F846  JMP  $5F500                    ; enter the decrunched game
```

`$50008` is the head of the crunched blob at `$50000`. The blob begins with a
length longword (`$00022C98`) and the decruncher proper:

```
$50008  MOVE.w #$7FFF,$DFF09A          ; INTENA: all interrupts off
$50010  MOVE.w #$7FFF,$DFF096          ; DMACON:  all DMA off
$50018  MOVE.l $50000(pc),d7           ; d7 = $22C98 (packed length)
$5001C  LEA    $50000(pc),a0
$50020  ADD.l  a0,d7                   ; d7 = $72C98  = end of packed data
$50022  …relocate the $34C-byte decrunch core $50040–$5038C → $7F000…
$5003A  JMP    $7F000
```

The relocated core is a **backward, byte-dispatched decruncher** (the backward
copy-then-decode is the usual "in-place" trick, but the byte-indexed jump table
below is its own scheme — not a bit-reader like ByteKiller/PowerPacker):

* it copies the packed stream up to the top of memory — `MOVE.b -(a0),-(a1)`
  from `$72C98` down into `$7EB00` going **downwards** — then decrunches from
  there back down into low memory, so source and destination can overlap;
* decoding is **byte-dispatched**: it builds a 256-entry jump table at `$90`
  (default handler `$5012E`, with a few control values overridden to the
  literal/match copiers `$50134`/`$5014E`/`$50170`), then the main loop reads a
  control byte, **writes it to `$DFF180`** (the flashing border bars you see
  while a cracked game "decrunches"), and `JMP`s through the table:

```
$50110  CMPA.l a1,a0
$50116  BCS    done
$5011E  MOVE.b (a0)+,d0                ; next control byte
$50120  MOVE.w d0,$DFF180.l            ; show it on the border
$50126  LSL.l  #2,d0                   ; ×4 → longword table index
$50128  MOVEA.l $0(a6,d0.w),a5
$5012C  JMP    (a5)                    ; dispatch
```

When it finishes it `RTS`es back to `$7F806`.

## 5. The trainer and the game entry

With the main part decrunched into low memory, the tail applies the trainer by
overwriting two longwords of the game with `BSR.w` instructions
(`$600CA`/`$600CE`) that divert into the cheat code (the "99 lives"), builds a
small launch stub at `$5F700`, and `JMP $5F500` to start the game. The patch and
entry addresses (`$5F500`, `$5F700`, `$600CA`) place the decrunched program in
roughly `$400`…`$60000+` of chip RAM.

That decrunched image — the actual Turrican program — is what Part III
disassembles. It is recovered by **re-implementing the `$50008` backward
byte-dispatch decoder in Go** (the "declarative" route, as for Marble Madness's
`c/zzz`), run on the crunched main part read straight from the disk; the FS-UAE/
GDB oracle (`tools/amiga/fsuae-debug`) is used only to *guide* the
reimplementation (single-stepping the handlers) and to *verify* it (diffing the
Go output against the emulator's). Producing that Go decruncher is the first
task of Part III.

# Part III — Game program architecture

> **Stub.** The 68000 startup of the unpacked program: interrupt and copper
> setup, the main loop, and the memory map.

# Part IV — Graphics and data formats

> **Stub.** Tile maps, sprites/BOBs, the level encodings, fonts and audio
> (Turrican's music is a TFMX/“Chris Hülsbeck” player).

# Part V — Game mechanics

> **Stub.** Player movement and the weapon system, enemies, the worlds and
> level structure, scoring and progression.

# Appendix A — Toolchain and reproduction

The disk facts above reproduce with the shared tools:

```sh
# size + hash (must match the README image table)
md5 "Turrican (Amiga)/Turrican.adf"

# confirm it is not an AmigaDOS volume
go run stupidcoder.com/tools/amiga/cmd/adfdump "Turrican (Amiga)/Turrican.adf"

# disassemble the boot block (code starts at +12)
go run stupidcoder.com/tools/cmd/dis68k -skip 12 -base 0xc "Turrican (Amiga)/Turrican.adf"

# the loader stages on the disk (skip = disk byte, base = the address it runs at)
A=Turrican.adf
go run stupidcoder.com/tools/cmd/dis68k -skip 0x400  -base 0x30000 "$A"   # first-stage intro -> $30000
go run stupidcoder.com/tools/cmd/dis68k -skip 0xf8   -base 0x7f800 "$A"   # tail/hand-off       -> $7F800
go run stupidcoder.com/tools/cmd/dis68k -skip 0x2c08 -base 0x50008 "$A"   # decruncher          -> $50008
```

The `$50008` decruncher's input is the crunched main part at disk `$2C00`
(`$22C98` bytes, blocks 22–301); a Go re-implementation of it (verified against
the FS-UAE oracle) yields the decrunched program disassembled in Part III.
