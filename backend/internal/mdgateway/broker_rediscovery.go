package mdgateway

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"alphaforge/internal/mdgateway/adapter/brokersearch"
)

// ConnErrorClass classifies a connection error into one of three tiers
// to decide whether broker host rediscovery should be triggered.
type ConnErrorClass int

const (
	// ErrAuth: wrong password, invalid account, investor-only — stop immediately.
	ErrAuth ConnErrorClass = iota
	// ErrHost: connection refused, DNS failure — host is wrong, trigger rediscovery.
	ErrHost
	// ErrTransient: timeout, EOF, broker service unavailable — host is correct,
	// broker is down. Do NOT rediscover; let existing backoff/retry handle it.
	ErrTransient
)

// ClassifyConnError inspects the error message and network type to classify it.
func ClassifyConnError(err error) ConnErrorClass {
	if err == nil {
		return ErrTransient
	}
	msg := strings.ToLower(err.Error())

	// Tier 1: Auth errors — stop immediately, no retry.
	authKeywords := []string{
		"invalid_account", "code=1001", "code=8",
		"invalid_credentials", "wrong password", "access denied",
		"invalid password", "not authorized", "account disabled",
	}
	for _, kw := range authKeywords {
		if strings.Contains(msg, kw) {
			return ErrAuth
		}
	}

	// Tier 2: Host errors — connection refused or DNS resolution failure.
	// These indicate the broker_host is wrong/stale → trigger rediscovery.
	if isHostError(err, msg) {
		return ErrHost
	}

	// Tier 3: Transient — timeout, EOF, broker unavailable, context deadline.
	// Host is correct, broker is temporarily down. No rediscovery.
	return ErrTransient
}

// isHostError returns true if the error indicates the host itself is wrong
// (DNS failure, connection refused, no such host).
func isHostError(err error, msg string) bool {
	if strings.Contains(msg, "no such host") || strings.Contains(msg, "dns") {
		return true
	}

	if strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connect: connection refused") ||
		strings.Contains(msg, "no route to host") ||
		strings.Contains(msg, "network is unreachable") {
		return true
	}

	// gRPC dial errors with "refused" indicate host is wrong.
	if strings.Contains(msg, "name resolution") ||
		strings.Contains(msg, "failed to connect") ||
		(strings.Contains(msg, "dial tcp") && strings.Contains(msg, "refused")) {
		return true
	}

	return false
}

// HostRediscoverer handles broker host rediscovery when a connection fails
// due to a host error (not auth, not transient). It searches for the broker
// company, tries each returned host, and on success updates the DB.
type HostRediscoverer struct {
	searcher *brokersearch.Searcher
	pg       *pgxpool.Pool
	log      *zap.Logger
}

// NewHostRediscoverer creates a rediscoverer. searcher and pg may be nil
// (rediscovery will be skipped in that case).
func NewHostRediscoverer(searcher *brokersearch.Searcher, pg *pgxpool.Pool, log *zap.Logger) *HostRediscoverer {
	return &HostRediscoverer{searcher: searcher, pg: pg, log: log}
}

// MaybeRediscover checks if the error warrants rediscovery, and if so,
// searches for the broker company, tries each host via tryConnect,
// and on success updates mt_accounts.broker_host.
//
// Returns:
//   - (newHost, nil) if rediscovery succeeded and host was updated
//   - ("", originalErr) if rediscovery was not triggered (auth/transient error)
//   - ("", rediscoveryErr) if rediscovery was triggered but failed
func (r *HostRediscoverer) MaybeRediscover(
	ctx context.Context,
	originalErr error,
	brokerCompany, platform, accountID string,
	tryConnect func(host string) error,
) (string, error) {
	class := ClassifyConnError(originalErr)

	switch class {
	case ErrAuth:
		// Auth failure — stop immediately.
		return "", originalErr
	case ErrTransient:
		// Transient — let existing backoff handle it.
		return "", originalErr
	case ErrHost:
		// Host error — attempt rediscovery.
	}

	if r.searcher == nil || brokerCompany == "" {
		r.log.Warn("broker_rediscovery: cannot rediscover — searcher or broker_company missing",
			zap.String("account", accountID),
			zap.String("brokerCompany", brokerCompany))
		return "", originalErr
	}

	r.log.Info("broker_rediscovery: host error detected, searching for fresh hosts",
		zap.String("account", accountID),
		zap.String("brokerCompany", brokerCompany),
		zap.String("platform", platform),
		zap.Error(originalErr))

	companies, err := r.searcher.Search(ctx, brokerCompany, platform)
	if err != nil {
		r.log.Warn("broker_rediscovery: search failed",
			zap.String("account", accountID), zap.Error(err))
		return "", fmt.Errorf("rediscovery search: %w (original: %v)", err, originalErr)
	}

	for _, c := range companies {
		for _, s := range c.GetServers() {
			for _, host := range s.GetAccess() {
				if host == "" {
					continue
				}
				if err := tryConnect(host); err != nil {
					r.log.Debug("broker_rediscovery: host attempt failed",
						zap.String("host", host), zap.Error(err))
					continue
				}

				// Success — update DB if we have a pool and accountID.
				if r.pg != nil && accountID != "" {
					if _, err := r.pg.Exec(ctx,
						`UPDATE mt_accounts SET broker_host = $1, updated_at = CURRENT_TIMESTAMP
						 WHERE id = $2 AND deleted_at IS NULL`,
						host, accountID); err != nil {
						r.log.Warn("broker_rediscovery: DB update failed",
							zap.String("account", accountID),
							zap.String("newHost", host), zap.Error(err))
					} else {
						r.log.Info("broker_rediscovery: updated broker_host in DB",
							zap.String("account", accountID),
							zap.String("newHost", host))
					}
				}

				return host, nil
			}
		}
	}

	r.log.Warn("broker_rediscovery: no working host found after trying all candidates",
		zap.String("account", accountID),
		zap.String("brokerCompany", brokerCompany))
	return "", fmt.Errorf("no working host after rediscovery (original: %v)", originalErr)
}
