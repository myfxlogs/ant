package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"anttrader/internal/config"
	"anttrader/internal/pkg/hash"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := logger.Init(&logger.Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init logger: %v\n", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), cfg.Database.DSN())
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	email := getEnv("ADMIN_EMAIL", "admin@1.com")
	password := getEnv("ADMIN_PASSWORD", "12345678")
	nickname := getEnv("ADMIN_NICKNAME", "超级管理员")
	role := getEnv("ADMIN_ROLE", "super_admin")

	passwordHash, err := hash.HashPassword(password)
	if err != nil {
		logger.Fatal("Failed to hash password", zap.Error(err))
	}

	id := uuid.New()
	query := `
		WITH existing AS (
			SELECT id FROM users WHERE lower(email) = lower($2) LIMIT 1
		), inserted AS (
			INSERT INTO users (id, email, password_hash, nickname, role, status)
			SELECT $1, $2, $3, $4, $5, 'active'
			WHERE NOT EXISTS (SELECT 1 FROM existing)
			RETURNING id
		)
		SELECT id FROM existing
		UNION ALL
		SELECT id FROM inserted
		LIMIT 1
	`

	var existingID uuid.UUID
	err = pool.QueryRow(context.Background(), query, id, email, passwordHash, nickname, role).Scan(&existingID)
	if err != nil {
		logger.Fatal("Failed to create admin user", zap.Error(err))
	}

	fmt.Printf("Admin user created/updated successfully!\n")
	fmt.Printf("Email: %s\n", email)
	fmt.Printf("Role: %s\n", role)
	if password == "Admin@123456" {
		fmt.Printf("Password: %s (DEFAULT PASSWORD - please change it!)\n", password)
	} else {
		fmt.Printf("Password: [HIDDEN - set via environment variable]\n")
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
