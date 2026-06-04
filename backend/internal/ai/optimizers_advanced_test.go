package ai

import (
	"math"
	"testing"
)

// TestDEOptimization verifies DE converges toward the optimum.
func TestDEOptimization(t *testing.T) {
	// 3 params: x∈[0,10], y∈[0,10], z∈[0,10], step=1
	params := []TunableParam{
		{Name: "x", Type: "int", Default: 5, Min: 0, Max: 10, Step: 1},
		{Name: "y", Type: "int", Default: 5, Min: 0, Max: 10, Step: 1},
		{Name: "z", Type: "int", Default: 5, Min: 0, Max: 10, Step: 1},
	}
	space := NormalizeSpace(params)
	de := NewDEOptimizer(space, 60) // 60 evals for 11³=1331 space for 11³=1331 space

	// Objective: maximize -(x-8)²-(y-3)²-(z-6)² → optimum at (8,3,6)
	objective := func(overrides map[string]interface{}) float64 {
		x := overrides["x"].(float64)
		y := overrides["y"].(float64)
		z := overrides["z"].(float64)
		return 100.0 - (x-8)*(x-8) - (y-3)*(y-3) - (z-6)*(z-6)
	}

	for !de.Done() {
		batch := de.Ask(5)
		var results []OptimizerResult
		for _, idx := range batch {
			overrides := IndexToOverrides(idx, space)
			score := objective(overrides)
			results = append(results, OptimizerResult{Indices: idx, Score: score})
		}
		de.Tell(results)
	}

	bestIdx, bestScore := de.Best()
	if bestIdx == nil { t.Fatal("DE should find a best solution") }
	bestOverrides := IndexToOverrides(bestIdx, space)
	x := bestOverrides["x"].(float64)
	y := bestOverrides["y"].(float64)
	z := bestOverrides["z"].(float64)
	t.Logf("DE best: x=%.0f y=%.0f z=%.0f score=%.1f (optimum: x=8 y=3 z=6 score=100)", x, y, z, bestScore)

	// Should be close to optimum
	if math.Abs(x-8) > 5 { t.Errorf("x=%.0f too far from 8", x) }
	if math.Abs(y-3) > 5 { t.Errorf("y=%.0f too far from 3", y) }
	if math.Abs(z-6) > 5 { t.Errorf("z=%.0f too far from 6", z) }
	if bestScore < 70 { t.Errorf("score %.1f < 70 (should improve over random)", bestScore) }
}

// TestTPEOptimization verifies TPE converges toward the optimum.
func TestTPEOptimization(t *testing.T) {
	params := []TunableParam{
		{Name: "fast", Type: "int", Default: 10, Min: 5, Max: 50, Step: 5},
		{Name: "slow", Type: "int", Default: 30, Min: 20, Max: 100, Step: 10},
	}
	space := NormalizeSpace(params)
	tpe := NewTPEOptimizer(space, 30)

	// Objective: optimum at fast=20, slow=40
	objective := func(overrides map[string]interface{}) float64 {
		fast := overrides["fast"].(float64)
		slow := overrides["slow"].(float64)
		return 100.0 - math.Abs(fast-20)*2 - math.Abs(slow-40)*0.5
	}

	for !tpe.Done() {
		batch := tpe.Ask(3)
		var results []OptimizerResult
		for _, idx := range batch {
			overrides := IndexToOverrides(idx, space)
			score := objective(overrides)
			results = append(results, OptimizerResult{Indices: idx, Score: score})
		}
		tpe.Tell(results)
	}

	_, bestScore := tpe.Best()
	t.Logf("TPE best score=%.1f (30 evals, optimum=100)", bestScore)
	if bestScore < 80 { t.Errorf("TPE score %.1f < 80", bestScore) }
}

// TestDEvsGrid compares DE against exhaustive grid on a small space.
func TestDEvsGrid(t *testing.T) {
	params := []TunableParam{
		{Name: "p", Type: "int", Default: 5, Min: 0, Max: 9, Step: 1},
		{Name: "q", Type: "int", Default: 5, Min: 0, Max: 9, Step: 1},
	}
	space := NormalizeSpace(params)

	// Exhaustive grid for ground truth
	grid := GridSearch(params, 100)
	bestGrid := 0.0
	for _, c := range grid {
		v := 100.0 - math.Abs(c["p"].(float64)-7)*5 - math.Abs(c["q"].(float64)-4)*5
		if v > bestGrid { bestGrid = v }
	}

	// DE with same budget
	de := NewDEOptimizer(space, 40)
	objective := func(overrides map[string]interface{}) float64 {
		return 100.0 - math.Abs(overrides["p"].(float64)-7)*5 - math.Abs(overrides["q"].(float64)-4)*5
	}
	for !de.Done() {
		batch := de.Ask(3)
		var results []OptimizerResult
		for _, idx := range batch {
			results = append(results, OptimizerResult{Indices: idx, Score: objective(IndexToOverrides(idx, space))})
		}
		de.Tell(results)
	}

	_, deBest := de.Best()
	t.Logf("Grid exhaustive best=%.1f, DE best=%.1f (20 evals, 100 combos)", bestGrid, deBest)
	if deBest < bestGrid-30 { t.Errorf("DE %.1f too far below grid %.1f", deBest, bestGrid) }
}
