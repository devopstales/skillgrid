package config

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// pricingDefault is a copy of skillgrid-cli/config.d/pricing.yaml so cost
// tracking works in any repo; TestPricingDefaultMatchesConfigD keeps them equal.
//
//go:embed pricing_default.yaml
var pricingDefault []byte

// ModelPrice is USD per million tokens.
type ModelPrice struct {
	Input  float64 `yaml:"input"`
	Output float64 `yaml:"output"`
	Cache  float64 `yaml:"cache"`
}

// Pricing maps a lowercase model id (or id prefix) to its price.
type Pricing map[string]ModelPrice

type pricingFile struct {
	Models map[string]ModelPrice `yaml:"models"`
}

// LoadPricing merges, lowest to highest: the built-in table, the machine file
// (~/.skillgrid/config.d/pricing.yaml), and the first config.d/pricing.yaml
// found walking up from startDir. Later files replace individual models.
func LoadPricing(startDir string) Pricing {
	p := Pricing{}
	p.merge(pricingDefault)
	if home, err := os.UserHomeDir(); err == nil {
		if data, err := os.ReadFile(filepath.Join(home, ".skillgrid", "config.d", "pricing.yaml")); err == nil {
			p.merge(data)
		}
	}
	if path, ok := findConfigD(startDir, "pricing.yaml"); ok {
		if data, err := os.ReadFile(path); err == nil {
			p.merge(data)
		}
	}
	return p
}

func (p Pricing) merge(data []byte) {
	var f pricingFile
	if yaml.Unmarshal(data, &f) != nil {
		return
	}
	for k, v := range f.Models {
		p[strings.ToLower(strings.TrimSpace(k))] = v
	}
}

// Lookup returns the price for a reported model id: provider prefix dropped,
// case-insensitive, longest matching key prefix wins.
func (p Pricing) Lookup(model string) (ModelPrice, bool) {
	m := strings.ToLower(strings.TrimSpace(model))
	if i := strings.LastIndex(m, "/"); i >= 0 {
		m = m[i+1:]
	}
	if m == "" {
		return ModelPrice{}, false
	}
	if v, ok := p[m]; ok {
		return v, true
	}
	best := ""
	for k := range p {
		if strings.HasPrefix(m, k) && len(k) > len(best) {
			best = k
		}
	}
	if best == "" {
		return ModelPrice{}, false
	}
	return p[best], true
}

// Cost prices a token count; nil when the model has no price.
func (p Pricing) Cost(model string, input, output, cache int64) *float64 {
	price, ok := p.Lookup(model)
	if !ok {
		return nil
	}
	c := (float64(input)*price.Input + float64(output)*price.Output + float64(cache)*price.Cache) / 1_000_000
	return &c
}

func findConfigD(startDir, name string) (string, bool) {
	if strings.TrimSpace(startDir) == "" {
		return "", false
	}
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", false
	}
	for {
		candidate := filepath.Join(dir, "config.d", name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
