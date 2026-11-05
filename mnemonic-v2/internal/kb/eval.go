package kb

import (
	"context"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// EvalCase is one retrieval regression fixture: a query plus the pages that
// must be retrieved for it.
type EvalCase struct {
	Query    string
	Expected []string
}

// EvalReport is the retrieval regression net result.
type EvalReport struct {
	Cases            int
	RecallAtK        float64
	MeanRankDist     float64
	TopScoreBelowOne float64
	Baseline         float64
	Failed           bool
}

// Eval runs the retrieval regression net over the fixture corpus and compares
// against the observed baseline. Failed means recall@K fell below baseline.
func Eval(ctx context.Context, st store.Store, cases []EvalCase, k int, baseline float64) (*EvalReport, error) {
	if k <= 0 {
		k = 10
	}
	rep := &EvalReport{Cases: len(cases), Baseline: baseline, Failed: false}
	if len(cases) == 0 {
		return rep, nil
	}
	ranks := []float64{}
	topScores := []float64{}
	recalls := []float64{}
	for _, c := range cases {
		res, err := Query(ctx, st, c.Query, QueryFilters{Project: "proj-a", Limit: k})
		if err != nil {
			return nil, err
		}
		found := map[string]bool{}
		best := 0.0
		topScore := 0.0
		for i, r := range res {
			found[r.RelPath] = true
			if float64(i+1) > best {
				best = float64(i + 1)
			}
			if i == 0 {
				topScore = r.Score
			}
		}
		hit := 0
		for _, e := range c.Expected {
			if found[e] {
				hit++
			}
		}
		expN := len(c.Expected)
		recalls = append(recalls, ratio(hit, expN))
		ranks = append(ranks, best)
		topScores = append(topScores, topScore)
	}
	sum := 0.0
	for _, v := range recalls {
		sum += v
	}
	rep.RecallAtK = sum / float64(len(recalls))
	sumRank := 0.0
	for _, v := range ranks {
		sumRank += v
	}
	rep.MeanRankDist = sumRank / float64(len(ranks))
	sumTop := 0.0
	for _, v := range topScores {
		sumTop += v
	}
	rep.TopScoreBelowOne = sumTop / float64(len(topScores))
	if rep.RecallAtK < baseline {
		rep.Failed = true
	}
	return rep, nil
}

func ratio(hit, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(hit) / float64(total)
}
