package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/repository"
)

var ErrUnauthorized = errors.New("unauthorized")

type BacktestDatasetRepo interface {
	Create(ctx context.Context, ds *repository.BacktestDataset) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*repository.BacktestDataset, error)
	List(ctx context.Context, userID uuid.UUID, accountID *uuid.UUID, symbol *string, timeframe *string, limit int, offset int) ([]*repository.BacktestDataset, error)
	SetFrozen(ctx context.Context, id uuid.UUID, frozen bool) error
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) (bool, error)
	BatchInsertBars(ctx context.Context, bars []*repository.BacktestDatasetBar) error
	ListBars(ctx context.Context, datasetID uuid.UUID, limit int) ([]*repository.BacktestDatasetBar, error)
}

type BacktestDatasetService struct {
	repo BacktestDatasetRepo
}

func NewBacktestDatasetService(repo BacktestDatasetRepo) *BacktestDatasetService {
	return &BacktestDatasetService{repo: repo}
}

func (s *BacktestDatasetService) CreateFrozenDatasetFromKlines(
	ctx context.Context,
	userID uuid.UUID,
	accountID uuid.UUID,
	symbol string,
	timeframe string,
	from *time.Time,
	to *time.Time,
	count int,
	klines []*KlineResponse,
	cost *BacktestCostModel,
) (uuid.UUID, error) {
	var costSnapshot []byte
	if cost != nil {
		b, err := json.Marshal(cost)
		if err == nil {
			costSnapshot = b
		}
	}

	ds := &repository.BacktestDataset{
		ID:        uuid.New(),
		UserID:    userID,
		AccountID: accountID,
		Symbol:    symbol,
		Timeframe: timeframe,
		FromTime:  from,
		ToTime:    to,
		Count:     count,
		Frozen:    false,
		CostModelSnapshot: costSnapshot,
	}
	id, err := s.repo.Create(ctx, ds)
	if err != nil {
		return uuid.Nil, err
	}

	bars := make([]*repository.BacktestDatasetBar, 0, len(klines))
	for _, k := range klines {
		if k == nil {
			continue
		}
		openAt, ok := parseKlineTime(k.OpenTime)
		if !ok {
			continue
		}
		closeAt, ok := parseKlineTime(k.CloseTime)
		if !ok {
			closeAt = openAt.Add(timeframeDuration(timeframe))
		}
		bars = append(bars, &repository.BacktestDatasetBar{
			DatasetID:  id,
			Symbol:     symbol,
			Timeframe:  timeframe,
			OpenTime:   openAt,
			CloseTime:  closeAt,
			OpenPrice:  k.OpenPrice,
			HighPrice:  k.HighPrice,
			LowPrice:   k.LowPrice,
			ClosePrice: k.ClosePrice,
			TickVolume: k.Volume,
			CreatedAt:  time.Now(),
		})
	}
	if err := s.repo.BatchInsertBars(ctx, bars); err != nil {
		return uuid.Nil, err
	}

	if err := s.repo.SetFrozen(ctx, id, true); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (s *BacktestDatasetService) GetFrozenDatasetCostModel(ctx context.Context, userID uuid.UUID, datasetID uuid.UUID) (*BacktestCostModel, bool, error) {
	ds, err := s.repo.GetByID(ctx, datasetID)
	if err != nil {
		return nil, false, err
	}
	if ds.UserID != userID {
		return nil, false, ErrUnauthorized
	}
	if len(ds.CostModelSnapshot) == 0 {
		return nil, false, nil
	}
	var cost BacktestCostModel
	if err := json.Unmarshal(ds.CostModelSnapshot, &cost); err != nil {
		return nil, false, nil
	}
	return &cost, true, nil
}

func (s *BacktestDatasetService) GetFrozenDatasetKlines(ctx context.Context, userID uuid.UUID, datasetID uuid.UUID, limit int) ([]*KlineResponse, error) {
	ds, err := s.repo.GetByID(ctx, datasetID)
	if err != nil {
		return nil, err
	}
	if ds.UserID != userID {
		return nil, ErrUnauthorized
	}
	rows, err := s.repo.ListBars(ctx, datasetID, limit)
	if err != nil {
		return nil, err
	}
	res := make([]*KlineResponse, 0, len(rows))
	for _, b := range rows {
		if b == nil {
			continue
		}
		res = append(res, &KlineResponse{
			Symbol:     b.Symbol,
			Timeframe:  b.Timeframe,
			OpenTime:   b.OpenTime.UTC().Format("2006-01-02T15:04:05Z"),
			CloseTime:  b.CloseTime.UTC().Format("2006-01-02T15:04:05Z"),
			OpenPrice:  b.OpenPrice,
			HighPrice:  b.HighPrice,
			LowPrice:   b.LowPrice,
			ClosePrice: b.ClosePrice,
			Volume:     b.TickVolume,
		})
	}
	return res, nil
}

func (s *BacktestDatasetService) ListDatasets(ctx context.Context, userID uuid.UUID, accountID *uuid.UUID, symbol *string, timeframe *string, limit int, offset int) ([]*repository.BacktestDataset, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("dataset service not available")
	}
	return s.repo.List(ctx, userID, accountID, symbol, timeframe, limit, offset)
}

func (s *BacktestDatasetService) DeleteDataset(ctx context.Context, userID uuid.UUID, datasetID uuid.UUID) (bool, error) {
	if s == nil || s.repo == nil {
		return false, errors.New("dataset service not available")
	}
	ds, err := s.repo.GetByID(ctx, datasetID)
	if err != nil {
		return false, err
	}
	if ds.UserID != userID {
		return false, ErrUnauthorized
	}
	return s.repo.Delete(ctx, datasetID, userID)
}
