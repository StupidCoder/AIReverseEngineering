// sndoracle drives Marble Madness's REAL sound code to capture a course's
// ground-truth audio stream. Unlike sndcapture (which only primed the engine),
// this models the two-thread, message-passing reality:
//
//   - the SfxTask ($20A3C) runs as its own cooperative context (snapshot/restore
//     of the m68k registers; shared RAM);
//   - exec ports are Go-side keyed message queues; PutMsg/GetMsg/WaitPort/ReplyMsg
//     are trapped at the sound module's own C wrappers (robust to the wrappers'
//     2-byte alignment), and a WaitPort on an empty port *yields* to the scheduler;
//   - timer.device (60 Hz, tv_micro $411A) and audio.device are a discrete-event
//     reply timeline on a virtual clock: a CMD_WRITE schedules its "note finished"
//     reply at now + cycles*length*period/3546895 s (PAL Paula); a timer
//     TR_ADDREQUEST reschedules the 60 Hz envelope tick.
//
// Bootstrap: load .dat + a course *Snd, point the directory global $21DD4 at it,
// run song_init (spawns the task via the trapped AddTask), let the task reach its
// WaitPort, then run the SfxTask startup ($20CAC) and play_sfx ($21ADC) for a
// soundID. Every CMD_WRITE / PERVOL the engine issues is logged with its virtual
// timestamp — the score a Go reimplementation must reproduce.
//
// Usage: sndoracle disk.adf decrypted.dat.hunk [-course prcsnd] [-id N] [-secs S]
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"sort"

	"stupidcoder.com/tools/amiga/adf"
	"stupidcoder.com/tools/amiga/hunk"
	"stupidcoder.com/tools/m68k"
)

const (
	execBase  = 0x10000
	dosBase   = 0x20000
	audioBase = 0x40000
	timerBase = 0x50000
	heap0     = 0x70000
	datBase   = 0x100000
	sndBase   = 0x400000 // loaded *Snd seglist image
	taskStack = 0x3E0000
	gameStack = 0x3F0000
	ramSize   = 0x1000000
	sentinel  = 0xDEADBEEF

	// sound-module C wrappers (trapped by address)
	wPutMsg     = 0x2499C
	wGetMsg     = 0x249B2
	wReplyMsg   = 0x249C8
	wWaitPort   = 0x249DC
	wOpenDevice = 0x249F0
	wBeginIO    = 0x24A3C // async BeginIO
	wDoIO       = 0x24A10 // synchronous DoIO/allocate
	wLoadSegSnd = 0x24874
	wAddTask    = 0x248D4
	wAllocMem   = 0x2488A
	wFindTask   = 0x24904

	// IOAudio / IORequest fields
	ioMsgRepl = 0x0E
	ioDevice  = 0x14
	ioCommand = 0x1C
	ioError   = 0x1F
	ioaData   = 0x22
	ioaLength = 0x26
	ioaPeriod = 0x2A
	ioaVolume = 0x2C
	ioaCycles = 0x2E

	paulaPAL = 3546895.0
)

// regs is a saved cooperative context.
type regs struct {
	D    [8]uint32
	A    [8]uint32
	PC   uint32
	X, N, Z, V, C bool
}

type ctx struct {
	name    string
	regs    regs
	alive   bool
	blocked uint32 // port address this ctx is parked on (0 = runnable)
}

type ev struct {
	t    float64
	port uint32 // deliver msg to this port
	msg  uint32
	kind string
}

type note struct {
	t                int // microframe (us)
	cmd              uint16
	data, length     uint32
	period, vol, cyc uint16
}

type machine struct {
	ram   []byte
	vol   *adf.Volume
	heap  uint32
	clock float64 // virtual seconds

	ports map[uint32][]uint32 // port addr -> queued msg addrs
	tl    []ev                 // device reply timeline
	notes []note
	log   []string
	trace bool

	cur  *ctx
	task *ctx
	game *ctx
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

func (m *machine) logf(f string, a ...interface{}) {
	s := fmt.Sprintf("[%8.4f] %s", m.clock, fmt.Sprintf(f, a...))
	m.log = append(m.log, s)
	if m.trace {
		fmt.Fprintln(os.Stderr, s)
	}
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

// --- context save/restore ---
func (m *machine) save(cpu *m68k.CPU, r *regs) {
	r.D = cpu.D
	r.A = cpu.A
	r.PC = cpu.PC
	r.X, r.N, r.Z, r.V, r.C = cpu.X, cpu.N, cpu.Z, cpu.V, cpu.C
}
func (m *machine) load(cpu *m68k.CPU, r *regs) {
	cpu.D = r.D
	cpu.A = r.A
	cpu.PC = r.PC
	cpu.X, cpu.N, cpu.Z, cpu.V, cpu.C = r.X, r.N, r.Z, r.V, r.C
}

// ret pops the return address and sets d0 (a trapped wrapper returning v).
func (m *machine) ret(cpu *m68k.CPU, v uint32) {
	cpu.D[0] = v
	cpu.PC = m.r32(cpu.A[7])
	cpu.A[7] += 4
}

func (m *machine) stackArg(cpu *m68k.CPU, off uint32) uint32 { return m.r32(cpu.A[7] + 4 + off) }

func main() {
	course := flag.String("course", "prcsnd", "*Snd file (course sound bank)")
	id := flag.Int("id", -1, "soundID to trigger (-1 = enumerate all)")
	secs := flag.Float64("secs", 12, "virtual seconds to render after the trigger")
	trace := flag.Bool("trace", false, "log every trapped call")
	flag.Parse()
	if flag.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "usage: sndoracle disk.adf decrypted.dat.hunk [-course prcsnd] [-id N] [-secs S]")
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
		ports: map[uint32][]uint32{},
		trace: *trace,
	}
	copy(m.ram[datBase:], prog.Image)
	m.w32(4, execBase)
	m.w32(datBase+0x1F4, execBase) // cached ExecBase global
	m.w32(datBase+0x1F8, dosBase)  // cached DosBase global

	// Load the course *Snd ourselves and point the directory global $21DD4 at it.
	sndData, err := vol.ReadFile(*course)
	must(err)
	sprog, err := hunk.Load(sndData, sndBase)
	must(err)
	copy(m.ram[sndBase:], sprog.Image)
	dir := sprog.Segments[firstData(sprog)].Base
	m.w32(datBase+0x21DD4, dir)
	m.ram[datBase+0x21DD0] = 1 // sfx enabled
	m.logf("loaded %s: dir @$%X (%d hunks) count=%d", *course, dir, len(sprog.Segments), m.r16(dir))
	for _, rid := range []uint32{7, 30, 31} {
		r := dir + 2 + rid*8
		m.logf("  rec%d: op=%d desc=$%X", rid, m.r16(r), m.r32(r+4))
	}

	cpu := m68k.NewCPU(m)
	m.task = &ctx{name: "task"}
	m.game = &ctx{name: "game", alive: true}

	traps := m.buildTraps(cpu)

	// Phase A: song_init clears state + AddTasks the SfxTask (trapped -> spawn ctx).
	m.runOnce(cpu, traps, m.game, gameStack, datBase+0x20B94)
	if !m.task.alive {
		m.logf("WARNING: AddTask not seen; task not spawned")
	}
	m.game.alive = false // game finished song_init; revive it for each phase below

	// Prime the task: run it until it sets up its port + devices and blocks on WaitPort,
	// so the game's messages have a port to land on.
	m.cur = nil
	m.switchTo(cpu, m.task)
	m.runCtx(cpu, traps)
	m.logf("task primed; port $20FA8=$%X blocked=$%X", m.r32(datBase+0x20FA8), m.task.blocked)

	// Phase B: SfxTask startup handshake (allocate channels, start the 60Hz timer).
	m.save(cpu, &m.task.regs) // sync the (blocked) task's regs before leaving it
	m.cur = nil
	m.game.regs.PC = datBase + 0x20CAC
	m.game.regs.A[7] = gameStack - 4
	m.w32(gameStack-4, sentinel)
	m.game.blocked = 0
	m.game.alive = true
	m.schedule(cpu, traps)

	// Phase C: trigger.
	d2 := m.r32(datBase + 0x21DD4)
	m.logf("pre-trigger: $21DD4=$%X count=%d rec7.desc=$%X", d2, m.r16(d2), m.r32(d2+2+7*8+4))
	ids := []int{*id}
	if *id < 0 {
		n := int(m.r16(dir))
		ids = nil
		for i := 0; i < n; i++ {
			ids = append(ids, i)
		}
		m.logf("directory has %d entries; triggering each", n)
	}
	for _, sid := range ids {
		before := len(m.notes)
		if m.cur != nil {
			m.save(cpu, &m.cur.regs)
		}
		m.cur = nil
		m.game.regs.PC = datBase + 0x21ADC
		m.game.regs.A[7] = gameStack - 8
		m.w32(gameStack-8, sentinel)
		m.w32(gameStack-4, uint32(sid)) // arg: soundID
		m.game.blocked = 0
		m.game.alive = true
		m.schedule(cpu, traps)
		// run the dual clock for a while so the score plays out
		m.runClock(cpu, traps, m.clock+*secs)
		m.logf("=== soundID %d: %d notes ===", sid, len(m.notes)-before)
	}

	fmt.Println("=== call log ===")
	for _, l := range m.log {
		fmt.Println(l)
	}
	fmt.Printf("\n=== %d audio commands ===\n", len(m.notes))
	for _, n := range m.notes {
		fmt.Printf("t%-7d cmd=$%X data=$%X len=%d per=%d vol=%d cyc=%d\n",
			n.t, n.cmd, n.data, n.length, n.period, n.vol, n.cyc)
	}
}

// runOnce runs a fresh context from entry until it RTSs to the sentinel.
func (m *machine) runOnce(cpu *m68k.CPU, traps map[uint32]func() bool, c *ctx, sp, entry uint32) {
	c.regs = regs{}
	c.regs.A[7] = sp - 4
	m.w32(sp-4, sentinel)
	c.regs.PC = entry
	c.alive = true
	c.blocked = 0
	m.cur = c
	m.load(cpu, &c.regs)
	for i := 0; i < 200_000_000; i++ {
		if cpu.PC == sentinel {
			return
		}
		if f, ok := traps[cpu.PC]; ok {
			if f() { // f returns true if it blocked (shouldn't here)
				m.logf("unexpected block in %s", c.name)
				return
			}
			continue
		}
		if cpu.Halted {
			m.logf("HALT %s @$%06X: %s", c.name, cpu.PC, cpu.HaltReason)
			return
		}
		cpu.Step()
	}
	m.logf("runOnce budget exhausted in %s @$%06X", c.name, cpu.PC)
}

// schedule runs the cooperative scheduler until the game context RTSs to sentinel.
func (m *machine) schedule(cpu *m68k.CPU, traps map[uint32]func() bool) {
	m.switchTo(cpu, m.game)
	for {
		done := m.runCtx(cpu, traps)
		if done && m.cur == m.game {
			return // game finished its phase
		}
		// current ctx blocked (or finished); pick another runnable ctx
		if !m.pickRunnable(cpu) {
			if !m.advanceClock(cpu) {
				return // nothing left to do
			}
		}
	}
}

// runClock keeps the scheduler going (devices + task) until the virtual clock
// reaches limit or everything is quiescent.
func (m *machine) runClock(cpu *m68k.CPU, traps map[uint32]func() bool, limit float64) {
	for m.clock < limit {
		m.runCtx(cpu, traps)
		if !m.pickRunnable(cpu) {
			if !m.advanceClock(cpu) {
				return
			}
		}
	}
}

// runCtx runs m.cur until it blocks or RTSs to sentinel. Returns true if it RTSd.
func (m *machine) runCtx(cpu *m68k.CPU, traps map[uint32]func() bool) bool {
	if m.cur == nil || !m.cur.alive {
		return true
	}
	dbg := os.Getenv("DBG") != ""
	for i := 0; i < 50_000_000; i++ {
		if cpu.PC == sentinel {
			m.cur.alive = false
			return true
		}
		if m.cur.blocked != 0 {
			return false
		}
		if dbg && i < 2000 {
			fmt.Fprintf(os.Stderr, "%s %3d PC=$%06X op=$%04X a7=$%X\n", m.cur.name, i, cpu.PC, m.r16(cpu.PC), cpu.A[7])
		}
		if f, ok := traps[cpu.PC]; ok {
			if f() { // blocked inside the trap (WaitPort on empty)
				return false
			}
			continue
		}
		if cpu.Halted {
			m.logf("HALT %s @$%06X: %s", m.cur.name, cpu.PC, cpu.HaltReason)
			m.cur.alive = false
			return true
		}
		cpu.Step()
	}
	m.logf("runCtx budget exhausted in %s @$%06X", m.cur.name, cpu.PC)
	m.cur.alive = false
	return true
}

func (m *machine) switchTo(cpu *m68k.CPU, c *ctx) {
	if m.cur != nil && m.cur != c {
		m.save(cpu, &m.cur.regs)
	}
	m.cur = c
	m.load(cpu, &c.regs)
}

// pickRunnable switches to a runnable (alive, unblocked) context other than cur.
func (m *machine) pickRunnable(cpu *m68k.CPU) bool {
	for _, c := range []*ctx{m.task, m.game} {
		if c != nil && c.alive && c.blocked == 0 {
			if c != m.cur {
				m.switchTo(cpu, c)
			}
			return true
		}
	}
	return false
}

// advanceClock delivers the earliest device reply, waking any waiter.
func (m *machine) advanceClock(cpu *m68k.CPU) bool {
	if len(m.tl) == 0 {
		return false
	}
	sort.SliceStable(m.tl, func(i, j int) bool { return m.tl[i].t < m.tl[j].t })
	e := m.tl[0]
	m.tl = m.tl[1:]
	m.clock = e.t
	m.deliver(cpu, e.port, e.msg)
	return m.pickRunnable(cpu)
}

// deliver puts msg on port and unblocks a ctx waiting on it.
func (m *machine) deliver(cpu *m68k.CPU, port, msg uint32) {
	m.ports[port] = append(m.ports[port], msg)
	for _, c := range []*ctx{m.task, m.game} {
		if c != nil && c.blocked == port {
			c.blocked = 0
		}
	}
}

func (m *machine) buildTraps(cpu *m68k.CPU) map[uint32]func() bool {
	t := map[uint32]func() bool{}
	ok := func(f func()) func() bool { return func() bool { f(); return false } }
	// Trap at the exec LVO targets (execBase-N): the C wrappers load a6=ExecBase and
	// JSR -N(a6) after copying the stack args into a0/a1/d0/d1, so registers are set.
	ex := func(n uint32, f func() bool) { t[execBase-n] = f }
	dev := func(base, n uint32, f func() bool) { t[base-n] = f }

	// Default: every exec jump-table slot is a harmless no-op (returns d0=0). This
	// stops incidental calls (Forbid/Permit, list ops AddHead/Remove/Enqueue, …) from
	// running off into zeroed RAM. The handlers below override the ones that matter.
	for k := uint32(1); k <= 170; k++ {
		ex(6*k, ok(func() { m.ret(cpu, 0) }))
	}

	ex(198, ok(func() { m.ret(cpu, m.alloc(cpu.D[0])) }))     // AllocMem(d0=size)
	ex(210, ok(func() { m.ret(cpu, 0) }))                     // FreeMem
	ex(294, ok(func() { m.ret(cpu, 0x30000) }))               // FindTask
	ex(282, ok(func() { // AddTask(a1=task,a2=initPC,a3=finalPC) -> spawn SfxTask ctx
		m.task.alive = true
		m.task.regs = regs{}
		m.task.regs.PC = cpu.A[2]
		m.task.regs.A[7] = taskStack - 4
		m.w32(taskStack-4, sentinel)
		m.logf("AddTask -> SfxTask spawned @$%X", cpu.A[2])
		m.ret(cpu, 0)
	}))
	ex(330, ok(func() { m.ret(cpu, 0x10) })) // AllocSignal -> a fake signal bit (#4)
	ex(336, ok(func() { m.ret(cpu, 0) }))    // FreeSignal
	ex(318, ok(func() { m.ret(cpu, cpu.D[0]) })) // Wait(d0=mask) -> pretend signalled
	ex(324, ok(func() { m.ret(cpu, 0) }))    // Signal
	ex(354, ok(func() { m.ret(cpu, 0) }))    // AddPort
	ex(360, ok(func() { m.ret(cpu, 0) }))    // RemPort
	ex(552, ok(func() { m.ret(cpu, 0x31000) })) // OpenLibrary
	ex(408, ok(func() { m.ret(cpu, 0x31000) })) // OldOpenLibrary
	ex(414, ok(func() { m.ret(cpu, 0) }))    // CloseLibrary

	ex(366, ok(func() { m.putMsg(cpu.A[0], cpu.A[1]); m.ret(cpu, 0) })) // PutMsg(a0=port,a1=msg)
	ex(372, ok(func() { // GetMsg(a0=port)
		port := cpu.A[0]
		q := m.ports[port]
		if len(q) == 0 {
			m.ret(cpu, 0)
			return
		}
		m.ports[port] = q[1:]
		m.ret(cpu, q[0])
	}))
	ex(378, ok(func() { msg := cpu.A[1]; m.deliver(cpu, m.r32(msg+ioMsgRepl), msg); m.ret(cpu, 0) })) // ReplyMsg
	t[execBase-384] = func() bool { // WaitPort(a0=port): block if empty
		port := cpu.A[0]
		if len(m.ports[port]) > 0 {
			m.ret(cpu, m.ports[port][0])
			return false
		}
		if m.trace {
			m.logf("%s WaitPort blocks on port $%X (caller $%X)", m.cur.name, port, m.r32(cpu.A[7])-datBase)
		}
		m.cur.blocked = port
		m.save(cpu, &m.cur.regs) // park with PC at the LVO so it retries on resume
		return true
	}

	ex(444, ok(func() { // OpenDevice(a0=name,a1=req,d0=unit,d1=flags)
		name := m.cstr(cpu.A[0])
		base := uint32(audioBase)
		if name == "timer.device" {
			base = timerBase
		}
		m.w32(cpu.A[1]+ioDevice, base)
		m.logf("OpenDevice(%q) req=$%X dev=$%X", name, cpu.A[1], base)
		m.ret(cpu, 0)
	}))
	ex(450, ok(func() { m.ret(cpu, 0) }))                                  // CloseDevice
	ex(456, ok(func() { m.beginIO(cpu, cpu.A[1], true); m.ret(cpu, 0) }))  // DoIO(a1=req)
	ex(462, ok(func() { m.beginIO(cpu, cpu.A[1], false); m.ret(cpu, 0) })) // SendIO(a1=req)
	ex(468, ok(func() { m.ret(cpu, 0) }))                                  // CheckIO
	ex(474, ok(func() { m.ret(cpu, 0) }))                                  // WaitIO
	ex(480, ok(func() { m.ret(cpu, 0) }))                                  // AbortIO

	// device BeginIO (reached via $24A3C: a6=io_Device; JSR -$1E(a6)); a1=req.
	dev(audioBase, 0x1E, ok(func() { m.beginIO(cpu, cpu.A[1], false); m.ret(cpu, 0) }))
	dev(timerBase, 0x1E, ok(func() { m.beginIO(cpu, cpu.A[1], false); m.ret(cpu, 0) }))
	return t
}

func (m *machine) putMsg(port, msg uint32) {
	m.ports[port] = append(m.ports[port], msg)
	for _, c := range []*ctx{m.task, m.game} {
		if c != nil && c.blocked == port {
			c.blocked = 0
		}
	}
	if m.trace {
		m.logf("PutMsg port=$%X msg=$%X cmd=%d", port, msg, m.r16(msg+ioCommand))
	}
}

// beginIO handles an IOAudio/timer request. sync=true completes it now.
func (m *machine) beginIO(cpu *m68k.CPU, req uint32, sync bool) {
	dev := m.r32(req + ioDevice)
	cmd := m.r16(req + ioCommand)
	m.ram[req+ioError] = 0
	if dev == timerBase {
		// TR_ADDREQUEST: tv_secs at +$20, tv_micro at +$24.
		secs := float64(m.r32(req+0x20)) + float64(m.r32(req+0x24))/1e6
		port := m.r32(req + ioMsgRepl)
		m.tl = append(m.tl, ev{t: m.clock + secs, port: port, msg: req, kind: "timer"})
		return
	}
	// audio.device
	switch cmd {
	case 3, 13: // CMD_WRITE / ADCMD_PERVOL
		n := note{
			t:      int(m.clock * 1e6),
			cmd:    cmd,
			data:   m.r32(req + ioaData),
			length: m.r32(req + ioaLength),
			period: m.r16(req + ioaPeriod),
			vol:    m.r16(req + ioaVolume),
			cyc:    m.r16(req + ioaCycles),
		}
		m.notes = append(m.notes, n)
		if cmd == 3 { // a played note finishes and replies after its duration
			port := m.r32(req + ioMsgRepl)
			dur := 0.0
			if n.period > 0 && n.cyc > 0 {
				dur = float64(n.cyc) * float64(n.length) * float64(n.period) / paulaPAL
			}
			if dur <= 0 {
				dur = 0.01 // looped/degenerate: nudge so it still chains
			}
			if len(m.notes) < 20000 {
				m.tl = append(m.tl, ev{t: m.clock + dur, port: port, msg: req, kind: "audio"})
			}
		}
	case 0x20: // ADCMD_ALLOCATE
		m.w16(datBase+0x211E6, 1) // the spin flag $20CAC waits on
		if !sync {
			port := m.r32(req + ioMsgRepl)
			m.tl = append(m.tl, ev{t: m.clock, port: port, msg: req, kind: "alloc"})
		}
	default:
		if m.trace {
			m.logf("audio cmd $%X req=$%X", cmd, req)
		}
		m.w16(datBase+0x211E6, 1)
	}
}

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

func firstData(p *hunk.Program) int {
	for i, s := range p.Segments {
		if s.Kind == "DATA" && s.Size > 0 {
			return i
		}
	}
	return 0
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "sndoracle:", err)
		os.Exit(1)
	}
}
