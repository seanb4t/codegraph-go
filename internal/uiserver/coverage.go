package uiserver

import (
	"context"

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

// coveragePageToProto maps internal/query.CoveragePage onto
// uiv1.GetCoverageResponse field-for-field.
func coveragePageToProto(p query.CoveragePage) *uiv1.GetCoverageResponse {
	rows := make([]*uiv1.CoverageRow, len(p.Rows))
	for i, r := range p.Rows {
		rows[i] = coverageRowToProto(r)
	}
	return &uiv1.GetCoverageResponse{
		Rows:          rows,
		NextPageToken: p.NextPageToken,
		Known:         p.Known,
	}
}

// GetCoverage pages the per-file coverage-gap row list (Phase 10
// HLT-05/HLT-06, D-10) that GetHealthResponse.coverage deliberately
// omits to stay bounded on a polled call. Uses the ORDINARY withEngine
// shape (Callers/Callees/Files/GetHealth/FileGraph/FileSymbols'
// convention) — not GetStatus's degrade path. A malformed page token
// surfaces as query.ErrInvalidArgument -> CodeInvalidArgument through
// the shared mapEngineError translation, unwrapped from inside the
// closure exactly as FileSymbols/GetNodeDetail/GetPermalink do it —
// no second mapping is added here.
func (s *uiService) GetCoverage(ctx context.Context, req *connect.Request[uiv1.GetCoverageRequest]) (*connect.Response[uiv1.GetCoverageResponse], error) {
	var resp *uiv1.GetCoverageResponse
	err := withEngine(ctx, s.repoPath, func(eng *query.Engine) error {
		page, err := eng.CoverageRows(query.CoverageRowsOptions{
			PageSize:  int(req.Msg.GetPageSize()),
			PageToken: req.Msg.GetPageToken(),
			Reason:    schema.ExclusionReason(req.Msg.GetReason()),
		})
		if err != nil {
			return err
		}
		resp = coveragePageToProto(page)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}
