package rest

import "net/http"

func RegisterHealth(mux *http.ServeMux) {
	if mux == nil {
		return
	}
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
}
