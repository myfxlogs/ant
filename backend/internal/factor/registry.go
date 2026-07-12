package factor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"alphaforge/internal/factor/dsl"
	"alphaforge/internal/mdgateway/adapter/mdtick"
)

// FactorSpec defines a DSL factor expression to evaluate on bar events.
type FactorSpec struct {
	Name      string // unique factor name, e.g. "trend_strength"
	Expression string // DSL expression, e.g. "ema($close, 20) / ema($close, 60) - 1"
	Symbol    string // canonical symbol, e.g. "EURUSD" (empty = all symbols)
	Timeframe string // bar period, e.g. "1m" (empty = all timeframes)
}

// FactorResult is the evaluated value of a factor for a given bar.
type FactorResult struct {
	Spec      FactorSpec
	Value     float64
	Timestamp time.Time
}

// compiledFactor holds a compiled DSL Op and its metadata.
type compiledFactor struct {
	spec FactorSpec
	op   dsl.Op
}

// FactorRegistry compiles and stores DSL factor expressions,
// keyed by symbol+timeframe for fast lookup during bar evaluation.
type FactorRegistry struct {
	mu       sync.RWMutex
	factors  map[string][]*compiledFactor // key: "symbol:timeframe" or ":all"
	allSyms  []*compiledFactor             // factors with empty symbol/timeframe
	compiler *dsl.Compiler
	log      *zap.Logger
}

// NewFactorRegistry creates a new FactorRegistry.
func NewFactorRegistry(log *zap.Logger) *FactorRegistry {
	fields := dsl.FieldIndex{Fields: map[string]int{"close": 0}}
	return &FactorRegistry{
		factors:  make(map[string][]*compiledFactor),
		compiler: dsl.NewCompiler(fields, nil),
		log:      log,
	}
}

// Register compiles and stores a factor expression.
func (r *FactorRegistry) Register(spec FactorSpec) error {
	op, err := r.compiler.Compile(spec.Expression)
	if err != nil {
		return fmt.Errorf("factor registry: compile %q: %w", spec.Name, err)
	}

	cf := &compiledFactor{spec: spec, op: op}

	r.mu.Lock()
	defer r.mu.Unlock()

	if spec.Symbol == "" && spec.Timeframe == "" {
		r.allSyms = append(r.allSyms, cf)
		// Invalidate all existing key lookups since allSyms changed.
		return nil
	}

	key := spec.Symbol + ":" + spec.Timeframe
	r.factors[key] = append(r.factors[key], cf)
	r.log.Info("factor registered",
		zap.String("name", spec.Name),
		zap.String("symbol", spec.Symbol),
		zap.String("timeframe", spec.Timeframe),
		zap.Int("warmup", op.Warmup()),
	)
	return nil
}

// Lookup returns compiled factors matching the given symbol and timeframe.
// Includes both specific matches and all-symbol factors.
func (r *FactorRegistry) Lookup(symbol, timeframe string) []*compiledFactor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := symbol + ":" + timeframe
	result := make([]*compiledFactor, 0, len(r.allSyms)+len(r.factors[key]))
	result = append(result, r.allSyms...)
	result = append(result, r.factors[key]...)
	return result
}

// FactorEvaluator reads bars from a Subscriber channel, evaluates
// registered factors, and emits results to the output channel.
type FactorEvaluator struct {
	sub       *Subscriber
	registry  *FactorRegistry
	output    chan FactorResult
	log       *zap.Logger
}

// NewFactorEvaluator creates a FactorEvaluator.
func NewFactorEvaluator(sub *Subscriber, registry *FactorRegistry, log *zap.Logger) *FactorEvaluator {
	return &FactorEvaluator{
		sub:      sub,
		registry: registry,
		output:   make(chan FactorResult, 256),
		log:      log,
	}
}

// Output returns the read-side channel for factor results.
func (e *FactorEvaluator) Output() <-chan FactorResult { return e.output }

// Start begins the evaluation loop. Blocks until ctx is cancelled.
func (e *FactorEvaluator) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case bar, ok := <-e.sub.Chan():
			if !ok {
				return
			}
			e.evaluateBar(bar)
		}
	}
}

func (e *FactorEvaluator) evaluateBar(bar *mdtick.Bar) {
	factors := e.registry.Lookup(bar.Canonical, bar.Period)
	if len(factors) == 0 {
		return
	}

	closeVal, ok := bar.Close.Float64()
	if !ok {
		return
	}

	ts := time.UnixMilli(bar.CloseTsUnixMs)
	for _, cf := range factors {
		val := cf.op.Eval(closeVal)
		select {
		case e.output <- FactorResult{Spec: cf.spec, Value: val, Timestamp: ts}:
		default:
			// Output channel full — drop result (backpressure).
		}
	}
}
