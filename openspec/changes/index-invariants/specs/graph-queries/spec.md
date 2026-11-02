# Spec Delta

## Purpose

How a query names a symbol that has several overloads, and how it reports across them, in the CLI and the MCP tools.

## ADDED Requirements

### Requirement: A bare name that is a set reports across its overloads

A query can name a symbol by its bare name. When that name has an `overload_set` node, `callers`, `callees` and `impact` SHALL report across every overload in the set. Each result SHALL carry the signature of the overload it belongs to. The ambiguous callers of the set SHALL be included and marked ambiguous.

#### Scenario: Callers of an overloaded name

- **WHEN** `Logger` declares `log(String)` and `log(String, Throwable)`, one file calls `log("a")`, another calls `log("b", e)`, and a user runs `codegraph callers log`
- **THEN** both calls are listed, each with the signature of the overload it targets

#### Scenario: An ambiguous caller

- **WHEN** a file calls `add(null)` where `add(int)` and `add(String)` exist, and a user runs `codegraph callers add`
- **THEN** the call is listed once and marked ambiguous

### Requirement: A signature selects one overload

`node`, `callers`, `callees` and `impact` SHALL accept `--signature <types>`, which selects one overload by its parameter types. They SHALL keep accepting `--line` for the same purpose. `callers` on one overload SHALL also list the set's ambiguous callers, marked ambiguous, and SHALL NOT list a caller that resolved to another overload.

#### Scenario: Callers of one overload

- **WHEN** `log(String)` and `log(String, Throwable)` exist, one file calls `log("a")` and another calls `log("b", e)`, and a user runs `codegraph callers log --signature "String, Throwable"`
- **THEN** only the call `log("b", e)` is listed

#### Scenario: A signature that matches no overload

- **WHEN** a user runs `codegraph callers log --signature "int"` and no `log(int)` exists
- **THEN** the command reports that no overload of `log` has that signature and lists the signatures that exist

### Requirement: The MCP tools expose the same selection

The MCP tools `codegraph_node`, `codegraph_callers`, `codegraph_callees` and `codegraph_impact` SHALL accept a `signature` input with the meaning of `--signature`. Their results SHALL carry the signature label of the CLI output.

#### Scenario: An MCP callers call on a set

- **WHEN** an agent calls `codegraph_callers` with `symbol: "log"` and no `signature`, and `log` has two overloads
- **THEN** the result lists the callers of both overloads, each with its signature
