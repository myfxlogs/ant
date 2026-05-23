package service

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
	"anttrader/internal/repository"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

type TrailingStopService struct {
	autoTradingRepo *repository.AutoTradingRepository
	accountRepo     *repository.AccountRepository
	engine          MatchingEngine
	mu              sync.RWMutex
	activeStops     map[uuid.UUID]*TrailingStopConfig
	stopChan        chan uuid.UUID
}

type TrailingStopConfig struct {
	AccountID   uuid.UUID
	Ticket      int64
	Symbol      string
	Direction   string // "buy" or "sell"
	OpenPrice   float64
	CurrentSL   float64
	CurrentTP   float64
	Volume      float64
	StopPips    float64
	TriggerPips float64 // 触发移动止损的点数
	Activated   bool    // 是否已激活
	MaxProfit   float64 // 最大盈利点数
	LastUpdate  time.Time
}

type ATRConfig struct {
	Period     int     // ATR 周期
	Multiplier float64 // ATR 倍数
}

func NewTrailingStopService(
	autoTradingRepo *repository.AutoTradingRepository,
	accountRepo *repository.AccountRepository,
	engine MatchingEngine,
) *TrailingStopService {
	return &TrailingStopService{
		autoTradingRepo: autoTradingRepo,
		accountRepo:     accountRepo,
		engine:          engine,
		activeStops:     make(map[uuid.UUID]*TrailingStopConfig),
		stopChan:        make(chan uuid.UUID, 100),
	}
}

func (s *TrailingStopService) Start() {
	go s.run()
}

func (s *TrailingStopService) Stop() {
	close(s.stopChan)
}

func (s *TrailingStopService) run() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.updateTrailingStops()
		case accountID := <-s.stopChan:
			s.removeTrailingStop(accountID)
		}
	}
}

func (s *TrailingStopService) AddTrailingStop(config *TrailingStopConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := uuid.NewSHA1(config.AccountID, []byte{byte(config.Ticket)})
	s.activeStops[key] = config

	return nil
}

func (s *TrailingStopService) RemoveTrailingStop(accountID uuid.UUID, ticket int64) {
	key := uuid.NewSHA1(accountID, []byte{byte(ticket)})
	s.stopChan <- key
}

func (s *TrailingStopService) removeTrailingStop(key uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.activeStops, key)
}

func (s *TrailingStopService) updateTrailingStops() {
	s.mu.RLock()
	stops := make([]*TrailingStopConfig, 0, len(s.activeStops))
	for _, config := range s.activeStops {
		stops = append(stops, config)
	}
	s.mu.RUnlock()

	for _, config := range stops {
		if err := s.updateSingleTrailingStop(config); err != nil {
			logger.Error("Failed to update trailing stop",
				zap.Int64("ticket", config.Ticket),
				zap.Error(err))
		}
	}
}

func (s *TrailingStopService) updateSingleTrailingStop(config *TrailingStopConfig) error {
	ctx := context.Background()

	positions, err := s.engine.GetPositions(ctx, config.AccountID, config.AccountID)
	if err != nil {
		return err
	}

	var currentPosition *PositionResponse
	for _, pos := range positions {
		if pos.Ticket == config.Ticket {
			currentPosition = pos
			break
		}
	}

	if currentPosition == nil {
		s.RemoveTrailingStop(config.AccountID, config.Ticket)
		return nil
	}

	currentPrice := currentPosition.CurrentPrice

	profitPips := s.calculateProfitPips(config.Direction, config.OpenPrice, currentPrice)

	if profitPips > config.MaxProfit {
		config.MaxProfit = profitPips
	}

	if !config.Activated && config.TriggerPips > 0 {
		if profitPips >= config.TriggerPips {
			config.Activated = true
		}
	}

	if !config.Activated {
		return nil
	}

	newSL := s.calculateNewStopLoss(config.Direction, currentPrice, config.StopPips)

	if config.Direction == "buy" {
		if newSL <= config.CurrentSL {
			return nil
		}
	} else {
		if newSL >= config.CurrentSL || config.CurrentSL == 0 {
			return nil
		}
	}

	modifyReq := &OrderModifyRequest{
		AccountID:  config.AccountID.String(),
		Ticket:     config.Ticket,
		StopLoss:   newSL,
		TakeProfit: config.CurrentTP,
	}

	_, err = s.engine.OrderModify(ctx, config.AccountID, modifyReq)
	if err != nil {
		return err
	}

	config.CurrentSL = newSL
	config.LastUpdate = time.Now()
	return nil
}

func (s *TrailingStopService) calculateProfitPips(direction string, openPrice, currentPrice float64) float64 {
	if direction == "buy" {
		return (currentPrice - openPrice) * 10000
	}
	return (openPrice - currentPrice) * 10000
}

func (s *TrailingStopService) calculateNewStopLoss(direction string, currentPrice, stopPips float64) float64 {
	pipValue := 0.0001

	if direction == "buy" {
		return currentPrice - (stopPips * pipValue)
	}
	return currentPrice + (stopPips * pipValue)
}

func (s *TrailingStopService) CalculateDynamicStopLoss(conditions map[string]interface{}, atrValue float64) float64 {
	atrMultiplier := 2.0
	if val, ok := conditions["atr_multiplier"].(float64); ok {
		atrMultiplier = val
	}

	return atrValue * atrMultiplier
}

func (s *TrailingStopService) CalculatePositionBasedStopLoss(entryPrice, accountBalance, riskPercent, pipValue float64, direction string) float64 {
	riskAmount := accountBalance * (riskPercent / 100)

	stopLossPips := riskAmount / pipValue

	pipSize := 0.0001

	if direction == "buy" {
		return entryPrice - (stopLossPips * pipSize)
	}
	return entryPrice + (stopLossPips * pipSize)
}

func (s *TrailingStopService) CalculateATRBasedStopLoss(entryPrice, atrValue, multiplier float64, direction string) float64 {
	stopDistance := atrValue * multiplier

	if direction == "buy" {
		return entryPrice - stopDistance
	}
	return entryPrice + stopDistance
}

func (s *TrailingStopService) CalculateBreakEvenStopLoss(entryPrice, breakEvenPips float64, direction string) float64 {
	pipSize := 0.0001

	if direction == "buy" {
		return entryPrice + (breakEvenPips * pipSize)
	}
	return entryPrice - (breakEvenPips * pipSize)
}

func (s *TrailingStopService) GetActiveStops() []TrailingStopConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stops := make([]TrailingStopConfig, 0, len(s.activeStops))
	for _, config := range s.activeStops {
		stops = append(stops, *config)
	}
	return stops
}

func (s *TrailingStopService) RoundToPip(price float64, pipSize float64) float64 {
	return math.Round(price/pipSize) * pipSize
}

func (s *TrailingStopService) ValidateStopLoss(entryPrice, stopLoss, minDistance float64, direction string) bool {
	distance := math.Abs(entryPrice - stopLoss)
	return distance >= minDistance
}

func (s *TrailingStopService) CreateTrailingStopFromPosition(ctx context.Context, userID, accountID uuid.UUID, position *PositionResponse, riskConfig *model.RiskConfig) error {
	if riskConfig == nil || !riskConfig.TrailingStopEnabled {
		return nil
	}

	direction := "buy"
	if position.Type == "sell" {
		direction = "sell"
	}

	config := &TrailingStopConfig{
		AccountID:   accountID,
		Ticket:      position.Ticket,
		Symbol:      position.Symbol,
		Direction:   direction,
		OpenPrice:   position.OpenPrice,
		CurrentSL:   position.StopLoss,
		CurrentTP:   position.TakeProfit,
		Volume:      position.Volume,
		StopPips:    riskConfig.TrailingStopPips,
		TriggerPips: riskConfig.TrailingStopPips * 2,
		Activated:   false,
		MaxProfit:   0,
		LastUpdate:  time.Now(),
	}

	return s.AddTrailingStop(config)
}
