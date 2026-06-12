// amigapng renders an Amiga graphic file to PNG. It auto-detects the format:
// an IFF FORM…ILBM bitmap, or a Workbench .info icon (one PNG per icon image).
//
// Usage: amigapng input output.png
//
// For icons with both normal and selected imagery, the second image is written
// alongside output.png with a "-sel" suffix.
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"

	"stupidcoder.com/tools/amiga/icon"
	"stupidcoder.com/tools/amiga/iff"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: amigapng input output.png")
		os.Exit(2)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "amigapng:", err)
		os.Exit(1)
	}

	var imgs []image.Image
	switch {
	case len(data) >= 12 && string(data[0:4]) == "FORM" && string(data[8:12]) == "ILBM":
		img, err := iff.DecodeILBM(data)
		if err != nil {
			fail(err)
		}
		imgs = []image.Image{img}
	case len(data) >= 2 && data[0] == 0xE3 && data[1] == 0x10:
		ic, err := icon.DecodeInfo(data)
		if err != nil {
			fail(err)
		}
		// Icons are drawn on the hi-res Workbench screen, whose pixels are about
		// twice as tall as wide, so double the height to restore the aspect.
		for _, im := range ic {
			imgs = append(imgs, doubleHeight(im))
		}
	default:
		fail(fmt.Errorf("unrecognised format (not IFF ILBM or .info icon)"))
	}

	base := strings.TrimSuffix(os.Args[2], ".png")
	for i, im := range imgs {
		name := os.Args[2]
		if i == 1 {
			name = base + "-sel.png"
		} else if i > 1 {
			name = fmt.Sprintf("%s-%d.png", base, i)
		}
		if err := write(name, im); err != nil {
			fail(err)
		}
		b := im.Bounds()
		fmt.Printf("%s  %dx%d\n", name, b.Dx(), b.Dy())
	}
}

// doubleHeight scales an image 1×2 (nearest-neighbour) to correct the tall
// hi-res Workbench pixel aspect for square-pixel display.
func doubleHeight(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()*2))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := src.At(b.Min.X+x, b.Min.Y+y)
			dst.Set(x, y*2, c)
			dst.Set(x, y*2+1, c)
		}
	}
	return dst
}

func write(name string, im image.Image) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, im)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "amigapng:", err)
	os.Exit(1)
}
