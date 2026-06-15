"""Render the machine model's VRAM to a PNG — the validation harness for the
decompilation. After running a translated screen loader, this decodes what it left
in `vdp.vram`/`vdp.cram` exactly as the hardware would and writes an image, so a
translation can be checked against the real game (or the Go `scenemap` renders)
rather than merely read and trusted. This is the "translation is falsifiable" claim
in practice.

Usage:
    import machine, boot, render
    machine.load_rom(); boot.load_worldmap()
    render.screen("/tmp/out.png", bg_palette=0x0C)
"""

from PIL import Image
import machine


def _palette(idx):
    """16 RGB colours from the bank-8 palette table (file $23400)."""
    rom = machine.rom
    tab = 0x23400
    ptr = rom[tab + idx * 2] | rom[tab + idx * 2 + 1] << 8
    off = tab + ptr
    out = []
    for i in range(16):
        w = rom[off + 2 * i] | rom[off + 2 * i + 1] << 8
        out.append(((w & 0xF) * 0x11, (w >> 4 & 0xF) * 0x11, (w >> 8 & 0xF) * 0x11))
    return out


def _tile(vram, n):
    """Decode one 4-bitplane tile from VRAM into 64 colour indices."""
    b = vram[n * 32:n * 32 + 32]
    px = []
    for y in range(8):
        p0, p1, p2, p3 = b[y * 4:y * 4 + 4]
        for x in range(8):
            s = 7 - x
            px.append((p0 >> s & 1) | (p1 >> s & 1) << 1 | (p2 >> s & 1) << 2 | (p3 >> s & 1) << 3)
    return px


def screen(path, bg_palette, crop_gg=True, scale=3):
    """Compose the name table ($3800) over the tiles, write a PNG. If crop_gg, crop
    to the Game Gear's visible 160x144 window; else the full 256x224 name table."""
    vram = machine.vdp.vram
    pal = _palette(bg_palette)
    img = Image.new("RGB", (256, 224))
    for ty in range(28):
        for tx in range(32):
            o = 0x3800 + (ty * 32 + tx) * 2
            t = vram[o] | (vram[o + 1] & 1) << 8
            px = _tile(vram, t)
            for y in range(8):
                for x in range(8):
                    img.putpixel((tx * 8 + x, ty * 8 + y), pal[px[y * 8 + x]])
    if crop_gg:
        img = img.crop((48, 24, 48 + 160, 24 + 144))
    img = img.resize((img.width * scale, img.height * scale), Image.NEAREST)
    img.save(path)
    return path
