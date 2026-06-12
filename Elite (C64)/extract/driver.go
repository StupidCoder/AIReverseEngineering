package main

// Elite-specific driver for the generic c64 machine model. The boot file
// autostarts the fastloader; the loader self-modifies its own wire format on
// the fly, so we run it rather than decoding it. This file supplies the hooks
// that stand in for the ROM/BASIC routines Elite's loader chain calls:
//
//   - the standard KERNAL tape routines (via InstallKernalTapeHooks)
//   - BASIC housekeeping stubs that the exit code calls before RUN
//   - the BASIC statement loop $A7EA, which we drive to execute the loaded
//     stub's LOAD : LOAD : SYS 30215
//   - KERNAL LOAD ($FFD5/$F49E) and the RTS-return trampoline, both of which
//     dispatch through the loader's ILOAD vector at $0330

import (
	"fmt"

	"stupidcoder.com/c64tools/c64"
)

// loadReturnPC is a synthetic return address pushed when the driver calls the
// ILOAD vector; loader stages that finish with RTS land here so we can advance
// the BASIC program.
const loadReturnPC = 0xF100

type driver struct {
	m          *c64.Machine
	queue      []basicAction
	queueBuilt bool
}

type basicAction struct {
	kind string // "load" or "sys"
	addr uint16
}

// newMachine builds a c64 machine seeded with the boot file and wired with
// Elite's hook set, ready to run from the autostart entry point.
func newMachine(d *driver) {
	m := d.m
	m.InstallKernalTapeHooks()
	// BASIC housekeeping called by the segment-2 exit code before RUN.
	m.HookRTS(0xA871, 0xA68E, 0xA67E, 0xA660)

	// BASIC: execute next statement -> drive the loaded stub.
	m.SetHook(0xA7EA, func(*c64.Machine) bool { return d.basicStep() })

	// KERNAL LOAD: dispatch through the loader's ILOAD vector at $0330.
	loadDispatch := func(*c64.Machine) bool {
		vec := uint16(m.RAM[0x0330]) | uint16(m.RAM[0x0331])<<8
		fmt.Printf("kernal LOAD -> ILOAD vector $%04X  (pulse %d)\n", vec, m.PulsePos)
		m.CPU.PC = vec
		return true
	}
	m.SetHook(0xFFD5, loadDispatch)
	m.SetHook(0xF49E, loadDispatch)

	// A loader stage that finishes with RTS returns here.
	m.SetHook(loadReturnPC, func(*c64.Machine) bool {
		fmt.Printf("LOAD returned via RTS (pulse %d)\n", m.PulsePos)
		return d.basicStep()
	})
}

// basicStep emulates the loaded BASIC stub one statement at a time:
//
//	10 IF F=0 THEN F=1:LOAD
//	20 IF F=1 THEN F=2:LOAD
//	30 SYS 30215
//
// i.e. effectively LOAD : LOAD : SYS 30215. The tokenised program at $0801 is
// parsed once, then one queued action is dispatched per arrival at $A7EA.
func (d *driver) basicStep() bool {
	m := d.m
	if !d.queueBuilt {
		d.queueBuilt = true
		d.queue = parseBasicStub(m.RAM[:], 0x0801)
		fmt.Printf("BASIC stub at $0801 parsed: %v\n", d.queue)
	}
	if len(d.queue) == 0 {
		m.CPU.Halt("BASIC program finished")
		return false
	}
	act := d.queue[0]
	d.queue = d.queue[1:]
	switch act.kind {
	case "load":
		vec := uint16(m.RAM[0x0330]) | uint16(m.RAM[0x0331])<<8
		fmt.Printf("BASIC: LOAD -> ILOAD vector $%04X  (pulse %d)\n", vec, m.PulsePos)
		ret := uint16(loadReturnPC - 1)
		m.CPU.Push(byte(ret >> 8))
		m.CPU.Push(byte(ret))
		m.CPU.PC = vec
	case "sys":
		fmt.Printf("BASIC: SYS %d ($%04X)  (pulse %d)\n", act.addr, act.addr, m.PulsePos)
		m.CPU.PC = act.addr
	}
	return true
}

// parseBasicStub walks a tokenised BASIC program and extracts the sequence of
// LOAD ($93) and SYS ($9E, decimal argument) statements.
func parseBasicStub(ram []byte, addr int) []basicAction {
	var acts []basicAction
	for {
		next := int(ram[addr]) | int(ram[addr+1])<<8
		if next == 0 {
			break
		}
		for i := addr + 4; i < next-1 && ram[i] != 0; i++ {
			switch ram[i] {
			case 0x93:
				acts = append(acts, basicAction{kind: "load"})
			case 0x9E:
				n := 0
				for j := i + 1; ram[j] >= '0' && ram[j] <= '9'; j++ {
					n = n*10 + int(ram[j]-'0')
				}
				acts = append(acts, basicAction{kind: "sys", addr: uint16(n)})
			}
		}
		addr = next
	}
	return acts
}
