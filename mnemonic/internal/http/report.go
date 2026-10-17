package http

import (
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/devopstales/skillgrid/mnemonic/internal/report"
	"github.com/devopstales/skillgrid/mnemonic/internal/report/emit"
	"github.com/devopstales/skillgrid/mnemonic/internal/scan"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// reportRunOverride replaces the report collector's scanner runner. Nil in
// prod (the real exec runner); tests stub it to serve fixture scanner output.
// Each GET /report constructs a fresh report.Service, so re-arming it per
// request is safe.
var reportRunOverride scan.RunnerFunc

// handleReport serves GET /report/{fmt} (fmt = "junit.xml" | "json"):
// collect the report over the project's findings store and emit it. The route
// is write-auth wrapped because Collect launches scanner subprocesses.
func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	s.withProjectHandle(w, r, func(h *service.ProjectHandle, _ string) {
		fmtName := r.PathValue("fmt")
		root := h.Root()
		dataDir := h.DataDir()
		if dataDir == "" {
			// Handles without a backing service have no data dir; fall back to
			// the service-level one so the raw scan cache lands in a real dir.
			dataDir = "."
		}
		svc := report.New(h.Store(), root, dataDir, reportConfigFrom(h))
		if reportRunOverride != nil {
			svc.SetRunner(reportRunOverride)
		}
		rep, err := svc.Collect(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		var (
			out     []byte
			content string
			e       error
		)
		switch fmtName {
		case "junit.xml":
			out, e = emit.JUnit(rep)
			content = "application/xml"
		case "json":
			out, e = emit.JSON(rep)
			content = "application/json"
		default:
			writeError(w, http.StatusBadRequest, `fmt must be "junit.xml" or "json"`)
			return
		}
		if e != nil {
			writeError(w, http.StatusInternalServerError, e.Error())
			return
		}
		w.Header().Set("Content-Type", content)
		_, _ = w.Write(out)
	})
}

// reportConfigFrom reads the downstream project's .skillgrid/config.yaml.
// The report needs only dast.url (and the optional coverage threshold). A
// missing file or key yields a zero Config (no DAST), fail-open.
func reportConfigFrom(h *service.ProjectHandle) report.Config {
	path := filepath.Join(h.Root(), ".skillgrid", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return report.Config{}
	}
	var c struct {
		Dast struct {
			URL string `yaml:"url"`
		} `yaml:"dast"`
		Quality struct {
			CoverageMin int `yaml:"coverage_min"`
		} `yaml:"quality"`
	}
	if err := yaml.Unmarshal(data, &c); err != nil {
		return report.Config{}
	}
	return report.Config{DastURL: c.Dast.URL, CoverageMin: float64(c.Quality.CoverageMin)}
}
