package uiserver

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/schema"
	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
)

// coverageToProto maps internal/query.CoverageSummary onto
// uiv1.Coverage field-for-field, mirroring healthToProto's convention —
// a named mapper, never an inline literal at the handler call site.
func coverageToProto(s query.CoverageSummary) *uiv1.Coverage {
	return &uiv1.Coverage{
		Known:            s.Known,
		Discovered:       s.Discovered,
		Indexed:          s.Indexed,
		Excluded:         s.Excluded,
		ExtractionFailed: s.ExtractionFailed,
		ExcludedByReason: s.ExcludedByReason,
	}
}

// exclusionReasonToProto maps internal/schema.ExclusionReason onto
// uiv1.ExclusionReason as a plain numeric cast — TestExclusionReasonEnumsAgree
// pins the two enums' (name, number) sets to each other in both
// directions, so this cast cannot silently skew (D-08).
func exclusionReasonToProto(r schema.ExclusionReason) uiv1.ExclusionReason {
	return uiv1.ExclusionReason(r)
}

// coverageRowKindToProto maps internal/query.CoverageRowKind onto
// uiv1.CoverageRowKind as a plain numeric cast — both enums are declared
// with the same 0/1/2 numbering (query.CoverageRowKind's own doc comment
// records this).
func coverageRowKindToProto(k query.CoverageRowKind) uiv1.CoverageRowKind {
	return uiv1.CoverageRowKind(k)
}

// coverageRowToProto maps one internal/query.CoverageRow onto its wire
// projection.
func coverageRowToProto(r query.CoverageRow) *uiv1.CoverageRow {
	return &uiv1.CoverageRow{
		Path:   r.Path,
		Kind:   coverageRowKindToProto(r.Kind),
		Reason: exclusionReasonToProto(r.Reason),
		Detail: r.Detail,
	}
}

// errCoverageUnimplemented is plan 10-01's RED-phase placeholder — GREEN
// replaces this handler body with the real eng.CoverageRows call.
var errCoverageUnimplemented = errors.New("uiserver: GetCoverage is not yet implemented (plan 10-01 RED phase)")

// GetCoverage pages the per-file coverage-gap row list (Phase 10
// HLT-05/HLT-06, D-10) that GetHealthResponse.coverage deliberately
// omits to stay bounded on a polled call. Uses the ORDINARY withEngine
// shape (Callers/Callees/Files/GetHealth/FileGraph/FileSymbols'
// convention) — not GetStatus's degrade path.
func (s *uiService) GetCoverage(ctx context.Context, req *connect.Request[uiv1.GetCoverageRequest]) (*connect.Response[uiv1.GetCoverageResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errCoverageUnimplemented)
}
