package mcp

// maxEditDistance is the maximum Levenshtein distance at which a suggestion
// is offered. Beyond this, the names are too different to be useful.
const maxEditDistance = 3

// suggestTool returns the closest registered tool name to query by
// Levenshtein distance, or "" if nothing is within maxEditDistance.
func suggestTool(query string, registered []string) string {
	best := ""
	bestDist := maxEditDistance + 1
	for _, name := range registered {
		d := levenshtein(query, name)
		if d < bestDist {
			bestDist = d
			best = name
		}
	}
	if bestDist <= maxEditDistance {
		return best
	}
	return ""
}

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	// Quick reject: if the length difference alone exceeds the max, skip.
	if la > lb {
		a, b = b, a
		la, lb = lb, la
	}
	if lb-la > maxEditDistance {
		return maxEditDistance + 1
	}
	prev := make([]int, la+1)
	curr := make([]int, la+1)
	for i := range prev {
		prev[i] = i
	}
	for j := 1; j <= lb; j++ {
		curr[0] = j
		for i := 1; i <= la; i++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[i] = min3(curr[i-1]+1, prev[i]+1, prev[i-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[la]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// toolSuggestion builds the "Did you mean X?" suffix for an unknown tool name.
func toolSuggestion(name string, registered []string) string {
	s := suggestTool(name, registered)
	if s == "" {
		return ""
	}
	return "; did you mean " + s + "?"
}


