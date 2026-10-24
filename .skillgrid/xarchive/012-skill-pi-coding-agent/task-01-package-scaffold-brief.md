## 01-package-scaffold

### Goal

A runnable npm package skeleton exists at repo root `skill-pi/` with correct identity (name `skill-pi`, bin `skill-pi` only, no `pi` alias), pinned `@earendil-works/pi-coding-agent`, and the six required companion/plugin dependencies declared.

### Out of scope / Non-Goals

- Extension behavior (step 02 onward)
- Asset sync (step 03)
- Go installer agent option (step 05)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-01` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Produces contracts listed under Interfaces are available to dependents
- [ ] No Global Constraint violated

> Depends on: none

**Files:**
- Create: `skill-pi/package.json`
- Create: `skill-pi/bin/skill-pi.mjs`
- Create: `skill-pi/tsconfig.json`
- Test: `skill-pi/test/scaffold.test.ts`

**Interfaces:**
- Consumes: none
- Produces: `skill-pi` npm package identity (name/bin), pinned Pi + companion deps, launcher entry — relied on by 02, 05, 08.

### Tasks

- [ ] 01.1 `[RED]` Accidental `pi` bin alias publish (THREAT) — pack/manifest asserts no `bin.pi` key and no `pi` alias
  - [ ] 01.1.a Write failing test: package `bin` must contain `skill-pi` and must NOT contain `pi`
  - [ ] 01.1.b Run to confirm fail — `Run: cd skill-pi && bun test test/scaffold.test.ts -t "no pi alias"` — Expected: FAIL
  - [ ] 01.1.c Minimal implementation: `package.json` `bin: { "skill-pi": "./bin/skill-pi.mjs" }` only
  - [ ] 01.1.d Run to confirm pass — `Run: cd skill-pi && bun test test/scaffold.test.ts -t "no pi alias"` — Expected: PASS
  - [ ] 01.1.e Commit — `feat(skill-pi): scaffold package with skill-pi bin only (no pi alias)`
- [ ] 01.2 `[RED]` Package declares pinned `@earendil-works/pi-coding-agent` dependency (WHAT: pinned Pi dep)
  - [ ] 01.2.a Write failing test: dependencies has `@earendil-works/pi-coding-agent` at a pinned (non-caret, non-star) version
  - [ ] 01.2.b Run to confirm fail — `Run: cd skill-pi && bun test test/scaffold.test.ts -t "pinned pi dep"` — Expected: FAIL
  - [ ] 01.2.c Minimal implementation: pin exact `@earendil-works/pi-coding-agent` version
  - [ ] 01.2.d Run to confirm pass — `Run: cd skill-pi && bun test test/scaffold.test.ts -t "pinned pi dep"` — Expected: PASS
  - [ ] 01.2.e Commit — `feat(skill-pi): pin pi-coding-agent dependency`
- [ ] 01.3 `[RED]` Unpinned / undeclared companion or plugin (THREAT: six deps present) — package identity fails if any of the six is missing
  - [ ] 01.3.a Write failing test: `pi-mcp-adapter`, `pi-web-access`, `pi-subagents`, `@juicesharp/rpiv-ask-user-question`, `@juicesharp/rpiv-todo`, `@juicesharp/rpiv-web-tools` all declared
  - [ ] 01.3.b Run to confirm fail — `Run: cd skill-pi && bun test test/scaffold.test.ts -t "companion deps present"` — Expected: FAIL
  - [ ] 01.3.c Minimal implementation: declare all six deps (pinned)
  - [ ] 01.3.d Run to confirm pass — `Run: cd skill-pi && bun test test/scaffold.test.ts -t "companion deps present"` — Expected: PASS
  - [ ] 01.3.e Commit — `feat(skill-pi): declare companion and plugin dependencies`
- [ ] 01.4 `[AFK]` Operator can npm install the local package and see bin `skill-pi` (WHAT) — `Run: cd skill-pi && bun test test/scaffold.test.ts -t "bin exposed"` — Expected: PASS
- [ ] 01.5 `[AFK]` Pi dependency missing or wrong version at runtime aborts with clear message (THREAT: shell/subprocess fail-closed on non-zero child) — `Run: cd skill-pi && bun test test/scaffold.test.ts -t "pi dep missing aborts"` — Expected: PASS

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `cd skill-pi && bun test test/scaffold.test.ts` | PASS | | |
| Acceptance `@step-01` / `@p0` | `cd skill-pi && bun test --integration -t "@step-01"` | PASS | | |
| Runtime harness | `cd skill-pi && bun x npm pack --dry-run` | PASS | | bin `skill-pi` only |
| Rollback boundary | remove `skill-pi/` package; no `pi` alias leaked | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(skill-pi): scaffold repo-root package with pinned Pi deps and skill-pi bin only`

---

