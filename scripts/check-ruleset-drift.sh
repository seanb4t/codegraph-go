#!/usr/bin/env bash
# scripts/check-ruleset-drift.sh
#
# Compares this repository's shared required-status-check fixture
# (.github/required-status-checks.txt, D-07) against the LIVE GitHub
# ruleset that actually gates `main` — GitHub ruleset 20157557
# (`protect-main`) — with exact set equality in both directions (D-05).
#
# Why this exists (GRD-12, promotes GRD-07 which was declined at v0.13.0
# for its "skip-clean offline" clause): a context required live but absent
# from the fixture under-asserts what this repo THINKS gates `main`; a
# context in the fixture but not required live asserts a requirement that
# does not exist (see the fixture file itself for today's example — the
# fixture is deliberately the only place a context string is written,
# D-07, so this comment does not restate one). Both are drift, and both
# directions are checked.
#
# This script is CI-only by design (D-06) — there is deliberately no
# Taskfile target wrapping it and no Go test performs the live fetch, so
# it can never be invoked offline and read as PASS. Every failure path
# (missing/empty fixture, non-200 response, empty body, unparseable JSON,
# wrong ruleset name, non-active enforcement, zero live contexts, or a
# real set mismatch) is a named `::error::` and a non-zero exit — never a
# skip, never `continue-on-error`.
#
# Usage: bash scripts/check-ruleset-drift.sh [--fixture <path>]
#
# Environment:
#   GITHUB_REPOSITORY   owner/repo to query (default: seanb4t/codegraph-go)
#   RULESET_ID          the ruleset id to fetch (default: 20157557)
#   RULESET_URL_BASE    API base URL (default: https://api.github.com)
#   GITHUB_TOKEN        optional; when non-empty, sent as
#                       "Authorization: Bearer $GITHUB_TOKEN" — this
#                       public repo's ruleset is readable unauthenticated,
#                       but CI supplies the default token for a higher
#                       rate limit (Metadata: read, already granted by
#                       ci.yml's workflow-level `contents: read`).
#
# This script never echoes the raw ruleset response body — only the
# extracted context list, counts, and diffs (RESEARCH ASVS V1).

set -euo pipefail

FIXTURE_PATH=".github/required-status-checks.txt"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --fixture)
      if [ "$#" -lt 2 ]; then
        echo "usage: $(basename "$0") [--fixture <path>]" >&2
        exit 2
      fi
      FIXTURE_PATH="$2"
      shift 2
      ;;
    *)
      echo "usage: $(basename "$0") [--fixture <path>]" >&2
      exit 2
      ;;
  esac
done

for bin in curl jq; do
  if ! command -v "${bin}" >/dev/null 2>&1; then
    echo "::error::ruleset-drift: required tool '${bin}' not found on PATH" >&2
    exit 1
  fi
done

GITHUB_REPOSITORY="${GITHUB_REPOSITORY:-seanb4t/codegraph-go}"
RULESET_ID="${RULESET_ID:-20157557}"
RULESET_URL_BASE="${RULESET_URL_BASE:-https://api.github.com}"

# --- Precondition: the fixture must exist and carry at least one context,
# checked BEFORE any network call — an empty fixture must never let the
# real comparison run against a vacuous local set (rule 84d1gfpywd).
if [ ! -f "${FIXTURE_PATH}" ]; then
  echo "::error::ruleset-drift: fixture not found: ${FIXTURE_PATH}" >&2
  exit 1
fi

FIXTURE_COUNT="$(awk 'NF' "${FIXTURE_PATH}" | wc -l | tr -d ' ')"
echo "ruleset-drift: fixture ${FIXTURE_PATH} lists ${FIXTURE_COUNT} contexts"
if [ "${FIXTURE_COUNT}" = "0" ]; then
  echo "::error::ruleset-drift: fixture ${FIXTURE_PATH} lists zero non-blank contexts — a missing or emptied fixture must fail loudly, never pass vacuously" >&2
  exit 1
fi

# --- Fetch the live ruleset once. Split the trailing HTTP status line
# from the body via a newline delimiter that cannot appear inside it.
CURL_ARGS=(-sS --connect-timeout 10 --max-time 30 -w '\n%{http_code}' -H 'Accept: application/vnd.github+json' -H 'X-GitHub-Api-Version: 2022-11-28')
if [ -n "${GITHUB_TOKEN:-}" ]; then
  CURL_ARGS+=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
fi

RESPONSE="$(curl "${CURL_ARGS[@]}" "${RULESET_URL_BASE}/repos/${GITHUB_REPOSITORY}/rulesets/${RULESET_ID}" || true)"
HTTP_CODE="$(printf '%s' "${RESPONSE}" | tail -n1)"
BODY="$(printf '%s' "${RESPONSE}" | sed '$d')"

if [ "${HTTP_CODE}" != "200" ] || [ -z "${BODY}" ]; then
  echo "::error::ruleset-drift: GET rulesets/${RULESET_ID} returned HTTP ${HTTP_CODE} (or an empty body) — hard failure, never a skip" >&2
  exit 1
fi

# --- Parse guards: this must be the right ruleset, and it must actually
# be enforcing — a matching fixture against a non-enforcing or swapped
# ruleset would gate nothing while reading as PASS.
RULESET_NAME="$(printf '%s' "${BODY}" | jq -e -r '.name' 2>/dev/null)" || {
  echo "::error::ruleset-drift: jq could not parse the ruleset response (unparseable JSON)" >&2
  exit 1
}
if [ "${RULESET_NAME}" != "protect-main" ]; then
  echo "::error::ruleset-drift: ruleset ${RULESET_ID} has name '${RULESET_NAME}', expected 'protect-main'" >&2
  exit 1
fi

RULESET_ENFORCEMENT="$(printf '%s' "${BODY}" | jq -e -r '.enforcement' 2>/dev/null)" || {
  echo "::error::ruleset-drift: jq could not parse the ruleset response (unparseable JSON)" >&2
  exit 1
}
if [ "${RULESET_ENFORCEMENT}" != "active" ]; then
  echo "::error::ruleset-drift: ruleset ${RULESET_ID} enforcement is '${RULESET_ENFORCEMENT}', expected 'active' — a matching fixture against a non-enforcing ruleset gates nothing" >&2
  exit 1
fi

# --- Extract the live required-status-check context list.
LIVE_LIST="$(printf '%s' "${BODY}" | jq -r '
    .rules[] | select(.type=="required_status_checks")
    | .parameters.required_status_checks[].context' 2>/dev/null | LC_ALL=C sort)" || {
  echo "::error::ruleset-drift: jq could not parse the ruleset response (unparseable JSON)" >&2
  exit 1
}
LIVE_COUNT="$(printf '%s\n' "${LIVE_LIST}" | awk 'NF' | wc -l | tr -d ' ')"
FIXTURE_LIST="$(awk 'NF' "${FIXTURE_PATH}" | LC_ALL=C sort)"

echo "ruleset-drift: live has ${LIVE_COUNT} contexts, fixture has ${FIXTURE_COUNT} contexts"

if [ "${LIVE_COUNT}" = "0" ]; then
  echo "::error::ruleset-drift: live ruleset ${RULESET_ID} has zero required_status_checks contexts" >&2
  exit 1
fi

# --- Comparator: byte-equality of two sorted newline lists. Prints a
# diff (live as left, fixture as right — `<` means live-only, `>` means
# fixture-only) and returns non-zero on any difference.
compare_lists() {
  local live="$1" fixture="$2"
  if [ "${live}" = "${fixture}" ]; then
    return 0
  fi
  echo "ruleset-drift: diff (< live-only, > fixture-only):"
  diff <(printf '%s\n' "${live}") <(printf '%s\n' "${fixture}") || true
  return 1
}

# --- Built-in positive control (rule 84d1gfpywd): before trusting the
# real comparison, prove the comparator CAN fail. Build a temp fixture
# equal to the real sorted fixture plus one planted extra line, and
# require the comparator to report drift against it. If it does not, the
# comparator cannot be trusted to fail, and this script refuses to trust
# it either.
SELF_CHECK_FIXTURE="$(mktemp)"
trap 'rm -f "${SELF_CHECK_FIXTURE}"' EXIT
{
  printf '%s\n' "${FIXTURE_LIST}"
  printf '%s\n' "__planted-context-that-must-not-exist__"
} | LC_ALL=C sort > "${SELF_CHECK_FIXTURE}"
PLANTED_LIST="$(cat "${SELF_CHECK_FIXTURE}")"

if compare_lists "${LIVE_LIST}" "${PLANTED_LIST}" >/dev/null 2>&1; then
  echo "::error::ruleset-drift: self-check: comparator did not report a planted extra context — refusing to trust a comparison that cannot fail" >&2
  exit 1
fi
echo "ruleset-drift: self-check PASS — planted context detected"

# --- The real comparison.
if ! compare_lists "${LIVE_LIST}" "${FIXTURE_LIST}"; then
  echo "::error::ruleset-drift: live ruleset ${RULESET_ID} and ${FIXTURE_PATH} disagree (exact set equality both directions, D-05)" >&2
  exit 1
fi

echo "ruleset-drift: PASS — ${LIVE_COUNT} contexts identical"
