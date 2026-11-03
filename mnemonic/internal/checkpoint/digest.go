package checkpoint

import (
	"fmt"
	"strings"
)

const maxTargetFieldLen = 120

// BuildDigest renders newest events first, capped at maxChars. Omitted older
// events are summarized with "(+N more)".
func BuildDigest(events []Event, maxChars int) string {
	if maxChars <= 0 || len(events) == 0 {
		return ""
	}

	lines := make([]string, 0, len(events))
	for i := len(events) - 1; i >= 0; i-- {
		lines = append(lines, formatEventLine(events[i]))
	}

	total := len(lines)
	joined := strings.Join(lines, "\n")
	if len(joined) <= maxChars {
		return joined
	}

	// Reserve space for "(+N more)" suffix; binary search how many newest lines fit.
	low, high := 0, total
	for low < high {
		mid := (low + high + 1) / 2
		omitted := total - mid
		suffix := fmt.Sprintf("(+%d more)", omitted)
		body := strings.Join(lines[:mid], "\n")
		candidate := body
		if omitted > 0 {
			if body != "" {
				candidate = body + "\n" + suffix
			} else {
				candidate = suffix
			}
		}
		if len(candidate) <= maxChars {
			low = mid
		} else {
			high = mid - 1
		}
	}

	omitted := total - low
	suffix := fmt.Sprintf("(+%d more)", omitted)
	if low == 0 {
		out := suffix
		if len(out) > maxChars {
			return out[:maxChars]
		}
		return out
	}
	body := strings.Join(lines[:low], "\n")
	out := body + "\n" + suffix
	if len(out) > maxChars {
		// Trim from the oldest retained line until the cap holds.
		for low > 0 && len(out) > maxChars {
			low--
			omitted = total - low
			suffix = fmt.Sprintf("(+%d more)", omitted)
			if low == 0 {
				out = suffix
				break
			}
			body = strings.Join(lines[:low], "\n")
			out = body + "\n" + suffix
		}
		if len(out) > maxChars {
			return out[:maxChars]
		}
	}
	return out
}

func formatEventLine(e Event) string {
	target := strings.TrimSpace(e.Path)
	if target == "" {
		target = strings.TrimSpace(e.Command)
	}
	target = sanitizePromptField(truncateRunes(target, maxTargetFieldLen))
	action := sanitizePromptField(e.Action)
	tool := sanitizePromptField(e.Tool)
	result := sanitizePromptField(e.Result)
	return fmt.Sprintf("#%d %s %s %s → %s", e.Sequence, action, tool, target, result)
}

func truncateRunes(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	return s[:max-1] + "…"
}
