---
created: 2026-08-10T00:00:00.000Z
title: brew trust instructions recommend the broader --tap grant and carry no security framing
area: docs
severity: medium
files:
  - docs/RELEASE.md:518-533
  - README.md:78
threat_ref: UF-2 (03-SECURITY.md)
---

## Problem

Homebrew 6.0.16 refuses to install casks from untrusted third-party taps, so the
published contract requires the user to run `brew trust` before `brew install`.
Two things are wrong with how that step is currently presented.

**1. No security framing.** `docs/RELEASE.md:518-533` and `README.md:78` instruct
the user to run the command the error message names, without stating what the
control *is*. Homebrew refuses because a third-party cask executes arbitrary Ruby
on the user's machine at install time — which this cask's own `postflight` block
(`.goreleaser.yaml:526-594`) does. We are asking every user to opt out of a
security control and telling them nothing about it.

**2. We recommend the broader grant.** The primary instruction is:

    brew trust --tap seanb4t/tap

which grants trust to **every current and future cask and command in the tap**.
Homebrew's own error offers the narrower form:

    brew trust --cask seanb4t/tap/codegraph

which grants trust to this one cask. We should recommend the narrow form and
mention the broad one only as an explicit convenience opt-in.

## Why it matters

Impact is bounded — T-03-07 and T-03-12 establish that the tap-writing credential
is scoped to `homebrew-tap` alone and was proven refused (`403`) against
`codegraph-go`, so a tap compromise cannot reach the release repository. But the
blast radius of the *user-side* grant is set by which form we tell them to run,
and we currently tell them to run the wider one.

Likelihood of the step is 100% (it is mandatory to install). Likelihood of
exploitation requires tap compromise.

## Why no threat row covers it

The Phase 3 threat register was authored before Homebrew 6.0.16's refusal was
discovered. Every local rehearsal missed it because a `file://` tap never crosses
the trust boundary. No row could have anticipated it.

## Fix shape

- Recommend `brew trust --cask seanb4t/tap/codegraph` as the primary form.
- Add one sentence naming what the control is and why Homebrew asks: a
  third-party cask runs arbitrary Ruby at install time, and this one does.
- Keep `--tap` documented as an explicit, labelled convenience for users who
  expect to install more than one cask from this tap.
