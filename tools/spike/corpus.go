package main

import (
	"embed"
	"io/fs"
	"sort"
)

// goCorpusFS and pythonCorpusFS embed the pinned-commit benchmark corpus
// (see testdata/ATTRIBUTION.md) directly into the spike binary/test binary,
// so both `go run ./tools/spike` (arbitrary cwd) and `go test ./tools/spike`
// read identical bytes regardless of invocation directory.
//
//go:embed testdata/go/*.go
var goCorpusFS embed.FS

//go:embed testdata/python/*.py
var pythonCorpusFS embed.FS

// corpusFile is one file's path (relative to its testdata/<lang> root) and
// its raw source bytes.
type corpusFile struct {
	Name   string
	Source []byte
}

func loadCorpus(fsys embed.FS, dir string) ([]corpusFile, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	files := make([]corpusFile, 0, len(names))
	for _, name := range names {
		b, err := fs.ReadFile(fsys, dir+"/"+name)
		if err != nil {
			return nil, err
		}
		files = append(files, corpusFile{Name: name, Source: b})
	}
	return files, nil
}

func mustLoadGoCorpus() []corpusFile {
	files, err := loadCorpus(goCorpusFS, "testdata/go")
	if err != nil {
		panic(err)
	}
	return files
}

func mustLoadPythonCorpus() []corpusFile {
	files, err := loadCorpus(pythonCorpusFS, "testdata/python")
	if err != nil {
		panic(err)
	}
	return files
}

func totalBytes(files []corpusFile) int64 {
	var n int64
	for _, f := range files {
		n += int64(len(f.Source))
	}
	return n
}
