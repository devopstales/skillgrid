package memory

import (
	"github.com/devopstales/skillgrid/mnemonic/internal/config"
)

// SetCLM stores the CLM config on the service (the Go-owns-state seam, ADR-0027).
// Plain assignment, consistent with the other config seams (SetBudget/SetTTL).
func (s *Service) SetCLM(cfg config.CLM) {
	if s != nil {
		s.clm = cfg
	}
}

// CLMConfig returns the current CLM config.
func (s *Service) CLMConfig() config.CLM {
	if s == nil {
		return config.CLM{}
	}
	return s.clm
}
