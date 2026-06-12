# Elite (C64) — Tape Image Format and Fastloader

Findings from a purely static analysis of `Elite.tap`. No external tools or
references were used; everything below was derived from the bytes in the image
and disassembly of the loader code it contains. The extraction tool in
`extract/` is built directly on these findings.

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

## 2. The standard (kernal ROM) part — boot file

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

Segment 1 is the 289-byte program `$029F-$03C0` plus XOR checksum. This load
address is the whole trick:

### Autostart

The kernal saves the IRQ vector at `$029F/$02A0` while it uses the tape, and
**restores `$0314/$0315` from there when the load finishes**. The file starts
at `$029F` and its first two bytes are `A8 02`:

```
029F  A8 02            ← restored into $0314/$0315 = IRQ vector $02A8
```

So the first timer IRQ after "FOUND ELITE" jumps straight into the loaded
code at `$02A8` — no `RUN` needed, and the file also overwrites the BASIC
vector table at `$0300-$0333` on the way (it keeps almost all default values;
`$0314/$0315` inside the file is `2C F9` = `$F92C`, a kernal tape-IRQ routine,
so the machine survives the moment the vector table is overwritten *during*
the load).

### Boot code (entry $02A8)

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

## 3. The fastloader

### 3.1 Bit encoding — CIA timer as pulse discriminator

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

### 3.2 Byte framing, pilot and sync

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

### 3.3 Block format

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

Note the loop never terminates by itself. That leads to the protection's
core trick.

### 3.4 Self-modification — the loader rewrites itself from the tape

**Exit trick.** To end a load, the tape simply sends a block whose address
range covers the loop itself, e.g. `$03BB-$03EB`. While that block loads, the
byte at `$03BB` changes from `B0` (`BCS`) to `A9` (`LDA #`), so when the
block is complete the loop *falls through* into the code that was just
loaded:

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

### 3.5 Exit-code variants

Three kinds of exit blocks occur:

1. **Continue** (mid-segment): restore `BCS`, fix up some zero-page/vector
   values, `JMP $0378` — re-synchronise on the next pilot and keep loading.
2. **Return to BASIC** (end of segment 2): `JSR $FCCA` (motor off), set
   `$2D/$2E` = `$082C` (end of BASIC program), `JSR $A871`, `JMP $A7EA` —
   i.e. `RUN` the BASIC stub that was just loaded.
3. **RTS** (end of segments 3-11): set `$90` (STATUS) = `$40` (EOF),
   `CLC`, `RTS` — return as a well-behaved `ILOAD` implementation to the
   kernal `LOAD` caller.

### 3.6 The multi-stage load chain

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

Net data rate of the turbo format is ≈ 190 bytes/s (9 pulses × ~570 µs
average), roughly five times the effective rate of the ROM format with its
duplicated blocks. There is **no checksum** on any turbo data.

## 4. The extractor (`extract/`)

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

Usage:

```
cd extract && go build && ./extract -o ../extracted ../Elite.tap
```

The run consumes all 801,536 pulses of the image; the emulated game ends up
in its idle loop with the tape exhausted. Output files prefixed `loader_`
are the self-modification blocks aimed at `$0280-$03FF`; the rest are the
actual program data listed in the segment table above.
