package present

import (
	"fmt"
	"io"
	"sync"
	"time"

	lipgloss "charm.land/lipgloss/v2"
)

// progressStyle renders each animated spinner frame glyph.
var progressStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

// spinnerFrames is the Braille-dot animation sequence rendered one frame
// per tick — the same glyph set RESEARCH Pattern 3 sketches.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// progressTickInterval is the animation cadence (RESEARCH Pattern 3:
// "cadence Claude's discretion, ~100ms" — tuned slightly faster here for a
// snappier human-facing animation without materially changing the visual
// effect).
const progressTickInterval = 80 * time.Millisecond

// clearLineSeq erases any partial frame left on the current line when
// Stop is called: \r returns the cursor to column 0, \x1b[K (EL — Erase in
// Line) clears from the cursor to the end of the line.
const clearLineSeq = "\r\x1b[K"

// Progress is a hand-rolled, non-interactive, TTY-gated progress
// indicator (TUI-05, D-08/D-09). It writes exactly one lipgloss-styled
// spinner frame per progressTickInterval tick to the io.Writer supplied at
// construction — callers at the internal/cli RunE boundary MUST pass
// os.Stderr, never os.Stdout (D-08: progress must never reach the
// stdout/MCP JSON-RPC stream). Progress reads no stdin and never spawns a
// bubbletea Program — it is built from lipgloss styling plus a stdlib
// time.Ticker only (D-09/D-13), never charm.land/bubbles or
// charm.land/bubbletea.
//
// Stop() terminates the ticker goroutine deterministically (T-06-08): it
// closes a stop channel and blocks until the goroutine has actually
// returned before clearing the line, so no goroutine is ever left running
// after Stop returns — the concurrency guarantee an interrupted or
// parallel index run depends on.
type Progress struct {
	w io.Writer

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	doneCh  chan struct{}
}

// NewProgress constructs a Progress that writes exclusively to w. w is
// never defaulted or substituted internally — Progress has no reference
// to os.Stdout anywhere in its implementation; the TTY/stream choice is
// entirely the caller's (D-08).
func NewProgress(w io.Writer) *Progress {
	return &Progress{w: w}
}

// Start launches the single ticker goroutine and begins rendering frames
// labeled with label. Calling Start on an already-running Progress is a
// no-op (it does not launch a second goroutine).
func (p *Progress) Start(label string) {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}
	p.running = true
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	p.stopCh = stopCh
	p.doneCh = doneCh
	p.mu.Unlock()

	go func() {
		defer close(doneCh)

		ticker := time.NewTicker(progressTickInterval)
		defer ticker.Stop()

		i := 0
		for {
			select {
			case <-stopCh:
				fmt.Fprint(p.w, clearLineSeq)
				return
			case <-ticker.C:
				frame := progressStyle.Render(spinnerFrames[i%len(spinnerFrames)])
				fmt.Fprintf(p.w, "\r%s %s", frame, label)
				i++
			}
		}
	}()
}

// Stop terminates the ticker goroutine deterministically: it closes the
// stop channel and blocks until the goroutine has confirmed it returned
// (via doneCh) before Stop itself returns — by the time Stop returns, the
// line-clear sequence has already been written and the goroutine is
// guaranteed gone (no leak). Calling Stop on a Progress that was never
// started, or calling it a second time, is a safe no-op (idempotent).
func (p *Progress) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	stopCh := p.stopCh
	doneCh := p.doneCh
	p.mu.Unlock()

	close(stopCh)
	<-doneCh
}
