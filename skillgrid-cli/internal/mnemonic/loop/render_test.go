package loop

import "testing"

func TestRenderPrimeStable(t *testing.T) {
	in := PrimeInput{
		Project:      "skillgrid",
		Task:         "mnemonic loop",
		Branch:       "release/2",
		StoppedAt:    "2026-10-02T00:00:00Z",
		OneLiner:     "wire prime",
		ChangedFiles: []string{"a.go", "b.go"},
		ImpactLines:  []string{"a.go Save callers: 2"},
	}
	got := RenderPrime(in)
	again := RenderPrime(in)
	if got != again {
		t.Fatal("prime block must be byte-stable")
	}
	for _, want := range []string{"project: skillgrid", "last_task: mnemonic loop", "- a.go", "code_explore before rg"} {
		if !contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
}

func TestRenderPrimeEmptyDiff(t *testing.T) {
	got := RenderPrime(PrimeInput{})
	if !contains(got, "changed:\n- (none)") {
		t.Fatalf("empty diff:\n%s", got)
	}
	if !contains(got, "last_task: -") {
		t.Fatalf("blank task:\n%s", got)
	}
}

func TestOneLinerFromSummary(t *testing.T) {
	summary := "## Goal\nwire the hooks\n\n## Next Steps\nlater\n"
	if got := OneLinerFromSummary(summary); got != "wire the hooks" {
		t.Fatalf("one liner %q", got)
	}
	if got := OneLinerFromSummary(""); got != "" {
		t.Fatalf("empty %q", got)
	}
}

func TestRenderCompactHasSixSections(t *testing.T) {
	got := RenderCompact(CompactInput{Task: "loop", Branch: "main", StoppedAt: "t", OneLiner: "done", DiffStat: "a.go | 1 +"})
	for _, h := range []string{"## Goal", "## Instructions", "## Discoveries", "## Accomplished", "## Next Steps", "## Relevant Files"} {
		if !contains(got, h) {
			t.Errorf("missing %s", h)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
