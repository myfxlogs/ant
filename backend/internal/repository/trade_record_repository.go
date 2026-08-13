package repository

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"alphaforge/internal/model"
)

type TradeRecordRepository struct {
	db *pgxpool.Pool
}

func NewTradeRecordRepository(db *pgxpool.Pool) *TradeRecordRepository {
	return &TradeRecordRepository{db: db}
}

func (r *TradeRecordRepository) Create(ctx context.Context, record *model.TradeRecord) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("create trade record: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := r.insertWithHashChain(ctx, tx, record); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

const maxBatchSize = 500

func (r *TradeRecordRepository) BatchCreate(ctx context.Context, records []*model.TradeRecord) error {
	if len(records) == 0 {
		return nil
	}

	// Split into chunks of maxBatchSize to avoid oversized transactions
	// and excessive lock contention.
	for start := 0; start < len(records); start += maxBatchSize {
		end := start + maxBatchSize
		if end > len(records) {
			end = len(records)
		}
		if err := r.batchCreateChunk(ctx, records[start:end]); err != nil {
			return fmt.Errorf("batch create trade record chunk [%d:%d]: %w", start, end, err)
		}
	}
	return nil
}

func (r *TradeRecordRepository) batchCreateChunk(ctx context.Context, records []*model.TradeRecord) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("batch create trade record: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, record := range records {
		if err := r.insertWithHashChain(ctx, tx, record); err != nil {
			return fmt.Errorf("batch create trade record ticket=%d: %w", record.Ticket, err)
		}
	}

	return tx.Commit(ctx)
}

func (r *TradeRecordRepository) GetByAccountID(ctx context.Context, userID, accountID uuid.UUID, start, end time.Time, limit int) ([]*model.TradeRecord, error) {
	query := `
		SELECT
			id, schedule_id, account_id, ticket, symbol, order_type, volume,
			open_price, close_price, profit, swap, commission,
			open_time, close_time, stop_loss, take_profit, order_comment, magic_number, platform,
			created_at, updated_at, seq, prev_hash, entry_hash
		FROM trade_records
		WHERE user_id = $1 AND account_id = $2 AND close_time >= $3 AND close_time <= $4
		ORDER BY close_time DESC
	`
	args := []interface{}{userID, accountID, start, end}

	if limit > 0 {
		query += " LIMIT $5"
		args = append(args, limit)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*model.TradeRecord
	for rows.Next() {
		var rec model.TradeRecord
		if err := rows.Scan(
			&rec.ID, &rec.ScheduleID, &rec.AccountID, &rec.Ticket, &rec.Symbol, &rec.OrderType,
			&rec.Volume, &rec.OpenPrice, &rec.ClosePrice, &rec.Profit, &rec.Swap, &rec.Commission,
			&rec.OpenTime, &rec.CloseTime, &rec.StopLoss, &rec.TakeProfit,
			&rec.OrderComment, &rec.MagicNumber, &rec.Platform,
			&rec.CreatedAt, &rec.UpdatedAt, &rec.Seq, &rec.PrevHash, &rec.EntryHash,
		); err != nil {
			return nil, err
		}
		records = append(records, &rec)
	}
	return records, rows.Err()
}

// GetByStrategyID returns live trade records linked to a strategy via
// trade_records.schedule_id → strategy_schedules.template_id.
func (r *TradeRecordRepository) GetByStrategyID(ctx context.Context, strategyID uuid.UUID) ([]*model.TradeRecord, error) {
	query := `
		SELECT tr.id, tr.schedule_id, tr.account_id, tr.ticket, tr.symbol, tr.order_type,
		       tr.volume, tr.open_price, tr.close_price, tr.profit, tr.swap, tr.commission,
		       tr.open_time, tr.close_time, tr.stop_loss, tr.take_profit,
		       tr.order_comment, tr.magic_number, tr.platform,
		       tr.created_at, tr.updated_at, tr.seq, tr.prev_hash, tr.entry_hash
		FROM trade_records tr
		JOIN strategy_schedules ss ON tr.schedule_id = ss.id
		WHERE ss.template_id = $1
		ORDER BY tr.open_time ASC
	`
	rows, err := r.db.Query(ctx, query, strategyID)
	if err != nil {
		return nil, fmt.Errorf("get trade records by strategy: %w", err)
	}
	defer rows.Close()

	var records []*model.TradeRecord
	for rows.Next() {
		var rec model.TradeRecord
		if err := rows.Scan(
			&rec.ID, &rec.ScheduleID, &rec.AccountID, &rec.Ticket, &rec.Symbol, &rec.OrderType,
			&rec.Volume, &rec.OpenPrice, &rec.ClosePrice, &rec.Profit, &rec.Swap, &rec.Commission,
			&rec.OpenTime, &rec.CloseTime, &rec.StopLoss, &rec.TakeProfit,
			&rec.OrderComment, &rec.MagicNumber, &rec.Platform,
			&rec.CreatedAt, &rec.UpdatedAt, &rec.Seq, &rec.PrevHash, &rec.EntryHash,
		); err != nil {
			return nil, fmt.Errorf("get trade records by strategy: scan: %w", err)
		}
		records = append(records, &rec)
	}
	return records, rows.Err()
}

func (r *TradeRecordRepository) GetLastSyncTime(ctx context.Context, userID, accountID uuid.UUID) (*time.Time, error) {
	query := `
		SELECT MAX(close_time) FROM trade_records WHERE user_id = $1 AND account_id = $2
	`
	var lastTime *time.Time
	err := r.db.QueryRow(ctx, query, userID, accountID).Scan(&lastTime)
	if err != nil {
		return nil, err
	}
	return lastTime, nil
}

func (r *TradeRecordRepository) CountByAccount(ctx context.Context, userID, accountID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM trade_records WHERE user_id = $1 AND account_id = $2`
	var count int
	err := r.db.QueryRow(ctx, query, userID, accountID).Scan(&count)
	return count, err
}

func (r *TradeRecordRepository) DeleteByAccount(ctx context.Context, userID, accountID uuid.UUID) error {
	query := `DELETE FROM trade_records WHERE user_id = $1 AND account_id = $2`
	_, err := r.db.Exec(ctx, query, userID, accountID)
	if err != nil {
		return fmt.Errorf("delete trade records by account: %w", err)
	}
	return nil
}

// insertWithHashChain inserts a trade record with hash chain linkage.
// Follows the same pattern as wallet_repo.ledgerChainInsert:
// advisory lock → read chain tail → insert → compute entry_hash → update.
// On conflict (upsert), the existing record is kept as-is (hash chain preserved).
func (r *TradeRecordRepository) insertWithHashChain(ctx context.Context, tx pgx.Tx, record *model.TradeRecord) error {
	// Advisory lock to serialize chain operations (same key space as wallet uses different key).
	const tradeChainLockKey = 20827
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, tradeChainLockKey); err != nil {
		return fmt.Errorf("trade hash chain: advisory lock: %w", err)
	}

	// Read chain tail for prev_hash.
	var prevHash []byte
	err := tx.QueryRow(ctx,
		`SELECT entry_hash FROM trade_records ORDER BY seq DESC LIMIT 1`,
	).Scan(&prevHash)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("trade hash chain: read tail: %w", err)
	}

	// Insert with prev_hash. seq is GENERATED ALWAYS AS IDENTITY.
	// ON CONFLICT DO NOTHING: duplicate (account_id, ticket, close_time) is
	// idempotent append-only — the existing row's hash chain is immutable.
	// Only newly inserted rows (RETURNING yields a row) need entry_hash set.
	var returnedID uuid.UUID
	var seq int64
	insertQuery := `
		INSERT INTO trade_records (
			user_id, schedule_id, account_id, ticket, symbol, order_type, volume,
			open_price, close_price, profit, swap, commission,
			open_time, close_time, stop_loss, take_profit,
			order_comment, magic_number, platform, prev_hash
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		) ON CONFLICT (account_id, ticket, close_time) DO NOTHING
		RETURNING id, seq
	`
	err = tx.QueryRow(ctx, insertQuery,
		record.UserID, record.ScheduleID, record.AccountID, record.Ticket, record.Symbol, record.OrderType, record.Volume,
		record.OpenPrice, record.ClosePrice, record.Profit, record.Swap, record.Commission,
		record.OpenTime, record.CloseTime, record.StopLoss, record.TakeProfit,
		record.OrderComment, record.MagicNumber, record.Platform, prevHash,
	).Scan(&returnedID, &seq)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Conflict — row already exists, hash chain preserved. Idempotent skip.
			return nil
		}
		return fmt.Errorf("trade hash chain: insert: %w", err)
	}
	record.ID = returnedID
	record.Seq = seq
	record.PrevHash = prevHash

	// Compute entry_hash = SHA256(prev_hash || seq || account_id || ticket || symbol || volume || open_price || close_price || profit || open_time || close_time).
	entryHash := computeTradeEntryHash(prevHash, seq, record.AccountID, record.Ticket, record.Symbol,
		record.Volume.String(), record.OpenPrice.String(), record.ClosePrice.String(),
		record.Profit.String(), [2]int64{record.OpenTime.UnixMilli(), record.CloseTime.UnixMilli()})

	// Update entry_hash (separate UPDATE because seq is GENERATED ALWAYS AS IDENTITY).
	// Only reached for newly inserted rows — conflict rows returned early above.
	if _, err := tx.Exec(ctx,
		`UPDATE trade_records SET entry_hash = $1 WHERE id = $2`, entryHash, returnedID,
	); err != nil {
		return fmt.Errorf("trade hash chain: set entry_hash: %w", err)
	}
	record.EntryHash = entryHash

	return nil
}

// VerifyChain checks the integrity of the trade record hash chain for a given account.
// Returns a list of ChainBreaks if any tampering is detected.
func (r *TradeRecordRepository) VerifyChain(ctx context.Context, userID, accountID uuid.UUID) ([]model.ChainBreak, error) {
	query := `
		SELECT seq, ticket, prev_hash, entry_hash, account_id, symbol, volume::text,
		       open_price::text, close_price::text, profit::text,
		       open_time, close_time
		FROM trade_records
		WHERE user_id = $1 AND account_id = $2
		ORDER BY seq ASC
	`
	rows, err := r.db.Query(ctx, query, userID, accountID)
	if err != nil {
		return nil, fmt.Errorf("verify chain: query: %w", err)
	}
	defer rows.Close()

	var breaks []model.ChainBreak
	var expectedPrevHash []byte
	for rows.Next() {
		var seq int64
		var ticket int64
		var prevHash, entryHash []byte
		var acctID uuid.UUID
		var symbol, volume, openPrice, closePrice, profit string
		var openTime, closeTime time.Time

		if err := rows.Scan(&seq, &ticket, &prevHash, &entryHash, &acctID, &symbol, &volume,
			&openPrice, &closePrice, &profit, &openTime, &closeTime); err != nil {
			return nil, fmt.Errorf("verify chain: scan: %w", err)
		}

		// Check chain linkage: prev_hash should match the previous record's entry_hash.
		if !bytesEqual(prevHash, expectedPrevHash) {
			breaks = append(breaks, model.ChainBreak{
				Seq:    seq,
				Ticket: ticket,
				Type:   "chain_break",
				Detail: fmt.Sprintf("prev_hash mismatch at seq=%d: expected %x, got %x", seq, expectedPrevHash, prevHash),
			})
		}

		// Recompute entry_hash and compare.
		computed := computeTradeEntryHash(prevHash, seq, acctID, ticket, symbol,
			volume, openPrice, closePrice, profit, [2]int64{openTime.UnixMilli(), closeTime.UnixMilli()})
		if !bytesEqual(entryHash, computed) {
			breaks = append(breaks, model.ChainBreak{
				Seq:    seq,
				Ticket: ticket,
				Type:   "hash_mismatch",
				Detail: fmt.Sprintf("entry_hash mismatch at seq=%d: expected %x, got %x", seq, computed, entryHash),
			})
		}

		expectedPrevHash = entryHash
	}
	return breaks, rows.Err()
}

// computeTradeEntryHash calculates SHA256(prev_hash || seq || account_id || ticket || symbol || volume || open_price || close_price || profit || open_time_ms || close_time_ms).
func computeTradeEntryHash(prevHash []byte, seq int64, accountID uuid.UUID, ticket int64, symbol, volume, openPrice, closePrice, profit string, timeMs [2]int64) []byte {
	h := sha256.New()
	h.Write(prevHash)
	var seqBuf [8]byte
	binary.BigEndian.PutUint64(seqBuf[:], uint64(seq))
	h.Write(seqBuf[:])
	h.Write(accountID[:])
	var ticketBuf [8]byte
	binary.BigEndian.PutUint64(ticketBuf[:], uint64(ticket))
	h.Write(ticketBuf[:])
	h.Write([]byte(symbol))
	h.Write([]byte(volume))
	h.Write([]byte(openPrice))
	h.Write([]byte(closePrice))
	h.Write([]byte(profit))
	var timeBuf [8]byte
	binary.BigEndian.PutUint64(timeBuf[:], uint64(timeMs[0]))
	h.Write(timeBuf[:])
	binary.BigEndian.PutUint64(timeBuf[:], uint64(timeMs[1]))
	h.Write(timeBuf[:])
	return h.Sum(nil)
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
