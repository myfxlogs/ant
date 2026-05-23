package app

import (
	"context"
	"time"

	"go.uber.org/zap"

	"anttrader/internal/cache"
	"anttrader/internal/config"
	"anttrader/internal/connection"
	"anttrader/internal/monitoring"
	"anttrader/internal/service"
	"anttrader/internal/stream"
	"anttrader/pkg/logger"
)

type Lifecycle struct {
	cfg             *config.Config
	connMgr         *connection.ConnectionManager
	streamMgr       *stream.Manager
	backtestWorker  *service.BacktestRunWorker
	scheduleRunner  *service.StrategyScheduleRunner
	cacheService    *cache.CacheService
	systemCollector *monitoring.SystemMetricsCollector
}

func NewLifecycle(cfg *config.Config, c *Container) *Lifecycle {
	return &Lifecycle{cfg: cfg, connMgr: c.ConnMgr, streamMgr: c.StreamMgr, backtestWorker: c.BacktestWorker, scheduleRunner: c.ScheduleRunner, cacheService: c.CacheService, systemCollector: c.SystemCollector}
}

func (l *Lifecycle) Start(ctx context.Context) {
	if l.systemCollector != nil {
		l.systemCollector.Start()
	}
	if l.backtestWorker != nil {
		go func() {
			if err := l.backtestWorker.Start(ctx); err != nil {
				logger.Warn("Backtest run worker stopped", zap.Error(err))
			}
		}()
	}
	if l.cacheService != nil {
		go l.cleanupCache(ctx)
	}
	if l.connMgr != nil {
		go func() {
			if err := l.connMgr.Start(); err != nil {
				logger.Warn("Failed to start connection manager", zap.Error(err))
			}
		}()
	}
	if l.streamMgr != nil {
		l.streamMgr.Start()
	}
	if l.scheduleRunner != nil {
		l.scheduleRunner.Start()
	}
}

func (l *Lifecycle) Stop() {
	if l.systemCollector != nil {
		l.systemCollector.Stop()
	}
	if l.connMgr != nil {
		l.connMgr.Stop()
	}
	if l.scheduleRunner != nil {
		l.scheduleRunner.Stop()
	}
}

func (l *Lifecycle) cleanupCache(ctx context.Context) {
	ticker := time.NewTicker(l.cfg.Cache.CleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanupCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			l.cacheService.Cleanup(cleanupCtx)
			cancel()
		}
	}
}
