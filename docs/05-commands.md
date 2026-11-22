# Commands

Skillgrid is driven by skills, not slash commands. The pipeline advances by invoking the skill that matches the current phase. This document explains how to invoke the pipeline and the commands available in the agent harness.

## How To Invoke

Skills are invoked two ways:

- **Model-invoked skills:** the agent reaches for them autonomously based on the `description` trigger. You just describe the work; the agent picks the right skill.
- **User-invoked skills:** you invoke them by name. Only the human can reach these.

The `skillgrid` skill is the top-level router. When you say "build X" or "continue the project," the router selects the right phase skill.

## Pipeline Commands

The happy path, in order:

```text
brainstorm   →  Turn a raw idea into an approved design and spec
blueprint    →  Turn a spec into a buildable implementation plan with BDD acceptance spec
slicing      →  Break the blueprint into vertical tracer-bullet tickets
execute      →  Execute the plan: workspace, mode, waves, QA handoff
test         →  Intent, functional, code-quality, security testing; four-state verdict
ship         →  Reflect, retro, documentation, commit, PR, archive
```

Each command produces durable artifacts that the next command consumes:

| Phase | Input artifact | Output artifact |
|---|---|---|
| brainstorm | user's idea | approved design spec |
| blueprint | approved spec | `blueprint.md` + BDD acceptance spec |
| slicing | `blueprint.md` | `tasks.md` + tracker records |
| execute | `tasks.md` | implemented code + test evidence |
| test | implemented code | four-state verdict + scan results |
| ship | passed test | commit + PR + projectwiki pages + archive |

## Cross-Cutting Commands

Reachable from any phase:

```text
debug              →  Four-phase bug investigation (hypothesis trail, 3-fix question, verdict)
drift              →  Code-vs-doc divergence check (report-only)
refactor           →  Codebase health: jscpd duplication, dead code, complexity
ui-design          →  Product/application UI design
writing            →  Human-facing text: guides, PR body, changelog, release note
projectwiki        →  Read/maintain the knowledge vault
youtube-transcript →  Extract a YouTube transcript
```

## Meta Commands

Operate on the skill library itself:

```text
skill-creator  →  Create, audit, iterate skills; 10-rule audit; lifecycle curation
skillgrid      →  Top-level router (the entry point)
```

## Typical Sessions

### New Feature

```text
1. "I want to build a rate limiter for our API"
   → brainstorm (questions, design, approval gate)
2. (approved spec)
   → blueprint (research, BDD spec, plan)
3. (blueprint.md ready)
   → slicing (tickets, dependency waves)
4. (tasks.md ready)
   → execute (workspace, TDD per ticket)
5. (code done)
   → test (areas, verdict)
6. (verdict PASS)
   → ship (commit, PR, archive)
```

### Bug Fix

```text
1. "The auth middleware is returning 500s on token refresh"
   → debug (four-phase investigation, root cause)
2. (root cause found, fix implemented)
   → test (regression, verdict)
3. (verdict PASS)
   → ship (commit, PR, archive)
```

### Tech Debt

```text
1. "Clean up the duplicated validation logic"
   → refactor (jscpd scan, findings, verification)
2. (refactored, jscpd delta positive)
   → test (regression, verdict)
3. (verdict PASS)
   → ship (commit, PR)
```

### Drift Check

```text
1. "Does the code match the blueprint?"
   → drift (scan, report)
2. (drift found)
   → execute (fix the drift)
   → test (verdict)
   → ship
```

## Configuration

Pipeline behavior is driven by `.skillgrid/config.yaml`:

| Key | Purpose |
|---|---|
| `ticketing.type` | Tracker: `gh`, `glab`, `jira`, `backlogmd`, `hermes-kanban` |
| `mnemonic.enabled` | Enable the mnemonic MCP server for projectwiki |
| `test.security` | Which security scanners to run |
| `execute.mode` | Default execution mode: `inline`, `subagent`, `parallel` |

The skills read the config at runtime and adapt. The config does not change the phase order; it changes which tools and trackers the phases use.

## Verification

Every phase produces evidence:

| Phase | Evidence |
|---|---|
| brainstorm | approved spec (user sign-off) |
| blueprint | `blueprint.md` + BDD `.feature` file |
| slicing | `tasks.md` + tracker records |
| execute | test output, task briefs, implementer reports |
| test | four-state verdict + scan results |
| ship | commit hash, PR URL, archive path |

A phase is not complete without its evidence. "Seems right" is not a checkbox.
