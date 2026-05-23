package service

import (
	"context"
	"math"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"anttrader/internal/model"
	"anttrader/internal/repository"
)

type TradeRecord = repository.TradeRecord

type AnalyticsService struct {
	accountRepo   *repository.AccountRepository
	tradeLogRepo  *repository.TradeLogRepository
	analyticsRepo *repository.AnalyticsRepository
}

func NewAnalyticsService(
	accountRepo *repository.AccountRepository,
	tradeLogRepo *repository.TradeLogRepository,
	analyticsRepo *repository.AnalyticsRepository,
) *AnalyticsService {
	return &AnalyticsService{
		accountRepo:   accountRepo,
		tradeLogRepo:  tradeLogRepo,
		analyticsRepo: analyticsRepo,
	}
}

type AnalyticsQuery struct {
	AccountID string `json:"account_id"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

func (s *AnalyticsService) verifyAccountAccess(ctx context.Context, userID, accountID uuid.UUID) (*model.MTAccount, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account.UserID != userID {
		return nil, ErrAccountNotFound
	}
	return account, nil
}

func (s *AnalyticsService) mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (s *AnalyticsService) stdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := s.mean(values)
	variance := 0.0
	for _, v := range values {
		variance += math.Pow(v-mean, 2)
	}
	variance /= float64(len(values) - 1)
	return math.Sqrt(variance)
}

func (s *AnalyticsService) filterNegative(values []float64) []float64 {
	var result []float64
	for _, v := range values {
		if v < 0 {
			result = append(result, v)
		}
	}
	return result
}

func (s *AnalyticsService) percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	index := (p / 100) * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	if lower == upper {
		return sorted[lower]
	}
	fraction := index - float64(lower)
	return sorted[lower] + fraction*(sorted[upper]-sorted[lower])
}

func (s *AnalyticsService) expectedShortfall(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	cutoff := int(math.Ceil(float64(len(sorted)) * p / 100))
	if cutoff == 0 {
		cutoff = 1
	}
	sum := 0.0
	for i := 0; i < cutoff && i < len(sorted); i++ {
		sum += sorted[i]
	}
	return sum / float64(cutoff)
}

func (s *AnalyticsService) formatDuration(seconds float64) string {
	if seconds < 60 {
		return "< 1分钟"
	} else if seconds < 3600 {
		minutes := int(seconds / 60)
		return strconv.Itoa(minutes) + "分钟"
	} else if seconds < 86400 {
		hours := int(seconds / 3600)
		return strconv.Itoa(hours) + "小时"
	} else {
		days := int(seconds / 86400)
		return strconv.Itoa(days) + "天"
	}
}

func isBalanceTradeRecord(record *TradeRecord) bool {
	return record.OrderType == "BALANCE" || record.OrderType == "CREDIT" ||
		record.OrderType == "balance" || record.OrderType == "credit"
}

func bonusSymbolKey(r *TradeRecord) string {
	s := strings.TrimSpace(r.Symbol)
	if s == "" {
		return "—"
	}
	return s
}

func tradeHoldSeconds(r *TradeRecord) float64 {
	if r.CloseTime.IsZero() || r.OpenTime.IsZero() {
		return 0
	}
	d := r.CloseTime.Sub(r.OpenTime).Seconds()
	if d < 0 {
		return 0
	}
	return d
}

func isBuySideRecord(r *TradeRecord) bool {
	ot := strings.ToLower(strings.TrimSpace(r.OrderType))
	return strings.HasPrefix(ot, "buy")
}

func isSellSideRecord(r *TradeRecord) bool {
	ot := strings.ToLower(strings.TrimSpace(r.OrderType))
	return strings.HasPrefix(ot, "sell")
}
