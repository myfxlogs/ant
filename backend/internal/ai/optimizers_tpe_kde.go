// optimizers_tpe_kde.go: Tree-structured Parzen Estimator with KDE density estimation.
//
// TPE models p(x|y) by splitting observations into two distributions:
//   l(x) = p(x | y > y*)   — good (top gamma fraction)
//   g(x) = p(x | y <= y*)  — bad  (remaining)
//
// Each dimension is modeled independently with 1D Gaussian KDE.
// Sampling draws from l(x) biased by the likelihood ratio l(x)/g(x),
// which approximates Expected Improvement.
//
// References:
//   Bergstra et al. "Algorithms for Hyper-Parameter Optimization" (2011)
//   QuantDinger experiment/tpe.py

package ai

import (
	"math"
	"math/rand"
	"sort"
)

// ── TPE Optimizer (KDE-based) ──

// TPEOptimizer implements Tree-structured Parzen Estimator for discrete
// parameter spaces. It models good vs bad configurations via 1D KDE and
// samples from the EI-proxy distribution.
type TPEOptimizer struct {
	space    ResolvedSpace
	dims     int
	maxEvals int
	gamma    float64 // fraction of observations treated as "good"
	startup  int     // random samples before TPE activates

	history    [][]int
	histScores []float64
	evalsUsed  int
	bestScore  float64
	bestVec    []int
	rng        *rand.Rand
}

func NewTPEOptimizer(space ResolvedSpace, maxEvals int) *TPEOptimizer {
	m := maxEvals
	if m < 8 {
		m = 8
	}
	st := m / 5
	if st < 4 {
		st = 4
	}
	return &TPEOptimizer{
		space:    space,
		dims:     len(space.Keys),
		maxEvals: m,
		gamma:    0.25,
		startup:  st,
		rng:      rand.New(rand.NewSource(rngSeed + 2)),
		bestScore: -1e9,
	}
}

func (t *TPEOptimizer) Ask(batchSize int) [][]int {
	rem := t.maxEvals - t.evalsUsed
	if batchSize <= 0 || batchSize > rem {
		batchSize = rem
	}
	out := make([][]int, 0, batchSize)
	for i := 0; i < batchSize; i++ {
		var vec []int
		if len(t.history) < t.startup {
			vec = t.randomVector()
		} else {
			vec = t.tpeSample()
		}
		out = append(out, vec)
		t.evalsUsed++
	}
	return out
}

func (t *TPEOptimizer) Tell(results []OptimizerResult) {
	for _, r := range results {
		if len(r.Indices) != t.dims {
			continue
		}
		t.history = append(t.history, r.Indices)
		t.histScores = append(t.histScores, r.Score)
		if r.Score > t.bestScore {
			t.bestScore = r.Score
			t.bestVec = r.Indices
		}
	}
}

func (t *TPEOptimizer) Best() ([]int, float64) { return t.bestVec, t.bestScore }
func (t *TPEOptimizer) Evaluations() int        { return t.evalsUsed }
func (t *TPEOptimizer) MaxEvals() int           { return t.maxEvals }
func (t *TPEOptimizer) Done() bool              { return t.evalsUsed >= t.maxEvals }

// ── TPE core ──

func (t *TPEOptimizer) tpeSample() []int {
	n := len(t.history)
	k := int(float64(n)*t.gamma) + 1
	if k > n {
		k = n
	}

	// Rank observations: sorted indices by score descending.
	ranked := rankByScore(t.histScores)
	goodIdx := ranked[:k] // top gamma

	// Extract per-dimension values for good and all.
	goodVals := make([][]float64, t.dims)
	allVals := make([][]float64, t.dims)
	for j := 0; j < t.dims; j++ {
		goodVals[j] = make([]float64, k)
		allVals[j] = make([]float64, n)
		for i, ri := range goodIdx {
			goodVals[j][i] = float64(t.history[ri][j])
		}
		for i := 0; i < n; i++ {
			allVals[j][i] = float64(t.history[i][j])
		}
	}

	// Build KDE for each dimension.
	kdes := make([]kde1D, t.dims)
	for j := 0; j < t.dims; j++ {
		kdes[j] = newKDE1D(allVals[j], goodVals[j], len(t.space.ValuesByKey[t.space.Keys[j]]))
	}

	// Sample: propose from l(x), accept by l(x)/g(x) ratio.
	vec := make([]int, t.dims)
	maxAttempts := 50
	for attempt := 0; attempt < maxAttempts; attempt++ {
		for j := 0; j < t.dims; j++ {
			vec[j] = kdes[j].sample(t.rng)
		}

		// Compute proxy-EI: product of l/g ratios across dimensions.
		if t.accept(kdes, vec) {
			return vec
		}
	}
	// Fallback: return the best-known vector.
	if t.bestVec != nil {
		return t.bestVec
	}
	return t.randomVector()
}

// accept decides whether to keep a candidate based on the l(x)/g(x) ratio.
// p(accept) = min(1, prod(l_j(x_j)/g_j(x_j)) / maxRatio)
func (t *TPEOptimizer) accept(kdes []kde1D, vec []int) bool {
	ratio := 1.0
	for j := 0; j < t.dims; j++ {
		x := float64(vec[j])
		l := kdes[j].densityGood(x)
		g := kdes[j].densityAll(x)
		if g < 1e-12 {
			g = 1e-12
		}
		r := l / g
		if r > 10.0 {
			r = 10.0 // clip per-dimension
		}
		ratio *= r
	}
	return t.rng.Float64() < math.Min(1.0, ratio)
}

func (t *TPEOptimizer) randomVector() []int {
	vec := make([]int, t.dims)
	for i, k := range t.space.Keys {
		vec[i] = t.rng.Intn(len(t.space.ValuesByKey[k]))
	}
	return vec
}

// ── 1D Gaussian KDE ──

type kde1D struct {
	// Good distribution samples (top gamma).
	good []float64
	// Bandwidth for good distribution (Scott's rule).
	bwGood float64
	// All samples for the g(x) background distribution.
	all []float64
	// Bandwidth for all distribution.
	bwAll float64
	// Valid index range [0, maxIdx].
	maxIdx int
}

func newKDE1D(all, good []float64, maxIdx int) kde1D {
	k := kde1D{good: good, all: all, maxIdx: maxIdx}
	k.bwGood = scottBandwidth(good)
	if k.bwGood < 0.3 {
		k.bwGood = 0.3
	}
	k.bwAll = scottBandwidth(all)
	if k.bwAll < 0.3 {
		k.bwAll = 0.3
	}
	return k
}

// scottBandwidth returns h = σ * n^(-1/5). Min 0.2 to avoid collapse.
func scottBandwidth(data []float64) float64 {
	n := len(data)
	if n < 2 {
		return 0.5
	}
	mean := 0.0
	for _, v := range data {
		mean += v
	}
	mean /= float64(n)
	variance := 0.0
	for _, v := range data {
		d := v - mean
		variance += d * d
	}
	variance /= float64(n - 1)
	std := math.Sqrt(variance)
	if std < 0.1 {
		std = 0.5
	}
	h := std * math.Pow(float64(n), -0.2)
	if h < 0.3 {
		h = 0.3
	}
	return h
}

// densityGood returns the KDE density at x considering only the good distribution.
func (k *kde1D) densityGood(x float64) float64 { return k.density(x, k.good, k.bwGood) }

// densityAll returns the KDE density at x considering all samples.
func (k *kde1D) densityAll(x float64) float64 { return k.density(x, k.all, k.bwAll) }

// density computes Gaussian KDE at x.
func (k *kde1D) density(x float64, data []float64, bw float64) float64 {
	if len(data) == 0 {
		return 1e-12
	}
	sum := 0.0
	invBw := 1.0 / bw
	norm := 1.0 / (math.Sqrt2 * math.SqrtPi)
	for _, xi := range data {
		u := (x - xi) * invBw
		sum += math.Exp(-0.5 * u * u)
	}
	return sum * norm * invBw / float64(len(data))
}

// sample draws a value from the good KDE using a random good point as mean
// and adding Gaussian noise scaled by bandwidth. Clamps to valid range.
func (k *kde1D) sample(rng *rand.Rand) int {
	mean := k.good[rng.Intn(len(k.good))]
	x := mean + rng.NormFloat64()*k.bwGood
	xi := int(math.Round(x))
	if xi < 0 {
		xi = 0
	}
	if xi >= k.maxIdx {
		xi = k.maxIdx - 1
	}
	return xi
}

// rankByScore returns indices sorted by score descending.
func rankByScore(scores []float64) []int {
	n := len(scores)
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(i, j int) bool { return scores[idx[i]] > scores[idx[j]] })
	return idx
}
