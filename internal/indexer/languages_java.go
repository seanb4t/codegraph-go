package indexer

import (
	"encoding/xml"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/indexer/javaextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// javaProjectDescriptor wraps a Maven pom.xml/Gradle build.gradle's
// resolved group identity — Java's own first ProjectDescriptor
// implementation (D-03), mirroring goProjectDescriptor's shape.
type javaProjectDescriptor struct {
	basePackage string
}

func (d javaProjectDescriptor) ModulePath() string { return d.basePackage }

// buildGradleGroupPattern matches a Gradle build script's `group` project
// property (`group = 'com.example'` or `group 'com.example'`, Groovy or
// Kotlin DSL, single or double quotes). This is a narrow, bounded
// single-line match — not a general Groovy/Kotlin parser — matching only
// the one property this project needs (T-05-Manifest: a malformed/unusual
// build.gradle simply fails to match, degrading to the path-identity
// fallback below, never a crash).
var buildGradleGroupPattern = regexp.MustCompile(`(?m)^\s*group\s*[:=]?\s*['"]([^'"]+)['"]`)

// mavenPOM is the minimal pom.xml shape this project reads: a project's own
// groupId, falling back to its parent's groupId (the common Maven
// multi-module convention of declaring groupId once on the parent POM).
type mavenPOM struct {
	XMLName xml.Name `xml:"project"`
	GroupID string   `xml:"groupId"`
	Parent  struct {
		GroupID string `xml:"groupId"`
	} `xml:"parent"`
}

// readJavaDescriptor resolves a Java repo root's base package/group
// identity from pom.xml (preferred) or build.gradle/build.gradle.kts,
// returning an error if neither manifest is present or parseable — the
// same "descriptor absent, malformed, or missing" signal Discover already
// contractually treats as "fall back to path-based identity" (D-03), never
// a hard failure (T-05-Manifest: accept, no crash, no code execution).
func readJavaDescriptor(root string) (string, error) {
	if data, err := os.ReadFile(filepath.Join(root, "pom.xml")); err == nil {
		var pom mavenPOM
		if err := xml.Unmarshal(data, &pom); err == nil {
			if pom.GroupID != "" {
				return pom.GroupID, nil
			}
			if pom.Parent.GroupID != "" {
				return pom.Parent.GroupID, nil
			}
		}
	}
	for _, name := range []string{"build.gradle", "build.gradle.kts"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			continue
		}
		if m := buildGradleGroupPattern.FindSubmatch(data); m != nil {
			return string(m[1]), nil
		}
	}
	return "", fmt.Errorf("indexer: no pom.xml/build.gradle(.kts) group identity found under %s", root)
}

// init registers Java as a LanguageSpec (LANG-02).
//
// ModuleKey's signature (descriptor, relPath) cannot see file content, so
// it can only ever compute a PATH-BASED placeholder — Java's real
// cross-file identity is declared IN the source (`package com.foo.bar;`,
// independent of directory layout, RESEARCH Pitfall 2). javaextract.Extract
// parses that declaration and overrides FileResult.ImportPath with it
// once parsing has actually happened; this ModuleKey is only ever
// load-bearing for a file with no package declaration at all.
func init() {
	registerLanguage(LanguageSpec{
		ID:         "java",
		Extensions: []string{".java"},
		NewParser: func() (parser.Parser, error) {
			return cgo.NewJavaParser()
		},
		Extract: javaextract.Extract,
		ModuleKey: func(descriptor ProjectDescriptor, relPath string) string {
			if descriptor == nil {
				// D-03 path-identity fallback (mirrors Go's own
				// nil-descriptor fallback, languages_go.go) — javaextract.
				// Extract overrides this with the parsed `package`
				// declaration whenever one is present.
				return relPath
			}
			base := descriptor.ModulePath()
			dir := path.Dir(relPath)
			if dir == "." {
				return base
			}
			return base + "." + strings.ReplaceAll(dir, "/", ".")
		},
		Descriptor: func(root string) (ProjectDescriptor, error) {
			base, err := readJavaDescriptor(root)
			if err != nil {
				return nil, err
			}
			return javaProjectDescriptor{basePackage: base}, nil
		},
	})
}
