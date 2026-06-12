package main

// extract — extracts the program files from the Elite C64 tape image.
//
// The tape consists of one standard (kernal ROM format) boot file plus a
// number of segments in a custom fastloader format. The fastloader constantly
// rewrites itself (bit order, bits-per-byte, header size and sync handling
// all change while loading), so instead of hard-coding the protocol this tool
// *runs* the actual loader code from the tape on a small 6502 emulator wired
// to a pulse-stream tape model, and records every memory write the loader
// performs. The written memory regions are then saved as .prg files.
//
// Usage: extract [-o outdir] file.tap

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"stupidcoder.com/c64tools/c64"
	"stupidcoder.com/c64tools/cbmtape"
	"stupidcoder.com/c64tools/tap"
)

func main() {
	outdir := flag.String("o", "extracted", "output directory")
	maxInstr := flag.Uint64("max", 2_000_000_000, "instruction budget")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: extract [-o outdir] file.tap")
		os.Exit(1)
	}
	if err := run(flag.Arg(0), *outdir, *maxInstr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(path, outdir string, maxInstr uint64) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	img, err := tap.Parse(data)
	if err != nil {
		return err
	}
	pulses := img.Pulses
	segs := tap.Segmentize(pulses)
	fmt.Printf("%s: %d pulses, %d pause-delimited segments\n", filepath.Base(path), len(pulses), len(segs))

	if err := os.MkdirAll(outdir, 0o755); err != nil {
		return err
	}

	// 1. standard kernal-format files (the boot loader)
	cbmFiles := cbmtape.Files(cbmtape.ScanBlocks(pulses))
	if len(cbmFiles) == 0 {
		return fmt.Errorf("no standard-format boot file found")
	}
	lastPulse := 0
	for i, f := range cbmFiles {
		fmt.Println(f)
		name := fmt.Sprintf("%02d_cbm_%s_%04x.prg", i, sanitize(f.Name), f.Start)
		if err := writePRG(filepath.Join(outdir, name), f.Start, f.Data); err != nil {
			return err
		}
		lastPulse = f.LastPulse
	}

	// 2. run the fastloader from the boot file under emulation
	boot := cbmFiles[0]
	m := c64.New(pulses, lastPulse)
	d := &driver{m: m}
	newMachine(d)
	copy(m.RAM[boot.Start:], boot.Data)
	// state the kernal would leave behind after LOAD: $AC/$AD = end address,
	// entry happens via the IRQ vector restored from $029F/$02A0.
	m.RAM[0xAC] = byte(boot.End)
	m.RAM[0xAD] = byte(boot.End >> 8)
	entry := uint16(m.RAM[0x029F]) | uint16(m.RAM[0x02A0])<<8
	fmt.Printf("autostart entry (from kernal IRQ-vector save area $029F/$02A0): $%04X\n", entry)
	m.CPU.PC = entry
	m.Run(maxInstr)
	fmt.Printf("emulation stopped: %s\n", m.CPU.HaltReason)
	fmt.Printf("  %d instructions, %d/%d pulses consumed, %d writes logged\n",
		m.CPU.Instrs, m.PulsePos, len(pulses), len(m.Writes))
	fmt.Printf("  recent PCs: ")
	for _, pc := range m.TraceTail(48) {
		fmt.Printf("%04X ", pc)
	}
	fmt.Println()

	if err := os.WriteFile(filepath.Join(outdir, "memory_final.bin"), m.RAM[:], 0o644); err != nil {
		return err
	}

	// 3. coalesce tape-driven writes into blocks and files
	blocks := coalesce(m.Writes)
	rep := report(blocks, pulses, segs)
	fmt.Print(rep)
	if err := os.WriteFile(filepath.Join(outdir, "report.txt"), []byte(rep), 0o644); err != nil {
		return err
	}
	return emit(outdir, blocks, pulses, segs)
}

// block is a maximal run of consecutive-address tape-driven writes.
type block struct {
	Start      uint16
	Data       []byte
	FirstPulse int
	LastPulse  int
}

// coalesce turns the raw write log into address-contiguous blocks, keeping
// only writes that happened while the tape was actively being read (i.e.
// within a short instruction window after a pulse was consumed). This drops
// memory-clearing loops, stack traffic and other non-tape activity.
func coalesce(events []c64.WriteEvent) []block {
	var blocks []block
	var cur *block
	var lastInstr uint64
	for _, e := range events {
		if e.SincePulse > 2000 {
			continue // not driven by tape data
		}
		if e.Addr < 0x0200 {
			continue // zero page (loader bookkeeping) and stack traffic
		}
		if cur != nil && e.Addr == cur.Start+uint16(len(cur.Data)) && e.Instr-lastInstr < 20000 {
			cur.Data = append(cur.Data, e.Val)
			cur.LastPulse = e.PulseIdx
		} else {
			blocks = append(blocks, block{})
			cur = &blocks[len(blocks)-1]
			*cur = block{Start: e.Addr, Data: []byte{e.Val}, FirstPulse: e.PulseIdx, LastPulse: e.PulseIdx}
		}
		lastInstr = e.Instr
	}
	return blocks
}

func segOf(segs []tap.Segment, pulse int) int {
	for i, s := range segs {
		if pulse >= s.First && pulse <= s.Last+1 {
			return i
		}
	}
	return -1
}

func report(blocks []block, pulses []tap.Pulse, segs []tap.Segment) string {
	out := "\ntape blocks written by the loader (in load order):\n"
	for _, b := range blocks {
		out += fmt.Sprintf("  seg %2d  $%04X-$%04X  (%d bytes)\n",
			segOf(segs, b.FirstPulse), b.Start, int(b.Start)+len(b.Data), len(b.Data))
	}
	return out
}

// emit merges each tape segment's blocks into a final memory image (later
// writes win) and stores every contiguous region as a .prg file.
func emit(outdir string, blocks []block, pulses []tap.Pulse, segs []tap.Segment) error {
	bySeg := map[int]map[int]byte{}
	var order []int
	for _, b := range blocks {
		if b.Start < 0x0200 && len(b.Data) < 8 {
			continue // zero-page bookkeeping (block headers etc.)
		}
		s := segOf(segs, b.FirstPulse)
		if bySeg[s] == nil {
			bySeg[s] = map[int]byte{}
			order = append(order, s)
		}
		for i, v := range b.Data {
			bySeg[s][int(b.Start)+i] = v
		}
	}
	sort.Ints(order)
	fileNo := 1
	fmt.Println("\noutput files:")
	for _, s := range order {
		mem := bySeg[s]
		var addrs []int
		for a := range mem {
			addrs = append(addrs, a)
		}
		sort.Ints(addrs)
		for i := 0; i < len(addrs); {
			j := i
			for j+1 < len(addrs) && addrs[j+1] == addrs[j]+1 {
				j++
			}
			start, end := addrs[i], addrs[j]
			buf := make([]byte, end-start+1)
			for k := start; k <= end; k++ {
				buf[k-start] = mem[k]
			}
			kind := ""
			if start >= 0x0280 && start < 0x0400 {
				kind = "loader_" // self-modification of the loader itself
			}
			name := fmt.Sprintf("%02d_seg%02d_%s%04x.prg", fileNo+1, s, kind, start)
			if err := writePRG(filepath.Join(outdir, name), uint16(start), buf); err != nil {
				return err
			}
			fmt.Printf("  %s  $%04X-$%04X (%d bytes)\n", name, start, end+1, len(buf))
			fileNo++
			i = j + 1
		}
	}
	return nil
}

func writePRG(path string, start uint16, data []byte) error {
	buf := append([]byte{byte(start), byte(start >> 8)}, data...)
	return os.WriteFile(path, buf, 0o644)
}

func sanitize(s string) string {
	out := []rune{}
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return "noname"
	}
	return string(out)
}
