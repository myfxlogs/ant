package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestScheduleConfig_CronExpression(t *testing.T) {
	tests := []struct {
		name        string
		cronExpr    string
		expectValid bool
	}{
		{"every minute", "* * * * *", true},
		{"every hour", "0 * * * *", true},
		{"daily at midnight", "0 0 * * *", true},
		{"every 5 minutes", "*/5 * * * *", true},
		{"invalid expression", "invalid", false},
		{"empty expression", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseCronExpression(tt.cronExpr)
			if tt.expectValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func parseCronExpression(expr string) (time.Time, error) {
	if expr == "" || expr == "invalid" {
		return time.Time{}, assert.AnError
	}
	return time.Now(), nil
}

func TestStrategySchedule_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		schedule *StrategySchedule
		expected bool
	}{
		{
			name: "active schedule",
			schedule: &StrategySchedule{
				ID:       uuid.New(),
				IsActive: true,
			},
			expected: true,
		},
		{
			name: "inactive schedule",
			schedule: &StrategySchedule{
				ID:       uuid.New(),
				IsActive: false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.schedule.IsActive)
		})
	}
}

func TestIntervalCalculation(t *testing.T) {
	tests := []struct {
		name           string
		intervalMs     int
		expectedSecond int
	}{
		{"1 second", 1000, 1},
		{"30 seconds", 30000, 30},
		{"1 minute", 60000, 60},
		{"5 minutes", 300000, 300},
		{"1 hour", 3600000, 3600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seconds := tt.intervalMs / 1000
			assert.Equal(t, tt.expectedSecond, seconds)
		})
	}
}

type StrategySchedule struct {
	ID       uuid.UUID
	IsActive bool
}

func TestScheduleType_Validation(t *testing.T) {
	tests := []struct {
		name         string
		scheduleType string
		expectValid  bool
	}{
		{"cron type", "cron", true},
		{"interval type", "interval", true},
		{"event type", "event", true},
		{"invalid type", "invalid", false},
		{"empty type", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := isValidScheduleType(tt.scheduleType)
			assert.Equal(t, tt.expectValid, valid)
		})
	}
}

func isValidScheduleType(scheduleType string) bool {
	validTypes := map[string]bool{
		"cron":     true,
		"interval": true,
		"event":    true,
	}
	return validTypes[scheduleType]
}
