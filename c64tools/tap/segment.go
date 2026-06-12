package tap

// Segment is a run of pulses delimited by pauses — typically one tape block
// or one fastloader load unit.
type Segment struct {
	First, Last int // inclusive pulse index range into the image's Pulses
}

// Segmentize returns the pause-delimited segments of a pulse stream.
func Segmentize(pulses []Pulse) []Segment {
	var segs []Segment
	start := -1
	for i, p := range pulses {
		if p.Pause {
			if start >= 0 {
				segs = append(segs, Segment{start, i - 1})
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		segs = append(segs, Segment{start, len(pulses) - 1})
	}
	return segs
}
