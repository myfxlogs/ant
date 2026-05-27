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
	// Auth check.
	if _, err := authInterceptor.UserIDFromHTTP(r); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse body.
	var input aigates.PipelineInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid input"}`, http.StatusBadRequest)
		return
	}

	// SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Run pipeline — send SSE event per gate.
	result := aigates.Pipeline(input)
	for _, gate := range result.Gates {
		data, _ := json.Marshal(gate)
		fmt.Fprintf(w, "event: gate\ndata: %s\n\n", data)
		flusher.Flush()
	}

	// Send final event.
	summary := map[string]interface{}{
		"passed":     result.Passed,
		"summary":    result.Summary,
		"first_fail": result.FirstFail,
	}
	data, _ := json.Marshal(summary)
	fmt.Fprintf(w, "event: completed\ndata: %s\n\n", data)
	flusher.Flush()

	s.log.Info("gate pipeline SSE completed", zap.Bool("passed", result.Passed))
}
