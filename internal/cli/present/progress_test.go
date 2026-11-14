package present

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// waitForFrames sleeps long enough for several progressTickInterval ticks
// to fire, so the test can assert on rendered frame content deterministically
// without depending on a single, potentially-missed tick.
func waitForFrames() {
	time.Sleep(4 * progressTickInterval)
}

// TestProgress_FramesContainLabelAndANSI covers Test 1 (frames render the
// label) and Test 3 (frames are lipgloss-styled ANSI, redrawn via \r) from
// the plan's behavior spec.
func TestProgress_FramesContainLabelAndANSI(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf)

	p.Start("indexing")
	waitForFrames()
	p.Stop()

	got := buf.String()
	if !strings.Contains(got, "indexing") {
		t.Errorf("Progress output %q does not contain label %q", got, "indexing")
	}
	if !strings.Contains(got, "\x1b[") {
		t.Errorf("Progress output %q does not contain any ANSI escape sequence (lipgloss styling expected)", got)
	}
	if !strings.Contains(got, "\r") {
		t.Errorf("Progress output %q does not contain a carriage-return redraw sequence", got)
	}
}

// TestProgress_StderrOnly covers Test 2: writes go ONLY to the injected
// io.Writer. This redirects the REAL os.Stdout to a pipe, runs a Progress
// against a completely separate buffer, and asserts zero bytes ever reach
// the redirected stdout capture — proving Progress has no internal
// reference to os.Stdout (D-08).
func TestProgress_StderrOnly(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = origStdout })

	var target bytes.Buffer
	p := NewProgress(&target)
	p.Start("syncing")
	waitForFrames()
	p.Stop()

	os.Stdout = origStdout
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}

	var captured bytes.Buffer
	if _, err := io.Copy(&captured, r); err != nil {
		t.Fatalf("read stdout capture: %v", err)
	}

	if captured.Len() != 0 {
		t.Errorf("Progress wrote %d bytes to os.Stdout, want 0 — writer must be exclusively the injected io.Writer (D-08): %q", captured.Len(), captured.String())
	}
	if target.Len() == 0 {
		t.Fatal("target buffer received zero bytes — Progress did not render anything, test cannot validate stderr-only discipline")
	}
}

// TestProgress_StopClearsLine covers Test 4's clear-line half: after Stop
// returns, the output ends with the \r\x1b[K clear sequence and no
// dangling partial frame remains.
func TestProgress_StopClearsLine(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf)

	p.Start("indexing")
	waitForFrames()
	p.Stop()

	got := buf.String()
	if !strings.HasSuffix(got, clearLineSeq) {
		t.Errorf("Progress output %q does not end with the clear-line sequence %q", got, clearLineSeq)
	}
}

// TestProgress_StopIdempotent covers Test 4's idempotency half: calling
// Stop multiple times (including on a Progress that was never Started)
// must never panic.
func TestProgress_StopIdempotent(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf)

	p.Start("indexing")
	waitForFrames()
	p.Stop()
	p.Stop() // second call must be a safe no-op

	np := NewProgress(&bytes.Buffer{})
	np.Stop() // Stop without a prior Start must also be a safe no-op
}

// TestProgress_NoGoroutineLeak checks that the ticker goroutine runs
// until Stop and has exited when Stop returns. The goroutine closes doneCh
// when it returns. TestMain's goleak check covers the whole package.
func TestProgress_NoGoroutineLeak(t *testing.T) {
	p := NewProgress(&bytes.Buffer{})
	p.Start("indexing")
	p.mu.Lock()
	done := p.doneCh
	p.mu.Unlock()
	if done == nil {
		t.Fatal("Start did not launch the ticker goroutine")
	}
	select {
	case <-done:
		t.Fatal("the ticker goroutine exited before Stop")
	default:
	}

	p.Stop()
	select {
	case <-done:
	default:
		t.Fatal("Stop returned while the ticker goroutine was still running")
	}
}
