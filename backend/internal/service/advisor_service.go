package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/internal/ai"
	"anttrader/internal/ai/prompt"
	"anttrader/internal/model"
	"anttrader/internal/repository"
	"anttrader/pkg/logger"
)

// TradeAdvice 交易建议结构
type TradeAdvice struct {
	Signal     string  `json:"signal"`      // buy/sell/hold
	Confidence float64 `json:"confidence"`  // 0.0-1.0
	Reason     string  `json:"reason"`      // 分析理由
	EntryPrice float64 `json:"entry_price"` // 建议入场价
	StopLoss   float64 `json:"stop_loss"`   // 建议止损价
	TakeProfit float64 `json:"take_profit"` // 建议止盈价
	RiskLevel  string  `json:"risk_level"`  // low/medium/high
}

// AdvisorService 交易建议服务
type AdvisorService struct {
	aiManager    *ai.Manager
	klineRepo    *repository.KlineRepository
	accountRepo  *repository.AccountRepository
	klineService *KlineService
}

// NewAdvisorService 创建交易建议服务
func NewAdvisorService(
	aiManager *ai.Manager,
	klineRepo *repository.KlineRepository,
	accountRepo *repository.AccountRepository,
	klineService *KlineService,
) *AdvisorService {
	return &AdvisorService{
		aiManager:    aiManager,
		klineRepo:    klineRepo,
		accountRepo:  accountRepo,
		klineService: klineService,
	}
}

// GetAdviceRequest 获取交易建议请求
type GetAdviceRequest struct {
	AccountID string `json:"account_id" binding:"required"`
	Symbol    string `json:"symbol" binding:"required"`
	Timeframe string `json:"timeframe"` // 可选，默认H1
}

// GetAdvice 获取交易建议
func (s *AdvisorService) GetAdvice(ctx context.Context, userID, accountID uuid.UUID, req *GetAdviceRequest) (*TradeAdvice, error) {
	// 验证账户权限
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		logger.Error("Failed to get account", zap.Error(err), zap.String("accountID", accountID.String()))
		return nil, ErrAccountNotFound
	}
	if account.UserID != userID {
		return nil, ErrAccountNotFound
	}
	if account.IsDisabled {
		return nil, ErrAccountNotFound
	}

	// 获取K线数据
	timeframe := req.Timeframe
	if timeframe == "" {
		timeframe = "H1"
	}

	klines, err := s.fetchKlines(ctx, accountID, req.Symbol, timeframe)
	if err != nil {
		logger.Error("Failed to fetch klines", zap.Error(err),
			zap.String("symbol", req.Symbol), zap.String("timeframe", timeframe))
		return nil, err
	}

	if len(klines) == 0 {
		return nil, ErrNoKlineData
	}

	// 格式化K线数据
	klineData := s.formatKlines(klines)

	// 构建AI请求
	messages := []ai.Message{
		{Role: "system", Content: prompt.AdvisorSystemPrompt},
		{Role: "user", Content: prompt.BuildAdvisorPrompt(req.Symbol, klineData)},
	}

	// 调用AI获取建议
	response, err := s.aiManager.Chat(ctx, messages)
	if err != nil {
		logger.Error("Failed to get AI response", zap.Error(err))
		return nil, err
	}

	// 解析AI响应
	advice, err := s.parseAdvice(response.Content)
	if err != nil {
		logger.Error("Failed to parse AI advice", zap.Error(err), zap.String("response", response.Content))
		return nil, err
	}

	return advice, nil
}

// GetAdviceStream 流式获取交易建议
func (s *AdvisorService) GetAdviceStream(ctx context.Context, userID, accountID uuid.UUID, req *GetAdviceRequest) (<-chan string, error) {
	// 验证账户权限
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, ErrAccountNotFound
	}
	if account.UserID != userID {
		return nil, ErrAccountNotFound
	}
	if account.IsDisabled {
		return nil, ErrAccountNotFound
	}

	// 获取K线数据
	timeframe := req.Timeframe
	if timeframe == "" {
		timeframe = "H1"
	}

	klines, err := s.fetchKlines(ctx, accountID, req.Symbol, timeframe)
	if err != nil {
		return nil, err
	}

	if len(klines) == 0 {
		return nil, ErrNoKlineData
	}

	// 格式化K线数据
	klineData := s.formatKlines(klines)

	// 构建AI请求
	messages := []ai.Message{
		{Role: "system", Content: prompt.AdvisorSystemPrompt},
		{Role: "user", Content: prompt.BuildAdvisorPrompt(req.Symbol, klineData)},
	}

	// 调用AI流式获取建议
	stream, err := s.aiManager.StreamChat(ctx, messages)
	if err != nil {
		return nil, err
	}

	// 转换流式输出
	outputChan := make(chan string, 100)
	go func() {
		defer close(outputChan)
		for chunk := range stream {
			if chunk.Error != nil {
				logger.Error("Stream error", zap.Error(chunk.Error))
				return
			}
			outputChan <- chunk.Content
			if chunk.Done {
				return
			}
		}
	}()

	return outputChan, nil
}

// fetchKlines 获取K线数据
func (s *AdvisorService) fetchKlines(ctx context.Context, accountID uuid.UUID, symbol, timeframe string) ([]*model.KlineData, error) {
	// 先从数据库获取最近100根K线
	to := time.Now()
	from := to.AddDate(0, 0, -7) // 最近7天

	klines, err := s.klineRepo.GetBySymbolAndTimeframe(ctx, symbol, timeframe, from, to, 100)
	if err != nil {
		return nil, err
	}

	// 如果数据库中没有数据，尝试从MT4/MT5获取
	if len(klines) == 0 {
		klineReq := &KlineRequest{
			AccountID: accountID.String(),
			Symbol:    symbol,
			Timeframe: timeframe,
			Count:     100,
		}

		_, err := s.klineService.GetKlines(ctx, uuid.Nil, accountID, klineReq)
		if err != nil {
			return nil, err
		}

		// 再次从数据库获取
		klines, err = s.klineRepo.GetBySymbolAndTimeframe(ctx, symbol, timeframe, from, to, 100)
		if err != nil {
			return nil, err
		}
	}

	return klines, nil
}

// formatKlines 格式化K线数据
func (s *AdvisorService) formatKlines(klines []*model.KlineData) string {
	var infos []prompt.KlineInfo
	for _, k := range klines {
		infos = append(infos, prompt.KlineInfo{
			Time:   k.OpenTime.Format("2006-01-02 15:04"),
			Open:   k.OpenPrice,
			High:   k.HighPrice,
			Low:    k.LowPrice,
			Close:  k.ClosePrice,
			Volume: k.TickVolume,
		})
	}
	return prompt.FormatKlineData(infos)
}

// parseAdvice 解析AI返回的建议
func (s *AdvisorService) parseAdvice(content string) (*TradeAdvice, error) {
	// 清理可能的markdown代码块标记
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var advice TradeAdvice
	if err := json.Unmarshal([]byte(content), &advice); err != nil {
		return nil, err
	}

	// 验证signal值
	advice.Signal = strings.ToLower(advice.Signal)
	if advice.Signal != "buy" && advice.Signal != "sell" && advice.Signal != "hold" {
		advice.Signal = "hold"
	}

	// 验证risk_level值
	advice.RiskLevel = strings.ToLower(advice.RiskLevel)
	if advice.RiskLevel != "low" && advice.RiskLevel != "medium" && advice.RiskLevel != "high" {
		advice.RiskLevel = "medium"
	}

	// 确保confidence在有效范围内
	if advice.Confidence < 0 {
		advice.Confidence = 0
	}
	if advice.Confidence > 1 {
		advice.Confidence = 1
	}

	return &advice, nil
}

// 错误定义
var (
	ErrNoKlineData = errors.New("no kline data available")
)
