package report

import (
	"context"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/scan"
)

// Collect gathers every configured source into a Report. Task 1 wires one
// real source (trivy) through the existing scan pipeline; later tasks add the
// rest of the static set, DAST, and coverage. Fail-open (ADR-0016): a source
// that errors is present with Status "error"; Collect itself never fails on a
// source error.
func (s *Service) Collect(ctx context.Context) (*Report, error) {
	rep := &Report{GeneratedAt: time.Now().UTC()}
	rep.Static = append(rep.Static, s.collectTrivy(ctx))
	return rep, nil
}

// collectTrivy runs the trivy tracer over the project: Start + StoreFindings
// through the real scan service, then reads the normalized findings back.
// Every failure path is fail-open: the source is present with Status "error".
func (s *Service) collectTrivy(ctx context.Context) SourceResult {
	scanSvc := scan.New(s.store, s.dataDir)
	scanSvc.Run = s.run
	sc, err := scanSvc.Start(ctx, "trivy", s.root)
	if err != nil {
		return SourceResult{Tool: "trivy", Status: "error"}
	}
	if sc.Status == "error" {
		return SourceResult{Tool: "trivy", Status: "error"}
	}
	if _, err := scanSvc.StoreFindings(ctx, sc.ID); err != nil {
		return SourceResult{Tool: "trivy", Status: "error"}
	}
	fs, err := scanSvc.FindingsByScan(ctx, sc.ID) // read back normalized findings
	if err != nil {
		return SourceResult{Tool: "trivy", Status: "error"}
	}
	return SourceResult{Tool: "trivy", Status: "ok", Findings: fs}
}
