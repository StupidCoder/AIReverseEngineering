#!/usr/bin/env python3
"""Instrumented 6502 emulator to trace Fort Apocalypse game init.

Kept as the documented dynamic-verification scratch tool used while writing
Fort_Apocalypse.md (it logs every data reader and video writer along the real
init/title/game path). For new work prefer the shared, tested Go core in
c64tools/mos6502 (CPU) + c64tools/c64 (machine model); this script predates it
and remains only to reproduce the trace cited in the documentation.
"""
import random, sys

mem = bytearray(0x10000)
prg = open('extracted/FORT-fast-7000.prg','rb').read()
load = prg[0] | prg[1]<<8
mem[load:load+len(prg)-2] = prg[2:]
# stage-2 leftovers don't matter; loader zeroed $0380-$6FFF already at $AFDB

N,V,B,D,I,Z,C = 0x80,0x40,0x10,0x08,0x04,0x02,0x01
class CPU:
    def __init__(s):
        s.a=s.x=s.y=0; s.sp=0xFF; s.p=0x24; s.pc=0
        s.cycles=0
    def push(s,v): mem[0x100+s.sp]=v&0xFF; s.sp=(s.sp-1)&0xFF
    def pop(s): s.sp=(s.sp+1)&0xFF; return mem[0x100+s.sp]

cpu = CPU()
raster = 0
data_readers = {}   # pc -> set of data addresses read in $7000-$86FF
write_pages = set() # pages written
write_ranges = {}   # page -> (min,max)
reads_log_on = True

def io_read(a):
    global raster
    if a == 0xD012:
        raster = (raster + 7) & 0xFF
        return raster
    if a == 0xD41B: return random.randrange(256)
    if a == 0xDC01: return 0xFF       # no key pressed
    if a == 0xDC00: return mem[a]     # last written (joystick patched in)
    return mem[a]

map_readers = {}    # pc -> count, reads of map buffer $0503-$2D02
scan_readers = {}   # pc -> count, reads of scanner buffer $2E00-$343F
screen_writers = {} # pc -> count, writes to $4400-$47FF
charset_writers = {}# pc -> count, writes to $4800-$5FFF

def rd(a):
    a &= 0xFFFF
    if 0xD000 <= a < 0xE000: v = io_read(a)
    else: v = mem[a]
    if reads_log_on:
        if 0x7000 <= a <= 0x86FF and not (0x7000 <= cpu.pc <= 0x86FF and abs(cpu.pc-a)<4):
            data_readers.setdefault(cpu.pc, set()).add(a)
        elif 0x0503 <= a <= 0x2D02:
            map_readers[cpu.pc] = map_readers.get(cpu.pc,0)+1
        elif 0x2E00 <= a <= 0x343F:
            scan_readers[cpu.pc] = scan_readers.get(cpu.pc,0)+1
    return v

def wr(a,v):
    a &= 0xFFFF; v &= 0xFF
    mem[a] = v
    if 0x4400 <= a < 0x4800: screen_writers[cpu.pc] = screen_writers.get(cpu.pc,0)+1
    elif 0x4800 <= a < 0x6000: charset_writers[cpu.pc] = charset_writers.get(cpu.pc,0)+1
    pg = a >> 8
    write_pages.add(pg)
    lo,hi = write_ranges.get(pg,(0x10000,-1))
    write_ranges[pg] = (min(lo,a),max(hi,a))

def setnz(v):
    cpu.p = (cpu.p & ~(N|Z)) | (v & N) | (Z if v==0 else 0)
    return v

def adc(v):
    if cpu.p & D:
        lo = (cpu.a & 0x0F) + (v & 0x0F) + (cpu.p & C)
        hi = (cpu.a >> 4) + (v >> 4)
        if lo > 9: lo += 6; hi += 1
        cpu.p = cpu.p & ~(C|Z|N|V)
        if ((cpu.a + v + (cpu.p & 0)) & 0xFF) == 0: pass
        r = (hi << 4 | (lo & 0x0F)) & 0xFF
        if hi > 9: hi += 6
        if hi > 15: cpu.p |= C
        r = (hi << 4 | (lo & 0x0F)) & 0xFF
        if r == 0: cpu.p |= Z
        cpu.p |= r & N
        cpu.a = r
    else:
        r = cpu.a + v + (cpu.p & C)
        cpu.p = cpu.p & ~(C|V)
        if r > 0xFF: cpu.p |= C
        if (~(cpu.a ^ v) & (cpu.a ^ r)) & 0x80: cpu.p |= V
        cpu.a = setnz(r & 0xFF)

def sbc(v):
    if cpu.p & D:
        c = (cpu.p & C)
        lo = (cpu.a & 0x0F) - (v & 0x0F) - (1 - c)
        hi = (cpu.a >> 4) - (v >> 4)
        if lo < 0: lo += 10; hi -= 1
        if hi < 0: hi += 10; cpu.p &= ~C
        else: cpu.p |= C
        bin_r = cpu.a - v - (1-c)
        cpu.p &= ~(Z|N)
        if (bin_r & 0xFF) == 0: cpu.p |= Z
        cpu.p |= (bin_r & 0x80)
        cpu.a = ((hi << 4) | (lo & 0x0F)) & 0xFF
    else:
        adc(v ^ 0xFF) if False else None
        r = cpu.a - v - (1 - (cpu.p & C))
        cpu.p = cpu.p & ~(C|V)
        if r >= 0: cpu.p |= C
        if ((cpu.a ^ v) & (cpu.a ^ r)) & 0x80: cpu.p |= V
        cpu.a = setnz(r & 0xFF)

def cmp_(reg,v):
    r = reg - v
    cpu.p = cpu.p & ~C | (C if r >= 0 else 0)
    setnz(r & 0xFF)

def fetch():
    v = mem[cpu.pc]; cpu.pc = (cpu.pc+1)&0xFFFF; return v

def addr(mode):
    if mode=='imm': a=cpu.pc; cpu.pc=(cpu.pc+1)&0xFFFF; return a
    if mode=='zp': return fetch()
    if mode=='zpx': return (fetch()+cpu.x)&0xFF
    if mode=='zpy': return (fetch()+cpu.y)&0xFF
    if mode=='abs': lo=fetch(); return lo|fetch()<<8
    if mode=='abx': lo=fetch(); return (lo|fetch()<<8)+cpu.x & 0xFFFF
    if mode=='aby': lo=fetch(); return (lo|fetch()<<8)+cpu.y & 0xFFFF
    if mode=='izx': z=(fetch()+cpu.x)&0xFF; return mem[z]|mem[(z+1)&0xFF]<<8
    if mode=='izy': z=fetch(); return (mem[z]|mem[(z+1)&0xFF]<<8)+cpu.y & 0xFFFF
    raise Exception(mode)

def branch(cond):
    off = fetch()
    if cond: cpu.pc = (cpu.pc + (off if off<128 else off-256)) & 0xFFFF

def asl(v): cpu.p=cpu.p&~C|(v>>7); return setnz((v<<1)&0xFF)
def lsr(v): cpu.p=cpu.p&~C|(v&1); return setnz(v>>1)
def rol(v): c=cpu.p&C; cpu.p=cpu.p&~C|(v>>7); return setnz(((v<<1)|c)&0xFF)
def ror(v): c=(cpu.p&C)<<7; cpu.p=cpu.p&~C|(v&1); return setnz((v>>1)|c)

def irq():
    cpu.push(cpu.pc>>8); cpu.push(cpu.pc&0xFF); cpu.push(cpu.p & ~B)
    cpu.p |= I
    # KERNAL $FF48 stub: PHA TXA PHA TYA PHA, then JMP ($0314)
    cpu.push(cpu.a); cpu.push(cpu.x); cpu.push(cpu.y)
    cpu.pc = mem[0x0314] | mem[0x0315]<<8

def step():
    op = fetch()
    # mode tables
    A=cpu
    if   op==0xA9: A.a=setnz(rd(addr('imm')))
    elif op==0xA5: A.a=setnz(rd(addr('zp')))
    elif op==0xB5: A.a=setnz(rd(addr('zpx')))
    elif op==0xAD: A.a=setnz(rd(addr('abs')))
    elif op==0xBD: A.a=setnz(rd(addr('abx')))
    elif op==0xB9: A.a=setnz(rd(addr('aby')))
    elif op==0xA1: A.a=setnz(rd(addr('izx')))
    elif op==0xB1: A.a=setnz(rd(addr('izy')))
    elif op==0xA2: A.x=setnz(rd(addr('imm')))
    elif op==0xA6: A.x=setnz(rd(addr('zp')))
    elif op==0xB6: A.x=setnz(rd(addr('zpy')))
    elif op==0xAE: A.x=setnz(rd(addr('abs')))
    elif op==0xBE: A.x=setnz(rd(addr('aby')))
    elif op==0xA0: A.y=setnz(rd(addr('imm')))
    elif op==0xA4: A.y=setnz(rd(addr('zp')))
    elif op==0xB4: A.y=setnz(rd(addr('zpx')))
    elif op==0xAC: A.y=setnz(rd(addr('abs')))
    elif op==0xBC: A.y=setnz(rd(addr('abx')))
    elif op==0x85: wr(addr('zp'),A.a)
    elif op==0x95: wr(addr('zpx'),A.a)
    elif op==0x8D: wr(addr('abs'),A.a)
    elif op==0x9D: wr(addr('abx'),A.a)
    elif op==0x99: wr(addr('aby'),A.a)
    elif op==0x81: wr(addr('izx'),A.a)
    elif op==0x91: wr(addr('izy'),A.a)
    elif op==0x86: wr(addr('zp'),A.x)
    elif op==0x96: wr(addr('zpy'),A.x)
    elif op==0x8E: wr(addr('abs'),A.x)
    elif op==0x84: wr(addr('zp'),A.y)
    elif op==0x94: wr(addr('zpx'),A.y)
    elif op==0x8C: wr(addr('abs'),A.y)
    elif op==0xAA: A.x=setnz(A.a)
    elif op==0xA8: A.y=setnz(A.a)
    elif op==0x8A: A.a=setnz(A.x)
    elif op==0x98: A.a=setnz(A.y)
    elif op==0xBA: A.x=setnz(A.sp)
    elif op==0x9A: A.sp=A.x
    elif op==0x48: A.push(A.a)
    elif op==0x68: A.a=setnz(A.pop())
    elif op==0x08: A.push(A.p|B|0x20)
    elif op==0x28: A.p=A.pop()|0x20
    elif op==0x29: A.a=setnz(A.a & rd(addr('imm')))
    elif op==0x25: A.a=setnz(A.a & rd(addr('zp')))
    elif op==0x35: A.a=setnz(A.a & rd(addr('zpx')))
    elif op==0x2D: A.a=setnz(A.a & rd(addr('abs')))
    elif op==0x3D: A.a=setnz(A.a & rd(addr('abx')))
    elif op==0x39: A.a=setnz(A.a & rd(addr('aby')))
    elif op==0x31: A.a=setnz(A.a & rd(addr('izy')))
    elif op==0x09: A.a=setnz(A.a | rd(addr('imm')))
    elif op==0x05: A.a=setnz(A.a | rd(addr('zp')))
    elif op==0x15: A.a=setnz(A.a | rd(addr('zpx')))
    elif op==0x0D: A.a=setnz(A.a | rd(addr('abs')))
    elif op==0x1D: A.a=setnz(A.a | rd(addr('abx')))
    elif op==0x19: A.a=setnz(A.a | rd(addr('aby')))
    elif op==0x11: A.a=setnz(A.a | rd(addr('izy')))
    elif op==0x49: A.a=setnz(A.a ^ rd(addr('imm')))
    elif op==0x45: A.a=setnz(A.a ^ rd(addr('zp')))
    elif op==0x55: A.a=setnz(A.a ^ rd(addr('zpx')))
    elif op==0x4D: A.a=setnz(A.a ^ rd(addr('abs')))
    elif op==0x5D: A.a=setnz(A.a ^ rd(addr('abx')))
    elif op==0x59: A.a=setnz(A.a ^ rd(addr('aby')))
    elif op==0x51: A.a=setnz(A.a ^ rd(addr('izy')))
    elif op==0x69: adc(rd(addr('imm')))
    elif op==0x65: adc(rd(addr('zp')))
    elif op==0x75: adc(rd(addr('zpx')))
    elif op==0x6D: adc(rd(addr('abs')))
    elif op==0x7D: adc(rd(addr('abx')))
    elif op==0x79: adc(rd(addr('aby')))
    elif op==0x71: adc(rd(addr('izy')))
    elif op==0xE9: sbc(rd(addr('imm')))
    elif op==0xE5: sbc(rd(addr('zp')))
    elif op==0xF5: sbc(rd(addr('zpx')))
    elif op==0xED: sbc(rd(addr('abs')))
    elif op==0xFD: sbc(rd(addr('abx')))
    elif op==0xF9: sbc(rd(addr('aby')))
    elif op==0xF1: sbc(rd(addr('izy')))
    elif op==0xC9: cmp_(A.a, rd(addr('imm')))
    elif op==0xC5: cmp_(A.a, rd(addr('zp')))
    elif op==0xD5: cmp_(A.a, rd(addr('zpx')))
    elif op==0xCD: cmp_(A.a, rd(addr('abs')))
    elif op==0xDD: cmp_(A.a, rd(addr('abx')))
    elif op==0xD9: cmp_(A.a, rd(addr('aby')))
    elif op==0xD1: cmp_(A.a, rd(addr('izy')))
    elif op==0xE0: cmp_(A.x, rd(addr('imm')))
    elif op==0xE4: cmp_(A.x, rd(addr('zp')))
    elif op==0xEC: cmp_(A.x, rd(addr('abs')))
    elif op==0xC0: cmp_(A.y, rd(addr('imm')))
    elif op==0xC4: cmp_(A.y, rd(addr('zp')))
    elif op==0xCC: cmp_(A.y, rd(addr('abs')))
    elif op==0x24: v=rd(addr('zp')); A.p=A.p&~(N|V|Z)|(v&(N|V))|(Z if (v&A.a)==0 else 0)
    elif op==0x2C: v=rd(addr('abs')); A.p=A.p&~(N|V|Z)|(v&(N|V))|(Z if (v&A.a)==0 else 0)
    elif op==0xE6: a=addr('zp'); wr(a,setnz((rd(a)+1)&0xFF))
    elif op==0xF6: a=addr('zpx'); wr(a,setnz((rd(a)+1)&0xFF))
    elif op==0xEE: a=addr('abs'); wr(a,setnz((rd(a)+1)&0xFF))
    elif op==0xFE: a=addr('abx'); wr(a,setnz((rd(a)+1)&0xFF))
    elif op==0xC6: a=addr('zp'); wr(a,setnz((rd(a)-1)&0xFF))
    elif op==0xD6: a=addr('zpx'); wr(a,setnz((rd(a)-1)&0xFF))
    elif op==0xCE: a=addr('abs'); wr(a,setnz((rd(a)-1)&0xFF))
    elif op==0xDE: a=addr('abx'); wr(a,setnz((rd(a)-1)&0xFF))
    elif op==0xE8: A.x=setnz((A.x+1)&0xFF)
    elif op==0xC8: A.y=setnz((A.y+1)&0xFF)
    elif op==0xCA: A.x=setnz((A.x-1)&0xFF)
    elif op==0x88: A.y=setnz((A.y-1)&0xFF)
    elif op==0x0A: A.a=asl(A.a)
    elif op==0x06: a=addr('zp'); wr(a,asl(rd(a)))
    elif op==0x16: a=addr('zpx'); wr(a,asl(rd(a)))
    elif op==0x0E: a=addr('abs'); wr(a,asl(rd(a)))
    elif op==0x1E: a=addr('abx'); wr(a,asl(rd(a)))
    elif op==0x4A: A.a=lsr(A.a)
    elif op==0x46: a=addr('zp'); wr(a,lsr(rd(a)))
    elif op==0x56: a=addr('zpx'); wr(a,lsr(rd(a)))
    elif op==0x4E: a=addr('abs'); wr(a,lsr(rd(a)))
    elif op==0x5E: a=addr('abx'); wr(a,lsr(rd(a)))
    elif op==0x2A: A.a=rol(A.a)
    elif op==0x26: a=addr('zp'); wr(a,rol(rd(a)))
    elif op==0x36: a=addr('zpx'); wr(a,rol(rd(a)))
    elif op==0x2E: a=addr('abs'); wr(a,rol(rd(a)))
    elif op==0x3E: a=addr('abx'); wr(a,rol(rd(a)))
    elif op==0x6A: A.a=ror(A.a)
    elif op==0x66: a=addr('zp'); wr(a,ror(rd(a)))
    elif op==0x76: a=addr('zpx'); wr(a,ror(rd(a)))
    elif op==0x6E: a=addr('abs'); wr(a,ror(rd(a)))
    elif op==0x7E: a=addr('abx'); wr(a,ror(rd(a)))
    elif op==0x4C: A.pc=addr('abs')
    elif op==0x6C: a=addr('abs'); A.pc=mem[a]|mem[(a&0xFF00)|((a+1)&0xFF)]<<8
    elif op==0x20:
        a=addr('abs'); ret=(A.pc-1)&0xFFFF
        A.push(ret>>8); A.push(ret&0xFF); A.pc=a
    elif op==0x60: A.pc=(A.pop()|A.pop()<<8)+1 & 0xFFFF
    elif op==0x40: A.p=A.pop()|0x20; A.pc=A.pop()|A.pop()<<8
    elif op==0x10: branch(not A.p&N)
    elif op==0x30: branch(A.p&N)
    elif op==0x50: branch(not A.p&V)
    elif op==0x70: branch(A.p&V)
    elif op==0x90: branch(not A.p&C)
    elif op==0xB0: branch(A.p&C)
    elif op==0xD0: branch(not A.p&Z)
    elif op==0xF0: branch(A.p&Z)
    elif op==0x18: A.p&=~C
    elif op==0x38: A.p|=C
    elif op==0x58: A.p&=~I
    elif op==0x78: A.p|=I
    elif op==0xB8: A.p&=~V
    elif op==0xD8: A.p&=~D
    elif op==0xF8: A.p|=D
    elif op==0xEA: pass
    elif op==0x00: raise Exception(f"BRK at {(cpu.pc-1)&0xffff:04x}")
    else: raise Exception(f"opcode {op:02x} at {(cpu.pc-1)&0xffff:04x}")
    cpu.cycles += 1

def run_until(stop_pcs, max_steps, irq_every=0):
    last_irq = 0
    for i in range(max_steps):
        if cpu.pc in stop_pcs: return cpu.pc
        if irq_every and cpu.cycles-last_irq > irq_every and not (cpu.p & I):
            last_irq = cpu.cycles; irq()
        step()
    return None

# ---- phase A: init $8927 -> spin at $8A9F
mem[0xDC00] = 0xFF
cpu.pc = 0x8927
r = run_until({0x8A9F}, 5_000_000)
print("phase A done:", hex(r) if r else "TIMEOUT", "steps:", cpu.cycles)

# ---- phase B: title IRQ frames until fire pressed -> main loop
mem[0xDC00] = 0xFF
# run a few title frames, tracing the first one
trace = []
cpu.p &= ~I
irq()
for i in range(200_000):
    if cpu.pc == 0x8A9F: break
    trace.append(cpu.pc)
    try:
        step()
    except Exception as e:
        print("first-frame STOP:", e)
        for pc in trace[-40:]:
            print(f"  {pc:04x}")
        sys.exit(1)
for f in range(4):
    cpu.p &= ~I
    irq()
    run_until({0x8A9F}, 200_000)  # returns via RTI to spin
print("title frames ok, $15 =", mem[0x15])

# press fire: bit4 low
mem[0xDC00] = 0xEF
cpu.p &= ~I
irq()   # title handler will see fire and JMP $8B97
# now run main loop with periodic IRQ injection; track $9D
seen = set()
stop_at = None
last_irq = 0
playing_since = None
for i in range(80_000_000):
    if i - last_irq > 12000 and not (cpu.p & I):
        last_irq = i; irq()
        if playing_since is not None:
            # fly right, then left, to force scrolling both ways
            f = mem[0x15]
            mem[0xDC00] = 0xF7 if (f & 0x40) == 0 else 0xFB
    if mem[0x9D] not in seen:
        seen.add(mem[0x9D])
        print(f"state $9D={mem[0x9D]} at step {i}, pc={cpu.pc:04x}")
        if mem[0x9D] == 2 and playing_since is None:
            playing_since = i
            mem[0xDC00] = 0xFF
            stop_at = i + 15_000_000  # ~ many frames of play
    if i % 10_000_000 == 0 and i:
        print(f"  ... step {i}, pc={cpu.pc:04x}, state={mem[0x9D]}, frame={mem[0x15]}")
    if stop_at and i > stop_at: break
    try:
        step()
    except Exception as e:
        print("STOP:", e); break
print("end pc:", hex(cpu.pc), "state:", mem[0x9D], "frames:", mem[0x15])

def top(d, name, n=10):
    print(f"\n--- {name} ---")
    for pc,c in sorted(d.items(), key=lambda kv:-kv[1])[:n]:
        print(f"pc ${pc:04x}: {c}")
top(map_readers, "map buffer readers ($0503-$2D02)")
top(scan_readers, "scanner buffer readers ($2E00-$343F)")
top(screen_writers, "screen writers ($4400-$47FF)")
top(charset_writers, "charset writers ($4800-$5FFF)")

# ---- reports
print("\n--- data readers ($7000-$86FF) by pc ---")
for pc in sorted(data_readers):
    s = data_readers[pc]
    print(f"pc ${pc:04x}: {len(s)} addrs, ${min(s):04x}-${max(s):04x}")

print("\n--- written ranges outside game file ---")
out = []
for pg in sorted(write_ranges):
    lo,hi = write_ranges[pg]
    if 0x70 <= pg <= 0xB8 or pg == 0x01 or pg == 0x00: continue
    out.append((lo,hi))
# merge
merged = []
for lo,hi in sorted(out):
    if merged and lo <= merged[-1][1]+2: merged[-1][1] = max(merged[-1][1],hi)
    else: merged.append([lo,hi])
for lo,hi in merged:
    print(f"${lo:04x}-${hi:04x}")

open('emu_mem.bin','wb').write(mem)
print("\nmemory dumped to emu_mem.bin")
