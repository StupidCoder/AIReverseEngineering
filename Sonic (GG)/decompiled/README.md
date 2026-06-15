# Sonic (Game Gear) — Python decompilation (experiment)

An experiment in translating the game's Z80 code into readable Python, function by
function from the entry point, instead of only annotating the disassembly. The bet
(see Sonic.md Part III discussion) is that concise pseudocode over a shared machine
model reveals structure — e.g. *which* of two similar screens is shown *when* — more
reliably than signature-hunting for assets, and that a translation is **falsifiable**
(it runs; wrong logic shows up) where an annotation is not.

## Layout

- `machine.py` — the global state: work RAM (`mem`), the VDP (`vdp`), the mapper, and
  the few CPU flags routines communicate through (`flags`). Plus the helpers a
  decompiler lifts from recognised idioms (a VDP register load, an LDIR clear, a port
  fill). `NAMES` is the live symbol table for RAM locations; it grows as labels land.
- `boot.py` — the translated routines, each tagged with its source address, starting
  at `reset()` (`$0000`) and working outward. A `WORKLIST` tracks discovered-but-
  untranslated callees, split into **frontier** stubs (raise when called — the edge of
  the translated region) and **no-op** stubs (modelled as nothing for now so the spine
  still runs end to end).

## Conventions

- Untranslated routines are named by address, **prefixed with their ROM bank** so the
  same Z80 address in two banks can't collide: `b3_call_4006`, `b1_sub_42DA`, `sub_0645`
  (bank 0). Renamed to a meaningful label once understood (`scene_dispatch`, `load_title`).
- State is global, mirroring the hardware; routines take no register arguments.
- Idioms are lifted, not transcribed: the 11-instruction VDP-register loop in `init`
  becomes `vdp_load_regs(table=0x031C, count=11, shadow=...)`, with a `# $02B7` tag.

## Status

Runs from the entry point through the screen loaders to the scene interpreter:

```
$ python3 boot.py
... reset -> init -> main_entry -> attract_loop -> scene_dispatch
    -> load_title (decompress + nt_load_rle run) -> ... -> scene_run $1414  (frontier)
```

Translated: `reset $0000`, `init $0296`, `main_entry $1356`, `attract_loop $13C5`,
`scene_dispatch $0BDD`, `load_title $0C20`, `load_worldmap $0C7A`, `finish_screen`/
`draw_scene_overlay`. The two graphics codecs (`decompress` = `$0406`, `nt_load_rle` =
`$0502`) are real Python in `machine.py`, so the loaders actually fill VRAM.

**Validated by execution** (`render.py`): running the *translated* `load_worldmap`
on the real ROM and rendering `vdp.vram` reproduces the zoomed world map pixel-for-
pixel against the Go `scenemap` output — proof the translation is correct, not just
plausible. That is the payoff of translating over annotating.

The bank-aware disassembler (`tools/cmd/disz80 -slots`) backs further translation into
the banked routines (the `b3_*` dispatcher, the level loaders).

## What the experiment has already shown

- **It runs**, and the spine reaches the scene dispatcher, correctly routing scene 0
  (a title scene) to `load_title`.
- **Translation catches errors annotation hides.** Writing `scene_dispatch` out showed
  `$0BFB` sets `$D217 = $FF` *unconditionally* every call, so the "only reload the
  background when the screen type changes" note in the annotations was wrong from this
  entry (the skip only happens via the alternate entry `$0C00`). The code made the
  mistake obvious; prose had glossed it.

## Friction encountered (the honest cost)

- **Flag-as-return.** Some routines communicate via the carry flag (`b1_sub_42DA` →
  `(IY+5).1`). Modelled with a global `flags.carry`; faithful but a thing to track.
- **Gotos into loop bodies.** The attract loop's "re-run this scene" path jumps back
  into the middle of the loop (`$13E4`); rendered as a nested `while`, readable but not
  a literal 1:1.
- **Banking.** Cross-bank calls need bank-qualified names and (eventually) a bank-aware
  disassembler; here they are no-op stubs until that path is traced with the bank paged.
