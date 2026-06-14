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
sizes are `.b`/`.w`/`.l` (8/16/32-bit). Parts I–III are complete and Part IV is
under way; Part V is still a stub.

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
  - [4. The decryptor (`c/zzz`)](#4-the-decryptor-czzz)
  - [5. The track loader (`c/xxx`)](#5-the-track-loader-cxxx)
- [Part III — Game program architecture](#part-iii--game-program-architecture)
  - [1. The multi-stage load](#1-the-multi-stage-load)
  - [2. The decoder](#2-the-decoder)
  - [3. The copy protection](#3-the-copy-protection)
  - [4. What the static analysis recovers — and where it stops](#4-what-the-static-analysis-recovers--and-where-it-stops)
  - [5. Inside the decrypted program — the code/data split](#5-inside-the-decrypted-program--the-codedata-split)
- [Part IV — Graphics and data formats](#part-iv--graphics-and-data-formats)
  - [1. The splash (boot) screen](#1-the-splash-boot-screen)
  - [2. The Workbench icons](#2-the-workbench-icons)
  - [3. The sprite banks (`.ilb` and `.vlb`)](#3-the-sprite-banks-ilb-and-vlb)
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
are not plain hunks: after a `$0000_03F3 8F01…` header their contents are
near-random (entropy ≈ 7.95 of 8 bits/byte), i.e. **stored encrypted** and
decrypted at load by the small `c/zzz` helper. Decrypting them — and thereby
reaching the main game code — is a Part II/III concern.

**System and boot files** — an ordinary AmigaDOS boot setup:

| file | role |
|------|------|
| `s/startup-sequence` | the boot script (21 bytes): `LoadWb` then `endcli` — it boots to Workbench |
| `c/LoadWb`, `c/EndCLI` | the AmigaDOS CLI commands the script runs |
| `c/splash` | the title/splash screen — an IFF ILBM bitmap (Part IV §1) |
| `c/bootscr` | the boot-screen program (a 50-hunk overlay) that displays the splash (Part II §3) |
| `c/zzz` | the decryptor — decrypts the encrypted files at load (Part III) |
| `c/xxx` | encrypted data decrypted by `c/zzz` (the disk fast-loader, Part II §5) |
| `c/sigfile` | a disk-signature / copy-protection table (`"DOW"`+incrementing bytes) |
| `libs/icon.library` | bundled so the disk shows icons without the user's Workbench disk |
| `MarbleMadness!.info`, `Disk.info`, `.info` | Workbench icons (`$E310` magic, not hunks) |

**The game program:**

| file | size | role |
|------|------|------|
| `MarbleMadness!` | 5,864 | the launcher executable (started from the Workbench icon) |
| `c/MarbleMadness!.dat` | 175,360 | the main game program — stored encrypted, decrypted at load by `c/zzz` |

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
course/level data, `.vlb` vector or obstacle ("`obsc`") graphics, and `*Snd` sound
effects. (`*Track` turns out **not** to be music but the per-course **level/object
layout** — actor placements and sprite-animation scripts, `LoadSeg`'d at course
init; see Part V.)

**Compression and encryption.** Several encodings appear on the disk, decoded in
this writeup where possible. The two executables that matter most —
`c/MarbleMadness!.dat` (the main program) and `c/xxx` — are **encrypted** by a
custom scheme (a shared `$0000_03F3 8F01…` header, contents at ≈ 7.95 of 8
bits/byte — high entropy from the cipher, **not** compression: the bodies are
stored at full size) and decrypted at load by `c/zzz`; reaching the main code
therefore means reversing that decryptor (Part III). The title screen `c/splash` is an IFF ILBM
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
per-course overlays (Parts III–IV).

Tracing the launcher's own code confirms the shape. Loaded into a flat, relocated
image by `tools/amiga/hunk` (whose symbol-table support labels the library stubs
the linker left in the file) and traced with `codetrace68k`, the C-runtime
startup takes the Workbench branch shown above and calls `main`. `main` then
parses the `WBStartup` message, uses its lock to `CurrentDir` into the program's
own drawer, creates a reply port, and proceeds to bring the game in. Because the
game's main file is *encrypted* (Part I §3), that load does not go through plain
`LoadSeg` — it goes through `c/zzz`, the subject of §4.

The full annotated disassembly is in
[`disasm/MarbleMadness.asm`](disasm/MarbleMadness.asm) (with the names and notes
in [`disasm/MarbleMadness.annotations.txt`](disasm/MarbleMadness.annotations.txt)). The boot-screen path is
worth following concretely, because `c/bootscr` is one of the unencrypted hunks
and so loads the ordinary way:

```
LoadSeg("c/bootscr")            ; -> seglist; bail to exit(1) if it fails
Lock("c/splash", ACCESS_READ)   ; does the splash file exist?
  if present:  run bootscr(seglist, 1, "c/splash")        ; paint the splash
  else:        run bootscr(seglist, 1, "lo-res/paintcan") ; fallback image
…                               ; (decrypt/stream the game meanwhile)
run bootscr(seglist, 0, 0)      ; tear the boot screen back down
UnLoadSeg(bootscr)
```

`c/bootscr` is thus a small overlay that the launcher `LoadSeg`s, *runs* via the
seglist-call thunk `call_seglist` (which converts the `LoadSeg` BPTR to an address
and `JSR`s into the first hunk with `d0`/`a0` arguments) — handing it the image
filename so the overlay loads the IFF and displays it — and then runs once more
with `(0, 0)` to dismiss it before `UnLoadSeg`ing it. The same `call_seglist`
thunk is how `c/zzz` and the decrypted `c/xxx` are entered.

One detail is worth pinning down because it looks suspicious: the launcher binary
**also** carries a complete copy of `c/zzz`'s decrypt engine — the key-table
builder, the keystream generator and the vector-keyed protection of Part III. The
copy is the *same compiled code* (the keystream generator, for instance, is
byte-identical to `c/zzz`'s but for the 15 bytes of its three relocated global
pointers). It is tempting to wonder whether the disk `c/zzz` is leftover junk and
this embedded copy is what runs — but it is the other way round. The embedded
decrypt engine is **dead code**: a scan of the whole binary finds *zero*
control-flow references into it. The launcher does the real decryption by
`LoadSeg`-ing `c/zzz` from disk and `JSR`-ing into it (twice — to decrypt `c/xxx`,
then to bring the game in — before `UnLoadSeg`-ing it). What drags the dead engine
in is one routine main genuinely uses, `checksum_seglist`: it EOR-folds the
decrypted `c/xxx`'s hunks into a 16-bit checksum and feeds that into the key array
for the next decrypt — so tampering with `c/xxx` corrupts the key, a small
integrity chain. That checksum routine sits in the same linker object module as
the decrypt engine, so the linker pulled the whole module in even though only the
checksum is called.

## 4. The decryptor (`c/zzz`)

`c/MarbleMadness!.dat` and `c/xxx` are not plain hunks — they are encrypted
(Part I §3), so AmigaDOS's own `LoadSeg` cannot read them. `c/zzz` is the small
program that can: a custom **decrypting `LoadSeg` replacement**. It
is itself a clean hunk, so the hunk loader and `codetrace68k` read it directly,
and its `HUNK_SYMBOL` table even names the system calls it uses — `_Open`,
`_Read`, `_Close`, `_AllocMem`, `_FreeMem`, `_FindTask`. Its flow, from the
disassembly:

- It `_Open`s the packed file and `_AllocMem`s a 512-byte read buffer (it streams
  the input in `$200`-byte blocks) plus a small work buffer.
- A buffered longword reader feeds a core loop that treats the *decoded* stream as
  an ordinary hunk file: it reads a longword, masks off the top two bits
  (`ANDI.l #$3FFFFFFF`), subtracts `$3E7` (`HUNK_UNIT`, the first hunk-block id),
  range-checks it against the 16 block types, and dispatches through a jump table
  — `JMP $2(pc,d0.l)` at `$3DE`. So each `CODE`/`DATA`/`BSS`/`RELOC32`/…/`END`
  block is handled by its own arm exactly as `LoadSeg` would, producing a normal
  relocated segment list (the `ANDI.l #$3FFFFFFF` on the hunk sizes at `$B62` is
  the header pass).
- The decoding is a **keystream XOR**, not compression — the bodies are stored at
  full size; the high entropy is encryption. A 55-entry table is built from the
  seed `$57319753` by a ×31 hash (`sub_BEC`, `sub_D06`); the stream is then an
  additive lagged-Fibonacci generator over that table (`sub_$EAC`). On top of that
  sits a copy-protection routine (`sub_DAA`) that perturbs the table from the
  host's **CPU exception/TRAP vector table** — so the decryption is bound to
  machine state, not just the disk. Part III §2–3 reverses this in full.

So `c/zzz` is where the real game reaches memory: it reads the encrypted `.dat`,
undoes the keyed XOR, relocates the hunks, and hands the loaded segments back to
the launcher, which runs them. Reproducing it as a standalone unpacker — the
prerequisite for disassembling the main game — is the subject of Part III, which
also explains why the copy protection stops a purely static unpack short. The
full annotated disassembly is in [`disasm/zzz.asm`](disasm/zzz.asm)
([`zzz.annotations.txt`](disasm/zzz.annotations.txt)).

## 5. The track loader (`c/xxx`)

Once `c/zzz` is reproduced (Part III) the second-stage file `c/xxx` can be
decrypted and disassembled. Decrypted it is **a small compiled-C program whose
job is to be a custom floppy *track loader* — a "fast loader"** — and it turns
out to be the most interesting piece of code on the disk.

`c/xxx` does **not** read its data through AmigaDOS or even through
`trackdisk.device`'s normal commands. It opens only `dos.library` (for the
allocation/exit plumbing), `FindTask`s itself, then **drives the floppy hardware
directly**:

* **CIA-B PRB (`$BFD100`)** — the drive control port: motor on/off, `/SEL0`
  drive-select, `/SIDE`, and the `/STEP`/`DIR` lines. `seek_track0` ($6055E)
  pulses `/STEP` here to recalibrate the head.
* **CIA-A PRA (`$BFE001`)** — the drive-status inputs: `/RDY`, `/TK0` (track 0),
  `/WPRO`, `/CHNG`. The loader polls these to know when a seek is done and the
  disk is ready, and CIA-A DDRA (`$BFE201`) is set up for them in `disk_hw_init`
  ($6046E).
* **Paula disk DMA (`$DFF000` base)** — `dma_setup` ($6050C) programs `DMACON`
  (`$DFF09A`) and `ADKCON` (`$DFF09E`) for MFM/word-sync mode, and `read_track`
  ($607CA) points `DSKPT` (`$DFF020`) at a track buffer and writes `DSKLEN`
  (`$DFF024`) `= (len/2)|$8000` **twice** — the canonical Amiga "arm disk DMA"
  sequence — to pull a full **`$36F2` (14 066) byte raw MFM track** in one DMA
  burst (with a 200 000-iteration timeout and a `DSKBLK` clear on `INTREQ`).

So `c/xxx` reads raw MFM straight off the platter and decodes it in the CPU,
bypassing the filesystem entirely. That is a classic fast-loader/streaming
design (and a copy-protection hook: a custom track reader can require
non-standard track formats a file copier won't reproduce). Its `main` first
`AllocMem`s the load area sized from the control block the launcher passes, then
`load_session` ($603A2) allocates the `$38`-byte IO struct plus a 512-byte CHIP
sector buffer and walks the track reads. The full annotated disassembly is in
[`disasm/cxxx.asm`](disasm/cxxx.asm)
([`cxxx.annotations.txt`](disasm/cxxx.annotations.txt)); it is produced by the
pure-Go reimplementation of the `c/zzz` decode in
[`extract/cmd/decode`](extract/cmd/decode), which turns the encrypted `c/xxx`
back into a clean 22-hunk AmigaDOS object — using the `c/zzz` copy-protection
inputs captured live from a Kickstart 1.2 machine (the values that defeated a
purely static unpack in Part III §4; the capture method is noted there).

**It loads by physical position, not by name — and that resolves an apparent
paradox.** `c/MarbleMadness!.dat` shows up as an ordinary 175 KB file in the
disk's directory, yet the loader reads tracks, not files. Following the two
through reconciles them. First, the whole disk is a normal AmigaDOS OFS volume:
all 1760 blocks belong to the 50 catalogued files (1661 are OFS data blocks; the
rest are the boot block, 50 file headers, three directories, the root and the
bitmap). There is **no hidden raw region** — whatever the loader reads, it is
reading the filesystem's own sectors. Second, the `.dat` is genuinely one of
those files: following its OFS data-block chain, its 360 data blocks occupy a
near-contiguous physical band at blocks ~1023–1396, i.e. **physical tracks
~93–126**, each block a standard 512-byte sector (24-byte OFS header + 488 bytes
of the encrypted payload; the first block's payload begins `00 00 03 F3 8F 01`,
the packer signature). Third — the clincher — the launcher's *entire* string
table names only `c/zzz`, `c/xxx`, `c/bootscr`, `c/splash` and
`lo-res/paintcan`: **the string `c/MarbleMadness!.dat` does not appear anywhere
in the launcher.** The main program is never opened by name. It is reached purely
by *physical track and sector position* by `c/xxx`, which reimplements
`trackdisk.device`'s read path from scratch — whole-track Paula DMA, find the
`$4489` sync, MFM-decode, and validate each sector's standard `[$FF, track,
sector<11, sectors-to-gap]` header (it even carries the `"trackdisk.device"`
string and an `IORequest` builder as a fallback path). So the loader is *not*
reading data we haven't seen; it is reading the `.dat`'s own bytes off the
platter by location. The `.dat` exists as a DOS file so the disk stays a valid,
bootable AmigaDOS volume and so those blocks are reserved and laid down
contiguously; it is read by position for speed (whole-track DMA beats per-sector
OS reads) and as a copy-protection hook (a from-scratch reader can demand
non-standard formatting, and reading by position bypasses file-level tampering —
reinforced by the `c/xxx` checksum integrity chain of Part II §3).

---

# Part III — Game program architecture

The main program (`c/MarbleMadness!.dat`) and the second-stage loader (`c/xxx`)
are encrypted, so reaching the game's own 68000 code means getting through
`c/zzz`'s decoder (Part II §4) and the copy protection wrapped around it. This
part reverses both completely. The protection binds the key to machine state that
is not on the disk — and §4 documents exactly where a purely disk-only static
attack stops and why. That wall is then cleared in two steps (§4 updates): the
~25 bytes of copy-protection input are read once off a real Kickstart, and the
`.dat`'s count-20 key array is reverse-engineered from the decrypted `c/xxx`.
**Both `c/xxx` and the 150 KB `c/MarbleMadness!.dat` are now fully decrypted**, so
the game body — and the startup/copper/blitter detail inside it — is open for
Parts IV–V.

## 1. The multi-stage load

The launcher (Part II §3) does not load the game in one step. From its
disassembly the sequence is:

1. `LoadSeg` `c/zzz` (a clean hunk).
2. Call `c/zzz` through the seglist-call thunk at `$50A10` — which converts the
   `LoadSeg` BPTR to an address and jumps in with `d0` = a control block and
   `a0` = a filename — to decrypt **`c/xxx`**. The control block here has its
   key-array count set to **0**.
3. The decrypted `c/xxx` seglist is then *run as code*: it is the real
   second-stage loader — the from-scratch floppy track reader of Part II §5. It
   pulls the 175 KB main program off the disk by **physical position** (the
   `.dat` is never opened by name), into a buffer recorded in the shared control
   block.
4. The launcher mutates that control block's key array — count becomes `$14`
   (20 longwords), each XORed with the `c/xxx` checksum (the integrity chain) —
   and runs **`c/zzz` a second time** to *decrypt* the loaded program with that
   count-20 key. The two stages cooperate through the shared control block:
   `c/xxx` is the fast raw reader, `c/zzz` is the decryptor.

So the chain is launcher → (`c/zzz` decrypts `c/xxx`) → (`c/xxx` reads the main
program by track) → (`c/zzz` decrypts it), with the key array changing between
the two `c/zzz` passes. The first pass — `c/xxx` with an empty key array — is the
one this part can read; the second needs the count-20 key (§4).

## 2. The decoder

`c/zzz`'s `LoadSeg` replacement, fully reversed. Its entry point (hunk 0) saves
`d0`→control block and `a0`→filename, opens a library, and calls `main`
(`$400B4`); `main` calls the key setup (`sub_$D06`), then the decode-and-load
(`sub_$2C8`), then frees the table.

**Key setup** (`sub_$D06`): `_AllocMem` 220 bytes; `sub_BEC` seeds a 55-entry
table — `table[0] = $57319753`, `table[i] = table[i-1] × 31 + i` (mod 2³²) — then
XORs in the caller's key array (the launcher's, `count` longwords), then runs the
protection (§3), and records the pointers (`$40EA0/4/8`) the generator reads.

The on-disk format is the key to reading it: a **standard AmigaDOS hunk with
selective encryption**.

- The first longword (`$000003F3`, `HUNK_HEADER`) and every hunk-block **type**
  marker — `$3E9` `CODE`, `$3EA` `DATA`, `$3EB` `BSS`, `$3EC` `RELOC32`, `$3F0`
  `SYMBOL`, `$3F2` `END` — are stored in **plaintext** (read raw, bypassing the
  keystream).
- `SYMBOL`-block symbol **names** are plaintext too — `_AllocMem`, `_FreeMem`,
  `_FindTask`, … sit there as readable ASCII inside the "encrypted" file.
- Everything else — hunk sizes, the relocation tables, and the `CODE`/`DATA`
  bodies — is XORed with the keystream, one keystream longword per stored
  longword, in file order.

There is **no compression**. The bodies occupy their full size on disk; the
apparent size mismatch with the header's hunk table is simply the `BSS` hunks,
which carry a size but no data. The ≈7.95 bits/byte entropy is the encryption,
not packing — which is why `c/zzz` is a *decryptor*, not a decruncher, despite
the packer-like `$0000_03F3 8F01…` header.

The keystream (`sub_$EAC`) is an **additive lagged-Fibonacci generator** over the
table: two indices `p` (start 0) and `q` (start 27); each call does
`table[p] += table[q]; p += 1; q += 2` (both mod 55) and returns the updated
`table[p]`. Being purely additive, the **low byte of every output is a linear
function (mod 256) of the table's low bytes** — the lever §4 pulls.

`extract/cmd/unpack` runs this real code on the `tools/m68k` core, trapping the
six AmigaDOS/Exec stubs and streaming the packed bytes through `_Read`. It
reproduces the keystream bit-exact (verified against an independent generator)
and decodes the header cleanly: `c/xxx` is a `HUNK_HEADER` with **22 hunks**
(`first_hunk = 0`, `last_hunk = 21`) followed by its size/flags table.

## 3. The copy protection

The teeth are in `sub_DAA`, run during key setup. After `sub_BEC` seeds the
table, `sub_DAA` folds bytes from two pieces of **live machine state** into
specific table entries.

From the host's **CPU exception/TRAP vector table** (absolute low memory, the
68000 vectors at `$0`–`$3FF`):

- `$8,$C,…,$20` → table entries 10–16; `$28`–`$38` → entries 10–14 again;
  `$80`–`$BC`, the 16 `TRAP` vectors → entries 32–47;
- from each vector it extracts `(vector >> 16) & 0xFF` (the byte-extractor
  `sub_D92`: `ASR.l #16` then mask) and XORs it into the table entry.

Then from the **running task** (`_FindTask` is called for exactly this — its
result is *not* unused): it reads the task's `tc_ExceptCode` (`$2A(task)`) and
`tc_TrapCode` (`$32(task)`) handler pointers, takes `(ptr >> 16) & 0xFF` of each
(the second further `ASR.l #4`), and XORs them into **table entries 30 and 31**
(`$78`/`$7C` of the buffer). So 25 of the 55 entries are perturbed — 23 from the
vector table, and entries 30/31 from the task's two handler pointers. Those
pointers are *fields read from the running process*, which only exists once
AmigaDOS has created it — they are not on the disk and not present at ROM
cold-start.

Because those entries feed the lagged-Fibonacci generator, **the keystream past
its first stretch depends on the vector table and the task structure**. The
header and the first hunk decode regardless — their keystream words are drawn
before the perturbed entries propagate — which is exactly why the structure
stays legible while the bodies scramble.

What is in those vectors? Booting `kick12.rom` on the same 68000 core
(`extract/cmd/bootrom`) shows that at Kickstart 1.x **cold-start** every
exception/TRAP vector points at a single ROM handler, `$00FC05B4`, so
`(vector>>16)&0xFF` is a uniform **`$FC`** — the `$FC0000` ROM page. Kickstart
2.0+ moved the ROM to `$F80000` (byte `$F8`), so the protection is implicitly
tied to the 1.x ROM layout. That is both ordinary for a 1986 title and a clean
explanation of why such games are Kickstart-version-locked: the decryption key
*is* the ROM page.

The early guess was that AmigaDOS would have redirected those vectors away from
the ROM by decode time, making the key a piece of deep runtime state. The live
capture in §4 shows it does **not**: AmigaDOS leaves the `$8`–`$BC` CPU
exception/TRAP vectors at their ROM handlers, so at decode time every one of them
still reads page `$FC` — the same uniform value as cold-start. The vector table
is therefore **not** the runtime differentiator; it is simply the ROM page, and
the protection is thereby **Kickstart-1.x-locked** (a 2.0+ ROM at `$F8` would key
differently).

The part that genuinely is not on the disk — and not available from the ROM
alone — is entries 30/31: the launcher process's `tc_ExceptCode`/`tc_TrapCode`,
which exec only fills in when it *creates* the process. For this launcher those
default handlers are themselves ROM addresses (captured pages `$FC` and `$FF`,
§4), so the whole key is ultimately ROM-derived — but reading the task fields
requires a booted AmigaDOS with the process actually constructed, which is the
state a purely static unpack cannot manufacture.

One could hope the launcher closes that gap by *installing* its own handlers
before it invokes `c/zzz` — which would put the key values back on the disk. It
does not. The launcher's full disassembly (Part II §3) shows no write to the
`$8`–`$BC` vector region, no `tc_ExceptCode`/`tc_TrapCode` write, and none of the
calls that would install a handler (`Supervisor`, `SetIntVector`, `AddIntServer`,
`SetFunction`); the only references to those task fields anywhere in the binary
are *reads*, inside the dead embedded copy of the engine. So the launcher inherits
the vector table from booted AmigaDOS and hands it, untouched, to the decoder —
the protection's inputs are produced by the running OS, not the disk.

Running the launcher confirms it dynamically. `extract/cmd/runlauncher` executes
the real `MarbleMadness!` on the m68k core in a faked Workbench environment —
trapping the dos/exec calls, serving files and `LoadSeg` out of the ADF — and it
faithfully walks the load chain: open `dos.library`, take the `WBStartup`
message, `LoadSeg` `c/zzz`, and run it on `c/xxx`. c/zzz streams the whole 6 116-
byte file, decrypts the header, and `AllocMem`s all twenty-two of `c/xxx`'s hunks
at sizes that match the static decode to the byte (3 296 = 822×4+8, 1 016 =
252×4+8, …). Then — with the exception vectors sitting at zero, exactly as the
launcher left them — the body decode loses the stream and the decryptor spins.
The header decodes, the bodies do not: the live OS state is the missing key, and
nothing on the disk supplies it.

## 4. What the static analysis recovers — and where it stops

Recovered from the disk alone:

- the complete decode and protection mechanism above;
- `c/xxx`'s shape — a 22-hunk, exec-heavy second-stage loader. Its plaintext
  `SYMBOL` blocks name the calls it imports: `_AllocMem`, `_FreeMem`,
  `_FindTask`, `_AllocSignal`, `_FreeSignal`, `_AddPort`, `_RemPort`,
  `_OpenDevice`, `_DoIO` — a loader that allocates memory and signals, makes a
  message port, and talks to a device (the disk) to stream the game in;
- the `.dat`'s own `HUNK_HEADER` (hunk count and sizes), which decodes the same
  way once its key array is supplied.

Not recovered *by a disk-only static attack* — the bodies (the updates below
then clear this). Two independent attacks were pushed to their limits:

1. **Known-plaintext linear algebra.** The keystream's low byte is linear
   (mod 256) in the 25 protection-perturbed table bytes; the plaintext structure
   (type markers, symbol names, marker-derived hunk sizes, the `0` that
   terminates every `RELOC32` block) yields known keystream values at indices
   that are themselves computable from the marker layout. But only ~14 of those
   equations fall in the region with reliable indices, against 25 unknowns —
   underdetermined.
2. **The vector table from the ROM alone.** Booting `kick12.rom` on the minimal
   core supplies the `$8`–`$BC` vector pages (uniform `$FC`), which decode
   `c/xxx`'s first few hunks — but the decode then loses the stream, because two
   of the 25 perturbed entries (30/31) come from the launcher process's
   `tc_ExceptCode`/`tc_TrapCode`, and at ROM cold-start no such process exists
   yet to read them from. Reproducing the key means booting all the way to a
   constructed AmigaDOS process, not just running the ROM — which the minimal
   CPU core cannot do (it has no AmigaDOS). (The vectors themselves were never
   the obstacle: they stay `$FC` from cold-start through decode time, §3.)

So the protection meets its goal against a static unpack: the payload's
decryption is bound to machine state the disk does not contain — the Kickstart
1.x ROM (via the exception-vector pages) *and*, the true wall, the launcher
process's `tc_ExceptCode`/`tc_TrapCode`, which only exist once AmigaDOS has
constructed the process. The mechanism is wholly understood; the bytes stay
gated behind a running, booted Amiga of the right vintage.

**Update — the gate, opened.** Rather than emulate the full boot in the minimal
core, the missing machine state was read off a real **Kickstart 1.2** under a
GDB-controllable FS-UAE build (the shared Amiga debugger
[`tools/amiga/fsuae-debug/`](../tools/amiga/fsuae-debug); the game-specific
capture scripts are in [`tools/fsuae-debug/`](tools/fsuae-debug)). A CLI auto-run
disk (`modadf.go` rewrites the `s/startup-sequence` in place and
fixes the OFS block checksum) boots straight to the decrypt, and the launcher
runs as the `Initial CLI` task — so its `FindTask(0)` `tc_ExceptCode`/`tc_TrapCode`
read directly off the live task *are* the values `sub_DAA` folds in. The captured
inputs are uniform: **every CPU exception/TRAP vector page byte `$FC`**, launcher
`tc_ExceptCode` page `$FC`, `tc_TrapCode` page `$FF`. With those baked in, a
**pure-Go reimplementation** ([`extract/cmd/decode`](extract/cmd/decode)) — seed
table → key-array XOR → `sub_DAA` perturbation → lagged-Fibonacci keystream →
structure-aware field/body decrypt — turns `c/xxx` (empty key array) into a clean
22-hunk object, verified by the hunk loader applying every `RELOC32` and by the
key array deriving to all-zeros against the known header plaintext. `c/xxx`
proves to be the disk's fast loader (Part II §5).

**Update — the `.dat` decrypted, statically, no second capture needed.** The
main program uses the *same* decode but with a 20-long key array, and recovering
that key array turned out not to need a debugger at all — only the decrypted
`c/xxx` and a 16-bit brute force. The key array is **built by `c/xxx` itself**:
running as the loader, its `run_loader` fills the 20-longword `ctrl->C` buffer
(which then *becomes* the key array for the second `c/zzz` pass) with
`base[i] = datalen / ((i+1) × 300)` — a multiply (`$61248`) then a divide
(`$61204`) over a load-length the loader clamps to ≈1500, giving the small,
non-uniform vector `[4, 2, 1, 1, 0, 0, …]`. The launcher then mutates it in
place, `key[i] ^= C`, where `C` is the 16-bit `checksum_seglist(c/xxx)` integrity
constant (Part II §3). So the only unknown is that single 16-bit `C`: brute it
against a valid-hunk check (`decode -brute -count 20`) and it falls out as
**`C = $CDDA`**. With `key[i] = base[i] ^ $CDDA` and the same captured protection
bytes, `c/MarbleMadness!.dat` decodes to a clean **347-segment, 150 KB** hunk
object — every `RELOC32` applies, the entry hunk disassembles to the textbook
C-runtime startup (`FindTask`, the `pr_CLI`/`$AC` WB-vs-CLI check), and the body
carries the game's own text: `THE ULTIMATE RACE!`, `SCORE:`, the six
`… RACE:` course banners, `GAME OVER`, `RED PLAYER` / `BLUE PLAYER`. The encrypted
game body is open. (Reproduce with `decode -count 20 -datalen 1200 -keyconst
0xCDDA`; both decrypted hunks are written to `extracted/` for analysis. The
captured copy-protection bytes were still needed — they are shared by both
passes — but the count-20 key array itself was reverse-engineered, not captured.)

## 5. Inside the decrypted program — the code/data split

With the body open, the first question is what 150 KB of program actually
contains, and how to separate code from data for analysis. The answer is built
in: the `.dat` is a **stripped AmigaDOS hunk load file**, and the hunk *tags* are
the split. It is **not** linked down to a few merged hunks — each object module
kept its own `CODE`/`DATA`/`BSS` triple, so the file is **347 hunks** (≈115
modules), with no `SYMBOL` or `DEBUG` blocks (routines are unnamed, as in
`c/xxx`). By kind:

| Kind | Hunks | Bytes | What it is |
|------|------:|------:|------------|
| `CODE` | 133 | ~117 KB | the engine — the overwhelming majority |
| `DATA` | 109 | ~24 KB | tables, text, and zero-initialised globals |
| `BSS`  | 105 | ~5.6 KB | uninitialised working storage (no file bytes) |

So the surprise is that the file really **is** mostly code. That is consistent
with the rest of the disk, not at odds with it: the bitmaps and samples live in
separate files (`.ilb`/`.vlb` sprite banks, `Snd`, `Track` — Part I §3), so the
program carries **no pixel or sample data** — only the engine that drives them.
And it drives them at the metal: the code writes the full **blitter** register
block (`$DFF040`–`$DFF066`) and `DMACON` (`$DFF096`) directly and reads
`JOYxDAT` for the trackball/joystick, so rendering is hand-rolled blitting, not a
library call.

The ~24 KB of `DATA` is smaller than it looks: roughly **16 KB is zero** —
working arrays the compiler emitted into `DATA` rather than `BSS` (one hunk is
7 352 bytes with *two* non-zero bytes; another is 7 188 bytes at 99 % zero). The
remaining ~8 KB is the real payload, and it is exactly what an engine-without-
assets would hold:

- **UI text** — `THE ULTIMATE RACE!`, `SCORE:`, `CONGRATULATIONS!`, `TOTAL`,
  `GO!`, `Difficulty:`, `GAME OVER`, `RED PLAYER` / `BLUE PLAYER`, and the six
  course banners (`PRACTICE`/`BEGINNER`/`INTERMEDIATE`/`AERIAL`/`SILLY`/`ULTIMATE
  RACE:`);
- **the level filenames** — `Practy.mlb`, `Beginr.mlb`, `Interm.mlb`,
  `Aerial.mlb`, `Silly.mlb`, … — i.e. the engine names and loads the per-course
  `.mlb` level modules at run time (the file-vs-track loading of Part II §5);
- **small lookup tables** — offset/coordinate/sine-like runs and a few pointer
  tables, the rest of the `DATA` hunks.

Cleanly splitting it for further work needs no extra tooling: `hunkload` prints
the per-hunk map (kind, address, size) — that *is* the code/data manifest — and
`codetrace68k`, seeded with every `CODE`-hunk base as an entry point, follows the
control flow and classifies reached bytes as code and the rest as data. A first
pass reaches **~91 KB of code in 393 routines**; the remaining ~28 KB of `CODE`
sits behind 11 indirect jump tables that need `-table` hints to resolve — the
starting point for the mechanics analysis of Part V.

```sh
# split + first-pass disassembly of the decrypted engine
go run stupidcoder.com/tools/amiga/cmd/hunkload -base 0 \
    extracted/c_MarbleMadness.dat.decrypted.hunk            # the hunk/code-data map
go run stupidcoder.com/tools/amiga/cmd/hunkload -base 0 -o /tmp/dat.bin \
    extracted/c_MarbleMadness.dat.decrypted.hunk            # flat relocated image
# -entry = every CODE-hunk base from the map above
go run stupidcoder.com/tools/cmd/codetrace68k -base 0 -entry <CODE-hunk bases> /tmp/dat.bin
```

---

# Part IV — Graphics and data formats

The boot-time graphics use standard Amiga formats the toolchain already reads end
to end — the title splash (§1) and the Workbench icons (§2). §3 turns to the
game's own formats: the sprite banks (`.ilb` course scenery and `.vlb` moving
objects), whose container, geometry, and **compression** (ByteRun1/PackBits,
identified from the decrypted engine — shared with the `.mlb` levels) are all
reversed here. The remaining per-course modules (`Snd`, `Track`) are still to be
decoded.

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
two. `c/bootscr` keeps its full `HUNK_SYMBOL` table, so its annotated
disassembly ([`disasm/bootscr.asm`](disasm/bootscr.asm)) reads almost like
source: it is built from the **EA IFF reader** (`_GetFoILBM`, `_GetBODY`,
`_UnPackRow` = ByteRun1), a **trackdisk mini-filesystem** that reads the disk's
sectors directly (`_MFOpen`/`ReadSecs`/`_MGetDir`), and **graphics.library**
display setup (`_MakeVPort`/`_LoadRGB4`/`_LoadView`). (A loose end alongside
them, `c/sigfile`, is not graphics either: it is a short table of
`"DOW"`+incrementing-byte entries, a disk-signature / copy-protection artefact.)

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

## 3. The sprite banks (`.ilb` and `.vlb`)

Each course carries an `.ilb` (an "image library") holding its static sprites,
and the moving objects live in matching `.vlb` files — `marbdat.vlb` (the
marble), `birdink.vlb` and `ooze.vlb` (creatures). Both extensions are the **same
format**; only the sprite geometry and counts differ. The `.ilb` sizes track how
much each course needs: the Silly and Ultimate courses share a minimal four-sprite
set (466 bytes), while Aerial packs sixty-three (35 KB).

**The container is one whole-file ByteRun1 stream — exactly like the `.mlb`.** An
earlier reading of the *raw* file (a 4-byte header + a table of "15-byte"
descriptors + packed bitmap, with the data section at `4 + count×15`) decoded the
sprites *correctly* but for the wrong reason: that computed offset just happened to
land on a PackBits control boundary, so the unpack re-synced. The honest model —
confirmed from the loader `$80B4` — is that the bank is **packed end-to-end** and
the structure only exists **after** unpacking. The loader does exactly that:
`dos.library Open(MODE_OLDFILE)`/`Read`/`Close` (**not** the track loader, **not**
a decompression library), then `$9118` ByteRun1-expands the *whole file* into a
work buffer, and walks the descriptors **in that buffer**.

```
unpacked buffer:
+0    word              cell count
+2    count × 20 bytes  cell descriptors (loader walks these at stride $14)
...                     contiguous planar pixel data the +8 fields point at
```

**The descriptors (20 bytes, in the unpacked buffer).** The first of `silly.ilb`,
read from the unpacked buffer — note it is *clean*, with none of the stray `$FE`
control bytes that the old raw-file read showed (those `$FE` were PackBits repeat
codes the raw read was straddling):

```
+0  byte   cell type      (1 = stored cell, 0 = engine-composited cell)
+1  byte   flags           (bit 0 = process this cell)
+2  word   width  in 16-px words   (silly: 1 → 16 px)
+4  word   height in rows          (silly: 33)
+6  word   one-plane byte size     (= width*2 * height; silly: 66)
+8  long   source offset into the unpacked buffer (relocated +buf at load)
+C  long   dest cell pointer       (0 in file; allocated from buffer slack at load)
+10 long   aux pointer             (0 in file; relocated at load)
```

The cells' pixel data is **contiguous and in `+8` order**, so a cell's source span
is the gap to the next cell's `+8` (or the buffer end), and the **plane count is
that span ÷ the `+6` one-plane size**. This is the key correction over the old
"everything is 2 bitplanes" reading: plane count **varies per cell**. Silly's
cells are 132 ÷ 66 = **2 planes** (16×33, 4-colour markers), but Practice's big
scenery blocks are 1920 ÷ 480 = **4 planes** (80×48, 16-colour), and a single
`.ilb` freely mixes both (e.g. `interm.ilb`: 16×33×2p markers alongside 96×64×4p,
80×64×4p and 48×79×4p scenery). `composite_planes` `$8026` confirms the storage:
it consumes the planes as **sequential `+6`-byte blocks** (not row-interleaved).

**The `.vlb` moving-object banks** are the same format (a `$0D` "heavy" flag vs the
`.ilb` `$09`/`$0D`), the same count word and 20-byte descriptors. `marbdat.vlb`
holds **130** cells — the marble at every rotation step, plus its mask cells —
while `birdink.vlb` (45) and `ooze.vlb` (48) are creature cycles; the obstacle
banks (`*obsc.vlb`) are uniform 16×64×2p strips.

**The codec is ByteRun1 / PackBits** — the *same* RLE as IFF ILBM bodies (Part IV
§1's `_UnPackRow`), routine `$9118`:

```
read a signed control byte n:
  n in   0..127 : copy the next n+1 bytes literally
  n in -127..-1 : repeat the next single byte (1 - n) times   (2..128)
  n == -128      : no-op
```

My earlier guess of `$FE` as a literal escape was the mistake that comes from
reading the *packed* stream as if it were structured: `$FE` is signed `-2`, i.e.
"repeat the next byte 3×". Unpacking the whole file with the signed PackBits rule
consumes each bank **exactly** to its last byte and yields the clean count +
descriptor + pixel layout above. The `.mlb` level modules go through the **same**
`$7E4E`→`$9118` load path with a different post-unpack header (flag `$01`, an
offset table), so all three bank kinds share one codec; only the unpacked layout
differs.

With the model corrected, [`extract/cmd/sprites`](extract/cmd/sprites) unpacks the
whole file, reads each 20-byte descriptor, derives per-cell width/height/plane
count, and flow-packs the cells (which now vary in size) into one sheet per bank in
[`rendered/`](rendered) (`<bank>.png`). The 4-plane scenery cells now render in
full 16 colours — `aerial.ilb` resolves into the recognisable isometric ramps,
hazard walls, columns and the marble-muncher; `marbdat.vlb` into the marble's
rotation frames (the mask cells render as their raw planes, pending the cell+mask
pairing).

**The palette is per-course, and it lives in the `.mlb` level module — not in the
program.** The engine sets colours with `graphics.library SetRGB4` (`$248FC`, via
the ViewPorts at `$1FA82`/`$21170`), never `LoadRGB4`, and the `.dat` itself holds
no colour table — which is why a search of the decrypted program comes up empty.
The source is each course's **`.mlb`**: sixteen big-endian `$0RGB` words, the
playfield palette (colours 0–15), at **offset `0x16` of the whole-file-unpacked
buffer** (the `.mlb` decode below). They also appear raw at file offset `0x17` as
a PackBits literal run — which is a usable shortcut for most courses but a trap for
beginner, whose file bytes there hold a bogus `$EE00` while the real unpacked value
is `$0000`; reading from the unpacked buffer is the correct source. This was
confirmed two ways: the six `.mlb` files carry six distinct palettes, and reading
the live OCS colour registers (`$DFF180`) on the training course under FS-UAE
returned **`000 333 444 666 999 BBB DDD 822 C60 CC0 622 A22 D33 F88 …`** —
byte-for-byte the words in `practy.mlb`. The first seven entries are a shared grey
ramp (the isometric scenery shading); 7–15 are the course's own object/accent
colours (practy red-orange, beginr blue, interm green, aerial tan, silly yellow,
ultima red). [`extract/cmd/sprites`](extract/cmd/sprites) reads each course palette
from the matching `.mlb` and renders every bank with it into
[`rendered/`](rendered).

**Two palette entries are engine-animated, not static.** Colours 14–15 are a
small **colour-cycling effect range** the engine drives at runtime (the animated
hazard/ice pools) — the static `.mlb` palette only holds a base/placeholder there
(beginner zeroes them, so its ice trap reads black in a static render rather than
the cycling cyan seen in-game). The cycling is generated by the display task the
main loop signals (`$1F392` → a separate task), not from any static colour table
in the `.dat`, so it is out of reach of a static palette extraction.

**The bit depth varies per cell, not per file.** Within an `.ilb`/`.vlb` bank the
small markers, the marble and the creatures are **2 bitplanes** (4 colours, the
palette's low four), while the larger scenery blocks are **4 bitplanes** (16
colours) — each cell's plane count is read straight from its descriptor (source
span ÷ the `+6` one-plane size, §3 above). The `.mlb` **level modules** are
uniformly 4 bitplanes (16 colours): the loader (`$7F38`) relocates four plane
pointers and a palette pointer at the head of its work buffer.

**The `.mlb` format, decoded from the consumer.** The whole file is **one
ByteRun1/PackBits stream**. The loader (`$7F38`) unpacks it into a work buffer and
relocates four plane pointers and the tilemap pointer that sit in the buffer's
header; the tile blitter (`$9910` → `$99C0`) then draws the course. The unpacked
buffer's layout (silly.mlb shown):

| Buffer offset | Field | silly value |
|---|---|---|
| `+0x00` word | tile count | `0x008F` (143) |
| `+0x02` long | plane-0 offset | `0x000036` |
| `+0x06` long | plane-1 offset | `0x001D76` |
| `+0x0A` long | plane-2 offset | `0x003AB6` |
| `+0x0E` long | plane-3 offset | `0x0057F6` |
| `+0x12` long | tilemap offset | `0x007536` |
| `+0x16 … +0x36` (32 B) | **palette** | 16 `$0RGB` words (also raw at file `0x17`) |
| `+0x36 … +0x1D76` | **plane 0** (7488 B) | tile bitplane 0 |
| `+0x1D76 … +0x3AB6` | plane 1 | tile bitplane 1 |
| `+0x3AB6 … +0x57F6` | plane 2 | tile bitplane 2 |
| `+0x57F6 … +0x7536` | plane 3 | tile bitplane 3 |
| `+0x7536 … +0x9DB6` | **tilemap** (10368 B) | 144 × 36 tile-index words |

(The four plane offsets differ per course but the plane-0 offset is always `0x36`,
and the planes are a constant stride apart — that stride is the plane size, so the
tile count = stride/8. The header palette appears raw in the file at `0x17` because
it is stored as a PackBits literal run.)

**Tiles** are **8×8, 4 bitplanes (16 colours)**. The blitter reads each tile as
eight 1-byte rows at `plane[(i>>1)*16 + (i&1) + 2*r]` — i.e. **even/odd tiles are
byte-interleaved** within 16-byte groups, rows at the even byte offsets — and
composites the four planes through the course palette. Tile 0 is the all-black
tile in every course.

**The full courses, assembled.** The **tilemap** is a row-major stream of
big-endian tile-index words; the blitter's 72-byte row stride fixes the course
**width at 36 tiles (288 px)**, and because Marble Madness scrolls vertically that
width is constant while the height varies — practice 36×75 up through ultimate
36×198. Placing each 8×8 tile by its index reproduces each **complete course**:
[`extract/cmd/sprites`](extract/cmd/sprites) emits the course to
[`rendered/`](rendered) as `<course>.png` and the tile set as `<course>.tiles.png`.
The practice course renders as its grey isometric checkered floor, red walls,
yellow/orange railings and the `GOAL` banner; all six courses come out in their
real per-course colours. (Reading the offsets and tilemap pointer from the proper
*whole-file-unpacked* buffer header — rather than guessing them from the raw file —
is what makes every course correct; an earlier model that unpacked only from `0x37`
mis-aligned beginner and silly.) Cracked end to end: container → ByteRun1 →
8×8×4 tiles → palette → tilemap → course image.

A second `.ilb`/`.vlb` refinement remains. **Exact cell boundaries:** the loader
expands each
15-byte file descriptor into a 20-byte in-memory record and relocates its
source/dest pointers (`$80B4`), and some banks (notably the marble) pair each cell
with a blitter **mask**, so a frame stride is twice the cell — the current
extractor slices on the cell size, exact for the opaque scenery but half-aligned
on the masked objects. Following the descriptor offsets and `$8026`'s
OR-compositing precisely is the last step to per-frame-perfect sprites.

---

*The per-course `.mlb`, `.vlb`, `Snd` and `Track` formats are to follow.*

---

# Part V — Game mechanics

## 1. The object/actor system and the `*Track` level files

The moving things in a course — the goal flag, enemies, the marble-munchers, the
animated obstacles — are driven by an **actor system** in the engine, fed by the
per-course **`*Track` file**. `*Track` is **not** music (that is `*Snd`); it is the
**level/object layout**: where each object starts and how it animates.

**Loading (`load_track_data $003176`).** At course init the engine indexes a
6-entry table `t_track_names` (`$353C`, keyed by the course global `$5D6`) holding
the names `PrcTrack`/`BegTrack`/`IntTrack`/`AerTrack`/`SilTrack`/`UltTrack`,
**`LoadSeg`s** the file (`dos.library` −150, via `$8CF0`), and stores the seglist
at `$3554`. The loaded segment opens with a header of **ten longword pointers**
(seg `+0 … +$24`) that the engine fans out to the actor-system globals — notably
`+$14 → $FD2C`, the **animation-script table**.

**Actors (`actor_update $01D3B2`).** Each frame the engine walks an array of
**20-byte actor records**:

```
+0  long   pointer to the current sprite CELL to draw
+4  long   animation-script pointer (8-byte entries: cell ptr + timing,
           $FFFFFFFF-terminated → loop); advanced when the frame timer expires
+8  word   frame timer
+A  word   x position
+C  word   y position (biased by the scroll offset $D30C)
+E/+10     animation / wrap state
```

So a cell is grouped into an animation by the **script** (a list of cell pointers
in the Track segment), and the actor carries the **position**. Frame durations are
randomised through the engine RNG (`$8F96`).

**Drawing (`draw_object_wrap $0104C4` → `blit_object $011D1C`).** There are **no
hardware sprites** (the program never touches `$DFF0A0–$DFF0DC`); everything is
blitted. The playfield is **4 bitplanes**, so every on-screen object is 16-colour.
`blit_object` reads the cell's width/source from its descriptor and loops the
screen bitmap's plane-pointer table for **four planes**, advancing the source by
one plane-size each step — i.e. it consumes **four consecutive one-plane blocks**
from the cell source. `draw_object_wrap` adds vertical wraparound for the 512-tall
scroll buffer (two blits across the seam).

**Consequence for the sprite banks (corrects Part IV §3 rendering).** Because the
blit pulls four planes and each `.ilb`/`.vlb` descriptor cell physically stores
only two, a 16-colour object is built from **two consecutive 2-plane cells** — the
goal flag is cells 0+1 (and 2+3 = a second wave frame), and the marble is an image
pair. This is why those cells render grey and "interlaced" when drawn one
descriptor at a time. The grouping is **not** a simple "pair every two cells",
though: the marble bank interleaves image cells with mask/shadow cells, so naive
pairing scrambles it. The correct cell→object grouping comes from the Track
segment's animation scripts (the cell-pointer lists), which is the next decode.

## 2. The per-course object-placement table

The Track header's field `+4` (`$129FC`) points to a small struct whose first
field is the **object-placement list**: an array of **3-byte records**, terminated
by a leading `$FF`:

```
+0  byte   X   (isometric grid cell; screen seed = X*8+4)
+1  byte   Y   (isometric grid cell; screen seed = Y*8+4)
+2  byte   type (object kind, 0..7)
```

The consumer (`place_objects $0122AC`) walks the list, filters by `type`, seeds the
screen position (`$6C4`/`$6C6`) and runs the isometric transform (`$6718`). Decoded
straight from the Track files by [`extract/cmd/tracks`](extract/cmd/tracks):

| Course | Track | Objects placed |
|---|---|---|
| Practice | `PrcTrack` | 59 |
| Beginner | `BegTrack` | 79 |
| Intermediate | `IntTrack` | 87 |
| Aerial | `AerTrack` | 159 |
| Silly | `SilTrack` | 104 |
| Ultimate | `UltTrack` | 144 |

What each `type` (0–7) *is* — which sprite/animation and behaviour, including the
marble's start cell and the enemies — is held in the other Track header pointers
(the per-type object/animation definitions, e.g. the animation table at `$FD2C`).
Mapping `type` → object is the next decode; it will also pin the correct
cell→object grouping that fixes the flag/marble render with no guessing.

## 2. Physics, controls, scoring

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
