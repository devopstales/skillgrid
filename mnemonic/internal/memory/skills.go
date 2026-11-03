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
// LIFECYCLE HOOKS are on by default per project (mnemonic.hooks.enabled,
// default true — observe-mode: writes rows, never blocks) and each runs
// under a per-hook timeout (mnemonic.hooks.timeout, default 30s). The hook
// types:
//
//   - session-start  — inject the project's recent memories + the skills
//     matched by the query's classified intent.
//   - pre-edit       — run the change-impact analysis (AnalyzeImpact, step 23)
//     for the target file and return the risk classification.
//   - prompt-submit  — classify the prompt's work intent (ClassifyWorkIntent).
//   - session-stop   — run the session-close distillation (layer.Distill) for
//     the session.
//   - compact        — before compaction, save one upserted continuity
//     observation. Its budget is min(configured timeout, 3s) and it fails
//     open: a timeout or save error returns a result, never hookTimeoutError.
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
	"database/sql"
	"encoding/json"
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
	ID        int64   `json:"id"`
	Title     string  `json:"title"`
	Content   string  `json:"content"`
	Intent    string  `json:"intent"`
	Language  string  `json:"language,omitempty"`
	TopicKey  string  `json:"topic_key,omitempty"`
	Relevance float64 `json:"relevance,omitempty"`
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
	HookSessionStart = "session-start"
	HookPreEdit      = "pre-edit"
	HookPromptSubmit = "prompt-submit"
	HookSessionStop  = "session-stop"
	// HookCompact writes one continuity observation before the runtime
	// compacts the conversation. A repeat upserts on topic_key
	// compaction/<session>.
	HookCompact = "compact"
	// HookPostToolUse records one finished tool call as an ordered session
	// event plus its per-session counter bump, in a single transaction.
	HookPostToolUse = "post_tool_use"
)

// DefaultHookTimeout is the per-hook execution budget (014 step 24.3). A
// hook that exceeds it is cut off and RunHook returns a descriptive timeout
// error. Tunable via the mnemonic.hooks.timeout config key (SetHooks).
const DefaultHookTimeout = 30 * time.Second

// compactHookBudget caps the compact hook. The configured per-hook timeout
// still applies when it is shorter; compact never waits longer than this,
// and a deadline is fail-open rather than hookTimeoutError.
const compactHookBudget = 3 * time.Second

// compactObservationType is the continuity row's coarse type. Save rejects
// anything outside validTypes, and session_summary is the compact contract,
// so the hook registers it on that map (same package) instead of a second writer.
const compactObservationType = "session_summary"

func init() {
	validTypes[compactObservationType] = struct{}{}
}

// HooksConfig tunes the lifecycle hooks (014 step 24.3). Enabled is the
// switch (config-level default true — observe-mode; the memory-package zero
// value is false, so a fresh Service has hooks off until the service layer
// routes the config via SetHooks). Timeout is the per-hook budget; a
// non-positive value falls back to DefaultHookTimeout inside SetHooks.
type HooksConfig struct {
	Enabled bool
	Timeout time.Duration
}

// HookPayload is the per-hook input (24.2). File is the pre-edit target; Query
// is the session-start / prompt-submit text to classify and retrieve on;
// SessionID is the session-stop distillation target. The post_tool_use
// extension (ActionType/ToolName/Command/ResultStatus/ContentHash/
// ContentPreview) carries one finished tool call; unused fields stay empty.
type HookPayload struct {
	File           string `json:"file,omitempty"`
	Query          string `json:"query,omitempty"`
	SessionID      string `json:"session_id,omitempty"`
	ActionType     string `json:"action_type,omitempty"`
	ToolName       string `json:"tool_name,omitempty"`
	Command        string `json:"command,omitempty"`
	ResultStatus   string `json:"result_status,omitempty"`
	ContentHash    string `json:"content_hash,omitempty"`
	ContentPreview string `json:"content_preview,omitempty"`
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
// hookTimeoutError, except compact, which uses min(configured, 3s) and fails
// open (DeadlineExceeded → Distilled false and a nil error; any other compact
// error → an empty result and a nil error). Unknown hook types are rejected
// before any work runs.
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
	case HookSessionStart, HookPreEdit, HookPromptSubmit, HookSessionStop, HookPostToolUse, HookCompact:
	default:
		return HookResult{}, fmt.Errorf("unknown hook type %q (valid: session-start, pre-edit, prompt-submit, session-stop, compact)", hookType)
	}
	hooksMu.Lock()
	if s.hookFns != nil {
		fn = s.hookFns[hookType]
	}
	hooksMu.Unlock()
	if fn == nil {
		fn = s.defaultHook
	}

	if hookType == HookCompact && timeout > compactHookBudget {
		timeout = compactHookBudget
	}
	hctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	res, err := fn(hctx, hookType, payload)
	if err != nil && hookType == HookCompact {
		if errors.Is(err, context.DeadlineExceeded) {
			return HookResult{Distilled: false}, nil
		}
		return HookResult{}, nil
	}
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

// defaultHook is the production dispatch for the built-in hook types.
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
	case HookCompact:
		return s.hookCompact(ctx, payload)
	case HookPostToolUse:
		return s.hookPostToolUse(ctx, payload)
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

// hookCompact persists one continuity observation before compaction. An empty
// session is a no-op. The body is the short list of titles from
// CompactionContext. Save upserts on topic_key compaction/<session>, so a
// repeat updates the same row. Errors propagate to RunHook, which fails this
// hook open.
func (s *Service) hookCompact(ctx context.Context, payload HookPayload) (HookResult, error) {
	sessionID := strings.TrimSpace(payload.SessionID)
	if sessionID == "" {
		return HookResult{}, nil
	}
	cc, err := s.CompactionContext(ctx, sessionID, 5)
	if err != nil {
		return HookResult{}, err
	}
	_, err = s.Save(ctx, SaveInput{
		SessionID: sessionID,
		Type:      compactObservationType,
		Title:     "compaction continuity",
		Content:   compactTitles(cc),
		TopicKey:  "compaction/" + sessionID,
		Owner:     sessionID,
		Scope:     "project",
	})
	if err != nil {
		return HookResult{}, err
	}
	return HookResult{}, nil
}

// compactTitles is the short continuity body: the session title, then each
// compaction observation's title. Save requires non-empty content, so a
// context with no titles still carries the continuity title.
func compactTitles(cc CompactionContext) string {
	var lines []string
	if title := strings.TrimSpace(cc.Title); title != "" {
		lines = append(lines, title)
	}
	for _, obs := range cc.Observations {
		title := obs
		if i := strings.Index(obs, " — "); i > 0 {
			title = obs[:i]
		}
		if title = strings.TrimSpace(title); title != "" {
			lines = append(lines, title)
		}
	}
	if len(lines) == 0 {
		return "compaction continuity"
	}
	return strings.Join(lines, "\n")
}

// hookPostToolUse records one finished tool call (TICKET-02): an ordered
// session_events row plus the per-session counter bump, committed in ONE
// transaction. Sequencing reuses the changes.go writer seam (appendSessionEvent
// + its internal nextSequenceTx); the same tx then enriches the row with the
// tool detail and bumps the sessions counters, so the event and its counters
// can never diverge.
//
// Tool→action mapping: Write→file_write, Read→file_read, Shell→command_exec,
// everything else→tool_use. Counter mapping: file_write→files_written,
// file_read→files_read, command_exec→commands_exec, an error result→errors, a
// sensitive path→sensitive_actions (on top of the action counter).
//
// Sensitive paths (IsSensitivePath) never persist raw content: the payload
// stores only {"content_hash","preview"} — the SHA-256 of the full content
// plus the masked first-200-chars preview from redactPreview.
func (s *Service) hookPostToolUse(ctx context.Context, payload HookPayload) (HookResult, error) {
	sessionID := strings.TrimSpace(payload.SessionID)
	if sessionID == "" {
		return HookResult{}, fmt.Errorf("post_tool_use hook requires a session_id in the payload")
	}
	rowProject, _, lerr := lookupSessionRow(ctx, s.store.DB, sessionID)
	if lerr != nil {
		if errors.Is(lerr, sql.ErrNoRows) {
			return HookResult{}, fmt.Errorf("session %s not found", sessionID)
		}
		return HookResult{}, fmt.Errorf("post_tool_use look-up: %w", lerr)
	}

	action := actionForTool(payload.ToolName, payload.ActionType)
	status := strings.TrimSpace(payload.ResultStatus)
	if status == "" {
		status = "success"
	}
	isErr := isErrorStatus(status)
	path := strings.TrimSpace(payload.File)
	sensitive := IsSensitivePath(path)

	storedPayload := strings.TrimSpace(payload.ContentPreview)
	if sensitive && storedPayload != "" {
		hash := strings.TrimSpace(payload.ContentHash)
		if hash == "" {
			hash, _ = redactPreview(storedPayload)
		}
		_, masked := redactPreview(storedPayload)
		raw, merr := json.Marshal(map[string]string{
			"content_hash": hash,
			"preview":      masked,
		})
		if merr != nil {
			return HookResult{}, fmt.Errorf("post_tool_use redact payload: %w", merr)
		}
		storedPayload = string(raw)
	}

	now := eventNow()
	tx, err := s.store.DB.BeginTx(ctx, nil)
	if err != nil {
		return HookResult{}, fmt.Errorf("post_tool_use begin: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	// Single insert path: the changes.go writer assigns the next sequence.
	seq, err := appendSessionEvent(ctx, tx, rowProject, sessionID, action, "", now)
	if err != nil {
		return HookResult{}, err
	}
	sensFlag := 0
	if sensitive {
		sensFlag = 1
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE session_events
		SET result_status = ?, is_sensitive = ?, tool_name = ?, path = ?, command = ?, payload = ?
		WHERE session_id = ? AND sequence = ?`,
		status, sensFlag,
		strings.TrimSpace(payload.ToolName), path, strings.TrimSpace(payload.Command),
		storedPayload, sessionID, seq,
	); err != nil {
		return HookResult{}, fmt.Errorf("post_tool_use enrich event: %w", err)
	}
	if bumps := counterBumps(action, isErr, sensitive); len(bumps) > 0 {
		sets := make([]string, 0, len(bumps))
		for _, col := range bumps {
			sets = append(sets, col+" = "+col+" + 1")
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE sessions SET `+strings.Join(sets, ", ")+` WHERE id = ? AND project = ?`,
			sessionID, rowProject,
		); err != nil {
			return HookResult{}, fmt.Errorf("post_tool_use counters: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return HookResult{}, fmt.Errorf("post_tool_use commit: %w", err)
	}
	committed = true
	s.promoteToolCall(ctx, sessionID, action, path, payload, isErr)
	return HookResult{}, nil
}

// promoteToolCall writes an observation for an interesting tool outcome.
// The event row is already committed. A save failure is ignored so capture
// never fails the hook.
func (s *Service) promoteToolCall(ctx context.Context, sessionID, action, path string, payload HookPayload, isErr bool) {
	if s == nil || IsSensitivePath(path) {
		return
	}
	writes := 0
	if action == "file_write" && path != "" {
		_ = s.store.DB.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM session_events
			WHERE session_id = ? AND path = ? AND action_type = 'file_write'`,
			sessionID, path).Scan(&writes)
	}
	if !shouldPromote(isErr, action, writes, payload.ContentPreview, payload.Command) {
		return
	}
	title := promoteTitle(isErr, action, path, payload.Command)
	body := strings.TrimSpace(payload.ContentPreview)
	if body == "" {
		body = strings.TrimSpace(payload.Command)
	}
	if body == "" {
		body = title
	}
	_, _ = s.Save(ctx, SaveInput{
		SessionID: sessionID,
		Title:     title,
		Content:   "**What**: " + title + "\n**Why**: tool outcome\n**Where**: " + path + "\n**Learned**: " + body,
		Type:      promoteType(isErr, action, payload.ContentPreview, payload.Command),
		TopicKey:  "loop/" + action,
		Scope:     "project",
	})
}

// actionForTool maps a tool name to its session event action type:
// Write/Edit/Delete/Patch→file_write, Read→file_read, Shell/bash/terminal→
// command_exec, everything else→tool_use. Matching is case-insensitive substring so "Write"/"write
// file" style names all map. An explicitly supplied canonical action type
// (file_write, file_read, command_exec, tool_use, or the harness alias
// "shell") is honored even when a tool name is present, so the plugin's
// mapToolType output (e.g. "bash"→"shell") drives the stored action.
func actionForTool(toolName, actionType string) string {
	if at := canonicalAction(strings.ToLower(strings.TrimSpace(actionType))); at != "" {
		return at
	}
	t := strings.ToLower(strings.TrimSpace(toolName))
	switch {
	case strings.Contains(t, "write"), strings.Contains(t, "edit"),
		strings.Contains(t, "delete"), strings.Contains(t, "patch"), t == "strreplace":
		return "file_write"
	case strings.Contains(t, "read"):
		return "file_read"
	case strings.Contains(t, "shell"), t == "bash", strings.Contains(t, "terminal"):
		return "command_exec"
	}
	return "tool_use"
}

// canonicalAction normalizes a harness-supplied action type to its stored
// form: "shell"→command_exec, "other"→tool_use, and the four stored names
// pass through. Returns "" for anything else so the caller falls back to
// tool-name derivation.
func canonicalAction(s string) string {
	switch s {
	case "file_write", "file_read", "command_exec", "tool_use":
		return s
	case "shell":
		return "command_exec"
	case "other":
		return "tool_use"
	}
	return ""
}

// isErrorStatus reports whether a result status counts as an error for the
// errors counter.
func isErrorStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "error", "failed", "failure", "fail":
		return true
	}
	return false
}

// counterBumps returns the sessions counter columns to bump for one tool-call
// event. Column names come from this fixed allowlist only — never from caller
// input — so the UPDATE is safe to assemble.
func counterBumps(action string, isErr, sensitive bool) []string {
	var bumps []string
	switch action {
	case "file_write":
		bumps = append(bumps, "files_written")
	case "file_read":
		bumps = append(bumps, "files_read")
	case "command_exec":
		bumps = append(bumps, "commands_exec")
	}
	if isErr {
		bumps = append(bumps, "errors")
	}
	if sensitive {
		bumps = append(bumps, "sensitive_actions")
	}
	return bumps
}

// hooksMu guards the hooks* fields on Service (SetHooks / SetHookFunc read
// and write them; RunHook reads them). A package-level mutex (not on *Service)
// so a nil receiver's accessors are safe and the pattern matches the
// package-level seams elsewhere in this file's family.
var (
	hooksMu   sync.Mutex
	distillMu sync.Mutex
)
