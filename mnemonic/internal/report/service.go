// Package report is the unified quality report: it collects scanner findings
// over the findings store (ADR-0030), one SourceResult per source that ran,
// and emits them as JUnit XML (CI) and JSON (dashboard). Fail-open
// (ADR-0016): a source that errors is present with Status "error", never a
// propagated error that kills the report.
package report

import (
	"context"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/scan"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// SourceResult is one source that ran (or failed) during Collect.
type SourceResult struct {
	Tool     string
	Status   string // "ok" | "error" | "unavailable"
	Findings []scan.Finding
}

// CoverageRow is a per-file line-coverage row (R9).
type CoverageRow struct {
	File    string
	Pkg     string
	PctLine float64
}

// Report is the aggregated quality report. Static sources always; Dynamic
// (DAST) only when dast.url is configured; Coverage is per-file line %.
type Report struct {
	Static      []SourceResult
	Dynamic     []SourceResult
	Coverage    []CoverageRow
	GeneratedAt time.Time
}

// Config is the report's downstream-project configuration, read from the
// project's .skillgrid/config.yaml. Zero values mean "not configured"
// (no DAST, no coverage floor).
type Config struct {
	DastURL     string
	CoverageMin float64
}

// Service is the report seam: Collect can be altered (add sources, change
// dispatch) without touching the emitters or the HTTP routes.
type Service struct {
	store   *store.Store
	root    string
	dataDir string
	cfg     Config
	run     scan.RunnerFunc // defaults to exec; tests stub
	probe   probeFunc       // defaults to the real HTTP HEAD probe; tests stub
}

// probeFunc probes whether a URL is reachable. A non-nil error means
// unreachable. Tests stub it; the default does an HTTP HEAD with a 3s timeout.
type probeFunc func(ctx context.Context, url string) error

// New creates a report service bound to a project store. root is the
// workspace root to scan; dataDir is the scan raw-output cache dir.
func New(st *store.Store, root, dataDir string, cfg Config) *Service {
	return &Service{store: st, root: root, dataDir: dataDir, cfg: cfg, run: scan.ExecRunner, probe: probeURL}
}

// SetRunner swaps the scanner runner (the real exec runner by default).
// Tests and the HTTP layer use it to stub scanner output.
func (s *Service) SetRunner(run scan.RunnerFunc) {
	if run != nil {
		s.run = run
	}
}
