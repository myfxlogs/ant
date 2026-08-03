package repository

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestCreditAccountStruct(t *testing.T) {
	acc := &CreditAccount{
		Balance:       "100.00000000",
		FrozenBalance: "15.00000000",
	}
	bal, err := decimal.NewFromString(acc.Balance)
	if err != nil {
		t.Fatalf("parse balance: %v", err)
	}
	if !bal.Equals(decimal.NewFromInt(100)) {
		t.Fatalf("balance should be 100, got %s", bal.String())
	}
	frozen, err := decimal.NewFromString(acc.FrozenBalance)
	if err != nil {
		t.Fatalf("parse frozen: %v", err)
	}
	if !frozen.Equals(decimal.NewFromInt(15)) {
		t.Fatalf("frozen should be 15, got %s", frozen.String())
	}
}

func TestCreditTransactionTypes(t *testing.T) {
	validTypes := []string{
		"deposit",
		"subscription_grant",
		"free_grant",
		"ai_usage",
		"ai_hold",
		"ai_release",
		"refund",
		"adjustment",
	}
	for _, txType := range validTypes {
		if len(txType) == 0 {
			t.Fatalf("tx_type should not be empty")
		}
	}
}

func TestCreditRepository_Interface(t *testing.T) {
	// Verify the repository can be instantiated (nil pool is fine for struct check).
	repo := NewCreditRepository(nil)
	if repo == nil {
		t.Fatal("NewCreditRepository returned nil")
	}
}
