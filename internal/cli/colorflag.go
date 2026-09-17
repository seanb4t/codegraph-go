package cli

import (
	"io"
	"os"

	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"
)

// colorFlagName is the persistent flag name registered on root (D-11).
const colorFlagName = "color"

// colorChoice is the closed enum --color accepts, implementing pflag.Value
// so an unknown value is rejected at parse time (D-09/D-11/CLI-03).
//
// PLACEHOLDER (RED phase, 04-03 Task 1): real validation lands in GREEN.
type colorChoice string

const (
	colorAuto   colorChoice = "auto"
	colorAlways colorChoice = "always"
	colorNever  colorChoice = "never"
)

func (c *colorChoice) String() string { return string(*c) }
func (c *colorChoice) Type() string   { return "string" }

// Set is a PLACEHOLDER: it accepts anything during RED. GREEN restricts it
// to auto|always|never.
func (c *colorChoice) Set(v string) error {
	*c = colorChoice(v)
	return nil
}

// addColorFlag is a PLACEHOLDER: it registers nothing during RED, so
// --color is not yet a recognized flag anywhere in the command tree.
func addColorFlag(root *cobra.Command) {}

// colorChoiceOf is a PLACEHOLDER: always returns colorAuto during RED.
func colorChoiceOf(cmd *cobra.Command) colorChoice {
	return colorAuto
}

// rewriteEnviron is a PLACEHOLDER: returns its input unchanged during RED.
func rewriteEnviron(choice colorChoice, environ []string) []string {
	return environ
}

// colorMode is the resolver's result: whether the styled branch should
// render, which colorprofile.Profile to downsample to, and whether the
// terminal's background is dark (D-09/D-10/D-11).
type colorMode struct {
	Styled  bool
	Profile colorprofile.Profile
	Dark    bool
}

// Writer is a PLACEHOLDER: returns w unwrapped during RED.
func (m colorMode) Writer(w io.Writer) io.Writer { return w }

// fdIsTerminal and queryDarkBackground are the two injectable seams
// (install.go idiom) colorflag_test.go overrides to exercise the
// dark-background query gate without a real pty.
var fdIsTerminal = func(f *os.File) bool { return false }
var queryDarkBackground = func(in, out *os.File) bool { return false }

// resolveColorFrom is a PLACEHOLDER: always returns the zero colorMode
// during RED, ignoring choice/environ/out/in and never calling the seams
// above or colorprofile.Detect.
func resolveColorFrom(choice colorChoice, environ []string, out io.Writer, in io.Reader) colorMode {
	return colorMode{}
}

// resolveColor is a PLACEHOLDER: always returns the zero colorMode during
// RED.
func resolveColor(cmd *cobra.Command) colorMode {
	return colorMode{}
}

// resolveColorStderr is a PLACEHOLDER: always returns the zero colorMode
// during RED.
func resolveColorStderr(cmd *cobra.Command) colorMode {
	return colorMode{}
}
