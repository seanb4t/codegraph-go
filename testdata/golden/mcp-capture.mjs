#!/usr/bin/env node
// testdata/golden/mcp-capture.mjs
//
// Drives the live TS CodeGraph 1.3.1 stdio MCP server to capture golden
// `codegraph_explore`/`codegraph_node` TOOL outputs — the MCP-surface half
// of D-01/D-06's parity oracle that `capture.sh`'s CLI-only invocations
// never exercised (RESEARCH.md Pitfall 3 / "No Analog Found": no existing
// Go or shell code in this repo drives an MCP server as a test client).
//
// Spawns `codegraph serve -p <projectPath> --mcp` (CODEGRAPH_MCP_TOOLS=
// explore,node — TS gates `node` off by default, per
// mcp/tools.js:668/DEFAULT_MCP_TOOLS), sends a minimal JSON-RPC 2.0
//
// DO NOT "fix" that env value to match this repo's Go server. This script
// drives the LIVE TypeScript CodeGraph 1.3.1 binary (see the version gate
// below), where CODEGRAPH_MCP_TOOLS is still an OPT-IN allowlist and `node`
// must be named to appear at all. The Go server inverted the same variable
// into an opt-out NARROWING filter (all eight tools by default), so the
// identical value means something different on each side — on the Go server
// this value would narrow the surface to explore+node rather than widen it
// to explore+node. Both land on the same two tools here, which is exactly
// what makes this an easy thing to get quietly wrong: the value is correct
// for this oracle for the OPPOSITE reason it would be on the Go side.
// initialize -> initialized -> tools/call handshake, and writes each tool's
// text result through the same wrap_text-equivalent {command, output}
// envelope and <CORPUS_PATH> normalization capture.sh's CLI path uses
// (D-02 canonicalization), so MCP and CLI fixtures are diffable with
// identical tooling.
//
// Usage:
//   node mcp-capture.mjs <corpusName> <projectPath> <exploreQuery> <nodeSymbol> <outDir>
//
// Requires: the live TS `codegraph` CLI (v1.3.1) on PATH. Gated by
// `codegraph --version` before spawning the MCP server — D-01: never
// silently capture golden fixtures against the wrong oracle version.

import { spawn, execFileSync } from 'node:child_process';
import { writeFileSync } from 'node:fs';
import { resolve as resolvePath } from 'node:path';

const REQUIRED_VERSION = '1.3.1';
const INIT_TIMEOUT_MS = 15000;
const CALL_TIMEOUT_MS = 30000;

function fail(msg) {
  console.error(`mcp-capture.mjs: ${msg}`);
  process.exit(1);
}

const [, , corpusName, projectPathArg, exploreQuery, nodeSymbol, outDir] = process.argv;
if (!corpusName || !projectPathArg || !exploreQuery || !nodeSymbol || !outDir) {
  fail(
    'usage: mcp-capture.mjs <corpusName> <projectPath> <exploreQuery> <nodeSymbol> <outDir>',
  );
}

const projectPath = resolvePath(projectPathArg);

// --- version gate (D-01): hard-fail rather than silently capture against
// the wrong TS version, or a missing TS install. ---
let version;
try {
  version = execFileSync('codegraph', ['--version'], { encoding: 'utf8' }).trim();
} catch (err) {
  fail(`could not run 'codegraph --version' — is TS codegraph on PATH? (${err.message})`);
}
if (version !== REQUIRED_VERSION) {
  fail(
    `live TS codegraph version is "${version}", expected "${REQUIRED_VERSION}" — ` +
      'refusing to capture golden fixtures against the wrong oracle (D-01)',
  );
}

// --- normalization: mirrors capture.sh's strip_json <CORPUS_PATH> rule for
// the machine-local absolute projectPath (README.md "Volatile fields"). ---
function normalize(text) {
  return text.split(projectPath).join('<CORPUS_PATH>');
}

function wrap(command, output) {
  return JSON.stringify({ command, output: normalize(output) }, null, 2) + '\n';
}

// --- spawn the TS stdio MCP server; explore+node both enabled (node is
// off by default in TS's DEFAULT_MCP_TOOLS). ---
const child = spawn('codegraph', ['serve', '-p', projectPath, '--mcp'], {
  stdio: ['pipe', 'pipe', 'pipe'],
  env: { ...process.env, CODEGRAPH_MCP_TOOLS: 'explore,node' },
});

let stdoutBuf = '';
let stderrBuf = '';
const pending = new Map();

child.stdout.on('data', (chunk) => {
  stdoutBuf += chunk.toString();
  let newlineIdx;
  while ((newlineIdx = stdoutBuf.indexOf('\n')) >= 0) {
    const line = stdoutBuf.slice(0, newlineIdx);
    stdoutBuf = stdoutBuf.slice(newlineIdx + 1);
    if (!line.trim()) continue;
    let msg;
    try {
      msg = JSON.parse(line);
    } catch {
      continue; // ignore non-JSON-RPC lines on stdout, if any
    }
    if (msg.id !== undefined && pending.has(msg.id)) {
      const resolveFn = pending.get(msg.id);
      pending.delete(msg.id);
      resolveFn(msg);
    }
  }
});
child.stderr.on('data', (chunk) => {
  stderrBuf += chunk.toString();
});
child.on('error', (err) => {
  fail(`failed to spawn 'codegraph serve --mcp': ${err.message}`);
});

let nextId = 1;
function call(method, params, timeoutMs) {
  return new Promise((resolveCall, rejectCall) => {
    const id = nextId++;
    const timer = setTimeout(() => {
      pending.delete(id);
      rejectCall(new Error(`timed out waiting for ${method} response (id=${id})`));
    }, timeoutMs);
    pending.set(id, (msg) => {
      clearTimeout(timer);
      resolveCall(msg);
    });
    child.stdin.write(JSON.stringify({ jsonrpc: '2.0', id, method, params }) + '\n');
  });
}
function notify(method, params) {
  child.stdin.write(JSON.stringify({ jsonrpc: '2.0', method, params }) + '\n');
}

function toolText(resp, toolName) {
  if (resp.error) {
    throw new Error(`${toolName} returned a JSON-RPC error: ${JSON.stringify(resp.error)}`);
  }
  const content = resp.result?.content;
  if (!Array.isArray(content) || content.length === 0 || content[0].type !== 'text') {
    throw new Error(
      `${toolName} returned an unexpected content shape: ${JSON.stringify(resp.result)}`,
    );
  }
  return content.map((c) => c.text).join('\n');
}

(async () => {
  try {
    await call(
      'initialize',
      {
        protocolVersion: '2025-06-18',
        capabilities: {},
        clientInfo: { name: 'codegraph-go-golden-capture', version: '0.0.1' },
      },
      INIT_TIMEOUT_MS,
    );
    notify('notifications/initialized', {});

    const exploreResp = await call(
      'tools/call',
      { name: 'codegraph_explore', arguments: { query: exploreQuery, projectPath } },
      CALL_TIMEOUT_MS,
    );
    const exploreText = toolText(exploreResp, 'codegraph_explore');
    writeFileSync(
      `${outDir}/explore-mcp.json`,
      wrap(
        `mcp codegraph_explore query=${JSON.stringify(exploreQuery)} -p ${corpusName}`,
        exploreText,
      ),
    );

    const nodeResp = await call(
      'tools/call',
      {
        name: 'codegraph_node',
        // includeCode:true matches the TS CLI's `node` command, which has
        // no flag to suppress full bodies (its multi-def path always
        // renders them) — the MCP tool's own default is false. Passing it
        // explicitly keeps the CLI and MCP captures testing the same
        // semantic scenario (full multi-def bodies) rather than
        // introducing an incidental TS-native CLI/MCP default asymmetry
        // unrelated to what NODE-04/EXPL-05 actually port.
        arguments: { symbol: nodeSymbol, projectPath, includeCode: true },
      },
      CALL_TIMEOUT_MS,
    );
    const nodeText = toolText(nodeResp, 'codegraph_node');
    writeFileSync(
      `${outDir}/node-mcp.json`,
      wrap(`mcp codegraph_node symbol=${JSON.stringify(nodeSymbol)} -p ${corpusName}`, nodeText),
    );

    child.kill();
    process.exit(0);
  } catch (err) {
    process.stderr.write(stderrBuf);
    fail(err.message ?? String(err));
  }
})();
