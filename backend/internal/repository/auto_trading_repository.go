package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrExecutionNotFound      = errors.New("execution not found")
	ErrRiskConfigNotFound     = errors.New("risk config not found")
	ErrGlobalSettingsNotFound = errors.New("global settings not found")
)

type AutoTradingRepository struct {
	db *pgxpool.Pool
}

func NewAutoTradingRepository(db *pgxpool.Pool) *AutoTradingRepository {
	return &AutoTradingRepository{db: db}
}

