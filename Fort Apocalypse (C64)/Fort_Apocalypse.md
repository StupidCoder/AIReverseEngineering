# Fort Apocalypse (C64) — tape format, loader, and game analysis

A complete reverse-engineering reference for `Fort_Apocalypse.tap`
("FORT APOCALYPSE — BY STEVE HALES — COMMODORE VERSION BY JOE VIERRA —
COPYRIGHT SYNSOFT", U.S. Gold tape release). It covers everything
learned in this analysis session, in reading order:

* **Part I** — the TAP container and both tape encodings (standard
  KERNAL and the custom fastloader), enough to extract every byte
  from the raw image;
* **Part II** — the boot chain from `LOAD` to the game's first
  instruction, including the loading screen and the copy-protection
  tricks;
* **Part III** — the game program: initialization, interrupt
  architecture, memory map;
* **Part IV** — all graphics data: character sets, sprites, the level
  maps and their compression, the scanner (with the extracted images
  inline);
* **Part V** — game mechanics: every object type, its movement,
  collision and spawn behavior, plus difficulty and progression —
  enough to reimplement the game.
* **Appendices** — toolchain and reproduction commands, all text
  strings and easter eggs, key routine/table reference.

Methods: a Go extraction toolchain (`tapextract/`), a table-driven
6502 disassembler (`tapextract/cmd/disprg`), a graphics renderer
(`tapextract/cmd/gfxrender`), and an instrumented Python 6502 emulator
(`emu.py`) that executed the real init/title/game path and logged all
reads/writes to confirm the static analysis. All addresses are C64
memory addresses; "frame" means one PAL frame.

---

# Part I — The tape image

## 1. TAP container

A TAP file records the signal a C64 sees on the cassette read line as
a sequence of pulse lengths, one byte each.

```
offset  size  content
0       12    magic "C64-TAPE-RAW"
12      1     version (this image: $01)
13      3     reserved
16      4     data length, little-endian
20      ...   pulse data
```

Header of this image:

```
43 36 34 2D 54 41 50 45 2D 52 41 57   "C64-TAPE-RAW"
01                                    version 1
00 00 00                              reserved
05 72 03 00                           data length $00037205 = 225,797
```

Each data byte `n > 0` is one pulse (falling edge to falling edge) of
`n * 8` clock cycles (PAL: 985,248 Hz). A `$00` byte is a pause
marker:

* version 0: pulse longer than $FF*8 cycles (no length given);
* version 1 (this image): followed by a 24-bit little-endian pause
  length **in cycles**, e.g. `00 00 10 00` = 4,096 cycles.

### Layout of this image

| TAP offset      | content                                          |
|-----------------|--------------------------------------------------|
| 20 – 35,396     | KERNAL: header block "FORT" + repeat copy        |
| 35,397          | pause `00 00 10 00` (4,096 cycles)               |
| 35,401 – 48,336 | KERNAL: data block (BASIC stub) + repeat copy    |
| 48,337          | pause `00 FC 23 12` (1,188,860 cycles ≈ 1.2 s)   |
| 48,341 – 225,812| fastloader stream, 177,472 pulses                |
| 225,813         | pause `00 15 74 70` (7,369,749 cycles ≈ 7.5 s)   |

Only four distinct pulse widths carry data:

| byte | cycles | used by                |
|------|--------|------------------------|
| $25  | 296    | fastloader, bit 0      |
| $2F  | 376    | KERNAL, short pulse    |
| $42  | 528    | KERNAL, medium pulse   |
| $54  | 672    | fastloader, bit 1      |
| $57  | 696    | KERNAL, long pulse     |

## 2. Standard KERNAL encoding (bootstrap part)

The ROM loader encodes each bit as a *pair* of pulses:

```
bit 0        = short  + medium     (2F 42)
bit 1        = medium + short      (42 2F)
byte marker  = long   + medium     (57 42)
end-of-data  = long   + short      (57 2F)
```

A byte frame is the marker, 8 data bits LSB first, and an odd-parity
bit. Example from this tape — the byte `$01` (header type):

```
57 42   byte marker
42 2F   bit0 = 1
2F 42   bit1 = 0
2F 42   bit2 = 0      value = $01 (LSB first)
2F 42   bit3 = 0
2F 42   bit4 = 0
2F 42   bit5 = 0
2F 42   bit6 = 0
2F 42   bit7 = 0
2F 42   parity = 0    (odd parity: 1 ^ XOR of data bits)
```

Each record is: a pilot of short pulses, a 9-byte countdown sync —
`89 88 87 86 85 84 83 82 81` for the first copy, `09 08 ... 01` for
the repeat copy — the payload, and one XOR checksum byte. Every record
is recorded twice.

### Records on this tape

**Header block** (192 bytes payload + checksum):

```
01          type 1 = relocatable (BASIC) program
49 CC       "start" $CC49      (ignored: type 1 relocates to $0801)
F8 CC       "end"   $CCF8      (end-start = 175 = real program length)
46 4F 52 54 20 ... 20          name "FORT" padded to 16 chars
48 98 48 AD 05 DC ...          171 extra bytes: loader IRQ handler code!
```

The KERNAL loads the whole header into the cassette buffer at $033C,
so the "unused" 171 bytes after the filename plant code at
**$0351–$03F5** for free. That code is the fastloader's IRQ handler
(Part II).

**Data block** (175 bytes + checksum), loaded to $0801:

```
0B 08       BASIC link pointer
00 00       line number 0
9E 32 30 36 31 00            SYS 2061
00 00
A9 93 20 D2 FF ...           machine code at $080D (loader setup)
```

## 3. The fastloader encoding

The custom loader (it identifies itself on screen as **NOVALOAD**,
serial "D100701") reads **one bit per pulse**, measured with CIA1
timer A latched to $03F4 and force-reloaded on every cassette FLAG
edge. The IRQ handler reads the timer high byte:

* pulse ≤ ~500 cycles → bit 0      (on tape: $25 = 296 cycles)
* pulse > ~500 cycles → bit 1      (on tape: $54 = 672 cycles)

(The exact threshold is $03F4 − $0200 = 500 cycles plus IRQ latency;
the discrimination is the `EOR #$02 / LSR / LSR` trick on the timer
high byte.)

Bits are shifted into a register with `ROR`, so the **first pulse of
a byte carries the least significant bit**. The register is
pre-loaded with $7F; the 0 that falls out of bit 0 after eight shifts
marks byte completion. During pilot search the register is *not*
reset on a mismatch, so a run of ≥8 zero bits followed by a single
one bit reads as the pilot byte $80 — that's how the decoder
self-synchronises.

### Stream layout

```
pilot      2,055 short pulses (0-bits), then one long pulse (1-bit)
sync       $AA
key        $55      (checked against the checksum seed, initialised $55)
records    [page#] [256 data bytes] [checksum]   repeated 84 times
end        $00 page number (only valid after the $F0 record)
trailer    252 bytes of $00 (filler, ignored)
```

* each record's 256 bytes load at `page# << 8` — pages may arrive in
  **any order** and gaps are fine;
* checksum = `(page# + sum of the 256 data bytes) mod 256`
  (`CLC ADC` per byte, carry dropped);
* the record with page# == **$F0** arms "end mode"; after it, a page#
  byte of **$00** terminates the load (other page numbers continue
  loading, and a 3-digit on-screen counter is decremented per page).

Example from this tape (TAP offset 48,341):

```
... 25 25 25 25 25 25 25 25      pilot: 0-bits
    54                           pilot terminator: 1-bit  (reads as $80)
    25 54 25 54 25 54 25 54      $AA   (LSB first: 0,1,0,1,0,1,0,1)
    54 25 54 25 54 25 54 25      $55   (1,0,1,0,1,0,1,0)
    25 25 25 25 25 54 54 54      $E0   first page number
    25 25 54 54 25 25 54 25      $4C   first data byte: JMP ...
```

Pages on this tape, in tape order:

```
E0 E1 E2 E3 E4 E5 E6 EF F0 EE F1        stage-2 loading screen
                                        ($E000-$E6FF, $EE00-$F1FF;
                                        $F0 arms end mode and releases
                                        the main thread's JSR $E000)
70 71 72 ... B8                         main game ($7000-$B8FF),
                                        loaded while stage 2 runs
00                                      end marker
```

All 84 record checksums on this image verify. Extracted program
files: `FORT.prg` ($0801, stub+loader), `FORT-fast-7000.prg`
($7000–$B8FF, the game), `FORT-fast-E000.prg`, `FORT-fast-EE00.prg`
(loading screen).

---

# Part II — Boot chain and loader internals

## 1. Overview

```
LOAD"",1 + RUN
SYS 2061 ($080D)                    KERNAL-loaded stub
  └─ banks ROMs out, vectors IRQ to $0351 (the code hidden in the
     tape *header*), arms a CIA FLAG interrupt per tape pulse,
     busy-waits at $03E5 until page $F0 arrives
     └─ JSR $E000                   stage 2: loading screen
          ├─ paints the U.S. Gold screen from a display script ($E364)
          ├─ runs scroller + 3-voice music while the game loads
          ├─ the music stream secretly patches $03F5: RTS -> JMP $8600
          ├─ on load error: wipe all RAM, JMP ($FFFC)   [reset]
          └─ on success: fade music, RTS
             └─ $03EE: ROMs back in, JSR $FF84 (IOINIT)
                └─ $03F5: JMP $8600   (was RTS until patched!)
                   └─ $8600: JMP $8927   game initialisation (Part III)
```

## 2. Loader setup ($080D, run by SYS 2061)

1. Clears the screen, copies the filename and `NOVALOAD D100701`
   (screen codes at $08A0) onto it.
2. `SEI`, `$01 = $05`: **banks out BASIC and KERNAL ROM**. The CPU IRQ
   vector is now the RAM vector $FFFE/$FFFF → pointed at $0351.
3. SID set up for the loading noise (each loaded byte is also written
   to $D401).
4. Zero-page loader state: `$FB/$FC` store pointer (low byte always 0,
   pages are aligned), `$FD` offset in page, `$AA = $55` checksum
   seed, `$AB` status ( $0F loading / $00 done / $80 error).
5. CIA1 timer A latch = $03F4; `$DC0D = $90` enables the FLAG (tape
   read line) interrupt only.
6. BASIC text pointer $7A/$7B → $03F6, which holds `3A 8A` (":" +
   `RUN` token) — a decoy, see below.
7. Busy-waits until `$FC == $F0` (the $F0 page has started loading),
   then `JSR $E000` while the IRQ keeps loading in the background.

## 3. The IRQ handler in the tape buffer ($0351)

Runs once per tape pulse. After the timer trick demodulates the bit
and `ROR $A9` assembles bytes (Part I §3), a **self-modifying branch
offset at $0365** dispatches a state machine:

```
PILOT  wait for byte $80 (no register reset on mismatch)
SYNC   expect $AA, else back to PILOT
CKSUM  byte must equal running checksum, else $AB=$80 and stop
PAGE   byte = page number; $F0 arms end mode
PAGE'  (end mode) $00 = load complete ($AB=0, FLAG IRQ off)
DATA   store 256 bytes at page*256+Y, checksum += byte
```

## 4. Stage 2 — the loading screen ($E000–$E6FF, $EE00–$F1FF)

| range        | content                                              |
|--------------|------------------------------------------------------|
| $E000–$E002  | `JMP $E008`                                          |
| $E003–$E007  | variables: 3 voice waveforms, music tempo, scroll speed |
| $E008–$E2AC  | code: screen painter, main loop, music player        |
| $E2AD–$E33F  | note frequency table                                 |
| $E340–$E363  | state variables                                      |
| $E364–$E6FF  | display script (the loading screen)                  |
| $EE6A–$EF7F  | scrolltext, stored **reversed**, 278 chars           |
| $EF80–$F1FF  | music command stream                                 |

The display script paints the U.S. Gold frame (`ALL AMERICAN
SOFTWARE`, `BLOCKS TO LOAD 075` — the three digits at $07D8–$07DA are
the cells the tape IRQ decrements per page). Script format: a
2-byte counter pointer, border+background colours, then runs of
screen codes with `$60` as escape (`60 0D` newline, `60 0n` colour n,
`60 04` end).

The scroller reads its text *backwards* from $EF7F with a
self-modified decrementing operand (byte `$60` wraps it):

> U.S.GOLD - THE ULTIMATE IN AMERICAN SOFTWARE - FROM THE BEST
> SOFTWARE PRODUCERS IN THE STATES, U.S. GOLD SELECTS THE MOST
> EXCITING TOP QUALITY PRODUCTS TO BRING YOU REAL ENTERTAINMENT
> VALUE. LOOK FOR THE U.S.GOLD SEAL. THESE ARE THE TITLES FOR YOU
> AND YOUR 64

## 5. The music stream is a program — and hides the game start

The music player's command stream ($EF80) has three command types:

```
nn dd        nn < $FE: play note nn for duration dd
FF lo hi     set stream read pointer (loop)
FE lo hi n   copy the next n stream bytes to address hi/lo
b1..bn
```

`$FE` is implemented by patching the operand of a `STA $nnnn,X`, so
the music data can **write anywhere in memory**. The first four
commands of the tune, executed on its first tick — long before
loading finishes:

```
EF80: FE 18 03 01  C1             ; $0318 <- $C1
EF85: FE 03 E0 05  41 41 41 0D 02 ; player variables re-init
EF8E: FE 00 D4 19  00 00 00 04 40 09 D0 (x3) 00 00 00 0F ; SID init
EFAB: FE F5 03 03  4C 00 86       ; $03F5 <- JMP $8600   *** game start
EFB2: ...notes...
F1C6: FF B2 EF                    ; loop the tune (patches run once)
```

* **`$0318 ← $C1`** redirects the KERNAL NMI indirection vector from
  $FE47 to $FEC1 — the `RTI` at the end of the ROM NMI exit sequence
  (`PLA TAY PLA TAX PLA RTI` at $FEBC). RUN/STOP–RESTORE becomes a
  clean no-op in the game.
* **`$03F5 ← 4C 00 86`** rewrites the loader epilogue: as loaded it
  ends `JSR $FF84 / RTS` followed by planted `:RUN` bytes — a decoy
  that static analysis (or a memory snapshot taken before the music
  played) sees as a harmless return to BASIC. The real entry address
  exists nowhere in code; it is data inside the tune.

## 6. End of loading, and the error path

On success ($AB=0) stage 2 fades the volume ~3 s, clears the SID, and
returns: ROMs in, IOINIT, then the patched `JMP $8600`. On a tape
error the loader IRQ freezes; stage 2 detects the stalled byte
counter and responds at $E220 by **wiping all RAM except its own page
$E2 and jumping through the reset vector** — anti-tamper as much as
error handling.

---

# Part III — Game program architecture

## 1. Initialization ($8600 → $8927)

$8600 holds `JMP $8927` followed by data tables. The init:

1. `SEI CLD`, clear zero page, `$01 = $2E` — **BASIC ROM out, KERNAL
   in** ($A000–$B8FF of the file is game code, called directly).
2. VIC bank 1 ($4000–$7FFF) via CIA2; screen $4400; `JSR $B11A` (SID
   reset; **voice 3 noise = the game's RNG, read at $D41B**).
3. `JSR $AFDB`: zero $0380–$6FFF.
4. Build both charsets and expand all sprites (Part IV).
5. Draw the HUD frame and title texts (double-width font renderer
   $9B56: each glyph drawn as char `n` plus char `n+$20`).
6. Install the title IRQ at raster line $F9 ($B0FF sets $0314/$0315 —
   the KERNAL dispatches IRQs through it) and end with
   **`$8A9F: JMP $8A9F`** — everything from here on is IRQ-driven.

## 2. Interrupt architecture

Two raster splits per frame:

```
line $F9 -> $9BD4: HUD part. $D018=$14 (charset $5000), scroll regs,
            read collision latches $D01E/$D01F into $68/$73,
            INC $15 (frame counter), keyboard/joystick, player sprite
            + bullets + enemy sprite updates, sound; next IRQ line $76
line $76 -> $AE19: playfield part. $D018=$16 (charset $5800),
            fine-scroll $D016/$D011, per-level colours $D022/$D023,
            charset animations, playfield window copy, SID effects;
            next IRQ line $F9
```

So screen rows 0–6 (HUD + scanner) and rows 7–24 (playfield) use
*different character sets*, switched mid-frame.

The **main game loop** at $8BB1 (entered from the title IRQ by a
stack-reset `JMP` when fire is pressed) waits for the frame counter
to change, then calls the per-frame game-logic chain (object engines,
zone checks, state dispatch) and loops. Game state lives in `$9D`:
1 title/attract, 9 demo-game, 3 new game, 4 get ready ("GET READY
PILOT", "PILOTS LEFT"), 5 life lost, 2 playing, 6 game over/debrief,
7 transition lock, $0A cavern teleport.

## 3. Memory map (during play)

| range        | content                                                  |
|--------------|----------------------------------------------------------|
| $0002–$00FF  | zero page: game state ($9D), frame ($15), camera ($56–$59), player ($64–$6A), pointers |
| $0100–$01FF  | stack                                                    |
| $0314/$0315  | IRQ vector: $9BD4 / $AE19 / $8AA2 (title), swapped per split |
| $0503–$2D02  | **decompressed level map**, 40 rows of 1 page each (215 content columns + empty pad + wrap-seam column 255) |
| $2E00–$343F  | **scanner soft bitmap**, 40×5 chars (320×40 px)          |
| $3500–$354D  | char-object X / row coordinate tables (39 slots)         |
| $3600–$361F  | prisoner tables (state, X, row, direction/phase)         |
| $3700–$374D  | patrol-craft state machine bytes                         |
| $4000–$43FF  | VIC sprite blocks; **blocks 1–14 = helicopter shapes**   |
| $4400–$47E7  | screen: rows 0–5 HUD image, row 6 frame/messages, rows 9–24 = map window |
| $47F8–$47FF  | sprite pointers (0 player, 1 enemy, 2–4 bullets)         |
| $4800–$487F  | sprite blocks $20/$21: bullet dots                       |
| $5000–$57FF  | HUD charset; $52E0–$53FF = scanner window soft chars     |
| $5800–$5FFF  | playfield charset; chars $00–$20 animated in place       |
| $7000–$B8FF  | the game file (below)                                    |
| $E000–$F1FF  | dead loader stage-2 remnants (never referenced)          |

### Game file layout ($7000–$B8FF)

| range        | content                                                  |
|--------------|----------------------------------------------------------|
| $7000–$762A  | level 0 map, RLE                                         |
| $762B–$7D36  | level 1 map, RLE                                         |
| $7D37–$7FFF  | unreferenced slack                                       |
| $8000–$81E8  | level 0 scanner bitmap, RLE (mode 1)                     |
| $81E9–$84EE  | level 1 scanner bitmap, RLE (mode 1)                     |
| $84EF–$8602  | unreferenced slack                                       |
| $8603–$86F2  | HUD screen image (rows 0–5, 240 bytes, literal)          |
| $86F3–$870E  | sprite shape pointer table (14 words)                    |
| $870F–$8906  | 14 packed sprite shapes (36 bytes each)                  |
| $8907–$8926  | energy-barrier char patterns                             |
| $8927–$B297  | code + small tables (run tables $8D2B/$8D42, level sources $8D46/$9116, solid-tile $A45D, heli anim $A320, masks $ADB5/$B0F7, difficulty $8DFC–$8E0F) |
| $B298–$B560  | HUD charset data (raw)                                   |
| $B561–$B8FF  | playfield charset data (raw)                             |

---

# Part IV — Graphics and data formats

## 1. Compression: table-selective RLE

One decompressor at **$8CDB** serves all level data:

```
read byte B
if B is in the run-table: count = next byte (0 means 256)
                          emit B & $7F, count times
else:                     emit B & $7F once (literal)
until the destination reaches its end address
```

Two run-tables select which byte values may be run-length encoded:

```
mode 0 (terrain), table $8D2B, 23 entries:
  00 61 0E 0F 10 11 0A 0B 0C 0D 03 07 1F 73 74 41 44 48 58 59 5A D8 C7
mode 1 (scanner bitmap), table $8D42:  00 55 AA FF
```

Any other byte is its own literal — no escape codes, at the cost of
fixing which values can repeat. Example from $7000:

```
1F D7  00 28  1F 01  00 00  00 00 ...
= 215×$1F | 40×$00 | 1×$1F | 256×$00 | ...   (the map's first rows:
  215 columns of rock border, 40 bytes of row padding, the wrap-seam
  copy of column 0 — see §4 — then an empty sky row)
```

There is **no encryption** anywhere in the game data — the only
transformations are this RLE, the sprite column packing (§3), and the
`AND #$7F` masking on decompressed bytes (map codes stay below $80).

## 2. Character sets

### HUD charset, $5000 (screen rows 0–6, $D018=$14)

Built at init ($899C) from **uncompressed** data: one continuous
stream copied in overlapping 256-byte strips, net effect
**$B298–$B588 → $500F–$52FF** (note the odd $0F start offset; the
rest of the charset stays zero). Contains the score font and HUD
furniture.
Chars $5C–$7F ($52E0–$53FF) are **soft characters**: the radar window
is rendered into them at runtime (§5).

![HUD charset](rendered/charset-hud.png)

### Playfield charset, $5800 (screen rows 7–24, $D018=$16)

Copied the same way, net effect **$B561–$B858 → $5908–$5BFF** (chars
$21–$7F: all terrain glyphs, 8×8 multicolor dither patterns — chars
$55/$56 are the mountain slopes, $57 flat dither, $58 solid block).
Chars $00–$20 are left for
**soft chars animated in place** by the playfield IRQ:

| chars       | routine | animation                                      |
|-------------|---------|------------------------------------------------|
| $01–$04,$09 | $A7ED   | energy barrier 1: pattern from $8907/$891F or blanked (timer $3D) |
| $05–$08,$09 | $A830   | energy barrier 2 (timer $3E, cap pattern $8917) |
| $0A–$0D     | $A86B   | every 128 frames, random chars get rows of $55 (shimmer) |
| $0E–$11     | $A8B8   | 4-phase rotation: one of four chars lit per phase |
| $20,$3F     | $A8F3   | every frame: `pattern($A927) AND $D41B` — noise flicker: **$20 is the explosion char, $3F the fort core** |

![Playfield charset](rendered/charset-playfield.png)

(Both charset sheets are rendered in multicolor interpretation:
00=black, 01=$D022, 10=$D023=white, 11=colour-RAM green. The letter
glyphs $21–$3A/$41–$5A double as the double-width HUD font *and* as
object graphics: $3B–$3E/$49–$4A are the prisoner, $40/$5B–$5F the
patrol craft, $6C–$72 tank and missiles.)

## 3. Sprites — packed column format

The 14 shapes live at **$870F–$8906**: **36 bytes per shape = two
18-byte pixel columns** (left rows 0–17, then right rows 0–17),
located by the pointer table at $86F3. Init code $B044 expands each
into a 64-byte VIC block at $4040+ (blocks 1–14): per row
`[left][right][$00]`. Sprites are hires (no sprite multicolor);
sprites 0/1 are X-expanded ($D01D=$03), colours: player $D027=7
(yellow), enemy $D028 from $9C4F per level, bullets white.

**Both helicopters share one animation table** at $A320 (18 entries,
indexed by bank/tilt 0–$11):

```
01 02 03 04 05 06 07 08 07 08 07 08 09 0A 0B 0C 0D 0E
```

= **7 banking poses × 2 rotor frames** (even/odd index = rotor
phase; the head-on level-flight pose 7/8 covers three tilt steps).
The player toggles the rotor bit every frame ($67 bit 0); the enemy
every 4 frames ($A313). Animation sheet, one row per pose
(full-left → level → full-right), the two rotor frames side by side:

![Helicopter animation sheet](rendered/sprite-anim-helicopter.png)

The two bullet sprites are built at runtime ($B0B0) from a 9-byte dot
pattern at $B0C2 into blocks $20 (pattern twice: angled shots) and
$21 (once: straight-down shots):

![Bullet sprites](rendered/sprites-bullets.png)

## 4. The level maps

Per level (index $5C), $8F8F decompresses the terrain from the source
table at $8D46 — level 0 ("VAULTS OF DRACONIS"): **$7000**, level 1
("CRYSTALLINE CAVES"): **$762B**, then repeats — into the buffer at
**$0503**, one 256-byte page per map row, 40 rows.

**Map bytes are screen character codes** — no tile indirection. Two
placeholder codes are post-processed after decompression ($8FC2):
every `$73` becomes a random char $62–$64 and every `$74` a random
$65–$67 ($D41B noise) — the mottled cave-rock texture.

### Actual width: 215 columns + wrap seam

The 256-byte rows are wider than the playfield. Verified against the
decompressed data of both levels:

* columns **0–214** hold the level content (215 columns);
* columns **215–254** are always $00 — padding to a full page;
* column **255** is a near-duplicate of column 0 (34/40 identical
  rows on level 0, 33/40 on level 1) — the **wrap seam**.

The world is a horizontal cylinder: the camera column wraps between
$02 and $D9 ($A666 scrolling right, $A688 left), and the window
source is `$0502 + camCol` (left edge shows byte `camCol-1`). At the
wrap position camCol=$D9 the screen's right edge displays byte 255 —
the artist stored a copy of the leftmost column there so the world's
left edge appears at the right edge of the screen as the camera wraps.

Address arithmetic for game coordinates ($A7D8):

```
map address = $04FE + X + (row << 8)     ; i.e. $0503 + (X-5) + row*256
```

The rendered maps (wrap-seam column first, then columns 0–214; player
spawn framed cyan, enemy-gun spawn candidates yellow):

**Level 0 — Vaults of Draconis** (surface with FUEL depots and the
LAND HERE pad, cavern levels below):

![Level 0 map](rendered/map-level0.png)

**Level 1 — Crystalline Caves** (the Kralthan fortress: central
shaft, the fort core chamber, and a large destructible-rock field —
plus a hidden "PLEXAR WAS HERE" graffito):

![Level 1 map](rendered/map-level1.png)

### Scrolling: brute-force window copy ($A72C)

When the camera moves a full character (or every 8 frames, so
map-embedded objects animate), $A72C rewrites the source operands of
an unrolled copy loop and block-copies **16 rows × 40 columns**
straight from the map buffer to screen rows 9–24. Pixel-fine movement
between copies is hardware scroll ($D016/$D011). Because moving
objects write themselves *into the map buffer*, the periodic re-copy
doubles as their screen update.

## 5. The scanner (radar)

A second RLE stream (mode 1) decompresses per level from
$8000 / $81E9 (table $9116) into **$2E00: a 1600-byte soft bitmap** —
the whole map as a 320×40-pixel image (40 chars × 5 rows). The HUD
rows 0–5 are a prebuilt 240-byte screen image at $8603 whose scanner
window consists of soft chars; each frame $ADD3 copies a 12×3-char
window of the bitmap (following the camera, $AD2E) into those chars'
definitions. Blips are XOR-plotted ($ADAB/$A488) with the pixel-pair
mask table at $ADB5 (`80 80 20 20 08 08 02 02`): the player every
frame, the enemy helicopter and the tank bases blinking.

## 6. HUD

Score $91–$93 (6 BCD digits, row 0), bonus $97/$98 (counts down 1 per
8 frames during play; set to 9999 when the fort blows), fuel $9B/$9C
(4 BCD digits), "MEN TO RESCUE" count, and the message row (screen
row 6) for flashing texts ("LOW ON FUEL", "MEN TO RESCUE n"). Digits
are drawn with leading-zero blanking ($A9D3).

---

# Part V — Game mechanics and objects

## 0. The collision architecture

1. **"Solid" is defined by pixels, not tables.** The central test
   ($98F2 / $A031) takes the char under an actor and scans its 8
   charset bytes: any non-zero byte = collision. A blanked
   energy-barrier char is genuinely non-solid for everything at once.
2. **Char-based actors carry their own collision**: they draw
   themselves into the map buffer (saving the background) and react
   to the character codes they find (see the code table in Part IV
   §2 / the matrix below).
3. **Sprites use the VIC latches**, read once per frame: $D01E
   (sprite-sprite) → $68, $D01F (sprite-background) → $73.
4. **Bullets bridge the two worlds**: they fly as sprites but stamp
   the explosion char $20 into the map on impact; char actors die
   from touching it.

Engine scheduling per frame: patrol-craft engine 15 of 39 slots
($94D2), one prisoner slot ($AABA), tank and missile engines on
difficulty countdowns ($992A/$9644), zone/fuel checks — all from the
main loop; player control, bullets, enemy chopper, camera and HUD run
in the IRQ chain.

## 1. The player — Rocket Copter (sprite 0)

State: `$64` mode (2/3 flying, 4 crashing, 6 landed, $0B prisoner
boarding, 1 inactive), `$65/$66` sprite x/y, `$69/$6A` map col/row
(col=(x−$24)/4+camera, row=(y−$58)/8+camRow), `$67` bank 0–$10
(8=level, bit 0 = rotor), `$99` engine (8 ok, 9 out of fuel,
$0A refueling).

**Movement ($A4CE, every frame):**
- Left/right accelerate the bank: $67 ±2 every 4th frame; sprite
  moves 1px per 2 frames, every frame once banked ≥ ~75%. The bank
  value indexes the sprite-shape table — the copter visibly tilts.
- Up/down move 1px/frame; bank auto-centers every 8 frames when
  moving vertically or hovering ($A62F).
- Gravity: sinks 1px every 16 frames (novice) or 8 (expert) —
  mask $F5.
- The camera keeps the sprite between x $6E–$82 and scrolls
  2px/frame; the world wraps (Part IV §4).
- Title attract mode replays 108 recorded joystick values from $AA4F.

**Terrain contact ($A066):** on the sprite-background latch, first
offer boarding to prisoners (§7); then probe the 3 cells under the
copter:
- barrier-capped pad on level 0 ($B230 finds barrier chars $01/$02/
  $05/$06 within 5 rows above, columns $28–$D7) → state $0A:
  **teleport to one of 4 random cavern drop points** ($9892).
- otherwise: gentle 1px bounce and mode 6 (landed); the position is
  saved as the **respawn checkpoint** ($A176) unless on prisoner
  floor or refueling.

**Crash ($A128):** enemy-sprite or enemy-bullet contact (latch $68),
or empty-tank landing deep (row ≥ 14). Grace timers: $4D (just
teleported), $9C4E (~15 frames post spawn). Crashing: falls 1px/2
frames for 30 frames with colour flashing → state 5, life lost
($F9 BCD, 3/5/7 by variant), respawn at checkpoint; 0 lives → state 6.

**Fuel:** BCD $9B/$9C, starts 0100, −1 per 16 frames flying; at 0 the
engine sputters ("LOW ON FUEL"). Refuel landed at a FUEL depot
(rows 9–12): +4 per 8 frames, cap 2000, with a 5-stage draining
depot graphic ($ACF8).

## 2. Player bullets (sprites 2–3)

Three slots ($78/$7B/$7E); 0–1 player, 2 enemy. Fire is
edge-triggered, spawns at the nose with a small spread. **Trajectory
follows the bank angle** (tables $A472/$A477, per-step px): full-left
(−4,+2), left (−4,0), level (0,+2) **straight down**, right (+4,0),
full-right (+4,+2). On impact (pixel test):
- fort core $3F on level 1 → `$ED=5` → **fort destruction sequence**
  ($97B0: expanding $20-char explosion, 16 colour flashes, bonus 9999);
- destructible rock $47 → permanently cleared;
- object chars (table $A45D) → bullet dies *without* explosion —
  direct hits never hurt char actors;
- other solid chars → explosion char $20 in that cell for 7 frames,
  then restored. The $20 is what kills nearby char actors.
- dies at playfield edges. Only the enemy helicopter is killed
  directly (sprite-sprite latch).

## 3. Enemy helicopter (sprite 1)

One at a time: `$6E` (1 idle, 3 hunting, 4 dying), map pos $76/$77,
own bank $71. Spawns after a delay at one of 8 per-level patrol
points ($9CBA/$9CCA), only ≥ $22 columns or ≥ 8 rows away from the
player. Moves every 4th/2nd/every frame by variant (mask $72); it is
a copy of the player physics — banks, homes toward the player
column- and row-wise, and **probes the map with the same pixel test**
(it navigates terrain). Fires bullet slot 2 every 5th move tick with
the same bank-angle trajectories. Dies from player bullets
(sprite-sprite) or background contact → 20-frame explosion, respawn
delay. XOR-blinks on the scanner.

## 4. Tanks (6 per level, char-based)

Slots 0–5: `$BE,X` (1 dead, 2 active, 4 exploding, 7 respawn), pos
$C4/$CA, direction $D0, background save $D6+3X. Fixed homes per level
($911C+/$9128+). A tank is **3 cells $6C $6D $6E plus a turret char
one row up: $6F/$70 always aiming at the player**. Patrols
horizontally (step every 4/3/2 frames by variant), reversing on any
obstacle-table char. Dies (→ three $20 cells for 10 ticks) when a
cell it occupies/enters contains $20/$71/$72 — i.e. explosions and
missiles, not direct bullets. Respawns at home once no tank-body
chars remain within 13 cells. Blinks on the scanner.

## 5. Homing missiles (one per tank, chars $71/$72)

Launch when the player is within ±9 columns and 0–14 rows above the
tank; spawn above the tank with 20 fuel. Per step (every 3/2/1 frames
by variant): one cell horizontally in facing direction ($71 left /
$72 right), vertically steered toward the player's row; if the player
gets behind it, or it leaves columns $2D–$D7, or fuel runs out, it
stops homing and falls. Detonates on any solid char (pixel test of
the cell it enters) without damaging the map; dies on explosions. Its
char kills patrol craft, tanks and prisoners it passes through — it
can be lured into them.

## 6. Patrol craft / whirlybirds (13/26/39 by difficulty)

39 slots ($3500/$3527/$3700/$3727), 15 stepped per frame round-robin;
the armed count $FC is the difficulty dial. Spawn: random position
(columns $32–$CD) until two adjacent **empty** cells are found. A
2-cell craft that flies horizontally, animating through 4 phases per
cell (char pairs `$40 —`, `$5B $5C`, `$5D $5E`, `— $5F`); the 2
destination cells must be exactly $00 or it reverses. Dies (no
respawn until next level start) when overlapped by $20/$71/$72.
Bullets fizzle on it harmlessly — kill it with wall-splash explosions
or missiles.

## 7. Prisoners — "MEN TO RESCUE" (8 per level)

Placed at level build by scanning the map for **floor $48 with rock
$1F directly above**; up to 8 random pattern cells ($90A4). A
prisoner is 2 chars tall (torso $49/$4A over legs $3B/$3C right,
$3D/$3E left) and **runs back and forth along the $48 walkway**,
reversing when the next floor cell isn't $48. One slot per frame.
Rescue: terrain contact within ±3 cells → he boards (player held in
mode $0B), rescued count $EC++, "MEN TO RESCUE n" reprinted. Death:
his cell becomes $00 (floor shot away), $20, or $71/$72. Dead or
rescued, he leaves $EB — and **both level exits require $EB = 0**.

## 8. Collision matrix

| agent ↓ hits →   | terrain pixels | barrier (lit) | explosion $20 | missile $71/$72 | patrol craft | tank chars | prisoner | enemy heli (sprite) | player (sprite) |
|------------------|----------------|---------------|---------------|-----------------|--------------|------------|----------|--------------------|-----------------|
| player copter    | land/bounce or crash | crash (solid) | — | — | solid | solid | board (±3 cells) | crash | n/a |
| player bullet    | 7-frame $20 splash; $47 destroyed; $3F = fort | splash | dies | dies quietly | dies quietly | dies quietly | dies quietly | **kills** (sprite latch) | n/a |
| enemy bullet     | same splash mechanics | splash | dies | dies | dies | dies | dies | n/a | **crash** (latch) |
| homing missile   | detonates | detonates | dies | diverts down | **kills it** (overlap) | **kills it** (overlap) | **kills it** | — | solid to the player (pixels: background-latch contact) |
| patrol craft     | reverses | reverses | **dies** | **dies** | reverses | reverses | reverses | — | solid to player |
| tank             | reverses | reverses | **dies** | **dies** | reverses | reverses | reverses | — | solid to player |

(“dies quietly” = removed without an explosion — the deliberate rule
that makes direct shots harmless to char actors.)

## 9. Difficulty variants

Three options on the title/options screen (their labels are in the
string table at $9452+):

* **GRAVITY SKILL** — WEAK / NORMAL / STRONG ($F4)
* **PILOT SKILL** — NOVICE / PRO / EXPERT ($F6)
* **ROBO PILOTS** — THREE / FIVE / SEVEN ($F8, lives)

| table | variable | values (easy→hard) | effect |
|-------|----------|--------------------|--------|
| $8DFC | $F5 gravity mask | $0F,$07,$03 | sink every 16 / 8 / 4 frames |
| $8E01 | $F9 lives | 3, 5, 7 | pilots |
| $8E04 | $EF barrier rate | +4, +8, +16 | barrier blink speed |
| $8DFE | $72 enemy-heli mask | 3, 1, 0 | moves every 4th/2nd/every frame |
| $8E07 | $FC patrol craft | 13, 26, 39 | active whirlybirds |
| $8E0A | $F1 tank period | 4, 3, 2 frames | tank speed |
| $8E0B | $F3 missile period | 3, 2, 1 frames | missile speed |

## 10. Level progression and victory

```
Level 0 "VAULTS OF DRACONIS" (surface + caverns)
  rescue all 8 men -> land on the bottom-center pad (row>=35, cols
  $7E-$88) -> floor-shift animation ($8C7D): the copter sinks through
  the floor ->
Level 1 "CRYSTALLINE CAVES" (the fortress)
  rescue the men, shoot the fort core ($3F) -> fort explodes
  (bonus 9999) -> fly out the top-center opening (row<2, cols
  $7E-$86) ->
Level 2 = the surface again, harder; landing back on the base deck
  ends the mission -> debrief (state 6, $916E): rescued men, fort
  bonus and variant bonuses tally into a 0-15 pilot rank ($4F),
  high-score check.
```

### Pilot ranks

The debrief screen shows "MISSION ABORTED" (or "MISSION COMPLETED"
when the full loop was flown, $5C=3) and "YOUR RANK IS" followed by
the rank name. The 16 rank values combine four bird names (strings at
$92BC–$92D5) with a class number ($924A):

```
bird  = rank / 4      -> SPARROW, CONDOR, HAWK, EAGLE
class = 4 - (rank & 3) -> displayed as "<bird> CLASS <n>"
```

| rank $4F | name            | rank $4F | name           |
|----------|-----------------|----------|----------------|
| 0 (worst)| SPARROW CLASS 4 | 8        | HAWK CLASS 4   |
| 1        | SPARROW CLASS 3 | 9        | HAWK CLASS 3   |
| 2        | SPARROW CLASS 2 | 10       | HAWK CLASS 2   |
| 3        | SPARROW CLASS 1 | 11       | HAWK CLASS 1   |
| 4        | CONDOR CLASS 4  | 12       | EAGLE CLASS 4  |
| 5        | CONDOR CLASS 3  | 13       | EAGLE CLASS 3  |
| 6        | CONDOR CLASS 2  | 14       | EAGLE CLASS 2  |
| 7        | CONDOR CLASS 1  | 15 (best)| EAGLE CLASS 1  |

---

# Appendix A — Toolchain and reproduction

All tools live in the Go module `tapextract/` (plus `emu.py`):

```
# 1. Extract all program files from the raw tape image
cd tapextract && go build -o tapextract . && cd ..
tapextract/tapextract -o extracted -dis Fort_Apocalypse.tap
#   -> extracted/FORT.prg, FORT-fast-7000.prg, FORT-fast-E000.prg,
#      FORT-fast-EE00.prg (+ loader disassemblies with -dis)

# 2. Disassemble anything
cd tapextract && go run ./cmd/disprg -start 8927 -end 8A40 ../extracted/FORT-fast-7000.prg

# 3. Render charsets, level maps (with spawn markers) and sprites
cd tapextract && go run ./cmd/gfxrender -o ../rendered -markers -scale 2 ../extracted/FORT-fast-7000.prg

# 4. Dynamic verification: emulate init/title/game start, log all
#    data readers and video writers, dump memory to emu_mem.bin
python3 emu.py

# run everything's tests
cd tapextract && go test ./...
```

Package overview: `tap` (TAP container), `kernal` (ROM-loader
decoder), `fastload` (fastloader state machine), `mos6502`
(disassembler), `fortgfx` (RLE decoder, charset/sprite/map
extraction and PNG rendering), `cmd/disprg`, `cmd/gfxrender`.

# Appendix B — Strings and easter eggs

All text in the game binary uses the double-width font encoding
(letter = screen code − $20, $00 = space, $FF terminator; drawn by
$9B56 as char n + char n+$20). The full string table:

* `$8B30+` title: FORT APOCALYPSE / BY STEVE HALES / COMMODORE
  VERSION / BY JOE VIERRA / COPYRIGHT / SYNSOFT SOFTWARE
* `$8EBE+` GET READY PILOT / PILOTS LEFT
* `$9134+` ENTERING / VAULTS OF DRACONIS / CRYSTALLINE CAVES
* `$9285+` MISSION / ABORTED / COMPLETED / YOUR RANK IS / CLASS /
  SPARROW / CONDOR / HAWK / EAGLE / HIGH SCORE
* `$943C+` options screen: OPTIONS / OPTION / SELECT / GRAVITY SKILL
  / PILOT SKILL / ROBO PILOTS / WEAK / NORMAL / STRONG / NOVICE /
  PRO / EXPERT / THREE / FIVE / SEVEN
* `$AC1B+` MEN TO RESCUE / LOW ON FUEL

Text embedded in the *map data* (drawn with the second font half,
code = letter + $20): level 0 has the `FUEL` sign (row 13) and
`LAND HERE` at the bottom-center exit pad (rows 37–38, column 126 —
exactly the level-exit trigger columns); level 1 has its own `FUEL`
sign and the developer graffito **`PLEXAR WAS HERE`** (row 15,
columns 14–28, visible in the level-1 map render). The fuel-depot
gauge frames at $ACF8 also embed the FUEL label chars.

Two more curiosities:

* **$9FF0: hidden ASCII `N0:JOE VIERMON.Z`** — plain ASCII (not the
  game's text encoding, never displayed), sitting in padding right
  before the routine at $A000; its last two bytes overlap the routine
  entry and execute as a harmless `EOR $76,X`. It contains the
  porter's name and looks like a CBM DOS command/filename remnant —
  a development leftover or deliberate signature.
* **$AA4F** (108 bytes) reads like text (`NNNNKKKKKIIII...`) but is
  the **attract-mode joystick recording** — the byte values $46-$4E
  happen to be letter codes.
* The loader stage carries its own texts: `NOVALOAD D100701` (stub),
  `ALL AMERICAN SOFTWARE` / `BLOCKS TO LOAD` (loading screen), and
  the U.S. Gold advert scrolltext stored reversed at $EE6A–$EF7F.

# Appendix C — Key routines and tables

| addr  | routine                                            |
|-------|----------------------------------------------------|
| $0351 | tape-loader IRQ handler (lives in the tape buffer) |
| $080D | loader setup (SYS 2061)                            |
| $E000 | loading screen entry                               |
| $8927 | game entry / hardware init                         |
| $8B97 | game-session setup → falls into main loop          |
| $8BB1 | **main game loop**                                 |
| $8CDB | RLE decompressor                                   |
| $8F8F | level builder: decompress, randomize, populate     |
| $90A4 | prisoner spawn scan ($48/$48/$1F pattern)          |
| $94D2 | patrol-craft engine (39 slots)                     |
| $9644 | missile engine                                     |
| $992A | tank engine                                        |
| $98F2 | char pixel-collision test                          |
| $9B56 | double-width text renderer                         |
| $9BD4 / $AE19 | IRQ handlers (HUD / playfield split)       |
| $A066 | player terrain-contact state machine               |
| $A332 | bullet engine                                      |
| $A4CE | joystick control / banking physics                 |
| $A72C | playfield window renderer (self-modifying copy)    |
| $A7D8 | coords → map buffer address                        |
| $AABA | prisoner engine                                    |
| $AC2B | fuel system                                        |
| $ADD3 | scanner window → soft chars copier                 |
| $B044 | sprite shape expander                              |
| $B0CB | sprite positioning (x doubled, $D010 MSB table $B0F7) |
| $9C52/$9CDA | enemy helicopter AI                          |

| table | content                                            |
|-------|----------------------------------------------------|
| $8D2B/$8D42 | RLE run tables (terrain / scanner)           |
| $8D46/$9116 | per-level map / scanner stream sources       |
| $8DFC–$8E0F | difficulty variants                          |
| $9107/$9110/$910A | per-level colour / camera / player start |
| $911C–$9133 | tank homes                                   |
| $963C | patrol-craft animation char pairs                  |
| $9CBA/$9CCA | enemy-heli patrol points                     |
| $A320 | helicopter animation (18 entries, 7 poses × 2 rotor) |
| $A45D | object/obstacle char table                         |
| $A469 | tank body chars ($6C $6D $6E)                      |
| $A472/$A477 | bullet trajectory dx/dy                      |
| $AC16 | landing-probe ignore chars (prisoner glyphs + $44) |
| $ACF8 | fuel-depot drain animation (5 frames × 30 bytes)   |
| $B0C2 | bullet sprite dot pattern                          |
