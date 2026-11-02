package graphstore

import (
	"bytes"
	"slices"
	"strings"
	"testing"
)

// segmentSeeds are the seed inputs of both fuzz targets: the empty string,
// a zero byte, and strings of 127 and 128 bytes, either side of the
// one-byte varint length boundary.
var segmentSeeds = []string{"", "\x00", strings.Repeat("a", 127), strings.Repeat("b", 128)}

// FuzzSegmentRoundTrip checks that decodeSegment gives back the string
// that appendSegment encoded and consumes the whole buffer.
func FuzzSegmentRoundTrip(f *testing.F) {
	for _, s := range segmentSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		buf := appendSegment(nil, s)
		got, end, err := decodeSegment(buf, 0)
		if err != nil {
			t.Fatalf("decodeSegment(appendSegment(%q)): %v", s, err)
		}
		if got != s || end != len(buf) {
			t.Fatalf("decodeSegment(appendSegment(%q)) = %q ending at %d, want %q ending at %d", s, got, end, s, len(buf))
		}
	})
}

// encodeSegments concatenates the encodings of segs.
func encodeSegments(segs ...string) []byte {
	var buf []byte
	for _, s := range segs {
		buf = appendSegment(buf, s)
	}
	return buf
}

// FuzzSegmentBoundaries checks that two different lists of segments never
// have the same concatenated encoding (T-02-01): a crafted segment cannot
// move a boundary. Decoding the concatenation gives back exactly the list,
// so the encoding of a list determines the list. Lists that move a
// boundary of [a, b, c] encode differently from it.
func FuzzSegmentBoundaries(f *testing.F) {
	for _, a := range segmentSeeds {
		for _, b := range segmentSeeds {
			f.Add(a, b, "c")
		}
	}
	f.Fuzz(func(t *testing.T, a, b, c string) {
		list := []string{a, b, c}
		buf := encodeSegments(list...)
		var got []string
		for off := 0; off < len(buf); {
			s, end, err := decodeSegment(buf, off)
			if err != nil {
				t.Fatalf("decode %q at offset %d: %v", buf, off, err)
			}
			got = append(got, s)
			off = end
		}
		if !slices.Equal(got, list) {
			t.Fatalf("decoding the encoding of %q gives %q", list, got)
		}
		for _, other := range [][]string{{a + b, c}, {a, b + c}, {a + b + c}} {
			if bytes.Equal(encodeSegments(other...), buf) {
				t.Fatalf("%q and %q have the same encoding", other, list)
			}
		}
	})
}
