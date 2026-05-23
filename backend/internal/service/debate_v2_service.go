package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"anttrader/internal/ai"
	"anttrader/internal/model"
	"anttrader/internal/repository"

	"github.com/google/uuid"
)

// debate_v2_service.go — orchestrator for the redesigned multi-expert flow.
// Sessions persist through the existing debate_sessions / debate_turns tables
// but use the v2:* status prefix and v2_* turn types to stay isolated from
// the legacy flow in debate_service.go.

func init() {
	// Whitelist the v2 turn types so AddTurn (in debate_service.go) accepts
	// them without needing its own knowledge of v2.
	validTurnTypes["v2_user"] = struct{}{}
	validTurnTypes["v2_assistant"] = struct{}{}
	validTurnTypes["v2_code"] = struct{}{}
}

// --------------------------------------------------------------------------
// Status / step-key helpers
// --------------------------------------------------------------------------

const (
	v2StatusPrefix     = "v2:"
	v2StepIntent       = "intent"
	v2StepCode         = "code"
	v2StepDone         = "done"
	v2StepAgentPrefix  = "agent:"
	v2StatusIntent     = "v2:intent"
	v2StatusCode       = "v2:code"
	v2StatusDone       = "v2:done"
	v2AgentStatusTempl = "v2:agent:"
)

// isV2Status reports whether a session row belongs to the v2 flow.
func isV2Status(s string) bool { return strings.HasPrefix(s, v2StatusPrefix) }

// stepKeyFromStatus strips the "v2:" prefix, turning "v2:agent:signals" into
// "agent:signals" (the shape the frontend already uses as StepKey).
func stepKeyFromStatus(status string) string {
	if !isV2Status(status) {
		return ""
	}
	return strings.TrimPrefix(status, v2StatusPrefix)
}

// statusFromStepKey does the opposite — used when we transition the session.
func statusFromStepKey(stepKey string) string { return v2StatusPrefix + stepKey }

// --------------------------------------------------------------------------
// Turn payload
// --------------------------------------------------------------------------

// v2TurnMeta is stored in debate_turns.content_json for every v2_* turn so we
// can reconstruct per-step history and tell kickoff messages apart from
// regular replies.
type v2TurnMeta struct {
	StepKey string `json:"stepKey"`
	Kind    string `json:"kind,omitempty"`   // "reply" | "kickoff" (only for v2_assistant)
	Python  string `json:"python,omitempty"` // only for v2_code
	// Provenance + usage for assistant turns. These are best-effort: when
	// the underlying provider does not return usage we leave the counters
	// at zero rather than failing the request.
	Provider         string `json:"provider,omitempty"`
	Model            string `json:"model,omitempty"`
	PromptTokens     int    `json:"promptTokens,omitempty"`
	CompletionTokens int    `json:"completionTokens,omitempty"`
	TotalTokens      int    `json:"totalTokens,omitempty"`
	Error            string `json:"error,omitempty"`
}

func marshalV2Meta(m v2TurnMeta) []byte {
	raw, _ := json.Marshal(m)
	return raw
}

func unmarshalV2Meta(raw []byte) v2TurnMeta {
	var m v2TurnMeta
	if len(raw) == 0 {
		return m
	}
	_ = json.Unmarshal(raw, &m)
	return m
}

// DTO 类型 + buildDTO/fetchDTO/resolveAgent 已搬到 debate_v2_dto.go。

// --------------------------------------------------------------------------
// Service
// --------------------------------------------------------------------------

// DebateV2Service drives the redesigned flow. It reuses the repository and
// AI provider infrastructure of the v1 DebateService.
type DebateV2Service struct {
	repo       *repository.DebateRepository
	aiCfgSvc   *AIConfigService
	aiAgentSvc *AIAgentService
	// systemAISvc is optional; when present, agents may be bound to a
	// "system:<provider>" locator and the debate flow will build the
	// provider on the fly from the system-level config.
	systemAISvc SystemAIProviderSource
	events      *debateV2EventHub
	advanceJobs *debateV2JobHub
	chatJobs    *debateV2JobHub
}

// SystemAIProviderSource is the slice of systemai.Service that the debate
// flow needs. Defined as an interface here to avoid a hard import cycle and
// to keep the dependency easy to fake in tests.
// 自 059 起每个用户拥有独立的 system_ai_configs 行，因此需要传入 userID。
type SystemAIProviderSource interface {
	BuildProviderConfig(ctx context.Context, userID uuid.UUID, providerID string) (*AIConfig, error)
}

func NewDebateV2Service(repo *repository.DebateRepository, aiCfgSvc *AIConfigService, aiAgentSvc *AIAgentService) *DebateV2Service {
	return &DebateV2Service{
		repo: repo, aiCfgSvc: aiCfgSvc, aiAgentSvc: aiAgentSvc,
		events: newDebateV2EventHub(), advanceJobs: newDebateV2JobHub(), chatJobs: newDebateV2JobHub(),
	}
}

// WithSystemAI lets the server wire in the system-level provider source
// without changing the constructor signature.
func (s *DebateV2Service) WithSystemAI(src SystemAIProviderSource) *DebateV2Service {
	if s != nil {
		s.systemAISvc = src
	}
	return s
}

// Start creates a new v2 session and puts it into the "intent" step.
func (s *DebateV2Service) Start(ctx context.Context, userID uuid.UUID, agents []string, title, locale string) (*V2SessionDTO, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("debate v2 service not initialized")
	}
	clean := sanitizeAgents(agents)
	effTitle := strings.TrimSpace(title)
	if effTitle == "" {
		effTitle = "Debate"
	}
	if r := []rune(effTitle); len(r) > 40 {
		effTitle = string(r[:40])
	}
	row, err := s.repo.CreateSession(ctx, userID, effTitle, clean)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateSession(ctx, row.ID, userID, &repository.SessionPatch{
		Status: strPtr(v2StatusIntent),
	}); err != nil {
		return nil, err
	}
	return s.fetchDTO(ctx, userID, row.ID, locale)
}

// Get returns a v2 session by id.
func (s *DebateV2Service) Get(ctx context.Context, userID, sessionID uuid.UUID, locale string) (*V2SessionDTO, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("debate v2 service not initialized")
	}
	return s.fetchDTO(ctx, userID, sessionID, locale)
}

func (s *DebateV2Service) Subscribe(ctx context.Context, userID, sessionID uuid.UUID, locale string) (<-chan *V2SessionDTO, error) {
	if s == nil || s.repo == nil || s.events == nil {
		return nil, errors.New("debate v2 service not initialized")
	}
	initial, err := s.fetchDTO(ctx, userID, sessionID, locale)
	if err != nil {
		return nil, err
	}
	out := make(chan *V2SessionDTO, 8)
	out <- initial
	updates := s.events.subscribe(ctx, sessionID)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-updates:
				if !ok {
					return
				}
				dto, err := s.fetchDTO(ctx, userID, sessionID, locale)
				if err != nil {
					return
				}
				select {
				case out <- dto:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

// List returns recent v2 sessions for the user.
func (s *DebateV2Service) List(ctx context.Context, userID uuid.UUID, limit int, locale string) ([]V2SessionDTO, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("debate v2 service not initialized")
	}
	rows, err := s.repo.ListSessions(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]V2SessionDTO, 0, len(rows))
	for i := range rows {
		if !isV2Status(rows[i].Status) {
			continue
		}
		dto, err := s.buildDTO(ctx, &rows[i], nil, locale)
		if err != nil {
			return nil, err
		}
		out = append(out, *dto)
	}
	return out, nil
}

// Delete removes a v2 session.
func (s *DebateV2Service) Delete(ctx context.Context, userID, sessionID uuid.UUID) error {
	if s == nil || s.repo == nil {
		return errors.New("debate v2 service not initialized")
	}
	return s.repo.DeleteSession(ctx, sessionID, userID)
}

// SetParamSchema overwrites the per-session strategy parameter schema with the
// provided list. The schema is stored as JSON in debate_sessions.param_schema
// and later reused when creating a strategy template or rendering backtest
// parameter forms.
func (s *DebateV2Service) SetParamSchema(ctx context.Context, userID, sessionID uuid.UUID, params []model.TemplateParameter, locale string) (*V2SessionDTO, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("debate v2 service not initialized")
	}
	data, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	jb := model.JSONB(data)
	if err := s.repo.UpdateSession(ctx, sessionID, userID, &repository.SessionPatch{
		ParamSchema: &jb,
	}); err != nil {
		return nil, err
	}
	return s.fetchDTO(ctx, userID, sessionID, locale)
}

// --------------------------------------------------------------------------
// Chat
// --------------------------------------------------------------------------

// Chat appends a user message in the current step and fetches the assistant
// reply. The code step is driven by Advance(), not Chat().
func (s *DebateV2Service) Chat(ctx context.Context, userID, sessionID uuid.UUID, message, locale string) (*V2SessionDTO, error) {
	msg := strings.TrimSpace(message)
	if msg == "" {
		return nil, errors.New("message is required")
	}
	sess, turns, err := s.loadSession(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	stepKey := stepKeyFromStatus(sess.Status)
	if stepKey == "" || stepKey == v2StepCode || stepKey == v2StepDone {
		return nil, errors.New("chat is not allowed in the current step")
	}

	// Persist user turn first so history reflects it for the LLM call.
	if _, err := s.addV2Turn(ctx, sessionID, userID, "v2_user", "user", msg, v2TurnMeta{StepKey: stepKey, Kind: "reply"}); err != nil {
		return nil, err
	}
	s.publishUpdate(sessionID)
	turns = append(turns, repository.DebateTurn{
		SessionID:   sessionID,
		Type:        "v2_user",
		Role:        "user",
		Status:      "approved",
		ContentText: msg,
		ContentJSON: marshalV2Meta(v2TurnMeta{StepKey: stepKey, Kind: "reply"}),
	})

	sess, turns, err = s.loadSession(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	reply, usage, err := s.invokeStep(ctx, userID, sess, turns, stepKey, locale, nil)
	if err != nil {
		if addErr := s.addAsyncError(ctx, sessionID, userID, stepKey, err); addErr != nil {
			return nil, addErr
		}
		return s.fetchDTO(ctx, userID, sessionID, locale)
	}
	reply = StripCodeBlocksV2(reply)
	meta := usage
	meta.StepKey = stepKey
	meta.Kind = "reply"
	if _, err := s.addV2Turn(ctx, sessionID, userID, "v2_assistant", "assistant", reply, meta); err != nil {
		return nil, err
	}
	s.publishUpdate(sessionID)
	return s.fetchDTO(ctx, userID, sessionID, locale)
}

// --------------------------------------------------------------------------
// Advance / Back
// --------------------------------------------------------------------------

// Advance moves from the current step to the next one.
// Transitions trigger a kickoff (agent step) or code generation (code step).
func (s *DebateV2Service) Advance(ctx context.Context, userID, sessionID uuid.UUID, locale string) (*V2SessionDTO, error) {
	sess, _, err := s.loadSession(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	current := stepKeyFromStatus(sess.Status)
	if current == "" || current == v2StepDone {
		return nil, errors.New("cannot advance from the current step")
	}
	if current == v2StepCode {
		// Already at code → mark done.
		if err := s.repo.UpdateSession(ctx, sessionID, userID, &repository.SessionPatch{Status: strPtr(v2StatusDone)}); err != nil {
			return nil, err
		}
		return s.fetchDTO(ctx, userID, sessionID, locale)
	}

	// Previously we required at least one assistant reply on the current step
	// before advancing. That turned out to be too strict: when a kickoff reply
	// fails (e.g. transient LLM error, pre-migration session with no persisted
	// turns), the user gets wedged and can't move forward. We now let the user
	// advance regardless — the downstream kickoff/code-gen for the *next* step
	// will produce whatever greeting / summary is needed.
	_ = current

	agents := []string(sess.Agents)
	nextStep := nextStepKey(current, agents)
	if nextStep == "" {
		return nil, errors.New("no further step")
	}

	if err := s.repo.UpdateSession(ctx, sessionID, userID, &repository.SessionPatch{
		Status: strPtr(statusFromStepKey(nextStep)),
	}); err != nil {
		return nil, err
	}
	s.publishUpdate(sessionID)
	sess, turns, err := s.loadSession(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	switch {
	case strings.HasPrefix(nextStep, v2StepAgentPrefix):
		if err := s.runAgentKickoff(ctx, userID, sess, turns, nextStep, locale, ""); err != nil {
			if addErr := s.addAsyncError(ctx, sessionID, userID, nextStep, err); addErr != nil {
				return nil, addErr
			}
			return s.fetchDTO(ctx, userID, sessionID, locale)
		}
	case nextStep == v2StepCode:
		if err := s.runCodeGeneration(ctx, userID, sess, turns, locale); err != nil {
			if addErr := s.addAsyncError(ctx, sessionID, userID, v2StepCode, err); addErr != nil {
				return nil, addErr
			}
			return s.fetchDTO(ctx, userID, sessionID, locale)
		}
	}

	return s.fetchDTO(ctx, userID, sessionID, locale)
}

// Back rewinds to the previous step without mutating transcripts.
func (s *DebateV2Service) Back(ctx context.Context, userID, sessionID uuid.UUID, locale string) (*V2SessionDTO, error) {
	sess, _, err := s.loadSession(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	current := stepKeyFromStatus(sess.Status)
	prev := prevStepKey(current, []string(sess.Agents))
	if prev == "" {
		return nil, errors.New("already at the first step")
	}
	if err := s.repo.UpdateSession(ctx, sessionID, userID, &repository.SessionPatch{
		Status: strPtr(statusFromStepKey(prev)),
	}); err != nil {
		return nil, err
	}
	s.publishUpdate(sessionID)
	return s.fetchDTO(ctx, userID, sessionID, locale)
}

// --------------------------------------------------------------------------
// Step machinery
// --------------------------------------------------------------------------

func nextStepKey(current string, agents []string) string {
	switch {
	case current == v2StepIntent:
		if len(agents) == 0 {
			return v2StepCode
		}
		return v2StepAgentPrefix + agents[0]
	case strings.HasPrefix(current, v2StepAgentPrefix):
		key := strings.TrimPrefix(current, v2StepAgentPrefix)
		for i, a := range agents {
			if a == key {
				if i+1 < len(agents) {
					return v2StepAgentPrefix + agents[i+1]
				}
				return v2StepCode
			}
		}
		return v2StepCode
	case current == v2StepCode:
		return v2StepDone
	}
	return ""
}

func prevStepKey(current string, agents []string) string {
	switch {
	case current == v2StepIntent:
		return ""
	case strings.HasPrefix(current, v2StepAgentPrefix):
		key := strings.TrimPrefix(current, v2StepAgentPrefix)
		for i, a := range agents {
			if a == key {
				if i == 0 {
					return v2StepIntent
				}
				return v2StepAgentPrefix + agents[i-1]
			}
		}
		return v2StepIntent
	case current == v2StepCode:
		if len(agents) == 0 {
			return v2StepIntent
		}
		return v2StepAgentPrefix + agents[len(agents)-1]
	case current == v2StepDone:
		return v2StepCode
	}
	return ""
}

func hasAssistantReply(turns []repository.DebateTurn, stepKey string) bool {
	for _, t := range turns {
		if t.Type != "v2_assistant" || t.Status != "approved" {
			continue
		}
		meta := unmarshalV2Meta(t.ContentJSON)
		if meta.StepKey == stepKey && strings.TrimSpace(t.ContentText) != "" {
			return true
		}
	}
	return false
}

// --------------------------------------------------------------------------
// LLM invocation
// --------------------------------------------------------------------------

// invokeStep runs the LLM for the current step using all persisted v2_user /
// v2_assistant turns belonging to that step as chat history. The returned
// meta carries provider/model + token usage so callers can persist it on
// the assistant turn for later display.
func (s *DebateV2Service) invokeStep(ctx context.Context, userID uuid.UUID, sess *repository.DebateSession, turns []repository.DebateTurn, stepKey, locale string, emitChunk func(string)) (string, v2TurnMeta, error) {
	sys, err := s.buildSystemPrompt(ctx, userID, sess, turns, stepKey, locale)
	if err != nil {
		return "", v2TurnMeta{}, err
	}
	history := collectStepHistory(turns, stepKey)
	msgs := make([]ai.Message, 0, len(history)+1)
	msgs = append(msgs, ai.Message{Role: "system", Content: sys})
	msgs = append(msgs, history...)

	providers, err := s.getProvidersForStep(ctx, userID, stepKey)
	if err != nil {
		return "", v2TurnMeta{}, err
	}
	if emitChunk != nil {
		text, provider, usage, err := streamChatWithFallback(ctx, providers, msgs, emitChunk)
		if err != nil {
			return "", v2TurnMeta{}, err
		}
		return strings.TrimSpace(text), providerUsageMeta(provider, &ai.Response{Content: text, Usage: usage}), nil
	}
	resp, provider, err := chatWithFallback(ctx, providers, msgs)
	if err != nil {
		return "", v2TurnMeta{}, err
	}
	return strings.TrimSpace(resp.Content), providerUsageMeta(provider, resp), nil
}

// runAgentKickoff injects the hidden kickoff user-turn (as role "user" with
// kind=kickoff) and asks the LLM for the agent's first reply.
func (s *DebateV2Service) runAgentKickoff(ctx context.Context, userID uuid.UUID, sess *repository.DebateSession, turns []repository.DebateTurn, stepKey, locale, streamJobID string) error {
	agentKey := strings.TrimPrefix(stepKey, v2StepAgentPrefix)
	agent := s.resolveAgent(ctx, userID, agentKey, locale)
	kickoff := KickoffUserMessageV2(agent.Name, agent.Type, locale)

	// Persist kickoff as a user turn so history stays consistent across
	// reloads, but mark it kind=kickoff so the UI hides it.
	if _, err := s.addV2Turn(ctx, sess.ID, userID, "v2_user", "user", kickoff, v2TurnMeta{StepKey: stepKey, Kind: "kickoff"}); err != nil {
		return err
	}
	s.publishUpdate(sess.ID)
	turns = append(turns, repository.DebateTurn{
		SessionID:   sess.ID,
		Type:        "v2_user",
		Role:        "user",
		Status:      "approved",
		ContentText: kickoff,
		ContentJSON: marshalV2Meta(v2TurnMeta{StepKey: stepKey, Kind: "kickoff"}),
	})

	var emitChunk func(string)
	if streamJobID != "" {
		emitChunk = func(delta string) { s.emitAdvanceChunk(streamJobID, delta) }
	}
	reply, usage, err := s.invokeStep(ctx, userID, sess, turns, stepKey, locale, emitChunk)
	if err != nil {
		return err
	}
	reply = StripCodeBlocksV2(reply)
	meta := usage
	meta.StepKey = stepKey
	meta.Kind = "reply"
	if _, err := s.addV2Turn(ctx, sess.ID, userID, "v2_assistant", "assistant", reply, meta); err != nil {
		return err
	}
	s.publishUpdate(sess.ID)
	return nil
}

func (s *DebateV2Service) runCodeGeneration(ctx context.Context, userID uuid.UUID, sess *repository.DebateSession, turns []repository.DebateTurn, locale string) error {
	return s.runCodeGenerationWithFeedback(ctx, userID, sess, turns, locale, "", "")
}

// runCodeGenerationWithFeedback regenerates the Python strategy. If feedback
// is non-empty, the user's rejection message is appended as the last user
// turn so the model sees what to improve.
func (s *DebateV2Service) runCodeGenerationWithFeedback(ctx context.Context, userID uuid.UUID, sess *repository.DebateSession, turns []repository.DebateTurn, locale, feedback, streamJobID string) error {
	if streamJobID != "" {
		return s.runCodeGenerationStreamed(ctx, userID, sess, turns, locale, feedback, streamJobID)
	}
	intentSummary := lastAssistantReply(turns, v2StepIntent)
	specs := collectAgentSummaries(turns, []string(sess.Agents))
	sys := CodeSystemPromptV2(intentSummary, specs, locale)

	providers, err := s.getProvidersForStep(ctx, userID, v2StepCode)
	if err != nil {
		return err
	}

	userMsg := "Please generate the complete runnable Python strategy based on the inputs above."
	if f := strings.TrimSpace(feedback); f != "" {
		userMsg = "Your previous strategy code was rejected by the user. Regenerate it addressing the following feedback. Keep obeying every sandbox constraint.\n\n[User feedback]\n" + f
	}

	resp, provider, err := chatWithFallback(ctx, providers, []ai.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: userMsg},
	})
	if err != nil {
		return err
	}
	text := strings.TrimSpace(resp.Content)
	python := ExtractPythonBlockV2(text)
	meta := providerUsageMeta(provider, resp)
	meta.StepKey = v2StepCode
	meta.Python = python
	if _, err := s.addV2Turn(ctx, sess.ID, userID, "v2_code", "assistant", text, meta); err != nil {
		return err
	}
	s.publishUpdate(sess.ID)
	return nil
}

// RejectCode persists the user's feedback as a v2_user turn on the code step
// and regenerates the strategy. The newest v2_code turn wins in buildDTO, so
// the frontend naturally sees the rewritten code.
func (s *DebateV2Service) RejectCode(ctx context.Context, userID, sessionID uuid.UUID, feedback, locale string) (*V2SessionDTO, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("debate v2 service not initialized")
	}
	fb := strings.TrimSpace(feedback)
	if fb == "" {
		return nil, errors.New("feedback is required")
	}
	sess, _, err := s.loadSession(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	current := stepKeyFromStatus(sess.Status)
	if current != v2StepCode && current != v2StepDone {
		return nil, errors.New("reject is only valid after code has been generated")
	}
	// Persist the rejection as a code-step user turn so it stays in history.
	if _, err := s.addV2Turn(ctx, sessionID, userID, "v2_user", "user", fb, v2TurnMeta{StepKey: v2StepCode, Kind: "reject"}); err != nil {
		return nil, err
	}
	s.publishUpdate(sessionID)
	// Ensure status is v2:code (not v2:done) so the UI keeps the code step active.
	if sess.Status != v2StatusCode {
		if err := s.repo.UpdateSession(ctx, sessionID, userID, &repository.SessionPatch{Status: strPtr(v2StatusCode)}); err != nil {
			return nil, err
		}
	}
	sess, turns, err := s.loadSession(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	if err := s.runCodeGenerationWithFeedback(ctx, userID, sess, turns, locale, fb, ""); err != nil {
		if addErr := s.addAsyncError(ctx, sessionID, userID, v2StepCode, err); addErr != nil {
			return nil, addErr
		}
		return s.fetchDTO(ctx, userID, sessionID, locale)
	}
	return s.fetchDTO(ctx, userID, sessionID, locale)
}

// --------------------------------------------------------------------------
// System prompt assembly
// --------------------------------------------------------------------------

func (s *DebateV2Service) buildSystemPrompt(ctx context.Context, userID uuid.UUID, sess *repository.DebateSession, turns []repository.DebateTurn, stepKey, locale string) (string, error) {
	if stepKey == v2StepIntent {
		return IntentSystemPromptV2(locale), nil
	}
	if strings.HasPrefix(stepKey, v2StepAgentPrefix) {
		agentKey := strings.TrimPrefix(stepKey, v2StepAgentPrefix)
		agent := s.resolveAgent(ctx, userID, agentKey, locale)
		intent := lastAssistantReply(turns, v2StepIntent)
		agents := []string(sess.Agents)
		upstream := make([]UpstreamSummary, 0)
		for _, key := range agents {
			if key == agentKey {
				break
			}
			text := lastAssistantReply(turns, v2StepAgentPrefix+key)
			if strings.TrimSpace(text) == "" {
				continue
			}
			other := s.resolveAgent(ctx, userID, key, locale)
			upstream = append(upstream, UpstreamSummary{Name: other.Name, Text: text})
		}
		return AgentSystemPromptV2(agent, intent, upstream, locale), nil
	}
	return "", errors.New("cannot build system prompt for step " + stepKey)
}

// --------------------------------------------------------------------------
// DTO + persistence helpers
// --------------------------------------------------------------------------

func (s *DebateV2Service) addV2Turn(ctx context.Context, sessionID, userID uuid.UUID, turnType, role, text string, meta v2TurnMeta) (*repository.DebateTurn, error) {
	_ = userID // ownership check already performed by caller via loadSession / repo.GetSession
	t := &repository.DebateTurn{
		SessionID:   sessionID,
		Type:        turnType,
		Role:        role,
		Status:      "approved",
		ContentText: text,
		ContentJSON: marshalV2Meta(meta),
	}
	return s.repo.AddTurn(ctx, t)
}

func (s *DebateV2Service) publishUpdate(sessionID uuid.UUID) {
	if s != nil && s.events != nil {
		s.events.publish(sessionID)
	}
}

func (s *DebateV2Service) addAsyncError(ctx context.Context, sessionID, userID uuid.UUID, stepKey string, err error) error {
	errCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = ctx
	msg := "AI generation failed: " + err.Error()
	turnType := "v2_assistant"
	if stepKey == v2StepCode {
		turnType = "v2_code"
	}
	_, addErr := s.addV2Turn(errCtx, sessionID, userID, turnType, "assistant", msg, v2TurnMeta{StepKey: stepKey, Kind: "reply", Error: err.Error()})
	if addErr == nil {
		s.publishUpdate(sessionID)
	}
	return addErr
}

func (s *DebateV2Service) loadSession(ctx context.Context, userID, sessionID uuid.UUID) (*repository.DebateSession, []repository.DebateTurn, error) {
	row, err := s.repo.GetSession(ctx, sessionID, userID)
	if err != nil {
		return nil, nil, err
	}
	if row == nil {
		return nil, nil, errors.New("session not found")
	}
	if !isV2Status(row.Status) {
		return nil, nil, errors.New("session does not belong to v2 flow")
	}
	turns, err := s.repo.ListTurns(ctx, sessionID)
	if err != nil {
		return nil, nil, err
	}
	return row, turns, nil
}
