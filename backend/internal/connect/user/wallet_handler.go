package user

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/service"
)

// WalletServer implements ant.v1.WalletServiceHandler.
type WalletServer struct {
	svc *service.WalletService
	log *zap.Logger
}

var _ antv1c.WalletServiceHandler = (*WalletServer)(nil)

func NewWalletServer(svc *service.WalletService, log *zap.Logger) *WalletServer {
	return &WalletServer{svc: svc, log: log}
}

// GetWallet returns the current user's wallet (auto-creates for legacy users).
func (s *WalletServer) GetWallet(ctx context.Context, req *connect.Request[antv1.GetWalletRequest]) (*connect.Response[antv1.GetWalletResponse], error) {
	uid, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	w, err := s.svc.GetOrCreateWallet(ctx, uid)
	if err != nil {
		s.log.Error("GetWallet: service error", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var acctNum string
	if w.AccountNumber != nil {
		acctNum = *w.AccountNumber
	}
	return connect.NewResponse(&antv1.GetWalletResponse{
		Wallet: &antv1.Wallet{
			Id: w.ID.String(), UserId: w.UserID.String(),
			Balance: w.Balance, FrozenBalance: w.FrozenBalance,
			Currency: w.Currency, AccountNumber: acctNum,
		},
	}), nil
}

// ListTransactions returns the current user's wallet transaction history.
func (s *WalletServer) ListTransactions(ctx context.Context, req *connect.Request[antv1.ListWalletTransactionsRequest]) (*connect.Response[antv1.ListWalletTransactionsResponse], error) {
	uid, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	page := int(req.Msg.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.Msg.PageSize)
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	txs, total, err := s.svc.ListTransactions(ctx, uid, page, pageSize)
	if err != nil {
		s.log.Error("ListTransactions: service error", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	pbTxs := make([]*antv1.WalletTransaction, len(txs))
	for i, t := range txs {
		desc := ""
		if t.Description != nil {
			desc = *t.Description
		}
		opID := ""
		if t.OperatorID != nil {
			opID = t.OperatorID.String()
		}
		pbTxs[i] = &antv1.WalletTransaction{
			Id: t.ID.String(), TxType: t.TxType, Amount: t.Amount,
			BalanceBefore: t.BalanceBefore, BalanceAfter: t.BalanceAfter,
			Description: desc, OperatorId: opID,
			CreatedAtTsMs: t.CreatedAt.UnixMilli(),
		}
	}
	return connect.NewResponse(&antv1.ListWalletTransactionsResponse{
		Transactions: pbTxs, Total: total,
	}), nil
}

// AdjustBalance allows admin to credit/debit a user's wallet.
func (s *WalletServer) AdjustBalance(ctx context.Context, req *connect.Request[antv1.AdjustBalanceRequest]) (*connect.Response[antv1.AdjustBalanceResponse], error) {
	operatorIDStr := interceptor.GetUserID(ctx)
	operatorID, err := uuid.Parse(operatorIDStr)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid operator id"))
	}
	r := req.Msg
	userID, err := uuid.Parse(r.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid user_id"))
	}

	txType := "adjustment"
	if len(r.Amount) > 0 && r.Amount[0] == '-' {
		txType = "withdrawal"
	}

	w, err := s.svc.AdjustBalance(ctx, userID, r.Amount, txType, r.Description, &operatorID)
	if err != nil {
		if err == service.ErrWalletNotFound {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("wallet not found"))
		}
		s.log.Error("AdjustBalance: service error", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var acctNum string
	if w.AccountNumber != nil {
		acctNum = *w.AccountNumber
	}
	return connect.NewResponse(&antv1.AdjustBalanceResponse{
		Wallet: &antv1.Wallet{
			Id: w.ID.String(), UserId: w.UserID.String(),
			Balance: w.Balance, FrozenBalance: w.FrozenBalance,
			Currency: w.Currency, AccountNumber: acctNum,
		},
	}), nil
}
