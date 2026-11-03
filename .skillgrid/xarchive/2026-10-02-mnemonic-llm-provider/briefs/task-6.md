### Task 6: Close task-029

**Files:**
- Modify: `.backlog/tasks/task-029 - FOLLOWUP-shared-LLM-client-attach-for-ask-extraction-dedup-seams.md` — status Done when code ships

**SATISFIES:** `happy path task-029 superseded by this change`

- [ ] **Step 1:** Confirm G9:

```bash
rg -n "2026-10-02-mnemonic-llm-provider" .backlog/tasks/task-029*
```

Expected: match (already true)

- [ ] **Step 2:** After TICKET-02/03 land, set status Done and check DoD boxes

- [ ] **Step 5: Commit** (with ship, not early)

```bash
git add .backlog/tasks/task-029*
git commit -m "$(cat <<'EOF'
docs(backlog): close task-029 via mnemonic-llm-provider

[skillgrid-context]
task: TASK-029
EOF
)"
```

## Self-Review

| Check | Status |
|-------|--------|
| Spec requirements 1–7 → tasks | yes (1–2 config/client, 3 attach, 4 fail-open, 5 install, 6 tracker) |
| No placeholder steps | code blocks in Tasks 1–5 |
| ADR-0016 / ADR-0023 cited | yes |
| Door check named | Task 1 + 3 |
| Serial queue honored | Queued behind project-init |
| Threat matrix RED tests named | yes |
