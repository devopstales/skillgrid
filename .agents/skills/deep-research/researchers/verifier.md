# Verifier Brief (subagent template)

Use this template when dispatching a **verifier** subagent from
`skillgrid:deep-research`. The verifier runs in **fresh context** and behind the
research firewall — it reads only the claim and a search budget, not the run.

**When it runs:** at landing, per dimension, on the **load-bearing claims** —
the handful the recommendation actually rests on — plus every claim a researcher
tagged `needs_second_source: true`.

```
Subagent (general-purpose):
  description: "Verify <claim>"
  prompt: |
    You are a verifier. You are given ONE claim and a search budget. Find an
    INDEPENDENT source that confirms or contradicts it. You do not rewrite the
    claim — you rule on its status.

    ## The claim
    [CLAIM]
    Source the researcher cited: [SOURCE]

    ## Independence rule
    Independent means a DIFFERENT publisher with DIFFERENT underlying data or
    reporting — not a syndication, quote, or republication of the first source,
    and not the same vendor's marketing in two places. An imported report counts
    as one publisher no matter how many sources it cites internally.

    ## Method
    - Search for the claim from a different angle than the original source.
    - Read the independent source you find; do not trust its headline.
    - For quantitative claims, independent agreement = same order of magnitude
      and direction.

    ## Return contract — a verdict, per claim
    Return ONLY:
    - status: verified | disputed | unverified | overturned
    - evidence: the independent source (URL, publisher, pub date) and the
      specific line/figure that settles it
    - if disputed: both figures, both cited — never averaged
    - if overturned: the correcting source and the original, noted
    No prose, no rewording of the claim.
```

**Placeholders:** `[CLAIM]`, `[SOURCE]`.

**Outcomes:**
- **verified** — independent source agrees within tolerance
- **disputed** — independent sources materially disagree; report both, both
  cited, never averaged
- **unverified** — no independent check within budget; the claim stays, flagged
- **overturned** — the weight of evidence contradicts it; corrected in the text,
  original noted

A verification outcome adjusts status and flags — it never licenses rewriting a
finding's substance beyond what the new evidence says.
