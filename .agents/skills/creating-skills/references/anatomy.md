# SKILL.md anatomy

## Frontmatter fields

| Field | Required | Rules |
|-------|----------|-------|
| `name` | Yes | 1-64 chars, `[a-z0-9-]` only, no leading/trailing hyphen, no consecutive hyphens, must equal the parent directory name |
| `description` | Yes | 1-1024 chars, what it does + when to use it, with trigger keywords. For **model-invoked** skills this is the always-loaded context pointer — keep it sharp. For **user-invoked** skills it is primarily human-facing |
| `disable-model-invocation` | No | Set `true` for **user-invoked** skills (description not offered to the agent as an auto-trigger). Omit or leave unset for model-invoked. Harness support varies — match sibling skills |
| `license` | No | Short: a license name or "LICENSE.txt has complete terms" |
| `compatibility` | No | 1-500 chars, only if there are real environment requirements |
| `metadata` | No | Map of string→string. Recommend unique keys, e.g. `author`, `version` |
| `allowed-tools` | No | Space-separated pre-approved tool list (experimental, support varies by client) |

### name: valid vs invalid

```yaml
name: pdf-processing    # valid
name: data-analysis     # valid
name: PDF-Processing    # invalid — uppercase
name: -pdf              # invalid — leading hyphen
name: pdf--processing   # invalid — consecutive hyphens
```

## Minimal valid SKILL.md

```markdown
---
name: skill-name
description: A description of what the skill does and when to use it.
---

Instructions here.
```

## Recommended body sections

- Steps (procedure) and/or reference (supporting material)
- Context pointers to branch-only `references/` files
- A working example (input → output) when format is non-obvious
- Gotchas (concrete corrections)
- Leading words repeated where they should steer behavior

## User-invoked example

```markdown
---
name: grill-me
description: Human-facing summary of when to run this skill.
disable-model-invocation: true
---

Instructions here.
```

## File reference style

From SKILL.md, reference other files with relative paths, one level deep. Prefer **context pointers** that say *when* to load:

```markdown
For the full quality rubric, read [references/checklist.md](references/checklist.md).
Run the extractor:
scripts/extract.py
```

## Validation

```bash
scripts/validate_skill.sh ./skill-dir     # this skill's bundled validator
# or, if skills-ref is available on the machine:
skills-ref validate ./skill-dir
```
