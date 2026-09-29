# Floor Convention (shared across all Skillgrid skills)

Single source of truth for the **floor** of a multi-part judgment. The rule it
exists to enforce: **the judgment is only as strong as its weakest part — the
floor, not the average, decides.** A strong overall can never hide one weak
dimension. Before this convention, a reviewer or gate could let a high average
compensate for a single failed check, which is the failure class this
convention removes.

## The Rule

1. **The floor is the minimum, not the mean.** A judgment over N parts (the QA
   gate's audits, a review's axes, a score's dimensions) is computed on the
   **weakest part**. The overall verdict can never be *higher* than the verdict
   the weakest part earns. If any dimension is below its floor, the whole
   judgment is at or below that floor — no matter how strong the rest are.
2. **Never average a failure away.** You may report the mean alongside the
   floor (for context), but the gate branches on the floor, never on the mean.
   A `9.0` average with one `6.0` dimension is a `6.0` judgment.
3. **Every dimension must be present to raise the floor.** A missing or
   `UNREADABLE` dimension is the lowest possible floor for that judgment — an
   absent dimension is a non-answer, never a silent pass (composes with
   `verification-scope.md`: a `UNREADABLE` input is the weakest scope, and the
   weakest scope is the weakest dimension).
4. **Keep the weakest part visible.** The report names the floor dimension and
   why it is the floor — a hidden weak dimension behind an enthusiastic overall
   is the exact failure this convention removes.

## Where It Applies

- **QA gate** (`skillgrid:qa`): the four-state verdict is the floor across every
  audit and gate — the weakest one (a FAILing gate, an `UNREADABLE` scope, a
  `MISSING_RED`) sets the ceiling, not the average of the passing ones.
- **Code review** (`skillgrid:requesting-code-review`,
  `skillgrid:parallel-code-review`): the overall review verdict is the floor
  across the axes; one axis that fails caps the whole review.
- **Any scored judgment** that reports a number over several dimensions: report
  the floor explicitly and gate on it.

## Red Flags

- A PASS or "ship it" resting on an average while a single audit is FAIL.
- A strong overall that buries the one weak dimension (no floor named).
- A missing / `UNREADABLE` dimension treated as a passing dimension instead of
  the floor.
- Reporting the mean and then branching on it rather than on the floor.
