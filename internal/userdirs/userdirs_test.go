package userdirs

import (
	"os"
	"path/filepath"
	"testing"
)

// xdgCase is one row of the shared table: the value the XDG variable
// holds (unset when set is false) and the directory the resolver must
// return, as a function of the fake home and of the variable's value.
type xdgCase struct {
	name string
	set  bool
	// value is the variable's value; "ABS" stands for an absolute temp
	// directory chosen per test, since the table cannot know it in advance.
	value string
	// wantUnderHome is the expected path relative to the fake home, or ""
	// when the variable's absolute value must win.
	wantUnderHome string
}

func xdgCases(defaultUnderHome string) []xdgCase {
	return []xdgCase{
		{name: "unset", set: false, wantUnderHome: defaultUnderHome},
		{name: "empty", set: true, value: "", wantUnderHome: defaultUnderHome},
		{name: "absolute", set: true, value: "ABS"},
		{name: "relative is ignored", set: true, value: "state", wantUnderHome: defaultUnderHome},
	}
}

func runXDGTable(t *testing.T, envKey, defaultUnderHome string, fn func() (string, error)) {
	t.Helper()
	for _, tc := range xdgCases(defaultUnderHome) {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)

			want := filepath.Join(home, tc.wantUnderHome, appDirName)
			if tc.set {
				value := tc.value
				if value == "ABS" {
					value = t.TempDir()
					want = filepath.Join(value, appDirName)
				}
				t.Setenv(envKey, value)
			} else {
				// t.Setenv registers the restore of the developer's own
				// value; os.Unsetenv then makes the row a true unset
				// rather than an empty string.
				t.Setenv(envKey, "")
				if err := os.Unsetenv(envKey); err != nil {
					t.Fatalf("unsetenv %s: %v", envKey, err)
				}
			}

			got, err := fn()
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			if got != want {
				t.Fatalf("resolve with %s=%q: got %q, want %q", envKey, tc.value, got, want)
			}
		})
	}
}

func TestStateDir(t *testing.T) {
	runXDGTable(t, "XDG_STATE_HOME", filepath.Join(".local", "state"), StateDir)
}

func TestConfigDir(t *testing.T) {
	runXDGTable(t, "XDG_CONFIG_HOME", ".config", ConfigDir)
}
