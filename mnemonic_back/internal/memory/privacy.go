package memory

import "strings"

const (
	privateOpen  = "<private>"
	privateClose = "</private>"
)

// StripPrivate removes every <private>…</private> span (case-insensitive,
// nested-safe). An unterminated opening tag removes through end of string.
// Runs of spaces/tabs left after removal collapse to one space; newlines stay.
func StripPrivate(s string) string {
	if s == "" {
		return ""
	}
	for {
		lower := strings.ToLower(s)
		start := strings.Index(lower, privateOpen)
		if start < 0 {
			break
		}
		i := start + len(privateOpen)
		depth := 1
		removed := false
		for depth > 0 && i <= len(s) {
			tail := strings.ToLower(s[i:])
			nextOpen := strings.Index(tail, privateOpen)
			nextClose := strings.Index(tail, privateClose)
			if nextClose < 0 {
				s = s[:start]
				removed = true
				break
			}
			if nextOpen >= 0 && nextOpen < nextClose {
				depth++
				i += nextOpen + len(privateOpen)
				continue
			}
			depth--
			if depth == 0 {
				end := i + nextClose + len(privateClose)
				s = s[:start] + s[end:]
				removed = true
				break
			}
			i += nextClose + len(privateClose)
		}
		if !removed {
			s = s[:start]
			break
		}
	}
	return collapseHorizontalSpace(s)
}

func collapseHorizontalSpace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		b.WriteRune(r)
		if r != '\n' && r != '\r' {
			prevSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}
