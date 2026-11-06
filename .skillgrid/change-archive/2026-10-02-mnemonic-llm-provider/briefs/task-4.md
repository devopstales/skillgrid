### Task 4: Feature flags + fail-open

**Files:**
- Test: `skillgrid-cli/internal/mnemonic/service/attach_llm_failopen_test.go` (or extend attach tests)
- Prefer calling existing ask/dedup paths with error Completer

**SATISFIES:** `happy path attached client with flags off uses floors`, `happy path llm error fails open to floors`

- [ ] **Step 1: Write the failing test**

```go
type errCompleter struct{}

func (errCompleter) Complete(context.Context, string, string) (string, error) {
	return "", errors.New("down")
}

func TestAttachedClientFlagsOffUsesFloors(t *testing.T) {
	// Attach errCompleter; leave extraction.llm / dedup.llm false
	// Run CapturePassive / SaveWithAction hash path
	// Assert Completer never needed (n==0) and floor results OK
}

func TestLLMErrorFailsOpen(t *testing.T) {
	AttachSharedLLM(enabledCfg, errCompleter{}, mem, dream)
	// mem_ask mode=llm → citations present (degraded/cited floor)
	// SaveWithAction with hash-duplicate → noop floor
}
```

Reuse fixtures from `ask_test.go` / `dedup_llm_test.go` patterns.

- [ ] **Step 2: Run**

```bash
go test ./skillgrid-cli/internal/mnemonic/... -count=1 -run 'TestAttachedClientFlagsOffUsesFloors|TestLLMErrorFailsOpen'
```

Expected: FAIL or already PASS (document if existing floors hold)

- [ ] **Step 3: Fix** only if attach forces LLM paths on or swallows fail-open

- [ ] **Step 4: Run** — expect PASS

- [ ] **Step 5: Commit** `test(mnemonic): shared LLM flags off and fail-open`

---

