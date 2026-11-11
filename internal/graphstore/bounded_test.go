package graphstore

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// emptyBatchLen is the length of a Pebble batch with no record: its header.
const emptyBatchLen = 12

// fataler is what the helpers below need of a test: *testing.T has it, and
// so has *rapid.T.
type fataler interface {
	Helper()
	Fatalf(format string, args ...any)
}

// sizedNode returns the node number i, whose record grows with size.
func sizedNode(i, size int) *schema.Node {
	return &schema.Node{Id: fmt.Sprintf("n%05d", i), Name: strings.Repeat("x", size)}
}

func hasNode(t fataler, s GraphStore, i int) bool {
	t.Helper()
	r, err := s.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer r.Close()
	_, err = r.GetNode(fmt.Sprintf("n%05d", i))
	if err != nil && err != ErrNotFound {
		t.Fatalf("GetNode: %v", err)
	}
	return err == nil
}

func hasMeta(t fataler, s GraphStore) bool {
	t.Helper()
	r, err := s.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer r.Close()
	_, err = r.GetMeta()
	if err != nil && err != ErrNotFound {
		t.Fatalf("GetMeta: %v", err)
	}
	return err == nil
}

// boundedRun writes nodes of the given sizes through a bounded writer with
// cap, then the metadata record if withMeta is set, and returns the commits.
// After each commit it checks the positive assertion of the cap tests:
// every record so far is in the store, and no later record is. So every
// record is in exactly one committed batch, in order.
type boundedRun struct {
	commits []Commit
	nodes   int
}

func runBounded(t fataler, s GraphStore, cap int, sizes []int, withMeta, commit bool) boundedRun {
	t.Helper()
	var run boundedRun
	committed := 0
	w, err := s.NewBoundedWriter(cap, func(c Commit) error {
		run.commits = append(run.commits, c)
		committed += c.Records
		for i := range sizes {
			if got, want := hasNode(t, s, i), i < committed; got != want {
				t.Fatalf("after commit %d of %d records in all: node %d in the store = %v, want %v", len(run.commits), committed, i, got, want)
			}
		}
		if got, want := hasMeta(t, s), withMeta && committed == len(sizes)+1; got != want {
			t.Fatalf("after commit %d: metadata in the store = %v, want %v", len(run.commits), got, want)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("NewBoundedWriter: %v", err)
	}
	defer w.Close()
	for i, size := range sizes {
		if err := w.PutNode(sizedNode(i, size)); err != nil {
			t.Fatalf("PutNode: %v", err)
		}
	}
	if withMeta {
		if err := w.PutMeta(&schema.Meta{NodeCount: int64(len(sizes))}); err != nil {
			t.Fatalf("PutMeta: %v", err)
		}
	}
	run.nodes = len(sizes)
	if !commit {
		return run
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	total := len(sizes)
	if withMeta {
		total++
	}
	if committed != total {
		t.Fatalf("%d records are in a committed batch, want all %d", committed, total)
	}
	return run
}

func newStaging(t testing.TB) GraphStore {
	t.Helper()
	s, err := OpenStaging(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatalf("OpenStaging: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// lenOf returns the length of a batch that holds nodes of sizes: the value
// that the cap rows below place their cap around.
func lenOf(t *testing.T, sizes []int) int {
	t.Helper()
	run := runBounded(t, newStaging(t), 1<<30, sizes, false, true)
	if len(run.commits) != 1 {
		t.Fatalf("%d commits under a cap that no batch reaches, want 1", len(run.commits))
	}
	return run.commits[0].Len
}

func records(commits []Commit) []int {
	out := make([]int, len(commits))
	for i, c := range commits {
		out[i] = c.Records
	}
	return out
}

func TestBatchWriterCapEdges(t *testing.T) {
	three := []int{100, 100, 100}
	full := lenOf(t, three)
	one := lenOf(t, []int{100}) - emptyBatchLen

	for _, tc := range []struct {
		name     string
		cap      int
		sizes    []int
		withMeta bool
		// noCommit calls Close in place of Commit.
		noCommit    bool
		wantRecords []int
		check       func(t *testing.T, commits []Commit)
	}{
		{
			name: "over the cap, then one more record", cap: full - 1, sizes: append(append([]int(nil), three...), 100),
			wantRecords: []int{3, 1},
		},
		{
			name: "exactly the cap, then one more record", cap: full, sizes: append(append([]int(nil), three...), 100),
			wantRecords: []int{4},
			check: func(t *testing.T, commits []Commit) {
				c := commits[0]
				if c.BeforeLast != full || c.Len-full != one {
					t.Errorf("BeforeLast = %d and Len = %d; want BeforeLast at the cap %d and Len over it by the last record, %d bytes", c.BeforeLast, c.Len, full, one)
				}
			},
		},
		{
			name: "a first record of the cap plus one, then one more", cap: one - 1, sizes: []int{100, 100},
			wantRecords: []int{1, 1},
		},
		{
			name: "over the cap, then the metadata as the last record", cap: full - 1, sizes: three, withMeta: true,
			wantRecords: []int{3, 1},
		},
		{
			name: "a cap of 0 and two records", cap: 0, sizes: []int{100, 100},
			wantRecords: []int{1, 1},
		},
		{
			name: "no record", cap: full, sizes: nil,
			wantRecords: []int{},
		},
		{
			name: "records below the cap, then Close with no Commit", cap: 1 << 30, sizes: three, noCommit: true,
			wantRecords: []int{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newStaging(t)
			run := runBounded(t, s, tc.cap, tc.sizes, tc.withMeta, !tc.noCommit)

			if got := records(run.commits); fmt.Sprint(got) != fmt.Sprint(tc.wantRecords) {
				t.Fatalf("records of each committed batch = %v, want %v", got, tc.wantRecords)
			}
			for i, c := range run.commits {
				if c.Records == 0 || c.Len <= emptyBatchLen {
					t.Errorf("commit %d is empty: %+v", i, c)
				}
				if c.Sync {
					t.Errorf("commit %d used Sync on a staging store: %+v", i, c)
				}
				if c.Records > 1 && c.BeforeLast > tc.cap {
					t.Errorf("commit %d has BeforeLast %d above the cap %d: %+v", i, c.BeforeLast, tc.cap, c)
				}
			}
			if tc.check != nil {
				tc.check(t, run.commits)
			}
			if tc.noCommit {
				for i := range tc.sizes {
					if hasNode(t, s, i) {
						t.Errorf("node %d is in the store after a Close with no Commit", i)
					}
				}
			}
		})
	}
}

// TestBatchWriterSyncLast checks the commit modes on a store with the
// write-ahead log on: only the commit of Commit syncs, and it holds the
// metadata record.
func TestBatchWriterSyncLast(t *testing.T) {
	three := []int{100, 100, 100}
	full := lenOf(t, three)
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	run := runBounded(t, s, full-1, three, true, true)
	if got := records(run.commits); fmt.Sprint(got) != "[3 1]" {
		t.Fatalf("records of each committed batch = %v, want [3 1]", got)
	}
	if run.commits[0].Sync || !run.commits[1].Sync {
		t.Errorf("Sync of the two commits = %v, %v; want only the last one to sync", run.commits[0].Sync, run.commits[1].Sync)
	}
	if !hasMeta(t, s) {
		t.Errorf("the store holds no metadata after Commit")
	}
}

// TestBatchWriterCapProperty draws a cap and record sizes around it. The
// draws put weight on the caps at and around the length of an empty batch.
func TestBatchWriterCapProperty(t *testing.T) {
	caps := rapid.OneOf(
		rapid.SampledFrom([]int{0, 11, 12, 13}),
		rapid.IntRange(0, 4096),
		rapid.IntRange(0, 1<<20),
	)
	root := t.TempDir()
	draws := 0
	rapid.Check(t, func(t *rapid.T) {
		cap := caps.Draw(t, "cap")
		// A draw writes at most 8 MiB, so that a cap of 1 MiB does not turn
		// 200 records into 400 MiB of writes. The budget lowers the record
		// count, not the sizes, which still reach twice the cap.
		budget := 8 << 20
		var sizes []int
		for range rapid.IntRange(0, 200).Draw(t, "records") {
			size := rapid.IntRange(0, 2*cap).Draw(t, "size")
			if budget -= size; budget < 0 {
				break
			}
			sizes = append(sizes, size)
		}
		draws++
		s, err := OpenStaging(filepath.Join(root, fmt.Sprintf("store-%d", draws)))
		if err != nil {
			t.Fatalf("OpenStaging: %v", err)
		}
		defer s.Close()

		// runBounded checks that each record commits once and in order.
		run := runBounded(t, s, cap, sizes, false, true)

		if len(sizes) == 0 && len(run.commits) != 0 {
			t.Fatalf("%d commits with no record, want none", len(run.commits))
		}
		if len(run.commits) > len(sizes) {
			t.Fatalf("%d commits for %d records", len(run.commits), len(sizes))
		}
		for i, c := range run.commits {
			if c.Records == 0 {
				t.Fatalf("commit %d is empty", i)
			}
			if c.Records > 1 && c.BeforeLast > cap {
				t.Fatalf("commit %d: BeforeLast %d is above the cap %d with %d records", i, c.BeforeLast, cap, c.Records)
			}
		}
	})
}

func exportOf(t *testing.T, s GraphStore) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := s.Export(&buf); err != nil {
		t.Fatalf("Export: %v", err)
	}
	return buf.Bytes()
}

// sourceStore returns the export of a store with n nodes, and with a
// metadata record if withMeta is set.
func sourceExport(t *testing.T, n int, withMeta bool) []byte {
	t.Helper()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	sizes := make([]int, n)
	for i := range sizes {
		sizes[i] = 40
	}
	runBounded(t, s, 1<<30, sizes, withMeta, true)
	return exportOf(t, s)
}

// TestImportCommitsInBatches is the import scenario of index-build: an
// import under a low cap commits in several batches, and the metadata is in
// the last one.
func TestImportCommitsInBatches(t *testing.T) {
	const cap = 64 << 10
	input := sourceExport(t, 5000, true)
	if input[0] != ExportKindMeta {
		t.Fatalf("the first frame of the export has kind %d, want the metadata kind: the test would not prove that Import moves it", input[0])
	}
	dst, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dst.Close()

	var commits []Commit
	metaAfter := []bool{}
	err = importWith(dst, bytes.NewReader(input), cap, func(c Commit) error {
		commits = append(commits, c)
		metaAfter = append(metaAfter, hasMeta(t, dst))
		return nil
	})
	if err != nil {
		t.Fatalf("importWith: %v", err)
	}

	if len(commits) < 2 {
		t.Fatalf("%d batches committed, want more than one", len(commits))
	}
	for i, c := range commits {
		last := i == len(commits)-1
		if metaAfter[i] != last {
			t.Errorf("after commit %d of %d: metadata in the store = %v, want %v", i+1, len(commits), metaAfter[i], last)
		}
		if c.Sync != last {
			t.Errorf("commit %d of %d: Sync = %v, want %v", i+1, len(commits), c.Sync, last)
		}
		if c.Records > 1 && c.BeforeLast > cap {
			t.Errorf("commit %d: BeforeLast %d is above the cap %d", i+1, c.BeforeLast, cap)
		}
	}
	if !bytes.Equal(exportOf(t, dst), input) {
		t.Errorf("a new export of the imported store differs from the input")
	}
}

func TestImportEdgeStreams(t *testing.T) {
	t.Run("an empty stream", func(t *testing.T) {
		dst, err := Open(t.TempDir())
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer dst.Close()
		commits := 0
		if err := importWith(dst, bytes.NewReader(nil), 64<<10, func(Commit) error { commits++; return nil }); err != nil {
			t.Fatalf("importWith: %v", err)
		}
		if commits != 0 || len(exportOf(t, dst)) != 0 {
			t.Errorf("%d commits and an export of %d bytes, want an empty store and no commit", commits, len(exportOf(t, dst)))
		}
	})
	t.Run("records and no metadata record", func(t *testing.T) {
		input := sourceExport(t, 500, false)
		dst, err := Open(t.TempDir())
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer dst.Close()
		var commits []Commit
		if err := importWith(dst, bytes.NewReader(input), 4<<10, func(c Commit) error { commits = append(commits, c); return nil }); err != nil {
			t.Fatalf("importWith: %v", err)
		}
		if len(commits) < 2 {
			t.Fatalf("%d batches committed, want more than one", len(commits))
		}
		for i, c := range commits {
			if last := i == len(commits)-1; c.Sync != last {
				t.Errorf("commit %d of %d: Sync = %v, want %v", i+1, len(commits), c.Sync, last)
			}
		}
		if !bytes.Equal(exportOf(t, dst), input) {
			t.Errorf("a new export of the imported store differs from the input")
		}
	})
}

// failingStore is a GraphStore whose bounded Writer fails on its second
// add. It records what the Writer saw.
type failingStore struct {
	GraphStore
	adds, commits, closes int
}

type failingWriter struct {
	Writer
	s *failingStore
}

var errStubAdd = errors.New("stub: the second add fails")

func (s *failingStore) NewBoundedWriter(int, func(Commit) error) (Writer, error) {
	return &failingWriter{s: s}, nil
}

func (w *failingWriter) PutNode(*schema.Node) error {
	w.s.adds++
	if w.s.adds == 2 {
		return errStubAdd
	}
	return nil
}

func (w *failingWriter) Commit() error { w.s.commits++; return nil }
func (w *failingWriter) Close() error  { w.s.closes++; return nil }

// TestImportClosesBatchOnError checks that a failed write releases the
// open batch and commits nothing.
func TestImportClosesBatchOnError(t *testing.T) {
	input := sourceExport(t, 5, false)
	s := &failingStore{}
	err := importWith(s, bytes.NewReader(input), 64<<10, nil)
	if !errors.Is(err, errStubAdd) {
		t.Fatalf("importWith error = %v, want the error of the failed add", err)
	}
	// Positive assertion: the writer got a record before the failure.
	if s.adds != 2 || s.closes != 1 || s.commits != 0 {
		t.Errorf("adds = %d, closes = %d, commits = %d; want 2 adds, one Close and no Commit", s.adds, s.closes, s.commits)
	}
}

func TestOpenStaging(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "store")
	s, err := OpenStaging(dir)
	if err != nil {
		t.Fatalf("OpenStaging: %v", err)
	}
	ps := s.(*pebbleStore)

	// A Sync commit has no write-ahead log to sync, and a NoSync commit
	// works: the two facts that the commit modes of the bounded writer
	// follow from.
	batch := ps.db.NewBatch()
	if err := batch.Set(nodeKey("probe"), []byte("x"), nil); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := ps.db.Apply(batch, syncOption(true)); err == nil {
		t.Fatalf("a Sync commit succeeded on a staging store")
	}
	batch.Close()

	run := runBounded(t, s, 1<<30, []int{10, 10}, true, true)
	if len(run.commits) != 1 || run.commits[0].Sync {
		t.Fatalf("commits = %+v, want one commit without Sync", run.commits)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Positive assertion: Close flushed, so the records are on disk with
	// no write-ahead log to replay.
	again, err := OpenExisting(dir)
	if err != nil {
		t.Fatalf("OpenExisting after Close: %v", err)
	}
	defer again.Close()
	if !hasNode(t, again, 0) || !hasNode(t, again, 1) || !hasMeta(t, again) {
		t.Errorf("the staging store lost committed records across Close")
	}
}
