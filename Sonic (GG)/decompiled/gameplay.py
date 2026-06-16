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


def tile_stream():
    pass    # $31BC — stream level tiles (3bpp -> 4bpp); separate from the map draw
