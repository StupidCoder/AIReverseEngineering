# Elite (C64) — tape format, loader, and game analysis

A reverse-engineering reference for `Elite.tap` (the tape carries a single
autostarting file named "ELITE"). So far it covers the tape image, the
self-modifying copy-protection loader, the game program's startup and the ship
graphics, with the remaining data formats and mechanics to follow, in reading
order:

* **Part I** — the TAP container and both tape encodings (standard KERNAL and
  the custom fastloader), enough to extract every byte from the raw image;
* **Part II** — the boot chain from the KERNAL autostart trick to the game's
  first instruction, including the self-modifying loader and its
  copy-protection tricks;
* **Part III** — the game program's startup: the layered decryption, the
  relocation of the engine, the hardware and interrupt setup, and the memory
  map after loading;
* **Part IV** — graphics and data formats; so far the wireframe ship models
  (their blueprint structure and the vector-drawing pipeline);
* *(more of Part IV, and game mechanics, to come.)*
* **Appendices** — toolchain and reproduction.

Methods: purely static analysis of the image bytes — no external tools or
references, everything below was derived from the bytes in the image and
disassembly of the loader and game code it carries. Because the fastloader
rewrites its own wire format as it runs, the Go extraction toolchain
(`extract/` plus the shared `c64tools/`) does not reimplement the protocol; it
*runs* the actual loader on a small 6502 emulator and logs what it writes
(Appendix A). The game code is encrypted on tape; the same emulator runs the
loaded image far enough to decrypt it in memory, which is how the routines in
Part III were recovered for disassembly. All addresses are C64 memory
addresses.

---

# Part I — The tape image

## 1. TAP container

The image is a standard TAP v1 file. The TAP format stores the duration of
every pulse (time between falling edges) that the datasette would deliver to
the C64:

```
0000  43 36 34 2d 54 41 50 45 2d 52 41 57    "C64-TAPE-RAW"
000C  01                                     version 1
000D  00 00 00                               (padding)
0010  24 3b 0c 00                            payload size = $000C3B24 (801572)
0014  30 30 30 30 ...                        pulse data
```

* Each payload byte `n ≠ 0` is one pulse of `n × 8` clock cycles
  (PAL clock = 985248 Hz, so `$30` = 384 cycles ≈ 390 µs).
* In version 1, a `00` byte is an escape: the next three bytes are a
  little-endian pulse length in cycles. This image uses it for the silent
  gaps between files (e.g. `00 40 e1 0f` = 1,040,000 cycles ≈ 1.06 s).

Only five pulse widths matter on this tape:

| byte | cycles | used by |
|------|--------|------------------------------|
| `$30`| 384    | ROM format *short*, fastloader *0-bit* |
| `$42`| 528    | ROM format *medium* |
| `$56`| 688    | ROM format *long* |
| `$5D`| 744    | fastloader *1-bit* |
| `00 …`| ≥ 10⁶ | pauses between segments |

### Layout of this image

The pauses split the tape into 12 segments:

| seg | TAP offset | content |
|-----|-----------|---------|
| 0 | $000014 | CBM ROM format: header block for file "ELITE" |
| 1 | $008A49 | CBM ROM format: data block, boot code `$029F-$03C0` |
| 2 | $00CEA5 | turbo: vector patch, BASIC stub `$0801`, exit patch |
| 3 | $00DD25 | turbo (BASIC `LOAD` #1): game part `$4000-$86CC` |
| 4 | $042E7D | turbo (BASIC `LOAD` #2): multi-load routine `$CE0E-$CF40`, `$8609-$86CC` |
| 5 | $044B91 | turbo (kernal LOAD 1): colour data `$6000-$6400` → `$D800` |
| 6 | $0485EC | turbo (kernal LOAD 2): colour data `$6000-$6400` |
| 7 | $04C33E | turbo (kernal LOAD 3): bitmap loading picture `$4000-$6000` |
| 8 | $06378A | turbo (kernal LOAD 4): game part `$1D00-$3ECF` |
| 9 | $07BEFC | turbo (kernal LOAD 5): game part `$7300-$CA6D` |
| 10 | $0BBEB2 | turbo (kernal LOAD 6): colour data `$6000-$6400` |
| 11 | $0BFFD4 | turbo (kernal LOAD 7): `$4000-$41EA` + `$6000-$6400` |

## 2. Standard KERNAL encoding (boot file)

The first two segments are a normal CBM tape file, readable by the kernal
ROM loader:

* Bits are pairs of pulses: `(short,medium)` = 0, `(medium,short)` = 1.
* A byte frame is a `(long,medium)` marker, 8 data bit pairs LSB-first, and an
  odd-parity bit pair. `(long,short)` marks end-of-data.
* A block is: leader of short pulses, sync countdown `89 88 87 86 85 84 83 82
  81`, payload, XOR checksum; the block then repeats with countdown
  `09 08 … 01`.

Decoded header block (segment 0):

```
03 9F 02 C0 03 45 4C 49 54 45 20 20 ...
│  └──┬──┘ └──┬──┘ E  L  I  T  E
│  $029F   $03C0   filename (16 chars, space padded)
└─ type 3 = non-relocatable program
```

Segment 1 is the 289-byte program `$029F-$03C0` plus XOR checksum — the boot
code. Its load address is the whole trick that autostarts the loader; that and
the boot code itself are covered in Part II.

## 3. The fastloader encoding

From segment 2 on, the tape uses a custom turbo encoding. This section
describes the bytes on the tape; how the loader code that reads them is itself
patched mid-stream is Part II.

### Bit encoding — CIA timer as pulse discriminator

Two pulse widths encode bits: `$30` (384 cycles) = **0**, `$5D` (744 cycles) =
**1**. The threshold is implemented with CIA 2 timer A, latch `$0243` = 579
cycles: the loader restarts the timer on every edge, and if it underflowed
during the pulse, the pulse was "long". Cassette edges arrive on CIA 1's FLAG
line (ICR bit 4, `$DC0D`).

```
0334  LDA #$10                       getbit:
0336  BIT $DC0D                        wait for FLAG (tape edge)
0339  BEQ $0336
033B  LDA #$11
033D  STA $DD0E                        restart CIA2 timer A (latch $0243)
0340  LDA $DD0D                        read CIA2 ICR
0343  LSR A                            bit 0 (timer A underflow) → carry
0344  BIT $DC0D                        clear FLAG
0347  ROL $FC                          shift bit into $FC
0349  RTS

034A  LDX #$09                       getbyte:
034C  JSR $0334                        read 9 bits into $FC
034F  INC $D020                        (border flicker)
0352  DEX
0353  BNE $034C
0355  LDA $FC
0357  RTS
```

Latch setup at the loader entry point:

```
0378  LDA #$43  STA $DD04            timer A latch = $0243 = 579 cycles
037D  LDA #$02  STA $DD05            (between 384 and 744)
0382  LDY #$7F  STY $DD0D  STY $DC0D disable CIA interrupts
038A  LDA #$07  STA $01              tape motor on, ROMs in
```

### Byte framing, pilot and sync

A byte is **9 pulses: one start bit (always 1) followed by 8 data bits,
MSB first** (`ROL` into `$FC`; the ninth shift pushes the start bit out).
Raw TAP bytes:

```
pilot, repeated ~256 times (= byte $00):      5D 30 30 30 30 30 30 30 30
sync byte $16 (1 00010110):                   5D 30 30 30 5D 30 5D 5D 30
```

Synchronisation (entry `$038E`):

1. shift bits into `$FC` (preset `$7F`) until 8 consecutive 0-bits arrive
   → guaranteed to happen only inside the pilot;
2. read byte frames while they decode to `$00`;
3. the first non-zero byte must be `$16`, otherwise restart at 1.

### Block format

After the sync byte comes a sequence of blocks, **back to back, with no
checksums and no gaps**:

```
end-hi  end-lo  start-hi  start-lo  data[end-start] ...next block...
```

Real example — the first frames after the very first `$16` (TAP offset
$00D7A5): header `03 34 03 00` = block `$0300-$0334`, followed by 52 data
bytes. The header is stored to `$AF,$AE,$AD,$AC` and the store loop is:

```
03A3  LDY #$03                       read 4 header bytes
03A5  JSR $034A
03A8  STA $00AC,Y                     Y=3..0 → $AF,$AE,$AD,$AC
03AB  DEY
03AC  BPL $03A5
03AE  JSR $034A                      data loop:
03B1  STA ($AC,X)                     store byte (X=0)
03B3  JSR $FCDB                       kernal: $AC/$AD++
03B6  JSR $FCD1                       kernal: compare with $AE/$AF
03B9  BCC $03AE                       until start == end
03BB  BCS $03A3                       then: next block header — forever!
```

Note the loop never terminates by itself: it always branches back for another
block header. That non-terminating loop is the hook the copy protection hangs
on — how the tape stops it (and rewrites the loader between blocks) is Part II.

Net data rate of the turbo format is ≈ 190 bytes/s (9 pulses × ~570 µs
average), roughly five times the effective rate of the ROM format with its
duplicated blocks. There is **no checksum** on any turbo data.

---

# Part II — Boot chain and loader internals

## 1. Autostart

The kernal saves the IRQ vector at `$029F/$02A0` while it uses the tape, and
**restores `$0314/$0315` from there when the load finishes**. The boot file
(segment 1) starts at `$029F` and its first two bytes are `A8 02`:

```
029F  A8 02            ← restored into $0314/$0315 = IRQ vector $02A8
```

So the first timer IRQ after "FOUND ELITE" jumps straight into the loaded
code at `$02A8` — no `RUN` needed, and the file also overwrites the BASIC
vector table at `$0300-$0333` on the way (it keeps almost all default values;
`$0314/$0315` inside the file is `2C F9` = `$F92C`, a kernal tape-IRQ routine,
so the machine survives the moment the vector table is overwritten *during*
the load).

## 2. Boot code (entry $02A8)

```
02A8  LDA #$20            fill $03C0-$07FF with spaces
02AE  STA ($AC),Y         ($AC/$AD = end-of-load address $03C0)
02B0  JSR $FCDB           kernal: increment $AC/$AD
...
02B9  LDA $FCD1,Y         fill $0800-$CFFF with kernal ROM garbage
...
02C7  LDA #$80  STA $9D   kernal messages off
02CB  STA $0800, STA $C6  zero BASIC start, clear keyboard buffer
02D2  JMP $0378           → fastloader
```

The jump to `$0378` enters the turbo loader described in Part I §3.

## 3. Self-modification — the loader rewrites itself from the tape

**Exit trick.** The block store loop (Part I §3) never terminates on its own.
To end a load, the tape simply sends a block whose address range covers the
loop itself, e.g. `$03BB-$03EB`. While that block loads, the byte at `$03BB`
changes from `B0` (`BCS`) to `A9` (`LDA #`), so when the block is complete the
loop *falls through* into the code that was just loaded:

```
03BB  A9 E6              was: BCS $03A3 — now falls through
03BD  LDA #$B0
03BF  STA $03BB          repair the BCS for the next use
03C2  ...                stage-specific exit code
```

**Protocol mutations.** Between payload blocks, the tape sends 1-3 byte
blocks aimed at single instructions of the loader, changing the wire format
on the fly (all values below were observed in the image):

| target | instruction | effect |
|--------|-------------|--------|
| `$0347` | `ROL $FC` ↔ `ROR $FC` (`$26`/`$66`) | bit order MSB-first ↔ LSB-first |
| `$034B` | `LDX #$09` operand → `$08`/`$0A` | bits per byte frame (8 = no start bit, 10 = two start bits) |
| `$03A4` | `LDY #$03` operand → `$04` | header grows to 5 bytes; the extra first byte lands in `$B0` and is never read — a decoy |
| `$03BB` | `BCS` opcode | exit trick, see above |
| `$0300-$0333` | vector table | rewritten over and over (ILOAD `$0330` → `$0378` keeps `LOAD` hijacked) |
| `$0350-$03C0` | whole loader tail | periodically re-written wholesale, sometimes byte-identical (a decoy — the stores race the executing loop and must match it) |

Because of the `ROR` flips, the *same* pilot/sync logic produces different
raw bit patterns in different phases: in LSB mode the sync byte `$16` appears
on tape as `01101000` (= `$68` read MSB-first). The pilot `$00` is a
palindrome and is unaffected.

The main payload of segment 3 (`$4000-$86CC`) is transmitted as ~70 blocks of
256 bytes, with patch blocks interleaved between almost every page —
extracting it without executing the patches is hopeless, which is clearly the
point: it is a copy-protection scheme, not just a fastloader.

## 4. Exit-code variants

Three kinds of exit blocks occur:

1. **Continue** (mid-segment): restore `BCS`, fix up some zero-page/vector
   values, `JMP $0378` — re-synchronise on the next pilot and keep loading.
2. **Return to BASIC** (end of segment 2): `JSR $FCCA` (motor off), set
   `$2D/$2E` = `$082C` (end of BASIC program), `JSR $A871`, `JMP $A7EA` —
   i.e. `RUN` the BASIC stub that was just loaded.
3. **RTS** (end of segments 3-11): set `$90` (STATUS) = `$40` (EOF),
   `CLC`, `RTS` — return as a well-behaved `ILOAD` implementation to the
   kernal `LOAD` caller.

## 5. The multi-stage load chain

```
kernal ROM load  →  boot "ELITE" $029F-$03C0 (autostart via IRQ vector $029F/$02A0)
$02A8 boot code  →  JMP $0378 fastloader
  seg 2:  vectors ($0330 ILOAD → $0378), BASIC stub $0801, exit → RUN
BASIC stub:      10 IF F=0 THEN F=1:LOAD
                 20 IF F=1 THEN F=2:LOAD
                 30 SYS 30215
  LOAD #1 (ILOAD = fastloader) → seg 3: game part $4000-$86CC
  LOAD #2                      → seg 4: $CE0E-$CF40, $8609-$86CC, patch $7604
SYS 30215 = $7607 → ... → JMP $CE0E:
  7 × JSR $FFD5 (kernal LOAD, still via ILOAD = $0378):
     seg 5  colours  $6000-$6400  → copied to colour RAM $D800
     seg 6  colours  $6000-$6400
     seg 7  bitmap   $4000-$6000  (loading picture, shown during...)
     seg 8  game     $1D00-$3ECF
     seg 9  game     $7300-$CA6D
     seg 10 colours  $6000-$6400  → copied to $D800
     seg 11 data     $4000-$41EA + $6000-$6400
  restore default vectors (table at $CF00, ILOAD back to $F4A5)
  JMP $1D1F  →  game starts
```

The BASIC-stub detour exists because a tape `LOAD` from inside a running
BASIC program restarts the program afterwards — the `F` flag variable makes
each line run once. The 5-byte headers, the bit-order flips and the no-op
rewrites have no functional purpose other than to frustrate exactly the kind
of analysis done here.

---

# Part III — Game program architecture

Part II ended at the two hand-off jumps: `SYS 30215` ($7607) and, after the
seven picture-covered loads, `JMP $1D1F`. This part follows the code from
there — how the loaded blob decrypts and relocates itself, how the display
and interrupts are set up, and how the game reaches its first frame.

Everything in the game image arrives **encrypted** and is unpacked in three
stages by three near-identical rolling-subtraction decryptors. None of the
game code is ever in plaintext on the tape, and even after loading it is
decrypted in pieces, at different times, with different keys — one more layer
of the protection.

## 1. Decryption and relocation (SYS 30215 → $7607)

The 18 KB blob loaded to `$4000–$86CC` (Part II, seg 3) is encrypted. `$7607`
first decrypts it with the routine at `$7631`: each byte is the previous
plaintext byte subtracted from the ciphertext (a rolling key), and the loop's
end address is self-modified at `$7604/$7605`:

```
7631  STX $1A            key = X
7633  STA $19            pointer high
7635  LDA #$00 STA $18   pointer low = 0
7639  LDA ($18),Y        ciphertext byte
763B  SEC SBC $1A        minus rolling key
763E  STA ($18),Y        store plaintext
7640  STA $1A            key for next byte = this plaintext
7642  TYA BNE $7647
7645  DEC $19            cross to previous page when Y wraps
7647  DEY
7648  CPY $7604          end-of-range low  (self-modified)
764B  BNE $7639
764D  LDA $19 CMP $7605  end-of-range high (self-modified)
7652  BNE $7639
7654  RTS
```

`$7607` calls it twice, walking **downward** through memory:

| pass | range          | initial key |
|------|----------------|-------------|
| 1    | `$7655–$86CB`  | `$8E`       |
| 2    | `$4000–$7600`  | `$6C`       |

The gap `$7601–$7654` is the decryptor itself, left in clear. Concretely, the
12 bytes at `$7655` go from ciphertext to code:

```
encrypted:  b8 bf a9 85 9d c1 b0 8c 9e c2 a9 85
decrypted:  a2 16 a9 00 85 18 a9 07 85 19 a9 00   (LDX #$16 / LDA #$00 / STA $18 …)
```

The decrypted `$7655` then **relocates** the engine and sets up the loading
screen. Copies use the page-mover at `$7885` (X pages, source `$1A/$1B` →
destination `$18/$19`):

```
7655  copy $16 pages  $4000-$55FF -> $0700        (engine code low)
7669  $01 = $04                                   bank RAM in under I/O
7671  copy $20 pages  $5600-$75FF -> $D000-$EFFF   (engine code hidden under I/O)
7684  $01 = $05                                   I/O back in
768C  DD02 |= $03, DD00 = (..&$FC)|$02            VIC bank
76A6  D018=$81, D011=$3B (bitmap), D016=$C0        loading-screen display
76C4  D025/D026/D029-D02E …                        sprite colours
```

Relocating `$5600–$75FF` underneath `$D000–$EFFF` hides the bulk of the engine
as RAM under the I/O area, reached by toggling the `$01` bank bits. After a few
more copies (character and sprite data) it `JMP`s to `$CE0E`, the multi-load
routine that pulls in the remaining seven segments behind the bitmap picture
(Part II §5) and finally jumps to `$1D1F`.

## 2. In-place decrypt and hand-off ($1D1F)

`$1D1F` is the game's real entry. It preserves the loader's zero page, decrypts
the last two regions in place, initialises the hardware, and starts the game:

```
1D1F  CLD
1D20  LDX #$02                  back up zero page $02-$FF
1D22  LDA $00,X  STA $CE00,X    -> $CE02-$CEFF
1D27  INX  BNE $1D22
1D2A  JSR $1D33                 in-place decrypt (below)
1D2D  JSR $B3B2                 hardware init (§3)
1D30  JMP $916F                 game start
```

`$1D33` is the same rolling-subtraction cipher as `$7631` (its end addresses
self-modify `$0452/$0453`), run over the two regions that arrived encrypted on
the last loads:

| range          | initial key |
|----------------|-------------|
| `$1D7E–$3ECE`  | `$36`       |
| `$7300–$CA6C`  | `$49`       |

The second region holds the running engine — the IRQ handler, hardware-init
and SID player all live in `$7300–$CA6C`.

## 3. Hardware init ($B3B2)

```
B3B2  clear $0400-$06FF
B3C7  $0318/$0319 = $B433       NMI vector -> CLI/RTI (RESTORE neutralised)
B3D1  $0326/$0327 = $BA61       BSOUT vector
B3DB  LDA #$05 JSR $8B8B        bank helper: all-RAM + I/O
B3E0  SEI; wait for raster $37  sync
B3ED  DC0D/DD0D = $03           disable CIA interrupts
B3F5  D418 = $0F                SID volume
B400  D01A = $01                enable raster IRQ
B40B  D012 = $28                first IRQ at line $28
B40D  D011 &= $7F               (clear high raster bit)
B410  $01 = $04                 KERNAL/BASIC banked out, I/O in
B41D  $FFFA/$FFFB = $B433       RAM NMI hardware vector
B427  $FFFE/$FFFF = $B1FA       RAM IRQ hardware vector
B431  CLI RTS
```

With the KERNAL ROM banked out (`$01` bit 1 = 0), the CPU takes its IRQ/NMI
vectors from **RAM** at `$FFFE`/`$FFFA`, so interrupts dispatch straight to the
game's own handlers with no ROM in the path.

## 4. Interrupt architecture

The IRQ at `$B1FA` is a table-driven raster-split engine. It reads the current
split index from `$B1D9` and loads that split's register values from seven
two-entry tables at `$B1DA–$B1E7`:

```
B20D  LDX $B1D9
B210  LDA $B1DA,X  STA $D018    char/screen base
B216  LDA $B1E0,X  STA $D016    mode / x-scroll
B21C  LDA $B1DE,X  STA $D012    next split's raster line
B222  LDA $B1E2,X  STA $D01C    sprite multicolour
B228  LDA $B1E4,X  STA $D028    sprite colour
B236  LDA $B1E6,X  STA $D021    background
B23C  LDA $B1DC,X  STA $B1D9    next split index
B242  BNE $B1ED                 not the last split -> just RTI
```

There are **two splits per frame**, raster lines `$33` and `$C2`:

| reg    | split 0 (`$33`) | split 1 (`$C2`) |
|--------|-----------------|-----------------|
| `$D012` next line | `$C2` | `$33` |
| `$D018`           | `$81` | `$81` |
| `$D016`           | `$C0` | `$C0` |
| `$D01C` spr. MC   | `$FE` | `$FC` |
| `$D028` spr. col  | `$02` | `$00` |
| `$D021` bg        | `$00` | `$00` |
| next index        | `1`   | `0`   |

When the index returns to 0 (the second split completes the frame), the
handler falls through to the per-frame work instead of exiting early: a
three-voice SID player driven from the tables at `$B313`, plus an optional
`JSR $BDDC` when `$1D03` bit 7 is set. The handler restores `$01` from `$8B9A`
and `RTI`s.

From `$B3B2`'s `CLI` onward the game is interrupt-driven. `$916F` runs the
one-time game setup — it clears the flag block `$1D01–$1D11`, then builds the
title/commander screen through a chain of subroutines (`$8CD6`, `$9563`,
`$9220` text, …) — before the foreground settles into its main loop (left to a
later part).

## 5. Memory map (after load)

| range         | content                                                       |
|---------------|---------------------------------------------------------------|
| `$0002–$00FF` | zero page (backed up to `$CE02` at game start)                |
| `$0100–$01FF` | stack                                                         |
| `$0300–$0333` | KERNAL vectors (restored defaults; NMI `$0318`→`$B433`, BSOUT `$0326`→`$BA61`) |
| `$0400–$06FF` | cleared at init                                               |
| `$0700–$1CFF` | engine code, relocated from `$4000–$55FF`                     |
| `$1D00–$1D7D` | game entry + in-place decryptor (plaintext)                   |
| `$1D7E–$3ECF` | game code/data, decrypted in place (key `$36`)                |
| `$7300–$CA6C` | game engine, decrypted in place (key `$49`): IRQ `$B1FA`, hardware-init `$B3B2`, game start `$916F`, SID player `$B313`/`$BDDC` |
| `$CE00–$CF40` | zero-page backup + the multi-load routine                     |
| `$D000–$EFFF` | engine code under I/O, relocated from `$5600–$75FF` (reached via the `$01` bank bits) |
| `$FFFA/$FFFE` | RAM NMI/IRQ hardware vectors (`$B433` / `$B1FA`)              |

The character sets, sprite shapes and in-game screen and colour memory are
graphics data, covered in a later part. The bitmap **loading** picture, which
also occupies `$4000–$6000` during the load, is covered next.

## 6. The loading screen

While the long segments stream in, Elite shows its title picture — the 3-D
"ELITE" logo and a Cobra Mk III over a starfield:

![Elite loading screen](rendered/loading-screen.png)

It is a **multicolor bitmap** (160×200 colour pixels), stored **uncompressed**
and split across three tape segments rather than packed — consistent with the
rest of the tape, which favours obfuscation over economy. The multi-load
routine at `$CE0E` (Part II §5) assembles it from those segments:

- seg 7 → `$4000–$5F3F`: the 8000-byte VIC bitmap (the pixel data);
- seg 6 → `$6000`: the 1000-byte video matrix (colours for bit-pairs 01/10);
- seg 5 → `$6000`, then copied to colour RAM `$D800` (`$CE3F` loop): the third
  colour, bit-pair 11. The background (bit-pair 00) is white — `$D021 = $01`,
  set at `$CE60`.

Multicolor mode is selected at `$CE56` (`$D016` bit 4), and the display is
switched on at `$CE65` (`$D011` DEN bit) *after* the bitmap and its colours are
in place but *before* the two largest loads — the game code at `$1D00–$3ECF`
and `$7300–$CA6D` — so the picture masks the slowest part of the load. When
those finish, the loader blanks the screen, zeroes `$4000–$5FFF` to reclaim the
bitmap RAM for the game, restores the KERNAL vectors and jumps to `$1D1F`.

The image above was produced by the `loadingscreen` tool (Appendix A), which
reassembles the three segments and renders the multicolor bitmap.

---

# Part IV — Graphics and data formats

## 1. Ship models

Elite's ships are filled-edge **wireframe vector models**: each is a list of
3-D vertices joined by edges, with face normals used to hide the back. All of
the model data lives in the engine block that the loader hid under the I/O area
at `$D000–$EFFF` (Part III §1), so the routines that read it bank ROM/I-O out
first.

### 1.1 The blueprint table

A pointer table of 16-bit little-endian addresses, indexed by ship type × 2,
sits at `$CFFE` (so type *T*'s blueprint address is at `$CFFE + T*2`; type 1 is
the first real entry, at `$D000`). There are **33 ship types**, with blueprints
packed from `$D0A5` to about `$EE2D`:

```
type  1: $D0A5     type 12: $DA4B     type 23: $E45B
type  2: $D1A3     type 13: $DB3D     type 24: $E50B
...                ...                ...
type 11: $D8C3     type 22: $E395     type 33: $EE2D
```

The table is read by the spawn routine (`$855B`, NWSHP) and the per-ship draw
path (`$2030` → `$ABA0`), each doing `LDA $CFFE,Y / STA $57 ; LDA $CFFF,Y / STA
$58` to point a zero-page vector (`$57/$58`) at the chosen blueprint.

### 1.2 Blueprint layout

Each blueprint is a 20-byte header followed by three packed arrays — vertices,
edges, faces:

```
+0           flags (laser/missile/AI bits)
+1 +2        targetable area (16-bit: bounding-radius², for laser hits)
+3           EDGES offset  (byte offset from blueprint start to the edge array)
+4           FACES offset  (low byte of the offset to the face array)
+5           visibility distance / model "size" (drawn as a dot beyond this)
+9           number of edges (NE)
+0E +0F      level-of-detail / max-size attributes (read by NWSHP and MVEIT)
+13          AI / energy attributes
... (remaining bytes: scaling and AI/economy attributes)
```

The header carries no explicit vertex or face count; both are derived from the
offsets, because every array has a fixed record size:

```
vertices start at offset 20           NV = (EDGES_offset − 20) / 6
edges    start at EDGES_offset         NE = header[+9]   (= FACES_offset − EDGES_offset, /4)
faces    start at FACES_offset         NF = (blueprint_length − FACES_offset) / 4
```

This was confirmed two ways. First, **header[+9] equals (FACES−EDGES)/4** on
26 of the 33 ships (the other seven are large models whose face offset exceeds
255, so the single header byte at +4 holds only the low byte — the true offset
is still `EDGES_offset + NE*4`). Second, the resulting vertex/edge/face counts
satisfy **Euler's polyhedron formula V − E + F = 2** for every model that is
stored contiguously, e.g.:

| type | NV | NE | NF | V−E+F |
|------|----|----|----|-------|
| 1    | 17 | 24 | 9  | 2     |
| 2    | 16 | 28 | 14 | 2     |
| 9    | 19 | 30 | 13 | 2     |
| 13   | 13 | 24 | 13 | 2     |

**Vertex record — 6 bytes:**

```
+0  |x|        magnitude of the x coordinate
+1  |y|        magnitude of the y coordinate
+2  |z|        magnitude of the z coordinate (depth; ships point along +z)
+3  %sss vvvvv bits 7-5 = sign of x,y,z; bits 4-0 = visibility distance
+4  %aaaa bbbb two face numbers (nibbles) this vertex belongs to
+5  %cccc dddd two more face numbers
```

The four face references let the projector decide a vertex is visible if **any**
of its faces is visible. The visibility-distance field is level-of-detail: fine
detail vertices carry a small value and are only drawn close up.

**Edge record — 4 bytes:**

```
+0  visibility distance (skip the edge when the ship is further than this)
+1  %aaaa bbbb the two faces on either side of the edge (nibbles)
+2  vertex 1 number × 4
+3  vertex 2 number × 4
```

Vertex numbers are pre-multiplied by 4 because the projected screen coordinates
are stored 4 bytes per vertex (x and y as 16-bit words) in a work buffer, so the
stored value indexes that buffer directly. An edge is drawn only if at least one
of its two faces is currently visible.

**Face record — 4 bytes:**

```
+0  %sss vvvvv bits 7-5 = sign of the normal's x,y,z; low bits = visibility/illum
+1  |normal_x|
+2  |normal_y|
+3  |normal_z|
```

The signed normal vector is dotted with the vector from the ship to the viewer;
a positive result means the face points towards the camera and is visible. This
back-face test is what makes the wireframe look solid — only the front edges are
drawn.

### 1.3 Worked example — type 1

```
header:  00 40 06 7a da 55 00 0a 66 18 00 00 24 0e 02 2c 00 00 02 00
         └+0   └+1+2  └+3 └+4 └+5       └+9
```

`+3 = $7A (122)` → vertices = (122−20)/6 = **17**.
`+4 = $DA (218)`, `+9 = $18 (24)` → edges = (218−122)/4 = **24**.
blueprint is 254 bytes → faces = (254−218)/4 = **9**.  17 − 24 + 9 = 2. ✓

```
vertex 0:  00 00 44 1f 10 32   (0, 0, 68); signs +,+,+; vis 31; faces {1,0,3,2}
vertex 1:  08 08 24 5f 21 54   (8, -8, 36); sign of y set; vis 31; faces {2,1,5,4}
edge 0:    1f 21 00 04         vis 31; faces 2,1; vertices 0 and 1 (00/4, 04/4)
face 0:    9f 40 00 10         normal (-64, 0, 16) (x sign set); always visible
```

Vertex 0 at (0, 0, 68) sits on the +z axis — the model's nose — shared by four
faces, exactly as expected for a pointed ship.

### 1.4 The rendering pipeline

Drawing one ship runs through these stages (all addresses below are also in the
table in §1.5):

1. **Per-ship setup (`$2030`).** The ship's 37-byte state block is copied from
   its universe slot into the zero-page workspace at `$0009`, and its blueprint
   pointer is loaded into `$57/$58`.
2. **Rotate & cull-by-distance (`$ABA0`, MVEIT).** The ship's orientation
   vectors are applied; the model "size" is clamped against header byte `+0F`
   for level-of-detail, and far ships are dropped or reduced.
3. **Project & build the line heap (`$A3A0`, LL9).** Each vertex is rotated into
   view space and perspective-projected (the depth divide) to a screen x,y,
   stored 4 bytes per vertex. Faces are back-face tested with their normals;
   each edge whose face(s) are visible and whose visibility distance passes is
   appended to the ship's **line heap at `$0580`** as a 4-byte record
   `(x1, y1, x2, y2)`. The heap begins with a length byte.
4. **Draw the heap.** The line-list drawer (`$AA72`) walks the heap — a count
   followed by 4-byte endpoint records — calling the **LINE** routine for each.
5. **LINE (`$B49D`).** A Bresenham line drawn into the multicolor space-view
   bitmap at `$4000`. The major/minor axis split is handled at `$B814`; the
   gradient comes from the reciprocal tables at `$9C00–$9F00`; the bitmap byte
   address is formed from the row tables `$A000` (low) / `$A100` (high) plus
   `(x & $F8)`; the inner plot loop EORs pixels (relocated, at `$8888`).
   Single points (stars, distant ships) instead use **PIXEL (`$2911`)**, which
   plots a 1-, 2- or 4-pixel dot depending on distance (`$A1`), using the
   2-bit multicolor masks at `$28C5`.

Because lines are plotted with EOR, the previous frame's ship can be erased by
drawing the same line heap again before the new one is built — the standard
Elite flicker-free redraw.

### 1.5 Routine and data map (ship rendering)

| address | name | role |
|---------|------|------|
| `$CFFE` | XX21 | blueprint pointer table (33 ships, word per type×2) |
| `$D0A5–$EE2D` | — | the 33 ship blueprints (under I/O) |
| `$0580` | line heap | per-ship list of projected line segments |
| `$0009–$002D` | workspace | the active ship's 37-byte state block (INWK) |
| `$2030` | — | per-ship processor: slot↔workspace copy, fetch blueprint, call MVEIT |
| `$ABA0` | MVEIT | rotate ship, level-of-detail/visibility, dispatch draw |
| `$A3A0` | LL9 | project vertices, back-face cull, build the line heap |
| `$855B` | NWSHP | create a ship in a universe slot (reads blueprint attrs +5,+0E,+0F,+13) |
| `$AA72` | — | draw a line list (count + 4-byte `x1,y1,x2,y2` records) via `$2A` |
| `$B49D` | LINE | draw one Bresenham line into the view bitmap (multicolor) |
| `$B814` | — | LINE's steep-axis (dy>dx) variant |
| `$8888` | — | LINE inner EOR plot loop (relocated under I/O) |
| `$2911` | PIXEL | plot a distance-scaled point (1/2/4 px) to the view bitmap |
| `$9C00–$9F00` | — | reciprocal/gradient tables used by LINE |
| `$A000 / $A100` | — | bitmap scanline address tables (low / high byte) |
| `$28B7` | — | 8-bit hires pixel-mask table |
| `$28C5` | — | multicolor 2-bit dot-mask table (used by PIXEL) |
| `$8DBB` | DORND | random-number generator |

The view bitmap itself is at `$4000` (VIC bank 1, multicolor — Part III), the
same RAM the loading picture used; once loading is done it becomes the live
space view that the ship renderer draws into.

### 1.6 Rendered models

The `shiprender` tool (Appendix A) reads the decoded blueprints and draws them
exactly as described above — projecting the vertices and applying the same
back-face hidden-surface removal the game uses (an edge appears only when one of
its two faces points towards the camera) — as white lines on black. It decodes
32 of the 33 table entries (one slot is not a model) and writes a montage plus a
rotating animation per ship:

![All ship models](rendered/ships-montage.png)

The animations spin each model around its up axis. Four examples — a faceted
freighter hull, a flat angular hull, a sharp-nosed fighter, and a rounded
many-faceted station-like hull (identified here only by blueprint type, since
the in-game names are stored as encrypted text tokens not yet decoded):

| ![type 10](rendered/ships/ship-10.png) | ![type 11](rendered/ships/ship-11.png) | ![type 19](rendered/ships/ship-19.png) | ![type 33](rendered/ships/ship-33.png) |
|:--:|:--:|:--:|:--:|
| type 10 | type 11 | type 19 | type 33 |

(These are animated PNGs; they spin in any viewer that supports APNG — including
GitHub's markdown — and show the first frame as a still everywhere else.)

---

# Appendix A — Toolchain and reproduction

Because the wire format is rewritten on the fly by code that arrives on the
wire itself, the extractor does not reimplement the protocol. Instead it
**runs the actual loader** from the image. Most of the machinery is shared
with the other games in this repository via the `c64tools` module (see the
root `README.md`); only the Elite-specific glue lives in `extract/`:

1. `c64tools/tap` parses the TAP container into pulses.
2. `c64tools/cbmtape` decodes the ROM-format boot file (checksum verified).
3. `c64tools/mos6502` (CPU) and `c64tools/c64` (machine model) run the boot
   code on a small 6502 emulator. The only hardware modelled is what the
   loader touches: CIA1 FLAG edges fed from the TAP pulse stream, and CIA2
   timers A/B as pulse-width discriminators against their latches. The
   standard KERNAL tape entry points ($FCDB, $FCD1, $FCCA, $FF90) are
   provided by `c64tools/c64`; the Elite-specific hooks in `extract/driver.go`
   add KERNAL LOAD ($FFD5/ILOAD) and the BASIC statement loop $A7EA, which
   drives the loaded `LOAD`/`LOAD`/`SYS` stub.
4. `extract/main.go` logs every memory write performed while tape pulses are
   being consumed; contiguous runs are coalesced into blocks and written out
   as `.prg` files (one per contiguous region per tape segment), plus
   `memory_final.bin` (the full 64 KB RAM image at the end) and `report.txt`
   (every block in load order).

```
# Run from this game folder ("Elite (C64)/"). The go.work workspace at the
# repository root lets the extract module find the shared c64tools packages.

# 1. Summarise the tape (pulse histogram + segment map)
go run stupidcoder.com/c64tools/cmd/tapdump Elite.tap

# 2. Extract all program files by running the loader under emulation
( cd extract && go build -o extract . )
extract/extract -o extracted Elite.tap

# 3. Render the multicolor-bitmap loading screen to rendered/loading-screen.png
( cd extract && go run ./cmd/loadingscreen -o ../rendered )

# 4. Render the wireframe ships (rotating animated PNGs + a montage)
( cd extract && go run ./cmd/shiprender -o ../rendered )

# 5. Disassemble anything (shared tool, run by import path) — e.g. the
#    getbit/getbyte routines at $0334 inside the boot file
( cd extract && go run stupidcoder.com/c64tools/cmd/disprg -start 0334 -end 0358 \
    ../extracted/00_cbm_ELITE_029f.prg )

# run this module's tests
( cd extract && go test ./... )
```

The run consumes all 801,536 pulses of the image; the emulated game ends up
in its idle loop with the tape exhausted. Output files with `loader_` in their
name are the self-modification blocks aimed at `$0280-$03FF`; the rest are the
actual program data listed in the segment table in Part I §1.

Package overview — game-specific (`extract/`): `main.go` (write coalescing and
file output), `driver.go` (the BASIC-stub driver and Elite-specific KERNAL
hooks), `shipmodel` (engine reconstruction + blueprint decoding),
`cmd/loadingscreen` (reassembles and renders the loading picture),
`cmd/shiprender` (wireframe ship animations). Shared (`c64tools/`): `tap` (TAP
container), `cbmtape` (ROM-loader decoder), `mos6502` (disassembler + CPU
emulator), `c64` (machine model), `gfx` (rendering primitives: multicolor
bitmap, line drawing, animated-PNG output), `cmd/disprg`, `cmd/tapdump`.
