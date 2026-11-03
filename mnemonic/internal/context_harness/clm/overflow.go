package clm

import "github.com/devopstales/skillgrid/mnemonic/internal/config"

// OverflowDecision is the result of the overflow guard.
type OverflowDecision struct {
	Withhold    bool
	WithholdIDs []string
	Estimate    int
	Budget      int
}

// ComputeOverflow decides whether to withhold the oldest tool results when the
// calibrated estimate exceeds budget - reserve. Guard off => never withhold.
func ComputeOverflow(m *Mirror, cfg config.CLM, factor float64) OverflowDecision {
	if factor <= 0 {
		factor = 4.0
	}
	totalChars := 0
	for _, b := range m.Blocks {
		totalChars += len(b.Content)
	}
	estimate := int(float64(totalChars) / factor)
	d := OverflowDecision{Estimate: estimate, Budget: cfg.Budget}
	if !cfg.Guard || cfg.Budget == 0 {
		return d
	}
	// withhold when estimate + reserve > budget
	if estimate+cfg.Reserve <= cfg.Budget {
		return d
	}
	// withhold oldest tool blocks (lowest index, role "tool") first
	for _, b := range m.Blocks {
		if b.Role == "tool" {
			d.WithholdIDs = append(d.WithholdIDs, b.ID)
		}
	}
	d.Withhold = len(d.WithholdIDs) > 0
	return d
}
