package repository

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// AIStrategyTemplate is a system-seeded strategy template for AI generation.
type AIStrategyTemplate struct {
	ID             uuid.UUID `db:"id"`
	Category       string    `db:"category"`
	Name           string    `db:"name"`
	DescriptionZh  string    `db:"description_zh"`
	CodeSkeleton   string    `db:"code_skeleton"`   // Go code skeleton for AI strategy generation
	ParameterSlots []byte    `db:"parameter_slots"` // proto binary TemplateParameterSlots
	RiskLevel      string    `db:"risk_level"`
	IsActive       bool      `db:"is_active"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

// ParameterSlotsString returns a human-readable description of the parameter slots
// for AI prompt injection (legacy callers expecting string format).
func (t *AIStrategyTemplate) ParameterSlotsString() string {
	if len(t.ParameterSlots) == 0 {
		return ""
	}
	var slots antv1.TemplateParameterSlots
	if err := proto.Unmarshal(t.ParameterSlots, &slots); err != nil {
		return ""
	}
	var s string
	for _, p := range slots.GetSlots() {
		s += fmt.Sprintf("%s(%s): default=%.1f range=%.1f:%.1f:%.1f %s\n",
			p.GetName(), p.GetType(), p.GetDefaultValue(),
			p.GetMin(), p.GetMax(), p.GetStep(), p.GetDescription())
	}
	return s
}
