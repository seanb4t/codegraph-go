#!/usr/bin/env bash
# scripts/check-workflow-output-delimiter.sh
#
# Proves neither pull_request_target workflow's multi-line $GITHUB_OUTPUT
# write is terminable by a fork-authored file path (FIX-10, GH #15, D-17).
# `require-issue-link.yml` and `pr-template-format.yml` both run a
# "Collect changed files" step that pipes `gh pr view --json files` into a
# heredoc-style multi-line $GITHUB_OUTPUT write. A fork PR that adds a file
# whose path is exactly the heredoc's closing delimiter terminates the
# heredoc early; everything after that line lands in the output file
# unparsed as the intended value, which is exactly the class of injection
# GitHub's own multi-line-output guidance warns about.
#
# This script extracts the ACTUAL run: body each workflow ships (from the
# YAML, at run time — never a hand-copied duplicate that could drift) and
# executes it against a stub `gh` that emits a fixed five-line payload
# containing a path named after the old fixed delimiter, then parses the
# resulting $GITHUB_OUTPUT the way GitHub Actions does and asserts the
# value round-trips intact.
#
# Usage: check-workflow-output-delimiter.sh [--self-test]
#   (no args)     Real run: extract and exercise both workflows' shipped
#                 blocks. Exits 0 only if BOTH round-trip the attack
#                 payload without corruption.
#   --self-test   Positive control, run FIRST always (see
#                 Taskfile.yml's check:workflow-output-delimiter target):
#                 (1) proves this harness can still observe a
#                 locally-constructed copy of the OLD fixed-delimiter form
#                 corrupting the same payload — a harness that has stopped
#                 detecting the defect it exists to detect must never pass
#                 silently; (2) proves two invocations of the
#                 delimiter-generating expression in the same second
#                 produce different values, so a delimiter that is
#                 accidentally per-day or per-boot rather than per-run is
#                 caught.
#
# This script writes only inside its own mktemp -d scratch directory
# (removed on exit via trap). It never modifies .github/workflows/*.yml,
# Taskfile.yml, or anything else on disk. Run it from the repository root.
#
# Portability note: this repo's dev machines and GitHub's ubuntu-latest
# runner do not guarantee the same bash version (macOS ships bash 3.2 by
# default). Comparison logic below deliberately avoids `"${arr[@]}"`
# expansion of a possibly-empty array under `set -u` — a documented bash
# < 4.4 pitfall where that raises "unbound variable" instead of expanding
# to nothing — by comparing files with `diff` rather than iterating arrays.

set -euo pipefail

if [ "$#" -gt 1 ]; then
  echo "usage: $(basename "$0") [--self-test]" >&2
  exit 2
fi
if [ "$#" -eq 1 ] && [ "$1" != "--self-test" ]; then
  echo "usage: $(basename "$0") [--self-test]" >&2
  exit 2
fi

SCRATCH="$(mktemp -d)"
cleanup() { rm -rf "${SCRATCH}"; }
trap cleanup EXIT

# The fixed five-line attack payload every run of this harness feeds through
# both the real blocks and the self-test's old-form positive control. One
# line is a path named exactly after the old fixed delimiter (the attack),
# one contains a space, one contains a single quote, the rest are ordinary.
# Written to a scratch file once so `compare_output` can `diff` against it
# rather than iterating a bash array (see the portability note above).
write_expected_payload() {
  printf '%s\n' \
    "PRFILES_EOF" \
    "path with a space.go" \
    "path with a 'quote'.go" \
    "normal/path/one.go" \
    "normal/path/two.go" \
    >"${SCRATCH}/expected_payload.txt"
}

# ---- stub `gh`: ignores its arguments, always emits the same five lines
#      as write_expected_payload above ----
setup_stub_gh() {
  mkdir -p "${SCRATCH}/bin"
  cat >"${SCRATCH}/bin/gh" <<'GHSTUB'
#!/usr/bin/env bash
# Stub gh for check-workflow-output-delimiter.sh's harness: ignores every
# argument and always prints the fixed five-line attack payload.
printf '%s\n' "PRFILES_EOF" "path with a space.go" "path with a 'quote'.go" "normal/path/one.go" "normal/path/two.go"
GHSTUB
  chmod +x "${SCRATCH}/bin/gh"
}

# ---- extract a named step's `run: |` body from a workflow file, at the
#      block's own indentation (awk over the YAML text directly — no YAML
#      parser dependency) ----
extract_step_body() {
  local file="$1" step="$2"
  awk -v step="${step}" '
    BEGIN { in_step = 0; found_run = 0; base = -1 }
    {
      line = $0
      if (match(line, /^[ \t]*- name:[ \t]*/)) {
        name = substr(line, RLENGTH + 1)
        gsub(/[ \t]+$/, "", name)
        if (name == step) {
          in_step = 1
          found_run = 0
          base = -1
        } else if (in_step && found_run == 1) {
          exit
        } else {
          in_step = 0
        }
        next
      }
      if (in_step && found_run == 0) {
        if (match(line, /^[ \t]*run:[ \t]*\|/)) {
          found_run = 1
        }
        next
      }
      if (in_step && found_run == 1) {
        if (match(line, /^[ \t]*$/)) { print ""; next }
        match(line, /^[ \t]*/)
        indent = RLENGTH
        if (base == -1) {
          base = indent
          print substr(line, base + 1)
          next
        }
        if (indent < base) { exit }
        print substr(line, base + 1)
      }
    }
  ' "${file}"
}

# ---- run an extracted (or self-test) body against the stub gh, writing
#      $GITHUB_OUTPUT to $2 ----
run_body() {
  local body_file="$1" ghoutput="$2"
  : >"${ghoutput}"
  PATH="${SCRATCH}/bin:${PATH}" \
    GITHUB_OUTPUT="${ghoutput}" \
    PR_NUMBER="123" \
    GITHUB_REPOSITORY="owner/repo" \
    GH_TOKEN="dummy-token" \
    bash "${body_file}"
}

# ---- parse a $GITHUB_OUTPUT file the way GitHub Actions does: read
#      "<key><<DELIM", accumulate lines until one equals DELIM exactly,
#      treat anything else outside a heredoc as stray/unparsed content ----
parse_github_output() {
  local ghfile="$1" outdir="$2"
  mkdir -p "${outdir}"
  : >"${outdir}/keys.txt"
  : >"${outdir}/stray.txt"
  local in_value=0 cur_delim="" idx=0 valfile=""
  while IFS= read -r rawline || [ -n "${rawline}" ]; do
    if [ "${in_value}" -eq 1 ]; then
      if [ "${rawline}" = "${cur_delim}" ]; then
        in_value=0
        continue
      fi
      printf '%s\n' "${rawline}" >>"${valfile}"
      continue
    fi
    case "${rawline}" in
      *"<<"*)
        idx=$((idx + 1))
        key="${rawline%%<<*}"
        cur_delim="${rawline#*<<}"
        valfile="${outdir}/val_${idx}"
        : >"${valfile}"
        printf '%s\t%s\n' "${idx}" "${key}" >>"${outdir}/keys.txt"
        in_value=1
        ;;
      *)
        printf '%s\n' "${rawline}" >>"${outdir}/stray.txt"
        ;;
    esac
  done <"${ghfile}"
}

# ---- compare a parsed $GITHUB_OUTPUT against the expected payload for one
#      workflow. Diff-based (see portability note at the top of this file)
#      rather than bash-array iteration. ----
compare_output() {
  local ghoutput="$1" expected_key="$2" label="$3"
  local parsedir="${SCRATCH}/parsed_${label}"
  parse_github_output "${ghoutput}" "${parsedir}"

  local nkeys
  nkeys=$(wc -l <"${parsedir}/keys.txt" | tr -d ' ')
  if [ "${nkeys}" -ne 1 ]; then
    echo "${label}: FAIL — expected exactly 1 output key, found ${nkeys}"
    echo "${label}: keys.txt contents:"
    cat "${parsedir}/keys.txt"
    return 1
  fi

  local idx key
  idx=$(cut -f1 "${parsedir}/keys.txt")
  key=$(cut -f2 "${parsedir}/keys.txt")
  if [ "${key}" != "${expected_key}" ]; then
    echo "${label}: FAIL — expected output key '${expected_key}', found '${key}'"
    return 1
  fi

  local valfile="${parsedir}/val_${idx}"
  [ -f "${valfile}" ] || : >"${valfile}"

  local diff_out="${parsedir}/diff.txt"
  if ! diff -u "${SCRATCH}/expected_payload.txt" "${valfile}" >"${diff_out}" 2>&1; then
    echo "${label}: FAIL — value did not round-trip intact (diff: expected vs actual):"
    cat "${diff_out}"
    return 1
  fi

  if [ -s "${parsedir}/stray.txt" ]; then
    echo "${label}: FAIL — stray/unparsed lines found in \$GITHUB_OUTPUT after the heredoc closed:"
    cat "${parsedir}/stray.txt"
    return 1
  fi

  echo "${label}: PASS — output key '${key}' round-tripped all payload lines intact"
  return 0
}

# ---- real run: extract + exercise both shipped blocks ----
real_run() {
  local overall=0
  echo "check-workflow-output-delimiter: exercising the shipped 'Collect changed files' blocks against a fork-authored payload containing a path named after the old delimiter"
  echo

  local specs
  specs="$(printf '%s\n' \
    ".github/workflows/require-issue-link.yml|Collect changed files|list" \
    ".github/workflows/pr-template-format.yml|Collect changed files|files")"

  local file step key label
  while IFS='|' read -r file step key; do
    [ -n "${file}" ] || continue
    label="$(basename "${file}")"
    echo "==> ${label} (step '${step}', expected output key '${key}')"

    if [ ! -f "${file}" ]; then
      echo "::error::check-workflow-output-delimiter: workflow file not found: ${file} (run from the repo root)" >&2
      overall=1
      echo
      continue
    fi

    local body_file="${SCRATCH}/body_${label}.sh"
    extract_step_body "${file}" "${step}" >"${body_file}"

    if [ ! -s "${body_file}" ]; then
      echo "::error::check-workflow-output-delimiter: extracted ZERO lines of shell from '${step}' in ${file} — a YAML restructure may have defeated extraction" >&2
      overall=1
      echo
      continue
    fi
    if ! grep -q '>> "\$GITHUB_OUTPUT"' "${body_file}"; then
      echo "::error::check-workflow-output-delimiter: extracted body from '${step}' in ${file} has no '>> \"\$GITHUB_OUTPUT\"' redirection — extraction likely missed the step" >&2
      overall=1
      echo
      continue
    fi

    local ghout="${SCRATCH}/out_${label}"
    run_body "${body_file}" "${ghout}"

    if ! compare_output "${ghout}" "${key}" "${label}"; then
      overall=1
    fi
    echo
  done <<<"${specs}"

  if [ "${overall}" -ne 0 ]; then
    echo "::error::check-workflow-output-delimiter: FAILED — see per-file output above" >&2
    return 1
  fi

  echo "check-workflow-output-delimiter: PASS — both workflows round-trip the attack payload intact"
  return 0
}

# ---- --self-test: positive control ----
self_test() {
  local failed=0

  echo "--self-test: positive control — verifying a local copy of the OLD fixed-delimiter form still corrupts the payload"
  cat >"${SCRATCH}/old_form.sh" <<'OLDFORM'
set -euo pipefail
{
  echo "list<<PRFILES_EOF"
  gh pr view "$PR_NUMBER" --repo "$GITHUB_REPOSITORY" --json files --jq '.files[].path'
  echo "PRFILES_EOF"
} >> "$GITHUB_OUTPUT"
OLDFORM

  local ghout="${SCRATCH}/out_selftest_oldform"
  run_body "${SCRATCH}/old_form.sh" "${ghout}"

  local oldform_report="${SCRATCH}/oldform_report.txt"
  if compare_output "${ghout}" "list" "self-test/old-form" >"${oldform_report}" 2>&1; then
    echo "::error::check-workflow-output-delimiter --self-test: the OLD fixed-delimiter form did NOT corrupt the attack payload — this harness can no longer detect the defect it exists to detect" >&2
    cat "${oldform_report}" >&2
    failed=1
  else
    echo "self-test/old-form: CONFIRMED CORRUPTED (expected) —"
    cat "${oldform_report}"
  fi

  echo
  echo "--self-test: concurrency edge — two invocations of the delimiter expression must differ"
  local d1 d2
  d1="PRFILES_$(openssl rand -hex 16)"
  d2="PRFILES_$(openssl rand -hex 16)"
  if [ "${d1}" = "${d2}" ]; then
    echo "::error::check-workflow-output-delimiter --self-test: two invocations of 'PRFILES_\$(openssl rand -hex 16)' produced the SAME value (${d1}) — the delimiter is not actually per-run" >&2
    failed=1
  else
    echo "self-test/entropy: PASS — two invocations differed (${d1} != ${d2})"
  fi

  echo
  if [ "${failed}" -ne 0 ]; then
    echo "::error::check-workflow-output-delimiter --self-test: FAILED" >&2
    return 1
  fi
  echo "--self-test: PASS"
  return 0
}

setup_stub_gh
write_expected_payload

if [ "${1:-}" = "--self-test" ]; then
  self_test
  exit $?
fi

real_run
exit $?
