package config

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestPricingDefaultMatchesConfigD(t *testing.T) {
	shipped, err := os.ReadFile(filepath.Join("..", "..", "..", "config.d", "pricing.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(shipped, pricingDefault) {
		t.Fatal("internal/mnemonic/config/pricing_default.yaml drifted from config.d/pricing.yaml; copy it over")
	}
}

func TestPricingLookupAndCost(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	p := LoadPricing(t.TempDir())

	cases := []struct {
		model string
		want  float64
		ok    bool
	}{
		// 1M input × $3 + 0.5M output × $15 + 2M cache × $0.30 = 3 + 7.5 + 0.6
		{"claude-sonnet-4-5-20250929", 11.1, true},
		{"anthropic/Claude-Sonnet-4-5", 11.1, true},
		// claude-opus-4-5 is a longer key than claude-opus-4 and wins.
		{"claude-opus-4-5", 1*5 + 0.5*25 + 2*0.5, true},
		{"claude-opus-4-1", 1*15 + 0.5*75 + 2*1.5, true},
		{"made-up-model", 0, false},
		{"", 0, false},
	}
	for _, c := range cases {
		got := p.Cost(c.model, 1_000_000, 500_000, 2_000_000)
		if (got != nil) != c.ok {
			t.Errorf("%q: priced=%v, want %v", c.model, got != nil, c.ok)
			continue
		}
		if got != nil && math.Abs(*got-c.want) > 1e-9 {
			t.Errorf("%q: cost=%v, want %v", c.model, *got, c.want)
		}
	}
}

func TestPricingRepoOverride(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "config.d", "pricing.yaml"),
		[]byte("models:\n  in-house-llm: { input: 1, output: 2 }\n  gpt-5: { input: 10, output: 10 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(repo, "pkg", "x")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	p := LoadPricing(sub)
	if c := p.Cost("in-house-llm", 1_000_000, 1_000_000, 0); c == nil || *c != 3 {
		t.Errorf("repo model cost = %v, want 3", c)
	}
	if c := p.Cost("gpt-5", 1_000_000, 0, 0); c == nil || *c != 10 {
		t.Errorf("override gpt-5 = %v, want 10", c)
	}
	if _, ok := p.Lookup("claude-haiku-4-5"); !ok {
		t.Error("built-in models must survive a partial override")
	}
}
