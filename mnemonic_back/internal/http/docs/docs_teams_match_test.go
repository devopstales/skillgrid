package docs

import (
	"testing"
	"time"
)

func TestMatchSession_PrefersTicketAndLeafWords(t *testing.T) {
	sessions := []MemberSession{
		{ID: "old", Title: "TICKET-06 mem_save MCP SaveWithAction additive res", StartedAt: "2026-10-02T07:16:03Z"},
		{ID: "hook", Title: "TICKET-06 fail-open compact hook", StartedAt: "2026-10-02T09:14:42Z"},
		{ID: "decay", Title: "TICKET-03 mnemonic.decay config", StartedAt: "2026-10-02T09:14:21Z"},
		{ID: "cache", Title: "Query embedding cache TICKET-04", StartedAt: "2026-10-02T09:14:30Z"},
	}
	cases := map[string]TeamMember{
		"hook":  {Agent: "compact-hook", Task: "TASK-030.03 TICKET-06"},
		"decay": {Agent: "decay-config", Task: "TASK-030.05 TICKET-03"},
		"cache": {Agent: "query-cache", Task: "TASK-030.02 TICKET-04"},
	}
	for want, m := range cases {
		got := MatchSession(m, sessions)
		if got == nil || got.ID != want {
			t.Errorf("%s: got %+v, want %s", m.Agent, got, want)
		}
	}
}

func TestMatchSession_NoTokenNoMatch(t *testing.T) {
	sessions := []MemberSession{{ID: "x", Title: "compact hook cleanup"}}
	if got := MatchSession(TeamMember{Agent: "compact-hook", Task: "Task 2"}, sessions); got != nil {
		t.Fatalf("expected no match without a ticket or task id, got %+v", got)
	}
}

func TestLiveStatus(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	if got := LiveStatus("ended", "2026-10-02T11:59:00Z", now); got != "finished" {
		t.Errorf("ended: got %s", got)
	}
	if got := LiveStatus("active", "2026-10-02T11:59:00Z", now); got != "working" {
		t.Errorf("recent: got %s", got)
	}
	if got := LiveStatus("active", "2026-10-02T09:00:00Z", now); got != "idle" {
		t.Errorf("quiet: got %s", got)
	}
}
