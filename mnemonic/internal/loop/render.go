// Package loop is the session loop around mnemonic: prime a session with
// the last step and the diff, compact that step on close, and keep a
// gitignored view of the store. Agents are adapters. This package is the
// implementation they share.
package loop

import (
	"fmt"
	"strings"
)

// PrimeInput is the deterministic session-start block. Callers fill it
// from the store and git; RenderPrime does not touch the network or a model.
type PrimeInput struct {
	Project      string
	Task         string
	Branch       string
	StoppedAt    string
	OneLiner     string
	ChangedFiles []string
	ImpactLines  []string
	MemoryIndex  string
}

// RenderPrime formats the block injected at session start.
func RenderPrime(in PrimeInput) string {
	var b strings.Builder
	b.WriteString("Mnemonic prime.\n")
	if in.Project != "" {
		fmt.Fprintf(&b, "project: %s\n", in.Project)
	}
	fmt.Fprintf(&b, "last_task: %s\n", dash(in.Task))
	fmt.Fprintf(&b, "branch: %s\n", dash(in.Branch))
	fmt.Fprintf(&b, "stopped_at: %s\n", dash(in.StoppedAt))
	fmt.Fprintf(&b, "one_liner: %s\n", dash(in.OneLiner))
	b.WriteString("changed:\n")
	if len(in.ChangedFiles) == 0 {
		b.WriteString("- (none)\n")
	} else {
		for _, f := range in.ChangedFiles {
			fmt.Fprintf(&b, "- %s\n", f)
		}
	}
	if len(in.ImpactLines) > 0 {
		b.WriteString("impact:\n")
		for _, line := range in.ImpactLines {
			fmt.Fprintf(&b, "- %s\n", line)
		}
	}
	b.WriteString("Call code_explore before rg. rg is the escape hatch when code_explore is empty.\n")
	if idx := strings.TrimSpace(in.MemoryIndex); idx != "" {
		b.WriteString("\n## Memory\n")
		b.WriteString(idx)
		if !strings.HasSuffix(idx, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// CompactInput is the four-field resume handle written at session close.
type CompactInput struct {
	Task      string
	Branch    string
	StoppedAt string
	OneLiner  string
	DiffStat  string
}

// RenderCompact formats the summary stored on the session and printed to the hook.
func RenderCompact(in CompactInput) string {
	var b strings.Builder
	b.WriteString("## Goal\n")
	b.WriteString(dash(in.Task))
	b.WriteString("\n\n## Instructions\n")
	b.WriteString("Resume from this one-liner. Call code_explore before rg.\n\n")
	b.WriteString("## Discoveries\n")
	if strings.TrimSpace(in.DiffStat) == "" {
		b.WriteString("(none)\n\n")
	} else {
		b.WriteString(strings.TrimSpace(in.DiffStat))
		b.WriteString("\n\n")
	}
	b.WriteString("## Accomplished\n")
	b.WriteString(dash(in.OneLiner))
	b.WriteString("\n\n## Next Steps\n")
	b.WriteString("Continue the in-progress task on this branch.\n\n")
	b.WriteString("## Relevant Files\n")
	fmt.Fprintf(&b, "branch: %s\nstopped_at: %s\n", dash(in.Branch), dash(in.StoppedAt))
	return b.String()
}

// OneLinerFromSummary pulls a single line out of a session summary.
// The Goal section wins; otherwise the first non-empty line.
func OneLinerFromSummary(summary string) string {
	text := strings.TrimSpace(summary)
	if text == "" {
		return ""
	}
	lower := strings.ToLower(text)
	if i := strings.Index(lower, "## goal"); i >= 0 {
		rest := text[i+len("## goal"):]
		rest = strings.TrimLeft(rest, "\r\n \t")
		if j := strings.Index(rest, "\n##"); j >= 0 {
			rest = rest[:j]
		}
		line := firstLine(rest)
		if line != "" {
			return line
		}
	}
	return firstLine(text)
}

func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			return line
		}
	}
	return ""
}

func dash(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	return s
}
