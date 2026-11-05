package semanticeval

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PythonVersion is the exact CPython the lexical bridge runs under.
const PythonVersion = "3.13.17"

// ClasspathEnv names the variable that holds the scip-javac plugin classpath.
const ClasspathEnv = "CODEGRAPH_SCIP_JAVAC_CLASSPATH"

//go:embed cpython_bridge.py
var cpythonBridge []byte

// ReferenceFor returns the reference that covers language.
func ReferenceFor(language string) Reference {
	switch language {
	case "go":
		return scipGo{}
	case "typescript", "javascript", "tsx":
		return scipTypeScript{}
	case "python":
		return cpython{}
	case "java":
		return scipJavac{}
	}
	return nil
}

func run(dir string, env []string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, out)
	}
	return nil
}

// scipGo reads definitions and reference occurrences from scip-go. Its
// columns are UTF-8 bytes. It cannot establish call kinds or runtime dispatch.
type scipGo struct{}

func (scipGo) Name() string { return "scip-go" }
func (scipGo) Version() (string, error) {
	if _, err := toolVersion("scip", "--version"); err != nil {
		return "", err
	}
	return toolVersion("scip-go", "--version")
}
func (scipGo) Establishes() []string {
	return []string{"declaration-identifiers", "reference-occurrences"}
}
func (scipGo) Index(dir, scratch string, c *Case) ([]Occurrence, error) {
	out := filepath.Join(scratch, "index.scip")
	env := []string{"GOPROXY=off", "GOFLAGS=-mod=mod"}
	if c.BuildContext != "" {
		env = append(env, strings.Split(c.BuildContext, ",")...)
	}
	if err := run(dir, env, "scip-go", "--output", out); err != nil {
		return nil, err
	}
	data, err := printSCIP(out)
	if err != nil {
		return nil, err
	}
	return ParseSCIP(data, dir, EncodingUTF8)
}

// scipTypeScript reads scip-typescript, whose columns are UTF-16 code units.
// It reports no declaration envelopes, so reviewed span rules supply them.
type scipTypeScript struct{}

func (scipTypeScript) Name() string { return "scip-typescript" }
func (scipTypeScript) Version() (string, error) {
	if _, err := toolVersion("scip", "--version"); err != nil {
		return "", err
	}
	return toolVersion("scip-typescript", "--version")
}
func (scipTypeScript) Establishes() []string {
	return []string{"declaration-identifiers", "reference-occurrences"}
}
func (scipTypeScript) Index(dir, scratch string, _ *Case) ([]Occurrence, error) {
	out := filepath.Join(scratch, "index.scip")
	if err := run(dir, nil, "scip-typescript", "index", "--output", out); err != nil {
		return nil, err
	}
	data, err := printSCIP(out)
	if err != nil {
		return nil, err
	}
	return ParseSCIP(data, dir, EncodingUTF16)
}

// scipJavac runs the scip-javac plugin through the pinned JDK. It runs no
// annotation processor and no build tool. Its columns are UTF-16 code units.
type scipJavac struct{}

func (scipJavac) Name() string { return "scip-javac" }
func (scipJavac) Version() (string, error) {
	if _, err := toolVersion("scip", "--version"); err != nil {
		return "", err
	}
	cp := os.Getenv(ClasspathEnv)
	if cp == "" {
		return "", fmt.Errorf("required reference input %s is not set", ClasspathEnv)
	}
	var versions []string
	for _, jar := range strings.Split(cp, string(os.PathListSeparator)) {
		if _, err := os.Stat(jar); err != nil {
			return "", fmt.Errorf("scip-javac classpath: %w", err)
		}
		versions = append(versions, strings.TrimSuffix(filepath.Base(jar), ".jar"))
	}
	javac, err := toolVersion("javac", "-version")
	if err != nil {
		return "", err
	}
	return javac + " " + strings.Join(versions, " "), nil
}
func (scipJavac) Establishes() []string {
	return []string{"declaration-identifiers", "declaration-envelopes", "reference-occurrences"}
}
func (scipJavac) Index(dir, scratch string, c *Case) ([]Occurrence, error) {
	cp := os.Getenv(ClasspathEnv)
	target := filepath.Join(scratch, "target")
	classes := filepath.Join(scratch, "classes")
	if err := os.MkdirAll(classes, 0o755); err != nil {
		return nil, err
	}
	args := []string{
		"-proc:none", "-encoding", "UTF-8", "-d", classes, "-cp", cp,
		"-Xplugin:scip -sourceroot:" + dir + " -targetroot:" + target,
	}
	for _, s := range c.Sources {
		if strings.HasSuffix(s, ".java") {
			args = append(args, s)
		}
	}
	if err := run(dir, nil, "javac", args...); err != nil {
		return nil, err
	}
	var out []Occurrence
	err := filepath.WalkDir(target, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".scip") {
			return err
		}
		data, err := printSCIP(p)
		if err != nil {
			return err
		}
		occs, err := ParseSCIP(data, dir, EncodingUTF16)
		out = append(out, occs...)
		return err
	})
	return out, err
}

// cpython reads declarations and lexical bindings from CPython's ast and
// symtable modules. It cannot establish dynamic call targets.
type cpython struct{}

func pythonArgs(args ...string) []string {
	return append([]string{"run", "--no-project", "--offline", "--python", PythonVersion, "python", "-I"}, args...)
}

func (cpython) Name() string { return "cpython-ast-symtable" }
func (cpython) Version() (string, error) {
	v, err := toolVersion("uv", pythonArgs("-c", "import sys; print(sys.version.split()[0])")...)
	if err != nil {
		return "", err
	}
	if !strings.HasSuffix(v, PythonVersion) {
		return "", fmt.Errorf("uv ran CPython %q, want %s", v, PythonVersion)
	}
	return "CPython " + PythonVersion, nil
}
func (cpython) Establishes() []string { return []string{"declaration-identifiers", "lexical-bindings"} }
func (cpython) Index(dir, scratch string, c *Case) ([]Occurrence, error) {
	bridge := filepath.Join(scratch, "cpython_bridge.py")
	if err := os.WriteFile(bridge, cpythonBridge, 0o644); err != nil {
		return nil, err
	}
	args := pythonArgs(append([]string{bridge, dir}, c.Sources...)...)
	cmd := exec.Command("uv", args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	raw, err := cmd.Output()
	if err != nil {
		return nil, errors.Join(err, errors.New(stderr.String()))
	}
	var rows []struct {
		Path       string `json:"path"`
		Line       int    `json:"line"`
		Col        int    `json:"col"`
		Symbol     string `json:"symbol"`
		Definition bool   `json:"definition"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("decode bridge output: %w", err)
	}
	out := make([]Occurrence, 0, len(rows))
	for _, r := range rows {
		out = append(out, Occurrence{Path: r.Path, At: Position{r.Line, r.Col}, Symbol: r.Symbol, Definition: r.Definition})
	}
	return out, nil
}
