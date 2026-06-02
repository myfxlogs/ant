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

	// Ensure the stream exists for account events.
	if err := ensureAccountEventsStream(js, log); err != nil {
		log.Warn("mdgateway: account events stream ensure failed", zap.Error(err))
		return
	}

	// Ephemeral consumer — only active while mdgateway is running.
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
			if mgr.IsDisconnecting(accountID) {
				log.Info("mdgateway: skipping reconnect — account is being disconnected by healthMonitor",
					zap.String("account", accountID))
				return
			}
			cfg, err := loadSingleAccountConfig(ctx, deps.PG, accountID)
			if err != nil || cfg == nil {
				log.Warn("mdgateway: load account config failed",
					zap.String("account", accountID), zap.Error(err))
				return
			}

			log.Info("mdgateway: dynamically starting gateway",
				zap.String("account", accountID), zap.String("platform", cfg.Platform))

			if _, err := startGatewayForAccount(ctx, *cfg, deps, mgr, log); err != nil {
				log.Error("mdgateway: dynamic gateway start failed",
					zap.String("account", accountID), zap.Error(err))
				nak = true // transient failure — request redelivery
			}

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
		sub.Unsubscribe()
	}()
	log.Info("mdgateway: account event subscriber started", zap.String("subject", "account.>"))
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
