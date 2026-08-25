package mql2go

import (
	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// ── MQL4 order property builtins ──────────────────────────────────────
// VM-CODE-HYGIENE-1: extracted from vm_builtin_trade.go for file-lines compliance.

func builtinOrderStopLoss(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentOrder != nil {
		return interp.DecimalVal(vm.currentOrder.StopLoss), nil
	}
	if vm.currentPos != nil {
		return interp.DecimalVal(vm.currentPos.StopLoss), nil
	}
	return interp.DecimalVal(decimal.Zero), nil
}

func builtinOrderTakeProfit(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentOrder != nil {
		return interp.DecimalVal(vm.currentOrder.TakeProfit), nil
	}
	if vm.currentPos != nil {
		return interp.DecimalVal(vm.currentPos.TakeProfit), nil
	}
	return interp.DecimalVal(decimal.Zero), nil
}

func builtinOrderTicket(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentOrder != nil {
		return interp.IntVal(int32(vm.currentOrder.Ticket)), nil
	}
	if vm.currentPos != nil {
		return interp.IntVal(int32(vm.currentPos.Ticket)), nil
	}
	return interp.IntVal(0), nil
}

// sideToOrderType maps sdk.PositionSide (SideBuy=1, SideSell=-1) to MQL4 OP_* constants.
// MQL4: OP_BUY=0, OP_SELL=1. MQL5 POSITION_TYPE: POSITION_TYPE_BUY=0, POSITION_TYPE_SELL=1.
func sideToOrderType(side sdk.PositionSide) int32 {
	switch side {
	case sdk.SideBuy:
		return 0 // OP_BUY / POSITION_TYPE_BUY
	case sdk.SideSell:
		return 1 // OP_SELL / POSITION_TYPE_SELL
	default:
		return -1
	}
}

// orderTypeToMQL4 maps sdk.OrderType + side to MQL4 OP_* constants.
// MQL4: OP_BUYLIMIT=2, OP_SELLLIMIT=3, OP_BUYSTOP=4, OP_SELLSTOP=5.
func orderTypeToMQL4(ot sdk.OrderType, side sdk.PositionSide) int32 {
	switch ot {
	case sdk.OrderLimit:
		if side == sdk.SideSell {
			return 3 // OP_SELLLIMIT
		}
		return 2 // OP_BUYLIMIT
	case sdk.OrderStop:
		if side == sdk.SideSell {
			return 5 // OP_SELLSTOP
		}
		return 4 // OP_BUYSTOP
	default:
		return 0
	}
}

// pendingPriceModifier allows modifying the price of a pending order.
// SimBroker implements this; live broker may not.
type pendingPriceModifier interface {
	PositionModifyPrice(ticket int64, price decimal.Decimal) (sdk.OrderResult, error)
}

func builtinOrderType(vm *VM, args []interp.Value) (interp.Value, error) {
	// Pending orders: return MQL4 OP_* constants for limit/stop types.
	if vm.currentOrder != nil {
		return interp.IntVal(orderTypeToMQL4(vm.currentOrder.Type, vm.currentOrder.Side)), nil
	}
	// Open positions: return OP_BUY/OP_SELL.
	if vm.currentPos != nil {
		return interp.IntVal(sideToOrderType(vm.currentPos.Side)), nil
	}
	return interp.IntVal(0), nil
}

func builtinOrderLots(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentOrder != nil {
		return interp.DecimalVal(vm.currentOrder.Volume), nil
	}
	if vm.currentPos != nil {
		return interp.DecimalVal(vm.currentPos.Volume), nil
	}
	return interp.DecimalVal(decimal.Zero), nil
}

func builtinOrderSymbol(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentOrder != nil {
		return interp.StringVal(vm.currentOrder.Symbol), nil
	}
	if vm.currentPos != nil {
		return interp.StringVal(vm.currentPos.Symbol), nil
	}
	return interp.StringVal(""), nil
}

func builtinOrderOpenPrice(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentOrder != nil {
		return interp.DecimalVal(vm.currentOrder.Price), nil
	}
	if vm.currentPos != nil {
		return interp.DecimalVal(vm.currentPos.OpenPrice), nil
	}
	return interp.DecimalVal(decimal.Zero), nil
}

func builtinOrderClosePrice(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	// For closed positions (from history pool), return the recorded close price.
	if vm.currentPos.ClosePrice.IsPositive() {
		return interp.DecimalVal(vm.currentPos.ClosePrice), nil
	}
	// For open positions, return current market price.
	if vm.ctx != nil {
		if vm.currentPos.Side == sdk.SideSell {
			return interp.DecimalVal(vm.ctx.Ask()), nil
		}
		return interp.DecimalVal(vm.ctx.Bid()), nil
	}
	return interp.DecimalVal(decimal.Zero), nil
}

func builtinOrderProfit(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(vm.currentPos.Profit), nil
}

func builtinOrderMagicNumber(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentOrder != nil {
		return interp.IntVal(vm.currentOrder.Magic), nil
	}
	if vm.currentPos != nil {
		return interp.IntVal(vm.currentPos.Magic), nil
	}
	return interp.IntVal(0), nil
}

func builtinOrderComment(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentOrder != nil {
		return interp.StringVal(vm.currentOrder.Comment), nil
	}
	if vm.currentPos != nil {
		return interp.StringVal(vm.currentPos.Comment), nil
	}
	return interp.StringVal(""), nil
}

func builtinOrderOpenTime(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentOrder != nil {
		return interp.IntVal(int32(vm.currentOrder.OpenTime.Unix())), nil
	}
	if vm.currentPos != nil {
		return interp.IntVal(int32(vm.currentPos.OpenTime.Unix())), nil
	}
	return interp.IntVal(0), nil
}

func builtinOrderCloseTime(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos != nil && !vm.currentPos.CloseTime.IsZero() {
		return interp.IntVal(int32(vm.currentPos.CloseTime.Unix())), nil
	}
	return interp.IntVal(0), nil
}

func builtinOrderCommission(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos != nil {
		return interp.DecimalVal(vm.currentPos.Commission), nil
	}
	return interp.DecimalVal(decimal.Zero), nil
}

func builtinOrderSwap(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos != nil {
		return interp.DecimalVal(vm.currentPos.Swap), nil
	}
	return interp.DecimalVal(decimal.Zero), nil
}
