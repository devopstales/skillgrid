package checkpoint

import (
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/config"
	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
)

// Event is a structured tool event for digest rendering (no tool output text).
type Event struct {
	Sequence int
	Action   string // action_type
	Tool     string
	Path     string
	Command  string
	Result   string
	At       string
}

const (
	ReasonBelowMinEvents = "below_min_events"
	ReasonCooldown       = "cooldown"
	ReasonDisabled       = "disabled"
	ReasonUnknownSession = "unknown_session"
)

// Decide reports whether a memory checkpoint is due for the session.
func Decide(st memory.CheckpointState, cfg config.Checkpoint, now time.Time) (due bool, reason string) {
	if !cfg.Enabled {
		return false, ReasonDisabled
	}
	if !st.Exists {
		return false, ReasonUnknownSession
	}
	if st.EventsSinceWrite < cfg.MinEvents {
		return false, ReasonBelowMinEvents
	}
	if !st.LastClaimedAt.IsZero() {
		if now.Sub(st.LastClaimedAt) < cfg.Cooldown {
			return false, ReasonCooldown
		}
	}
	return true, ""
}
