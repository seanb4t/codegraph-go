// highlight.ts is the ONE rendering path for repository source bytes
// (D-19/BRW-06). It imports highlight.js/lib/core plus exactly the
// per-language modules the indexer can produce — never the package's
// aggregate entry point, and never the bundled-common-languages module
// the package also ships — because the built bundle is committed to git
// AND embedded in the signed release binary (web/embed.go): bundle size
// here is a supply-chain constraint, not a load-time preference.
//
// The indexed set is 14 registered LanguageSpec.ID values (verified against
// internal/indexer/languages_*.go), not 12: languages_typescript.go's one
// init() registers three IDs — "typescript", "tsx", "javascript" — sharing
// one extractor. Only 13 hljs modules are registered below because hljs's
// own "typescript" grammar already declares "tsx" as an alias
// (registerLanguage() auto-registers a module's own aliases), so no 14th
// import exists for "tsx".
//
// Registration identifiers below MUST equal the exact string
// internal/schema.Node.Language sends on the wire (LanguageSpec.ID).
import hljs from 'highlight.js/lib/core';
// highlight.js's markup carries `hljs-*` CLASS NAMES only — it never
// inlines colour itself, so a theme stylesheet is required for the
// registered spans to render as anything but plain text. `github.css` is
// the one theme this app loads: this app has no dark-mode toggle today
// (verified: no `dark:` classes or mode-watcher anywhere in web/src), and
// GitHub's light palette is already the visual reference point BRW-09's
// permalink feature evokes elsewhere in this phase.
import 'highlight.js/styles/github.css';

import c from 'highlight.js/lib/languages/c';
import cpp from 'highlight.js/lib/languages/cpp';
import csharp from 'highlight.js/lib/languages/csharp';
import go from 'highlight.js/lib/languages/go';
import java from 'highlight.js/lib/languages/java';
import javascript from 'highlight.js/lib/languages/javascript';
import kotlin from 'highlight.js/lib/languages/kotlin';
import php from 'highlight.js/lib/languages/php';
import python from 'highlight.js/lib/languages/python';
import ruby from 'highlight.js/lib/languages/ruby';
import rust from 'highlight.js/lib/languages/rust';
import swift from 'highlight.js/lib/languages/swift';
import typescript from 'highlight.js/lib/languages/typescript';

hljs.registerLanguage('c', c);
hljs.registerLanguage('cpp', cpp);
hljs.registerLanguage('csharp', csharp);
hljs.registerLanguage('go', go);
hljs.registerLanguage('java', java);
hljs.registerLanguage('javascript', javascript);
hljs.registerLanguage('kotlin', kotlin);
hljs.registerLanguage('php', php);
hljs.registerLanguage('python', python);
hljs.registerLanguage('ruby', ruby);
hljs.registerLanguage('rust', rust);
hljs.registerLanguage('swift', swift);
hljs.registerLanguage('typescript', typescript);

// HLJS_ALIAS_COVERAGE: a declared map from an indexed language identifier
// with NO module of its own to the registered module that already covers
// it. Today exactly one entry: "tsx" is covered by the "typescript"
// module, whose own alias list already claims it (hljs's typescript.js:
// aliases: ['ts','tsx','mts','cts']). Declared rather than inferred so
// this fact — true only inside a third-party package — cannot go stale
// silently.
export const HLJS_ALIAS_COVERAGE: Record<string, string> = {
	tsx: 'typescript'
};

// HIGHLIGHT_COVERAGE is the complete coverage set — every registered
// module identifier above plus every HLJS_ALIAS_COVERAGE key — declared
// as a flat, sorted array of quoted string literals, ONE PER LINE, with
// NO computed values.
//
// web/highlight_coverage_test.go PARSES THIS EXACT DECLARATION as text
// (Go cannot import TypeScript). Changing this array's shape — a
// computed expression, a multi-element line, a spread, a `.map()` — will
// break that guard. A TS-side test (web/tests/browse-tracer.test.ts)
// independently derives the expected set from the registration list above
// and the alias map, and asserts it equals this array, so drift between
// this literal and the registration calls above is still caught — just
// not by making this declaration itself computed.
export const HIGHLIGHT_COVERAGE: string[] = [
	'c',
	'cpp',
	'csharp',
	'go',
	'java',
	'javascript',
	'kotlin',
	'php',
	'python',
	'ruby',
	'rust',
	'swift',
	'tsx',
	'typescript'
];

// EXTENSION_LANGUAGE maps a file extension (including the leading ".") to
// the language identifier registered above — a rendering-layer best-effort
// hint for file-mode opens, which carry no Node.language field (only
// NODE_DETAIL_MODE_SINGLE_DEF/MULTI_DEF do, via Node.language). It is
// transcribed by hand from internal/indexer/languages_*.go's own
// LanguageSpec.Extensions lists (WR-02): nothing in the type system binds
// the two together, so web/highlight_extension_coverage_test.go parses
// THIS EXACT DECLARATION as text (Go cannot import TypeScript) and asserts
// it is set-equal, key AND value, to indexer.RegisteredLanguageExtensions()
// — mirroring HIGHLIGHT_COVERAGE's own guard below. Changing this map's
// shape — a computed key, a spread, a multi-entry line — will break that
// guard exactly as changing HIGHLIGHT_COVERAGE's shape would.
//
// An unmatched extension is never guessed at: languageForPath returns ''
// and highlightSource degrades to its own honest plaintext-escape path.
export const EXTENSION_LANGUAGE: Record<string, string> = {
	'.c': 'c',
	'.h': 'c',
	'.cpp': 'cpp',
	'.cc': 'cpp',
	'.cxx': 'cpp',
	'.hpp': 'cpp',
	'.hh': 'cpp',
	'.cs': 'csharp',
	'.go': 'go',
	'.java': 'java',
	'.js': 'javascript',
	'.jsx': 'javascript',
	'.mjs': 'javascript',
	'.cjs': 'javascript',
	'.kt': 'kotlin',
	'.kts': 'kotlin',
	'.php': 'php',
	'.py': 'python',
	'.rb': 'ruby',
	'.rs': 'rust',
	'.swift': 'swift',
	'.ts': 'typescript',
	'.tsx': 'tsx'
};

// languageForPath derives a best-effort language identifier from a file
// path's extension via EXTENSION_LANGUAGE, or '' when the extension is
// absent or unregistered — never a guess.
export function languageForPath(path: string): string {
	const dot = path.lastIndexOf('.');
	if (dot === -1) return '';
	const ext = path.slice(dot).toLowerCase();
	return EXTENSION_LANGUAGE[ext] ?? '';
}

function escapeHtml(input: string): string {
	return input.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

// highlightSource(code, language) returns highlight.js markup for a
// REGISTERED language, or the input ESCAPED AS PLAINTEXT for anything
// else — never the library's own automatic-language-detection entry
// point. Auto-detection over an unknown indexed language does not degrade to
// plain text, it degrades to a CONFIDENT WRONG GUESS — colouring a Zig
// file as Rust reads as a working feature while being incorrect, and
// Task 3's set-equality guard means the registered set is supposed to
// equal the indexed set, so this branch should be unreachable in a
// correct tree and loud (a console warning) when it is reached.
//
// The input is verbatim, occasionally adversarial third-party repository
// content. hljs.highlight()'s default output escapes `<`, `>` and `&`
// with no HTML pass-through plugin loaded (v11 default) — this is the
// ONLY place in web/src that builds markup from repository bytes; no
// string concatenation, no pre-templating.
export function highlightSource(code: string, language: string): string {
	if (hljs.getLanguage(language)) {
		return hljs.highlight(code, { language }).value;
	}
	console.warn(
		`highlightSource: unregistered language identifier "${language}" — rendering as escaped plaintext, not auto-detected`
	);
	return escapeHtml(code);
}
