package wireoracle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// binPath is the absolute path to the real codegraph binary TestMain
// builds once per `go test` run — mirroring
// test/integration/main_test.go's TestMain exactly (VRFY-04: the oracle
// must run against the real, unmodified binary).
var binPath string

// TestMain builds the real binary hermetically, once, into a package-level
// temp dir before any test in this package runs. A build failure is a hard
// stop: it prints the build output and os.Exit(1) rather than letting
// every test silently skip or fail with a confusing "file not found".
func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "codegraph-wireoracle-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "wireoracle: TestMain: MkdirTemp: %v\n", err)
		os.Exit(1)
	}

	binPath = filepath.Join(tmpDir, "codegraph")
	buildCmd := exec.Command("go", "build", "-o", binPath, "github.com/seanb4t/codegraph-go/cmd/codegraph")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "wireoracle: TestMain: go build github.com/seanb4t/codegraph-go/cmd/codegraph failed: %v\n%s\n", err, out)
		_ = os.RemoveAll(tmpDir)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}
