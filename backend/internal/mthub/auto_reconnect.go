package mthub

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"
)

// isSessionError returns true if the error indicates the MT4/MT5 session
// is invalid or the gRPC connection is broken — conditions where a
// reconnect-and-retry is the correct response.
func isSessionError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Invalid account") ||
		strings.Contains(msg, "not connected") ||
		strings.Contains(msg, "DeadlineExceeded") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "transport is closing") ||
		strings.Contains(msg, "rpc error: code = Unavailable")
}

// reconnectAndRetry attempts to reconnect the gateway for the given account
// and then retries the operation once. If ReconnectGateway is not configured
// or the reconnect fails, the original error is returned.
// A 60-second cooldown per account prevents reconnect loops when the MT4/MT5
// server itself is rejecting sessions.
func (s *MtHubService) reconnectAndRetry(accountID string, op func() error) error {
	if s.hub.ReconnectGateway == nil {
		return op()
	}
	s.reconnectMu.Lock()
	lastAt, ok := s.reconnectLastAt[accountID]
	if ok && Clk.Now().Sub(lastAt) < 60*time.Second {
		s.reconnectMu.Unlock()
		if s.logger != nil {
			s.logger.Warn("mthub: session error — reconnect skipped (cooldown)",
				zap.String("account", accountID),
				zap.Duration("since_last", Clk.Now().Sub(lastAt)))
		}
		return op()
	}
	s.reconnectLastAt[accountID] = Clk.Now()
	s.reconnectMu.Unlock()
	if s.logger != nil {
		s.logger.Warn("mthub: session error — auto-reconnecting gateway",
			zap.String("account", accountID))
	}
	reconnCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.hub.ReconnectGateway(reconnCtx, accountID); err != nil {
		if s.logger != nil {
			s.logger.Error("mthub: auto-reconnect failed",
				zap.String("account", accountID), zap.Error(err))
		}
		return op()
	}
	if s.logger != nil {
		s.logger.Info("mthub: auto-reconnect succeeded, retrying operation",
			zap.String("account", accountID))
	}
	return op()
}
