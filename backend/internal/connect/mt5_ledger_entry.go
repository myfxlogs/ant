package connect

import (
	"strings"

	v1 "anttrader/gen/proto"
	mt5pb "anttrader/mt5"
)

func BuildMT5LedgerEntryEvent(accountID string, deal *mt5pb.DealInternal, trans *mt5pb.TransactionInfo) *v1.LedgerEntryEvent {
	if accountID == "" || deal == nil {
		return nil
	}

	dt := deal.GetType().String()
	if !isMT5LedgerDealType(dt) {
		return nil
	}

	currency := ""
	if trans != nil {
		currency = trans.GetCurrency()
	}

	amount := deal.GetProfit()
	// Prefer specific fields when meaningful.
	if strings.Contains(dt, "Commission") {
		if deal.GetCommission() != 0 {
			amount = deal.GetCommission()
		} else if deal.GetFee() != 0 {
			amount = deal.GetFee()
		}
	} else if dt == "DealType_InterestRate" {
		// Interest may be represented as profit in the gateway.
		amount = deal.GetProfit()
	} else if dt == "DealType_Charge" {
		if deal.GetFee() != 0 {
			amount = deal.GetFee()
		}
	}

	return &v1.LedgerEntryEvent{
		AccountId:     accountID,
		EntryType:     dt,
		Amount:        amount,
		Currency:      currency,
		Time:          deal.GetOpenTime(),
		Comment:       deal.GetComment(),
		RelatedTicket: deal.GetTicketNumber(),
	}
}

func isMT5LedgerDealType(dt string) bool {
	switch dt {
	case "DealType_Balance", "DealType_Credit", "DealType_Charge", "DealType_Correction", "DealType_Bonus",
		"DealType_Commission", "DealType_DailyCommission", "DealType_MonthlyCommission",
		"DealType_DailyAgentCommission", "DealType_MonthlyAgentCommission", "DealType_InterestRate":
		return true
	default:
		return false
	}
}
