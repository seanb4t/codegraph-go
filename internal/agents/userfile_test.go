package agents

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/seanb4t/codegraph-go/internal/realpath"
	"github.com/seanb4t/codegraph-go/internal/realpath/realpathtest"
)

func TestResolveUserFile_GlobalReturnsTheRealLocation(t *testing.T) {
	dir := realpathtest.TempDir(t)
	target := filepath.Join(dir, "dotfiles", "claude.json")
	link := filepath.Join(dir, "home", ".claude.json")
	realpathtest.Symlink(t, target, link)

	got, err := resolveUserFile(link, LocationGlobal)
	if err != nil || got != target {
		t.Fatalf("resolveUserFile = %q, %v; want %q", got, err, target)
	}
}

func TestResolveUserFile_LocalInsideTheProject(t *testing.T) {
	project, _ := projectAndOutside(t)
	realpathtest.Symlink(t, "AGENTS.md", "GEMINI.md")

	got, err := resolveUserFile("GEMINI.md", LocationLocal)
	if want := filepath.Join(project, "AGENTS.md"); err != nil || got != want {
		t.Fatalf("resolveUserFile = %q, %v; want %q", got, err, want)
	}
}

func TestResolveUserFile_LocalOutsideIsErrOutsideProject(t *testing.T) {
	_, outside := projectAndOutside(t)
	target := filepath.Join(outside, "zshrc")
	realpathtest.Symlink(t, target, "AGENTS.md")

	_, err := resolveUserFile("AGENTS.md", LocationLocal)
	if !errors.Is(err, errOutsideProject) {
		t.Fatalf("resolveUserFile error = %v, want errOutsideProject", err)
	}
	if !strings.Contains(err.Error(), target) {
		t.Fatalf("error %q does not name the real location %s", err, target)
	}
}

// With no working directory there is no project to compare with, so a
// local file is refused and never edited.
func TestResolveUserFile_LocalWithNoWorkingDirectoryIsRefused(t *testing.T) {
	fakeHome(t)
	gone := filepath.Join(realpathtest.TempDir(t), "gone")
	if err := os.Mkdir(gone, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(gone)
	if err := os.Remove(gone); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(realpathtest.TempDir(t), "AGENTS.md")

	if got, err := resolveUserFile(file, LocationLocal); err == nil {
		t.Fatalf("resolveUserFile = %q, want an error", got)
	}
	var result WriteResult
	_ = recordUserFile(&result, file, LocationLocal, opInstall, func(string) (FileResult, error) {
		t.Fatal("edit ran with no project root")
		return FileResult{}, nil
	})
	if len(result.Errors) != 1 {
		t.Fatalf("errors = %v, want one", result.Errors)
	}
}

func TestRecordUserFile_ResultNamesTheConfiguredPath(t *testing.T) {
	dir := realpathtest.TempDir(t)
	target := filepath.Join(dir, "real.json")
	link := filepath.Join(dir, "link.json")
	realpathtest.Symlink(t, target, link)

	var result WriteResult
	var edited string
	err := recordUserFile(&result, link, LocationGlobal, opInstall, func(real string) (FileResult, error) {
		edited = real
		return FileResult{Path: real, Action: ActionCreated}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if edited != target {
		t.Errorf("edit got %q, want the real location %q", edited, target)
	}
	if len(result.Files) != 1 || result.Files[0] != (FileResult{Path: link, Action: ActionCreated}) {
		t.Errorf("result.Files = %v, want one entry under the configured path %s", result.Files, link)
	}
}

func TestRecordUserFile_HelperErrorNamesTheRealLocation(t *testing.T) {
	dir := realpathtest.TempDir(t)
	target := filepath.Join(dir, "real.json")
	link := filepath.Join(dir, "link.json")
	realpathtest.Symlink(t, target, link)

	var result WriteResult
	_ = recordUserFile(&result, link, LocationGlobal, opInstall, func(string) (FileResult, error) {
		return FileResult{}, errors.New("does not parse")
	})
	if !errorNaming(result, link, target, "does not parse") {
		t.Fatalf("errors %v do not name both %s and %s", result.Errors, link, target)
	}
}

func TestRecordUserFile_UnlinkedLocalErrorKeepsItsText(t *testing.T) {
	projectAndOutside(t)

	var result WriteResult
	_ = recordUserFile(&result, "AGENTS.md", LocationLocal, opInstall, func(string) (FileResult, error) {
		return FileResult{}, errors.New("does not parse")
	})
	if len(result.Errors) != 1 || result.Errors[0].Error() != "AGENTS.md: does not parse" {
		t.Fatalf("errors = %v, want exactly [AGENTS.md: does not parse]", result.Errors)
	}
}

func TestRecordUserFile_ReturnsTheRecordedError(t *testing.T) {
	dir := realpathtest.TempDir(t)
	var result WriteResult
	helperErr := errors.New("write failed")
	err := recordUserFile(&result, filepath.Join(dir, "f"), LocationGlobal, opInstall, func(string) (FileResult, error) {
		return FileResult{}, helperErr
	})
	if !errors.Is(err, helperErr) {
		t.Fatalf("recordUserFile returned %v, want the helper error", err)
	}
	if len(result.Errors) != 1 || len(result.Files) != 0 {
		t.Fatalf("result = %+v, want one error and no file", result)
	}
}

func TestRecordUserFile_InstallOutsideRecordsAnError(t *testing.T) {
	_, outside := projectAndOutside(t)
	target := filepath.Join(outside, "zshrc")
	realpathtest.Symlink(t, target, "AGENTS.md")

	var result WriteResult
	called := false
	err := recordUserFile(&result, "AGENTS.md", LocationLocal, opInstall, func(string) (FileResult, error) {
		called = true
		return FileResult{Action: ActionUpdated}, nil
	})
	if called {
		t.Fatal("edit ran for a file outside the project")
	}
	if !errors.Is(err, errOutsideProject) {
		t.Fatalf("recordUserFile returned %v, want errOutsideProject", err)
	}
	if !errorNaming(result, "AGENTS.md", target, "--location global") {
		t.Fatalf("errors %v do not name the path, the real location and the ways forward", result.Errors)
	}
}

func TestRecordUserFile_UninstallOutsideRecordsKeptAndANote(t *testing.T) {
	_, outside := projectAndOutside(t)
	realpathtest.Symlink(t, filepath.Join(outside, "agents.md"), "AGENTS.md")

	var result WriteResult
	called := false
	err := recordUserFile(&result, "AGENTS.md", LocationLocal, opUninstall, func(string) (FileResult, error) {
		called = true
		return FileResult{Action: ActionRemoved}, nil
	})
	if err != nil || called || len(result.Errors) != 0 {
		t.Fatalf("err = %v, edit called = %v, errors = %v; want none of them", err, called, result.Errors)
	}
	assertFileResult(t, result.Files, "AGENTS.md", ActionKept)
	if !noteNaming(result, "AGENTS.md") {
		t.Fatalf("notes %v do not name AGENTS.md", result.Notes)
	}
}

func TestRecordUserAction_UninstallOutsideRecordsKeptAndANote(t *testing.T) {
	_, outside := projectAndOutside(t)
	realpathtest.Symlink(t, filepath.Join(outside, "agents.md"), "AGENTS.md")

	var result WriteResult
	err := recordUserAction(&result, "AGENTS.md", LocationLocal, opUninstall, func(string) (FileAction, error) {
		t.Fatal("edit ran for a file outside the project")
		return "", nil
	})
	if err != nil || len(result.Errors) != 0 {
		t.Fatalf("err = %v, errors = %v; want none", err, result.Errors)
	}
	assertFileResult(t, result.Files, "AGENTS.md", ActionKept)
	if !noteNaming(result, "AGENTS.md") {
		t.Fatalf("notes %v do not name AGENTS.md", result.Notes)
	}
}

// TestResolveUserFile_LocalNeverLeavesTheProject draws random trees of
// links across a project and an outside directory. Whatever location
// resolveUserFile returns at local scope, a file made there is where the
// operating system reaches for the configured path, and it is inside the
// project.
func TestResolveUserFile_LocalNeverLeavesTheProject(t *testing.T) {
	project, outside := projectAndOutside(t)
	inside, refused := 0, 0
	rapid.Check(t, func(rt *rapid.T) {
		in, err := os.MkdirTemp(project, "tree")
		if err != nil {
			rt.Fatal(err)
		}
		out, err := os.MkdirTemp(outside, "tree")
		if err != nil {
			rt.Fatal(err)
		}
		for _, p := range realpathtest.Draw(rt, in, out).Paths {
			got, err := resolveUserFile(p, LocationLocal)
			if errors.Is(err, errOutsideProject) {
				// A refusal is right only for a path that leads outside.
				if real, rerr := realpath.Resolve(p); rerr != nil || realpath.Within(project, real) {
					rt.Fatalf("resolveUserFile(%q) refused a path whose real location is %q, %v", p, real, rerr)
				}
				refused++
				continue
			}
			if err != nil {
				continue
			}
			if _, statErr := os.Lstat(got); statErr != nil {
				if err := os.MkdirAll(filepath.Dir(got), 0o755); err != nil {
					rt.Fatalf("resolveUserFile(%q) = %q, and its directory cannot be made: %v", p, got, err)
				}
				if err := os.WriteFile(got, []byte("x"), 0o644); err != nil {
					rt.Fatalf("resolveUserFile(%q) = %q, and no file can be written there: %v", p, got, err)
				}
			}
			reached, err := filepath.EvalSymlinks(p)
			if err != nil || reached != got {
				rt.Fatalf("resolveUserFile(%q) = %q, but the operating system reaches %q, %v", p, got, reached, err)
			}
			if !realpath.Within(project, reached) {
				rt.Fatalf("resolveUserFile(%q) = %q, which is outside the project %s", p, got, project)
			}
			inside++
		}
	})
	if inside == 0 || refused == 0 {
		t.Fatalf("drew %d inside cases and %d outside cases, want both above zero", inside, refused)
	}
}
