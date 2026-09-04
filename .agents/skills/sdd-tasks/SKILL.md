---
name: sdd-tasks
description: "RETIRED in SDD v3. Use sdd-spec (writes tasks.md + acceptance.feature). This stub only redirects."
disable-model-invocation: true
user-invocable: false
license: MIT
metadata:
  author: devopstales
  version: "3.0"
  family: sdd
  part-of: skillgrid
  status: retired
  successor: sdd-spec
  phase-order: "init → explore → propose → spec → apply → verify → archive"
  delegate_only: true
---

# sdd-tasks — RETIRED

**Do not use this skill.** In SDD v3, `sdd-spec` absorbs task decomposition and writes change-level **`tasks.md`** + **`acceptance.feature`** from templates. There is no `steps/` directory.

| Was (v2) | Now (v3) |
|---|---|
| `sdd-tasks` → `steps/<NN>/tasks.md` | `sdd-spec` → `tasks.md` |
| `sdd-spec` → `steps/<NN>/acceptance.feature` | `sdd-spec` → `acceptance.feature` (`@step-NN`) |

**Next action:** load [`../sdd-spec/SKILL.md`](../sdd-spec/SKILL.md).

Templates: [`../_shared/templates/template-tasks.md`](../_shared/templates/template-tasks.md), [`../_shared/templates/template-acceptance.feature`](../_shared/templates/template-acceptance.feature).

Layout: [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md).
