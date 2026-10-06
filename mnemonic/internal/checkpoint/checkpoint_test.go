package checkpoint

import (
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/config"
	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
)

func TestDecide(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	cfg := config.DefaultCheckpoint()

	tests := []struct {
		name       string
		st         memory.CheckpointState
		cfg        config.Checkpoint
		now        time.Time
		wantDue    bool
		wantReason string
	}{
		{
			name:       "disabled",
			st:         memory.CheckpointState{Exists: true, EventsSinceWrite: 10},
			cfg:        config.Checkpoint{Enabled: false, MinEvents: 5, Cooldown: 10 * time.Minute},
			now:        now,
			wantDue:    false,
			wantReason: ReasonDisabled,
		},
		{
			name:       "unknown_session",
			st:         memory.CheckpointState{Exists: false, EventsSinceWrite: 10},
			cfg:        cfg,
			now:        now,
			wantDue:    false,
			wantReason: ReasonUnknownSession,
		},
		{
			name:       "below_min_events",
			st:         memory.CheckpointState{Exists: true, EventsSinceWrite: 4},
			cfg:        cfg,
			now:        now,
			wantDue:    false,
			wantReason: ReasonBelowMinEvents,
		},
		{
			name: "cooldown",
			st: memory.CheckpointState{
				Exists:           true,
				EventsSinceWrite: 5,
				LastClaimedAt:    now.Add(-3 * time.Minute),
			},
			cfg:        cfg,
			now:        now,
			wantDue:    false,
			wantReason: ReasonCooldown,
		},
		{
			name: "due_never_claimed",
			st: memory.CheckpointState{
				Exists:           true,
				EventsSinceWrite: 5,
			},
			cfg:        cfg,
			now:        now,
			wantDue:    true,
			wantReason: "",
		},
		{
			name: "due_after_cooldown",
			st: memory.CheckpointState{
				Exists:           true,
				EventsSinceWrite: 7,
				LastClaimedAt:    now.Add(-11 * time.Minute),
			},
			cfg:        cfg,
			now:        now,
			wantDue:    true,
			wantReason: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotDue, gotReason := Decide(tc.st, tc.cfg, tc.now)
			if gotDue != tc.wantDue || gotReason != tc.wantReason {
				t.Fatalf("Decide() = (%v, %q), want (%v, %q)", gotDue, gotReason, tc.wantDue, tc.wantReason)
			}
		})
	}
}
