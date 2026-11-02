package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/invariant"
)

// propDecl is one declaration in the model of TestSyncEqualsFreshIndex.
type propDecl struct {
	Name string
	// Class makes the declaration a class (TypeScript, Java) or a struct
	// type (Go) with one method, run.
	Class bool
	// Repeat marks a copy of an earlier declaration of the same name. In
	// TypeScript it is written inside an it(...) callback, as in a test
	// file.
	Repeat bool
	// Overload gives a Java method a String parameter, so that it is an
	// overload of the method of the same name with none (design.md D7).
	Overload bool
	Calls    []propCall
}

// propCall is one call in a declaration body. Args is the argument text,
// such as a String literal for a typed call. Receiver names another Java
// file's class: the call goes through a local typed from that class's
// factory, `var r = Receiver.make(); r.callee(args);`, so the local's
// type comes from a call into another file (design.md D17).
type propCall struct {
	Callee   string
	Args     string
	Receiver string
}

// propModel is the tree that the property writes to disk.
type propModel struct {
	files map[string][]propDecl
}

var (
	propPaths = []string{"a.go", "b.go", "sub/c.go", "x.ts", "y.ts", "sub/z.ts", "App.java", "Util.java", "sub/Deep.java"}
	propExts  = []string{".go", ".ts", ".java"}
	propNames = []string{"alpha", "beta", "gamma"}
)

// javaPaths returns the model's Java files.
func (m *propModel) javaPaths() []string {
	var out []string
	for _, p := range m.paths() {
		if strings.HasSuffix(p, ".java") {
			out = append(out, p)
		}
	}
	return out
}

// javaClass is the class a Java file of the model declares: its stem.
func javaClass(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".java")
}

func (m *propModel) paths() []string {
	out := make([]string, 0, len(m.files))
	for p := range m.files {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func (m *propModel) unused(ext string) []string {
	var out []string
	for _, p := range propPaths {
		if _, ok := m.files[p]; !ok && strings.HasSuffix(p, ext) {
			out = append(out, p)
		}
	}
	return out
}

func renderGo(decls []propDecl) string {
	var b strings.Builder
	b.WriteString("package prop\n")
	for _, d := range decls {
		if d.Class {
			fmt.Fprintf(&b, "\ntype %s struct{}\n\nfunc (%s) run() {}\n", d.Name, d.Name)
			continue
		}
		fmt.Fprintf(&b, "\nfunc %s() {\n", d.Name)
		for _, c := range d.Calls {
			fmt.Fprintf(&b, "\t%s(%s)\n", c.Callee, c.Args)
		}
		b.WriteString("}\n")
	}
	return b.String()
}

// renderJava writes one class named after the file, with each function
// declaration as a method, each class declaration as a nested class, and
// a static factory for the typed-local operation.
func renderJava(class string, decls []propDecl) string {
	var b strings.Builder
	fmt.Fprintf(&b, "package prop;\n\npublic class %s {\n    public static %s make() {\n        return new %s();\n    }\n", class, class, class)
	for _, d := range decls {
		if d.Class {
			fmt.Fprintf(&b, "\n    public static class %s {\n        public void run() {\n        }\n    }\n", d.Name)
			continue
		}
		params := ""
		if d.Overload {
			params = "String s"
		}
		fmt.Fprintf(&b, "\n    public void %s(%s) {\n", d.Name, params)
		for i, c := range d.Calls {
			if c.Receiver != "" {
				fmt.Fprintf(&b, "        var r%d = %s.make();\n        r%d.%s(%s);\n", i, c.Receiver, i, c.Callee, c.Args)
				continue
			}
			fmt.Fprintf(&b, "        %s(%s);\n", c.Callee, c.Args)
		}
		b.WriteString("    }\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func renderTS(decls []propDecl) string {
	var b strings.Builder
	for _, d := range decls {
		var body strings.Builder
		if d.Class {
			fmt.Fprintf(&body, "class %s {\n  run() {}\n}\n", d.Name)
		} else {
			fmt.Fprintf(&body, "function %s() {\n", d.Name)
			for _, c := range d.Calls {
				fmt.Fprintf(&body, "  %s(%s);\n", c.Callee, c.Args)
			}
			body.WriteString("}\n")
		}
		if d.Repeat {
			fmt.Fprintf(&b, "\nit(%q, () => {\n%s});\n", d.Name, body.String())
		} else {
			fmt.Fprintf(&b, "\nexport %s", body.String())
		}
	}
	return b.String()
}

func (m *propModel) render(path string) string {
	switch filepath.Ext(path) {
	case ".go":
		return renderGo(m.files[path])
	case ".java":
		return renderJava(javaClass(path), m.files[path])
	}
	return renderTS(m.files[path])
}

// propOps are the eleven operations of design.md D7: three change files,
// eight change declarations. Each returns a description for the log, or
// "" when it does not apply to the current model.
var propOps = []func(t *rapid.T, m *propModel) string{
	func(t *rapid.T, m *propModel) string { // add a file
		free := m.unused(rapid.SampledFrom(propExts).Draw(t, "ext"))
		if len(free) == 0 {
			return ""
		}
		p := rapid.SampledFrom(free).Draw(t, "path")
		m.files[p] = []propDecl{{Name: rapid.SampledFrom(propNames).Draw(t, "name")}}
		return "add file " + p
	},
	func(t *rapid.T, m *propModel) string { // delete a file
		if len(m.files) == 0 {
			return ""
		}
		p := rapid.SampledFrom(m.paths()).Draw(t, "path")
		delete(m.files, p)
		return "delete file " + p
	},
	func(t *rapid.T, m *propModel) string { // rename a file
		if len(m.files) == 0 {
			return ""
		}
		from := rapid.SampledFrom(m.paths()).Draw(t, "from")
		free := m.unused(filepath.Ext(from))
		if len(free) == 0 {
			return ""
		}
		to := rapid.SampledFrom(free).Draw(t, "to")
		m.files[to] = m.files[from]
		delete(m.files, from)
		return "rename file " + from + " to " + to
	},
	func(t *rapid.T, m *propModel) string { // add a declaration
		if len(m.files) == 0 {
			return ""
		}
		p := rapid.SampledFrom(m.paths()).Draw(t, "path")
		name := rapid.SampledFrom(propNames).Draw(t, "name")
		m.files[p] = append(m.files[p], propDecl{Name: name})
		return "add function " + name + " to " + p
	},
	func(t *rapid.T, m *propModel) string { // remove a declaration
		p, i, ok := drawDecl(t, m, false)
		if !ok {
			return ""
		}
		name := m.files[p][i].Name
		m.files[p] = append(m.files[p][:i:i], m.files[p][i+1:]...)
		return fmt.Sprintf("remove declaration %d (%s) from %s", i, name, p)
	},
	func(t *rapid.T, m *propModel) string { // rename a declaration
		p, i, ok := drawDecl(t, m, false)
		if !ok {
			return ""
		}
		name := rapid.SampledFrom(propNames).Draw(t, "name")
		old := m.files[p][i].Name
		m.files[p][i].Name = name
		return fmt.Sprintf("rename declaration %d of %s from %s to %s", i, p, old, name)
	},
	func(t *rapid.T, m *propModel) string { // add a call
		p, i, ok := drawDecl(t, m, true)
		if !ok {
			return ""
		}
		callee := rapid.SampledFrom(propNames).Draw(t, "callee")
		m.files[p][i].Calls = append(m.files[p][i].Calls, propCall{Callee: callee})
		return fmt.Sprintf("add call to %s in %s of %s", callee, m.files[p][i].Name, p)
	},
	func(t *rapid.T, m *propModel) string { // add a repeated declaration
		p, i, ok := drawDecl(t, m, true)
		if !ok {
			return ""
		}
		d := m.files[p][i]
		d.Repeat = true
		d.Calls = append([]propCall(nil), d.Calls...)
		m.files[p] = append(m.files[p], d)
		return fmt.Sprintf("repeat function %s in %s", d.Name, p)
	},
	func(t *rapid.T, m *propModel) string { // add a repeated class with a method
		if len(m.files) == 0 {
			return ""
		}
		p := rapid.SampledFrom(m.paths()).Draw(t, "path")
		name := rapid.SampledFrom(propNames).Draw(t, "name")
		m.files[p] = append(m.files[p], propDecl{Name: name, Class: true}, propDecl{Name: name, Class: true, Repeat: true})
		return "add class " + name + " twice to " + p
	},
	func(t *rapid.T, m *propModel) string { // add an overload and a typed call to it
		java := m.javaPaths()
		if len(java) == 0 {
			return ""
		}
		p := rapid.SampledFrom(java).Draw(t, "path")
		name := rapid.SampledFrom(propNames).Draw(t, "name")
		m.files[p] = append(m.files[p], propDecl{Name: name, Overload: true})
		q, i, ok := drawDecl(t, m, true)
		if ok {
			m.files[q][i].Calls = append(m.files[q][i].Calls, propCall{Callee: name, Args: `"x"`})
			return fmt.Sprintf("add overload %s(String) to %s and a typed call from %s of %s", name, p, m.files[q][i].Name, q)
		}
		return fmt.Sprintf("add overload %s(String) to %s", name, p)
	},
	func(t *rapid.T, m *propModel) string { // add a call through a local typed from another Java file
		java := m.javaPaths()
		if len(java) < 2 {
			return ""
		}
		p := rapid.SampledFrom(java).Draw(t, "path")
		var others []string
		for _, q := range java {
			if q != p {
				others = append(others, q)
			}
		}
		q := rapid.SampledFrom(others).Draw(t, "other")
		var funcs []int
		for i, d := range m.files[p] {
			if !d.Class {
				funcs = append(funcs, i)
			}
		}
		if len(funcs) == 0 {
			return ""
		}
		i := funcs[rapid.IntRange(0, len(funcs)-1).Draw(t, "decl")]
		callee := rapid.SampledFrom(propNames).Draw(t, "callee")
		m.files[p][i].Calls = append(m.files[p][i].Calls, propCall{Callee: callee, Receiver: javaClass(q)})
		return fmt.Sprintf("add call to %s through a local of %s in %s of %s", callee, javaClass(q), m.files[p][i].Name, p)
	},
}

// drawDecl draws a declaration of the model, a function only when
// functions is true.
func drawDecl(t *rapid.T, m *propModel, functions bool) (string, int, bool) {
	type ref struct {
		path string
		i    int
	}
	var refs []ref
	for _, p := range m.paths() {
		for i, d := range m.files[p] {
			if !functions || !d.Class {
				refs = append(refs, ref{p, i})
			}
		}
	}
	if len(refs) == 0 {
		return "", 0, false
	}
	r := refs[rapid.IntRange(0, len(refs)-1).Draw(t, "decl")]
	return r.path, r.i, true
}

// writeModel makes the tree under root match m: it writes each file whose
// content changed and removes each file that m no longer has.
func writeModel(t *rapid.T, root string, m *propModel, written map[string]string) {
	for p := range written {
		if _, ok := m.files[p]; !ok {
			if err := os.Remove(filepath.Join(root, p)); err != nil {
				t.Fatalf("remove %s: %v", p, err)
			}
			delete(written, p)
		}
	}
	for _, p := range m.paths() {
		src := m.render(p)
		if prev, ok := written[p]; ok && prev == src {
			continue
		}
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", p, err)
		}
		if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
		// Sync's stat pre-filter compares mtime and size: a distinct
		// mtime makes it read every rewritten file.
		stamp := time.Now().Add(time.Duration(writeFileSeq.Add(1)) * time.Second)
		if err := os.Chtimes(full, stamp, stamp); err != nil {
			t.Fatalf("chtimes %s: %v", p, err)
		}
		written[p] = src
	}
}

// checkStore runs invariant.Check on the store at storeDir, which must
// hold at least one node for each of the model's files.
func checkStore(t *rapid.T, storeDir string, files int) {
	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()
	r, err := store.Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	defer r.Close()
	c, err := invariant.Check(r)
	if err != nil {
		t.Fatalf("invariant.Check:\n%v", err)
	}
	if c.Nodes < files {
		t.Fatalf("invariant.Check scanned %d nodes for %d files", c.Nodes, files)
	}
}

// TestSyncEqualsFreshIndex runs Sync after each step of a generated
// sequence of file and declaration edits, checks the invariants after each
// sync, and then checks that the store equals a fresh index of the final
// tree (INDX-03, INDX-04).
func TestSyncEqualsFreshIndex(t *testing.T) {
	base := t.TempDir()
	rapid.Check(t, func(t *rapid.T) {
		dir, err := os.MkdirTemp(base, "run")
		if err != nil {
			t.Fatalf("MkdirTemp: %v", err)
		}
		t.Cleanup(func() { os.RemoveAll(dir) })
		root, storeDir, freshDir := filepath.Join(dir, "repo"), filepath.Join(dir, "store"), filepath.Join(dir, "fresh")
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/prop\n\ngo 1.26\n"), 0o644); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}

		m := &propModel{files: map[string][]propDecl{
			"a.go": {{Name: "alpha"}},
			"x.ts": {{Name: "alpha"}},
		}}
		written := make(map[string]string)
		writeModel(t, root, m, written)
		if _, err := Run(root, storeDir, Options{Workers: 1}); err != nil {
			t.Fatalf("Run: %v", err)
		}
		checkStore(t, storeDir, len(m.files))

		steps := rapid.IntRange(1, 8).Draw(t, "steps")
		for i := 0; i < steps; i++ {
			op := rapid.IntRange(0, len(propOps)-1).Draw(t, "op")
			desc := propOps[op](t, m)
			if desc == "" {
				continue
			}
			t.Logf("step %d: %s", i+1, desc)
			writeModel(t, root, m, written)
			if _, err := Sync(root, storeDir, Options{Workers: 1}); err != nil {
				t.Fatalf("Sync after %q: %v", desc, err)
			}
			checkStore(t, storeDir, len(m.files))
		}

		if _, err := Run(root, freshDir, Options{Workers: 1}); err != nil {
			t.Fatalf("Run (fresh): %v", err)
		}
		synced, closeSynced := openRapidSnapshot(t, storeDir)
		defer closeSynced()
		fresh, closeFresh := openRapidSnapshot(t, freshDir)
		defer closeFresh()
		if err := invariant.Equal(synced, fresh); err != nil {
			t.Fatalf("the synced store differs from a fresh index (first store: synced):\n%v", err)
		}
	})
}

func openRapidSnapshot(t *rapid.T, storeDir string) (graphstore.Reader, func()) {
	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("open %s: %v", storeDir, err)
	}
	r, err := store.Snapshot()
	if err != nil {
		store.Close()
		t.Fatalf("snapshot %s: %v", storeDir, err)
	}
	return r, func() { r.Close(); store.Close() }
}
