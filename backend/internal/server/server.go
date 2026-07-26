// Package server provides the ant v2 HTTP server startup.
package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Run starts the ant v2 HTTP server and blocks until ctx is cancelled or a listen error occurs.
func Run(ctx context.Context, handler http.Handler, port string, log *zap.Logger) error {
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      0, // disabled: streaming endpoints (SSE/ConnectRPC server-stream) hold writes open indefinitely
		IdleTimeout:       0, // disabled: streaming connections (SSE/ConnectRPC) must stay open indefinitely
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("ant v2 server starting", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("server: listen: %w", err)
	case <-ctx.Done():
		log.Info("context cancelled, shutting down...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
