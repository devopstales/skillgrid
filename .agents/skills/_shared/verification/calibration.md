# Score Calibration (anti-inflation)

Internal evaluation — a self-assessment, a self-review, an agent scoring its own or a sibling agent's work — is biased **upward**. An evaluator that wrote the code, drafted the spec, or shares the implementer's context tends to grade toward approval. Calibration is the standing rule that keeps a self-score from being mistaken for an independent one. It applies to any quantitative or threshold claim an agent makes about work it is also producing or judging (QA self-checks at T0, self-scored review, readiness claims, "confidence: high" calls on its own output).

## When it applies

Any time an agent:
- scores or rates work it authored or that a shared-context sibling authored,
- claims a threshold was met ("cleared the bar", "score > 8", "ready to ship") based on an internal score, or
- reports a confidence level on its own judgment rather than an external check.

It does **not** apply to an external, fresh-context evaluator (a dispatched reviewer subagent, a human, a second model family) that saw only the work product — that score is already independent; report it as-is.

## The rule

1. **Freeze the raw score first.** Collect the internal evaluation and write down the raw value *before* any adjustment or any threshold comparison. Never adjust in place.
2. **Deflate the internal score.** Subtract the inflation delta — **0.8** on a 0–10 scale by default (≈ 8 percentage points on a 0–100 scale) — unless a prior project calibration in memory says otherwise. The delta is a starting prior, not a law; if `mem_search("calibration")` surfaces a project-tuned delta, use that.
3. **Report both.** Every internal score is reported as a pair:
   ```
   Raw (self):      8.4
   Calibrated:      7.6   (raw − 0.8)
   Threshold:       8.0
   Clears:          NO
   ```
   A threshold claim ("cleared", "met", "passing") is only valid against the **calibrated** value. The raw value may exceed the threshold; that is expected and is exactly what the delta corrects.
4. **Take the floor, not the mean.** When several lenses / dimensions / reviewers are scored, use the **lowest** calibrated value as the gate number — never the average. A single weak dimension is the gate; averaging hides it behind enthusiasm.
5. **Calibration only after freeze.** Apply the deflation after the raw independent-style reports exist, then apply the project threshold only after calibrated scores exist. Order matters: raw → calibrated → threshold.

## Independence grade

Label every self-touching evaluation with an independence grade so its weight is explicit (mirrors the evaluator protocol):

- **A** — scored by a fresh context with no shared history with the producer (dispatched reviewer, second model family). No deflation required; report raw.
- **B** — fresh contexts but same model family / same toolchain as the producer. Apply the default delta.
- **C** — scored in the producer's own context (self-grade) or prior scores leaked in. Apply the delta **and** mark the result **diagnostic only** — it can guide a fix, but it cannot support an "independently validated" or "ready" claim.

A Grade C score may never be the sole evidence for a readiness verdict.

## Cap on simulated approval

A simulated or self-generated approval ("a simulated reader liked it", "the self-check passed", "confidence: high from the same model") is capped at a **flag**, not a pass, until a human or a genuinely external evaluator confirms it. Readiness / "ship it" claims require at least one Grade A or human-validated input.

## Red flags

- Claiming a threshold was met using the raw self-score, not the calibrated one
- Reporting only the calibrated number and hiding the raw (or vice versa)
- Averaging across dimensions / reviewers and calling the mean the gate
- Adjusting a score in place instead of freezing raw → deflating → comparing
- A Grade C (self) score standing as the only evidence for "ready to ship"
- "Confidence: high" on one's own output with no external check named

## Reference

Source pattern: test-book `book-genesis` Literary Barrier loop + `book-swarm-panel` scoring (median per lens, −0.8 deflation, lowest calibrated lens as the score, independence grades A/B/C, simulated-approval cap).
