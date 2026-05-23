package service

import (
	"strings"

	mt4pb "anttrader/mt4"
	mt5pb "anttrader/mt5"
)

func ParseOrderTypeMT4(orderType string) (mt4pb.Op, error) {
	switch strings.ToLower(orderType) {
	case "buy":
		return mt4pb.Op_Op_Buy, nil
	case "sell":
		return mt4pb.Op_Op_Sell, nil
	case "buy_limit":
		return mt4pb.Op_Op_BuyLimit, nil
	case "sell_limit":
		return mt4pb.Op_Op_SellLimit, nil
	case "buy_stop":
		return mt4pb.Op_Op_BuyStop, nil
	case "sell_stop":
		return mt4pb.Op_Op_SellStop, nil
	default:
		return mt4pb.Op_Op_Buy, ErrInvalidOrderType
	}
}

func ParseOrderTypeMT5(orderType string) (mt5pb.OrderType, error) {
	switch strings.ToLower(orderType) {
	case "buy":
		return mt5pb.OrderType_OrderType_Buy, nil
	case "sell":
		return mt5pb.OrderType_OrderType_Sell, nil
	case "buy_limit":
		return mt5pb.OrderType_OrderType_BuyLimit, nil
	case "sell_limit":
		return mt5pb.OrderType_OrderType_SellLimit, nil
	case "buy_stop":
		return mt5pb.OrderType_OrderType_BuyStop, nil
	case "sell_stop":
		return mt5pb.OrderType_OrderType_SellStop, nil
	case "buy_stop_limit":
		return mt5pb.OrderType_OrderType_BuyStopLimit, nil
	case "sell_stop_limit":
		return mt5pb.OrderType_OrderType_SellStopLimit, nil
	default:
		return mt5pb.OrderType_OrderType_Buy, ErrInvalidOrderType
	}
}

func OrderTypeToString(op int32) string {
	switch op {
	case 0:
		return "buy"
	case 1:
		return "sell"
	case 2:
		return "buy_limit"
	case 3:
		return "sell_limit"
	case 4:
		return "buy_stop"
	case 5:
		return "sell_stop"
	case 6:
		return "buy_stop_limit"
	case 7:
		return "sell_stop_limit"
	case 100:
		return "BALANCE"
	case 101:
		return "CREDIT"
	default:
		return "unknown"
	}
}

func DealTypeToString(dealType int32) string {
	switch dealType {
	case 0:
		return "buy"
	case 1:
		return "sell"
	case 2:
		return "BALANCE"
	case 3:
		return "CREDIT"
	case 4:
		return "CHARGE"
	case 5:
		return "CORRECTION"
	case 6:
		return "BONUS"
	case 7:
		return "COMMISSION"
	default:
		return "unknown"
	}
}
