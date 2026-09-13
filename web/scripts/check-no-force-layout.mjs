#!/usr/bin/env node
// check-no-force-layout.mjs (WR-03a: narrowed claim) — proves that no
// cytoscape layout is invoked with a forbidden name WRITTEN AS A STRING
// LITERAL at the `name:` option position, or as a `cytoscape-<x>` import/
// dependency specifier. It does NOT prove "no force-directed layout is
// reachable from any code path" in full generality: a layout name built
// at runtime (string concatenation, a ternary, an imported constant, an
// options object assembled elsewhere and spread into the call) is NOT
// resolved by this scan. WR-03b's unresolvedLayoutName check narrows that
// residual gap further (see below) but does not close it completely —
// resolving an arbitrary expression to its runtime value is a data-flow
// problem this plain-regex scan does not attempt.
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
//
// WR-03b (unresolvedLayoutNames, REPORTED BUT NON-FATAL — does not affect
// the PASS/FAIL verdict): additionally flags every `layout(` / `.layout(`
// call site under web/src whose `name` option cannot be statically
// resolved to a quoted string literal within that call site's own line
// plus the following 5 lines — this catches both a bare variable
// (`name: layoutName`) and a spread of an options object declared
// elsewhere (`{ ...LAYOUT_OPTIONS, fit }`), since neither shows a
// resolvable literal at the call site itself. This is a genuinely weaker
// guarantee than a full data-flow trace: it does NOT verify what the
// referenced variable/spread actually resolves to, only that the call
// site's own text does not. Verified against this scan's own real tree:
// GraphCanvas.svelte's one production call site spreads LAYOUT_OPTIONS
// (declared a few lines above with a literal `name: 'elk'`) and IS
// flagged by this heuristic today — a real, acknowledged residual gap,
// which is exactly why this check is advisory (reported, with file:line)
// rather than build-breaking. Per WR-03's own guidance, GraphCanvas.svelte
// is NOT rewritten to dodge this scan; the gap is documented here instead.
import * as fs from 'node:fs';
import * as os from 'node:os';
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

// WR-03b: any `layout(`/`.layout(` call site (word-boundary catches both
// the bare identifier and the `.` method-call form).
const LAYOUT_CALL_RE = /\blayout\(/g;
// A resolvable literal `name:` value — quoted, anchored the same way
// LAYOUT_NAME_RE/ELK_NAME_RE are. If a call site's own window (its line
// plus the 5 following) does not contain this, the name could not be
// statically resolved to a literal from the call site's own text.
const LITERAL_NAME_IN_WINDOW_RE = /\bname\s*:\s*['"][^'"]*['"]/;

// walk decides recursion with fs.statSync (which FOLLOWS symlinks) rather
// than a readdirSync Dirent's own isDirectory() (which does NOT — a
// symlink-to-directory Dirent reports isDirectory() === false, so the old
// check-based-on-the-Dirent version never recursed into, or scanned, a
// symlinked directory anywhere under web/src — a real blind spot in the
// "reachable from any code path" claim, WR-02). visitedRealDirs tracks the
// REAL (symlink-resolved) path of every directory already descended into,
// so a symlink cycle (a directory that, through one or more symlinks,
// points back at an ancestor) terminates instead of recursing forever.
function walk(dir, files, visitedRealDirs = new Set([fs.realpathSync(dir)])) {
	for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
		const full = path.join(dir, entry.name);
		let stat;
		try {
			stat = fs.statSync(full);
		} catch {
			// A broken symlink (or a race with something else deleting the
			// entry) has nothing to scan — skip it rather than crash the scan.
			continue;
		}
		if (stat.isDirectory()) {
			const real = fs.realpathSync(full);
			if (visitedRealDirs.has(real)) continue;
			visitedRealDirs.add(real);
			walk(full, files, visitedRealDirs);
		} else if (SCAN_EXTENSIONS.has(path.extname(entry.name))) {
			// A symlinked FILE with a scanned extension is scanned too —
			// stat() already resolved it to a regular file above.
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

// scanUnresolvedLayoutNames (WR-03b) finds every layout( / .layout( call
// site whose own line + the following 5 lines contain no resolvable
// quoted `name:` literal — a bare variable (`name: layoutName`) or a
// spread of an options object declared elsewhere both look the same to
// this line-windowed check: unresolved from the call site's own text.
function scanUnresolvedLayoutNames(file, text, out) {
	const lines = text.split('\n');
	LAYOUT_CALL_RE.lastIndex = 0;
	let m;
	while ((m = LAYOUT_CALL_RE.exec(text)) !== null) {
		const lineNum = lineOf(text, m.index);
		const window = lines.slice(lineNum - 1, lineNum - 1 + 6).join('\n');
		if (!LITERAL_NAME_IN_WINDOW_RE.test(window)) {
			out.push({ file, line: lineNum, snippet: lines[lineNum - 1].trim() });
		}
	}
}

// scanFileTexts is the ONE scan function both the real-tree walk and
// --self-test's injected-source path call — the same mechanism, never two
// independently-written scanners that could silently drift apart.
function scanFileTexts(fileTexts) {
	let elkLayoutRefs = 0;
	let elkImportRefs = 0;
	const forbiddenMatches = [];
	const unresolvedLayoutNames = [];

	for (const { file, text } of fileTexts) {
		elkLayoutRefs += countMatches(text, ELK_NAME_RE);
		elkImportRefs += countMatches(text, ELK_IMPORT_RE);
		scanForbidden(LAYOUT_NAME_RE, file, text, forbiddenMatches);
		scanForbidden(LAYOUT_IMPORT_RE, file, text, forbiddenMatches);
		scanUnresolvedLayoutNames(file, text, unresolvedLayoutNames);
	}

	return { filesScanned: fileTexts.length, elkLayoutRefs, elkImportRefs, forbiddenMatches, unresolvedLayoutNames };
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
	// unresolvedLayoutNames (WR-03b) is reported but deliberately excluded
	// from the verdict below — see the header comment for why this is
	// advisory, not build-breaking.
	const unresolvedLayoutNames = srcResult.unresolvedLayoutNames;

	const verdict =
		filesScanned > 0 && elkLayoutRefs >= 1 && elkImportRefs >= 1 && forbiddenMatches.length === 0
			? 'PASS'
			: 'FAIL';

	return { filesScanned, elkLayoutRefs, elkImportRefs, forbiddenMatches, unresolvedLayoutNames, verdict };
}

function printReport(report) {
	console.log(JSON.stringify(report));
	console.log(
		`check-no-force-layout: scanned ${report.filesScanned} files; elk layout refs ${report.elkLayoutRefs}; elk import refs ${report.elkImportRefs}; forbidden matches ${report.forbiddenMatches.length}; unresolved layout names (advisory) ${report.unresolvedLayoutNames.length}; verdict ${report.verdict}`
	);
	for (const m of report.forbiddenMatches) {
		console.log(`  forbidden: ${m.file}:${m.line} — ${m.match}`);
	}
	for (const m of report.unresolvedLayoutNames) {
		console.log(`  unresolved (advisory, does not fail verdict): ${m.file}:${m.line} — ${m.snippet}`);
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

// selfTestSymlinkTraversal (WR-02) proves the walker's symlink-following
// fix on REAL disk structure, not just the in-memory scan mechanism: a
// forbidden layout name planted in a directory that is reachable ONLY
// through a symlink must still be found. Both temp roots live under
// os.tmpdir() and are always removed, even on failure/throw.
function selfTestSymlinkTraversal() {
	const base = fs.mkdtempSync(path.join(os.tmpdir(), 'check-no-force-layout-planted-'));
	const linkRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'check-no-force-layout-root-'));
	try {
		const plantedDir = path.join(base, 'planted');
		fs.mkdirSync(plantedDir, { recursive: true });
		fs.writeFileSync(path.join(plantedDir, 'injected.ts'), "const layout = { name: 'cose' };\n");
		fs.symlinkSync(plantedDir, path.join(linkRoot, 'linkdir'), 'dir');

		const files = walk(linkRoot, []);
		const fileTexts = files.map((file) => ({ file, text: fs.readFileSync(file, 'utf8') }));
		const result = scanFileTexts(fileTexts);
		const found = result.forbiddenMatches.some((m) => m.match.includes('cose'));

		if (!found) {
			console.log(
				'check-no-force-layout self-test (symlink traversal): FAIL — a forbidden match reachable only through a symlinked directory was NOT detected'
			);
			return false;
		}
		console.log(
			'check-no-force-layout self-test (symlink traversal): PASS — forbidden match found through linkdir -> planted'
		);
		return true;
	} finally {
		fs.rmSync(linkRoot, { recursive: true, force: true });
		fs.rmSync(base, { recursive: true, force: true });
	}
}

// selfTest runs the SAME scan function over the real tree plus two
// injected in-memory sources: one planting `name: 'cose'` (the forbidden-
// literal case) and one planting `cy.layout({ name: layoutName })` (the
// WR-03b unresolved-name case). It passes only if the forbidden injection
// is the ONLY forbidden match reported (the real tree stayed clean), the
// unresolved injection is detected as an advisory finding, AND the real
// tree's own file count proves the walk is not silently empty — a scan
// that cannot detect a match it was JUST handed must never be trusted, no
// matter how clean its real-tree verdict looks. It also runs
// selfTestSymlinkTraversal (WR-02), a real on-disk case the in-memory
// injections above cannot exercise.
function selfTest() {
	const injected = { file: '<self-test>/injected.ts', text: "const layout = { name: 'cose' };" };
	const injectedUnresolved = {
		file: '<self-test>/unresolved.ts',
		text: 'cy.layout({ name: layoutName });'
	};
	const report = runScan({ extraFiles: [injected, injectedUnresolved] });

	const injectedMatch = report.forbiddenMatches.find((m) => m.file === injected.file && m.line === 1);
	const onlyInjectedMatch = report.forbiddenMatches.length === 1 && injectedMatch !== undefined;
	const bigEnough = report.filesScanned >= 50;
	const symlinkOk = selfTestSymlinkTraversal();

	const unresolvedFound = report.unresolvedLayoutNames.some(
		(m) => m.file === injectedUnresolved.file && m.line === 1
	);
	if (unresolvedFound) {
		console.log(
			`check-no-force-layout self-test (WR-03b): PASS — unresolved layout name detected at ${injectedUnresolved.file}:1`
		);
	} else {
		console.log(
			`check-no-force-layout self-test (WR-03b): FAIL — unresolved layout name at ${injectedUnresolved.file}:1 was NOT detected`
		);
	}

	if (onlyInjectedMatch && bigEnough && symlinkOk && unresolvedFound) {
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
