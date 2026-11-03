package report

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/scan"
)

// staticScanTools are the scanner-backed static sources, run through the
// shared scan pipeline (Start → StoreFindings → read back).
var staticScanTools = []string{"trivy", "semgrep", "jscpd"}

// Collect gathers every configured source into a Report. The static set runs
// always: the scanner-backed tools, the debt marker grep, and the detected-
// dispatch coverage source. The DAST section runs only when dast.url is
// configured. Fail-open (ADR-0016): a source that errors is present with
// Status "error"; Collect itself never fails on a source error.
func (s *Service) Collect(ctx context.Context) (*Report, error) {
	rep := &Report{GeneratedAt: time.Now().UTC()}
	for _, tool := range staticScanTools {
		rep.Static = append(rep.Static, s.collectScanTool(ctx, tool, s.root))
	}
	rep.Static = append(rep.Static, s.collectDebt(ctx))
	covRes, covRows := s.collectCoverage(ctx)
	rep.Static = append(rep.Static, covRes)
	rep.Coverage = covRows
	rep.Dynamic = s.collectDAST(ctx)
	return rep, nil
}

// dastTools are the dynamic (DAST) sources, run against cfg.DastURL when it is
// set. wapiti/nuclei/akca all have scannerCommands + Parse arms.
var dastTools = []string{"wapiti", "nuclei", "akca"}

// collectDAST runs the DAST sources against cfg.DastURL. Empty DastURL → nil
// (no DAST section). Unreachable target → all three tools present with Status
// "unavailable" and no scanner run. Reachable → each tool runs through the
// scan pipeline; a per-tool failure is that tool's "error", the others
// continue (fail-open, ADR-0016).
func (s *Service) collectDAST(ctx context.Context) []SourceResult {
	if s.cfg.DastURL == "" {
		return nil
	}
	url := s.cfg.DastURL
	if err := s.probe(ctx, url); err != nil {
		res := make([]SourceResult, 0, len(dastTools))
		for _, tool := range dastTools {
			res = append(res, SourceResult{Tool: tool, Status: "unavailable"})
		}
		return res
	}
	res := make([]SourceResult, 0, len(dastTools))
	for _, tool := range dastTools {
		res = append(res, s.collectScanTool(ctx, tool, url))
	}
	return res
}

// collectScanTool runs one scanner (static or DAST tool) against a target and
// reads its normalized findings back: Start + StoreFindings through the real
// scan service, then FindingsByScan read-back. Fail-open (ADR-0016): every
// error path is Status "error" with no returned error.
func (s *Service) collectScanTool(ctx context.Context, tool, target string) SourceResult {
	scanSvc := scan.New(s.store, s.dataDir)
	scanSvc.Run = s.run
	sc, err := scanSvc.Start(ctx, tool, target)
	if err != nil {
		return SourceResult{Tool: tool, Status: "error"}
	}
	if sc.Status == "error" {
		return SourceResult{Tool: tool, Status: "error"}
	}
	if _, err := scanSvc.StoreFindings(ctx, sc.ID); err != nil {
		return SourceResult{Tool: tool, Status: "error"}
	}
	fs, err := scanSvc.FindingsByScan(ctx, sc.ID) // read back normalized findings
	if err != nil {
		return SourceResult{Tool: tool, Status: "error"}
	}
	return SourceResult{Tool: tool, Status: "ok", Findings: fs}
}

// probeURL is the default reachability probe: an HTTP HEAD with a 3s timeout.
// Any failure (timeout, connection refused, non-2xx/3xx) counts as unreachable
// — a DAST target that won't answer HEAD is not worth scanning.
func probeURL(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return fmt.Errorf("probe %s: %w", url, err)
	}
	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	client := &http.Client{}
	resp, err := client.Do(req.WithContext(probeCtx))
	if err != nil {
		return fmt.Errorf("probe %s: %w", url, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("probe %s: status %d", url, resp.StatusCode)
	}
	return nil
}
