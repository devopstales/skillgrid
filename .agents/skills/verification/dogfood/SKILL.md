---
name: dogfood
description: "Exploratory QA of a web app in a real browser: plan a sitemap, click through flows, check the console, and report classified findings with evidence. Use when asked to QA, dogfood, or exploratory-test a URL."
license: MIT
metadata:
  author: Teknium, Hermes Agent (ported; see ADR-0015)
  version: "1.0"
  part-of: skillgrid
---

# Dogfood: Systematic Web Application QA Testing

**Announce at start:** "I'm using the skillgrid:dogfood skill to exploratory-test this app."

## Overview

Systematic exploratory QA of a web application in a real browser: plan the sitemap, walk every flow, check the JS console after every navigation and interaction, capture screenshot evidence per issue, classify by severity × category, and deliver a structured report from `templates/dogfood-report-template.md`.

This is the exploratory complement to `skillgrid:qa` (the gate) and `skillgrid:requesting-code-review` (the bug hunt on a diff): those verify a change, this one finds what nobody specified. A genuine correctness bug surfaced here is reported as a separate "found a bug" note — and may become a `skillgrid:structured-debugging` task — not folded into cleanup.

Ported from the Hermes `dogfood` skill under MIT (see ADR-0015). Behavioral delta from upstream: Hermes `browser_*` tool names mapped to skillgrid browser tools (table below); output dir defaults to the skillgrid scratch zone.

## When to Use

- User asks to QA, dogfood, exploratory-test, or "click through" a URL (local dev server, preview deploy, production)
- Pre-release sweep of user-facing flows the acceptance contract doesn't cover
- Triaging "the app feels broken" with no specific failing test

**When NOT to use:** For verifying a spec'd change — use `skillgrid:qa` (+ `acceptance.feature`). For reviewing a diff — use `skillgrid:requesting-code-review`. When no browser tooling is available in this session — say so and stop, don't narrate a test you can't run.

## Prerequisites

- Browser tooling available (agent-browser `agent-browser_*` or Playwright `playwright-browser_*`). Confirm before Phase 1 — the whole skill is browser-driven.
- From the user: **target URL**, **scope** (areas/flows, or "full site"), optional **output dir** (default `.skillgrid/sdd/dogfood-output/`, gitignored).

## Tool mapping (Hermes → skillgrid)

| Hermes (upstream wording) | skillgrid |
|---|---|
| `browser_navigate` | navigate to URL (`agent-browser open` / `playwright navigate`) |
| `browser_snapshot` | accessibility snapshot |
| `browser_click(ref)` / `browser_type(ref)` | click / type by snapshot `@ref` |
| `browser_press` / `browser_scroll` / `browser_back` | press key / scroll / back |
| `browser_console` | console messages (clear after every nav + interaction) |
| `browser_vision(question, annotate)` | screenshot + snapshot refs for element labels |

## The Process

### Phase 1 — Plan

Create `{output_dir}/screenshots/`. Fix the scope, then sketch the sitemap under test: landing, header/footer/sidebar nav, key flows (signup, login, search, checkout), forms + interactive elements, edge cases (empty states, error pages, 404s).

### Phase 2 — Explore

Per page/feature: navigate → snapshot → **console (cleared)** → screenshot-assess → exercise interactives (valid + invalid + empty inputs, keyboard Tab/Enter, scroll below the fold) → after each interaction re-check console, visual diff, expected-vs-actual. Silent JS errors are the highest-value findings — never skip the console check.

### Phase 3 — Collect evidence

Per issue record: URL, steps to reproduce, expected vs actual, console errors, screenshot path. Classify with `references/issue-taxonomy.md` (severity Critical/High/Medium/Low × Functional/Visual/Accessibility/Console/UX/Content).

### Phase 4 — Categorize

De-duplicate (same bug in two places = one issue), finalize severity/category, sort Critical-first, count by severity × category for the summary.

### Phase 5 — Report

Render `templates/dogfood-report-template.md` to `{output_dir}/report.md`: executive summary with counts, per-issue sections (repro, expected/actual, screenshot refs, console output), summary table, coverage (tested / not-tested / blockers), notes.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "The snapshot looks fine, skip the screenshot" | Visual-only bugs (overlap, broken media, layout shift) never appear in the tree. Screenshot every page. |
| "No console errors on load, checking once is enough" | Interactions throw what loads don't. Check after every significant interaction. |
| "I'll test the happy path; edge cases are qa's job" | `qa` verifies specified behavior. Empty states, invalid input, and 404s are exactly what this pass owns. |
| "One screenshot per issue is overkill" | The screenshot IS the finding's proof. No evidence, no issue. |
| "This overlaps qa, I'll skip dogfood" | Different passes: qa gates a change against its contract, dogfood finds what no contract states. |

## Red Flags

- Testing without a written scope (drifts into random clicking)
- Console never cleared between steps (stale errors attributed to the wrong interaction)
- Severity assigned by annoyance instead of the taxonomy (re-read `references/issue-taxonomy.md`)
- Blocking findings (auth walls, broken nav) not recorded as blockers in the report
- Screenshots saved outside `{output_dir}/screenshots/` (evidence must travel with the report)

## Verification

- [ ] Scope fixed up front; report lists tested / not-tested / blockers
- [ ] Console checked after every navigation and significant interaction
- [ ] Every issue has repro steps, expected-vs-actual, severity × category, and a screenshot path
- [ ] Issues de-duplicated and sorted Critical-first with severity counts
- [ ] Report rendered from the template to `{output_dir}/report.md`
- [ ] Genuine correctness bugs surfaced as separate "found a bug" notes (not silently fixed)
