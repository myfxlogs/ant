package ai

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/repository"
)

// ── Mock CreditRepo ──

type mockCreditRepo struct {
	balance       decimal.Decimal
	frozen        decimal.Decimal
	transactions  []*repository.CreditTransaction
	addCreditsErr error
	holdErr       error
	settleErr     error
	lastAmount    decimal.Decimal
	lastTxType    string
}

func (m *mockCreditRepo) GetOrCreateAccount(ctx context.Context, userID uuid.UUID) (*repository.CreditAccount, error) {
	return &repository.CreditAccount{
		ID:            uuid.New(),
		UserID:        userID,
		Balance:       m.balance.StringFixed(8),
		FrozenBalance: m.frozen.StringFixed(8),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}, nil
}

func (m *mockCreditRepo) GetBalance(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	return m.balance, nil
}

func (m *mockCreditRepo) AddCredits(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, txType, source, description string, operatorID *uuid.UUID) (*repository.CreditTransaction, error) {
	if m.addCreditsErr != nil {
		return nil, m.addCreditsErr
	}
	m.lastAmount = amount
	m.lastTxType = txType
	newBal := m.balance.Add(amount)
	m.balance = newBal
	return &repository.CreditTransaction{
		ID:            uuid.New(),
		TxType:        txType,
		Amount:        amount.StringFixed(8),
		BalanceBefore: m.balance.StringFixed(8),
		BalanceAfter:  newBal.StringFixed(8),
		CreatedAt:     time.Now(),
	}, nil
}

func (m *mockCreditRepo) HoldCredits(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, description string) (*repository.CreditTransaction, error) {
	if m.holdErr != nil {
		return nil, m.holdErr
	}
	return &repository.CreditTransaction{
		ID:      uuid.New(),
		TxType:  "ai_hold",
		Amount:  amount.StringFixed(8),
		CreatedAt: time.Now(),
	}, nil
}

func (m *mockCreditRepo) SettleCredits(ctx context.Context, userID uuid.UUID, holdAmount, actualCost decimal.Decimal, description string) error {
	return m.settleErr
}

func (m *mockCreditRepo) ListTransactions(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*repository.CreditTransaction, int64, error) {
	return m.transactions, int64(len(m.transactions)), nil
}

func ctxWithUserID(userID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), interceptor.UserIDKey, userID.String())
}

// ── txToProto tests ──

func TestTxToProto_FullFields(t *testing.T) {
	id := uuid.New()
	src := "admin_manual"
	desc := "test deposit"
	tx := &repository.CreditTransaction{
		ID:            id,
		TxType:        "deposit",
		Amount:        "100.00000000",
		BalanceBefore: "0.00000000",
		BalanceAfter:  "100.00000000",
		Source:        &src,
		Description:   &desc,
		CreatedAt:     time.Now(),
	}
	p := txToProto(tx)
	if p.Id != id.String() {
		t.Fatalf("id mismatch")
	}
	if p.TxType != "deposit" {
		t.Fatalf("tx_type mismatch")
	}
	if p.Amount != "100.00000000" {
		t.Fatalf("amount mismatch")
	}
	if p.Source != "admin_manual" {
		t.Fatalf("source mismatch")
	}
	if p.Description != "test deposit" {
		t.Fatalf("description mismatch")
	}
}

func TestTxToProto_NilFields(t *testing.T) {
	tx := &repository.CreditTransaction{
		ID:        uuid.New(),
		TxType:    "ai_usage",
		Amount:    "5.00000000",
		CreatedAt: time.Now(),
	}
	p := txToProto(tx)
	if p.Source != "" {
		t.Fatalf("source should be empty")
	}
	if p.Description != "" {
		t.Fatalf("description should be empty")
	}
}

// ── CreditServer tests ──

func TestGetCreditBalance_Success(t *testing.T) {
	repo := &mockCreditRepo{balance: decimal.NewFromInt(500), frozen: decimal.NewFromInt(15)}
	srv := NewCreditServer(nil, repo, zap.NewNop())
	uid := uuid.New()
	ctx := ctxWithUserID(uid)
	resp, err := srv.GetCreditBalance(ctx, connect.NewRequest(&antv1.GetCreditBalanceRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Balance != "500.00000000" {
		t.Fatalf("balance mismatch: %s", resp.Msg.Balance)
	}
	if resp.Msg.FrozenBalance != "15.00000000" {
		t.Fatalf("frozen mismatch: %s", resp.Msg.FrozenBalance)
	}
}

func TestGetCreditBalance_NoUserID(t *testing.T) {
	repo := &mockCreditRepo{}
	srv := NewCreditServer(nil, repo, zap.NewNop())
	_, err := srv.GetCreditBalance(context.Background(), connect.NewRequest(&antv1.GetCreditBalanceRequest{}))
	if err == nil {
		t.Fatal("should error without user ID")
	}
}

func TestListCreditTransactions_Success(t *testing.T) {
	repo := &mockCreditRepo{
		transactions: []*repository.CreditTransaction{
			{ID: uuid.New(), TxType: "deposit", Amount: "100.00000000", CreatedAt: time.Now()},
			{ID: uuid.New(), TxType: "ai_usage", Amount: "5.00000000", CreatedAt: time.Now()},
		},
	}
	srv := NewCreditServer(nil, repo, zap.NewNop())
	uid := uuid.New()
	ctx := ctxWithUserID(uid)
	resp, err := srv.ListCreditTransactions(ctx, connect.NewRequest(&antv1.ListCreditTransactionsRequest{Page: 1, PageSize: 20}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Transactions) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(resp.Msg.Transactions))
	}
	if resp.Msg.Total != 2 {
		t.Fatalf("expected total 2, got %d", resp.Msg.Total)
	}
}

func TestListCreditTransactions_Empty(t *testing.T) {
	repo := &mockCreditRepo{}
	srv := NewCreditServer(nil, repo, zap.NewNop())
	uid := uuid.New()
	ctx := ctxWithUserID(uid)
	resp, err := srv.ListCreditTransactions(ctx, connect.NewRequest(&antv1.ListCreditTransactionsRequest{Page: 1, PageSize: 20}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Transactions) != 0 {
		t.Fatalf("expected 0 transactions")
	}
}

// ── AdminCreditServer tests ──

func TestAddCredits_Success(t *testing.T) {
	repo := &mockCreditRepo{balance: decimal.NewFromInt(0)}
	srv := NewAdminCreditServer(repo, zap.NewNop())
	uid := uuid.New()
	adminID := uuid.New()
	ctx := ctxWithUserID(adminID)
	resp, err := srv.AddCredits(ctx, connect.NewRequest(&antv1.AddCreditsRequest{
		UserId:      uid.String(),
		Amount:      "500",
		Description: "test top-up",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.NewBalance == "" {
		t.Fatal("new balance should not be empty")
	}
	if !repo.lastAmount.Equals(decimal.NewFromInt(500)) {
		t.Fatalf("amount should be 500, got %s", repo.lastAmount.String())
	}
	if repo.lastTxType != "deposit" {
		t.Fatalf("tx_type should be deposit, got %s", repo.lastTxType)
	}
}

func TestAddCredits_InvalidUserID(t *testing.T) {
	repo := &mockCreditRepo{}
	srv := NewAdminCreditServer(repo, zap.NewNop())
	ctx := ctxWithUserID(uuid.New())
	_, err := srv.AddCredits(ctx, connect.NewRequest(&antv1.AddCreditsRequest{
		UserId: "not-a-uuid",
		Amount: "100",
	}))
	if err == nil {
		t.Fatal("should error on invalid user ID")
	}
}

func TestAddCredits_NegativeAmount(t *testing.T) {
	repo := &mockCreditRepo{}
	srv := NewAdminCreditServer(repo, zap.NewNop())
	uid := uuid.New()
	ctx := ctxWithUserID(uuid.New())
	_, err := srv.AddCredits(ctx, connect.NewRequest(&antv1.AddCreditsRequest{
		UserId: uid.String(),
		Amount: "-50",
	}))
	if err == nil {
		t.Fatal("should error on negative amount")
	}
}

func TestAddCredits_ZeroAmount(t *testing.T) {
	repo := &mockCreditRepo{}
	srv := NewAdminCreditServer(repo, zap.NewNop())
	uid := uuid.New()
	ctx := ctxWithUserID(uuid.New())
	_, err := srv.AddCredits(ctx, connect.NewRequest(&antv1.AddCreditsRequest{
		UserId: uid.String(),
		Amount: "0",
	}))
	if err == nil {
		t.Fatal("should error on zero amount")
	}
}

func TestAddCredits_RepoError(t *testing.T) {
	repo := &mockCreditRepo{addCreditsErr: fmt.Errorf("db down")}
	srv := NewAdminCreditServer(repo, zap.NewNop())
	uid := uuid.New()
	ctx := ctxWithUserID(uuid.New())
	_, err := srv.AddCredits(ctx, connect.NewRequest(&antv1.AddCreditsRequest{
		UserId: uid.String(),
		Amount: "100",
	}))
	if err == nil {
		t.Fatal("should error on repo failure")
	}
}

func TestRefundCredits_Success(t *testing.T) {
	repo := &mockCreditRepo{balance: decimal.NewFromInt(500)}
	srv := NewAdminCreditServer(repo, zap.NewNop())
	uid := uuid.New()
	ctx := ctxWithUserID(uuid.New())
	resp, err := srv.RefundCredits(ctx, connect.NewRequest(&antv1.RefundCreditsRequest{
		UserId: uid.String(),
		Amount: "200",
		Description: "test refund",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.NewBalance == "" {
		t.Fatal("new balance should not be empty")
	}
	if !repo.lastAmount.Equals(decimal.NewFromInt(-200)) {
		t.Fatalf("amount should be -200, got %s", repo.lastAmount.String())
	}
	if repo.lastTxType != "refund" {
		t.Fatalf("tx_type should be refund, got %s", repo.lastTxType)
	}
}

func TestRefundCredits_InvalidUserID(t *testing.T) {
	repo := &mockCreditRepo{}
	srv := NewAdminCreditServer(repo, zap.NewNop())
	ctx := ctxWithUserID(uuid.New())
	_, err := srv.RefundCredits(ctx, connect.NewRequest(&antv1.RefundCreditsRequest{
		UserId: "bad-uuid",
		Amount: "100",
	}))
	if err == nil {
		t.Fatal("should error on invalid user ID")
	}
}

func TestRefundCredits_NegativeAmount(t *testing.T) {
	repo := &mockCreditRepo{}
	srv := NewAdminCreditServer(repo, zap.NewNop())
	uid := uuid.New()
	ctx := ctxWithUserID(uuid.New())
	_, err := srv.RefundCredits(ctx, connect.NewRequest(&antv1.RefundCreditsRequest{
		UserId: uid.String(),
		Amount: "-10",
	}))
	if err == nil {
		t.Fatal("should error on negative amount")
	}
}

func TestRefundCredits_RepoError(t *testing.T) {
	repo := &mockCreditRepo{addCreditsErr: fmt.Errorf("insufficient balance")}
	srv := NewAdminCreditServer(repo, zap.NewNop())
	uid := uuid.New()
	ctx := ctxWithUserID(uuid.New())
	_, err := srv.RefundCredits(ctx, connect.NewRequest(&antv1.RefundCreditsRequest{
		UserId: uid.String(),
		Amount: "100",
	}))
	if err == nil {
		t.Fatal("should error on repo failure")
	}
}

func TestListAllCreditTransactions_WithUserID(t *testing.T) {
	repo := &mockCreditRepo{
		transactions: []*repository.CreditTransaction{
			{ID: uuid.New(), TxType: "deposit", Amount: "100.00000000", CreatedAt: time.Now()},
		},
	}
	srv := NewAdminCreditServer(repo, zap.NewNop())
	uid := uuid.New()
	resp, err := srv.ListAllCreditTransactions(context.Background(), connect.NewRequest(&antv1.ListAllCreditTransactionsRequest{
		UserId: uid.String(),
		Page:   1,
		PageSize: 20,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(resp.Msg.Transactions))
	}
}

func TestListAllCreditTransactions_InvalidUserID(t *testing.T) {
	repo := &mockCreditRepo{}
	srv := NewAdminCreditServer(repo, zap.NewNop())
	_, err := srv.ListAllCreditTransactions(context.Background(), connect.NewRequest(&antv1.ListAllCreditTransactionsRequest{
		UserId: "bad-uuid",
	}))
	if err == nil {
		t.Fatal("should error on invalid user ID")
	}
}

func TestListAllCreditTransactions_NoUserID(t *testing.T) {
	repo := &mockCreditRepo{}
	srv := NewAdminCreditServer(repo, zap.NewNop())
	resp, err := srv.ListAllCreditTransactions(context.Background(), connect.NewRequest(&antv1.ListAllCreditTransactionsRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Transactions) != 0 {
		t.Fatalf("expected 0 transactions for no user filter")
	}
}

// ── parseDecimal tests ──

func TestParseDecimal_Valid(t *testing.T) {
	d, err := parseDecimal("123.45")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !d.Equals(decimal.NewFromFloat(123.45)) {
		t.Fatalf("expected 123.45, got %s", d.String())
	}
}

func TestParseDecimal_Invalid(t *testing.T) {
	_, err := parseDecimal("not-a-number")
	if err == nil {
		t.Fatal("should error on invalid decimal")
	}
}

func TestParseDecimal_Empty(t *testing.T) {
	_, err := parseDecimal("")
	if err == nil {
		t.Fatal("should error on empty string")
	}
}
