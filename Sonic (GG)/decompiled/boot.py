"""Sonic the Hedgehog (Game Gear) — decompiled boot spine.

Translated function-by-function from the entry point ($0000) outward. Each routine
is tagged with its source address. Calls to routines not yet translated are real
Python calls to stubs at the bottom; the WORKLIST tracks what is still to do.

Run:  python3 -c "import boot; boot.reset()"   (will stop at the first stub)
"""

from machine import (
    mem, vdp, flags, load_rom, u16,
    di, ei, im, set_sp, page, mapper_control, vcounter,
    mem_fill, copy_rom, vdp_load_regs, vdp_fill, display, decompress, nt_load_rle,
)

# The scene table lives in bank 5 at Z80 $5600; with bank 5 paged into slot 1 that is
# flat file offset $15600.  Each entry is a word: the offset (from $5600) of a 40-byte
# scene descriptor.
SCENE_TABLE = 5 * 0x4000 + (0x5600 - 0x4000)   # = $15600

# ---------------------------------------------------------------------------
# CPU entry points (Part I §5 / Part II)
# ---------------------------------------------------------------------------

def reset():                                    # $0000
    di()
    im(1)
    while vcounter() != 0xB0:                    # wait for raster line $B0
        pass
    init()                                       # $0296


def init():                                     # $0296 — cold start
    # Program the Sega mapper: control = $80, slots 0/1/2 <- banks 0/1/2.
    mapper_control(0x80)
    page(0, 0); page(1, 1); page(2, 2)
    # Clear work RAM $C000..$DFEF (overlapping LDIR) and park SP just below it.
    mem_fill(0xC000, 0x00, 0x1FF0)
    set_sp(0xDFEF)
    # Load VDP registers 0..10 from the table at $031C, mirroring them to $D219.
    vdp_load_regs(table=0x031C, count=11, shadow=mem_addr("vdp_regs_shadow"))
    # Hide all 64 sprites: fill the SAT Y column ($3F00) with the off-screen $E0.
    vdp_fill(0x3F00, 0x40, 0xE0)
    b3_call_4006()                               # banked setup call ($02F8 -> bank 3)
    set_iy(0xD200)                               # IY = the game-state block base
    main_entry()                                 # $1356


# ---------------------------------------------------------------------------
# Main entry and the attract loop (Part III §1)
# ---------------------------------------------------------------------------

def main_entry():                               # $1356
    mem.set_iy_bit(0, 0, True)
    ei()
    set_slot(1, 1); set_slot(2, 2)               # running banks for slots 1 and 2
    mem.set_iy_bit(2, 0, False); mem.set_iy_bit(2, 1, False)
    sub_0645()                                   # init the sprite display-list buffer
    sega_logo()                                  # $1CD7 — show the SEGA logo
    palette_fade()                               # $0AA3 — fade it in
    mem.game_mode = 3
    mem[0xD2F8] = 5
    mem[0xD239] = 0x1C
    mem.scene = 0                                # start of the attract sequence
    mem[0xD224] = 0
    mem.set_iy(13, 0)
    mem_fill(0xD279, 0x00, 8)                    # clear the per-zone progress counters
    mem_fill(0xD200, 0x00, 0x0E)
    mem_fill(0xD2BB, 0x00, 4)
    mem_fill(0xD306, 0x00, 0x18)
    sub_0645()
    b1_sub_42DA()                                # returns carry  ($13B8)
    mem.set_iy_bit(5, 1, not flags.carry)        # (IY+5).1 = NOT carry
    attract_loop()                               # $13C5


def attract_loop():                             # $13C5
    while True:
        if mem.scene >= 0x13:                    # past the last scene -> replay
            return restart_attract()             # $135B (= back into main_entry's tail)

        mem.set_iy_bit(2, 0, False); mem.set_iy_bit(2, 1, False)
        sub_0645()
        scene_dispatch()                         # $0BDD — load this scene's screen
        if mem.iy_bit(5, 1) and flags.carry:     # demo/replay path  ($13DB/$13DF)
            return restart_attract()

        # Re-entry point for "re-run this scene" (the r == 2 case jumps here, $13E4).
        while True:
            palette_fade()                       # $0AA3
            sub_0645()
            start_pressed = mem.iy_bit(6, 4)     # Start was pressed (skip the idle wait)
            if not mem.iy_bit(5, 0) and not start_pressed:
                wait_frames(0x3C)                # idle ~1s between attract scenes
                b3_call_4006()                   # $1401 (RST $20)
            r = scene_run()                      # $1414 -> 0 restart / 1 next / 2 re-run
            if r == 0:
                return restart_attract()
            if r == 1:
                break                            # advance to the next scene
            # r == 2: re-run this scene (loop back to the fade)


def restart_attract():                          # $135B — re-enter main_entry's body
    # main_entry falls through to here on every replay; modelled as a tail call.
    main_entry()


# ---------------------------------------------------------------------------
# The scene dispatcher (Part III §1) — maps the scene counter to a screen loader
# ---------------------------------------------------------------------------

def scene_dispatch():                           # $0BDD
    mem[0xD24B] = 0; mem[0xD24C] = 0; mem[0xD300] = 0
    if mem.iy_bit(5, 1):                         # demo/replay: pick scene from $D305 bit0
        mem.scene = 6 if (mem[0xD305] & 1) else 0
    mem[0xD217] = 0xFF                           # prev screen-type = none -> force a reload
    dispatch_screen()                            # $0C00 (alternate entry keeps $D217)


def dispatch_screen():                          # $0C00
    if mem.scene >= 0x12:                        # scenes >= $12 have no screen
        return
    kind = 2 if mem.scene >= 9 else 1            # 0..8 = title background, 9..$11 = world map
    if mem[0xD217] == kind:                      # already showing this screen type
        return draw_scene_overlay()             # $0CD9 — just repaint the per-scene overlay
    mem[0xD217] = kind
    if kind == 2:
        load_worldmap()                          # $0C7A
    else:
        load_title()                             # $0C1C
    draw_scene_overlay()                         # both fall through into the overlay


# ---------------------------------------------------------------------------
# Screen loaders (Part IV §3 / Part III §3)
# ---------------------------------------------------------------------------

def load_title():                               # $0C1C / $0C20 — the title screen
    display(False)
    decompress(0x0C, 0x0000, dest=0x0000)        # title tiles  (bank 12)
    decompress(0x09, 0x4AD0, dest=0x2000)        # sprite tiles (bank 9)
    decompress(0x09, 0xB354, dest=0x3000)
    nt_load_rle(5, 0x6962, 0x019B, dest=0x3800, hi=0x10)   # stored map, priority layer
    nt_load_rle(5, 0x6AFD, 0x0170, dest=0x3800, hi=0x00)   # overlay layer
    palette_fade_to(0x0B0A)
    finish_screen()                              # $0CD6 — shared with the world map


def load_worldmap():                            # $0C7A — the (zoomed) world map screen
    display(False)
    decompress(0x0C, 0x171A, dest=0x0000)        # map tiles  (bank 12)
    decompress(0x09, 0x51A7, dest=0x2000)
    decompress(0x09, 0xB354, dest=0x3000)
    nt_load_rle(5, 0x6C6D, 0x0156, dest=0x3800, hi=0x10)   # stored map, priority layer
    nt_load_rle(5, 0x6DC3, 0x0198, dest=0x3800, hi=0x00)   # overlay layer
    palette_fade_to(0x0D0C)                      # bg $0C / spr $0D, faded up from black
    finish_screen()                              # $0CD6


def finish_screen():                            # $0CD6 — common title/map tail
    b3_rst18(0x07)                               # banked setup
    draw_scene_overlay()                         # $0CD9


def draw_scene_overlay():                       # $0CD9 — per-scene route/zone overlay
    sub_0E23()
    mem.nt_hi = 0x10
    nt_string(scene_overlay_ptr())               # $0612 — draw this scene's text/route
    place_scene_markers()                        # $0CF3+ (per-scene marker positions)


def scene_overlay_ptr():                         # $0CDC — table $1163 indexed by scene
    return u16(0x1163 + mem.scene * 2)


# ---------------------------------------------------------------------------
# The scene interpreter (Part III §1) — runs one attract-sequence scene
# ---------------------------------------------------------------------------

def scene_run():                                # $1414 -> 0 restart / 1 next / 2 re-run
    set_slot(1, 5)                               # the scene table lives in bank 5
    i = mem[0xD2D4] if mem.iy_bit(6, 4) else mem.scene   # Start jumps to a target scene
    off = u16(SCENE_TABLE + i * 2)               # this scene's descriptor offset
    if off == 0:
        return run_special_scene()              # $1FAE (no descriptor)
    return run_scene_descriptor(SCENE_TABLE + off)   # $185D


def run_scene_descriptor(desc):                 # $185D
    display(False)
    copy_rom(desc, 0xD355, 0x28)                 # the 40-byte descriptor -> work RAM $D355
    mem.set_iy(11, mem.iy(5)); mem.set_iy(12, mem.iy(6))  # snapshot the flag bytes
    init_scene_state()                           # $1884.. clear a swathe of scene RAM
    # The scene's actual behaviour — what it draws and how long it runs — is driven by
    # the 40 descriptor bytes now at $D355 and a per-scene script, branching on $D238.
    # That is DATA, not code: see the note in the worklist. << the wall
    return run_scene_behaviour(desc)


# ===========================================================================
# WORKLIST / FRONTIER NOTE — the data-driven wall (Part III §1)
# ---------------------------------------------------------------------------
# scene_run reaches a clean, readable shell, but the scene's *behaviour* is encoded
# as a 40-byte descriptor (now copied to $D355) plus a per-scene script, interpreted
# by the rest of $185D and the per-frame handlers. The 40 fields are not yet decoded
# (that needs tracing how $D355+n is consumed across the scene handlers). So the
# descriptor stays raw bytes for now and run_scene_behaviour is the frontier — this is
# exactly the predicted "logic lives in data" boundary, not a control-flow dead end.
#   Next, to push past it: decode the $D355 descriptor fields into a Scene dataclass.
# ===========================================================================


# ---------------------------------------------------------------------------
# helpers that wrap a couple of two-step idioms for readability
# ---------------------------------------------------------------------------

def mem_addr(name):
    from machine import NAMES
    return NAMES[name]

def set_iy(base):
    # IY is fixed at $D200 for the whole game; nothing to do in this model.
    assert base == 0xD200

def set_slot(slot, bank):
    """Page a bank into a slot AND update its RAM shadow ($D22F/$D230)."""
    page(slot, bank)
    if slot == 1:
        mem.slot1_bank = bank
    elif slot == 2:
        mem.slot2_bank = bank


# ===========================================================================
# WORKLIST — discovered callees, not yet translated.
#   FRONTIER (raise on call — the edge of the translated region):
#     scene_run          $1414  — index the bank-5 scene table, run the descriptor  << next
#     place_scene_markers $0CF3 — per-scene route-marker math (the blinking route)
#     b3_rst18(7)        bank3  — banked setup via the RST $18 dispatcher
#   MODELLED (structural placeholder; refine when needed; don't block the spine):
#     palette_fade_to, nt_string, sub_0E23, sub_0645, sega_logo, palette_fade,
#     wait_frames, b3_call_4006, b1_sub_42DA
# ===========================================================================

def _todo(addr, note=""):
    raise NotImplementedError(f"FRONTIER, not yet translated: {addr} {note}".rstrip())

def _noop(*_):  # an effect we haven't translated yet but that doesn't block the flow
    pass

# frontier — translating outward stops here
def run_scene_behaviour(desc):  _todo("$1885+", "per-scene script over the $D355 descriptor (data-driven)")

# modelled structurally for now (so the spine runs end to end)
def run_special_scene():    _noop()   # $1FAE: scenes with no descriptor
def init_scene_state():     _noop()   # $1884+: clear the scene RAM block
def place_scene_markers():  _noop()   # $0CF3: per-scene route-marker math (blinking route)
def b3_rst18(fn):           _noop()   # bank3 RST$18 dispatch: banked setup
def palette_fade_to(src):   _noop()   # $0AAB: load + fade the palette toward its target
def nt_string(src):         _noop()   # $0612: name-table string/run blitter
def sub_0E23():             _noop()
def sub_0645():             _noop()
def sega_logo():            _noop()
def palette_fade():         _noop()
def wait_frames(n):         _noop()
def b3_call_4006():         _noop()
def b1_sub_42DA():          setattr(flags, "carry", False)  # placeholder demo/replay flag


if __name__ == "__main__":
    load_rom()
    reset()
