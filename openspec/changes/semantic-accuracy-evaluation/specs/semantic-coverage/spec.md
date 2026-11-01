# Spec Delta

## Purpose

Give maintainers reproducible evidence about extracted facts and useful query answers. Expose gaps that graph consistency and regression snapshots cannot detect.

## ADDED Requirements

### Requirement: Explicit evaluation inventory

The evaluation MUST inventory Go, TypeScript, JavaScript, TSX, Python, and Java separately. Each construct and query case MUST state its scope, target specification revision, evidence source, expected facts, and status. Evaluated planned forms within that target contract MUST use known-gap status until supported. Explicitly excluded forms MUST use unsupported status. Missing or unsupported cases MUST remain visible.

#### Scenario: A language lacks a feature
- **WHEN** Python attribute extraction has no passing acceptance case
- **THEN** the report identifies that gap instead of inferring support from successful function extraction

#### Scenario: An expected input disappears
- **WHEN** a fixture or required language contributes no inputs
- **THEN** the evaluation fails and names the missing input

### Requirement: Independent expected facts

The evaluation MUST derive expected facts from reviewed source annotations or an independent compiler interface. CodeGraph output MUST NOT supply expected facts. Each compiler adapter MUST declare which facts it can establish.

#### Scenario: A whole declaration disappears
- **WHEN** CodeGraph omits every node for a grouped variable declaration
- **THEN** the comparison reports the independently expected declarations as missing

#### Scenario: Dynamic behavior exceeds reference evidence
- **WHEN** an independent parser cannot determine a runtime call target
- **THEN** the report classifies any emitted target edge as unknown and outside reference scope
- **AND** that edge enters neither accuracy denominator nor true positives

#### Scenario: Reviewed source forbids an edge
- **WHEN** CodeGraph emits a relationship that a reviewed expectation explicitly forbids
- **THEN** the relationship counts as a false positive rather than an unknown observation

### Requirement: Comparable semantic identities

The evaluation MUST match declarations one-to-one by language, path, owner, role, name, and independent identifier occurrence. Each construct MUST define its declaration-span rule. Identifier and declaration spans MUST NOT require equal endpoints. Internal node IDs MUST NOT define expected identity. Positions MUST normalize to UTF-8 bytes.

#### Scenario: Repeated names and Unicode
- **WHEN** separate owners contain equal names after non-ASCII source text
- **THEN** the comparison distinguishes the declarations and maps their source positions correctly

#### Scenario: Two calls produce one graph edge
- **WHEN** two call sites share the same source node, target node, and relationship kind
- **THEN** the graph-level denominator counts one expected edge
- **AND** any call-site measurement uses a separately labeled denominator

#### Scenario: Several Go names share a declaration span
- **WHEN** a Go variable spec contains two independently identified names within one declaration span
- **THEN** the comparison matches each binding exactly once through its name, owner, and construct span rule
- **AND** a wrong span or an ambiguous same-name match fails acceptance

### Requirement: Accuracy and coverage accounting

The report MUST show true positives, false positives, false negatives, precision, and recall by language and fact kind, with raw denominators. Unknown facts MUST appear separately and MUST NOT enter either accuracy denominator or true positives. Excluded inputs, parse failures, and known gaps MUST remain visible. A zero denominator MUST display as unavailable.

#### Scenario: A graph contains only functions
- **WHEN** the source also contains independently expected variables
- **THEN** the report shows missing variables and reduced declaration recall

#### Scenario: A file fails to parse
- **WHEN** the reference expects declarations in that file
- **THEN** those declarations remain in the declaration recall denominator as false negatives
- **AND** the report distinguishes parse failure from successful extraction

#### Scenario: A source sample has unlabeled facts
- **WHEN** a sample has ten evaluated emitted facts and five observations without expected labels
- **THEN** the precision denominator is ten and the unknown observation count is five
- **AND** the unknown observations do not increase recall

### Requirement: Public query acceptance

The evaluation MUST exercise definitions, callers, callees, impact, and explore through public interfaces. Cases MUST declare required facts, forbidden facts, and applicable query limits. CLI and MCP agreement alone MUST NOT establish correctness.

#### Scenario: Both interfaces omit a caller
- **WHEN** CLI and MCP return matching answers that omit an independently expected caller
- **THEN** both answers fail the semantic acceptance case

#### Scenario: A bounded explore response
- **WHEN** an explore case specifies a fixed output budget
- **THEN** the evaluation measures its reviewed required facts within that budget
- **AND** the report distinguishes truncation from a missing graph fact

### Requirement: Adversarial and held-out cases

The inventory MUST include omissions, extra edges, shadowing, repeated names, malformed input, and cross-file resolution. Manifests MUST distinguish development and acceptance partitions. `task eval:semantic:dev` MUST run only development cases. `task eval:semantic` and pull request CI MUST run both partitions. Reports MUST name the partition and expectation revision. Aggregate counts MUST NOT substitute for these cases.

#### Scenario: Unrelated corpus edges mask missing TypeScript edges
- **WHEN** a repository contains many Go edges and no expected TypeScript edge
- **THEN** the TypeScript acceptance case fails regardless of the total graph size

#### Scenario: An acceptance fixture changes
- **WHEN** an acceptance expectation requires revision
- **THEN** the report names the changed expectation revision and its acceptance partition

### Requirement: Reproducible evaluation inputs

The evaluation MUST record source revisions, source digests, reference tool versions, build context, configuration, and expectation revision. Small fixtures MUST run without network access. Scheduled source collections MUST use immutable revisions and report missing inputs as failures.

#### Scenario: A reference tool is absent
- **WHEN** a required compiler adapter cannot run
- **THEN** the evaluation reports an infrastructure failure and returns failure
- **AND** it does not report the affected cases as passing or silently skipped

### Requirement: Honest regression gates

Cases MUST distinguish supported, known-gap, unsupported, and unevaluated status. Supported cases MUST have zero missing or extra facts. Only known-gap cases can carry temporary exceptions with exact mismatches, an owner, an issue, and a review date. These mismatches MUST reduce accuracy. New mismatches and expired or stale exceptions MUST fail the gate.

#### Scenario: A known omission grows
- **WHEN** a known failure lists one missing variable but the new result misses two
- **THEN** the gate fails and names the additional omission

#### Scenario: A planned field is not implemented yet
- **WHEN** the target specification includes a #239 field form that the current extractor omits
- **THEN** its manifest uses known-gap status with an exact exception linked to #239
- **AND** the missing field reduces recall before the implementation lands

#### Scenario: A known omission is fixed
- **WHEN** every listed mismatch of a known-gap case disappears and no new mismatch occurs
- **THEN** the gate requires removal of the stale exception
- **AND** the corrected fact increases the measured accuracy
- **AND** the case changes from known-gap to supported

#### Scenario: Only part of a known gap is fixed
- **WHEN** one listed mismatch disappears but others remain
- **THEN** the gate requires removal of the repaired mismatch from the exception
- **AND** the case remains known-gap with its remaining exact differences

### Requirement: Unsupported and unevaluated accounting

Unsupported constructs outside the declared product contract and unevaluated cases without independent evidence MUST remain visible as separate counts. Their facts MUST stay outside both accuracy denominators and the regression gate. Neither status can establish support. Authored manifests MUST define the evaluated scope rather than accepting every reference-tool fact as an expectation.

#### Scenario: A reference emits an out-of-scope Java record component
- **WHEN** its manifest labels that construct unsupported
- **THEN** the report counts it separately without a gate exception or accuracy denominator contribution
- **AND** no passing record-component support claim follows

### Requirement: Semantic lifecycle agreement

The evaluation MUST compare fresh and incremental semantic facts and public answers after no-op synchronization, edits, renames, deletion, and restoration. Both paths MUST use the same configuration and analysis context. Each comparison MUST name its evaluated fact subset and excluded categories. Exclusions MUST NOT count as passing facts.

#### Scenario: An edit changes a binding target
- **WHEN** incremental indexing processes an edit and a fresh index processes the resulting sources
- **THEN** their evaluated declarations, relationships, and public answers agree
- **AND** both reports retain any excluded route or unsupported fact categories

### Requirement: Evidence-backed support claims

Published language capability claims MUST identify their evaluated constructs and unresolved limits. Successful file indexing MUST NOT imply semantic completeness. An aggregate score MUST NOT hide a failing language or construct category.

#### Scenario: Structural tests pass while semantic tests fail
- **WHEN** the graph contains valid IDs and edges but lacks expected fields
- **THEN** the report distinguishes structural success from semantic failure
- **AND** the language documentation does not claim complete field support
