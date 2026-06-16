"""Sonic (Game Gear) — gameplay drawing, decompiled.

This is the payoff of translating instead of byte-correlating: the level-map format
that resisted "guess the stride" reverse-engineering simply reads off the code.

gp_vdp_update ($0130) runs each gameplay frame. It pages banks 8/9 to stream new
tiles (tile_stream $31BC), then restores banks 1/2 and, if the camera moved, calls
scroll_draw to draw the newly-revealed map. The key fact the code makes obvious:
scroll_draw blits ONE pre-expanded 2x2 block (8 bytes = four name-table cells, each
tile+attr) from the pointer at $D2AF — it does not expand blocks itself, and with
bank 1 paged into slot 1 the block data lives in bank 1, name-table-ready. (So the
"layout" the map is read from is a stream of these 8-byte expanded blocks, which is
why no block-index stride reproduced the screen.)
"""

from machine import mem, vdp, memw, page, WRITE_VRAM


def gp_vdp_update():                            # $0130 — per-frame gameplay VDP update
    page(1, 8); page(2, 9)                       # banks 8/9: the tile data
    mem.slot1_bank = 8; mem.slot2_bank = 9
    if mem.iy_bit(7, 7):
        tile_stream()                            # $31BC — stream new 3bpp level tiles
    page(1, 1); page(2, 2)                       # back to the running banks
    mem.slot1_bank = 1; mem.slot2_bank = 2
    if mem[0xD2AC] & 0x80 == 0:                  # camera moved this frame?
        scroll_draw()                            # $3282
    mem[0xD2AC] = 0xFF
    vdp.write_reg(1, mem.vdp_reg1_shadow)        # re-assert display/VDP reg 1
    mem.set_iy_bit(0, 0, True)


def scroll_draw():                              # $3282 — draw one revealed 2x2 block
    # Horizontal: how far (in 8-px units) has the camera X crossed since last draw?
    col = (memw(0xD2AB) & 0xFFF8) - (memw(0xD254) & 0xFFF8)
    if col < 0 or col > 0xFF or col < 0x08:      # behind / too far / <1 block -> nothing
        return
    # name-table column = ((camera_x_aligned + scrollX) / 8) mod 32, doubled (2 bytes/cell)
    x = (col + (mem[0xD24B] & 0xF8)) >> 3
    nt_col = (x & 0x1F) * 2

    # Vertical: same test on the camera Y.
    row = (memw(0xD2AD) & 0xFFF8) - (memw(0xD257) & 0xFFF8)
    if row < 0 or row > 0xFF or row >= 0xC0:
        return
    y = (row + (mem[0xD24C] & 0xF8)) >> 3
    if y >= 0x1C:                                # wrap at 28 rows
        y -= 0x1C

    # VRAM name-table address of the block's top-left cell ($3800 + y*$40 + nt_col).
    addr = 0x3800 + (y * 0x40) + nt_col

    # Blit the 2x2 block: two rows, each four bytes (= two cells, tile+attr) copied
    # straight from the source at $D2AF. This is the whole "map draw" — the block is
    # already in name-table form; $D2AF was pointed at it by the column streamer.
    src = memw(0xD2AF)
    for _ in range(2):                           # B = 2 rows
        vdp.set_addr(addr, WRITE_VRAM)
        for _ in range(4):                       # 4 bytes = 2 cells
            vdp.write_data(mem[src]); src += 1
        addr += 0x40                             # next name-table row


def advance_draw_x():                           # $29AC-$2A1D — horizontal scroll catch-up
    """Move the 'drawn-up-to' X ($D254) toward the camera ($D3FF) by <= 8 px this frame.

    This is what answers "why every 8 px, not 16?": the engine doesn't step block by
    block. Each frame it nudges the drawn position toward the camera by at most one tile
    (8 px) — clamped to 8 unless a fine-scroll flag asks for 1 — and scroll_draw then
    blits the 2x2 block straddling the freshly-exposed tile edge. So the *draw cadence*
    is the 8-px catch-up step (tile resolution, capped to bound per-frame VRAM work),
    while the 16-px macro-block is just the *source unit* $D2AF points at. A given block
    is therefore touched at both of its tile edges — a small redundancy traded for not
    having to track block parity against the camera.
    """
    camera = memw(0xD3FF)
    drawn = memw(0xD254)
    fine = mem.iy_bit(5, 5)                      # BIT 5,(IY+5): step 1 px instead of 8
    if drawn + memw(0xD259) < camera:            # camera is ahead -> scroll right
        gap = camera - (drawn + memw(0xD25B))
        if gap <= 0:
            return
        step = 1 if fine else min(gap, 0x08)
        mem_setw(0xD254, drawn + step)           # catch up by <= 8 px
    elif drawn > camera:                         # camera is behind -> scroll left
        gap = drawn - camera
        step = 1 if fine else min(gap, 0x08)
        mem_setw(0xD254, drawn - step)


def mem_setw(a, v):
    mem[a] = v & 0xFF
    mem[a + 1] = (v >> 8) & 0xFF


def tile_stream():
    pass    # $31BC — stream level tiles (3bpp -> 4bpp); separate from the map draw


# ---------------------------------------------------------------------------
# The block-definition table (decoded; bank 1 $7B99)
# ---------------------------------------------------------------------------
# scroll_draw blits a block from $D2AF; the only routine that points $D2AF at one
# ($7B42, LD HL,$7B99) shows the format:  $D2AF = BLOCK_DEFS + index*8.
BLOCK_DEFS = 0x7B99     # z80 addr with bank 1 in slot 1 (= file $07B99)

def block_addr(index):
    """Address of macro-block `index` in the block-definition table.

    A block is 8 bytes = a 2x2 of name-table cells, ROW-MAJOR, each cell two bytes
    (tile, attr): (TL_t,TL_a, TR_t,TR_a, BL_t,BL_a, BR_t,BR_a). Verified against the
    live VRAM: block 0 = 00*8 (sky); block 1 = F0 00 F1 00 E2 00 F2 00 (a cloud).
    """
    return BLOCK_DEFS + index * 8


# ===========================================================================
# WALL — the terrain block-index MAP encoding (Part III/IV frontier)
# ---------------------------------------------------------------------------
# Decoded so far (verified): the 16-wide 4bpp tile sheet; the block-def table above
# (and a compact 4-byte/block copy at bank 4 $10000); scroll_draw blitting one block;
# the 8-px catch-up cadence (advance_draw_x); and that the map is SPARSE (block 0 sky
# dominates) with objects (trees/flowers) drawn separately and a fixed parallax bg
# pattern ($7BC1).
#
# NOT cracked: where the per-position terrain block INDICES are stored in ROM and how a
# column is addressed. Every $D2AF-setter traced ($7B42 bg pattern, $73BB/$8432 objects)
# and the level-descriptor pointers (+13..+19) lead to sub-systems or dense tile/def
# data, not a sparse block-index grid. The two ways to finish are blocked: the one oracle
# capture is sky-heavy (sky runs match everywhere, defeating correlation), and the
# simplified machine can't scroll to get a ground-heavy capture (player physics don't
# advance). So this last layer needs either a higher-fidelity oracle or a deep trace of
# the level-load full-screen terrain draw — a genuine wall for the static+single-capture
# method used here.
# ===========================================================================
