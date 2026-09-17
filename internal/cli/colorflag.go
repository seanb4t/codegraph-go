package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/seanb4t/codegraph-go/internal/cli/present"
)

// colorFlagName is the persistent flag name registered on root (D-11).
const colorFlagName = "color"

// colorChoice is the closed enum --color accepts, implementing pflag.Value
// (String/Set/Type) so an unknown value is rejected at cobra's own
// flag-parse time via the caller's usual single error path — the same
// idiom install.go's parseLocationFlag uses for --location (D-09/D-11/
// CLI-03).
type colorChoice string

const (
	colorAuto   colorChoice = "auto"
	colorAlways colorChoice = "always"
	colorNever  colorChoice = "never"
)

func (c *colorChoice) String() string { return string(*c) }
func (c *colorChoice) Type() string   { return "string" }

// Set accepts exactly auto|always|never; anything else (including the
// empty string from a bare --color=) is a usage error naming all three
// values, matching parseLocationFlag's error shape.
func (c *colorChoice) Set(v string) error {
	switch colorChoice(v) {
	case colorAuto, colorAlways, colorNever:
		*c = colorChoice(v)
		return nil
	default:
		return fmt.Errorf("--color must be \"auto\", \"always\" or \"never\" (got %q)", v)
	}
}

// addColorFlag registers the persistent --color flag on root, default
// auto. No short form — -c is not one of D-12's four pinned shorts.
func addColorFlag(root *cobra.Command) {
	choice := colorAuto
	root.PersistentFlags().Var(&choice, colorFlagName, "colour output: auto (detect), always, or never")
}

// colorChoiceOf reads the --color value off cmd's own or inherited flag
// set. A nil lookup (a standalone command built without root, as some
// tests do) falls back to colorAuto rather than panicking.
func colorChoiceOf(cmd *cobra.Command) colorChoice {
	f := cmd.Flags().Lookup(colorFlagName)
	if f == nil {
		return colorAuto
	}
	return colorChoice(f.Value.String())
}

// rewriteEnviron is the D-09 environ rewrite --color applies before the
// resolver's single colorprofile.Detect call. Below --color, precedence is
// colorprofile@v0.4.3's own (NO_COLOR > CLICOLOR_FORCE > CLICOLOR > TTY;
// github.com/charmbracelet/colorprofile@v0.4.3 env.go) — cited here, never
// re-tested (D-00). Node's FORCE_COLOR convention is the opposite of
// CLICOLOR_FORCE's "force on" semantics, so nobody should "fix" this
// precedence by analogy.
//
//   - always: drops every NO_COLOR=/CLICOLOR= entry and appends
//     CLICOLOR_FORCE=1.
//   - never: appends NO_COLOR=1 AND drops every CLICOLOR_FORCE=/CLICOLOR=
//     entry (amendment A1, verified this session against v0.4.3:
//     colorProfile's cliColorForced branch returns before NO_COLOR is ever
//     consulted when the output is not a TTY — env.go's colorProfile only
//     checks `envNoColor(env) && isatty` before falling through to
//     `cliColorForced(env)`, so on a pipe a bare NO_COLOR=1 never reaches
//     the floor-at-ASCII branch at all if CLICOLOR_FORCE=1 is also set;
//     without dropping CLICOLOR_FORCE, `CLICOLOR_FORCE=1 ... --color=never
//     | cat` would still render styled).
//   - auto: passes through, with two normalizations of OUR OWN input
//     before it reaches colorprofile, never a re-test of colorprofile's
//     parsing:
//     (1) a non-empty NO_COLOR is rewritten to NO_COLOR=1 — colorprofile's
//     envNoColor is strconv.ParseBool-gated (env.go), so a raw
//     NO_COLOR=banana would NOT disable colour there, diverging from this
//     project's historical present.ChoosePresentation contract (any
//     non-empty value disables). An empty NO_COLOR= is left untouched
//     ("unset").
//     (2) amendment A2: a non-empty CLICOLOR whose value strconv.ParseBool
//     reads as false (0, false, …) appends NO_COLOR=1, UNLESS
//     CLICOLOR_FORCE is ParseBool-true. colorprofile only floors on a
//     truthy CLICOLOR and never lowers on a falsy one (verified this
//     session: CLICOLOR=0 alone is a no-op against v0.4.3), so without
//     this normalization CLI-03's stated NO_COLOR > CLICOLOR_FORCE >
//     CLICOLOR=0 > TTY order would not actually hold for a user's own
//     CLICOLOR=0.
//
// The input slice is never mutated — a fresh slice is built and returned.
func rewriteEnviron(choice colorChoice, environ []string) []string {
	out := make([]string, 0, len(environ)+1)
	var clicolorVal string
	var clicolorForceVal string
	sawClicolor := false
	sawClicolorForce := false

	for _, e := range environ {
		switch {
		case strings.HasPrefix(e, "NO_COLOR="):
			switch choice {
			case colorAlways, colorNever:
				continue // dropped; never's own NO_COLOR=1 is appended below
			default: // auto
				v := strings.TrimPrefix(e, "NO_COLOR=")
				if v != "" {
					out = append(out, "NO_COLOR=1")
				} else {
					out = append(out, e)
				}
			}
		case strings.HasPrefix(e, "CLICOLOR_FORCE="):
			sawClicolorForce = true
			clicolorForceVal = strings.TrimPrefix(e, "CLICOLOR_FORCE=")
			switch choice {
			case colorNever, colorAlways:
				continue // A1 (never) / superseded by our own CLICOLOR_FORCE=1 (always)
			default:
				out = append(out, e)
			}
		case strings.HasPrefix(e, "CLICOLOR="):
			sawClicolor = true
			clicolorVal = strings.TrimPrefix(e, "CLICOLOR=")
			switch choice {
			case colorNever, colorAlways:
				continue
			default:
				out = append(out, e)
			}
		default:
			out = append(out, e)
		}
	}

	switch choice {
	case colorAlways:
		out = append(out, "CLICOLOR_FORCE=1")
	case colorNever:
		out = append(out, "NO_COLOR=1")
	default: // auto — amendment A2
		if sawClicolor && clicolorVal != "" {
			if b, err := strconv.ParseBool(clicolorVal); err == nil && !b {
				forced := false
				if sawClicolorForce && clicolorForceVal != "" {
					if fb, ferr := strconv.ParseBool(clicolorForceVal); ferr == nil && fb {
						forced = true
					}
				}
				if !forced {
					out = append(out, "NO_COLOR=1")
				}
			}
		}
	}

	return out
}

// colorMode is the resolver's result: whether the styled branch should
// render, which colorprofile.Profile downstream downsampling should use,
// and whether the terminal's background is dark.
type colorMode struct {
	Styled  bool
	Profile colorprofile.Profile
	Dark    bool
}

// Writer wraps w in a colorprofile.Writer at m.Profile — the RunE
// boundary's downsampling step (D-10). present never sees the profile.
func (m colorMode) Writer(w io.Writer) io.Writer {
	return &colorprofile.Writer{Forward: w, Profile: m.Profile}
}

// fdIsTerminal and queryDarkBackground are injectable seams (the
// install.go idiom) so colorflag_test.go can force the dark-background
// query gate without a real pty.
var fdIsTerminal = func(f *os.File) bool { return term.IsTerminal(int(f.Fd())) }
var queryDarkBackground = func(in, out *os.File) bool { return lipgloss.HasDarkBackground(in, out) }

// resolveColorFrom is the D-09/D-10/D-11 resolver: rewrite the environ per
// choice, call colorprofile.Detect exactly once, derive styled/noColor
// through the UNCHANGED present.ChoosePresentation, and — only when
// styled AND both out and in are real terminal *os.File values — query
// the background exactly once (D-11 corrected gate: lipgloss's
// HasDarkBackground, charm.land/lipgloss/v2@v2.0.5 query.go, requires
// BOTH ends to be a TTY and puts stdin into raw mode, blocking up to its
// own 2s defaultQueryTimeout on a non-answering terminal — never issued
// on a pipe or redirected stdin).
func resolveColorFrom(choice colorChoice, environ []string, out io.Writer, in io.Reader) colorMode {
	env := rewriteEnviron(choice, environ)
	p := colorprofile.Detect(out, env)
	isTTY := p > colorprofile.NoTTY
	noColor := ""
	if p <= colorprofile.ASCII {
		noColor = "1"
	}
	styled := present.ChoosePresentation(isTTY, noColor)

	dark := true
	if styled {
		of, ok1 := out.(*os.File)
		inf, ok2 := in.(*os.File)
		if ok1 && ok2 && fdIsTerminal(of) && fdIsTerminal(inf) {
			dark = queryDarkBackground(inf, of)
		}
	}

	return colorMode{Styled: styled, Profile: p, Dark: dark}
}

// resolveColor is the single environment read for a command's stdout
// path: colorChoiceOf(cmd) resolves --color, os.Environ() is read exactly
// once, and cmd.OutOrStdout()/cmd.InOrStdin() supply the real fds a
// styled RunE branch renders through.
func resolveColor(cmd *cobra.Command) colorMode {
	return resolveColorFrom(colorChoiceOf(cmd), os.Environ(), cmd.OutOrStdout(), cmd.InOrStdin())
}

// resolveColorStderr mirrors resolveColor for stderr-bound chrome (plan
// 07's serve banners) — same environ rewrite, Detect on
// cmd.ErrOrStderr(), but NEVER queries the background: Dark always
// defaults true. progress_cli.go's own, separate stderr resolver is out
// of scope for this phase (04-CONTEXT.md <domain>) and stays untouched.
func resolveColorStderr(cmd *cobra.Command) colorMode {
	choice := colorChoiceOf(cmd)
	env := rewriteEnviron(choice, os.Environ())
	p := colorprofile.Detect(cmd.ErrOrStderr(), env)
	isTTY := p > colorprofile.NoTTY
	noColor := ""
	if p <= colorprofile.ASCII {
		noColor = "1"
	}
	styled := present.ChoosePresentation(isTTY, noColor)
	return colorMode{Styled: styled, Profile: p, Dark: true}
}
