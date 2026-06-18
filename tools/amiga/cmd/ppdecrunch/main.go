// ppdecrunch decompresses a PowerPacker ("PP20") file, or a PP20 block embedded
// at a given offset in a larger file.
//
//	ppdecrunch [-off N] [-len N] [-o out] file
//
// With no -off it expects file to begin with the "PP20" magic. -off/-len select a
// sub-slice (e.g. a PP20 module embedded in a decrunched game image).
package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"os"

	"stupidcoder.com/tools/amiga/powerpacker"
)

func main() {
	off := flag.Int("off", 0, "byte offset of the PP20 stream within the file")
	length := flag.Int("len", 0, "length of the PP20 stream (0 = to end of file)")
	out := flag.String("o", "", "write the decompressed data here (default: stdout)")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: ppdecrunch [-off N] [-len N] [-o out] file")
		os.Exit(2)
	}

	data, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "read:", err)
		os.Exit(1)
	}
	end := len(data)
	if *length > 0 {
		end = *off + *length
	}
	if *off < 0 || end > len(data) || *off > end {
		fmt.Fprintf(os.Stderr, "slice [%d:%d] out of range (%d bytes)\n", *off, end, len(data))
		os.Exit(1)
	}

	img, err := powerpacker.Decrunch(data[*off:end])
	if err != nil {
		fmt.Fprintln(os.Stderr, "decrunch:", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "decrunched %d bytes, md5 %x\n", len(img), md5.Sum(img))
	if *out == "" {
		os.Stdout.Write(img)
		return
	}
	if err := os.WriteFile(*out, img, 0644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", *out)
}
