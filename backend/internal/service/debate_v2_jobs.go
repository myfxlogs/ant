package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"anttrader/internal/repository"

	"github.com/google/uuid"
)

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
	extras, err := s.loadExtras(ctx, uid, userID)
	if err != nil {
		return uuid.Nil, err
	}
	steps, err := s.parseSteps(extras)
	if err != nil {
		return uuid.Nil, err
	}
	if len(steps) == 0 {
		return uuid.Nil, fmt.Errorf("session has no steps")
	}
	last := &steps[len(steps)-1]
	last.Messages = append(last.Messages, V2Message{
		ID:      uuid.NewString(),
		Role:    "user",
		Content: message,
	})
	if err := s.saveSteps(ctx, uid, userID, steps, sess.Status); err != nil {
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

func (s *DebateV2Service) RunChatJob(jobID, callerUserID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	pair, exists := s.jobSessions[jobID]
	if !exists {
		return fmt.Errorf("job %s not found", jobID)
	}
	if pair[1] != callerUserID {
		return fmt.Errorf("job %s not found", jobID)
	}
	if _, running := s.jobChans[jobID]; running {
		return fmt.Errorf("job %s is already running", jobID)
	}
	ch := make(chan *debateV2JobEvent, 64)
	s.jobChans[jobID] = ch
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
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		sess, err := s.repo.GetSession(bgCtx, sessionID, userID)
		if err != nil || sess == nil {
			return
		}
		extras, err := s.loadExtras(bgCtx, sessionID, userID)
		if err != nil {
			return
		}
		steps, err := s.parseSteps(extras)
		if err != nil {
			s.log.Error("RunChatJob goroutine: parseSteps failed", zap.Error(err))
			return
		}
		if len(steps) > 0 {
			last := &steps[len(steps)-1]
			last.Messages = append(last.Messages, V2Message{
				ID:      uuid.NewString(),
				Role:    "assistant",
				Content: msg,
			})
			if err := s.saveSteps(bgCtx, sessionID, userID, steps, sess.Status); err != nil {
				s.log.Error("RunChatJob goroutine: saveSteps failed", zap.String("session_id", sessionID.String()), zap.Error(err))
			}
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
