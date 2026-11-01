package context_harness

import (
	"fmt"
	"strings"
)

// RenderContextBlock formats a RetrieveResult as a compact, token-cost-annotated
// markdown block for session injection. It returns an empty string when the
// result is nil or carries no items.
func RenderContextBlock(res *RetrieveResult, projectID string) string {
	if res == nil || len(res.Items) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Session Context Injection\n\n")
	fmt.Fprintf(&b, "**Project:** %s\n", projectID)
	if res.Degraded {
		b.WriteString("**Mode:** BM25-only (no embedder)\n")
	}
	b.WriteString("\n")
	for i, item := range res.Items {
		fmt.Fprintf(&b, "%d. **%s** (%s, %d tokens)\n", i+1, item.Title, item.Source, item.TokenCost)
		if item.Snippet != "" {
			fmt.Fprintf(&b, "   %s\n", item.Snippet)
		}
	}
	fmt.Fprintf(&b, "\n**Total:** %d tokens total\n", res.TotalTokens)
	return b.String()
}
