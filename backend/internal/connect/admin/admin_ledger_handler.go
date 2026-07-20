package admin

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// GetLedgerSummary returns the platform's total liabilities and hash chain tip
// for off-host solvency verification (ADR-0026 R10).
func (s *AdminBillingServer) GetLedgerSummary(ctx context.Context, req *connect.Request[antv1.GetLedgerSummaryRequest]) (*connect.Response[antv1.GetLedgerSummaryResponse], error) {
	// Total liabilities = SUM(balance + frozen_balance) across all user wallets.
	var totalLiabilities string
	err := s.repo.DB().QueryRow(ctx,
		`SELECT COALESCE(SUM(balance + frozen_balance), 0)::text FROM user_wallets`,
	).Scan(&totalLiabilities)
	if err != nil {
		s.log.Error("GetLedgerSummary: total liabilities", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("query failed"))
	}

	// Hash chain tip: latest seq + entry_hash + created_at.
	var latestSeq int64
	var entryHashBytes []byte
	var createdAt *time.Time
	err = s.repo.DB().QueryRow(ctx,
		`SELECT seq, entry_hash, created_at
		 FROM wallet_transactions
		 ORDER BY seq DESC LIMIT 1`,
	).Scan(&latestSeq, &entryHashBytes, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No transactions yet — return zeros.
			return connect.NewResponse(&antv1.GetLedgerSummaryResponse{
				TotalLiabilities:  totalLiabilities,
				LatestSeq:         0,
				LatestEntryHash:   "",
				LatestEntryTime:   nil,
			}), nil
		}
		s.log.Error("GetLedgerSummary: chain tip", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("query failed"))
	}

	resp := &antv1.GetLedgerSummaryResponse{
		TotalLiabilities:  totalLiabilities,
		LatestSeq:         latestSeq,
		LatestEntryHash:   hex.EncodeToString(entryHashBytes),
	}
	if createdAt != nil {
		resp.LatestEntryTime = timestamppb.New(*createdAt)
	}

	return connect.NewResponse(resp), nil
}
