package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
	mt4pb "anttrader/mt4"
	mt5pb "anttrader/mt5"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

func (s *KlineService) saveKlinesMT4(accountID uuid.UUID, symbol, timeframe string, bars []*mt4pb.Bar) {
	ctx := context.Background()
	now := time.Now()

	var klines []*model.KlineData
	for _, bar := range bars {
		openTime := alignOpenTime(bar.GetTime().AsTime(), timeframe)
		closeTime := openTime.Add(timeframeDuration(timeframe))
		klineDate := time.Date(openTime.Year(), openTime.Month(), openTime.Day(), 0, 0, 0, 0, openTime.Location())
		klines = append(klines, &model.KlineData{
			ID:         uuid.New(),
			Symbol:     symbol,
			Timeframe:  timeframe,
			OpenTime:   openTime,
			CloseTime:  closeTime,
			KlineDate:  klineDate,
			OpenPrice:  bar.GetOpen(),
			HighPrice:  bar.GetHigh(),
			LowPrice:   bar.GetLow(),
			ClosePrice: bar.GetClose(),
			TickVolume: int64(bar.GetVolume()),
			CreatedAt:  now,
			UpdatedAt:  now,
		})
	}

	if err := s.klineRepo.BatchCreate(ctx, klines); err != nil {
		logger.Error("Failed to save klines", zap.Error(err), zap.String("symbol", symbol))
	}
}

func (s *KlineService) saveKlinesMT5(accountID uuid.UUID, symbol, timeframe string, bars []*mt5pb.Bar) {
	ctx := context.Background()
	now := time.Now()

	var klines []*model.KlineData
	for _, bar := range bars {
		openTime := alignOpenTime(bar.GetTime().AsTime(), timeframe)
		closeTime := openTime.Add(timeframeDuration(timeframe))
		klineDate := time.Date(openTime.Year(), openTime.Month(), openTime.Day(), 0, 0, 0, 0, openTime.Location())
		klines = append(klines, &model.KlineData{
			ID:         uuid.New(),
			Symbol:     symbol,
			Timeframe:  timeframe,
			OpenTime:   openTime,
			CloseTime:  closeTime,
			KlineDate:  klineDate,
			OpenPrice:  bar.GetOpenPrice(),
			HighPrice:  bar.GetHighPrice(),
			LowPrice:   bar.GetLowPrice(),
			ClosePrice: bar.GetClosePrice(),
			TickVolume: int64(bar.GetTickVolume()),
			CreatedAt:  now,
			UpdatedAt:  now,
		})
	}

	if err := s.klineRepo.BatchCreate(ctx, klines); err != nil {
		logger.Error("Failed to save klines", zap.Error(err), zap.String("symbol", symbol))
	}
}

func (s *KlineService) getAccountAndVerify(ctx context.Context, userID, accountID uuid.UUID) (*model.MTAccount, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account.UserID != userID {
		return nil, ErrAccountNotFound
	}
	if account.IsDisabled {
		return nil, ErrAccountNotFound
	}
	return account, nil
}

func (s *KlineService) parseHostPort(hostPort string) (string, int32) {
	parts := strings.Split(hostPort, ":")
	if len(parts) == 2 {
		host := parts[0]
		port, _ := strconv.ParseInt(parts[1], 10, 32)
		return host, int32(port)
	}
	return hostPort, 443
}
