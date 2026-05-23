// Package symbolsync — service entry point.
// Ported from alfq. Orchestrates symbol fetching, canonical normalization, and PG upsert.
package symbolsync

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// Service orchestrates symbol sync for a broker account.
type Service struct {
	repo   *Repo
	mapper *CanonicalMapper
	log    *zap.Logger
}

// NewService creates a symbol sync service.
func NewService(db *sqlx.DB, log *zap.Logger) *Service {
	return &Service{
		repo:   NewRepo(db),
		mapper: NewCanonicalMapper(db),
		log:    log,
	}
}

// SyncParams holds the data needed to sync symbols for an account.
type SyncParams struct {
	BrokerID  string // broker UUID from accounts.broker_id
	Platform  string // "MT4" or "MT5"
	SessionID string // gRPC session token
	// Fetcher is a function that returns all BrokerSymbols for this connection.
	// The service applies Canonicalize() + dict validation on top.
	Fetcher func(ctx context.Context) ([]BrokerSymbol, error)
}

// Sync pulls symbol metadata and upserts to broker_symbols.
// Dispatch per platform: MT4 or MT5 path.
func (s *Service) Sync(ctx context.Context, params SyncParams) error {
	platform := strings.ToUpper(params.Platform)
	s.log.Info("symbol sync started",
		zap.String("broker_id", params.BrokerID),
		zap.String("platform", platform),
	)

	// Fetch raw symbols from the MT gateway
	symbols, err := params.Fetcher(ctx)
	if err != nil {
		return fmt.Errorf("symbolsync: fetch: %w", err)
	}

	// Normalize each symbol: apply Canonicalize() + dict validation
	for i := range symbols {
		symbols[i].BrokerID = params.BrokerID
		canon, partial := s.mapper.Resolve(ctx, symbols[i].SymbolRaw)
		symbols[i].Canonical = canon
		symbols[i].Partial = partial
	}

	// Batch upsert to broker_symbols
	if err := s.repo.UpsertSymbols(ctx, symbols); err != nil {
		return fmt.Errorf("symbolsync: upsert: %w", err)
	}

	s.log.Info("symbol sync complete",
		zap.Int("total", len(symbols)),
		zap.String("broker_id", params.BrokerID),
	)
	return nil
}

// RefreshCache reloads the canonical mapper cache from broker_symbols.
func (s *Service) RefreshCache(ctx context.Context) error {
	return s.mapper.RefreshCache(ctx)
}
