# Spec Delta

## Purpose

Give maintainers reproducible evidence about extracted facts and useful query answers. Expose gaps that graph consistency and regression snapshots cannot detect.

## ADDED Requirements

### Requirement: Explicit evaluation inventory

The evaluation MUST inventory Go, TypeScript, JavaScript, TSX, Python, and Java separately. Each construct and query case MUST state its scope, evidence source, expected facts, and evaluation status. Missing or unsupported cases MUST remain visible.

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
- **THEN** the report classifies the target as unresolved or outside the reference scope
- **AND** the report does not count that target as a correct resolved edge

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
- **THEN** those declarations remain in the source-coverage denominator
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

The inventory MUST include omissions, incorrect extra edges, shadowing, repeated names, malformed input, and cross-file resolution. Manifests MUST distinguish development and acceptance partitions. Routine tuning MUST exclude acceptance cases, while pull request evaluation MUST run them. Reports MUST name the partition and expectation revision. Aggregate counts MUST NOT substitute for these cases.

#### Scenario: Unrelated corpus edges mask missing TypeScript edges
- **WHEN** a repository contains many Go edges and no expected TypeScript edge
- **THEN** the TypeScript acceptance case fails regardless of the total graph size

#### Scenario: An acceptance fixture changes
- **WHEN** an acceptance expectation requires revision
- **THEN** a separate review records its language-contract reason and source evidence before the revised result establishes support

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

#### Scenario: A known omission is fixed
- **WHEN** an expected mismatch no longer occurs
- **THEN** the gate requires removal of the stale exception
- **AND** the corrected fact increases the measured accuracy
- **AND** the case changes from known-gap to supported

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
