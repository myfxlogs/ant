package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"anttrader/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	stepKeyIntent = "v2:intent"
	stepKeyAgent  = "v2:agent"
	stepKeyCode   = "v2:code"
	stepKeyDone   = "v2:done"
)

// DebateV2Service manages the multi-expert debate flow (v2).
type DebateV2Service struct {
	repo        *repository.DebateRepository
	jobRepo     *repository.JobRepository
	pg          *pgxpool.Pool
	mu          sync.Mutex
	jobChans    map[uuid.UUID]chan *debateV2JobEvent
	jobSessions map[uuid.UUID][2]uuid.UUID // jobID → {sessionID, userID}
}

type debateV2JobEvent struct {
	Phase   string `json:"phase"`
	Event   string `json:"event"`
	Content string `json:"content"`
	Message string `json:"message"`
}

// V2Step is a single step in the debate v2 flow.
type V2Step struct {
	StepKey   string      `json:"stepKey"`
	AgentKey  string      `json:"agentKey,omitempty"`
	AgentName string      `json:"agentName,omitempty"`
	Messages  []V2Message `json:"messages"`
}

// V2Message is a single message within a step.
type V2Message struct {
	ID      string `json:"id"`
	Role    string `json:"role"`
	Content string `json:"content"`
	Kind    string `json:"kind,omitempty"`
}

// V2Code holds generated strategy code.
type V2Code struct {
	Text   string `json:"text"`
	Python string `json:"python"`
}

// V2Usage holds cumulative token counts.
type V2Usage struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
	TotalTokens      int32 `json:"total_tokens"`
}

// V2Session is the full debate v2 session view.
type V2Session struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	CurrentStep string   `json:"currentStep"`
	Agents      []string `json:"agents"`
	Steps       []V2Step `json:"steps"`
	ParamSchema string   `json:"paramSchema"`
	Code        *V2Code  `json:"code,omitempty"`
	Provider    string   `json:"provider"`
	Model       string   `json:"model"`
	Usage       *V2Usage `json:"usage,omitempty"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

// v2Extras holds the extra columns added by migration 119.
type v2Extras struct {
	Steps    []byte
	Code     []byte
	Provider string
	Model    string
	Usage    []byte
}

func NewDebateV2Service(pg *pgxpool.Pool, jobRepo *repository.JobRepository) *DebateV2Service {
	return &DebateV2Service{
		repo:        repository.NewDebateRepository(pg),
		jobRepo:     jobRepo,
		pg:          pg,
		jobChans:    make(map[uuid.UUID]chan *debateV2JobEvent),
		jobSessions: make(map[uuid.UUID][2]uuid.UUID),
	}
}

// --- Session CRUD ---

func (s *DebateV2Service) Start(ctx context.Context, userID uuid.UUID, agents []string, title string) (*V2Session, error) {
	if len(agents) == 0 {
		return nil, fmt.Errorf("at least one agent is required")
	}
	if title == "" {
		title = "New Debate"
	}
	sess, err := s.repo.CreateSession(ctx, userID, title, agents)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	v2status := stepKeyIntent
	intentStep := V2Step{
		StepKey: "intent",
		Messages: []V2Message{{
			ID:      uuid.NewString(),
			Role:    "assistant",
			Content: "你好！请描述你想要创建的量化策略的目标和需求。",
		}},
	}
	stepsJSON, _ := json.Marshal([]V2Step{intentStep})
	_, err = s.pg.Exec(ctx,
		`UPDATE debate_sessions SET status=$1, steps=$2, updated_at=NOW() WHERE id=$3`,
		v2status, stepsJSON, sess.ID)
	if err != nil {
		return nil, fmt.Errorf("init v2 session: %w", err)
	}
	return s.toV2(ctx, sess.ID, userID)
}

func (s *DebateV2Service) Get(ctx context.Context, id string, userID uuid.UUID) (*V2Session, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid session id")
	}
	return s.toV2(ctx, uid, userID)
}

func (s *DebateV2Service) List(ctx context.Context, userID uuid.UUID, limit int) ([]*V2Session, error) {
	sessions, err := s.repo.ListSessions(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	out := make([]*V2Session, 0, len(sessions))
	for _, sess := range sessions {
		v2, err := s.toV2(ctx, sess.ID, userID)
		if err != nil {
			continue
		}
		out = append(out, v2)
	}
	return out, nil
}

func (s *DebateV2Service) Delete(ctx context.Context, id string, userID uuid.UUID) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid session id")
	}
	return s.repo.DeleteSession(ctx, uid, userID)
}

// --- Step Management ---

func (s *DebateV2Service) Chat(ctx context.Context, id string, userID uuid.UUID, message string) (*V2Session, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid session id")
	}
	sess, err := s.repo.GetSession(ctx, uid, userID)
	if err != nil || sess == nil {
		return nil, fmt.Errorf("session not found")
	}
	extras, err := s.loadExtras(ctx, uid)
	if err != nil {
		return nil, err
	}
	steps := s.parseSteps(extras)
	if len(steps) == 0 {
		return nil, fmt.Errorf("session has no steps")
	}
	last := &steps[len(steps)-1]
	last.Messages = append(last.Messages, V2Message{
		ID:      uuid.NewString(),
		Role:    "user",
		Content: message,
	})
	last.Messages = append(last.Messages, V2Message{
		ID:      uuid.NewString(),
		Role:    "assistant",
		Content: fmt.Sprintf("收到你的消息：%s\n\n（AI 策略引擎正在开发中，此回复为占位响应。）", message),
	})
	if err := s.saveSteps(ctx, uid, steps, sess.Status); err != nil {
		return nil, err
	}
	return s.toV2(ctx, uid, userID)
}

func (s *DebateV2Service) Advance(ctx context.Context, id string, userID uuid.UUID) (*V2Session, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid session id")
	}
	sess, err := s.repo.GetSession(ctx, uid, userID)
	if err != nil || sess == nil {
		return nil, fmt.Errorf("session not found")
	}
	extras, err := s.loadExtras(ctx, uid)
	if err != nil {
		return nil, err
	}
	steps := s.parseSteps(extras)
	agents := []string(sess.Agents)
	currentStep := sess.Status

	nextStep := s.nextStepKey(currentStep, agents)
	if nextStep == "" {
		return nil, fmt.Errorf("no next step available")
	}
	if currentStep != nextStep {
		steps = append(steps, s.initStep(nextStep))
	}
	if err := s.saveSteps(ctx, uid, steps, nextStep); err != nil {
		return nil, err
	}
	return s.toV2(ctx, uid, userID)
}

func (s *DebateV2Service) Back(ctx context.Context, id string, userID uuid.UUID) (*V2Session, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid session id")
	}
	sess, err := s.repo.GetSession(ctx, uid, userID)
	if err != nil || sess == nil {
		return nil, fmt.Errorf("session not found")
	}
	extras, err := s.loadExtras(ctx, uid)
	if err != nil {
		return nil, err
	}
	steps := s.parseSteps(extras)
	agents := []string(sess.Agents)

	prevKey := s.prevStepKey(sess.Status, agents)
	if prevKey == "" {
		return nil, fmt.Errorf("already at first step")
	}
	if len(steps) > 0 && steps[len(steps)-1].StepKey == sess.Status {
		steps = steps[:len(steps)-1]
	}
	if err := s.saveSteps(ctx, uid, steps, prevKey); err != nil {
		return nil, err
	}
	return s.toV2(ctx, uid, userID)
}

func (s *DebateV2Service) SetParams(ctx context.Context, id string, userID uuid.UUID, paramsJSON string) (*V2Session, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid session id")
	}
	_, err = s.pg.Exec(ctx,
		`UPDATE debate_sessions SET param_schema=$1, updated_at=NOW() WHERE id=$2`,
		paramsJSON, uid)
	if err != nil {
		return nil, fmt.Errorf("set params: %w", err)
	}
	return s.toV2(ctx, uid, userID)
}

// --- Job System ---

func (s *DebateV2Service) PrepareChatJob(ctx context.Context, id string, userID uuid.UUID, message string) (uuid.UUID, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid session id")
	}
	sess, err := s.repo.GetSession(ctx, uid, userID)
	if err != nil || sess == nil {
		return uuid.Nil, fmt.Errorf("session not found")
	}
	extras, err := s.loadExtras(ctx, uid)
	if err != nil {
		return uuid.Nil, err
	}
	steps := s.parseSteps(extras)
	if len(steps) == 0 {
		return uuid.Nil, fmt.Errorf("session has no steps")
	}
	last := &steps[len(steps)-1]
	last.Messages = append(last.Messages, V2Message{
		ID:      uuid.NewString(),
		Role:    "user",
		Content: message,
	})
	if err := s.saveSteps(ctx, uid, steps, sess.Status); err != nil {
		return uuid.Nil, err
	}
	job := &repository.Job{
		ID:             uuid.New(),
		UserID:         userID,
		Kind:           "debate_v2_chat",
		Status:         "queued",
		RequestSummary: message,
	}
	if err := s.jobRepo.CreateJob(ctx, job); err != nil {
		return uuid.Nil, fmt.Errorf("create chat job: %w", err)
	}
	s.mu.Lock()
	s.jobSessions[job.ID] = [2]uuid.UUID{uid, userID}
	s.mu.Unlock()
	return job.ID, nil
}

func (s *DebateV2Service) RunChatJob(jobID uuid.UUID) error {
	s.mu.Lock()
	ch := make(chan *debateV2JobEvent, 64)
	s.jobChans[jobID] = ch
	pair := s.jobSessions[jobID]
	s.mu.Unlock()
	sessionID := pair[0]
	userID := pair[1]

	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.jobChans, jobID)
			delete(s.jobSessions, jobID)
			s.mu.Unlock()
			close(ch)
		}()

		msg := "感谢你的问题。我正在分析你的量化策略需求...\n\n这是模拟的 AI 回复（策略引擎正在集成中）。"
		for i := 0; i < len(msg); i += 5 {
			end := i + 5
			if end > len(msg) {
				end = len(msg)
			}
			ch <- &debateV2JobEvent{Phase: "running", Event: "chunk", Content: msg[i:end]}
			time.Sleep(30 * time.Millisecond)
		}
		ch <- &debateV2JobEvent{Phase: "completed", Event: "completed"}

		if sessionID == uuid.Nil || userID == uuid.Nil {
			return
		}
		bgCtx := context.Background()
		sess, err := s.repo.GetSession(bgCtx, sessionID, userID)
		if err != nil || sess == nil {
			return
		}
		extras, err := s.loadExtras(bgCtx, sessionID)
		if err != nil {
			return
		}
		steps := s.parseSteps(extras)
		if len(steps) > 0 {
			last := &steps[len(steps)-1]
			last.Messages = append(last.Messages, V2Message{
				ID:      uuid.NewString(),
				Role:    "assistant",
				Content: msg,
			})
			_ = s.saveSteps(bgCtx, sessionID, steps, sess.Status)
		}
	}()
	return nil
}

func (s *DebateV2Service) GetJob(ctx context.Context, jobID, userID uuid.UUID) (phase, message string, err error) {
	job, err := s.jobRepo.GetJob(ctx, userID, jobID)
	if err != nil {
		return "", "", fmt.Errorf("get job: %w", err)
	}
	return job.Status, job.ErrorMessage, nil
}

func (s *DebateV2Service) JobChannel(jobID, userID uuid.UUID) (<-chan *debateV2JobEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, ok := s.jobChans[jobID]
	if !ok {
		return nil, fmt.Errorf("job %s not running", jobID)
	}
	// Verify the job belongs to the requesting user.
	if ids, exists := s.jobSessions[jobID]; !exists || ids[1] != userID {
		return nil, fmt.Errorf("job %s not found", jobID)
	}
	return ch, nil
}

// --- helpers ---

func (s *DebateV2Service) loadExtras(ctx context.Context, id uuid.UUID) (*v2Extras, error) {
	var e v2Extras
	err := s.pg.QueryRow(ctx,
		`SELECT steps, code, provider, model, usage
		 FROM debate_sessions WHERE id=$1`, id,
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
	extras, err := s.loadExtras(ctx, id)
	if err != nil {
		return nil, err
	}
	return sessionToV2(sess, extras), nil
}

func sessionToV2(sess *repository.DebateSession, e *v2Extras) *V2Session {
	steps := parseStepsFromRaw(e.Steps)
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
	}
}

func (s *DebateV2Service) parseSteps(e *v2Extras) []V2Step {
	return parseStepsFromRaw(e.Steps)
}

func parseStepsFromRaw(raw []byte) []V2Step {
	var steps []V2Step
	if len(raw) > 0 {
		json.Unmarshal(raw, &steps)
	}
	return steps
}

func (s *DebateV2Service) saveSteps(ctx context.Context, id uuid.UUID, steps []V2Step, status string) error {
	b, err := json.Marshal(steps)
	if err != nil {
		return fmt.Errorf("marshal steps: %w", err)
	}
	_, err = s.pg.Exec(ctx,
		`UPDATE debate_sessions SET steps=$1, status=$2, updated_at=NOW() WHERE id=$3`,
		b, status, id)
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
