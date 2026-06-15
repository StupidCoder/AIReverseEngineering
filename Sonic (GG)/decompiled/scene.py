"""The bank-5 scene/level descriptor — decoded.

`scene_run` ($1414) indexes a table at bank-5 $5600 by the scene/act number ($D238)
to a 40-byte descriptor, which $185D copies to work RAM at $D355. Empirically (by
aligning the descriptors for all acts, and resolving the pointer fields), the table is
the per-act **level resource table**: byte +0 is the zone, bytes +20..+29 are a block
shared by every act in a zone, and +24/+25 points at the zone's compressed tile set.

Confirmed fields are named; the rest are kept as raw bytes with best-effort labels and
honest uncertainty — this is the data-format frontier, decoded as far as the evidence
currently supports.
"""

from dataclasses import dataclass

import machine


@dataclass
class SceneDescriptor:
    zone: int        # +0  zone number 0..5 (mirrored at +29); 3 acts per zone
    gfx_bank: int    # +23 ROM bank holding this zone's graphics (observed: $09)
    tiles_ptr: int   # +24/+25 -> compressed tile set in gfx_bank (128 tiles / block)
    map_ptr: int     # +21/+22 -> zone map/layout pointer (encoding not fully pinned)
    act_ptr: int     # +30/+31 -> per-act data (steps once per act within a zone)
    raw: bytes       # all 40 bytes, for the fields still to be decoded

    @classmethod
    def decode(cls, file_off):
        d = machine.rom[file_off:file_off + 0x28]
        return cls(
            zone=d[0],
            gfx_bank=d[23],
            tiles_ptr=d[24] | d[25] << 8,
            map_ptr=d[21] | d[22] << 8,
            act_ptr=d[30] | d[31] << 8,
            raw=bytes(d),
        )

    def tiles_file_off(self):
        """Flat ROM file offset of this zone's compressed tile set."""
        bank, addr = self.gfx_bank, self.tiles_ptr
        while addr >= 0x4000:        # normalise like the $0406 source setup
            addr -= 0x4000
            bank += 1
        return bank * 0x4000 + addr
