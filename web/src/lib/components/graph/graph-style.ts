// graph-style.ts exports the Cytoscape style sheet as plain data — a
// selector-and-properties array with no cytoscape import. GraphCanvas.svelte
// is the only file permitted to import cytoscape itself; this module hands
// it a plain array cytoscape accepts as its own `style` option unmodified.
//
// A file-node style, a directory-compound style that makes the grouping
// legible, a COLLAPSED-directory style that distinguishes a directory
// rendered as a single collapsed node from one
// whose files are showing, and a default edge style whose width is
// derived from totalCount so a heavier dependency reads as a heavier
// line.
//
// The cycle selectors at the bottom match `.graph-cycle`, the class
// file-graph-transform.ts attaches to any node or edge the wire already
// marked as a cycle member (a file's non-zero cycleId, a collapsed
// directory's non-empty cycleIds union, or an edge's inCycle flag). They
// are placed LAST in this array — see the selector-order note below —
// and distinguish a cycle member on two independent channels (a dashed
// border/line plus colour), never colour alone, so the distinction still
// reads for someone who cannot see the colour difference. cytoscape.js
// does not resolve CSS custom properties in a style value (confirmed
// against its own property-type documentation: colours are name/hex/RGB/
// HSL only) — the colour below is a literal hex value chosen to MATCH
// this application's own `--destructive` design token (Tailwind's
// red-600, oklch(0.577 0.245 27.325) ≈ #dc2626), not a `var()` reference
// this renderer cannot read.
//
// Selector ORDER matters: the collapsed-directory selector is placed
// AFTER the general directory selector. A collapsed directory node
// matches BOTH selectors (isDirectory is true on both), and cytoscape's
// cascade — like CSS — lets the LATER rule win for any property both
// selectors set. Putting the specific rule after the general one is what
// makes the distinguishing properties below actually apply.
//
// Boolean data selectors use cytoscape's truthy/falsy existence syntax
// (`[?field]` / `[!field]`), NOT a `[field = true]` equality comparison —
// confirmed live against a real browser this task: `[isDirectory = true]`
// logs "The selector ... is invalid" to the console and the rule never
// matches, silently leaving every node under cytoscape's default style.
export const fileGraphStyle: unknown[] = [
	{
		selector: 'node[?isDirectory]',
		style: {
			shape: 'round-rectangle',
			'background-color': '#e5e7eb',
			'background-opacity': 0.5,
			'border-width': 1,
			'border-color': '#9ca3af',
			label: 'data(label)',
			'text-valign': 'top',
			'text-halign': 'center',
			'text-events': 'yes',
			'font-size': 10,
			color: '#374151',
			padding: '12px'
		}
	},
	{
		// Matches a directory node that ALSO carries collapsed:true —
		// i.e. a directory rendered as a single node with its files not
		// showing. Distinguished from the general directory style above
		// on non-colour properties (border-style, text-wrap, explicit
		// width/height) so a collapsed directory is visually
		// distinguishable at a glance and large enough that a click at
		// its own centre lands on it.
		selector: 'node[?isDirectory][?collapsed]',
		style: {
			shape: 'round-rectangle',
			'background-color': '#dbeafe',
			'background-opacity': 0.7,
			'border-width': 2,
			'border-style': 'dashed',
			'border-color': '#60a5fa',
			label: 'data(label)',
			'text-valign': 'center',
			'text-halign': 'center',
			'text-wrap': 'wrap',
			'text-max-width': '140px',
			'font-size': 10,
			color: '#1e3a8a',
			width: 140,
			height: 60
		}
	},
	{
		selector: 'node[!isDirectory]',
		style: {
			shape: 'ellipse',
			width: 16,
			height: 16,
			'background-color': '#2563eb',
			label: 'data(label)',
			'font-size': 8,
			color: '#111827',
			'text-valign': 'bottom',
			'text-margin-y': 4
		}
	},
	{
		selector: 'edge',
		style: {
			width: 'mapData(totalCount, 1, 20, 1, 6)',
			'line-color': '#94a3b8',
			'target-arrow-color': '#94a3b8',
			'target-arrow-shape': 'triangle',
			'curve-style': 'bezier',
			opacity: 0.6
		}
	},
	{
		// Matches ANY node carrying the cycle class — a file node whose
		// wire cycleId is non-zero, or a collapsed directory whose union
		// of member cycleIds is non-empty. Placed after every other node
		// selector so its border overrides whichever base rule
		// (file/directory/collapsed-directory) already matched, per the
		// selector-order note above.
		selector: 'node.graph-cycle',
		style: {
			'border-width': 4,
			'border-style': 'dashed',
			'border-color': '#dc2626'
		}
	},
	{
		// Matches an edge the server marked in-cycle. line-style and
		// target-arrow-shape are the non-colour channels; width is left
		// alone so the count-driven mapData sizing above still applies.
		selector: 'edge.graph-cycle',
		style: {
			'line-style': 'dashed',
			'line-color': '#dc2626',
			'target-arrow-color': '#dc2626',
			'target-arrow-shape': 'diamond'
		}
	}
];
