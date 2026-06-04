package ai

import (
	"math"
	"math/rand"
)

// Optimizer is the ask/tell contract for iterative optimization.
type Optimizer interface {
	Ask(batchSize int) [][]int
	Tell(results []OptimizerResult)
	Best() ([]int, float64)
	Evaluations() int
	MaxEvals() int
	Done() bool
}

// OptimizerResult pairs a parameter index vector with its score.
type OptimizerResult struct {
	Indices []int
	Score   float64
}

// IndexToOverrides converts an index vector to a parameter override map.
func IndexToOverrides(indices []int, space ResolvedSpace) map[string]interface{} {
	out := make(map[string]interface{}, len(indices))
	for i, k := range space.Keys {
		vals := space.ValuesByKey[k]
		if indices[i] >= 0 && indices[i] < len(vals) { out[k] = vals[indices[i]] }
	}
	return out
}

// OverridesToIndex converts a parameter override map to an index vector.
func OverridesToIndex(overrides map[string]interface{}, space ResolvedSpace) []int {
	indices := make([]int, len(space.Keys))
	for i, k := range space.Keys {
		vals := space.ValuesByKey[k]
		v, ok := overrides[k]
		if !ok { indices[i] = 0; continue }
		indices[i] = findIndex(v, vals)
	}
	return indices
}

func findIndex(v interface{}, vals []float64) int {
	fv, ok := v.(float64)
	if !ok { return 0 }
	for i, val := range vals {
		if math.Abs(val-fv) < 1e-10 { return i }
	}
	return 0
}

func indicesEqual(a, b []int) bool {
	if len(a) != len(b) { return false }
	for i := range a { if a[i] != b[i] { return false } }
	return true
}

// ── Differential Evolution ──

// DEOptimizer implements rand/1/bin DE over index-encoded discrete spaces.
type DEOptimizer struct {
	space   ResolvedSpace
	dims    int
	popSize int
	maxEvals int
	F       float64
	CR      float64

	pop       [][]int
	scores    []float64
	trials     [][]int
	trialTargets []int
	evalsUsed int
	bestIdx   int
	bestScore float64
	rng       *rand.Rand
	seeded    bool
}

func NewDEOptimizer(space ResolvedSpace, maxEvals int) *DEOptimizer {
	dims := len(space.Keys)
	popSize := maxEvals / 3
	if popSize < 4 { popSize = 4 }
	if popSize > 12 { popSize = 12 }
	if popSize > maxEvals { popSize = maxEvals }
	return &DEOptimizer{
		space: space, dims: dims, popSize: popSize,
		maxEvals: max(8, maxEvals), F: 0.6, CR: 0.7,
		rng: rand.New(rand.NewSource(rngSeed)), bestScore: -1e9,
	}
}

func (d *DEOptimizer) Ask(batchSize int) [][]int {
	if d.seeded { return d.evolve(batchSize) }
	return d.seed(batchSize)
}

func (d *DEOptimizer) seed(batchSize int) [][]int {
	needed := d.popSize
	if needed > d.maxEvals-d.evalsUsed { needed = d.maxEvals - d.evalsUsed }
	if batchSize > 0 && batchSize < needed { needed = batchSize }
	out := make([][]int, 0, needed)
	for i := 0; i < needed; i++ {
		idx := d.randomVector()
		out = append(out, idx)
		d.pop = append(d.pop, idx)
		d.scores = append(d.scores, 0)
		d.evalsUsed++
	}
	d.seeded = len(d.pop) >= d.popSize
	return out
}

func (d *DEOptimizer) evolve(batchSize int) [][]int {
	remaining := d.maxEvals - d.evalsUsed
	if batchSize <= 0 || batchSize > remaining { batchSize = remaining }
	out := make([][]int, 0, batchSize)
	for i := 0; i < batchSize && d.evalsUsed < d.maxEvals; i++ {
		target := d.evalsUsed % d.popSize
		a, b, c := d.rng.Intn(d.popSize), d.rng.Intn(d.popSize), d.rng.Intn(d.popSize)
		for a == target { a = d.rng.Intn(d.popSize) }
		for b == target || b == a { b = d.rng.Intn(d.popSize) }
		for c == target || c == a || c == b { c = d.rng.Intn(d.popSize) }
		trial := make([]int, d.dims)
		R := d.rng.Intn(d.dims)
		for j := 0; j < d.dims; j++ {
			if d.rng.Float64() < d.CR || j == R {
				mut := d.pop[a][j] + int(d.F*float64(d.pop[b][j]-d.pop[c][j]))
				hi := len(d.space.ValuesByKey[d.space.Keys[j]]) - 1
				if mut < 0 { mut = -mut }
				if mut > hi { mut = 2*hi - mut }
				if mut < 0 { mut = 0 }
				if mut > hi { mut = hi }
				trial[j] = mut
			} else { trial[j] = d.pop[target][j] }
		}
		out = append(out, trial)
		d.evalsUsed++
	}
	return out
}

func (d *DEOptimizer) Tell(results []OptimizerResult) {
	if !d.seeded {
		// Seed phase: match by index equality
		for _, r := range results {
			if len(r.Indices) != d.dims { continue }
			for i := range d.pop {
				if indicesEqual(d.pop[i], r.Indices) {
					if r.Score >= d.scores[i] { d.scores[i] = r.Score }
					if r.Score > d.bestScore { d.bestScore = r.Score; d.bestIdx = i }
					break
				}
			}
		}
		return
	}
	// Evolve phase: greedy selection via trial-target pairs
	for i, r := range results {
		if len(r.Indices) != d.dims { continue }
		if i < len(d.trialTargets) {
			target := d.trialTargets[i]
			if r.Score >= d.scores[target] {
				d.pop[target] = d.trials[i]
				d.scores[target] = r.Score
			}
			if r.Score > d.bestScore {
				d.bestScore = r.Score
				d.bestIdx = target
			}
		}
	}
	d.trials = d.trials[:0]
	d.trialTargets = d.trialTargets[:0]
}

func (d *DEOptimizer) Best() ([]int, float64) {
	if d.bestIdx < len(d.pop) { return d.pop[d.bestIdx], d.bestScore }
	return nil, 0
}

func (d *DEOptimizer) Evaluations() int { return d.evalsUsed }
func (d *DEOptimizer) MaxEvals() int    { return d.maxEvals }
func (d *DEOptimizer) Done() bool       { return d.evalsUsed >= d.maxEvals }
func (d *DEOptimizer) randomVector() []int {
	vec := make([]int, d.dims)
	for i, k := range d.space.Keys { vec[i] = d.rng.Intn(len(d.space.ValuesByKey[k])) }
	return vec
}
