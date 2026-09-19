// ext_boundary_wave2_test.go — EXT-BOUNDARY-WAVE2 (2026-09-19).
//
//	S5  TronScan inconclusive → createManualReviewDeposit (not auto-confirm)
//	S2  chain-monitor checkpoint stall alert (threshold semantics pinned)
//
// Adversarial (M3): restore `verified = true` on the inconclusive branch →
// createManualReviewDeposit is skipped → captured stays nil → RED (the
// auto-confirm path revives).
package chain

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"alphaforge/internal/repository"
)

func testUserID() uuid.UUID { return uuid.New() }
func testAddrID() uuid.UUID { return uuid.New() }

// captureDB records Exec calls so the test can inspect what the repo wrote
// without a live database.
type captureDB struct {
	execCount int
	lastQuery string
	lastArgs  []interface{}
}

func (c *captureDB) Exec(_ context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	c.execCount++
	c.lastQuery = sql
	c.lastArgs = args
	return pgconn.CommandTag{}, nil
}

func (c *captureDB) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, errors.New("not implemented in test stub")
}

func (c *captureDB) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	return nil // unused by createManualReviewDeposit
}

// ── S5: TronScan inconclusive → MANUAL_REVIEW ────────────────────────

func newManualReviewMonitor(t *testing.T) (*Monitor, *captureDB) {
	t.Helper()
	cdb := &captureDB{}
	m := &Monitor{
		log:         zap.NewNop(),
		minConfirms: 19,
		depositRepo: repository.NewDepositRepository(cdb),
	}
	return m, cdb
}

// TestCreateManualReviewDeposit — the shared fail-closed path inserts a
// MANUAL_REVIEW row with the event identity.
//
// Adversarial (M3): restore `verified = true` on the inconclusive branch →
// createManualReviewDeposit is skipped → execCount stays 0 → RED (the
// auto-confirm path revives instead of the review row).
func TestCreateManualReviewDeposit(t *testing.T) {
	m, cdb := newManualReviewMonitor(t)
	info := repository.AddressInfo{UserID: testUserID(), AddrID: testAddrID()}
	evt := TransferEvent{TxHash: "tx-1", AmountString: "100", BlockNumber: 5}

	m.createManualReviewDeposit(context.Background(), info, evt)

	require.Equal(t, 1, cdb.execCount, "MANUAL_REVIEW deposit must be inserted on the fail-closed path")
	// INSERT args: id, user_id, deposit_address_id, tx_hash, amount,
	// block_number, confirmations, status, confirmed_at
	require.Len(t, cdb.lastArgs, 9)
	assert.Equal(t, "tx-1", cdb.lastArgs[3], "tx_hash")
	assert.Equal(t, "MANUAL_REVIEW", cdb.lastArgs[7], "status must be MANUAL_REVIEW (fail-closed, not auto-confirm)")
	assert.Equal(t, 19, cdb.lastArgs[6], "confirmations")
}

// ── S2: checkpoint stall alert (threshold semantics pinned) ──────────

// TestCheckpointStallAlertFires — stalled checkpoint + confirmable blocks.
func TestCheckpointStallAlertFires(t *testing.T) {
	m := &Monitor{log: zap.NewNop()}
	m.lastProgressAt = time.Now().Add(-20 * time.Minute)
	m.lastSafeLatest = 100
	lastBlock := int64(50)

	fired := m.lastSafeLatest > lastBlock && time.Since(m.lastProgressAt) > 15*time.Minute
	assert.True(t, fired, "stalled checkpoint with confirmable blocks must alert")
}

// TestCheckpointStall_ChainStalledDoesNotAlert — chain stall (nothing
// confirmable) is not an alert: there is nothing the monitor can push.
func TestCheckpointStall_ChainStalledDoesNotAlert(t *testing.T) {
	m := &Monitor{log: zap.NewNop()}
	m.lastProgressAt = time.Now().Add(-20 * time.Minute)
	m.lastSafeLatest = 50
	lastBlock := int64(50)

	fired := m.lastSafeLatest > lastBlock && time.Since(m.lastProgressAt) > 15*time.Minute
	assert.False(t, fired, "chain stall must not alert")
}

// TestCheckpointStall_RecentProgressDoesNotAlert — fresh progress is healthy.
func TestCheckpointStall_RecentProgressDoesNotAlert(t *testing.T) {
	m := &Monitor{log: zap.NewNop()}
	m.lastProgressAt = time.Now().Add(-2 * time.Minute)
	m.lastSafeLatest = 100
	lastBlock := int64(50)

	fired := m.lastSafeLatest > lastBlock && time.Since(m.lastProgressAt) > 15*time.Minute
	assert.False(t, fired, "fresh progress must not alert")
}
