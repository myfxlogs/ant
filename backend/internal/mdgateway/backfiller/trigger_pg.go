package backfiller

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"
)

// PGNotifier abstracts a pgx connection that delivers NOTIFY payloads.
// Real impl wraps pgx.Conn.WaitForNotification; tests use an in-memory channel.
type PGNotifier interface {
	WaitForNotification(ctx context.Context) (channel string, payload string, err error)
	Close() error
}

// BackfillCallback is invoked when a NOTIFY payload arrives. M10.5-8: when a new
// mt_account_subscription is inserted, PG fires NOTIFY new_subscription → backfiller
// triggers an immediate BackfillAccount(accountID) without waiting for the 6h cron.
type BackfillCallback func(ctx context.Context, accountID string) error

type notifyPayload struct {
	AccountID string `json:"account_id"`
}

// PGTrigger is a long-running listener that maps PG NOTIFY events to backfill calls.
type PGTrigger struct {
	log *zap.Logger
	cb  BackfillCallback
}

// NewPGTrigger constructs a trigger. cb is typically Backfiller.BackfillAccount.
func NewPGTrigger(log *zap.Logger, cb BackfillCallback) *PGTrigger {
	return &PGTrigger{log: log, cb: cb}
}

// Run blocks until ctx is canceled or notifier returns a fatal error.
// Each NOTIFY payload is parsed and dispatched to the callback. Callback errors
// are logged but do not stop the listener (best-effort backfill).
func (t *PGTrigger) Run(ctx context.Context, notifier PGNotifier) error {
	if notifier == nil {
		return nil
	}
	defer notifier.Close()
	t.log.Info("backfiller: PG NOTIFY listener active")
	for {
		_, payload, err := notifier.WaitForNotification(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		var np notifyPayload
		if jerr := json.Unmarshal([]byte(payload), &np); jerr != nil {
			t.log.Debug("backfiller: bad NOTIFY payload", zap.String("payload", payload), zap.Error(jerr))
			continue
		}
		if np.AccountID == "" {
			continue
		}
		if cerr := t.cb(ctx, np.AccountID); cerr != nil {
			t.log.Warn("backfiller: PG-triggered backfill failed",
				zap.String("account_id", np.AccountID), zap.Error(cerr))
		} else {
			t.log.Info("backfiller: PG-triggered backfill succeeded",
				zap.String("account_id", np.AccountID))
		}
	}
}
