# AI Reverse Engineering

Are the latest LLMs like Claude Fable smart enough to reverse engineer old
software purely by static analysis, without access to a debugger? This
repository contains the results of a few tests, attempting to answer that
question.

Used prompts:

> Write a GoLang application that extracts program files from a C64 tape image
> in TAP format. The image in question might use a non-standard fastloader,
> that you need to reverse engineer by disassembling the loader. Use only
> static analysis of the image file without looking up external sources or
> external tools like Vice. Ignore memories from previous reverse engineering
> sessions. Document your findings about the tape image format in general and
> the fastloader in particular in a markdown file. Use example byte sequences
> in that documentation where appropriate. The tape image file in question is
> <GAME.TAP> in the current folder.

## Repository structure

Each game lives in its own `<Game> (<Platform>)/` folder following a common
contract, and the reusable C64 building blocks live in a shared `c64tools`
module at the root. A `go.work` ties the modules together.

```
AIReverseEngineering/
├── go.work                     # Go workspace over c64tools + each game's extract/
├── README.md                   # this file
├── c64tools/                   # shared C64 tooling (module stupidcoder.com/c64tools)
│   ├── tap/                    #   TAP container parser + segmentation
│   ├── cbmtape/                #   standard KERNAL ROM tape-format decoder
│   ├── mos6502/                #   6502 disassembler + executable CPU core
│   ├── c64/                    #   machine model for running hostile loaders
│   ├── gfx/                    #   palette, char/sprite/bitmap rendering, PNG
│   └── cmd/
│       ├── disprg/             #   disassemble a .prg file
│       └── tapdump/            #   pulse histogram + segment listing for a .tap
│
├── Elite (C64)/
│   ├── Elite.tap               # raw tape image
│   ├── TapeFormat.md           # tape container + fastloader writeup
│   ├── extract/                # module elite/extract — the extraction tool
│   └── extracted/              # generated .prg files (regenerable; git-ignored)
│
└── Fort Apocalypse (C64)/
    ├── Fort_Apocalypse.tap      # raw tape image
    ├── Fort_Apocalypse.md       # full game + tape writeup
    ├── emu.py                   # dynamic-verification scratch emulator
    ├── extract/                 # module fortapoc/extract — extraction + gfx tools
    ├── extracted/               # generated .prg files (regenerable; git-ignored)
    └── rendered/                # generated PNGs (charsets, maps, sprites)
```

Per-game folder contract: `<Game>.<ext>` raw image, a markdown writeup of the
format and/or game internals, an `extract/` Go module that produces the
program files, and `extracted/` / `rendered/` output directories. The
extraction tool is always named `extract` (not tied to tape — a future game
could be disk- or cartridge-based, while the per-game tool keeps the same
name).

## Shared tools (`c64tools`)

| Package / command | What it does |
|-------------------|--------------|
| `tap` | Parse a TAP v0/v1 image (C64/C16) into a pulse stream; `Segmentize` splits it at pauses. |
| `cbmtape` | Decode the standard Commodore KERNAL (ROM loader) tape encoding: blocks, headers, and paired header+data files with checksum verification. |
| `mos6502` | One opcode table driving both a `Disassemble` function and an executable `CPU` core (all documented opcodes, binary + BCD). |
| `c64` | A minimal C64 machine model — RAM, the `mos6502` CPU, a CIA pulse-feed tape model, a PC-hook registry and a RAM write log — for *running* a self-modifying loader instead of decoding it. Optional standard KERNAL tape hooks included. |
| `gfx` | Generic rendering: the C64 palette, multicolor characters, hires sprites, marker drawing and PNG output. |
| `cmd/disprg` | Disassemble a `.prg` file (2-byte load address + data), optionally over an address range. |
| `cmd/tapdump` | Print a pulse-width histogram and the pause-delimited segment map of a `.tap` — the usual first look at an unknown tape. |

## Building and running

The `go.work` workspace lets the game tools resolve the local `c64tools`
module. Build and test each module from its own directory:

```sh
( cd c64tools && go test ./... )
( cd "Elite (C64)/extract" && go test ./... )
( cd "Fort Apocalypse (C64)/extract" && go test ./... )
```

(The integration tests skip automatically when the `.tap` image is absent.)
The shared module's import path is `stupidcoder.com/c64tools`, so its commands
are run by that full path from anywhere in the workspace (the `go.work` is
found by walking up from the current directory), e.g.
`go run stupidcoder.com/c64tools/cmd/tapdump ...`.

Inspect an unknown tape, then extract a game:

```sh
go run stupidcoder.com/c64tools/cmd/tapdump "Elite (C64)/Elite.tap"

cd "Elite (C64)/extract" && go build && ./extract -o ../extracted ../Elite.tap

cd "Fort Apocalypse (C64)/extract" && go build && \
    ./extract -o ../extracted -dis ../Fort_Apocalypse.tap
```

Disassemble any extracted file with the shared tool:

```sh
go run stupidcoder.com/c64tools/cmd/disprg -start 8927 -end 8A40 \
    "Fort Apocalypse (C64)/extracted/FORT-fast-7000.prg"
```

## Two extraction strategies

The two tapes need fundamentally different approaches, and the shared tools
support both:

- **Declarative decoding** (Fort Apocalypse). The fastloader is static: a
  fixed Novaload-family format with page records and checksums. The extractor
  reimplements that format on top of `tap` + `cbmtape` and reads the payload
  straight off the pulse stream.

- **Run-the-loader emulation** (Elite). The fastloader rewrites its own wire
  format on the fly — bit order, bits-per-byte, header size and sync handling
  all change mid-load, driven by patch blocks that arrive on the tape itself.
  Reimplementing the protocol is hopeless, so the extractor *runs the actual
  loader* on the `c64/` + `mos6502` machine model, feeding it the tape pulses
  and logging every memory write the loader performs.

For a new C64 tape the workflow is: `tapdump` to see the encoding, `cbmtape`
to read the boot file, `disprg` to disassemble the loader, then choose a
strategy — a static loader is a new decoder package; a hostile self-modifying
one is a set of hooks on the `c64` machine model.
