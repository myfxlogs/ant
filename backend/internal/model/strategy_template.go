package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type StrategyTemplate struct {
	ID          uuid.UUID   `json:"id" db:"id"`
	UserID      uuid.UUID   `json:"user_id" db:"user_id"`
	Name        string      `json:"name" db:"name"`
	Description string      `json:"description" db:"description"`
	Code        string      `json:"code" db:"code"`
	Status      string      `json:"status" db:"status"`
	Parameters  JSONB       `json:"parameters" db:"parameters"`
	I18n        JSONB       `json:"i18n" db:"i18n"`
	IsPublic    bool        `json:"is_public" db:"is_public"`
	IsSystem    bool        `json:"is_system" db:"is_system"`
	Tags        StringArray `json:"tags" db:"tags"`
	UseCount    int         `json:"use_count" db:"use_count"`
	CreatedAt   time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at" db:"updated_at"`
}

// TemplateI18n 用于对 StrategyTemplate.I18n 提供类型安全的访问视图。
//
// 结构示例：
//
//	{
//	  "name": {"zh-CN": "ATR 趋势策略", "en": "ATR Trend Strategy"},
//	  "description": {"zh-CN": "..."},
//	  "params": {
//	    "atr_period": {
//	      "label": {"zh-CN": "ATR 周期", "en": "ATR period"},
//	      "description": {"zh-CN": "..."}
//	    }
//	  }
//	}
type TemplateI18n struct {
	Name        map[string]string            `json:"name,omitempty"`
	Description map[string]string            `json:"description,omitempty"`
	Params      map[string]TemplateParamI18n `json:"params,omitempty"`
}

type TemplateParamI18n struct {
	Label       map[string]string `json:"label,omitempty"`
	Description map[string]string `json:"description,omitempty"`
}

// GetI18n 解析 I18n JSONB 字段。
func (t *StrategyTemplate) GetI18n() (*TemplateI18n, error) {
	if len(t.I18n) == 0 {
		return nil, nil
	}
	var out TemplateI18n
	if err := json.Unmarshal(t.I18n, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetI18n 将类型安全的 TemplateI18n 编码回 I18n JSONB 字段。
func (t *StrategyTemplate) SetI18n(v *TemplateI18n) error {
	if v == nil {
		t.I18n = nil
		return nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	t.I18n = data
	return nil
}

const (
	StrategyTemplateStatusDraft     = "draft"
	StrategyTemplateStatusPublished = "published"
	StrategyTemplateStatusCanceled  = "canceled"
)

func IsValidStrategyTemplateStatus(status string) bool {
	switch status {
	case StrategyTemplateStatusDraft, StrategyTemplateStatusPublished, StrategyTemplateStatusCanceled:
		return true
	default:
		return false
	}
}

func CanTransitionStrategyTemplateStatus(from, to string) bool {
	if from == to {
		return IsValidStrategyTemplateStatus(from)
	}
	switch from {
	case StrategyTemplateStatusDraft:
		return to == StrategyTemplateStatusPublished || to == StrategyTemplateStatusCanceled
	default:
		return false
	}
}

func CanRunStrategyTemplateOnline(status string) bool {
	return status == StrategyTemplateStatusPublished
}

type StringArray []string

func (s StringArray) Value() (driver.Value, error) {
	if s == nil || len(s) == 0 {
		return "{}", nil
	}
	var result string
	for i, v := range s {
		if i > 0 {
			result += ","
		}
		result += `"` + v + `"`
	}
	return "{" + result + "}", nil
}

func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}
	var str string
	switch v := value.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return errors.New("type assertion to []byte or string failed")
	}
	if str == "{}" || str == "" {
		*s = []string{}
		return nil
	}
	str = strings.Trim(str, "{}")
	if str == "" {
		*s = []string{}
		return nil
	}
	var result []string
	inQuote := false
	current := ""
	for _, r := range str {
		if r == '"' {
			inQuote = !inQuote
		} else if r == ',' && !inQuote {
			result = append(result, current)
			current = ""
		} else {
			current += string(r)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	*s = result
	return nil
}

type TemplateParameter struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Default     string   `json:"default,omitempty"`
	Min         string   `json:"min,omitempty"`
	Max         string   `json:"max,omitempty"`
	Step        string   `json:"step,omitempty"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Options     []string `json:"options,omitempty"`
}

func (t *StrategyTemplate) GetParameters() ([]TemplateParameter, error) {
	if len(t.Parameters) == 0 {
		return nil, nil
	}
	var params []TemplateParameter
	err := json.Unmarshal(t.Parameters, &params)
	return params, err
}

func (t *StrategyTemplate) SetParameters(params []TemplateParameter) error {
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	t.Parameters = data
	return nil
}

func NewStrategyTemplate(userID uuid.UUID, name, code string) *StrategyTemplate {
	now := time.Now()
	return &StrategyTemplate{
		ID:         uuid.New(),
		UserID:     userID,
		Name:       name,
		Code:       code,
		Status:     StrategyTemplateStatusPublished,
		Parameters: []byte("[]"),
		Tags:       []string{},
		UseCount:   0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}
