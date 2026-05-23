package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

func (s *KlineService) getKlinesMT5(ctx context.Context, account *model.MTAccount, accountID uuid.UUID, req *KlineRequest) ([]*KlineResponse, error) {
	err := s.connectionMgr.Connect(ctx, account)
	if err != nil {
		logger.Warn("Failed to connect MT5, trying to get existing connection", zap.Error(err))
	}

	mt5Conn, err := s.connectionMgr.GetMT5Connection(accountID)
	if err != nil {
		logger.Error("Failed to get MT5 connection", zap.Error(err))
		return s.getKlinesFromDBFallback(req)
	}

	timeframe := s.parseTimeframeMT5(req.Timeframe)
	from := req.From
	if from == "" {
		from = time.Now().AddDate(0, -1, 0).Format("2006-01-02T15:04:05")
	}

	count := int32(req.Count)
	if count <= 0 {
		count = 500
	}

	bars, err := mt5Conn.PriceHistoryEx(ctx, req.Symbol, timeframe, from, count)
	if err != nil {
		logger.Error("Failed to get price history from MT5", zap.Error(err))
		return s.getKlinesFromDBFallback(req)
	}

	var result []*KlineResponse
	for _, bar := range bars {
		openAt := alignOpenTime(bar.GetTime().AsTime(), req.Timeframe)
		closeAt := openAt.Add(timeframeDuration(req.Timeframe))
		result = append(result, &KlineResponse{
			Symbol:     req.Symbol,
			Timeframe:  req.Timeframe,
			OpenTime:   openAt.Format("2006-01-02T15:04:05Z"),
			CloseTime:  closeAt.Format("2006-01-02T15:04:05Z"),
			OpenPrice:  bar.GetOpenPrice(),
			HighPrice:  bar.GetHighPrice(),
			LowPrice:   bar.GetLowPrice(),
			ClosePrice: bar.GetClosePrice(),
			Volume:     int64(bar.GetTickVolume()),
		})
	}

	if len(result) == 0 {
		return s.getKlinesFromDBFallback(req)
	}

	go s.saveKlinesMT5(account.ID, req.Symbol, req.Timeframe, bars)

	return result, nil
}

func (s *KlineService) parseTimeframeMT5(tf string) int32 {
	switch strings.ToLower(tf) {
	case "m1", "1m":
		return 1
	case "m5", "5m":
		return 5
	case "m15", "15m":
		return 15
	case "m30", "30m":
		return 30
	case "h1", "1h":
		return 60
	case "h4", "4h":
		return 240
	case "d1", "1d":
		return 1440
	case "w1", "1w":
		return 10080
	case "mn", "mn1":
		return 43200
	default:
		return 60
	}
}
