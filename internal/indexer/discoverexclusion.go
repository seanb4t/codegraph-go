package indexer

import (
	"go/build"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// Discovery is DiscoverAll's result: the same walk feeds BOTH the
// discovered-file list and the exclusion-reason list (Phase 10 D-01) — one
// walk, one source of truth, so the denominator and the per-file reasons
// can never disagree.
type Discovery struct {
	Files      []DiscoveredFile
	Excluded   []*schema.ExcludedFile
	ModulePath string
}

// newExcludedFile constructs one c/ namespace record (Phase 10 D-05).
// relPath is the slash-normalized repo-relative path, exactly the shape
// File.path already uses.
func newExcludedFile(relPath string, reason schema.ExclusionReason, detail string, sizeBytes int64) *schema.ExcludedFile {
	return &schema.ExcludedFile{
		Path:      relPath,
		Reason:    reason,
		Detail:    detail,
		SizeBytes: sizeBytes,
	}
}

// buildTagDetail renders the build context Discover's Go build-tag check
// (decision point 3) matched against — the SAME context, never
// runtime.GOOS/GOARCH, which a GOOS/GOARCH environment override would
// desynchronise from what MatchFile actually evaluated.
func buildTagDetail(ctx build.Context) string {
	return ctx.GOOS + "/" + ctx.GOARCH
}

// DiscoverAll walks root and returns every file whose extension is
// claimed by a registered LanguageSpec (D-03), sorted by RelPath in
// ascending byte order, PLUS one ExcludedFile record per path the walker
// visited but did not index — the same walk result feeds both (Phase 10
// D-01), so the discovered-count denominator and the per-file reason list
// can never disagree. This stable order is determinism's first line of
// defense: the same input tree always yields the same output order,
// regardless of filesystem walk order.
//
// vendor/ directories and any dot-prefixed directory (.git, .codegraph,
// etc.) are skipped entirely (ShouldSkipDir, shared verbatim with the
// Phase-4 watcher) — a pruned directory is NOT (yet, Plan 02) recorded as
// a directory-level exclusion; this plan's tracer records ONE reason only
// (BUILD_TAG). A candidate file is included iff its extension is
// registered in the extension->language registry (languages.go); an
// unsupported extension (.md, .json, ...) is, for now, silently skipped
// exactly as before (Plan 02 records UNSUPPORTED_EXTENSION here). Go
// source files additionally require go/build.Context.MatchFile to report
// they belong to the default build context (GOOS/GOARCH, build tags) —
// the same primitive the go toolchain itself uses — gated to Language==
// "go" only, since no other language in the registry has a build-tag
// concept; a miss here IS recorded, as an EXCLUSION_REASON_BUILD_TAG
// record (this plan's tracer path).
//
// After the walk, each language actually present is given exactly one
// chance to resolve its repo-root project descriptor (go.mod, pom.xml,
// *.csproj, ...) via LanguageSpec.Descriptor. A descriptor that is absent,
// malformed, or simply not implemented for that language does NOT fail
// DiscoverAll (D-03/T-05-Manifest) — LanguageSpec.ModuleKey is called with
// a nil descriptor and is required to degrade to a path-based identity
// rather than dropping the file.
//
// DiscoverAll's third return value (Discovery.ModulePath) remains the
// repo's Go module path specifically (as resolved by the "go"
// LanguageSpec's own descriptor, if any); it is "" when no go.mod was
// found.
func DiscoverAll(root string) (Discovery, error) {
	ctx := build.Default
	var pending []pendingFile
	var excluded []*schema.ExcludedFile

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

		// Phase 10 D-01: stat every extension-matched, non-directory
		// entry BEFORE the build-tag check — the size is needed for
		// every record kind this walk can produce (a discovered file's
		// own SizeBytes, or an excluded file's record), so computing it
		// once here, up front, avoids a second lstat at the exclusion
		// site below.
		info, err := d.Info()
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

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
				excluded = append(excluded, newExcludedFile(relPath, schema.ExclusionReason_EXCLUSION_REASON_BUILD_TAG, buildTagDetail(ctx), info.Size()))
				return nil
			}
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
		return Discovery{}, walkErr
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
	sort.Slice(excluded, func(i, j int) bool { return excluded[i].GetPath() < excluded[j].GetPath() })

	modulePath := ""
	if d, ok := descriptors["go"]; ok {
		modulePath = d.ModulePath()
	}

	return Discovery{Files: files, Excluded: excluded, ModulePath: modulePath}, nil
}
