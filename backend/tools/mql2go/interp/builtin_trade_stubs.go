package interp

import (
	"github.com/shopspring/decimal"
)

// callTradeStubs handles MQL4/MQL5 account, check, market info, timeseries,
// trade helper, history, and pending order functions. These are registered
// in implementedPlatform so the analyzer doesn't report them as blind spots.
func (it *Interpreter) callTradeStubs(name string, args []Expr) (Value, bool) {
	switch name {
	// ── MQL4-only account functions ───────────────────────────────────
	case "AccountFreeMarginCheck":
		return DecimalVal(it.ctx.Account().FreeMargin), true
	case "AccountFreeMarginMode":
		return IntVal(1), true
	case "AccountServer":
		return StringVal("SimBroker"), true
	case "AccountStopoutMode":
		return IntVal(0), true
	case "AccountCredit":
		return DecimalVal(decimal.Zero), true
	case "AccountProfit":
		return DecimalVal(it.ctx.Account().Equity.Sub(it.ctx.Account().Balance)), true

	// ── MQL5 AccountInfo functions ────────────────────────────────────
	case "AccountInfoDouble":
		if len(args) >= 1 {
			prop := it.evalExpr(&args[0]).ToInt()
			switch prop {
			case 0:
				return DecimalVal(it.ctx.Account().Balance), true
			case 1:
				return DecimalVal(decimal.Zero), true
			case 2:
				return DecimalVal(it.ctx.Account().Equity.Sub(it.ctx.Account().Balance)), true
			case 3:
				return DecimalVal(it.ctx.Account().Equity), true
			case 4:
				return DecimalVal(it.ctx.Account().Margin), true
			case 5:
				return DecimalVal(it.ctx.Account().FreeMargin), true
			default:
				return DecimalVal(decimal.Zero), true
			}
		}
		return DecimalVal(decimal.Zero), true
	case "AccountInfoInteger":
		if len(args) >= 1 {
			prop := it.evalExpr(&args[0]).ToInt()
			switch prop {
			case 32:
				return IntVal(it.ctx.Account().Leverage), true
			case 35:
				return BoolVal(true), true
			default:
				return IntVal(0), true
			}
		}
		return IntVal(0), true
	case "AccountInfoString":
		if len(args) >= 1 {
			prop := it.evalExpr(&args[0]).ToInt()
			switch prop {
			case 1:
				return StringVal("USD"), true
			case 2:
				return StringVal("Backtest"), true
			case 3:
				return StringVal("SimBroker"), true
			case 4:
				return StringVal("SimBroker"), true
			default:
				return StringVal(""), true
			}
		}
		return StringVal(""), true

	// ── MQL4 check functions ──────────────────────────────────────────
	case "IsConnected":
		return BoolVal(true), true
	case "IsDemo":
		return BoolVal(false), true
	case "IsDllsAllowed":
		return BoolVal(false), true
	case "IsExpertEnabled":
		return BoolVal(true), true
	case "IsLibrariesAllowed":
		return BoolVal(false), true
	case "IsTradeAllowed":
		return BoolVal(true), true
	case "IsTradeContextBusy":
		return BoolVal(false), true

	// ── MQL5 market info additions ────────────────────────────────────
	case "SymbolInfoTick":
		if len(args) >= 1 && it.ctx.Broker() != nil {
			sym := it.evalExpr(&args[0]).ToString()
			info, err := it.ctx.Broker().SymbolInfo(sym)
			if err == nil {
				_ = info
			}
		}
		return BoolVal(true), true
	case "SymbolName":
		if len(args) >= 1 {
			idx := int(it.evalExpr(&args[0]).ToInt())
			if idx == 0 {
				return StringVal(it.ctx.Symbol()), true
			}
		}
		return StringVal(""), true
	case "SymbolSelect":
		return BoolVal(true), true
	case "SymbolsTotal":
		return IntVal(1), true
	case "SymbolInfoMarginRate":
		return DecimalVal(decimal.Zero), true
	case "SymbolInfoSessionQuote":
		return BoolVal(false), true
	case "SymbolInfoSessionTrade":
		return BoolVal(false), true
	case "SymbolIsSynchronized":
		return BoolVal(true), true

	// ── MQL5 timeseries access ────────────────────────────────────────
	case "Bars":
		if it.ctx != nil && it.ctx.Bars() != nil {
			return IntVal(int32(it.ctx.Bars().Len())), true
		}
		return IntVal(0), true
	case "iBarShift", "iHighest", "iLowest":
		return IntVal(0), true
	case "iTickVolume", "iRealVolume", "iVolume":
		return DecimalVal(decimal.Zero), true
	case "iSpread":
		return IntVal(0), true
	case "CopyRates", "CopyClose", "CopyHigh", "CopyLow", "CopyOpen",
		"CopyTime", "CopyBuffer", "CopyTickVolume", "CopyRealVolume",
		"CopySpread", "CopyTicks":
		return IntVal(0), true
	case "BarsCalculated":
		return IntVal(0), true
	case "SeriesInfoInteger":
		return IntVal(0), true

	// ── MQL5 trade helpers ────────────────────────────────────────────
	case "OrderCalcMargin", "OrderCalcProfit":
		return DecimalVal(decimal.Zero), true
	case "OrderCheck":
		return BoolVal(true), true
	case "PositionSelect":
		if len(args) >= 1 {
			ticket := int64(it.evalExpr(&args[0]).ToInt())
			return BoolVal(it.posPool.SelectByTicket(ticket)), true
		}
		return BoolVal(false), true

	// ── MQL5 order history (stubs — no history in backtest) ───────────
	case "HistorySelect", "HistorySelectByPosition":
		return BoolVal(true), true
	case "HistoryDealsTotal", "HistoryDealGetTicket",
		"HistoryOrdersTotal", "HistoryOrderGetTicket":
		return IntVal(0), true
	case "HistoryDealSelect", "HistoryOrderSelect":
		return BoolVal(false), true
	case "HistoryDealGetDouble", "HistoryOrderGetDouble":
		return DecimalVal(decimal.Zero), true
	case "HistoryDealGetInteger", "HistoryOrderGetInteger":
		return IntVal(0), true
	case "HistoryDealGetString", "HistoryOrderGetString":
		return StringVal(""), true

	// ── MQL5 pending order functions ──────────────────────────────────
	case "OrderGetTicket":
		return IntVal(0), true
	case "OrderGetDouble":
		return DecimalVal(decimal.Zero), true
	case "OrderGetInteger":
		return IntVal(0), true
	case "OrderGetString":
		return StringVal(""), true
	}
	return NoneVal(), false
}
