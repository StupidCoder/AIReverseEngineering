// codetracez80 is a recursive-descent ("code-tracing") Z80 disassembler. Starting
// from one or more entry points it follows every branch, jump, call and RST,
// marks which bytes are reachable code, and leaves everything else as data — so
// tables and graphics don't get mis-decoded as instructions. Indirect jumps
// (JP (HL)/(IX)/(IY)) and self-modified dispatch can't be followed statically;
// their jump tables are supplied with -table, and any unresolved ones are reported.
//
// It works on a single ≤64 KB Z80 address space (one cartridge bank configuration);
// banking is the caller's concern — assemble the image you want and give -load.
//
// Usage:
//
//	codetracez80 [-load HEX] -entry A,B,C [-table ADDR:N ...] [-annotate FILE] [-o out.asm] image.bin
//
// image.bin is raw Z80 code loaded at -load (hex, default 0). Addresses are hex.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"stupidcoder.com/tools/z80"
)

func main() {
	load := flag.String("load", "0", "load address of the binary (hex)")
	entry := flag.String("entry", "", "comma-separated entry addresses (hex)")
	var tables multiFlag
	flag.Var(&tables, "table", "jump table to seed as code, ADDR:N (N little-endian words); repeatable")
	annotate := flag.String("annotate", "", "annotations file: lines \"ADDR name description\" (# comments)")
	out := flag.String("o", "", "write disassembly to this file (default stdout)")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: codetracez80 [-load HEX] -entry A,B,C [-table ADDR:N ...] [-annotate FILE] [-o out] image.bin")
		os.Exit(2)
	}
	if err := run(flag.Arg(0), *load, *entry, tables, *annotate, *out); err != nil {
		fmt.Fprintln(os.Stderr, "codetracez80:", err)
		os.Exit(1)
	}
}

type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(s string) error {
	*m = append(*m, s)
	return nil
}

func hx(s string) (uint16, error) {
	v, err := strconv.ParseUint(strings.TrimPrefix(strings.TrimPrefix(s, "$"), "0x"), 16, 16)
	return uint16(v), err
}

type annot struct{ name, desc string }

// loadAnnotations reads "ADDR name rest-of-line-is-description" entries.
func loadAnnotations(path string) (map[uint16]annot, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := map[uint16]annot{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		a, err := hx(f[0])
		if err != nil {
			return nil, fmt.Errorf("annotations: bad address %q", f[0])
		}
		name := f[1]
		rest := strings.TrimSpace(line[len(f[0]):]) // after the address
		desc := strings.TrimSpace(rest[len(name):]) // after the name
		m[a] = annot{name: name, desc: desc}
	}
	return m, nil
}

func run(path, loadStr, entryStr string, tables multiFlag, annPath, outPath string) error {
	ann, err := loadAnnotations(annPath)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	base, err := hx(loadStr)
	if err != nil {
		return fmt.Errorf("bad -load %q: %v", loadStr, err)
	}
	mem := make([]byte, 0x10000)
	copy(mem[base:], raw)
	lo, hi := int(base), int(base)+len(raw)
	if hi > 0x10000 {
		hi = 0x10000
	}

	// Seed addresses: explicit entries plus every word in each -table.
	var seeds []uint16
	for _, s := range strings.Split(entryStr, ",") {
		if s == "" {
			continue
		}
		a, err := hx(s)
		if err != nil {
			return fmt.Errorf("bad -entry %q: %v", s, err)
		}
		seeds = append(seeds, a)
	}
	for _, t := range tables {
		parts := strings.SplitN(t, ":", 2)
		a, err := hx(parts[0])
		if err != nil || len(parts) != 2 {
			return fmt.Errorf("bad -table %q (want ADDR:N)", t)
		}
		n, _ := strconv.Atoi(parts[1])
		for i := 0; i < n; i++ {
			p := int(a) + i*2
			seeds = append(seeds, uint16(mem[p])|uint16(mem[p+1])<<8)
		}
	}

	tr := trace(mem, seeds)

	w := bufio.NewWriter(os.Stdout)
	if outPath != "" {
		f, err := os.Create(outPath)
		if err != nil {
			return err
		}
		defer f.Close()
		w = bufio.NewWriter(f)
	}
	defer w.Flush()
	emit(w, mem, lo, hi, tr, ann)

	code := 0
	for _, in := range tr.instr {
		code += in.Len
	}
	fmt.Fprintf(os.Stderr, "image $%04X-$%04X (%d bytes): %d code, %d data; %d routines, %d unresolved indirect jumps, %d stop-hits\n",
		lo, hi-1, hi-lo, code, (hi-lo)-code, len(tr.callers), len(tr.indirect), len(tr.stops))
	return nil
}

type traced struct {
	instr    map[uint16]z80.Inst // instruction at each start address
	covered  []bool              // byte reachable as code (incl. operands)
	callers  map[uint16]int      // CALL/RST target -> caller count
	indirect []uint16            // addresses of JP (HL)/(IX)/(IY)
	stops    []uint16            // addresses where a path hit an undefined opcode
}

func trace(mem []byte, seeds []uint16) *traced {
	t := &traced{instr: map[uint16]z80.Inst{}, covered: make([]bool, 0x10000), callers: map[uint16]int{}}
	work := append([]uint16(nil), seeds...)
	queued := map[uint16]bool{}
	for _, s := range seeds {
		queued[s] = true
	}
	push := func(a uint16) {
		if !queued[a] {
			queued[a] = true
			work = append(work, a)
		}
	}
	for len(work) > 0 {
		pc := work[len(work)-1]
		work = work[:len(work)-1]
		for {
			if _, done := t.instr[pc]; done {
				break // already traced this path
			}
			in := z80.Decode(mem[pc:], pc)
			t.instr[pc] = in
			for i := 0; i < in.Len; i++ {
				t.covered[int(pc)+i] = true
			}
			switch in.Flow {
			case z80.FlowBranch:
				push(in.Target)
				pc += uint16(in.Len)
			case z80.FlowCall:
				t.callers[in.Target]++
				push(in.Target)
				pc += uint16(in.Len)
			case z80.FlowJump:
				push(in.Target)
				goto pathEnd
			case z80.FlowReturn:
				goto pathEnd
			case z80.FlowIndJump:
				t.indirect = append(t.indirect, in.Addr)
				goto pathEnd
			case z80.FlowStop:
				t.stops = append(t.stops, in.Addr)
				goto pathEnd
			default: // FlowSeq
				pc += uint16(in.Len)
			}
		}
	pathEnd:
	}
	sort.Slice(t.indirect, func(i, j int) bool { return t.indirect[i] < t.indirect[j] })
	return t
}

// emit writes the listing: a header before each subroutine (CALL/RST target) or
// annotated address, the decoded instructions, and condensed .byte runs for data.
func emit(w *bufio.Writer, mem []byte, lo, hi int, t *traced, ann map[uint16]annot) {
	pos := lo
	for pos < hi {
		a := uint16(pos)
		an, named := ann[a]
		if in, ok := t.instr[a]; ok {
			switch {
			case t.callers[a] > 0 && named:
				fmt.Fprintf(w, "\n; ==== %s  $%04X  (%d caller%s) — %s ====\n", an.name, a, t.callers[a], plural(t.callers[a]), an.desc)
			case t.callers[a] > 0:
				fmt.Fprintf(w, "\n; ==== sub_%04X (%d caller%s) ====\n", a, t.callers[a], plural(t.callers[a]))
			case named:
				fmt.Fprintf(w, "\n; --- %s  $%04X — %s ---\n", an.name, a, an.desc)
			}
			raw := make([]string, in.Len)
			for i := 0; i < in.Len; i++ {
				raw[i] = fmt.Sprintf("%02X", mem[pos+i])
			}
			fmt.Fprintf(w, "%04X  %-11s %s\n", a, strings.Join(raw, " "), in.Text)
			pos += in.Len
			continue
		}
		if named {
			fmt.Fprintf(w, "\n; --- %s  $%04X — %s (data) ---\n", an.name, a, an.desc)
		}
		start := pos
		pos++
		for pos < hi {
			if _, ok := t.instr[uint16(pos)]; ok {
				break
			}
			if _, ok := ann[uint16(pos)]; ok {
				break
			}
			pos++
		}
		emitData(w, mem, start, pos)
	}
}

func emitData(w *bufio.Writer, mem []byte, start, end int) {
	for p := start; p < end; p += 16 {
		n := end - p
		if n > 16 {
			n = 16
		}
		bs := make([]string, n)
		asc := make([]byte, n)
		for i := 0; i < n; i++ {
			bs[i] = fmt.Sprintf("%02X", mem[p+i])
			c := mem[p+i]
			if c >= 0x20 && c < 0x7f {
				asc[i] = c
			} else {
				asc[i] = '.'
			}
		}
		fmt.Fprintf(w, "%04X  .byte %-47s ; %s\n", p, strings.Join(bs, " "), string(asc))
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
