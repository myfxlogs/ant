package sentry

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

// Init initializes the global Sentry client if SENTRY_DSN is set.
// Returns a cleanup function that flushes pending events.
// If SENTRY_DSN is empty, Sentry is disabled and the cleanup is a no-op.
func Init(log *zap.Logger) func() {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		log.Info("sentry: DSN not set, error tracking disabled")
		return func() {}
	}

	env := os.Getenv("SENTRY_ENVIRONMENT")
	if env == "" {
		env = "development"
	}

	release := os.Getenv("SENTRY_RELEASE")
	if release == "" {
		release = "alphaforge@dev"
	}

	sampleRate := 1.0
	if os.Getenv("SENTRY_SAMPLE_RATE") == "0" {
		sampleRate = 0
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      env,
		Release:          release,
		TracesSampleRate: sampleRate,
		AttachStacktrace: true,
		EnableTracing:    false,
	})
	if err != nil {
		log.Error("sentry: init failed", zap.Error(err))
		return func() {}
	}

	log.Info("sentry: initialized", zap.String("env", env), zap.String("release", release))

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if !sentry.Flush(5 * time.Second) {
			log.Warn("sentry: flush timed out")
		}
		_ = ctx
	}
}

// CaptureError captures a non-fatal error with optional context tags.
func CaptureError(err error, tags ...map[string]string) {
	if err == nil {
		return
	}
	hub := sentry.CurrentHub()
	scope := hub.Scope()
	for _, t := range tags {
		for k, v := range t {
			scope.SetTag(k, v)
		}
	}
	hub.CaptureException(err)
}

// CapturePanic recovers from a panic, reports it to Sentry, and re-panics.
// Use in goroutines to prevent silent crashes.
func CapturePanic(log *zap.Logger) {
	if r := recover(); r != nil {
		err := fmt.Errorf("panic: %v", r)
		sentry.CurrentHub().Recover(r)
		sentry.Flush(2 * time.Second)
		log.Error("panic recovered", zap.Error(err))
	}
}
