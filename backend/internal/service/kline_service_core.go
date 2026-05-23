package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/config"
	"anttrader/internal/connection"
	"anttrader/internal/model"
	"anttrader/internal/repository"
)

type KlineAccountRepo interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.MTAccount, error)
}

type KlineRepo interface {
	BatchCreate(ctx context.Context, klines []*model.KlineData) error
	GetBySymbolAndTimeframe(ctx context.Context, symbol, timeframe string, from, to time.Time, limit int) ([]*model.KlineData, error)
}

type KlineService struct {
	accountRepo   KlineAccountRepo
	klineRepo     KlineRepo
	connectionMgr *connection.ConnectionManager
	mt4Config     *config.MT4Config
	mt5Config     *config.MT5Config

	fetchRemote func(ctx context.Context, account *model.MTAccount, accountID uuid.UUID, req *KlineRequest) ([]*KlineResponse, error)
}

type KlineRequest struct {
	AccountID string `json:"account_id" binding:"required"`
	Symbol    string `json:"symbol" binding:"required"`
	Timeframe string `json:"timeframe" binding:"required"`
	From      string `json:"from"`
	To        string `json:"to"`
	Count     int    `json:"count"`
}

type KlineResponse struct {
	Symbol     string  `json:"symbol"`
	Timeframe  string  `json:"timeframe"`
	OpenTime   string  `json:"open_time"`
	CloseTime  string  `json:"close_time"`
	OpenPrice  float64 `json:"open"`
	HighPrice  float64 `json:"high"`
	LowPrice   float64 `json:"low"`
	ClosePrice float64 `json:"close"`
	Volume     int64   `json:"volume"`
}

func (s *KlineService) SetRemoteFetcherForTest(fetcher func(ctx context.Context, account *model.MTAccount, accountID uuid.UUID, req *KlineRequest) ([]*KlineResponse, error)) {
	if s == nil {
		return
	}
	s.fetchRemote = fetcher
}

func NewKlineService(
	accountRepo *repository.AccountRepository,
	klineRepo *repository.KlineRepository,
	connectionMgr *connection.ConnectionManager,
	mt4Config *config.MT4Config,
	mt5Config *config.MT5Config,
) *KlineService {
	return NewKlineServiceWithDeps(accountRepo, klineRepo, connectionMgr, mt4Config, mt5Config)
}

func NewKlineServiceWithDeps(
	accountRepo KlineAccountRepo,
	klineRepo KlineRepo,
	connectionMgr *connection.ConnectionManager,
	mt4Config *config.MT4Config,
	mt5Config *config.MT5Config,
) *KlineService {
	return &KlineService{
		accountRepo:   accountRepo,
		klineRepo:     klineRepo,
		connectionMgr: connectionMgr,
		mt4Config:     mt4Config,
		mt5Config:     mt5Config,
	}
}

func (s *KlineService) GetKlines(ctx context.Context, userID, accountID uuid.UUID, req *KlineRequest) ([]*KlineResponse, error) {
	account, err := s.getAccountAndVerify(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}

	from, to, limit := computeRange(req)
	// Prefer DB-first to keep backtests stable and reduce MT load.
	if s.klineRepo != nil {
		if rows, dbErr := s.klineRepo.GetBySymbolAndTimeframe(ctx, req.Symbol, req.Timeframe, from, to, limit); dbErr == nil && len(rows) > 0 {
			resp := make([]*KlineResponse, 0, len(rows))
			for _, k := range rows {
				if k == nil {
					continue
				}
				openAt := alignOpenTime(k.OpenTime, k.Timeframe)
				closeAt := openAt.Add(timeframeDuration(k.Timeframe))
				resp = append(resp, &KlineResponse{
					Symbol:     k.Symbol,
					Timeframe:  k.Timeframe,
					OpenTime:   openAt.Format("2006-01-02T15:04:05Z"),
					CloseTime:  closeAt.Format("2006-01-02T15:04:05Z"),
					OpenPrice:  k.OpenPrice,
					HighPrice:  k.HighPrice,
					LowPrice:   k.LowPrice,
					ClosePrice: k.ClosePrice,
					Volume:     k.TickVolume,
				})
			}
			if limit > 0 && len(resp) >= limit {
				if len(resp) > limit {
					resp = resp[len(resp)-limit:]
				}
				return resp, nil
			}

			// Conditional backfill only when request is close to "now".
			if time.Since(to) <= klineBackfillWindow() && shouldAttemptRemoteBackfill(to, req.Timeframe) {
				var remote []*KlineResponse
				if s.fetchRemote != nil {
					remote, _ = s.fetchRemote(ctx, account, accountID, req)
				} else if account.MTType == "MT4" {
					remote, _ = s.getKlinesMT4(ctx, account, accountID, req)
				} else {
					remote, _ = s.getKlinesMT5(ctx, account, accountID, req)
				}
				merged := mergeKlines(resp, remote, limit)
				if len(merged) > 0 {
					return merged, nil
				}
			}

			return resp, nil
		}
	}

	// No DB or empty result: fall back to remote fetch.
	if s.fetchRemote != nil {
		return s.fetchRemote(ctx, account, accountID, req)
	}
	if account.MTType == "MT4" {
		return s.getKlinesMT4(ctx, account, accountID, req)
	}
	return s.getKlinesMT5(ctx, account, accountID, req)
}
