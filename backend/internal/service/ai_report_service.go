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

// AIReport AI生成的交易分析报告
type AIReport struct {
	Summary        string   `json:"summary"`
	Strengths      []string `json:"strengths"`
	Weaknesses     []string `json:"weaknesses"`
	Suggestions    []string `json:"suggestions"`
	RiskAssessment string   `json:"risk_assessment"`
	Score          int      `json:"score"`
	GeneratedAt    string   `json:"generated_at"`
}

// AIReportService AI交易分析报告服务
type AIReportService struct {
	aiManager        *ai.Manager
	analyticsRepo    *repository.AnalyticsRepository
	accountRepo      *repository.AccountRepository
	tradeRecordRepo  *repository.TradeRecordRepository
	analyticsService *AnalyticsService
}

// NewAIReportService 创建AI报告服务
func NewAIReportService(
	aiManager *ai.Manager,
	analyticsRepo *repository.AnalyticsRepository,
	accountRepo *repository.AccountRepository,
	tradeRecordRepo *repository.TradeRecordRepository,
	analyticsService *AnalyticsService,
) *AIReportService {
	return &AIReportService{
		aiManager:        aiManager,
		analyticsRepo:    analyticsRepo,
		accountRepo:      accountRepo,
		tradeRecordRepo:  tradeRecordRepo,
		analyticsService: analyticsService,
	}
}

// GenerateReportRequest 生成报告请求
type GenerateReportRequest struct {
	AccountID string `json:"account_id" binding:"required"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// GenerateReport 生成交易分析报告
func (s *AIReportService) GenerateReport(ctx context.Context, userID, accountID uuid.UUID, startDate, endDate time.Time) (*AIReport, error) {
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

	// 获取交易统计
	tradeStats, err := s.analyticsService.GetTradeSummary(ctx, userID, accountID, startDate, endDate)
	if err != nil {
		logger.Error("Failed to get trade stats", zap.Error(err))
		return nil, err
	}

	// 获取风险指标
	riskMetrics, err := s.analyticsService.GetRiskMetrics(ctx, userID, accountID, startDate, endDate)
	if err != nil {
		logger.Error("Failed to get risk metrics", zap.Error(err))
		// 风险指标获取失败不阻塞，继续处理
		riskMetrics = &model.RiskMetrics{}
	}

	// 获取品种统计
	symbolStats, err := s.analyticsService.GetSymbolStats(ctx, userID, accountID, startDate, endDate)
	if err != nil {
		logger.Error("Failed to get symbol stats", zap.Error(err))
		// 品种统计获取失败不阻塞
		symbolStats = []*model.SymbolStats{}
	}

	// 获取最近交易记录
	recentTrades, err := s.analyticsRepo.GetTradeRecordsWithLimit(ctx, accountID, startDate, endDate, 50)
	if err != nil {
		logger.Error("Failed to get recent trades", zap.Error(err))
		recentTrades = []*model.TradeRecord{}
	}

	// 检查是否有足够的交易数据
	if tradeStats.TotalTrades == 0 {
		return nil, ErrNoTradeData
	}

	// 构建AI请求
	messages := []ai.Message{
		{Role: "system", Content: prompt.ReportSystemPrompt},
		{Role: "user", Content: prompt.BuildReportPrompt(tradeStats, riskMetrics, symbolStats, recentTrades)},
	}

	// 调用AI生成报告
	response, err := s.aiManager.Chat(ctx, messages)
	if err != nil {
		logger.Error("Failed to get AI response", zap.Error(err))
		return nil, err
	}

	// 解析AI响应
	report, err := s.parseReport(response.Content)
	if err != nil {
		logger.Error("Failed to parse AI report", zap.Error(err), zap.String("response", response.Content))
		return nil, err
	}

	// 设置生成时间
	report.GeneratedAt = time.Now().Format("2006-01-02 15:04:05")

	return report, nil
}

// GenerateReportStream 流式生成交易分析报告
func (s *AIReportService) GenerateReportStream(ctx context.Context, userID, accountID uuid.UUID, startDate, endDate time.Time) (<-chan string, error) {
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

	// 获取交易统计
	tradeStats, err := s.analyticsService.GetTradeSummary(ctx, userID, accountID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// 获取风险指标
	riskMetrics, err := s.analyticsService.GetRiskMetrics(ctx, userID, accountID, startDate, endDate)
	if err != nil {
		riskMetrics = &model.RiskMetrics{}
	}

	// 获取品种统计
	symbolStats, err := s.analyticsService.GetSymbolStats(ctx, userID, accountID, startDate, endDate)
	if err != nil {
		symbolStats = []*model.SymbolStats{}
	}

	// 获取最近交易记录
	recentTrades, err := s.analyticsRepo.GetTradeRecordsWithLimit(ctx, accountID, startDate, endDate, 50)
	if err != nil {
		recentTrades = []*model.TradeRecord{}
	}

	// 检查是否有足够的交易数据
	if tradeStats.TotalTrades == 0 {
		return nil, ErrNoTradeData
	}

	// 构建AI请求
	messages := []ai.Message{
		{Role: "system", Content: prompt.ReportSystemPrompt},
		{Role: "user", Content: prompt.BuildReportPrompt(tradeStats, riskMetrics, symbolStats, recentTrades)},
	}

	// 调用AI流式生成
	stream, err := s.aiManager.StreamChat(ctx, messages)
	if err != nil {
		return nil, err
	}

	// 转换流式输出
	outputChan := make(chan string, 100)
	go func() {
		defer close(outputChan)

		var fullResponse strings.Builder

		for chunk := range stream {
			if chunk.Error != nil {
				logger.Error("Stream error", zap.Error(chunk.Error))
				errorJSON, _ := json.Marshal(map[string]interface{}{
					"type":  "error",
					"error": chunk.Error.Error(),
				})
				outputChan <- string(errorJSON)
				return
			}

			if chunk.Done {
				// 流结束，解析完整响应
				fullStr := fullResponse.String()
				report, err := s.parseReport(fullStr)
				if err != nil {
					logger.Error("Failed to parse streamed report",
						zap.Error(err),
						zap.String("response", fullStr))
					errorJSON, _ := json.Marshal(map[string]interface{}{
						"type":  "error",
						"error": err.Error(),
					})
					outputChan <- string(errorJSON)
					return
				}

				// 设置生成时间
				report.GeneratedAt = time.Now().Format("2006-01-02 15:04:05")

				// 发送最终结果
				reportJSON, _ := json.Marshal(map[string]interface{}{
					"type":   "complete",
					"report": report,
				})
				outputChan <- string(reportJSON)

				return
			}

			// 累积响应内容
			fullResponse.WriteString(chunk.Content)

			// 发送增量内容
			chunkJSON, _ := json.Marshal(map[string]interface{}{
				"type":    "chunk",
				"content": chunk.Content,
			})
			outputChan <- string(chunkJSON)
		}
	}()

	return outputChan, nil
}

// parseReport 解析AI返回的报告
func (s *AIReportService) parseReport(content string) (*AIReport, error) {
	// 清理可能的markdown代码块标记
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var report AIReport
	if err := json.Unmarshal([]byte(content), &report); err != nil {
		return nil, err
	}

	// 验证并修正评分范围
	if report.Score < 0 {
		report.Score = 0
	}
	if report.Score > 100 {
		report.Score = 100
	}

	// 确保数组不为nil
	if report.Strengths == nil {
		report.Strengths = []string{}
	}
	if report.Weaknesses == nil {
		report.Weaknesses = []string{}
	}
	if report.Suggestions == nil {
		report.Suggestions = []string{}
	}

	return &report, nil
}

// 错误定义
var (
	ErrNoTradeData = errors.New("no trade data available for analysis")
)
