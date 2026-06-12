# AI Reverse Engineering

Are the latest LLMs smart enough to reverse engineer old
software purely by static analysis, without access to a 3rd party debugger? This
repository contains the results of a few tests, attempting to answer that
question. All code and documents contained within (except this introduction) were
written by Claude Fable 5 or Opus 4.8. In some cases my Fable session got flagged, probably due to the nature of reverse engineering, and downgraded to Opus. Opus still did a fine job though.

Even though I explicitly restricted internet searches (released source code or other reverse engineering projects) Claude had detailed knowledge if the games I've had it analyze so far. When I asked how it knew exact function names for Elite it admitted that the original source code of the game, which has been publically released years ago, was part of its training data. The same is true for character and level names of Fort Apocalypse, for example.

Some of the prompts I used (for reference):

> Write a GoLang application that extracts program files from a C64 tape image
> in TAP format. The image in question might use a non-standard fastloader,
> that you need to reverse engineer by disassembling the loader. Use only
> static analysis of the image file without looking up external sources or
> external tools like Vice. Ignore memories from previous reverse engineering
> sessions. Document your findings about the tape image format in general and
> the fastloader in particular in a markdown file. Use example byte sequences
> in that documentation where appropriate. The tape image file in question is
> <GAME.TAP> in the current folder.

> Now that the program has been extracted from the tape, I want to enhance the scope of the project. Using the disassembler you already wrote, analyze the extracted program's startup and document your findings in a markdown file, again using assembly snippets or byte sequences as example where appropriate.

> Next, analyze the actual game code in the same way, again producing a markdown file that describes the initialization of the game until it reaches the main game loop. Put extra focus on the graphical elements like character sets, sprite data and level map. Describe compression or encryption schemes you find and build a memory map.

> Analyze the game code and describe the different types of objects (player, enemies, obstacles) in the game and how they behave in a markdown file. Be as detailed as possible about each object's behavior, movement patterns and collision behaviors.

To do:
* Elite (C64)
    * Part IV
        * List all strings, including missiong briefings
        * Try to reproduce exact system naming of initial galaxy (Diso, Leesti, Lave, etc.)
* Fort Apocalypse (C64)
    * Run new disassembler and add known annotations
    * Part IV
        * Visualize radar map and objects (tank, SPM, etc.)
* Marble Madness (Amiga)
    * Kick off project by writing ADF extractor, disassembler and emulator for 86k code
* Tools
    * Disassembler should be better at segmenting functions; currently jumps within a function are treated as separate sub-routines; try to document parameters of sub-routines (which registers are used?)

## Repository structure

Each game lives in its own `<Game> (<Platform>)/` folder following a common
contract, and the reusable building blocks live in a shared `tools` module at
the root. Platform-neutral tools sit directly under `tools/`; platform-specific
ones live in a per-platform subfolder (`tools/c64/` today; `tools/amiga/`,
`tools/snes/`, … as other games are added). A `go.work` ties the modules
together.

```
AIReverseEngineering/
├── go.work                     # Go workspace over tools + each game's extract/
├── README.md                   # this file
├── tools/                      # shared tooling (module stupidcoder.com/tools)
│   ├── mos6502/                #   6502 disassembler + executable CPU core (any 6502 platform)
│   ├── m68k/                   #   Motorola 68000 disassembler (any 68k platform)
│   ├── cmd/disprg/             #   linear disassembler for a .prg file (6502)
│   ├── cmd/dis68k/             #   linear disassembler for a raw 68000 blob
│   ├── cmd/codetrace/          #   recursive-descent disassembler (code/data separation)
│   ├── c64/                    #   C64-specific tools
│   │   ├── tap/                #     TAP container parser + segmentation
│   │   ├── cbmtape/            #     standard KERNAL ROM tape-format decoder
│   │   ├── c64/                #     machine model for running hostile loaders
│   │   ├── gfx/                #     palette, char/sprite/bitmap rendering, lines, PNG/APNG
│   │   └── cmd/tapdump/        #     pulse histogram + segment listing for a .tap
│   └── amiga/                  #   Amiga-specific tools
│       ├── adf/                #     AmigaDOS floppy image (ADF) reader — OFS/FFS
│       └── cmd/adfdump/        #     list and extract files from an .adf
│
├── Elite (C64)/
│   ├── Elite.tap               # raw tape image
│   ├── Elite.md                # tape + loader + startup writeup (more to follow)
│   ├── extract/                # module elite/extract — extraction + graphics tools
│   ├── extracted/              # generated .prg files (regenerable; git-ignored)
│   ├── rendered/               # generated PNGs (loading screen, ship wireframes)
│   └── disasm/                 # annotated recursive-descent disassembly + annotations.txt
│
└── Fort Apocalypse (C64)/
    ├── Fort_Apocalypse.tap      # raw tape image
    ├── Fort_Apocalypse.md       # full game + tape writeup
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

## Image files

Many differing dumps of these games circulate online. All results in this
repository were produced from these exact image files; the documentation and
the golden extraction tests assume them byte for byte. The MD5 (and size)
below pin the precise copy, so the work stays reproducible.

| Image | Size (bytes) | MD5 |
|-------|-------------:|-----|
| `Elite (C64)/Elite.tap` | 801,592 | `d51b7f84fd1bec6eb24f4bf210c8cc74` |
| `Fort Apocalypse (C64)/Fort_Apocalypse.tap` | 225,817 | `bec7409816865f3ad160af9984f127cd` |

Verify a copy before reusing it, e.g. `md5 "Elite (C64)/Elite.tap"`
(`md5sum` on Linux).

## Shared tools (`tools`)

Platform-neutral packages sit at the top level; platform-specific ones live in a
per-platform subfolder (`c64/`, `amiga/`, …).

| Package / command | What it does |
|-------------------|--------------|
| `mos6502` | One opcode table driving both a `Disassemble` function and an executable `CPU` core (all documented opcodes, binary + BCD) — usable by any 6502 platform. |
| `m68k` | Motorola 68000 disassembler with the same surface as `mos6502`: a `Decode` returning one classified instruction (length, text, and a `Flow` category for recursive-descent tracing) plus a `Disassemble` helper. Covers the documented 68000 instruction set and all addressing modes; usable by any 68k platform (Amiga, ST, Genesis, …). |
| `cmd/disprg` | Linear disassembler for a `.prg` file (2-byte load address + data), optionally over an address range. |
| `cmd/dis68k` | Linear disassembler for a raw 68000 code blob loaded at a given base address (`-skip` steps past an AmigaDOS hunk header). |
| `cmd/codetrace` | Recursive-descent disassembler: from given entry points (and seeded jump tables) it follows every branch/jump/call, marks reachable code vs data, lists routines and unresolved indirect jumps — so tables and graphics aren't mis-decoded as instructions. |
| `c64/tap` | Parse a TAP v0/v1 image (C64/C16) into a pulse stream; `Segmentize` splits it at pauses. |
| `c64/cbmtape` | Decode the standard Commodore KERNAL (ROM loader) tape encoding: blocks, headers, and paired header+data files with checksum verification. |
| `c64/c64` | A minimal C64 machine model — RAM, the `mos6502` CPU, a CIA pulse-feed tape model, a PC-hook registry, a RAM write log and an optional read probe — for *running* a self-modifying loader instead of decoding it, or tracing which game routine touches which memory. Optional standard KERNAL tape hooks included. |
| `c64/gfx` | C64/VIC rendering (palette, multicolor characters, hires sprites, multicolor bitmaps) plus general 2-D helpers (line drawing, markers, still/animated PNG output). |
| `c64/cmd/tapdump` | Print a pulse-width histogram and the pause-delimited segment map of a `.tap` — the usual first look at an unknown tape. |
| `amiga/adf` | Read a standard AmigaDOS floppy image (ADF): detect OFS/FFS, walk the directory tree, and extract file contents (handles hash chains, OFS data-block headers and multi-block file-extension chains). |
| `amiga/cmd/adfdump` | List an `.adf`'s volume, directory tree and file sizes; `-x outdir` extracts every file preserving the directory structure. |

## Building and running

The `go.work` workspace lets the game tools resolve the local `tools`
module. Build and test each module from its own directory:

```sh
( cd tools && go test ./... )
( cd "Elite (C64)/extract" && go test ./... )
( cd "Fort Apocalypse (C64)/extract" && go test ./... )
```

(The integration tests skip automatically when the `.tap` image is absent.)
The shared module's import path is `stupidcoder.com/tools`, so its commands
are run by that full path from anywhere in the workspace (the `go.work` is
found by walking up from the current directory), e.g.
`go run stupidcoder.com/tools/c64/cmd/tapdump ...`.

`tapdump` is generic — point it at any C64 `.tap` to see its pulse encoding
and segment layout (the first step when approaching an unfamiliar tape):

```sh
go run stupidcoder.com/tools/c64/cmd/tapdump path/to/any.tap
```

`adfdump` is likewise generic for Amiga disks — it lists and extracts the files
of any standard AmigaDOS floppy image:

```sh
go run stupidcoder.com/tools/amiga/cmd/adfdump path/to/disk.adf            # list
go run stupidcoder.com/tools/amiga/cmd/adfdump -x out path/to/disk.adf     # extract
```

The `extract` tools are not generic: each one is written for its game's
specific loader, so it only runs on that game's image. Run a game's extractor
from its own folder:

```sh
cd "Elite (C64)/extract" && go build && ./extract -o ../extracted ../Elite.tap

cd "Fort Apocalypse (C64)/extract" && go build && \
    ./extract -o ../extracted -dis ../Fort_Apocalypse.tap
```

Extracting a *new* game means writing a new per-game `extract` tool on top of
`tools` (see "Two extraction strategies" below), not reusing one of these.

Disassemble any extracted file with the shared tools — `disprg` for 6502,
`dis68k` for 68000 (`-skip` steps past an Amiga hunk header):

```sh
go run stupidcoder.com/tools/cmd/disprg -start 8927 -end 8A40 \
    "Fort Apocalypse (C64)/extracted/FORT-fast-7000.prg"

go run stupidcoder.com/tools/cmd/dis68k -skip 36 amiga-code-hunk.bin
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
  loader* on the `c64/c64` + `mos6502` machine model, feeding it the tape
  pulses and logging every memory write the loader performs.

For a new C64 tape the workflow is: `tapdump` to see the encoding, `cbmtape`
to read the boot file, `disprg` to disassemble the loader, then choose a
strategy — a static loader is a new decoder package; a hostile self-modifying
one is a set of hooks on the `c64/c64` machine model.
