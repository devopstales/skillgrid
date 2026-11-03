package tiered

import (
	"strings"
	"unicode/utf8"
)

// Summarizer produces L0 abstracts and L1 overviews from full (L2) markdown.
type Summarizer interface {
	Abstract(full string) (string, error)
	Overview(full string) (string, error)
}

// HeuristicSummarizer is a Pure Go stub: first sentence / first paragraphs.
type HeuristicSummarizer struct{}

func (HeuristicSummarizer) Abstract(full string) (string, error) {
	text := strings.TrimSpace(full)
	if text == "" {
		return "", nil
	}
	// First non-empty line, truncated.
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			// Prefer heading body without hashes when present.
			if strings.HasPrefix(line, "#") {
				line = strings.TrimSpace(strings.TrimLeft(line, "#"))
				if line != "" {
					return truncateRunes(line, 160), nil
				}
			}
			continue
		}
		return truncateRunes(line, 160), nil
	}
	return truncateRunes(text, 160), nil
}

func (HeuristicSummarizer) Overview(full string) (string, error) {
	text := strings.TrimSpace(full)
	if text == "" {
		return "", nil
	}
	paras := strings.Split(text, "\n\n")
	var b strings.Builder
	n := 0
	for _, p := range paras {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(p)
		n++
		if n >= 3 || b.Len() >= 1200 {
			break
		}
	}
	return truncateRunes(b.String(), 2000), nil
}

// FailSummarizer always errors — used in tests for warn+continue paths.
type FailSummarizer struct {
	Err error
}

func (f FailSummarizer) Abstract(string) (string, error) { return "", f.Err }
func (f FailSummarizer) Overview(string) (string, error) { return "", f.Err }

func truncateRunes(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max]) + "…"
}
