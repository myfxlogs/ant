package mthub

import (
	"anttrader/internal/mdgateway"

	"google.golang.org/grpc"
)

// Session wraps a mdgateway.Gateway with per-account metadata.
type Session struct {
	AccountID string
	Gateway   mdgateway.Gateway
}

// Platform returns the session platform ("mt4" or "mt5").
func (s *Session) Platform() string { return s.Gateway.Platform() }

// Conn returns the underlying gRPC connection (may be nil).
func (s *Session) Conn() *grpc.ClientConn { return s.Gateway.Conn() }

// SessionID returns the MT session token (empty before Connect).
func (s *Session) SessionID() string { return s.Gateway.SessionID() }

// BrokerID returns the broker UUID for this session.
func (s *Session) BrokerID() string { return s.Gateway.BrokerID() }
