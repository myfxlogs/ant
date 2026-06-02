package model

import (
	"time"

	"github.com/google/uuid"
)

// AIConfig AI模型配置
type AIConfig struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	Provider    string    `json:"provider" db:"provider"`
	APIKey      string    `json:"-" db:"api_key"` // 不在JSON中暴露
	ModelName   string    `json:"model_name" db:"model_name"`
	MaxTokens   int       `json:"max_tokens" db:"max_tokens"`
	Temperature float64   `json:"temperature" db:"temperature"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
