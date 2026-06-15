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
8-bit. Parts I–II are complete; the rest is stubbed.

---

## Contents

- [Part I — The cartridge image](#part-i--the-cartridge-image)
  - [1. The ROM dump](#1-the-rom-dump)
  - [2. The Z80 address space and bank switching](#2-the-z80-address-space-and-bank-switching)
  - [3. The memory map](#3-the-memory-map)
  - [4. The cartridge header (`TMR SEGA`)](#4-the-cartridge-header-tmr-sega)
  - [5. The CPU vectors](#5-the-cpu-vectors)
- [Part II — Boot and initialization](#part-ii--boot-and-initialization)
  - [1. Cold-start init (`$0296`)](#1-cold-start-init-0296)
  - [2. Cross-bank calls and the `RST` gateways](#2-cross-bank-calls-and-the-rst-gateways)
  - [3. The frame-interrupt handler (`$0073`)](#3-the-frame-interrupt-handler-0073)
  - [4. The main entry (`$1356`)](#4-the-main-entry-1356)
- [Part III — Engine architecture](#part-iii--engine-architecture)
- [Part IV — Graphics and data formats](#part-iv--graphics-and-data-formats)
- [Part V — Game mechanics](#part-v--game-mechanics)
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

*Stub.* The main loop, the interrupt-driven frame timing, the work-RAM layout, and
how banked level/graphics resources are addressed.

# Part IV — Graphics and data formats

*Stub.* The VDP Mode 4 encodings — 8×8 4-bitplane tiles, the 32×28 name table, the
12-bit CRAM palettes, the sprite attribute table — and Sonic's level, object and
(if any) compressed asset formats.

# Part V — Game mechanics

*Stub.* Sonic's movement and physics, the object/enemy system, the zones and act
structure, rings, scoring and progression.

---

# Appendix A — Toolchain and reproduction

Static analysis only, with the Z80 toolchain in the shared `tools/` module:

- [`tools/z80`](tools/z80) — a Z80 decoder (`Decode`/`Disassemble`) built on the
  CPU's regular x/y/z/p/q opcode bit-fields, covering the `CB`/`ED`/`DD`/`FD`
  prefix pages.
- [`tools/cmd/disz80`](tools/cmd/disz80) — linear disassembler over a file slice
  mapped at a Z80 address: `disz80 -off FILEOFF -len N -base ADDR rom.gg`.
- [`tools/cmd/codetracez80`](tools/cmd/codetracez80) — recursive-descent
  disassembler from given entry points: `codetracez80 -load 0 -entry 0000,0038,0066 rom.gg`.

Reproduce the boot listing in §5:

```sh
go run stupidcoder.com/tools/cmd/disz80 -off 0 -len 0x0C -base 0 \
    "Sonic (GG)/Sonic The Hedgehog (Japan, USA).gg"
```
