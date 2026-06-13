// unpack runs the disk's own decrypting LoadSeg-replacement, c/zzz, on the
// tools/m68k 68000 core to decode a packed Marble Madness file (c/xxx, the
// second-stage loader, or c/MarbleMadness!.dat, the game). We load c/zzz at a
// high base, trap its six AmigaDOS/Exec stubs (_Open/_Read/_Close/_AllocMem/
// _FreeMem/_FindTask), stream the packed bytes through _Read, and capture the
// segments it AllocMems. Phase 1 calls the key-init (sub_$D06: AllocMem a 55-
// long table, seed $57319753 via the x31 hash in sub_BEC, XOR a caller key-
// array, then the protection sub_DAA); phase 2 calls decrunch-load sub_$2C8.
//
// DECODE (fully reversed): the first file longword ($000003F3 HUNK_HEADER) and
// every hunk-block TYPE longword are stored plaintext; all structural fields
// and CODE/DATA bodies are XORed with an additive lagged-Fibonacci keystream
// over the table (gen at $40EAC). The keystream is bit-exact reproducible here
// (verified vs an independent generator) and c/xxx decrypts to a clean 22-hunk
// HUNK_HEADER.
//
// LIMIT (the copy protection): sub_DAA mixes the host's low-memory CPU
// exception/TRAP vectors ($8-$38, $80-$BC; (vec>>16)&0xFF) into table entries
// 10-16 and 32-47. Those are Kickstart-ROM values, absent from the disk. Our
// emulated vectors are zero, so the table is correct for ~the first 57
// keystream words (header + first hunk + a few reloc offsets all decode), then
// the corruption propagates and bodies/relocs decode to garbage. The trap
// vectors take many distinct ROM-dependent values (not one constant), so the
// payload cannot be fully decoded from the disk image alone.
//
// Usage: unpack disk.adf [innerpath] [-o outdir] [-count N] [-dumpks N] [-trace]
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
	base      = 0x040000 // c/zzz load base (clear of the absolute $4 ExecBase ref)
	stackTop  = 0x090000
	heapStart = 0x100000
	fnamePtr  = 0x000200 // a dummy filename string (_Open ignores it)
	sentinel  = 0xFFFFF0 // return address that marks "decrunch returned"
	ramSize   = 0x400000
)

// zzz stub entry addresses (from its HUNK_SYMBOL table), relative to base.
const (
	sOpen     = 0xEE4
	sClose    = 0xF00
	sRead     = 0xF14
	sAllocMem = 0xF30
	sFreeMem  = 0xF48
	sFindTask = 0xF60
	decrunch  = 0x2C8  // sub_$2C8: decrunch-load(filename) -> seglist
	keyinit   = 0xD06  // sub_$D06: AllocMem table, seed (sub_BEC), XOR keyarray, sub_DAA
	keyArray  = 0x0300 // caller key-array (arg0->C): loaded from c/sigfile
	sentinel2 = 0xFFFFE0
)

type machine struct {
	ram     []byte
	packed  []byte
	rpos    int         // read position in the packed file
	heap    uint32      // bump allocator
	allocs  [][2]uint32 // (addr,size) of every AllocMem
	trace   bool
	ksCalls int
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

func main() {
	out := flag.String("o", "unpacked", "output directory for the decrunched segments")
	trace := flag.Bool("trace", false, "log every trapped system call")
	keycount := flag.Int("count", -1, "keyarray longword count (arg0->8); -1 = len(sigfile)/4")
	dumpks := flag.Int("dumpks", 0, "dump N keystream words to /tmp/ks.bin and exit")
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: unpack disk.adf [innerpath] [-o outdir] [-trace]")
		os.Exit(2)
	}
	inner := "c/MarbleMadness!.dat"
	if len(args) >= 2 {
		inner = args[1]
	}

	image, err := os.ReadFile(args[0])
	must(err)
	vol, err := adf.Open(image)
	must(err)
	zzzData, err := vol.ReadFile("c/zzz")
	must(err)
	packed, err := vol.ReadFile(inner)
	must(err)

	sig, err := vol.ReadFile("c/sigfile")
	must(err)

	prog, err := hunk.Load(zzzData, base)
	must(err)

	m := &machine{ram: make([]byte, ramSize), packed: packed, heap: heapStart, trace: *trace}
	copy(m.ram[base:], prog.Image)
	m.w32(4, 0x1000) // a plausible ExecBase pointer at $4

	cpu := m68k.NewCPU(m)

	traps := map[uint32]func(){
		base + sOpen:     func() { m.ret(cpu, 1) },      // handle = 1
		base + sClose:    func() { m.ret(cpu, 1) },      //
		base + sFindTask: func() { m.ret(cpu, 0x3000) }, // fake task
		base + sFreeMem:  func() { m.ret(cpu, 0) },      // keep the memory
		base + sAllocMem: func() { m.allocMem(cpu) },    //
		base + sRead:     func() { m.read(cpu) },        //
	}

	// Phase 1: build the keystream table. sub_$D06 AllocMems 220 bytes, seeds it
	// (sub_BEC: table[0]=$57319753, 55 entries via the x31 hash), XORs in the
	// caller key-array (keyCount longwords), runs the machine-state protection
	// (sub_DAA), and records table pointers in the globals ($40EA0/4/8) the
	// decruncher reads. sub_DAA reads the host CPU exception/TRAP vectors, which
	// are zero in our low memory -- correct for the first ~57 keystream words but
	// diverging from a real Kickstart afterwards (see the file header comment).
	copy(m.ram[keyArray:], sig)
	count := uint32(len(sig) / 4)
	if *keycount >= 0 {
		count = uint32(*keycount)
	}
	m.call(cpu, base+keyinit, sentinel2, keyArray, count)
	m.run(cpu, traps, sentinel2, *trace)

	if *dumpks > 0 {
		ks := make([]uint32, *dumpks)
		for k := range ks {
			m.call(cpu, base+0xEAC, sentinel2)
			m.run(cpu, traps, sentinel2, false)
			ks[k] = cpu.D[0]
		}
		f, _ := os.Create("/tmp/ks.bin")
		binary.Write(f, binary.BigEndian, ks)
		f.Close()
		fmt.Printf("dumped %d keystream words to /tmp/ks.bin (first: %08X %08X %08X)\n",
			len(ks), ks[0], ks[1], ks[2])
		return
	}

	// Phase 2: decrunch-load(filename). filename != 0 makes it _Open the packed
	// file; we trap _Open/_Read to stream the bytes we already have in memory.
	copy(m.ram[fnamePtr:], []byte("x\x00"))
	m.call(cpu, base+decrunch, sentinel, fnamePtr)
	m.run(cpu, traps, sentinel, *trace)

	report(m, cpu, *out)
}

// call sets up a C-convention call frame at the given entry: args pushed
// right-to-left under a sentinel return address, then PC at the entry.
func (m *machine) call(cpu *m68k.CPU, entry, retAddr uint32, args ...uint32) {
	sp := uint32(stackTop)
	for i := len(args) - 1; i >= 0; i-- {
		sp -= 4
		m.w32(sp, args[i])
	}
	sp -= 4
	m.w32(sp, retAddr)
	cpu.A[7] = sp
	cpu.PC = entry
}

// run steps the CPU until PC reaches stopAt, servicing trapped syscalls.
func (m *machine) run(cpu *m68k.CPU, traps map[uint32]func(), stopAt uint32, trace bool) {
	for steps := 0; steps < 40_000_000; steps++ {
		if cpu.PC == stopAt {
			return
		}
		if trace {
			switch cpu.PC {
			case base + 0x3DE: // hunk-block dispatch: d0 = (type-$3E7)<<2
				fmt.Fprintf(os.Stderr, "dispatch type=$%X at input pos %d\n",
					(cpu.D[0]>>2)+0x3E7, m.rpos)
			case base + 0x2C4: // readlong: decrypted structural longword in (a0)
				if m.ksCalls < 70 {
					fmt.Fprintf(os.Stderr, "  readlong[%d] = $%08X\n", m.ksCalls, m.r32(m.r32(cpu.A[6]+0xC)))
				}
			case base + 0xEAC: // keystream advance
				m.ksCalls++
			}
		}
		if t, ok := traps[cpu.PC]; ok {
			t()
			continue
		}
		if cpu.Halted {
			fmt.Fprintf(os.Stderr, "halted at $%06X: %s\n", cpu.PC, cpu.HaltReason)
			os.Exit(1)
		}
		cpu.Step()
	}
	fmt.Fprintf(os.Stderr, "step budget exhausted at pc $%06X\n", cpu.PC)
	os.Exit(1)
}

// arg returns the i-th C-convention stack argument at a trap stub entry, where
// $0(a7) is the return address and $4(a7) is the first argument.
func (m *machine) arg(cpu *m68k.CPU, i int) uint32 { return m.r32(cpu.A[7] + 4 + uint32(i)*4) }

// ret sets d0 and returns from a trapped stub (pop the return address).
func (m *machine) ret(cpu *m68k.CPU, v uint32) {
	cpu.D[0] = v
	cpu.PC = m.r32(cpu.A[7])
	cpu.A[7] += 4
}

func (m *machine) allocMem(cpu *m68k.CPU) {
	size := (m.arg(cpu, 0) + 7) &^ 7 // longword-ish align
	addr := m.heap
	m.heap += size
	m.allocs = append(m.allocs, [2]uint32{addr, size})
	for i := uint32(0); i < size; i++ {
		m.ram[addr+i] = 0 // MEMF_CLEAR
	}
	if m.trace {
		fmt.Fprintf(os.Stderr, "AllocMem(%d) = $%06X\n", size, addr)
	}
	m.ret(cpu, addr)
}

func (m *machine) read(cpu *m68k.CPU) {
	buf, n := m.arg(cpu, 1), int(m.arg(cpu, 2))
	if m.rpos+n > len(m.packed) {
		n = len(m.packed) - m.rpos
	}
	copy(m.ram[buf:int(buf)+n], m.packed[m.rpos:m.rpos+n])
	m.rpos += n
	if m.trace {
		fmt.Fprintf(os.Stderr, "Read(buf=$%06X, %d) -> %d (pos %d/%d)\n", buf, m.arg(cpu, 2), n, m.rpos, len(m.packed))
	}
	m.ret(cpu, uint32(n))
}

func report(m *machine, cpu *m68k.CPU, outdir string) {
	fmt.Printf("decrunch returned d0=$%06X; %d AllocMem regions, %d/%d input bytes read\n",
		cpu.D[0], len(m.allocs), m.rpos, len(m.packed))
	sort.Slice(m.allocs, func(i, j int) bool { return m.allocs[i][1] > m.allocs[j][1] })
	os.MkdirAll(outdir, 0o755)
	for i, a := range m.allocs {
		tag := ""
		if a[1] >= 1024 {
			name := fmt.Sprintf("%s/seg%02d_%06X_%d.bin", outdir, i, a[0], a[1])
			os.WriteFile(name, m.ram[a[0]:a[0]+a[1]], 0o644)
			tag = " -> " + name
		}
		fmt.Printf("  alloc %2d  $%06X  %8d bytes%s\n", i, a[0], a[1], tag)
	}
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "unpack:", err)
		os.Exit(1)
	}
}
