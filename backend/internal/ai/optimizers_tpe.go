package ai

import (
	"math"
	"math/rand"
)

// ── TPE (Tree-structured Parzen Estimator) ──

type TPEOptimizer struct {
	space    ResolvedSpace
	dims     int
	maxEvals int
	gamma    float64
	startup  int

	history    [][]int
	histScores []float64
	evalsUsed  int
	bestIdx    int
	bestScore  float64
	rng        *rand.Rand
}

func NewTPEOptimizer(space ResolvedSpace, maxEvals int) *TPEOptimizer {
	return &TPEOptimizer{
		space: space, dims: len(space.Keys), maxEvals: max(8, maxEvals),
		gamma: 0.25, startup: min(8, maxEvals/3),
		rng: rand.New(rand.NewSource(rngSeed+1)), bestScore: -1e9,
	}
}

func (tp *TPEOptimizer) Ask(batchSize int) [][]int {
	remaining := tp.maxEvals - tp.evalsUsed
	if batchSize <= 0 || batchSize > remaining { batchSize = remaining }
	out := make([][]int, 0, batchSize)
	for i := 0; i < batchSize; i++ {
		var vec []int
		if len(tp.history) < tp.startup { vec = tp.randomVector() } else { vec = tp.tpeSample() }
		out = append(out, vec)
		tp.evalsUsed++
	}
	return out
}

func (tp *TPEOptimizer) Tell(results []OptimizerResult) {
	for _, r := range results {
		if len(r.Indices) != tp.dims { continue }
		tp.history = append(tp.history, r.Indices)
		tp.histScores = append(tp.histScores, r.Score)
		if r.Score > tp.bestScore { tp.bestScore = r.Score; tp.bestIdx = len(tp.history) - 1 }
	}
}

func (tp *TPEOptimizer) Best() ([]int, float64) {
	if tp.bestIdx < len(tp.history) { return tp.history[tp.bestIdx], tp.bestScore }
	return nil, 0
}

func (tp *TPEOptimizer) Evaluations() int { return tp.evalsUsed }
func (tp *TPEOptimizer) MaxEvals() int    { return tp.maxEvals }
func (tp *TPEOptimizer) Done() bool       { return tp.evalsUsed >= tp.maxEvals }

func (tp *TPEOptimizer) tpeSample() []int {
	progress := float64(tp.evalsUsed) / float64(tp.maxEvals)
	sigmaFrac := 0.25 * (1.0 - progress)
	if sigmaFrac < 0.05 { sigmaFrac = 0.05 }

	n := len(tp.history)
	k := int(float64(n) * tp.gamma)
	if k < 1 { k = 1 }
	ranked := make([]int, n)
	for i := range ranked { ranked[i] = i }
	sortByScore(ranked, tp.histScores)

	goodIdx := ranked[tp.rng.Intn(k)]
	vec := make([]int, tp.dims)
	for j := 0; j < tp.dims; j++ {
		hi := len(tp.space.ValuesByKey[tp.space.Keys[j]]) - 1
		sigma := float64(hi) * sigmaFrac
		if sigma < 0.5 { sigma = 0.5 }
		mut := int(math.Round(float64(tp.history[goodIdx][j]) + tp.rng.NormFloat64()*sigma))
		if mut < 0 { mut = 0 }
		if mut > hi { mut = hi }
		vec[j] = mut
	}
	return vec
}

func (tp *TPEOptimizer) randomVector() []int {
	vec := make([]int, tp.dims)
	for i, k := range tp.space.Keys { vec[i] = tp.rng.Intn(len(tp.space.ValuesByKey[k])) }
	return vec
}

func sortByScore(indices []int, scores []float64) {
	n := len(indices)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if scores[indices[j]] < scores[indices[j+1]] {
				indices[j], indices[j+1] = indices[j+1], indices[j]
			}
		}
	}
}
