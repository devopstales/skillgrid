# ASSUMPTIONS — [Product Name]

> Create this as the root `.skillgrid/ASSUMPTIONS.md` (the understanding +
> decisions record) and fill in every section before implementation. For a
> multi-project repo, merge rather than overwrite: keep existing content and add
> the new project's scope alongside it.
>
> The file has four tiers — **VERIFIED**, **INFERRED (HYPOTHESIS)**, **LOCKED**,
> and **Open Questions** (plus the in-force ADR set). Everything below is the
> VERIFIED product-facts content that used to live in a separate PRD; the
> decisions (ADRs) live as `### ADR-NNNN` entries in the `## LOCKED` section
> (see the `skillgrid:architectural-decision-records` templates), and the locked
> constraints live in `## LOCKED → ### Locked constraints`.
>
> Maintenance: VERIFIED is written by brainstorming as facts are confirmed during
> the interview. INFERRED is a hypothesis (name the assumption it rests on).
> LOCKED requires an explicit user OK. Supersede ADRs by link + table flip —
> never delete an ADR entry.

## VERIFIED

Confirmed against the code, a spike, or a primary source. These are the facts the rest of the record stands on.

### Product Overview

**Product Vision:** [1-2 sentence description of what this product is and why it exists]

**Target Users:** [Primary and secondary user personas]

**Business Objectives:** [Key business goals this product aims to achieve]

**Success Metrics:** [How success will be measured — specific and measurable]

## User Personas

### Persona 1: [Name]
- **Demographics:** [Age, occupation, technical proficiency]
- **Goals:** [What they want to accomplish]
- **Pain Points:** [Current challenges they face]
- **User Journey:** [How they'll interact with this product]

### Persona 2: [Name]
- **Demographics:** [Age, occupation, technical proficiency]
- **Goals:** [What they want to accomplish]
- **Pain Points:** [Current challenges they face]
- **User Journey:** [How they'll interact with this product]

## Feature Requirements

Prioritize with MoSCoW: **M**ust-have / **S**hould-have / **C**ould-have / **W**on't-have (this release).

| Feature | Description | User Stories | Priority | Acceptance Criteria | Dependencies |
|---------|-------------|--------------|----------|---------------------|--------------|
| **[Feature 1]** | [Brief description] | [As a user, I want to...] | [Must/Should/Could/Won't] | [List of criteria] | [Dependencies] |
| **[Feature 2]** | [Brief description] | [As a user, I want to...] | [Must/Should/Could/Won't] | [List of criteria] | [Dependencies] |
| **[Feature 3]** | [Brief description] | [As a user, I want to...] | [Must/Should/Could/Won't] | [List of criteria] | [Dependencies] |

## User Flows

### Flow 1: [Name, e.g., User Registration]
1. [Step 1]
2. [Step 2]
3. [Step 3]
   - [Alternative path]
   - [Error state]

### Flow 2: [Name]
1. [Step 1]
2. [Step 2]
3. [Step 3]
   - [Alternative path]
   - [Error state]

## Non-Functional Requirements

### Performance
- **Load Time:** [Target load time]
- **Concurrent Users:** [Expected number]
- **Response Time:** [Target response time]

### Security
- **Authentication:** [Requirements]
- **Authorization:** [User permission levels]
- **Data Protection:** [Requirements]

### Compatibility
- **Devices:** [Supported devices]
- **Browsers:** [Supported browsers and versions]
- **Screen Sizes:** [Supported dimensions]

### Accessibility
- **Compliance Level:** [e.g., WCAG 2.1 AA]
- **Specific Requirements:** [Key accessibility features]

## Technical Specifications

### Frontend
- **Technology Stack:** [Framework, libraries]
- **Design System:** [Design system to use]
- **Responsive Design:** [Requirements]

### Backend
- **Technology Stack:** [Languages, frameworks]
- **API Requirements:** [RESTful, GraphQL, etc.]
- **Database:** [Database type and structure]

### Infrastructure
- **Hosting:** [Hosting solutions]
- **Scaling:** [Scaling requirements]
- **CI/CD:** [Deployment process]

## Analytics & Monitoring

- **Key Metrics:** [Metrics to track]
- **Events:** [User events to capture]
- **Dashboards:** [Required dashboards]
- **Alerting:** [Alert thresholds]

## Release Planning

### MVP (v1.0)
- **Features:** [List of MVP features]
- **Timeline:** [Expected release date]
- **Success Criteria:** [How to measure MVP success]

### Future Releases
- **v1.1:** [Feature set and expected timeline]
- **v1.2:** [Feature set and expected timeline]
- **v2.0:** [Feature set and expected timeline]

## INFERRED (HYPOTHESIS)

Treated as true but not yet confirmed. Each rests on a named assumption; promote
to VERIFIED (with evidence) or LOCKED (with a user OK) before it carries a
decision.

- **H1:** [Hypothesis] — Assumed [what]. *Not confirmed* by [what].

## LOCKED

Decisions (ADRs) and user-locked constraints. Adding here requires an explicit
user OK.

### In-force set

Single source for what is **currently in force**. **In force** = `status: accepted`
AND no later ADR's `supersedes` names it. Superseded / deprecated entries stay in
the table (frozen) but are marked out of force. **IRON RULE: never delete an ADR
entry — supersede by adding a new entry that names it and flipping its row here.**

| # | Title | Status | Supersedes | Amends | Date | In force |
|---|-------|--------|------------|--------|------|----------|
| 0001 | [title] | accepted | — | — | YYYY-MM-DD | yes |

**Highest sequence in use:** 0001 (next ADR is `### ADR-0002` below).

### ADR-0001 — [title]

**Status.** accepted
**Date.** YYYY-MM-DD
**Supersedes.** —

[Body per the `skillgrid:architectural-decision-records` `adr_style` template.]

### Locked constraints

These override per-change decisions and are the hard limits a change must respect.
A constraint is locked only when the user says so — inferred limits belong in
VERIFIED or an ADR entry, not here. The `### Rules` section of `AGENTS.md` is
rendered from this list, one bullet per constraint. `state.yaml constraints_ref`
points at this file.

- [constraint 1]
- [constraint 2]

*Unlocked (historical):* (none yet). A constraint that is later unlocked gets moved
here with the date — it is not deleted, so the boundary history stays readable.

### Locked assumptions

- [locked assumption 1]

## Open Questions

Questions that are not yet decided and not yet spiked.

- **Question 1:** [Open question]
- **Question 2:** [Open question]

## Appendix

### Competitive Analysis
- **Competitor 1:** [Strengths and weaknesses]
- **Competitor 2:** [Strengths and weaknesses]

### User Research Findings
- **Finding 1:** [Key insight from research]
- **Finding 2:** [Key insight from research]

### Glossary
- **Term 1:** [Definition]
- **Term 2:** [Definition]
