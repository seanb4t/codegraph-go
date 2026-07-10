// Command spike is the Plan 01-07 parser-decision spike harness (D-05). It
// is build/test-time tooling only, never shipped in the production binary.
//
// It has two jobs:
//
//  1. "static-build": attempt a CGO_ENABLED=0 build of both parser arms
//     (internal/parser/cgo, internal/parser/wazero) and, if `zig` is on
//     PATH, attempt a cross-compile of the CGo arm to a non-native target —
//     the static-build-impact dimension of D-05 (RESEARCH Pitfall 3).
//  2. "crash-case <backend> <lang> <case>": run exactly one Parse call
//     against deliberately malformed/adversarial input and print whether it
//     returned an error or succeeded. This subcommand is invoked as a
//     *subprocess* by tools/spike/crashisolation_test.go specifically so
//     that an uncatchable CGo-side crash (a C segfault, RESEARCH Pitfall 5)
//     kills this child process — observable from outside via its exit
//     code/signal — rather than taking down the test binary itself.
package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/parser"
	cgoparser "github.com/seanb4t/codegraph-go/internal/parser/cgo"
	wazeroparser "github.com/seanb4t/codegraph-go/internal/parser/wazero"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: spike static-build | spike crash-case <backend> <lang> <case>")
		os.Exit(2)
	}

	switch os.Args[1] {
	case "static-build":
		runStaticBuildCheck()
	case "crash-case":
		if len(os.Args) != 5 {
			fmt.Fprintln(os.Stderr, "usage: spike crash-case <cgo|wazero> <go|python> <case>")
			os.Exit(2)
		}
		execCrashCase(os.Args[2], os.Args[3], os.Args[4])
	default:
		fmt.Fprintf(os.Stderr, "unknown mode %q\n", os.Args[1])
		os.Exit(2)
	}
}

// StaticBuildResult is the machine-readable outcome of the static-build-impact
// measurement, folded into PARSER-DECISION.md by Task 3.
type StaticBuildResult struct {
	CGoStaticBuildOK    bool   `json:"cgo_static_build_ok"`
	CGoStaticBuildError string `json:"cgo_static_build_error,omitempty"`
	WazeroStaticBuildOK bool   `json:"wazero_static_build_ok"`
	WazeroBuildError    string `json:"wazero_build_error,omitempty"`
	ZigAvailable        bool   `json:"zig_available"`
	CrossCompileOK      bool   `json:"cross_compile_ok,omitempty"`
	CrossCompileError   string `json:"cross_compile_error,omitempty"`
	CrossCompileNote    string `json:"cross_compile_note,omitempty"`
}

func runStaticBuildCheck() {
	result := StaticBuildResult{}

	// The CGo arm is EXPECTED to fail under CGO_ENABLED=0 — that failure
	// itself is the measurement (RESEARCH Pitfall 3: "single static binary
	// for users" != "no CGo toolchain needed anywhere").
	if out, err := runEnvCmd(map[string]string{"CGO_ENABLED": "0"}, "go", "build", "-o", os.DevNull, "./internal/parser/cgo/..."); err != nil {
		result.CGoStaticBuildOK = false
		result.CGoStaticBuildError = strings.TrimSpace(out)
	} else {
		result.CGoStaticBuildOK = true
	}

	// The wazero arm is expected to build cleanly with CGO_ENABLED=0 — it is
	// pure Go end-to-end (the WASM binary is embedded data, not a C
	// dependency of the host build).
	if out, err := runEnvCmd(map[string]string{"CGO_ENABLED": "0"}, "go", "build", "-o", os.DevNull, "./internal/parser/wazero/..."); err != nil {
		result.WazeroStaticBuildOK = false
		result.WazeroBuildError = strings.TrimSpace(out)
	} else {
		result.WazeroStaticBuildOK = true
	}

	if zigPath, zigErr := exec.LookPath("zig"); zigErr == nil {
		result.ZigAvailable = true
		env := map[string]string{
			"CGO_ENABLED": "1",
			"GOOS":        "linux",
			"GOARCH":      "arm64",
			"CC":          zigPath + " cc -target aarch64-linux-gnu",
			"CXX":         zigPath + " c++ -target aarch64-linux-gnu",
		}
		if out, err := runEnvCmd(env, "go", "build", "-o", os.DevNull, "./internal/parser/cgo/..."); err != nil {
			result.CrossCompileOK = false
			result.CrossCompileError = strings.TrimSpace(out)
		} else {
			result.CrossCompileOK = true
		}
	} else {
		result.ZigAvailable = false
		result.CrossCompileNote = "zig not found on PATH in this spike environment (RESEARCH Environment Availability); cross-compile was NOT executed. The documented approach (per goreleaser/example-zig-cgo, RESEARCH Don't Hand-Roll) is to set CC/CXX to `zig cc -target <triple>` / `zig c++ -target <triple>` with CGO_ENABLED=1 and GOOS/GOARCH set to the cross target; full validation deferred to Phase 8 per RESEARCH."
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(result)
}

func runEnvCmd(env map[string]string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// execCrashCase runs exactly one Parse call for the given backend/language
// against one of the malformed-input cases defined in cases.go, then prints
// a result line and exits 0. If the backend crashes the process outright
// (the CGo arm's uncatchable C-level segfault path), this process simply
// never reaches the print statement — the crash IS the observation, made
// visible to the parent test via this process's abnormal exit/signal.
func execCrashCase(backend, lang, caseName string) {
	source, ok := malformedCase(caseName)
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown case %q\n", caseName)
		os.Exit(2)
	}

	p, err := newParser(backend, lang)
	if err != nil {
		fmt.Printf("SETUP_ERROR: %v\n", err)
		os.Exit(1)
	}
	defer p.Close()

	tree, err := p.Parse(source, nil)
	if err != nil {
		fmt.Printf("PARSE_ERROR: %v\n", err)
		os.Exit(0)
	}
	if tree == nil {
		fmt.Println("PARSE_NIL_TREE")
		os.Exit(0)
	}
	fmt.Println("PARSE_OK")
	os.Exit(0)
}

func newParser(backend, lang string) (parser.Parser, error) {
	switch backend + "/" + lang {
	case "cgo/go":
		return cgoparser.NewGoParser()
	case "cgo/python":
		return cgoparser.NewPythonParser()
	case "wazero/go":
		return wazeroparser.NewGoParser()
	case "wazero/python":
		return wazeroparser.NewPythonParser()
	default:
		return nil, fmt.Errorf("unknown backend/lang combination %q/%q", backend, lang)
	}
}

// randomBytes returns n cryptographically random bytes, used to build the
// "garbage" adversarial case deterministically-sized but content-random.
func randomBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}
