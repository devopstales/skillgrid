# {Skill Title}

> **Template.** Copy this file to `.agents/skills/{skill-name}/SKILL.md`, rename
> `{skill-name}` to match, and replace every `<...>` and `{{...}}`. Delete this
> block and any `<!-- hint -->` comment when done. The governing spec is
> [`conventions/skill-anatomy.md`](../conventions/skill-anatomy.md) — if this
> template and the anatomy ever disagree, the anatomy wins.

<!-- ===== FRONTMATTER (rules: anatomy §Frontmatter) ===== -->
---
name: {skill-name}                              # == directory name == kebab-case
description: Guides agents through <the task, third person>. Use when <trigger 1>. Use when <trigger 2>.
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: <origin, if ported — or use `source: derived from X`; omit if native>
---

<!--
DESCRIPTION RULES (it is the only thing an agent sees before deciding to load
the skill):
- What it does (third person) + when to use (one or more "Use when ...").
- NO workflow steps here. If the description contains a procedure, the agent
  may follow the summary and never read the body.
- Max 1024 chars.
-->

# {Skill Title}

<!-- Title == `name`, title-cased. Never a subtitle, never a suffix like (TDD),
     never the based-on line. -->

**Announce at start:** "I'm using the skillgrid:{skill-name} skill to <purpose>."

<!--
ANNOUNCE RULE: required for invoked phases. SKIP for meta/always-on skills
where announcing is noise — `using-skillgrid` (the router), `test-driven-development`
(always-on habit), `test-driven-verification` (a claim-check, fires at "done").
-->

## Overview

<!-- Elevator pitch: what it does + why it matters, 2-4 sentences. This is
     where an Iron Law / core principle goes if the skill has one, e.g.
     "ALWAYS find root cause before attempting fixes." Keep it to the
     principle — the procedure belongs in the Core section below. -->

<One or two sentences: what the skill does and why it matters.>

## When to Use

<!-- BOTH directions, always. Positive triggers as bullets, then the negative
     exclusion. The "When NOT to use" line is what stops over-application. -->

- <trigger: task type, symptom, or moment>
- <trigger>

**When NOT to use:** <the task where this skill would be the wrong tool.>

## <Core Process / The Workflow>

<!-- The heart. Numbered steps or phases, specific and actionable:
     Good: "Run `npm test` and confirm exit 0."  Bad: "Make sure the tests work."
     Use an ASCII flowchart (``` fence) for decision points / loops — ASCII over
     Mermaid (cheaper, diffs cleanly, reads in any terminal). Read
     `.skillgrid/config.yaml` here if config-driven, naming the exact keys. -->

1. <step — specific, with the command / file / expected value>
2. <step>
3. <step>

<!-- ASCII flowchart for a decision point, if the process branches:
```
<decision>
├── <condition A> → <action>
└── <condition B> → <action>
```
-->

## <skill-specific section>

<!-- Add as many as needed: rules, patterns, techniques, examples. Keep what's
     under ~50 lines inline; push longer reference detail to `references/`. -->

## Common Rationalizations

<!-- REQUIRED. The highest-leverage anti-rationalization device. A 2-column
     table pairing the excuse an agent uses to skip a step with the factual
     rebuttal. Every skip-worthy step in the Core section needs ≥1 row. -->

| Rationalization | Reality |
|---|---|
| "<excuse the agent will make>" | <why the excuse is wrong — factual, not a lecture.> |
| "<excuse>" | <rebuttal.> |

## Red Flags

<!-- REQUIRED. A bullet list of OBSERVABLE signs the skill is being violated —
     symptoms you'd spot in review or self-monitoring, not rules. -->

- <observable sign of violation>
- <observable sign>

## Verification

<!-- REQUIRED. The exit criteria. A - [ ] checklist where EVERY box is
     evidence-gated: provable with test output, a build result, an exit code,
     a diff, or a screenshot. "Seems right" is not a checkbox.
     (The skill checks itself here; `qa` checks the change at the ladder level.) -->

- [ ] <exit criterion — verifiable with evidence>
- [ ] <exit criterion>
- [ ] <exit criterion>

## References

<!-- OPTIONAL. Last section, only if this skill has `references/` or `templates/`
     files. One line each so the agent knows what to load and when. -->

- [references/<name>.md](references/<name>.md) — <what it holds, load when>.

<!-- ===== COMPLIANCE GATE — confirm before the skill is "done" =====
     (from anatomy §New-skill checklist)
- [ ] Directory name == `name` == kebab-case
- [ ] description has what + when, NO workflow steps
- [ ] license: MIT + metadata block (based_on/source if derived)
- [ ] Title is `# {Name}` title-cased; Announce line present (or documented skip)
- [ ] `## When to Use` has a `**When NOT to use:**` line
- [ ] `## Common Rationalizations` table present (≥1 row per skip-worthy step)
- [ ] `## Red Flags` list present
- [ ] `## Verification` checklist present, every box evidence-gated
- [ ] Total ≤ 400 lines (or ≤ 500 heavy / ≤ 600 orchestrator; else push tail to references/)
- [ ] No content duplicated from another skill — referenced by `skillgrid:{name}` instead
-->
