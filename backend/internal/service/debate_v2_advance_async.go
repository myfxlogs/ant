package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/ai"
	"anttrader/internal/repository"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

// StartAdvanceJob moves to the next step and runs LLM work (agent kickoff or
// code generation) in the background when that step requires it.
func (s *DebateV2Service) StartAdvanceJob(ctx context.Context, userID, sessionID uuid.UUID, locale string) (jobID string, err error) {
	sess, _, err := s.loadSession(ctx, userID, sessionID)
	if err != nil {
		return "", err
	}
	current := stepKeyFromStatus(sess.Status)
	if current == "" || current == v2StepDone {
		return "", errors.New("cannot advance from the current step")
	}
	if current == v2StepCode {
		return "", errors.New("already at code step; use AdvanceDebateV2 to finish")
	}
	agents := []string(sess.Agents)
	nextStep := nextStepKey(current, agents)
	if nextStep == "" {
		return "", errors.New("no further step")
	}
	asyncLLM := strings.HasPrefix(nextStep, v2StepAgentPrefix) || nextStep == v2StepCode
	if !asyncLLM {
		return "", errors.New("next step does not require async advance")
	}
	if err := s.repo.UpdateSession(ctx, sessionID, userID, &repository.SessionPatch{
		Status: strPtr(statusFromStepKey(nextStep)),
	}); err != nil {
		return "", err
	}
	s.publishUpdate(sessionID)
	sess, turns, err := s.loadSession(ctx, userID, sessionID)
	if err != nil {
		return "", err
	}
	jid := uuid.NewString()
	s.advanceJobs.put(jid, userID, sessionID)
	go s.runAdvanceAsyncJob(jid, userID, sessionID, locale, nextStep, sess, turns)
	return jid, nil
}

func (s *DebateV2Service) runAdvanceAsyncJob(jobID string, userID, sessionID uuid.UUID, locale, nextStep string, sess *repository.DebateSession, turns []repository.DebateTurn) {
	bg := context.Background()
	s.advanceJobs.setPhase(jobID, "running", "")
	s.advanceJobs.emit(jobID, `{"event":"running"}`)
	stopKA := s.advanceJobs.startKeepalive(jobID)
	defer stopKA()
	var err error
	switch {
	case strings.HasPrefix(nextStep, v2StepAgentPrefix):
		err = s.runAgentKickoff(bg, userID, sess, turns, nextStep, locale, jobID)
	case nextStep == v2StepCode:
		err = s.runCodeGenerationWithFeedback(bg, userID, sess, turns, locale, "", jobID)
	default:
		err = errors.New("unexpected next step")
	}
	if err != nil {
		if addErr := s.addAsyncError(bg, sessionID, userID, nextStep, err); addErr != nil {
			err = addErr
		}
		line, _ := json.Marshal(map[string]string{"event": "failed", "message": err.Error()})
		s.advanceJobs.emit(jobID, string(line))
		s.advanceJobs.setPhase(jobID, "failed", err.Error())
	} else {
		s.advanceJobs.emit(jobID, `{"event":"completed"}`)
		s.advanceJobs.setPhase(jobID, "completed", "")
	}
	s.publishUpdate(sessionID)
	s.advanceJobs.scheduleRemove(jobID, 2*time.Minute)
}

// StartRejectCodeJob persists reject feedback and runs code regeneration in the background (same advance SSE path).
func (s *DebateV2Service) StartRejectCodeJob(ctx context.Context, userID, sessionID uuid.UUID, feedback, locale string) (jobID string, err error) {
	fb := strings.TrimSpace(feedback)
	if fb == "" {
		return "", errors.New("feedback is required")
	}
	sess, _, err := s.loadSession(ctx, userID, sessionID)
	if err != nil {
		return "", err
	}
	current := stepKeyFromStatus(sess.Status)
	if current != v2StepCode && current != v2StepDone {
		return "", errors.New("reject is only valid after code has been generated")
	}
	if _, err := s.addV2Turn(ctx, sessionID, userID, "v2_user", "user", fb, v2TurnMeta{StepKey: v2StepCode, Kind: "reject"}); err != nil {
		return "", err
	}
	s.publishUpdate(sessionID)
	if sess.Status != v2StatusCode {
		if err := s.repo.UpdateSession(ctx, sessionID, userID, &repository.SessionPatch{Status: strPtr(v2StatusCode)}); err != nil {
			return "", err
		}
	}
	sess, turns, err := s.loadSession(ctx, userID, sessionID)
	if err != nil {
		return "", err
	}
	jid := uuid.NewString()
	s.advanceJobs.put(jid, userID, sessionID)
	go s.runRejectCodeAsyncJob(jid, userID, sessionID, locale, fb, sess, turns)
	return jid, nil
}

func (s *DebateV2Service) runRejectCodeAsyncJob(jobID string, userID, sessionID uuid.UUID, locale, feedback string, sess *repository.DebateSession, turns []repository.DebateTurn) {
	bg := context.Background()
	s.advanceJobs.setPhase(jobID, "running", "")
	s.advanceJobs.emit(jobID, `{"event":"running"}`)
	stopKA := s.advanceJobs.startKeepalive(jobID)
	defer stopKA()
	err := s.runCodeGenerationWithFeedback(bg, userID, sess, turns, locale, feedback, jobID)
	if err != nil {
		if addErr := s.addAsyncError(bg, sessionID, userID, v2StepCode, err); addErr != nil {
			err = addErr
		}
		line, _ := json.Marshal(map[string]string{"event": "failed", "message": err.Error()})
		s.advanceJobs.emit(jobID, string(line))
		s.advanceJobs.setPhase(jobID, "failed", err.Error())
	} else {
		s.advanceJobs.emit(jobID, `{"event":"completed"}`)
		s.advanceJobs.setPhase(jobID, "completed", "")
	}
	s.publishUpdate(sessionID)
	s.advanceJobs.scheduleRemove(jobID, 2*time.Minute)
}

// GetAdvanceJobStatus returns job phase (same user only); optional single read for terminal reconcile, not a poll loop.
func (s *DebateV2Service) GetAdvanceJobStatus(jobID string, userID uuid.UUID) (phase, msg, sessionID string, err error) {
	return s.advanceJobs.get(jobID, userID)
}

// SubscribeAdvanceJobEvents streams JSON event lines for one job (SSE layer).
func (s *DebateV2Service) SubscribeAdvanceJobEvents(jobID string, userID uuid.UUID) (<-chan string, func(), error) {
	return s.advanceJobs.subscribe(jobID, userID)
}

// emitAdvanceChunk pushes a token delta to SSE subscribers (QuantDinger-style UX).
func (s *DebateV2Service) emitAdvanceChunk(jobID, delta string) {
	if s == nil || jobID == "" || delta == "" || s.advanceJobs == nil {
		return
	}
	s.advanceJobs.emitChunk(jobID, delta)
}

func (s *DebateV2Service) runCodeGenerationStreamed(ctx context.Context, userID uuid.UUID, sess *repository.DebateSession, turns []repository.DebateTurn, locale, feedback, jobID string) error {
	streamStart := time.Now()
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
	msgs := []ai.Message{{Role: "system", Content: sys}, {Role: "user", Content: userMsg}}
	emit := func(delta string) { s.emitAdvanceChunk(jobID, delta) }
	text, provider, usage, err := streamChatWithFallback(ctx, providers, msgs, emit)
	if err != nil {
		logger.Warn("debate_v2_code_gen_stream_failed",
			zap.String("job_id", jobID),
			zap.String("session_id", sess.ID.String()),
			zap.Duration("duration", time.Since(streamStart)),
			zap.Int("system_prompt_chars", len(sys)),
			zap.Error(err))
		return err
	}
	text = strings.TrimSpace(text)
	python := ExtractPythonBlockV2(text)
	meta := providerUsageMeta(provider, &ai.Response{Content: text, Usage: usage})
	meta.StepKey = v2StepCode
	meta.Python = python
	if _, err := s.addV2Turn(ctx, sess.ID, userID, "v2_code", "assistant", text, meta); err != nil {
		return err
	}
	logger.Info("debate_v2_code_gen_stream_done",
		zap.String("job_id", jobID),
		zap.String("session_id", sess.ID.String()),
		zap.String("provider", provider.GetProviderName()),
		zap.String("model", provider.GetModelName()),
		zap.Duration("duration", time.Since(streamStart)),
		zap.Int("system_prompt_chars", len(sys)),
		zap.Int("raw_chars", len(text)),
		zap.Int("python_fence_chars", len(python)),
		zap.Int("prompt_tokens", usage.PromptTokens),
		zap.Int("completion_tokens", usage.CompletionTokens),
		zap.Int("total_tokens", usage.TotalTokens))
	s.publishUpdate(sess.ID)
	return nil
}
