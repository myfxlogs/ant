package mql2go

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// ── Risk checks ────────────────────────────────────────────────────

func extractRiskChecksCST(root *sitter.Node, version string) []RiskCheck {
	var checks []RiskCheck
	walkCST(root, func(n *sitter.Node) bool {
		if n.Type() != "call_expression" {
			return true
		}
		name := callFuncName(n)
		switch name {
		case "AccountFreeMargin":
			checks = append(checks, RiskCheck{
				Kind:      "margin_check",
				Condition: "AccountFreeMargin() <= 0",
				Action:    "skip_order",
				Trigger:   "pre_order",
			})
		case "IsTradeAllowed":
			checks = append(checks, RiskCheck{
				Kind:      "trade_allowed",
				Condition: "!IsTradeAllowed()",
				Action:    "skip_order",
				Trigger:   "pre_order",
			})
		case "IsExpertEnabled":
			checks = append(checks, RiskCheck{
				Kind:      "expert_enabled",
				Condition: "!IsExpertEnabled()",
				Action:    "skip_order",
				Trigger:   "pre_order",
			})
		}
		return true
	})
	return checks
}

// ── Blind spots ────────────────────────────────────────────────────

func detectBlindSpots(source string, root *sitter.Node, intent *StrategyIntent) []BlindSpot {
	var spots []BlindSpot

	unsupportedCalls := map[string]BlindSpot{
		"OrderModify": {
			ID:          "BS_OrderModify",
			Category:    "order_modify",
			Severity:    "信息",
			Description: "OrderModify (修改订单 SL/TP) 已部分转译为 ctx.Broker().PositionModify()",
			Handling:    "检查生成的修改逻辑是否覆盖原始条件",
		},
		"MarketInfo": {
			ID:          "BS_MarketInfo",
			Category:    "market_data",
			Severity:    "信息",
			Description: "MarketInfo() 调用未转译，需手动替换为 ctx 方法",
			Handling:    "用 ctx.Bid()/ctx.Ask() 等替代",
		},
		"ObjectCreate": {
			ID:          "BS_ObjectCreate",
			Category:    "chart_objects",
			Severity:    "信息",
			Description: "图表对象操作 (ObjectCreate/ObjectDelete) 不支持，Go SDK 无图表 API",
			Handling:    "忽略，图表对象不影响策略逻辑",
		},
		"SendMail": {
			ID:          "BS_SendMail",
			Category:    "notification",
			Severity:    "信息",
			Description: "SendMail/SendNotification 不支持",
			Handling:    "用 ctx.Notify() 替代或忽略",
		},
		"FileOpen": {
			ID:          "BS_FileIO",
			Category:    "file_io",
			Severity:    "警告",
			Description: "文件 I/O 操作不支持，Go 策略无文件系统访问",
			Handling:    "用 ctx.ParamString() 读取配置替代",
		},
		"DLLImport": {
			ID:          "BS_DLLImport",
			Category:    "dll_import",
			Severity:    "致命",
			Description: "DLL 导入 (#import) 不支持，Go 策略无法调用外部 DLL",
			Handling:    "需手动用 Go 重新实现 DLL 功能",
		},
	}

	walkCST(root, func(n *sitter.Node) bool {
		if n.Type() != "call_expression" {
			return true
		}
		name := callFuncName(n)
		if spot, ok := unsupportedCalls[name]; ok {
			spot.Location = fmt.Sprintf("line %d", n.StartPoint().Row+1)
			spot.UserActionRequired = spot.Severity == "致命"
			spots = append(spots, spot)
		}
		return true
	})

	if strings.Contains(source, "#import") {
		spots = append(spots, unsupportedCalls["DLLImport"])
	}

	hasOrdersTotal := false
	walkCST(root, func(n *sitter.Node) bool {
		if n.Type() == "call_expression" && callFuncName(n) == "OrdersTotal" {
			hasOrdersTotal = true
			return false
		}
		return true
	})
	if hasOrdersTotal {
		spots = append(spots, BlindSpot{
			ID:                 "BS_OrdersTotal",
			Category:           "order_iteration",
			Severity:           "警告",
			Description:        "OrdersTotal() 遍历模式已转译为 ctx.Broker().Positions() 遍历，但条件过滤可能不完整",
			Handling:           "检查生成的 closeAll 逻辑是否覆盖原始遍历条件",
			UserActionRequired: false,
		})
	}

	if strings.Contains(source, "StringConcatenate") || strings.Contains(source, "StringFormat") {
		spots = append(spots, BlindSpot{
			ID:          "BS_StringOps",
			Category:    "string_ops",
			Severity:    "信息",
			Description: "字符串操作函数未转译，需手动用 Go fmt.Sprintf 替代",
			Handling:    "用 fmt.Sprintf 替代",
		})
	}

	for _, ind := range intent.Indicators {
		if ind.SDKMethod == "i_custom" {
			spots = append(spots, BlindSpot{
				ID:          "BS_iCustom",
				Category:    "custom_indicator",
				Severity:    "警告",
				Description: "自定义指标 iCustom() 无法自动转译，需手动实现",
				Handling:    "在 ctx.Indicators().ICustom() 中实现或用标准指标替代",
			})
			break
		}
	}

	onArrayIndicators := []string{
		"iMAOnArray", "iRSIOnArray", "iBandsOnArray",
		"iCCIOnArray", "iStdDevOnArray", "iMomentumOnArray",
	}
	for _, name := range onArrayIndicators {
		if strings.Contains(source, name) {
			spots = append(spots, BlindSpot{
				ID:          "BS_" + name,
				Category:    "indicator_on_array",
				Severity:    "警告",
				Description: name + "() 在自定义数组上计算指标，Go SDK 不支持",
				Handling:    "需手动实现数组指标计算逻辑",
			})
			break
		}
	}

	for _, fn := range findFunctions(root) {
		if funcName(fn) == "OnTester" {
			spots = append(spots, BlindSpot{
				ID:          "BS_OnTester",
				Category:    "on_tester",
				Severity:    "信息",
				Description: "OnTester() 回测优化自定义指标函数不支持，Go 回测引擎无对应接口",
				Handling:    "忽略，不影响策略执行逻辑",
			})
			break
		}
	}

	if intent.Meta.MQLVersion == "mql5" {
		mql5Events := map[string]BlindSpot{
			"OnTrade": {
				ID:          "BS_OnTrade",
				Category:    "on_trade",
				Severity:    "信息",
				Description: "OnTrade() 交易事件回调不支持，Go SDK 无交易事件通知接口",
				Handling:    "忽略，策略逻辑应在 OnTick 中检查持仓状态",
			},
			"OnTradeTransaction": {
				ID:          "BS_OnTradeTransaction",
				Category:    "on_trade_transaction",
				Severity:    "信息",
				Description: "OnTradeTransaction() 交易事务回调不支持，Go SDK 无交易事务通知接口",
				Handling:    "忽略，策略逻辑应在 OnTick 中检查持仓状态",
			},
			"OnBookEvent": {
				ID:          "BS_OnBookEvent",
				Category:    "on_book_event",
				Severity:    "信息",
				Description: "OnBookEvent() 市场深度事件回调不支持，Go SDK 无市场深度接口",
				Handling:    "忽略，不影响策略执行逻辑",
			},
			"OnTesterInit": {
				ID:          "BS_OnTesterInit",
				Category:    "on_tester_init",
				Severity:    "信息",
				Description: "OnTesterInit() 优化开始事件不支持，Go 回测引擎无对应接口",
				Handling:    "忽略，不影响策略执行逻辑",
			},
			"OnTesterDeinit": {
				ID:          "BS_OnTesterDeinit",
				Category:    "on_tester_deinit",
				Severity:    "信息",
				Description: "OnTesterDeinit() 优化结束事件不支持，Go 回测引擎无对应接口",
				Handling:    "忽略，不影响策略执行逻辑",
			},
			"OnTesterPass": {
				ID:          "BS_OnTesterPass",
				Category:    "on_tester_pass",
				Severity:    "信息",
				Description: "OnTesterPass() 优化数据帧事件不支持，Go 回测引擎无对应接口",
				Handling:    "忽略，不影响策略执行逻辑",
			},
		}
		for _, fn := range findFunctions(root) {
			name := funcName(fn)
			if bs, ok := mql5Events[name]; ok {
				spots = append(spots, bs)
			}
		}
	}

	if intent.Meta.MQLVersion == "mql5" {
		hasNativeOrderSend := false
		walkCST(root, func(n *sitter.Node) bool {
			if n.Type() == "call_expression" && callFuncName(n) == "OrderSend" {
				args := childByType("", n, "argument_list")
				if args != nil {
					named := getNamedChildren(args)
					if len(named) <= 2 {
						hasNativeOrderSend = true
						return false
					}
				}
			}
			return true
		})
		if hasNativeOrderSend {
			spots = append(spots, BlindSpot{
				ID:          "BS_NativeOrderSend",
				Category:    "native_ordersend",
				Severity:    "警告",
				Description: "MQL5 原生 OrderSend(MqlTradeRequest, MqlTradeResult) 结构体方式下单不支持自动转译",
				Handling:    "需手动转换为 ctx.Broker().OrderSend() 调用",
			})
		}
	}

	if intent.Meta.MQLVersion == "mql5" {
		mql5PosProps := []string{"PositionGetDouble", "PositionGetInteger", "PositionGetString", "PositionGetSymbol"}
		for _, propFunc := range mql5PosProps {
			if strings.Contains(source, propFunc) && len(intent.PositionLoops) > 0 {
				spots = append(spots, BlindSpot{
					ID:          "BS_" + propFunc,
					Category:    "position_property",
					Severity:    "信息",
					Description: propFunc + "() 持仓属性函数已识别为 PositionLoopRule 的一部分",
					Handling:    "检查生成的遍历逻辑是否覆盖原始属性访问",
				})
				break
			} else if strings.Contains(source, propFunc) {
				spots = append(spots, BlindSpot{
					ID:          "BS_" + propFunc,
					Category:    "position_property",
					Severity:    "警告",
					Description: propFunc + "() 持仓属性函数已检测到但未识别为标准遍历模式",
					Handling:    "需手动检查持仓属性访问是否正确转译",
				})
				break
			}
		}
	}

	if intent.Meta.MQLVersion == "mql4" {
		hasOrderSelect := false
		walkCST(root, func(n *sitter.Node) bool {
			if n.Type() == "call_expression" && callFuncName(n) == "OrderSelect" {
				hasOrderSelect = true
				return false
			}
			return true
		})
		if hasOrderSelect && len(intent.OrderLoops) > 0 {
			spots = append(spots, BlindSpot{
				ID:          "BS_OrderSelect",
				Category:    "order_select",
				Severity:    "信息",
				Description: "OrderSelect() + Order* 属性函数遍历模式已识别为 OrderLoopRule",
				Handling:    "检查生成的遍历逻辑是否覆盖原始条件过滤",
			})
		} else if hasOrderSelect {
			spots = append(spots, BlindSpot{
				ID:          "BS_OrderSelect",
				Category:    "order_select",
				Severity:    "警告",
				Description: "OrderSelect() 调用已检测到但未识别为标准遍历模式",
				Handling:    "需手动检查订单遍历逻辑是否正确转译",
			})
		}
	}

	if strings.Contains(source, "CTrade") {
		spots = append(spots, BlindSpot{
			ID:                 "BS_CTrade",
			Category:           "mql5_ctrade",
			Severity:           "警告",
			Description:        "MQL5 CTrade 类用法已部分识别，但 OOP 方法调用可能不完整",
			Handling:           "检查生成的入场/出场逻辑是否覆盖所有 CTrade.Buy/Sell/PositionClose 调用",
			UserActionRequired: false,
		})
	}

	if intent.Meta.MQLVersion == "mql5" {
		mql5Calls := map[string]BlindSpot{
			"PositionsTotal": {
				ID:          "BS_PositionsTotal",
				Category:    "mql5_position_iter",
				Severity:    "信息",
				Description: "MQL5 PositionsTotal() 已转译为 ctx.Broker().Positions() 遍历",
				Handling:    "检查生成的遍历逻辑是否覆盖原始条件",
			},
			"PositionSelect": {
				ID:          "BS_PositionSelect",
				Category:    "mql5_position_select",
				Severity:    "警告",
				Description: "MQL5 PositionSelect() 按品种选择持仓的模式未完全转译",
				Handling:    "需手动检查生成的持仓选择逻辑",
			},
			"PositionGetTicket": {
				ID:          "BS_PositionGetTicket",
				Category:    "mql5_position_ticket",
				Severity:    "警告",
				Description: "MQL5 PositionGetTicket() 获取持仓票据的模式未转译",
				Handling:    "用 ctx.Broker().Positions() 返回的 Position.Ticket 替代",
			},
		}
		walkCST(root, func(n *sitter.Node) bool {
			if n.Type() != "call_expression" {
				return true
			}
			name := callFuncName(n)
			if spot, ok := mql5Calls[name]; ok {
				spot.Location = fmt.Sprintf("line %d", n.StartPoint().Row+1)
				spots = append(spots, spot)
			}
			return true
		})
	}

	if intent.Meta.MQLVersion == "mql4" {
		hasCloseBy := false
		walkCST(root, func(n *sitter.Node) bool {
			if n.Type() == "call_expression" && callFuncName(n) == "OrderCloseBy" {
				hasCloseBy = true
				return false
			}
			return true
		})
		if hasCloseBy {
			spots = append(spots, BlindSpot{
				ID:          "BS_OrderCloseBy",
				Category:    "order_close_by",
				Severity:    "信息",
				Description: "MQL4 OrderCloseBy (对冲平仓) 已转译为 closeAll 逻辑",
				Handling:    "检查生成的平仓逻辑是否覆盖原始对冲平仓条件",
			})
		}
	}

	return spots
}
