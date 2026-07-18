package present

import "testing"

func TestSanitizeControl(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"clean ascii unchanged", "main.go", "main.go"},
		{"clean unicode unchanged", "café/münchen.txt", "café/münchen.txt"},
		{"empty", "", ""},
		{"strips ESC (ANSI SGR injection)", "a\x1b[31mred\x1b[0m", "a[31mred[0m"},
		{"strips OSC introducer, keeps printable payload", "file\x1b]0;pwned\x07.go", "file]0;pwned.go"},
		{"strips embedded newline", "line1\nline2", "line1line2"},
		{"strips tab and CR", "a\tb\rc", "abc"},
		{"strips DEL", "a\x7fbc", "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeControl(tt.in); got != tt.want {
				t.Errorf("sanitizeControl(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestSanitizeControl_CleanStringIdentity guards the fast path: a string with
// no control characters must be returned unchanged so normal filenames are
// never mutated (protects the RenderFiles output for ordinary trees).
func TestSanitizeControl_CleanStringIdentity(t *testing.T) {
	const clean = "internal/cli/present/files.go"
	if got := sanitizeControl(clean); got != clean {
		t.Fatalf("clean string was altered: got %q, want %q", got, clean)
	}
}
