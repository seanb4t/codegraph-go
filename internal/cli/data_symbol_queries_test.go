package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// dataSymbolFiles is a Java class with a field balance that deposit reads
// and writes, and a field and a method that share the name size. Ledger
// is a distractor whose method name contains "balance".
var dataSymbolFiles = map[string]string{
	"p/Account.java": `package p;

public class Account {
    private int balance;
    int size;

    public int deposit(int amount) {
        balance += amount;
        return balance;
    }

    public int size() {
        return size;
    }
}
`,
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
