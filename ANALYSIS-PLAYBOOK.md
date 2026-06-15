# Analysis playbook — how these projects were reverse-engineered

A field guide to the methods used across the games in this repository, written so
that a future session (human or AI) can start a new project with a plan instead
of a blank page. It distils what actually worked on two C64 tape games (Elite,
Fort Apocalypse) and one Amiga disk game (Marble Madness), and generalises the
moves that transfer to an unknown medium and unknown formats.

The house rule for all of it: **static analysis first.** No third-party
emulator, debugger, or disassembler; no looking up released source or other RE
projects. Build the tools, read the bytes, disassemble the code, and only reach
for a *self-built* emulator or debugger when the answer is genuinely not on the
medium (state that the running machine/OS produces). When you do, the deliverable
is still *documented understanding plus a clean reimplementation*, never dumped
data.

---

## 0. The shape every project converges on

Both platforms produced the same five-part arc. When you start a new game, this
is your default table of contents — fill it top to bottom, because each part
unlocks the next:

| Part | Question | C64 (tape) | Amiga (disk) |
|------|----------|------------|--------------|
| **I — The image** | How is the medium framed, and what files does it hold? | TAP pulse stream → KERNAL + fastloader encodings → program segments | ADF sectors → OFS/FFS filesystem → catalogued files |
| **II — Boot chain** | What runs first, and how does it bring in the rest? | autostart → bootstrap → turbo loader → multi-stage load | bootblock → dos.library → startup-sequence → launcher → loaders |
| **III — Program architecture** | How does the loaded game decrypt/relocate/initialise itself? | decrypt+relocate, hardware init, IRQ architecture, memory map | multi-stage decrypt, the decryptor, copy protection, static limits |
| **IV — Graphics & data** | What are the asset formats? | charsets, sprites, bitmaps, maps, compression | IFF ILBM, icons, custom sprite banks, level/sound modules |
| **V — Game mechanics** | How do the objects behave? | object tables, movement/AI, collisions, scoring | object/actor structs, movement/physics, state machines, collisions, scoring |

The cells name the *medium-typical* form each part tends to take — not a fixed
checklist, and not the specifics of any one game (those live in the per-game writeups,
§2–§3). Write it up *as you go*, with byte examples and assembly snippets. The writeup
is the deliverable; the tools are how you earn it.

---

## 1. The reusable toolkit

Everything is one Go workspace (`go.work`) over a shared `tools` module plus a
per-game `extract/` module. Platform-neutral tools sit in `tools/`; per-platform
ones in `tools/<platform>/`. The investment that paid off most was a **CPU core
you can both disassemble with and execute**, per instruction-set:

- **Disassemblers, two flavours each — both still earn their keep.** A *linear*
  sweep (`dis6502`, `dis68k`) and a *recursive-descent* tracer (`codetrace6502`,
  `codetrace68k`). Reach for the **tracer** to make a whole 60 KB binary legible: it
  follows control flow from entry points, segments functions, and merges an
  **annotations** sidecar so labels and notes ride along into the listing — feed it
  every entry you find (vectors, jump tables, library stubs). Reach for the **linear**
  sweep for a quick bounded read of a *known* address range, and — the case that
  recurs — for code the tracer marks as **data** because it is only reached by
  computed/indirect jumps (a `JMP (pc,Dn)` dispatch, self-modified vectors). Rather
  than hunting for an entry point to feed the tracer, just linearly disassemble that
  byte range; the two are complements, not a hierarchy.
- **An executable CPU core** (`tools/mos6502`, `tools/m68k`). This is the secret
  weapon. When a loader's bit-framing or a decryptor's key schedule is painful to
  reverse purely on paper, *run it* on your own core with the real input and
  watch what it produces. It is also how you validate a reimplementation
  bit-for-bit. Expect to implement instructions lazily — add them when the target
  code first hits them (Marble Madness needed EXG, the bit ops, NOT/NEG/NEGX
  added to the 68k core mid-project; each came with a unit test).
- **Container/format readers** per platform: `tools/c64/{tap,cbmtape}`,
  `tools/amiga/{adf,hunk,iff,icon}`. Each is a library plus a `cmd/` front-end
  (`tapdump`, `adfdump`, `hunkload`, `amigapng`).
- **Graphics** (`tools/c64/gfx`): palettes, char/sprite/bitmap rendering, PNG and
  APNG output. Rendering an asset is how you *prove* you decoded it.

Principle: **make each finding reproducible from the raw image with one
command.** Git-ignore the regenerable or copyrighted intermediates (`extracted/`,
compiled binaries, the ROM/ADF images themselves); but **do commit** the render
outputs the writeup embeds (`rendered/`) — they are figures *in* the documentation,
reproducible yet part of the deliverable.

---

## 2. Flow A — C64 tape games (Elite, Fort Apocalypse)

The medium is a `.tap`: a stream of **pulse-width** bytes, not bytes of program.
The whole of Part I is turning pulses back into program.

1. **Pulse histogram first (`tapdump`).** Plot the distribution of pulse widths.
   Commodore tapes cluster around a few widths (short/medium/long); the peaks
   tell you the threshold values *this* tape uses. A standard KERNAL tape and a
   custom turbo each have their own characteristic histogram — you can often see
   the boundary in the image where one gives way to the other.
2. **Decode the standard KERNAL part (`cbmtape`).** The first file is almost
   always a normal ROM-loadable bootstrap: pilot tone → sync bytes → header
   (load address, length) → data, with the KERNAL's checksum and the
   pilot/sync/dipole framing. Decoding this gives you the bootstrap program *and*
   teaches you the tape's standard timings as a baseline.
3. **Reverse the fastloader from the bootstrap.** The bootstrap installs a turbo
   loader (often into the tape buffer / low RAM) and hands the tape to it.
   Disassemble it. You are looking for: how it times a pulse (the threshold
   compare), how it frames a **bit** (pulse → 0/1), how it frames a **byte**
   (bit order, sync), how it frames a **block** (count, load address, checksum),
   and where it stores the result. Watch for **self-modifying code** — Elite's
   loader rewrites itself from the tape, so the bytes you disassemble statically
   are not the bytes that run; trace the modifications.
4. **Emulate when framing is ambiguous.** If the exact bit/byte rule is hard to
   pin on paper, run the disassembled loader on the 6502 core with the real pulse
   stream as input and capture what it writes to memory. This is faster and more
   certain than arguing about edge cases, and the C64 machine model
   (`tools/c64/c64`) exists precisely to run hostile loaders.
5. **Extract the segments → `.prg` files.** Now you have the program in memory
   images. Map the multi-stage chain (each stage often loads the next, sometimes
   with a different encoding or under an IRQ).
6. **Program architecture (Part III).** Trace the loaded code: the
   decrypt/relocate step (Elite `SYS 30215 → $7607`; in-place decrypt and
   hand-off), hardware init, and the **interrupt architecture** — C64 games live
   in raster IRQs, so find the IRQ vector chain early; it structures everything.
   Build a memory map.
7. **Assets (Part IV) and mechanics (Part V).** Charsets, sprites
   (packed-column formats appear — Fort Apocalypse), bitmaps, and **compression**
   (Fort Apocalypse uses a table-selective RLE; identify the control-byte scheme
   and the literal runs). Then the object tables and behaviours: the player,
   enemies, bullets, collision matrix, scoring, level progression.

C64-specific instincts: everything is anchored to fixed addresses (`$0314` IRQ
vector, VIC-II `$D000`, SID `$D400`, CIA `$DC00`); banking via `$01` swaps ROM
for RAM under the I/O space; sprites and charsets are page-aligned, which helps
you spot them by address.

---

## 3. Flow B — Amiga disk games (Marble Madness)

The medium is an `.adf`: a flat dump of 1760 × 512-byte sectors. Unlike a tape,
there is usually a real **filesystem**, so Part I is filesystem archaeology, not
signal processing.

1. **Parse the filesystem (`adfdump`).** OFS/FFS: boot block (blocks 0–1, "open
   dos.library"), root block (880), the bitmap (free-space) blocks, and directory
   **hash chains** of file-header blocks. Each OFS file-header block lists its
   data blocks; each **OFS data block** is a 512-byte sector = 24-byte header
   (type 8, header-key, sequence, size, next-pointer, checksum) + 488 payload
   bytes. Extract every file, preserving the tree.
2. **Know storage layout vs content.** The OFS 24-byte-per-sector framing is a
   *storage layout*, not compression — the reader undoes it on extraction. Don't
   confuse a high-entropy *extracted* file (encryption/compression) with the
   on-disk block framing.
3. **AmigaDOS hunk format (`hunk`/`hunkload`).** Executables are hunk files:
   `HUNK_HEADER` (table of segment sizes) then `CODE`/`DATA`/`BSS`, `RELOC32`
   relocation tables, optional `SYMBOL` and `DEBUG`. Load to a flat, relocated
   image for the disassembler. **Keep the `SYMBOL` table if present** — it names
   the routines (Marble Madness's `c/bootscr` shipped 106 symbols and read almost
   like source); a stripped file you name from behaviour.
4. **Boot chain (Part II).** Bootblock → `dos.library` resident → `startup-
   sequence` → Workbench or CLI → the game launcher. Disassemble the launcher
   (`codetrace68k` + the m68k core). Library calls are negative offsets off a
   base in `a6` (exec base at absolute `$4`; `AllocMem = -198`, `FindTask =
   -294`, `DoIO = -456`, …) — resolving those LVOs is most of reading Amiga code.
5. **Encrypted payloads.** Spot them by **entropy** (≈7.95/8 bits/byte) and a
   shared packer signature. Reverse the decryptor as ordinary compiled C: find
   the input layer, the structure walk, and the key schedule. Marble Madness's
   `c/zzz` is a `LoadSeg` replacement whose decode is a seeded table + an
   additive lagged-Fibonacci keystream XORed over *selected* fields (type markers
   and symbol names stay plaintext). **Reimplement the cipher in Go** and validate
   it against known plaintext (a hunk file's header longwords are mostly zero/
   known — derive the keystream from them).
6. **Direct-hardware loaders.** A game may bypass the filesystem for speed and
   protection: Marble Madness's decrypted `c/xxx` drives Paula disk DMA (`$DFF000`
   — `DSKPT`/`DSKLEN` written twice to arm DMA) and the CIAs (`$BFD100` drive
   control, `$BFE001` status) directly, reading raw MFM tracks, finding `$4489`
   sync words, MFM-decoding standard `[$FF,track,sector,gap]` sectors. The tell
   that it reads **by physical position, not by name**: the main data file never
   appears as a filename string in the launcher, yet the disk is 100% filesystem
   (classify every block to prove there's no hidden region). The file is a real
   DOS file so the disk stays bootable and its blocks stay reserved and
   contiguous; it's read by track for speed and as an anti-copy hook.
7. **Graphics (Part IV).** Standard formats decode with stock readers: IFF
   `FORM…ILBM` (BMHD/CMAP/BODY ByteRun1/CRNG cycling) via `iff`; Workbench
   `.info` icons (`$E310` `DiskObject` + planar `Image`s) via `icon`. Custom
   sprite banks you reverse by hand: a fixed-size descriptor table + a packed
   bitmap stream — derive the cell geometry from the descriptor stride, then
   crack the pack codec (or note where it resists static solving because its only
   spec lives in the still-encrypted consumer).

---

## 4. Cross-cutting techniques (the transferable bits)

These are the moves that recur regardless of platform — the ones worth reaching
for on any unknown format:

- **Entropy triage.** Bytes/byte near 8.0 ⇒ compressed *or* encrypted. A packer
  signature header + full-entropy body that decodes to *full-size* output (sizes
  match the header's segment table) ⇒ **encryption, not compression**. Compressed
  data decodes to something *larger* than the stored bytes; encrypted data is the
  same size. (Marble Madness's `.dat`/`c/xxx` are encrypted, full-size — calling
  the loader a "decruncher" was the early mislabel; it is a decryptor.)
- **Structure as a crib.** Most formats keep *some* fields in clear: hunk type
  markers, symbol names, IFF chunk IDs, a file's own size table. Those are
  known-plaintext. For an additive/XOR keystream cipher, known plaintext at a
  known position gives you keystream values directly; solve the generator or the
  table from them.
- **Emulation as an oracle.** When reversing a transform on paper is slow or
  uncertain (loader bit-framing, a key schedule, a self-modifying routine), run
  the real code on your own CPU core with real input and *observe*. This both
  answers the question and gives you a golden reference to test the
  reimplementation against. It is the single highest-leverage tool in the kit.
- **Round-trip verification.** You haven't decoded something until you've
  re-emitted it and checked: relocations apply cleanly, checksums hold, the image
  renders to a recognisable picture, the extractor's output matches a golden
  test. "Looks plausible" is not done.
- **Separate the layers.** Storage framing (filesystem, sector headers, tape
  block structure) ≠ compression ≠ encryption ≠ the asset format underneath.
  Peel them one at a time and name each explicitly; conflating them is the most
  common source of a wrong writeup.
- **Logical vs physical access.** A file in the directory may be read by the game
  through the OS *or* by raw position on the medium. If the loader touches
  hardware directly, ask *where on the medium* it reads, and whether that maps to
  a catalogued file — the answer is often "the same bytes, reached a faster/
  sneakier way," not "secret data."

---

## 5. Reversing behaviour, not just formats

Decoding a *format* (entropy → crib → emulate → round-trip, §4) and understanding
what the code *does* (physics, state machines, game logic) are different modes — and
the second one bit hardest here. The Marble Madness slope/physics trace took many
wrong turns before it converged. What those wrong turns taught, in order of how much
time each would have saved:

- **Trace from the effect backwards; never infer behaviour from the shape of data.**
  The move that finally worked for "what makes the marble roll" was to find the exact
  instruction that *writes the velocity*, then walk backward to its inputs and forward
  to where that data is built. Staring at a byte array and guessing ("this looks like
  a heightmap / these must be the slopes") produced a string of confident, wrong
  stories. If you cannot name the instruction and address that causes the effect, you
  are guessing — say so, or go find it.
- **Decode a concrete instance's real data early; it settles arguments code-reading
  can't.** Hours of reasoning about "the structure" went in circles; decoding one
  actual course's bytes (66 static slope records vs 13 dynamic scripts) settled the
  whole static-vs-scripted question in minutes. Look at the data the code actually
  consumes, for a real instance, not just the code that consumes it.
- **Don't generalise one proven path to the whole system.** A mechanism proven for one
  case (the scripted/dynamic regions, pulling toward a reference point) got written up
  as *the* answer for all slopes — but the static slopes ran an entirely separate code
  path (a height-field gradient). Always ask: does the case I care about actually flow
  through the path I just proved? A plain user question — "is a static slope just a
  one-frame animation?" — exposed the over-reach. Re-derive for each case.
- **Same address/offset ≠ same thing.** One work buffer (`$CCA`) was reused across
  phases at three different strides; one struct offset (`+$1A`) meant different things
  on the marble vs a region vs an enemy. A raw "who writes offset N" scan is therefore
  contaminated — confirm the *base register's identity* (is this pointer really the
  marble?) before trusting a hit.
- **Count from the bound, not by eye.** A dispatch's range check *is* the table size:
  `CMPI.l #$C` ⇒ 12 entries (states 0–11). Reading one slot past the table invented a
  phantom "13th state."
- **A visualisation is a hypothesis test — check it against ground truth.** The
  height-field render became trustworthy only once it reproduced a *known* image (the
  course tilemap) feature-for-feature; doing that comparison caught a real
  horizontal-mirror bug and a profile-indexing error (visible as speckle) that
  code-reading alone had missed.
- **The user's hands-on knowledge outranks a plausible code reading.** "Dizzy" was
  asserted as a death/respawn animation straight from the code; the user, from
  *playing the game*, knew it is a recoverable stun. State game behaviour as fact only
  once it is traced; otherwise flag it as inference and defer to the domain expert in
  the loop.
- **Keep generated tools deterministic, and verify the artefact, not the run.**
  Map-iteration order made PNG output change run-to-run; a `head` on a regenerate
  SIGPIPE-killed it mid-way and left stale files. `git diff`/`status` on the *committed
  output* caught both — sort before emitting, and check what actually changed, not just
  that the command exited 0.

---

## 6. When static analysis legitimately stops — and what then

Some state is simply not on the medium: it is produced by the booted machine.
Marble Madness binds its decryption key to the host **Kickstart ROM** exception-
vector pages *and* the running process's `tc_ExceptCode`/`tc_TrapCode` handler
fields — values that exist only once AmigaDOS has constructed the process. No
amount of reading the disk recovers them.

The escalation ladder, in order of preference:

1. **Boot the ROM on your own core** to read fixed ROM-derived state (a minimal
   reset/early-init run was enough to read the cold-start vector table).
2. **Known-plaintext / linear-algebra** against the cipher to recover as much key
   as the structure pins down (got the header and first segments; stalled where
   too few reliable equations met too many unknowns).
3. **A self-built debugger, as a last resort, to capture *only* the irreducible
   runtime bytes.** For Marble Madness that meant building a headless FS-UAE with
   a GDB stub (`tools/amiga/fsuae-debug/`), booting a real Kickstart, and reading
   ~25 bytes of OS state at decode time — then baking those constants into the Go
   decoder. The algorithm stays reimplemented in Go; the emulator supplies and
   verifies a handful of bytes, nothing more.

Capture tricks that helped: patch the `startup-sequence` in place to auto-run the
target from CLI (no GUI/mouse needed), and read the launcher's *own* task fields
live (CLI commands run in-process, so `FindTask(0)` is the right task). Keep the
debugger itself honest — a working continue/breakpoint/step loop is worth the
effort to fix, because it turns a one-shot guess into repeatable measurement.

---

## 7. A starting checklist for a new, unknown image

1. `file`/`hexdump` the image; measure size; compute an entropy profile across
   it (flat high-entropy regions = packed/encrypted; structured low-entropy =
   headers/tables/code).
2. Identify the **container**: is it a signal stream (tape pulses), a sector dump
   (disk), a cartridge ROM, or a raw program? Build/borrow the container reader.
3. Find the **first code that runs** (autostart vector, boot block, reset
   vector) and disassemble outward from it with recursive descent.
4. Map the **load chain** — what each stage reads, decodes, and hands to the
   next — before diving into any one stage.
5. For any opaque blob: entropy → signature → is it compression or encryption?
   → find and reverse the transform → reimplement → verify by round-trip.
6. Bring up an **executable CPU core** early; use it to run loaders/decryptors
   you can't fully reason about, and to validate every reimplementation.
7. Only when a value is provably not on the medium, escalate to ROM-boot, then
   known-plaintext, then a self-built debugger — capturing the minimum.
8. When the question turns from *format* to *behaviour* (what does this code do?),
   switch modes (§5): trace from the effect — the write to the field that matters —
   backward to its inputs and forward to its data source; anchor every claim to an
   instruction + address; decode one concrete instance's data to check the story; and
   don't assume one proven path covers every case.
9. Write each part up with byte/asm examples as you confirm it. Keep raw images
   and copyrighted ROMs out of git; keep every result one command from
   reproducible.
