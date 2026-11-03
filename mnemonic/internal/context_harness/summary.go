package context_harness

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
)

// minEventsForSummary is the floor of events a session needs before a summary
// is worth producing. Tiny sessions have no meaningful summary.
const minEventsForSummary = 10

// EstimateTokens is the token-cost heuristic: 1 token ≈ 4 characters. It is an
// estimate, not an exact count.
func EstimateTokens(s string) int {
	if len(s) == 0 {
		return 0
	}
	return (len(s) + 3) / 4
}

// DistillSummary produces the L1 summary for a session: a deterministic,
// privacy-filtered, token-capped markdown block built from the session's
// events plus the project's recent observations. It returns "" (no error) for
// sessions with fewer than minEventsForSummary events.
func DistillSummary(ctx context.Context, svc *memory.Service, sessionID string, maxTokens int) (string, error) {
	if maxTokens <= 0 {
		maxTokens = 800
	}
	events, from, to, err := svc.SessionChanges(ctx, sessionID)
	if err != nil {
		return "", err
	}
	if len(events) < minEventsForSummary {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString("## Prior Session Summary\n\n")
	sb.WriteString(fmt.Sprintf("**Commit range:** `%s` → `%s`\n\n", shortSHA(from), shortSHA(to)))

	byType := map[string][]memory.Event{}
	for _, e := range events {
		if !Injectable(e) {
			continue
		}
		byType[e.ActionType] = append(byType[e.ActionType], e)
	}

	types := make([]string, 0, len(byType))
	for t := range byType {
		types = append(types, t)
	}
	sort.Strings(types)

	sb.WriteString("### Activity\n\n")
	for _, t := range types {
		evs := byType[t]
		sb.WriteString(fmt.Sprintf("- **%s** (%d events)\n", t, len(evs)))
		start := len(evs) - 3
		if start < 0 {
			start = 0
		}
		for _, e := range evs[start:] {
			sb.WriteString(fmt.Sprintf("  - `%s` %s\n", e.Timestamp[:10], renderEvent(e)))
		}
	}

	if obs, err := svc.RecentObservations(ctx, 5); err == nil {
		var injectable []memory.Observation
		for _, o := range obs {
			if ObservationInjectable(o) {
				injectable = append(injectable, o)
			}
		}
		if len(injectable) > 0 {
			sort.Slice(injectable, func(i, j int) bool { return injectable[i].ID < injectable[j].ID })
			sb.WriteString("\n### Key Observations\n\n")
			for _, o := range injectable {
				snippet := redactFullPaths(o.Content)
				if len(snippet) > 200 {
					snippet = snippet[:200] + "…"
				}
				sb.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", o.Title, o.Type, snippet))
			}
		}
	}

	result := sb.String()
	if EstimateTokens(result) > maxTokens {
		result = truncateToTokens(result, maxTokens)
	}
	return result, nil
}

func renderEvent(e memory.Event) string {
	var parts []string
	if e.ToolName != "" {
		parts = append(parts, e.ToolName)
	}
	if e.Path != "" {
		parts = append(parts, e.Path)
	}
	if e.Command != "" {
		cmd := e.Command
		if len(cmd) > 60 {
			cmd = cmd[:60] + "…"
		}
		parts = append(parts, cmd)
	}
	if e.Commit != "" {
		parts = append(parts, shortSHA(e.Commit))
	}
	return redactFullPaths(strings.Join(parts, " "))
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	if sha == "" {
		return "none"
	}
	return sha
}

func truncateToTokens(s string, maxTokens int) string {
	maxChars := maxTokens * 4
	if len(s) <= maxChars {
		return s
	}
	omitted := EstimateTokens(s[maxChars:])
	suffix := fmt.Sprintf("\n… (%d tokens truncated)", omitted)
	budget := maxTokens - EstimateTokens(suffix)
	if budget < 0 {
		budget = 0
	}
	cut := budget * 4
	if cut > len(s) {
		cut = len(s)
	}
	if budget <= 0 {
		return s[:cut]
	}
	return s[:cut] + suffix
}
