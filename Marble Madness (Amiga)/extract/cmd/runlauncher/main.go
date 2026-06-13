// runlauncher runs the real MarbleMadness! launcher on the tools/m68k core in a
// faked Workbench environment: dos.library/exec.library calls are trapped at
// their (libbase-offset) entry points, LoadSeg pulls hunk files out of the ADF,
// and Open/Read stream files. The point is to drive the actual load chain — the
// boot screen, c/zzz, the c/xxx decrypt — dynamically rather than by static
// reading, and in particular to show what state the decoder runs against (the
// CPU exception vectors are left at 0, because the launcher never writes them).
//
// Usage: runlauncher disk.adf [-trace] [-steps N]
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"

	"stupidcoder.com/tools/amiga/adf"
	"stupidcoder.com/tools/amiga/hunk"
	"stupidcoder.com/tools/m68k"
)

const (
	execBase = 0x10000
	dosBase  = 0x20000
	lauBase  = 0x50000 // launcher relocation base
	taskAddr = 0x30000
	wbMsg    = 0x30200
	wbArg    = 0x30220
	wbName   = 0x30240
	heap0    = 0x60000  // AllocMem bump arena
	seg0     = 0x200000 // LoadSeg'd modules
	ramSize  = 0x800000
)

type file struct {
	data []byte
	pos  int
}

type machine struct {
	ram       []byte
	vol       *adf.Volume
	heap      uint32
	seg       uint32
	files     map[uint32]*file // handle -> open file
	nextH     uint32
	trace     bool
	gaveMsg   bool
	zeroReads int
	log       []string
}

func (m *machine) Read(a uint32) byte {
	if int(a) < len(m.ram) {
		return m.ram[a]
	}
	return 0
}
func (m *machine) Write(a uint32, v byte) {
	if int(a) < len(m.ram) {
		m.ram[a] = v
	}
}
func (m *machine) r32(a uint32) uint32 { return binary.BigEndian.Uint32(m.ram[a : a+4]) }
func (m *machine) w32(a, v uint32)     { binary.BigEndian.PutUint32(m.ram[a:a+4], v) }
func (m *machine) cstr(a uint32) string { // null-terminated string at a (or a BSTR via a<<2)
	if a == 0 {
		return ""
	}
	end := a
	for end < uint32(len(m.ram)) && m.ram[end] != 0 && end-a < 64 {
		end++
	}
	s := string(m.ram[a:end])
	if len(s) > 0 && s[0] >= 0x20 {
		return s
	}
	// try BSTR: a is a BPTR -> (a<<2) -> [len][chars]
	b := a << 2
	if int(b) < len(m.ram) {
		n := uint32(m.ram[b])
		return string(m.ram[b+1 : b+1+n])
	}
	return ""
}

func (m *machine) logf(f string, a ...interface{}) {
	s := fmt.Sprintf(f, a...)
	m.log = append(m.log, s)
	if m.trace {
		fmt.Fprintln(os.Stderr, s)
	}
}

func main() {
	trace := flag.Bool("trace", false, "log every trapped call as it happens")
	steps := flag.Int("steps", 60_000_000, "instruction budget")
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: runlauncher disk.adf [-trace]")
		os.Exit(2)
	}
	image, err := os.ReadFile(flag.Arg(0))
	must(err)
	vol, err := adf.Open(image)
	must(err)
	lau, err := vol.ReadFile("MarbleMadness!")
	must(err)
	prog, err := hunk.Load(lau, lauBase)
	must(err)

	m := &machine{ram: make([]byte, ramSize), vol: vol, heap: heap0, seg: seg0,
		files: map[uint32]*file{}, nextH: 0x9000, trace: *trace}
	copy(m.ram[lauBase:], prog.Image)

	// minimal environment
	m.w32(4, execBase)                  // AbsExecBase
	m.w32(taskAddr+0xAC, 0)             // pr_CLI = 0 -> Workbench path
	m.w32(taskAddr+0x5C, taskAddr+0x5C) // pr_MsgPort (self, unused; WaitPort/GetMsg are trapped)
	// WBStartup message
	m.w32(wbMsg+0x1C, 1)     // sm_NumArgs
	m.w32(wbMsg+0x24, wbArg) // sm_ArgList
	m.w32(wbMsg+0x0E, 0)     // sm_Segment
	m.w32(wbArg+0, 0)        // wa_Lock (0 = root)
	m.w32(wbArg+4, wbName)   // wa_Name
	copy(m.ram[wbName:], "MarbleMadness!\x00")

	cpu := m68k.NewCPU(m)
	// AmigaDOS enters the program at hunk 0; d0/a0 are the CLI arg length/ptr (0 for WB)
	cpu.A[7] = 0x40000
	cpu.PC = lauBase
	cpu.D[0] = 0
	cpu.A[0] = 0

	traps := m.buildTraps(cpu)

	ended := "completed (returned to AmigaDOS)"
	done := false
	for i := 0; i < *steps; i++ {
		if cpu.PC == 0xDEADBEEF { // sentinel: launcher returned to AmigaDOS
			done = true
			break
		}
		if t, ok := traps[cpu.PC]; ok {
			t()
			continue
		}
		if cpu.Halted {
			ended = fmt.Sprintf("halted at $%06X: %s", cpu.PC, cpu.HaltReason)
			break
		}
		cpu.Step()
	}
	if !done && !cpu.Halted {
		ended = fmt.Sprintf("step budget (%d) exhausted at pc $%06X", *steps, cpu.PC)
	}
	fmt.Println("=== call log ===")
	for _, l := range m.log {
		fmt.Println(l)
	}
	fmt.Println("\n=== outcome ===")
	fmt.Println("end state:", ended)
	if f := m.files[0x9000]; f != nil {
		fmt.Printf("c/xxx: read %d of %d bytes before stopping\n", f.pos, len(f.data))
	}
}

// ret pops the JSR return address and resumes, with d0 = v.
func (m *machine) ret(cpu *m68k.CPU, v uint32) {
	cpu.D[0] = v
	cpu.PC = m.r32(cpu.A[7])
	cpu.A[7] += 4
}

func (m *machine) alloc(size uint32) uint32 {
	size = (size + 7) &^ 7
	a := m.heap
	m.heap += size
	for i := uint32(0); i < size; i++ {
		m.ram[a+i] = 0
	}
	return a
}

func (m *machine) buildTraps(cpu *m68k.CPU) map[uint32]func() {
	t := map[uint32]func(){}
	ex := func(off uint32, f func()) { t[execBase-off] = f }
	ds := func(off uint32, f func()) { t[dosBase-off] = f }

	// --- exec.library ---
	ex(294, func() { m.ret(cpu, taskAddr) }) // FindTask
	ex(552, func() {                         // OpenLibrary
		name := m.cstr(cpu.A[1])
		if name == "dos.library" {
			m.logf("OpenLibrary(%q) = dos $%X", name, dosBase)
			m.ret(cpu, dosBase)
		} else {
			m.logf("OpenLibrary(%q) = 0 (not provided)", name)
			m.ret(cpu, 0)
		}
	})
	ex(408, func() { m.logf("OldOpenLibrary(%q) = dos", m.cstr(cpu.A[1])); m.ret(cpu, dosBase) })
	ex(414, func() { m.ret(cpu, 0) })        // CloseLibrary
	ex(132, func() { m.ret(cpu, 0) })        // Forbid
	ex(138, func() { m.ret(cpu, 0) })        // Permit
	ex(216, func() { m.ret(cpu, 0x100000) }) // AvailMem -> 1MB
	ex(198, func() {                         // AllocMem(d0=size, d1=flags)
		a := m.alloc(cpu.D[0])
		m.logf("AllocMem(%d, $%X) = $%06X", cpu.D[0], cpu.D[1], a)
		m.ret(cpu, a)
	})
	ex(210, func() { m.ret(cpu, 0) })     // FreeMem
	ex(330, func() { m.ret(cpu, 0) })     // AllocSignal -> signal 0
	ex(336, func() { m.ret(cpu, 0) })     // FreeSignal
	ex(354, func() { m.ret(cpu, 0) })     // AddPort
	ex(360, func() { m.ret(cpu, 0) })     // RemPort
	ex(384, func() { m.ret(cpu, wbMsg) }) // WaitPort -> the WB message
	ex(372, func() {                      // GetMsg -> WB message once, then 0
		if !m.gaveMsg {
			m.gaveMsg = true
			m.logf("GetMsg -> WBStartup $%X", wbMsg)
			m.ret(cpu, wbMsg)
		} else {
			m.ret(cpu, 0)
		}
	})
	ex(366, func() { m.ret(cpu, 0) })                                         // PutMsg
	ex(378, func() { m.ret(cpu, 0) })                                         // ReplyMsg
	ex(108, func() { m.logf("Alert($%X) — fatal", cpu.D[7]); m.ret(cpu, 0) }) // Alert

	// --- dos.library (name in d1, mode/extra in d2; CSTR pointers in our world) ---
	ds(84, func() { // Lock
		name := m.cstr(cpu.D[1])
		ok := m.exists(name)
		m.logf("Lock(%q, %d) = %v", name, int32(cpu.D[2]), ok)
		if ok {
			m.ret(cpu, 0x8000) // a non-zero fake lock
		} else {
			m.ret(cpu, 0)
		}
	})
	ds(90, func() { m.ret(cpu, 0) })                            // UnLock
	ds(126, func() { m.ret(cpu, 0) })                           // CurrentDir
	ds(54, func() { m.ret(cpu, 0x7001) })                       // Input
	ds(60, func() { m.ret(cpu, 0x7002) })                       // Output
	ds(114, func() { m.ret(cpu, 0) })                           // Info (returns 0 = fail; harmless)
	ds(150, func() { m.loadSeg(cpu) })                          // LoadSeg
	ds(156, func() { m.ret(cpu, 0) })                           // UnLoadSeg
	ds(30, func() { m.open(cpu) })                              // Open
	ds(36, func() { delete(m.files, cpu.D[1]); m.ret(cpu, 0) }) // Close
	ds(42, func() { m.read(cpu) })                              // Read
	ds(66, func() { m.ret(cpu, 0) })                            // Seek (files are read sequentially)
	return t
}

func (m *machine) exists(name string) bool {
	_, err := m.vol.ReadFile(name)
	return err == nil
}

func (m *machine) open(cpu *m68k.CPU) {
	name := m.cstr(cpu.D[1])
	data, err := m.vol.ReadFile(name)
	if err != nil {
		m.logf("Open(%q) = 0 (missing)", name)
		m.ret(cpu, 0)
		return
	}
	h := m.nextH
	m.nextH += 4
	m.files[h] = &file{data: data}
	m.logf("Open(%q, %d) = handle $%X (%d bytes)", name, int32(cpu.D[2]), h, len(data))
	m.ret(cpu, h)
}

func (m *machine) read(cpu *m68k.CPU) {
	h, buf, n := cpu.D[1], cpu.D[2], int(cpu.D[3])
	f := m.files[h]
	if f == nil {
		m.ret(cpu, 0xFFFFFFFF)
		return
	}
	if f.pos+n > len(f.data) {
		n = len(f.data) - f.pos
	}
	copy(m.ram[buf:int(buf)+n], f.data[f.pos:f.pos+n])
	f.pos += n
	// Stall detector: once the file is exhausted, a healthy decode stops asking
	// for more. If it keeps reading 0 bytes it has lost the stream (a garbage
	// length from a corrupted keystream) and is looping — report and stop.
	if n == 0 {
		m.zeroReads++
		if m.zeroReads > 50 {
			cpu.Halt("c/zzz read loop: decode lost the stream after the header (keystream diverged)")
		}
	} else {
		m.zeroReads = 0
	}
	m.ret(cpu, uint32(n))
}

func (m *machine) loadSeg(cpu *m68k.CPU) {
	name := m.cstr(cpu.D[1])
	data, err := m.vol.ReadFile(name)
	if err != nil {
		m.logf("LoadSeg(%q) = 0 (missing)", name)
		m.ret(cpu, 0)
		return
	}
	base := (m.seg + 16) &^ 7
	// Boot screen needs graphics/intuition we don't emulate; stub it to MOVEQ#0,RTS.
	if name == "c/bootscr" {
		m.placeSeg(base, []byte{0x70, 0x00, 0x4E, 0x75}) // MOVEQ #0,d0 ; RTS
		m.logf("LoadSeg(%q) = $%X (stubbed: graphics overlay not emulated)", name, base)
		m.ret(cpu, (base-4)>>2)
		return
	}
	prog, err := hunk.Load(data, base) // relocate for the exact placement base
	if err != nil {
		m.logf("LoadSeg(%q) hunk error: %v", name, err)
		m.ret(cpu, 0)
		return
	}
	m.placeSeg(base, prog.Image)
	m.logf("LoadSeg(%q) = seglist @$%X (%d hunks, %d bytes)", name, base, len(prog.Segments), len(prog.Image))
	m.ret(cpu, (base-4)>>2)
}

// placeSeg copies relocated code at base with a single-node seglist header
// (next-pointer at base-4 = 0, size at base-8).
func (m *machine) placeSeg(base uint32, code []byte) {
	copy(m.ram[base:], code)
	m.w32(base-4, 0)
	m.w32(base-8, uint32(len(code)))
	m.seg = base + uint32(len(code)) + 16
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "runlauncher:", err)
		os.Exit(1)
	}
}
