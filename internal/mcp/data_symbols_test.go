package mcp

import (
	"strings"
	"testing"
)

// TestDataSymbolPublicQueries covers the MCP side of user-visible data
// symbol access. codegraph_node names a field's owner, kind, language and
// source location, and keeps a field apart from a method of the same
// name. codegraph_search with the field kind returns the field and not
// the method. codegraph_explore over the field and its reader lists both
// with their kinds.
func TestDataSymbolPublicQueries(t *testing.T) {
	call := treeServer(t, map[string]string{
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
	}, "node", "search", "explore")

	requireAll(t, call("codegraph_node", map[string]any{"symbol": "balance"}),
		"**balance** (field)\n",
		"**Location:** p/Account.java:4\n",
		"**Owner:** `Account`\n",
		"**Language:** java\n",
	)
	requireAll(t, call("codegraph_node", map[string]any{"symbol": "size"}),
		"**size** (field)", "**size** (method)")

	search := call("codegraph_search", map[string]any{"query": "size", "kind": "field"})
	requireAll(t, search, "p/Account.java:5")
	if strings.Contains(search, "p/Account.java:12") {
		t.Errorf("a field-kind search lists the method size:\n%s", search)
	}

	requireAll(t, call("codegraph_explore", map[string]any{"query": "deposit balance"}),
		"balance(field)", "deposit(method)")
}
