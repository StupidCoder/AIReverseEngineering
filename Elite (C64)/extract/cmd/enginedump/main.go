// enginedump writes the full decrypted Elite engine as a .prg with load
// address $0000, reconstructing the page range the loader hid under the I/O
// area ($D000-$DFFF) from the decrypted SYS segment (see shipmodel.LoadEngine
// and Elite.md Part III). The result is the input for static tools such as
// `codetrace` that need the running game's complete memory image.
//
// Usage: enginedump [-extracted dir] out.prg
package main

import (
	"flag"
	"fmt"
	"os"

	"elite/extract/shipmodel"
)

func main() {
	extracted := flag.String("extracted", "../extracted", "directory of extracted files")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: enginedump [-extracted dir] out.prg")
		os.Exit(2)
	}
	mem, err := shipmodel.LoadEngine(*extracted)
	if err != nil {
		fmt.Fprintln(os.Stderr, "enginedump:", err)
		os.Exit(1)
	}
	// .prg = 2-byte little-endian load address ($0000) followed by the 64 KB.
	out := append([]byte{0x00, 0x00}, mem...)
	if err := os.WriteFile(flag.Arg(0), out, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "enginedump:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (64 KB engine image, load $0000)\n", flag.Arg(0))
}
