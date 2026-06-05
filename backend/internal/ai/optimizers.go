// optimizers.go: Grid Search + Random Search over index-encoded discrete parameter spaces.
// Adapted from QuantDinger experiment/evolution.py + optimizers.py.

package ai

import (
	"math"
	"math/rand"
)

const rngSeed = 0xC0DE

// ResolvedSpace holds the resolved discrete values for each parameter key.
type ResolvedSpace struct {
	Keys        []string
	ValuesByKey map[string][]float64
}

// NormalizeSpace converts TunableParam definitions into a resolved discrete space.
// For "int" and "float" types, values are generated via min:step:max.
// For "choice" types, values come from the Choices slice (converted to float64 indices).
func NormalizeSpace(params []TunableParam) ResolvedSpace {
	keys := make([]string, 0, len(params))
	vals := make(map[string][]float64, len(params))
	for _, p := range params {
		keys = append(keys, p.Name)
		if p.Step <= 0 || p.Min >= p.Max {
			vals[p.Name] = []float64{p.Default}
			continue
		}
		n := int(math.Floor((p.Max-p.Min)/p.Step)) + 1
		if n > 5000 {
			n = 5000 // safety cap
		}
		out := make([]float64, 0, n)
		for cursor := p.Min; cursor <= p.Max+1e-12; cursor += p.Step {
			if p.Type == "int" {
				out = append(out, math.Round(cursor))
			} else {
				out = append(out, math.Round(cursor*1e10)/1e10)
			}
		}
		vals[p.Name] = out
	}
	return ResolvedSpace{Keys: keys, ValuesByKey: vals}
}

// cartesianSize returns the total number of unique parameter combinations.
func cartesianSize(space ResolvedSpace) int {
	size := 1
	for _, k := range space.Keys {
		size *= len(space.ValuesByKey[k])
	}
	return size
}

// GridSearch generates parameter combinations from the Cartesian product with
// deterministic shuffle for fairness (avoids bias toward first parameter varying slowest).
// Truncates to maxCandidates.
func GridSearch(params []TunableParam, maxCandidates int) []map[string]interface{} {
	if len(params) == 0 {
		return nil
	}
	space := NormalizeSpace(params)
	total := cartesianSize(space)
	rng := rand.New(rand.NewSource(rngSeed))
	n := maxCandidates
	if total < n {
		n = total
	}
	// Generate random indices via mixed-radix conversion: O(candidates*dims)
	// vs old recursive buildCartesian which was O(product_of_all_dims).
	indices := make([]int, n)
	if total <= n*10 {
		all := make([]int, total)
		for i := range all { all[i] = i }
		rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })
		copy(indices, all[:n])
	} else {
		for i := 0; i < n; i++ { indices[i] = rng.Intn(total) }
	}
	out := make([]map[string]interface{}, 0, n)
	for _, idx := range indices {
		idxs := cartesianNth(space, idx)
		pm := make(map[string]interface{}, len(idxs))
		for j, k := range space.Keys { pm[k] = space.ValuesByKey[k][idxs[j]] }
		out = append(out, pm)
	}
	return out
}

func cartesianNth(space ResolvedSpace, n int) []int {
	idxs := make([]int, len(space.Keys))
	for i := len(space.Keys) - 1; i >= 0; i-- {
		size := len(space.ValuesByKey[space.Keys[i]])
		idxs[i] = n % size
		n /= size
	}
	return idxs
}

// RandomSearch generates random parameter combinations by independently sampling
// each parameter's value list. Uses fixed seed for reproducibility.
func RandomSearch(params []TunableParam, maxCandidates int) []map[string]interface{} {
	if len(params) == 0 {
		return nil
	}
	space := NormalizeSpace(params)
	rng := rand.New(rand.NewSource(rngSeed))
	out := make([]map[string]interface{}, 0, maxCandidates)
	for i := 0; i < maxCandidates; i++ {
		combo := make(map[string]interface{}, len(space.Keys))
		for _, k := range space.Keys {
			vals := space.ValuesByKey[k]
			combo[k] = vals[rng.Intn(len(vals))]
		}
		out = append(out, combo)
	}
	return out
}

// paramCombo holds an index vector and its resolved parameter map.
type paramCombo struct {
	indices []int
	params  map[string]interface{}
}

// buildCartesian recursively builds all index combinations, capped at maxSize.
func buildCartesian(space ResolvedSpace, depth int, indices []int, out *[]paramCombo, rng *rand.Rand, maxSize int) {
	if len(*out) >= maxSize {
		return
	}
	if depth == len(space.Keys) {
		idxCopy := make([]int, len(indices))
		copy(idxCopy, indices)
		pm := make(map[string]interface{}, len(indices))
		for i, k := range space.Keys {
			pm[k] = space.ValuesByKey[k][indices[i]]
		}
		*out = append(*out, paramCombo{idxCopy, pm})
		return
	}
	k := space.Keys[depth]
	for i := 0; i < len(space.ValuesByKey[k]); i++ {
		indices[depth] = i
		buildCartesian(space, depth+1, indices, out, rng, maxSize)
		if len(*out) >= maxSize {
			return
		}
	}
}
