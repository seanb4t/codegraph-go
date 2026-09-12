package cli

import (
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/uiserver"
)

// fakeGetenv builds an editorResolveInputs.getenv backed by a map, so
// each test controls exactly the environment resolveEditorLink observes
// without touching a real process environment.
func fakeGetenv(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

// countingDiscover wraps a discoveredEditor result in a func that
// increments calls each time it is invoked, so a test can assert
// discovery was never reached (D-16, D-17's fail-fast paths).
func countingDiscover(calls *int, result discoveredEditor, found bool) func() (discoveredEditor, bool) {
	return func() (discoveredEditor, bool) {
		*calls++
		return result, found
	}
}

const vscodeTemplate = "vscode://file/{path}:{line}:{col}"

// TestResolveEditorLinkPrecedence tables every precedence row from this
// plan's <behavior> block: flag > env > discovered > none, both explicit
// values validated regardless of which wins, and discovery's nil-vs-not-
// found distinction.
func TestResolveEditorLinkPrecedence(t *testing.T) {
	cases := []struct {
		name              string
		in                editorResolveInputs
		wantOptions       uiserver.EditorLinkOptions
		wantErrSubstrs    []string
		wantDiscoverCalls int
	}{
		{
			name: "flag only",
			in: editorResolveInputs{
				flagTemplate: vscodeTemplate,
				getenv:       fakeGetenv(nil),
			},
			wantOptions: uiserver.EditorLinkOptions{Template: vscodeTemplate, Source: uiserver.EditorTemplateFlag},
		},
		{
			name: "env only",
			in: editorResolveInputs{
				getenv: fakeGetenv(map[string]string{editorURLEnvVar: "cursor://file/{path}:{line}"}),
			},
			wantOptions: uiserver.EditorLinkOptions{Template: "cursor://file/{path}:{line}", Source: uiserver.EditorTemplateEnv},
		},
		{
			name: "flag beats env, both valid",
			in: editorResolveInputs{
				flagTemplate: vscodeTemplate,
				getenv:       fakeGetenv(map[string]string{editorURLEnvVar: "cursor://file/{path}:{line}"}),
			},
			wantOptions: uiserver.EditorLinkOptions{Template: vscodeTemplate, Source: uiserver.EditorTemplateFlag},
		},
		{
			name: "flag valid, env malformed — both validated, env error surfaces",
			in: editorResolveInputs{
				flagTemplate: vscodeTemplate,
				getenv:       fakeGetenv(map[string]string{editorURLEnvVar: "javascript:{path}"}),
			},
			wantErrSubstrs: []string{editorURLEnvVar, "javascript"},
		},
		{
			name: "flag malformed — discover never called",
			in: editorResolveInputs{
				flagTemplate: "javascript:{path}",
				getenv:       fakeGetenv(nil),
			},
			wantErrSubstrs: []string{"--editor-url", "javascript"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls int
			if tc.in.discover == nil {
				tc.in.discover = countingDiscover(&calls, discoveredEditor{}, false)
			}
			got, err := resolveEditorLink(tc.in)
			if len(tc.wantErrSubstrs) > 0 {
				if err == nil {
					t.Fatalf("resolveEditorLink(%+v) = nil error, want one containing %v", tc.in, tc.wantErrSubstrs)
				}
				for _, substr := range tc.wantErrSubstrs {
					if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(substr)) {
						t.Fatalf("resolveEditorLink error = %q, want it to contain %q", err.Error(), substr)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveEditorLink(%+v): %v", tc.in, err)
			}
			if got != tc.wantOptions {
				t.Fatalf("resolveEditorLink(%+v) = %+v, want %+v", tc.in, got, tc.wantOptions)
			}
			if calls != tc.wantDiscoverCalls {
				t.Fatalf("discover called %d times, want %d", calls, tc.wantDiscoverCalls)
			}
		})
	}

	t.Run("discovered default", func(t *testing.T) {
		var calls int
		in := editorResolveInputs{
			getenv:   fakeGetenv(nil),
			discover: countingDiscover(&calls, discoveredEditor{Launcher: "code", PresetID: "vscode", Template: vscodeTemplate}, true),
		}
		got, err := resolveEditorLink(in)
		if err != nil {
			t.Fatalf("resolveEditorLink: %v", err)
		}
		want := uiserver.EditorLinkOptions{Template: vscodeTemplate, Source: uiserver.EditorTemplateDiscovered, Editor: "code"}
		if got != want {
			t.Fatalf("resolveEditorLink = %+v, want %+v", got, want)
		}
		if calls != 1 {
			t.Fatalf("discover called %d times, want 1", calls)
		}
	})

	t.Run("nil discover func", func(t *testing.T) {
		in := editorResolveInputs{getenv: fakeGetenv(nil), discover: nil}
		got, err := resolveEditorLink(in)
		if err != nil {
			t.Fatalf("resolveEditorLink: %v", err)
		}
		want := uiserver.EditorLinkOptions{Source: uiserver.EditorTemplateNone}
		if got != want {
			t.Fatalf("resolveEditorLink = %+v, want %+v", got, want)
		}
	})

	t.Run("discover finds nothing", func(t *testing.T) {
		var calls int
		in := editorResolveInputs{
			getenv:   fakeGetenv(nil),
			discover: countingDiscover(&calls, discoveredEditor{}, false),
		}
		got, err := resolveEditorLink(in)
		if err != nil {
			t.Fatalf("resolveEditorLink: %v", err)
		}
		want := uiserver.EditorLinkOptions{Source: uiserver.EditorTemplateNone}
		if got != want {
			t.Fatalf("resolveEditorLink = %+v, want %+v", got, want)
		}
		if calls != 1 {
			t.Fatalf("discover called %d times, want 1", calls)
		}
	})
}

// TestResolveEditorLinkFailsFastOnMalformedExplicitValue proves D-17:
// a malformed explicit value (flag or env) is refused before discovery
// ever runs, naming the source and the cause.
func TestResolveEditorLinkFailsFastOnMalformedExplicitValue(t *testing.T) {
	t.Run("malformed flag", func(t *testing.T) {
		var calls int
		in := editorResolveInputs{
			flagTemplate: "javascript:{path}",
			getenv:       fakeGetenv(nil),
			discover:     countingDiscover(&calls, discoveredEditor{}, false),
		}
		_, err := resolveEditorLink(in)
		if err == nil {
			t.Fatal("resolveEditorLink = nil error, want a refusal")
		}
		if !strings.Contains(err.Error(), "--editor-url") {
			t.Fatalf("error = %q, want it to name --editor-url", err.Error())
		}
		if calls != 0 {
			t.Fatalf("discover called %d times, want 0 (never reached)", calls)
		}
	})

	t.Run("malformed env", func(t *testing.T) {
		var calls int
		in := editorResolveInputs{
			getenv:   fakeGetenv(map[string]string{editorURLEnvVar: "data:{path}"}),
			discover: countingDiscover(&calls, discoveredEditor{}, false),
		}
		_, err := resolveEditorLink(in)
		if err == nil {
			t.Fatal("resolveEditorLink = nil error, want a refusal")
		}
		if !strings.Contains(err.Error(), editorURLEnvVar) {
			t.Fatalf("error = %q, want it to name %s", err.Error(), editorURLEnvVar)
		}
		if calls != 0 {
			t.Fatalf("discover called %d times, want 0 (never reached)", calls)
		}
	})
}

// TestResolveEditorLinkOffSwitch proves D-16: --no-editor-url and
// CODEGRAPH_NO_EDITOR_URL (case-insensitive true|1|yes) disable editor
// links and skip discovery; an unparseable value is an error naming the
// variable and the value; a malformed flag beats the off switch (D-17's
// explicit-values-first ordering).
func TestResolveEditorLinkOffSwitch(t *testing.T) {
	t.Run("flag disables, discover skipped", func(t *testing.T) {
		var calls int
		in := editorResolveInputs{
			noEditorURL: true,
			getenv:      fakeGetenv(nil),
			discover:    countingDiscover(&calls, discoveredEditor{Launcher: "code", Template: vscodeTemplate}, true),
		}
		got, err := resolveEditorLink(in)
		if err != nil {
			t.Fatalf("resolveEditorLink: %v", err)
		}
		want := uiserver.EditorLinkOptions{Source: uiserver.EditorTemplateDisabled}
		if got != want {
			t.Fatalf("resolveEditorLink = %+v, want %+v", got, want)
		}
		if calls != 0 {
			t.Fatalf("discover called %d times, want 0", calls)
		}
	})

	for _, v := range []string{"true", "1", "yes", "TRUE", "Yes"} {
		t.Run("env disables: "+v, func(t *testing.T) {
			var calls int
			in := editorResolveInputs{
				getenv:   fakeGetenv(map[string]string{noEditorURLEnvVar: v}),
				discover: countingDiscover(&calls, discoveredEditor{}, true),
			}
			got, err := resolveEditorLink(in)
			if err != nil {
				t.Fatalf("resolveEditorLink(%q): %v", v, err)
			}
			if got.Source != uiserver.EditorTemplateDisabled {
				t.Fatalf("resolveEditorLink(%q).Source = %v, want EditorTemplateDisabled", v, got.Source)
			}
			if calls != 0 {
				t.Fatalf("discover called %d times, want 0", calls)
			}
		})
	}

	for _, v := range []string{"", "false", "0", "no"} {
		t.Run("env does not disable: "+v, func(t *testing.T) {
			in := editorResolveInputs{
				getenv: fakeGetenv(map[string]string{noEditorURLEnvVar: v}),
			}
			got, err := resolveEditorLink(in)
			if err != nil {
				t.Fatalf("resolveEditorLink(%q): %v", v, err)
			}
			if got.Source == uiserver.EditorTemplateDisabled {
				t.Fatalf("resolveEditorLink(%q).Source = Disabled, want not-disabled", v)
			}
		})
	}

	t.Run("unparseable env value errors naming variable and value", func(t *testing.T) {
		in := editorResolveInputs{
			getenv: fakeGetenv(map[string]string{noEditorURLEnvVar: "maybe"}),
		}
		_, err := resolveEditorLink(in)
		if err == nil {
			t.Fatal("resolveEditorLink = nil error, want a refusal")
		}
		if !strings.Contains(err.Error(), noEditorURLEnvVar) || !strings.Contains(err.Error(), "maybe") {
			t.Fatalf("error = %q, want it to name %q and %q", err.Error(), noEditorURLEnvVar, "maybe")
		}
	})

	t.Run("malformed flag beats the off switch", func(t *testing.T) {
		in := editorResolveInputs{
			flagTemplate: "javascript:{path}",
			noEditorURL:  true,
			getenv:       fakeGetenv(nil),
		}
		_, err := resolveEditorLink(in)
		if err == nil {
			t.Fatal("resolveEditorLink = nil error, want the malformed-value error to win")
		}
		if !strings.Contains(err.Error(), "--editor-url") {
			t.Fatalf("error = %q, want the malformed --editor-url error, not the off-switch state", err.Error())
		}
	})
}
