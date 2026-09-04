---
name: use-skillgrid
description: "Use when starting a conversation or beginning feature/change work — routes Skillgrid SDD (uninitialized → sdd-init; else → sdd-explore then propose → spec → apply → verify → archive). Triggers: skillgrid, SDD, start change, new feature, empty repo."
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  family: sdd
  part-of: skillgrid
  role: orchestrator-entry
---

# use-skillgrid

Entry skill for Skillgrid SDD — the counterpart to Superpowers' `using-superpowers`.
It does **not** reimplement phases; it **detects, announces, and invokes** the right `sdd-*` skill.

Layout contract: [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md).
Phase order: `init → explore → propose → spec → apply → verify → archive`.

<SUBAGENT-STOP>
If you were dispatched as a dedicated `sdd-*` sub-agent, ignore this skill and continue that phase.
</SUBAGENT-STOP>

## The Rule

**Before freestyle coding or inventing a change process**, invoke `use-skillgrid` when the request is (or might be) product/code change work.

1. Announce: `Using use-skillgrid to <route>`.
2. Run the checklist below.
3. Load and follow the target `sdd-*` skill exactly (delegate if `delegate_only`).

User instructions (`AGENTS.md`, direct “skip SDD”) override. No UserPromptSubmit / platform hook is required — agent discipline only.

## Checklist

```
[ ] 1. Classify: SDD change work vs Q&A / lookup vs spike
[ ] 2. Detect Skillgrid initialized? (see Detection)
[ ] 3. If NO  → invoke sdd-init; stop until user validates findings
[ ] 4. If YES → invoke sdd-explore (unless resuming mid-change — see Resume)
[ ] 5. Walk sdd-propose → sdd-spec; present user gate before apply
[ ] 6. On apply path: sdd-apply → sdd-verify → sdd-archive
```

## Detection — initialized?

**Uninitialized** (treat as “empty” for SDD) when **any** of:

1. No `docs/skillgrid/config.yaml`
2. No `<!-- skillgrid-sdd:start -->` … `<!-- skillgrid-sdd:end -->` block in `AGENTS.md` / `CLAUDE.md` / `GEMINI.md`

Optional corroboration: missing Mnemonic `sdd-init/{project}`.

A repo with application code but no Skillgrid skeleton is still **uninitialized** → **`sdd-init`**.
Git with zero commits is a hint only, not the sole signal.

**Initialized** when `docs/skillgrid/config.yaml` exists (prefer also the AGENTS skillgrid block).

## Router

| Condition | First skill | Then |
|---|---|---|
| Uninitialized | **`sdd-init`** | After success, if user already stated a change: **`sdd-explore`** |
| Initialized + change request | **`sdd-explore`** | `sdd-propose` → `sdd-spec` → **user gate** → `sdd-apply` → `sdd-verify` → `sdd-archive` |
| Initialized + Q&A / lookup only | Do **not** force the pipeline | `mem_search` / `code_search` / `investigate` OK |
| Spike / throwaway feasibility | **`design-spike`** | Not full SDD unless findings become a real change |
| Mid-change resume (`changes/<NNN-slug>/` exists) | Read `tasks.md` `## State.phase` | Resume that phase; skip re-explore unless State/user says stale |

```
use-skillgrid
    │
    ├─ uninitialized? ──YES──► sdd-init ──► (optional) sdd-explore ──► …
    │
    └─ initialized + change? ──► sdd-explore ──► sdd-propose ──► sdd-spec
                                                      │
                                              user gate│
                                                      ▼
                                              sdd-apply ──► sdd-verify ──► sdd-archive
```

## User gate (mandatory)

After `sdd-spec` writes `tasks.md` + `acceptance.feature`, **stop** and ask:

1. **Implement** — `sdd-apply` (then verify → archive)
2. **Revise** — back to `sdd-propose` (and/or `questioning`)

Do not auto-apply.

## Resume

If `docs/skillgrid/changes/<NNN-slug>/tasks.md` exists, read `## State`:

| `phase` | Action |
|---|---|
| `spec` | Finish or re-run `sdd-spec` if incomplete |
| `apply` | `sdd-apply` for current_step |
| `verify` | `sdd-verify` |
| `archive` | `sdd-archive` |
| missing / legacy tree | Prefer explore+propose only after user confirms; do not rewrite legacy `001`–`006` unless asked |

## Red flags

| Thought | Reality |
|---|---|
| "Repo already has code, skip init" | Missing `docs/skillgrid/config.yaml` ⇒ still `sdd-init` |
| "I'll just edit the files" | Change work goes explore → propose → … |
| "Init is overkill for one file" | Bounded still uses `sdd-propose` (short `change.md`) |
| "I know the stack" | `config.yaml` / init observations are source of truth |
| "Spec is done, start coding" | User gate first |
| "This is just a question" | Q&A may skip the pipeline; change requests may not |

## Skill priority

Process order for change work:

1. **`use-skillgrid`** (this skill) — route
2. **`sdd-*`** phase skill — execute phase
3. Domain skills (`tdd`, `codebase-design`, `glossary`, …) — as the phase skill loads them

## References

- [`../sdd-init/SKILL.md`](../sdd-init/SKILL.md)
- [`../sdd-explore/SKILL.md`](../sdd-explore/SKILL.md)
- [`../sdd-propose/SKILL.md`](../sdd-propose/SKILL.md)
- [`../sdd-spec/SKILL.md`](../sdd-spec/SKILL.md)
- [`../sdd-apply/SKILL.md`](../sdd-apply/SKILL.md)
- [`../sdd-verify/SKILL.md`](../sdd-verify/SKILL.md)
- [`../sdd-archive/SKILL.md`](../sdd-archive/SKILL.md)
- [`../design-spike/SKILL.md`](../design-spike/SKILL.md)
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md)
- [`docs/plan/01-workflow.md`](../../../docs/plan/01-workflow.md)
