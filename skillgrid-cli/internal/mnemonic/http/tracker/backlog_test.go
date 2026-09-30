package tracker

import (
	"testing"
)

// ─────────────────────── Change A: AC parsing ───────────────────────

func TestCountAC_AllChecked(t *testing.T) {
	body := `## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 All 7 tickets done
- [x] #2 go test passes
- [x] #3 something done
<!-- AC:END -->
`
	completed, total := countAC(body)
	if completed != 3 || total != 3 {
		t.Errorf("countAC = (%d, %d), want (3, 3)", completed, total)
	}
}

func TestCountAC_Mixed(t *testing.T) {
	body := `## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 done
- [ ] #2 pending
- [x] #3 done
- [ ] #4 pending
<!-- AC:END -->
`
	completed, total := countAC(body)
	if completed != 2 || total != 4 {
		t.Errorf("countAC = (%d, %d), want (2, 4)", completed, total)
	}
}

func TestCountAC_NoACSection(t *testing.T) {
	body := `## Definition of Done
- [ ] Review complete
- [ ] Tests pass

Some other text with a checkbox:
- [ ] not an AC
`
	completed, total := countAC(body)
	if completed != 0 || total != 0 {
		t.Errorf("countAC = (%d, %d), want (0, 0) — DOD lines must not count", completed, total)
	}
}

func TestCountAC_EmptyBlock(t *testing.T) {
	body := `## Acceptance Criteria
<!-- AC:BEGIN -->
<!-- AC:END -->
`
	completed, total := countAC(body)
	if completed != 0 || total != 0 {
		t.Errorf("countAC = (%d, %d), want (0, 0)", completed, total)
	}
}

func TestBacklogTaskFrom_SetsACFields(t *testing.T) {
	body := `## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 done
- [ ] #2 pending
<!-- AC:END -->
`
	fm := map[string]string{"id": "task-001", "title": "Test", "status": "in-progress"}
	task := backlogTaskFrom(fm, map[string][]string{}, body)
	if task.ACCompleted != 1 || task.ACTotal != 2 {
		t.Errorf("backlogTaskFrom AC = (%d, %d), want (1, 2)", task.ACCompleted, task.ACTotal)
	}
}
