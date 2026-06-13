// decode is a standalone, pure-Go reimplementation of c/zzz's decryption: it
// turns an encrypted Marble Madness loader file (c/xxx, the second-stage loader,
// or c/MarbleMadness!.dat, the game) back into a plain AmigaDOS hunk file. No
// emulator needed -- the copy-protection inputs c/zzz reads from live Kickstart
// 1.2 are baked in (captured via the FS-UAE GDB stub): every CPU exception/TRAP
// vector page byte = $FC, the launcher process tc_ExceptCode page = $FC and
// tc_TrapCode page = $FF.
//
// c/xxx uses an empty key array (count=0). The game .dat is decrypted in a second
// c/zzz pass keyed on a 20-long key array that the launcher XOR-mutates with a
// single 16-bit constant (checksum of the decrypted c/xxx ^ a c/zzz status word);
// -brute recovers that constant by trying all 65536 values and keeping the one
// whose output is a valid hunk file.
//
// Usage: decode disk.adf innerpath out.hunk [-keyconst HEX] [-count N] [-brute]
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"

	"stupidcoder.com/tools/amiga/adf"
	"stupidcoder.com/tools/amiga/hunk"
)

const (
	hCODE    = 0x3E9
	hDATA    = 0x3EA
	hBSS     = 0x3EB
	hRELOC32 = 0x3EC
	hSYMBOL  = 0x3F0
	hDEBUG   = 0x3F1
	hEND     = 0x3F2
	hHEADER  = 0x3F3
	hUNIT    = 0x3E7
	hNAME    = 0x3E8
)

// keyTable builds the 55-long key table: seed (x31 hash) -> caller key array XOR
// (count longs) -> sub_DAA copy-protection perturbation with the captured KS1.2
// inputs (all vectors $FC, tc_ExceptCode $FC, tc_TrapCode $FF -> entry31 ^= $F).
func keyTable(keyArray []uint32) []uint32 {
	t := make([]uint32, 55)
	t[0] = 0x57319753
	for i := uint32(1); i < 55; i++ {
		t[i] = t[i-1]*31 + i
	}
	for i := 0; i < len(keyArray) && i < 55; i++ {
		t[i] ^= keyArray[i]
	}
	const vec = 0xFC
	for i := 10; i <= 16; i++ {
		t[i] ^= vec
	}
	for i := 10; i <= 14; i++ {
		t[i] ^= vec
	}
	for i := 32; i <= 47; i++ {
		t[i] ^= vec
	}
	t[30] ^= 0xFC
	t[31] ^= 0x0F
	return t
}

type dec struct {
	in   []byte
	out  []byte
	pos  int
	t    []uint32
	p, q int
}

func (d *dec) ks() uint32 {
	d.t[d.p] += d.t[d.q]
	d.p = (d.p + 1) % 55
	d.q = (d.q + 2) % 55
	return d.t[d.p]
}
func (d *dec) raw() uint32 { v := binary.BigEndian.Uint32(d.in[d.pos:]); d.pos += 4; return v }
func (d *dec) xor() uint32 {
	c := binary.BigEndian.Uint32(d.in[d.pos:])
	pl := c ^ d.ks()
	binary.BigEndian.PutUint32(d.out[d.pos:], pl)
	d.pos += 4
	return pl
}
func (d *dec) rawN(longs uint32) { d.pos += int(longs) * 4 }

// decode runs the structure-aware decrypt; returns the plaintext hunk file or an
// error if the stream parses to garbage (out-of-range or an unknown block type).
func decode(in []byte, keyArray []uint32) (out []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("decode overran (bad key): %v", r)
		}
	}()
	d := &dec{in: in, out: append([]byte(nil), in...), t: keyTable(keyArray), p: 0, q: 27}
	if d.raw()&0x3FFFFFFF != hHEADER {
		return nil, fmt.Errorf("not a HUNK_HEADER")
	}
	for {
		n := d.xor()
		if n == 0 {
			break
		}
		if n > 64 {
			return nil, fmt.Errorf("lib name too long")
		}
		d.rawN(n)
	}
	d.xor() // table_size
	first := d.xor()
	last := d.xor()
	if first > last || last > 100000 {
		return nil, fmt.Errorf("bad header first=%d last=%d", first, last)
	}
	for i := first; i <= last; i++ {
		d.xor()
	}
	for d.pos+4 <= len(in) {
		typ := d.raw() & 0x3FFFFFFF
		switch typ {
		case hCODE, hDATA:
			size := d.xor() & 0x3FFFFFFF
			for j := uint32(0); j < size; j++ {
				d.xor()
			}
		case hBSS:
			d.xor()
		case hRELOC32:
			for {
				cnt := d.xor()
				if cnt == 0 {
					break
				}
				d.xor()
				for j := uint32(0); j < cnt; j++ {
					d.xor()
				}
			}
		case hSYMBOL:
			for {
				n := d.xor()
				if n == 0 {
					break
				}
				d.rawN(n)
				d.xor()
			}
		case hDEBUG:
			n := d.xor()
			for j := uint32(0); j < n; j++ {
				d.xor()
			}
		case hEND:
		case hUNIT, hNAME:
			n := d.xor()
			d.rawN(n)
		default:
			return nil, fmt.Errorf("unknown block $%X at byte %d", typ, d.pos-4)
		}
	}
	return d.out, nil
}

func main() {
	keyconst := flag.Uint("keyconst", 0, "16-bit key-array mutation constant (for the .dat)")
	count := flag.Int("count", 0, "key-array length in longs (c/xxx=0, .dat=20)")
	datalen := flag.Uint("datalen", 0, "c/xxx load length: key base[i]=datalen/((i+1)*300); 0 = uniform base (c/xxx). The .dat uses 1200 (bucket 1200-1499) with -keyconst 0xCDDA")
	brute := flag.Bool("brute", false, "brute-force the 16-bit key constant against valid-hunk output")
	flag.Parse()
	if flag.NArg() != 3 {
		fmt.Fprintln(os.Stderr, "usage: decode disk.adf innerpath out.hunk [-keyconst H -count N | -brute]")
		os.Exit(2)
	}
	img, err := os.ReadFile(flag.Arg(0))
	chk(err)
	vol, err := adf.Open(img)
	chk(err)
	in, err := vol.ReadFile(flag.Arg(1))
	chk(err)

	mkKey := func(c uint32, n int) []uint32 {
		ka := make([]uint32, n)
		for i := range ka {
			ka[i] = c
		}
		return ka
	}

	// The .dat key-array base is not zero: c/xxx's run_loader fills the 20-long
	// ctrl->C array (which becomes the key array) with base[i] = datalen/((i+1)*300)
	// (a multiply $61248 then a divide $61204), and the launcher then mutates each
	// entry ^= a 16-bit checksum constant C. So key[i] = datalen/((i+1)*300) ^ C.
	mkKeyBase := func(datalen, c uint32, n int) []uint32 {
		ka := make([]uint32, n)
		for i := range ka {
			ka[i] = (datalen / ((uint32(i) + 1) * 300)) ^ c
		}
		return ka
	}

	if *brute {
		n := *count
		if n == 0 {
			n = 20
		}
		try := func(ka []uint32, label string) bool {
			out, e := decode(in, ka)
			if e != nil {
				return false
			}
			if _, e2 := hunk.Load(out, 0x40000); e2 == nil {
				fmt.Printf("FOUND %s -> valid hunk file\n", label)
				chk(os.WriteFile(flag.Arg(2), out, 0644))
				return true
			}
			return false
		}
		// (1) uniform base (base = 0).
		for c := uint32(0); c <= 0xFFFF; c++ {
			if try(mkKey(c, n), fmt.Sprintf("uniform key constant $%04X (count %d)", c, n)) {
				return
			}
		}
		// (2) the c/xxx-derived base for every distinct datalen bucket (the divide
		// collapses many datalen values onto the same base vector).
		seen := map[string]bool{}
		for datalen := uint32(1); datalen <= 20000; datalen++ {
			base := mkKeyBase(datalen, 0, n)
			key := fmt.Sprint(base)
			if seen[key] {
				continue
			}
			seen[key] = true
			for c := uint32(0); c <= 0xFFFF; c++ {
				if try(mkKeyBase(datalen, c, n), fmt.Sprintf("base datalen=%d const $%04X (count %d)", datalen, c, n)) {
					return
				}
			}
		}
		fmt.Fprintln(os.Stderr, "brute-force found no valid key")
		os.Exit(1)
	}

	ka := mkKey(uint32(*keyconst), *count)
	if *datalen != 0 {
		ka = mkKeyBase(uint32(*datalen), uint32(*keyconst), *count)
	}
	out, err := decode(in, ka)
	chk(err)
	chk(os.WriteFile(flag.Arg(2), out, 0644))
	prog, e := hunk.Load(out, 0x40000)
	if e != nil {
		fmt.Printf("decoded %d bytes -> %s (WARNING: not a clean hunk file: %v)\n", len(out), flag.Arg(2), e)
	} else {
		fmt.Printf("decoded %d bytes -> %s (valid hunk file, %d segments)\n", len(out), flag.Arg(2), len(prog.Segments))
	}
}

func chk(e error) {
	if e != nil {
		panic(e)
	}
}
