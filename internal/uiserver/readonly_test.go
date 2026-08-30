package uiserver

import (
	"reflect"
	"strings"
	"testing"

	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
	"github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/uiv1connect"
)

// wantUIServiceMethods is a literal fixture of the nine method names
// SRV-03's read-only surface is asserted to be EXACTLY, transcribed from
// internal/uiproto/uiv1/ui.proto's `service UIService` block as of
// 2026-08-23 (plan 01-09, the wave that completes the nine-method
// surface). Engine.Query is DELIBERATELY absent: plan 01-08's file-level
// proto comment records the rationale ("Deliberate omission — Query vs
// Search") — Query's fuller shape (calls, called-by, the multi-def
// candidate set) is strictly subsumed by GetNodeDetail, and exposing both
// Query and Search would give the UI two overlapping name-lookup rpcs
// with no rule about which to reach for. This fixture asserts set
// equality in BOTH directions via len+membership: an ADDED method (a
// mutating verb or otherwise) fails the length check or the membership
// check, and a REMOVED method fails the membership check — neither
// direction can pass vacuously.
// UPDATED at plan 03-05: GetPermalink (D-06) is the tenth read-only rpc,
// added additively to internal/uiproto/uiv1/ui.proto's `service
// UIService` block. It is a read-only verb — it performs no network
// operation and mutates nothing — and belongs in this read set for
// exactly the same reason the original nine do.
// UPDATED at plan 04-03: GetHealth (D-01) is the eleventh read-only rpc,
// added additively to internal/uiproto/uiv1/ui.proto's `service
// UIService` block. It projects internal/query.StatusResult's richer
// per-language/per-kind/index-health data onto the wire, separately from
// GetStatus (which stays cheap for every-navigation polling, D-01). It is
// a read-only verb — it performs no network operation and mutates
// nothing — and belongs in this read set for exactly the same reason the
// other ten do.
var wantUIServiceMethods = map[string]struct{}{
	"GetStatus":     {},
	"Search":        {},
	"Files":         {},
	"Callers":       {},
	"Callees":       {},
	"Impact":        {},
	"Affected":      {},
	"GetNodeDetail": {},
	"Explore":       {},
	"GetPermalink":  {},
	"GetHealth":     {},
}

// TestUIServiceMethodSetIsExactlyTheReadSet reflects over the generated
// uiv1connect.UIServiceHandler interface — the machine-readable method
// inventory a mutating rpc would have to appear in before it could ever
// be dispatched — and asserts the observed method-name set is EXACTLY
// wantUIServiceMethods: same length (11, as of plan 04-03's GetHealth)
// AND same membership. A negative-only guard ("no method name contains a
// write verb") passes vacuously the moment its verb list stops matching
// a newly-added verb; this positive set-equality guard instead fails in
// BOTH directions (SRV-03, T-01-04).
func TestUIServiceMethodSetIsExactlyTheReadSet(t *testing.T) {
	typ := reflect.TypeOf((*uiv1connect.UIServiceHandler)(nil)).Elem()

	got := make(map[string]struct{}, typ.NumMethod())
	for i := 0; i < typ.NumMethod(); i++ {
		got[typ.Method(i).Name] = struct{}{}
	}

	if len(got) != 11 {
		t.Fatalf("uiv1connect.UIServiceHandler has %d methods, want exactly 11: %v", len(got), got)
	}
	if len(got) != len(wantUIServiceMethods) {
		t.Fatalf("observed method set size %d != fixture size %d — the fixture itself is stale", len(got), len(wantUIServiceMethods))
	}

	for name := range wantUIServiceMethods {
		if _, ok := got[name]; !ok {
			t.Fatalf("uiv1connect.UIServiceHandler is missing expected read method %q — a method was REMOVED from the service without updating this fixture", name)
		}
	}
	for name := range got {
		if _, ok := wantUIServiceMethods[name]; !ok {
			t.Fatalf("uiv1connect.UIServiceHandler declares unexpected method %q, not in the ten-name read-only fixture — a method (mutating or otherwise) was ADDED to the service without updating this fixture", name)
		}
	}
}

// mutatingVerbs is a literal fixture of write-verb prefixes/substrings
// that would signal a mutating rpc if they ever appeared in a UIService
// method name, transcribed from common REST/RPC mutation-verb convention
// as of 2026-08-23 (plan 01-09). This is the COMPLEMENTARY, negative-style
// check to TestUIServiceMethodSetIsExactlyTheReadSet's positive set
// equality — kept non-vacuous by asserting a positive count of inspected
// method names BEFORE applying the verb check, so an empty method set
// (e.g. from a reflection mistake) cannot pass this test by inspecting
// nothing.
var mutatingVerbs = []string{
	"Create", "Update", "Delete", "Remove", "Set", "Put", "Post",
	"Write", "Add", "Insert", "Mutate", "Patch", "Modify", "Sync",
	"Reindex", "Index", "Clear", "Reset", "Save",
}

// TestUIServiceDeclaresNoMutatingMethod is the complementary, negative-
// style check to TestUIServiceMethodSetIsExactlyTheReadSet, written so it
// cannot go vacuous: it asserts a POSITIVE count of method names
// inspected before applying the verb check and fails outright when that
// count is zero (rule 84d1gfpywd) — an empty or broken method-name
// enumeration must never read as "no mutating verb found".
func TestUIServiceDeclaresNoMutatingMethod(t *testing.T) {
	typ := reflect.TypeOf((*uiv1connect.UIServiceHandler)(nil)).Elem()

	inspected := 0
	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name
		inspected++
		for _, verb := range mutatingVerbs {
			if strings.Contains(name, verb) {
				t.Fatalf("method %q contains mutating verb %q — SRV-03 forbids any mutating rpc on UIService", name, verb)
			}
		}
	}

	t.Logf("inspected %d UIServiceHandler method names against %d mutating verbs", inspected, len(mutatingVerbs))
	if inspected == 0 {
		t.Fatal("inspected 0 method names — the enumeration is broken and this guard is vacuous")
	}
}

// uiProtoFieldNumber is one transcribed (message, field, number) fact
// from internal/uiproto/uiv1/ui.proto, dated at transcription time so a
// future reader can tell how stale the fixture is relative to the
// source — mirroring internal/schema/meta_commit_test.go's
// knownMetaFieldNumber convention exactly.
type uiProtoFieldNumber struct {
	message string
	field   string
	number  int32
}

// uiProtoFieldFixtureLenAtPlan0109 pins uiProtoFieldNumbers' length as a
// named, exported-in-package constant, declared immediately above the
// fixture it counts. Plans 01-10 and 01-11 EXTEND this fixture (never
// rewrite it) and each declare their own length constant in terms of
// this one — uiProtoFieldFixtureLenAtPlan0110 = uiProtoFieldFixtureLenAtPlan0109 + 9
// (01-10's nine SourceBlob source fields) and
// uiProtoFieldFixtureLenAtPlan0111 = uiProtoFieldFixtureLenAtPlan0110 + 2
// (01-11's store_exists/indexing_in_progress) — never as a bare literal,
// so the chain is checkable by the compiler and this test at every wave,
// not carried only in a prose SUMMARY across waves.
const uiProtoFieldFixtureLenAtPlan0109 = 97

// uiProtoFieldFixtureLenAtPlan0110 EXTENDS uiProtoFieldFixtureLenAtPlan0109
// by exactly 9 (plan 01-10, RPC-05, dated 2026-08-23): SourceBlob's own
// six fields (content, truncated, total_lines, total_bytes,
// returned_lines, returned_bytes) plus the three attachment fields this
// plan adds — GetNodeDetailResponse.source = 9, NodeDefinition.source = 5,
// ExploreGroup.source = 4. Declared in terms of the prior constant, never
// as a bare literal, so an edit to 01-09's fixture that changes its
// length breaks THIS assertion rather than silently shifting the
// arithmetic.
const uiProtoFieldFixtureLenAtPlan0110 = uiProtoFieldFixtureLenAtPlan0109 + 9

// uiProtoFieldFixtureLenAtPlan0111 EXTENDS uiProtoFieldFixtureLenAtPlan0110
// by exactly 2 (plan 01-11, SRV-04, dated 2026-08-23): the final two
// extension in this phase, adding GetStatusResponse.store_exists = 8 and
// .indexing_in_progress = 9 — the numbers plan 01-08 allocated and this
// plan's own D-14/D-16 degrade path finally populates. Declared in terms
// of the prior constant, never as a bare literal, mirroring plan 01-10's
// own +9 pattern.
const uiProtoFieldFixtureLenAtPlan0111 = uiProtoFieldFixtureLenAtPlan0110 + 2

// uiProtoFieldFixtureLenAtPlan0305 EXTENDS uiProtoFieldFixtureLenAtPlan0111
// by exactly 6 (plan 03-05, BRW-09/SRV-05): GetPermalinkRequest's three
// fields (path = 1, line = 2, end_line = 3) and GetPermalinkResponse's
// three (url = 1, availability = 2, reason = 3) — the tenth rpc's two new
// messages, additive from field 1 on each since both are new messages.
// Declared in terms of the prior constant, never as a bare literal,
// mirroring plan 01-10's and 01-11's own chained-extension pattern.
const uiProtoFieldFixtureLenAtPlan0305 = uiProtoFieldFixtureLenAtPlan0111 + 6

// uiProtoFieldNumbers is a literal fixture transcribed from
// internal/uiproto/uiv1/ui.proto as of 2026-08-23 (Phase 1, plan 01-09,
// the wave that completes GetNodeDetail and Explore). Per the corrected
// scope rule (cycle-3 H1): this fixture covers EVERY field of EVERY
// message that EXISTS in the generated descriptor at THIS plan's wave —
// a strictly larger set than plan 01-09's own declarations, and exactly
// the set that is resolvable, which is the property that makes this
// guard non-vacuous. It explicitly includes GetStatusRequest and
// GetStatusResponse fields 1-7 (commit_sha = 7, allocated by plan 01-08)
// and IndexingInProgress (allocated by plan 01-01), neither of which this
// plan declares.
//
// Plan 01-10 (RPC-05, dated 2026-08-23) EXTENDED this fixture with
// SourceBlob's own six fields and the three attachment fields on
// GetNodeDetailResponse, NodeDefinition and ExploreGroup — the numbers
// 01-09 recorded as intent in ui.proto's file-level comment, now spent.
//
// Plan 01-11 (SRV-04, dated 2026-08-23) is the final extender in this
// phase (append entries only): it adds GetStatusResponse.store_exists = 8
// and .indexing_in_progress = 9, the numbers plan 01-08 allocated, now
// finally declared in the descriptor and populated by 01-11's D-14/D-16
// degrade path. Every entry written by an earlier plan still resolves
// unchanged after this extension — that is the real, and only
// enforceable, cross-wave protection D-02a provides: a later plan may ADD
// a field number, it may never DISTURB one an earlier plan allocated.
var uiProtoFieldNumbers = []uiProtoFieldNumber{
	{"Node", "id", 1},
	{"Node", "kind", 2},
	{"Node", "name", 3},
	{"Node", "qualified_name", 4},
	{"Node", "file_path", 5},
	{"Node", "language", 6},
	{"Node", "start_line", 7},
	{"Node", "end_line", 8},
	{"Node", "start_col", 9},
	{"Node", "end_col", 10},
	{"Node", "signature", 11},
	{"Node", "docstring", 12},
	{"Node", "visibility", 13},
	{"Node", "is_exported", 14},
	{"Node", "return_type", 15},
	{"Location", "name", 1},
	{"Location", "kind", 2},
	{"Location", "file_path", 3},
	{"Location", "start_line", 4},
	{"GetStatusRequest", "path", 1},
	{"GetStatusResponse", "initialized", 1},
	{"GetStatusResponse", "version", 2},
	{"GetStatusResponse", "node_count", 3},
	{"GetStatusResponse", "edge_count", 4},
	{"GetStatusResponse", "file_count", 5},
	{"GetStatusResponse", "stale", 6},
	{"GetStatusResponse", "commit_sha", 7},
	{"SearchRequest", "term", 1},
	{"SearchRequest", "kind", 2},
	{"SearchRequest", "limit", 3},
	{"SearchResponse", "locations", 1},
	{"FileEntry", "path", 1},
	{"FileEntry", "language", 2},
	{"FileEntry", "node_count", 3},
	{"FileEntry", "edge_count", 4},
	{"FileTreeNode", "name", 1},
	{"FileTreeNode", "is_dir", 2},
	{"FileTreeNode", "path", 3},
	{"FileTreeNode", "language", 4},
	{"FileTreeNode", "children", 5},
	{"FilesRequest", "pattern", 1},
	{"FilesRequest", "filter", 2},
	{"FilesRequest", "dir", 3},
	{"FilesRequest", "depth", 4},
	{"FilesRequest", "format", 5},
	{"FilesResponse", "format", 1},
	{"FilesResponse", "files", 2},
	{"FilesResponse", "tree", 3},
	{"CallersRequest", "symbol", 1},
	{"CallersRequest", "limit", 2},
	{"CallersResponse", "symbol", 1},
	{"CallersResponse", "callers", 2},
	{"CalleesRequest", "symbol", 1},
	{"CalleesRequest", "limit", 2},
	{"CalleesResponse", "symbol", 1},
	{"CalleesResponse", "callees", 2},
	{"ImpactRequest", "symbol", 1},
	{"ImpactRequest", "depth", 2},
	{"ImpactResponse", "symbol", 1},
	{"ImpactResponse", "depth", 2},
	{"ImpactResponse", "node_count", 3},
	{"ImpactResponse", "edge_count", 4},
	{"ImpactResponse", "affected", 5},
	{"AffectedRequest", "files", 1},
	{"AffectedRequest", "depth", 2},
	{"AffectedResponse", "files", 1},
	{"AffectedResponse", "affected_tests", 2},
	{"GetNodeDetailRequest", "symbol", 1},
	{"GetNodeDetailRequest", "file", 2},
	{"GetNodeDetailRequest", "line", 3},
	{"NodeDefinition", "node", 1},
	{"NodeDefinition", "calls", 2},
	{"NodeDefinition", "called_by", 3},
	{"NodeDefinition", "detail_gathered", 4},
	{"GetNodeDetailResponse", "mode", 1},
	{"GetNodeDetailResponse", "path", 2},
	{"GetNodeDetailResponse", "node", 3},
	{"GetNodeDetailResponse", "calls", 4},
	{"GetNodeDetailResponse", "called_by", 5},
	{"GetNodeDetailResponse", "symbol", 6},
	{"GetNodeDetailResponse", "definitions", 7},
	{"GetNodeDetailResponse", "total_candidates", 8},
	{"ExploreRequest", "query", 1},
	{"ExploreRequest", "max_files", 2},
	{"ExploreGroup", "path", 1},
	{"ExploreGroup", "symbols", 2},
	{"ExploreGroup", "skeletonized", 3},
	{"BlastEntry", "symbol", 1},
	{"BlastEntry", "caller_count", 2},
	{"BlastEntry", "test_files", 3},
	{"ExploreResponse", "query", 1},
	{"ExploreResponse", "empty", 2},
	{"ExploreResponse", "stale", 3},
	{"ExploreResponse", "symbol_count", 4},
	{"ExploreResponse", "groups", 5},
	{"ExploreResponse", "blasts", 6},
	{"IndexingInProgress", "message", 1},

	// Plan 01-10 (RPC-05, dated 2026-08-23): SourceBlob's own six fields,
	// plus the three attachment fields on messages plan 01-09 already
	// declared — the numbers 01-09 recorded as intent in ui.proto's
	// file-level comment, spent here. Nine entries total, matching
	// uiProtoFieldFixtureLenAtPlan0110's +9.
	{"SourceBlob", "content", 1},
	{"SourceBlob", "truncated", 2},
	{"SourceBlob", "total_lines", 3},
	{"SourceBlob", "total_bytes", 4},
	{"SourceBlob", "returned_lines", 5},
	{"SourceBlob", "returned_bytes", 6},
	{"NodeDefinition", "source", 5},
	{"GetNodeDetailResponse", "source", 9},
	{"ExploreGroup", "source", 4},

	// Plan 01-11 (SRV-04, dated 2026-08-23): the two fields plan 01-08
	// allocated and this plan finally declares and populates (D-14/D-16).
	// Two entries total, matching uiProtoFieldFixtureLenAtPlan0111's +2.
	{"GetStatusResponse", "store_exists", 8},
	{"GetStatusResponse", "indexing_in_progress", 9},

	// Plan 03-05 (BRW-09/SRV-05): GetPermalink's two new messages, the
	// tenth rpc. Six entries total, matching
	// uiProtoFieldFixtureLenAtPlan0305's +6.
	{"GetPermalinkRequest", "path", 1},
	{"GetPermalinkRequest", "line", 2},
	{"GetPermalinkRequest", "end_line", 3},
	{"GetPermalinkResponse", "url", 1},
	{"GetPermalinkResponse", "availability", 2},
	{"GetPermalinkResponse", "reason", 3},
}

// TestUIProtoFieldNumbersAreStableAndUnique replaces a contiguity
// assertion (the wrong property: additive-only evolution with `reserved`
// gaps legitimately produces non-contiguous numbers) with the two
// properties D-02a actually protects, read from the GENERATED
// DESCRIPTOR — never a source grep, which could not tell a real field
// from a comment (mirroring internal/schema/meta_commit_test.go's
// TestKnownMetaFieldNumbersAreStable for the UI surface):
//
//  1. Stability: every uiProtoFieldNumbers entry still maps to its
//     expected number in the generated descriptor. A renumber between
//     revisions fails; a newly ADDED field (not in the fixture) passes,
//     as it should.
//  2. Uniqueness: no field number appears twice within any one message
//     (protoc already rejects this outright; this is a cheap
//     belt-and-braces re-assertion over the generated descriptor).
//
// Both directions of resolution are checked, never skipped: every
// fixture entry must resolve to a real declared field (an unresolved
// entry FAILS, named, rather than being silently skipped), AND every
// field the descriptor actually declares must be covered by some fixture
// entry (an uncovered field FAILS, named) — this is what "the fixture
// covers every field of every message that exists at this wave" means in
// an executable form, not merely an assertion in prose.
func TestUIProtoFieldNumbersAreStableAndUnique(t *testing.T) {
	if len(uiProtoFieldNumbers) != uiProtoFieldFixtureLenAtPlan0305 {
		t.Fatalf("len(uiProtoFieldNumbers) = %d, want uiProtoFieldFixtureLenAtPlan0305 (%d) — the fixture and its pinned length constant have drifted apart", len(uiProtoFieldNumbers), uiProtoFieldFixtureLenAtPlan0305)
	}
	if len(uiProtoFieldNumbers) == 0 {
		t.Fatal("uiProtoFieldNumbers is empty — this guard is vacuous")
	}

	fd := (&uiv1.GetStatusRequest{}).ProtoReflect().Descriptor().ParentFile()
	msgs := fd.Messages()

	// byMsgField is every "Message.field" -> number the descriptor
	// ACTUALLY declares, built while checking per-message uniqueness
	// (property 2) in the same pass.
	byMsgField := make(map[string]int32)
	fieldsInspected := 0
	for i := 0; i < msgs.Len(); i++ {
		md := msgs.Get(i)
		fields := md.Fields()
		seenInMessage := make(map[int32]string, fields.Len())
		for j := 0; j < fields.Len(); j++ {
			fdesc := fields.Get(j)
			num := int32(fdesc.Number())
			if existing, dup := seenInMessage[num]; dup {
				t.Fatalf("message %q: field number %d is shared by both %q and %q — D-02a forbids two fields colliding on one number", md.Name(), num, existing, fdesc.Name())
			}
			seenInMessage[num] = string(fdesc.Name())
			byMsgField[string(md.Name())+"."+string(fdesc.Name())] = num
			fieldsInspected++
		}
	}

	messagesInspected := msgs.Len()
	t.Logf("inspected %d messages and %d fields in the generated uiv1 descriptor", messagesInspected, fieldsInspected)
	if messagesInspected == 0 || fieldsInspected == 0 {
		t.Fatal("inspected 0 messages or 0 fields — the descriptor enumeration is broken and this guard is vacuous")
	}

	// Direction 1: every fixture entry resolves to a real declared field
	// at its expected number. Named failure, never a silent skip.
	resolved := 0
	for _, known := range uiProtoFieldNumbers {
		key := known.message + "." + known.field
		gotNum, present := byMsgField[key]
		if !present {
			t.Fatalf("known field %s (number %d) is missing from the generated descriptor entirely", key, known.number)
		}
		if gotNum != known.number {
			t.Fatalf("field %s is numbered %d in the generated descriptor, want %d — this is exactly the renumber D-02a forbids", key, gotNum, known.number)
		}
		resolved++
	}
	if resolved != len(uiProtoFieldNumbers) {
		t.Fatalf("resolved %d of %d fixture entries — every entry must resolve", resolved, len(uiProtoFieldNumbers))
	}
	if resolved == 0 {
		t.Fatal("resolved 0 fixture entries — this guard is vacuous")
	}

	// Direction 2: every field the descriptor actually declares is
	// covered by some fixture entry. This is what makes the fixture's
	// SCOPE non-vacuous (cycle-3 H1) — a fixture scoped only to this
	// plan's own declarations would leave GetStatusResponse pinned by
	// nothing, exactly the defect this direction catches.
	fixtureKeys := make(map[string]struct{}, len(uiProtoFieldNumbers))
	for _, known := range uiProtoFieldNumbers {
		fixtureKeys[known.message+"."+known.field] = struct{}{}
	}
	for key := range byMsgField {
		if _, ok := fixtureKeys[key]; !ok {
			t.Fatalf("descriptor declares field %s, but no fixture entry covers it — the fixture must cover every field of every message that exists at this wave", key)
		}
	}
}
