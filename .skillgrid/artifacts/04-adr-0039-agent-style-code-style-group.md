# Code-style group: agent-style as the prose standard (generation + review)

---
status: "accepted"
supersedes: none
date: 2026-10-19
---

**Deciders:** paladm
**Related:** ADR-0008 (dependency/precedent recording pattern), ADR-0019 (ADR index as pointer)

## Context

Skillgrid agents produce large volumes of Markdown — specs, blueprints, ADRs, reports, changelogs — but the only writing standard in force is `_shared/rules/code-standards.md`, which covers code structure and naming, not prose. The `agent-style` ruleset (v0.9.0, CC-BY-4.0, Yue Zhao) exists as a literature-backed 21-rule prose ruleset (RULE-01..12 canonical + RULE-A..I field-observed LLM-output tells) with a `style-review` skill for post-hoc auditing. Without a decision, the agent-style content can only be ported ad hoc and will drift from upstream.

## Decision

**A new top-level skill group `.agents/skills/code-style/` owns how skillgrid output is written**, and it vendors agent-style v0.9.0 in its complete two-phase form:

- **Generation-time:** the 21 compact directives live at `_shared/rules/prose-style.md` (shared convention, referenced from `AGENTS.md` and `skill-anatomy.md`). Agents apply the rules while writing. This is the content of the upstream `AGENTS.md` adapter, re-anchored for skillgrid's layout.
- **Review-time:** the `style-review` skill (`.agents/skills/code-style/style-review/`) is the opt-in post-hoc audit — 21-row scorecard, on-confirm write-beside polish (`FILE.reviewed.md`), A/B compare. Rule bodies, detectors, revision prompt, and the 10-file fixture corpus are vendored under its `references/`.
- **Group invariants** (`.agents/skills/code-style/README.md`): write-beside never in-place; rules are judgment defaults, not gates (agent-style's escape hatch — only the firm rules are non-negotiable: no invented citations/numbers, no placeholders in finished text); no new facts on revision (RULE-H stays flagged).
- **Placement:** `code-style/` is a top-level group parallel to `craft/` (how skills are built) and `verification/` (correctness). It is a specialist class in `using-skillgrid` — named by the user or a pipeline skill, no spec folder.
- **Vendoring:** the upstream `manifest.json` is dropped (it hashes a `.agent-style/` tree this repo does not carry). Bumps are manual: replace the vendored `references/` files + `prose-style.md` and update the version in the skill's `based_on` and the self-verification handshake.

### Why not the alternatives

- **A single `style-review` skill only (no generation rules):** the skill's own contract is that it "complements the generation-time rules" — review without generation is a second pass over prose the first pass never tilted. Rejected.
- **Dropping the compact rules into `AGENTS.md` inline:** the upstream adapter is a ~55-line block; inlining it into the operational-rules file makes AGENTS.md the vendor point and couples it to an upstream release. A shared convention file matches the existing pattern (`code-standards.md`, `commits.md`) — AGENTS.md references, never inlines. Rejected.
- **Folding the group into `craft/`:** `craft/` is about skill construction (anatomy, effort budgets, anti-rationalization devices). Prose style governs *output*, not *skills*. A distinct group keeps both boundaries honest. Rejected.

## Consequences

- **Positive:** every skill that writes Markdown has a named prose standard; the review skill has a generation counterpart; the vendored corpus (rules + fixtures) makes the audit testable against pinned expected output.
- **Negative / cost:** a ~15 KB convention file referenced from AGENTS.md; a vendored copy of agent-style that must be bumped by hand on upstream releases (no dependency, so no lockfile).
- **No new Go dependencies:** agent-style is content (Markdown + one optional CLI invoked via `exec` at review time), not a Go import. The skill degrades to a semantic-only audit when the `agent-style` binary is absent (fail-open, matching ADR-0016's posture).
- **Deferred:** per-project rule overrides (e.g., a project opting out of RULE-I contractions); wiring `style-review` into the `qa` gate as an advisory check; the upstream `claude-code.md` / `CLAUDE.md` adapters (they are install-time files for other repos' agents, not skillgrid skills).
