// editorurl.go resolves codegraph ui's editor-link configuration ONCE at
// startup (D-14/D-15/D-16), before uiserver.Listen ever binds a port.
// This is the CLI's operator boundary counterpart to
// internal/uiserver/editorlink.go's per-request handler: both call the
// SAME uiserver.ValidateEditorTemplate — this package declares no
// second validator.
package cli

import (
	"fmt"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/uiserver"
)

// editorURLEnvVar and noEditorURLEnvVar are the two environment
// variables `codegraph ui` reads for editor-link configuration
// (D-14/D-16), following internal/watch/debounce.go's and
// internal/corpora/manifest.go's existing CODEGRAPH_* named-const, read-
// once convention.
const (
	editorURLEnvVar   = "CODEGRAPH_EDITOR_URL"
	noEditorURLEnvVar = "CODEGRAPH_NO_EDITOR_URL"
)

// discoveredEditor is the shape plan 09-02's discoverEditor returns:
// which launcher was found on PATH or in a platform app directory, the
// matching preset id, and the template to use.
type discoveredEditor struct {
	Launcher string
	PresetID string
	Template string
}

// editorResolveInputs is resolveEditorLink's dependency-injected input
// set, so the full precedence table is table-tested without touching a
// real environment or a real filesystem probe.
type editorResolveInputs struct {
	// flagTemplate is --editor-url's raw value ("" means unset).
	flagTemplate string
	// noEditorURL is --no-editor-url's value.
	noEditorURL bool
	// getenv is os.Getenv in production; a map-backed fake in tests.
	getenv func(string) string
	// discover is plan 09-02's discoverEditor in production; nil means
	// "no discovered-default source is available at all" (distinct
	// from a non-nil discover that simply finds nothing).
	discover func() (discoveredEditor, bool)
}

// resolveEditorLink resolves the server's effective editor-link default
// ONCE, in this exact precedence order, each step documented with its
// decision id:
//
//  1. D-17: explicit values are validated FIRST — the flag value AND
//     the env value are BOTH checked, even though only one can win, an
//     operator who typed a template meant it, so a typo in the losing
//     source must not be silently ignored.
//  2. D-16: the off switch — checked only after both explicit values
//     have passed validation, so a malformed explicit value never
//     silently falls through to "disabled" either.
//  3. D-14: flag, then env, then discovery (run here — once — and
//     never per request, Pattern 3), then unconfigured.
//
// A nil discover means the discovered-default source is absent
// entirely; plan 09-02's discoverEditor is the production value passed
// from newUiCmd. A config file is deliberately not a source in this
// phase (09-CONTEXT.md Deferred Ideas).
//
// STUB (RED phase, plan 09-01 Task 2): always returns a zero-value
// uiserver.EditorLinkOptions and a nil error, so the named tests fail on
// their own assertions rather than on a missing symbol. Replaced with
// the real precedence chain in the GREEN commit.
func resolveEditorLink(in editorResolveInputs) (uiserver.EditorLinkOptions, error) {
	return uiserver.EditorLinkOptions{}, nil
}

// parseBoolEnv parses a CODEGRAPH_NO_EDITOR_URL-shaped boolean env
// value: "true"/"1"/"yes" (case-insensitive) is on; ""/"false"/"0"/"no"
// (case-insensitive) is off; anything else is an error naming the
// offending value — the caller (resolveEditorLink) wraps it with the
// variable name.
func parseBoolEnv(v string) (bool, error) {
	switch strings.ToLower(v) {
	case "", "false", "0", "no":
		return false, nil
	case "true", "1", "yes":
		return true, nil
	default:
		return false, fmt.Errorf("invalid value %q (want true|1|yes or false|0|no)", v)
	}
}
