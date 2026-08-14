package strategy

import (
	"go.uber.org/zap"

	"alphaforge/internal/mthub"
)

// diagBarRecv logs bar delivery + shouldRunOnBar pass/fail for diagnosis.
// DIAG: temporary — remove after bar delivery is confirmed.
func diagBarRecv(log *zap.Logger, cfg LiveStrategyConfig, bar *mthub.BarUpdate) {
	log.Info("DIAG: bar recv",
		zap.String("run", cfg.RunID.String()),
		zap.String("bsym", bar.Symbol),
		zap.String("bper", bar.Period),
		zap.Bool("bcl", bar.Closed),
		zap.String("csym", cfg.Symbol),
		zap.String("ctf", cfg.Timeframe),
		zap.Bool("pass", shouldRunOnBar(bar, cfg.Symbol, cfg.Timeframe)),
	)
}
