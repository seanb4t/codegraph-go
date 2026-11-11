package spill

import (
	"os"
	"sync"
	"sync/atomic"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/recordio"
)

// frameKind is the kind byte of every frame of a spill file. The file
// holds one kind of record, so the byte only keeps the frame shape of
// recordio.
const frameKind uint8 = 1

// File is a spill file. Append is safe for concurrent use, and so is Read,
// also while other goroutines append.
type File struct {
	f *os.File

	// mu covers off and the write of one frame, so each record is
	// contiguous at the offset that Append returns, and no two records
	// share an offset.
	mu  sync.Mutex
	off int64

	// size is off, for a reader. Read takes it as the ceiling on the
	// length of a frame, and it must not wait for a writer to do so.
	size atomic.Int64
}

// Create creates the spill file at path, which must not exist.
func Create(path string) (*File, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, err
	}
	return &File{f: f}, nil
}

// Append writes r as one record and returns the offset to read it back.
func (s *File) Append(r *goextract.FileResult) (int64, error) {
	data, err := Encode(r)
	if err != nil {
		return 0, err
	}
	return s.appendPayload(data)
}

// appendPayload writes one frame at the end of the file.
func (s *File) appendPayload(payload []byte) (int64, error) {
	frame := recordio.AppendFrame(make([]byte, 0, len(payload)+recordio.MaxHeaderLen), frameKind, payload)
	s.mu.Lock()
	defer s.mu.Unlock()
	off := s.off
	if _, err := s.f.WriteAt(frame, off); err != nil {
		return 0, err
	}
	s.off += int64(len(frame))
	s.size.Store(s.off)
	return off, nil
}

// Read returns the record that Append wrote at off.
func (s *File) Read(off int64) (goextract.FileResult, error) {
	data, err := s.readPayload(off)
	if err != nil {
		return goextract.FileResult{}, err
	}
	return Decode(data)
}

// readPayload reads one frame. The run wrote the file itself, so it sets
// no limit on how long a record can be. The only ceiling is the length of
// the file: a frame cannot be longer than the file that holds it. So a
// damaged length prefix gives an error, not an allocation of that length.
func (s *File) readPayload(off int64) ([]byte, error) {
	_, payload, _, err := recordio.ReadFrameAt(s.f, off, uint64(max(s.size.Load(), 1)))
	return payload, err
}

// Close closes the file. It does not remove it.
func (s *File) Close() error {
	return s.f.Close()
}
