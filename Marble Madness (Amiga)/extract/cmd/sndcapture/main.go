// sndcapture runs the real Marble Madness sound code (from the decrypted game
// body) on the tools/m68k core to capture the GROUND-TRUTH audio.device command
// stream for a course's music — which sample plays, at what period and volume, on
// which frame. That stream is the score a Go reimplementation must reproduce.
//
// The decrypted .dat is loaded above fake library bases; exec/dos calls are
// trapped (reusing the runlauncher approach), *Snd files are served from the ADF
// via LoadSeg, and audio.device is emulated: OpenDevice succeeds, and every
// BeginIO/DoIO on an IOAudio request is logged (and auto-replied) so the engine's
// allocate/play handshake proceeds.
//
// Usage: sndcapture disk.adf decrypted.dat.hunk [-entry 0xADDR] [-steps N] [-trace]
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
	execBase  = 0x10000
	dosBase   = 0x20000
	audioBase = 0x40000
	heap0     = 0x60000
	datBase   = 0x100000 // decrypted .dat relocated here (above the fake bases)
	seg0      = 0x400000 // LoadSeg'd *Snd modules
	stackTop  = 0x3F0000
	ramSize   = 0x1000000
	sentinel  = 0xDEADBEEF
)

// IOAudio field offsets (standard Exec layout; request is $44 bytes).
const (
	ioCommand = 0x1C // UWORD
	ioFlags   = 0x1E // UBYTE
	ioError   = 0x1F // BYTE
	ioaData   = 0x22 // APTR  (8-bit sample, chip)
	ioaLength = 0x26 // ULONG
	ioaPeriod = 0x2A // UWORD
	ioaVolume = 0x2C // UWORD
	ioaCycles = 0x2E // UWORD
	ioMsgRepl = 0x0E // mn_ReplyPort (within io_Message)
)

type file struct {
	data []byte
	pos  int
}

type audioCmd struct {
	frame             int
	cmd               uint16
	chan_             int
	data, length      uint32
	period, vol, cyc  uint16
}

type machine struct {
	ram    []byte
	vol    *adf.Volume
	heap   uint32
	seg    uint32
	files  map[uint32]*file
	nextH  uint32
	trace  bool
	log    []string
	frame   int
	cmds    []audioCmd
	course  string // *snd filename to force when the game's course global is unset
	curChan int    // channel being pumped (for note attribution)
	msgq    []uint32 // player-task message queue (PutMsg -> GetMsg/WaitPort)
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
func (m *machine) r16(a uint32) uint16  { return binary.BigEndian.Uint16(m.ram[a:]) }
func (m *machine) r32(a uint32) uint32  { return binary.BigEndian.Uint32(m.ram[a:]) }
func (m *machine) w16(a uint32, v uint16) { binary.BigEndian.PutUint16(m.ram[a:], v) }
func (m *machine) w32(a, v uint32)      { binary.BigEndian.PutUint32(m.ram[a:], v) }

func (m *machine) cstr(a uint32) string {
	if a == 0 || int(a) >= len(m.ram) {
		return ""
	}
	end := a
	for end < uint32(len(m.ram)) && m.ram[end] != 0 && end-a < 64 {
		end++
	}
	return string(m.ram[a:end])
}

func (m *machine) logf(f string, a ...interface{}) {
	s := fmt.Sprintf(f, a...)
	m.log = append(m.log, s)
	if m.trace {
		fmt.Fprintln(os.Stderr, s)
	}
}

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

func main() {
	entry := flag.Uint("entry", 0x21DC0, "offset (in the base-0 image) of the routine to call after sound_init")
	steps := flag.Int("steps", 80_000_000, "instruction budget")
	trace := flag.Bool("trace", false, "log every trapped call")
	course := flag.String("course", "prcsnd", "*snd file to force-load")
	song := flag.Uint("song", 0x400010, "absolute address of the loaded *snd song struct (h1) to start")
	flag.Parse()
	if flag.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "usage: sndcapture disk.adf decrypted.dat.hunk [-entry 0xADDR]")
		os.Exit(2)
	}
	adfData, err := os.ReadFile(flag.Arg(0))
	must(err)
	vol, err := adf.Open(adfData)
	must(err)
	datData, err := os.ReadFile(flag.Arg(1))
	must(err)
	prog, err := hunk.Load(datData, datBase)
	must(err)

	m := &machine{
		ram:   make([]byte, ramSize),
		vol:   vol,
		heap:  heap0,
		seg:   seg0,
		files:  map[uint32]*file{},
		nextH:  0x9000,
		trace:  *trace,
		course: *course,
	}
	copy(m.ram[datBase:], prog.Image)
	m.w32(4, execBase) // AbsExecBase
	// The engine caches library bases in its own (relocated) globals, normally set
	// up by an init we bypass. Seed the cached ExecBase global at datBase+$1F4.
	m.w32(datBase+0x1F4, execBase)

	cpu := m68k.NewCPU(m)
	cpu.A[7] = stackTop
	traps := m.buildTraps(cpu)

	// Game side: sound_init + load+play ($21DC0). $21DC0 PutMsgs the play commands to the
	// player task's port (our message queue) and LoadSegs the *Snd.
	m.callRoutine(cpu, traps, 0x1FF34, *steps)
	m.callRoutine(cpu, traps, uint32(*entry), *steps)
	m.logf("--- %d messages queued; running the player task $20A52 ---", len(m.msgq))
	// Player task: its reply loop drains the queue (channel registrations, the play
	// command -> parse + start the song) and the audio replies (each finished note ->
	// the next, enqueued in audioIO). Capture the CMD_WRITE stream.
	m.callRoutine(cpu, traps, 0x20A52, *steps)
	_ = song
	_ = m.call2057A
	_ = m.callRoutineArg

	fmt.Println("=== call log ===")
	for _, l := range m.log {
		fmt.Println(l)
	}
	fmt.Printf("\n=== %d audio commands ===\n", len(m.cmds))
	for _, c := range m.cmds {
		fmt.Printf("f%-4d cmd=$%X ch%d data=$%X len=%d per=%d vol=%d cyc=%d\n",
			c.frame, c.cmd, c.chan_, c.data, c.length, c.period, c.vol, c.cyc)
	}
}

// call2057A invokes the sequencer for one channel: $2057A(ch, request, 0).
func (m *machine) call2057A(cpu *m68k.CPU, traps map[uint32]func(), ch, req uint32, steps int) {
	sp := uint32(stackTop)
	sp -= 4
	m.w32(sp, 0) // arg3: flag = 0
	sp -= 4
	m.w32(sp, req) // arg2: the channel's IOAudio request
	sp -= 4
	m.w32(sp, ch) // arg1: channel index
	sp -= 4
	m.w32(sp, sentinel) // return address
	cpu.A[7] = sp
	cpu.PC = datBase + 0x2057A
	for i := 0; i < steps; i++ {
		if cpu.PC == sentinel {
			return
		}
		if t, ok := traps[cpu.PC]; ok {
			t()
			continue
		}
		if cpu.Halted {
			m.logf("HALTED in $2057A at $%06X: %s", cpu.PC, cpu.HaltReason)
			return
		}
		cpu.Step()
	}
}

// callRoutineArg runs a subroutine at datBase+off with one stack argument.
func (m *machine) callRoutineArg(cpu *m68k.CPU, traps map[uint32]func(), off, arg uint32, steps int) {
	sp := uint32(stackTop) - 4
	m.w32(sp, arg)
	sp -= 4
	m.w32(sp, sentinel)
	cpu.A[7] = sp
	cpu.PC = datBase + off
	for i := 0; i < steps; i++ {
		if cpu.PC == sentinel {
			return
		}
		if t, ok := traps[cpu.PC]; ok {
			t()
			continue
		}
		if cpu.Halted {
			m.logf("HALT in $%X at $%06X: %s", off, cpu.PC, cpu.HaltReason)
			return
		}
		cpu.Step()
	}
}

// callRoutine runs a subroutine at datBase+off until it RTSs to the sentinel.
func (m *machine) callRoutine(cpu *m68k.CPU, traps map[uint32]func(), off uint32, steps int) {
	cpu.A[7] = stackTop - 4
	m.w32(cpu.A[7], sentinel)
	cpu.PC = datBase + off
	itrace := os.Getenv("ITRACE") != ""
	for i := 0; i < steps; i++ {
		if cpu.PC == sentinel {
			return
		}
		if itrace && i < 600 {
			fmt.Fprintf(os.Stderr, "%3d PC=$%06X op=$%04X\n", i, cpu.PC, m.r16(cpu.PC))
		}
		if t, ok := traps[cpu.PC]; ok {
			t()
			continue
		}
		if cpu.Halted {
			m.logf("HALTED at $%06X: %s", cpu.PC, cpu.HaltReason)
			return
		}
		cpu.Step()
	}
	m.logf("step budget exhausted at $%06X", cpu.PC)
}

func (m *machine) buildTraps(cpu *m68k.CPU) map[uint32]func() {
	t := map[uint32]func(){}
	ex := func(off uint32, f func()) { t[execBase-off] = f }
	ds := func(off uint32, f func()) { t[dosBase-off] = f }
	au := func(off uint32, f func()) { t[audioBase-off] = f }

	// --- exec.library ---
	ex(294, func() { m.ret(cpu, 0x30000) })                    // FindTask
	ex(198, func() { a := m.alloc(cpu.D[0]); m.logf("AllocMem(%d,$%X)=$%X <-$%X", cpu.D[0], cpu.D[1], a, m.r32(cpu.A[7])-datBase); m.ret(cpu, a) })
	ex(210, func() { m.ret(cpu, 0) })                          // FreeMem
	ex(552, func() { m.ret(cpu, libByName(m.cstr(cpu.A[1]))) }) // OpenLibrary
	ex(408, func() { m.ret(cpu, libByName(m.cstr(cpu.A[1]))) }) // OldOpenLibrary
	ex(414, func() { m.ret(cpu, 0) })                          // CloseLibrary
	ex(330, func() { m.ret(cpu, 0) })                          // AllocSignal
	ex(336, func() { m.ret(cpu, 0) })                          // FreeSignal
	ex(354, func() { m.ret(cpu, 0) })                          // AddPort
	ex(360, func() { m.ret(cpu, 0) })                          // RemPort
	ex(366, func() { m.msgq = append(m.msgq, cpu.A[1]); m.ret(cpu, 0) }) // PutMsg -> enqueue
	ex(372, func() { // GetMsg -> dequeue
		if len(m.msgq) == 0 {
			m.ret(cpu, 0)
			return
		}
		msg := m.msgq[0]
		m.msgq = m.msgq[1:]
		m.ret(cpu, msg)
	})
	ex(378, func() { m.ret(cpu, 0) }) // ReplyMsg
	ex(384, func() { m.ret(cpu, cpu.A[0]) }) // WaitPort -> return the port (a message is queued)
	ex(444, func() { // OpenDevice(d0=unit, a0=name, a1=ioreq, d1=flags)
		name := m.cstr(cpu.A[0])
		req := cpu.A[1]
		m.w32(req+0x14, audioBase) // io_Device  (+$14 = io_Device in IORequest)
		m.logf("OpenDevice(%q, unit %d, req $%X) = 0", name, cpu.D[0], req)
		m.ret(cpu, 0) // success
	})
	ex(450, func() { m.ret(cpu, 0) })                  // CloseDevice
	ex(456, func() { m.audioIO(cpu, cpu.A[1]); m.ret(cpu, 0) }) // DoIO(a1=req)
	ex(462, func() { m.audioIO(cpu, cpu.A[1]); m.ret(cpu, 0) }) // SendIO(a1=req)
	ex(468, func() { m.ret(cpu, 0) })                  // CheckIO
	ex(474, func() { m.ret(cpu, 0) })                  // WaitIO
	ex(480, func() { m.ret(cpu, 0) })                  // AbortIO

	// audio.device vectors (reached via MOVEA.l io_Device(a1),a6; JSR -30(a6))
	au(30, func() { m.audioIO(cpu, cpu.A[1]); m.ret(cpu, 0) }) // BeginIO
	au(36, func() { m.ret(cpu, 0) })                           // AbortIO

	// --- dos.library ---
	ds(150, func() { m.loadSeg(cpu) })                            // LoadSeg
	ds(156, func() { m.ret(cpu, 0) })                             // UnLoadSeg
	ds(30, func() { m.open(cpu) })                                // Open
	ds(36, func() { delete(m.files, cpu.D[1]); m.ret(cpu, 0) })   // Close
	ds(42, func() { m.read(cpu) })                                // Read
	ds(84, func() { m.ret(cpu, boolU32(m.exists(m.cstr(cpu.D[1])))) }) // Lock
	ds(90, func() { m.ret(cpu, 0) })                              // UnLock
	return t
}

// audioIO inspects (and logs) an IOAudio request, capturing CMD_WRITE/PERVOL, and
// marks it done so the engine's wait loops complete.
func (m *machine) audioIO(cpu *m68k.CPU, req uint32) {
	cmd := m.r16(req + ioCommand)
	switch cmd {
	case 3, 10: // CMD_WRITE / ADCMD_PERVOL — a note
		m.cmds = append(m.cmds, audioCmd{
			frame:  m.frame,
			cmd:    cmd,
			chan_:  m.curChan,
			data:   m.r32(req + ioaData),
			length: m.r32(req + ioaLength),
			period: m.r16(req + ioaPeriod),
			vol:    m.r16(req + ioaVolume),
			cyc:    m.r16(req + ioaCycles),
		})
		// audio.device replies the finished request to the task's port -> next note.
		// Cap to keep the capture finite.
		if len(m.cmds) < 6000 {
			m.msgq = append(m.msgq, req)
		}
	default:
		m.logf("audio cmd $%X on req $%X (unit $%X)", cmd, req, m.r16(req+0x18))
	}
	m.ram[req+ioError] = 0
	// The engine spins on a relocated reply flag ($211E6) after the allocate; set it
	// so the handshake completes (we run synchronously, no real device interrupt).
	m.w16(datBase+0x211E6, 1)
}

func libByName(name string) uint32 {
	switch name {
	case "dos.library":
		return dosBase
	case "exec.library":
		return execBase
	}
	return 0x30000 // any other library: a harmless non-nil base
}

func boolU32(b bool) uint32 {
	if b {
		return 0x8000
	}
	return 0
}

func (m *machine) exists(name string) bool { _, err := m.vol.ReadFile(name); return err == nil }

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
	m.logf("Open(%q) = $%X (%d bytes)", name, h, len(data))
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
	m.ret(cpu, uint32(n))
}

func (m *machine) loadSeg(cpu *m68k.CPU) {
	name := m.cstr(cpu.D[1])
	if name == "" && m.course != "" { // game init bypassed: force the chosen course
		name = m.course
	}
	data, err := m.vol.ReadFile(name)
	if err != nil {
		m.logf("LoadSeg(%q) = 0 (missing)", name)
		m.ret(cpu, 0)
		return
	}
	base := (m.seg + 16) &^ 7
	prog, err := hunk.Load(data, base)
	if err != nil {
		m.logf("LoadSeg(%q) error: %v", name, err)
		m.ret(cpu, 0)
		return
	}
	copy(m.ram[base:], prog.Image)
	m.w32(base-4, 0) // seglist next = 0 (single node; parser uses h1's directory)
	m.seg = base + uint32(len(prog.Image)) + 16
	caller := m.r32(cpu.A[7])
	m.logf("LoadSeg(%q) = seglist @$%X (%d hunks, %d bytes) <- caller $%X", name, base, len(prog.Segments), len(prog.Image), caller-datBase)
	m.ret(cpu, (base-4)>>2) // BPTR
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "sndcapture:", err)
		os.Exit(1)
	}
}
