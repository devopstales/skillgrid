package service

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/config"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/llm"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// AttachSharedLLM wires the single shared OpenAI-compatible LLM (ADR-0023)
// to the mnemonic LLM seams. It is the ONE attach point: one *llm.Client for
// the whole process, constructed here and handed to every seam that needs it.
// cfg is the config.LLMConfig from config.Load(root) (cfg.LLM).
//
// This function owns two things, by design:
//
//  1. The two PROCESS-level seams (package-level, process-wide): AskLLM
//     (service.SetAskLLM) and the dedup func (service.SetDedupLLMFunc). The
//     *llm.Client satisfies both verbatim (Complete is the shared signature),
//     so it is attached directly with NO adapter.
//
//  2. The shared *llm.Client itself, RETURNED so the caller (openProject) can
//     attach the two MEMORY-level seams (per *memory.Service instance): the
//     extraction adapter (mem.SetExtractionLLM) and the dedup memory backend
//     (mem.SetDedupLLM). Those seams live on the memory service, which only
//     exists inside openProject, so they cannot be attached here.
//
// Fail-open (ADR-0016): if cfg.Enabled is false, this is a no-op returning
// (nil, nil) — no seam is touched, so every deterministic LLM floor (cited
// ask, hash dedup, regex extraction, lossless dream join) stays in force.
// If a later HTTP call errors at runtime, each seam's own error-fallback
// already treats an LLM error as "use the floor" — AttachSharedLLM does not
// have to handle it.
//
// The one case it DOES surface is a config mistake: cfg.Enabled is true but
// BaseURL or Model is empty. That would silently attach a client that can
// never complete (the G6 "requires URL+model" gate), so it is logged AND
// returned as an error rather than silently no-opping. The caller
// (openProject) treats even that as fail-open: it logs the warning and keeps
// the floors. (Gates G5/G6/G7.)
//
// Returns the shared *llm.Client (nil when no client is built) and a
// descriptive error only for the enabled-but-incomplete config case.
func AttachSharedLLM(cfg config.LLMConfig) (*llm.Client, error) {
	// G5: off → no-op. Nothing attached, every floor in force.
	if !cfg.Enabled {
		return nil, nil
	}
	// G6: on but incomplete → a config mistake. Surface it (log + error) and
	// attach nothing so we never build a client that can never complete.
	if cfg.BaseURL == "" || cfg.Model == "" {
		err := fmt.Errorf("mnemonic.llm.enabled is true but base_url and model are required (base_url=%q model=%q)", cfg.BaseURL, cfg.Model)
		log.Printf("warning: %v", err)
		return nil, err
	}
	// G7: on and complete → build the ONE shared client and attach the two
	// process-level seams to it (the client satisfies both verbatim).
	client := llm.New(llm.Config{
		BaseURL: cfg.BaseURL,
		Model:   cfg.Model,
		APIKey:  cfg.APIKey,
		Timeout: cfg.Timeout,
	})
	SetAskLLM(client)
	SetDedupLLMFunc(client.Complete)
	return client, nil
}

// extractionLLMSystem is the system prompt the extraction adapter sends with
// the session text. It intentionally does NOT import memory's unexported
// extractionPrompt: the extraction SCHEMA (the exact JSON shape) is enforced
// by the memory package's parser (ExtractWithLLM), so this prompt only needs
// to say "extract learnings, return strict JSON." Keep it ~1-2 lines.
const extractionLLMSystem = `You extract structured learnings (decisions, bugfixes, discoveries, gotchas) from session text. Return ONLY a strict JSON object of the form {"learnings": [{"text": string, "type": string}]}; never invent content.`

// extractionLLMAdapter adapts the shared llm.Completer to the
// memory.ExtractionLLM seam (a single Extract method, not Complete). It is
// the thin bridge between the one shared client and the per-memory-service
// extraction seam (mem.SetExtractionLLM). The deterministic regex floor
// remains the caller's fallback when the adapter errors (ADR-0016).
type extractionLLMAdapter struct {
	c llm.Completer
}

// Extract routes the raw session text to the shared Completer with the fixed
// extraction system prompt. The returned string is the LLM's raw response,
// which the memory package parses into learnings (its parser owns the schema).
func (a *extractionLLMAdapter) Extract(ctx context.Context, text string) (string, error) {
	return a.c.Complete(ctx, extractionLLMSystem, text)
}

// newExtractionLLMAdapter builds the extraction seam adapter from the shared
// Completer. openProject calls it after AttachSharedLLM returns a non-nil
// client, so the memory-level extraction seam has a backend when
// cfg.Extraction.LLM is armed.
func newExtractionLLMAdapter(c llm.Completer) *extractionLLMAdapter {
	return &extractionLLMAdapter{c: c}
}

// dreamLLMSystemConsolidate / dreamLLMSystemSynthesize are the fixed system
// prompts for the dream consolidate/synthesize phases. They mirror the
// memory.DreamLLM contract: each returns a single text block (no JSON), and
// the deterministic lossless-join floor remains the caller's fallback when
// the adapter errors (ADR-0016).
const (
	dreamLLMSystemConsolidate = "Consolidate the observations into one coherent record; return the consolidated text only."
	dreamLLMSystemSynthesize  = "Lift these session summaries into one higher-tier summary; return the summary text only."
)

// dreamLLMAdapter adapts the shared llm.Completer to the memory.DreamLLM
// seam (ConsolidatePrompt + SynthesizePrompt). It is built + unit-tested so
// the seam is proven, but is NOT attached at boot: at boot there is no
// *memory.DreamExecutor yet (it is constructed lazily per distill run). The
// future lazy distill path attaches it with:
//
//	dreamExec.SetLLM(service.NewDreamLLMAdapter(sharedClient))
type dreamLLMAdapter struct {
	c llm.Completer
}

// ConsolidatePrompt folds a group of same-topic observations into one
// coherent record. It sends the topic plus a numbered observations list and
// returns the LLM's consolidated text.
func (a *dreamLLMAdapter) ConsolidatePrompt(ctx context.Context, topic string, contents []string) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "Topic: %s\nObservations:\n", topic)
	for i, c := range contents {
		fmt.Fprintf(&b, "%d. %s\n", i+1, c)
	}
	return a.c.Complete(ctx, dreamLLMSystemConsolidate, b.String())
}

// SynthesizePrompt lifts a set of lower-tier session summaries into one
// higher-tier summary. It sends a numbered summaries list and returns the
// LLM's synthesized text.
func (a *dreamLLMAdapter) SynthesizePrompt(ctx context.Context, summaries []string) (string, error) {
	var b strings.Builder
	b.WriteString("Summaries:\n")
	for i, s := range summaries {
		fmt.Fprintf(&b, "%d. %s\n", i+1, s)
	}
	return a.c.Complete(ctx, dreamLLMSystemSynthesize, b.String())
}

// NewDreamLLMAdapter builds the dream seam adapter from the shared Completer.
// It is exported so the future lazy distill path (where *memory.DreamExecutor
// is constructed) can attach it: dreamExec.SetLLM(NewDreamLLMAdapter(client)).
// The return type is memory.DreamLLM so the caller can pass it straight to
// (*memory.DreamExecutor).SetLLM without a further assertion.
func NewDreamLLMAdapter(c llm.Completer) memory.DreamLLM {
	return &dreamLLMAdapter{c: c}
}
