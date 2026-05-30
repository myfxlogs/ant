package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"go.uber.org/zap"

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
	log         *zap.Logger
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

func NewDebateV2Service(pg *pgxpool.Pool, jobRepo *repository.JobRepository, log *zap.Logger) *DebateV2Service {
	return &DebateV2Service{
		repo:        repository.NewDebateRepository(pg),
		jobRepo:     jobRepo,
		pg:          pg,
		log:         log,
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

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	sessionID := uuid.New()
	_, err = tx.Exec(ctx,
		`INSERT INTO debate_sessions (id, user_id, title, status, agents, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, NOW(), NOW())`,
		sessionID, userID, title, "idle", agents)
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
	stepsJSON, err := json.Marshal([]V2Step{intentStep})
	if err != nil {
		return nil, fmt.Errorf("marshal steps: %w", err)
	}
	_, err = tx.Exec(ctx,
		`UPDATE debate_sessions SET status=$1, steps=$2, updated_at=NOW() WHERE id=$3`,
		v2status, stepsJSON, sessionID)
	if err != nil {
		return nil, fmt.Errorf("init v2 session: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return s.toV2(ctx, sessionID, userID)
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
			s.log.Warn("List: skip session due to conversion error", zap.String("session_id", sess.ID.String()), zap.Error(err))
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
	extras, err := s.loadExtras(ctx, uid, userID)
	if err != nil {
		return nil, err
	}
	steps, err := s.parseSteps(extras)
	if err != nil {
		return nil, err
	}
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
	if err := s.saveSteps(ctx, uid, userID, steps, sess.Status); err != nil {
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
	extras, err := s.loadExtras(ctx, uid, userID)
	if err != nil {
		return nil, err
	}
	steps, err := s.parseSteps(extras)
	if err != nil {
		return nil, err
	}
	agents := []string(sess.Agents)
	currentStep := sess.Status

	nextStep := s.nextStepKey(currentStep, agents)
	if nextStep == "" {
		return nil, fmt.Errorf("no next step available")
	}
	if currentStep != nextStep {
		steps = append(steps, s.initStep(nextStep))
	}
	if err := s.saveSteps(ctx, uid, userID, steps, nextStep); err != nil {
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
	extras, err := s.loadExtras(ctx, uid, userID)
	if err != nil {
		return nil, err
	}
	steps, err := s.parseSteps(extras)
	if err != nil {
		return nil, err
	}
	agents := []string(sess.Agents)

	prevKey := s.prevStepKey(sess.Status, agents)
	if prevKey == "" {
		return nil, fmt.Errorf("already at first step")
	}
	if len(steps) > 0 && steps[len(steps)-1].StepKey == sess.Status {
		steps = steps[:len(steps)-1]
	}
	if err := s.saveSteps(ctx, uid, userID, steps, prevKey); err != nil {
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
		`UPDATE debate_sessions SET param_schema=$1, updated_at=NOW() WHERE id=$2 AND user_id=$3`,
		paramsJSON, uid, userID)
	if err != nil {
		return nil, fmt.Errorf("set params: %w", err)
	}
	return s.toV2(ctx, uid, userID)
}
