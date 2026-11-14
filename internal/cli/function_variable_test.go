package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
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

// edgeKey names one stored edge as "<edge kind> <target kind> <target
// name> <access>".
type edgeKey = string

// outEdges returns the stored edges that leave the symbol name declared
// in path.
func outEdges(t *testing.T, dir, path, name string) []edgeKey {
	t.Helper()
	store, err := graphstore.Open(filepath.Join(dir, codegraphDirName, storeDirName))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()
	r, err := store.Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	defer r.Close()
	it, err := r.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	var from string
	for it.Next() {
		if n := it.Node(); n.FilePath == path && n.Name == name {
			from = n.Id
		}
	}
	it.Close()
	if from == "" {
		t.Fatalf("no symbol %s in %s", name, path)
	}
	eit, err := r.IterateEdges(from)
	if err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}
	defer eit.Close()
	var out []edgeKey
	for eit.Next() {
		e := eit.Edge()
		n, err := r.GetNode(e.Target)
		if err != nil {
			t.Fatalf("GetNode(%s): %v", e.Target, err)
		}
		out = append(out, e.Kind+" "+n.Kind+" "+n.Name+" "+e.Metadata["access"])
	}
	return out
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
		cmd := outEdges(t, dir, "internal/cli/daemon.go", "newDaemonCmd")
		if !slices.Contains(cmd, "references variable daemonList read") {
			t.Errorf("newDaemonCmd edges %v, want a read reference to the binding daemonList", cmd)
		}
		for _, e := range cmd {
			if kind, _, _ := strings.Cut(e, " "); kind == "calls" && !strings.HasPrefix(e, "calls function ") && !strings.HasPrefix(e, "calls method ") {
				t.Errorf("newDaemonCmd has the call %q to something that is not a function", e)
			}
		}
		binding := outEdges(t, dir, "internal/cli/daemon.go", "daemonList")
		if !slices.Contains(binding, "references function List read") {
			t.Errorf("daemonList edges %v, want a reference to the function List it holds", binding)
		}
		for _, e := range binding {
			if strings.HasPrefix(e, "calls ") {
				t.Errorf("the initializer of daemonList makes the call %q", e)
			}
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
