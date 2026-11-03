package process

import (
	"context"
	"errors"
)

// stubLLM is the injected LLM for tests: it returns a fixed label, so label
// caching is testable without a live LLM. The error-mode variant makes it
// return an error (the LLM-down path). Label has a value receiver so the stub
// is usable both as a value (in the Run call) and as a pointer (when the
// test needs to read back call counts).
type stubLLM struct {
	label string
	err   error
}

func (s stubLLM) Label(ctx context.Context, summary string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.label, nil
}

// callCounter is a pointer-receiver LLM that records how many times Label was
// called, for the caching tests (unchanged flow must not re-call).
type callCounter struct {
	label string
	calls int
}

func (c *callCounter) Label(ctx context.Context, summary string) (string, error) {
	c.calls++
	return c.label, nil
}

var errLLMDown = errors.New("llm down")
