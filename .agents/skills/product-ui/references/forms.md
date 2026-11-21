# Forms & Multi-step Flows

Conventions for product UI forms and multi-step flows (checkout, configuration
wizards, onboarding). Load this when the surface contains significant forms or
flows.

## Label & field conventions

- **Label above the input.** Never placeholder-as-label. A placeholder is an
  example, not a label — it disappears on input.
- **Helper text** (optional) sits between label and input or below the input;
  it explains *how*, not *what*.
- **Error text** below the input, associated via `aria-describedby`, in a
  distinct color *and* an icon (color alone is not a valid error signal).
- **Required indicators** are consistent: mark only required fields (marking
  everything is the same as marking nothing).
- **One column** for most forms. Two columns only for genuinely paired, short
  fields (first/last name, date ranges). Do not use a grid to save vertical
  space at the cost of scanability.

## Validation timing

- **Validate on blur, not on keystroke** — for most fields. Validating on
  every keystroke is noisy and punishes typing.
- **Validate the whole form on submit.** On submit, run all validations, show
  all errors, and **focus the first error** (move focus, not just scroll).
- **Re-validate a field on the next interaction** after it was marked invalid,
  so the user sees the error clear as they fix it.
- **Do not validate before the user has interacted** with a field (no red
  borders on a pristine form).
- **Format, don't block, where possible** — auto-format phone / card / date
  fields as the user types rather than rejecting raw input.

## Error placement & summary

- **Inline, per field** — the default. The error appears at the field that
  failed.
- **Error summary** — for multi-error forms, also render a summary at the top
  of the form on submit, listing each error with a link that focuses the
  corresponding field. Focus the summary first, then the first field error on
  navigation.
- **Never** `window.alert()` for form errors. Never a full-page redirect that
  loses the user's input.

## Async submit

- **Optimistic where safe** — for low-risk actions, update the UI immediately
  and roll back on failure with a clear message.
- **Loading state on the submit control** — disable the button, show a
  progress indicator, and change the label ("Saving…"). Prevent double-submit.
- **Preserve input on failure** — the user must not re-type a failed form.
- **Partial success** — if some of a batch succeeds and some fails, report
  exactly which and why; do not fail the whole batch silently.

## Multi-step flows

- **One step = one decision.** Do not bundle unrelated decisions into a step.
- **Show progress** — a step indicator ("Step 2 of 4") or a progress bar. The
  user must always know where they are and how much is left.
- **Never lose entered data** — keep all prior step data in state; navigating
  back and forth must not reset fields.
- **Validate at step boundaries** — the user cannot advance past a step with
  invalid required fields, but the error appears at the step boundary, not on
  the field mid-typing.
- **A visible back path** — every step (after the first) has a back control.
  Hiding "back" in a flow increases abandonment.
- **Review step** — for consequential actions (checkout, submission), a final
  review/confirm step before the irreversible action.
- **Persist drafts** — for long flows, persist progress so a refresh or
  revisit does not start over.

## Onboarding-specific

- **One action per screen.** Onboarding is a Flow — each screen asks for one
  decision or one input.
- **Skippable where non-critical** — a user should be able to skip optional
  setup and return to it later. Mark skippable steps as optional.
- **Explain why** — one line of "why we're asking this" per step reduces
  friction more than any amount of polish.
- **No dead-ends** — every onboarding screen ends in a next action (or an
  explicit "finish and go to my dashboard").

## Accessibility (forms)

- Every input has a programmatically associated `<label>` (not just visual).
- Focus moves to the first error on submit; focus order matches visual order.
- Error messages are associated with the field (`aria-describedby` +
  `aria-invalid`).
- Auto-complete attributes on predictable fields (name, email, address,
  card) so password managers and browsers help.
- Touch targets ≥ 44×44pt; no tap target smaller than the visible control
  without an expanded hit area.
