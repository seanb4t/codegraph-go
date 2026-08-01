# Security Policy

## Reporting a vulnerability

**Please do not open a public issue for a security vulnerability.**

Report privately via GitHub's
[Security Advisories](https://github.com/seanb4t/codegraph-go/security/advisories/new)
form, which creates a channel visible only to maintainers.

Please include: what an attacker can do, how to reproduce it, the affected
version (`codegraph version`), and your platform. A proof of concept helps but
is not required to report.

This is a small project without a funded security team. You should expect an
acknowledgement within a week, and an honest estimate rather than a guaranteed
fix window. If a report is judged not to be a vulnerability, you'll get the
reasoning, not silence.

## Supported versions

Pre-1.0, only the **latest release** receives fixes. There are no maintained
release branches. Fixes ship as the next tagged release.

## Verifying what you run

Every release binary is signed with cosign keyless signing, and the signing
identity is compiled into the binary itself. Verification instructions are in
[`docs/RELEASE.md`](docs/RELEASE.md); the short version:

```sh
cosign verify-blob \
  --bundle codegraph_<tag>_<goos>_<goarch>.sigstore.json \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity-regexp \
    '^https://github\.com/seanb4t/codegraph-go/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^[:space:]]*$' \
  codegraph_<tag>_<goos>_<goarch>
```

The identity regex is anchored at both ends and scoped to one workflow file and
a tag-triggered ref. That is deliberate: an unanchored or prefix-only pattern
would accept a signature produced by *any* workflow in this repository,
including ones running against untrusted pull-request code.

SLSA provenance covers the platform binaries directly and is verifiable with
`slsa-verifier verify-artifact --source-tag <tag>`.

**If verification fails, do not run the binary.** Report it through the channel
above — a signature that doesn't verify is itself a security report.

## Threat model

What this project defends against:

- **Tampered release artifacts.** Signature verification is enforced by
  `codegraph upgrade` against a compiled-in identity, not a configurable one.
- **A malicious release built from elsewhere.** The SAN pattern binds signatures
  to this repository's release workflow at a tag ref.
- **Dependency vulnerabilities.** `govulncheck` gates every merge; it is
  call-graph aware, so it flags reachable vulnerabilities rather than every CVE
  in the dependency tree.

What it does **not** defend against, stated plainly:

- **Malicious source code in a repository you index.** `codegraph` parses code;
  it does not execute it. But tree-sitter grammars include hand-written C
  scanners, and a memory-safety bug in one could in principle be reached by a
  crafted input file. Parsing runs in-process, so there is no sandbox boundary.
  Do not point `codegraph` at code you would not open in an editor.
- **A compromised GitHub account or Actions environment.** Signing identity is
  derived from the workflow; an attacker with commit access to the release
  workflow could produce validly-signed artifacts.
- **The graph as a confidentiality boundary.** `.codegraph/` contains source
  excerpts from the indexed repository, in plaintext. It inherits the
  filesystem's permissions and nothing more. Do not commit it, and do not treat
  it as safe to share when the source is not.
- **MCP transport security.** `serve --mcp` speaks stdio to a local client and
  performs no authentication, by design. Do not expose it over a network.

## Scope

In scope: signature or provenance verification that can be bypassed; path
traversal or arbitrary write via indexing, migration, or the MCP surface;
privilege escalation through the installer or git hooks; anything letting a
crafted repository achieve code execution.

Out of scope: vulnerabilities requiring an already-compromised machine or
account; denial of service from deliberately pathological inputs on a local tool
you invoked yourself; findings in dependencies with no reachable call path from
this codebase (`govulncheck` is the arbiter).
