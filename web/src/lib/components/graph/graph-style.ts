// graph-style.ts exports the Cytoscape style sheet as plain data — a
// selector-and-properties array with no cytoscape import (05-03 Task 2,
// GRF-02). GraphCanvas.svelte (05-03 Task 3) is the only file permitted
// to import cytoscape itself; this module hands it a plain array cytoscape
// accepts as its own `style` option unmodified.
//
// Deliberately minimal for this plan's tracer scope: a file-node style, a
// directory-compound style that makes the grouping legible, and a default
// edge style whose width is derived from totalCount so a heavier
// dependency reads as a heavier line. Cycle styling (GRF-04) is 05-05's
// job and must NOT be added here — this module only knows about the
// directory/file/edge distinction file-graph-transform.ts produces.
export const fileGraphStyle: unknown[] = [
	{
		selector: 'node[isDirectory = true]',
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
		selector: 'node[isDirectory = false]',
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
