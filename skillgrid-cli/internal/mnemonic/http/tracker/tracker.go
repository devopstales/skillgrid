// Package tracker exposes the repo's active issue tracker (Backlog.md,
// GitHub, GitLab, or Jira) over HTTP by shelling out to its CLI.
// The provider is resolved, never guessed: SKILLGRID_TRACKER override,
// then docs/skillgrid/agents/issue-tracker.md, then config.yaml,
// defaulting to backlogmd.
package tracker

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// Provider identifiers.
const (
	ProviderBacklogMD = "backlogmd"
	ProviderGitHub    = "github"
	ProviderGitLab    = "gitlab"
	ProviderJira      = "jira"
)

// Item is the provider-independent task DTO served on /tracker/*.
type Item struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Status       string   `json:"status"`
	StatusDetail string   `json:"status_detail,omitempty"`
	Description  string   `json:"description,omitempty"`
	Type         string   `json:"type,omitempty"`
	Priority     string   `json:"priority,omitempty"`
	Assignees    []string `json:"assignees,omitempty"`
	Labels       []string `json:"labels,omitempty"`
	DocRefs      []string `json:"doc_refs,omitempty"`
	// Board is the canonical kanban column (todo|in_progress|blocked|done)
	// mapped from the provider-native status so every provider renders one
	// consistent board (demo: STATUS_COLUMNS).
	Board       string `json:"board"`
	ACCompleted int    `json:"ac_completed,omitempty"`
	ACTotal     int    `json:"ac_total,omitempty"`
	IsReady     *bool  `json:"is_ready,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	Provider    string `json:"provider"`
}

// boardFor maps a provider-native status to its canonical column.
func boardFor(provider, status string) string {
	s := strings.ToLower(strings.TrimSpace(status))
	switch provider {
	case ProviderGitHub:
		if s == "closed" {
			return "done"
		}
		return "todo"
	case ProviderGitLab:
		if s == "closed" {
			return "done"
		}
		return "todo"
	case ProviderJira:
		switch {
		case strings.Contains(s, "progress") || strings.Contains(s, "review") || strings.Contains(s, "doing"):
			return "in_progress"
		case strings.Contains(s, "done") || strings.Contains(s, "closed") || strings.Contains(s, "resolved") || strings.Contains(s, "complete"):
			return "done"
		case strings.Contains(s, "block"):
			return "blocked"
		default:
			return "todo"
		}
	default: // backlogmd + unknown: triage-ish → todo, active → in_progress
		switch s {
		case "blocked":
			return "blocked"
		case "done", "wontfix":
			return "done"
		case "ready-for-agent", "ready-for-human", "in-progress", "in_progress":
			return "in_progress"
		default:
			return "todo"
		}
	}
}

// TrackerConfig is served on GET /tracker/config.
type TrackerConfig struct {
	Provider   string   `json:"provider"`
	Statuses   []string `json:"statuses"`
	Types      []string `json:"types,omitempty"`
	Priorities []string `json:"priorities,omitempty"`
	Version    string   `json:"version,omitempty"`
	Schema     string   `json:"schema_version,omitempty"`
	Repo       string   `json:"repo,omitempty"`
	ProjectKey string   `json:"project_key,omitempty"`
}

// Typed bridge errors; handlers map them to HTTP statuses.
type errCLIMissing struct{ CLI, Provider string }

func (e *errCLIMissing) Error() string { return fmt.Sprintf("%s CLI not found", e.CLI) }

type errCLIFailed struct {
	CLI, Provider, Stderr, Output string
	Timeout, Auth                 bool
}

func (e *errCLIFailed) Error() string {
	if e.Timeout {
		return fmt.Sprintf("%s CLI timed out", e.CLI)
	}
	return fmt.Sprintf("%s CLI failed: %s", e.CLI, e.Stderr)
}

type errBadOutput struct{ CLI, Provider, Reason string }

func (e *errBadOutput) Error() string {
	return fmt.Sprintf("%s CLI output invalid: %s", e.CLI, e.Reason)
}

type errNotFound struct{ ID string }

func (e *errNotFound) Error() string { return fmt.Sprintf("item %s not found", e.ID) }

type errBadStatus struct {
	Status string
	Valid  []string
}

func (e *errBadStatus) Error() string {
	return fmt.Sprintf("invalid status %q (valid: %s)", e.Status, strings.Join(e.Valid, ", "))
}

type errUnsupported struct{ Reason string }

func (e *errUnsupported) Error() string { return e.Reason }

type errUnresolvable struct{ Reason string }

func (e *errUnresolvable) Error() string { return e.Reason }

// Provider is one tracker backend.
type Provider interface {
	// Name returns backlogmd | github | gitlab | jira.
	Name() string
	// CLIName returns the binary shelled out to.
	CLIName() string
	Config(ctx context.Context) (TrackerConfig, error)
	List(ctx context.Context) ([]Item, error)
	Get(ctx context.Context, id string) (Item, error)
	SetStatus(ctx context.Context, id, status string) (Item, error)
}

// ResolveProvider maps override/tracker-doc/config inputs to a provider.
// Empty inputs fall through; unknown values are errors (never guessed).
func ResolveProvider(envOverride, trackerDoc, configValue string) (string, error) {
	if v := strings.TrimSpace(envOverride); v != "" {
		n := normalizeProvider(v)
		if n == "" {
			return "", &errUnresolvable{Reason: fmt.Sprintf("unknown SKILLGRID_TRACKER %q (want backlogmd|github|gitlab|jira)", v)}
		}
		return n, nil
	}
	if v := providerFromTrackerDoc(trackerDoc); v != "" {
		return v, nil
	}
	if v := normalizeProvider(configValue); v != "" {
		return v, nil
	}
	return ProviderBacklogMD, nil
}

func normalizeProvider(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "backlogmd", "backlog", "backlog.md":
		return ProviderBacklogMD
	case "github", "gh":
		return ProviderGitHub
	case "gitlab", "glab":
		return ProviderGitLab
	case "jira":
		return ProviderJira
	}
	return ""
}

// providerFromTrackerDoc reads "# Issue tracker: X" from the onboarded doc.
func providerFromTrackerDoc(doc string) string {
	for _, line := range strings.Split(doc, "\n") {
		l := strings.TrimSpace(strings.ToLower(line))
		if !strings.HasPrefix(l, "# issue tracker:") {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(l, "# issue tracker:"))
		if n := normalizeProvider(name); n != "" {
			return n
		}
		return ""
	}
	return ""
}

// commandTimeout is 10s; SKILLGRID_TRACKER_TIMEOUT_SECS overrides (tests).
func commandTimeout() time.Duration {
	if v := strings.TrimSpace(os.Getenv("SKILLGRID_TRACKER_TIMEOUT_SECS")); v != "" {
		var secs int
		if _, err := fmt.Sscanf(v, "%d", &secs); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return 10 * time.Second
}

// runCLI executes one CLI call with LookPath + timeout semantics.
// Timeout kills the whole process GROUP (fixture scripts spawn children like
// `sleep` that would otherwise hold the stdout pipe open past the deadline).
func runCLI(ctx context.Context, cli string, args ...string) (stdout string, runErr error) {
	if _, err := exec.LookPath(cli); err != nil {
		return "", &errCLIMissing{CLI: cli}
	}
	timeout := commandTimeout()
	cmd := exec.Command(cli, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Start(); err != nil {
		return "", &errCLIFailed{CLI: cli, Stderr: truncate200(err.Error())}
	}
	timedOut := false
	timer := time.AfterFunc(timeout, func() {
		timedOut = true
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	})
	err := cmd.Wait()
	timer.Stop()
	stderr := truncate200(errBuf.String())
	output := truncate200(outBuf.String())
	if timedOut || ctx.Err() == context.DeadlineExceeded {
		return "", &errCLIFailed{CLI: cli, Stderr: stderr, Output: output, Timeout: true}
	}
	if err != nil {
		lower := strings.ToLower(stderr + " " + outBuf.String())
		auth := strings.Contains(lower, "auth") || strings.Contains(lower, "login") ||
			strings.Contains(lower, "token") || strings.Contains(lower, "unauthorized") ||
			strings.Contains(lower, "401") || strings.Contains(lower, "403")
		return "", &errCLIFailed{CLI: cli, Stderr: stderr, Output: output, Auth: auth}
	}
	return outBuf.String(), nil
}

// failedOutput returns the stderr+stdout excerpts from a CLI failure (for
// not-found sniffing — CLIs report misses on either stream).
func failedOutput(err error) string {
	var fe *errCLIFailed
	if errors.As(err, &fe) {
		return fe.Stderr + "\n" + fe.Output
	}
	return ""
}

func truncate200(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 200 {
		return s[:200]
	}
	if s == "" {
		return "exit status 1"
	}
	return s
}

// ForName resolves inputs to a live Provider (used by the HTTP layer).
func ForName(envOverride, trackerDoc, configValue string) (Provider, error) {
	name, err := ResolveProvider(envOverride, trackerDoc, configValue)
	if err != nil {
		return nil, err
	}
	switch name {
	case ProviderGitHub:
		return &githubAdapter{}, nil
	case ProviderGitLab:
		return &gitlabAdapter{}, nil
	case ProviderJira:
		return &jiraAdapter{}, nil
	default:
		return &backlogAdapter{}, nil
	}
}

// StatusForHTTP maps bridge errors to HTTP statuses (exported for handlers).
func StatusForHTTP(err error) int { return statusForHTTP(err) }

func statusForHTTP(err error) int {
	switch err.(type) {
	case *errCLIMissing:
		return http.StatusServiceUnavailable
	case *errCLIFailed, *errBadOutput:
		return http.StatusBadGateway
	case *errNotFound:
		return http.StatusNotFound
	case *errBadStatus:
		return http.StatusBadRequest
	case *errUnsupported, *errUnresolvable:
		return http.StatusNotImplemented
	}
	return http.StatusInternalServerError
}
