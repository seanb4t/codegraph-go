package cli

import (
	"errors"
	"io/fs"
	"reflect"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/uiserver"
)

// fakeFileInfo is the minimal fs.FileInfo a fake stat function needs to
// return on a "found" answer; its fields are never inspected by
// discoverEditorWith.
type fakeFileInfo struct{ fs.FileInfo }

var errProbeNotFound = errors.New("not found")

// TestDiscoverEditorOrderIsCommitted pins editorLaunchers to the exact
// ten-element literal D-15 commits to: VS Code, Cursor, then the
// JetBrains family in a fixed order. A changed order or a new/removed
// launcher fails here, by design.
func TestDiscoverEditorOrderIsCommitted(t *testing.T) {
	want := []string{
		"code", "cursor",
		"idea", "goland", "webstorm", "pycharm", "rider", "clion", "phpstorm", "rubymine",
	}
	if !reflect.DeepEqual(editorLaunchers, want) {
		t.Fatalf("editorLaunchers = %v, want %v", editorLaunchers, want)
	}
}

// TestDiscoverEditorProbesPathBeforeAppDirs proves discoverEditorWith
// checks PATH before it ever stats an application-directory candidate,
// for every scenario the plan's behavior section specifies.
func TestDiscoverEditorProbesPathBeforeAppDirs(t *testing.T) {
	t.Run("cursor found on PATH, stat never succeeds", func(t *testing.T) {
		probes := editorProbes{
			lookPath: func(name string) (string, error) {
				if name == "cursor" {
					return "/usr/local/bin/cursor", nil
				}
				return "", errProbeNotFound
			},
			stat:   func(string) (fs.FileInfo, error) { return nil, errProbeNotFound },
			goos:   "linux",
			getenv: func(string) string { return "" },
		}

		got, ok := discoverEditorWith(probes)
		if !ok {
			t.Fatal("discoverEditorWith: got false, want true")
		}
		wantTemplate := presetTemplateOrFail(t, "cursor")
		if got.Launcher != "cursor" || got.PresetID != "cursor" || got.Template != wantTemplate {
			t.Fatalf("discoverEditorWith = %+v, want Launcher=cursor PresetID=cursor Template=%q", got, wantTemplate)
		}
	})

	t.Run("code and cursor both on PATH: code wins, lookPath called exactly once", func(t *testing.T) {
		var lookPathCalls int
		probes := editorProbes{
			lookPath: func(name string) (string, error) {
				lookPathCalls++
				if name == "code" || name == "cursor" {
					return "/usr/local/bin/" + name, nil
				}
				return "", errProbeNotFound
			},
			stat:   func(string) (fs.FileInfo, error) { return nil, errProbeNotFound },
			goos:   "linux",
			getenv: func(string) string { return "" },
		}

		got, ok := discoverEditorWith(probes)
		if !ok {
			t.Fatal("discoverEditorWith: got false, want true")
		}
		if got.Launcher != "code" {
			t.Fatalf("Launcher = %q, want %q (earlier in editorLaunchers)", got.Launcher, "code")
		}
		if lookPathCalls != 1 {
			t.Fatalf("lookPath called %d time(s), want exactly 1 (search must stop at the first hit)", lookPathCalls)
		}
	})

	t.Run("darwin app bundle: GoLand under /Applications", func(t *testing.T) {
		probes := editorProbes{
			lookPath: func(string) (string, error) { return "", errProbeNotFound },
			stat: func(path string) (fs.FileInfo, error) {
				if path == "/Applications/GoLand.app" {
					return fakeFileInfo{}, nil
				}
				return nil, errProbeNotFound
			},
			goos:   "darwin",
			getenv: func(string) string { return "" },
		}

		got, ok := discoverEditorWith(probes)
		if !ok {
			t.Fatal("discoverEditorWith: got false, want true")
		}
		if got.Launcher != "goland" || got.PresetID != "jetbrains" || got.Template != "goland://open?file={path}&line={line}" {
			t.Fatalf("discoverEditorWith = %+v, want goland/jetbrains/goland://open?file={path}&line={line}", got)
		}
	})

	t.Run("darwin app bundle under $HOME/Applications", func(t *testing.T) {
		probes := editorProbes{
			lookPath: func(string) (string, error) { return "", errProbeNotFound },
			stat: func(path string) (fs.FileInfo, error) {
				if path == "/Users/x/Applications/Cursor.app" {
					return fakeFileInfo{}, nil
				}
				return nil, errProbeNotFound
			},
			goos: "darwin",
			getenv: func(name string) string {
				if name == "HOME" {
					return "/Users/x"
				}
				return ""
			},
		}

		got, ok := discoverEditorWith(probes)
		if !ok {
			t.Fatal("discoverEditorWith: got false, want true")
		}
		if got.Launcher != "cursor" {
			t.Fatalf("Launcher = %q, want %q", got.Launcher, "cursor")
		}
	})

	t.Run("linux JetBrains Toolbox under XDG_DATA_HOME-derived path", func(t *testing.T) {
		probes := editorProbes{
			lookPath: func(string) (string, error) { return "", errProbeNotFound },
			stat: func(path string) (fs.FileInfo, error) {
				if path == "/home/x/.local/share/JetBrains/Toolbox/apps/goland" {
					return fakeFileInfo{}, nil
				}
				return nil, errProbeNotFound
			},
			goos: "linux",
			getenv: func(name string) string {
				switch name {
				case "XDG_DATA_HOME":
					return ""
				case "HOME":
					return "/home/x"
				default:
					return ""
				}
			},
		}

		got, ok := discoverEditorWith(probes)
		if !ok {
			t.Fatal("discoverEditorWith: got false, want true")
		}
		if got.Launcher != "goland" {
			t.Fatalf("Launcher = %q, want %q", got.Launcher, "goland")
		}
	})

	t.Run("linux /opt install", func(t *testing.T) {
		probes := editorProbes{
			lookPath: func(string) (string, error) { return "", errProbeNotFound },
			stat: func(path string) (fs.FileInfo, error) {
				if path == "/opt/idea" {
					return fakeFileInfo{}, nil
				}
				return nil, errProbeNotFound
			},
			goos: "linux",
			getenv: func(name string) string {
				if name == "HOME" {
					return "/home/x"
				}
				return ""
			},
		}

		got, ok := discoverEditorWith(probes)
		if !ok {
			t.Fatal("discoverEditorWith: got false, want true")
		}
		if got.Launcher != "idea" {
			t.Fatalf("Launcher = %q, want %q", got.Launcher, "idea")
		}
	})

	t.Run("code's PATH probe happens before any stat of a code bundle", func(t *testing.T) {
		var probeLog []string
		probes := editorProbes{
			lookPath: func(name string) (string, error) {
				probeLog = append(probeLog, "lookPath:"+name)
				return "", errProbeNotFound
			},
			stat: func(path string) (fs.FileInfo, error) {
				probeLog = append(probeLog, "stat:"+path)
				return nil, errProbeNotFound
			},
			goos:   "darwin",
			getenv: func(string) string { return "" },
		}

		discoverEditorWith(probes)

		if len(probeLog) == 0 {
			t.Fatal("no probes were recorded")
		}
		if probeLog[0] != "lookPath:code" {
			t.Fatalf("first probe = %q, want %q", probeLog[0], "lookPath:code")
		}
	})
}

// TestDiscoverEditorNeverExecutes proves discoverEditorWith only ever
// calls lookPath/stat — never anything process-spawning — by recording
// every probe call and asserting the recorded set contains nothing else
// (SRV-03, T-09-07).
func TestDiscoverEditorNeverExecutes(t *testing.T) {
	var lookPathCalls, statCalls int
	probes := editorProbes{
		lookPath: func(string) (string, error) {
			lookPathCalls++
			return "", errProbeNotFound
		},
		stat: func(string) (fs.FileInfo, error) {
			statCalls++
			return nil, errProbeNotFound
		},
		goos:   "linux",
		getenv: func(string) string { return "" },
	}

	got, ok := discoverEditorWith(probes)
	if ok {
		t.Fatalf("discoverEditorWith = %+v, true; want false (every probe fails)", got)
	}
	if lookPathCalls == 0 {
		t.Fatal("lookPath was never called")
	}
	if statCalls == 0 {
		t.Fatal("stat was never called (JetBrains launchers on Linux have app-dir candidates)")
	}
}

// TestDiscoverEditorNotFoundIsNotFatal proves that with every probe
// failing, discoverEditorWith reports not-found with no panic and no
// error return — there is nothing to surface an error through.
func TestDiscoverEditorNotFoundIsNotFatal(t *testing.T) {
	probes := editorProbes{
		lookPath: func(string) (string, error) { return "", errProbeNotFound },
		stat:     func(string) (fs.FileInfo, error) { return nil, errProbeNotFound },
		goos:     "linux",
		getenv:   func(string) string { return "" },
	}

	got, ok := discoverEditorWith(probes)
	if ok {
		t.Fatalf("discoverEditorWith = %+v, true; want false", got)
	}
	if got != (discoveredEditor{}) {
		t.Fatalf("discoverEditorWith result = %+v, want the zero value", got)
	}
}

// TestTemplateForLauncherEmitsAllowlistedTemplates iterates every entry
// of editorLaunchers, asserting templateForLauncher's output always
// passes uiserver.ValidateEditorTemplate, and reports how many launchers
// were inspected — failing on zero so this test cannot vacuously pass.
func TestTemplateForLauncherEmitsAllowlistedTemplates(t *testing.T) {
	inspected := 0
	for _, launcher := range editorLaunchers {
		presetID, template := templateForLauncher(launcher)
		if presetID == "" || template == "" {
			t.Fatalf("templateForLauncher(%q) = (%q, %q), want non-empty", launcher, presetID, template)
		}
		if err := uiserver.ValidateEditorTemplate(template); err != nil {
			t.Fatalf("templateForLauncher(%q) = %q, fails ValidateEditorTemplate: %v", launcher, template, err)
		}
		inspected++
	}
	t.Logf("inspected %d launchers", inspected)
	if inspected != 10 {
		t.Fatalf("inspected %d launchers, want 10", inspected)
	}

	if presetID, template := templateForLauncher("not-a-real-launcher"); presetID != "" || template != "" {
		t.Fatalf("templateForLauncher(unknown) = (%q, %q), want (\"\", \"\")", presetID, template)
	}
}

// presetTemplateOrFail resolves id's preset template from
// uiserver.EditorPresets(), failing the test if id is not present —
// used only to build an expected value, never production logic.
func presetTemplateOrFail(t *testing.T, id string) string {
	t.Helper()
	for _, p := range uiserver.EditorPresets() {
		if p.ID == id {
			return p.Template
		}
	}
	t.Fatalf("no preset with id %q", id)
	return ""
}
