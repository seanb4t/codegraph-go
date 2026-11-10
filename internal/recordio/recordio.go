// Package recordio holds the record framing and the deterministic protobuf
// marshal that the graph export, the graph import and the indexer's spill
// file share.
//
// A frame is [kind byte][uvarint length][payload]. The kind byte makes a
// stream self-describing, and the length prefix lets a reader skip to the
// next frame or read one frame at a known offset.
package recordio

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"
)

// maxHeaderLen is the longest frame header: one kind byte and a uvarint.
const maxHeaderLen = 1 + binary.MaxVarintLen64

// Marshal marshals msg with the keys of every map field in a fixed order.
// Go's protobuf runtime otherwise walks a map in a random order on each
// call, so a message with a map field (schema.Edge.Metadata) gets different
// bytes from run to run. Every record that reaches the store, an export
// stream or the spill file goes through this function.
func Marshal(msg proto.Message) ([]byte, error) {
	return proto.MarshalOptions{Deterministic: true}.Marshal(msg)
}

// AppendFrame appends one frame to dst and returns the extended slice.
func AppendFrame(dst []byte, kind uint8, payload []byte) []byte {
	dst = append(dst, kind)
	dst = binary.AppendUvarint(dst, uint64(len(payload)))
	return append(dst, payload...)
}

// WriteFrame writes one frame to w and returns the number of bytes written.
func WriteFrame(w io.Writer, kind uint8, payload []byte) (int, error) {
	header := make([]byte, 0, maxHeaderLen)
	header = append(header, kind)
	header = binary.AppendUvarint(header, uint64(len(payload)))
	n, err := w.Write(header)
	if err != nil {
		return n, err
	}
	m, err := w.Write(payload)
	return n + m, err
}

// Reader reads frames in sequence from a stream.
type Reader struct {
	br      *bufio.Reader
	ceiling uint64
}

// NewReader returns a Reader over r. ceiling is the longest payload the
// Reader accepts, and zero means no ceiling. A caller that reads a stream
// it did not write passes a ceiling, so a hostile or corrupt length prefix
// cannot drive an unbounded allocation.
func NewReader(r io.Reader, ceiling uint64) *Reader {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}
	return &Reader{br: br, ceiling: ceiling}
}

// Next returns the next frame. It returns io.EOF when the stream ends on a
// frame boundary, and another error when the stream ends inside a frame.
func (r *Reader) Next() (kind uint8, payload []byte, err error) {
	kind, err = r.br.ReadByte()
	if err != nil {
		if err == io.EOF {
			return 0, nil, io.EOF
		}
		return 0, nil, fmt.Errorf("recordio: read kind: %w", err)
	}
	length, err := binary.ReadUvarint(r.br)
	if err != nil {
		return 0, nil, fmt.Errorf("recordio: read length: %w", noEOF(err))
	}
	if err := checkCeiling(length, r.ceiling); err != nil {
		return 0, nil, err
	}
	payload = make([]byte, length)
	if _, err := io.ReadFull(r.br, payload); err != nil {
		return 0, nil, fmt.Errorf("recordio: read payload: %w", noEOF(err))
	}
	return kind, payload, nil
}

// ReadFrameAt reads the one frame that starts at offset off of r. It
// returns the frame and its whole length in bytes, header included.
// ceiling has the meaning it has in NewReader.
func ReadFrameAt(r io.ReaderAt, off int64, ceiling uint64) (kind uint8, payload []byte, n int, err error) {
	var header [maxHeaderLen]byte
	got, err := r.ReadAt(header[:], off)
	if err != nil && err != io.EOF {
		return 0, nil, 0, fmt.Errorf("recordio: read header at %d: %w", off, err)
	}
	if got == 0 {
		return 0, nil, 0, fmt.Errorf("recordio: read header at %d: %w", off, io.ErrUnexpectedEOF)
	}
	length, used := binary.Uvarint(header[1:got])
	if used <= 0 {
		return 0, nil, 0, fmt.Errorf("recordio: bad length prefix at %d", off)
	}
	if err := checkCeiling(length, ceiling); err != nil {
		return 0, nil, 0, err
	}
	headerLen := 1 + used
	payload = make([]byte, length)
	if length > 0 {
		if _, err := r.ReadAt(payload, off+int64(headerLen)); err != nil {
			return 0, nil, 0, fmt.Errorf("recordio: read payload at %d: %w", off, noEOF(err))
		}
	}
	return header[0], payload, headerLen + int(length), nil
}

func checkCeiling(length, ceiling uint64) error {
	if ceiling != 0 && length > ceiling {
		return fmt.Errorf("recordio: record length %d exceeds ceiling %d", length, ceiling)
	}
	return nil
}

// noEOF turns an io.EOF inside a frame into io.ErrUnexpectedEOF, so that
// only a clean end of stream reads as io.EOF.
func noEOF(err error) error {
	if errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}

// Frame is one decoded frame of a stream.
type Frame struct {
	Kind    uint8
	Payload []byte
}

// ReadAll reads every frame of r. ceiling has the meaning it has in
// NewReader.
func ReadAll(r io.Reader, ceiling uint64) ([]Frame, error) {
	fr := NewReader(r, ceiling)
	var frames []Frame
	for {
		kind, payload, err := fr.Next()
		if err == io.EOF {
			return frames, nil
		}
		if err != nil {
			return nil, err
		}
		frames = append(frames, Frame{Kind: kind, Payload: payload})
	}
}

// AppendFrames appends every frame to dst and returns the extended slice.
func AppendFrames(dst []byte, frames []Frame) []byte {
	for _, f := range frames {
		dst = AppendFrame(dst, f.Kind, f.Payload)
	}
	return dst
}
