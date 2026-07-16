// stdout_detection_selftest_test.go proves the three stdout-detection
// predicates (isOSStdoutRef, isBareFmtPrint, isLogSetOutput) defined in
// stdout_confinement_test.go actually resolve identifiers via go/types
// rather than merely appearing plausible: a synthetic positive fixture
// exercising every flagged construct must be fully flagged, and a
// synthetic negative fixture exercising every documented non-match case
// (os.Stderr, an explicit-writer fmt.Fprintf, log.Printf, and a locally
// shadowed identifier named os) must be flagged zero times. This is the
// mutation-proof-at-the-detection-layer discipline (D-08 applied to
// HYG-02, RESEARCH Pitfall 4) — proving the predicates CAN fail, not
// just that they currently pass.
package archtest

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

// stdoutGuardPositiveFixture exercises exactly one instance of each
// flagged construct: a real os.Stdout reference, a bare fmt.Println
// call, and a log.SetOutput call (its argument is os.Stderr, not
// os.Stdout, so the fixture's stdout-reference count stays exactly 1).
const stdoutGuardPositiveFixture = `package fixture

import (
	"fmt"
	"log"
	"os"
)

func Bad() {
	_ = os.Stdout
	fmt.Println("noisy")
	log.SetOutput(os.Stderr)
}
`

// stdoutGuardNegativeFixture exercises every documented non-match case:
// os.Stderr (not Stdout), fmt.Fprintf with an explicit writer, log.Printf
// (compliant stdlib log usage), and a locally shadowed identifier named
// os whose Stdout is a struct field, not the os package's var.
const stdoutGuardNegativeFixture = `package fixture

import (
	"fmt"
	"log"
	"os"
)

type shadowed struct {
	Stdout int
}

func Good(w *os.File) {
	_ = os.Stderr
	fmt.Fprintf(w, "quiet")
	log.Printf("quiet")

	os := shadowed{Stdout: 1}
	_ = os.Stdout
}
`

// stdoutGuardCounts tallies how many times each predicate fired while
// walking a single parsed and type-checked fixture file.
type stdoutGuardCounts struct {
	stdoutRefs int
	barePrints int
	setOutputs int
}

// scanStdoutGuardFixture parses and type-checks src (a complete,
// self-contained Go source file) and walks its AST with the same
// predicates TestNoStdoutNoiseInServeReachablePackages uses, tallying how
// many times each one fires.
func scanStdoutGuardFixture(t *testing.T, name, src string) stdoutGuardCounts {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name+".go", src, 0)
	if err != nil {
		t.Fatalf("%s: parser.ParseFile: %v", name, err)
	}

	info := &types.Info{
		Uses: make(map[*ast.Ident]types.Object),
	}
	conf := types.Config{Importer: importer.Default()}
	if _, err := conf.Check("fixture", fset, []*ast.File{file}, info); err != nil {
		t.Fatalf("%s: types.Check: %v", name, err)
	}

	var counts stdoutGuardCounts
	ast.Inspect(file, func(n ast.Node) bool {
		switch expr := n.(type) {
		case *ast.SelectorExpr:
			if isOSStdoutRef(expr, info) {
				counts.stdoutRefs++
			}
		case *ast.CallExpr:
			if isBareFmtPrint(expr, info) {
				counts.barePrints++
			}
			if isLogSetOutput(expr, info) {
				counts.setOutputs++
			}
		}
		return true
	})
	return counts
}

// TestStdoutGuardDetectsViolations is the self-test mandated by
// CONTEXT.md D-06(b) / RESEARCH Pitfall 4: it proves the detection
// predicates driving TestNoStdoutNoiseInServeReachablePackages can
// actually fail, by flagging a synthetic positive fixture's os.Stdout
// reference, bare fmt.Println call, and log.SetOutput call exactly once
// each — and by NOT flagging any construct in a synthetic negative
// fixture built from every documented non-match case.
func TestStdoutGuardDetectsViolations(t *testing.T) {
	pos := scanStdoutGuardFixture(t, "positive", stdoutGuardPositiveFixture)
	if pos.stdoutRefs != 1 {
		t.Errorf("positive fixture: want 1 os.Stdout reference flagged, got %d", pos.stdoutRefs)
	}
	if pos.barePrints != 1 {
		t.Errorf("positive fixture: want 1 bare fmt.Print* flagged, got %d", pos.barePrints)
	}
	if pos.setOutputs != 1 {
		t.Errorf("positive fixture: want 1 log.SetOutput flagged, got %d", pos.setOutputs)
	}

	neg := scanStdoutGuardFixture(t, "negative", stdoutGuardNegativeFixture)
	if neg.stdoutRefs != 0 {
		t.Errorf("negative fixture: want 0 os.Stdout references flagged, got %d", neg.stdoutRefs)
	}
	if neg.barePrints != 0 {
		t.Errorf("negative fixture: want 0 bare fmt.Print* flagged, got %d", neg.barePrints)
	}
	if neg.setOutputs != 0 {
		t.Errorf("negative fixture: want 0 log.SetOutput flagged, got %d", neg.setOutputs)
	}
}
