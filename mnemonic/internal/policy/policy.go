// Package policy evaluates pre-tool rules (ADR-0021): opt-in, first match
// wins, field matchers only. The HTTP API (/policy/evaluate), the harness
// hooks, and `skillgrid policy test` all go through Evaluate.
package policy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Effects a rule can carry.
const (
	Block = "block"
	Warn  = "warn"
	Guide = "guide"
	Allow = "allow"
)

// Result is the session_events.result_status a decision is recorded with.
func Result(effect string) string {
	switch effect {
	case Block:
		return "blocked"
	case Warn:
		return "warned"
	case Guide:
		return "guided"
	}
	return ""
}

// Counters a rule may threshold on (session totals before this call).
var Counters = []string{"tool_calls", "files_read", "files_written", "commands_exec", "errors", "blocked_actions"}

// List is a YAML scalar or sequence of strings; any entry may match.
type List []string

func (l *List) UnmarshalYAML(n *yaml.Node) error {
	switch n.Kind {
	case yaml.ScalarNode:
		*l = List{n.Value}
		return nil
	case yaml.SequenceNode:
		var xs []string
		if err := n.Decode(&xs); err != nil {
			return err
		}
		*l = xs
		return nil
	}
	return fmt.Errorf("line %d: want a string or a list of strings", n.Line)
}

// Match holds the field matchers. Every set field must match (AND); within a
// field any entry may match (OR). Path and command are globs: ** spans
// directories, * spans anything except '/' in paths and anything in commands,
// ? is one character.
type Match struct {
	Action   List              `yaml:"action" json:"action,omitempty"`
	Path     List              `yaml:"path" json:"path,omitempty"`
	Command  List              `yaml:"command" json:"command,omitempty"`
	Tool     List              `yaml:"tool" json:"tool,omitempty"`
	Agent    List              `yaml:"agent" json:"agent,omitempty"`
	Project  List              `yaml:"project" json:"project,omitempty"`
	Counters map[string]string `yaml:"counters" json:"counters,omitempty"`
}

// Rule is one policy entry.
type Rule struct {
	Name    string `yaml:"name" json:"name"`
	Match   Match  `yaml:"match" json:"match"`
	Effect  string `yaml:"effect" json:"effect"`
	Message string `yaml:"message" json:"message,omitempty"`
	Source  string `yaml:"-" json:"source"`

	path, command []*regexp.Regexp
	counters      []threshold
}

type threshold struct {
	name string
	op   string
	n    int
}

type file struct {
	Policy struct {
		Enabled *bool  `yaml:"enabled"`
		Rules   []Rule `yaml:"rules"`
	} `yaml:"policy"`
}

// Policy is the merged rule set: repo rules first, then machine rules.
type Policy struct {
	Enabled bool     `json:"enabled"`
	Rules   []Rule   `json:"rules"`
	Files   []string `json:"files"`
}

// Input is a pending tool call.
type Input struct {
	Action    string         `json:"action"`
	Path      string         `json:"path"`
	Command   string         `json:"command"`
	Tool      string         `json:"tool"`
	Agent     string         `json:"agent"`
	Project   string         `json:"project"`
	Directory string         `json:"directory"`
	Counters  map[string]int `json:"counters,omitempty"`
}

// Decision is the evaluator's answer. Rule is "" when nothing matched.
type Decision struct {
	Effect  string `json:"effect"`
	Message string `json:"message"`
	Rule    string `json:"rule"`
}

// Evaluate returns the first matching rule's effect, or allow.
func (p *Policy) Evaluate(in Input) Decision {
	if p == nil || !p.Enabled {
		return Decision{Effect: Allow}
	}
	for i := range p.Rules {
		r := &p.Rules[i]
		if r.matches(in) {
			return Decision{Effect: r.Effect, Message: r.Message, Rule: r.Name}
		}
	}
	return Decision{Effect: Allow}
}

func (r *Rule) matches(in Input) bool {
	m := r.Match
	if len(m.Action) > 0 && !anyEqual(m.Action, in.Action) {
		return false
	}
	if len(m.Tool) > 0 && !anyEqual(m.Tool, in.Tool) {
		return false
	}
	if len(m.Agent) > 0 && !anyEqual(m.Agent, in.Agent) {
		return false
	}
	if len(m.Project) > 0 && !anyEqual(m.Project, in.Project) {
		return false
	}
	if len(r.path) > 0 && !matchPath(r.path, in.Path, in.Directory) {
		return false
	}
	if len(r.command) > 0 && !anyRegexp(r.command, strings.TrimSpace(in.Command)) {
		return false
	}
	for _, t := range r.counters {
		if !t.holds(in.Counters[t.name]) {
			return false
		}
	}
	return true
}

func anyEqual(xs List, v string) bool {
	for _, x := range xs {
		if strings.EqualFold(strings.TrimSpace(x), strings.TrimSpace(v)) {
			return true
		}
	}
	return false
}

func anyRegexp(rs []*regexp.Regexp, v string) bool {
	for _, r := range rs {
		if r.MatchString(v) {
			return true
		}
	}
	return false
}

// matchPath tries the path as given and relative to the session directory,
// so `secrets/**` matches both "secrets/db.env" and "/repo/secrets/db.env".
func matchPath(rs []*regexp.Regexp, p, dir string) bool {
	p = strings.TrimSpace(p)
	if p == "" {
		return false
	}
	cands := []string{filepath.ToSlash(strings.TrimPrefix(p, "./"))}
	if dir != "" && filepath.IsAbs(p) {
		if rel, err := filepath.Rel(dir, p); err == nil && !strings.HasPrefix(rel, "..") {
			cands = append(cands, filepath.ToSlash(rel))
		}
	}
	for _, c := range cands {
		if anyRegexp(rs, c) {
			return true
		}
	}
	return false
}

func (t threshold) holds(v int) bool {
	switch t.op {
	case ">":
		return v > t.n
	case ">=":
		return v >= t.n
	case "<":
		return v < t.n
	case "<=":
		return v <= t.n
	case "==", "=":
		return v == t.n
	case "!=":
		return v != t.n
	}
	return false
}

var thresholdRE = regexp.MustCompile(`^\s*(>=|<=|==|!=|=|>|<)?\s*(\d+)\s*$`)

// globRegexp compiles a policy glob. pathMode keeps * inside one path
// segment; a leading **/ also matches at the root.
func globRegexp(g string, pathMode bool) (*regexp.Regexp, error) {
	g = strings.TrimSpace(g)
	if g == "" {
		return nil, errors.New("empty glob")
	}
	var b strings.Builder
	b.WriteString("^")
	rs := []rune(g)
	for i := 0; i < len(rs); i++ {
		c := rs[i]
		switch {
		case c == '*' && i+1 < len(rs) && rs[i+1] == '*':
			i++
			if i+1 < len(rs) && rs[i+1] == '/' {
				i++
				b.WriteString("(?:.*/)?")
			} else {
				b.WriteString(".*")
			}
		case c == '*':
			if pathMode {
				b.WriteString("[^/]*")
			} else {
				b.WriteString(".*")
			}
		case c == '?':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

// compile validates a rule and builds its matchers.
func (r *Rule) compile(i int) error {
	where := fmt.Sprintf("rule %d", i+1)
	if r.Name != "" {
		where += fmt.Sprintf(" (%s)", r.Name)
	} else {
		r.Name = fmt.Sprintf("rule-%d", i+1)
	}
	switch r.Effect {
	case Block, Warn, Guide, Allow:
	case "":
		return fmt.Errorf("%s: effect is required (block, warn, guide, allow)", where)
	default:
		return fmt.Errorf("%s: unknown effect %q (block, warn, guide, allow)", where, r.Effect)
	}
	for _, a := range r.Match.Action {
		switch a {
		case "file_read", "file_write", "command_exec", "tool_use":
		default:
			return fmt.Errorf("%s: unknown action %q (file_read, file_write, command_exec, tool_use)", where, a)
		}
	}
	r.path, r.command, r.counters = nil, nil, nil
	for _, g := range r.Match.Path {
		re, err := globRegexp(g, true)
		if err != nil {
			return fmt.Errorf("%s: path %q: %v", where, g, err)
		}
		r.path = append(r.path, re)
	}
	for _, g := range r.Match.Command {
		re, err := globRegexp(g, false)
		if err != nil {
			return fmt.Errorf("%s: command %q: %v", where, g, err)
		}
		r.command = append(r.command, re)
	}
	names := make([]string, 0, len(r.Match.Counters))
	for k := range r.Match.Counters {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		known := false
		for _, c := range Counters {
			known = known || c == k
		}
		if !known {
			return fmt.Errorf("%s: unknown counter %q (%s)", where, k, strings.Join(Counters, ", "))
		}
		m := thresholdRE.FindStringSubmatch(r.Match.Counters[k])
		if m == nil {
			return fmt.Errorf("%s: counter %s: want a threshold like \"> 50\", got %q", where, k, r.Match.Counters[k])
		}
		op := m[1]
		if op == "" {
			op = ">="
		}
		n, _ := strconv.Atoi(m[2])
		r.counters = append(r.counters, threshold{name: k, op: op, n: n})
	}
	if r.Effect != Allow && strings.TrimSpace(r.Message) == "" {
		return fmt.Errorf("%s: %s needs a message the agent will see", where, r.Effect)
	}
	return nil
}

// Parse reads one policy file's contents.
func Parse(data []byte, source string) (enabled *bool, rules []Rule, err error) {
	var f file
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", source, err)
	}
	for i := range f.Policy.Rules {
		if err := f.Policy.Rules[i].compile(i); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", source, err)
		}
		f.Policy.Rules[i].Source = source
	}
	return f.Policy.Enabled, f.Policy.Rules, nil
}

// RepoFile is the repo policy path for root; HomeFile is the machine policy.
func RepoFile(root string) string { return filepath.Join(root, ".skillgrid", "policy.yaml") }

func HomeFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".skillgrid", "policy.yaml")
}

// FindRepoFile walks up from dir to the nearest .skillgrid/policy.yaml.
func FindRepoFile(dir string) (string, bool) {
	if strings.TrimSpace(dir) == "" {
		return "", false
	}
	d, err := filepath.Abs(dir)
	if err != nil {
		return "", false
	}
	for {
		p := RepoFile(d)
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", false
		}
		d = parent
	}
}

// Load merges the repo file found from dir and the machine file. The repo
// file's `enabled` wins when set. A missing file is not an error; a broken
// one is, and callers fail open on it (ADR-0021).
func Load(dir string) (*Policy, error) {
	p := &Policy{}
	var repoEnabled, homeEnabled *bool
	var homeRules []Rule
	if path, ok := FindRepoFile(dir); ok {
		data, err := os.ReadFile(path)
		if err != nil {
			return &Policy{}, err
		}
		en, rules, err := Parse(data, path)
		if err != nil {
			return &Policy{}, err
		}
		repoEnabled = en
		p.Rules = append(p.Rules, rules...)
		p.Files = append(p.Files, path)
	}
	if hf := HomeFile(); hf != "" {
		if data, err := os.ReadFile(hf); err == nil {
			en, rules, err := Parse(data, hf)
			if err != nil {
				return &Policy{}, err
			}
			homeEnabled = en
			homeRules = rules
			p.Files = append(p.Files, hf)
		}
	}
	p.Rules = append(p.Rules, homeRules...)
	switch {
	case repoEnabled != nil:
		p.Enabled = *repoEnabled
	case homeEnabled != nil:
		p.Enabled = *homeEnabled
	}
	return p, nil
}

// Starter is the file `skillgrid policy init` writes: disabled, with
// examples of each effect.
const Starter = `# Skillgrid pre-tool policy (ADR-0021). Opt-in: nothing is enforced until
# enabled is true. Rules run in order; the first match wins; no match allows.
#
# match fields (all optional, every set field must match, lists mean "any of"):
#   action:   file_read | file_write | command_exec | tool_use
#   path:     glob ("secrets/**", "**/*.pem"); ** spans directories
#   command:  glob ("rm -rf *", "*curl * | sh*"); * spans anything
#   tool:     tool name (Shell, Read, write, mcp__github__create_issue)
#   agent:    cursor | opencode | kilo
#   project:  project id
#   counters: session totals before this call, e.g. commands_exec: "> 200"
# effect: block (deny the call) | warn (allow, flag it) | guide (allow, nudge)
#         | allow (stop evaluating)
#
# The hooks fail open: if skillgrid serve is down, every call is allowed.
policy:
  enabled: false
  rules:
    - name: no-secret-writes
      match:
        action: file_write
        path: ["secrets/**", "**/*.pem", "**/.env", "**/.env.*"]
      effect: block
      message: Writing secrets files is blocked by .skillgrid/policy.yaml.

    - name: no-pipe-to-shell
      match:
        action: command_exec
        command: ["*curl * | *sh*", "*wget * | *sh*"]
      effect: block
      message: Piping a download into a shell is blocked. Download, inspect, then run.

    - name: careful-rm
      match:
        action: command_exec
        command: "*rm -rf *"
      effect: warn
      message: rm -rf recorded. Double-check the path.

    - name: prefer-mnemonic-search
      match:
        tool: [WebSearch, websearch]
      effect: guide
      message: Check mem_search and web_cache_lookup before searching the web.
`
