package ai

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"
)

// CreditRepo is the interface for credit account/transaction operations.
// Implemented by *repository.CreditRepository.
type CreditRepo interface {
	GetOrCreateAccount(ctx context.Context, userID uuid.UUID) (*repository.CreditAccount, error)
	GetBalance(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error)
	AddCredits(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, txType, source, description string, operatorID *uuid.UUID) (*repository.CreditTransaction, error)
	HoldCredits(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, sessionID, description string) (*repository.CreditTransaction, error)
	SettleCredits(ctx context.Context, userID uuid.UUID, holdAmount, actualCost decimal.Decimal, description string) error
	ListTransactions(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*repository.CreditTransaction, int64, error)
}

// CreditServer implements CreditServiceHandler (user-facing).
type CreditServer struct {
	creditSvc  *service.CreditService
	creditRepo CreditRepo
	log        *zap.Logger
}

func NewCreditServer(creditSvc *service.CreditService, creditRepo CreditRepo, log *zap.Logger) *CreditServer {
	return &CreditServer{creditSvc: creditSvc, creditRepo: creditRepo, log: log}
}

var _ antv1c.CreditServiceHandler = (*CreditServer)(nil)

func (s *CreditServer) GetCreditBalance(ctx context.Context, req *connect.Request[antv1.GetCreditBalanceRequest]) (*connect.Response[antv1.GetCreditBalanceResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	acc, err := s.creditRepo.GetOrCreateAccount(ctx, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
	}
	return connect.NewResponse(&antv1.GetCreditBalanceResponse{
		Balance:       acc.Balance,
		FrozenBalance: acc.FrozenBalance,
	}), nil
}

func (s *CreditServer) ListCreditTransactions(ctx context.Context, req *connect.Request[antv1.ListCreditTransactionsRequest]) (*connect.Response[antv1.ListCreditTransactionsResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	txs, total, err := s.creditRepo.ListTransactions(ctx, uid, int(req.Msg.Page), int(req.Msg.PageSize))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
	}
	out := make([]*antv1.CreditTransaction, len(txs))
	for i, t := range txs {
		out[i] = txToProto(t)
	}
	return connect.NewResponse(&antv1.ListCreditTransactionsResponse{
		Transactions: out,
		Total:        total,
	}), nil
}

// AdminCreditServer implements AdminCreditServiceHandler.
type AdminCreditServer struct {
	creditRepo CreditRepo
	log        *zap.Logger
}

func NewAdminCreditServer(creditRepo CreditRepo, log *zap.Logger) *AdminCreditServer {
	return &AdminCreditServer{creditRepo: creditRepo, log: log}
}

var _ antv1c.AdminCreditServiceHandler = (*AdminCreditServer)(nil)

func (s *AdminCreditServer) AddCredits(ctx context.Context, req *connect.Request[antv1.AddCreditsRequest]) (*connect.Response[antv1.AddCreditsResponse], error) {
	uid, err := uuid.Parse(req.Msg.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid user_id"))
	}
	amount, err := parseDecimal(req.Msg.Amount)
	if err != nil || !amount.IsPositive() {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("amount must be positive"))
	}
	adminID, _ := userIDFromCtx(ctx)
	tx, err := s.creditRepo.AddCredits(ctx, uid, amount, "deposit", "admin_manual", req.Msg.Description, &adminID)
	if err != nil {
		s.log.Error("admin add credits failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
	}
	return connect.NewResponse(&antv1.AddCreditsResponse{
		NewBalance: tx.BalanceAfter,
	}), nil
}

func (s *AdminCreditServer) RefundCredits(ctx context.Context, req *connect.Request[antv1.RefundCreditsRequest]) (*connect.Response[antv1.RefundCreditsResponse], error) {
	uid, err := uuid.Parse(req.Msg.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid user_id"))
	}
	amount, err := parseDecimal(req.Msg.Amount)
	if err != nil || !amount.IsPositive() {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("amount must be positive"))
	}
	adminID, _ := userIDFromCtx(ctx)
	negAmount := amount.Neg()
	tx, err := s.creditRepo.AddCredits(ctx, uid, negAmount, "refund", "admin_refund", req.Msg.Description, &adminID)
	if err != nil {
		s.log.Error("admin refund credits failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("refund failed — check user has sufficient balance"))
	}
	return connect.NewResponse(&antv1.RefundCreditsResponse{
		NewBalance: tx.BalanceAfter,
	}), nil
}

func (s *AdminCreditServer) ListAllCreditTransactions(ctx context.Context, req *connect.Request[antv1.ListAllCreditTransactionsRequest]) (*connect.Response[antv1.ListAllCreditTransactionsResponse], error) {
	if req.Msg.UserId != "" {
		uid, err := uuid.Parse(req.Msg.UserId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid user_id"))
		}
		txs, total, err := s.creditRepo.ListTransactions(ctx, uid, int(req.Msg.Page), int(req.Msg.PageSize))
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
		}
		out := make([]*antv1.CreditTransaction, len(txs))
		for i, t := range txs {
			out[i] = txToProto(t)
		}
		return connect.NewResponse(&antv1.ListAllCreditTransactionsResponse{
			Transactions: out,
			Total:        total,
		}), nil
	}
	// No user filter — return empty for now (platform-wide listing needs separate query).
	return connect.NewResponse(&antv1.ListAllCreditTransactionsResponse{}), nil
}

func txToProto(t *repository.CreditTransaction) *antv1.CreditTransaction {
	p := &antv1.CreditTransaction{
		Id:            t.ID.String(),
		TxType:        t.TxType,
		Amount:        t.Amount,
		BalanceBefore: t.BalanceBefore,
		BalanceAfter:  t.BalanceAfter,
		CreatedAtTsMs: t.CreatedAt.UnixMilli(),
	}
	if t.Source != nil {
		p.Source = *t.Source
	}
	if t.Description != nil {
		p.Description = *t.Description
	}
	return p
}
