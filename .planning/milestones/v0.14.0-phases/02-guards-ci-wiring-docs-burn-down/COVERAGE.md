# API Coverage — GitHub REST (repository rulesets)

> Full coverage by default. Opt-outs are explicit, reasoned decisions.
>
> Scope: the single external API read this phase integrates — GRD-12's ruleset-drift CI step
> (D-05/D-06). The step is a read-only equality check of the live `protect-main` ruleset's
> `required_status_checks[].context` set against `.github/required-status-checks.txt`. It never
> writes, never reads history, and never touches rule-suites. `Metadata:read` is the only scope;
> the request works unauthenticated on this public repository and uses the default `GITHUB_TOKEN`
> in CI only to lift the unauthenticated rate limit (`contents: read` at the workflow level already
> grants `Metadata: read`).

| capability | decision | reason |
|---|---|---|
| GET /repos/{owner}/{repo}/rulesets/{ruleset_id} | INTEGRATE | |
| all other repository/organization ruleset and rule-suite endpoints | OPT-OUT | not needed: the step is a read-only equality check of `required_status_checks[].context` on ruleset 20157557; no write, no history, no rule-suites, no listing (the id is fixed by the maintainer's ruleset, not discovered) |
