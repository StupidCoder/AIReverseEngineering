"""The bank-5 scene/level descriptor — decoded against its consumer.

`scene_run` ($1414) indexes a table at bank-5 $5600 by the scene/act number ($D238)
to a 40-byte descriptor, which $185D copies to work RAM at $D355. The descriptor is
read through a pushed pointer, not absolute addressing, so the fields are named by
following where $185D *unpacks* it (at $1912, popping that pointer):

    +0        -> $D2D5   zone number (0..5; three acts per zone)
    +1/+2     -> $D232   a word  (small values; level bound / extent)
    +3/+4     -> $D234   a word  (small values; level bound / extent)
    +5..+12   -> $D26D   an 8-byte sub-block; the bank-1 scroll engine reads $D26D and
                         $D26F from it ($4F33/$4F5B) as MAP-region pointers
    +23       graphics bank ($09)
    +24/+25   pointer to the zone's compressed tile set (128 tiles) — verified by
              decompressing it to coherent level graphics

(An earlier guess that +21/+22 was the map pointer was wrong: +20..+29 is a per-zone
block, identical across a zone's acts, and +21/+22 does not resolve to a block. The
real map pointers are the per-act $D26D/$D26F surfaced from +5..+12; fully resolving
them to map bytes needs the bank-1 scroll engine's map format — the next layer.)
"""

from dataclasses import dataclass

import machine


@dataclass
class SceneDescriptor:
    zone: int        # +0    zone number 0..5, mirrored at +29
    bound1: int      # +1/+2 -> $D232  (level extent / bound)
    bound2: int      # +3/+4 -> $D234  (level extent / bound)
    map_block: bytes # +5..+12 -> $D26D  (the scroll engine reads $D26D/$D26F as map ptrs)
    gfx_bank: int    # +23   ROM bank holding this zone's graphics (observed: $09)
    tiles_ptr: int   # +24/+25 -> compressed tile set in gfx_bank (128 tiles / block)
    raw: bytes       # all 40 bytes, for fields still to be decoded

    @classmethod
    def decode(cls, file_off):
        d = machine.rom[file_off:file_off + 0x28]
        return cls(
            zone=d[0],
            bound1=d[1] | d[2] << 8,
            bound2=d[3] | d[4] << 8,
            map_block=bytes(d[5:13]),
            gfx_bank=d[23],
            tiles_ptr=d[24] | d[25] << 8,
            raw=bytes(d),
        )

    @property
    def map_ptr_a(self):  # $D26D  = +5/+6
        return self.map_block[0] | self.map_block[1] << 8

    @property
    def map_ptr_b(self):  # $D26F  = +7/+8
        return self.map_block[2] | self.map_block[3] << 8

    def tiles_file_off(self):
        """Flat ROM file offset of this zone's compressed tile set."""
        bank, addr = self.gfx_bank, self.tiles_ptr
        while addr >= 0x4000:        # normalise like the $0406 source setup
            addr -= 0x4000
            bank += 1
        return bank * 0x4000 + addr
