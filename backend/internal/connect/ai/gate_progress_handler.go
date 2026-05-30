package ai

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	aigates "anttrader/internal/ai"
	"anttrader/internal/interceptor"
)

// GateProgressServer handles SSE streaming of AI gate pipeline evaluation.
type GateProgressServer struct {
	log *zap.Logger
}

// NewGateProgressServer creates a GateProgressServer.
func NewGateProgressServer(log *zap.Logger) *GateProgressServer {
	return &GateProgressServer{log: log}
}

// HandleGateProgressSSE runs the 6-gate pipeline and streams results via SSE.
func (s *GateProgressServer) HandleGateProgressSSE(w http.ResponseWriter, r *http.Request, authInterceptor *interceptor.AuthInterceptor) {
	// Auth check — save userID for logging.
	userID, err := authInterceptor.UserIDFromHTTP(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse body with size limit (1 MB max) to prevent memory exhaustion.
	const maxBodySize = 1 << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	var input aigates.PipelineInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid input or body too large"}`, http.StatusBadRequest)
		return
	}

	// Flusher check BEFORE SSE headers so Content-Type is not set prematurely.
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Run pipeline — send SSE event per gate. Check for client disconnect.
	result := aigates.Pipeline(input)
	for _, gate := range result.Gates {
		select {
		case <-r.Context().Done():
			return
		default:
		}
		data, err := json.Marshal(gate)
		if err != nil {
			s.log.Error("HandleGateProgressSSE: json marshal gate failed", zap.Error(err))
			continue
		}
		if _, err := fmt.Fprintf(w, "event: gate\ndata: %s\n\n", data); err != nil {
			return
		}
		flusher.Flush()
	}

	// Send final event.
	// PipelineResult already carries json tags matching the SSE contract.
	data, err := json.Marshal(result)
	if err != nil {
		s.log.Error("HandleGateProgressSSE: json marshal summary failed", zap.Error(err))
		return
	}
	if _, err := fmt.Fprintf(w, "event: completed\ndata: %s\n\n", data); err != nil {
		return
	}
	flusher.Flush()

	s.log.Info("gate pipeline SSE completed", zap.String("user_id", userID.String()), zap.Bool("passed", result.Passed))
}
