# Spike Benchmark Corpus — Provenance

Pinned per RESEARCH §"Open Questions" #1: the parser spike's benchmark corpus
is a pinned-commit public OSS repo (not a machine-local path), so throughput
numbers in `PARSER-DECISION.md` reproduce across machines/CI without cloning
anything at test time. Files below were extracted once from the pinned commits
and committed as static test fixtures.

## Go corpus — `spf13/cobra`

- Source: https://github.com/spf13/cobra
- Pinned ref: tag `v1.8.1`
- Pinned commit: `e94f6d0dd9a5e5738dca6bce03c4b1207ffbc0ec`
- License: Apache-2.0
- Selection: all top-level `*.go` files excluding `*_test.go` (14 files, ~220 KiB)
- Path: `tools/spike/testdata/go/`

## Python corpus — `pallets/flask`

- Source: https://github.com/pallets/flask
- Pinned ref: tag `3.0.3`
- Pinned commit: `c12a5d874c5a014495eb2db8a73f40037bc813ac`
- License: BSD-3-Clause
- Selection: all `src/flask/*.py` files (18 files, ~244 KiB)
- Path: `tools/spike/testdata/python/`

Both repos exercise real, idiomatic source at moderate scale (not synthetic
micro-benchmarks) while staying small enough to commit as fixtures. Flask
additionally exercises meaningful Python indentation depth/complexity, the
grammar whose external INDENT/DEDENT scanner is the crash-isolation dimension
this spike measures (D-05).
