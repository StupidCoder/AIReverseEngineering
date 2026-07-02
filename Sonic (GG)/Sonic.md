# Sonic the Hedgehog (Game Gear) — cartridge format and game analysis

A reverse-engineering reference for `Sonic The Hedgehog (Japan, USA).gg`, the Sega
Game Gear release of Sonic the Hedgehog. This is the first Z80 / Sega title in this
repository — and the first cartridge ROM rather than a tape or disk — and the
writeup follows the same shape as the C64 and Amiga games, in reading order:

* **Part I** — the cartridge image: the flat ROM dump, the Game Gear's memory map,
  the bank-switching mapper, and the cartridge header;
* **Part II** — boot and initialization: the Z80 reset sequence, the VDP, RAM and
  mapper setup, and the path to the main loop;
* **Part III** — engine architecture: the main loop, interrupt handling, the RAM
  layout and how banked resources are reached;
* **Part IV** — graphics and data formats: the VDP tile/tilemap/palette/sprite
  encodings and the level and object data;
* **Part V** — game mechanics: Sonic's physics, the objects, the zones, scoring
  and progression.
* **Appendix** — toolchain and reproduction.

Methods: purely static analysis of the ROM image, plus the Z80 toolchain built for
it in the shared `tools/` module — the disassemblers (`tools/cmd/disz80`,
`tools/cmd/codetracez80`) over the `tools/z80` decoder. All addresses are Z80
addresses (16-bit, `$0000`–`$FFFF`) unless a *file offset* is called out; bytes are
8-bit. Parts I–III are complete and Part IV is under way; Part V is stubbed.

---

## Contents

- [Part I — The cartridge image](#part-i--the-cartridge-image)
  - [1. The ROM dump](#1-the-rom-dump)
  - [2. The Z80 address space and bank switching](#2-the-z80-address-space-and-bank-switching)
  - [3. The memory map](#3-the-memory-map)
  - [4. The cartridge header (`TMR SEGA`)](#4-the-cartridge-header-tmr-sega)
  - [5. The CPU vectors](#5-the-cpu-vectors)
  - [6. What's in each bank](#6-whats-in-each-bank)
- [Part II — Boot and initialization](#part-ii--boot-and-initialization)
  - [1. Cold-start init (`$0296`)](#1-cold-start-init-0296)
  - [2. Cross-bank calls and the `RST` gateways](#2-cross-bank-calls-and-the-rst-gateways)
  - [3. The frame-interrupt handler (`$0073`)](#3-the-frame-interrupt-handler-0073)
  - [4. The main entry (`$1356`)](#4-the-main-entry-1356)
- [Part III — Engine architecture](#part-iii--engine-architecture)
  - [1. The attract loop and the scene state machine](#1-the-attract-loop-and-the-scene-state-machine)
  - [2. The title, and waiting for Start](#2-the-title-and-waiting-for-start)
  - [3. The world map (decoded by tracing, not the oracle)](#3-the-world-map-decoded-by-tracing-not-the-oracle)
  - [4. How far pure tracing reaches — and the level-load frontier](#4-how-far-pure-tracing-reaches--and-the-level-load-frontier)
- [Part IV — Graphics and data formats](#part-iv--graphics-and-data-formats)
  - [1. The VDP formats](#1-the-vdp-formats)
  - [2. The graphics decompressor](#2-the-graphics-decompressor)
  - [3. The opening screens: how they are built](#3-the-opening-screens-how-they-are-built)
  - [4. Level maps: how a zone is stored and drawn](#4-level-maps-how-a-zone-is-stored-and-drawn)
- [Part V — Game mechanics](#part-v--game-mechanics)
  - [1. Objects](#1-objects)
  - [2. Movement and collision](#2-movement-and-collision)
- [Appendix A — Toolchain and reproduction](#appendix-a--toolchain-and-reproduction)

---

# Part I — The cartridge image

A cartridge is the simplest image format in this repository. There is **no
container, no filesystem and no loader** — unlike the C64 tape (a pulse stream you
have to decode) or the Amiga disk (an AmigaDOS filesystem you have to walk). The
`.gg` file is a verbatim copy of the cartridge's mask-ROM chip: byte *N* of the
file is exactly the byte the Z80 reads from the chip at ROM offset *N*. So Part I
is short — there is nothing to *extract*. The only real structure is the **memory
map** the console imposes on those bytes (because the ROM is bigger than the CPU
can address at once) and a small **header** Sega stamps near the front.

## 1. The ROM dump

The image is **262,144 bytes = 256 KB = 2 Mbit**, an exact power of two. It carries
**no 512-byte copier header** (some circulating `.sms`/`.gg` dumps prepend one; this
one does not — the size is a clean power of two and the Sega header lands exactly at
its canonical offset, [§4](#4-the-cartridge-header-tmr-sega)). The exact copy this
analysis is based on is pinned by size and MD5 in the repository
[README](README.md#image-files).

That's the whole "format". Everything else in this part is about how the **console**
sees those 256 KB.

## 2. The Z80 address space and bank switching

The Game Gear's CPU is a Zilog Z80 with a **16-bit address bus**, so it can only
address **64 KB at a time**. The cartridge holds **256 KB**, four times that. The
ROM therefore cannot be mapped flat; it is divided into **16 banks of 16 KB**
(bank *b* = file offset `b × $4000`), and a small mapping circuit — the standard
**Sega memory mapper** — pages a chosen bank into one of three 16 KB *slots* in the
low 48 KB of the Z80's address space. The top 16 KB is the console's work RAM.

Which bank is visible in each slot is selected by writing the bank number to one of
four mapper registers, which live at the very top of the address space:

| Register | Effect |
|---|---|
| `$FFFC` | mapper control — cartridge-RAM enable / which RAM bank maps into slot 2 |
| `$FFFD` | bank number for **slot 0** (`$0000`–`$3FFF`) |
| `$FFFE` | bank number for **slot 1** (`$4000`–`$7FFF`) |
| `$FFFF` | bank number for **slot 2** (`$8000`–`$BFFF`) |

Those registers physically *are* the top four bytes of work RAM (the RAM is mirrored
into `$FFFC`–`$FFFF`), so a write both stores the byte and reprograms the mapper. At
reset the slots default to banks **0 / 1 / 2**, which is why the first 48 KB of the
ROM is the natural place for boot and core code. One important subtlety: the **first
1 KB (`$0000`–`$03FF`) is hard-wired to bank 0** and is *not* affected by `$FFFD`, so
the CPU vectors and the mapper-setup code below them are always reachable no matter
how slot 0 is paged.

For reverse engineering, this means a disassembler has to be told *which bank
configuration* it is looking at. The `tools/cmd/disz80` linear disassembler takes a
file offset and the Z80 address it is mapped to (`-off … -base …`), and
`tools/cmd/codetracez80` traces one ≤64 KB configuration at a time; following calls
*across* a bank switch is a higher-level concern handled when the code is analysed
(Part II onward).

## 3. The memory map

Putting the mapper together with the console's RAM and I/O, the Z80 sees:

| Z80 range | Size | Contents |
|---|---:|---|
| `$0000`–`$03FF` | 1 KB | ROM **bank 0, fixed** (CPU vectors; never paged) |
| `$0400`–`$3FFF` | 15 KB | ROM **slot 0** (bank from `$FFFD`, default bank 0) |
| `$4000`–`$7FFF` | 16 KB | ROM **slot 1** (bank from `$FFFE`, default bank 1) |
| `$8000`–`$BFFF` | 16 KB | ROM **slot 2** (bank from `$FFFF`, default bank 2) — or cartridge RAM |
| `$C000`–`$DFFF` | 8 KB | **work RAM** |
| `$E000`–`$FFFB` | ~8 KB | work-RAM **mirror** of `$C000`–`$DFFF` |
| `$FFFC`–`$FFFF` | 4 B | **mapper registers** (in the RAM mirror; see §2) |

The graphics and sound hardware is *not* in this memory map — the Z80 reaches the
VDP and the PSG through the **I/O ports** (`IN`/`OUT`), which is exactly what the
reset code does (`IN A,($7E)` reads the VDP V-counter; see §5 and Part II). The
ports relevant here:

| Port | Direction | Use |
|---|---|---|
| `$00`–`$06` | write | Game Gear registers (start button, **stereo** sound control, …) |
| `$3E` | write | memory-control (enable/disable I/O, BIOS, RAM, card, …) |
| `$3F` | write | I/O port control (joypad TH lines) |
| `$7E`/`$7F` | read/write | VDP **V-counter / H-counter** (read) and **PSG** (write) |
| `$BE` | read/write | VDP **data** port |
| `$BF` | read/write | VDP **control/status** port |

(The Game Gear's 8 KB of work RAM is the only general-purpose RAM; there are no
hardware sprites' worth of extra RAM — the VDP's 16 KB VRAM and 64-byte CRAM are
addressed indirectly through the VDP data/control ports, covered in Part IV.)

## 4. The cartridge header (`TMR SEGA`)

Sega stamps a 16-byte header into the ROM at **`$7FF0`** — the last 16 bytes of the
first 32 KB, i.e. the tail of bank 1, a region always present in slots 0–1 at boot.
(The hardware also allows it at `$1FF0` or `$3FF0` for smaller ROMs; a 256 KB ROM
uses the canonical `$7FF0`.) Its purpose on the original hardware is the Master
System / export BIOS region+checksum check; the Game Gear has no such BIOS gate, so
the field is informational here. The bytes in this ROM:

```
$7FF0: 54 4D 52 20 53 45 47 41   "TMR SEGA"   8-byte magic
$7FF8: 00 00                      reserved
$7FFA: 00 00                      checksum (LE word) = $0000  (unused on GG)
$7FFC: 08 24 00                   BCD product code + version
$7FFF: 60                         region (hi nibble) + ROM-size code (lo nibble)
```

Decoded:

| Field | Bytes | Value | Meaning |
|---|---|---|---|
| Magic | `$7FF0`–`$7FF7` | `"TMR SEGA"` | identifies a Sega cartridge header |
| Checksum | `$7FFA`–`$7FFB` | `$0000` | left blank — the Game Gear never verifies it |
| Product code | `$7FFC`–`$7FFE` hi | BCD `…2408` | catalogue number (BCD digits, little-endian) |
| Version | `$7FFE` lo nibble | `0` | revision 0 |
| Region | `$7FFF` hi nibble | `6` | **Game Gear, export/international** |
| ROM size | `$7FFF` lo nibble | `0` | size code `$0` = **256 KB** — matches the file |

The region nibble distinguishes the platform/region the same way across all Sega
8-bit carts (`3` = SMS Japan, `4` = SMS Export, `5` = GG Japan, `6` = GG Export,
`7` = GG International); the `6` here is consistent with the "(Japan, USA)" dump
name. The ROM-size nibble (`$0` ⇒ 256 KB) agreeing with the actual 262,144-byte
file is a useful sanity check that the dump is whole and un-padded.

## 5. The CPU vectors

Because the first 1 KB is fixed to bank 0 (§2), the Z80's hard-wired entry points
all live at the bottom of the ROM and are always reachable. The Z80 has a fixed
reset address, eight one-byte `RST` call targets spaced 8 bytes apart, a maskable
interrupt vector and a non-maskable interrupt vector:

| Address | Vector | This ROM |
|---|---|---|
| `$0000` | **reset** (power-on / `RST $00`) | the boot sequence (below) |
| `$0008`–`$0030` | `RST $08`–`RST $30` call targets | the ones Sonic uses (`$18`/`$20`/`$28`) are each a `JP` to a common routine; the rest are unused/overlapped |
| `$0038` | **maskable interrupt** (`IM 1`) / `RST $38` | `JP $0073` (the VDP frame-interrupt handler) |
| `$0066` | **NMI** (the **Start/Pause** button) | the pause handler |

The reset code is the textbook Master System / Game Gear opening — disable
interrupts, select interrupt mode 1, busy-wait on the VDP until the raster reaches a
known line, then jump to the real initialization:

```
$0000  F3        DI               ; mask interrupts
$0001  ED 56     IM 1             ; mode 1 → INT vectors through $0038
$0003  DB 7E     IN A,($7E)       ; read the VDP V-counter
$0005  FE B0     CP $B0           ; reached scanline $B0?
$0007  20 FA     JR NZ,$0003      ; no → keep polling
$0009  C3 96 02  JP $0296         ; → main initialization (Part II)
```

The `RST` slots are a Z80 code-density trick: `RST $nn` is a **one-byte** call to a
fixed page-0 address, so the game routes its hottest common subroutines through them
(each vector is just a `JP` to the real code higher up). Recursive-descent tracing
from the three hardware entry points (`$0000`, `$0038`, `$0066`) confirms this —
`RST $38` alone has dozens of callers — and that is where Part II picks up, following
`JP $0296` into the initialization proper.

## 6. What's in each bank

§2 explained *how* banks are paged; this is *what* they hold, as far as the analysis
has reached. Banks 0, 3 and 8 are **traced** (Parts II / IV); the rest are
characterised here by content and by how the code pages them, and will be pinned down
as later parts trace them. Two cheap signals: the Shannon **entropy** of each 16 KB
bank (compressed data runs ~7 bits/byte, code and tables ~6, sparse data lower), and
whether the bank is paged with an **immediate** `LD A,n / LD ($FFFE/$FFFF),A` (a
fixed resource) or from a **variable** (which signals level-number-driven paging —
there are both in this ROM).

| Bank | File offset | Entropy | Role | Status |
|---:|---|---:|---|---|
| 0 | `$00000` | 6.85 | **main game code** — vectors, init, the VDP / load / maths routines | traced (Part II) |
| 1 | `$04000` | 6.72 | game code (the default slot-1 bank; the most-paged of all) | traced (called) |
| 2 | `$08000` | 6.43 | code / data (the default slot-2 bank) | paged ×9 |
| 3 | `$0C000` | 5.50 | the **`RST` dispatcher** — opens with a `JP` table (`C3 …`) | traced (Part II §2) |
| 4 | `$10000` | 5.96 | the **block → tiles table** (16 B/block = 4×4 tile indices) + block attributes | traced (Part IV §4) |
| 5 | `$14000` | 5.89 | **level data** — the act-descriptor table (`$5600`) + compressed block-index maps | traced (Part IV §4) |
| 6 | `$18000` | 6.18 | a pointer / resource **table** (4-byte records) + more compressed **level maps** (e.g. Green Hills Act 3) | partly traced (Part IV §4) |
| 7 | `$1C000` | 6.23 | data | inferred |
| 8 | `$20000` | 5.58 | **graphics** — tile patterns + the palette pointer table (`$7400`) | traced (Part IV) |
| 9 | `$24000` | 7.01 | **compressed** graphics / data | inferred (high entropy) |
| 10 | `$28000` | 6.11 | compressed data — likely **zone / level** data | inferred |
| 11–15 | `$2C000`–`$3FFFF` | 6.2–6.5 | **zone / level data and graphics**, paged by level number | inferred |

So your intuition holds: bank 0 is the engine, banks 8–9 hold graphics, and the upper
banks (10–15) carry the bulk, high-entropy data the code reaches through *variable*
bank writes — exactly the shape of level/zone assets spread across the ROM. Tying each
upper bank to a specific zone is a Part IV/V job, once the level loader is traced.

---

# Part II — Boot and initialization

The reset code (Part I §5) ends with `JP $0296`. That is the real cold start: it
programs the cartridge mapper, clears RAM and sets the stack, brings the VDP up in
Mode 4, hides the sprites, runs a setup routine in another bank through the game's
banked-call gateway, and hands off to the main entry at `$1356`. This part walks
that path and the per-frame interrupt the init arms.

## 1. Cold-start init (`$0296`)

**Mapper.** First it re-asserts the default bank layout (Part I §2):

```
$0296  LD A,$80 / LD ($FFFC),A   ; mapper control ($80 = ROM mapping, no cart RAM)
$029B  LD A,$00 / LD ($FFFD),A   ; slot 0 <- bank 0
$02A0  LD A,$01 / LD ($FFFE),A   ; slot 1 <- bank 1
$02A5  LD A,$02 / LD ($FFFF),A   ; slot 2 <- bank 2
```

**RAM clear + stack.** The classic Z80 "fill by overlapping `LDIR`" — write one
zero, then copy it forward through itself:

```
$02AA  LD HL,$C000 / LD DE,$C001 / LD BC,$1FEF
$02B3  LD (HL),L                 ; (HL) = $00  (L is $00)
$02B4  LDIR                      ; propagate $00 across $C000..$DFEF
$02B6  LD SP,HL                  ; SP = $DFEF
```

It clears the 8 KB of work RAM up to `$DFEF`, stopping 16 bytes short of the top so
it does not clobber the mapper-register mirror at `$DFFC`–`$DFFF` (Part I §2), then
parks the stack at the top of the cleared region.

**VDP registers.** Eleven registers are written from a table at `$031C`, with a
shadow copy kept in RAM at `$D219` (the interrupt handler reads it back, §3):

```
$02B7  LD HL,$031C / LD DE,$D219     ; table, RAM shadow
$02BD  LD B,$0B / LD C,$8B           ; 11 registers
$02C1  loop: LD A,(HL) / LD (DE),A / INC HL / INC DE
             OUT ($BF),A             ; the value -> VDP control port $BF
             LD A,C / SUB B / OUT ($BF),A   ; ($8B-B) = $80|reg -> control port
             DJNZ loop
```

A VDP register write is two bytes to control port `$BF`: the value, then
`$80 | regnum`. The table (`26 A2 FF FF FF FF FF 00 00 00 FF`):

| Reg | Value | Meaning |
|---|---|---|
| 0 | `$26` | Mode Control 1: **Mode 4**, hide the left 8-px column |
| 1 | `$A2` | Mode Control 2: display **off** (during init), frame interrupt **on**, 8×16 sprites |
| 2 | `$FF` | name-table base → `$3800` |
| 3, 4 | `$FF` | unused on this VDP |
| 5 | `$FF` | sprite-attribute-table base → `$3F00` |
| 6 | `$FF` | sprite-pattern base → `$2000` |
| 7 | `$00` | backdrop colour = palette entry 0 |
| 8, 9 | `$00` | horizontal / vertical scroll = 0 |
| 10 | `$FF` | line counter (line interrupt off) |

The display stays off here and is turned on later once the first screen is built;
the detailed register semantics are Part IV.

**Hide the sprites.** A VDP fill clears the Sprite Attribute Table:

```
$02CD  LD HL,$3F00 / LD BC,$0040 / LD A,$E0
$02D5  CALL $05F0
```

`$05F0` is the engine's **VDP fill** primitive — `fill(addr=HL, count=BC, byte=A)`:

```
$05F0  LD E,A
       LD A,L / OUT ($BF),A          ; VRAM address, low byte
       LD A,H / OR $40 / OUT ($BF),A ; address high | $40  ($40 = "write VRAM")
       loop: LD A,E / OUT ($BE),A    ; byte -> VDP data port $BE
             DEC BC / LD A,B / OR C / JR NZ
       RET
```

The high address byte is OR'd with the VDP's write-VRAM command (`$00` read VRAM,
`$40` write VRAM, `$80` write register, `$C0` write CRAM). Here it writes `$E0`
across the 64 bytes of the Sprite Attribute Table at `$3F00`, setting every sprite's
Y off-screen — **hiding all 64 sprites** before the display comes on.

**Handoff.**

```
$02D8  CALL $02F8       ; run a setup routine in bank 3 (§2)
$02DB  LD IY,$D200      ; IY = the game-state RAM block
$02DF  JP $1356         ; -> main entry (§4)
```

## 2. Cross-bank calls and the `RST` gateways

`$02F8` is one of three short **banked-call thunks** at the bottom of the home bank,
and they are exactly the `RST $18/$20/$28` vectors (Part I §5: `$0018 JP $02E2`,
`$0020 JP $02F8`, `$0028 JP $0309`). Each pages a fixed bank into slot 1, calls a
fixed entry in it, then restores the previous bank:

```
$02F8  DI
       LD A,$03 / LD ($FFFE),A       ; slot 1 <- bank 3
       CALL $4006                    ; call the bank-3 routine
       LD A,($D22F) / LD ($FFFE),A   ; slot 1 <- previous bank (shadow)
       EI
       RET
```

So a one-byte `RST $20` is a gateway into bank 3 at `$4006` (and `RST $18`→`$4012`,
`RST $28`→`$4015`); bank 3 holds a dispatcher the engine reaches through these
1-byte calls. The "previous bank" is read back from a RAM **shadow** at `$D22F`: the
game keeps `$D22F` = the current slot-1 bank and `$D230` = the current slot-2 bank so
banked calls can nest and restore correctly. (The banking/dispatch system is Part III.)

## 3. The frame-interrupt handler (`$0073`)

The `$0038` maskable-interrupt vector is `JP $0073` — the per-frame (vblank) handler
that drives timing once `EI` runs at the main entry:

```
$0073  DI / PUSH AF / PUSH HL / PUSH DE / PUSH BC
       IN A,($BF)               ; read VDP status — acknowledges the interrupt
       BIT 7,(IY+6) / JR Z,…    ; only do frame work when a game-state flag is set
       … VDP scroll + line-counter (reg 10 / $8A) setup …
       PUSH IX / PUSH IY
       LD HL,($D22F)            ; preserve the banked context across the IRQ
       …
```

Reading the VDP status port `$BF` is the interrupt acknowledge (it clears the
pending-interrupt flag). The handler is gated on a game-state bit (`IY+6` bit 7),
preserves the index registers and the bank shadows, and reprograms the VDP **line
counter** to fire a mid-frame line interrupt — the standard trick for a fixed status
bar above a scrolling playfield.

## 4. The main entry (`$1356`)

```
$1356  SET 0,(IY+0)                       ; arm the main game-state flag
       EI                                 ; interrupts on — the frame handler now runs
       LD A,$01 / LD ($FFFE),A / LD ($D22F),A   ; slot 1 <- bank 1 (+ shadow)
       LD A,$02 / LD ($FFFF),A / LD ($D230),A   ; slot 2 <- bank 2 (+ shadow)
       RES 0,(IY+2) / RES 1,(IY+2)
       CALL $0645 / CALL $1CD7 / CALL $0AA3     ; subsystem init
       LD A,$03 / LD ($D240),A                  ; game mode <- 3
       …
```

The handoff into the game proper: enable interrupts, set the bank shadows to the
running configuration, clear state flags, run the subsystem initializers, and set
the top-level **game-mode** variable `$D240` (the state-machine selector — title,
level, … — the subject of Part III). From here control is in the main loop.

### RAM landmarks established so far

| Address | Use |
|---|---|
| `$D200` | game-state block (the `IY` base; `IY+0/+2/+6` are flag bytes) |
| `$D219`… | shadow copy of VDP registers 0–10 |
| `$D22F` | current **slot-1** bank number (for banked-call restore) |
| `$D230` | current **slot-2** bank number |
| `$D240` | top-level **game mode** |

# Part III — Engine architecture

This part follows the engine *forward* from the opening, **by tracing the code**, to
see how far the title → world map → level chain can be reconstructed without running
the game. The answer turns out to be: the whole control skeleton, and every screen up
to and including the world map, comes out of static disassembly; the level load
itself sits behind a cross-bank dispatcher and is the marked frontier (§4).

## 1. The attract loop and the scene state machine

After init (Part II), `main_entry` (`$1356`) loads the SEGA logo and then enters an
**attract loop**. The loop is driven by a one-byte **scene counter** `$D238`: each
pass loads a screen, fades it in, runs it, and advances; when `$D238` reaches `$13`
the loop jumps back to `$135B` and replays from the logo. The skeleton is:

```
$13C5  LD A,($D238) / CP $13 / JP NC,$135B   ; past the last scene -> restart attract
$13D8  CALL $0BDD                            ; scene dispatcher: load this scene's screen
$13E4  CALL $0AA3                            ; fade the screen in
$13F0  BIT 4,(IY+6) / JR NZ,$1402            ; Start pressed? skip the idle wait
$13F6  (wait ~60 frames via $0327)
$1402  CALL $1414                            ; run the scene; returns 0/1/2 = restart/next/stay
```

Each scene is described by a **~40-byte descriptor** in a table in bank 5 at `$5600`
(18 entries; `$1414` indexes it by `$D238`, and `$185D` copies the descriptor into RAM
at `$D355` to run it). The dispatcher `$0BDD` maps the scene to a **screen type**:
scenes 0–8 are type 1 (the title background, loaded near `$0C1C`), scenes 9–17 are
type 2 (the world map, `$0C7A`). It only *reloads* the background when the type changes
(`$0C0E`: compare the new type with the previous one in `$D217`, skip if equal) — so
the title background stays up across scenes 0–8 while only the foreground animates, and
the map likewise persists across its scenes.

## 2. The title, and waiting for Start

Scenes 0–8 keep the title background (Part IV §3) up while their scripts animate Sonic
(the finger-tapping pose is sprite animation, driven by the per-scene script and the
sprite engine of Part IV §3) and blink `PRESS START BUTTON` via the string blitter
`$0612`. The wait for Start is not a poll loop; it is folded into the attract loop
through two flags:

- `$0602` reads the controller every frame into `(IY+3)` — **Start** is GG port `$00`
  bit 7, the D-pad/buttons are port `$DC`.
- When Start is pressed, a handler raises the **launch flag** `(IY+6) bit 4` and writes
  a **target scene** into `$D2D4` (e.g. `$53C4` in bank 1, or `$017FE`). On the next
  pass the attract loop sees the flag: `$13F0` skips the idle wait, and `$141F` uses
  `$D2D4` instead of the counter — so Start *jumps the sequence* out of the demo and
  into the post-title flow rather than letting it free-run.

This is also why the logo and title are skippable: `$1D47` (`BIT 7,(IY+3)`) checks the
very same Start bit during the logo.

## 3. The world map (decoded by tracing, not the oracle)

The world map is reconstructed here entirely from reading the loader code — no
emulator. A map loader (e.g. `$0C7A`) does three things, each traceable:

- background tiles: `$0406`-decompressed from a `(bank $0C, addr)` block;
- name table: a **stored RLE map** in bank 5, loaded by `$0502` (the codec of Part IV
  §3) — `extract/decomp.LoadRLE` reimplements it;
- palette: `$0AAB` loads a black start (index `$16`) and **fades toward** the real
  targets from the bank-8 table.

[`extract/cmd/scenemap`](extract/cmd/scenemap) replays exactly those steps.

### Two maps, switched by an upcoming-level countdown

There are **two** world-map screens — a different tile set *and* a different stored
map for each — and the engine picks between them by the level you are heading into.
The selector is the byte `$D279`, which is **decremented after each level** (bank 1
`$46F9`: `DEC A / LD ($D279),A`), so it is a *countdown* toward the final stage. Every
map loader branches on it the same way:

```
LD A,($D279) / CP $06 / JR C,zoomed      ; <6 left -> zoom in; >=6 -> wide
```

This exact test appears at `$2006`, bank 1 `$466A`, and bank 2 `$BF0D`. Because
`$D279` counts down, **early** levels (count ≥ 6) show the **wide** island and **late**
levels (count < 6) **zoom in** on the mountain-top goal — which is the progression you
see in play. The two variants compose like this (both rendered statically by `scenemap`):

| variant | branch | tiles | bank-5 map | palette |
|---------|--------|-------|------------|---------|
| wide island | `$D279 ≥ 6` | `(bank $0C,$0000)` | `$6F86` + `$716A` | bg `$0A` / spr `$0B` |
| zoomed peak | `$D279 < 6` | `(bank $0C,$171A)` | `$6C6D` + `$6DC3` | bg `$0C` / spr `$0D` |

![Wide-angle island map — shown for the early levels ($D279 >= 6)](rendered/worldmap_wide.gg.png)

![Zoomed mountain-top map — shown for the late levels ($D279 < 6)](rendered/worldmap_zoom.gg.png)

It is the same island both times: the castle/ruin on the peak of the wide shot is the
detailed city that fills the zoomed shot. Decoding each from its own tile set and
`$0502` map confirms the switch is a genuine two-asset branch, not a camera move.

### The blinking route

On either map the **route and the zone name** are not in the stored map — they are a
per-scene overlay. A map loader's tail calls the string blitter `$0612` with per-scene
data (the `$1163`/`$0EA8` tables → `$D211`), so each stop lights a different zone. The
blink is that overlay repainted on a timer — the same `$0612` name-table-run mechanism
as the blinking `PRESS START` (Part IV §3), just driven by per-scene position data.

## 4. How far pure tracing reaches — and the level-load frontier

Up to here, **everything was traceable statically**: the attract state machine, the
Start hand-off, and the world map (rendered above from located data, the same way the
opening screens were). The level load is where it gets harder, and following the
launch reveals exactly why.

**Gameplay is launched from inside a scene, not from the loop.** The attract loop
(§1) never exits: its tail (`$1405`) only ever restarts the demo, advances to the next
scene, or re-fades the current one. So pressing Start does not "return into the game" —
it selects a target scene (§2) whose script *itself* loads the level and jumps into the
gameplay engine, never coming back. The scene scripts are the bank-5 data interpreted
by `$185D`, so the trigger is a data table, not a call you can read directly.

**The cross-bank calls go through a dispatcher in bank 3.** The `RST $18/$20/$28`
gateways (Part II §2) land on a jump table at bank 3 `$4000`; `RST $28` (`$46FF`) and
`RST $18` (`$46EB`) are themselves sub-dispatchers that index a function table by the
selector in `A` and call into yet another bank. This is how the engine reaches code
outside the home 64 KB — and it is why a bank-0-only disassembly can't follow the
thread: the function tables point into banks that aren't mapped in this configuration.

**The gameplay engine and the level variables live in bank 1.** Tracing the level
RAM gives concrete coordinates: the tile-stream source `$D289` and count `$D28D` are
set at bank 1 `$4EC4`/`$4EC9`, the camera `$D2AB` at `$502C`, the map base `$D2AF`
around `$5050`/`$73BB`/`$7B78` (and bank 2). The routine at bank 1 `$4EA0` is the
per-frame **scroll/camera update** — each frame it recomputes the tile-stream source
from the camera position, which is what feeds the streamer `$31BC` and `scroll_draw`
`$3282` (the gameplay update `$0130`, which has no `CALL` site in bank 0 because it is
reached by jump from this engine, not called from home code).

So the honest verdict on doing it "without cheating": the opening — logo, title, the
Start wait, and the world map — is **fully** recoverable by tracing, data and all. The
launch *mechanism* is traced too (scene-script trigger → bank-3 dispatcher → the bank-1
gameplay engine). What pure bank-0 tracing could not finish was the **level format**
itself, because the engine that draws it runs from banked code the home disassembly
can't follow. That decode is done in **Part IV §4**, with the Game Gear machine model
as an oracle to pin the banked routines and tables: the compressed block-index map, the
block → tiles table, and the tile graphics — enough to reconstruct a whole zone from the
ROM. (One correction lands there: the terrain is drawn by the column expander `$0760`,
*not* `scroll_draw` `$3282`, which handles objects.) The **object layout** is what is
left for Part V.

# Part IV — Graphics and data formats

Graphics on the Game Gear are two layers of problem: the fixed **VDP hardware
formats** the data ends up in (§1), and the game-specific **compression** it is
stored under in the ROM (§2). With both decoded we can take a routine that loads a
screen and reproduce its pixels exactly — §3 does this for the first screen the
console shows, and §4 does it for a whole scrolling level, both as end-to-end checks.

## 1. The VDP formats

These are standard Mode-4 hardware formats, decoded by the reusable
[`tools/gamegear`](tools/gamegear) package (so they are shared, not Sonic-specific):

- **Tiles** — 8×8 pixels, **4 bitplanes** (16 colours), **32 bytes** each, stored
  *row-interleaved*: each pixel row is 4 bytes (one per bitplane), and pixel *x*'s
  colour index is bit `7-x` of each plane (low plane = bit 0). `DecodeTile`.
- **Palette (CRAM)** — 32 entries of **2 bytes**, **12-bit** colour, 4 bits per
  channel in **BGR** order (`0000 BBBB GGGG RRRR`); a 4-bit channel scales to 8-bit
  by ×17. Indices 0–15 are the **background** palette, 16–31 the **sprite** palette.
  `Palette`.
- **Name table** — the 32×28 background map at VRAM `$3800`; each cell is a 2-byte
  word: the 9-bit tile number plus per-cell horizontal/vertical **flip**, **palette
  select** and **priority** bits. `DecodeNameEntry` / `RenderNameTable`.

## 2. The graphics decompressor

Almost all of Sonic's art is compressed (Part I §6: the upper banks run at ~7
bits/byte). The decompressor is the routine at **`$0406`**; it is a **4-byte-unit
LZ** — a literal/back-reference scheme whose unit is one *tile row* (4 bitplane
bytes), so a repeated row (a blank row, a flat fill) costs a single bit. It is a
game-specific *software* codec, so it lives in the game's
[`extract/decomp`](extract/decomp) module, not in `tools/gamegear`.

A compressed block is addressed as a `(bank, address)` pair; the routine's prologue
**normalises** it (`while addr ≥ $4000: addr -= $4000; bank++`) and maps two
consecutive banks into the slots, so a block may span banks. The block then is:

```
+0  word   (skipped)
+2  word1  offset to the match-info stream
+4  word2  offset to the literal stream (also the back-reference base)
+6  word3  count — number of 4-byte output units
+8  …      control bitmap: one bit per output unit
+word1     match-info stream: variable-length back-reference offsets
+word2     literal stream: 4-byte units
```

Decoding walks `count` units; for unit *i*, control bit *i* (`bitmap[i>>3] &
(1<<(i&7))`) selects:

- **0 — literal:** emit the next 4 bytes of the literal stream and advance it.
- **1 — match:** read an offset from the match-info stream — one byte `b`, but if
  `b ≥ $F0` it is a two-byte big value `((b-$F0)<<8) | next` — and emit the 4 bytes
  at `literal_base + offset×4` (a back-reference to an earlier unit).

So the literal stream is the set of distinct 4-byte tile rows in first-appearance
order, and the bitmap + match offsets reconstruct the full tile data from them.

## 3. The opening screens: how they are built

The two screens the console shows before the menu — the **SEGA logo** and the
**title** — are interesting because they reach the same VRAM by *opposite* routes.
The logo's tile map is **computed in code**; the title's is a **stored, compressed
map** loaded wholesale. Both then get a per-frame sprite layer and, on the title, a
blinking text overlay. This section follows the actual routines.

To do that I let the game run and watched what it did, using the Game Gear machine
model in [`tools/gamegear`](../../tools/gamegear) (the `z80` core wired to RAM, the
Sega mapper and a VDP that captures VRAM/CRAM) as an *oracle*. Two things make it a
tracing tool rather than just a renderer: [`extract/cmd/screentrace`](extract/cmd/screentrace)
fingerprints VRAM every frame (so palette fades don't hide the moments the picture
actually changes), and the VDP carries a **write watchpoint** — arm a VRAM address
range and it records the PC of every routine that writes there. The final images
below are read back from the real VRAM the code produced; the addresses are from the
disassembler. Each confirms the other.

### The SEGA logo — a procedural reveal (`$1CD7`)

`sega_logo` first decompresses its tiles (`$0406` from `(bank $0C,$FA74)`, 96 logo
tiles to VRAM `$0000`; a sprite set to `$2000`) and selects palette index `$12`, the
blue-on-white set. Then it **clears the whole name table to tile `$70`** (`$1D1B`: a
896-entry fill, which the watchpoint sees as 896 writes from `$1D1F`). There is no
stored tile map for the logo at all.

The picture is built by a loop (`$1D3E`–`$1D5C`) that walks a position counter
`$D213` from `$2C` to `$6C` in steps of 4 — sixteen iterations, one per column. Each
iteration calls `$1E8A`, which writes **one vertical 6-tile column** straight into the
name table: it derives the column's VRAM address from `$D213`, then writes tiles
`c, c+16, c+32, … c+80` down the column (stride `$40` = one row), high byte `$00`.
Because the column index is `c` and each row adds 16, cell *(row,col)* ends up holding
tile `col + 16·row` — a plain `0…95` identity grid over a 16×6 rectangle. The logo
tiles are stored in reading order, so the "map" *is* the identity; the cleverness is
that the loop lays it down **one column at a time, left to right**, so the logo wipes
in. (Render it mid-build and you get `SEG` before the `A` — exactly that left-to-right
edge.) In lockstep, `$1EC1`→`$2F07` build a small **sprite** display list in RAM at
`$D000` — three rows of entries, `$FE`=skip / `$FF`=end, offset per row by the
symmetric table at `$1F14` (`00 FD FB F8 … F0 … F8 FB FD`, i.e. a parabolic arc) —
which the frame interrupt flushes to the sprite-attribute table at `$3F00`/`$3F80`
(`$033F`) every vblank. **That sprite is Sonic**: he jumps across the screen along
that arc, left to right, and the logo columns draw in behind him as he passes; he
then jumps back to the left and comes to rest beside the finished logo. (The
symmetric `$1F14` table is his jump trajectory.)

So the logo is: an identity background grid revealed column-by-column as Sonic jumps
across it, then Sonic landing beside the completed logo. Read back from the oracle's
VRAM, the finished frame is:

![SEGA logo — the name table the boot code built, column by column](rendered/sega.gg.png)

### The title — a stored, compressed map (`$0C20`)

The title takes the other approach. `$0C20` decompresses three tile blocks (`$0406`
from bank `$0C` to VRAM `$0000`, and two blocks from bank 9 to `$2000`/`$3000`), then
**loads the name table from a stored map** with `$0502` — twice:

```
$0C4C  HL=$6962  BC=$019B  DE=$3800  $D20F=$10   ; base layer, 411 entries, priority bit
$0C5F  HL=$6AFD  BC=$0170  DE=$3800  $D20F=$00   ; overlay,    368 entries
```

`$0502` is a tiny **RLE name-table loader**: it streams tile bytes from the source
(in bank 5), expanding runs of a repeated tile, until an `$FF` terminator, and writes
each entry as *(tile byte, `$D20F`)* — so `$D20F` supplies the shared high byte
(palette select and the priority bit) for the whole layer. The two passes compose the
image: a priority base (`$D20F=$10`, drawn in front of sprites) and a plain overlay
(`$00`). 411 + 368 ≈ the 767 non-background cells the screen ends up with — a
full-screen picture loaded in one shot (the watchpoint attributes those ~768 writes to
`$0502`'s literal and run loops at `$052D`/`$054B`). No per-tile code, no reveal: the
map was authored offline and compressed into the ROM.

![Sonic title screen — the stored map loaded by $0502, with the blinking prompt](rendered/title.gg.png)

### `PRESS START BUTTON` — a text overlay (`$0612`)

The prompt is not part of the stored map; it is painted on afterwards and blinks
(the oracle shows it off at frame 620, on at 660, off again at 700). It is drawn by
`$0612`, a **name-table string blitter**: it reads a packed *(row,col)* header,
computes the `$3800` address, then writes a run of tile bytes from a source list
(terminated by `$FF`), again taking the high byte from `$D20F`. Its inner loop
(`$063A`) is the single busiest name-table writer in the whole attract sequence,
because it repaints the prompt every time it toggles. The blink and the "is Start
held?" test share one input byte: `$0602` reads the controller — Start from GG port
`$00` bit 7, the D-pad from port `$DC` — into `(IY+3)`, the same flag the logo checks
(`BIT 7,(IY+3)` at `$1D47`) so the sequence can be skipped.

### A note on the sequence (and a correction)

The progression logo → title → menu is **timed/scripted**, not a single mode switch.
In particular `$D240` — earlier guessed to be the top-level "game mode" — holds `$0A`
unchanged across the whole logo→title transition; the disassembly shows it used as a
name-table scroll index (`$0E35`/`$4969`, ×10) and a frame countdown (`$472E`), not a
screen selector. Mapping the actual attract script is Part III's job; the point here
is the drawing, and the drawing is fully accounted for above.

## 4. Level maps: how a zone is stored and drawn

The opening screens (§3) are single static pictures. A *level* is different: it is far
wider than the screen, it scrolls, and it is built from a small alphabet of reusable
**blocks** rather than per-cell tile numbers. Sonic stores a level as three nested
layers — a compressed **block-index map**, a **block → tiles** table, and the **tile
graphics** — plus a palette. Decoding all three lets us reproduce a whole level from the
ROM alone; the result for Zone 0 (Green Hills Act 1) is the image at the end of this
section, rendered by [`extract/cmd/levelmap`](extract/cmd/levelmap).

This was the one part the oracle could **not** brute-force. The machine model loads and
draws a level, but it is not cycle-accurate enough to play one: holding Right+Jump makes
Sonic run, yet he stalls roughly a quarter of the way in, so no amount of scrolling
reveals the whole map. The only way to the full level was to decode the storage formats
and expand them statically — which is what follows. **Everything below comes from ROM**:
the map, the block→tile table, the tile graphics and the palette. The oracle is kept only
to *validate* — booting each level and checking the from-ROM tiles and palette against the
loader's live VRAM/CRAM. They match except for a handful of animated water tiles and a
three-colour palette cycle, which a static frame cannot represent.

### The pipeline

```
bank 5 compressed map ──$0A73 RLE──▶ RAM $C000 window ──$0760 expand──▶ name table ──$0860──▶ VRAM
 (block indices)                      (16×256 block grid)   │ block→16 tiles               (one column
                                                            ▼   (bank 4 table)               at a time)
                                                       tile graphics in VRAM (bank-stored, $0406-decompressed)
```

A level is loaded by the resource loader at **`$199D`**, driven by a 40-byte
**descriptor** for the act (the descriptor table is at bank 5 `$5600`, indexed by act
number; see Part III). The loader reads a chain of `(address, length)` records out of
the descriptor and, for each, pages the right bank and decompresses it: the **map** to
RAM `$C000` (`$0A73`), then it records the **block tile table** address in `$D249`, then
it `$0406`-decompresses the **tile graphics** into VRAM. So everything below is reachable
from one descriptor.

### The block-index map (`$0A73`)

The map itself is a **run-length-compressed list of block indices**. For Zone 0 it lives
in **bank 5, file `$17430`** (z80 `$7430`), is `$0786` = 1926 bytes, and decompresses to
exactly **4096 bytes** — a **16-row × 256-column** grid, row-major with a 256-byte stride
(`block = map[row*256 + col]`; the stride comes from `$D232`). Row is vertical (row 0 =
top of the sky, row 15 = the bottom ground fill), column is horizontal. The decompressor
`$0A73` writes this grid to the RAM window at **`$C000`**, where the engine reads it as it
scrolls.

`$0A73` is a small RLE codec, the same family as the `$0502` name-table loader (§3) but
for a raw byte stream:

- it keeps the previously written byte; a source byte **equal** to it triggers a **run**
  — the duplicate is followed by a **count** byte and the value is emitted *count* more
  times (the count is a `DJNZ` counter, so `$00` means **256**);
- any other byte is a **literal**;
- after a run the "previous byte" is re-armed so the next byte always starts fresh;
- there is **no terminator** — decoding is bounded purely by the length (`BC`).

It is reimplemented as [`extract/decomp.LoadMapRLE`](extract/decomp/nametable.go) and
verified byte-perfect against the live decompressor's `$C000` output. At 1926 → 4096
bytes it is a modest 2.1×, because most of the map is long runs of sky (block 0) and
ground fill (block 1). The **scenery is part of the block map**, though: the hills, palm
trees, flowers and ring graphics in the render below are all drawn from blocks — that
surface detail is exactly what the run-length coding can't collapse, and why the ratio is
2.1× rather than higher. The block map *is* the whole static scene. What it does **not**
contain is the **object** layer — the moving, interactive entities (Sonic, enemies, and
whatever the game tracks for collection and collision); see the note after the render.

### Blocks → tiles (bank 4, `$10000`)

Each block index expands to a **4×4 grid of 8×8 tiles** — a 32×32-pixel chunk — through a
**block tile table** at **bank 4, file `$10000`** (z80 `$4000`, read with bank 4 paged
into slot 1, which the expander's prologue at `$0726` arranges; the loader pointed `$D249`
here). Each block is **16 bytes**, row-major 4 wide, so

```
tile(r, c) = rom[$10000 + index*16 + r*4 + c]      r, c ∈ 0…3
```

For example block `$01` (ground) is `02 03 02 03 / 03 02 03 02 / …` — the checkerboard;
block `$00` (sky) is sixteen `00`s. The expander `$0760` reads one map column, looks up
each block's tiles and writes name-table cells into a column buffer at RAM `$D180`, which
`$0860` uploads to VRAM a column at a time as the camera crosses each 8-pixel boundary.
(The index→offset arithmetic in `$0760` — `RLCA`×4 then `XOR` the high nibble — *looks*
like a scramble but works out to exactly `index*16`.)

A second per-block table, the **attribute** table (`$D211`, found via `$343D[zone]`, also
in bank 4), gives one byte per block, of which only **bit 7** is used — a name-table
**priority** bit. There is no per-block flip or palette select, so all terrain draws from
the **background palette** and the pixels come entirely from the tile indices plus the
loaded tile set.

### The tile graphics and palette

The actual 8×8 tile patterns are compressed with the `$0406` codec (§2). The descriptor's
tile-set pointer locates the compressed set (file `$30000` + word, banks `$0C`–`$0F`);
`decomp.Decompress` inflates it to the **256 background tiles** the block table indexes as
plain 0…255 tile numbers. The **palette** is loaded by index: the descriptor's BG palette
index (byte +29 → `$D22C`) is resolved by `load_palette` (`$0586`) through a per-index
offset table at bank 8 `$7400` to a 32-byte (16-colour) palette; `romPalette` reads the
same. So both the tiles and the palette come straight from ROM — light blue sky, greens,
the orange/brown ground.

### Where it all lives (Zone 0, Green Hills Act 1)

| Data | Stored at | Runtime | Format |
|---|---|---|---|
| Act descriptor table | bank 5 `$15600` (z80 `$5600`) | — | 18 word-pointers → 40-byte descriptors (Part III) |
| Compressed block-index map | bank 5 `$17430` (z80 `$7430`), 1926 B | → RAM `$C000` | `$0A73` RLE → 4096 B |
| Decompressed map | — | RAM `$C000`, 4096 B | 16 rows × 256 cols, row-major, stride 256 |
| Block tile table | bank 4 `$10000` (z80 `$4000`) | base in `$D249` | 16 B/block = 4×4 tile indices, row-major |
| Block attribute table | bank 4, via `$343D[zone]` | base in `$D211` | 1 B/block, bit 7 = priority |
| BG tile graphics | bank `$0C` `$32ED5` (file `$30000` + word) | → 256 tiles | 4-byte-unit LZ, `$0406` (§2) → `decomp.Decompress` |
| BG palette | bank 8, via `$7400`[index] (index = descriptor +29) | CRAM 0–15 | 12-bit BGR (§1) → `romPalette` |

A block is 32×32 px (4×4 tiles). The decompressed map is always a fixed **16×256-block**
grid, but only the level's own width is reached in-game. `$D26F` is the **maximum camera
position** (the screen's top-left), so the level is visible up to **one screen (160 px = 5
blocks) past it** — the width is `$D26F/32 + 5` (203 blocks for Act 1). Columns beyond that
are off-level storage padding, cropped (no content trimming — legitimate empty space within
the width is kept). Expanded through the table above — every byte from ROM — the render
reproduces the level:

![Green Hills Act 1 — the full level, reconstructed from the ROM block-index map, block tile table and tile graphics](rendered/level_greenhills_act1_overview.png)

*(Scaled to fit; the full-resolution 8192×512 render is
[`rendered/level_greenhills_act1.png`](rendered/level_greenhills_act1.png).)*

### The other two acts of the zone

Because the act-resource table (`$5600`) is static data, the rest of the zone falls out
without running anything new. Green Hills is **acts 0–2** of the 18; reading their
descriptors shows the three share **the same tile table (`$10000`), the same tile-set
pointer (`$2ED5`) and the same palette/graphics fields byte-for-byte** — they differ
**only** in the block-index map (and the level width):

| Act | Map source | Length → output | Width | Ratio |
|---|---|---|---|---|
| 1 | bank 5 `$17430` (z80 `$3430`) | 1926 B → 4096 | 198 blocks | 2.1× |
| 2 | bank 5 `$17BB6` (z80 `$3BB6`) | 1716 B → 4096 | 101 blocks | 2.4× |
| 3 | bank 6 `$1826A` (z80 `$426A`) | 829 B → 4096 | 80 blocks | 4.9× |

So decoding acts 2 and 3 is just `LoadMapRLE` on those two sources, then the same block →
tile expansion with the shared table and tile set. `cmd/levelmap` renders all three;
acts 2 (a more cave-and-platform layout) and 3 (the short final act) come out as coherent
Green Hills levels in the identical art:

![Green Hills Act 2 — same tiles and palette, a different map](rendered/level_greenhills_act2_overview.png)

![Green Hills Act 3 — the short final act, same art](rendered/level_greenhills_act3_overview.png)

This is the payoff of the descriptor format: one decode, and every act of the zone (and,
by the same table, the other zones) is reachable from the ROM alone.

### The act descriptor — a field map

Each act descriptor is **37 bytes**, reached as `$5600 + word($5600 + act×2)` in bank 5
(file `$15600`). Words are little-endian. Offsets verified against `cmd/levelmap` /
`cmd/webexport` (0.00 % pixel diff) and the loader (`$19D0`–`$1A6A`); a few in the
`+23…+28` graphics-pointer region and `+32/+34/+35` aren't individually pinned.

| Off | Size | Field | Notes |
|----|------|-------|-------|
| +0 | 1 | **zone** 0–7 | the engine zone byte (`$D2D5`): selects the per-zone attribute/collision table (`$343D`) etc. **6 = the special stage, 7 = the Sky Base interior** (Sky Base 3 and the Act 2 teleporter room) — for acts 0–16 it equals act÷3, for act 17 it is 7 |
| +1 | 2 | **map stride** (`$D232`) | column count, hi byte: `$80`→128, `$40`→64, `$20`→32, `$10`→16, else 256 — so the map is `(4096/stride)` rows tall |
| +3 | 2 | (vertical scroll extent — not pinned) | |
| +5 | 2 | **left scroll bound** (`$D26D`) | |
| +7 | 2 | **right scroll bound** (`$D26F`) | ÷32 = the played width in blocks |
| +9 | 2 | **top scroll bound** | |
| +11 | 2 | **bottom scroll bound** | |
| +13 | 1 | **spawn block X** | Sonic's start column |
| +14 | 1 | **spawn block Y + 1** | the loader uses `byte − 1` |
| +15 | 2 | **map data offset** | file = `$14000 + word` (banks 5–7); `$0A73` RLE → a 4096-byte block-index map |
| +17 | 2 | **map data length** (compressed) | |
| +19 | 2 | **block-tile-table offset** | file = `$10000 + word`; 16 bytes/block = a 4×4 tile grid |
| +21 | 2 | **tile-set offset** | file = `$30000 + word`; `$0406`-compressed BG tiles |
| +23 | 6 | graphics-stream pointers/banks | the BG + sprite tile-load sources (`+24/+25` is a second `$0406` source); not all individually pinned |
| +29 | 1 | **BG palette index** (`$D22C`) | resolved through the bank-8 `$7400` offset table (Part IV §1) |
| +30 | 2 | **object-table offset** | `$15600 + word`; `[count]` then `[type, blockX, blockY] × count` (Part V §1) |
| +32 | 1 | (not pinned) | |
| +33 | 1 | **flags** | bit 7 = Labyrinth water (the underwater raster split, Part V §3); bit 6 set on Sky Base Act 2 |
| +34 | 2 | (not pinned) | |
| +36 | 1 | **music id** (`$D2F7`) | the loader (`$1A66`) plays it via `RST $18`; indexes the `$4716` song table (Part VI). Zones → 0–5 (Sky Base acts 1 & 3 reuse Scrap Brain's id 4); the special stage → **16** |

### Every zone — and a surprise: the map isn't always 16×256

The same table has 18 entries (6 zones × 3 acts), and `cmd/levelmap` renders them all (to
`rendered/level_<zone>_act<N>.png`), entirely from ROM. The zones — Green Hills, Bridge,
Jungle, Labyrinth, Scrap Brain, Sky Base — each have their own tile set and block table,
shared by the zone's three acts; the palette is usually shared too, except **Sky Base**,
whose three acts use the same tile set with three *different* palettes. The validation
oracle is loaded once per distinct (tile set, palette) combination, forcing the act number
`$D238` during the load.

The surprise is the map **shape**. The decompressed map is always 4096 bytes, but it is
*not* always 16×256: the expander reads a **stride** from `$D232` (`$0938` selects 256 /
128 / 64 / 32 / 16 from the top bits), and the grid is `(4096 / stride) × stride`. So the
stride is the column count, and a *small* stride is a **tall, narrow** level. Most acts are
wide (stride 256 → 16×256, or 128 → 32×128), but Jungle Act 2 has **stride 16 → a 256-row ×
16-column vertical level** — the waterfall climb, 16 blocks wide and hundreds tall. The
fixed-grid insight from Green Hills holds (4096 cells), but how those cells are laid out —
wide platformer vs. vertical climb — is data-driven per act.

### The object layer (the start of Part V)

The terrain above is the static block map. The **interactive** world — Sonic, the enemies,
items — is a separate **object** layer. Each act's descriptor points (word at `+30`) to a
per-act **object table** in bank 5: a count byte followed by 3-byte entries
`[type, blockX, blockY]`. At level load `$1A80`/`$1AB3` expands the table into the object
array at RAM `$D3FD` — 32 records of 26 bytes, type at `+0` and the world position
`X = blockX×32` (`+2/3`), `Y = blockY×32` (`+5/6`). Record 0 is **Sonic**, placed from the
spawn pointer `($D217)` — on a fresh act start that is the descriptor's own spawn field
(`+13/+14`; Green Hills 1 = block **(5, 11)**, oracle-verified — see Part V).

For Green Hills Act 1 there are 26 placed objects. The types (all now named, below):
`$08` = **crab** (4 of them), `$10` = **beetle**, `$01`/`$02`/`$03` = bonus items, `$09` =
swinging platform, `$0F` = horizontal moving platform, plus eight `$50` (**background
animators** — the twinkling sea waves and clouds) and one `$51` (a **checkpoint**). None
are rings — the rings are baked into the block map (below). Note the table position is the
*spawn*, not where an object is seen in play: the engine settles each object on activation
(Part V §1), and the engine-exact scene is rendered by `cmd/placeshot`
([`rendered/placement_greenhills1.png`](rendered/placement_greenhills1.png)).

Each object **type** indexes an 8-byte record at `$2560` — its per-frame **culling
margins** (the `$2BD8` pre-pass; valid types are `< $57`) — and a behaviour handler
at `$24B2` (below) for what it *does*.

### Animated tiles (rings and flowers)

The rings and the spinning yellow flowers are **baked into the block map**, but they are
**two separate animations on two separate sets of four tiles** — a Game Gear tile is 8×8,
so each 16×16 graphic is **4 tiles**:

- the **rings** are tiles **252–255** (blocks 121–123), spinning through ~6 frames;
- the Green Hills **flowers** are tiles **12–15**, a 2-frame animation.

Both tile sets are *empty in the base tile set*; the per-frame update at `$15FF` copies a
fresh frame into them each cycle (~every 10 frames — which is why the validation flags
those eight tiles plus a 3-colour palette cycle). There is no generic "animation table" —
each animation is hardcoded: the **rings** are copied from a fixed bank-11 source
(`$2F73D`) for *every* zone, while the **flowers** are a 2-frame toggle from bank 11
`$7A3D`/`$7ABD` gated on the zone being Green Hills. (Green Hills has no water; the
zone-specific animation there is the flowers — earlier mislabelled "water".)

So a still frame can't capture the spin, but it *can* show them: the render loads one
frame of each group into the empty slots (`applyAnimFrame`), which is why the rings and
flowers now appear. Other zones likely animate more than the rings (e.g. real water);
only the rings were observed in the idle probe so far.

The mapping of each **type** to its behaviour is the body of Part V (below). The machine
model can place the objects but isn't cycle-accurate enough to *run* them — Sonic falls
through the floor instead of running the level — so identifying a handler needs the static
decode rather than the oracle.

### Sprites — Sonic and the enemies

Objects are drawn as **hardware sprites** (VDP 8×16 mode), not from the tilemap. The loader
decompresses a per-zone **sprite tile set into VRAM `$2000`** (sprite pattern base, reg 6),
and the sprite-attribute table at `$3F00` lists each active sprite's `(Y, X, tile)`. The
per-object draw builds a display list in RAM `$D000` (the helpers `$2F07` lay out a tile grid,
`$2F5D`/`$2FA8` append single sprites), which `sat_flush $033F` copies to the SAT each vblank.

Sonic's animation is **data-driven and streamed**: his pose id `(IX+20)` indexes an animation
table at `$5C5B` → a frame sequence (a byte stream; bit-7-set bytes are loop/jump controls),
and the current frame id both selects a sprite-layout (`$5C1B`) and computes a ROM source whose
**16 tiles are streamed into the sprite VRAM** when the frame changes — so only the live frame
(plus neighbours) is resident, not a full sheet.

Because the frames stream, the cleanest capture is the **oracle**: boot an act, let the engine
draw, then read `VRAM $2000` + the SAT + the sprite palette (CRAM 16–31) and compose the real
sprites. `extract/cmd/spriteprobe` dumps each zone's resident sprite tile sheet
(`rendered/sprites/`); `extract/cmd/spritecompose` lifts Sonic's spawn frame; and
`extract/cmd/enemyprobe` scrolls Green Hills so enemies activate and crops each one (the crab
`$08` and beetle `$10` came out clean). Those PNGs are what the level viewer now draws on its
Objects layer in place of the coloured boxes for Sonic, the crab and the beetle.

# Part V — Game mechanics

*In progress.* Object **placement** is decoded (Part IV §4); the **behaviours** are the
current frontier. Each object's `type` byte selects its behaviour and sprite. The table
below collects the types as they are identified — from the object placements, from play
testing, and confirmed against the behaviour code. The per-frame culling pass at `$2BD8`
indexes a **bounding-box** table at `$2560` by `type` (8 bytes each, valid types `< $57`)
for each type's *size*; the **behaviour** is dispatched by a separate table at `$24B2`
(below).

## 1. Objects

### Object types — the master dispatch

The behaviour dispatch is now found, and the earlier lead (`$4740`) was a red herring.
Every live object in the `$D3FD` array is processed once per frame by `$2CBA` (bank 0),
which dispatches through a **word-pointer table at `$24B2`** (bank 0), indexed by
`type×2`. This is the real master object dispatch: it is valid for types **`$00`–`$56`**
(`$4D` and `$4F` are null — unused slots; `$57` is the end), so it covers *all* the
play-tested types, not just the low ones. The chosen routine runs, then falls through to
the common "apply velocity to position" code at `$2CD4`.

The handlers live in the **home banking config (banks 0/1/2)**: the object loop pages
banks 1 and 2 into slots 1/2, runs, then restores the level-graphics banks (4/5). So a
table address in `$4000`–`$7FFF` is in **bank 1** and one in `$8000`–`$BFFF` is in
**bank 2**. (The old `$4740` table in bank 3, indexed `type×4`, only covers `$00`–`$23`
and is a *secondary* interaction table, not the per-frame dispatch — it does not reach
the bosses, the capsule, the seesaw, etc., which is why it looked incomplete.)

Names below come from play-testing, each confirmed against a real handler at the `$24B2`
slot:

| Type | Name | Handler (`$24B2`) | Notes |
|---|---|---|---|
| `$00` | Sonic | b1 `$4AD0` | the player (object 0) |
| `$01`/`$02`/`$03` | bonus item | b1 `$5DE1`/`$5EB1`/`$5EDD` | |
| `$04` | shield power-up | b1 `$5FAF` | |
| `$06` | chaos emerald | b1 `$6183` | |
| `$07` | goal sign | b1 `$61F8` | end of act. Loads its **own sprite sheet** over VRAM `$2000` (bank 9 file `$27AB8`) + palette `$0E` on its first frame; idles **static** on the "?" plate (state sequence `$64C2`), and when Sonic crosses it hops and **spins** (per-frame sequence `$64A8`, the same one-byte-per-frame format as Sonic's animations, plates 0→3→2→4 × 6 frames; layout = `$652D` + plate×18), stopping on the outcome plate (`$64C5`/`$64DF`/`$64F9`/`$6513`). Also clamps the camera (`$D26D/$D26F`) so the act ends at the sign |
| `$08` | **crab** | b1 `$65F9` | walker (~0.16 px/frame) that stops to fire a projectile each side; 4 in Green Hills Act 1 |
| `$09` | swinging platform | b1 `$6747` | pendulum: 180° arc, radius ~51 px, ~3.7 s/cycle; carries Sonic |
| `$0E` | bird | b1 `$6BD9` | enemy |
| `$0F` | horizontal platform | b1 `$6DCA` | back-and-forth: 1 px/frame, 160 px out-and-back; carries Sonic |
| `$10` | **beetle** | b1 `$6E65` | enemy: marches back and forth at 1 px/frame (same script engine as the crab, no attack) |
| `$12` | **world 1 boss** | b1 `$7065` | Robotnik's pod; bytecode-scripted sweeps; 8 hits to defeat (`$D2ED`) |
| `$25` | capsule | b1 `$736B` | jumped on to free the animals — ends each world |
| `$26` | fish | b1 `$7D25` | enemy: jumps 128 px (4 blocks) straight up, ~2.6 s/cycle |
| `$2C` | **world 3 boss** | b2 `$806B` | |
| `$2D` | porcupine | b2 `$82FB` | spiky walker, 0.25 px/frame, ~160 px patrol each way; no attack |
| `$48` | **world 2 boss** | b2 `$84AB` | |
| `$49` | **world 4 boss** | b2 `$9271` | |
| `$4E` | seesaw | b2 `$8681` | tilt-arm catapult; launch height scales with Sonic's landing impact (momentum transfer) |
| `$0B` | sinking platform | b1 `$69ED` | gives under Sonic's weight: sinks 16 px at ½ px/frame while stood on, floats back up when left |
| `$29` | floating log | b2 `$7EFC` | Jungle's rideable log: floats on the water (spawns lifted 24 px, then a gentle 5 px bob on a 40-frame cycle); ridden, it becomes a **log-roll** — Sonic steers it at *half* his speed (the handler moves the log by `vel/2` and writes the log's X back to Sonic), the roll phase (`IX+18/19`, mod `$900`) picking one of **three roll layouts** (`$803A`+n×18, the rotating grain) — and it stops against solid terrain probed beside it |
| `$3B` | bobbing platform | b1 `$B4E8` | free-floating (no terrain contact): Y-velocity ±$10 sub-px/frame on an 160-frame phase, clamped to ±2 px/frame — from the spawn it sinks ~160 px, then bobs in a 96 px, 160-frame cycle (Bridge's floating logs; rideable via `$7CF5`) |
| `$50` | background animator | b1 `$7B29` | repaints its *own* map cell each frame through the scroll-draw request (`$D2AB/$D2AD` target, `$D2AF` source): a 4-phase sequence of block pairs (`$7BC1` patterns → `$7B99` blocks) — the growing flowers and twinkling sea. Steady-state timing: bare and full-grown hold ~240 frames, the transitions 16 (`webexport` emits each placement as a `cellAnims` entry and the Studio viewer runs them). Not a camera lock: `$D2AB` is the blit request, the camera is `$D254` |
| `$51` | **checkpoint** | b1 `$6010` | on contact, writes the *checkpoint's own* block position into the respawn table `$D32F + act×2` (the `$6034` respawn-save code) |

Unnamed but present (handlers confirmed, behaviour not yet identified): `$05` b1 `$5FD7`,
`$0A`/`$0C`/`$0D` b1, `$11` b1 `$6F61`, `$13`–`$24` (mostly bank 2),
`$27`/`$28`/`$2A`/`$2B`, `$2E`–`$3A`, `$3C`–`$4C`, `$52`–`$56`. The full table is dumped by inspecting `$24B2`.

### Object sprites — the metasprite format

An object's graphic is a **metasprite**: a small grid of hardware-sprite tiles, drawn at
the object's screen position. Each object record holds a pointer to its current layout in
**`IX+15/16`**, and the per-object draw (the `$2CD4` tail at `$2EDE`, which calls the
sprite writer `$2F07` when that pointer is non-zero) interprets it as a **3-row × 6-column grid of 8×16-sprite
tile indices** — 18 bytes. The writer walks the grid emitting one sprite record per cell,
advancing **+8 px per column** and **+16 px per row**; a cell value of `$FE` is a *skip*
(a hole in the shape) and a `$FF` at the start of a row *ends* the sprite. Because the VDP
runs in 8×16 mode, a layout byte `b` draws tile `b` on top of tile `b+1`.

The referenced tiles come from sprite VRAM, which the level loader fills in **two**
pieces: the **per-zone sprite tile set** decompressed (the `$0406` codec) to `$2000` =
tiles `$00`–`$7F` (descriptor **+23** = bank, **+24/25** = address; **+26** = the sprite
**palette** index, stored to `$D22D` and resolved through the bank-8 `$7400` table), and
a **common sprite block** decompressed to `$3000` = tiles `$80`–`$BF` (bank 11, file
`$2F354` — HUD digits, sparkles, springs, the item-box bottom; verified byte-identical
to live VRAM). A layout may reference either — the item box's bottom row is tiles
`$AA`–`$AF` in the common block, which is why a zone-sheet-only render lost its base.
Two further slots are dynamic: the item handlers lazily stream a 16×16 **screen icon**
into tiles `$5C`–`$5F` (`$0BA8`: 8 bytes/frame from bank 5; the `LD HL,nn` before the
call names each type's icon — bonus `$01` = `$15200`, `$02` = `$15280`, emerald =
`$15480`), and Sonic's animation frames stream into `$B4`–`$BB` (see *Sonic's spawn*).
So an object sprite is *the composed sheet* plus an *18-byte layout* held in the
handler's bank — all reproduced by `objplace.SpriteSheet`/`ApplyIconUpload`.

Most animated enemies don't set `IX+15/16` directly; they go through the **shared
animation routine `$7C75`**. It takes the object's *frame-layout base* in `DE` and an
*animation sequence* in `BC` (pairs of `frameId, duration` — a step shows for
duration+1 frames — looping on `$FF`), and sets `IX+15/16 = base + frameId × 18` — so an
enemy's frames are consecutive 18-byte layouts, and frame 0 (the base) is the idle pose.
The sequence pointer itself usually comes from a small **per-state pointer table** read
just before the call (`LD HL,table / ADD HL,DE / LD C,(HL) / INC HL / LD B,(HL)`); entry
0 is the default walk. Reading those tables gives each enemy's full animation: the crab's
four-step claw walk (`$66F5`: frames 0,1,2,1 × 13), the beetle's two-frame walk
(`$6F14`: 0,1 × 9, with entry 2 the facing variant) and the bird's fast two-frame flap
(`$6D4F`: 0,1 × 3). `objplace.Anim` extracts these statically and `cmd/spriterip`
exports every type as a horizontal strip of 48×48 frames with the engine durations in
`sprites/index.json` — the Studio viewer plays them in its object layer.

Two families cannot be extracted this way. The **bosses** (`$12`/`$2C`/`$48`/`$49`)
are self-contained set-pieces: each decompresses its **own graphics over the zone sprite
sheet** (`LD HL,src / LD DE,$2000 / LD A,bank / CALL $0406` in the handler) and draws a
multi-part sprite from its script — a single extracted metasprite renders as tile salad
(the "mess in the sky" early in Bridge Act 1 was the Egg Mobile flyby drawn with the
wrong tiles). The **capsule** (`$25`) draws with the boss's runtime tiles too. The
extractor skips them (`objplace.OwnGfx`) and the viewer shows its labelled marker.

Initial facing matters as well: a walker's handler has one `$7C75` site per direction,
each preceded by its X-velocity store — the extractor prefers the **negative** one, since
every walker starts toward the player (the porcupine's first site is its facing-*right*
walk, which is what briefly pointed it the wrong way in the viewer).

One "animation" turned out not to be one. The pickup TV alternates two layouts on the
frame rotor (`$5E17`): the full metasprite (`$5E97`, 3 of 8 frames) and a variant that
*drops the middle top cell* (`$5EA4`, 5 of 8). Composited with the icon overlay the two
frames are **pixel-identical** — the opaque 16×16 icon (sprites `$5C`/`$5E` at +4/+12,
emitted before the metasprite and therefore on top) fully covers the alternated cell. The
blink is a **sprite-per-scanline budget trick**: the VDP draws at most 8 sprites per
line, so the game drops the invisible covered cell most frames to free a slot on the
TV's lines. The extractor detects the identical frames and exports the TV as static —
which is also exactly how it looks in play. Platforms, items and bosses instead load an
explicit layout pointer; some pick it **by zone** (`LD HL, zone0-ptr` then `LD A,($D2D5)`
overwriting `HL` with the zone-1.. variants), e.g. the swing platform is `$6910` in Green
Hills else `$6922`, the horizontal platform `$6910`/`$6930`/`$6922` for zones 0/1/else
(in the jungle it is a *log*). The World 1 boss decompresses its *own* graphics and points
at `$7359`.

**Where the sprite is drawn.** The draw is the tail of the shared move code (`$2CD4`, at
`$2EDE`): it reads the object's world position (`IX+2/3`, `IX+5/6`), subtracts the camera
(`$D254`/`$D257`), and calls `$2F07` — the metasprite grid's **top-left lands exactly at
the world position**, with no origin offset of any kind. Where the art sits *inside* the
48×48 grid is authored into the layout (Sonic's standing frame fills cells (0,0)–(1,1), so
his visible sprite is exactly his 16×32 hitbox at the grid origin).

**Where the object *is*, though, is not the spawn.** An object is spawned (`$1AB3`) at
`(blockX×32, blockY×32)`, but that is the top-left of its **hitbox** (width `IX+13`,
height `IX+14`, stored by each handler as its first act — crab 16×31, beetle 10×16, item
box 20×24), and the same shared move code runs the **floor snap** (§2) on the object's
first live frame: if the hitbox's bottom centre is inside a block with a floor profile,
`Y := rowBase + profile − height` — the object pops **up** to stand on the surface. A crab
placed *as* the ground block at `(42,13)` (y = 416) rests at **y = 401** = surface 432 −
height 31 the moment the camera gets near (objects are dormant until the `$2BD8` cull pass
activates them — margins from the per-type table at `$2560`). The pickup family (item
boxes `$01`–`$05`, emerald, checkpoint, continue) additionally applies a one-shot spawn
adjust first (`$6089`): **X += 4** (centring the monitor on its totem block), or
**(+22,+22)** if the block at its position is `$B0` — an item socketed into a Green Hills
slope. A handler with per-frame gravity (and terrain collision) additionally pulls its object
down to the floor **below** when the spawn block has none: Bridge Act 1's porcupine
spawns at (480, 320) over a gap and drops 48 px onto the lower ground the moment it
activates — the exporter detects the gravity idiom (`objplace.HasGravity`, minus the
`SET 5,(IX+24)` no-collision types) and applies the same drop. All of this is
reimplemented in `extract/objplace` and **verified against the live engine** by
`cmd/objsettle`, which watches the `$D3FD` array as the camera sweeps Green Hills Act 1: every stationary object's live rest position matches the static prediction
exactly (walkers/flyers differ only by their own first step). The extractor keeps each
sprite as the **whole 48×48 metasprite grid** (transparent where the layout has no tile)
and the viewer draws it with its top-left at the object's **rest position** — so the crab
stands on the grass, the item box sits on its pillar, and the beetle (whose 16-px height
happens to equal its block's surface offset, `352 = 368 − 16`) doesn't move at all.
`cmd/placeshot` renders a level strip with everything placed this way for a by-eye check:

![Green Hills Act 1 with engine-placed objects](rendered/placement_greenhills1.png)

This is enough to extract every object's sprite **straight from the ROM**, with no
emulator. `cmd/spriterip` reads each `$24B2` handler for its layout pointer (the `DE` base
fed to `$7C75`, or a direct `IX+15/16` load), decompresses each zone's sprite tile set and
palette, and renders the metasprite — **68 sprites across the seven zones**: the crab,
beetle, bird, fish, porcupine, the moving/swinging platforms (the jungle's are *logs*), the
item boxes, and all four world bosses, each in its own zone's colours. The **goal sign** is the one own-gfx object that is still a plain metasprite, so it IS
extracted: its own sheet (bank 9 `$27AB8`) and palette (`$0E`), spinning through plates
`$652D`+n×18 (the viewer shows the static "?" plate with a periodic spin — in play it
spins only while Sonic crosses it). The few that stay markers are invisible triggers and
the boss set-pieces. The Studio viewer draws all of this in its **Objects** layer.

### Horizontal moving platform (`$0F`)

The handler (bank 1 `$6DCA`) is a tidy worked example of how a behaviour reads off the
code. The platform keeps a 16-bit **phase counter** (`IX+18/19`) and a **direction byte**
(`IX+20`) in its object record. Each frame it increments the counter and adds `±1` to its
integer X (`IX+2/3`) — so it glides at **exactly 1 pixel per frame**, with no sub-pixel
fraction. When the counter reaches **`$A0` = 160**, it resets to zero and toggles bit 0 of
the direction byte, flipping the sign. The result is a symmetric oscillation about the
placed position: **160 px (5 macro-blocks) out and back**, reversing every **160 frames
(~2.7 s at 60 Hz)**, starting rightward (the record is zero-cleared at spawn). Sonic is
carried along explicitly: the handler runs a contact test (`($D215) = $0806`, `CALL
$3328`); if he is standing on top it glues him on (`CALL $7CF5`) and then adds the
platform's per-frame `±1` *directly to Sonic's X* — which is `($D3FF)`, because Sonic is
object 0 at `$D3FD` and `+2` is his X word — so he slides 1:1 with the platform. The whole
ride is skipped when `($D409)` is negative (Sonic not in a standable state, e.g.
mid-jump). The platform's artwork is chosen by zone (`$D2D5`): zone 0 → `$6910`, zone 1 →
`$6930`, else `$6922` — same motion, different sprite per zone.

### Swinging platform (`$09`)

The pendulum platform (bank 1 `$6747`) records its placed position as an **anchor**
(`IX+18/19` = pivot X, `IX+20/21` = pivot Y) on the first frame, then each frame reads a
position from a **113-point arc table at `$682E`** and sets `X = anchorX + table_dx`,
`Y = anchorY + table_dy`. The table is a signed `(dx, dy)` list tracing a **semicircle of
radius ~51 px** (≈ 1.6 blocks) *below* the pivot: it runs from `(-51, 0)` on the left,
down through `(-2, +51)` at the bottom, to `(+51, 0)` on the right — a 180° swing (9
o'clock → 6 → 3), 102 px wide, dipping 51 px at the lowest point. A phase index (`IX+17`)
walks the table and **ping-pongs between 0 and `$E0` (224) at ±2 per frame**, so one sweep
takes 112 frames and a full there-and-back is **224 frames (~3.7 s at 60 Hz)**. Sonic
rides it the same way as the horizontal platform: the handler keeps the frame's X
displacement in `($D20F)`, runs the standing test (`$3328`), and on contact adds that
delta to Sonic's X (`($D3FF)`) while `$7CF5` keeps him glued on top — so he follows the
arc both horizontally and vertically. Artwork by zone: zone 0 → `$6910`, else `$6922`.

Both moving platforms' cycles are exported for the Studio viewer as **movement paths** —
per-frame `(dx, dy)` offsets from the placement, sampled straight from the handlers'
own tables (`objplace.PlatformPaths` → `sprites/index.json` `paths`): the swing's
224-frame arc sweep, the horizontal platform's 320-frame triangle, the bobbing
platform's 160-frame float and the floating log's 40-frame water bob. The viewer moves
the sprites along them at engine timing.

### Sinking platform (`$0B`)

The third platform type (bank 1 `$69ED`, same zone artwork as the other two) doesn't move
on its own: when Sonic stands on it (`$3328`), it takes on a downward velocity of ½ px per
frame until it has sunk **16 px** into its block (`Y & $1F ≥ $10`); with no rider it
floats back up to its resting height. A platform that gives under his weight.

### Sonic's animations (`$5C5B`)

Sonic is not animated through `$7C75` like the enemies: his handler keeps **one**
metasprite layout (`$5C1B` — his 16×32 box at the grid origin) and instead re-streams
the tile **graphics** each time his pose changes (`$4E9A`: source = bank 8 +
graphicFrame × 192, 3bpp, into the dynamic sprite tiles `$B4`–`$BB`). The animation
sequencer (`$4E6D`) maps the anim id (`IX+20`) through a **word table at `$5C5B`** to a
sequence that is simply **one byte per engine frame** — the graphic frame to show — with
a bit-7 byte as the control *set cursor* (the loop point). Decoded and oracle-verified:

| anim | sequence | meaning |
|---|---|---|
| `$01` | frames 4–9, 8 frames each, loop | the 6-frame **run cycle** |
| `$02` | frames 11–14 × 4, loop | **rolling** |
| `$05` | frame 0, loop | **standing** |
| `$0A` | 25 × 6, 26 × 6, loop | skid |
| `$0D` | 2 × 16, 1 × 18, then loop (2 × 16, 3 × 16) | **bored** — he turns to the camera and taps his foot (set by the idle timeout, `$5379`) |

The Studio viewer plays the idle → bored program for the placed Sonic: standing for the
game's ~6-second idle timeout, then the foot-tap (the strip and the explicit play
sequence are exported by `cmd/spriterip`; `objplace.SonicSeq` reads the table).

### Seesaw (`$4E`)

The catapult (bank 2 `$8681`) is the most involved of these — a tilting arm with a weight,
simulated rather than scripted. A single byte `IX+17` holds the **tilt angle**, an integer
`0…$1C` (28 steps); the two standable ends sit at *complementary* heights derived from it
(one collision box at `IX+17`, the other at `$1C − IX+17`), so as one end drops the other
rises. The weight is a little physics loop: a 16-bit tilt **velocity** (`IX+18/19`)
accelerates by **`+$0038` per frame** and integrates into a fine position accumulator
(`IX+20…22`), i.e. the loaded side falls under gravity. The catapult itself is **momentum
transfer**: when Sonic lands on an end (one of two `$3328` tests, gated by `($D409)` ≥ 0 =
grounded), his downward impact speed — read from `($D407)` — is *negated* into the tilt
velocity (`IX+18/19`) and added to the angle, then a sub-action fires (`RST $28`, index
`$04`). So a harder landing tilts the arm harder, which throws whatever is on the opposite
end higher. Because the launch is driven by impact speed, **there is no single fixed launch
height** — it scales with how fast Sonic comes down, exactly matching the in-game feel
(jump on the high end, the weight on the low end is flung up, and on its way down it
catapults Sonic). This one I've only read statically; the two-stage momentum sim would be
worth confirming against footage if we want exact numbers.

### Crab (`$08`)

The crab (bank 1 `$65F9`, a Crabmeat-style walker, box 16×31) is **script-driven**. A
16-bit counter (`IX+17/18`) advances `+8` per frame, and its high byte indexes a 26-entry
**state table at `$66D0`** — so each entry holds for 32 frames, and reading a `0` wraps to
the start. The entries are state codes: `1` = walk right, `2` = walk left, `3` = stop,
`4` = **attack**. The sequence is *walk right (9 entries ≈ 288 frames) → stop → attack →
walk left (10 entries) → stop → attack → repeat*. Walking is slow: the state sets an X
velocity of only `$28/256` ≈ **0.156 px/frame**, so each leg covers ~45 px. The attack
state (when the sub-phase byte `IX+17 == $20`) spawns a projectile to each side — it calls
the spawner `$AC5C` twice with direction `($D213) = −1` then `+1` and fires `RST $28`
index `$0A` — i.e. the classic crab that stops and shoots both ways. Gravity (`+$0020`/
frame on the Y velocity) keeps it on the ground; touching it calls the hurt routine
`$2FC1`.

### Beetle (`$10`)

The beetle (bank 1 `$6E65`, box 10×16) uses the **same script engine** as the crab — a
`+8`/frame counter into a state table at `$6EEF`, 32 frames per entry — but a simpler
program and no attack: `1` = walk left, `2` = walk right, `3`/`4` = stop (the two "stop"
codes differ only in which idle sprite plays). It marches *left (9 entries ≈ 288 frames) →
stop → right → stop → repeat* at a brisk **1 px/frame** (`±$0100` velocity), ~6× the
crab's pace. It clears the "solid" flag (`RES 5,(IX+24)`) so Sonic can't stand on it, and a
constant `+2 px/frame` downward component (`IX+10..12 = $000200`, reset every frame rather
than accumulating) keeps it pinned to the surface as it crawls. Contact calls the hurt
routine `$2FC1`.

### Fish (`$26`)

The fish (bank 1 `$7D25`) is a pure vertical jumper that hides underwater between leaps.
It keeps a **cooldown timer** in `IX+20`: while it is non-zero the fish decrements it and
blanks its sprite (`IX+15/16 = 0` — invisible), so it sits unseen until the timer expires.
When it fires, it sets an upward velocity of **−4 px/frame** (`IX+10..12 = $FFFC00`, a
24-bit 16.8 fixed-point value) and flags itself airborne; thereafter each frame **gravity
adds `$0010` (≈ 0.0625 px/frame²)** to that velocity, with the downward speed clamped to a
terminal **+4 px/frame** (`$0400`). The launch and gravity give a peak height of
`v² / 2a = 4² / (2·0.0625) =` **128 px (4 blocks / 8 tiles)**, reached after 64 frames; the
fall is symmetric, so it is airborne ≈ 128 frames. When it falls back to its reference
height (`IX+18/19`, recorded once at launch) the handler snaps it to rest, zeroes the
velocity, and reloads the cooldown with **`$1E` = 30 frames (0.5 s)** — so the full cycle
is roughly **~158 frames ≈ 2.6 s**. There is no horizontal motion (only a one-time +8 px
nudge); the X velocity is never touched. On the way up it triggers a sub-action
(`RST $28`, index `$12` — the splash) and, via the contact test (`($D215) = $0204`,
`$3328`), calls the hurt routine `$2FC1` if Sonic touches it.

### Porcupine (`$2D`)

The porcupine (bank 2 `$82FB`, boxes up to 20×32) is a plain spiky ground-walker with no
attack — the spikes *are* the threat. A phase counter `IX+17` is incremented by 1 **every
8 frames** (gated on `($D224) & 7`) and wraps at `$A0` (160); its value picks the
direction — walk left while `IX+17 < $50`, walk right otherwise — at a slow **±0.25
px/frame** (`±$40/256`). So it spends 80 ticks (≈ 640 frames) crawling each way, covering
~160 px (5 blocks) before it reverses — roughly a 21 s round trip. Gravity (`+$0020`/frame
on the Y velocity) keeps it on the ground. It runs **two contact tests** — a tight `$0608`
box and a larger `$1006` box — both of which hurt Sonic on touch (`$2FD9` / `$2FC1`, the
two hurt-Sonic variants).

### World 1 boss (`$12`)

The world-1 boss (bank 1 `$7065` — Robotnik in his pod) is the most elaborate object here,
and it sets *itself* up as a self-contained set-piece. On spawn it decompresses its own
sprite graphics from bank 9 (`$A8B7` → VRAM `$2000`) and loads its palette (index `$11`),
then arms a small **bytecode script**: a program counter (`IX+18`) walks a script whose
base is `(IX+20/21)`, and each opcode is dispatched through a jump table at `$729A` (a `0`
byte is a jump — the following byte is the new PC). That script flies the pod **back and
forth across the arena** — bounded by `($D26D) + $22` on the left and `+ $BA` on the right
(~152 px) — with vertical swoops, overlaid on a gentle bob read from a 64-entry waveform
table at `$72B0`, while it trails exhaust particles (`$7A36`). The fight is decided in
`$77FD`: it runs a contact test, and if Sonic touches the pod while **not** in his
attacking (rolling/jumping) state — checked via `($D415)` — Sonic is hurt and knocked back
(`$2FD9`). If he *is* attacking, the pod takes a hit: a **hit counter at `($D2ED)`**
increments, Sonic rebounds, and the boss flashes invulnerable for `$18` = 24 frames
(`($D2B1)`). It takes **8 hits** (`($D2ED)` reaching `$08`) to win — at which point the
defeat sequence (`$787D`) runs and the captured animals are freed. (Read statically; the
8-hit count and the attacking-state gate are the parts worth confirming against play.)

### Sonic's spawn

Sonic is object 0; the loader places him at the position the spawn pointer `($D217)`
points to, × 32 — **exactly** `(blockX×32, blockY×32)`, no offset (earlier readings of a
"+9" origin offset and a clouds-height spawn were capture artifacts of a dormant-slot
probe). On a fresh act start the pointer targets the level descriptor's RAM copy at its
spawn field (`$D355+13` = `$D362`, i.e. **descriptor bytes +13/+14**; Green Hills Act 1 =
block (5,11) → (160, 352), oracle-verified). After a death, `$1959` re-points it at the
act's checkpoint entry (`$D32F + act×2`, written by the checkpoint object via `$6034`) —
same mechanism, different table.

The spawn is placed a little off the floor and the handler settles him within the level
fade-in — with one subtlety that matters: his handler's **initial state uses the short
16×21 box** (`$55D8`; the paired ±11 Y adjustments at `$55CE`/`$4CAA` keep his feet fixed
across every 21↔32 box switch), so the *first* floor probe runs with his feet at
`spawnY+21`. That catches a floor line inside the spawn block itself: in **Bridge Act 1**
and **Labyrinth Act 1** the spawn block carries the floor (profile 16) and Sonic snaps
**up** 16 px onto it — with a 32-tall probe he would fall straight past it, which is
exactly the bug the first export had. Everywhere else the pattern is strikingly
consistent: a **16 px drop** onto the floor line below (GH1: 352 → 368, feet at 400),
with the real exceptions being Bridge 2 (48 px), Sky Base 3 (112 px) and the Special
Stages (208–368 px — their trademark visible fall). A vertical act spawns at the
**bottom** (Jungle Act 2 — the start of the climb). The camera starts 3 blocks left of
him (`$D251 = blockX − 3`, `$1959`). All 18 acts oracle-verified rest-exact by
`cmd/spawncheck`.

One more field mattered here: the collision zone is the descriptor's **byte +0** (the
loader's `$D2D5`), *not* `act÷3` — **Sky Base 3 runs as zone 7**, the interior
pseudo-zone it shares with the Sky Base 2 teleporter room, with its own `$343D`
attribute table. Using `act÷3` read the wrong shape table for that act (and for the
viewer's collision layer).

The Studio viewer places Sonic's standing sprite at the settled position
(`objplace.DropToFloor`); his standing frame itself is ripped from ROM — the player state
code (`$4E9A`) computes the tile-stream source as *bank 8 + frame×192* (8 tiles × 24
bytes, **3bpp**, streamed into the dynamic sprite slots `$B4`–`$BB` at VRAM `$3680`
whenever his pose changes), and frame 0 at file `$20000` is the standing pose, verified
byte-identical to live VRAM.

### Checkpoint (`$51`)

The respawn point — where Sonic reappears after a death — lives in a **table at RAM
`$D32F`** (`$D32F + act×2`, one block coordinate per act), and it is written by the
**checkpoint** object (type `$51`, handler bank 1 `$6010`). The handler has a 20×24
collision box (`IX+13/14`) and runs a contact test against Sonic each frame (`($D215) =
$0003`, `$3328`), refined by a horizontal-proximity check (`$60CC`). On the first contact
it runs the save at `$6034`: `($D238)` (the act) × 2 indexes `$D32F`, and the **checkpoint
object's *own* position** — `IX+2/3` and `IX+5/6`, each `×8` and taking the high byte to
convert pixels → blocks, with the Y block minus 1 — is stored there. (So you respawn *at
the checkpoint*, one block above it, not at the exact pixel you touched it — and note this
saves the checkpoint's coordinates, **not** Sonic's, correcting an earlier reading.) It
also sets a per-act "checkpoint reached" bit in the `$D312` bitmask (`$0B8D` picks the bit
by act number) so the save fires once, then draws its sprite (`$5500` via `$5E0C`).

This resolves the long-standing guess: the checkpoint *is* a placed object, but it is type
`$51`, not `$50` (which turned out to be the background animator — it feeds the `$D2AB`
scroll-draw request, repainting its own map cell).

### Special-stage objects — bumper (`$21`), Continue powerup (`$52`)

The bonus stages (Part IV — the "Special Stage" zone) use a small object cast of their own.

The **bumper** is type `$21` (handler bank 2 `$9AD0`) — placed in the stage tables (3–6 per
round), and the object an earlier pass mislabelled a "ring". It oscillates horizontally (phase
`IX+18`, `0…$C0`) around its anchor and, on contact (`$3328` box `$0602`), **reverses Sonic's
velocity** (`$D2E7/$D2E9` are his negated velocity; it writes the negation back to `$D407`)
and plays the bounce sound (`RST $28` idx `$07`) — a spring/bumper, not a collectible.

The **Continue powerup** is type `$52` (handler bank 1 `$6061`). On contact (`$3328` box
`$0003`, refined by `$60CC`) it sets a flag (`(IY+9) bit 3`) and runs the collect path
(`$5E01`). (Its exact award — a Continue — is from in-game observation; the handler only shows
the collect.)

(Type `$20`, handler `$96FA`, is another sprite-drawn object in this zone — drawn through the
`$D2DF` sprite allocator — but its role isn't pinned down, so it is left unlabelled.)

**Where the rings come from.** The bonus stages are full of rings, but **none of them are
objects** — they are baked into the level the same way as every normal zone: block indices
`$79`–`$7B` (the special-stage map uses ~290 such cells), which expand to tiles `252`–`255`.
Those four tiles are *empty* in every zone's base set, including this one; the engine's
animation update fills them with the spinning-ring frames at runtime (Part IV — animated
tiles). So a static render must load a ring frame to show them, exactly as for Green Hills'
rings. (There is no "place a cluster of rings" object — the ring fields are authored straight
into the block map.)

### Teleporter (`$13`) — and the hidden Scrap Brain maze

Object type `$13` (handler bank 2 `$9B4A`) is an **invisible teleporter**: it draws no sprite
(`IX+15/16 = 0`), just a 30×96-px contact box. On contact it converts its *own* block
position to a `(blockX, blockY)` key, looks it up in a **5-entry table at `$9BAE`**, and writes
the matched **destination scene** to `$D2D4`, sets `($D283) = 1` and `(IY+6) bit 4` — the very
same "launch this scene next" mechanism the title screen's Start button uses (`scene_run
$1414` loads `$D2D4` instead of `$D238` when that flag is set). So a teleporter's destination
is **hardwired to where it sits**:

| Teleporter block | → scene |
|---|---|
| `(124, 1)` | 20 |
| `(124, 25)` | 21 |
| `(1, 1)` | 25 |
| `(1, 59)` | 24 |
| `(20, 15)` | 26 |

Mapping every placement against that table exposes something the act list hides: **Scrap Brain
Act 2 is not one level but seven** — scene 13 plus six **sub-scenes (table indices 20–25,
named here Scrap Brain Act 2a–2f)** that never appear in the 18-act table and are reachable
*only* through teleporters. They are wired into a maze with deliberate **loops** — e.g. Act 2a's
pad `(1,59)` → 2e, and Act 2e's pad `(124,1)` → 2a, so the wrong choice sends you in circles
(the infamous Scrap Brain Act 2 maze). Several sub-scenes share a map source entered at a
different spawn, so they *look* alike — which is the whole trick.

The other teleporter is in **Sky Base Act 2**: the pad at `(20, 15)` warps to **scene 26
(Sky Base Act 2a)**, a **7th pseudo-"zone" (`zone = 7`)** — a Scrap-Brain-styled fortress
interior (it reuses that tile set) holding a **goal sign and a Chaos Emerald** (`$06`). So the
route to that emerald is a hidden teleporter room, outside the normal act flow entirely.

All seven hidden scenes are exported as their own viewable acts (`extract/cmd/webexport`).

## 2. Movement and collision

Sonic himself is just object type `$00`, dispatched like any other through `$24B2` to the
handler at bank 1 `$4AD0` — but that handler is the largest in the game, and it is where
the controller becomes motion. This section traces the spine *input → acceleration →
friction → speed → world position*, plus rolling and the terrain sampler. The fine
vertical resolution (slopes, gravity, the jump arc) is located but not yet fully decoded —
flagged as the frontier at the end.

### Reading the controller

The pad is sampled once per frame by `$0602` into `(IY+3)` — the raw Game Gear input byte,
**active-low** (a pressed button reads `0`). The player handler decodes it bit by bit:
bit 1 = **Down**, bit 2 = **Left**, bit 3 = **Right**, bit 4 = **Button 1 / jump**, with
`$FF` meaning "nothing pressed" (used as an idle test at `$4C52`). So the handler is a
chain of `BIT n,(IY+3)` tests that route to small per-direction routines.

### Horizontal acceleration, skidding and friction

Before the directional code runs, the handler picks a **physics parameter set** for the
frame and copies it into three words:

- `($D23A)` — **acceleration** (added to speed while a direction is held),
- `($D23C)` — **friction / deceleration** (a negative value applied when nothing is held),
- `($D23E)` — the **top-speed cap**.

Which set is chosen depends on Sonic's state flags (`IX+24`) and whether he is on the
ground. Two groups are loaded per state: a **9-byte horizontal block** copied to `($D20F…)`
(its first word is the **top-speed cap**, `$D213` an acceleration term), *and* three
separate words `($D23A/$D23C/$D23E)`. Crucially — and this corrects an earlier draft of
this section — `$D23C`/`$D23E` are **not** horizontal friction and cap; they feed the
*vertical* (jump/gravity) path (next sub-section). The raw values:

| State | h. accel `$D23A` | top speed `($D20F)` | jump impulse `$D23C` | gravity `$D23E` | source |
|---|---|---|---|---|---|
| running on ground | `$0300` | `$0010` | `$FD00` | `$0038` | `$4FD7` |
| (variant) | `$0C00` | `$0010` | `$FD00` | `$0038` | `$4FE0` |
| rolling / ball form | `$0100` | `$0004` | `$FDC0` | `$0010` | `$4FE9` |

Rolling has the **smallest acceleration and the lowest top speed** (`$0004` vs `$0010`) —
you can't speed up by rolling — exactly the Mega Drive feel. (Constants are raw fixed-point;
the exact px/frame scale is pinned with the position integration, still on the frontier, so
this is *roles and values*, not yet pixels per frame; and the precise split between `$D23A`
and the block's `$D213` accel term, plus why gravity `$D23E` differs when rolling, is still
being pinned.)

Holding **Right** jumps to `$515E`, **Left** to `$51B9`. Each adds the acceleration to
Sonic's speed in the pressed direction (clamped to the top speed in `($D20F)`) and sets the
animation index `IX+20` (`$01` = run). The neat detail is the **skid**: if you press the
opposite way to your current motion, the routine takes a *braking* branch instead — it
applies a larger fixed deceleration (`$0100`) and sets the skid animation (`IX+20 = $0A`)
until the speed crosses zero and Sonic turns around. With **no** direction held, a no-input
path (around `$4CDE`) instead decays the speed toward zero using the block's deceleration
term (`$D213`) — the ground friction.

### From speed to a position on the map

The computed speed is written back into Sonic's own velocity words in his object record —
`($D404…)` and `($D407…)` (i.e. `IX+7…` and `IX+10…`, since Sonic is object 0 at `$D3FD`) —
and a sign-flipped copy is kept at `($D2E7/$D2E9)` (the same "velocity and its negation"
pair the platforms and the seesaw read when they carry or launch him). The common-move
step then integrates velocity into his world position, `($D3FF)` X and `($D402)` Y. The
tail of the handler is mostly animation: `IX+20` indexes a frame table (`$5C5B` → `$5BE1`)
and builds the VRAM sprite source at `($D289)`, and the on-screen offset is clamped to a
few pixels (`$D405`/`$D408` capped to ±`$0A`/`$0C`) so the camera lag never tears the
sprite off-screen.

### Rolling

Pressing **Down** calls `$5335`, which is gated: it does nothing if Sonic is already a ball
(`IX+24` bit 0) or not on the ground (`IX+24` bit 7), but otherwise **sets `IX+24` bit 0 —
the rolling/ball flag** — and, *only if he is actually moving* (`($D404) ≠ 0`), fires the
roll sound (`RST $28`, action `$06`). Standing still and pressing Down is therefore a crouch
(the flag is set but there's no roll). Once bit 0 is set, the per-frame setup selects the
ball physics constants above (low accel, low friction) and the update takes a rolling
branch (`$4C92 → $55C7`), so a roll holds its momentum and coasts rather than accelerating —
you steer a little but mostly carry speed, rolling farther than a walking stop would slide,
just like the original.

### Ground collision — the terrain sampler

Collision reads the **decoded block map in the `$C000` RAM window** (Part IV §4: the
row-major level map streamed from bank 4/5/6). The key routine is **`$30D5`**: given an
object (via `IX`) and an `(X, Y)` offset in `BC`/`DE`, it returns `HL` = the address of the
**block index at that world point**, computing `row·stride + col` and adding `$C000`. It
even dispatches on the level's stride byte `($D232 = $80/$40/$20/$10/…)` — the *same*
variable that reshapes the map in the level decoder — so a sampled point lands on the right
block whatever the level's aspect. `$30D5` is a *shared* map reader — several systems use
it — so the Sonic handler calls it at the relevant offset, reads the block index
(`LD A,(HL)`), and dispatches on the value. One consumer is fully decoded below (ring
pickup); the **solid-terrain** consumer — the one that resolves Sonic's feet against ground
blocks — is still on the frontier. `$30D5` is the bridge between the static level format
already decoded and the live physics, whichever system is asking.

### Rings

The first decoded use of the sampler turns out to be **ring collection**, not solid
collision. Rings are baked into the block map (Part IV §4) as **block indices `$79`–`$7B`**
(blocks 121–123). A 32×32 block *could* geometrically hold a 2×2 grid of 16×16 rings, but
this engine only uses a **left/right pair** — the low **two** bits of the index mark which
of the two 16-px-wide halves still hold a ring: `$79` = left only, `$7A` = right only,
`$7B` = both (`$78` = empty). That is a deliberate choice in the *code*, not a guess: the
collector picks the ring purely from Sonic's **X** position — `$5000` does
`mask = ((X+8) >> 4) & 1, then +1` → `1` or `2`, and never consults Y for the bit (Y is
only a coarse on-screen gate). So vertically-stacked rings aren't four quarters of one
block; they're separate block cells in the rows above/below. Each frame the handler samples
the block at Sonic's centre (`$30D5` with offset `(8, 8)`), masks the priority bit
(`AND $7F`), and if the result is `≥ $79` calls **`$5000`**, which — if the overlapped
half's bit is set — **clears it from the live map** (`(DE) = block XOR mask`, so the
graphic downgrades two → one → none), spawns a sparkle, and calls the counter **`$337E`**.
`$337E` adds to the **BCD ring count at `($D2A9)`**, plays the pickup sound (`RST $28`,
action `$02`), and — the nice touch — when the count rolls past 100 grants an extra life
(`($D240)++` and the 1-up jingle, action `$09`). Because the rings live *in the map*,
collecting one is a single byte-write back into the `$C000` window; there is no separate
ring object array.

### Jumping and gravity

Walking the **`$4D60` vertical path** in full: it is the **Y-velocity integrator** — gravity
and jumping, writing Sonic's Y velocity `($D407…)`. Pressing jump on the ground calls
**`$5300`**, which seeds a hold-timer `($D288) = $10` and plays the jump sound — the
**variable jump height** mechanism: while the button stays held and `$D288` counts down, the
upward impulse `($D23C)` keeps being added (`$4D9D`); once released or the timer expires, the
path applies the downward constant `($D23E)` — **gravity** — each frame instead (`$4DCF` →
`$4DED ADD HL,DE` → store). That is the whole vertical force model. What this pass settles is
a negative: **the floor probe is *not* in this path** — `$4D60` only changes the *velocity*;
nothing here samples terrain or snaps Y.

### Springs, spikes and conveyors — the special-terrain dispatch

The terrain *interaction*, though, is now decoded. Each frame the handler samples the block
at Sonic's feet (`$30D5`, offset `(8, 16)`), and — after paging **bank 5** into slot 2
(`$4BEF`) — maps that block index through a **per-zone table** to a *collision type*: a word
pointer at bank 5 `$A200 + zone×2` gives the zone's table, indexed by the block index, giving
a type `$00`–`$1C`; types `≥ $1D` mean "nothing special". The type then indexes a **29-entry
handler table at `$5BE1`** and the handler runs (with bank 2 paged into slot 2 for its data).
Decoding the Green Hills (zone 0) map and the handlers gives a clear picture — this is the
springs/spikes/conveyor layer:

| Type | Handler | Effect |
|---|---|---|
| `$00` | `$5759` | ordinary block — just clears the jump-recharge flag (`IX+24` bit 4) |
| `$01` | `$5763` | **hurt** — calls `$2FD9` (spikes / hazard) |
| `$02` | `$576B` | bounce (recomputes Y velocity, sound `$05`) |
| `$03`/`$05` | `$57AC`/`$57F3` | **horizontal spring** — sets X velocity to ∓8 px/frame, sound `$04` |
| `$04` | `$57CA` | **vertical spring** — sets Y velocity to **−12 px/frame** (up), sound `$04` |
| `$06`/`$07` | `$5815`/`$582D` | **conveyor / slope drift** — adds a steady ±X to the sub-pixel position |

Each spring/hazard is gated on which 16-px sub-cell of the 32-px block Sonic is in (e.g.
`(IX+2)+8 AND $1F, CP …`), so only the active part of the block triggers. The `RST $28`
values are the sound effects. This is the *interaction* layer for special blocks (in Green
Hills almost every block is type `$00`, an ordinary block); the plain **solid floor** is a
separate routine, decoded next.

### The solid-ground floor — height profiles and slope angles

The plain floor is now fully decoded — and finding it first required fixing a **bug in our
own Z80 emulator** (see the box below), because that bug was corrupting the very Y value I
was trying to read. With a correct CPU the oracle behaves like hardware: Sonic falls, then
**lands** — his Y snaps to the surface and his Y-velocity is **zeroed** — and he then **runs
along the ground** without sinking. (So the earlier "velocity is never zeroed / clamped every
frame" reading was the emulator bug, not the game.)

The floor routine lives in **bank 0** (always mapped — which is why it never appeared in the
bank-1 player code), entered at **`$2DF4`/`$2E02`** and reached every frame through the common
object-update. It is generic over `IX`, so it serves every object, Sonic included. Per frame:

1. **Clear** the on-ground flag (`IX+24` bit 7) and **sample the block at the feet**
   (`$30D5`, `$2E16`) — yes, the same shared sampler, just called from bank 0.
2. **Solidity / shape.** The block index selects an attribute byte from a per-zone table at
   **`$343D`**; `AND $3F` is the **collision shape** (`0`–`$3F`). **Shape `0` ⇒ no
   collision** — Sonic passes straight through. *These are the non-solid blocks* — e.g. the
   ones at the start of Green Hills Act 2 that drop you into the cave. (Bit 7 of that same
   attribute byte is the render-priority bit from Part IV §4 — one byte, both jobs.)
3. **Per-column height.** The shape selects a **height profile** (pointer tables at `$3BDA`
   / `$3E7A`); Sonic's X-column within the 32-px block (`(IX+2)+… AND $1F`) indexes the
   profile to give the **surface height at that column** (`$80` = no surface there). Because
   the height varies per column, **slopes are smooth**, not 32-px steps — exactly as you'd
   expect from the way he glides up and down hills.
4. **Snap and stop.** On contact it writes Y to the block-aligned position plus that surface
   height (`$2EA6`), **zeroes the Y-velocity** (`$2EBA`, `IX+10..12 = 0`), **sets on-ground**
   (`$2E81`, `IX+24` bit 7), and stores the **surface angle** for the column — from a signed
   table at **`$3978`** (`$1C` = +28°, `$E4` = −28°, `$12` = +18°, …) — into `IX+25`, which
   the running code uses to move Sonic *along* the slope.

So the model is precisely the one predicted: a per-block solidity/shape, a per-column height
array for smooth slopes, and a per-shape angle for ground movement.

Reading the whole of the surrounding move code (`$2CD4` — the routine every `$24B2` handler
returns into; it had to be seeded as a disassembly entry because it is only ever reached as
a *pushed return address*) pins the exact geometry. "The feet" is the **hitbox's bottom
centre**: the probe point is `(X + width/2, Y + height)` with the per-type width in `IX+13`
and height in `IX+14` — so the object's world position is its hitbox **top-left**, and the
landing condition is `(bottom & $1F) + bias ≥ profile` with a per-shape **landing-bias
table at `$39DA`**; the snap is `Y := bottomRowBase + profile − height`. The same code runs
a **wall pass** first (`$2D15`): probe `(X + width, Y + height/2)` when moving right (or
`X` when moving left) against separate per-shape **wall profiles** (pointer tables `$3B0A`
right / `$3A0A` left, indexed by the *row* within the block; `$3BDA` is the ceiling
counterpart for rising objects), push out to `blockBase + profile − width`, zero the X
velocity, and add a per-shape **slope kick** (`$39A8`) to the Y velocity. Velocity and
position are 24-bit (sub-pixel byte + 16-bit integer: X `IX+1..3`/`IX+7..9`, Y
`IX+4..6`/`IX+10..12`). The placement corollary — objects rest at *surface − height*, not
at their spawn block — is what fixed the Studio viewer's object layer (see §1).

The shape field is 6 bits, but the floor pointer table at `$3E7A` has only **48 entries**
(`$00`–`$2F`) — shape 1's profile sits at `$3E7A + 48×2`, so the table ends there; values
`$30`+ would read into the profile data. Rendering all 48 profiles as solid/empty block
silhouettes (one column per byte, surface = the signed height, `$80` = no surface) makes the
vocabulary obvious:

![Collision height profiles](rendered/block_collision_profiles.png)

Shape `$00` and `$18`–`$26` are empty (non-solid placeholders, marked `-`); `$01`–`$0A` are
straight slopes of increasing steepness (½, full and double gradients, both directions);
`$0B`–`$17` are gently curved surfaces (bumps and valleys — note the U-shaped dip at `$11`);
`$27`–`$2C` are flats at various heights plus a peaked hill (`$2B`); and `$2D`–`$2F` are
half/edge blocks. The figure is built by reading the profiles straight from `$3E7A` in ROM.

> **Toolchain fix — `LD (IX+d),n` operand swap.** Chasing this floor snap, a value-capturing
> write trace showed an instruction at `$4C75` (`LD (IX+20),$05`) writing the *wrong* address
> and value. The cause was in `tools/z80`: `case 6 /*LD r,n*/ { c.setR(y, c.fetch()) }` — Go
> evaluates the `c.fetch()` argument first, so for `LD (IX+d),n` it read the **immediate
> before the displacement** and swapped them, corrupting every IX/IY-relative immediate store.
> Fixed (resolve the `(idx+disp)` address before fetching `n`) with a regression test; this
> also makes Sonic *run on the ground* in the oracle instead of sinking. A reminder that the
> oracle is only as trustworthy as the CPU under it — and that a falsifiable, value-level
> trace is what exposes such bugs.

**On tunneling** (a fair worry): the engine *does* zero the Y-velocity on contact, and the
fall speed is capped (terminal ≈ 8 px/frame, well under the 32-px block), and the foot sensor
is re-evaluated every frame, so Sonic can't out-run the floor check — no tunnelling.

Still to do: the remaining unidentified handler slots in `$24B2`; scoring and the timer.

## 3. Underwater — Labyrinth's flooded acts

Labyrinth is the only zone with water, and its three acts are *half-flooded*: a horizontal
water line splits each act into an air part (top) and a submerged part (bottom). Crossing it
changes three things at once — **the palette**, **the physics**, and a **drowning timer** —
and tracing the one flag that drives all three pins down exactly how the game does it.

### The water surface is an object

The water line is not a level constant; it is a real **object, type `$40`** (handler in
bank 2 at `$8D59`, reached through the master dispatch `$24B2 + $40×2`). It is the **first
object** placed in every Labyrinth act, and its placement block-Y *is* the water level:

| Act | Water object block-Y | Water world-Y (`×32`) |
|-----|----------------------|------------------------|
| Labyrinth Act 1 (`$D238`=9)  | 13 | **416** (`$01A0`) |
| Labyrinth Act 2 (`$D238`=10) | 27 | **864** (`$0360`) |
| Labyrinth Act 3 (`$D238`=11) | 10 | **320** (`$0140`) |

Each frame the handler writes its current world-Y to **`$D2DD`** (with a sub-pixel sine bob
from the wave table at `$8E4A`, so the surface ripples) and derives the **water-line
*scanline*** `$D2DC = waterY − cameraY($D257)`, clamped to the visible range `[$0C,$B4]`.
`$D2DC = $FF` means the line is above the top of the screen (the whole view is underwater);
`$00` means it is below the bottom (all air). The level loader (`$185D`) arms this only for
acts 9–11 (`$18B6`: `CP 9 … CP $0B`); every other act leaves `$D2DC = 0`, so the split code
never fires.

> The oracle can load a Labyrinth act but never *runs* the type-`$40` handler (its simplified
> object engine leaves `$D2DD` at the loader's placeholder `$0800`/`$0020`), which is why the
> underwater state could not be reproduced live earlier — this section is static, traced from
> the handler table and the IRQ.

### The palette swap is a raster split (`$0216`)

There is no second palette index loaded for water — the swap is a **mid-frame raster effect**.
The IRQ handler (`$0073`) programs a VDP **line interrupt** (register 10) at scanline `$D2DC`
(`$0091`–`$00A1`). When that line is reached, the line-interrupt service (`$01B4`/`$01D7`,
state machine in `$D241`) rewrites the **16 background CRAM colours** straight from a fixed
table in bank 0 at **`$0216`**, cycle-precise with unrolled `OUT ($BE)` writes during active
display. At vblank CRAM is restored (`$019F`, from `$0216`/`$0256`) to the surface palette for
the next frame's top. The net effect on screen:

- **Above the water line:** the normal Labyrinth surface palette (`romPalette` index **3**)
  *with* the water/green-ooze colour **cycle** running (Part IV §4).
- **Below the water line:** the static underwater palette at `$0216`, **no cycle** —
  blue/cyan shifted (`#002244 #004477 #007777 #00aa99 #4477bb #0011ee #33eebb #0055ff #004488
  #001199 #bbffaa #66bb99 #007733 #bbff77 #002255 #aaffff`). This is why submerged vines and
  walls read as **cyan** rather than the surface greens/purples.

(An earlier guess that the underwater palette was `palTable` index 8 was wrong; the real
colours are this hand-written bank-0 table, found by following the line interrupt.)

### The physics flag, and what it changes

The Sonic handler (`$4AD0`) carries a per-zone *"this is a water level"* bit (`IY+6` bit 7,
set by the loader). When it is set, each frame it calls **`$535C`**, which compares Sonic's
world-Y (`$D402`) with the water surface (`$D2DD`):

```
$535C  HL=($D2DD); DE=($D402); SBC HL,DE
       JP C,$5845        ; Sonic below the surface → underwater
       ($D296)=0; RES 4,(IX+24); RET   ; above → clear timer + flag
$5845  BIT 4,(IX+24); JR NZ,…          ; already wet?
       LD A,$12; RST $28               ; entering → SPLASH sub-action ($12)
       SET 4,(IX+24); RET              ; set Sonic's underwater flag
```

So **`IX+24` bit 4 is the underwater flag**. In the handler's physics-block selector
(`$4B30`: `BIT 4,(IX+24); JP Z,$4B57`), the underwater branch (`$4B37`) loads a *different,
slower* constant set into the movement variables. Comparing the blocks (each copied to
`$D20F`, plus the explicit `$D23A/$D23C/$D23E`):

| Variable | Normal (ground) | Underwater (`$4FE9` / `$4B37`) | Effect |
|----------|-----------------|--------------------------------|--------|
| `$D20F` acceleration | `$0010` | `$0004` | **¼** accel |
| `$D20F+2` deceleration | `$0030` | `$000C` | ~0.4× |
| `$D23A` top speed | `$0300`–`$0C00` | `$0100` | much slower |
| `$D23C` jump impulse (up) | `$FD00` (−768) | `$FDC0` (−576) | weaker jump |
| `$D23E` gravity | `$0038` (56) | `$0010` (16) | **~0.45×** gravity (slow sink) |

Gravity is the constant added to vertical velocity at `$4DCF` (`DE=($D23E)`); the lower value
is what makes Sonic drift down slowly underwater. So both the user-observed "lower gravity and
acceleration" fall straight out of the one flag.

### Drowning

The flag also gates an underwater-only sub-handler, **`$526C`** (run via `$4B0D`: `BIT
4,(IX+24); CALL NZ,$526C`). It is fenced to Labyrinth and skips the Act-3 boss
(`CP $03`/`CP $0B`), then runs an **air timer `$D296`**: it counts up while submerged and,
past `$0300` (768 frames ≈ 12.8 s), fires the warning/countdown effect (`RST $28` idx `$1A`)
and eventually drowns Sonic. (So the 8-bit game *does* have drowning, contrary to a common
belief that only the 16-bit version does.)

## 4. Bonus stages (the special stage)

Clearing Act 1 or 2 of a zone **with 50 or more rings** sends you to a bonus stage instead
of the next act. The trigger is in the **goal-sign handler `$61F8`**: when you reach the
sign it saves the ring count (`$D2AA`) and, at the act-complete branch, checks
`LD A,($D2AA) / CP $50` — 50 in BCD — and on `>=` selects completion type `$D282 = 4`. The
`$D282` state machine entry for 4 (`$17E3`) **sets the bonus flag `(IY+7) bit 0`**.

The next time a scene loads, the attract/level loop (`$178A`) sees that flag and, instead of
the normal `INC ($D238)`, runs the bonus path: it swaps in **`$D239`** as the scene index,
plays it, then restores the real progression. `$D239` is the **bonus-stage cursor** —
initialised to **`$1C` (28)** at boot (`$1386`) and **incremented after each bonus**, so each
visit plays the next one in sequence.

So the bonus stages are simply **scene-table entries 28–35** — eight more descriptors in the
same `$5600` table, in a seventh "zone" (descriptor `+0 = 6`). They are reached through the
exact same loader as the normal acts (`scene_run $1414`: `descriptor = $5600 + word($5600 +
index*2)`).

A surprise about their data: **all eight share one tilemap, one tile set, one block table and
one palette** (map file `$1EE30`, tiles `$3C7F5`, blocks `$149B0`, BG palette index 7 — the
pink-checkered starfield). What differs per round is the **spawn**, the **camera bounds**
(descriptor `+9/+10` top, `+11/+12` bottom) and the **object layout** — the rings (`type
$21`), the collectible goal (`type $52`) and assorted items. The eight rounds cycle through
four spawn/bounds configurations (top corridor, middle staircase, bottom corridor, right
column) in two object variants each, so although the ROM stores one map, each round plays as
a distinct section of it.

The export adds all eight as a **"Special Stage"** zone (`extract/cmd/webexport`; rendered
straight from ROM with `cmd/bonusshot` for verification). Like every zone, its ring fields are
baked into the block map (`$79`–`$7B` → tiles `252`–`255`, empty until the runtime ring
animation fills them), so the export applies that animation here too (see §1, the
special-stage objects).

---

# Part VI — Sound

The Game Gear's audio is the **SN76489 PSG** — three square-wave tone channels and one
noise channel — programmed through a single write port, **`$7F`**. (There is no FM on the
Game Gear; the music is square waves.) Each tone channel has a 10-bit period (pitch) and a
4-bit attenuation (volume); the noise channel has a 3-bit control. The music driver
reprograms these registers **once per video frame** to play the melody.

## The sound driver

A sound is started by **`RST $18`** with a sound id in `A`; the gateway pages **bank 3** into
slot 1 and jumps to **`$46EB`**, which indexes the **song-pointer table at `$4716`** (a word
per id) and calls the loader **`$4018`**. A song begins with a header of **five relative
channel offsets**; the loader adds the song's base address and stores the five pointers in RAM
at **`$DC1C`–`$DC25`** — channels 0–2 are the **square tones**, channel 3 the **noise**, and
the fifth slot is the **sound-effect channel** (its offset is 0, so it points back at the
header and stays idle during music). The song bases: Green Hills `$47D0`, Bridge `$574A`,
Jungle `$524A`, Labyrinth `$760C`, Scrap Brain `$5B4F`, Sky Base `$61A7`, special `$64C3`.

The **per-frame sequencer `$423E`** loads each channel's pointer, runs the decoder **`$42F4`**
to advance it, and renders it with **`$43DE`**.

## The channel data format

Each channel is a byte stream. A duration counter (`IX+2/3`) is decremented by a global tick
each frame; when it runs out the decoder reads the next bytes:

- **`$00`–`$6F` — a note**, encoded **`(octave << 4) | note`**. The pitch is the 10-bit PSG
  period **`freqtable[note] >> octave`** — the table at **`$44D5`** is one chromatic octave
  (`$0356`=C down to `$01C4`=B), and the octave field shifts it down. The **next byte is the
  duration** (a tick count; `0` ⇒ the default duration). So a note is *two* bytes.
- **`$7F` — a rest** (+ a duration byte): the volume mask is cleared, silencing the channel.
- **`$70`–`$7E` — a voice** (+ a duration byte): the low nibble picks an 8-byte instrument
  from the table at **`$43CE`** — **byte 0 is the noise mode**, bytes 1–6 the ADSR envelope.
  The **noise channel's "notes" are voice commands**: each `$70`/`$71` is a drum hit (a noise
  mode + a short percussive envelope), which is the zone's percussion.
- **`$80`–`$FF` — a command** (no time of its own), dispatched through `$4529`:
  `$80` set tempo — a note-length multiplier (the duration counter is `durByte × multiplier`,
  decremented by a per-frame tick) written **both per-channel and to a global**; only the
  control channel carries `$80`, so the other channels inherit the global multiplier (getting
  this wrong runs the non-control voices several times too fast). `$81` volume, `$82` instrument
  envelope (6 bytes, inline), `$83` vibrato (5 bytes), `$84` detune, `$85` skip,
  **`$86`/`$87` repeat a block** (`$86` pushes a counter, `$87 <n> <addr>` loops back to
  `addr+base` *n* times — a nested loop stack), **`$88` mark loop point** (saves the pointer
  in `IX+34/35`), `$89` noise mode, `$8A` default duration, `$8B`/`$8C` volume up/down, `$8D`
  tie (don't retrigger the envelope), **`$FE` end**, and **`$FF` loop** (jump the pointer back
  to the `$88` mark). The loop is therefore **in the data** — `$88` … `$FF` bracket the
  repeating section.

Per frame the render (`$43DE`) forms the period as **`(freqtable[note] + detune + vibrato) >>
octave`** — note the octave shift is applied *after* adding the vibrato (`$445A` then
`$4468`), so the vibrato depth is divided down with the pitch. `$83`'s vibrato is a triangle
LFO that, after a start delay, steps the offset up/down: at each direction flip it reloads the
full depth and **skips the step that frame** (keeping the triangle symmetric — getting this
wrong makes held notes drift out of tune). An **ADSR envelope** (the `$82` params: attack,
decay, sustain, decay2, sustain2, release) scales the volume; the result is written to the PSG
as the period (two writes) and `15 − volume` attenuation.

## Synthesising from the ROM

**`extract/cmd/musicrom`** is a Go port of this driver: it parses a song's channels straight
from the ROM, simulates the sequencer frame-by-frame (notes, durations, the `$86/$87` repeat
stack, ADSR, vibrato, and the `$88`/`$FF` loop), and emits the PSG state, which it synthesises
(phase-continuous squares + a 16-bit LFSR for noise) and pipes through `ffmpeg` (`libmp3lame`)
to MP3s in `site/public/sonic/music/`. The **loop comes from the data**: the clip is trimmed
to one loop of the longest (melody) channel, measured by running the port — no pattern-matching
on emulator output. Verification against the oracle's PSG (`soundprobe`) is exact: the loop
lengths match the hardware *to the frame* (Green Hills 38.4 s, Bridge 25.6 s, Jungle/Labyrinth
28.8 s, Scrap Brain 51.2 s, Sky Base 16.0 s, the special stage 59.7 s).

**Which track plays** is the act's **music id** = descriptor byte **+36** (Part IV §4), which
indexes the `$4716` table. The 21 ids resolve to ~15 distinct tunes: the six **zone themes**
(ids 0–5; Sky Base acts 1 & 3 actually reuse Scrap Brain's id 4, only act 2 uses id 5), the
**special stage** (id 16 — *not* id 6, which is the **title** theme), the **world map** (7),
**act-clear** (9), **boss** (11, shared by every boss handler), and assorted jingles. The
exporter writes each act's track name into its JSON and the viewer plays it per act, so e.g.
Sky Base Act 1 correctly gets the Scrap Brain music.

(The earlier `extract/cmd/soundprobe`/`musicbake` capture the PSG from the running oracle —
kept only as the verification reference; the shipped music is `musicrom`'s, from the ROM.)

*Open ends:* a few of the shorter ids (the 1-up / game-over / continue jingles, an "ending"
tune) are baked but not yet matched to their exact in-game cue; the LFSR-noise + volume curve
are approximations of the real chip.

---

# Appendix A — Toolchain and reproduction

Static analysis only, with the Z80 toolchain in the shared `tools/` module:

- [`tools/z80`](../../tools/z80) — a Z80 decoder (`Decode`/`Disassemble`) built on
  the CPU's regular x/y/z/p/q opcode bit-fields, covering the `CB`/`ED`/`DD`/`FD`
  prefix pages, plus an **execution core** (`cpu.go`) for running real code.
- [`tools/gamegear`](../../tools/gamegear) — the Game Gear hardware: the VDP tile,
  palette and name-table decoders, and a minimal `Machine` (RAM + Sega mapper + VDP
  ports) that drives the Z80 core as an emulation oracle, with a per-region write
  counter and a **VRAM write watchpoint** (record the PC of whatever draws a given
  address range) used to attribute each part of a screen to its routine (Part IV §3).
- [`tools/cmd/disz80`](../../tools/cmd/disz80) — linear disassembler over a file slice
  mapped at a Z80 address: `disz80 -off FILEOFF -len N -base ADDR rom.gg`.
- [`tools/cmd/codetracez80`](../../tools/cmd/codetracez80) — recursive-descent
  disassembler from given entry points: `codetracez80 -load 0 -entry 0000,0038,0066 rom.gg`.
- [`extract/cmd/segascreen`](extract/cmd/segascreen) — runs the boot on the oracle and
  composes an exact opening screen from the live VRAM: `segascreen rom.gg out [frames]
  [name]` (the SEGA logo by default, the title screen at `650 title`; Part IV §3).
- [`extract/cmd/screentrace`](extract/cmd/screentrace) — fingerprints VRAM each frame
  to time the discrete screen loads and reports, via the watchpoint, which routines
  drew the name table (Part IV §3).
- [`extract/decomp`](extract/decomp) — standalone Go reimplementations of the two
  located codecs: the `$0406` tile decompressor (Part IV §2) and the `$0502` RLE
  name-table loader (`LoadRLE`, Part III §3).
- [`extract/cmd/titlegfx`](extract/cmd/titlegfx) — runs the `$0406` reimplementation on
  the logo tile block, cross-checking it against the game's own decompressor.
- [`extract/cmd/scenemap`](extract/cmd/scenemap) — statically decodes and renders the
  world map from the traced `$0C7A` loader (tiles + the `$0502` RLE map + palette
  `$0C`/`$0D`), with no emulation (Part III §3).

Reproduce the boot listing in Part I §5:

```sh
go run stupidcoder.com/tools/cmd/disz80 -off 0 -len 0x0C -base 0 \
    "Sonic (GG)/Sonic The Hedgehog (Japan, USA).gg"
```
