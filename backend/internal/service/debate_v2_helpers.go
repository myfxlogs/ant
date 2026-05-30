package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"anttrader/internal/repository"

	"github.com/google/uuid"
)

// --- helpers ---

func (s *DebateV2Service) loadExtras(ctx context.Context, id, userID uuid.UUID) (*v2Extras, error) {
	var e v2Extras
	err := s.pg.QueryRow(ctx,
		`SELECT steps, code, provider, model, usage
		 FROM debate_sessions WHERE id=$1 AND user_id=$2`, id, userID,
	).Scan(&e.Steps, &e.Code, &e.Provider, &e.Model, &e.Usage)
	if err != nil {
		return nil, fmt.Errorf("load v2 extras: %w", err)
	}
	return &e, nil
}

func (s *DebateV2Service) toV2(ctx context.Context, id, userID uuid.UUID) (*V2Session, error) {
	sess, err := s.repo.GetSession(ctx, id, userID)
	if err != nil || sess == nil {
		return nil, fmt.Errorf("session not found")
	}
	extras, err := s.loadExtras(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	return sessionToV2(sess, extras)
}

func sessionToV2(sess *repository.DebateSession, e *v2Extras) (*V2Session, error) {
	steps, err := parseStepsFromRaw(e.Steps)
	if err != nil {
		return nil, fmt.Errorf("parse steps: %w", err)
	}
	agents := []string(sess.Agents)
	var code *V2Code
	if len(e.Code) > 0 {
		var c V2Code
		if json.Unmarshal(e.Code, &c) == nil {
			code = &c
		}
	}
	var usage *V2Usage
	if len(e.Usage) > 0 {
		var u V2Usage
		if json.Unmarshal(e.Usage, &u) == nil {
			usage = &u
		}
	}
	return &V2Session{
		ID:          sess.ID.String(),
		Title:       sess.Title,
		Status:      sess.Status,
		CurrentStep: v2StatusToCurrentStep(sess.Status),
		Agents:      agents,
		Steps:       steps,
		ParamSchema: string(sess.ParamSchema),
		Code:        code,
		Provider:    e.Provider,
		Model:       e.Model,
		Usage:       usage,
		CreatedAt:   sess.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   sess.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *DebateV2Service) parseSteps(e *v2Extras) ([]V2Step, error) {
	return parseStepsFromRaw(e.Steps)
}

func parseStepsFromRaw(raw []byte) ([]V2Step, error) {
	var steps []V2Step
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &steps); err != nil {
			return nil, fmt.Errorf("unmarshal steps: %w", err)
		}
	}
	return steps, nil
}

func (s *DebateV2Service) saveSteps(ctx context.Context, id, userID uuid.UUID, steps []V2Step, status string) error {
	b, err := json.Marshal(steps)
	if err != nil {
		return fmt.Errorf("marshal steps: %w", err)
	}
	_, err = s.pg.Exec(ctx,
		`UPDATE debate_sessions SET steps=$1, status=$2, updated_at=NOW() WHERE id=$3 AND user_id=$4`,
		b, status, id, userID)
	return err
}

func (s *DebateV2Service) nextStepKey(current string, agents []string) string {
	switch {
	case current == stepKeyIntent:
		if len(agents) > 0 {
			return "v2:agent:" + agents[0]
		}
		return stepKeyCode
	case isAgentStep(current):
		agent := agentFromStep(current)
		for i, a := range agents {
			if a == agent && i+1 < len(agents) {
				return "v2:agent:" + agents[i+1]
			}
		}
		return stepKeyCode
	case current == stepKeyCode:
		return stepKeyDone
	default:
		return ""
	}
}

func (s *DebateV2Service) prevStepKey(current string, agents []string) string {
	switch {
	case current == stepKeyCode:
		if len(agents) > 0 {
			return "v2:agent:" + agents[len(agents)-1]
		}
		return stepKeyIntent
	case isAgentStep(current):
		agent := agentFromStep(current)
		for i, a := range agents {
			if a == agent && i > 0 {
				return "v2:agent:" + agents[i-1]
			}
		}
		return stepKeyIntent
	case current == stepKeyDone:
		return stepKeyCode
	default:
		return ""
	}
}

func (s *DebateV2Service) initStep(stepKey string) V2Step {
	agentKey := ""
	name := stepKey
	if isAgentStep(stepKey) {
		agentKey = agentFromStep(stepKey)
		name = agentKey
	}
	content := "开始讨论吧。"
	if stepKey == stepKeyCode {
		content = "正在生成策略代码..."
	}
	return V2Step{
		StepKey:   stepKey,
		AgentKey:  agentKey,
		AgentName: name,
		Messages: []V2Message{{
			ID:      uuid.NewString(),
			Role:    "assistant",
			Content: content,
			Kind:    "kickoff",
		}},
	}
}

func isAgentStep(status string) bool {
	return len(status) > 9 && status[:9] == "v2:agent:"
}

func agentFromStep(status string) string {
	if isAgentStep(status) {
		return status[9:]
	}
	return ""
}

func v2StatusToCurrentStep(status string) string {
	if status == stepKeyDone {
		return "code"
	}
	return status
}
