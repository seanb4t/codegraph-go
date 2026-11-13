package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// overloadCLIFixture writes and indexes a Java tree with two overloads of
// Logger.log, called from useA and useB.
func overloadCLIFixture(t *testing.T) string {
	t.Helper()
	return initTree(t, map[string]string{
		"Logger.java": "package app;\n\npublic class Logger {\n    public void log(String msg) {}\n\n    public void log(String msg, Throwable t) {}\n}\n",
		"A.java":      "package app;\n\npublic class A {\n    public void useA(Logger logger) {\n        logger.log(\"a\");\n    }\n}\n",
		"B.java":      "package app;\n\npublic class B {\n    public void useB(Logger logger, Throwable e) {\n        logger.log(\"b\", e);\n    }\n}\n",
	})
}

// TestSignatureFlagSelectsOneOverload checks --signature on the four
// commands that take it: each one narrows to the named overload, and the
// plain output of callers carries the overload of each row without it.
func TestSignatureFlagSelectsOneOverload(t *testing.T) {
	dir := overloadCLIFixture(t)
	names := func(out, field string) []string {
		var result map[string]json.RawMessage
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, out)
		}
		var rows []struct {
			Name      string `json:"name"`
			Signature string `json:"signature"`
		}
		if err := json.Unmarshal(result[field], &rows); err != nil {
			t.Fatalf("invalid %s: %v\n%s", field, err, out)
		}
		var out2 []string
		for _, r := range rows {
			out2 = append(out2, r.Name)
		}
		return out2
	}

	t.Run("callers", func(t *testing.T) {
		out, _, err := execCmd("callers", "log", "-p", dir, "--signature", "String, Throwable", "--json")
		if err != nil {
			t.Fatalf("callers: %v", err)
		}
		if got := names(out, "callers"); len(got) != 1 || got[0] != "useB" {
			t.Errorf("callers log --signature = %v, want [useB]", got)
		}
		plain, _, err := execCmd("callers", "log", "-p", dir)
		if err != nil {
			t.Fatalf("callers (plain): %v", err)
		}
		if !strings.Contains(plain, "[(String msg)]") || !strings.Contains(plain, "[(String msg, Throwable t)]") {
			t.Errorf("plain callers lacks the overload of each row:\n%s", plain)
		}
	})
	t.Run("impact", func(t *testing.T) {
		out, _, err := execCmd("impact", "log", "-p", dir, "--signature", "String", "--json")
		if err != nil {
			t.Fatalf("impact: %v", err)
		}
		got := names(out, "affected")
		has := func(name string) bool {
			for _, n := range got {
				if n == name {
					return true
				}
			}
			return false
		}
		if !has("useA") || has("useB") {
			t.Errorf("impact log --signature = %v, want useA and not useB", got)
		}
	})
	t.Run("callees", func(t *testing.T) {
		out, _, err := execCmd("callees", "log", "-p", dir, "--signature", "String", "--json")
		if err != nil {
			t.Fatalf("callees: %v", err)
		}
		if got := names(out, "callees"); len(got) != 0 {
			t.Errorf("callees log --signature = %v, want none", got)
		}
	})
	t.Run("node", func(t *testing.T) {
		out, _, err := execCmd("node", "log", "-p", dir, "--signature", "String, Throwable")
		if err != nil {
			t.Fatalf("node: %v", err)
		}
		if !strings.Contains(out, "(String msg, Throwable t)") || strings.Contains(out, "2 definitions") {
			t.Errorf("node log --signature did not select the two-parameter overload:\n%s", out)
		}
		if _, _, err := execCmd("node", "log", "-p", dir, "--signature", "int"); err == nil {
			t.Error("node log --signature int: expected an error naming the signatures")
		}
	})
}
