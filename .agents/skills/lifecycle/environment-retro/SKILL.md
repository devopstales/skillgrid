---
name: environment-retro
description: "Suggest improvements to the agent's environment, not the code. Use when closing a shipped change (called by skillgrid:reflect), or standalone after any session to make the next run cheaper and safer."
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: mattpocock-skills:retro
---

# Environment Retro

**Announce at start:** "I'm using the skillgrid:environment-retro skill to make the next run cheaper."

You are looking back at a coding session and suggesting changes to the **environment the agent operates in** — not the code. The code was already built and shipped; this is about making the *next* run faster, cheaper, and safer. The sharpest rule: a **mechanical** mistake gets a **deterministic check** (a linter rule, a pre-commit hook, a CI job), and the steering files are reserved for **judgement calls**. A repo with no guardrail at all is itself a finding — an un-linted, un-checked repo is a standing missed opportunity, not a neutral default.

## When to Use

- Inside `skillgrid:reflect`, after the change retro half and before session close — to close the loop on the environment, not just the change.
- Standalone, after any session, when you want the next run to start from a better environment.

**When NOT to use:** to change shipped *code* — that is `skillgrid:ship` / a follow-up change. To capture what the *change* did — that is `skillgrid:reflect`'s retro half (Decisions / Lessons / Patterns / Surprises). This skill is about the harness around the work, and it writes suggestions, never code.

## The Six Categories

Work the categories in this order. For each, present a candidate **only when its "Use when" fires**; a clean session legitimately has no finding for a category.

| Category | Question | Use when |
|---|---|---|
| **Navigation** | Was the agent slow to find the right files? Hidden dependencies between files? | The session took a long time to find a piece of information. |
| **Automated checks** | Is there a check that could have caught an error the agent made? | The agent made a mistake a check could have caught — **or** the repo has no guardrail at all. |
| **Coding standards** | Should the reviewer enforce a new rule, or drop/clarify an old one? | The reviewer failed to catch a mistake. |
| **Steering-file no-ops** | Is there an instruction that does not change agent behavior? | The steering files (AGENTS.md / `state.yaml`) are large and unwieldy. |
| **Tool economy** | Did the agent make expensive, token-inefficient tool calls? | The agent made an expensive tool call (CLI, MCP) that could be streamlined. |
| **Information access** | What did the agent need but could not see? | A crucial piece of information was not available to the agent. |

### Automated checks — read the repo's own checks first

Before proposing a new check, read the repo's existing guardrails: `config.yaml` `testing.runner` + `setup`, the package manager's `lint`/`check`/`test` scripts, and the CI workflow. If a check already exists but sits **unwired or silently broken, that is the finding** — not a reinvention. Default to the cheapest place the repo's language and existing guardrail make it natural: a custom linter rule, a pre-commit hook, or a CI job.

### Coding standards — classify first

Classify the violation **before** deciding where it lands:

- **Mechanical** (a fixed syntactic pattern, a banned API, an import shape, a file-location rule) → a deterministic check, full stop. Do **not** write it into the standards file.
- **Judgement** (cross-file consistency, "matches the surrounding style", anything no guardrail could substitute) → the steering/standards file (`.agents/skills/_shared/rules/code-standards.md` or a project `CODING_STANDARDS.md`).

Default to building the check over writing the rule.

## Output

Append a `## Environment Retro` section to the change's `report.md` (in the archive when run from `reflect`; in the active spec folder when standalone). Every entry cites a source (a file:line, a commit, a tool call, or the session's symptom) and names **where the fix lands**:

```markdown
## Environment Retro

| Category | Finding | Fix lands in | Source |
|----------|---------|--------------|--------|
| Automated checks | repo has no pre-commit lint hook | add pre-commit hook running `<lint cmd>` | session: 3 lint errors caught only at review |
| Navigation | agent spent ~5 min finding the auth config | add pointer in `ARCHITECTURE.md` → `config/auth` | session: file search trail |
| Coding standards | mechanical: `console.log` left in | add linter rule `no-console` (not a standards line) | `src/x.ts:42` |
```

Then, for each finding that is a real build (a new hook, linter rule, CI job, or steering-file edit):

1. **`mem_save`** the durable finding (`type: pattern` or `config`, `scope: project`).
2. **Offer a follow-up ticket** (via `skillgrid:ticketing` when a tracker is configured) for any fix that is real work. This skill *proposes*; it does not implement. Present candidates in order of severity.

## Rules

- **Environment, never code.** You do not edit shipped code. You point at the harness: steering files, checks, tooling, information access.
- **Check before rule.** A mechanical violation becomes a deterministic check; the standards file is for judgement calls only.
- **Read existing guardrails first.** An unwired or broken existing check is the finding — not a new invention.
- **Propose, don't implement.** Real builds become tickets + `mem_save`, not edits made in this skill.
- **Every finding is sourced.** No source = a guess, not a finding.
- **A clean category is fine.** Not every session has a finding in all six categories. An empty session is a legitimate result.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "The code is done; skip the environment." | The next run pays for every missing check and pointer. This is where the next session gets cheaper. |
| "I'll just add a line to the standards file." | If it's mechanical, a linter rule or hook catches it every time; the standards file is for judgement calls only. |
| "We don't need a linter, it's a small repo." | A repo with no guardrail is itself a finding. The first lint rule is the cheapest insurance you can buy. |
| "There was no mistake, so nothing to retro." | No-mistake sessions still surface navigation gaps, tool-economy wins, and steering no-ops. Absence of a bug is not absence of a finding. |
| "I'll fix the check right now." | This skill proposes; implementing is a follow-up ticket. Keep the retro a read-only sweep so it stays fast and safe. |

## Red Flags

- A finding with no source — no file:line, commit, tool call, or session symptom cited.
- A mechanical violation written into the standards file instead of a deterministic check.
- A "new check" proposed when an existing check is just unwired or broken.
- Edits made to shipped *code* (instead of the environment / harness).
- All six categories forced to produce a finding even when the session was clean.

## Verification

- [ ] The six categories were each checked against the session; a finding is present **only** where its "Use when" fired.
- [ ] A `## Environment Retro` section was written (to `report.md` in the archive when from `reflect`, or the active spec folder when standalone) with every row sourced and naming where the fix lands.
- [ ] Every mechanical violation was routed to a deterministic check, and only judgement calls went to the standards/steering file.
- [ ] Existing guardrails (`config.yaml`, lint/check scripts, CI) were read before any new check was proposed.
- [ ] Each real build is captured as a `mem_save` and (when a tracker is configured) an offered follow-up ticket — not implemented inline.

## References

- [`../reflect/SKILL.md`](../reflect/SKILL.md) — the caller; runs this skill after the change retro half, before session close.
- [`../../_shared/rules/sdd-structure.md`](../../_shared/rules/sdd-structure.md) — the `report.md` retro half and the `archive/` layout this section lands in.
- [`../../_shared/rules/mnemonic-memory.md`](../../_shared/rules/mnemonic-memory.md) — the `mem_save` shape for durable environment findings.
- [`../../_shared/rules/code-standards.md`](../../_shared/rules/code-standards.md) — the judgement-call standards file the mechanical-vs-judgement split feeds.
- [`../ticketing/SKILL.md`](../ticketing/SKILL.md) — publishes the follow-up tickets for real builds.
