#!/usr/bin/env bash
# scripts/live-push-concurrency-check.sh
#
# Criterion 5's real-process gate (06-07, LIV-01): "With codegraph daemon and
# serve --mcp running against the same store, a live-push session survives
# repeated real re-index flushes without starving a sync or holding the
# store open — verified against the real processes, not a stub."
#
# FOUR real OS processes participate; THREE of them are product processes
# this criterion is actually about:
#   - `codegraph daemon start`  (product, the writer — a FOREGROUND process,
#     internal/cli/daemon.go: "the explicit foreground blocking watch/index
#     server"; this script backgrounds it and keeps its PID)
#   - `codegraph serve --mcp`   (product, a second real reader — spawned BY
#     the mcp probe as ITS child; the PID recorded for this entry is the
#     child's PID the probe reports, never the probe's own)
#   - `codegraph ui`            (product, the live-push publisher)
#   - the `stream` probe        (harness — holds a live-push session OPEN
#     for the whole run; not one of the three product processes)
#
# Methodology (06-CONTEXT.md's binding criterion-5 block, 06-07-PLAN.md):
#   1. Seed a throwaway scratch git repository (never this repository's own
#      working tree) and index it once.
#   2. Start the daemon ALONE and drive N real re-index flushes, recording
#      baselineMaxFlushDurationMs — the no-UI number every later comparison
#      is bounded against. Without this, a flush that finishes ten times
#      slower than it should would score a clean pass on completion count
#      alone.
#   3. Start `codegraph serve --mcp` (via the mcp probe, held open the whole
#      run), `codegraph ui`, and the `stream` probe (a live-push session
#      held open the whole run) — the full concurrent topology.
#   4. Drive the SAME N real re-index flushes again, this time under that
#      full topology, and compare.
#
# Each flush is driven by writing a small Go source file into the scratch
# repo whose new content declares one uniquely-named function — a
# content-derived marker, not a bare last_sync_unix_ms timestamp check
# (last_sync advancing proves *an* index ran, not that THIS revision
# landed; the daemon may be reconciling something else entirely).
#
# Flush interval accounting (see 06-07-PLAN.md for the full derivation):
#   - clock_start = write_time + configured_debounce. This is a bounded,
#     CONSERVATIVE ESTIMATE of when indexing began, never an observation:
#     internal/watch/debounce.go's Debouncer has no leading edge and no
#     max-wait, so flush_start >= write_time + window always, and
#     therefore clock_start <= flush_start always. The interval this
#     produces can only OVER-measure, never under.
#   - clock_end = the first instant the flush's own unique marker token is
#     queryable from the index (polled via `codegraph search`), not merely
#     the instant last_sync_unix_ms moves.
#   - adjMs = clock_end - clock_start (the value compared against the
#     degradation bound); rawMs = clock_end - write_time (the UNADJUSTED
#     elapsed, recorded alongside so the subtraction is auditable and an
#     internal/daemon ErrStoreLocked requeue — which re-Adds to the
#     debouncer and hides extra windows inside the duration — is visible
#     rather than silent).
#
# "Starved" is defined operationally, not as a feeling: a flush is starved
# when EITHER (a) the daemon's own log output for that flush's window
# contains a store-lock contention line (matched verbatim, never inferred),
# OR (b) the flush did not become queryable within flushTimeoutMs.
#
# Writes corpora/live-push-concurrency-check.json (or --out) on EVERY run,
# including a harness failure, so a broken run is never silently absent.
set -uo pipefail

# ---------------------------------------------------------------------------
# Config (flag-overridable; defaults chosen for a fast, real, non-flaky run)
# ---------------------------------------------------------------------------
FLUSH_COUNT=5
OUT_JSON="corpora/live-push-concurrency-check.json"
DEBOUNCE_MS=100          # matches this repo's own established test-only convention
FLUSH_TIMEOUT_MS=5000
SKIP_BASELINE=false
BIN_OVERRIDE=""

while [ $# -gt 0 ]; do
  case "$1" in
    --out) OUT_JSON="$2"; shift 2 ;;
    --flushes) FLUSH_COUNT="$2"; shift 2 ;;
    --debounce-ms) DEBOUNCE_MS="$2"; shift 2 ;;
    --flush-timeout-ms) FLUSH_TIMEOUT_MS="$2"; shift 2 ;;
    --skip-baseline) SKIP_BASELINE=true; shift 1 ;;
    --bin) BIN_OVERRIDE="$2"; shift 2 ;;
    *) echo "live-push-concurrency-check: unknown argument: $1" >&2; exit 2 ;;
  esac
done

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

# HOLD_SECONDS: how long the mcp/stream probes hold their sessions open.
# Sized off the SAME parameters driving the flush loop so the hold period
# genuinely overlaps the concurrent flushes without an arbitrary guess.
HOLD_MS=$(( FLUSH_COUNT * (DEBOUNCE_MS + 3200) + 5000 ))
HOLD_SECONDS=$(( (HOLD_MS + 999) / 1000 ))

WORKDIR="$(mktemp -d "${TMPDIR:-/tmp}/live-push-concurrency-check.XXXXXX")"
SCRATCH_REPO="$WORKDIR/scratchrepo"
BIN="$WORKDIR/codegraph"
DAEMON_LOG="$WORKDIR/daemon.log"
UI_LOG="$WORKDIR/ui.log"
MCP_PROBE_OUT="$WORKDIR/mcp-probe.json"
STREAM_OUT="$WORKDIR/stream-events.jsonl"

DAEMON_PID=""
UI_PID=""
MCP_PROBE_PID=""
STREAM_PID=""

cleanup() {
  local ec=$?
  set +u
  [ -n "${STREAM_PID:-}" ] && kill -9 "$STREAM_PID" >/dev/null 2>&1
  [ -n "${MCP_PROBE_PID:-}" ] && kill -9 "$MCP_PROBE_PID" >/dev/null 2>&1
  [ -n "${SERVE_MCP_PID:-}" ] && kill -9 "$SERVE_MCP_PID" >/dev/null 2>&1
  [ -n "${UI_PID:-}" ] && kill -9 "$UI_PID" >/dev/null 2>&1
  [ -n "${DAEMON_PID:-}" ] && kill -9 "$DAEMON_PID" >/dev/null 2>&1
  wait >/dev/null 2>&1
  rm -rf "$WORKDIR"
  exit "$ec"
}
trap cleanup EXIT INT TERM

mkdir -p "$(dirname "$OUT_JSON")"

write_failure_json() {
  # Write it on every run, including failure (rule 84d1gfpywd's positive
  # assertion is meaningless if a harness crash just leaves nothing on
  # disk at all).
  local reason="$1"
  jq -n --arg error "$reason" '{success:false, error:$error}' > "$OUT_JSON"
}

fail() {
  echo "live-push-concurrency-check: FATAL: $1" >&2
  write_failure_json "$1"
  exit 1
}

echo "live-push-concurrency-check: workdir=$WORKDIR flushes=$FLUSH_COUNT debounceMs=$DEBOUNCE_MS flushTimeoutMs=$FLUSH_TIMEOUT_MS holdSeconds=$HOLD_SECONDS" >&2

if [ -n "$BIN_OVERRIDE" ]; then
  BIN="$BIN_OVERRIDE"
  echo "live-push-concurrency-check: using pre-built binary $BIN" >&2
else
  echo "live-push-concurrency-check: building codegraph binary" >&2
  GOTOOLCHAIN=go1.26.5 go build -o "$BIN" ./cmd/codegraph || fail "go build ./cmd/codegraph failed"
fi

echo "live-push-concurrency-check: seeding scratch repository at $SCRATCH_REPO" >&2
mkdir -p "$SCRATCH_REPO"
git -C "$SCRATCH_REPO" init -q || fail "git init in scratch repo failed"
printf 'module scratchmod\n\ngo 1.26.5\n' > "$SCRATCH_REPO/go.mod"
printf 'package main\n\nfunc main() {}\n' > "$SCRATCH_REPO/main.go"
git -C "$SCRATCH_REPO" add -A
git -C "$SCRATCH_REPO" -c user.email=probe@example.com -c user.name=probe commit -q -m init || fail "git commit in scratch repo failed"

"$BIN" init "$SCRATCH_REPO" --quiet || fail "codegraph init on scratch repo failed"

ms_now() { node -e 'console.log(Date.now())'; }

# run_flush idx phase
#   phase is "baseline" or "conc" — used only for output file naming.
# Writes $WORKDIR/${phase}-flush-${idx}.json.
run_flush() {
  local idx="$1" phase="$2"
  local token="LivePushFlush${phase}${idx}_$(ms_now)_${RANDOM}"
  local file="$SCRATCH_REPO/livepushprobe_flush_${phase}_${idx}.go"

  local before_lines
  before_lines=$(wc -l < "$DAEMON_LOG" 2>/dev/null | tr -d ' ')
  [ -z "$before_lines" ] && before_lines=0

  local write_time
  write_time=$(ms_now)
  printf 'package scratchmod\n\nfunc %s() {}\n' "$token" > "$file"

  local clock_start=$(( write_time + DEBOUNCE_MS ))
  local debounce_sleep_s
  debounce_sleep_s=$(awk "BEGIN{printf \"%.3f\", ${DEBOUNCE_MS}/1000}")
  sleep "$debounce_sleep_s"

  local deadline=$(( write_time + FLUSH_TIMEOUT_MS ))
  local queryable=false
  local end_time=0
  while true; do
    local now
    now=$(ms_now)
    if [ "$now" -ge "$deadline" ]; then
      break
    fi
    local result
    result=$("$BIN" search "$token" --path "$SCRATCH_REPO" --json 2>/dev/null)
    if [ -n "$result" ] && [ "$result" != "[]" ]; then
      end_time=$(ms_now)
      queryable=true
      break
    fi
    sleep 0.1
  done

  local after_lines
  after_lines=$(wc -l < "$DAEMON_LOG" 2>/dev/null | tr -d ' ')
  [ -z "$after_lines" ] && after_lines=0

  local evidence=""
  if [ "$after_lines" -gt "$before_lines" ]; then
    evidence=$(sed -n "$((before_lines+1)),${after_lines}p" "$DAEMON_LOG" 2>/dev/null | (rg -i 'store-lock race|graphstore: store lock held' || true))
  fi

  local raw_ms=0 adj_ms=0
  if [ "$queryable" = true ]; then
    raw_ms=$(( end_time - write_time ))
    adj_ms=$(( end_time - clock_start ))
    if [ "$adj_ms" -lt 1 ]; then adj_ms=1; fi
  fi

  local starved=false
  if [ "$queryable" = false ] || [ -n "$evidence" ]; then
    starved=true
  fi

  jq -n \
    --argjson index "$idx" \
    --arg token "$token" \
    --argjson queryable "$queryable" \
    --argjson rawMs "$raw_ms" \
    --argjson adjMs "$adj_ms" \
    --argjson starved "$starved" \
    --arg evidence "$evidence" \
    '{index:$index, token:$token, queryable:$queryable, rawMs:$rawMs, adjMs:$adjMs, starved:$starved, evidence:$evidence}' \
    > "$WORKDIR/${phase}-flush-${idx}.json"

  echo "live-push-concurrency-check: [$phase $idx] token=$token queryable=$queryable rawMs=$raw_ms adjMs=$adj_ms starved=$starved" >&2

  # Settle gap BEFORE the next flush's write. The live publisher's own
  # watcher (internal/uiserver's changeDetector) debounces on the SAME
  # CODEGRAPH_DEBOUNCE_MS window over .codegraph/store/, and Pebble's own
  # background churn (compaction, WAL rotation) keeps that debounce timer
  # re-armed for a while after data is already queryable. Measured directly
  # this session against a real running publisher: a 600ms gap between two
  # sequential flushes still coalesced into exactly ONE delivered
  # generation (not two); a 3s gap reliably delivered both as distinct
  # generations. 2.5s is this measurement's own margin, not a guess —
  # smaller values were observed to coalesce.
  sleep 2.5
}

: > "$DAEMON_LOG"

echo "live-push-concurrency-check: starting daemon (this process runs for the WHOLE script, baseline and concurrent phases alike)" >&2
CODEGRAPH_DEBOUNCE_MS="$DEBOUNCE_MS" "$BIN" daemon start --path "$SCRATCH_REPO" --quiet > "$DAEMON_LOG" 2>&1 &
DAEMON_PID=$!
sleep 1
if ! kill -0 "$DAEMON_PID" 2>/dev/null; then
  cat "$DAEMON_LOG" >&2
  fail "daemon start exited immediately"
fi

if [ "$SKIP_BASELINE" = true ]; then
  echo "live-push-concurrency-check: --skip-baseline set — using a floor baseline of 1ms" >&2
  jq -n '{index:0, token:"skipped", queryable:true, rawMs:1, adjMs:1, starved:false, evidence:""}' > "$WORKDIR/baseline-flush-0.json"
else
  echo "live-push-concurrency-check: BASELINE phase — daemon alone, $FLUSH_COUNT real re-index flushes" >&2
  for i in $(seq 1 "$FLUSH_COUNT"); do
    run_flush "$i" baseline
  done
fi

echo "live-push-concurrency-check: CONCURRENT phase begins — starting codegraph ui, the mcp probe, and the stream probe" >&2

"$BIN" ui --no-open --path "$SCRATCH_REPO" > "$UI_LOG" 2>&1 &
UI_PID=$!

UI_URL=""
for _ in $(seq 1 50); do
  if [ -s "$UI_LOG" ]; then
    UI_URL=$(head -n1 "$UI_LOG" | tr -d '\r\n')
    [ -n "$UI_URL" ] && break
  fi
  sleep 0.1
done
if [ -z "$UI_URL" ]; then
  cat "$UI_LOG" >&2
  fail "codegraph ui did not print a URL"
fi
echo "live-push-concurrency-check: ui at $UI_URL (pid $UI_PID)" >&2

GOTOOLCHAIN=go1.26.5 go run "$REPO_ROOT/scripts/live-push-probe.go" mcp \
  --path "$SCRATCH_REPO" --seconds "$HOLD_SECONDS" --bin "$BIN" \
  > "$MCP_PROBE_OUT" 2>"$WORKDIR/mcp-probe.stderr" &
MCP_PROBE_PID=$!

GOTOOLCHAIN=go1.26.5 go run "$REPO_ROOT/scripts/live-push-probe.go" stream \
  --url "$UI_URL" --out "$STREAM_OUT" --seconds "$HOLD_SECONDS" \
  > "$WORKDIR/stream-probe.stdout" 2>"$WORKDIR/stream-probe.stderr" &
STREAM_PID=$!

# Give both probes time to actually establish their sessions (MCP
# initialize handshake; the stream's seed event) BEFORE driving any
# flushes, so every flush below happens genuinely concurrently with all
# three product processes and the harness stream session.
sleep 2

echo "live-push-concurrency-check: CONCURRENT phase — full topology, $FLUSH_COUNT real re-index flushes" >&2
for i in $(seq 1 "$FLUSH_COUNT"); do
  run_flush "$i" conc
done

echo "live-push-concurrency-check: concurrent flushes done — waiting for the mcp probe's own hold period and tool call" >&2
wait "$MCP_PROBE_PID" 2>/dev/null
MCP_PROBE_PID=""

if [ ! -s "$MCP_PROBE_OUT" ]; then
  cat "$WORKDIR/mcp-probe.stderr" >&2
  fail "mcp probe produced no output"
fi

SERVE_MCP_PID=$(jq -r '.serveMcpPid // empty' "$MCP_PROBE_OUT")
MCP_RESPONSE=$(jq -r '.toolCallResponse // ""' "$MCP_PROBE_OUT")
MCP_ALIVE=false
if [ -n "$MCP_RESPONSE" ]; then
  MCP_ALIVE=true
fi

DAEMON_ALIVE=false
if kill -0 "$DAEMON_PID" 2>/dev/null; then DAEMON_ALIVE=true; fi
UI_ALIVE=false
if kill -0 "$UI_PID" 2>/dev/null; then UI_ALIVE=true; fi
STREAM_ALIVE=false
if kill -0 "$STREAM_PID" 2>/dev/null; then STREAM_ALIVE=true; fi

LIVE_EVENTS_RECEIVED=$(wc -l < "$STREAM_OUT" 2>/dev/null | tr -d ' ')
[ -z "$LIVE_EVENTS_RECEIVED" ] && LIVE_EVENTS_RECEIVED=0

# The FIRST event any subscriber receives is the seed (D-04's
# seed-on-subscribe: a fresh subscriber is pre-loaded with the publisher's
# CURRENT state under the same lock that registers it — never a genuinely
# new change). Exclude it by generation: every real triggered flush
# strictly increases the monotonic per-process generation counter, so any
# received generation greater than the FIRST received generation is a
# genuine new event, never the seed.
LIVE_EVENTS_EXCLUDING_SEED=0
if [ "$LIVE_EVENTS_RECEIVED" -gt 0 ]; then
  LIVE_EVENTS_EXCLUDING_SEED=$(jq -s '
    if length == 0 then 0 else
      (.[0].generation) as $seed
      | [.[] | select(.generation > $seed)] | length
    end
  ' "$STREAM_OUT")
fi

echo "live-push-concurrency-check: post-run — daemonAlive=$DAEMON_ALIVE uiAlive=$UI_ALIVE mcpAlive=$MCP_ALIVE (via tool-call response) streamAlive=$STREAM_ALIVE liveEventsReceived=$LIVE_EVENTS_RECEIVED liveEventsExcludingSeed=$LIVE_EVENTS_EXCLUDING_SEED" >&2

# Record the PIDs for the JSON BEFORE tearing anything down — clearing
# DAEMON_PID/UI_PID/STREAM_PID here (so the EXIT trap does not double-kill
# them) must not also erase the values the processes[] array below needs.
DAEMON_PID_RECORDED="${DAEMON_PID:-0}"
UI_PID_RECORDED="${UI_PID:-0}"
STREAM_PID_RECORDED="${STREAM_PID:-0}"

# Tear down the concurrent-phase processes now — everything below is pure
# aggregation over data already captured.
kill -9 "$STREAM_PID" >/dev/null 2>&1; STREAM_PID=""
[ -n "$SERVE_MCP_PID" ] && kill -9 "$SERVE_MCP_PID" >/dev/null 2>&1
kill -9 "$UI_PID" >/dev/null 2>&1
kill -9 "$DAEMON_PID" >/dev/null 2>&1; DAEMON_PID=""; UI_PID=""

jq -s 'sort_by(.index)' "$WORKDIR"/baseline-flush-*.json > "$WORKDIR/baseline-all.json"
jq -s 'sort_by(.index)' "$WORKDIR"/conc-flush-*.json > "$WORKDIR/conc-all.json"

FLUSHES_COMPLETED=$(jq '[.[] | select(.queryable==true)] | length' "$WORKDIR/conc-all.json")
FLUSHES_STARVED=$(jq '[.[] | select(.starved==true)] | length' "$WORKDIR/conc-all.json")
MAX_FLUSH_MS=$(jq '([.[].adjMs] | if length==0 then 1 else max end) as $m | if $m < 1 then 1 else $m end' "$WORKDIR/conc-all.json")
BASELINE_MAX_MS=$(jq '([.[].adjMs] | if length==0 then 1 else max end) as $m | if $m < 1 then 1 else $m end' "$WORKDIR/baseline-all.json")
STARVATION_EVIDENCE_JSON=$(jq '[.[] | select(.evidence != "") | (.evidence | split("\n")) ] | flatten | map(select(length>0))' "$WORKDIR/conc-all.json")
FLUSH_DURATIONS_JSON=$(jq '[.[].adjMs]' "$WORKDIR/conc-all.json")
RAW_ELAPSED_JSON=$(jq '[.[].rawMs]' "$WORKDIR/conc-all.json")
MARKER_TOKENS_JSON=$(jq '[.[] | {token:.token, queryable:.queryable}]' "$WORKDIR/conc-all.json")

MAX_ALLOWED=$(( 3 * BASELINE_MAX_MS + 1000 ))

SUCCESS=true
[ "$FLUSHES_STARVED" -eq 0 ] || SUCCESS=false
[ "$FLUSHES_COMPLETED" -eq "$FLUSH_COUNT" ] || SUCCESS=false
[ "$FLUSH_COUNT" -ge 5 ] || SUCCESS=false
[ "$LIVE_EVENTS_EXCLUDING_SEED" -ge 5 ] || SUCCESS=false
[ "$MAX_FLUSH_MS" -le "$MAX_ALLOWED" ] || SUCCESS=false
[ "$MCP_ALIVE" = true ] || SUCCESS=false
[ "$DAEMON_ALIVE" = true ] || SUCCESS=false
[ "$UI_ALIVE" = true ] || SUCCESS=false

echo "live-push-concurrency-check: flushesAttempted=$FLUSH_COUNT flushesCompleted=$FLUSHES_COMPLETED flushesStarved=$FLUSHES_STARVED maxFlushDurationMs=$MAX_FLUSH_MS baselineMaxFlushDurationMs=$BASELINE_MAX_MS (allowed<=$MAX_ALLOWED) success=$SUCCESS" >&2

PROCESSES_JSON=$(jq -n \
  --argjson daemonPid "$DAEMON_PID_RECORDED" --argjson daemonAlive "$DAEMON_ALIVE" \
  --argjson mcpPid "${SERVE_MCP_PID:-0}" --argjson mcpAlive "$MCP_ALIVE" \
  --argjson uiPid "$UI_PID_RECORDED" --argjson uiAlive "$UI_ALIVE" \
  --argjson streamPid "$STREAM_PID_RECORDED" --argjson streamAlive "$STREAM_ALIVE" \
  '[
    {name:"codegraph daemon start", role:"product", pid:$daemonPid, aliveAtEnd:$daemonAlive},
    {name:"codegraph serve --mcp", role:"product", pid:$mcpPid, aliveAtEnd:$mcpAlive},
    {name:"codegraph ui", role:"product", pid:$uiPid, aliveAtEnd:$uiAlive},
    {name:"live-push-probe stream", role:"harness", pid:$streamPid, aliveAtEnd:$streamAlive}
  ]')

jq -n \
  --argjson flushesAttempted "$FLUSH_COUNT" \
  --argjson flushesCompleted "$FLUSHES_COMPLETED" \
  --argjson flushesStarved "$FLUSHES_STARVED" \
  --argjson starvationEvidence "$STARVATION_EVIDENCE_JSON" \
  --argjson flushDurationsMs "$FLUSH_DURATIONS_JSON" \
  --argjson maxFlushDurationMs "$MAX_FLUSH_MS" \
  --argjson baselineMaxFlushDurationMs "$BASELINE_MAX_MS" \
  --argjson flushTimeoutMs "$FLUSH_TIMEOUT_MS" \
  --argjson debounceExcludedMs "$DEBOUNCE_MS" \
  --argjson rawWriteToQueryableMs "$RAW_ELAPSED_JSON" \
  --argjson flushMarkerTokens "$MARKER_TOKENS_JSON" \
  --argjson liveEventsReceived "$LIVE_EVENTS_RECEIVED" \
  --argjson liveEventsExcludingSeed "$LIVE_EVENTS_EXCLUDING_SEED" \
  --argjson sessionHeldSeconds "$HOLD_SECONDS" \
  --argjson mcpAliveAfterRun "$MCP_ALIVE" \
  --arg mcpToolCallResponse "$MCP_RESPONSE" \
  --argjson daemonAliveAfterRun "$DAEMON_ALIVE" \
  --argjson uiAliveAfterRun "$UI_ALIVE" \
  --argjson processes "$PROCESSES_JSON" \
  --argjson success "$SUCCESS" \
  '{
    flushesAttempted: $flushesAttempted,
    flushesCompleted: $flushesCompleted,
    flushesStarved: $flushesStarved,
    starvationEvidence: $starvationEvidence,
    flushDurationsMs: $flushDurationsMs,
    maxFlushDurationMs: $maxFlushDurationMs,
    baselineMaxFlushDurationMs: $baselineMaxFlushDurationMs,
    flushTimeoutMs: $flushTimeoutMs,
    debounceExcludedMs: $debounceExcludedMs,
    rawWriteToQueryableMs: $rawWriteToQueryableMs,
    flushMarkerTokens: $flushMarkerTokens,
    liveEventsReceived: $liveEventsReceived,
    liveEventsExcludingSeed: $liveEventsExcludingSeed,
    sessionHeldSeconds: $sessionHeldSeconds,
    mcpAliveAfterRun: $mcpAliveAfterRun,
    mcpToolCallResponse: $mcpToolCallResponse,
    daemonAliveAfterRun: $daemonAliveAfterRun,
    uiAliveAfterRun: $uiAliveAfterRun,
    processes: $processes,
    success: $success
  }' > "$OUT_JSON"

echo "live-push-concurrency-check: wrote $OUT_JSON" >&2

if [ "$SUCCESS" = true ]; then
  exit 0
else
  exit 1
fi
