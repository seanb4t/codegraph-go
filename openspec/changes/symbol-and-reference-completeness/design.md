# Design

## Context

See [proposal.md](proposal.md) for motivation and scope. [Issue #175](https://github.com/seanb4t/codegraph-go/issues/175) identifies a broken path through a Go function variable.

The [Go extractor](../../../internal/indexer/goextract/types.go) emits package variables but does not expose struct fields as data nodes. The grouped-variable omission already received a separate fix.

The [TypeScript extractor](../../../internal/indexer/tsextract/types.go), [Python extractor](../../../internal/indexer/pyextract/types.go), and [Java extractor](../../../internal/indexer/javaextract/types.go) do not provide the full proposed declaration contract.

[Query traversal](../../../internal/query/traverse.go) follows call edges for caller relationships. Treating every reference as a call changes that meaning and creates incorrect results.

## Goals / Non-Goals

Goals: Extend existing extractor and resolver seams. Preserve stable identities and explain every derived call through a source binding.

Non-goals: Add production compiler dependencies or a general data-flow engine. Do not change all query traversal to follow arbitrary references.

## Decisions

Reuse the identity finalization and rebuild mechanisms after the `index-invariants` implementation lands. PR #237 proposes that implementation but does not contain it. Do not duplicate occurrence-ID logic. Finalization must remap every node-keyed side table and unresolved reference introduced here.

Use existing `variable` and `constant` kinds where they already apply. Add `field` as an additive kind. Preserve current callable identities for functions stored in exported constants. Do not emit both a function and a data node for the same declaration.

Apply this declaration-kind mapping consistently across extraction and evaluation:

| Language and form | Kind | Boundary |
| --- | --- | --- |
| Go package `var` / `const` | `variable` / `constant` | Explicitly named struct members use `field`. |
| TypeScript, JavaScript, TSX module `var` / `let` / `const` | `variable` / `variable` / `constant` | A direct arrow or function expression uses `function`, including unexported bindings. An alias to another function retains its data kind. |
| TypeScript, JavaScript, TSX class member | `field` | Static, readonly, private, and callable fields remain fields. Preserve modifiers as metadata. |
| Python module assignment or annotation | `variable` | Uppercase spelling does not establish a constant. Class and declared-receiver attributes use `field`. |
| Java class or interface member | `field` | Static, final, and interface fields retain modifiers without changing kind. |

Qualify fields through their declaring type. Collect repeated writes before identity finalization so a Python attribute has one logical owner/name binding. Choose its earliest qualifying declaration or assignment as its source anchor. Keep write sites as reference evidence.

Keep fields out of the bare-name index used for unqualified call resolution, including store-seeded incremental indexes. Resolve field references through owner-qualified lookup. A callable field must pass binding resolution before it can produce a call to a function. A field named `size` must not displace a method named `size`.

Apply discard rules per language. Go's blank identifier and destructuring holes do not declare symbols. A Python or JavaScript variable named `_` remains a valid binding.

Implement this bounded declaration matrix:

| Language | Included declarations | Explicit limits |
| --- | --- | --- |
| Go | Package `var` and `const`, grouped declarations, multi-name specs, explicitly named struct fields | Preserve existing embedding relationships. Implicit embedded-field names and unrestricted local variables remain separate work. |
| TypeScript, JavaScript, TSX | Module `var`, `let`, `const`, identifier destructuring, class fields with static names, private fields | Do not invent names for computed property expressions. Preserve existing callable-constant representation. |
| Python | Module assignments and annotations, named tuple/list targets, class assignments, receiver attributes in instance methods | Identify the actual declared receiver. Do not treat every variable named `self` as a receiver. Dynamic attributes and metaclass effects remain unresolved. |
| Java | Class and interface fields, including multiple declarators and static fields | Record components, enum constants, implicit synthetic fields, and overload resolution remain visible gaps. |

Python class-method receiver attributes require an explicit `classmethod` form and an established class owner. Static methods do not establish an instance receiver. Aliased or custom decorators remain unresolved unless existing evidence establishes the owner.

Resolve direct identifier reads and writes in their lexical scope. Track local bindings only far enough to reject shadowed global names. Resolve member accesses through explicit receiver types or established `this`, `self`, and class owners. Reject ambiguous owners.

For Java and TypeScript, follow an explicit, uniquely resolved base-class chain for accessible fields. Reference the declaring base owner unless a derived declaration shadows that name. Do not resolve inaccessible private fields or ambiguous bases. This does not add arbitrary receiver inference or Go embedded-field promotion.

For Python, a declared instance receiver's assignment establishes a field on its lexical class, even when a base defines a same-named class attribute. Prefer that established instance field for supported reads. Otherwise, follow only a uniquely established class chain. Leave descriptors, dynamic method resolution, and ambiguous class/instance storage unresolved.

An explicit `classmethod` establishes a class receiver for declarations in its lexical class. Do not assume that inherited calls bind `cls` to that class. Preserve one logical field per owner/name and record class or instance assignment evidence. This field identity does not equate their runtime storage.

Represent data access with the existing `references` edge and additive `access` metadata: `read`, `write`, or `read_write`. Merge access roles deterministically because storage combines edges with equal source, kind, and target. Preserve initializer binding evidence with an additive metadata key.

Use the containing function or method as the source of executable reads and writes. Use the data declaration as the source of its initializer relationships. Reuse existing file or module anchors for top-level statements without a declaration owner.

Retain type relationships where the existing extractor can establish them. This change does not infer arbitrary receiver types or broaden interface satisfaction rules.

For callable aliases, retain a reference from the data symbol to the function value. Represent the caller's access to the data binding as a read reference, not a calls edge. Emit a calls edge only to a uniquely established function target. Mark it with static-binding evidence. Initialization alone never emits that call.

Apply the competing-write rule to mutable module bindings even when their direct function expression uses the `function` kind. Kind alone cannot authorize direct call resolution. For `let f = () => 1; f = g; f()`, preserve value references but withhold a definitive target. This preserves existing callable identities without bypassing conservative resolution.

Resolve bounded alias chains with cycle detection. Require a single established target and no conflicting writes in the declared analysis context. A direct import alias can participate. Attribute lookup with an unknown receiver cannot.

Collect syntactic assignments conservatively within each analysis context. A competing write, augmented assignment, unknown target assignment, or relevant parse failure prevents definitive alias resolution. Do not simulate branches or execution order. Known value references remain available in explore even when call resolution is withheld.

For Go, evaluate non-test call sites in `go-production`, which excludes `_test.go` assignments. Evaluate `go-test` with production and test assignments together. Retain test declarations and references in the graph. Attach `binding_context` to each derived call's evidence. A call established only in production makes no claim about the same function's execution under tests.

Test-only callers use the test context. Non-test callers can carry evidence for either context when that context establishes a unique target. Test conflicts remove only test-context evidence. Other languages use the declared indexed scope without filename-based test exclusions. Build tags and platform selection remain explicit evaluation boundaries rather than new production compiler analysis.

When direct and derived calls share source, kind, and target, retain one edge with a sorted, deduplicated evidence set. Preserve direct-call evidence, binding paths, and analysis contexts together. Merge before edge collapse and storage writes so extraction order cannot discard evidence.

These calls describe statically established potential calls, not guaranteed runtime execution. Unindexed code and external mutation remain documented limits. The #175 acceptance includes this repository's `internal/cli/daemon.go` binding and invocation alongside replacements in `internal/cli/daemon_test.go`. Require production call visibility, withheld conflicting test evidence, and a production-reassignment variant. Do not close #175 solely because the declaration becomes searchable.

Keep callers and callees on call edges. Existing impact traversal benefits from the derived call edge. Definitions and explore expose the binding and data-reference path. Broader impact analysis for arbitrary data reads remains separate work.

Audit allowed kinds, result filters, graph presentation, and serialization. Change generated clients only through their owning generator when a protocol source actually changes. String kind values alone do not justify unrelated protocol changes.

Allocate a rebuild capability marker after the `index-invariants` implementation lands. Do not reserve a protobuf field number from an unmerged snapshot. An index without the marker follows the existing clean rebuild path so unchanged files gain new facts.

Compare fresh indexing with edits, renames, deletion, and restoration. Include a field that moves between owners and an alias whose target changes. Identity stability applies to unchanged declarations, not removed or moved symbols.

Persist each file's binding-write contributions and lookup dependencies through the existing schema source and generator. Include unresolved lookups, binding chains, analysis contexts, and every owner/name consulted by inherited-field lookup. Compare old and new contributions for each synchronization batch. Re-resolve dependent files when a write set, owner chain, or shadowing declaration changes, even when those files are unchanged.

Propagate affected dependency keys until the dependent-file set stops growing, with cycle detection. Rebuild their outgoing relationships from the complete post-batch context and commit contributions and facts together. Removing a conflict must restore eligible calls, just as adding one removes them. PR #237 supplies structural invariants, not this new dependency mechanism.

Land shared graph conventions first, followed by language-specific extraction changes and query acceptance. Each language must meet its exact fixture contract before its capability claim changes. Issue #238 supplies independent expectations and exposes remaining failures.

Rollback uses the prior binary and a fresh index in its supported format. Do not rely on older versions preserving newly introduced kinds or metadata.

## Risks / Trade-offs

- Local shadowing can produce false global references. Include negative tests for parameters, blocks, nested functions, and equal names in different owners.
- Python attributes do not always have static owners. Keep unresolved cases visible instead of inventing a class.
- Conservative alias resolution leaves some real calls unresolved. Preserve binding references and report the limitation through #238.
- More nodes can affect ranking and output budgets. Add public query cases with relevant fields and unrelated distractors.
- Rebuilds add one-time work for existing indexes. Reuse the current migration path and test unchanged source files.
