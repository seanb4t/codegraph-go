package cli

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

// functionVariableTree indexes copies of this repository's daemon list
// seam: internal/cli/daemon.go, which binds daemonList to daemon.List and
// calls it from the daemon command, daemon_test.go, which replaces it,
// and internal/daemon/registry.go, which declares List. extra is appended
// to the copy of daemon.go.
func functionVariableTree(t *testing.T, extra string) string {
	t.Helper()
	read := func(path string) string {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		return string(b)
	}
	return initTree(t, map[string]string{
		"go.mod":                      "module github.com/seanb4t/codegraph-go\n\ngo 1.22\n",
		"internal/cli/daemon.go":      read("daemon.go") + extra,
		"internal/cli/daemon_test.go": read("daemon_test.go"),
		"internal/daemon/registry.go": read("../daemon/registry.go"),
	})
}

// queryNames runs a query command with --json and returns the names of
// the rows of field.
func queryNames(t *testing.T, field string, args ...string) []string {
	t.Helper()
	out, _, err := execCmd(append(args, "--json")...)
	if err != nil {
		t.Fatalf("%v: %v", args, err)
	}
	var result map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON from %v: %v\n%s", args, err, out)
	}
	var rows []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(result[field], &rows); err != nil {
		t.Fatalf("invalid %s from %v: %v\n%s", field, args, err, out)
	}
	var names []string
	for _, r := range rows {
		names = append(names, r.Name)
	}
	return names
}

// TestFunctionVariableQueries covers the daemonList seam. The daemon command
// calls daemon.List through the daemonList variable, so callers, callees
// and impact show that call. The test file's replacements are a local and
// a function literal, which a production call site does not count. The
// variable's initializer calls nothing, so it is never a caller. A
// production reassignment to an unknown value withholds the call, and the
// test file stays indexed in both trees.
func TestFunctionVariableQueries(t *testing.T) {
	t.Run("production binding", func(t *testing.T) {
		dir := functionVariableTree(t, "")
		callers := queryNames(t, "callers", "callers", "List", "-p", dir)
		if !slices.Contains(callers, "newDaemonCmd") {
			t.Errorf("callers List = %v, want newDaemonCmd", callers)
		}
		if slices.Contains(callers, "daemonList") {
			t.Errorf("callers List = %v: the initializer of daemonList calls nothing", callers)
		}
		if callees := queryNames(t, "callees", "callees", "newDaemonCmd", "-p", dir); !slices.Contains(callees, "List") {
			t.Errorf("callees newDaemonCmd = %v, want List", callees)
		}
		if affected := queryNames(t, "affected", "impact", "List", "-p", dir); !slices.Contains(affected, "newDaemonCmd") {
			t.Errorf("impact List = %v, want newDaemonCmd", affected)
		}
		if got := queryNames(t, "callers", "callers", "withStubbedDaemonList", "-p", dir); len(got) == 0 {
			t.Error("callers withStubbedDaemonList is empty: daemon_test.go is not indexed")
		}
	})
	t.Run("production reassignment", func(t *testing.T) {
		dir := functionVariableTree(t, "\nfunc init() {\n\tdaemonList = func() ([]daemon.Record, []string, error) { return nil, nil, nil }\n}\n")
		if callers := queryNames(t, "callers", "callers", "List", "-p", dir); slices.Contains(callers, "newDaemonCmd") {
			t.Errorf("callers List = %v: a production reassignment must withhold the call", callers)
		}
		if got := queryNames(t, "callers", "callers", "withStubbedDaemonList", "-p", dir); len(got) == 0 {
			t.Error("callers withStubbedDaemonList is empty: daemon_test.go is not indexed")
		}
	})
}
