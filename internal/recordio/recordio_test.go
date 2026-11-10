package recordio

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// TestFrameReaderCeiling checks the edge of the ceiling: a payload of
// exactly the ceiling reads back, and one more byte is refused.
func TestFrameReaderCeiling(t *testing.T) {
	const ceiling = 1024
	for _, tc := range []struct {
		name    string
		size    int
		wantErr bool
	}{
		{"at the ceiling", ceiling, false},
		{"one byte over", ceiling + 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stream := AppendFrame(nil, 7, bytes.Repeat([]byte{0xAB}, tc.size))

			kind, payload, err := NewReader(bytes.NewReader(stream), ceiling).Next()
			_, atPayload, _, atErr := ReadFrameAt(bytes.NewReader(stream), 0, ceiling)
			if !tc.wantErr {
				if err != nil || atErr != nil {
					t.Fatalf("Next: %v, ReadFrameAt: %v, want no error", err, atErr)
				}
				if kind != 7 || len(payload) != tc.size || len(atPayload) != tc.size {
					t.Fatalf("kind %d, lengths %d and %d, want kind 7 and length %d", kind, len(payload), len(atPayload), tc.size)
				}
				return
			}
			for _, e := range []error{err, atErr} {
				if e == nil || !strings.Contains(e.Error(), "exceeds ceiling 1024") {
					t.Fatalf("error %v, want one that names the ceiling 1024", e)
				}
			}
		})
	}
}

// TestFrameReaderTruncated checks that a stream that ends inside a frame
// does not read as a clean end of stream.
func TestFrameReaderTruncated(t *testing.T) {
	stream := AppendFrame(nil, 1, []byte("payload"))
	for cut := 1; cut < len(stream); cut++ {
		_, _, err := NewReader(bytes.NewReader(stream[:cut]), 0).Next()
		if err == nil || err == io.EOF {
			t.Fatalf("cut at %d: error %v, want an error that is not io.EOF", cut, err)
		}
	}
}

// TestFrameRoundTripProperty writes frames of every length class of the
// uvarint prefix, the empty frame included, and reads each one back both
// in sequence and at its offset.
func TestFrameRoundTripProperty(t *testing.T) {
	sizes := rapid.OneOf(
		rapid.SampledFrom([]int{0, 127, 128, 16383, 16384}),
		rapid.IntRange(0, 20000),
	)
	rapid.Check(t, func(t *rapid.T) {
		type frame struct {
			kind    uint8
			payload []byte
			off     int64
		}
		var (
			buf    bytes.Buffer
			frames []frame
		)
		for i, n := 0, rapid.IntRange(0, 12).Draw(t, "frames"); i < n; i++ {
			f := frame{
				kind:    rapid.Uint8().Draw(t, "kind"),
				payload: bytes.Repeat([]byte{rapid.Byte().Draw(t, "fill")}, sizes.Draw(t, "size")),
				off:     int64(buf.Len()),
			}
			written, err := WriteFrame(&buf, f.kind, f.payload)
			if err != nil {
				t.Fatalf("WriteFrame: %v", err)
			}
			if int64(written) != int64(buf.Len())-f.off {
				t.Fatalf("WriteFrame returned %d, the stream grew by %d", written, int64(buf.Len())-f.off)
			}
			frames = append(frames, f)
		}
		stream := buf.Bytes()

		r := NewReader(bytes.NewReader(stream), 0)
		for i, f := range frames {
			kind, payload, err := r.Next()
			if err != nil || kind != f.kind || !bytes.Equal(payload, f.payload) {
				t.Fatalf("frame %d in sequence: kind %d, %d bytes, err %v", i, kind, len(payload), err)
			}
			kind, payload, n, err := ReadFrameAt(bytes.NewReader(stream), f.off, 0)
			if err != nil || kind != f.kind || !bytes.Equal(payload, f.payload) {
				t.Fatalf("frame %d at offset %d: kind %d, %d bytes, err %v", i, f.off, kind, len(payload), err)
			}
			end := int64(len(stream))
			if i+1 < len(frames) {
				end = frames[i+1].off
			}
			if f.off+int64(n) != end {
				t.Fatalf("frame %d: length %d from offset %d, want the next frame at %d", i, n, f.off, end)
			}
		}
		if _, _, err := r.Next(); err != io.EOF {
			t.Fatalf("after the last frame: err %v, want io.EOF", err)
		}
	})
}
