package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/config"
	"anttrader/internal/interceptor"
	"anttrader/internal/repository"
	"anttrader/internal/service"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

func main() {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		panic(err)
	}

	if err := logger.Init(&logger.Config{Level: cfg.Log.Level, Format: cfg.Log.Format, Output: cfg.Log.Output}); err != nil {
		panic(err)
	}

	userIDStr := os.Getenv("API_KEY_USER_ID")
	if userIDStr == "" {
		logger.Fatal("API_KEY_USER_ID is required")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		logger.Fatal("invalid API_KEY_USER_ID", zap.Error(err))
	}

	name := os.Getenv("API_KEY_NAME")
	if name == "" {
		name = "default"
	}

	preset := strings.ToLower(strings.TrimSpace(os.Getenv("API_KEY_PRESET")))

	scopesEnv := os.Getenv("API_KEY_SCOPES")
	var scopes []string
	if scopesEnv != "" {
		for _, s := range strings.Split(scopesEnv, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				scopes = append(scopes, s)
			}
		}
	} else if preset == "sandbox" {
		scopes = []string{interceptor.ScopeMarketRead, interceptor.ScopeStrategyRun}
		if name == "default" {
			name = "sandbox"
		}
	}

	expiresAtEnv := os.Getenv("API_KEY_EXPIRES_AT")
	var expiresAt *time.Time
	if expiresAtEnv != "" {
		t, err := time.Parse(time.RFC3339, expiresAtEnv)
		if err != nil {
			logger.Fatal("invalid API_KEY_EXPIRES_AT, must be RFC3339", zap.Error(err))
		}
		expiresAt = &t
	}

	sqlxDB, err := repository.NewSQLXDB(&cfg.Database)
	if err != nil {
		logger.Fatal("failed to connect db", zap.Error(err))
	}
	defer sqlxDB.Close()

	repo := repository.NewAPIKeyRepository(sqlxDB)
	svc := service.NewAPIKeyService(repo)

	raw, id, err := svc.Create(context.Background(), userID, name, scopes, expiresAt)
	if err != nil {
		logger.Fatal("failed to create api key", zap.Error(err))
	}

	// Print raw key ONCE. Do not log it.
	fmt.Printf("API_KEY_ID=%s\n", id.String())
	fmt.Printf("API_KEY=%s\n", raw)
}
