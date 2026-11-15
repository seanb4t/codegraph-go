package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/testtree"
)

// dataSymbolFiles is a Java class with a field balance that deposit reads
// and writes, and a field and a method that share the name size. Ledger
// is a distractor whose method name contains "balance".
var dataSymbolFiles = map[string]string{
	"p/Account.java": testtree.AccountJava,
	"p/Ledger.java": `package p;

public class Ledger {
    public String balanceReport() {
        return "report";
    }
}
`,
}

func requireContains(t *testing.T, what, text string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Errorf("%s lacks %q:\n%s", what, want, text)
		}
	}
}

// TestDataSymbolPublicQueries covers the CLI side of user-visible data
// symbol access. node names a field's owner, kind, language and source
// location, and keeps a field apart from a method of the same name. The
// field kind filter of search returns only the field. explore over the
// field and its reader lists both with their kinds.
func TestDataSymbolPublicQueries(t *testing.T) {
	dir := initTree(t, dataSymbolFiles)

	out, _, err := execCmd("node", "balance", "-p", dir)
	if err != nil {
		t.Fatalf("node balance: %v", err)
	}
	requireContains(t, "node balance", out,
		"**balance** (field)\n",
		"**Location:** p/Account.java:4\n",
		"**Owner:** `Account`\n",
		"**Language:** java\n",
	)

	out, _, err = execCmd("node", "size", "-p", dir)
	if err != nil {
		t.Fatalf("node size: %v", err)
	}
	requireContains(t, "node size", out, "**size** (field)", "**size** (method)", "**Owner:** `Account`")

	out, _, err = execCmd("search", "size", "--kind", "field", "--full", "--json", "-p", dir)
	if err != nil {
		t.Fatalf("search size --kind field: %v", err)
	}
	var rows []struct {
		Node struct {
			Kind          string `json:"kind"`
			QualifiedName string `json:"qualifiedName"`
			Language      string `json:"language"`
		} `json:"node"`
	}
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("invalid search JSON: %v\n%s", err, out)
	}
	if len(rows) != 1 || rows[0].Node.Kind != "field" || rows[0].Node.QualifiedName != "Account.size" || rows[0].Node.Language != "java" {
		t.Errorf("search size --kind field = %+v, want only the java field Account.size", rows)
	}

	out, _, err = execCmd("explore", "deposit balance", "-p", dir)
	if err != nil {
		t.Fatalf("explore: %v", err)
	}
	requireContains(t, "explore deposit balance", out, "balance(field)", "deposit(method)")
}

// TestNodeShowsDataAccess covers the node scenarios of the requirements
// for the access and initializer lines of a data symbol.
func TestNodeShowsDataAccess(t *testing.T) {
	gomod := "module example.com/m\n\ngo 1.22\n"
	java := initTree(t, dataSymbolFiles)
	readers := initTree(t, map[string]string{
		"go.mod": gomod,
		"a.go":   "package p\n\nfunc first() int { return count }\n",
		"b.go":   "package p\n\nfunc second() int { return count }\n",
		"c.go":   "package p\n\nvar count = 1\n\nconst limit = 3\n",
	})
	node := func(t *testing.T, dir, symbol string) string {
		t.Helper()
		out, _, err := execCmd("node", symbol, "-p", dir)
		if err != nil {
			t.Fatalf("node %s: %v", symbol, err)
		}
		return out
	}
	accessLine := func(t *testing.T, out string) string {
		t.Helper()
		for line := range strings.SplitSeq(out, "\n") {
			if strings.HasPrefix(line, "**Accessed by ←**") {
				return line
			}
		}
		t.Fatalf("output has no access line:\n%s", out)
		return ""
	}

	t.Run("a method reads and writes a field", func(t *testing.T) {
		requireContains(t, "node balance", node(t, java, "balance"), "**Accessed by ←** deposit (p/Account.java:7, read_write)\n")
	})

	t.Run("readers in two files", func(t *testing.T) {
		line := accessLine(t, node(t, readers, "count"))
		first, second := strings.Index(line, "first (a.go:"), strings.Index(line, "second (b.go:")
		if first < 0 || second < 0 || first > second {
			t.Fatalf("access line = %q, want first (a.go:...) before second (b.go:...)", line)
		}
	})

	t.Run("module-level code writes a variable", func(t *testing.T) {
		dir := initTree(t, map[string]string{
			"go.mod":  gomod,
			"main.go": "package main\n\nfunc f() int { return 1 }\n\nvar x = f()\n",
		})
		requireContains(t, "node x", node(t, dir, "x"), "**Accessed by ←** main.go (main.go:1, write)\n")
	})

	t.Run("a variable holds a function", func(t *testing.T) {
		dir := initTree(t, map[string]string{
			"go.mod":  gomod,
			"main.go": "package main\n\nfunc start() {}\n\nvar run = start\n",
		})
		requireContains(t, "node run", node(t, dir, "run"), "**Initialized to →** start (main.go:3)\n")
	})

	t.Run("nothing accesses a constant", func(t *testing.T) {
		out := node(t, readers, "limit")
		if !strings.HasSuffix(out, "**Accessed by ←** \n**Initialized to →** \n") {
			t.Fatalf("node limit does not end with two empty data lines:\n%q", out)
		}
	})

	t.Run("the data lines appear only for data symbols", func(t *testing.T) {
		out := node(t, java, "size")
		field, method := strings.Index(out, "**size** (field)"), strings.Index(out, "**size** (method)")
		if field < 0 || method < 0 {
			t.Fatalf("node size lacks a section:\n%s", out)
		}
		sections := map[string]string{}
		if field < method {
			sections["field"], sections["method"] = out[field:method], out[method:]
		} else {
			sections["method"], sections["field"] = out[method:field], out[field:]
		}
		requireContains(t, "the field section", sections["field"], "**Accessed by ←** size (p/Account.java:12, read)")
		for _, label := range []string{"**Accessed by ←**", "**Initialized to →**"} {
			if strings.Contains(sections["method"], label) {
				t.Errorf("the method section has %q:\n%s", label, sections["method"])
			}
		}
	})

	t.Run("a function keeps its output", func(t *testing.T) {
		out := node(t, java, "deposit")
		for _, label := range []string{"**Accessed by ←**", "**Initialized to →**"} {
			if strings.Contains(out, label) {
				t.Errorf("node deposit has %q:\n%s", label, out)
			}
		}
		requireContains(t, "node deposit", out, "**deposit** (method)", "**Called by ←**")
	})
}
