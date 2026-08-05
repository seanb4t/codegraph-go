package mcp

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// sessionLinePrefix is the fixed, additive-only prefix (D-16) of VRFY-03's
// always-on negotiated-version stderr line. This prefix and the four key
// names formatSessionLine emits (requested, negotiated, client, tools) may
// gain siblings in a later phase but must never be renamed or removed —
// this is a published diagnostic users will be asked to paste.
const sessionLinePrefix = "codegraph: mcp-session"

// formatSessionLine renders VRFY-03's always-on stderr line in the fixed
// key order requested, negotiated, client, tools (D-14), terminated by a
// trailing newline. All four client-supplied values (requested, negotiated,
// clientName, clientVersion) are self-reported by the connecting MCP
// client — see sanitizeClientField's doc comment for why every one of them
// is sanitized before formatting (T-01-01).
func formatSessionLine(requested, negotiated, clientName, clientVersion string, tools int) string {
	return fmt.Sprintf("%s requested=%s negotiated=%s client=%s/%s tools=%d\n",
		sessionLinePrefix,
		sanitizeClientField(requested),
		sanitizeClientField(negotiated),
		sanitizeClientField(clientName),
		sanitizeClientField(clientVersion),
		tools,
	)
}

// sanitizeClientField makes s safe to embed as one space-delimited
// key=value field in the always-on stderr session line. s is always
// client-supplied (the requested/negotiated protocol version strings and
// both clientInfo fields all originate from the connecting MCP client's
// initialize request) — a hostile or buggy client could otherwise inject a
// fabricated second diagnostic line, or corrupt the line's parseability,
// via embedded newlines, spaces, or invalid UTF-8 (T-01-01).
//
// In order: invalid UTF-8 byte sequences are replaced with U+FFFD; every
// remaining control character (including CR and LF) and every space is
// replaced with '_'; the result is truncated to at most 256 bytes on a
// rune boundary; and an empty or all-stripped result becomes the literal
// "<unknown>".
func sanitizeClientField(s string) string {
	if s == "" {
		return "<unknown>"
	}

	valid := strings.ToValidUTF8(s, "�")

	sanitized := strings.Map(func(r rune) rune {
		if r == ' ' || unicode.IsControl(r) {
			return '_'
		}
		return r
	}, valid)

	if len(sanitized) > 256 {
		sanitized = truncateOnRuneBoundary(sanitized, 256)
	}

	if sanitized == "" {
		return "<unknown>"
	}
	return sanitized
}

// truncateOnRuneBoundary returns the prefix of s that is at most maxBytes
// long, walking backward from maxBytes to the nearest rune boundary so a
// multi-byte UTF-8 sequence is never split.
func truncateOnRuneBoundary(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	for maxBytes > 0 && !utf8.RuneStart(s[maxBytes]) {
		maxBytes--
	}
	return s[:maxBytes]
}
