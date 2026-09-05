---
name: sdd-constraints
description: "Interview and write the quality bar in config under docs/skillgrid/config.yaml rules.* only. Use when onboarding, setting TDD/coverage/verify gates, or refreshing SDD quality rules."
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  family: sdd
---

# SDD Constraints

Capture the project's **quality bar in config** — `docs/skillgrid/config.yaml` → `rules.*` only. No root `CONSTRAINTS.md` by default (optional human export only; never a second SoT).

Layout: [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md).  
Interview style: load `questioning` when answers branch or conflict.

## Hard Rules

- Write **only** `rules.*` (and related testing fields the user confirms). Do not invent parallel constraint docs.
- Detect existing `rules:` before asking — propose diffs, don't blind overwrite.
- Keep `context:` short (<10 lines); quality gates live under `rules`, not `context`.
- Default + confirm — do not present a long equal menu of philosophies.

## Workflow

```
[ ] 1. Read docs/skillgrid/config.yaml (create via sdd-init if missing)
[ ] 2. Interview the quality bar
[ ] 3. Patch rules.* only
[ ] 4. Confirm with user; summarize
```

### 1. Read config

If `config.yaml` missing → stop and run/ask for `sdd-init` first.

### 2. Interview (short confirmations)

Ask one cluster at a time:

1. **Propose gates** — rollback, success criteria, ADR-style rationale, threat rows?
2. **Spec gates** — NN ownership, task sizing, acceptance coverage (happy + edge + failure)?
3. **Apply** — TDD on/off, `test_command`, "follow existing patterns", mark `[x]` as you go?
4. **Verify** — `test_command`, `build_command`, `coverage_threshold`?
5. **Archive** — require PASS / PASS WITH WARNINGS on every step?

Reuse detected testing commands from config/`sdd-init` as defaults.

### 3. Write `rules.*`

Shape (match existing project schema; example keys):

```yaml
rules:
  explore: []
  propose:
    - Include success criteria (measurable)
  spec:
    - Own NN allocation; one ## section per Step Blueprint entry
  apply:
    guidelines:
      - Follow existing code patterns
      - Mark tasks [x] as you go (in tasks.md)
    tdd: false
    test_command: ""
  verify:
    test_command: ""
    build_command: ""
    coverage_threshold: 0
  archive:
    - Require every step in tasks.md to have PASS or PASS WITH WARNINGS before move
```

Only change keys the user confirmed. Leave unrelated YAML untouched.

### 4. Summary

List `rules` keys updated. State explicitly: SoT is `config.yaml` — not root CONSTRAINTS.md.

## Gotchas

- Root `CONSTRAINTS.md` duplicates and drifts — do not create unless the user explicitly wants an export that *cites* `rules.*`.
- Empty `test_command` is fine until the user supplies one — do not invent fake CI commands.
- `tdd: true` without a real `test_command` is a footgun — confirm both together.
