package main

// m68k.go — a small Motorola 68000 interpreter, just enough to RUN Turrican's actual
// in-game sound driver code (the decoded $1BB00 overlay) rather than reimplementing it.
// This is the "1:1 port": every routine and opcode of the driver executes as the real
// CPU would. Hardware register writes (Paula $DFF0A0.., DMACON $DFF096, CIA timer
// $BFD600/$BFD700) are intercepted to drive the Paula mixer and recover the tick rate.
//
// The driver uses a modest subset of the 68000 ISA; unimplemented opcodes panic with the
// PC so they can be added. Memory is a flat 16 MB (24-bit) address space.

import "fmt"

const memSize = 0x1000000 // 16 MB, 24-bit address space

type m68k struct {
	d   [8]uint32 // data registers
	a   [8]uint32 // address registers (a[7] = stack pointer)
	pc  uint32
	sr  uint16 // status register; low byte = CCR (X N Z V C)
	mem []byte

	// intercepted hardware state
	paula   [4]paulaReg // AUDxLC/LEN/PER/VOL + DMA
	dmacon  uint16
	ciaTBlo uint8
	ciaTBhi uint8

	halt bool
	insn uint64 // executed-instruction counter (guard against runaways)
}

// paulaReg mirrors one Paula audio channel's registers as the driver sets them.
type paulaReg struct {
	lc       uint32 // AUDxLC (sample start, byte address)
	length   uint16 // AUDxLEN (in words)
	period   uint16 // AUDxPER
	vol      uint16 // AUDxVOL
	dma      bool   // DMACON channel bit
	retrig   bool   // DMA went off->on since last read (restart sample)
	playLC   uint32 // lc latched at the last DMA-on (Paula's initial pointer)
	playLEN  uint16 // length latched at the last DMA-on
}

func newM68k() *m68k { return &m68k{mem: make([]byte, memSize)} }

// ---- memory access (big-endian) with I/O interception ----

func (m *m68k) isIO(a uint32) bool {
	return a >= 0xDFF000 && a < 0xDFF200 || a >= 0xBFD000 && a <= 0xBFEFFF
}

func (m *m68k) rd8(a uint32) uint8 {
	a &= 0xFFFFFF
	if m.isIO(a) {
		return uint8(m.ioRead(a) >> ((1 - (a & 1)) * 8))
	}
	return m.mem[a]
}
func (m *m68k) rd16(a uint32) uint16 {
	a &= 0xFFFFFF
	if m.isIO(a) {
		return m.ioRead(a)
	}
	return uint16(m.mem[a])<<8 | uint16(m.mem[a+1])
}
func (m *m68k) rd32(a uint32) uint32 {
	return uint32(m.rd16(a))<<16 | uint32(m.rd16(a+2))
}
func (m *m68k) wr8(a uint32, v uint8) {
	a &= 0xFFFFFF
	if m.isIO(a) {
		m.ioWrite8(a, v)
		return
	}
	m.mem[a] = v
}
func (m *m68k) wr16(a uint32, v uint16) {
	a &= 0xFFFFFF
	if m.isIO(a) {
		m.ioWrite16(a, v)
		return
	}
	m.mem[a] = byte(v >> 8)
	m.mem[a+1] = byte(v)
}
func (m *m68k) wr32(a uint32, v uint32) {
	m.wr16(a, uint16(v>>16))
	m.wr16(a+2, uint16(v))
}

// ---- hardware register interception ----

func (m *m68k) ioRead(a uint32) uint16 {
	// Custom/CIA registers are mostly write-only here; reads return 0 (DMACONR etc.
	// aren't used by the driver in a way that needs a real value).
	return 0
}

func (m *m68k) ioWrite8(a uint32, v uint8) {
	switch a {
	case 0xBFD600:
		m.ciaTBlo = v
	case 0xBFD700:
		m.ciaTBhi = v
	default:
		// other CIA bytes: ignore
	}
}

func (m *m68k) ioWrite16(a uint32, v uint16) {
	switch {
	case a == 0xDFF096: // DMACON
		set := v&0x8000 != 0
		for ch := 0; ch < 4; ch++ {
			if v&(1<<uint(ch)) != 0 {
				if set && !m.paula[ch].dma {
					m.paula[ch].retrig = true
					m.paula[ch].playLC = m.paula[ch].lc // Paula latches start + length here
					m.paula[ch].playLEN = m.paula[ch].length
				}
				m.paula[ch].dma = set
			}
		}
		if v&0x8000 != 0 {
			m.dmacon |= v & 0x7FFF
		} else {
			m.dmacon &^= v & 0x7FFF
		}
	case a >= 0xDFF0A0 && a < 0xDFF0E0: // Paula channel registers
		ch := (a - 0xDFF0A0) / 0x10
		switch (a - 0xDFF0A0) % 0x10 {
		case 0x0:
			m.paula[ch].lc = m.paula[ch].lc&0x0000FFFF | uint32(v)<<16
		case 0x2:
			m.paula[ch].lc = m.paula[ch].lc&0xFFFF0000 | uint32(v)
		case 0x4:
			m.paula[ch].length = v
		case 0x6:
			m.paula[ch].period = v
		case 0x8:
			m.paula[ch].vol = v & 0x7F
		}
	case a == 0xBFD600:
		m.ciaTBlo = uint8(v)
	case a == 0xBFD700:
		m.ciaTBhi = uint8(v)
	}
}

// ---- flags ----

const (
	flagC = 1 << 0
	flagV = 1 << 1
	flagZ = 1 << 2
	flagN = 1 << 3
	flagX = 1 << 4
)

func (m *m68k) getf(f uint16) bool { return m.sr&f != 0 }
func (m *m68k) setf(f uint16, on bool) {
	if on {
		m.sr |= f
	} else {
		m.sr &^= f
	}
}

func msb(v uint32, size int) bool {
	switch size {
	case 1:
		return v&0x80 != 0
	case 2:
		return v&0x8000 != 0
	}
	return v&0x80000000 != 0
}
func sizeMask(size int) uint32 {
	switch size {
	case 1:
		return 0xFF
	case 2:
		return 0xFFFF
	}
	return 0xFFFFFFFF
}

// setNZ sets N and Z from a result, clears V and C (logic ops).
func (m *m68k) setNZ(v uint32, size int) {
	v &= sizeMask(size)
	m.setf(flagN, msb(v, size))
	m.setf(flagZ, v == 0)
	m.setf(flagV, false)
	m.setf(flagC, false)
}

func signExtend(v uint32, size int) int32 {
	switch size {
	case 1:
		return int32(int8(v))
	case 2:
		return int32(int16(v))
	}
	return int32(v)
}

// ---- effective address ----

type ea struct {
	reg  bool // register direct (Dn or An)
	isA  bool // an address register
	idx  int  // register index for reg modes
	addr uint32
}

// decodeEA resolves mode/reg into an ea, consuming any extension words from the PC.
func (m *m68k) decodeEA(mode, reg, size int) ea {
	switch mode {
	case 0: // Dn
		return ea{reg: true, idx: reg}
	case 1: // An
		return ea{reg: true, isA: true, idx: reg}
	case 2: // (An)
		return ea{addr: m.a[reg]}
	case 3: // (An)+
		e := ea{addr: m.a[reg]}
		m.a[reg] += uint32(size)
		if reg == 7 && size == 1 {
			m.a[reg]++ // keep stack word-aligned
		}
		return e
	case 4: // -(An)
		m.a[reg] -= uint32(size)
		if reg == 7 && size == 1 {
			m.a[reg]--
		}
		return ea{addr: m.a[reg]}
	case 5: // d16(An)
		d := int32(int16(m.fetch16()))
		return ea{addr: uint32(int32(m.a[reg]) + d)}
	case 6: // d8(An,Xn)
		return ea{addr: m.brief(m.a[reg])}
	case 7:
		switch reg {
		case 0: // abs.w
			return ea{addr: uint32(int32(int16(m.fetch16())))}
		case 1: // abs.l
			return ea{addr: m.fetch32()}
		case 2: // d16(PC)
			base := m.pc
			d := int32(int16(m.fetch16()))
			return ea{addr: uint32(int32(base) + d)}
		case 3: // d8(PC,Xn)
			base := m.pc
			return ea{addr: m.brief(base)}
		case 4: // #imm
			var v uint32
			if size == 4 {
				v = m.fetch32()
			} else {
				w := m.fetch16()
				if size == 1 {
					v = uint32(w & 0xFF)
				} else {
					v = uint32(w)
				}
			}
			return ea{reg: true, idx: -1, addr: v} // immediate carried in addr
		}
	}
	panic(fmt.Sprintf("bad EA mode=%d reg=%d @%06X", mode, reg, m.pc))
}

// brief handles the brief extension word for d8(An,Xn)/d8(PC,Xn).
func (m *m68k) brief(base uint32) uint32 {
	ext := m.fetch16()
	d := int32(int8(uint8(ext)))
	xn := (ext >> 12) & 7
	var idx int32
	if ext&0x8000 != 0 {
		idx = int32(m.a[xn])
	} else {
		idx = int32(m.d[xn])
	}
	if ext&0x0800 == 0 { // word index
		idx = int32(int16(idx))
	}
	return uint32(int32(base) + d + idx)
}

func (m *m68k) eaLoad(e ea, size int) uint32 {
	if e.reg {
		if e.idx == -1 { // immediate
			return e.addr & sizeMask(size)
		}
		if e.isA {
			return m.a[e.idx] & sizeMask(size)
		}
		return m.d[e.idx] & sizeMask(size)
	}
	switch size {
	case 1:
		return uint32(m.rd8(e.addr))
	case 2:
		return uint32(m.rd16(e.addr))
	}
	return m.rd32(e.addr)
}

func (m *m68k) eaStore(e ea, size int, v uint32) {
	if e.reg {
		if e.isA { // address regs always store full 32 bits (sign-extended for word)
			if size == 2 {
				m.a[e.idx] = uint32(int32(int16(v)))
			} else {
				m.a[e.idx] = v
			}
			return
		}
		mask := sizeMask(size)
		m.d[e.idx] = m.d[e.idx]&^mask | v&mask
		return
	}
	switch size {
	case 1:
		m.wr8(e.addr, uint8(v))
	case 2:
		m.wr16(e.addr, uint16(v))
	default:
		m.wr32(e.addr, v)
	}
}

// ---- instruction fetch ----

func (m *m68k) fetch16() uint16 {
	v := m.rd16(m.pc)
	m.pc += 2
	return v
}
func (m *m68k) fetch32() uint32 {
	v := m.rd32(m.pc)
	m.pc += 4
	return v
}

// ---- run ----

// call runs a subroutine at addr until it returns to the sentinel address.
func (m *m68k) call(addr uint32, regs map[int]uint32) {
	const sentinel = 0x00FFFFF0
	m.a[7] -= 4
	m.wr32(m.a[7], sentinel)
	for k, v := range regs {
		if k < 8 {
			m.d[k] = v
		} else {
			m.a[k-8] = v
		}
	}
	m.pc = addr
	m.halt = false
	for !m.halt {
		if m.pc == sentinel {
			return
		}
		m.step()
	}
}

func (m *m68k) tickRate() float64 {
	reload := uint32(m.ciaTBhi)<<8 | uint32(m.ciaTBlo)
	if reload == 0 {
		return 24.74
	}
	return 709379.0 / float64(reload)
}
