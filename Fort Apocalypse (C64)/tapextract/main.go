// tapextract extracts program files from a C64 TAP tape image, including
// the payload of the Novaload-family fastloader used by Fort Apocalypse.
//
// Usage: tapextract [-o outdir] [-dis] [-v] file.tap
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tapextract/fastload"
	"tapextract/kernal"
	"tapextract/mos6502"
	"tapextract/tap"
)

func main() {
	outDir := flag.String("o", "extracted", "output directory for .prg files")
	dis := flag.Bool("dis", false, "also write a disassembly of the loader code")
	verbose := flag.Bool("v", false, "verbose block listing")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: tapextract [-o outdir] [-dis] [-v] file.tap")
		os.Exit(2)
	}
	if err := run(flag.Arg(0), *outDir, *dis, *verbose); err != nil {
		fmt.Fprintln(os.Stderr, "tapextract:", err)
		os.Exit(1)
	}
}

func run(path, outDir string, dis, verbose bool) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	img, err := tap.Parse(raw)
	if err != nil {
		return err
	}
	fmt.Printf("%s: TAP v%d, %d pulses\n", filepath.Base(path), img.Version, len(img.Pulses))
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	// Pass 1: standard KERNAL blocks (header/data pairs, each recorded twice).
	blocks := kernal.ScanBlocks(img.Pulses)
	lastKernalPulse := 0
	var curName string
	var curHeader *kernal.Header
	fileNo := 0
	for _, b := range blocks {
		if b.EndPulse > lastKernalPulse {
			lastKernalPulse = b.EndPulse
		}
		kind := "data"
		if len(b.Payload) == 192 && (b.Payload[0] >= 1 && b.Payload[0] <= 5) {
			kind = "header"
		}
		if verbose {
			fmt.Printf("  kernal %-6s copy=%d pulses %d..%d len=%d checksum_ok=%v\n",
				kind, copyNo(b), b.StartPulse, b.EndPulse, len(b.Payload), b.ChecksumOK)
		}
		if !b.ChecksumOK {
			fmt.Printf("  WARNING: kernal block at pulse %d has a bad checksum\n", b.StartPulse)
		}
		if b.Repeat {
			continue // extract from first copies only
		}
		if kind == "header" {
			h, err := kernal.ParseHeader(b.Payload)
			if err != nil {
				return err
			}
			curHeader = h
			curName = strings.TrimRight(h.Name, " ")
			fmt.Printf("  kernal header: type=%d name=%q start=$%04X end=$%04X\n",
				h.Type, curName, h.StartAddr, h.EndAddr)
			if len(strings.TrimRight(string(h.Extra), "\x00")) > 0 && dis {
				p := filepath.Join(outDir, safeName(curName)+"-header-code.txt")
				lines := mos6502.Disassemble(h.Extra, 0x0351)
				if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
					return err
				}
				fmt.Printf("  wrote %s (loader IRQ handler from header extra bytes)\n", p)
			}
			continue
		}
		// Data block: pair it with the preceding header.
		load := uint16(0x0801)
		name := fmt.Sprintf("file%d", fileNo)
		if curHeader != nil {
			name = safeName(curName)
			if curHeader.Type == 3 {
				load = curHeader.StartAddr // absolute PRG
			}
			// Type 1 is relocated to the start of BASIC ($0801) by LOAD"",1.
		}
		fileNo++
		p := filepath.Join(outDir, name+".prg")
		if err := writePRG(p, load, b.Payload); err != nil {
			return err
		}
		fmt.Printf("  wrote %s ($%04X-$%04X, %d bytes)\n", p, load, int(load)+len(b.Payload)-1, len(b.Payload))
		if dis && curHeader != nil && len(b.Payload) > 14 {
			// Disassemble the ML part after the BASIC stub (SYS target).
			lines := mos6502.Disassemble(b.Payload[12:], load+12)
			dp := filepath.Join(outDir, name+"-stub-code.txt")
			if err := os.WriteFile(dp, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
				return err
			}
			fmt.Printf("  wrote %s (loader setup code)\n", dp)
		}
	}

	// Pass 2: fastloader stream(s) after the KERNAL part.
	base := safeName(curName)
	if base == "" {
		base = "fast"
	}
	from := lastKernalPulse
	streamNo := 0
	for from < len(img.Pulses) {
		res := fastload.Decode(img.Pulses, from)
		if len(res.Records) == 0 {
			break
		}
		streamNo++
		bad := 0
		for _, r := range res.Records {
			if verbose {
				fmt.Printf("  fast page $%02X at pulse %d checksum_ok=%v\n", r.Page, r.PulseIndex, r.ChecksumOK)
			}
			if !r.ChecksumOK {
				bad++
			}
		}
		fmt.Printf("  fastload stream %d: %d page records, %d bad checksums, terminated=%v\n",
			streamNo, len(res.Records), bad, res.Terminated)
		if res.Err != nil {
			fmt.Printf("  WARNING: %v\n", res.Err)
		}
		for _, rg := range res.Ranges() {
			p := filepath.Join(outDir, fmt.Sprintf("%s-fast-%04X.prg", base, rg.Start))
			if err := writePRG(p, rg.Start, rg.Data); err != nil {
				return err
			}
			fmt.Printf("  wrote %s ($%04X-$%04X, %d bytes)\n", p, rg.Start, int(rg.Start)+len(rg.Data)-1, len(rg.Data))
		}
		from = res.EndPulse
	}
	if streamNo == 0 {
		fmt.Println("  no fastloader stream found")
	}
	return nil
}

func copyNo(b kernal.Block) int {
	if b.Repeat {
		return 2
	}
	return 1
}

func safeName(s string) string {
	var out []rune
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			out = append(out, r)
		} else if r == ' ' {
			out = append(out, '_')
		}
	}
	return string(out)
}

func writePRG(path string, load uint16, data []byte) error {
	buf := make([]byte, 2+len(data))
	buf[0] = byte(load)
	buf[1] = byte(load >> 8)
	copy(buf[2:], data)
	return os.WriteFile(path, buf, 0o644)
}
