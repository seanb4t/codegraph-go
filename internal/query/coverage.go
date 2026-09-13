package query

import (
	"encoding/base64"
	"strings"
	"unicode/utf8"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// This file is the D-14 target of a file-scoped source scan
// (TestCoverageSourceNeverWalksDisk): every count and every row below is
// READ BACK through graphstore.Reader's IterateFiles/IterateExcludedFiles
// — never reconstructed by a filesystem walk at query time. A disk-scan
// call here would let a query-time re-walk silently disagree with what
// Discover actually recorded (D-14/T-10-07); it must never appear.
//
// CoverageSummary/CoverageRows read from whatever snapshot the caller's
// Engine holds; two consecutive CoverageRows page calls against a live
// server may be answered from different snapshots (each request opens
// its own Engine, SRV-04), so paging is a best-effort walk of a
// changing graph, not a single consistent cursor across calls.

// CoverageSummary is the discovered-vs-indexed denominator (Phase 10
// HLT-05): Known is false for a graph that predates this phase's
// Meta.has_coverage field (D-06/D-15) — never 0/0, never an error.
// Discovered == Indexed + ExtractionFailed + (Excluded minus
// directory-level records) — asserted as an invariant by callers, never
// hard-coded, since later plans add more exclusion reasons to the same
// walk (D-01).
type CoverageSummary struct {
	Known bool

	Discovered       int64
	Indexed          int64
	Excluded         int64
	ExtractionFailed int64

	// ExcludedByReason keys are schema.ExclusionReason.String()'s full
	// generated names (e.g. "EXCLUSION_REASON_BUILD_TAG") — the
	// generated TS client resolves numbers to these same full names via
	// ExclusionReasonSchema.values, so full names are the easiest to
	// display and cannot collide (D-09 discretion).
	ExcludedByReason map[string]int64
}

// CoverageRowKind distinguishes a CoverageRow's origin: an ExcludedFile
// record or a File record with a non-empty Errors list. Numbered to
// match uiv1.CoverageRowKind exactly (0 reserved/unspecified) so
// internal/uiserver's mapper is a plain numeric cast.
type CoverageRowKind uint8

const (
	_ CoverageRowKind = iota
	// CoverageRowExcluded is a pre-extraction exclusion (an ExcludedFile
	// record).
	CoverageRowExcluded
	// CoverageRowExtractionFailed is a File record with a non-empty
	// Errors list.
	CoverageRowExtractionFailed
)

// CoverageRow is one per-file (or per-directory) coverage gap.
type CoverageRow struct {
	Path   string
	Kind   CoverageRowKind
	Reason schema.ExclusionReason // zero value (UNSPECIFIED) for CoverageRowExtractionFailed
	Detail string
}

// CoverageRowsOptions configures one CoverageRows page request.
type CoverageRowsOptions struct {
	PageSize  int
	PageToken string
	// Reason, when non-UNSPECIFIED, restricts rows to ExcludedFile
	// records with exactly this reason — extraction-failed rows are
	// never returned when a reason filter is active.
	Reason schema.ExclusionReason
}

// CoveragePage is one CoverageRows result page.
type CoveragePage struct {
	Known         bool
	Rows          []CoverageRow
	NextPageToken string
}

// CoverageDefaultPageSize and CoverageMaxPageSize bound CoverageRows'
// PageSize the same way every other paged rpc in this codebase clamps a
// caller-supplied size (T-10-03's transport-cap lesson).
const (
	CoverageDefaultPageSize = 200
	CoverageMaxPageSize     = 1000

	// coverageDetailMaxBytes bounds an extraction-failure detail string
	// (T-10-05): an absolute host path replaced by "." still leaves the
	// rest of a long parser error message, which is cut here on a rune
	// boundary rather than left unbounded.
	coverageDetailMaxBytes = 256
)

// CoverageSummary reports the discovered-vs-indexed denominator for the
// graph e reads (Phase 10 HLT-05). A graph whose Meta lacks has_coverage
// (a pre-Phase-10 graph, or a store with no Meta record at all) returns
// CoverageSummary{Known:false} with every count zero and a nil
// ExcludedByReason map — the D-06/D-15 unknown short-circuit runs BEFORE
// any scan.
func (e *Engine) CoverageSummary() (CoverageSummary, error) {
	meta, err := e.IndexMeta()
	if err != nil {
		return CoverageSummary{}, err
	}
	if meta == nil || !meta.GetHasCoverage() {
		return CoverageSummary{Known: false}, nil
	}

	fit, err := e.reader.IterateFiles()
	if err != nil {
		return CoverageSummary{}, err
	}
	var indexed, extractionFailed int64
	for fit.Next() {
		if len(fit.File().GetErrors()) == 0 {
			indexed++
		} else {
			extractionFailed++
		}
	}
	ferr := fit.Err()
	fit.Close()
	if ferr != nil {
		return CoverageSummary{}, ferr
	}

	xit, err := e.reader.IterateExcludedFiles()
	if err != nil {
		return CoverageSummary{}, err
	}
	var excludedTotal, fileLevel int64
	byReason := make(map[string]int64)
	for xit.Next() {
		x := xit.ExcludedFile()
		excludedTotal++
		byReason[x.GetReason().String()]++
		if !schema.IsDirectoryExclusion(x.GetReason()) {
			fileLevel++
		}
	}
	xerr := xit.Err()
	xit.Close()
	if xerr != nil {
		return CoverageSummary{}, xerr
	}

	return CoverageSummary{
		Known:            true,
		Discovered:       indexed + extractionFailed + fileLevel,
		Indexed:          indexed,
		Excluded:         excludedTotal,
		ExtractionFailed: extractionFailed,
		ExcludedByReason: byReason,
	}, nil
}

// CoverageRows returns one page of per-file coverage-gap rows: every
// extraction-failed File record (segment 'f'), then every ExcludedFile
// record (segment 'x', optionally filtered by opts.Reason), each segment
// walked in the store's own key order. A graph with no recorded coverage
// returns CoveragePage{Known:false} (the same D-06/D-15 contract as
// CoverageSummary).
func (e *Engine) CoverageRows(opts CoverageRowsOptions) (CoveragePage, error) {
	meta, err := e.IndexMeta()
	if err != nil {
		return CoveragePage{}, err
	}
	if meta == nil || !meta.GetHasCoverage() {
		return CoveragePage{Known: false}, nil
	}

	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = CoverageDefaultPageSize
	}
	if pageSize > CoverageMaxPageSize {
		pageSize = CoverageMaxPageSize
	}

	cursorSeg, cursorPath, err := decodeCoverageToken(opts.PageToken)
	if err != nil {
		return CoveragePage{}, err
	}

	filterByReason := opts.Reason != schema.ExclusionReason_EXCLUSION_REASON_UNSPECIFIED
	if filterByReason && cursorSeg == 'f' {
		return CoveragePage{}, invalidArgumentf("coverage: page token from the extraction-failed segment is invalid when filtering by reason")
	}

	var rows []CoverageRow

	if !filterByReason {
		fit, err := e.reader.IterateFiles()
		if err != nil {
			return CoveragePage{}, err
		}
		for len(rows) <= pageSize && fit.Next() {
			f := fit.File()
			if len(f.GetErrors()) == 0 {
				continue
			}
			path := f.GetPath()
			if cursorSeg == 'f' && path <= cursorPath {
				continue
			}
			rows = append(rows, CoverageRow{
				Path:   path,
				Kind:   CoverageRowExtractionFailed,
				Detail: coverageExtractionDetail(f, e.repoRoot),
			})
		}
		ferr := fit.Err()
		fit.Close()
		if ferr != nil {
			return CoveragePage{}, ferr
		}
	}

	if len(rows) <= pageSize {
		xit, err := e.reader.IterateExcludedFiles()
		if err != nil {
			return CoveragePage{}, err
		}
		for len(rows) <= pageSize && xit.Next() {
			x := xit.ExcludedFile()
			if filterByReason && x.GetReason() != opts.Reason {
				continue
			}
			path := x.GetPath()
			if cursorSeg == 'x' && path <= cursorPath {
				continue
			}
			rows = append(rows, CoverageRow{
				Path:   path,
				Kind:   CoverageRowExcluded,
				Reason: x.GetReason(),
				Detail: x.GetDetail(),
			})
		}
		xerr := xit.Err()
		xit.Close()
		if xerr != nil {
			return CoveragePage{}, xerr
		}
	}

	var nextToken string
	if len(rows) > pageSize {
		rows = rows[:pageSize]
		last := rows[pageSize-1]
		seg := byte('f')
		if last.Kind == CoverageRowExcluded {
			seg = 'x'
		}
		nextToken = encodeCoverageToken(seg, last.Path)
	}

	return CoveragePage{Known: true, Rows: rows, NextPageToken: nextToken}, nil
}

// coverageExtractionDetail joins f's Errors into one display string,
// replacing repoRoot (when non-empty) with "." (T-10-05: never leak the
// absolute host checkout path onto the wire) and cutting the result at
// coverageDetailMaxBytes on a rune boundary.
func coverageExtractionDetail(f *schema.File, repoRoot string) string {
	detail := strings.Join(f.GetErrors(), "; ")
	if repoRoot != "" {
		detail = strings.ReplaceAll(detail, repoRoot, ".")
	}
	return truncateAtRuneBoundary(detail, coverageDetailMaxBytes)
}

// truncateAtRuneBoundary cuts s to at most maxBytes bytes, never splitting
// a multi-byte rune.
func truncateAtRuneBoundary(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	b := s[:maxBytes]
	for len(b) > 0 {
		r, size := utf8.DecodeLastRuneInString(b)
		if r != utf8.RuneError || size != 1 {
			break
		}
		b = b[:len(b)-1]
	}
	return b
}

// encodeCoverageToken frames a resume cursor as [kind byte][path bytes],
// base64url-encoded. The token is OPAQUE to the caller and is used ONLY
// as an in-memory string comparison (T-10-04) — never as a Pebble bound,
// never as a filesystem path.
func encodeCoverageToken(kind byte, path string) string {
	buf := make([]byte, 0, 1+len(path))
	buf = append(buf, kind)
	buf = append(buf, path...)
	return base64.RawURLEncoding.EncodeToString(buf)
}

// decodeCoverageToken decodes a token produced by encodeCoverageToken. An
// empty token means "first page" (kind 0, path ""). Any malformed value —
// bad base64, empty payload, an unrecognized kind byte, invalid UTF-8, or
// a path over 4096 bytes — is rejected as ErrInvalidArgument (T-10-04);
// never a filesystem path, never a Pebble bound.
func decodeCoverageToken(tok string) (kind byte, path string, err error) {
	if tok == "" {
		return 0, "", nil
	}
	data, decErr := base64.RawURLEncoding.DecodeString(tok)
	if decErr != nil || len(data) < 1 {
		return 0, "", invalidArgumentf("coverage: malformed page token")
	}
	kind = data[0]
	if kind != 'f' && kind != 'x' {
		return 0, "", invalidArgumentf("coverage: malformed page token")
	}
	path = string(data[1:])
	if !utf8.ValidString(path) || len(path) > 4096 {
		return 0, "", invalidArgumentf("coverage: malformed page token")
	}
	return kind, path, nil
}
