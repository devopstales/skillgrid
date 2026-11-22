package main

// mem_prompt.go — the host-capture verb (ADR-0067): the pi/omp extension
// and the opencode/kilo plugin spawn it on every prompt with the
// adapter's session id; cursor's beforeSubmitPrompt hook pipes its JSON
// envelope. Fail-open contract: prints nothing on success, never blocks
// the host beyond the store's busy_timeout, a non-zero exit is a drop the
// extension counts — host payload data never appears in the command line
// (it travels on stdin).

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/core/config"
)

// stdinReader is the verb's stdin seam (the tests swap it; production is
// os.Stdin — the same seam pattern as storeFor/handlerFor/gitToplevel).
var stdinReader io.Reader = os.Stdin

// savePromptStdinCap bounds the captured prompt; the excess is discarded
// (fail-open — the content ceiling inside redactPrompt bounds the row).
const savePromptStdinCap = 256 << 10

// readStdinCapped reads at most savePromptStdinCap bytes from r (reads
// one byte over to detect excess, then discards it).
func readStdinCapped(r io.Reader) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, savePromptStdinCap+1))
	if len(b) > savePromptStdinCap {
		b = b[:savePromptStdinCap]
	}
	return b, err
}

// memSavePrompt runs `memory save-prompt`.
func memSavePrompt(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("memory save-prompt", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var pf projectFlag
	pf.add(fs)
	var session, sourceInboxID string
	fs.StringVar(&session, "id", "", "the adapter's session id (required)")
	fs.StringVar(&sourceInboxID, "source-inbox-id", "", "optional source inbox id")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if session == "" {
		return errExit("memory save-prompt: --id is required")
	}
	raw, err := readStdinCapped(stdinReader)
	if err != nil {
		fmt.Fprintf(errW(), "mnemonic save-prompt: stdin: %v\n", err)
		return exitErr
	}
	h, st, err := openRuntime(cwd, fl, pf.project)
	if err != nil {
		fmt.Fprintf(errW(), "mnemonic save-prompt: %v\n", err)
		return exitErr
	}
	defer st.Close()
	if _, err := h.SavePromptOnce(context.Background(), session, extractPrompt(raw), pf.project, sourceInboxID); err != nil {
		fmt.Fprintf(errW(), "mnemonic save-prompt: %v\n", err)
		return exitErr
	}
	return exitOK // prints nothing
}

// extractPrompt pulls the prompt text out of the stdin payload: a JSON
// object with a non-empty string "prompt" field (cursor's
// beforeSubmitPrompt shape) yields that field; anything else is the raw
// text, trimmed. A malformed envelope is text, not an error (fail-open).
func extractPrompt(raw []byte) string {
	t := strings.TrimSpace(string(raw))
	if strings.HasPrefix(t, "{") {
		var env struct {
			Prompt string `json:"prompt"`
		}
		if json.Unmarshal([]byte(t), &env) == nil && env.Prompt != "" {
			return env.Prompt
		}
	}
	return t
}
