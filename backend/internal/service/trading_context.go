package service

import (
	"context"
	"strings"
)

type tradeTriggerSourceKey struct{}

const (
	TriggerSourceManual   = "manual"
	TriggerSourceStrategy = "strategy"
	TriggerSourceRecovery = "recovery"
)

func WithTradeTriggerSource(ctx context.Context, source string) context.Context {
	normalized := normalizeTriggerSource(source)
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, tradeTriggerSourceKey{}, normalized)
}

func TradeTriggerSourceFromContext(ctx context.Context) string {
	if ctx == nil {
		return TriggerSourceManual
	}
	if v, ok := ctx.Value(tradeTriggerSourceKey{}).(string); ok {
		return normalizeTriggerSource(v)
	}
	return TriggerSourceManual
}

func normalizeTriggerSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case TriggerSourceStrategy:
		return TriggerSourceStrategy
	case TriggerSourceRecovery:
		return TriggerSourceRecovery
	default:
		return TriggerSourceManual
	}
}
