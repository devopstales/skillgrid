package kb

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// frontmatter.go — the OKF page header (001-structure §5). The agent authors
// title/type/project/aliases/anchors/claims/sources; the engine owns
// status/stale_after/generated/verified (the agent must not author them).
// The frontmatter block is the KB's claims sidecar (KB-INV-1: files are truth);
// kb_claims is its derived read side.

type Stamp struct {
	By string `yaml:"by" json:"by"`
	At string `yaml:"at" json:"at"`
}

type Frontmatter struct {
	Title      string   `yaml:"title" json:"title"`
	Type       string   `yaml:"type" json:"type"`
	Project    string   `yaml:"project,omitempty" json:"project,omitempty"`
	Aliases    []string `yaml:"aliases,omitempty" json:"aliases,omitempty"`
	Anchors    []Anchor `yaml:"anchors,omitempty" json:"anchors,omitempty"`
	Claims     []Claim  `yaml:"claims,omitempty" json:"claims,omitempty"`
	Sources    Sources  `yaml:"sources,omitempty" json:"sources,omitempty"`
	Status     string   `yaml:"status,omitempty" json:"status,omitempty"`
	StaleAfter string   `yaml:"stale_after,omitempty" json:"stale_after,omitempty"`
	Generated  *Stamp   `yaml:"generated,omitempty" json:"generated,omitempty"`
	Verified   []Stamp  `yaml:"verified,omitempty" json:"verified,omitempty"`
}

// GroundedTypes are the page types that pin code (001 §4).
var GroundedTypes = map[string]bool{
	"decision": true, "invariant": true, "workflow": true,
	"invariantcontract": true, "businessworkflow": true,
}

// IsGrounded reports whether the page type is a grounded type (case-insensitive).
func IsGrounded(pageType string) bool {
	return GroundedTypes[strings.ToLower(strings.ReplaceAll(pageType, " ", ""))]
}

// Anchor is a pin target (001 §5.2 / 006 §2): a path plus exactly one selector
// form — all_of (token), line+version (span), or symbol (graph).
type Anchor struct {
	Path       string   `yaml:"path" json:"path"`
	AllOf      []string `yaml:"all_of,omitempty" json:"all_of,omitempty"`
	Line       int      `yaml:"line,omitempty" json:"line,omitempty"`
	Version    string   `yaml:"version,omitempty" json:"version,omitempty"`
	Symbol     string   `yaml:"symbol,omitempty" json:"symbol,omitempty"`
	CalledFrom []string `yaml:"called_from,omitempty" json:"called_from,omitempty"`
}

// Kind returns the anchor's selector kind: "token" | "span" | "symbol" | "".
func (a Anchor) Kind() string {
	switch {
	case len(a.AllOf) > 0:
		return "token"
	case a.Version != "" || a.Line != 0:
		return "span"
	case a.Symbol != "":
		return "symbol"
	default:
		return ""
	}
}

// Claim is a must-hold assertion (001 §5.3): a literal that must hold at the
// anchor site. Path is optional — an unpathed claim is checked against the
// file(s) of the page's anchors (006 §2).
type Claim struct {
	Subject string `yaml:"subject" json:"subject"`
	Literal string `yaml:"literal" json:"literal"`
	Path    string `yaml:"path,omitempty" json:"path,omitempty"`
}

// Source is a provenance citation (001 §5.4): an inline `kind:value` string
// (memory:<sync_id>, code:<path>#Lx-Ly) or an external entry with identity
// fields (neuledge model).
type Source struct {
	Inline           string `yaml:"-" json:"inline,omitempty"`
	External         string `yaml:"external,omitempty" json:"external,omitempty"`
	Version          string `yaml:"version,omitempty" json:"version,omitempty"`
	SourceURL        string `yaml:"source_url,omitempty" json:"source_url,omitempty"`
	SourceCommit     string `yaml:"source_commit,omitempty" json:"source_commit,omitempty"`
	BuildFingerprint string `yaml:"build_fingerprint,omitempty" json:"build_fingerprint,omitempty"`
}

// MemorySyncID returns the sync_id for a `memory:<sync_id>` citation.
func (s Source) MemorySyncID() string {
	return strings.TrimPrefix(s.Inline, "memory:")
}

type Sources []Source

func (s *Sources) UnmarshalYAML(node *yaml.Node) error {
	out := Sources{}
	for _, item := range node.Content {
		switch item.Kind {
		case yaml.ScalarNode:
			out = append(out, Source{Inline: item.Value})
		case yaml.MappingNode:
			var m map[string]string
			if err := item.Decode(&m); err != nil {
				return fmt.Errorf("kb: source mapping: %w", err)
			}
			out = append(out, Source{
				External: m["external"], Version: m["version"],
				SourceURL: m["source_url"], SourceCommit: m["source_commit"],
				BuildFingerprint: m["build_fingerprint"],
			})
		default:
			return fmt.Errorf("kb: source entry must be a string or mapping")
		}
	}
	*s = out
	return nil
}

func (s Sources) MarshalYAML() (interface{}, error) {
	items := make([]interface{}, 0, len(s))
	for _, e := range s {
		if e.Inline != "" {
			items = append(items, e.Inline)
			continue
		}
		m := map[string]string{"external": e.External}
		if e.Version != "" {
			m["version"] = e.Version
		}
		if e.SourceURL != "" {
			m["source_url"] = e.SourceURL
		}
		if e.SourceCommit != "" {
			m["source_commit"] = e.SourceCommit
		}
		if e.BuildFingerprint != "" {
			m["build_fingerprint"] = e.BuildFingerprint
		}
		items = append(items, m)
	}
	return items, nil
}

// EmitFrontmatter renders the `---`-delimited frontmatter block (the inverse
// of SplitFrontmatter for the engine-owned writes — cite-source). Field
// order follows the struct; a round-trip may normalize YAML formatting but
// never the values.
func EmitFrontmatter(fm Frontmatter) (string, error) {
	b, err := yaml.Marshal(fm)
	if err != nil {
		return "", fmt.Errorf("kb: emit frontmatter: %w", err)
	}
	return "---\n" + string(b) + "---\n", nil
}

// SplitFrontmatter separates the leading YAML frontmatter block from the body.
// Content without a terminated `---` block parses as zero-value frontmatter +
// the whole content as body (lint flags a missing frontmatter).
func SplitFrontmatter(content string) (Frontmatter, string, error) {
	if !strings.HasPrefix(content, "---\n") {
		return Frontmatter{}, content, nil
	}
	rest := content[len("---\n"):]
	j := strings.Index(rest, "\n---")
	if j < 0 {
		return Frontmatter{}, content, nil
	}
	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(rest[:j]), &fm); err != nil {
		return Frontmatter{}, content, fmt.Errorf("kb: frontmatter: %w", err)
	}
	body := rest[j+len("\n---"):]
	if strings.HasPrefix(body, "\n") {
		body = body[1:]
	}
	return fm, body, nil
}

// Render re-serializes the frontmatter block + body. The engine normalizes the
// block's formatting on every write it makes (byte-stability across engine
// writes, not across hand formatting).
func (f Frontmatter) Render(body string) string {
	b, err := yaml.Marshal(f)
	if err != nil {
		return body
	}
	return "---\n" + string(b) + "---\n" + body
}
