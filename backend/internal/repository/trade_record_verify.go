package repository

// Trade ledger hash-chain verification — the global union walk.
// VERIFY-CHAIN-SEMANTIC-1: writes chain to the global tail, and archived
// dedup_log rows are chain members; verification therefore walks
// trade_records ∪ trade_record_dedup_log by seq. Extracted from
// trade_record_repository.go to keep the write path and the verifier in
// their own files.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"

	"alphaforge/internal/model"
)

// VerifyGlobalChain walks the trade ledger's hash chain end to end.
// VERIFY-CHAIN-SEMANTIC-1: writes chain to the GLOBAL tail, so verification
// walks the global union — live rows UNION ALL dedup-archived rows (archived
// rows are chain members; their full-column snapshots make deleted links
// whole again). Per row:
//   - linkage: prev_hash must equal the previous union row's entry_hash
//     (first row NULL prev == nil expected: genesis is legal); later gaps →
//     "chain_break".
//   - recompute: two candidate encodings of the same ::text column values —
//     the canonical form itself (covers canonical-era writes, D2) and the
//     normalized decimal form (recovers legacy rows whose write-time float
//     representation differs from the column). Either hit verifies; both
//     missing → "hash_mismatch" (canonical-era rows: definitive tampering;
//     legacy rows: tampering or float-representation drift, indistinguishable).
//   - NULL-hash rows are never silently dropped: they surface as "unhashed"
//     informational findings outside the linkage walk.
func (r *TradeRecordRepository) VerifyGlobalChain(ctx context.Context) ([]model.ChainBreak, error) {
	breaks, err := r.verifyChainLinkage(ctx)
	if err != nil {
		return nil, err
	}
	unhashed, err := r.unhashedFindings(ctx)
	if err != nil {
		return nil, err
	}
	return append(breaks, unhashed...), nil
}

const globalChainUnionQuery = `
	SELECT seq, ticket, prev_hash, entry_hash, account_id, symbol, volume::text,
	       open_price::text, close_price::text, profit::text, open_time, close_time, 'live' AS src
	FROM trade_records
	WHERE entry_hash IS NOT NULL
	UNION ALL
	SELECT seq, ticket, prev_hash, entry_hash, account_id, symbol, volume::text,
	       open_price::text, close_price::text, profit::text, open_time, close_time, 'archived' AS src
	FROM trade_record_dedup_log
	WHERE entry_hash IS NOT NULL
	ORDER BY seq ASC, src ASC
`

const liveChainUnionQuery = `
	SELECT seq, ticket, prev_hash, entry_hash, account_id, symbol, volume::text,
	       open_price::text, close_price::text, profit::text, open_time, close_time, 'live' AS src
	FROM trade_records
	WHERE entry_hash IS NOT NULL
	ORDER BY seq ASC
`

const globalUnhashedUnionQuery = `
	SELECT seq, ticket, account_id, 'live' AS src
	FROM trade_records WHERE entry_hash IS NULL
	UNION ALL
	SELECT seq, ticket, account_id, 'archived' AS src
	FROM trade_record_dedup_log WHERE entry_hash IS NULL
	ORDER BY seq ASC, src ASC
`

const liveUnhashedUnionQuery = `
	SELECT seq, ticket, account_id, 'live' AS src
	FROM trade_records WHERE entry_hash IS NULL
	ORDER BY seq ASC
`

// verifyChainLinkage walks the hashed union rows and reports linkage gaps and
// recompute misses. A missing dedup_log (pre-migration database, or after the
// down migration dropped it) retries with the live side only — breaks then
// report as-is ("no archive basis"); every other error propagates.
func (r *TradeRecordRepository) verifyChainLinkage(ctx context.Context) ([]model.ChainBreak, error) {
	rows, err := r.db.Query(ctx, globalChainUnionQuery)
	if err != nil {
		if !isUndefinedTable(err) {
			return nil, fmt.Errorf("verify global chain: query: %w", err)
		}
		rows, err = r.db.Query(ctx, liveChainUnionQuery)
		if err != nil {
			return nil, fmt.Errorf("verify global chain: query: %w", err)
		}
	}
	defer rows.Close()

	var breaks []model.ChainBreak
	var expectedPrevHash []byte
	for rows.Next() {
		var seq, ticket int64
		var prevHash, entryHash []byte
		var acctID uuid.UUID
		var symbol, volume, openPrice, closePrice, profit, src string
		var openTime, closeTime time.Time

		if err := rows.Scan(&seq, &ticket, &prevHash, &entryHash, &acctID, &symbol, &volume,
			&openPrice, &closePrice, &profit, &openTime, &closeTime, &src); err != nil {
			return nil, fmt.Errorf("verify global chain: scan: %w", err)
		}

		if !bytesEqual(prevHash, expectedPrevHash) {
			breaks = append(breaks, model.ChainBreak{
				Seq:       seq,
				Ticket:    ticket,
				AccountID: acctID,
				Type:      "chain_break",
				Detail: fmt.Sprintf("prev_hash mismatch at seq=%d [%s]: expected %x, got %x",
					seq, src, expectedPrevHash, prevHash),
			})
		}

		if !tradeEntryHashVerifies(entryHash, prevHash, seq, acctID, ticket, symbol, volume, openPrice, closePrice, profit, openTime, closeTime) {
			breaks = append(breaks, model.ChainBreak{
				Seq:       seq,
				Ticket:    ticket,
				AccountID: acctID,
				Type:      "hash_mismatch",
				Detail: fmt.Sprintf("entry_hash mismatch at seq=%d [%s]: no candidate encoding recomputes "+
					"(canonical-era row: definitive tampering; legacy row: tampering or float-representation drift)",
					seq, src),
			})
		}

		expectedPrevHash = entryHash
	}
	return breaks, rows.Err()
}

// unhashedFindings discloses rows without an entry_hash (rows predating the
// chain era, plus any unhashed stragglers) — informational, outside the
// linkage walk. Same 42P01 fallback as the linkage walk.
func (r *TradeRecordRepository) unhashedFindings(ctx context.Context) ([]model.ChainBreak, error) {
	rows, err := r.db.Query(ctx, globalUnhashedUnionQuery)
	if err != nil {
		if !isUndefinedTable(err) {
			return nil, fmt.Errorf("verify global chain: unhashed query: %w", err)
		}
		rows, err = r.db.Query(ctx, liveUnhashedUnionQuery)
		if err != nil {
			return nil, fmt.Errorf("verify global chain: unhashed query: %w", err)
		}
	}
	defer rows.Close()

	var findings []model.ChainBreak
	for rows.Next() {
		var seq, ticket int64
		var acctID uuid.UUID
		var src string
		if err := rows.Scan(&seq, &ticket, &acctID, &src); err != nil {
			return nil, fmt.Errorf("verify global chain: unhashed scan: %w", err)
		}
		findings = append(findings, model.ChainBreak{
			Seq:       seq,
			Ticket:    ticket,
			AccountID: acctID,
			Type:      "unhashed",
			Detail:    fmt.Sprintf("entry_hash NULL [%s] — outside chain linkage, disclosed as-is", src),
		})
	}
	return findings, rows.Err()
}

// isUndefinedTable reports whether err is PostgreSQL 42P01 (relation does
// not exist) — the dedup-log fallback signal.
func isUndefinedTable(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42P01"
}

// tradeEntryHashVerifies recomputes the entry hash over two candidate
// encodings of the same ::text column values (D3): the canonical form itself,
// then the normalized decimal form — the latter recovers legacy rows whose
// write-time `Decimal.String()` differs from the persisted representation.
func tradeEntryHashVerifies(entryHash, prevHash []byte, seq int64, accountID uuid.UUID, ticket int64,
	symbol, volume, openPrice, closePrice, profit string, openTime, closeTime time.Time) bool {
	ms := [2]int64{openTime.UnixMilli(), closeTime.UnixMilli()}
	if bytesEqual(entryHash, computeTradeEntryHash(prevHash, seq, accountID, ticket, symbol,
		volume, openPrice, closePrice, profit, ms)) {
		return true
	}
	normalized := computeTradeEntryHash(prevHash, seq, accountID, ticket, symbol,
		decimal.RequireFromString(volume).String(),
		decimal.RequireFromString(openPrice).String(),
		decimal.RequireFromString(closePrice).String(),
		decimal.RequireFromString(profit).String(), ms)
	return bytesEqual(entryHash, normalized)
}

// VerifyChain returns the global chain findings that involve the given
// account. VERIFY-CHAIN-SEMANTIC-1: the chain is global (writes chain to the
// global tail — the old per-account walk produced 2,113 baseline false
// breaks), so verification walks the global union and filters; "该账户涉及的断点"
// is the honest per-account semantic. userID is kept for signature stability
// only — account_id is globally unique.
func (r *TradeRecordRepository) VerifyChain(ctx context.Context, userID, accountID uuid.UUID) ([]model.ChainBreak, error) {
	all, err := r.VerifyGlobalChain(ctx)
	if err != nil {
		return nil, err
	}
	filtered := make([]model.ChainBreak, 0, len(all))
	for _, b := range all {
		if b.AccountID == accountID {
			filtered = append(filtered, b)
		}
	}
	return filtered, nil
}
