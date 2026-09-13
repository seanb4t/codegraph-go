#!/usr/bin/env node
// check-no-force-layout.mjs — GRF-06's "no force-directed layout mode is
// reachable from any code path" proof (D-12b), by static scan.
//
// This file does NOT run the SPA, does NOT launch a browser, and does NOT
// parse an AST — it is a plain regex scan SCOPED TO LAYOUT-NAME POSITIONS
// (a cytoscape `name: '<x>'` option literal, or a `cytoscape-<x>` import/
// dependency specifier), never a bare word search across arbitrary text.
// An empty scan, or a scan that cannot even see the required `elk`
// positive control, is a FAIL — never a pass. This is the same discipline
// graph-verdict.mjs's header states for its own comparator: the most
// likely way this script returns a wrong answer is a scan that never
// looked at anything being read as one that found nothing wrong, so every
// half below reports the count it inspected BEFORE asserting a verdict on
// it (repo rule 84d1gfpywd).
//
// Word-boundary + `name:`-anchored regexes are what keep identifiers like
// LIVE_BACKOFF_JITTER_SPREAD (web/src/lib/live/live-client.ts) and prose
// like "never force-directed" (GraphCanvas.svelte's own layout comment)
// from ever matching — neither sits in a `name: '<x>'` position, so
// neither is a layout-name reference at all.
import * as fs from 'node:fs';
import * as path from 'node:path';
import * as url from 'node:url';

const __filename = url.fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
// web/scripts/.. -> web/.. so this resolves the repo root from any cwd.
const WEB_ROOT = path.resolve(__dirname, '..');
const REPO_ROOT = path.resolve(WEB_ROOT, '..');
const SRC_ROOT = path.join(WEB_ROOT, 'src');
const PACKAGE_JSON_PATH = path.join(WEB_ROOT, 'package.json');

const SCAN_EXTENSIONS = new Set(['.ts', '.js', '.svelte', '.mjs']);

// The full forbidden-layout-name vocabulary this scan rejects, named once
// for the report/comment surface — the regexes below are written as
// literal patterns (not built from this array) so the layout-name-position
// anchoring is visible and auditable directly in this file's source.
const FORBIDDEN_LAYOUT_NAMES = ['cose', 'cose-bilkent', 'fcose', 'cola', 'euler', 'spread', 'force'];

// Layout-option literal: cytoscape's own `layout: { name: '<x>' }` shape.
const LAYOUT_NAME_RE = /\bname\s*:\s*['"](cose|cose-bilkent|fcose|cola|euler|spread|force)['"]/g;
// Import/dependency specifier: a cytoscape layout extension package name.
// No "cytoscape-force" package exists, so "force" has no import-specifier
// form — only the layout-option literal covers it.
const LAYOUT_IMPORT_RE = /['"]cytoscape-(cose-bilkent|fcose|cola|euler|spread)['"]/g;

// The required positive control: elk must be reachable both as a layout
// name and as an import specifier, or this scan cannot be trusted to see
// anything at all.
const ELK_NAME_RE = /\bname\s*:\s*['"]elk['"]/g;
const ELK_IMPORT_RE = /['"]cytoscape-elk['"]/g;

function walk(dir, files) {
	for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
		const full = path.join(dir, entry.name);
		if (entry.isDirectory()) {
			walk(full, files);
		} else if (SCAN_EXTENSIONS.has(path.extname(entry.name))) {
			files.push(full);
		}
	}
	return files;
}

function countMatches(text, re) {
	re.lastIndex = 0;
	let count = 0;
	while (re.exec(text) !== null) count++;
	return count;
}

function lineOf(text, index) {
	return text.slice(0, index).split('\n').length;
}

function scanForbidden(re, file, text, out) {
	re.lastIndex = 0;
	let m;
	while ((m = re.exec(text)) !== null) {
		out.push({ file, line: lineOf(text, m.index), match: m[0] });
	}
}

// scanFileTexts is the ONE scan function both the real-tree walk and
// --self-test's injected-source path call — the same mechanism, never two
// independently-written scanners that could silently drift apart.
function scanFileTexts(fileTexts) {
	let elkLayoutRefs = 0;
	let elkImportRefs = 0;
	const forbiddenMatches = [];

	for (const { file, text } of fileTexts) {
		elkLayoutRefs += countMatches(text, ELK_NAME_RE);
		elkImportRefs += countMatches(text, ELK_IMPORT_RE);
		scanForbidden(LAYOUT_NAME_RE, file, text, forbiddenMatches);
		scanForbidden(LAYOUT_IMPORT_RE, file, text, forbiddenMatches);
	}

	return { filesScanned: fileTexts.length, elkLayoutRefs, elkImportRefs, forbiddenMatches };
}

function scanPackageJson(packageJsonPath) {
	const pkg = JSON.parse(fs.readFileSync(packageJsonPath, 'utf8'));
	const deps = { ...(pkg.dependencies ?? {}), ...(pkg.devDependencies ?? {}) };
	let elkImportRefs = 0;
	const forbiddenMatches = [];
	const relPath = path.relative(REPO_ROOT, packageJsonPath);
	for (const key of Object.keys(deps)) {
		if (key === 'cytoscape-elk') elkImportRefs += 1;
		if (/^cytoscape-(cose-bilkent|fcose|cola|euler|spread)$/.test(key)) {
			forbiddenMatches.push({ file: relPath, line: 0, match: key });
		}
	}
	return { elkImportRefs, forbiddenMatches };
}

// runScan walks web/src plus web/package.json's dependencies, optionally
// with extra in-memory sources spliced in (--self-test's injected source).
function runScan({ extraFiles = [] } = {}) {
	const files = walk(SRC_ROOT, []);
	const fileTexts = files.map((file) => ({
		file: path.relative(REPO_ROOT, file),
		text: fs.readFileSync(file, 'utf8')
	}));
	for (const extra of extraFiles) fileTexts.push(extra);

	const srcResult = scanFileTexts(fileTexts);
	const pkgResult = scanPackageJson(PACKAGE_JSON_PATH);

	const filesScanned = srcResult.filesScanned;
	const elkLayoutRefs = srcResult.elkLayoutRefs;
	const elkImportRefs = srcResult.elkImportRefs + pkgResult.elkImportRefs;
	const forbiddenMatches = [...srcResult.forbiddenMatches, ...pkgResult.forbiddenMatches];

	const verdict =
		filesScanned > 0 && elkLayoutRefs >= 1 && elkImportRefs >= 1 && forbiddenMatches.length === 0
			? 'PASS'
			: 'FAIL';

	return { filesScanned, elkLayoutRefs, elkImportRefs, forbiddenMatches, verdict };
}

function printReport(report) {
	console.log(JSON.stringify(report));
	console.log(
		`check-no-force-layout: scanned ${report.filesScanned} files; elk layout refs ${report.elkLayoutRefs}; elk import refs ${report.elkImportRefs}; forbidden matches ${report.forbiddenMatches.length}; verdict ${report.verdict}`
	);
	for (const m of report.forbiddenMatches) {
		console.log(`  forbidden: ${m.file}:${m.line} — ${m.match}`);
	}
}

// assertLayoutNameRegexCoversDeclaredFamilies is a startup self-consistency
// check: every family named in FORBIDDEN_LAYOUT_NAMES must actually appear
// inside LAYOUT_NAME_RE's own pattern source. This is what keeps the
// declared vocabulary and the regex that enforces it from silently
// drifting apart (adding a name to the list without also adding it to the
// regex would otherwise be a scan that quietly stopped checking for it).
function assertLayoutNameRegexCoversDeclaredFamilies() {
	const source = LAYOUT_NAME_RE.source;
	const missing = FORBIDDEN_LAYOUT_NAMES.filter((name) => !source.includes(name));
	if (missing.length > 0) {
		console.log(
			`::error::check-no-force-layout: LAYOUT_NAME_RE does not mention: ${missing.join(', ')} — the declared forbidden-family list and the enforcing regex have drifted apart`
		);
		process.exit(1);
	}
}
assertLayoutNameRegexCoversDeclaredFamilies();

// selfTest runs the SAME scan function over the real tree plus one
// injected in-memory source planting `name: 'cose'`. It passes only if
// that injected match is the ONLY forbidden match reported (the real tree
// stayed clean) AND the real tree's own file count proves the walk is not
// silently empty — a scan that cannot detect a match it was JUST handed
// must never be trusted, no matter how clean its real-tree verdict looks.
function selfTest() {
	const injected = { file: '<self-test>/injected.ts', text: "const layout = { name: 'cose' };" };
	const report = runScan({ extraFiles: [injected] });

	const injectedMatch = report.forbiddenMatches.find((m) => m.file === injected.file && m.line === 1);
	const onlyInjectedMatch = report.forbiddenMatches.length === 1 && injectedMatch !== undefined;
	const bigEnough = report.filesScanned >= 50;

	if (onlyInjectedMatch && bigEnough) {
		console.log(`check-no-force-layout self-test: PASS — injected 'cose' detected at ${injected.file}:1`);
		process.exit(0);
	}

	console.log('check-no-force-layout self-test: FAIL — the scan did not detect the injected layout name');
	printReport(report);
	process.exit(1);
}

const args = process.argv.slice(2);
if (args.includes('--self-test')) {
	selfTest();
} else {
	const report = runScan();
	printReport(report);
	process.exit(report.verdict === 'PASS' ? 0 : 1);
}
