package service

import (
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
