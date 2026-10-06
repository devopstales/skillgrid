package tracker

import (
	"context"
	"os"
	"path/filepath"
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

// ─────────────────────── Milestones ───────────────────────

func TestBacklogMilestones_ReadsFrontmatter(t *testing.T) {
	// tasksDir = <root>/tasks, so milestonesDir = <root>/milestones
	root := t.TempDir()
	tasksDir := filepath.Join(root, "tasks")
	msDir := filepath.Join(root, "milestones")
	os.MkdirAll(tasksDir, 0o755)
	os.MkdirAll(msDir, 0o755)
	os.WriteFile(filepath.Join(msDir, "m-0.md"), []byte(`---
id: m-0
title: session-events-layer
description: Session events layer
---
Milestone body.
`), 0o644)
	os.WriteFile(filepath.Join(msDir, "m-1.md"), []byte(`---
id: m-1
title: local-ollama-models
---
Milestone body.
`), 0o644)
	// Non-md file should be skipped
	os.WriteFile(filepath.Join(msDir, "notes.txt"), []byte("not a milestone"), 0o644)
	// Missing id should be skipped
	os.WriteFile(filepath.Join(msDir, "no-id.md"), []byte(`---
title: no id
---
body
`), 0o644)

	a := &backlogAdapter{tasksDir: tasksDir}
	ms, err := a.Milestones(context.Background())
	if err != nil {
		t.Fatalf("Milestones: %v", err)
	}
	if len(ms) != 2 {
		t.Fatalf("expected 2 milestones, got %d: %+v", len(ms), ms)
	}
	if ms[0].ID != "m-0" || ms[0].Title != "session-events-layer" {
		t.Errorf("m-0 wrong: %+v", ms[0])
	}
	if ms[0].Description != "Session events layer" {
		t.Errorf("m-0 description wrong: %q", ms[0].Description)
	}
	if ms[1].ID != "m-1" || ms[1].Title != "local-ollama-models" {
		t.Errorf("m-1 wrong: %+v", ms[1])
	}
}

func TestBacklogMilestones_MissingDir(t *testing.T) {
	root := t.TempDir()
	tasksDir := filepath.Join(root, "tasks")
	os.MkdirAll(tasksDir, 0o755)
	// No milestones dir → empty slice, no error
	a := &backlogAdapter{tasksDir: tasksDir}
	ms, err := a.Milestones(context.Background())
	if err != nil {
		t.Fatalf("Milestones: %v", err)
	}
	if len(ms) != 0 {
		t.Errorf("expected 0 milestones, got %d", len(ms))
	}
}

func TestBacklogMilestones_SortedByID(t *testing.T) {
	root := t.TempDir()
	tasksDir := filepath.Join(root, "tasks")
	msDir := filepath.Join(root, "milestones")
	os.MkdirAll(tasksDir, 0o755)
	os.MkdirAll(msDir, 0o755)
	// Write in non-sorted order
	os.WriteFile(filepath.Join(msDir, "z.md"), []byte("---\nid: z\ntitle: Z\n---\n"), 0o644)
	os.WriteFile(filepath.Join(msDir, "a.md"), []byte("---\nid: a\ntitle: A\n---\n"), 0o644)
	os.WriteFile(filepath.Join(msDir, "m.md"), []byte("---\nid: m\ntitle: M\n---\n"), 0o644)

	a := &backlogAdapter{tasksDir: tasksDir}
	ms, err := a.Milestones(context.Background())
	if err != nil {
		t.Fatalf("Milestones: %v", err)
	}
	if len(ms) != 3 {
		t.Fatalf("expected 3, got %d", len(ms))
	}
	if ms[0].ID != "a" || ms[1].ID != "m" || ms[2].ID != "z" {
		t.Errorf("not sorted by id: %+v", ms)
	}
}
