package memory

// Skills framework + lifecycle hooks for context injection (014, step 24).
//
// A SKILL is an observation with memory_type = "skill" whose content is the
// Markdown skill body. Its matching metadata is the topic_key, shaped as
// skill/<intent>/<lang>: the intent segment selects the work intent the skill
// applies to (exploration / debugging / review / refactor — the 4-intent
// contract reused from steps 19/22) and the lang segment is an empty language
// tag (any-language skill) or a specific project language (e.g. "go").
// MatchSkills filters skills by intent first, then ranks language-matched
// skills ahead of language-unspecific ones.
//
// LIFECYCLE HOOKS are opt-in per project (mnemonic.hooks.enabled, default
// false) and each runs under a per-hook timeout (mnemonic.hooks.timeout,
// default 30s). The four hook types:
//
//   - session-start  — inject the project's recent memories + the skills
//     matched by the query's classified intent.
//   - pre-edit       — run the change-impact analysis (AnalyzeImpact, step 23)
//     for the target file and return the risk classification.
//   - prompt-submit  — classify the prompt's work intent (ClassifyWorkIntent).
//   - session-stop   — run the session-close distillation (layer.Distill) for
//     the session.
//
// RunHook is the single entry point: it checks the opt-in switch, wraps the
// hook work in a context.WithTimeout budget, and dispatches by hook type. A
// test seam (SetHookFunc) can replace a hook's work, which is how the timeout
// is exercised deterministically (the seam observes the context deadline).
//
// The layer package imports this package, so the distillation is invoked
// through an injectable seam (SetDistillRunner) rather than a direct import:
// the service layer wires the real layer.Distill at open time, and a test
// installs its own. The zero value (no runner) reports distillation as
// skipped, never as an error.

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ── Skills (24.1) ─────────────────────────────────────────────────────────

// skillTopicPrefix is the topic_key prefix every skill observation carries.
const skillTopicPrefix = "skill/"

// skillMatchLimit caps how many skills of an intent are scanned during
// matching. Skills are a small, curated set per project; the cap keeps the
// SQL bounded.
const skillMatchLimit = 100

// Skill is one matched skill: the observation id + title, the skill body
// (Markdown content), the classified intent it applies to, and its language
// tag ("" = any-language).
type Skill struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	Intent     string `json:"intent"`
	Language   string `json:"language,omitempty"`
	TopicKey   string `json:"topic_key,omitempty"`
	Relevance  float64 `json:"relevance,omitempty"`
}

// skillParts splits a skill topic_key into (intent, lang). A non-skill key
// (no skill/ prefix) yields ("", "") and ok = false.
func skillParts(topicKey string) (intent, lang string, ok bool) {
	if !strings.HasPrefix(topicKey, skillTopicPrefix) {
		return "", "", false
	}
	segs := strings.Split(strings.TrimPrefix(topicKey, skillTopicPrefix), "/")
	if len(segs) < 1 || segs[0] == "" {
		return "", "", false
	}
	intent = strings.ToLower(segs[0])
	if len(segs) > 1 {
		lang = strings.ToLower(segs[1])
	}
	return intent, lang, true
}

// MatchSkills returns the project's skills matching intent, ranked by
// relevance (24.1). Skills are observations with memory_type = "skill" and
// topic_key = skill/<intent>/<lang>. Matching is by intent first (the topic_key
// intent segment equals the given intent); then skills whose language matches
// one of projectLangs, or whose language matches a mentioned file's extension,
// rank ahead of language-unspecific (any-language) skills. The result is
// sorted by relevance descending, then topic_key for determinism.
func (s *Service) MatchSkills(ctx context.Context, intent Intent, mentionedFiles, projectLangs []string) ([]Skill, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, fmt.Errorf("memory service not initialized")
	}
	intentKey := strings.ToLower(strings.TrimSpace(string(intent)))
	if intentKey == "" {
		return nil, fmt.Errorf("intent is required")
	}

	// Languages worth matching: the declared project languages plus the
	// extensions of mentioned files (a file like auth.go signals "go").
	wantedLangs := map[string]struct{}{}
	for _, l := range projectLangs {
		if l = strings.ToLower(strings.TrimSpace(l)); l != "" {
			wantedLangs[l] = struct{}{}
		}
	}
	for _, f := range mentionedFiles {
		if lang := extLanguage(f); lang != "" {
			wantedLangs[lang] = struct{}{}
		}
	}

	// Scan the project's skills of this intent. memory_type = "skill" AND a
	// topic_key in the skill/<intent>/... shape: the topic_key is the single
	// source of the intent, so the query is intent-scoped from the start.
	prefix := skillTopicPrefix + intentKey + "/"
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT `+obsSelectCols+`
		FROM observations
		WHERE deleted_at IS NULL AND project = ? AND memory_type = ?
		  AND topic_key LIKE ?
		ORDER BY created_at DESC, id DESC
		LIMIT ?`,
		s.projectID, MemoryTypeSkill, prefix+"%", skillMatchLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("match skills: %w", err)
	}
	defer rows.Close()

	obsList, err := scanObservations(rows)
	if err != nil {
		return nil, fmt.Errorf("match skills scan: %w", err)
	}

	type scored struct {
		sk        Skill
		relevance float64
	}
	var out []scored
	for _, obs := range obsList {
		if obs.TopicKey == "" {
			continue
		}
		intent2, lang, ok := skillParts(obs.TopicKey)
		if !ok || intent2 != intentKey {
			continue
		}
		// Relevance: language-matched skills outrank any-language skills.
		// A specific (non-empty) language that is in the wanted set scores 1.0;
		// everything else (any-language, or a language not in the set) scores
		// 0.5. With no wanted languages at all, every skill is equally
		// relevant (the any-language default).
		relevance := 0.5
		if len(wantedLangs) > 0 && lang != "" {
			if _, ok := wantedLangs[lang]; ok {
				relevance = 1.0
			}
		}
		out = append(out, scored{
			sk: Skill{
				ID:        obs.ID,
				Title:     obs.Title,
				Content:   obs.Content,
				Intent:    intent2,
				Language:  lang,
				TopicKey:  obs.TopicKey,
				Relevance: relevance,
			},
			relevance: relevance,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].relevance != out[j].relevance {
			return out[i].relevance > out[j].relevance
		}
		return out[i].sk.TopicKey < out[j].sk.TopicKey
	})
	skills := make([]Skill, 0, len(out))
	for _, e := range out {
		skills = append(skills, e.sk)
	}
	return skills, nil
}

// extLanguage maps a file path to its language tag by extension (the small
// set the code index recognizes). A non-matching or extension-less path yields
// "" (no language signal).
func extLanguage(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx", ".mjs":
		return "javascript"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".rb":
		return "ruby"
	case ".cs":
		return "csharp"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "cpp"
	case ".swift":
		return "swift"
	case ".kt":
		return "kotlin"
	case ".php":
		return "php"
	default:
		return ""
	}
}

// ── Lifecycle hooks (24.2 / 24.3) ─────────────────────────────────────────

// Hook types (24.2).
const (
	HookSessionStart  = "session-start"
	HookPreEdit       = "pre-edit"
	HookPromptSubmit  = "prompt-submit"
	HookSessionStop   = "session-stop"
)

// DefaultHookTimeout is the per-hook execution budget (014 step 24.3). A
// hook that exceeds it is cut off and RunHook returns a descriptive timeout
// error. Tunable via the mnemonic.hooks.timeout config key (SetHooks).
const DefaultHookTimeout = 30 * time.Second

// HooksConfig tunes the lifecycle hooks (014 step 24.3). Enabled is the
// opt-in switch (default false — no hook runs until enabled). Timeout is the
// per-hook budget; a non-positive value falls back to DefaultHookTimeout
// inside SetHooks.
type HooksConfig struct {
	Enabled bool
	Timeout time.Duration
}

// HookPayload is the per-hook input (24.2). File is the pre-edit target; Query
// is the session-start / prompt-submit text to classify and retrieve on;
// SessionID is the session-stop distillation target. Unused fields are empty.
type HookPayload struct {
	File      string `json:"file,omitempty"`
	Query     string `json:"query,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

// HookResult is the per-hook output (24.2). Memories/Skills are the
// session-start context injection; Intent is the classified intent (also
// filled by session-start from the query); Risk is the pre-edit impact
// classification; Distilled reports whether session-stop ran the distillation
// pass (false when the runner is unconfigured — a no-op, not an error).
type HookResult struct {
	Memories  []Observation `json:"memories,omitempty"`
	Skills    []Skill       `json:"skills,omitempty"`
	Intent    Intent        `json:"intent,omitempty"`
	Risk      *ImpactResult `json:"risk,omitempty"`
	Distilled bool          `json:"distilled,omitempty"`
}

// hooksDisabledError marks "hooks are not enabled for this project".
type hooksDisabledError struct{}

func (hooksDisabledError) Error() string {
	return "lifecycle hooks are disabled for this project (mnemonic.hooks.enabled is false)"
}

// IsHooksDisabled reports whether err is the opt-in-disabled sentinel.
func IsHooksDisabled(err error) bool {
	var d hooksDisabledError
	return errors.As(err, &d)
}

// hookTimeoutError marks "the hook exceeded its timeout budget".
type hookTimeoutError struct{ hook string }

func (e hookTimeoutError) Error() string {
	return fmt.Sprintf("%s hook timed out after the configured budget", e.hook)
}

// IsHookTimeout reports whether err is a hook timeout.
func IsHookTimeout(err error) bool {
	var e hookTimeoutError
	return errors.As(err, &e)
}

// hookFunc is the work one hook type performs. It receives the (possibly
// deadline-bound) context so a slow hook observes ctx.Done() and can return
// early. It is the seam SetHookFunc swaps for a test.
type hookFunc func(ctx context.Context, hookType string, payload HookPayload) (HookResult, error)

// distillRunner is the injectable session-close distillation seam (24.2): the
// service layer wires the real layer.Distill; a test installs its own. It
// returns (distilled, err) — distilled false with a nil error is the
// unconfigured/no-op case.
type distillRunner func(ctx context.Context, svc *Service, sessionID string) (bool, error)

// SetHooks configures the lifecycle hooks (014 step 24.3). Enabled is the
// opt-in switch; a non-positive Timeout falls back to DefaultHookTimeout. The
// service layer calls it from the mnemonic.hooks config key.
func (s *Service) SetHooks(cfg HooksConfig) {
	if s == nil {
		return
	}
	to := cfg.Timeout
	if to <= 0 {
		to = DefaultHookTimeout
	}
	hooksMu.Lock()
	defer hooksMu.Unlock()
	s.hooksEnabled = cfg.Enabled
	s.hooksTimeout = to
}

// HookTimeout returns the active per-hook budget (DefaultHookTimeout when
// unset). Exposed for tests that position the timeout deterministically.
func (s *Service) HookTimeout() time.Duration {
	if s == nil {
		return DefaultHookTimeout
	}
	hooksMu.Lock()
	defer hooksMu.Unlock()
	if s.hooksTimeout <= 0 {
		return DefaultHookTimeout
	}
	return s.hooksTimeout
}

// SetHookFunc installs a custom hookFunc for hookType (a test seam for the
// timeout test). It does not enable the hooks — the opt-in switch is
// independent. An empty hookType is ignored.
func (s *Service) SetHookFunc(hookType string, fn hookFunc) {
	if s == nil || hookType == "" {
		return
	}
	hooksMu.Lock()
	defer hooksMu.Unlock()
	if s.hookFns == nil {
		s.hookFns = map[string]hookFunc{}
	}
	s.hookFns[hookType] = fn
}

// distillRunnerFn is the seam's accessor (guarded by distillMu).
func (s *Service) distillRunnerFn() distillRunner {
	if s == nil {
		return nil
	}
	distillMu.Lock()
	defer distillMu.Unlock()
	return s.distillRunner
}

// SetDistillRunner installs the session-close distillation seam (24.2). The
// service layer wires the real layer.Distill at open time; nil clears the seam
// (session-stop reports distilled = false, a no-op).
func (s *Service) SetDistillRunner(fn distillRunner) {
	if s == nil {
		return
	}
	distillMu.Lock()
	defer distillMu.Unlock()
	s.distillRunner = fn
}

// RunHook executes one lifecycle hook (24.2). It first checks the opt-in
// switch (disabled → hooksDisabledError, no work runs), then wraps the hook
// work in a context.WithTimeout budget (the configured per-hook timeout) and
// dispatches by hook type. On timeout it returns a descriptive
// hookTimeoutError. Unknown hook types are rejected before any work runs.
func (s *Service) RunHook(ctx context.Context, hookType string, payload HookPayload) (HookResult, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return HookResult{}, fmt.Errorf("memory service not initialized")
	}
	enabled, timeout := s.hooksState()
	if !enabled {
		return HookResult{}, hooksDisabledError{}
	}
	var fn hookFunc
	switch hookType {
	case HookSessionStart, HookPreEdit, HookPromptSubmit, HookSessionStop:
	default:
		return HookResult{}, fmt.Errorf("unknown hook type %q (valid: session-start, pre-edit, prompt-submit, session-stop)", hookType)
	}
	hooksMu.Lock()
	if s.hookFns != nil {
		fn = s.hookFns[hookType]
	}
	hooksMu.Unlock()
	if fn == nil {
		fn = s.defaultHook
	}

	hctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	res, err := fn(hctx, hookType, payload)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return HookResult{}, hookTimeoutError{hook: hookType}
		}
		return HookResult{}, err
	}
	return res, nil
}

// hooksState returns the active (enabled, timeout) pair.
func (s *Service) hooksState() (bool, time.Duration) {
	hooksMu.Lock()
	defer hooksMu.Unlock()
	to := s.hooksTimeout
	if to <= 0 {
		to = DefaultHookTimeout
	}
	return s.hooksEnabled, to
}

// defaultHook is the production dispatch for the four built-in hook types.
func (s *Service) defaultHook(ctx context.Context, hookType string, payload HookPayload) (HookResult, error) {
	switch hookType {
	case HookSessionStart:
		return s.hookSessionStart(ctx, payload)
	case HookPreEdit:
		return s.hookPreEdit(ctx, payload)
	case HookPromptSubmit:
		return s.hookPromptSubmit(ctx, payload)
	case HookSessionStop:
		return s.hookSessionStop(ctx, payload)
	default:
		return HookResult{}, fmt.Errorf("unknown hook type %q", hookType)
	}
}

// hookSessionStart injects the project's recent memories + the skills matched
// by the query's classified intent (24.2). With no query, the intent is
// exploration (the default) and only memories are returned.
func (s *Service) hookSessionStart(ctx context.Context, payload HookPayload) (HookResult, error) {
	query := strings.TrimSpace(payload.Query)
	intent := IntentExploration
	if query != "" {
		intent = ClassifyWorkIntent(query)
	}
	res := HookResult{Intent: intent}
	if mems, err := s.Recent(ctx, defaultContextLimit); err == nil {
		res.Memories = mems
	}
	if query != "" {
		if skills, err := s.MatchSkills(ctx, intent, nil, nil); err == nil {
			res.Skills = skills
		}
	}
	return res, nil
}

// hookPreEdit runs the change-impact analysis for the target file (24.2): the
// risk classification (hub / dependents / impact level) for a single file.
func (s *Service) hookPreEdit(ctx context.Context, payload HookPayload) (HookResult, error) {
	file := strings.TrimSpace(payload.File)
	if file == "" {
		return HookResult{}, fmt.Errorf("pre-edit hook requires a file in the payload")
	}
	results, err := s.AnalyzeImpact(ctx, []string{file})
	if err != nil {
		return HookResult{}, err
	}
	if len(results) == 0 {
		return HookResult{}, nil
	}
	r := results[0]
	return HookResult{Risk: &r}, nil
}

// hookPromptSubmit classifies the prompt's work intent (24.2).
func (s *Service) hookPromptSubmit(ctx context.Context, payload HookPayload) (HookResult, error) {
	_ = ctx
	return HookResult{Intent: ClassifyWorkIntent(payload.Query)}, nil
}

// hookSessionStop runs the session-close distillation for the session (24.2).
// Without a configured distill runner (the default for a directly-built
// service) it reports distilled = false — a no-op, not an error.
func (s *Service) hookSessionStop(ctx context.Context, payload HookPayload) (HookResult, error) {
	sessionID := strings.TrimSpace(payload.SessionID)
	if sessionID == "" {
		return HookResult{}, fmt.Errorf("session-stop hook requires a session_id in the payload")
	}
	runner := s.distillRunnerFn()
	if runner == nil {
		return HookResult{Distilled: false}, nil
	}
	distilled, err := runner(ctx, s, sessionID)
	if err != nil {
		return HookResult{}, err
	}
	return HookResult{Distilled: distilled}, nil
}

// hooksMu guards the hooks* fields on Service (SetHooks / SetHookFunc read
// and write them; RunHook reads them). A package-level mutex (not on *Service)
// so a nil receiver's accessors are safe and the pattern matches the
// package-level seams elsewhere in this file's family.
var (
	hooksMu  sync.Mutex
	distillMu sync.Mutex
)
