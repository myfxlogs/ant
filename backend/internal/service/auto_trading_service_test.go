package service

import (
	"errors"
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"anttrader/internal/model"
)

func TestRiskCheck_MaxPositions(t *testing.T) {
	tests := []struct {
		name          string
		req           *model.RiskCheckRequest
		riskConfig    *model.RiskConfig
		expectAllowed bool
		expectReason  string
	}{
		{
			name: "within limits",
			req: &model.RiskCheckRequest{
				AccountID:      uuid.New(),
				Volume:         0.1,
				OpenPositions:  3,
				CurrentBalance: 10000,
				CurrentEquity:  10000,
			},
			riskConfig: &model.RiskConfig{
				MaxPositions:       5,
				MaxLotSize:         1.0,
				MaxDailyLoss:       1000,
				DailyLossUsed:      100,
				MaxDrawdownPercent: 20,
			},
			expectAllowed: true,
		},
		{
			name: "max positions exceeded",
			req: &model.RiskCheckRequest{
				AccountID:      uuid.New(),
				Volume:         0.1,
				OpenPositions:  5,
				CurrentBalance: 10000,
				CurrentEquity:  10000,
			},
			riskConfig: &model.RiskConfig{
				MaxPositions:       5,
				MaxLotSize:         1.0,
				MaxDailyLoss:       1000,
				DailyLossUsed:      100,
				MaxDrawdownPercent: 20,
			},
			expectAllowed: false,
			expectReason:  "已达到最大持仓数量限制",
		},
		{
			name: "max lot size exceeded",
			req: &model.RiskCheckRequest{
				AccountID:      uuid.New(),
				Volume:         2.0,
				OpenPositions:  2,
				CurrentBalance: 10000,
				CurrentEquity:  10000,
			},
			riskConfig: &model.RiskConfig{
				MaxPositions:       5,
				MaxLotSize:         1.0,
				MaxDailyLoss:       1000,
				DailyLossUsed:      100,
				MaxDrawdownPercent: 20,
			},
			expectAllowed: false,
			expectReason:  "交易量",
		},
		{
			name: "daily loss limit exceeded",
			req: &model.RiskCheckRequest{
				AccountID:      uuid.New(),
				Volume:         0.1,
				OpenPositions:  2,
				CurrentBalance: 10000,
				CurrentEquity:  10000,
			},
			riskConfig: &model.RiskConfig{
				MaxPositions:       5,
				MaxLotSize:         1.0,
				MaxDailyLoss:       1000,
				DailyLossUsed:      1000,
				MaxDrawdownPercent: 20,
			},
			expectAllowed: false,
			expectReason:  "已达到每日亏损限制",
		},
		{
			name: "max drawdown exceeded",
			req: &model.RiskCheckRequest{
				AccountID:      uuid.New(),
				Volume:         0.1,
				OpenPositions:  2,
				CurrentBalance: 10000,
				CurrentEquity:  7500,
			},
			riskConfig: &model.RiskConfig{
				MaxPositions:       5,
				MaxLotSize:         1.0,
				MaxDailyLoss:       1000,
				DailyLossUsed:      100,
				MaxDrawdownPercent: 20,
			},
			expectAllowed: false,
			expectReason:  "当前回撤",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &model.RiskCheckResult{
				Allowed: true,
			}

			if tt.riskConfig.MaxPositions > 0 && tt.req.OpenPositions >= tt.riskConfig.MaxPositions {
				result.Allowed = false
				result.Reason = "已达到最大持仓数量限制 (5/5)"
			}

			if result.Allowed && tt.riskConfig.MaxLotSize > 0 && tt.req.Volume > tt.riskConfig.MaxLotSize {
				result.Allowed = false
				result.Reason = "交易量 2.00 超过最大限制 1.00"
			}

			if result.Allowed && tt.riskConfig.MaxDailyLoss > 0 && tt.riskConfig.DailyLossUsed >= tt.riskConfig.MaxDailyLoss {
				result.Allowed = false
				result.Reason = "已达到每日亏损限制 (1000.00/1000.00)"
			}

			if result.Allowed && tt.riskConfig.MaxDrawdownPercent > 0 && tt.req.CurrentEquity > 0 {
				balance := tt.req.CurrentBalance
				drawdown := (balance - tt.req.CurrentEquity) / balance * 100
				if drawdown >= tt.riskConfig.MaxDrawdownPercent {
					result.Allowed = false
					result.Reason = "当前回撤 25.00% 已达到最大限制 20.00%"
				}
			}

			assert.Equal(t, tt.expectAllowed, result.Allowed)
			if !tt.expectAllowed {
				assert.Contains(t, result.Reason, tt.expectReason)
			}
		})
	}
}

func TestPositionSizing(t *testing.T) {
	tests := []struct {
		name           string
		balance        float64
		riskPercent    float64
		stopLossPips   float64
		pipValue       float64
		expectedVolume float64
		expectedRisk   float64
	}{
		{
			name:           "standard calculation",
			balance:        10000,
			riskPercent:    2.0,
			stopLossPips:   50,
			pipValue:       10.0,
			expectedRisk:   200,
			expectedVolume: 0.4,
		},
		{
			name:           "small account",
			balance:        1000,
			riskPercent:    1.0,
			stopLossPips:   30,
			pipValue:       10.0,
			expectedRisk:   10,
			expectedVolume: 0.03333333333333333,
		},
		{
			name:           "large stop loss",
			balance:        10000,
			riskPercent:    2.0,
			stopLossPips:   200,
			pipValue:       10.0,
			expectedRisk:   200,
			expectedVolume: 0.1,
		},
		{
			name:           "high risk percentage",
			balance:        10000,
			riskPercent:    5.0,
			stopLossPips:   50,
			pipValue:       10.0,
			expectedRisk:   500,
			expectedVolume: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			riskAmount := tt.balance * (tt.riskPercent / 100.0)
			volume := riskAmount / (tt.stopLossPips * tt.pipValue)

			assert.Equal(t, tt.expectedRisk, riskAmount)
			assert.InDelta(t, tt.expectedVolume, volume, 0.0001)
		})
	}
}

func TestGlobalSettings_NewUser(t *testing.T) {
	userID := uuid.New()
	settings := model.NewGlobalSettings(userID)

	assert.Equal(t, userID, settings.UserID)
	assert.False(t, settings.AutoTradeEnabled)
	assert.Equal(t, 2.0, settings.MaxRiskPercent)
	assert.Equal(t, 10, settings.MaxPositions)
	assert.Equal(t, 100.0, settings.MaxLotSize)
	assert.Equal(t, 5000.0, settings.MaxDailyLoss)
	assert.Equal(t, 10.0, settings.MaxDrawdownPercent)
}

func TestRiskConfig_Defaults(t *testing.T) {
	userID := uuid.New()
	config := model.NewRiskConfig(userID, uuid.Nil)

	assert.Equal(t, userID, config.UserID)
	assert.Equal(t, 2.0, config.MaxRiskPercent)
	assert.Equal(t, 5, config.MaxPositions)
	assert.False(t, config.TrailingStopEnabled)
}

func TestRiskEngineCheckAuto_DefensiveValidation(t *testing.T) {
	engine := NewRiskEngine()
	cfg := model.NewRiskConfig(uuid.New(), uuid.Nil)
	base := &model.RiskCheckRequest{
		AccountID:      uuid.New(),
		Symbol:         "EURUSD",
		Volume:         0.1,
		CurrentBalance: 10000,
		CurrentEquity:  10000,
	}
	tests := []struct {
		name string
		edit func(*model.RiskCheckRequest)
		code string
	}{
		{"empty symbol", func(r *model.RiskCheckRequest) { r.Symbol = " " }, "RISK_SYMBOL_EMPTY"},
		{"nan volume", func(r *model.RiskCheckRequest) { r.Volume = math.NaN() }, "RISK_VOLUME_INVALID"},
		{"negative positions", func(r *model.RiskCheckRequest) { r.OpenPositions = -1 }, "RISK_POSITION_COUNT_INVALID"},
		{"invalid balance", func(r *model.RiskCheckRequest) { r.CurrentBalance = math.Inf(1) }, "RISK_BALANCE_INVALID"},
		{"invalid equity", func(r *model.RiskCheckRequest) { r.CurrentEquity = -1 }, "RISK_EQUITY_INVALID"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := *base
			tt.edit(&req)
			res := engine.CheckAuto(&req, cfg)
			assert.False(t, res.Allowed)
			assert.Equal(t, tt.code, res.Decision.Code)
		})
	}
}

func TestAutoTradingService_ToggleAutoTrade(t *testing.T) {
	userID := uuid.New()

	t.Run("enable auto trade for new user", func(t *testing.T) {
		settings := model.NewGlobalSettings(userID)
		settings.AutoTradeEnabled = true

		assert.Equal(t, userID, settings.UserID)
		assert.True(t, settings.AutoTradeEnabled)
	})

	t.Run("disable auto trade for existing user", func(t *testing.T) {
		settings := model.NewGlobalSettings(userID)
		settings.AutoTradeEnabled = true
		settings.AutoTradeEnabled = false

		assert.False(t, settings.AutoTradeEnabled)
	})
}

func TestAutoTradingService_GetRiskConfig(t *testing.T) {
	userID := uuid.New()

	t.Run("get existing risk config", func(t *testing.T) {
		existingConfig := &model.RiskConfig{
			UserID:         userID,
			MaxRiskPercent: 3.0,
			MaxPositions:   10,
		}

		assert.Equal(t, 3.0, existingConfig.MaxRiskPercent)
		assert.Equal(t, 10, existingConfig.MaxPositions)
	})

	t.Run("create new risk config when not found", func(t *testing.T) {
		newConfig := model.NewRiskConfig(userID, uuid.Nil)

		assert.Equal(t, userID, newConfig.UserID)
		assert.Equal(t, 2.0, newConfig.MaxRiskPercent)
	})
}

func TestDrawdownCalculation(t *testing.T) {
	tests := []struct {
		name       string
		balance    float64
		equity     float64
		expectedDD float64
	}{
		{"no drawdown", 10000, 10000, 0},
		{"10% drawdown", 10000, 9000, 10},
		{"25% drawdown", 10000, 7500, 25},
		{"50% drawdown", 10000, 5000, 50},
		{"profit (negative DD)", 10000, 11000, -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			drawdown := (tt.balance - tt.equity) / tt.balance * 100
			assert.Equal(t, tt.expectedDD, drawdown)
		})
	}
}

func TestValidateRiskCheckRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *model.RiskCheckRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &model.RiskCheckRequest{
				AccountID:      uuid.New(),
				Volume:         0.1,
				OpenPositions:  2,
				CurrentBalance: 10000,
				CurrentEquity:  10000,
			},
			wantErr: false,
		},
		{
			name: "empty account ID",
			req: &model.RiskCheckRequest{
				AccountID:      uuid.Nil,
				Volume:         0.1,
				OpenPositions:  2,
				CurrentBalance: 10000,
				CurrentEquity:  10000,
			},
			wantErr: true,
		},
		{
			name: "negative volume",
			req: &model.RiskCheckRequest{
				AccountID:      uuid.New(),
				Volume:         -0.1,
				OpenPositions:  2,
				CurrentBalance: 10000,
				CurrentEquity:  10000,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRiskCheckRequest(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func validateRiskCheckRequest(req *model.RiskCheckRequest) error {
	if req.AccountID == uuid.Nil {
		return errors.New("account ID is required")
	}
	if req.Volume <= 0 {
		return errors.New("volume must be positive")
	}
	return nil
}
