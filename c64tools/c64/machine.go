// Package c64 is a minimal Commodore 64 machine model for running tape
// fastloaders under emulation, so their payload can be recovered without
// reimplementing a self-modifying wire format.
//
// Only the hardware a typical tape loader touches is emulated:
//   - CIA1 ICR ($DC0D) bit 4: cassette read FLAG. Each falling edge on the
//     tape line sets it; the loader busy-waits on it, then clears it. Edges
//     are fed from the TAP pulse stream.
//   - CIA2 timer A ($DD04/05 latch, $DD0E control) and ICR ($DD0D) bit 0:
//     loaders restart the timer on every edge and check whether it underflowed
//     during the pulse, i.e. pulse longer than the latch -> a 1 bit. Timer B
//     ($DD06/07, ICR bit 1) is modelled the same way.
//   - $D011/$D012 raster: returns changing/stable values so raster polls end.
//   - CIA1 keyboard ports ($DC00/$DC01): no key pressed.
//
// ROM is not present. Callers register Go hooks at the ROM/RAM entry points
// their loader calls (see SetHook and InstallKernalTapeHooks). A jump into a
// banked-in ROM region without a hook stops emulation with a diagnostic.
package c64

import (
	"stupidcoder.com/c64tools/mos6502"
	"stupidcoder.com/c64tools/tap"
)

// WriteEvent records a RAM write performed by emulated code, tagged with how
// far it sits from the most recent tape pulse so callers can tell loader
// payload from incidental memory traffic.
type WriteEvent struct {
	Addr       uint16
	Val        byte
	PulseIdx   int    // pulses consumed when the write happened
	PC         uint16 // PC of the writing instruction
	Instr      uint64 // instruction count when the write happened
	SincePulse uint64 // instructions since the last tape pulse was consumed
}

// Hook is a Go implementation of a routine the emulated code jumps to. It runs
// in place of the instruction at its address. Returning false stops the run.
type Hook func(m *Machine) bool

// Machine is the emulated C64: RAM, CPU, the tape pulse feed and the CIA state
// the loader observes.
type Machine struct {
	RAM [65536]byte
	CPU *mos6502.CPU

	Pulses   []tap.Pulse
	PulsePos int

	// Writes is the log of all RAM writes (I/O writes are not logged).
	Writes []WriteEvent

	// Watchdog stops the run after this many instructions without a tape
	// pulse being consumed. Zero disables it.
	Watchdog uint64

	hooks map[uint16]Hook

	// CIA state
	latchA, latchB uint16
	icr2           byte
	flag1          bool
	nextEdgeCycle  uint64

	lastPulseInstr uint64
	anyPulse       bool

	traceBuf [256]uint16
	tracePos int
}

// New creates a machine that will start feeding tape pulses from startPulse.
func New(pulses []tap.Pulse, startPulse int) *Machine {
	m := &Machine{
		Pulses:   pulses,
		PulsePos: startPulse,
		hooks:    map[uint16]Hook{},
		Watchdog: 30_000_000,
	}
	m.CPU = mos6502.NewCPU(m)
	return m
}

// SetHook installs a hook at addr, replacing any previous one.
func (m *Machine) SetHook(addr uint16, h Hook) { m.hooks[addr] = h }

// HookRTS installs hooks at the given addresses that do nothing but return
// (RTS). Useful for stubbing out harmless ROM housekeeping calls.
func (m *Machine) HookRTS(addrs ...uint16) {
	for _, a := range addrs {
		m.SetHook(a, func(m *Machine) bool { m.RTS(); return true })
	}
}

// RTS pops a return address off the stack and continues there, exactly as the
// 6502 RTS instruction would. Hooks that stand in for a subroutine call it to
// return to their caller.
func (m *Machine) RTS() {
	lo := m.CPU.Pop()
	hi := m.CPU.Pop()
	m.CPU.PC = (uint16(hi)<<8 | uint16(lo)) + 1
}

// Read implements mos6502.Bus.
func (m *Machine) Read(addr uint16) byte {
	switch {
	case addr == 0xDC0D: // CIA1 ICR: cassette FLAG in bit 4 (and bit 7)
		m.deliverEdge()
		var v byte
		if m.flag1 {
			v = 0x90
		}
		m.flag1 = false
		return v
	case addr == 0xDD0D: // CIA2 ICR: timer A/B underflow in bits 0/1
		v := m.icr2
		if v != 0 {
			v |= 0x80
		}
		m.icr2 = 0
		return v
	case addr == 0xD012: // raster line low byte, just needs to change
		return byte(m.CPU.Cycles >> 6)
	case addr == 0xD011:
		return 0x1B
	case addr == 0xDC00, addr == 0xDC01: // CIA1 keyboard ports: no key
		return 0xFF
	}
	return m.RAM[addr]
}

// deliverEdge makes the next tape pulse visible once enough emulated time has
// passed. Pulses are delivered in order; the exact arrival time only has to
// be "not within the few instructions between the loader's wait-loop read and
// its flag-clearing read", which real pulse lengths guarantee.
func (m *Machine) deliverEdge() {
	if m.flag1 || m.PulsePos >= len(m.Pulses) {
		return
	}
	if m.CPU.Cycles < m.nextEdgeCycle {
		return
	}
	p := m.Pulses[m.PulsePos]
	m.PulsePos++
	m.flag1 = true
	if uint16(min(p.Cycles, 0xFFFF)) > m.latchA {
		m.icr2 |= 0x01
	}
	if uint16(min(p.Cycles, 0xFFFF)) > m.latchB {
		m.icr2 |= 0x02
	}
	m.nextEdgeCycle = m.CPU.Cycles + uint64(min(p.Cycles, 100000))
	m.lastPulseInstr = m.CPU.Instrs
	m.anyPulse = true
}

// Write implements mos6502.Bus.
func (m *Machine) Write(addr uint16, v byte) {
	switch addr {
	case 0xDD04:
		m.latchA = m.latchA&0xFF00 | uint16(v)
		return
	case 0xDD05:
		m.latchA = m.latchA&0x00FF | uint16(v)<<8
		return
	case 0xDD06:
		m.latchB = m.latchB&0xFF00 | uint16(v)
		return
	case 0xDD07:
		m.latchB = m.latchB&0x00FF | uint16(v)<<8
		return
	}
	if addr >= 0xD000 && addr < 0xE000 {
		return // other I/O: ignore
	}
	m.RAM[addr] = v
	since := ^uint64(0)
	if m.anyPulse {
		since = m.CPU.Instrs - m.lastPulseInstr
	}
	m.Writes = append(m.Writes, WriteEvent{addr, v, m.PulsePos, m.CPU.PC, m.CPU.Instrs, since})
}

// step runs registered hooks and the ROM trap before each instruction.
// Returns false if emulation should stop.
func (m *Machine) step() bool {
	pc := m.CPU.PC
	if h, ok := m.hooks[pc]; ok {
		return h(m)
	}
	// Trap jumps into ROM only while that ROM is actually banked in
	// (port $01 bits 0/1 = LORAM/HIRAM).
	port := m.RAM[1]
	if pc >= 0xA000 && pc < 0xC000 && port&3 == 3 {
		m.CPU.Halt("jumped into BASIC ROM at $%04X (no hook)", pc)
		return false
	}
	if pc >= 0xE000 && port&2 != 0 {
		m.CPU.Halt("jumped into kernal ROM at $%04X (no hook)", pc)
		return false
	}
	return true
}

// Run executes until the CPU halts, a hook stops it, the watchdog fires, or
// the instruction budget is reached.
func (m *Machine) Run(maxInstr uint64) {
	for !m.CPU.Halted && m.CPU.Instrs < maxInstr {
		if !m.step() {
			break
		}
		if m.Watchdog != 0 && m.CPU.Instrs-m.lastPulseInstr > m.Watchdog {
			m.CPU.Halt("no tape activity for %d instructions (PC=$%04X)", m.Watchdog, m.CPU.PC)
			break
		}
		m.traceBuf[m.tracePos&255] = m.CPU.PC
		m.tracePos++
		m.CPU.Step()
	}
	if m.CPU.Instrs >= maxInstr {
		m.CPU.Halt("instruction budget exhausted (PC=$%04X)", m.CPU.PC)
	}
}

// TraceTail returns the most recently executed PCs (oldest first).
func (m *Machine) TraceTail(n int) []uint16 {
	if n > 256 {
		n = 256
	}
	if n > m.tracePos {
		n = m.tracePos
	}
	out := make([]uint16, 0, n)
	for i := m.tracePos - n; i < m.tracePos; i++ {
		out = append(out, m.traceBuf[i&255])
	}
	return out
}
