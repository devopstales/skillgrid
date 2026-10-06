package docs

import (
	"regexp"
	"strings"
	"time"
)

// MemberSession is the Mnemonic session that worked a ledger row: its harness
// (cursor, opencode, …) and live state. Ledger rows carry no session id, so the
// join is by the ticket or task id the agent put in its session title.
type MemberSession struct {
	ID         string `json:"id"`
	Project    string `json:"project"`
	Agent      string `json:"agent,omitempty"`
	Status     string `json:"status,omitempty"`
	Title      string `json:"title,omitempty"`
	StartedAt  string `json:"started_at,omitempty"`
	LastActive string `json:"last_active,omitempty"`
	Live       string `json:"live,omitempty"`
}

var memberIDRE = regexp.MustCompile(`(?i)\b(?:TASK-\d+(?:\.\d+)?|TICKET-\d+)\b`)

// WorkingWindow is how recent the last session event must be for "working".
const WorkingWindow = 2 * time.Minute

// MatchSession returns the session whose title names the row's ticket or task
// id. Leaf-name words break ties between sessions sharing a ticket number, then
// the newest start wins. A row with no ticket or task id matches nothing.
func MatchSession(m TeamMember, sessions []MemberSession) *MemberSession {
	ids := memberIDRE.FindAllString(m.Task+" "+m.Agent, -1)
	if len(ids) == 0 {
		return nil
	}
	words := leafWords(m.Agent)
	best, bestScore := -1, 0
	for i, s := range sessions {
		title := strings.ToLower(s.Title)
		score := 0
		for _, id := range ids {
			if strings.Contains(title, strings.ToLower(id)) {
				score += 10
			}
		}
		if score == 0 {
			continue
		}
		for _, w := range words {
			if strings.Contains(title, w) {
				score++
			}
		}
		if score > bestScore || (score == bestScore && best >= 0 && s.StartedAt > sessions[best].StartedAt) {
			best, bestScore = i, score
		}
	}
	if best < 0 {
		return nil
	}
	out := sessions[best]
	return &out
}

func leafWords(agent string) []string {
	var out []string
	for _, w := range strings.FieldsFunc(strings.ToLower(agent), func(r rune) bool {
		return r == '-' || r == '_' || r == ' ' || r == '.'
	}) {
		if len(w) >= 3 {
			out = append(out, w)
		}
	}
	return out
}

// LiveStatus maps a session to Herdr-style readings: an ended session is
// finished, a session with an event inside WorkingWindow is working, any other
// open session is idle.
func LiveStatus(status, lastActive string, now time.Time) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "ended", "finished", "done":
		return "finished"
	}
	if t, err := time.Parse(time.RFC3339, strings.TrimSpace(lastActive)); err == nil {
		if d := now.Sub(t); d >= 0 && d <= WorkingWindow {
			return "working"
		}
	}
	return "idle"
}
