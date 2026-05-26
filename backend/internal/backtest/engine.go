package backtest

import (
	"math/rand"
	"sort"
)

// Bar represents an OHLCV bar for backtesting.
type Bar struct {
	OpenTime  int64
	CloseTime int64
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// StrategyFunc is the user-provided strategy function.
// direction: 1=long, -1=short, 0=flat. volume is in lots.
type StrategyFunc func(bar Bar, currentPosition float64) (direction int, volume float64)

// Engine runs bar-replay backtests.
type Engine struct {
	InitialCapital float64
	FillModel      *FillModel
	Bars           []Bar

	equity      []float64
	trades      []Trade
	position    float64
	entryPrice  float64
	entryTime   int64
	contractSize float64
	rng         *rand.Rand
}

// NewEngine creates a backtest engine.
func NewEngine(bars []Bar, initialCapital float64, fillModel *FillModel) *Engine {
	sorted := make([]Bar, len(bars))
	copy(sorted, bars)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].OpenTime < sorted[j].OpenTime
	})

	cs := 100000.0
	if fillModel != nil && fillModel.ContractSize > 0 {
		cs = fillModel.ContractSize
	}

	return &Engine{
		InitialCapital: initialCapital,
		FillModel:      fillModel,
		Bars:           sorted,
		equity:         make([]float64, 0, len(bars)),
		trades:         make([]Trade, 0),
		contractSize:   cs,
		rng:            rand.New(rand.NewSource(42)),
	}
}

// SetSeed sets the random seed for reproducible runs.
func (e *Engine) SetSeed(seed int64) {
	e.rng = rand.New(rand.NewSource(seed))
}

// Run executes the backtest across all bars.
func (e *Engine) Run(strategy StrategyFunc) *Metrics {
	e.equity = make([]float64, 0, len(e.Bars))
	e.trades = make([]Trade, 0)
	e.position = 0
	balance := e.InitialCapital

	for _, bar := range e.Bars {
		if bar.Close <= 0 {
			e.equity = append(e.equity, balance)
			continue
		}

		direction, volume := strategy(bar, e.position)
		unrealizedPnL := e.unrealizedPnL(bar.Close)
		currentEquity := balance + unrealizedPnL

		if e.position == 0 && direction != 0 && volume > 0 {
			fill := e.FillModel.SimulateFill(direction, volume, bar.Close, 0, e.rng)
			e.position = float64(direction) * fill.FilledVolume
			e.entryPrice = fill.GrossPrice
			e.entryTime = bar.CloseTime
			balance -= fill.TotalCost
		} else if e.position != 0 {
			currentDir := 1
			if e.position < 0 {
				currentDir = -1
			}
			if direction != 0 && direction != currentDir {
				posVolume := absFloat(e.position)
				exitFill := e.FillModel.SimulateFill(-currentDir, posVolume, bar.Close, 0, e.rng)

				// PnL = volume(lots) * contractSize * (exitPrice - entryPrice) for long
				grossPnL := posVolume * e.contractSize * (exitFill.GrossPrice - e.entryPrice)
				if currentDir < 0 {
					grossPnL = posVolume * e.contractSize * (e.entryPrice - exitFill.GrossPrice)
				}
				netPnL := grossPnL - exitFill.TotalCost

				e.trades = append(e.trades, Trade{
					EntryPrice: e.entryPrice,
					ExitPrice:  exitFill.GrossPrice,
					Volume:     posVolume,
					Direction:  currentDir,
					GrossPnL:   grossPnL,
					NetPnL:     netPnL,
					Commission: exitFill.Commission,
					Swap:       exitFill.SwapCost,
					Slippage:   exitFill.SlippageCost,
					EntryTime:  e.entryTime,
					ExitTime:   bar.CloseTime,
				})
				balance += netPnL
				e.position = 0
				e.entryPrice = 0

				if direction != 0 && volume > 0 {
					fill := e.FillModel.SimulateFill(direction, volume, bar.Close, 0, e.rng)
					e.position = float64(direction) * fill.FilledVolume
					e.entryPrice = fill.GrossPrice
					e.entryTime = bar.CloseTime
					balance -= fill.TotalCost
				}
			}
		}

		currentEquity = balance + e.unrealizedPnL(bar.Close)
		e.equity = append(e.equity, currentEquity)
	}

	// Force-close any remaining position at last bar
	if e.position != 0 && len(e.Bars) > 0 {
		lastBar := e.Bars[len(e.Bars)-1]
		dir := 1
		if e.position < 0 {
			dir = -1
		}
		posVolume := absFloat(e.position)
		exitFill := e.FillModel.SimulateFill(-dir, posVolume, lastBar.Close, 0, e.rng)

		grossPnL := posVolume * e.contractSize * (exitFill.GrossPrice - e.entryPrice)
		if dir < 0 {
			grossPnL = posVolume * e.contractSize * (e.entryPrice - exitFill.GrossPrice)
		}
		netPnL := grossPnL - exitFill.TotalCost

		e.trades = append(e.trades, Trade{
			EntryPrice: e.entryPrice,
			ExitPrice:  exitFill.GrossPrice,
			Volume:     posVolume,
			Direction:  dir,
			GrossPnL:   grossPnL,
			NetPnL:     netPnL,
			Commission: exitFill.Commission,
			Swap:       exitFill.SwapCost,
			Slippage:   exitFill.SlippageCost,
			EntryTime:  e.entryTime,
			ExitTime:   lastBar.CloseTime,
		})
		balance += netPnL
		e.position = 0
		e.equity = append(e.equity, balance)
	}

	return ComputeMetrics(e.trades, e.InitialCapital, e.equity)
}

func (e *Engine) unrealizedPnL(currentPrice float64) float64 {
	if e.position == 0 || e.entryPrice <= 0 {
		return 0
	}
	posVolume := absFloat(e.position)
	if e.position > 0 {
		return posVolume * e.contractSize * (currentPrice - e.entryPrice)
	}
	return posVolume * e.contractSize * (e.entryPrice - currentPrice)
}

// Trades returns the completed trades from the last run.
func (e *Engine) Trades() []Trade { return e.trades }

// EquityCurve returns the equity curve from the last run.
func (e *Engine) EquityCurve() []float64 { return e.equity }

func absFloat(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
