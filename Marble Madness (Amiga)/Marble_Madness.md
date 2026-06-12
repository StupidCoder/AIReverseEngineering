# Marble Madness (Amiga) — disk format and game analysis

A reverse-engineering reference for `Marble_Madness.adf`, the Amiga release of
Marble Madness (disk volume `MarbleMadness!`). This is the first Amiga title in
this repository and the writeup follows the same shape as the C64 games, in
reading order:

* **Part I** — the disk image: the ADF container, the AmigaDOS filesystem on it,
  and the full file inventory — enough to pull every byte off the disk;
* **Part II** — the boot chain: from the boot block through the AmigaDOS
  startup to the game launcher and how the program and its overlays load;
* **Part III** — the game program: the 68000 startup, interrupt/copper setup
  and memory map;
* **Part IV** — graphics and data formats: the per-course level, image, vector
  and audio modules and how they are encoded;
* **Part V** — game mechanics: the marble, the courses, the hazards and enemies,
  scoring and progression.
* **Appendices** — toolchain and reproduction.

Methods: purely static analysis of the disk image, plus the 68000 toolchain
built for it in the shared `tools/` module — the AmigaDOS reader
(`tools/amiga/adf`), the disassemblers (`tools/cmd/dis68k`,
`tools/cmd/codetrace68k`) and an instruction-level 68000 execution core
(`tools/m68k`) for dynamic verification. All addresses are 68000 addresses;
sizes are `.b`/`.w`/`.l` (8/16/32-bit). Parts I and II are complete and Part IV
is under way; Parts III and V are still stubs.

---

## Contents

- [Part I — The disk image](#part-i--the-disk-image)
  - [1. The ADF container](#1-the-adf-container)
  - [2. The AmigaDOS filesystem](#2-the-amigados-filesystem)
  - [3. The disk contents](#3-the-disk-contents)
- [Part II — Boot chain](#part-ii--boot-chain)
  - [1. The boot block](#1-the-boot-block)
  - [2. AmigaDOS startup and the Workbench launch](#2-amigados-startup-and-the-workbench-launch)
  - [3. The launcher](#3-the-launcher)
- [Part III — Game program architecture](#part-iii--game-program-architecture)
- [Part IV — Graphics and data formats](#part-iv--graphics-and-data-formats)
  - [1. The splash (boot) screen](#1-the-splash-boot-screen)
  - [2. The Workbench icons](#2-the-workbench-icons)
  - [3. The per-course sprite banks (`.ilb`)](#3-the-per-course-sprite-banks-ilb)
- [Part V — Game mechanics](#part-v--game-mechanics)
- [Appendix A — Toolchain and reproduction](#appendix-a--toolchain-and-reproduction)

---

# Part I — The disk image

## 1. The ADF container

An ADF is the simplest possible disk image: a flat dump of the floppy's logical
blocks with no header or metadata. Marble Madness ships on one standard
double-density disk — **1760 blocks of 512 bytes = 901,120 bytes** — so block
*N* is simply the 512 bytes at file offset *N* × 512. The exact copy this
analysis is based on is pinned by size and MD5 in the repository
[README](../README.md#image-files).

There is no copy-protection track scheme to defeat here the way the C64 tapes
needed a fastloader reverse-engineered: an ADF is already the decoded block
image. The work in this part is therefore reading the *filesystem* laid over
those blocks.

## 2. The AmigaDOS filesystem

The disk is formatted with the original AmigaDOS filesystem (**OFS**). The boot
block (blocks 0–1) opens with the 4-byte signature `"DOS\0"`; the trailing zero
is the filesystem-flags byte, and zero means OFS (a `1` would be FFS). The same
header points the root block at **block 880** (`numBlocks / 2` for a DD disk),
and the root block carries the volume name, **`MarbleMadness!`**.

The on-disk structures, all decoded by `tools/amiga/adf`, are the standard
AmigaDOS ones:

- **Directories** (the root and the three subdirectories) store their entries in
  a 72-slot hash table; entries that collide in a slot are threaded through a
  hash-chain field in each header block.
- **File headers** list their data blocks in a table filled from the end of the
  block; a file larger than ~72 blocks continues into *file-extension* blocks.
- **Data blocks**, under OFS, are not raw payload: each 512-byte block begins
  with a 24-byte header (the owning file, a sequence number, the next-block
  link, and a valid-byte count) followed by up to 488 bytes of file data. (FFS
  would store 512 raw bytes per block; this disk does not.)

The boot block itself contains the **standard AmigaDOS boot code**, not a custom
loader: its 68000 fragment references the string `dos.library`, opens it through
an Exec library call and returns its base, which is exactly what an ordinary
bootable AmigaDOS disk does. So unlike the C64 titles, there is no bespoke
on-disk loader to reverse — the disk boots through the normal ROM/AmigaDOS path,
and the interesting boot logic is the game's own startup (Part II).

## 3. The disk contents

The volume holds **50 files across 3 directories** (`c/`, `s/`, `libs/`). Almost
every file begins with `$0000_03F3`, the **`HUNK_HEADER`** magic of an Amiga
loadable object — a relocatable code/data segment that AmigaDOS brings in with
`LoadSeg`. So the game is not one monolithic binary but a launcher plus a main
program plus a large set of per-course overlays, each a hunk file loaded on
demand.

A **hunk file** is the Amiga's relocatable program format, produced by the linker
and loaded by `LoadSeg`. It opens with a header (the `$3F3` magic, then the size
of each segment) and is followed by the segments themselves — `CODE`, `DATA` and
zero-filled `BSS` blocks — each optionally trailed by a 32-bit relocation table.
Because a segment may be placed anywhere in memory, that table lists every
longword inside the segment that holds an address, and `LoadSeg` adds the
segment's real load address to each of them as it brings the file in; the program
is therefore position-independent. The shared reader `tools/amiga/hunk` does the
same — it lays the segments out from a chosen base, applies the relocations, and
returns a flat image that `dis68k`/`codetrace68k` can disassemble.

Two files are exceptions. `c/MarbleMadness!.dat` (the main program) and `c/xxx`
are not plain hunks: after a `$0000_03F3 8F01…` packer header their contents are
near-random (entropy ≈ 7.95 of 8 bits/byte), i.e. **stored crunched** and
decompressed at load by the small `c/zzz` helper. Unpacking them — and thereby
reaching the main game code — is a Part II/III concern.

**System and boot files** — an ordinary AmigaDOS boot setup:

| file | role |
|------|------|
| `s/startup-sequence` | the boot script (21 bytes): `LoadWb` then `endcli` — it boots to Workbench |
| `c/LoadWb`, `c/EndCLI` | the AmigaDOS CLI commands the script runs |
| `c/splash` | the title/splash screen — an IFF ILBM bitmap (Part IV §1) |
| `c/bootscr` | the boot-screen program (a 50-hunk overlay) that displays the splash (Part II §3) |
| `c/zzz` | the decruncher — unpacks the crunched files at load (Part III) |
| `c/xxx` | crunched data unpacked by `c/zzz` |
| `c/sigfile` | a disk-signature / copy-protection table (`"DOW"`+incrementing bytes) |
| `libs/icon.library` | bundled so the disk shows icons without the user's Workbench disk |
| `MarbleMadness!.info`, `Disk.info`, `.info` | Workbench icons (`$E310` magic, not hunks) |

**The game program:**

| file | size | role |
|------|------|------|
| `MarbleMadness!` | 5,864 | the launcher executable (started from the Workbench icon) |
| `c/MarbleMadness!.dat` | 175,360 | the main game program — stored crunched, unpacked at load by `c/zzz` |

**Per-course modules.** The bulk of the disk is six parallel families of files,
one per Marble Madness course, keyed by a short prefix and a type suffix:

| prefix | course | level | image | vector | sound | music |
|--------|--------|-------|-------|--------|-------|-------|
| `prc` / `practy` | Practice | `practy.mlb` | `practy.ilb` | — | `PrcSnd` | `PrcTrack` |
| `beg` / `beginr` | Beginner | `beginr.mlb` | `beginr.ilb` | `begobsc.vlb` | `BegSnd` | `BegTrack` |
| `int` / `interm` | Intermediate | `interm.mlb` | `interm.ilb` | `intobsc.vlb` | `IntSnd` | `IntTrack` |
| `aer` / `aerial` | Aerial | `aerial.mlb` | `aerial.ilb` | `aerobsc.vlb` | `AerSnd` | `aertrack` |
| `sil` / `silly` | Silly | `silly.mlb` | `silly.ilb` | `silobsc.vlb`, `slink.vlb` | `SilSnd` | `SilTrack` |
| `ult` / `ultima` | Ultimate | `ultima.mlb` | `ultima.ilb` | `ultobsc.vlb` | `UltSnd` | `ulttrack` |

Plus shared assets: `marbdat` / `marbdat.vlb` (the marble), and `birdink.vlb`
and `ooze.vlb` (the creatures). The suffix conventions: `.ilb` is the per-course
**sprite bank** (its container is decoded in Part IV §3); the rest are read off
the names and sizes and stay **provisional** until Part IV decodes them — `.mlb`
course/level data, `.vlb` vector or obstacle ("`obsc`") graphics, `*Snd` sound
effects, and `*Track` music.

**Compression and encryption.** Several encodings appear on the disk, decoded in
this writeup where possible. The two executables that matter most —
`c/MarbleMadness!.dat` (the main program) and `c/xxx` — are **crunched** by a
custom packer (a shared `$0000_03F3 8F01…` header, contents at ≈ 7.95 of 8
bits/byte) and unpacked at load by `c/zzz`; reaching the main code therefore means
reversing that decruncher (Part III). The title screen `c/splash` is an IFF ILBM
whose pixels are **ByteRun1 (PackBits)** compressed (Part IV §1). The per-course
sprite banks (`.ilb`) carry their own **run-length sprite packing** (Part IV §3).
`c/sigfile` is not compression but a copy-protection signature table. The
remaining per-course formats (`.mlb`, `.vlb`, `Snd`, `Track`) and whatever
encoding they use are still to be decoded. (The on-disk filesystem also frames
every file in OFS data blocks — a storage layout, not compression — which the
`adf` reader undoes when it extracts the files, Part I §2.)

Putting it together, the boot model is: a standard AmigaDOS disk boots to
Workbench via `startup-sequence`; the player launches the game from the
`MarbleMadness!` icon; the launcher pulls in `c/MarbleMadness!.dat` (the main
program) which then loads each course's `.mlb`/`.ilb`/`.vlb`/`Snd`/`Track`
overlays as the game reaches that course. Tracing that startup is Part II.

---

# Part II — Boot chain

Where the C64 tapes hid a custom fastloader that had to be reverse-engineered
before a single byte could be read, the Amiga disk boots through entirely
**stock AmigaDOS**. The chain is: ROM runs the boot block → the boot block
hands off to `dos.library` → AmigaDOS runs `s/startup-sequence` → Workbench
comes up → the player launches the game from its icon → a small compiled
launcher pulls the game in with `LoadSeg`. Each link is a standard mechanism;
the only game-specific code is the launcher.

## 1. The boot block

Blocks 0–1 are the boot block: the `"DOS\0"` signature, a checksum longword, the
root-block pointer (880), and then a short 68000 routine. Disassembled, it is the
**unmodified AmigaDOS 1.x boot code**:

```
0000: 43FA 0018     LEA   dosName(pc),a1     ; a1 -> "dos.library"
0004: 4EAE FFA0     JSR   -$60(a6)           ; FindResident()   (a6 = ExecBase)
0008: 4A80          TST.l d0
000A: 670A          BEQ   fail
000C: 2040          MOVEA.l d0,a0             ; a0 = the Resident node
000E: 2068 0016     MOVEA.l $16(a0),a0        ; a0 = rt_Init  (DOS boot entry)
0012: 7000          MOVEQ #0,d0               ; d0 = 0  -> success
0014: 4E75          RTS
fail:
0016: 70FF          MOVEQ #-1,d0
0018: 60FA          BRA   $0014
001A: "dos.library",0
```

It calls Exec's `FindResident` (LVO `-$60`) to locate the resident `dos.library`,
reads its `rt_Init` field (offset `$16` of the Resident node — the DOS boot
point) into `a0`, and returns success. The ROM then calls that boot point to
bring DOS up. There is no decryption, no custom track format, no game code here
at all — exactly the standard bootstrap, which is why an ADF reader plus stock
AmigaDOS knowledge is enough to get at everything (Part I).

## 2. AmigaDOS startup and the Workbench launch

Once DOS is running it mounts the volume and executes the boot script
`s/startup-sequence`, which is just two lines:

```
LoadWb
endcli > nil:
```

`LoadWb` starts Workbench; `endcli` closes the boot shell. Workbench here is the
Amiga's desktop GUI — and it is not a single program but a service of the
operating system: the windowing, menu and gadget machinery (`intuition.library`,
`graphics.library`, and the kernel `exec`) lives in the Kickstart **ROM**, and
`LoadWb` is just the small command that brings the desktop up on top of those ROM
libraries. The disk itself carries no Workbench binary; the only GUI pieces it
bundles are `libs/icon.library` (which draws icons) and the `.info` files, so that
it can show its own window and icons even on a bare machine without the user's
Workbench floppy. The game is then launched the Workbench way — by double-clicking
the **`MarbleMadness!`** icon, which is a *tool* icon (icon type 3), so Workbench
`LoadSeg`s and runs the `MarbleMadness!` program directly.

## 3. The launcher

`MarbleMadness!` is a small (5,864-byte) compiled program — 23 hunks, with a
`HUNK_SYMBOL` table left in. Its entry point is a standard C-runtime startup that
works out how it was launched:

```
MOVEM.l d3-d7/a0-a6,-(a7)     ; save registers
MOVE.l  a7,$1C.l              ; stash the stack pointer
MOVE.l  d0,$24.l / a0,$28.l   ; CLI argument length / pointer (0 from Workbench)
MOVEA.l $4.l,a6               ; a6 = ExecBase (AbsExecBase at $0000.0004)
SUBA.l  a1,a1
JSR     -$126(a6)             ; FindTask(0) -> our task
MOVEA.l d0,a4
TST.l   $AC(a4)               ; Process.pr_CLI : zero => launched from Workbench
BEQ     wbStartup             ; …which is the path taken here
```

The `pr_CLI` test (`$AC`) is the textbook Workbench-vs-CLI check, and because the
game is started from its icon it takes the Workbench branch and collects the
`WBStartup` message.

What the launcher then does is read off its symbol table and data hunk rather
than guessed: the symbols name the exact library entry points it links against —
from `dos.library` `_LoadSeg` / `_UnLoadSeg` / `_Lock` / `_CurrentDir` /
`_Input` / `_Output`, and from `exec.library` `_AllocMem` / `_FreeMem` /
`_FindTask` / `_CreatePort` / `_DeletePort` / `_PutMsg` / `_GetMsg` — so it
allocates memory, sets up message ports, and brings program segments in with
`LoadSeg`. Its data hunk holds the asset names it pulls from the `c/` directory:

| asset | what it is |
|-------|------------|
| `c/splash` | the title/splash screen — an IFF `FORM…ILBM` image |
| `c/bootscr` | a boot screen (hunk object) |
| `c/xxx` | opaque, high-entropy data (likely packed) |
| `c/zzz` | a second small compiled stage (also links `dos.library`) |

So the launcher displays the splash and then uses `LoadSeg` to load the game
proper. The main program is `c/MarbleMadness!.dat` — the 175 KB hunk from Part I —
with `c/zzz` and `c/xxx` as supporting loader/data pieces. Every loadable element,
the launcher included, is an ordinary Amiga hunk object brought in through
`LoadSeg`, which is the same mechanism the running game later uses to stream its
per-course overlays (Parts III–IV). (The precise hand-off order between the
launcher, `c/zzz`, `c/xxx` and the main `.dat` is compiled-C loader detail; what
matters for the boot chain is that it is all stock `LoadSeg`, with the game code
arriving as the relocatable `MarbleMadness!.dat` hunk.)

---

# Part III — Game program architecture

*To follow.* The 68000 startup of the main program, the interrupt/copper/blitter
setup, and the memory map during play.

---

# Part IV — Graphics and data formats

The boot-time graphics use standard Amiga formats the toolchain already reads end
to end — the title splash (§1) and the Workbench icons (§2). §3 starts on the
game's own formats with the per-course sprite banks (`.ilb`); the remaining
per-course modules (`.mlb`, `.vlb`, `Snd`, `Track`) are still to be decoded.

## 1. The splash (boot) screen

The boot/title screen the player sees is `c/splash`, stored as a standard **IFF
`FORM…ILBM`** bitmap (the Amiga's usual image format). Its `BMHD` describes a
**320×200, 4-bitplane (16-colour)** image; the `BODY` is **ByteRun1 (PackBits)
compressed**; a `CMAP` chunk carries the 16-colour palette; and four `CRNG`
chunks define colour-cycling ranges, so parts of the logo animate on the real
machine. The decoder `tools/amiga/iff` parses those chunks, unpacks the BODY,
de-interleaves the four bitplanes into colour indices and looks them up in the
CMAP:

![Marble Madness splash screen](rendered/splash.png)

Decoding it also recovers the on-disk attribution that the screen displays: the
Amiga conversion is credited to **Larry Reed**, under **© 1984, 1986 Atari Games
Corp. & Electronic Arts** — facts read straight out of the image, not from any
outside source.

The image is the *data*; `c/bootscr` is the *code* that puts it on screen. The
launcher (Part II) `LoadSeg`s both, and `c/bootscr` is the overlay that displays
this splash at boot — it is a 50-hunk compiled program (its first hunk is
`HUNK_CODE`), not a second picture, which is why there is one bitmap here, not
two. (A loose end alongside them, `c/sigfile`, is not graphics either: it is a
short table of `"DOW"`+incrementing-byte entries, a disk-signature /
copy-protection artefact.)

## 2. The Workbench icons

The `.info` files are standard Workbench icons: a `DiskObject` header (`$E310`
magic) followed by one or two planar `Image` structures. Icons carry no palette
of their own — they are drawn in the Workbench screen pens — so `tools/amiga/icon`
renders them with the standard Workbench 1.x four-colour palette (pen 0 blue,
1 white, 2 black, 3 orange). They are also authored for the hi-res Workbench
screen, whose pixels are about twice as tall as wide, so the renders below are
scaled 2× vertically to restore the intended aspect.

`MarbleMadness!.info` (the icon the player double-clicks, Part II) is a **64×29,
2-plane** image — the marble: a dark sphere with a white specular highlight.

![MarbleMadness! icon](rendered/icon-marblemadness.png)

`Disk.info` is the **32×16** disk icon — the familiar white floppy with an orange
label.

![Disk icon](rendered/icon-disk.png)

## 3. The per-course sprite banks (`.ilb`)

Each course carries an `.ilb` (an "image library") holding its sprites — the
marble and that course's creatures and objects. Their sizes track how much each
course needs: the Silly and Ultimate courses share a minimal four-sprite set
(466 bytes), while Aerial packs sixty-three (35 KB).

**The container.** Every `.ilb` has the same layout, confirmed across all six
files: a 4-byte header, a table of fixed 15-byte image descriptors, then the
packed bitmap data.

```
+0   byte   flag (9 on the lighter banks, 13 on the heavier)
+1   word   image count (big-endian)
+3   byte   $01
+4   …      count × 15-byte descriptor records
            └ data section begins at 4 + count*15
```

The image count drives the layout — Silly/Ultimate 4, Practice 10, Beginner 21,
Intermediate 23, Aerial 63 — and in every file the data section begins exactly at
`4 + count×15`, opening with the tell-tale left-aligned bitmap ramp
(`80 C0 E0 F0 F8 FC FE FF`):

```
silly.ilb  (count 4):  data @64  = 00 C0 00 C0 00 80 00 E0 …
aerial.ilb (count 63): data @949 = C0 00 80 00 FE FF E0 00 …
```

**The descriptors.** Each 15-byte record describes one sprite. The first four of
`silly.ilb`:

```
01 00  01 00  21 00 42  FE 00  00 52  F9  00  FF 01
05 00  01 00  21 00 42  FE 00  00 D6  F9  00  FF 01
09 00  01 00  21 00 42  00 00  01 5A  F9  00  FF 01
09 00  01 00  21 00 42  00 00  01 DE  D9  00  08 80
```

Two things stand out. Bytes 9–10 hold a 16-bit value that increases by a constant
**132** from one record to the next (`$0052 → $00D6 → $015A → $01DE`); a constant
stride means a fixed-size slot. And the recurring `21 00 42` is `$21`=**33** and
`$42`=**66** = 2×33. Together they pin the sprite geometry: a 16-pixel-wide object
**33 rows tall at 2 bitplanes** is 33 × 2 bytes × 2 planes = **132 bytes** — a
small blitter object, one per slot. (A 16×16, 4-plane object is also 132; the
descriptor's 33/66 favour the taller form.) The remaining descriptor fields — a
small index that steps `01/05/09…`, a signed `FE 00`, and the trailing flag bytes
— read provisionally as the sprite's placement and draw flags.

**The bitmap data is run-length packed.** The data section is not raw planar
bytes: it interleaves the bitmap ramps with `FE`-prefixed escape codes
(`FE 00 53`, `FE FF E0`, …), so each object is compressed and its packed record is
variable length — which is why the per-sprite byte cost differs so much between
courses. Rendering the data at any fixed width shows recognisable sprite fragments
broken up by those control bytes, confirming it is sprite graphics rather than,
say, a display list.

So the container, the per-sprite 132-byte slot and the object geometry are
established; the one remaining piece is the exact unpacking rule for the
run-length stream, which is what would let each sprite be extracted pixel-perfect
and rendered the way the splash and icons are above.

---

*The per-course `.mlb`, `.vlb`, `Snd` and `Track` formats are to follow.*

---

# Part V — Game mechanics

*To follow.* The marble's physics and controls, the six courses and their
hazards and enemies, the timer, scoring and progression.

---

# Appendix A — Toolchain and reproduction

All work is reproducible from the image with the shared `tools/` module. From
the repository root:

```sh
# 1. List the disk's filesystem and files
go run stupidcoder.com/tools/amiga/cmd/adfdump "Marble Madness (Amiga)/Marble_Madness.adf"

# 2. Extract every file (preserving the directory tree)
go run stupidcoder.com/tools/amiga/cmd/adfdump -x mm-files \
    "Marble Madness (Amiga)/Marble_Madness.adf"

# 3. Disassemble a 68000 code hunk (-skip steps past the HUNK_HEADER to the code)
go run stupidcoder.com/tools/cmd/dis68k -skip 36 mm-files/c/EndCLI

# 4. Recursive-descent trace from an entry point
go run stupidcoder.com/tools/cmd/codetrace68k -base 0 -skip 36 -entry 0 mm-files/c/EndCLI
```

Dynamic verification uses the instruction-level 68000 core in `tools/m68k`
(`m68k.CPU` over a `Bus`), the same way the C64 games are checked against the
`tools/c64/c64` machine model.

The disk image is not committed (it is a copyrighted game); its size and MD5 are
recorded in the repository [README](../README.md#image-files) so the exact copy
can be verified.
