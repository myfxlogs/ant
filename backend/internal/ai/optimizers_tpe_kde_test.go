package ai

import (
	"math"
	"testing"
)

func TestTPEOptimizer_Convergence(t *testing.T) {
	// Quadratic bowl: f(x0,x1) = -((x0-5)^2 + (x1-3)^2)
	// Minimum at (5, 3) → maximum score = 0
	params := []TunableParam{
		{Name: "p0", Type: "int", Min: 0, Max: 10, Step: 1, Default: 0},
		{Name: "p1", Type: "int", Min: 0, Max: 10, Step: 1, Default: 0},
	}
	space := NormalizeSpace(params)

	opt := NewTPEOptimizer(space, 50)
	// Startup phase: random sampling (first 10 evals)
	for i := 0; i < 10; i++ {
		batch := opt.Ask(1)
		for _, vec := range batch {
			x0 := space.ValuesByKey[space.Keys[0]][vec[0]]
			x1 := space.ValuesByKey[space.Keys[1]][vec[1]]
			score := -((x0-5)*(x0-5) + (x1-3)*(x1-3))
			opt.Tell([]OptimizerResult{{Indices: vec, Score: score}})
		}
	}
	if opt.Evaluations() != 10 {
		t.Fatalf("expected 10 evals, got %d", opt.Evaluations())
	}

	// TPE phase: should converge toward (5, 3)
	for i := 0; i < 40; i++ {
		batch := opt.Ask(1)
		for _, vec := range batch {
			x0 := space.ValuesByKey[space.Keys[0]][vec[0]]
			x1 := space.ValuesByKey[space.Keys[1]][vec[1]]
			score := -((x0-5)*(x0-5) + (x1-3)*(x1-3))
			opt.Tell([]OptimizerResult{{Indices: vec, Score: score}})
		}
	}
	if opt.Evaluations() != 50 {
		t.Fatalf("expected 50 evals, got %d", opt.Evaluations())
	}
	if !opt.Done() {
		t.Fatal("expected Done() = true")
	}

	best, score := opt.Best()
	if best == nil {
		t.Fatal("expected non-nil best")
	}
	t.Logf("best: %v, score=%.2f (target: [5 3], 0)", best, score)
	if score < -2.0 {
		t.Errorf("TPE failed to converge: best score %.2f (want >= -2.0)", score)
	}
	if best[0] < 3 || best[0] > 7 || best[1] < 1 || best[1] > 5 {
		t.Errorf("TPE best far from optimum: %v (want near [5 3])", best)
	}
}

func TestTPEOptimizer_Exploration(t *testing.T) {
	// Sinusoidal landscape: many local optima at x where sin(x) peaks
	params := []TunableParam{
		{Name: "x", Type: "int", Min: 0, Max: 50, Step: 1, Default: 0},
	}
	space := NormalizeSpace(params)
	opt := NewTPEOptimizer(space, 60)

	for !opt.Done() {
		batch := opt.Ask(1)
		for _, vec := range batch {
			x := space.ValuesByKey[space.Keys[0]][vec[0]]
			score := math.Sin(x * 0.3) // peaks at x = 5.24, 15.71, 26.18, 36.65, 47.12
			opt.Tell([]OptimizerResult{{Indices: vec, Score: score}})
		}
	}
	_, score := opt.Best()
	if score < 0.8 {
		t.Errorf("TPE sin: expected best score >= 0.8, got %.3f", score)
	}
	t.Logf("sin best score: %.3f", score)
}

func TestScottBandwidth(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5}
	bw := scottBandwidth(data)
	if bw < 0.3 {
		t.Errorf("bandwidth too small: %.3f", bw)
	}
	if bw > 3.0 {
		t.Errorf("bandwidth too large: %.3f", bw)
	}
	t.Logf("Scott bandwidth for [1,2,3,4,5]: %.3f", bw)
}

func TestKDE1D_Density(t *testing.T) {
	data := []float64{5.0, 5.0, 5.0} // all at 5
	k := newKDE1D(data, data, 10)

	// At the mode, density should be high
	d := k.densityGood(5.0)
	if d < 0.1 {
		t.Errorf("KDE density at mode too low: %.6f", d)
	}
	// Far from mode, density should be much lower
	dFar := k.densityGood(0.0)
	if dFar > d*0.5 {
		t.Errorf("KDE density at 0 should be much lower than at 5: near=%.6f far=%.6f", d, dFar)
	}
	t.Logf("KDE density at 5: %.6f, at 0: %.6f", d, dFar)
}

func TestRankByScore(t *testing.T) {
	scores := []float64{1.0, 3.0, 2.0, 5.0, 0.5}
	ranked := rankByScore(scores)
	// Expected: index 3 (5.0), 1 (3.0), 2 (2.0), 0 (1.0), 4 (0.5)
	expected := []int{3, 1, 2, 0, 4}
	for i, want := range expected {
		if ranked[i] != want {
			t.Errorf("rank[%d] = %d, want %d", i, ranked[i], want)
		}
	}
}
