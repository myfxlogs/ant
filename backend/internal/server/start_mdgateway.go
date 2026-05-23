package server

import (
	"context"
	"fmt"

	"anttrader/internal/mdgateway"
	"anttrader/internal/mdgateway/chmigrate"

	"go.uber.org/zap"
)

// startMarketDataGateway initializes ClickHouse schema and starts the mdgateway runner.
func (s *Server) startMarketDataGateway(ctx context.Context, log *zap.Logger) {
	// Create CH connection for migration
	chCfg := mdgateway.CHConnConfig{
		Addr:     fmt.Sprintf("%s:%d", s.cfg.ClickHouse.Host, s.cfg.ClickHouse.Port),
		Database: s.cfg.ClickHouse.Database,
		User:     s.cfg.ClickHouse.User,
		Password: s.cfg.ClickHouse.Password,
	}

	chConn, err := mdgateway.NewCHConn(chCfg, log)
	if err != nil {
		log.Warn("start_mdgateway: CH not available, skipping chmigrate",
			zap.Error(err),
		)
		return
	}

	// 1. Run chmigrate
	conn, err := chConn.Conn(ctx)
	if err != nil {
		log.Warn("start_mdgateway: chmigrate skipped", zap.Error(err))
	} else {
		if err := chmigrate.Run(ctx, conn, log); err != nil {
			log.Error("start_mdgateway: chmigrate failed", zap.Error(err))
		} else {
			log.Info("start_mdgateway: chmigrate complete")
		}
	}

	// 2. Create and start runner
	runner, err := mdgateway.NewRunner(s.cfg, s.container.DB, log)
	if err != nil {
		log.Error("start_mdgateway: runner creation failed", zap.Error(err))
		return
	}

	if err := runner.Start(ctx); err != nil {
		log.Error("start_mdgateway: runner start failed", zap.Error(err))
	} else {
		log.Info("start_mdgateway: runner started, accounts loaded from PG")
	}
}
