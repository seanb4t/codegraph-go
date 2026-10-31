---
name: gh-issue-draft
description: File or rewrite a GitHub issue for this repository from a kind (bug, enhancement, feature, chore) and the facts for its body. Reads the matching issue template for sections and labels, writes the body in plain English, and runs gh issue create or gh issue edit. Use when asked to open, file, draft or rewrite an issue here.
argument-hint: "<bug|enhancement|feature|chore> <title> | edit <number>"
model: haiku
context: fork
agent: general-purpose
background: false
allowed-tools: Read, Write, Edit, Bash(gh issue:*), Bash(gh label:*), Bash(mktemp:*)
---

# GitHub issue draft

This skill files one issue, or rewrites one, in this repository. It runs in a
forked subagent on Haiku. It does not open pull requests and it does not post
comments.

## Inputs

The caller gives a kind, a title and the facts for the body. For a rewrite,
the caller gives the issue number instead of a kind. Keep the kind an existing
issue already has.

| kind | template | title prefix | label |
|---|---|---|---|
| `bug` | `.github/ISSUE_TEMPLATE/bug_report.yml` | `fix: ` | `bug` |
| `enhancement` | `.github/ISSUE_TEMPLATE/enhancement.yml` | `feat: ` | `enhancement` |
| `feature` | `.github/ISSUE_TEMPLATE/feature_request.yml` | `feat: ` | `feature` |
| `chore` | `.github/ISSUE_TEMPLATE/chore.yml` | `chore: ` | `chore` |

A chore title adds a scope in parentheses when the facts name one, for
example `chore(test): ...` or `chore(ci): ...`.

## Steps

1. Read the template file for the kind. Each body element's `label:` is a
   section heading, written as `### <label>` in that order. A `dropdown`
   section holds one of its `options`. A `checkboxes` section reproduces
   every option as a Markdown checkbox, with `[x]` only where the facts
   support it. An element with `validations.required: true` must have
   content.
2. Search for a duplicate: `gh issue list --state all --search "<title
   words> in:title" --limit 10`. Name any related issue in the body as
   `#<number>`. Check the "Before submitting" box only after this search.
3. Pick area labels from `gh label list`: `area:ci`, `area:indexer`,
   `area:mcp`, `area:agents` or `area:ui`, when the facts place the issue in
   one of those areas. Do not add `needs-triage` (a workflow adds it), and do
   not add a `priority:*` or `approved-*` label (the maintainer adds those).
4. Write the body to a file outside the repository, in the session
   scratchpad or a `mktemp` directory. Never write it inside the repository.
   Follow the writing rules below. If a lint hook reports violations after
   the write, fix them and write again until it reports none.
5. End the body with the attribution lines from the session's instructions,
   verbatim. When the session gives none, add no footer.
6. Create or edit the issue with one `gh` command, not chained with other
   commands:
   - create: `gh issue create --title "<prefix><title>" --label <label>
     [--label area:<x>] --body-file <path>`
   - rewrite: `gh issue edit <number> --body-file <path>`, and `--title`
     only when the caller asks for a new title.
7. Report the issue URL, the labels set, and any duplicate the search found.

## Writing rules

The body is for a reader who was not in the conversation.

- Each section states facts with their evidence: a command and its output, a
  CI run id, a PR or issue number, a file path with a line number.
- One topic per paragraph, and at most six sentences per paragraph.
- A descriptive sentence has at most 25 words. An instruction has at most 20
  words and gives one instruction.
- Active voice and simple tenses. Name the actor.
- Use `can`, `will` and `must`. Do not use `should`, `would`, `may`, `might`
  or `could`.
- Put a condition before its command: "If the build fails, read the log."
- Write "whether" for an open question, not a trailing "if".
- No contractions, no semicolons, no em dashes. Use "because", "but" or two
  sentences instead.
- Keep code, identifiers, commands, paths, quoted errors and product names
  exactly as they are, in backticks.
- A list is for three or more parallel items or steps. A table is for a
  comparison with the same columns per row.
- Say what is wrong or missing, not "TODO". Delete words such as "simply",
  "robust", "comprehensive", "leverage" and "in order to".

## Repository notes

- A bug title starts with `fix: `, and both feature and enhancement titles
  start with `feat: `. The template's `title:` field is the source of truth
  when these change.
- The chore template asks "Does this touch anything load-bearing?". Tick a
  box only when the change edits that artifact, and say in the body when a
  review reads one without changing it.
- An issue that a pull request will close is named in that PR as
  `Closes #<number>`. A proposal-only PR says `Refs #<number>`.
