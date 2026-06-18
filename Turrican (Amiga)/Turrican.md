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
            ├─ patch $600CA/$600CE with BSR.W              the trainer (§6)
            └─ JMP $5F500                                  enter the decrunched game (§6)
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

The relocated core is **three decoders chained back-to-back**, not one. The
driver at `$50050` runs them in sequence — a canonical **Huffman** bit-reader
(`$502C2`), then an **LZ77** copier (`$5019A`), then an **RLE** expander
(`$500CA`) — relocating the intermediate result to the top of memory before each
pass and decoding it back down into low memory. Two of the three (LZ and RLE) are
**byte-dispatched**: each builds a 256-entry jump table at `$90` whose default
handler is a literal copier and whose few escape control values are overridden to
the match/run handlers, then the main loop reads a control byte, **writes it to
`$DFF180`** (the flashing border bars you see while a cracked game "decrunches"),
and `JMP`s through the table:

```
$50110  CMPA.l a1,a0
$50116  BCS    done
$5011E  MOVE.b (a0)+,d0                ; next control byte
$50120  MOVE.w d0,$DFF180.l            ; show it on the border
$50126  LSL.l  #2,d0                   ; ×4 → longword table index
$50128  MOVEA.l $0(a6,d0.w),a5
$5012C  JMP    (a5)                    ; dispatch
```

The Huffman pass is the exception — a 32-bit MSB-first bit-reader, no jump table.
Section 5 documents all three passes and the pure-Go reimplementation that
reproduces the decrunched image exactly. When the last pass finishes the core
`RTS`es back to `$7F806`.

## 5. The three passes, reimplemented

The crunched main part is not a single packed stream — it is the output of three
compressors applied in series. Decompression therefore runs the three decoders in
the opposite order: **Huffman → LZ77 → RLE**. Reading the disassembly of the
relocated core (`$50040`–`$5038C`) gives the whole algorithm; it is reimplemented
verbatim in Go in `Turrican (Amiga)/extract/decrunch`, and the result is verified
against the FS-UAE oracle (below) — **not** scraped from it.

### The blob and the parameter block

The boot loader's main read places this blob at `$50000` (disk `$2C00`):

| offset | bytes | meaning |
|--------|-------|---------|
| `$000` | long | `packedLen` = `$22C98` (whole blob length) |
| `$004` | long | `0` |
| `$008` | `$38` | bootstrap: disable DMA/INT, relocate the core to `$7F000`, `JMP` |
| `$040` | `$34C` | the decruncher core (relocated to `$7F000` at runtime) |
| `$38C` | `$12` | parameter block (below) |
| `$39E` | … | the packed stream, up to `packedLen` |

The 18-byte parameter block at `$38C` is copied to zero-page `$A4`:

```
+$00 word  $0000      unused
+$02 long  $00043880  output base   — where the final image loads
+$06 long  $0005F500  entry point   — where the game is entered
+$0A long  $00034580  (scratch: overwritten by escape-byte reads; also = final size)
+$0E long  $000228FA  (scratch)
```

The `$43880` base and `$5F500` entry drive the loader; the `$34580` at `+$0A` is a
neat cross-check — it is exactly the size of the fully decoded image (see below).

### The driver

`$50050` lays out five scratch pointers in zero page (`$90`…`$A0`), copies the
packed stream **backward** to end at `$7EB00`, then calls the three passes,
each writing into the output buffer at `$43880` and then being relocated back up
to `$7EB00` to feed the next pass:

```
$5008C  BSR $502C2     ; pass 1 — Huffman   (packed stream → LZ stream)
$5009E  BSR $5019A     ; pass 2 — LZ77      (LZ stream     → RLE stream)
$500B0  BSR $500CA     ; pass 3 — RLE       (RLE stream    → final image)
$500B2  MOVEA.l $AA.w,a0 ; a0 = $5F500 (entry), then RTS
```

### Pass 1 — Huffman (`$502C2`)

A canonical, threshold-table Huffman decoder over a **32-bit MSB-first**
bitstream. The pass header is, in order:

```
long            decodedLen          ; output length (= LZ-stream length)
256 bytes       symVal[256]         ; the byte values codes resolve to
long            levels              ; number of code-length classes
levels × long   thr[levels]         ; first-codeword thresholds, left-justified to 32 bits
levels × byte   symBase[levels]     ; base index into symVal for each class
levels × byte   codeLen[levels]     ; codeword bit length for each class
… bitstream …
```

For each output byte the decoder takes the 32-bit window at the current bit
position, finds the **smallest class `L` with `window ≥ thr[L]`** (the thresholds
decrease, so short/frequent codes sit in class 0 — the code special-cases it for
speed at `$50332`), then:

```
rem    = window − thr[L]
value  = rem >> (32 − codeLen[L])          ; the top codeLen[L] bits
emit     symVal[(symBase[L] + value) & 0xFF]
bitpos += codeLen[L]
```

This is the textbook "compare against left-justified first codewords" scheme; the
low bits of the window beyond the current codeword are the next code and never
affect class selection. Decoding stops after exactly `decodedLen` bytes.

### Pass 2 — LZ77 (`$5019A`)

Six **escape bytes** head the stream. The pass builds a 256-entry dispatch table:
every byte is a literal except the six escapes, each of which introduces a
back-reference (`copy length bytes from offset behind the cursor`). A `0` directly
after an escape emits that escape byte as a literal (the escape-the-escape case).
Later escapes overwrite earlier ones in the table, matching the 68000.

| escape | following bytes | offset | length |
|--------|-----------------|--------|--------|
| `esc0` `$5021A` | len, off-hi, off-lo | 16-bit | `len` |
| `esc1` `$50232` | len, off | 8-bit | `len` |
| `esc2` `$50248` | off | 8-bit | `3` |
| `esc3` `$5025C` | off-hi, off-lo | 16-bit | `4` |
| `esc4` `$50274` | `b` | `(b & $F) + 1` | `(b >> 4) + 3` |
| `esc5` `$50296` | `b`, `c` | `((b & $F) << 8) \| c` | `(b >> 4) + 4` |

Copies are byte-by-byte, so an offset smaller than the length produces a repeating
run (true LZ77 overlap).

### Pass 3 — RLE (`$500CA`)

Three escape bytes, same dispatch idea, expanding runs:

| escape | first byte `n`/`b` | emits |
|--------|--------------------|-------|
| `esc0` `$50134` | `n == 0` | 16-bit count + fill byte → count copies of fill |
| | `n == 1` | literal `esc0` |
| | `n ≥ 2` | fill byte → `n` copies of fill |
| `esc1` `$5014E` | `n == 0` | 16-bit count → that many `$00` |
| | `n == 1` | literal `esc1` |
| | `n ≥ 2` | `n` × `$00` |
| `esc2` `$50170` | `b == 0` | literal `esc2` |
| | `b ≠ 0` | three copies of `b` |

After RLE the image is complete: **`$34580` bytes (214,400) at `$43880`**,
ending at `$77E00`, with the game entered `$1BC80` into it at `$5F500`.

### The Go reimplementation and verification

`extract/decrunch` is a dependency-free package implementing the three passes
exactly as above; `extract/cmd/decrunch` runs it on the disk image:

```sh
cd "Turrican (Amiga)"
go run turrican/extract/cmd/decrunch -o /tmp/turrican.bin Turrican.adf
# base  = $43880
# entry = $5F500 (offset $1BC80 into image)
# size  = $34580 (214400 bytes), ends at $77E00
# md5   = 94327d996cc03f8d9039d81ba880642e
```

Per the project rule, the oracle confirms — it does not supply — the result. The
real `$50008` decruncher was run in isolation under FS-UAE/GDB: write the crunched
blob to `$50000`, set `PC = $50008` and `SP = $80000`, breakpoint the core's final
`RTS` (relocated to `$7F076`), and read `$43880`…`$77E00`. The emulator's output
is **byte-identical** to the Go decoder — same `$34580` bytes, same MD5
`94327d996cc03f8d9039d81ba880642e`. That `$34580` also equals the size field the
compressor left at parameter-block `+$0A`, an independent confirmation that all
three passes consume and produce the right counts.

## 6. The trainer and the game entry

With the main part decrunched into low memory, the tail applies the trainer by
overwriting two longwords of the game with `BSR.w` instructions
(`$600CA`/`$600CE`) that divert into the cheat code (the "99 lives"), builds a
small launch stub at `$5F700`, and `JMP $5F500` to start the game. The patch and
entry addresses (`$5F500`, `$5F700`, `$600CA`) place the decrunched program in
roughly `$400`…`$60000+` of chip RAM.

That decrunched image — the actual Turrican program, base `$43880`, entry
`$5F500` — is what Part III analyses. It is produced by the `extract/decrunch`
decoder above (verified byte-identical against the oracle), so the rest of the
work needs no emulator: a flat binary at a known load address.

# Part III — Game program architecture

The decruncher hands control to `$5F500` in the flat `$43880` image. This part
documents that program; it grows as the disassembly is annotated.

## 1. Disassembly and the `disasm/` annotation store

Following the repo's per-game convention (see Marble Madness's `disasm/`), the
unpacked program is disassembled into a committed, annotated source that is the
long-term home for everything learned about the code. Two files live in
`Turrican (Amiga)/disasm/`:

* `turrican.asm` — the generated 68000 disassembly of the `$43880` image;
* `turrican.annotations.txt` — the hand-maintained annotations (`ADDR  name
  description`, `#` comments), consumed by `codetrace68k -annotate` to label
  routines and inject notes. This is where analysis accumulates; the `.asm` is
  regenerated from it.

Regeneration (from the repo root):

```sh
# decode the main part to a flat image (base $43880, entry $5F500)
go run turrican/extract/cmd/decrunch -o /tmp/turrican.bin "Turrican (Amiga)/Turrican.adf"

# recursive-descent trace from the game's in-place segment, applying the annotations
go run stupidcoder.com/tools/cmd/codetrace68k -base 0x43880 \
  -entry 0x5F500,0x60000,0x602D0 \
  -annotate "Turrican (Amiga)/disasm/turrican.annotations.txt" \
  -o "Turrican (Amiga)/disasm/turrican.asm" /tmp/turrican.bin
```

## 2. Two segments: the relocation at `$5F500`

The flat image is not loaded where it runs. The very first thing `$5F500` does is
split itself in two:

```
05F500  game_entry:
        LEA  $43890,a0 ; LEA $10,a1
        MOVE.l (a0)+,(a1)+ … until a0 = $5F000   ; copy $43890..$5F000 → $10
        … fire-button option select, wait for fire …
        BRA  $60000                              ; game_init
```

So the image has **two segments**:

* **Work RAM** — image `$43890…$5F000` — copied down to **`$10…$1B780`**. This is
  *not* code: it starts zeroed (the vector table, buffers, the screen/state area)
  and is just the running game's scratch memory.
* **The setup / init / ISR + data** — image `$5F500…$77E00` — runs **in place**
  (the image sits at `$43880`, so image address = run address). `turrican.asm`
  covers this segment.

The actual game logic is in *neither* — it is an overlay streamed off the floppy
(§4).

## 3. The in-place setup segment

What runs before the game proper takes over, all from `turrican.asm`:

* **`game_entry` (`$5F500`)** — relocates the game proper (above); pokes the
  reset/illegal vectors at `$8`/`$C`; reads the CIA-A fire buttons
  (`$BFE001` bits 6/7) to pick **trainer/option** settings (poking game vars
  `$E0E`/`$D04`); waits for a fire press+release; `BRA game_init`.
* **`game_init` (`$60000`)** — the hardware bring-up: supervisor mode, stack at
  `$602CC`, all INTENA/DMACON/INTREQ off, then it unpacks and runs several
  sub-modules — `pp20_decrunch $6C186 → $30000` (a **PowerPacker "PP20"** block)
  and `JSR $30002`, a raw block-copy to `$50000` and `JSR $50000`, and
  `load_block_1BB00` — installs the level-3 (vblank) vector and enables the
  display via `setup_display`. The two `NOP`s at **`$600CA`** are the TRSI
  trainer hook (Part II §6), patched to `BSR` the 99-lives code at boot.
* **`vblank_isr` (`$602D0`)** — the level-3 interrupt: checks the VERTB bit in
  `INTREQR`, bumps a frame counter, runs `palette_cycle`, and `JSR $1BB24` (the
  per-frame **game tick**, in the overlay — §4).
* **Decompressors / loader** — the game reuses the **same Huffman decoder** as the
  `$50008` decruncher (`huff_decode $5F000`) to unpack graphics/level blocks at
  runtime, alongside `pp20_decrunch` and a floppy **trackloader**
  (`disk_read $604FA` / `disk_setup_sync $6056E`, `DSKSYNC = $4489`) that streams
  level data off the disk during play.

## 4. The streamed overlays, and the music driver at `$1BB00`

The resident image holds neither the game logic nor the sound code — `game_init`
streams those off the floppy. `load_block_1BB00` reads the packed block at
**ADF `$26000`** (length `$C268`) to `$1BB00` and `huff_decode`s it; the numbers
are self-consistent (`huff_decode`'s source window is `$1BB00…$27D68` = exactly
`$C268` bytes). Because the in-game decoder is the *same three passes* as the
`$50008` decruncher, the overlay is recovered with the same Go code —
`extract/cmd/block` reads the disk slice and calls `decrunch.DecrunchBlock` (which
skips the 18-byte block header and runs Huffman → LZ77 → RLE):

```sh
go run turrican/extract/cmd/block -off 0x26000 -len 0xC268 -base 0x1BB00 \
  -o /tmp/block_1BB00.bin "Turrican (Amiga)/Turrican.adf"
# base $1BB00, size $11964 (72036 bytes), ends $2D464
```

The decoded overlay is an **AmigaDOS HUNK blob** (`$000003F3` header, one
`HUNK_CODE` at `$1BB18`, body from `$1BB20`). It is loaded — not relocated — at a
fixed `$1BB00`, so its absolute `$1Cxxxx` addresses are baked to that base. Its
body opens with a **BRA dispatch table** — its public API, which the resident
code jumps into at fixed addresses.

This overlay is the **music / sound driver**. Its `vblank_isr` entry runs the
player once per frame, and internally it processes three voices, each with a
period LFO (vibrato), a pitch slide (portamento) and a volume envelope, writing
the Amiga `AUDxPER`/`AUDxVOL` registers — while `audio_silence` zeros all four
volumes. The API:

| call | slot | does |
|------|------|------|
| `vblank_isr` | `$1BB24` → `sound_tick $1BB78` | advance the player one frame; `a6` = `player_state $1CC22`, `a5` walks `voice_table $1CC7C` |
| `game_init` | `$1BB2C` | start playback (song index, clear pause) |
| `game_init` | `$1BB34` | init the player from song + sample data pointers (`$1CFF4`, `$20E90`) |
| `game_init` | `$1BB48` | set a config byte — master volume / channel mask (`$40`) |

It is disassembled into its own pair, `disasm/overlay_1bb00.{asm,annotations.txt}`
(`player_state $1CC22`, `update_voices`, `voice_vibrato/portamento/envelope`).
This matches the cabinet credit — the music is a **Chris Hülsbeck** score.

> **Next.** The game logic proper is a *further* streamed module — the `$30000`
> (`PowerPacker "PP20"`) and `$50000` loads `game_init` runs before the sound
> driver. Decode those next (the same `cmd/block`/PowerPacker route), then the
> level/graphics formats (Parts IV–V).

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
(`$22C98` bytes, blocks 22–301). The Go re-implementation (Part II §5, verified
byte-identical against the FS-UAE oracle) produces the decrunched program:

```sh
# decode the main part to a flat image ($43880 base, $5F500 entry)
go run turrican/extract/cmd/decrunch -o /tmp/turrican.bin "Turrican (Amiga)/Turrican.adf"

# then disassemble the unpacked game from its load address
go run stupidcoder.com/tools/cmd/dis68k -base 0x43880 /tmp/turrican.bin
```
