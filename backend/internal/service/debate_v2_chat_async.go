package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/repository"
)

var (
	// ErrChatJobNotRunnable is returned when Run is invalid for the job state (wrong user, missing, or already started).
	ErrChatJobNotRunnable = errors.New("chat job is not runnable")
)

// PrepareChatJob persists the user turn and registers a chat job in queued state without starting the LLM.
// The browser must open the SSE stream, then call RunChatJob.
func (s *DebateV2Service) PrepareChatJob(ctx context.Context, userID, sessionID uuid.UUID, message, locale string) (jobID string, err error) {
	msg := strings.TrimSpace(message)
	if msg == "" {
		return "", errors.New("message is required")
	}
	sess, _, err := s.loadSession(ctx, userID, sessionID)
	if err != nil {
		return "", err
	}
	stepKey := stepKeyFromStatus(sess.Status)
	if stepKey == "" || stepKey == v2StepCode || stepKey == v2StepDone {
		return "", errors.New("chat is not allowed in the current step")
	}

	if _, err := s.addV2Turn(ctx, sessionID, userID, "v2_user", "user", msg, v2TurnMeta{StepKey: stepKey, Kind: "reply"}); err != nil {
		return "", err
	}
	s.publishUpdate(sessionID)

	jid := uuid.NewString()
	s.chatJobs.putPreparedChatJob(jid, userID, sessionID, locale, stepKey)
	return jid, nil
}

// RunChatJob starts the background LLM stream for a prepared chat job (must be phase=queued).
func (s *DebateV2Service) RunChatJob(ctx context.Context, userID uuid.UUID, jobID string) error {
	jid := strings.TrimSpace(jobID)
	if jid == "" {
		return errors.New("job_id is required")
	}
	sid, locale, stepKey, ok := s.chatJobs.tryConsumeQueuedChat(jid, userID)
	if !ok {
		return ErrChatJobNotRunnable
	}
	s.chatJobs.emit(jid, `{"event":"running"}`)
	sess, turns, err := s.loadSession(ctx, userID, sid)
	if err != nil {
		line, _ := json.Marshal(map[string]string{"event": "failed", "message": err.Error()})
		s.chatJobs.emit(jid, string(line))
		s.chatJobs.setPhase(jid, "failed", err.Error())
		s.publishUpdate(sid)
		s.chatJobs.scheduleRemove(jid, 2*time.Minute)
		return err
	}
	go s.runChatInvokeBody(jid, userID, sid, locale, stepKey, sess, turns)
	return nil
}

func (s *DebateV2Service) runChatInvokeBody(jobID string, userID, sessionID uuid.UUID, locale, stepKey string, sess *repository.DebateSession, turns []repository.DebateTurn) {
	bg := context.Background()
	stopKA := s.chatJobs.startKeepalive(jobID)
	defer stopKA()
	emitChunk := func(delta string) { s.emitChatChunk(jobID, delta) }
	reply, usage, err := s.invokeStep(bg, userID, sess, turns, stepKey, locale, emitChunk)
	if err != nil {
		if addErr := s.addAsyncError(bg, sessionID, userID, stepKey, err); addErr != nil {
			err = addErr
		}
		line, _ := json.Marshal(map[string]string{"event": "failed", "message": err.Error()})
		s.chatJobs.emit(jobID, string(line))
		s.chatJobs.setPhase(jobID, "failed", err.Error())
		s.publishUpdate(sessionID)
		s.chatJobs.scheduleRemove(jobID, 2*time.Minute)
		return
	}
	reply = StripCodeBlocksV2(reply)
	meta := usage
	meta.StepKey = stepKey
	meta.Kind = "reply"
	if _, err := s.addV2Turn(bg, sessionID, userID, "v2_assistant", "assistant", reply, meta); err != nil {
		line, _ := json.Marshal(map[string]string{"event": "failed", "message": err.Error()})
		s.chatJobs.emit(jobID, string(line))
		s.chatJobs.setPhase(jobID, "failed", err.Error())
		s.publishUpdate(sessionID)
		s.chatJobs.scheduleRemove(jobID, 2*time.Minute)
		return
	}
	s.publishUpdate(sessionID)
	s.chatJobs.emit(jobID, `{"event":"completed"}`)
	s.chatJobs.setPhase(jobID, "completed", "")
	s.chatJobs.scheduleRemove(jobID, 2*time.Minute)
}

// GetChatJobStatus returns job phase (same user only); optional single read for terminal reconcile, not a poll loop.
func (s *DebateV2Service) GetChatJobStatus(jobID string, userID uuid.UUID) (phase, msg, sessionID string, err error) {
	return s.chatJobs.get(jobID, userID)
}

// SubscribeChatJobEvents streams JSON event lines for one chat job (SSE layer).
func (s *DebateV2Service) SubscribeChatJobEvents(jobID string, userID uuid.UUID) (<-chan string, func(), error) {
	return s.chatJobs.subscribe(jobID, userID)
}

func (s *DebateV2Service) emitChatChunk(jobID, delta string) {
	if s == nil || jobID == "" || delta == "" || s.chatJobs == nil {
		return
	}
	s.chatJobs.emitChunk(jobID, delta)
}
