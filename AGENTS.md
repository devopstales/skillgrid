# Repository Instructions

This is the **real source of truth for the operational rules** a change must respect — the hard limits a build, commit, or change must follow. The onboarding skill renders the `

<!-- skillgrid-preamble:start -->
# Standards

## Environment & Tooling

- **Language**: go build ./...
- **Package manager**: <detect>
- **Task runner**: <detect>
- **Install**: go build ./...
- **Test**: go test ./...
- **Lint**: <detect>
- **Format**: gofmt -w ./...
- **Build (dev)**: go build ./...
- **Run (dev)**: <detect>
- **Docker (dev deps)**: <detect>
- **Security scan**: trivy fs . --scanners vuln --severity CRITICAL,HIGH,MEDIUM,LOW

## Engineering Standards

No engineering standards yet — see `.skillgrid/projectwiki/ASSUMPTIONS.md`.

## Security & Escalation Boundaries

No security boundaries yet — see `.skillgrid/projectwiki/ASSUMPTIONS.md`.

## Dependency Policies

No dependency policies yet — see `the manifest`.

## Architecture Constraints

No architecture constraints yet — see `.skillgrid/projectwiki/ASSUMPTIONS.md`.

## Definition of Done

No definition of done yet — see `CI config`.

## Issue Tracker

Backlog.md — see the `## Skillgrid` Issue Tracker block for conventions.

## Memory

Mnemonic — reference `~/.agents/skills/_shared/rules/mnemonic-memory.md`

## Code Indexing

Mnemonic — reference `~/.agents/skills/_shared/rules/mnemonic-memory.md` (Code Index Ladder)

## Testing standards

reference `~/.agents/skills/_shared/rules/code-standards.md`

## SDD Standards

reference `~/.agents/skills/_shared/rules/sdd-structure.md`
<!-- skillgrid-preamble:end -->

<!-- skillgrid:start -->` … `<!-- skillgrid:end -->` block below into this file.

**What belongs here:** operational rules (toolchain floor, commit format, dependency policy, spec/code zone order, enforcement hooks). **What does NOT belong here:** design *decisions* — those are ADRs, indexed in `.skillgrid/projectwiki/ASSUMPTIONS.md` § `LOCKED` with their bodies in `.skillgrid/projectwiki/decisions/04-adr-NNNN-slug.md`. If a rule is really "we decided X over Y because Z," it is an ADR, not an operational rule.

**What this is:** skillgrid — a local-first engine for agent memory + code intelligence. Single Go binary, three transports (CLI, MCP stdio, HTTP/REST + embedded SPA). Decisions, observations, and code context persist outside the chat so AI coding agents keep working across sessions. See `.skillgrid/projectwiki/ASSUMPTIONS.md` § VERIFIED for the full product + stack description.

<!-- skillgrid:start -->
## Skillgrid

This project is configured with Skillgrid. Project: **skillgrid-v2**.

Config: `.skillgrid/config.yaml` (static) — read it before running any Skillgrid skill.
State: `.skillgrid/meta/state.yaml` (dynamic) — where the project is right now (phase, current change, progress).

### Artifacts

| Artifact | Path |
|----------|------|
| Project knowledge (product understanding, architecture, terms, ADRs, findings) | `.skillgrid/projectwiki/` (OKF bundle: `decisions/` `invariants/` `workflows/` `architecture/` `gotchas/` `glossary/` + derived `index/`) |
| Project state (phase, current change, progress) | `.skillgrid/meta/state.yaml` |
| Specs (briefing, blueprint, tasks) | `.skillgrid/change-specs/` |
| Execution ledger | `.skillgrid/meta/.sdd/` (gitignored) |

**Domain model:** read the vocabulary + ADRs under `.skillgrid/` (`glossary/` + `decisions/`, index: `knowledge/index/`) before designing or implementing. The ADR index is `.skillgrid/projectwiki/ASSUMPTIONS.md` § `LOCKED`; open the linked file for the full Context / Decision / Consequences.

**Rules & standards:** the operational rules live under `### Rules` below. Shared standards are referenced, never inlined — each loads via the skills that apply it.

- **Coding conventions** — reference `.agents/skills/_shared/rules/code-standards.md` for detailed coding conventions.
- **Prose & writing style** — Markdown the agent writes follows `_shared/rules/prose-style.md` (agent-style v0.9.0, 21 rules, generation-time); `skillgrid:style-review` is the post-hoc audit against the same rules.
- **Testing conventions** — reference `.agents/skills/_shared/references/strict-tdd.md` for the TDD cycle and testing conventions.
- **Commits & verification** — reference `.agents/skills/_shared/rules/commits.md` for the commit contract, `.agents/skills/_shared/rules/verification-ladder.md` for the verification ladder, and `.agents/skills/_shared/planning/rigor-tiers.md` for rigor tiers.
- **Memory, code index & web cache** — reference the `skillgrid:mnemonic` skill (shared rules: `.agents/skills/_shared/rules/mnemonic-memory.md`, `.agents/skills/_shared/rules/mnemonic-artifacts.md`, `.agents/skills/_shared/rules/mnemonic-code-indexing.md`).

### Rules

*Operational rules only — the real source of truth is this section (edit here). Design decisions do not live here; they are ADRs in `.skillgrid/projectwiki/ASSUMPTIONS.md` § `LOCKED` → `.skillgrid/projectwiki/decisions/04-adr-NNNN-slug.md`.*

- **Go 1.22+ minimum to build** (ADR-0031). A `go.mod` below 1.22 is a drift signal; raising the floor is a new superseding ADR.
- **No new dependencies without an ADR.** Adding a third-party dependency is a decision; record it in `.skillgrid/projectwiki/ASSUMPTIONS.md` § `LOCKED` before it lands (ADR-0008 is the `yaml`-package precedent).
- **Trivy is advisory-only** — findings are reported, never blocking the QA gate.
- **Serial development:** one change at a time, no parallel branches.
- **Parallel subagents use isolated worktrees:** parallel tickets in the same wave MUST use separate `git worktree add` checkouts, never a shared checkout. Shared checkouts cause index-lock conflicts and cherry-pick overhead.
- **Conventional commits only;** no AI-attribution trailers (commit-msg hook enforces).
- **Spec-zone changes commit before code-zone changes** (pre-commit zone guard enforces).
- **The repo is the source of truth** (ADR-0013): when an external tracker, session memory, or dashboard disagrees with a committed artifact, the committed artifact wins. A commit that closes a Backlog task names that task ID in the `[skillgrid-context]` block so the `task → commit → diff` chain is recoverable from git alone.

### Issue Tracker

Backlog.md — reference `.agents/skills/_shared/rules/ticketing/` (backlogmd, github, gitlab, jira CLI + formatting standards) for ticketing conventions. Tasks live under `<backlog_directory>` (root `backlog.config.yml` field `backlog_directory`; this repo: `.skillgrid/projectwiki/backlog/`), managed via the `backlog` CLI.

### Memory

Mnemonic (local SQLite + FTS5): `mem_*`, `code_*`, `web_*` over MCP and HTTP.

### Workflow

Entry: at the start of any skillgrid task, first read `using-skillgrid` — it classifies the request, runs the start gates, and names exactly one first skill.

`brainstorming` → `writing-blueprints` → `slicing` → `ticketing` → execution (`subagent-execution` or `simple-execution`) → `qa` → `code-review` → `receiving-code-review` → `ship` → `reflect`

Run `skillgrid:onboarding` to update config after stack changes.
<!-- project-wiki:start -->
### ProjectWiki

This project uses the skillgrid project wiki: an OKF vault at `.skillgrid/projectwiki/` + a derived FTS index, served by the `projectwiki` engine — `mnemonic projectwiki ...` CLI and the `projectwiki_*` MCP tools (the shared ProjectWikiHandler — CLI and MCP never diverge).

- **Read**: `projectwiki_query` (stage-1 ranked retrieval) → `projectwiki_read` (full page / `##` section + pin-check state) → `projectwiki_search` (raw FTS power path).
- **Save**: `projectwiki_generate` — the project wiki's proactive save (run it after significant `mem_save`; incremental by default, `--full` re-derives).
- **Verify**: grounded pages pin live code (working-tree-relative anchors); `projectwiki validate --strict` is the CI gate (drift → exit 1).
- **Maintain**: `projectwiki lint [--fix]` · `projectwiki index` (incremental) / `projectwiki reindex` (full) · `projectwiki status` · `projectwiki rebuild-meta` · `projectwiki cite-source`.
- Vault files are truth (PW-INV-1); `index/**` + the `projectwiki_*` rows are derived — never hand-edit (PW-INV-3).
<!-- project-wiki:end -->
<!-- skillgrid:end -->

<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **skillgrid** (9108 symbols, 28882 relationships, 625 execution flows).

> Index stale? Run `node .gitnexus/run.cjs analyze --index-only` from the project root — it auto-selects an available runner. No `.gitnexus/run.cjs` yet? Bootstrap with `npx`, `bunx`, or `pnpm dlx` — e.g. `bunx gitnexus@latest analyze` (npm 11 npx crash; #1939).

## Always Do

- **MUST run impact before editing.** Use `impact({target: "symbolName", direction: "upstream"})` or `node .gitnexus/run.cjs impact "symbolName" --direction upstream --repo .`; report callers, processes, and risk. Never substitute grep for graph analysis.
- **MUST analyze graph changes before committing.** Use `detect_changes({scope: "all"})` (MCP) or `node .gitnexus/run.cjs detect-changes --scope all --repo .` (CLI fallback). `partial: true` or `truncated: true` is not a clean check — a zero means unseen, not unaffected; re-run it. For regression review: `detect_changes({scope: "compare", base_ref: "main"})` or `node .gitnexus/run.cjs detect-changes --scope compare --base-ref "main" --repo .`.
- MUST warn on HIGH/CRITICAL `risk` pre-edit; never use `riskSharedAxes` to waive a HIGH/CRITICAL `risk` warning. Compare File/symbol: MCP File omits axes; Graph-RAG expands File.
- **MUST treat `risk: UNKNOWN` as unresolved, not as low.** An empty caller set is not evidence the symbol is unused — it can also mean the callers are not resolvable by the index (plain-object property access, dynamic dispatch, cross-language calls). `impact` pairs `UNKNOWN` with a `riskNote` saying so. Confirm with a text search before treating the symbol as safe to change or delete; do not proceed on the strength of a zero.
- **MUST use `query({search_query: "concept"})` for concepts/flows, `context({name: "symbolName"})` for a named symbol, or `impact` for blast radius, on read-only callers, dependencies, imports, or execution flow.** Graph first; text search only for empty/`UNKNOWN`/literals.
- For security review, `explain({target: "fileOrSymbol"})` lists taint findings (source→sink flows; needs `analyze --pdg`).

## Never Do

- NEVER edit a function, class, or method before MCP/CLI impact analysis.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis, and never read `UNKNOWN` as an all-clear — it means the walk could not answer, which is the one verdict that requires confirming by other means.
- NEVER rename symbols with find-and-replace — use `rename` which understands the call graph.
- NEVER commit before MCP/CLI graph change analysis.

## Resources

| Resource | Use for |
| --- | --- |
| `gitnexus://repo/skillgrid/context` | Codebase overview, check index freshness |
| `gitnexus://repo/skillgrid/clusters` | All functional areas |
| `gitnexus://repo/skillgrid/processes` | All execution flows |
| `gitnexus://repo/skillgrid/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
| --- | --- |
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->

<!-- OPENWIKI:START -->

## OpenWiki

This repository has a generated `openwiki/` evidence index. It is optional just-in-time context, not required startup reading.

- Do not enumerate, preload, or search wikis at task start. Use retrieval when the user asks for it, when unfamiliar architecture or dependency behavior materially affects the task, or when source inspection leaves an important uncertainty. Stop once the question is grounded.
- When those conditions apply and OpenWiki retrieval tools are available, use `openwiki_search` for just-in-time context and `openwiki_read` for the relevant complete sections. If search returns `workspace_required`, ask which listed workspace to use and retry with its ID.
- Use `openwiki_list_workspaces` or `openwiki_list_wikis` when workspace membership itself needs to be discovered.
- If the retrieval tools are unavailable, read `openwiki/quickstart.md` and follow its links to the relevant pages.
- Treat source code and tests as authoritative. A brief's unknowns and review items are verification gaps, not automatic requirements.
- Prefer the narrowest quiet validation that proves the changed behavior. Preserve complete failure output.

Use the repository's documented maintenance process to refresh generated pages after source changes. Check its configuration and completed runs before relying on automatic updates. Do not hand-edit generated OpenWiki pages unless explicitly asked; prefer updating source code/docs and letting OpenWiki regenerate.

<!-- OPENWIKI:END -->
