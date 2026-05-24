package main

import (
	"context"
	"net/http"
	"os"

	"go.uber.org/zap"

	"anttrader/internal/server"
)

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("ant ok"))
	})

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }

	if err := server.Run(context.Background(), mux, port, log); err != nil {
		log.Fatal("server failed", zap.Error(err))
	}
}
