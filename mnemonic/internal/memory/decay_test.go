package memory

import (
	"database/sql"
	"math"
	"testing"
	"time"
)

func TestReinforcementDecayRanksHotRow(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	seen := now.Add(-40 * 24 * time.Hour).Format(time.RFC3339)
	cold := Observation{ID: 1, Title: "cold", LastSeenAt: seen, RetrievalUsage: 0}
	hot := Observation{ID: 2, Title: "hot", LastSeenAt: seen, RetrievalUsage: 20}
	got := rankByDecay([]Observation{cold, hot}, now, DefaultDecayConfig())
	if len(got) != 2 || got[0].ID != 2 {
		t.Fatalf("hot row should rank first, got ids %d,%d", got[0].ID, got[1].ID)
	}
}

func TestDecayImmunityFreezesHalfLife(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	seen := now.Add(-400 * 24 * time.Hour).Format(time.RFC3339)
	o := Observation{
		LastSeenAt:      seen,
		RetrievalUsage:  0,
		ImportanceScore: sql.NullFloat64{Float64: 4, Valid: true},
	}
	got := Reinforcement(o, now, DefaultDecayConfig())
	want := 4 * math.Max(1, math.Log(1+0)) * 1
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("immune reinforcement = %v, want %v", got, want)
	}
}

func TestEqualDecayKeepsOrder(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	in := []Observation{{ID: 7, Title: "a"}, {ID: 3, Title: "b"}, {ID: 9, Title: "c"}}
	got := rankByDecay(in, now, DefaultDecayConfig())
	if len(got) != 3 || got[0].ID != 7 || got[1].ID != 3 || got[2].ID != 9 {
		t.Fatalf("equal factors must keep input order, got %d %d %d", got[0].ID, got[1].ID, got[2].ID)
	}
}
