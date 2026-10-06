package secondbrain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// llmTimeout bounds the LLM synthesis call in mem_ask llm mode; on timeout
// the answer fails open to the cited floor (blueprint Task 2).
const llmTimeout = 3 * time.Second

const llmSystemPrompt = "You are a concise research assistant. Answer using ONLY the provided sources. Cite every claim as [obs:<id>]."

// Ask is the top-level mem_ask entry. mode "cited" (the default) returns the
// deterministic, no-LLM cited floor. mode "llm" runs the cited floor first,
// then synthesizes prose with [obs:<id>] citations; on any LLM error or the
// 3s timeout it returns the cited result unchanged (fail open, no error).
// Sources are always populated from the cited floor, even when the LLM omits
// markers.
func Ask(ctx context.Context, svc *service.Service, mode, query, projectID string, allProjects bool, maxTokens int) (*AskResult, error) {
	cited, err := AskCited(ctx, svc, query, projectID, allProjects, maxTokens)
	if err != nil {
		return nil, err
	}
	if mode != "llm" || len(cited.Citations) == 0 {
		// No citations: nothing to synthesize over (empty query is a no-op,
		// zero hits is the floor itself) — the LLM is never reached.
		return cited, nil
	}
	prose, perr := synthesize(ctx, query, cited)
	if perr != nil {
		return cited, nil // fail open: the cited floor is the answer
	}
	cited.Answer = prose
	return cited, nil
}

// synthesize calls the LLM seam over the cited floor and returns the prose.
// The 3s timeout is enforced here; a nil seam (no LLM configured) is an error
// to the caller, which fails open.
func synthesize(ctx context.Context, query string, cited *AskResult) (string, error) {
	llm := service.AskLLMSeam()
	if llm == nil {
		return "", fmt.Errorf("no ask LLM configured")
	}
	// The 3s budget never extends an earlier deadline already on ctx.
	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		deadline = time.Now().Add(llmTimeout)
	}
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return "", context.DeadlineExceeded
	}
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	var b strings.Builder
	fmt.Fprintf(&b, "Question: %s\n\nSources:\n", query)
	for _, c := range cited.Citations {
		fmt.Fprintf(&b, "[obs:%d] (%s) %s: %s\n", c.ID, c.Type, c.Title, c.Snippet)
	}
	prose, err := llm.Complete(ctx, llmSystemPrompt, b.String())
	if err != nil {
		return "", err
	}
	prose = strings.TrimSpace(prose)
	if prose == "" {
		return "", fmt.Errorf("ask LLM returned an empty answer")
	}
	return prose, nil
}
