package service

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTemplateNotFound = errors.New("template not found")
	ErrScheduleNotFound = errors.New("schedule not found")
	ErrSignalNotFound   = errors.New("signal not found")
)

type StrategySvc struct {
	pg *pgxpool.Pool
}

func NewStrategySvc(pg *pgxpool.Pool) *StrategySvc {
	return &StrategySvc{pg: pg}
}
