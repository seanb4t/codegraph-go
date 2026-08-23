package query

import (
	"errors"
	"testing"
)

// TestClassifiedErrorsPreserveTheirMessages asserts, for each converted
// site, that the message equals a literal recorded live from the
// pre-conversion tree (captured by hand during this task's execution,
// before any site below was converted — see 01-05-SUMMARY.md for the full
// capture transcript) — proving the classifiedError carrier changes the
// error's TYPE without moving a single message byte.
func TestClassifiedErrorsPreserveTheirMessages(t *testing.T) {
	e := newDetailFixtureEngine(t)

	cases := []struct {
		name string
		err  error
		want string
	}{
		{"validateLimit negative", validateLimit(-1), "query: limit -1 must be non-negative"},
		{"validateLimit exceeds max", validateLimit(MaxLimit + 1), "query: limit 1001 exceeds maximum 1000"},
		{"validateMaxFiles negative", validateMaxFiles(-1), "query: max-files -1 must be non-negative"},
		{"validateMaxFiles exceeds max", validateMaxFiles(MaxFiles + 1), "query: max-files 1001 exceeds maximum 1000"},
		{"validateDepth negative", validateDepth(-1), "query: depth -1 must be non-negative"},
		{"validateFilesDepth negative", validateFilesDepth(-1), "query: depth -1 must be non-negative"},
		{"validateFilesDepth exceeds max", validateFilesDepth(MaxDepth + 1), "query: depth 51 exceeds maximum 50"},
		{"ValidateKind unknown", ValidateKind("banana"), `query: unknown kind "banana" — allowed kinds: constant, file, function, interface, method, package, struct, type_alias, variable`},
	}

	callCases := []struct {
		name string
		err  error
		want string
	}{
		{"Files bogus format", filesErr(t, e, FilesOptions{Format: "bogus"}), `query: unknown files format "bogus" — allowed: flat, tree`},
		{"Query empty term", queryErr(t, e), "query: search term must not be empty"},
		{"Search empty term", searchErr(t, e), "query: search term must not be empty"},
		{"Explore empty query", exploreErrFor(t, e, "", 5), "query: explore query must not be empty"},
		{"Explore negative maxFiles", exploreErrFor(t, e, "anything", -1), "query: max-files -1 must be non-negative"},
		{"Callers not found", callersErr(t, e), `query: symbol "nosuchsymbolxyz" not found`},
		{"SourceFor empty path", sourceForErr(t, e, ""), "query: empty file path"},
		{"SourceFor absolute path", sourceForErr(t, e, "/etc/passwd"), `query: absolute path "/etc/passwd" is not allowed`},
		{"SourceFor escapes repo root", sourceForErr(t, e, "../outside.txt"), `query: path "../outside.txt" escapes the repo root`},
		{"NodeDetail empty args", nodeDetailErr(t, e, "", ""), "query: node requires a symbol name or a file path"},
		{"NodeDetail not found", nodeDetailErr(t, e, "nosuchsymbolxyz", ""), `query: symbol "nosuchsymbolxyz" not found`},
	}

	total := 0
	for _, c := range cases {
		total++
		t.Run(c.name, func(t *testing.T) {
			if c.err == nil {
				t.Fatalf("%s: got nil error, want a non-nil error carrying %q", c.name, c.want)
			}
			if c.err.Error() != c.want {
				t.Fatalf("%s: message = %q, want recorded literal %q", c.name, c.err.Error(), c.want)
			}
		})
	}
	for _, c := range callCases {
		total++
		t.Run(c.name, func(t *testing.T) {
			if c.err == nil {
				t.Fatalf("%s: got nil error, want a non-nil error carrying %q", c.name, c.want)
			}
			if c.err.Error() != c.want {
				t.Fatalf("%s: message = %q, want recorded literal %q", c.name, c.err.Error(), c.want)
			}
		})
	}
	t.Logf("TestClassifiedErrorsPreserveTheirMessages exercised %d rows", total)
}

func filesErr(t *testing.T, e *Engine, opts FilesOptions) error {
	t.Helper()
	_, err := e.Files(opts)
	return err
}

func queryErr(t *testing.T, e *Engine) error {
	t.Helper()
	_, err := e.Query("", "", 0)
	return err
}

func searchErr(t *testing.T, e *Engine) error {
	t.Helper()
	_, err := e.Search("", "", 0)
	return err
}

func exploreErrFor(t *testing.T, e *Engine, query string, maxFiles int) error {
	t.Helper()
	_, err := e.Explore(query, maxFiles)
	return err
}

func callersErr(t *testing.T, e *Engine) error {
	t.Helper()
	_, err := e.Callers("nosuchsymbolxyz", 0)
	return err
}

func sourceForErr(t *testing.T, e *Engine, path string) error {
	t.Helper()
	_, err := e.SourceFor(path)
	return err
}

func nodeDetailErr(t *testing.T, e *Engine, symbol, file string) error {
	t.Helper()
	_, err := e.NodeDetail(symbol, file, nil)
	return err
}

// TestEveryReachableErrorIsClassified is a table over the enumerated
// caller-reachable sites (see 01-05-SUMMARY.md for the full file/line/
// class list): each row drives a real Engine entry point with a
// known-bad input, asserts the expected classification with errors.Is,
// asserts the exact recorded message, and asserts the error is NOT the
// OTHER class. The count of rows exercised is logged and the test fails
// if it drops below the recorded total (rule 84d1gfpywd) — the
// classification table cannot silently shrink.
func TestEveryReachableErrorIsClassified(t *testing.T) {
	const recordedTotal = 15
	e := newDetailFixtureEngine(t)

	type row struct {
		name  string
		err   error
		class error
		want  string
	}
	rows := []row{
		{"Node/NodeDetail empty args", nodeDetailErr(t, e, "", ""), ErrInvalidArgument, "query: node requires a symbol name or a file path"},
		{"Node/NodeDetail not found", nodeDetailErr(t, e, "nosuchsymbolxyz", ""), ErrNotFound, `query: symbol "nosuchsymbolxyz" not found`},
		{"Node/NodeDetail file escapes repo root", nodeDetailErr(t, e, "", "../outside.txt"), ErrInvalidArgument, `query: path "../outside.txt" escapes the repo root`},
		{"SourceFor empty path", sourceForErr(t, e, ""), ErrInvalidArgument, "query: empty file path"},
		{"SourceFor absolute path", sourceForErr(t, e, "/etc/passwd"), ErrInvalidArgument, `query: absolute path "/etc/passwd" is not allowed`},
		{"Query empty term", queryErr(t, e), ErrInvalidArgument, "query: search term must not be empty"},
		{"Search empty term", searchErr(t, e), ErrInvalidArgument, "query: search term must not be empty"},
		{"Explore empty query", exploreErrFor(t, e, "", 5), ErrInvalidArgument, "query: explore query must not be empty"},
		{"Explore negative maxFiles", exploreErrFor(t, e, "anything", -1), ErrInvalidArgument, "query: max-files -1 must be non-negative"},
		{"Callers not found", callersErr(t, e), ErrNotFound, `query: symbol "nosuchsymbolxyz" not found`},
		{"Callees negative limit", calleesErr(t, e, -1), ErrInvalidArgument, "query: limit -1 must be non-negative"},
		{"Impact negative depth", impactErr(t, e, -1), ErrInvalidArgument, "query: depth -1 must be non-negative"},
		{"Files bogus format", filesErr(t, e, FilesOptions{Format: "bogus"}), ErrInvalidArgument, `query: unknown files format "bogus" — allowed: flat, tree`},
		{"Files negative depth", filesErr(t, e, FilesOptions{Depth: -1}), ErrInvalidArgument, "query: depth -1 must be non-negative"},
		{"Query unknown kind", queryKindErr(t, e, "banana"), ErrInvalidArgument, `query: unknown kind "banana" — allowed kinds: constant, file, function, interface, method, package, struct, type_alias, variable`},
	}

	other := func(class error) error {
		if class == ErrNotFound {
			return ErrInvalidArgument
		}
		return ErrNotFound
	}

	exercised := 0
	for _, r := range rows {
		exercised++
		t.Run(r.name, func(t *testing.T) {
			if r.err == nil {
				t.Fatalf("%s: got nil error, want a classified error", r.name)
			}
			if !errors.Is(r.err, r.class) {
				t.Fatalf("%s: errors.Is(err, wantClass) = false, want true (err=%q)", r.name, r.err.Error())
			}
			if errors.Is(r.err, other(r.class)) {
				t.Fatalf("%s: errors.Is(err, otherClass) = true, want false — an error must never be BOTH classes", r.name)
			}
			if r.err.Error() != r.want {
				t.Fatalf("%s: message = %q, want recorded literal %q", r.name, r.err.Error(), r.want)
			}
		})
	}

	t.Logf("TestEveryReachableErrorIsClassified exercised %d of %d recorded rows", exercised, recordedTotal)
	if exercised < recordedTotal {
		t.Fatalf("TestEveryReachableErrorIsClassified: exercised %d rows, want at least the recorded total %d — the classification table shrank", exercised, recordedTotal)
	}
}

func calleesErr(t *testing.T, e *Engine, limit int) error {
	t.Helper()
	_, err := e.Callees("whatever", limit)
	return err
}

func impactErr(t *testing.T, e *Engine, depth int) error {
	t.Helper()
	_, err := e.Impact("whatever", depth)
	return err
}

func queryKindErr(t *testing.T, e *Engine, kind string) error {
	t.Helper()
	_, err := e.Query("something", kind, 0)
	return err
}
