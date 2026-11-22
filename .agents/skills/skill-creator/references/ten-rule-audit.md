# The 10-Rule Audit

## 6. The 10-Rule Audit

Run this audit before a skill ships. It covers the four phases plus
cross-cutting concerns from Anthropic's updated Skills Guide.

```
 1. > 500 lines? → Split into references/
 2. > 100 lines, no TOC? → Add one
 3. Too many MUST/NEVER for a creative skill? → Reduce to reasoning.
    Same concept named two ways (e.g. "endpoint" + "URL", "field" +
    "control")? → Pick one term, sweep the file.
 4. Model-specific behavior? → Add model notes
 5. "You are X" preamble or domain explanations? → Delete.
    Time-sensitive content ("before August 2025", "currently in beta")?
    → Move to a collapsible "Old patterns" section or delete.
 6. Multi-step where ORDER matters, without copyable checklist? → Add with
    go-back lines. If order doesn't matter, don't use a checklist — give the
    goal instead.
 7. No verify-then-repeat? → Add feedback loop (see §7)
 8. No template/examples/conditional for defined output? → Add.
    Output not reproducible: an input that changes the output is not
    pinned in the Input Contract, scripted, or explicitly variable?
    → Pin it (Input Contract) or name it (Output Contract).
 9. Assumes installed tools without instructions? → Add prerequisites
10. Critical rules in prose that should be code? → Identify hook candidates
```

Four-phase checks (T=Trigger, S=Structure, St=Steering, P=Pruning):

```
 T1. Invocation mode deliberate (model- vs user-invoked is a chosen trade-off)
 T2. Description: third-person, trigger-specific, ≤ 1024 chars, no steps; AND
     passes the pointer-quality rules in §2 (leading word front-loaded, one
     trigger per branch, no identity restating the body)
 T3. Invocation consequences hold (§2 "Consequences"): user-invoked shells
     only call model-invoked skills (never another user-invoked); shared
     reference needed by >1 user-invoked skill lives in _shared/ (a plain
     file), not in a user-invoked skill; a read-only skill is a pointer, not a
     skill
 S1. Every step classified: deterministic or judgment
 S2. Every deterministic step has a tested, executable script
 S3. Single-branch reference behind a context pointer, not inlined
 S4. Scripts have documented exit-code contracts (0/1/2)
 S5. Operational skill: Input Contract + Decision Gates + Output Contract
     present (or the skip is deliberate and stated); every gates row sourced
     from an existing rule — no net-new policy (§3e)
 St1. At least one leading word, repeated consistently, and it passes both
     tests in per _shared/craft/leading-words.md (no-op test, pretrain-first)
 St2. Legwork split considered where a step needs more effort
 U1. Interactive gates follow the EGM format (§5c): one question, sole tool
     call, no (Recommended) prefix, open-ended vs single-select correctly
     applied, gate turn = chat summary then gate
 U2. Multi-step workflows carry a Mermaid diagram (§5d) with gate diamonds,
     labeled option edges, explicit loop-backs; diagram matches prose
 P1. No-ops, sediment, duplication pass the deletion test
 P2. skill-size-budget.mjs exits 0
 F1. check-frontmatter.mjs check . exits 0 (field set, order, name=dir,
     description <= 1024, metadata block with quoted semver + part-of)
```

Report findings as a table with columns: Rule, Status (PASS/WARN/FAIL),
Finding, Fix.

---

