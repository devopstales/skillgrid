package main

import (
	"context"
	"encoding/json"
	"io"
	"regexp"
	"strconv"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// memoryBudgetCfg is the CLI's tunable read-budget flags (change 013, step 03).
// Zero fields fall back to the memory package defaults.
type memoryBudgetCfg struct {
	Items     int
	Chars     int
	TimeoutNs int64
}

// memoryBudget adapts the CLI budget flags to the memory package's BudgetConfig.
func memoryBudget(c memoryBudgetCfg) memory.BudgetConfig {
	return memory.BudgetConfig{Items: c.Items, Chars: c.Chars, TimeoutNs: c.TimeoutNs}
}

// memoryShareInput adapts the CLI share flags to the memory package's ShareInput.
type memoryShareInput = memory.ShareInput

// hCtx returns the read context for a CLI memory read (the budget's context
// timeout is enforced via Bound over this; the CLI uses Background).
func hCtx() context.Context {
	return context.Background()
}

// budgetDeadlineLapsed reports whether a deadline-bound read context has lapsed
// (the read ran up against the budget's context timeout and was cut). Mirrors
// the MCP handler's timeout check so the CLI honors --timeout.
func budgetDeadlineLapsed(ctx context.Context) bool {
	dl, ok := ctx.Deadline()
	return ok && !time.Now().Before(dl)
}

// budgetSessionsCLI applies the uniform read budget to a session list (CLI
// parity with the MCP mem_context handler): item cap + char budget (explicit
// "N chars omitted"). Returns the budgeted list plus the truncation flags.
func budgetSessionsCLI(b *memory.Budget, sessions []memory.Session) ([]memory.Session, bool, string) {
	capped := sessions
	if len(capped) > b.Config().Items {
		capped = capped[:b.Config().Items]
	}
	truncated := len(sessions) > b.Config().Items
	reason := ""
	if truncated {
		reason = "item-cap"
	}
	out := make([]memory.Session, len(capped))
	copy(out, capped)
	for i := range out {
		if len(out[i].Summary) > b.Config().Chars {
			out[i].Summary = memory.TruncateWithMarker(out[i].Summary, b.Config().Chars)
			truncated = true
			if reason == "" {
				reason = "char-budget"
			}
		}
	}
	return out, truncated, reason
}

func newJSONEncoder(w io.Writer) *json.Encoder {
	return json.NewEncoder(w)
}

var memWindowRe = regexp.MustCompile(`^(\d+)([smhd])$`)

// parseMemWindow parses a timeline window (e.g. "30m", "2h"); default 1h.
func parseMemWindow(s string) time.Duration {
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, "")
	if s == "" {
		return time.Hour
	}
	if m := memWindowRe.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		switch m[2] {
		case "s":
			return time.Duration(n) * time.Second
		case "m":
			return time.Duration(n) * time.Minute
		case "h":
			return time.Duration(n) * time.Hour
		case "d":
			return time.Duration(n) * 24 * time.Hour
		}
	}
	return time.Hour
}
