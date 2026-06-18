package powerpacker

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
)

// encodeLiterals builds a minimal valid PP20 stream that encodes out as a single
// literal run (no back-references). This exercises the magic/trailer parsing, the
// backward longword bit reader, the skip padding, the 2-bit literal-count chunks
// and the backward byte writes — everything except the match path, which is
// verified separately against the FS-UAE oracle on real crunched data.
func encodeLiterals(out []byte) []byte {
	var bits []int
	push := func(val, n int) { // MSB-first, matching bitReader.bits
		for i := n - 1; i >= 0; i-- {
			bits = append(bits, (val>>uint(i))&1)
		}
	}

	bits = append(bits, 0) // literal-run flag
	t := len(out) - 1      // count is encoded as (run length - 1)
	for t >= 3 {
		push(3, 2)
		t -= 3
	}
	push(t, 2)
	for i := len(out) - 1; i >= 0; i-- { // bytes are read in reverse output order
		push(int(out[i]), 8)
	}

	// Pack the read-order bit list into big-endian longwords consumed from the
	// end, LSB-first, with the leading padding accounted for by skip.
	m := len(bits)
	k := (m + 31) / 32
	if k == 0 {
		k = 1
	}
	skip := 32*k - m
	longs := make([]uint32, k)
	for j := 0; j < m; j++ {
		if bits[j] == 1 {
			o := skip + j
			longs[k-1-o/32] |= 1 << uint(o%32)
		}
	}

	buf := bytes.NewBufferString(Magic)
	buf.Write([]byte{9, 10, 12, 13}) // efficiency table (unused for literals)
	for _, l := range longs {
		binary.Write(buf, binary.BigEndian, l)
	}
	binary.Write(buf, binary.BigEndian, uint32(len(out))<<8|uint32(skip))
	return buf.Bytes()
}

func TestDecrunchLiterals(t *testing.T) {
	cases := [][]byte{
		[]byte("A"),
		[]byte("Hi"),
		[]byte("PowerPacker"),
		[]byte("The quick brown fox jumps over the lazy dog."),
		bytes.Repeat([]byte{0xAB}, 200), // forces multi-chunk counts + many longwords
	}
	for i := 0; i < 256; i++ { // every byte value round-trips
		cases = append(cases, []byte{byte(i)})
	}
	for _, want := range cases {
		got, err := Decrunch(encodeLiterals(want))
		if err != nil {
			t.Fatalf("Decrunch(%q): %v", want, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("round-trip mismatch:\n got %q\nwant %q", got, want)
		}
	}
}

func TestDecrunchRejectsBadMagic(t *testing.T) {
	if _, err := Decrunch([]byte("PP11\x00\x00\x00\x00\x00\x00\x00\x00")); err == nil {
		t.Fatal("expected error for non-PP20 magic")
	}
	if _, err := Decrunch([]byte("short")); err == nil {
		t.Fatal("expected error for short input")
	}
}

func ExampleDecrunch() {
	out, _ := Decrunch(encodeLiterals([]byte("hello")))
	fmt.Printf("%s\n", out)
	// Output: hello
}
