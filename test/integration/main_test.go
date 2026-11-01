// Package integration is TEST-04's subprocess integration harness (D-17).
// It is a normal Go package, not testdata/, so `go test ./...` reaches it.
//
// Each behavior test is a testscript script under testdata/. Every script
// drives codegraph as a separate process, with its real argv, its real
// process cwd and real stdio JSON-RPC: the cwd/argv->path-derivation->handler
// seam that in-process BuildServer->CallTool tests bypass (D-19). Phase 2's
// CR-01 regression lived in that seam while every in-process test passed.
package integration

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"

	"github.com/seanb4t/codegraph-go/internal/cli"
	"github.com/seanb4t/codegraph-go/internal/gitmeta"
)

// testBinEnvVar names an external codegraph binary for the scripts to run
// in place of this test binary. verify:notarized-suite sets it to the
// downloaded, notarized release binary, so the suite tests the published
// artifact.
const testBinEnvVar = "CODEGRAPH_TEST_BIN"

// minScripts is the number of scripts in testdata/. TestScripts fails if
// the directory holds fewer, so a lost or renamed script cannot pass by
// running nothing.
const minScripts = 11

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"codegraph": runCodegraph,
		"within":    runWithin,
		"mcpclient": runMCPClient,
	})
}

// runCodegraph is the scripts' codegraph command. With testBinEnvVar set
// it replaces this process with that file, so the external binary gets the
// argv, cwd, stdio and environment unchanged and its exit status is the
// command's. Otherwise it runs the CLI in this process, as
// cmd/codegraph/main.go does.
func runCodegraph() {
	if bin := os.Getenv(testBinEnvVar); bin != "" {
		err := syscall.Exec(bin, append([]string{bin}, os.Args[1:]...), os.Environ())
		fmt.Fprintf(os.Stderr, "codegraph: exec %s=%s: %v\n", testBinEnvVar, bin, err)
		os.Exit(1)
	}
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func TestScripts(t *testing.T) {
	scripts, err := filepath.Glob(filepath.Join("testdata", "*.txtar"))
	if err != nil {
		t.Fatal(err)
	}
	if len(scripts) < minScripts {
		t.Fatalf("testdata holds %d scripts, want at least %d", len(scripts), minScripts)
	}

	bin := os.Getenv(testBinEnvVar)
	if bin != "" {
		// The scripts run in their own work directories, so a relative
		// path must be resolved here.
		if bin, err = filepath.Abs(bin); err != nil {
			t.Fatal(err)
		}
	}

	testscript.Run(t, testscript.Params{
		Dir:                 "testdata",
		RequireExplicitExec: true,
		Setup: func(env *testscript.Env) error {
			if bin != "" {
				env.Setenv(testBinEnvVar, bin)
			}
			// Every codegraph and git process gets a fake home: `serve
			// --mcp` registers its daemon under XDG_STATE_HOME and
			// `install` writes agent configs under XDG_CONFIG_HOME.
			home := filepath.Join(env.WorkDir, "home")
			for key, value := range map[string]string{
				"HOME":            home,
				"XDG_CONFIG_HOME": filepath.Join(home, ".config"),
				"XDG_STATE_HOME":  filepath.Join(home, ".local", "state"),
				"HERMES_HOME":     filepath.Join(home, ".hermes"),
			} {
				if err := os.MkdirAll(value, 0o755); err != nil {
					return err
				}
				env.Setenv(key, value)
			}
			// The D-11 notice glyph (U+26A0, no U+FE0F) comes from
			// gitmeta itself, so the scripts cannot drift from it.
			notice := (&gitmeta.Mismatch{WorktreeRoot: "/a", IndexRoot: "/b"}).Notice()
			env.Setenv("NOTICE_GLYPH", string([]rune(notice)[0]))
			return nil
		},
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"fixture": cmdFixture,
			"exits":   cmdExits,
		},
	})
}

// cmdFixture copies internal/indexer/testdata/gofixture, a Go module, into
// the named directory.
//
//	fixture dir
func cmdFixture(ts *testscript.TestScript, neg bool, args []string) {
	if neg || len(args) != 1 {
		ts.Fatalf("usage: fixture dir")
	}
	src, err := filepath.Abs(filepath.Join("..", "..", "internal", "indexer", "testdata", "gofixture"))
	ts.Check(err)
	dst := ts.MkAbs(args[0])
	ts.Check(filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	}))
}

// cmdExits runs a program as exec does and requires an exact exit status,
// which `! exec` cannot check.
//
//	exits status program [args...]
func cmdExits(ts *testscript.TestScript, neg bool, args []string) {
	if neg || len(args) < 2 {
		ts.Fatalf("usage: exits status program [args...]")
	}
	want, err := strconv.Atoi(args[0])
	ts.Check(err)
	err = ts.Exec(args[1], args[2:]...)
	got := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			ts.Fatalf("%s: %v", args[1], err)
		}
		got = exitErr.ExitCode()
	}
	if got != want {
		ts.Fatalf("%s exited with status %d, want %d", args[1], got, want)
	}
}
