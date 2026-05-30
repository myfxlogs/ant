package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/interceptor"
	debateV2 "anttrader/internal/service"
)

// --- SSE HTTP handlers ---

func (h *DebateV2Server) HandleDebateV2AdvanceJobSSE(w http.ResponseWriter, r *http.Request, authInterceptor *interceptor.AuthInterceptor) {
	h.handleJobSSE(w, r, authInterceptor)
}

func (h *DebateV2Server) HandleDebateV2ChatJobSSE(w http.ResponseWriter, r *http.Request, authInterceptor *interceptor.AuthInterceptor) {
	h.handleJobSSE(w, r, authInterceptor)
}

func (h *DebateV2Server) handleJobSSE(w http.ResponseWriter, r *http.Request, authInterceptor *interceptor.AuthInterceptor) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/sse/debate-v2/"), "/")
	if len(parts) < 3 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	jobID, err := uuid.Parse(parts[1])
	if err != nil {
		http.Error(w, "invalid job id", http.StatusBadRequest)
		return
	}
	userID, err := authInterceptor.UserIDFromHTTP(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	ch, err := h.svc.JobChannel(jobID, userID)
	if err != nil {
		http.Error(w, `{"error":"job not running"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(ev)
			if err != nil {
				h.log.Error("handleJobSSE: json marshal failed", zap.Error(err))
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return
			}
			flusher.Flush()
			if ev.Phase == "completed" || ev.Phase == "failed" {
				return
			}
		default:
			// Don't block if neither channel nor context is ready.
		}
	}
}

// --- proto conversion ---

func toProtoSession(sess *debateV2.V2Session) *antv1.DebateV2Session {
	if sess == nil {
		return &antv1.DebateV2Session{}
	}
	pb := &antv1.DebateV2Session{
		Id:          sess.ID,
		Title:       sess.Title,
		Status:      sess.Status,
		CurrentStep: sess.CurrentStep,
		Agents:      sess.Agents,
		Provider:    sess.Provider,
		Model:       sess.Model,
		CreatedAt:   sess.CreatedAt,
		UpdatedAt:   sess.UpdatedAt,
	}
	for _, step := range sess.Steps {
		s := &antv1.DebateV2Step{
			StepKey:   step.StepKey,
			AgentKey:  step.AgentKey,
			AgentName: step.AgentName,
		}
		for _, msg := range step.Messages {
			s.Messages = append(s.Messages, &antv1.DebateV2Message{
				Id:      msg.ID,
				Role:    msg.Role,
				Content: msg.Content,
				Kind:    msg.Kind,
			})
		}
		pb.Steps = append(pb.Steps, s)
	}
	if sess.Code != nil {
		pb.Code = &antv1.DebateV2Code{
			Text:   sess.Code.Text,
			Python: sess.Code.Python,
		}
	}
	if sess.Usage != nil {
		pb.Usage = &antv1.DebateV2Usage{
			PromptTokens:     sess.Usage.PromptTokens,
			CompletionTokens: sess.Usage.CompletionTokens,
			TotalTokens:      sess.Usage.TotalTokens,
		}
	}
	return pb
}
