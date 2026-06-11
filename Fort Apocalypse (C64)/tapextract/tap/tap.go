// Package tap reads C64 TAP (raw tape) image files, versions 0 and 1.
package tap

import (
	"encoding/binary"
	"fmt"
)

const headerSize = 20

var magics = []string{"C64-TAPE-RAW", "C16-TAPE-RAW"}

// Pulse is one tape event: either a pulse of Cycles clock cycles between
// two falling edges of the cassette read line, or a pause (silence).
type Pulse struct {
	Cycles int  // duration in C64 clock cycles
	Pause  bool // true if this is a pause/gap, not a pulse
	Offset int  // byte offset of this entry in the TAP file (for diagnostics)
}

// Image is a parsed TAP file.
type Image struct {
	Version  byte
	Platform byte
	Video    byte
	Pulses   []Pulse
}

// Parse decodes a complete TAP file.
func Parse(b []byte) (*Image, error) {
	if len(b) < headerSize {
		return nil, fmt.Errorf("tap: file too short (%d bytes)", len(b))
	}
	magic := string(b[:12])
	ok := false
	for _, m := range magics {
		if magic == m {
			ok = true
		}
	}
	if !ok {
		return nil, fmt.Errorf("tap: bad magic %q", magic)
	}
	img := &Image{Version: b[12], Platform: b[13], Video: b[14]}
	if img.Version > 1 {
		return nil, fmt.Errorf("tap: unsupported version %d", img.Version)
	}
	dataLen := int(binary.LittleEndian.Uint32(b[16:20]))
	data := b[headerSize:]
	if dataLen != len(data) {
		return nil, fmt.Errorf("tap: header says %d data bytes, file has %d", dataLen, len(data))
	}
	for i := 0; i < len(data); i++ {
		off := headerSize + i
		v := data[i]
		if v != 0 {
			// Regular pulse: duration = value * 8 cycles.
			img.Pulses = append(img.Pulses, Pulse{Cycles: int(v) * 8, Offset: off})
			continue
		}
		if img.Version == 0 {
			// v0: 0x00 = overflow, a pulse longer than 255*8 cycles.
			img.Pulses = append(img.Pulses, Pulse{Cycles: 256 * 8, Pause: true, Offset: off})
			continue
		}
		// v1: 0x00 followed by 24-bit little-endian pause length in cycles.
		if i+3 >= len(data) {
			return nil, fmt.Errorf("tap: truncated pause marker at offset %d", off)
		}
		cycles := int(data[i+1]) | int(data[i+2])<<8 | int(data[i+3])<<16
		img.Pulses = append(img.Pulses, Pulse{Cycles: cycles, Pause: true, Offset: off})
		i += 3
	}
	return img, nil
}
