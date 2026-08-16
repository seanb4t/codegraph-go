## ⚠️ Wrong template — please pick the one for your PR type

This is the default template. It is not the one to submit with.

| PR type | When | Template |
|---|---|---|
| **Fix** | Correcting a bug, crash, or behavior that contradicts the docs | [Use fix template](?template=PULL_REQUEST_TEMPLATE/fix.md) |
| **Enhancement** | Improving something that already works — better output, edge cases, performance | [Use enhancement template](?template=PULL_REQUEST_TEMPLATE/enhancement.md) |
| **Feature** | Adding something that does not exist today | [Use feature template](?template=PULL_REQUEST_TEMPLATE/feature.md) |

Selecting one replaces this body. Anything you have already typed here will be
lost, so switch before writing.

---

### Not sure which?

- **Corrects broken behavior** → Fix
- **Improves existing behavior**, no new commands or concepts → Enhancement
- **Adds something that does not exist** → Feature
- **Still unsure** → open a [Discussion](https://github.com/seanb4t/codegraph-go/discussions) first

A change that alters output an agent already consumes is an Enhancement even if
it feels like a fix — output shapes are a compatibility surface here.

---

### Issues are approved before PRs

Link an issue, and for anything beyond a mechanical fix let it be approved first:

- **Features** — the issue should carry `approved-feature`
- **Enhancements** — the issue should carry `approved-enhancement`

This is not ceremony. Behavior-affecting changes are resolved against the
project's recorded decisions in `.planning/` — changes that look like obvious
improvements are sometimes deliberate calls recorded there.
The issue is where that surfaces before you have written the code.

Trivially mechanical fixes — typo, dead link, broken reference — can skip this.
Say so in the PR.

---

### Exempt PRs

CI/tooling, dependency bumps, and doc-only changes do not need a typed template.
Say which applies and carry on:

```
<!-- pr-template-exempt: docs-only -->
```

---

> **These gates are automated.** Draft PRs are closed, a missing issue link
> fails a check, a body that is not a typed template fails a check, and a
> `feat:`/`perf:` PR without an approved issue is closed. Collaborators get a
> warning where outsiders get a failure, and `fix:` never needs approval.
>
> Also enforced, by CI and branch rules: a Conventional-Commits PR title, six
> required checks, resolved review threads, and squash-only merges into a linear
> `main`.
>
> One documented gap rather than an implied guarantee: GitHub does not fire
> `pull_request_target` for fork branches named like a Git SHA, so the
> draft-close can be evaded that way. Such a PR still cannot merge — it fails
> every other gate.

See [CONTRIBUTING.md](../blob/main/CONTRIBUTING.md) for the full process.
