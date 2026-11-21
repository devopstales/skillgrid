# Glossary

One meaning per term. Do not use a synonym in its place.

## People

| Term | Definition | Use when | Avoid |
|---|---|---|---|
| **Programmer** | The person directing SkillGrid: owns the repository, states the idea, answers gates, approves ship. The role this skill addresses. | Any instruction, question, or checkpoint addressed to the person running the skill. | "user", "author", "operator", "developer" (unless the host's own UI uses it). |
| **User** | The person who runs the finished program. A role in the product's world, not in the build. | Describing end-user behavior, UX, "user-facing", or simulated users in blind validation. | "programmer" (the builder), "customer" unless that is the business term. |
| **Author** | Book-era word for the programmer. Retired in 2.0. | Only when reading pre-2.0 text. | New text. Write "programmer". |
| **Maintainer** | A person who approves merge or release when that authority is not the programmer's. | Release approval, protected branches, upstream projects. | "user", "owner" (ambiguous with code ownership). |
| **Reviewer** | An agent or person that reads a diff or artifact and returns findings, independent of the producer. | Code review, blueprint review, adversarial audit. | "editor" (book-era), "critic" when a reviewer is meant. |

## Work

| Term | Definition | Use when | Avoid |
|---|---|---|---|
| **Program** | The software being built. | The product scope, README, delivery. | "book", "manuscript", "sourcecode" (one word). |
| **Change** | One self-contained pipeline unit under `.skillgrid/change-specs/<topic>/`, from briefing to archive. | Referring to the unit of work end to end. | "ticket" (a smaller slice), "PR" (an integration artifact). |
| **Work unit** | A slice of a change that can ship as one focused commit or PR, with its own test and rollback boundary. | Slicing, execution waves, PR planning. | "chapter", "step" (overloaded with SDD steps). |
| **Ticket** | The tracker record for a work unit (Backlog.md, GitHub, GitLab, Jira). | Ticketing, dependency edges, status transitions. | "task" when the tracker record is meant; "change" (broader). |
| **Phase** | A named stage of the pipeline (`briefing`, `blueprint`, `acceptance`, `slicing`, `execution`, `qa`, `review`, `ship`, `reflect`). | State, routing, checkpoints. | "step" (tickets and SDD steps are different). |
| **Gate** | A persisted decision point in `state.yaml` (`blueprint-validation`, `user-gate-slice`, `qa-verdict`, `ship-decision`) that waits for the programmer. The id `user-gate-slice` is a protocol constant; the person it waits for is the programmer. | Check-in, resume, blocking. | "checkpoint" (the screen shown at a gate), "pause" (not persisted). |
| **Artifact** | A file a phase persists (`briefing.md`, `blueprint.md`, `tasks.md`, `acceptance.feature`, `report.md`, `review.md`). | Cross-referencing phase outputs. | "document" (too generic). |
| **Ship** | Integrate verified work to its base branch (merge, PR, or keep) and move the change folder to `change-archive/`. | The integration phase. | "publish", "release" (release mechanics are out of scope). |

## Genesis run

| Term | Definition | Use when | Avoid |
|---|---|---|---|
| **Genesis block** | The optional `genesis:` object under a change entry in `state.yaml`, holding independence, audit status, pass counts and the quality target. | Reading or writing run state. | "genesis.yaml" (there is no sidecar). |
| **Independence grade** | A, B or C: how far the reviewer is from the producer. A = another model family, B = isolated subagent, C = same context. | Every verdict. | "independent" with no grade. |
| **Audit status** | The auditor's single verdict: `pass`, `revise` or `major_rewrite`. | Phase 4, Phase 5 routing. | "approved" (the gate is separate). |
| **Quality target** | The calibrated score the score gate compares against, default 8.5. | Phase 5, Phase 6. | "passing score" (unquantified). |
| **Panel** | The four simulated users: primary, hostile, adjacent, casual. | Phase 0, 3, 5. | "reviewers" (the panel is not the audit). |

The Genesis run terms are internal to this skill. Onboarding copies only the People and Work tables into a project's business glossary.

## Rule

If a sentence could be read as "the person using the program" or "the person building it", rewrite it. "The programmer approves the ship" and "the user sees an error" are different claims; never collapse them into "the user approves".
