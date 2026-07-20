package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// LedgerShipper reads ledger_outbox entries and forwards them to external
// notification channels (email/Telegram) for tamper-evident off-host record (R8).
// Uses PostgreSQL LISTEN/NOTIFY for push-based notification with a fallback
// ticker every 10s to catch any missed notifications.
type LedgerShipper struct {
	pg       *pgxpool.Pool
	notifier Notifier
	log      *zap.Logger
	interval time.Duration
	batch    int
}

// Notifier is the minimal interface for sending ledger entries off-host.
// SendTo sends to a specific recipient (used by WebAuthn 2FA confirmations).
type Notifier interface {
	Send(subject, body string) error
	SendTo(recipient, subject, body string) error
}

// NewLedgerShipper creates a ledger outbox shipper.
func NewLedgerShipper(pg *pgxpool.Pool, notifier Notifier, log *zap.Logger) *LedgerShipper {
	return &LedgerShipper{
		pg:       pg,
		notifier: notifier,
		log:      log,
		interval: 10 * time.Second,
		batch:    100,
	}
}

// Run starts the shipper loop. Blocks until ctx is cancelled.
// Uses PostgreSQL LISTEN/NOTIFY for push-based notification instead of polling.
// A fallback ticker runs every 10s to catch any missed notifications.
func (s *LedgerShipper) Run(ctx context.Context) {
	s.log.Info("ledger shipper: started")

	// Start fallback ticker in case LISTEN misses a notification.
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Start LISTEN goroutine.
	listenDone := make(chan struct{})
	go s.runListen(ctx, listenDone)

	for {
		select {
		case <-ctx.Done():
			s.log.Info("ledger shipper: stopped")
			<-listenDone
			return
		case <-ticker.C:
			if err := s.shipBatch(ctx); err != nil {
				s.log.Error("ledger shipper: fallback batch error", zap.Error(err))
			}
		}
	}
}

func (s *LedgerShipper) shipBatch(ctx context.Context) error {
	rows, err := s.pg.Query(ctx, `
		SELECT id, seq, entry_hash
		FROM ledger_outbox
		WHERE sent_at IS NULL
		ORDER BY seq ASC
		LIMIT $1
	`, s.batch)
	if err != nil {
		return fmt.Errorf("ledger shipper: query: %w", err)
	}
	defer rows.Close()

	type entry struct {
		id   string
		seq  int64
		hash string
	}
	var entries []entry

	for rows.Next() {
		var e entry
		var hashBytes []byte
		if err := rows.Scan(&e.id, &e.seq, &hashBytes); err != nil {
			return fmt.Errorf("ledger shipper: scan: %w", err)
		}
		e.hash = hex.EncodeToString(hashBytes)
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("ledger shipper: rows: %w", err)
	}

	if len(entries) == 0 {
		return nil
	}

	// Build summary and send notification.
	body := fmt.Sprintf("Ledger entries (%d):\n", len(entries))
	for _, e := range entries {
		body += fmt.Sprintf("  seq=%d hash=%s\n", e.seq, e.hash)
	}

	if s.notifier != nil {
		if err := s.notifier.Send("Ledger Update", body); err != nil {
			s.log.Error("ledger shipper: notify failed", zap.Error(err))
			// Don't mark as sent — will retry next batch.
			return nil
		}
	}

	// Mark all as sent in a single batch UPDATE.
	ids := make([]string, len(entries))
	for i, e := range entries {
		ids[i] = e.id
	}
	_, err = s.pg.Exec(ctx,
		`UPDATE ledger_outbox SET sent_at = NOW() WHERE id = ANY($1)`, ids)
	if err != nil {
		s.log.Error("ledger shipper: batch mark sent", zap.Error(err))
	}

	s.log.Info("ledger shipper: shipped entries", zap.Int("count", len(entries)))
	return nil
}

// runListen acquires a dedicated connection, LISTENs on the ledger_outbox
// channel, and ships entries whenever a NOTIFY is received. This provides
// push-based notification instead of relying solely on the fallback ticker.
// On connection loss, automatically reconnects with exponential backoff.
func (s *LedgerShipper) runListen(ctx context.Context, done chan struct{}) {
	defer close(done)

	backoff := time.Second
	const maxBackoff = 30 * time.Second

	for {
		if ctx.Err() != nil {
			return
		}

		conn, err := s.pg.Acquire(ctx)
		if err != nil {
			s.log.Error("ledger shipper: LISTEN acquire failed", zap.Error(err))
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		if _, err := conn.Exec(ctx, `LISTEN ledger_outbox`); err != nil {
			s.log.Error("ledger shipper: LISTEN failed", zap.Error(err))
			conn.Release()
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		s.log.Info("ledger shipper: LISTEN connected")
		backoff = time.Second

		for {
			_, err := conn.Conn().WaitForNotification(ctx)
			if err != nil {
				if ctx.Err() != nil {
					conn.Release()
					return
				}
				s.log.Error("ledger shipper: WaitForNotification, reconnecting", zap.Error(err))
				break
			}
			if err := s.shipBatch(ctx); err != nil {
				s.log.Error("ledger shipper: LISTEN batch error", zap.Error(err))
			}
		}
		conn.Release()
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}
		backoff = min(backoff*2, maxBackoff)
	}
}
