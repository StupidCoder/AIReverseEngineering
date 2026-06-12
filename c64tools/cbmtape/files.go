package cbmtape

import "fmt"

// File is a complete KERNAL-format tape file: a header block paired with its
// data block (first copies; repeats are used only for validation by the
// caller if desired).
type File struct {
	Type       byte   // 1: relocatable PRG, 3: non-relocatable PRG, 4: data
	Start, End uint16 // load address range from the header (End exclusive)
	Name       string // 16 PETSCII chars, space padded
	Data       []byte // data block payload
	ChecksumOK bool   // both header and data block checksums valid
	LastPulse  int    // pulse index just past the file's data block (first copy)
}

func (f File) String() string {
	return fmt.Sprintf("CBM file %q type %d  $%04X-$%04X (%d bytes, checksum ok=%v)",
		f.Name, f.Type, f.Start, f.End, len(f.Data), f.ChecksumOK)
}

// Files pairs the header/data blocks found in a pulse stream into complete
// files. Blocks that do not form a header+data pair are ignored, as are
// repeat copies. The LastPulse of the final file tells callers where a
// non-standard (fastloader) part of the tape may begin.
func Files(blocks []Block) []File {
	var files []File
	var hdr *Header
	hdrOK := false
	for _, b := range blocks {
		if b.Repeat {
			continue
		}
		if len(b.Payload) == 192 && b.Payload[0] >= 1 && b.Payload[0] <= 5 {
			h, err := ParseHeader(b.Payload)
			if err == nil && (h.Type == 1 || h.Type == 3 || h.Type == 4) {
				hdr, hdrOK = h, b.ChecksumOK
				continue
			}
		}
		if hdr == nil {
			continue // data block without header
		}
		files = append(files, File{
			Type:       hdr.Type,
			Start:      hdr.StartAddr,
			End:        hdr.EndAddr,
			Name:       hdr.Name,
			Data:       b.Payload,
			ChecksumOK: hdrOK && b.ChecksumOK,
			LastPulse:  b.EndPulse,
		})
		hdr = nil
	}
	return files
}
