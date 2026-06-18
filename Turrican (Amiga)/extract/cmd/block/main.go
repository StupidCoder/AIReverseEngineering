// Command block extracts and decodes an in-game data block streamed off the
// floppy by the trackloader and unpacked at runtime by huff_decode ($5F000).
//
//	block -off DISKOFF -len LEN [-base ADDR] [-o out.bin] [Turrican.adf]
//
// The game-code overlay (where the routines the init chain calls — $1BB24 etc. —
// live) is ADF offset $26000, length $C268, loaded at $1BB00:
//
//	block -off 0x26000 -len 0xC268 -base 0x1BB00 -o /tmp/block_1BB00.bin Turrican.adf
package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"os"

	"turrican/extract/decrunch"
)

func main() {
	off := flag.Int("off", 0x26000, "disk (ADF) byte offset of the packed block")
	length := flag.Int("len", 0xC268, "packed block length in bytes")
	base := flag.Int("base", 0x1BB00, "runtime load address of the decoded block")
	out := flag.String("o", "", "write the decoded block to this file")
	flag.Parse()

	adfPath := flag.Arg(0)
	if adfPath == "" {
		adfPath = "Turrican.adf"
	}
	adf, err := os.ReadFile(adfPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read adf:", err)
		os.Exit(1)
	}
	if *off+*length > len(adf) {
		fmt.Fprintf(os.Stderr, "block $%X+$%X exceeds image ($%X)\n", *off, *length, len(adf))
		os.Exit(1)
	}

	img, err := decrunch.DecrunchBlock(adf[*off : *off+*length])
	if err != nil {
		fmt.Fprintln(os.Stderr, "decrunch block:", err)
		os.Exit(1)
	}

	fmt.Printf("packed = ADF $%05X .. $%05X ($%X bytes)\n", *off, *off+*length, *length)
	fmt.Printf("base   = $%05X\n", *base)
	fmt.Printf("size   = $%05X (%d bytes), ends at $%05X\n", len(img), len(img), *base+len(img))
	fmt.Printf("md5    = %x\n", md5.Sum(img))

	if *out != "" {
		if err := os.WriteFile(*out, img, 0644); err != nil {
			fmt.Fprintln(os.Stderr, "write:", err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s\n", *out)
	}
}
