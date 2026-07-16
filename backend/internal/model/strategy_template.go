package model

import (
	"database/sql/driver"
	"errors"
	"strings"
)

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

type StringArray []string

func (s StringArray) Value() (driver.Value, error) {
	if len(s) == 0 {
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
