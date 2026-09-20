package cli

import (
	"bytes"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"
)

// sortedEnv returns a sorted copy of env, for set-equality comparison
// against rewriteEnviron's output (order is not part of its contract —
// only which entries survive/get added, D-09's own <behavior> table never
// specifies an order).
func sortedEnv(env []string) []string {
	out := append([]string(nil), env...)
	sort.Strings(out)
	return out
}

// TestRewriteEnviron pins D-09's environ-rewrite rules — including
// amendments A1 (never also drops CLICOLOR_FORCE/CLICOLOR) and A2 (auto
// normalizes a ParseBool-false CLICOLOR to NO_COLOR=1 unless
// CLICOLOR_FORCE is truthy) — and that the input slice itself is never
// mutated. Verified empirically against colorprofile@v0.4.3 this session
// (04-03-PLAN.md's own <behavior> table).
func TestRewriteEnviron(t *testing.T) {
	tests := []struct {
		name    string
		choice  colorChoice
		environ []string
		want    []string
	}{
		{
			name:    "always drops NO_COLOR/CLICOLOR, forces CLICOLOR_FORCE",
			choice:  colorAlways,
			environ: []string{"NO_COLOR=1", "CLICOLOR=1", "TERM=x"},
			want:    []string{"TERM=x", "CLICOLOR_FORCE=1"},
		},
		{
			name:    "never drops CLICOLOR_FORCE/CLICOLOR (A1), sets NO_COLOR",
			choice:  colorNever,
			environ: []string{"CLICOLOR_FORCE=1", "CLICOLOR=1", "TERM=x"},
			want:    []string{"TERM=x", "NO_COLOR=1"},
		},
		{
			name:    "auto normalizes a non-empty non-boolean NO_COLOR to 1",
			choice:  colorAuto,
			environ: []string{"NO_COLOR=banana"},
			want:    []string{"NO_COLOR=1"},
		},
		{
			name:    "auto leaves an empty NO_COLOR alone",
			choice:  colorAuto,
			environ: []string{"NO_COLOR="},
			want:    []string{"NO_COLOR="},
		},
		{
			name:    "auto appends NO_COLOR=1 for a ParseBool-false CLICOLOR (A2)",
			choice:  colorAuto,
			environ: []string{"CLICOLOR=0", "TERM=x"},
			want:    []string{"CLICOLOR=0", "TERM=x", "NO_COLOR=1"},
		},
		{
			name:    "auto's A2 normalization is skipped when CLICOLOR_FORCE is truthy",
			choice:  colorAuto,
			environ: []string{"CLICOLOR=0", "CLICOLOR_FORCE=1"},
			want:    []string{"CLICOLOR=0", "CLICOLOR_FORCE=1"},
		},
		{
			name:    "auto leaves a truthy CLICOLOR untouched",
			choice:  colorAuto,
			environ: []string{"CLICOLOR=1"},
			want:    []string{"CLICOLOR=1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputCopy := append([]string(nil), tt.environ...)

			got := rewriteEnviron(tt.choice, tt.environ)

			if gs, ws := sortedEnv(got), sortedEnv(tt.want); !reflect.DeepEqual(gs, ws) {
				t.Errorf("rewriteEnviron(%q, %v) = %v, want (set-equal to) %v", tt.choice, tt.environ, got, tt.want)
			}
			if !reflect.DeepEqual(tt.environ, inputCopy) {
				t.Errorf("rewriteEnviron(%q, ...) mutated its input slice: got %v, want unchanged %v", tt.choice, tt.environ, inputCopy)
			}
		})
	}
}

// colorMatrixCase is one TestResolveColorMatrix row: a (choice, environ)
// input and an assertion function so exact-profile rows and
// styled-only rows share one table.
type colorMatrixCase struct {
	name    string
	choice  colorChoice
	environ []string
	check   func(t *testing.T, mode colorMode)
}

func wantStyled(want bool) func(t *testing.T, mode colorMode) {
	return func(t *testing.T, mode colorMode) {
		if mode.Styled != want {
			t.Errorf("Styled = %v, want %v (profile=%v)", mode.Styled, want, mode.Profile)
		}
	}
}

func wantStyledAndProfile(styled bool, profile colorprofile.Profile) func(t *testing.T, mode colorMode) {
	return func(t *testing.T, mode colorMode) {
		if mode.Styled != styled {
			t.Errorf("Styled = %v, want %v", mode.Styled, styled)
		}
		if mode.Profile != profile {
			t.Errorf("Profile = %v, want %v", mode.Profile, profile)
		}
	}
}

func wantStyledAndMinProfile(styled bool, min colorprofile.Profile) func(t *testing.T, mode colorMode) {
	return func(t *testing.T, mode colorMode) {
		if mode.Styled != styled {
			t.Errorf("Styled = %v, want %v", mode.Styled, styled)
		}
		if mode.Profile < min {
			t.Errorf("Profile = %v, want >= %v", mode.Profile, min)
		}
	}
}

// TestResolveColorMatrix covers the (choice × TTY × env) matrix we own —
// resolveColorFrom's Styled/Profile output — over 13 rows. tty rows carry
// TTY_FORCE=1 in the environ (colorprofile's own forced-isatty hook,
// v0.4.3 env.go, used purely as a test input per the plan's interfaces
// block); pipe rows omit it and rely on the bytes.Buffer `out` not
// satisfying term.File. Exact Profile values for four rows were verified
// empirically this session by running colorprofile@v0.4.3 directly
// (04-03-PLAN.md's <behavior> + interfaces block); the remaining rows
// assert Styled only, per D-00 (never re-test colorprofile's own
// precedence beyond what our resolver's contract needs).
func TestResolveColorMatrix(t *testing.T) {
	tests := []colorMatrixCase{
		{name: "always/pipe/empty", choice: colorAlways, environ: nil, check: wantStyledAndMinProfile(true, colorprofile.ANSI)},
		{name: "always/pipe/NO_COLOR=1", choice: colorAlways, environ: []string{"NO_COLOR=1"}, check: wantStyled(true)},
		{name: "always/pipe/TERM=dumb", choice: colorAlways, environ: []string{"TERM=dumb"}, check: wantStyledAndProfile(true, colorprofile.ANSI)},
		{name: "never/tty/CLICOLOR_FORCE", choice: colorNever, environ: []string{"CLICOLOR_FORCE=1", "TERM=xterm-256color", "TTY_FORCE=1"}, check: wantStyled(false)},
		{name: "never/pipe/CLICOLOR_FORCE", choice: colorNever, environ: []string{"CLICOLOR_FORCE=1"}, check: wantStyled(false)},
		{name: "auto/pipe/TERM=xterm-256color", choice: colorAuto, environ: []string{"TERM=xterm-256color"}, check: wantStyledAndProfile(false, colorprofile.NoTTY)},
		{name: "auto/tty/TERM=xterm-256color", choice: colorAuto, environ: []string{"TERM=xterm-256color", "TTY_FORCE=1"}, check: wantStyledAndProfile(true, colorprofile.ANSI256)},
		{name: "auto/tty/TERM=xterm", choice: colorAuto, environ: []string{"TERM=xterm", "TTY_FORCE=1"}, check: wantStyledAndProfile(true, colorprofile.ANSI)},
		{name: "auto/tty/TERM=dumb", choice: colorAuto, environ: []string{"TERM=dumb", "TTY_FORCE=1"}, check: wantStyled(false)},
		{name: "auto/tty/NO_COLOR=banana", choice: colorAuto, environ: []string{"NO_COLOR=banana", "TERM=xterm-256color", "TTY_FORCE=1"}, check: wantStyled(false)},
		{name: "auto/tty/NO_COLOR=empty", choice: colorAuto, environ: []string{"NO_COLOR=", "TERM=xterm-256color", "TTY_FORCE=1"}, check: wantStyled(true)},
		{name: "auto/tty/CLICOLOR=0", choice: colorAuto, environ: []string{"CLICOLOR=0", "TERM=xterm-256color", "TTY_FORCE=1"}, check: wantStyled(false)},
		{name: "auto/tty/CLICOLOR=0/CLICOLOR_FORCE=1", choice: colorAuto, environ: []string{"CLICOLOR=0", "CLICOLOR_FORCE=1", "TERM=xterm-256color", "TTY_FORCE=1"}, check: wantStyled(true)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			var in bytes.Buffer
			mode := resolveColorFrom(tt.choice, tt.environ, &out, &in)
			tt.check(t, mode)
		})
	}
}

// TestColorFlagInvalidValue covers D-11's usage-error contract:
// --color=bogus and --color= (empty) are both rejected naming all three
// accepted values, and --color=never produces byte-identical output to
// the bare (flagless) invocation — execCmd's bytes.Buffer stdout is never
// a TTY, so the plain path is already selected without the flag.
func TestColorFlagInvalidValue(t *testing.T) {
	dir := setupIndexedFixture(t)

	for _, bad := range []string{"bogus", ""} {
		args := []string{"status", "-p", dir, "--color=" + bad}
		_, stderr, err := execCmd(args...)
		if err == nil {
			t.Fatalf("execCmd(%v): expected an error, got none", args)
		}
		msg := err.Error() + " " + stderr
		for _, want := range []string{"auto", "always", "never"} {
			if !strings.Contains(msg, want) {
				t.Errorf("execCmd(%v): error/stderr missing %q: %q", args, want, msg)
			}
		}
	}

	bareOut, _, err := execCmd("status", "-p", dir)
	if err != nil {
		t.Fatalf("execCmd(status -p %s): unexpected error: %v", dir, err)
	}
	neverOut, _, err := execCmd("status", "-p", dir, "--color=never")
	if err != nil {
		t.Fatalf("execCmd(status -p %s --color=never): unexpected error: %v", dir, err)
	}
	if neverOut != bareOut {
		t.Errorf("status --color=never output differs from the bare (no-flag) plain output:\n--- bare ---\n%s\n--- never ---\n%s", bareOut, neverOut)
	}
}

// TestDarkBackgroundQueryGate covers D-11's corrected gate: the
// lipgloss.HasDarkBackground seam (queryDarkBackground) fires at most
// once, and only when Styled AND both out and in are *os.File values
// fdIsTerminal reports as real terminals — never on a bytes.Buffer, never
// when only one side is a terminal, never when the resolver chose the
// plain branch.
func TestDarkBackgroundQueryGate(t *testing.T) {
	origFdIsTerminal := fdIsTerminal
	origQueryDarkBackground := queryDarkBackground
	t.Cleanup(func() {
		fdIsTerminal = origFdIsTerminal
		queryDarkBackground = origQueryDarkBackground
	})

	newCounterStub := func(answer bool) (*int, func(in, out *os.File) bool) {
		calls := 0
		return &calls, func(in, out *os.File) bool {
			calls++
			return answer
		}
	}

	t.Run("both stdin and stdout are terminals: queried exactly once", func(t *testing.T) {
		fdIsTerminal = func(f *os.File) bool { return true }
		calls, stub := newCounterStub(false)
		queryDarkBackground = stub

		mode := resolveColorFrom(colorAuto, []string{"TTY_FORCE=1", "TERM=xterm-256color"}, os.Stdout, os.Stdin)

		if !mode.Styled {
			t.Fatalf("precondition failed: expected Styled=true, got false (profile=%v)", mode.Profile)
		}
		if *calls != 1 {
			t.Errorf("queryDarkBackground called %d times, want 1", *calls)
		}
		if mode.Dark {
			t.Errorf("Dark = %v, want false (the stub's answer)", mode.Dark)
		}
	})

	t.Run("stdout is a terminal but stdin is not: never queried, defaults dark", func(t *testing.T) {
		fdIsTerminal = func(f *os.File) bool { return f == os.Stdout }
		calls, stub := newCounterStub(false)
		queryDarkBackground = stub

		mode := resolveColorFrom(colorAuto, []string{"TTY_FORCE=1", "TERM=xterm-256color"}, os.Stdout, os.Stdin)

		if !mode.Styled {
			t.Fatalf("precondition failed: expected Styled=true, got false (profile=%v)", mode.Profile)
		}
		if *calls != 0 {
			t.Errorf("queryDarkBackground called %d times, want 0", *calls)
		}
		if !mode.Dark {
			t.Errorf("Dark = %v, want true (default)", mode.Dark)
		}
	})

	t.Run("a bytes.Buffer stdout is never a terminal: never queried, defaults dark", func(t *testing.T) {
		fdIsTerminal = func(f *os.File) bool { return true }
		calls, stub := newCounterStub(false)
		queryDarkBackground = stub

		var out bytes.Buffer
		mode := resolveColorFrom(colorAuto, []string{"TTY_FORCE=1", "TERM=xterm-256color"}, &out, os.Stdin)

		if !mode.Styled {
			t.Fatalf("precondition failed: expected Styled=true, got false (profile=%v)", mode.Profile)
		}
		if *calls != 0 {
			t.Errorf("queryDarkBackground called %d times, want 0", *calls)
		}
		if !mode.Dark {
			t.Errorf("Dark = %v, want true (default)", mode.Dark)
		}
	})

	t.Run("styled false (never): never queried", func(t *testing.T) {
		fdIsTerminal = func(f *os.File) bool { return true }
		calls, stub := newCounterStub(false)
		queryDarkBackground = stub

		mode := resolveColorFrom(colorNever, []string{"TTY_FORCE=1", "TERM=xterm-256color"}, os.Stdout, os.Stdin)

		if mode.Styled {
			t.Fatalf("precondition failed: expected Styled=false, got true")
		}
		if *calls != 0 {
			t.Errorf("queryDarkBackground called %d times, want 0", *calls)
		}
	})
}
