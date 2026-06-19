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
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"turrican/extract/decrunch"
	"turrican/extract/scene"
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
	all := flag.Bool("all", false, "render every sub-song of every TFMX module (overlay + 5 worlds) to WAVs + manifest.json")
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

	if *all {
		renderAll(adf, overlay, *out, *secs)
		return
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

// tfmxModule is one TFMX song/sample bank: a runtime address for naming, the mdat
// (song/pattern/macro/trackstep tables) and the sample bank it draws from.
type tfmxModule struct {
	label string // human label (where it comes from)
	addr  int    // mdat runtime address (used in song names)
	mdat  []byte
	smpl  []byte
}

// modulesOf returns every TFMX module in the game: the $1BB00 sound overlay (title /
// menu / jingle music) plus each of the 5 worlds' in-game themes. select_scene wires a
// world's music as mdat = block+$10, smpl = block+$0C — both inside the decoded block.
func modulesOf(adf, overlay []byte) ([]tfmxModule, error) {
	mods := []tfmxModule{{
		label: "sound overlay $1BB00",
		addr:  mdatAddr,
		mdat:  overlay[mdatAddr-soundBase : smplAddr-soundBase],
		smpl:  overlay[smplAddr-soundBase:],
	}}
	g, err := scene.Load(adf)
	if err != nil {
		return nil, err
	}
	for w := 0; w < scene.NumWorlds; w++ {
		blk := g.Block(w)
		mAddr := int(binary.BigEndian.Uint32(blk.Data[scene.BlockBase+0x10-blk.Base:]))
		sAddr := int(binary.BigEndian.Uint32(blk.Data[scene.BlockBase+0x0C-blk.Base:]))
		mOff, sOff := mAddr-blk.Base, sAddr-blk.Base
		if mOff < 0 || mOff >= len(blk.Data) || sOff < 0 || sOff >= len(blk.Data) {
			continue
		}
		mods = append(mods, tfmxModule{
			label: fmt.Sprintf("world %d", w),
			addr:  mAddr,
			mdat:  blk.Data[mOff:],
			smpl:  blk.Data[sOff:],
		})
	}
	return mods, nil
}

// songEntry is one renderable sub-song, written to the manifest.
type songEntry struct {
	File  string `json:"file"`
	Addr  string `json:"addr"`  // mdat module address, hex
	Song  int    `json:"song"`  // sub-song slot index
	Start string `json:"start"` // trackstep start, hex
	Label string `json:"label"` // source (overlay / world N)
}

// renderAll renders every distinct, audible sub-song of every module to a WAV named by
// its module address + trackstep start, and writes manifest.json describing them.
func renderAll(adf, overlay []byte, out string, secs int) {
	const sr = 44100
	mods, err := modulesOf(adf, overlay)
	if err != nil {
		fail(err)
	}
	var manifest []songEntry
	for _, m := range mods {
		be16 := func(o int) int { return int(binary.BigEndian.Uint16(m.mdat[o:])) }
		// The 32-slot song table is mostly padding: api_play(any) must be safe, so unused
		// slots point at a single "stop" step (e.g. $73-$73 / $1-$1) repeated 20+ times,
		// and slot 31 is the $1FF terminator. Real sub-songs are the distinct entries; a
		// song's trackstep start identifies it (tempo can be 0 — timing then comes from the
		// in-song $EFFE command, as in world 3). Count each (start,end) to spot the filler,
		// and keep one entry per distinct start (the widest trackstep range).
		cnt := map[[2]int]int{}
		for i := 0; i < 32; i++ {
			cnt[[2]int{be16(0x100 + i*2), be16(0x140 + i*2)}]++
		}
		best := map[int]int{} // trackstep start -> chosen slot index
		for i := 0; i < 32; i++ {
			s, e := be16(0x100+i*2), be16(0x140+i*2)
			if s > e || s >= 0x100 || e >= 0x100 { // out of range / $1FF terminator
				continue
			}
			if cnt[[2]int{s, e}] >= 8 { // a repeated stop-step, not a song
				continue
			}
			if cur, ok := best[s]; !ok || e-s > be16(0x140+cur*2)-s {
				best[s] = i
			}
		}
		for s := 0; s < 0x100; s++ {
			i, ok := best[s]
			if !ok {
				continue
			}
			e, t := be16(0x140+i*2), be16(0x180+i*2)
			_ = e
			_ = t
			pl := newPlayer(m.mdat, m.smpl)
			pl.start(i)
			pcm := pl.renderN(sr, secs, true)
			if rms(pcm) < 0.004 { // empty stub / silence
				continue
			}
			name := fmt.Sprintf("mus_%X_%02X", m.addr, s)
			if err := writeWAV(filepath.Join(out, name+".wav"), pcm, sr); err != nil {
				fail(err)
			}
			manifest = append(manifest, songEntry{
				File:  name + ".mp3", // WAVs are intermediate; the site serves MP3
				Addr:  fmt.Sprintf("%X", m.addr),
				Song:  i,
				Start: fmt.Sprintf("%X", s),
				Label: m.label,
			})
			fmt.Printf("%-20s %s  song %2d (track $%02X-$%02X) %.1fs rms=%.3f\n",
				m.label, name, i, s, e, float64(len(pcm)/2)/sr, rms(pcm))
		}
	}
	j, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "manifest.json"), j, 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("\n%d songs -> %s/manifest.json\n", len(manifest), out)
}

func rms(pcm []float32) float64 {
	var sum float64
	for _, s := range pcm {
		sum += float64(s) * float64(s)
	}
	if len(pcm) == 0 {
		return 0
	}
	return sqrt(sum / float64(len(pcm)))
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
