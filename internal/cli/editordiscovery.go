// editordiscovery.go implements D-14/D-15's startup-time editor
// discovery: the third rung of resolveEditorLink's precedence chain
// (internal/cli/editorurl.go), consulted only when neither --editor-url
// nor CODEGRAPH_EDITOR_URL configured a template.
//
// Discovery is a PROBE, never a launch (SRV-03): it only asks the
// operating system "is this on PATH?" or "does this application
// directory exist?" via exec.LookPath and os.Stat — it never spawns a
// process. discoverEditorWith runs the probe order once, at
// codegraph ui's startup, before the server binds a port; it is never
// re-run per request. Finding nothing is a normal outcome, not a
// failure: the server still starts and GetEditorLink answers
// NO_TEMPLATE, exactly as it does when no source configured a template
// at all (D-15).
//
// The probe order (editorLaunchers below) is a committed fact: VS Code,
// then Cursor, then the JetBrains family in a fixed order. Changing it
// is a deliberate, tested decision, not an incidental refactor.
//
// Native Windows app-directory probing is deliberately absent: native
// Windows support was dropped at v0.4.0 (WSL2 only, per
// quick task 260807-gho) — there is no host left for such a probe to
// run on. Any platform other than macOS or Linux falls back to PATH
// probing only.
package cli

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/seanb4t/codegraph-go/internal/uiserver"
)

// editorLaunchers is the committed, ordered probe list (D-15): VS Code,
// then Cursor, then the JetBrains family. discoverEditorWith stops at
// the first hit, so this order IS the popularity/precedence ranking —
// changing it, or the set of launchers named, is a tested, deliberate
// decision (TestDiscoverEditorOrderIsCommitted pins the exact literal).
var editorLaunchers = []string{
	"code", "cursor",
	"idea", "goland", "webstorm", "pycharm", "rider", "clion", "phpstorm", "rubymine",
}

// jetbrainsLaunchers is the set of editorLaunchers entries that are
// JetBrains IDEs — every entry after code and cursor. Membership drives
// both templateForLauncher's IDE-side URL form and which launchers get
// probed under a JetBrains Toolbox app directory on Linux.
var jetbrainsLaunchers = func() map[string]struct{} {
	set := make(map[string]struct{}, len(editorLaunchers)-2)
	for _, l := range editorLaunchers[2:] {
		set[l] = struct{}{}
	}
	return set
}()

// macAppBundles maps each launcher to the macOS .app bundle name(s) that
// indicate it is installed, when it is not found on PATH. Each name is
// probed under /Applications and then under $HOME/Applications.
var macAppBundles = map[string][]string{
	"code":      {"Visual Studio Code.app"},
	"cursor":    {"Cursor.app"},
	"idea":      {"IntelliJ IDEA.app", "IntelliJ IDEA CE.app"},
	"goland":    {"GoLand.app"},
	"webstorm":  {"WebStorm.app"},
	"pycharm":   {"PyCharm.app", "PyCharm CE.app"},
	"rider":     {"Rider.app"},
	"clion":     {"CLion.app"},
	"phpstorm":  {"PhpStorm.app"},
	"rubymine":  {"RubyMine.app"},
}

// linuxAppDirs returns the Linux application-directory candidates for a
// JetBrains launcher: a package-manager install under /opt/<launcher>,
// and a JetBrains Toolbox install under
// <XDG_DATA_HOME or ~/.local/share>/JetBrains/Toolbox/apps/<launcher>.
// code and cursor are not probed here on Linux — their package
// installers place a launcher on PATH, which the lookPath probe already
// covers.
func linuxAppDirs(launcher string, getenv func(string) string) []string {
	dataHome := getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(getenv("HOME"), ".local/share")
	}
	return []string{
		filepath.Join("/opt", launcher),
		filepath.Join(dataHome, "JetBrains", "Toolbox", "apps", launcher),
	}
}

// editorProbes is discoverEditorWith's fully injected environment: every
// filesystem/PATH touch and the goos it runs against, so ordering and
// the never-executes property are unit-tested against fakes rather than
// the real host.
type editorProbes struct {
	lookPath func(string) (string, error)
	stat     func(string) (fs.FileInfo, error)
	goos     string
	getenv   func(string) string
}

// appDirCandidates returns the application-directory paths to stat for
// launcher on the given goos, or nil when that platform has no
// application-directory probe for this launcher (Linux's code/cursor,
// and every launcher on any platform other than macOS or Linux).
func appDirCandidates(launcher, goos string, getenv func(string) string) []string {
	switch goos {
	case "darwin":
		bundles := macAppBundles[launcher]
		dirs := make([]string, 0, len(bundles)*2)
		home := getenv("HOME")
		for _, bundle := range bundles {
			dirs = append(dirs, filepath.Join("/Applications", bundle))
			dirs = append(dirs, filepath.Join(home, "Applications", bundle))
		}
		return dirs
	case "linux":
		if _, ok := jetbrainsLaunchers[launcher]; ok {
			return linuxAppDirs(launcher, getenv)
		}
		return nil
	default:
		return nil
	}
}

// templateForLauncher composes the (presetID, template) pair a
// discovered launcher maps to (D-15). "code" and "cursor" read their
// template from uiserver.EditorPresets() — the single source those two
// presets already have — rather than restating either string here.
// Every JetBrains launcher maps to that IDE's own IDE-side URL form,
// matching editorpresets.go's [ASSUMED] JetBrains template shape. An
// unknown launcher (one not in editorLaunchers) returns ("", "").
func templateForLauncher(launcher string) (presetID, template string) {
	// RED phase stub (plan 09-02 Task 1): declared with its final
	// signature so editordiscovery_test.go compiles, but not yet wired
	// to editorLaunchers/presetTemplate/jetbrainsLaunchers — every
	// caller sees a not-found answer until the GREEN commit.
	return "", ""
}

// presetTemplate looks up id's template from uiserver.EditorPresets(),
// the shared preset source templateForLauncher never duplicates.
func presetTemplate(id string) (string, string) {
	for _, p := range uiserver.EditorPresets() {
		if p.ID == id {
			return p.ID, p.Template
		}
	}
	return "", ""
}

// discoverEditorWith runs the committed probe order (editorLaunchers)
// against p: for each launcher, p.lookPath is tried FIRST, and only on
// failure are that launcher's application-directory candidates stat'ed
// in order. The search stops at the first hit. A lookPath or stat error
// of any kind (not found, permission denied, anything else) is treated
// uniformly as "not here" — this function has no error return, and
// finding nothing is reported as (discoveredEditor{}, false), never a
// panic or an error.
func discoverEditorWith(p editorProbes) (discoveredEditor, bool) {
	// RED phase stub (plan 09-02 Task 1): declared with its final
	// signature so editordiscovery_test.go compiles, but the probe loop
	// (PATH then app directories, in editorLaunchers order) is not yet
	// implemented — every call reports not-found until the GREEN
	// commit.
	return discoveredEditor{}, false
}

// discoverEditor is discoverEditorWith's production binding: the real
// PATH lookup, the real filesystem stat, the real host platform and the
// real environment.
func discoverEditor() (discoveredEditor, bool) {
	return discoverEditorWith(editorProbes{
		lookPath: exec.LookPath,
		stat:     os.Stat,
		goos:     runtime.GOOS,
		getenv:   os.Getenv,
	})
}
