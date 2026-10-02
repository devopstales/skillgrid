package memory

import (
	"database/sql"
	"testing"
)

func TestRankByUsePinnedStaysFirst(t *testing.T) {
	s := &Service{importanceCfg: defaultImportanceConfig()}
	rows := []Observation{
		{ID: 1, Pinned: false, ImportanceScore: sql.NullFloat64{Float64: 9, Valid: true}},
		{ID: 2, Pinned: true, ImportanceScore: sql.NullFloat64{Float64: 0.1, Valid: true}},
		{ID: 3, Pinned: false, ImportanceScore: sql.NullFloat64{Float64: 3, Valid: true}},
	}
	got := s.rankByUse(rows)
	if got[0].ID != 2 || got[1].ID != 1 || got[2].ID != 3 {
		t.Fatalf("order = %d %d %d", got[0].ID, got[1].ID, got[2].ID)
	}
}

func TestRankByUseZeroScoresKeepOrder(t *testing.T) {
	s := &Service{}
	rows := []Observation{{ID: 1}, {ID: 2}, {ID: 3}}
	got := s.rankByUse(rows)
	if got[0].ID != 1 || got[1].ID != 2 || got[2].ID != 3 {
		t.Fatalf("unstamped order moved: %d %d %d", got[0].ID, got[1].ID, got[2].ID)
	}
}
