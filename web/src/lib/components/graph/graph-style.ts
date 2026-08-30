// graph-style.ts exports the Cytoscape style sheet as plain data — a
// selector-and-properties array with no cytoscape import. GraphCanvas.svelte
// is the only file permitted to import cytoscape itself; this module hands
// it a plain array cytoscape accepts as its own `style` option unmodified.
//
// Deliberately minimal for this tracer's scope: a file-node style, a
// directory-compound style that makes the grouping legible, and a default
// edge style whose width is derived from totalCount so a heavier
// dependency reads as a heavier line. Cycle styling is a later
// concern and must NOT be added here — this module only knows about the
// directory/file/edge distinction file-graph-transform.ts produces.
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
			'font-size': 10,
			color: '#374151',
			padding: '12px'
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
	}
];
