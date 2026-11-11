package spill

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/rand/v2"
	"path/filepath"
	"sync"
	"testing"
)

func newFile(t *testing.T) *File {
	t.Helper()
	s, err := Create(filepath.Join(t.TempDir(), "spill"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// payloadFor returns the payload of record i of goroutine g: its length
// and its bytes follow from the two numbers, so a reader can check them.
// Every eighth record is empty.
func payloadFor(g, i int) []byte {
	if i%8 == 0 {
		return nil
	}
	return bytes.Repeat([]byte{byte(g*31 + i)}, 1+(g*7919+i*104729)%4096)
}

func TestSpillConcurrentAppends(t *testing.T) {
	const goroutines, perGoroutine = 8, 1000
	s := newFile(t)
	offsets := make([][]int64, goroutines)
	var wg sync.WaitGroup
	for g := range goroutines {
		offsets[g] = make([]int64, perGoroutine)
		wg.Go(func() {
			for i := range perGoroutine {
				off, err := s.appendPayload(payloadFor(g, i))
				if err != nil {
					t.Errorf("append: %v", err)
					return
				}
				offsets[g][i] = off
			}
		})
	}
	wg.Wait()

	seen := make(map[int64]bool, goroutines*perGoroutine)
	for g := range goroutines {
		for i, off := range offsets[g] {
			if seen[off] {
				t.Fatalf("offset %d was returned twice", off)
			}
			seen[off] = true
			got, err := s.readPayload(off)
			if err != nil {
				t.Fatalf("read at %d: %v", off, err)
			}
			if !bytes.Equal(got, payloadFor(g, i)) {
				t.Fatalf("record %d of goroutine %d at offset %d is not the record that was appended there", i, g, off)
			}
		}
	}
	if len(seen) != goroutines*perGoroutine {
		t.Fatalf("%d distinct offsets, want %d", len(seen), goroutines*perGoroutine)
	}
}

func TestSpillConcurrentReads(t *testing.T) {
	const goroutines, records = 8, 1000
	s := newFile(t)
	offsets := make([]int64, records)
	for i := range records {
		off, err := s.appendPayload(payloadFor(1, i))
		if err != nil {
			t.Fatalf("append: %v", err)
		}
		offsets[i] = off
	}

	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Go(func() {
			for _, i := range rand.New(rand.NewPCG(uint64(g), 1)).Perm(records) {
				got, err := s.readPayload(offsets[i])
				if err != nil {
					t.Errorf("read record %d: %v", i, err)
					return
				}
				if !bytes.Equal(got, payloadFor(1, i)) {
					t.Errorf("record %d did not come back", i)
					return
				}
			}
		})
	}
	wg.Wait()
}

// TestSpillReadsLargeRecord checks that the spill reader has no ceiling: a
// record one byte over the 64 MiB ceiling of an import reads back.
func TestSpillReadsLargeRecord(t *testing.T) {
	const size = 64<<20 + 1
	s := newFile(t)
	payload := make([]byte, size)
	payload[0], payload[size-1] = 0xA5, 0x5A
	if _, err := s.appendPayload([]byte("first")); err != nil {
		t.Fatalf("append: %v", err)
	}
	off, err := s.appendPayload(payload)
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	got, err := s.readPayload(off)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(got) != size || got[0] != 0xA5 || got[size-1] != 0x5A {
		t.Fatalf("read %d bytes, want the %d bytes that were appended", len(got), size)
	}
}

// TestSpillReadOfDamagedLength damages the length prefix of a record and
// checks that the read gives an error. A frame cannot be longer than the
// file, so the read does not try to allocate the length it finds.
func TestSpillReadOfDamagedLength(t *testing.T) {
	s := newFile(t)
	off, err := s.appendPayload([]byte("record"))
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	// Positive assertion: the record reads back before the damage.
	if got, err := s.readPayload(off); err != nil || string(got) != "record" {
		t.Fatalf("read before the damage: %q, %v", got, err)
	}
	for _, length := range []uint64{1 << 40, math.MaxUint64} {
		prefix := binary.AppendUvarint(nil, length)
		if _, err := s.f.WriteAt(prefix, off+1); err != nil {
			t.Fatalf("WriteAt: %v", err)
		}
		if got, err := s.readPayload(off); err == nil {
			t.Errorf("read of a record with the length %d gave %d bytes and no error", length, len(got))
		}
	}
}
