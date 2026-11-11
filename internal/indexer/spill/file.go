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

// writeBuffer is the number of appended bytes that a File collects before
// it writes them. The workers of pass 1 append under one mutex, and a
// system call for each record under that mutex makes them wait on each
// other.
const writeBuffer = 1 << 20

// File is a spill file. Append is safe for concurrent use, and so is Read,
// also while other goroutines append.
type File struct {
	f *os.File

	// mu covers off, buf and the write of the buffer, so each record is
	// contiguous at the offset that Append returns, and no two records
	// share an offset.
	mu  sync.Mutex
	off int64
	// buf holds the frames from the offset flushed up to off, which the
	// file does not hold yet.
	buf []byte
	// flushed is the length of the file. A reader takes it without mu: a
	// record below it is in the file, and a record at or above it needs a
	// flush first.
	flushed atomic.Int64

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

// appendPayload adds one frame at the end of the file. The frame goes
// into the buffer, and the buffer goes to the file once it is full.
func (s *File) appendPayload(payload []byte) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	off := s.off
	s.buf = recordio.AppendFrame(s.buf, frameKind, payload)
	s.off = s.flushed.Load() + int64(len(s.buf))
	s.size.Store(s.off)
	if len(s.buf) >= writeBuffer {
		if err := s.flushLocked(); err != nil {
			return 0, err
		}
	}
	return off, nil
}

// flushLocked writes the buffer to the file. The caller holds mu.
func (s *File) flushLocked() error {
	if len(s.buf) == 0 {
		return nil
	}
	if _, err := s.f.WriteAt(s.buf, s.flushed.Load()); err != nil {
		return err
	}
	s.flushed.Store(s.off)
	// A record far above the buffer size must not pin its memory.
	if cap(s.buf) > 2*writeBuffer {
		s.buf = nil
	}
	s.buf = s.buf[:0]
	return nil
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
	if off >= s.flushed.Load() {
		s.mu.Lock()
		err := s.flushLocked()
		s.mu.Unlock()
		if err != nil {
			return nil, err
		}
	}
	_, payload, _, err := recordio.ReadFrameAt(s.f, off, uint64(max(s.size.Load(), 1)))
	return payload, err
}

// Close closes the file. It does not remove it.
func (s *File) Close() error {
	return s.f.Close()
}
