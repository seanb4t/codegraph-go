package query

import (
	"regexp"
	"strings"
)

// stopWords is the FTS-term stopword set consumed by extractSearchTerms —
// ported verbatim from TS CodeGraph 1.3.1's search/query-utils.js
// (exports.STOP_WORDS, query-utils.js:102-120). This is a SEPARATE,
// smaller list than commonWords (used by extractSymbolsFromQuery) — do
// not conflate the two (RESEARCH Anti-Pattern "Conflating the two
// tokenizers").
var stopWords = map[string]bool{
	// English
	"the": true, "a": true, "an": true, "and": true, "or": true, "but": true, "in": true, "on": true, "at": true, "to": true, "for": true,
	"of": true, "with": true, "by": true, "from": true, "is": true, "it": true, "that": true, "this": true, "are": true, "was": true,
	"be": true, "has": true, "had": true, "have": true, "do": true, "does": true, "did": true, "will": true, "would": true, "could": true,
	"should": true, "may": true, "might": true, "can": true, "shall": true, "not": true, "no": true, "all": true, "each": true,
	"every": true, "how": true, "what": true, "where": true, "when": true, "who": true, "which": true, "why": true,
	"i": true, "me": true, "my": true, "we": true, "our": true, "you": true, "your": true, "he": true, "she": true, "they": true,
	"show": true, "give": true, "tell": true,
	"been": true, "done": true, "made": true, "used": true, "using": true, "work": true, "works": true, "found": true,
	"also": true, "into": true, "then": true, "than": true, "just": true, "more": true, "some": true, "such": true,
	"over": true, "only": true, "out": true, "its": true, "so": true, "up": true, "as": true, "if": true,
	"look": true, "need": true, "needs": true, "want": true, "happen": true, "happens": true,
	"affect": true, "affected": true, "break": true, "breaks": true, "failing": true,
	"implemented": true, "implement": true,
	// Code-specific noise (avoid filtering common symbol names like
	// get/set/add/build/find/list).
	"code": true, "file": true, "files": true, "function": true, "method": true, "class": true, "type": true,
	"fix": true, "bug": true, "called": true,
}

var (
	searchCompoundPattern = regexp.MustCompile(`\b([a-zA-Z][a-zA-Z0-9]*(?:[A-Z][a-z]+)+|[A-Z][a-z]+(?:[A-Z][a-z]*)+)\b`)
	searchSnakePattern    = regexp.MustCompile(`\b([a-zA-Z][a-zA-Z0-9]*(?:_[a-zA-Z0-9]+)+)\b`)
	searchCamelLower      = regexp.MustCompile(`([a-z])([A-Z])`)
	searchCamelAcronym    = regexp.MustCompile(`([A-Z]+)([A-Z][a-z])`)
	searchNormalizeSep    = regexp.MustCompile(`[_.]+`)
	searchWordSplit       = regexp.MustCompile(`[^a-zA-Z0-9]+`)
)

// extractSearchTerms is TS CodeGraph 1.3.1's H2 tokenizer
// (search/query-utils.js:189-242, extractSearchTerms) — EXPL-01's literal
// "stopword-filtered" target, feeding the FTS gather channel (plan 07).
// Ported verbatim: preserves camelCase/PascalCase and snake_case compound
// identifiers (>=3 chars, lowercased) alongside their split sub-words,
// then drops any split word shorter than 3 chars or present in
// stopWords. Tokens are returned in first-seen scan order (D-04
// determinism) — never map iteration.
//
// Security (V5/WR-05): an empty or whitespace-only query returns an
// empty slice — never a token set a downstream FTS gather could treat as
// "match everything."
//
// Divergence (D-02): TS's getStemVariants() FTS-prefix stem expansion
// (query-utils.js:129-175) is deliberately deferred, not ported here — a
// follow-on plan can add it when the FTS gather channel (plan 07) needs
// it. This function has no stem-variant hook.
func extractSearchTerms(query string) []string {
	if strings.TrimSpace(query) == "" {
		return []string{}
	}

	tokens := make([]string, 0)
	seen := make(map[string]bool)
	add := func(s string) {
		if !seen[s] {
			seen[s] = true
			tokens = append(tokens, s)
		}
	}

	// 1. Preserve compound identifiers before splitting: camelCase/
	//    PascalCase ("scrapeLoop", "UserService") and snake_case
	//    ("scrape_loop"), >=3 chars, lowercased.
	for _, m := range searchCompoundPattern.FindAllString(query, -1) {
		lower := strings.ToLower(m)
		if len(lower) >= 3 {
			add(lower)
		}
	}
	for _, m := range searchSnakePattern.FindAllString(query, -1) {
		lower := strings.ToLower(m)
		if len(lower) >= 3 {
			add(lower)
		}
	}

	// 2. Split camelCase/PascalCase into words, underscores/dots ->
	//    space, then split on any non-alphanumeric run.
	camelSplit := searchCamelLower.ReplaceAllString(query, "$1 $2")
	camelSplit = searchCamelAcronym.ReplaceAllString(camelSplit, "$1 $2")
	normalised := searchNormalizeSep.ReplaceAllString(camelSplit, " ")
	for _, word := range searchWordSplit.Split(normalised, -1) {
		if word == "" {
			continue
		}
		lower := strings.ToLower(word)
		if len(lower) < 3 || stopWords[lower] {
			continue
		}
		add(lower)
	}

	return tokens
}
