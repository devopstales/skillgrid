### Task 5: Onboarding skill hands off to CLI

**Files:**
- Modify: `.agents/skills/lifecycle/onboarding/SKILL.md` (after Step 3 Write / before Step 4 Verify)
- Modify: `skillgrid-cli/cmd/skillgrid/init_cmd_test.go`

**Interfaces:**
- Consumes: `skillgrid init` from Tasks 1–4
- Produces: skill contract only
- Seam: none
- Deletion test: agents would invent ingest
- Adapters: 1

**SATISFIES:** happy path onboarding skill calls skillgrid init; skill passes extra docs through; skill does not contain a second ingest procedure

- [ ] **Step 1: Write the failing tests**

```go
func TestOnboardingSkillCallsSkillgridInit(t *testing.T) {
	body, err := os.ReadFile(onboardingSkillPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte("skillgrid init")) {
		t.Fatal("onboarding must name skillgrid init")
	}
}

func TestOnboardingSkillHasNoSecondIngest(t *testing.T) {
	body, err := os.ReadFile(onboardingSkillPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(body, []byte("walk docs/ and mem_save each file")) {
		t.Fatal("skill must not re-specify ingest")
	}
}

func onboardingSkillPath(t *testing.T) string {
	t.Helper()
	// repo root: this file is skillgrid-cli/cmd/skillgrid
	p := filepath.Join("..", "..", "..", ".agents", "skills", "lifecycle", "onboarding", "SKILL.md")
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
	return p
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./skillgrid-cli/cmd/skillgrid -count=1 -run 'TestOnboardingSkill'`
Expected: FAIL — `skillgrid init` absent

- [ ] **Step 3: Add one step to the skill**

After the Write steps, add:

```
**9. Project init (deterministic finish):**
- Run `skillgrid init` from the project root (add `--force` only if the user asked to rebuild the preamble).
- If the user named extra documentation paths, pass each as `--docs <path>`.
- Do not walk files or call `mem_save` for docs here — the CLI owns ingest and the code index.
- If `skillgrid` is not on PATH, tell the user to install the CLI and re-run `skillgrid init`; continue the rest of onboarding.
```

Do not add a file-walk ingest procedure.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./skillgrid-cli/cmd/skillgrid -count=1 -run 'TestOnboardingSkill|TestInit'`
Expected: PASS

- [ ] **Step 5: Commit**

```
docs(onboarding): finish with skillgrid init
```

## Self-review

| Spec requirement | Task |
|---|---|
| 1 Init command | Task 1 |
| 2 Boot file | Task 2 |
| 3 Forced code index | Task 3 |
| 4 Default ingest | Task 4 |
| 5 Extra `--docs` | Task 4 |
| 6 Skill handoff | Task 5 |

Must-haves map 1:1. No one-way doors. No placeholders. Types: `initResult`, `projectInit`, `writeBootFile`, `ingestPaths`, `runIndex` stay consistent.

## Owed-decision gate

Values and sources: flag names (spec), preamble headings (this blueprint), topic key prefix `init/docs/` (spec), observation types (spec), path jail (threat matrix), sentinel text (existing `block.md`), index via `RunCodeIndex` (existing API), Save via `memory.Save` (existing API). No owed decision.

## Plan review (inline)

1. **Failure modes:** Unreadable `--docs` file → `Errors`. Index boom → `runIndex` stub + `Errors`. Boot file on a non-dir → fatal. Concurrent init not in scope (local CLI). Empty project → skipped defaults, still writes AGENTS.
2. **Scope:** No Kanban, no KBs, no MCP tool. Matches briefing out-of-scope.
3. **Feasibility:** All APIs exist (`RunCodeIndex`, `OpenForDirectory`, `Save`, `SessionStart`).
4. **Ordering:** Command → boot → index → ingest → skill. Skill last so the verb exists.
5. **Tests:** Gates G1–G12 named in `acceptance.feature` match these `Test*` names.
6. **ADR:** Keep-store honored; no new table.

**Verdict:** READY FOR EXECUTION — after `2026-10-02-mnemonic-memory-checkpoint` ships or parks.

## Execution handoff

5 tasks → slicing is due when this change becomes current. Do not slice or execute while memory-checkpoint owns the pipeline.

When it is current: **1. Subagent-Driven (recommended)** or **2. Inline Execution**. Say which then.

## Ticket (tasks.md TICKET-05)
### TICKET-05 — Onboarding skill calls skillgrid init

- **Scope:** Onboarding skill runs `skillgrid init` (optional `--force` / `--docs`); no second ingest procedure.
- **Acceptance:** `TestOnboardingSkillCallsSkillgridInit` and `TestOnboardingSkillHasNoSecondIngest` PASS.
- **SATISFIES:** happy path onboarding skill calls skillgrid init; skill passes extra docs through; skill does not contain a second ingest procedure
- **Files:** `.agents/skills/lifecycle/onboarding/SKILL.md`, `init_cmd_test.go`
- **Size:** ~40 (S)
- **Blocks:** none
- **Blocked by:** TICKET-01
- **Reversibility:** reversible
- **Fails-when:** `go test ... -run TestOnboardingSkill` is non-zero

