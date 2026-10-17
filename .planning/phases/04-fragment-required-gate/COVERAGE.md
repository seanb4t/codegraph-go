# API Coverage — GitHub REST (pull-request files, repository rulesets)

> Full coverage by default. Opt-outs are explicit, reasoned decisions.
>
> Scope: the GitHub REST surface this phase uses.
>
> 1. The `fragment-required` gate (04-01). It needs exactly one thing from GitHub: the complete list of a pull request's changed files, with each file's status, plus a count to prove the list is complete. It reads the PR body from the event payload (`PR_BODY`, D-02), not from the API, and writes nothing (D-03: `contents: read`, `pull-requests: read`).
> 2. The D-07 ruleset edit (04-05). The executor makes two read-only reads of ruleset 20157557. The single PUT is applied by the maintainer, or by the executor only on an explicit `run-prepared` reply at the blocking-human checkpoint.
>
> All calls go through `gh api` with argument lists.

| capability | decision | reason |
|---|---|---|
| GET /repos/{owner}/{repo}/pulls/{pull_number} (changed_files count) | INTEGRATE | |
| GET /repos/{owner}/{repo}/pulls/{pull_number}/files (paginated) | INTEGRATE | |
| GET /repos/{owner}/{repo}/rulesets/{ruleset_id} | INTEGRATE | |
| PUT /repos/{owner}/{repo}/rulesets/{ruleset_id} | INTEGRATE | |
| GraphQL pullRequest.files (`gh pr view --json files`) | OPT-OUT | not needed: it stops at 100 files with no CLI pagination (RESEARCH Pitfall 1), and the paginated REST listing above is complete |
| GET /repos/{owner}/{repo}/compare/{basehead} | OPT-OUT | not needed: it describes a commit range rather than the PR's own change set, and stops at 300 files |
| pulls/{pull_number} `body` field | OPT-OUT | not needed: the body arrives through the event payload in PR_BODY (D-02), which also defines GATE-03's unreadable-body state |
| POST/PATCH /repos/{owner}/{repo}/issues/{issue_number}/comments | OPT-OUT | explicitly out of scope: D-03, the gate posts no PR comment; its guidance is the check output |
| POST check-runs and commit statuses (write) | OPT-OUT | not needed: the job's own exit status is the `fragment-required` check, so no write scope is held |
| GET pulls/{pull_number}/commits, /reviews, /requested_reviewers | OPT-OUT | not needed: the verdict depends only on the changed files and the body |
| GET /repos/{owner}/{repo}/contents/{path} (reading a fragment's YAML) | OPT-OUT | not needed: D-04 defines a fragment by path and ADDED status; changie validates fragment content (CHG-04) |
| other ruleset and rule-suite endpoints | OPT-OUT | not needed: one fixed ruleset id gains one context; no listing, creation or history |
| classic branch-protection endpoints | OPT-OUT | not needed: main is protected by ruleset 20157557, not classic branch protection |
