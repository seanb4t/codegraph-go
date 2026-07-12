package indexer

import (
	"fmt"
	"go/build"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/modfile"
)

// DiscoveredFile is one discovered source file of any registered language,
// the shape Pass 1 (extract) and Pass 2 (resolve) consume.
type DiscoveredFile struct {
	// AbsPath is the absolute path on disk, for reading bytes.
	AbsPath string
	// RelPath is the slash-normalized path relative to the repo root, the
	// file_path stored on Node/File records.
	RelPath string
	// ImportPath is this file's cross-file symbol-index key, computed by
	// its language's LanguageSpec.ModuleKey (D-03/Pitfall 2). For Go this
	// is the module path joined with the file's relative directory (""
	// relative directory means the module root package) — byte-identical
	// to the pre-Phase-5 behavior. A file whose language has no resolvable
	// project descriptor still gets a value here (its LanguageSpec's
	// path-based fallback), never a dropped file.
	ImportPath string
	// Language is the registered LanguageSpec.ID this file's extension
	// resolved to ("go", "java", "python", ...) — the key extract.go's
	// worker pool uses to select the correct parser + extractor per file
	// (Pitfall 1).
	Language string

	// MtimeUnixNs and SizeBytes are the file's on-disk stat info at
	// discovery time (Phase 4 D-01a) — carried through Extract into the
	// committed File record so Sync's stat pre-filter has something cheap
	// to compare against on the next invocation, without hashing every
	// file every sync.
	MtimeUnixNs int64
	SizeBytes   int64
}

// ShouldSkipDir reports whether a directory named name should be excluded
// from traversal — Discover's own WalkDir callback and, per Phase 4 D-04,
// the native filesystem watcher's recursive-add loop both call this exact
// predicate so the two never silently diverge on which paths they cover.
// vendor/ and any dot-prefixed directory (.git, .codegraph, etc.) are
// excluded.
func ShouldSkipDir(name string) bool {
	return name == "vendor" || strings.HasPrefix(name, ".")
}

// pendingFile is Discover's own walk-time intermediate, before each
// language's ModuleKey has been computed (that requires the language's
// repo-root descriptor to be resolved first, which happens once per
// language AFTER the walk completes, not per file during it).
type pendingFile struct {
	abs, relPath, language string
	mtimeUnixNs, sizeBytes int64
}

// Discover walks root and returns every file whose extension is claimed by
// a registered LanguageSpec (D-03), sorted by RelPath in ascending byte
// order. This stable order is determinism's first line of defense: the
// same input tree always yields the same output order, regardless of
// filesystem walk order.
//
// vendor/ directories and any dot-prefixed directory (.git, .codegraph,
// etc.) are skipped entirely (ShouldSkipDir, shared verbatim with the
// Phase-4 watcher). A candidate file is included iff its extension is
// registered in the extension->language registry (languages.go); an
// unsupported extension (.md, .json, ...) is never returned. Go source
// files additionally require go/build.Context.MatchFile to report they
// belong to the default build context (GOOS/GOARCH, build tags) — the same
// primitive the go toolchain itself uses — gated to Language=="go" only,
// since no other language in the registry has a build-tag concept.
//
// After the walk, each language actually present is given exactly one
// chance to resolve its repo-root project descriptor (go.mod, pom.xml,
// *.csproj, ...) via LanguageSpec.Descriptor. A descriptor that is absent,
// malformed, or simply not implemented for that language does NOT fail
// Discover (D-03/T-05-Manifest) — LanguageSpec.ModuleKey is called with a
// nil descriptor and is required to degrade to a path-based identity
// rather than dropping the file. This is the one behavioral relaxation
// from the pre-Phase-5 contract: a root with no go.mod (and only Go files)
// used to be a hard Discover error; it now succeeds with Go's own
// nil-descriptor fallback (languages_go.go).
//
// Discover's second return value remains the repo's Go module path
// specifically (as resolved by the "go" LanguageSpec's own descriptor, if
// any) — every existing caller (Sync, Resolve, symbolindex.go) consumes
// this Go-specific value unchanged; it is "" when no go.mod was found.
func Discover(root string) ([]DiscoveredFile, string, error) {
	ctx := build.Default
	var pending []pendingFile

	walkErr := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && ShouldSkipDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(d.Name()))
		spec, ok := lookupLanguageByExt(ext)
		if !ok {
			return nil
		}

		abs, err := filepath.Abs(p)
		if err != nil {
			return err
		}

		if spec.ID == "go" {
			// Pitfall 5: MatchFile must be given the file's OWN parent
			// directory, never a hoisted/cached value, or build-tag
			// evaluation silently mis-fires. No other registered language
			// has a build-tag concept, so this stays Go-only.
			match, err := ctx.MatchFile(filepath.Dir(abs), filepath.Base(abs))
			if err != nil {
				return err
			}
			if !match {
				return nil
			}
		}

		relPath, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		info, err := d.Info()
		if err != nil {
			return err
		}

		pending = append(pending, pendingFile{
			abs:         abs,
			relPath:     relPath,
			language:    spec.ID,
			mtimeUnixNs: info.ModTime().UnixNano(),
			sizeBytes:   info.Size(),
		})
		return nil
	})
	if walkErr != nil {
		return nil, "", walkErr
	}

	// Resolve each present language's project descriptor exactly once per
	// repo root (D-03) — never per file. A language with no Descriptor
	// hook, or whose Descriptor call errors (missing/malformed manifest),
	// simply has no entry in descriptors; ModuleKey is called with nil in
	// that case and is contractually required to fall back to a
	// path-based identity rather than dropping the file.
	descriptors := make(map[string]ProjectDescriptor)
	descriptorAttempted := make(map[string]bool)

	files := make([]DiscoveredFile, 0, len(pending))
	for _, pf := range pending {
		spec, ok := lookupLanguageByID(pf.language)
		if !ok {
			// A file was matched by extension during the walk but its
			// language was deregistered before this second pass ran —
			// cannot happen in practice (registrations are init()-time
			// and never removed), but skip defensively rather than panic.
			continue
		}

		if !descriptorAttempted[pf.language] {
			descriptorAttempted[pf.language] = true
			if spec.Descriptor != nil {
				if d, err := spec.Descriptor(root); err == nil {
					descriptors[pf.language] = d
				}
			}
		}

		var importPath string
		if spec.ModuleKey != nil {
			importPath = spec.ModuleKey(descriptors[pf.language], pf.relPath)
		} else {
			importPath = pf.relPath
		}

		files = append(files, DiscoveredFile{
			AbsPath:     pf.abs,
			RelPath:     pf.relPath,
			ImportPath:  importPath,
			Language:    pf.language,
			MtimeUnixNs: pf.mtimeUnixNs,
			SizeBytes:   pf.sizeBytes,
		})
	}

	sort.Slice(files, func(i, j int) bool { return files[i].RelPath < files[j].RelPath })

	modulePath := ""
	if d, ok := descriptors["go"]; ok {
		modulePath = d.ModulePath()
	}

	return files, modulePath, nil
}

// readModulePath parses root/go.mod and returns its declared module path.
// This is Go's own LanguageSpec.Descriptor implementation (languages_go.go)
// and is also consulted directly by symbolindex.go's store-seeded index.
func readModulePath(root string) (string, error) {
	goModPath := filepath.Join(root, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("indexer: reading %s: %w", goModPath, err)
	}

	f, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return "", fmt.Errorf("indexer: parsing %s: %w", goModPath, err)
	}
	if f.Module == nil {
		return "", fmt.Errorf("indexer: %s has no module directive", goModPath)
	}
	return f.Module.Mod.Path, nil
}

// importPathFor computes a file's Go import path from the module's base
// import path and the file's slash-normalized path relative to the module
// root (Pattern 5: modulePath + "/" + relDir; modulePath alone for the
// root package). This is Go's own LanguageSpec.ModuleKey implementation
// (languages_go.go) and is also consulted directly by symbolindex.go's
// store-seeded index.
func importPathFor(modulePath, relPath string) string {
	relDir := path.Dir(relPath)
	if relDir == "." {
		return modulePath
	}
	return modulePath + "/" + relDir
}
