package gfx

// Minimal animated-PNG (APNG) encoder, standard library only. APNG is an
// ordinary PNG (the first frame shows in any PNG viewer) plus the animation
// control chunks acTL/fcTL and the frame-data chunks fdAT. We write 8-bit
// truecolour (RGB) frames with no inter-frame compositing (each frame is a
// full opaque image), which keeps the chunk bookkeeping simple.

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"image"
	"os"
)

// WriteAPNG encodes frames (all the same size) as an animated PNG looping
// forever, with each frame shown for delayNum/delayDen seconds.
func WriteAPNG(path string, frames []*image.RGBA, delayNum, delayDen uint16) error {
	if len(frames) == 0 {
		return nil
	}
	b := frames[0].Bounds()
	w, h := b.Dx(), b.Dy()

	var buf bytes.Buffer
	buf.Write([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A})

	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], uint32(w))
	binary.BigEndian.PutUint32(ihdr[4:], uint32(h))
	ihdr[8] = 8 // bit depth
	ihdr[9] = 2 // colour type 2 = truecolour RGB
	writeChunk(&buf, "IHDR", ihdr)

	actl := make([]byte, 8)
	binary.BigEndian.PutUint32(actl[0:], uint32(len(frames)))
	binary.BigEndian.PutUint32(actl[4:], 0) // play count 0 = infinite
	writeChunk(&buf, "acTL", actl)

	var seq uint32
	for i, fr := range frames {
		fctl := make([]byte, 26)
		binary.BigEndian.PutUint32(fctl[0:], seq) // sequence number
		seq++
		binary.BigEndian.PutUint32(fctl[4:], uint32(w))
		binary.BigEndian.PutUint32(fctl[8:], uint32(h))
		// x/y offset 0 (bytes 12..19)
		binary.BigEndian.PutUint16(fctl[20:], delayNum)
		binary.BigEndian.PutUint16(fctl[22:], delayDen)
		fctl[24] = 0 // dispose: none
		fctl[25] = 0 // blend: source (overwrite)
		writeChunk(&buf, "fcTL", fctl)

		data := rgbIDAT(fr)
		if i == 0 {
			writeChunk(&buf, "IDAT", data)
		} else {
			fdat := make([]byte, 4+len(data))
			binary.BigEndian.PutUint32(fdat[0:], seq) // fdAT carries its own seq
			seq++
			copy(fdat[4:], data)
			writeChunk(&buf, "fdAT", fdat)
		}
	}
	writeChunk(&buf, "IEND", nil)
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// rgbIDAT returns the zlib-compressed, filtered (filter type 0) RGB scanlines
// for one frame — the payload shared by IDAT and fdAT chunks.
func rgbIDAT(img *image.RGBA) []byte {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	raw := make([]byte, 0, h*(1+w*3))
	for y := 0; y < h; y++ {
		raw = append(raw, 0) // filter: none
		for x := 0; x < w; x++ {
			c := img.RGBAAt(b.Min.X+x, b.Min.Y+y)
			raw = append(raw, c.R, c.G, c.B)
		}
	}
	var z bytes.Buffer
	zw := zlib.NewWriter(&z)
	zw.Write(raw)
	zw.Close()
	return z.Bytes()
}

func writeChunk(buf *bytes.Buffer, typ string, data []byte) {
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(data)))
	buf.Write(length[:])
	crc := crc32.NewIEEE()
	buf.WriteString(typ)
	crc.Write([]byte(typ))
	buf.Write(data)
	crc.Write(data)
	var sum [4]byte
	binary.BigEndian.PutUint32(sum[:], crc.Sum32())
	buf.Write(sum[:])
}
