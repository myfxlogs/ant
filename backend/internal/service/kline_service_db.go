package service

import (
	"context"
	"time"
)

func (s *KlineService) getKlinesFromDBFallback(req *KlineRequest) ([]*KlineResponse, error) {
	from := time.Now().AddDate(0, 0, -30)
	to := time.Now()
	if req.From != "" {
		from, _ = time.Parse("2006-01-02T15:04:05", req.From)
	}
	if req.To != "" {
		to, _ = time.Parse("2006-01-02T15:04:05", req.To)
	}
	count := req.Count
	if count <= 0 {
		count = 500
	}

	return s.GetKlinesFromDB(context.Background(), req.Symbol, req.Timeframe, from, to, count)
}

func (s *KlineService) GetKlinesFromDB(ctx context.Context, symbol, timeframe string, from, to time.Time, limit int) ([]*KlineResponse, error) {
	klines, err := s.klineRepo.GetBySymbolAndTimeframe(ctx, symbol, timeframe, from, to, limit)
	if err != nil {
		return nil, err
	}

	var result []*KlineResponse
	for _, k := range klines {
		openAt := alignOpenTime(k.OpenTime, k.Timeframe)
		closeAt := openAt.Add(timeframeDuration(k.Timeframe))
		result = append(result, &KlineResponse{
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
	return result, nil
}
