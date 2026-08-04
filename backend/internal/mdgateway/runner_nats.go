package mdgateway

import (
	"context"
	"strings"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// startAccountEventSubscriber listens for NATS JetStream account lifecycle
// events and dynamically starts/stops gateways.
func startAccountEventSubscriber(ctx context.Context, deps RunnerDeps, mgr *Manager, log *zap.Logger) {
	if deps.NATSConn == nil {
		return
	}
	js, err := deps.NATSConn.JetStream()
	if err != nil {
		log.Warn("mdgateway: JetStream not available for account events", zap.Error(err))
		return
	}

	if err := ensureAccountEventsStream(js, log); err != nil {
		log.Warn("mdgateway: account events stream ensure failed", zap.Error(err))
		return
	}

	sub, err := js.Subscribe("account.>", func(m *nats.Msg) {
		var nak bool
		defer func() {
			if nak {
				_ = m.Nak()
			} else {
				_ = m.Ack()
			}
		}()
		log.Info("mdgateway: account event received",
			zap.String("subject", m.Subject),
			zap.String("data", string(m.Data)))

		parts := strings.Split(m.Subject, ".")
		if len(parts) < 3 {
			return
		}
		action := parts[1]
		accountID := parts[2]

		switch action {
		case "connect", "reconnect":
			nak = handleAccountConnect(ctx, deps, mgr, log, accountID)
		case "disconnect":
			_ = mgr.RemoveGateway(ctx, accountID)
			log.Info("mdgateway: dynamically stopped gateway", zap.String("account", accountID))
		}
	}, nats.DeliverAll(), nats.AckExplicit())
	if err != nil {
		log.Warn("mdgateway: account event subscribe failed", zap.Error(err))
		return
	}
	go func() {
		<-ctx.Done()
		_ = sub.Unsubscribe()
	}()
	log.Info("mdgateway: account event subscriber started", zap.String("subject", "account.>"))
}

// handleAccountConnect loads account config and starts a gateway.
// Returns true if the message should be nak'd for redelivery (transient failure).
func handleAccountConnect(ctx context.Context, deps RunnerDeps, mgr *Manager, log *zap.Logger, accountID string) bool {
	if mgr.IsDisconnecting(accountID) {
		log.Info("mdgateway: skipping reconnect — account is being disconnected by healthMonitor",
			zap.String("account", accountID))
		return false
	}
	cfg, err := loadSingleAccountConfig(ctx, deps.PG, deps.Secrets, accountID)
	if err != nil || cfg == nil {
		errMsg := "load account config failed"
		if err != nil {
			errMsg = err.Error()
		}
		log.Warn("mdgateway: load account config failed",
			zap.String("account", accountID), zap.Error(err))
		if deps.OnAccountStatus != nil {
			deps.OnAccountStatus(accountID, "", "disconnected", errMsg)
		}
		return false
	}

	log.Info("mdgateway: dynamically starting gateway",
		zap.String("account", accountID), zap.String("platform", cfg.Platform))

	if _, err := startGatewayForAccount(ctx, *cfg, deps, mgr, log); err != nil {
		log.Error("mdgateway: dynamic gateway start failed",
			zap.String("account", accountID), zap.Error(err))
		msg := err.Error()
		if len(msg) > 512 {
			msg = msg[:512]
		}
		if deps.PG != nil {
			_, _ = deps.PG.Exec(ctx,
				`UPDATE mt_accounts SET account_status = 'disconnected',
				 last_error = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND deleted_at IS NULL`,
				accountID, msg)
		}
		if deps.OnAccountStatus != nil {
			deps.OnAccountStatus(accountID, cfg.UserID, "disconnected", msg)
		}
		if isPermanentGatewayError(err) {
			log.Warn("mdgateway: permanent failure — acking to stop redelivery",
				zap.String("account", accountID))
			return false
		}
		return true
	}
	return false
}

// ensureAccountEventsStream creates the JetStream stream for account lifecycle events if it doesn't exist.
func ensureAccountEventsStream(js nats.JetStreamContext, log *zap.Logger) error {
	_, err := js.StreamInfo("ACCOUNT_EVENTS")
	if err == nil {
		return nil // Already exists.
	}
	_, err = js.AddStream(&nats.StreamConfig{
		Name:      "ACCOUNT_EVENTS",
		Subjects:  []string{"account.>"},
		MaxAge:    24 * 3600 * 1e9, // 24h in nanoseconds
		Storage:   nats.FileStorage,
		Replicas:  1,
	})
	if err != nil {
		return err
	}
	log.Info("mdgateway: created JetStream ACCOUNT_EVENTS")
	return nil
}

// isPermanentGatewayError returns true for errors that will not resolve
// by retrying (e.g. Invalid account, wrong password). These should be
// acked to prevent infinite NATS redelivery loops.
func isPermanentGatewayError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Invalid account") ||
		strings.Contains(msg, "code=65") ||
		strings.Contains(msg, "not connected") && strings.Contains(msg, "password")
}
