// community-palette.ts is the presentation vocabulary for the wire
// `communityId` (FileGraphNode.communityId, GRF-06/GRF-08 — server-computed
// deterministic Louvain community assignment). This module MUST NOT compute
// community membership of any kind — the client never derives a grouping
// that the server did not already send (the D-06/cycleId discipline of
// Phase 5, restated for community_id in this phase's 11-CONTEXT.md D-10).
// It only maps an already-computed integer onto a fixed, literal colour.
//
// Twelve literal hex values, never CSS custom properties: cytoscape.js does
// not resolve `var()` in a style value (the same constraint graph-style.ts's
// header documents for the cycle/symbol colours). Index = (id - 1) % 12,
// cycling beyond twelve (D-10) — communityPaletteIndex below is the ONE
// place that arithmetic lives; callers pass ids >= 1 (a fresh-compute
// community id is never 0, per D-03).
//
// Palette provenance: the dataviz skill's reference categorical palette's
// eight slots (blue, orange, aqua, yellow, magenta, green, violet, red)
// extended by four more hues chosen for maximum separation from those eight
// and from each other — teal, brown, purple, lime. Validated at plan time
// with the dataviz skill's `validate_palette.js --mode light`: lightness
// band PASS, chroma floor PASS, adjacent-pair CVD separation PASS (worst
// deltaE 9.1 against an 8-floor), normal-vision floor PASS (worst 19.6
// against a 15-floor); contrast WARN on four hues (`#1baf7a`, `#eda100`,
// `#e87ba4`, `#86a800`) against the canvas background, relieved by the file
// label every file node already renders beneath it (D-10) — the colour is
// never the ONLY encoding of which file belongs to which community. All-
// pairs separation across all twelve hues simultaneously is not achievable
// at this count and is not claimed here; position (directory grouping) and
// the label are this rendering's secondary encodings, exactly as the cycle
// styling already relies on a border in addition to colour.
// prettier-ignore
export const COMMUNITY_PALETTE: readonly string[] = ['#2a78d6', '#eb6834', '#1baf7a', '#eda100', '#e87ba4', '#008300', '#4a3aa7', '#e34948', '#12a0b0', '#b06a1c', '#a259d9', '#86a800'];

export const COMMUNITY_CLASS_PREFIX = 'graph-community-';

// communityPaletteIndex maps a 1-based, canonical community id onto its
// palette slot, cycling beyond COMMUNITY_PALETTE.length (D-10). Callers
// pass ids >= 1 — id 0 ("not computed") is never rendered as a colour and
// must be filtered out by the caller before reaching this function.
export function communityPaletteIndex(communityId: number): number {
	return (communityId - 1) % COMMUNITY_PALETTE.length;
}

export function communityDiscriminatorClass(communityId: number): string {
	return `${COMMUNITY_CLASS_PREFIX}${communityPaletteIndex(communityId)}`;
}
