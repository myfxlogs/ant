package server

import (
	"fmt"
	"net/http"

	"anttrader/internal/interceptor"
	"anttrader/internal/service"
)

func registerDebateV2AdvanceSSE(mux *http.ServeMux, auth *interceptor.AuthInterceptor, debate *service.DebateV2Service) {
	mux.Handle("GET /antrader/sse/debate-v2/advance-jobs/{jobId}/stream", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth == nil || debate == nil {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}
		userID, err := auth.UserIDFromHTTP(r)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		jobID := r.PathValue("jobId")
		if jobID == "" {
			http.Error(w, "missing job", http.StatusBadRequest)
			return
		}
		ch, unsub, err := debate.SubscribeAdvanceJobEvents(jobID, userID)
		if err != nil {
			if err.Error() == "forbidden" {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}
		defer unsub()

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		fl, _ := w.(http.Flusher)

		for {
			select {
			case <-r.Context().Done():
				return
			case line, ok := <-ch:
				if !ok {
					return
				}
				_, _ = fmt.Fprintf(w, "data: %s\n\n", line)
				if fl != nil {
					fl.Flush()
				}
			}
		}
	}))
}
