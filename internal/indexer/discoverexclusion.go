package indexer

import (
	"go/build"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/parser"
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

// dirExclusionReason classifies a pruned directory name into a Phase 10
// D-02 exclusion reason. It re-inspects exactly the two branches
// ShouldSkipDir ORs together — name == "vendor" or a dot-prefixed name —
// and MUST NOT call ShouldSkipDir itself: discoverexclusion_test.go's
// TestDirExclusionReasonAgreesWithShouldSkipDir is the guard that pins the
// two never diverging, and that guard would be a tautology if this
// function simply delegated to the predicate it is supposed to agree
// with. The walk callback below keeps calling ShouldSkipDir as the actual
// prune authority (shared verbatim with the fsnotify watcher, Phase 4
// D-04) — this helper only decides which reason a prune gets. ok is false
// for any name neither branch covers (the directory is not pruned at
// all).
func dirExclusionReason(name string) (schema.ExclusionReason, bool) {
	switch {
	case name == "vendor":
		return schema.ExclusionReason_EXCLUSION_REASON_DIR_VENDOR, true
	case strings.HasPrefix(name, "."):
		return schema.ExclusionReason_EXCLUSION_REASON_DIR_DOTPREFIX, true
	default:
		return schema.ExclusionReason_EXCLUSION_REASON_UNSPECIFIED, false
	}
}

// exceedsSizeLimit reports whether sizeBytes is strictly greater than
// parser.MaxSourceBytes (Phase 10 D-04) — the pre-extraction, stat-only
// size pre-check. Strict '>': a file of exactly MaxSourceBytes is NOT
// excluded by this check (HLT-05's boundary); parser.ErrSourceTooLarge
// remains the backstop for a file that grows between this stat and
// Extract's later read.
func exceedsSizeLimit(sizeBytes int64) bool {
	return sizeBytes > parser.MaxSourceBytes
}

// unsupportedExtensionDetail renders the detail field for an
// EXCLUSION_REASON_UNSUPPORTED_EXTENSION record. ext is the already
// lower-cased extension the walk computed for the registry lookup
// (decision point 2) — this function never re-derives it. An empty
// extension (an extensionless file that still missed the registry — not
// reachable today since every registered extension is non-empty, but
// guarded defensively rather than emitting a blank detail) renders as
// "(none)".
func unsupportedExtensionDetail(ext string) string {
	if ext == "" {
		return "(none)"
	}
	return ext
}

// sizeLimitDetail renders the detail field for an
// EXCLUSION_REASON_SIZE_LIMIT record — the byte count that tripped the
// pre-check, alongside the ceiling it exceeded, for a human reading the
// coverage page without needing to cross-reference parser.MaxSourceBytes
// separately.
func sizeLimitDetail(sizeBytes int64) string {
	return strconv.FormatInt(sizeBytes, 10) + " bytes > " + strconv.Itoa(parser.MaxSourceBytes)
}

// DiscoverAll walks root and returns every file whose extension is
// claimed by a registered LanguageSpec (D-03), sorted by RelPath in
// ascending byte order, PLUS one ExcludedFile record per path or pruned
// directory the walker visited but did not index — the same walk result
// feeds both (Phase 10 D-01), so the discovered-count denominator and the
// per-file reason list can never disagree. This stable order is
// determinism's first line of defense: the same input tree always yields
// the same output order, regardless of filesystem walk order.
//
// Four decision points in the walk each record their own exclusion
// reason, in this order: (1) a pruned directory — vendor/ or any
// dot-prefixed directory (.git, .codegraph, etc.), via ShouldSkipDir,
// shared verbatim with the Phase-4 watcher — gets ONE directory-level
// record (DIR_VENDOR / DIR_DOTPREFIX, D-02); its contents are never
// visited, so they contribute nothing to the discovered count. (2) a file
// whose extension is not registered in the extension->language registry
// (languages.go) gets an UNSUPPORTED_EXTENSION record (D-03). (3) a Go
// source file that go/build.Context.MatchFile reports does not belong to
// the default build context (GOOS/GOARCH, build tags) — the same
// primitive the go toolchain itself uses, gated to Language=="go" only
// since no other registered language has a build-tag concept — gets a
// BUILD_TAG record. (4) a file whose stat size is strictly greater than
// parser.MaxSourceBytes gets a SIZE_LIMIT record (D-04); its bytes are
// never read at discovery time — parser.ErrSourceTooLarge remains the
// backstop for a file that grows between this stat and Extract's later
// read.
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
			// Decision point 1: ShouldSkipDir remains the single prune
			// authority (shared verbatim with the fsnotify watcher,
			// Phase 4 D-04) — dirExclusionReason only classifies WHICH
			// reason a prune gets, never re-decides the prune itself.
			if p != root && ShouldSkipDir(d.Name()) {
				relPath, relErr := filepath.Rel(root, p)
				if relErr != nil {
					return relErr
				}
				relPath = filepath.ToSlash(relPath)
				if reason, ok := dirExclusionReason(d.Name()); ok {
					excluded = append(excluded, newExcludedFile(relPath, reason, d.Name(), 0))
				}
				return fs.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		// Phase 10 D-01: stat every regular file the walker visits BEFORE
		// the extension/build-tag/size checks — the size is needed for
		// every record kind this walk can produce (a discovered file's
		// own SizeBytes, or any excluded-file record), so computing it
		// once here, up front, avoids a second lstat at any exclusion
		// site below.
		info, err := d.Info()
		if err != nil {
			return err
		}

		// Decision point 2.
		ext := strings.ToLower(filepath.Ext(d.Name()))
		spec, ok := lookupLanguageByExt(ext)
		if !ok {
			excluded = append(excluded, newExcludedFile(relPath, schema.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION, unsupportedExtensionDetail(ext), info.Size()))
			return nil
		}

		abs, err := filepath.Abs(p)
		if err != nil {
			return err
		}

		// Decision point 3.
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

		// Decision point 4 (NEW, D-04): the bytes are never read here —
		// only the stat size already computed above.
		if exceedsSizeLimit(info.Size()) {
			excluded = append(excluded, newExcludedFile(relPath, schema.ExclusionReason_EXCLUSION_REASON_SIZE_LIMIT, sizeLimitDetail(info.Size()), info.Size()))
			return nil
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
