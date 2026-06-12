package gfx

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestLine(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	white := color.RGBA{255, 255, 255, 255}
	Line(img, 0, 0, 9, 9, white)
	for i := 0; i < 10; i++ {
		if img.RGBAAt(i, i) != white {
			t.Errorf("diagonal pixel (%d,%d) not set", i, i)
		}
	}
	// out-of-bounds endpoints must not panic
	Line(img, -5, 5, 20, 5, white)
}

func TestWriteAPNG(t *testing.T) {
	mk := func(c color.RGBA) *image.RGBA {
		im := image.NewRGBA(image.Rect(0, 0, 4, 4))
		for i := range im.Pix {
			im.Pix[i] = 0
		}
		im.SetRGBA(1, 1, c)
		return im
	}
	frames := []*image.RGBA{
		mk(color.RGBA{255, 0, 0, 255}),
		mk(color.RGBA{0, 255, 0, 255}),
		mk(color.RGBA{0, 0, 255, 255}),
	}
	path := filepath.Join(t.TempDir(), "anim.png")
	if err := WriteAPNG(path, frames, 5, 100); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Valid PNG signature, contains the APNG chunks, and the first frame
	// decodes as a normal PNG with the expected pixel.
	if !bytes.HasPrefix(raw, []byte{0x89, 'P', 'N', 'G'}) {
		t.Fatal("bad PNG signature")
	}
	for _, c := range []string{"acTL", "fcTL", "fdAT", "IDAT", "IEND"} {
		if !bytes.Contains(raw, []byte(c)) {
			t.Errorf("missing chunk %s", c)
		}
	}
	im, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("first frame does not decode: %v", err)
	}
	r, _, _, _ := im.At(1, 1).RGBA()
	if r>>8 != 255 {
		t.Errorf("first frame pixel red = %d, want 255", r>>8)
	}
}
