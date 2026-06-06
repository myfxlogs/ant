package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrScheduleNotFound = errors.New("strategy schedule not found")

type StrategyScheduleRepository struct {
	db *pgxpool.Pool
}

func NewStrategyScheduleRepository(db *pgxpool.Pool) *StrategyScheduleRepository {
	return &StrategyScheduleRepository{db: db}
}
