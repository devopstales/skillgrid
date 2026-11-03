package context_harness

import (
	"fmt"
	"sort"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
)

// IndexConfig limits the session-start Memory Index section.
type IndexConfig struct {
	Summaries    int
	Observations int
	MaxTokens    int
}

// RenderIndex builds the plain-text Memory Index body (no ## Memory header).
// It returns "" when both inputs are empty after trimming limits.
func RenderIndex(summaries []memory.Session, observations []memory.Observation, cfg IndexConfig) string {
	maxTok := cfg.MaxTokens
	if maxTok <= 0 {
		maxTok = 800
	}

	sumLimit := cfg.Summaries
	if sumLimit < 0 {
		sumLimit = 0
	}
	obsLimit := cfg.Observations
	if obsLimit < 0 {
		obsLimit = 0
	}

	var sumLines []string
	for i, s := range summaries {
		if i >= sumLimit {
			break
		}
		if line := formatSummaryLine(s); line != "" {
			sumLines = append(sumLines, line)
		}
	}

	obs := orderObservations(observations)
	if obsLimit > 0 && len(obs) > obsLimit {
		obs = obs[:obsLimit]
	}
	var obsEntries []obsEntry
	for _, o := range obs {
		obsEntries = append(obsEntries, obsEntry{id: o.ID, line: formatObservationLine(o)})
	}

	if len(sumLines) == 0 && len(obsEntries) == 0 {
		return ""
	}

	omittedObs := 0
	omittedSum := 0
	for {
		footerID := int64(0)
		if len(obsEntries) > 0 {
			footerID = obsEntries[0].id
		}
		body := joinIndexBody(sumLines, obsEntries, omittedObs, omittedSum, footerID)
		if EstimateTokens(body) <= maxTok {
			return body
		}
		if len(obsEntries) > 0 {
			obsEntries = obsEntries[:len(obsEntries)-1]
			omittedObs++
			continue
		}
		if len(sumLines) > 0 {
			sumLines = sumLines[:len(sumLines)-1]
			omittedSum++
			continue
		}
		return body
	}
}

type obsEntry struct {
	id   int64
	line string
}

func orderObservations(obs []memory.Observation) []memory.Observation {
	if len(obs) == 0 {
		return nil
	}
	out := append([]memory.Observation(nil), obs...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Pinned != out[j].Pinned {
			return out[i].Pinned
		}
		if out[i].CreatedAt != out[j].CreatedAt {
			return out[i].CreatedAt > out[j].CreatedAt
		}
		return out[i].ID > out[j].ID
	})
	return out
}

func formatSummaryLine(s memory.Session) string {
	sentence := firstSentence(s.Summary)
	if sentence == "" {
		return ""
	}
	date := indexDate(s.EndedAt)
	if date == "" {
		date = indexDate(s.StartedAt)
	}
	return fmt.Sprintf("- %s %s: %s", shortSessionID(s.ID), date, sentence)
}

func formatObservationLine(o memory.Observation) string {
	tok := EstimateTokens(o.Title)
	return fmt.Sprintf("- #%d %s %s · %s · ~%d", o.ID, o.Type, o.Title, indexDate(o.CreatedAt), tok)
}

func joinIndexBody(sumLines []string, obsEntries []obsEntry, omittedObs, omittedSum int, footerID int64) string {
	var b strings.Builder
	for _, line := range sumLines {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	if len(sumLines) > 0 && len(obsEntries) > 0 {
		b.WriteByte('\n')
	}
	for _, e := range obsEntries {
		b.WriteString(e.line)
		b.WriteByte('\n')
	}
	if omittedObs > 0 || omittedSum > 0 {
		b.WriteString(formatOmittedNote(omittedObs, omittedSum))
		b.WriteByte('\n')
	}
	footer := formatIndexFooter(footerID)
	if footer != "" {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(footer)
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatIndexFooter(exampleID int64) string {
	if exampleID <= 0 {
		return "Full text: mem_get_observation · mem_timeline · mem_search <query>"
	}
	return fmt.Sprintf("Full text: mem_get_observation %d · mem_timeline %d · mem_search <query>", exampleID, exampleID)
}

func formatOmittedNote(omittedObs, omittedSum int) string {
	var parts []string
	if omittedObs > 0 {
		word := "observations"
		if omittedObs == 1 {
			word = "observation"
		}
		parts = append(parts, fmt.Sprintf("%d %s", omittedObs, word))
	}
	if omittedSum > 0 {
		word := "summaries"
		if omittedSum == 1 {
			word = "summary"
		}
		parts = append(parts, fmt.Sprintf("%d %s", omittedSum, word))
	}
	return fmt.Sprintf("… (%s omitted)", strings.Join(parts, ", "))
}

func shortSessionID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func indexDate(ts string) string {
	ts = strings.TrimSpace(ts)
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}

func firstSentence(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range text {
		if r == '\n' {
			break
		}
		b.WriteRune(r)
		if r == '.' || r == '!' || r == '?' {
			break
		}
	}
	out := strings.TrimSpace(b.String())
	out = strings.TrimLeft(out, "#")
	return strings.TrimSpace(out)
}
