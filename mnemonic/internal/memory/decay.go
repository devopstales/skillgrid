package memory

import (
	"math"
	"time"
)

// DecayConfig is the query-time reinforcement decay (ADR-0018). The zero
// value passed to SetDecay becomes DefaultDecayConfig: enabled, 30-day
// half-life, immunity at importance 4 or 3 accesses.
type DecayConfig struct {
	Enabled               bool
	HalfLifeDays          float64
	ImmunityMinImportance float64
	ImmunityMinAccess     int
}

// DefaultDecayConfig is the production default: decay on.
func DefaultDecayConfig() DecayConfig {
	return DecayConfig{
		Enabled:               true,
		HalfLifeDays:          30,
		ImmunityMinImportance: 4,
		ImmunityMinAccess:     3,
	}
}

// SetDecay installs the ranker config. Enabled is stored as given so an
// explicit false stays off. Non-positive half-life and non-positive immunity
// floors fall back to the defaults (a 0 importance floor would make every
// row immune). Absent YAML is the loader's job: it calls SetDecay with
// DefaultDecayConfig, not the zero struct.
func (s *Service) SetDecay(cfg DecayConfig) {
	if s == nil {
		return
	}
	def := DefaultDecayConfig()
	if cfg.HalfLifeDays <= 0 {
		cfg.HalfLifeDays = def.HalfLifeDays
	}
	if cfg.ImmunityMinImportance <= 0 {
		cfg.ImmunityMinImportance = def.ImmunityMinImportance
	}
	if cfg.ImmunityMinAccess <= 0 {
		cfg.ImmunityMinAccess = def.ImmunityMinAccess
	}
	s.decayCfg = cfg
}

// Reinforcement is base × max(1, ln(1+usage)) × half-life term × edge_factor 1.
// Days come from LastSeenAt only; an empty or unparseable timestamp is 0 days.
// Immunity (importance or access) forces the half-life term to 1.
func Reinforcement(o Observation, now time.Time, cfg DecayConfig) float64 {
	if cfg.HalfLifeDays <= 0 {
		cfg.HalfLifeDays = 30
	}
	base := 1.0
	if o.ImportanceScore.Valid && o.ImportanceScore.Float64 > 0 {
		base = o.ImportanceScore.Float64
	}
	days := daysSinceAccess(o, now)
	half := math.Pow(0.5, days/cfg.HalfLifeDays)
	if base >= cfg.ImmunityMinImportance || o.RetrievalUsage >= cfg.ImmunityMinAccess {
		half = 1
	}
	reinforce := math.Max(1, math.Log(1+float64(o.RetrievalUsage)))
	return base * reinforce * half
}

// SignalRecency is the half-life term in (0,1] before immunity.
func SignalRecency(o Observation, now time.Time, cfg DecayConfig) float64 {
	if cfg.HalfLifeDays <= 0 {
		cfg.HalfLifeDays = 30
	}
	v := math.Pow(0.5, daysSinceAccess(o, now)/cfg.HalfLifeDays)
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// SignalDecay clamps the log reinforcement into [0,1]. 100 accesses ≈ 1.
func SignalDecay(o Observation, now time.Time, cfg DecayConfig) float64 {
	_ = now
	_ = cfg
	den := math.Log(1 + 100)
	if den == 0 {
		return 0
	}
	v := math.Log(1+float64(o.RetrievalUsage)) / den
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// SignalImportance clamps base/5 into [0,1].
func SignalImportance(o Observation) float64 {
	base := 0.0
	if o.ImportanceScore.Valid && o.ImportanceScore.Float64 > 0 {
		base = o.ImportanceScore.Float64
	}
	v := base / 5
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func daysSinceAccess(o Observation, now time.Time) float64 {
	raw := o.LastSeenAt
	if raw == "" {
		return 0
	}
	t, ok := parseSeen(raw)
	if !ok {
		return 0
	}
	days := now.Sub(t).Hours() / 24
	if days < 0 {
		return 0
	}
	return days
}

func parseSeen(raw string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02T15:04:05"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// rankByDecay keeps pinned rows first, then sorts the rest by Reinforcement
// descending. Equal factors preserve input order.
func rankByDecay(results []Observation, now time.Time, cfg DecayConfig) []Observation {
	if len(results) < 2 {
		return results
	}
	pinned := make([]Observation, 0, len(results))
	rest := make([]Observation, 0, len(results))
	for _, o := range results {
		if o.Pinned {
			pinned = append(pinned, o)
		} else {
			rest = append(rest, o)
		}
	}
	type scored struct {
		o Observation
		s float64
		i int
	}
	rows := make([]scored, len(rest))
	for i, o := range rest {
		rows[i] = scored{o: o, s: Reinforcement(o, now, cfg), i: i}
	}
	// Stable insertion by score descending, original index on ties.
	for i := 1; i < len(rows); i++ {
		j := i
		for j > 0 && (rows[j].s > rows[j-1].s || (rows[j].s == rows[j-1].s && rows[j].i < rows[j-1].i)) {
			rows[j], rows[j-1] = rows[j-1], rows[j]
			j--
		}
	}
	out := make([]Observation, 0, len(results))
	out = append(out, pinned...)
	for _, r := range rows {
		out = append(out, r.o)
	}
	return out
}
