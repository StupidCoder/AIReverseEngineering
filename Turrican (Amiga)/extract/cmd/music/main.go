// Command music extracts Turrican's TFMX music data from the disk — the first
// stage of the audio pipeline (Part V). Turrican's score is Chris Hülsbeck's, in
// his own TFMX format (Turrican is *the* canonical TFMX game), played by the
// in-game sound driver in the $1BB00 sound overlay (streamed from ADF $26000 at
// game_init). The driver's api_init ($1CB62) is handed two pointers, which
// game_init fills with:
//
//	mdat  $1CFF4  the "TFMX-SONG" song/pattern/macro data
//	smpl  $20E90  raw 8-bit signed PCM sample data (to the overlay's end)
//
// The mdat layout (verified against the driver's trackstep processor $1BED6):
//
//	+$100/+$140/+$180  song table: start/end/tempo word per sub-song (3 real)
//	+$400              pattern pointer table: 128 longs (offset from mdat to a pattern)
//	+$600              macro pointer table:   128 longs (offset from mdat to a macro)
//	+$800              trackstep table: 16 bytes/entry = 8 channel words. A word's
//	                   bit15 = channel off; else pattern# = (w>>8)&$7F, transpose =
//	                   w&$FF. A first word of $EFFE marks a command step.
//
// A pattern is a stream of 4-byte entries (note/instrument + $F0-$FF commands); a
// macro is a stream of 4-byte instrument commands ($00-$22) that set the sample,
// volume, period, vibrato/portamento/envelope etc. Samples are raw signed 8-bit.
//
// This command writes mdat.bin + smpl.bin and prints the song table; the synthesis
// player (next stage) reimplements the driver over these to render PCM.
//
// Usage: music [-o dir] [Turrican.adf]
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"turrican/extract/decrunch"
)

const (
	soundOff  = 0x26000 // ADF offset of the packed sound overlay
	soundLen  = 0xC268
	soundBase = 0x1BB00 // its runtime load address
	mdatAddr  = 0x1CFF4 // api_init d0 (song/pattern/macro data)
	smplAddr  = 0x20E90 // api_init d1 (sample data)
)

func main() {
	out := flag.String("o", "rendered/music", "output directory")
	render := flag.Int("render", -1, "render this sub-song to song<N>.wav (-1 = none)")
	secs := flag.Int("secs", 60, "max seconds to render")
	traceN := flag.Int("trace", 0, "print per-tick voice state for N ticks")
	traceSong := flag.Int("tracesong", 0, "sub-song for -trace")
	flag.Parse()
	adfPath := flag.Arg(0)
	if adfPath == "" {
		adfPath = "Turrican.adf"
	}
	adf, err := os.ReadFile(adfPath)
	if err != nil {
		fail(err)
	}
	overlay, err := decrunch.DecrunchBlock(adf[soundOff : soundOff+soundLen])
	if err != nil {
		fail(err)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fail(err)
	}

	mdat := overlay[mdatAddr-soundBase : smplAddr-soundBase]
	smpl := overlay[smplAddr-soundBase:]
	if err := os.WriteFile(filepath.Join(*out, "mdat.bin"), mdat, 0o644); err != nil {
		fail(err)
	}
	if err := os.WriteFile(filepath.Join(*out, "smpl.bin"), smpl, 0o644); err != nil {
		fail(err)
	}

	be16 := func(o int) int { return int(binary.BigEndian.Uint16(mdat[o:])) }
	fmt.Printf("mdat: %d bytes (TFMX-SONG)\nsmpl: %d bytes (8-bit PCM)\n", len(mdat), len(smpl))
	fmt.Println("sub-songs (trackstep start/end, tempo):")
	for i := 0; i < 32; i++ {
		s, e, t := be16(0x100+i*2), be16(0x140+i*2), be16(0x180+i*2)
		if s == 0 && e == 0 && t == 0 {
			continue
		}
		fmt.Printf("  %2d: start=$%04X end=$%04X tempo=$%04X\n", i, s, e, t)
	}

	if *traceN > 0 {
		pl := newPlayer(mdat, smpl)
		pl.tracing = true
		pl.start(*traceSong)
		for i := 0; i < *traceN; i++ {
			pl.stepTick()
		}
		for t, row := range pl.Trace {
			fmt.Printf("%d %d %d %d %d %d %d %d %d\n", t, row[0], row[1], row[2], row[3], row[4], row[5], row[6], row[7])
		}
		return
	}

	if *render >= 0 {
		const sr = 44100
		pl := newPlayer(mdat, smpl)
		pl.start(*render)
		pcm := pl.render(sr, *secs)
		name := fmt.Sprintf("song%d.wav", *render)
		if err := writeWAV(filepath.Join(*out, name), pcm, sr); err != nil {
			fail(err)
		}
		// signal stats
		var sum, peak float64
		for _, s := range pcm {
			f := float64(s)
			sum += f * f
			if f < 0 {
				f = -f
			}
			if f > peak {
				peak = f
			}
		}
		rms := 0.0
		if len(pcm) > 0 {
			rms = sqrt(sum / float64(len(pcm)))
		}
		fmt.Printf("rendered song %d: %d s @ %d Hz, tick=%.1f Hz -> %s (rms=%.3f peak=%.3f)\n",
			*render, *secs, sr, pl.tickHz, name, rms, peak)
	}
}

// writeWAV writes interleaved stereo float32 [-1,1] as 16-bit PCM WAV.
func writeWAV(path string, pcm []float32, sr int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	n := len(pcm)
	dataLen := n * 2
	hdr := make([]byte, 44)
	copy(hdr[0:], "RIFF")
	binary.LittleEndian.PutUint32(hdr[4:], uint32(36+dataLen))
	copy(hdr[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(hdr[16:], 16)
	binary.LittleEndian.PutUint16(hdr[20:], 1)              // PCM
	binary.LittleEndian.PutUint16(hdr[22:], 2)              // stereo
	binary.LittleEndian.PutUint32(hdr[24:], uint32(sr))     // rate
	binary.LittleEndian.PutUint32(hdr[28:], uint32(sr*2*2)) // byte rate
	binary.LittleEndian.PutUint16(hdr[32:], 4)              // block align
	binary.LittleEndian.PutUint16(hdr[34:], 16)             // bits
	copy(hdr[36:], "data")
	binary.LittleEndian.PutUint32(hdr[40:], uint32(dataLen))
	if _, err := f.Write(hdr); err != nil {
		return err
	}
	buf := make([]byte, dataLen)
	for i, s := range pcm {
		v := int16(s * 32000)
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(v))
	}
	_, err = f.Write(buf)
	return err
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	g := x
	for i := 0; i < 40; i++ {
		g = (g + x/g) / 2
	}
	return g
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "music:", err)
	os.Exit(1)
}
