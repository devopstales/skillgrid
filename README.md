# AISkillGrid

<div align="center">

<img width="768" alt="skillgrid brand" src="docs/assets/v9NDj7Jw.jpeg" />

A **configuration hub** for opinionated AI-assisted development: reusable **skills**, **slash commands**, and spec-driven workflow with OpenSpec-style change management.

<p>
<a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License: Apache-2.0"></a>
<img src="https://img.shields.io/badge/workflow-SDD-orange" alt="Workflow: SDD">
<img src="https://img.shields.io/badge/docs-numbered%20guides-success" alt="Docs: Numbered Guides">
</p>

</div>

---

## What It Does

AISkillGrid is a local-first operating layer for AI-assisted development. It turns open-ended chat into a structured engineering workflow with phase commands, reusable skills, durable artifacts, verification-first quality gates, and memory integration.

## Highlights

| Feature | What it does | Why it matters |
|---------|--------------|----------------|
| SDD Workflow | Guides work through init, explore, brainstorm, plan, apply, verify, and finish | Keeps agent work tied to explicit phases, artifacts, and exit checks |
| Multi-IDE command hub | Ships `/sdd-*` commands for Cursor, Kilo, OpenCode, and GitHub Copilot prompts | One workflow travels across the IDEs you use |
| Agent skills catalog | Provides reusable skills for TDD, review, security, UI design, research, OpenSpec, and more | Agents get focused operating procedures instead of ad hoc chat instructions |
| File-first handoff | Stores PRDs, OpenSpec changes, handoff files, event logs, checkpoints under the repo | Work survives context resets without requiring a database or hosted service |
| Intent-gated loop | Adds `/sdd-loop` for the next safe phase or `[AFK]` slice, with explicit HITL and verification stop conditions | Long-running agent work stays bounded by artifacts and user authority |

## The Basic Workflow

1. **Init** (`/sdd-init`) - Bootstraps project context, detects stack, configures persistence (hybrid mode recommended), and creates `.skillgrid/` and `openspec/` directories.

2. **Explore** (`/sdd-explore`) - Free-form codebase investigation. Maps architecture, identifies patterns, compares approaches. Makes NO code changes.

3. **Brainstorm** (`/sdd-brainstorm`) - Full planning pipeline: explore → clarify → propose → spec → design → tasks. Produces artifacts in `openspec/changes/<name>/`.

4. **Apply** (`/sdd-apply` or `/sdd-loop`) - Implements tasks using TDD. Two-stage review per task: spec compliance then code quality. Fresh subagent per task.

5. **Verify** (`/sdd-verify`) - Stage 1: Spec compliance verification. Traces every requirement to code/test evidence. PASS/FAIL/PARTIAL verdict.

6. **Review** (`/sdd-review`) - Stage 2: Code quality review. Evaluates style, DRY, errors, tests, security, performance. APPROVED/CHANGES_REQUESTED.

7. **Archive** (`/sdd-archive`) - Pre-merge gate (tests green, lint clean, worktree clean) then merges/PRs/keeps/discards per configuration.

**The agent checks for relevant skills before any task. Mandatory workflows, not suggestions.**

## Two-Stage Review

**Spec compliance** (`sdd-verify`) - Traces every requirement to code/test evidence. PASS/FAIL/PARTIAL verdict with gap analysis.

**Code quality** (`sdd-review`) - Evaluates style, DRY, errors, tests, security, performance. Severity-tagged issues (CRITICAL/IMPORTANT/MINOR) and APPROVED/CHANGES_REQUESTED verdict.

---

## Quick Start

1. Open this repository in your agent-enabled IDE.
2. Bootstrap project context:

   ```text
   /sdd-init
   ```

3. Start a change:

   ```text
   /sdd-brainstorm <change-name>
   ```

4. Implement and verify:

   ```text
   /sdd-apply
   /sdd-verify
   /sdd-archive
   ```

---

## High Council

Specialist **Norse** personas are delegated viewpoints—not owners of the workflow. The parent session (typically **`odin`**) merges reports; **`tyr`** and **`heimdall`** can **hard-gate** progression on critical findings. Details, routing matrix, and commands: [`subagent-personas`](docs/09-subagent-personas.md).

| Persona | Job |
| --- | --- |
| Odin | Primary session owner: SDD sequencing, tools, delegation (default OpenCode primary). |
| Board | Persona-board chair: route personas, run parallel reports, merge verdicts, enforce gates (`/sdd-persona-board`, `/sdd-board`, …). |
| Kvasir | Fast read-only codebase recon: map, entrypoints, dependency direction before big edits. |
| Thor | Implementation enforcer: delivery feasibility, execution quality, momentum. |
| Tyr | Spec and compliance verifier: traceability and acceptance criteria; **critical = hard stop** until resolved or accepted. |
| Heimdall | Security and release-gate sentinel: threat model and exploitability; **critical = hard stop** until resolved or accepted. |
| Frigg | UX and product-clarity reviewer: flows, accessibility, content quality. |
| Loki | Adversarial critic: counterexamples and assumption stress-tests; can flag conflicts needing HITL. |
| Mimir | Bootstrap / memory continuity and architecture coherence; strategic voice on architecture-style boards. |
| Bragi | Structured artifact author: specs, tasks, and clear traceable requirement wording. |
| Vidar | Root-cause debugging: systematic investigation, evidence, regression prevention. |

---

## Philosophy

- **TDD** - Write tests first, always
- **Systematic over ad-hoc** - Process over guessing  
- **Complexity reduction** - Simplicity as primary goal
- **Evidence over claims** - Verify before declaring success

---

## Documentation

| Doc | Contents |
|-----|----------|
| [docs/00-start-here.md](docs/00-start-here.md) | Start-here overview and manifesto: human-in-the-loop pipelines, spec-driven guidance |
| [docs/01-installation.md](docs/01-installation.md) | Install toolchain and workflow CLIs |
| [docs/02-workflow-usage.md](docs/02-workflow-usage.md) | Skillgrid phases, `.skillgrid/config.json`, PRD and OpenSpec handoff |
| [docs/03-skillgrid-logic.md](docs/03-skillgrid-logic.md) | PRD/INDEX/OpenSpec hierarchy and `.skillgrid/templates/` blanks |
| [docs/04-commands.md](docs/04-commands.md) | Slash commands and where they live per IDE |
| [docs/05-skills.md](docs/05-skills.md) | Catalog of all skills with paths and summaries |
| [docs/06-rules-and-governance.md](docs/06-rules-and-governance.md) | Where project rules live and how they are maintained |
| [docs/07-hooks-and-automation.md](docs/07-hooks-and-automation.md) | Shared hooks and automation policy |
| [docs/08-multi-agent-work.md](docs/08-multi-agent-work.md) | Subagents, personas, dependency waves, handoff/event logs |
| [docs/09-subagent-personas.md](docs/09-subagent-personas.md) | Specialist persona catalog |
| [docs/10-mcp-servers.md](docs/10-mcp-servers.md) | MCP server connections |
| [docs/11-memory-and-indexing.md](docs/11-memory-and-indexing.md) | Durable context and codebase search |
| [docs/12-ticketing-integrations.md](docs/12-ticketing-integrations.md) | Local and external work tracking |
| [docs/13-webui.md](docs/13-webui.md) | Local web dashboard |

---

## Contributing

- Keep changes aligned to active `/sdd-*` workflow commands.
- Update numbered docs whenever command or skill behavior changes.
- Prefer small, reviewable PRs with clear verification evidence.

## License

Apache-2.0. See `LICENSE`.

