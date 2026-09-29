@change-2026-09-29-llmwiki-okf-wiki
# Acceptance: wiki compiler (Pillar 1)
# Executable contract for `skillgrid wiki compile` / `skillgrid wiki lint`.
# Each Feature is tagged @step-NN and maps to a `tasks.md` step.

@step-01 @okf-conformance
Feature: Every emitted concept is OKF v0.2 conformant
  As an OKF consumer
  I want every `.wiki/wiki/**/*.md` page to carry valid v0.2 frontmatter
  So the bundle is portable to any OKF reader without edits

  Scenario: A page always carries the single required key
    Given a compiled `.wiki/` from a project with at least one ADR
    When I read any emitted concept file
    Then its frontmatter MUST contain a `type` key
    And the `type` value MUST be one of ADR, Constraint, Term, Spec, Spike, State, Architecture, Finding, Source

  Scenario: All timestamps are absolute UTC ISO 8601
    Given a concept with a `generated` block
    When I read its `generated.at`
    Then it MUST be an absolute UTC instant (no relative TTL, e.g. "90d")
    And a `stale_after`, when present, MUST also be an absolute UTC instant

  Scenario: sources entries are well-formed
    Given a Finding concept with sources
    When I read each `sources[]` entry
    Then it MUST contain a `resource` field
    And `author` and `last_modified`, when present, MUST be non-empty

  Scenario: status is absent or a valid enum
    Given any emitted concept
    When I read its optional `status` key
    Then if present it MUST be one of draft, stable, deprecated
    And if absent the concept is treated as stable


@step-02 @source-adapters
Feature: The compiler projects the Pillar-1 sources
  As a project owner
  I want the committed knowledge to appear in `.wiki/`
  So a fresh clone or an agent gets a portable bundle of what we decided

  Scenario: ADRs are emitted from ASSUMPTIONS.md
    Given `.skillgrid/ASSUMPTIONS.md` with in-force ADRs 0001..0014 and one superseded ADR
    When I run `skillgrid wiki compile`
    Then `.wiki/wiki/adr/` MUST contain one concept per ADR
    And the in-force table MUST drive Supersedes/Amends edges
    And the superseded ADR's concept MUST carry `status: deprecated`

  Scenario: Locked constraints are emitted as Constraint concepts
    Given `.skillgrid/ASSUMPTIONS.md` has a `### Locked constraints` list
    When I run `skillgrid wiki compile`
    Then `.wiki/wiki/concepts/` MUST contain a Constraint concept per bullet

  Scenario: Glossary terms are emitted as Term concepts
    Given `artifacts/01-business-terms.md` and `02-technical-terms.md` exist
    When I run `skillgrid wiki compile`
    Then `.wiki/wiki/concepts/` MUST contain a Term concept per defined term

  Scenario: state.yaml is emitted as a single State concept
    Given `.skillgrid/state.yaml` with pipeline.current_phase
    When I run `skillgrid wiki compile`
    Then `.wiki/wiki/entities/` MUST contain exactly one State concept

  Scenario: spec and spike dirs are emitted (existence-gated)
    Given `.skillgrid/specs/<change>/` and `.skillgrid/spikes/<NNN-name>/` exist
    When I run `skillgrid wiki compile`
    Then a Spec concept MUST be emitted per change dir
    And a Spike concept MUST be emitted per spike dir

  Scenario: ARCHITECTURE.md is existence-gated
    Given `.skillgrid/ARCHITECTURE.md` exists
    When I run `skillgrid wiki compile`
    Then an Architecture concept MUST be emitted
    And given the file is absent, no Architecture concept MUST be emitted and no error MUST occur

  Scenario: The compiler is read-only over its sources
    Given a snapshot of `.skillgrid/`, fresh `web_cache` rows, and `raw/`
    When I run `skillgrid wiki compile`
    Then the bytes of `.skillgrid/**`, the `web_cache` store, and `raw/**` MUST be unchanged


@step-03 @web-cache
Feature: Fresh web_cache rows compile into Findings without a raw/ middle step
  As a researcher
  I want cached web research to surface in the wiki
  So provenance (sources/verified/stale_after) travels with the finding

  Scenario: A fresh cited row becomes a Finding in wiki/
    Given a fresh `context7|exa|deepwiki|fetch` row cited by an emitted page
    When I run `skillgrid wiki compile`
    Then `.wiki/wiki/entities/` MUST contain a Finding concept
    And its `sources[]` MUST reference the row url
    And its `verified[]` MUST carry `by: process:<source>` and `at: <fetched_at>`
    And its `stale_after` MUST equal `<fetched_at + 90d>` as an absolute instant
    And NO file MUST be written to `raw/` for this row

  Scenario: A fresh uncited row becomes a research/ draft
    Given a fresh web_cache row not cited by any emitted page
    When I run `skillgrid wiki compile`
    Then `.wiki/wiki/research/` MUST contain a Finding concept with `status: draft`

  Scenario: Non-qualifying rows are skipped
    Given a `manual` row, or a fresh-row whose `expires_at <= now`
    When I run `skillgrid wiki compile`
    Then no concept MUST be emitted from that row


@step-04 @raw-index
Feature: User-dropped raw markdown is indexed but never rewritten
  As a power user
  I want my clipped/notes markdown to become wiki pages
  So my drop zone is a source I fully control

  Scenario: A raw md with OKF frontmatter keeps its declared type
    Given `raw/note.md` with frontmatter containing `type`
    When I run `skillgrid wiki compile`
    Then a concept MUST be emitted using that `type`
    And its `source_path` MUST be `raw/note.md`

  Scenario: A raw md without frontmatter becomes a bare Source
    Given `raw/plain.md` with no frontmatter
    When I run `skillgrid wiki compile`
    Then a Source concept MUST be emitted with title derived from the filename
    And no `verified` block MUST be present
    And its `source_path` MUST be `raw/plain.md`

  Scenario: raw/ is never modified
    Given a snapshot of `raw/**`
    When I run `skillgrid wiki compile`
    Then `raw/**` MUST be byte-identical to the snapshot
    And an absent `raw/` MUST be a no-op (no error, no directory created)


@step-05 @determinism
Feature: The compile is a pure, no-churn function
  As a maintainer
  I want `.wiki/` to be a clean git commit
  So a diff of `.wiki/` always reflects a real source change

  Scenario: A no-op recompile is byte-identical
    Given a committed `.wiki/` produced by a prior compile
    When I run `skillgrid wiki compile` again with no source change
    Then `git diff --exit-code .wiki` MUST be 0
    And each unchanged concept MUST keep its prior `generated.at` (content-hash gate)

  Scenario: Output ordering is stable
    Given two compiles of the same input
    When I diff the emitted file lists
    Then concepts MUST be ordered by (TypeDir, Slug) and edges by (From, To, Kind)

  Scenario: Only changed concepts are rewritten
    Given one source page changed between compiles
    When I run `skillgrid wiki compile`
    Then ONLY that concept's file MUST be rewritten (its `generated.at` set to the compile clock)
    And all other files MUST retain their prior bytes and `generated.at`


@step-06 @index-log
Feature: The bundle is self-describing for llmwiki
  As an llmwiki consumer
  I want the workspace scaffold and a catalog
  So the extension can list raw sources, search the catalog, and read the log

  Scenario: index.md is a grouped catalog
    Given a compiled `.wiki/`
    When I read `.wiki/wiki/index.md`
    Then it MUST group concepts by type directory with their titles

  Scenario: log.md is an append-only compile history
    Given two compiles
    When I read `.wiki/wiki/log.md`
    Then it MUST contain one entry per compile (append-only, never rewritten)

  Scenario: AGENTS.md documents the schema
    Given a compiled `.wiki/`
    When I read `.wiki/AGENTS.md`
    Then it MUST document the type taxonomy, the slug rule, and the lint rule


@step-07 @lint
Feature: wiki lint validates the bundle
  As a QA gate
  I want a command that checks conformance and drift
  So a hand-edited or drifted `.wiki/` is caught before it misleads

  Scenario: A conformant bundle passes
    Given a `.wiki/` produced by the compiler
    When I run `skillgrid wiki lint`
    Then it MUST exit 0 and report no conformance errors

  Scenario: A broken page is flagged
    Given a `.wiki/` page missing its `type` key
    When I run `skillgrid wiki lint`
    Then it MUST exit non-zero and name the offending file

  Scenario: A drifted page is flagged
    Given a concept file whose bytes no longer match its manifest content-hash
    When I run `skillgrid wiki lint`
    Then it MUST report the file as drifted (compiler-owned)


@step-08 @cli
Feature: wiki is a first-class CLI command
  As an operator
  I want to drive the compiler from the CLI
  So the capability works headless and in the pipeline

  Scenario: wiki compile and wiki lint are registered
    Given the `skillgrid` binary
    When I run `skillgrid wiki --help`
    Then it MUST list `compile` and `lint` subcommands

  Scenario: compile resolves project from CWD
    Given a CWD inside the project
    When I run `skillgrid wiki compile`
    Then it MUST resolve the project via the CWD-resolved handle (mem.go pattern)
    And it MUST open the store and read fresh `web_cache` rows for the compile

  Scenario: no new Go dependencies are introduced
    Given `go.mod` before the change
    When the change is built
    Then `go.mod` MUST be unchanged (stdlib-only `internal/mnemonic/wiki/`)
