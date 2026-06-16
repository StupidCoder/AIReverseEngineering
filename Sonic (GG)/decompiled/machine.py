"""A machine model for the Sonic (Game Gear) decompilation experiment.

The Z80 code is translated into Python function-by-function (see boot.py). Rather
than thread registers through every call, the program's state is *global* — exactly
as it is on the hardware — and lives here: work RAM, the VDP, the cartridge mapper,
and the handful of CPU flags routines actually communicate through.

Naming conventions for the translation:
  * Routines not yet understood keep an address name, prefixed by their ROM bank so
    the same Z80 address in two banks never collides:  b3_sub_4006, sub_0645 (bank 0).
    As a routine's purpose is identified it is renamed (sega_logo, scene_dispatch...).
  * Named RAM locations live in the NAMES table below — that table *is* the symbol
    list, and it grows as labels are identified (the "labelling flywheel").
  * Recognised asm idioms (a VDP register load, an LDIR clear, a port fill) are lifted
    to a named helper here instead of being transcribed instruction-by-instruction.

This is a readability model, not a cycle-accurate emulator: timing, exact flag
side effects and banking arithmetic are modelled only as far as the logic needs.
"""

import os

# --- the cartridge ROM ------------------------------------------------------

rom = b""

def load_rom(path=None):
    global rom
    if path is None:
        path = os.path.join(os.path.dirname(__file__), "..",
                            "Sonic The Hedgehog (Japan, USA).gg")
    with open(path, "rb") as f:
        rom = f.read()
    return rom

def u16(off):  # little-endian word from the ROM
    return rom[off] | rom[off + 1] << 8

def memw(a):   # little-endian word from work RAM
    return mem[a] | mem[a + 1] << 8

# --- named work-RAM locations (the symbol table; extend as labels are found) -

NAMES = {
    "vdp_regs_shadow": 0xD219,   # 11-byte mirror of VDP registers 0..10
    "vdp_reg1_shadow": 0xD21A,   # display-enable / frame-int live here
    "bg_pal_index":    0xD22C,   # background palette index
    "spr_pal_index":   0xD22D,   # sprite palette index
    "slot1_bank":      0xD22F,   # bank currently paged into slot 1
    "slot2_bank":      0xD230,   # bank currently paged into slot 2
    "scene":           0xD238,   # attract-sequence scene counter (0..$12)
    "game_mode":       0xD240,   # a state byte (NOT the top-level mode)
    "nt_hi":           0xD20F,   # shared high byte for name-table writes
    "level_countdown": 0xD279,   # counts down toward the final stage (map zoom)
}

class Mem:
    """8 KB work RAM ($C000-$DFFF), addressable by Z80 address or by symbol name."""

    def __init__(self):
        object.__setattr__(self, "ram", bytearray(0x2000))

    # raw access by absolute Z80 address
    def __getitem__(self, a):
        return self.ram[a & 0x1FFF]

    def __setitem__(self, a, v):
        self.ram[a & 0x1FFF] = v & 0xFF

    # access named locations as attributes:  mem.scene, mem.level_countdown
    def __getattr__(self, name):
        a = NAMES.get(name)
        if a is None:
            raise AttributeError(name)
        return self.ram[a & 0x1FFF]

    def __setattr__(self, name, v):
        a = NAMES.get(name)
        if a is None:
            object.__setattr__(self, name, v)
        else:
            self.ram[a & 0x1FFF] = v & 0xFF

    # The game keeps a state block at $D200 that IY points at; flags live in its
    # low bytes as (IY+n).bit.  Until each bit's meaning is known it is referred to
    # by position, e.g. mem.iy_bit(6, 4) is "Start pressed".
    def iy(self, n):
        return self.ram[(0xD200 + n) & 0x1FFF]

    def set_iy(self, n, v):
        self.ram[(0xD200 + n) & 0x1FFF] = v & 0xFF

    def iy_bit(self, n, b):
        return (self.iy(n) >> b) & 1

    def set_iy_bit(self, n, b, on):
        v = self.iy(n)
        self.set_iy(n, (v | (1 << b)) if on else (v & ~(1 << b)))

# --- the VDP (video chip): VRAM, palette RAM, registers ---------------------

WRITE_VRAM, READ_VRAM, REGISTER, WRITE_CRAM = 0x40, 0x00, 0x80, 0xC0

class VDP:
    def __init__(self):
        self.vram = bytearray(0x4000)
        self.cram = bytearray(0x40)
        self.regs = [0] * 16
        self.addr = 0
        self.to_cram = False

    def write_reg(self, n, val):
        self.regs[n] = val & 0xFF

    def set_addr(self, addr, code):
        self.addr = addr & 0x3FFF
        self.to_cram = (code == WRITE_CRAM)

    def write_data(self, v):
        if self.to_cram:
            self.cram[self.addr & 0x3F] = v & 0xFF
        else:
            self.vram[self.addr] = v & 0xFF
        self.addr += 1

# --- the cartridge mapper and a few CPU flags -------------------------------

class Mapper:
    def __init__(self):
        self.control = 0
        self.slot = [0, 1, 2]   # ROM bank paged into each 16 KB slot

class Flags:
    """The CPU condition flags that translated routines communicate through."""
    carry = False
    zero = False

# --- the global machine -----------------------------------------------------

mem = Mem()
vdp = VDP()
mapper = Mapper()
flags = Flags()

# interrupt / SP state, modelled loosely (the logic rarely depends on it)
class _CPU:
    iff = False
    im = 1
    sp = 0xDFF0
cpu = _CPU()

# --- primitives and lifted idioms -------------------------------------------

def di():       cpu.iff = False
def ei():       cpu.iff = True
def im(n):      cpu.im = n
def set_sp(v):  cpu.sp = v

def page(slot, bank):
    """Page a ROM bank into a 16 KB slot (mapper register $FFFD/E/F)."""
    mapper.slot[slot] = bank

def mapper_control(v):
    mapper.control = v

def vcounter():
    """Read the VDP vertical-line counter (port $7E); the boot polls it."""
    return 0xB0   # the value the reset loop waits for

def mem_fill(addr, val, count):
    """Fill `count` bytes of work RAM from `addr` with `val`  (the $140F idiom)."""
    for i in range(count):
        mem[addr + i] = val

def copy_rom(src_off, dest_z80, count):
    """Copy `count` ROM bytes (flat file offset) into work RAM (the LDIR idiom)."""
    for i in range(count):
        mem[dest_z80 + i] = rom[src_off + i]

def vdp_load_regs(table, count, shadow):
    """Load VDP registers 0..count-1 from a ROM table, mirroring to `shadow`  ($02B7)."""
    for i in range(count):
        val = rom[table + i]
        mem[shadow + i] = val
        vdp.write_reg(i, val)

def vdp_fill(addr, count, val):     # $05F0
    """Write `val` to `count` consecutive VRAM bytes from `addr`."""
    vdp.set_addr(addr, WRITE_VRAM)
    for _ in range(count):
        vdp.write_data(val)

def display(on):
    """Toggle the display-enable bit (VDP reg 1 bit 6) via its RAM shadow $D21A."""
    s = mem.vdp_reg1_shadow
    mem.vdp_reg1_shadow = (s | 0x40) if on else (s & ~0x40)
    vdp.write_reg(1, mem.vdp_reg1_shadow)

def decompress(bank, addr, dest):   # $0406 — the 4-byte-unit LZ tile codec
    """Inflate the block at (bank, addr) and stream it to VRAM from `dest`."""
    while addr >= 0x4000:           # normalise so addr can span banks
        addr -= 0x4000; bank += 1
    src = bank * 0x4000 + addr
    word1, word2, count = u16(src + 2), u16(src + 4), u16(src + 6)
    ctrl, match, lit, litp = src + 8, src + word1, src + word2, src + word2
    vdp.set_addr(dest, WRITE_VRAM)
    for i in range(count):          # one 4-byte tile row per unit
        if rom[ctrl + i // 8] & (1 << (i & 7)):     # match: back-reference
            b = rom[match]; match += 1
            if b >= 0xF0:
                off = (((b - 0xF0) << 8) | rom[match]) * 4; match += 1
            else:
                off = b * 4
            for k in range(4):
                vdp.write_data(rom[lit + off + k])
        else:                                       # literal
            for k in range(4):
                vdp.write_data(rom[litp + k])
            litp += 4

def nt_load_rle(bank, addr, count, dest, hi):   # $0502 — RLE name-table codec
    """Stream a stored name-table map (bank-5 region) to VRAM; each entry is (tile, hi)."""
    src = bank * 0x4000 + (addr - 0x4000)         # addr is in the slot-1 window
    vdp.set_addr(dest, WRITE_VRAM)
    i, end = src, src + count
    prev = (~rom[i]) & 0xFF                        # sentinel: first byte is a literal
    while i < end:
        b = rom[i]
        if b == prev:                              # run: duplicate + a count byte
            i += 1
            if i >= end:
                break
            for _ in range(rom[i]):
                vdp.write_data(b); vdp.write_data(hi)
            i += 1
            prev = (~rom[i]) & 0xFF if i < len(rom) else 0
        elif b == 0xFF:                            # end of stream
            break
        else:                                      # literal
            vdp.write_data(b); vdp.write_data(hi)
            prev = b; i += 1
