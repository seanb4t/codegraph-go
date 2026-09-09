#!/usr/bin/env bash
# scripts/inject-cosign-key.sh
#
# Injects a --key= flag into the signs: pipe's sign-blob args in a GENERATED
# COPY of a GoReleaser config, then asserts the copy differs from the input
# by additions only, AND that exactly one --key= line was added (GRD-03,
# threat T-02-08).
#
# Why the count assertion exists: without an injected --key=, cosign
# sign-blob falls through to the keyless Fulcio/OIDC flow the calling job is
# denied, and hangs or hard-fails. The additions-only diff guard alone does
# not catch this — if the awk anchor below stops matching (a re-indent, a
# requote, a renamed key), the injection becomes a no-op, the generated
# config is byte-identical to the input, the diff is empty, and every
# additions-only check is trivially satisfied by that empty diff. The guard
# reported success at exactly the moment it should have refused. Counting
# the added --key= lines and requiring exactly 1 closes that gap: zero means
# the anchor no longer matches, two or more means a duplicated sign-blob
# block — both are config errors worth refusing before goreleaser ever runs.
#
# Usage: inject-cosign-key.sh <committed-config> <generated-config> <cosign-key>
#   $1  committed config path to read (must be readable)
#   $2  generated config path to write
#   $3  cosign key path to embed as --key=$3 (not checked for existence —
#       both call sites generate this key immediately before invoking, and
#       not requiring it here is what makes this guard runnable standalone
#       on any host, which the RED demonstration depends on)
#
# This script writes only to $2. It does not read, write, or reference
# dist/ or the repository's working-tree .goreleaser.yaml by name, and it
# creates no temporary directory — each call site owns its own run-scoped
# directory and cleanup trap.

set -euo pipefail

if [ "$#" -ne 3 ]; then
  echo "usage: $(basename "$0") <committed-config> <generated-config> <cosign-key>" >&2
  exit 2
fi

COMMITTED_CONFIG="$1"
GENERATED_CONFIG="$2"
COSIGN_KEY="$3"

if [ ! -r "${COMMITTED_CONFIG}" ]; then
  echo "::error::inject-cosign-key.sh: cannot read committed config: ${COMMITTED_CONFIG}" >&2
  exit 2
fi

# The key path is embedded verbatim inside a YAML double-quoted scalar. A
# literal double quote or backslash in it would emit a malformed line that
# is still, syntactically, an ADDITION — so the additions-only diff guard
# below would accept it. Refuse such a path here instead of escaping it:
# both call sites hand over a mktemp-scoped path that can never contain
# either byte, so a match is a caller bug worth a loud stop (review WR-01).
case "${COSIGN_KEY}" in
  *'"'*|*'\'*)
    echo "::error::inject-cosign-key.sh: cosign key path must not contain a double quote or backslash (it is embedded in a YAML double-quoted scalar): ${COSIGN_KEY}" >&2
    exit 2
    ;;
esac

# Inject --key into a GENERATED COPY of the committed config — the whole
# mechanism, not optional. The anchor's exact six-space indent and exact
# quoting are load-bearing: the committed .goreleaser.yaml carries exactly
# one line matching it today.
awk -v keyline="      - \"--key=${COSIGN_KEY}\"" '
  { print }
  /^      - "sign-blob"$/ { print keyline }
' "${COMMITTED_CONFIG}" > "${GENERATED_CONFIG}"

# Assert the copy is a MINIMAL, ADDITIONS-ONLY delta before invoking
# anything, so a rehearsal cannot silently drift into testing a different
# configuration than the one that ships. `<` lines in default diff(1)
# output are deletions/replacements — their presence is a hard failure.
# Every `>` (added) line must carry --key= or COSIGN_PASSWORD.
DIFF_OUT="$(diff "${COMMITTED_CONFIG}" "${GENERATED_CONFIG}" || true)"
if printf '%s\n' "${DIFF_OUT}" | grep -q '^<'; then
  echo "::error::${GENERATED_CONFIG} differs from ${COMMITTED_CONFIG} by more than additions:" >&2
  printf '%s\n' "${DIFF_OUT}"
  exit 1
fi
BAD_ADDITIONS="$(printf '%s\n' "${DIFF_OUT}" | grep '^> ' | grep -v -- '--key=' | grep -v 'COSIGN_PASSWORD' || true)"
if [ -n "${BAD_ADDITIONS}" ]; then
  echo "::error::${GENERATED_CONFIG} has additions not matching --key= or COSIGN_PASSWORD:" >&2
  printf '%s\n' "${BAD_ADDITIONS}"
  exit 1
fi
echo "generated config differs from ${COMMITTED_CONFIG} by additions only:"
printf '%s\n' "${DIFF_OUT}"

# The positive assertion this guard exists to add (D-06): count the added
# --key= lines and require exactly 1. Guarded with `|| true` so a zero
# match does not abort the script under `set -e` via grep -c's exit status.
KEY_LINE_COUNT="$(printf '%s\n' "${DIFF_OUT}" | grep -c '^> .*--key=' || true)"
echo "injected --key= lines: ${KEY_LINE_COUNT}"
if [ "${KEY_LINE_COUNT}" != "1" ]; then
  echo "::error::expected exactly 1 injected --key= line, found ${KEY_LINE_COUNT} — zero means the sign-blob anchor no longer matches (the hang case), two or more means a duplicated sign-blob block" >&2
  exit 1
fi
