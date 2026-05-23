package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	
	"anttrader/internal/model"
	"anttrader/internal/mt4client"
	mt4pb "anttrader/mt4"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

func (s *KlineService) getKlinesMT4(ctx context.Context, account *model.MTAccount, accountID uuid.UUID, req *KlineRequest) ([]*KlineResponse, error) {
	host, port := s.parseHostPort(account.BrokerHost)
	loginInt, _ := strconv.ParseInt(account.Login, 10, 64)

	client := mt4client.NewMT4Client(s.mt4Config)
	conn, err := client.Connect(ctx, int32(loginInt), account.Password, host, port)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(context.Background(), conn.GetAccountID())

	timeframe := s.parseTimeframeMT4(req.Timeframe)
	count := int32(req.Count)
	if count <= 0 {
		count = 500
	}

	fromTime := req.From
	if fromTime == "" {
		fromTime = time.Now().AddDate(0, 0, -7).Format("2006-01-02T15:04:05")
	}

	bars, err := conn.QuoteHistory(ctx, req.Symbol, mt4pb.Timeframe(timeframe), fromTime, count)
	if err != nil {
		logger.Error("MT4 QuoteHistory error", zap.Error(err), zap.String("symbol", req.Symbol), zap.String("timeframe", req.Timeframe))
		return nil, err
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
			OpenPrice:  bar.GetOpen(),
			HighPrice:  bar.GetHigh(),
			LowPrice:   bar.GetLow(),
			ClosePrice: bar.GetClose(),
			Volume:     int64(bar.GetVolume()),
		})
	}

	go s.saveKlinesMT4(account.ID, req.Symbol, req.Timeframe, bars)

	return result, nil
}

func (s *KlineService) parseTimeframeMT4(tf string) int32 {
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
