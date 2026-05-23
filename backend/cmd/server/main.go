package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"anttrader/internal/config"
	"anttrader/internal/pkg/jwt"
	"anttrader/internal/server"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "config file path")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := logger.Init(&logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
		Output: cfg.Log.Output,
	}); err != nil {
		fmt.Printf("Failed to init logger: %v\n", err)
		os.Exit(1)
	}

	jwt.Init(
		cfg.JWT.Secret,
		cfg.JWT.AccessExpire,
		cfg.JWT.RefreshExpire,
		cfg.JWT.Issuer,
	)

	srv, err := server.New(cfg)
	if err != nil {
		logger.Fatal("Failed to create server", zap.Error(err))
	}

	go func() {
		if err := srv.Start(); err != nil {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	logger.Info("Server started",
		zap.Int("http_port", cfg.Server.HTTPPort),
		zap.String("mode", cfg.Server.Mode),
	)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}

	logger.Info("Server stopped")
}
