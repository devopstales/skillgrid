# Skill Template — AISkillGrid Skill Definition

**Skill name:** `<skill-name>`
**Description:** [When to use — 1 sentence]
**Trigger:** [Context pattern that activates skill]
**License:** Apache-2.0 (default)

---

## Metadata Header (required)

All skill files MUST start with YAML front matter:

```yaml
---
name: <kebab-case-skill-name>
description: >-
  Clear description of skill's purpose and when to invoke it.
  Multi-line allowed.
trigger: <comma-separated trigger patterns>
mode: <orchestrator | subagent | manual | reviewer | gate>
enforcement: <mandatory | recommended | optional>
version: "1.0"
dependencies: [<other skill names>]
author: <your-id>
license: Apache-2.0
metadata:
  custom_key: custom_value
  integration_source: superpowers  # if adapted from external methodology
---
```

**Required fields:** `name`, `description`, `trigger`
**Recommended:** `mode`, `enforcement`, `version`, `dependencies`

---

## Overview

One-paragraph summary of what the skill does. How it fits into the workflow. Which phase(s) it participates in.

**Core principle:** [fundamental rule this skill enforces]

---

## When to Use

List triggers explicitly:
- Invoked by: which command(s)?
- Auto-trigger conditions: which context patterns?
- Manual invocation: `/skill-name` command?

**Do NOT use when:** [common misapplication]

---

## Input

What the skill expects to find:
- Artifacts (spec, tasks, design)
- Filesystem state (branch, working tree)
- Configuration keys
- Engram entries

If artifact store mode aware (engram/openspec/hybrid): document retrieval pattern.

---

## Execution

Step-by-step procedure:

1. **Step 1 title**
   Detailed instructions, code snippets, commands with expected output

2. **Step 2 title**
   Another step with examples

Use checklists where appropriate:
- [ ] Item checked
- [ ] Another item

---

## Output

What the skill produces:
- Files created/modified
- Engram entries saved
- State updates
- Return envelope per `skills/_shared/sdd-return-envelope.md`

Example return:
```yaml
status: completed
executive_summary:
  overview: "..."
  used_tokens:
    input: 0
    output: 0
    total: 0
detailed_report: null
artifacts:
  - path/to/file.md
next_recommended: "..."
risks: none
skill_resolution: injected
```

---

## Rules

Bulleted list of enforceable rules:
- MUST / MUST NOT / SHOULD / SHOULD NOT
- Specific prohibition or requirement
- Validation logic

---

## Anti-patterns

What not to do:
- ❌ Common mistake with example
- ❌ Another pitfall

**Contrast:** ✅ Correct approach.

---

## Integration

Skills thisskill calls/invokes:
```
Invoke: other-skill-name (use skillgrid-handoff.md pattern)
```

Skills that call thisskill:
```
Used by: sdd-apply, sdd-verify, ...
```

Command routing:
```
/some-command → delegates to this skill
```

---

## Configuration

Config keys in `.skillgrid/config.json` that affect behavior (see `docs/04-config-reference.md`):

```json
{
  "skill_name": {
    "option_a": true,
    "max_retries": 3
  }
}
```

---

## Testing

**Unit tests:** How to test skill logic in isolation (mock inputs, verify outputs)

**Integration tests:** How skill behaves in full SDD workflow

**Test scenarios:**
- Normal path
- Error paths (missing artifacts, invalid input)
- Edge cases (empty tasks, huge task list)

---

## References

- Source methodology (if adapted): `superpowers:original-skill-name`
- Related skills: `skill-a`, `skill-b`
- Documentation: `docs/XX-skill-name.md` (if separate explainer)
- ADR: `.skillgrid/adr/NNNN-skill-adoption.md` (if architecture decision involved)

---

## Changelog

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2026-05-13 | Initial version (Superpowers integration) |
