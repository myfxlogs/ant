"""Mapping tables: MQL constructs → Python SDK equivalents (T2.1).

Every mapping is deterministic — no LLM, no guessing.  Rules are applied
in order; first match wins.  Unmatched constructs become TRANSPILER-GAP.
"""

from __future__ import annotations

import re
from dataclasses import dataclass
from typing import Callable, Dict, List, Optional, Tuple

# ── Lifecycle function mappings ────────────────────────────────────────

LIFECYCLE_MAP: Dict[str, str] = {
    "OnInit": "on_init",
    "OnTick": "on_tick",
    "OnTimer": "on_timer",
    "OnTrade": "on_trade",
    "OnDeinit": "on_deinit",
    "OnCalculate": "on_bar",  # approximate; may need manual adjustment
}

# ── MQL type → Python type ─────────────────────────────────────────────

MQL_TYPE_MAP: Dict[str, str] = {
    "int": "int",
    "double": "float",
    "bool": "bool",
    "string": "str",
    "color": "str",         # colors → hex strings
    "datetime": "int",       # unix timestamp
    "void": "None",
    "uint": "int",
    "long": "int",
    "ulong": "int",
    "float": "float",
    "short": "int",
    "ushort": "int",
    "char": "str",
    "uchar": "int",
}

# ── Indicator function mappings ─────────────────────────────────────────

# Each entry: (mql_pattern, sdk_replacement_template)
# Template vars: {period}, {shift}, {method}, {symbol}, {tf}, {buffer}, {params}
INDICATOR_MAP: List[Tuple[str, str]] = [
    # iMA(symbol, tf, period, shift, method, applied_price) → indicators.ma(period, shift, method)
    (
        r"iMA\s*\(\s*\w+\s*,\s*\w+\s*,\s*(?P<period>\d+)\s*,\s*(?P<shift>\d+)\s*,\s*(?P<method>MODE_\w+)\s*,",
        r"self.indicators.ma(period={period}, shift={shift}, method='{method_lower}')",
    ),
    (
        r"iMA\s*\(\s*\w+\s*,\s*\w+\s*,\s*(?P<period>\d+)\s*,\s*(?P<shift>\d+)\s*,\s*(?P<method>\w+)\s*\)",
        r"self.indicators.ma(period={period}, shift={shift}, method='{method_lower}')",
    ),
    # iRSI(symbol, tf, period, applied_price, shift)
    (
        r"iRSI\s*\(\s*\w+\s*,\s*\w+\s*,\s*(?P<period>\d+)\s*,\s*\w+\s*,\s*(?P<shift>\d+)\s*\)",
        r"self.indicators.rsi(period={period}, shift={shift})",
    ),
    # iBands(symbol, tf, period, deviation, shift, applied_price, mode, shift2)
    (
        r"iBands\s*\(\s*\w+\s*,\s*\w+\s*,\s*(?P<period>\d+)\s*,\s*(?P<deviation>[\d.]+)\s*,",
        r"self.indicators.bands(period={period}, deviation={deviation})",
    ),
    # iMACD(symbol, tf, fast, slow, signal, applied_price, mode, shift)
    (
        r"iMACD\s*\(\s*\w+\s*,\s*\w+\s*,\s*(?P<fast>\d+)\s*,\s*(?P<slow>\d+)\s*,\s*(?P<signal>\d+)\s*,",
        r"self.indicators.macd(fast={fast}, slow={slow}, signal={signal})",
    ),
    # iATR(symbol, tf, period, shift)
    (
        r"iATR\s*\(\s*\w+\s*,\s*\w+\s*,\s*(?P<period>\d+)\s*,\s*(?P<shift>\d+)\s*\)",
        r"self.indicators.atr(period={period}, shift={shift})",
    ),
    # iStochastic(symbol, tf, k, d, slowing, method, price_field, mode, shift)
    (
        r"iStochastic\s*\(\s*\w+\s*,\s*\w+\s*,\s*(?P<k>\d+)\s*,\s*(?P<d>\d+)\s*,",
        r"self.indicators.stochastic(k_period={k}, d_period={d})",
    ),
    # iCCI(symbol, tf, period, applied_price, shift)
    (
        r"iCCI\s*\(\s*\w+\s*,\s*\w+\s*,\s*(?P<period>\d+)\s*,",
        r"self.indicators.cci(period={period})",
    ),
    # iCustom(symbol, tf, name, ..., mode, shift)
    (
        r"iCustom\s*\(\s*\w+\s*,\s*\w+\s*,\s*\"(?P<name>[^\"]+)\"\s*,\s*(?P<params>[\d,\s]+)\s*,\s*(?P<buffer>\d+)\s*,\s*(?P<shift>\d+)\s*\)",
        r"self.indicators.i_custom(name='{name}', params=[{params}], buffer={buffer}, shift={shift})",
    ),
]

# ── Method name conversion ─────────────────────────────────────────────

METHOD_MAP: Dict[str, str] = {
    "MODE_SMA": "sma",
    "MODE_EMA": "ema",
    "MODE_SMMA": "smma",
    "MODE_LWMA": "lwma",
}


def method_name(mql_method: str) -> str:
    return METHOD_MAP.get(mql_method.strip(), mql_method.strip().lower().replace("mode_", ""))


# ── Trading function mappings ───────────────────────────────────────────

# OrderSend → broker.order_send
ORDERSEND_RE = re.compile(
    r"OrderSend\s*\(\s*"
    r"(?P<symbol>[^,]+)\s*,\s*"
    r"(?P<cmd>[^,]+)\s*,\s*"
    r"(?P<volume>[^,]+)\s*,\s*"
    r"(?P<price>[^,]+)\s*,\s*"
    r"(?P<slippage>[^,]+)\s*,\s*"
    r"(?P<sl>[^,]+)\s*,\s*"
    r"(?P<tp>[^,]+)"
    r"(?:\s*,\s*(?P<comment>[^,]*))?"
    r"(?:\s*,\s*(?P<magic>[^,]*))?"
    r"(?:\s*,\s*(?P<expiration>[^,]*))?"
    r"(?:\s*,\s*(?P<arrow>[^)]*))?"
    r"\s*\)"
)

# OrderClose → broker.position_close
ORDERCLOSE_RE = re.compile(
    r"OrderClose\s*\(\s*(?P<ticket>[^,]+)\s*,\s*(?P<volume>[^,]+)\s*,\s*(?P<price>[^,]+)\s*,\s*(?P<slippage>[^)]+)\s*\)"
)

# OrderModify → broker.position_modify
ORDERMODIFY_RE = re.compile(
    r"OrderModify\s*\(\s*(?P<ticket>[^,]+)\s*,\s*(?P<price>[^,]+)\s*,\s*(?P<sl>[^,]+)\s*,\s*(?P<tp>[^,]+)\s*,\s*(?P<expiration>[^)]*)\s*\)"
)

# OrderDelete → broker.order_delete
ORDERDELETE_RE = re.compile(
    r"OrderDelete\s*\(\s*(?P<ticket>[^)]+)\s*\)"
)

# ── Order type constants ────────────────────────────────────────────────

CMD_MAP: Dict[str, str] = {
    "OP_BUY": "OrderType.BUY",
    "OP_SELL": "OrderType.SELL",
    "OP_BUYLIMIT": "OrderType.BUY_LIMIT",
    "OP_SELLLIMIT": "OrderType.SELL_LIMIT",
    "OP_BUYSTOP": "OrderType.BUY_STOP",
    "OP_SELLSTOP": "OrderType.SELL_STOP",
    "0": "OrderType.BUY",
    "1": "OrderType.SELL",
    "2": "OrderType.BUY_LIMIT",
    "3": "OrderType.SELL_LIMIT",
    "4": "OrderType.BUY_STOP",
    "5": "OrderType.SELL_STOP",
}

# ── Variable declarations ───────────────────────────────────────────────

EXTERN_RE = re.compile(
    r"extern\s+(?P<type>\w+)\s+(?P<name>\w+)\s*=\s*(?P<value>[^;]+)\s*;"
)
INPUT_RE = re.compile(
    r"input\s+(?P<type>\w+)\s+(?P<name>\w+)\s*=\s*(?P<value>[^;]+)\s*;"
)

# ── OrderSelect loop detection ──────────────────────────────────────────

ORDERSELECT_RE = re.compile(r"OrderSelect\s*\(([^)]+)\)")
ORDERSTOTAL_RE = re.compile(r"OrdersTotal\s*\(\s*\)")
POSITIONSELECT_RE = re.compile(r"PositionSelect\s*\(([^)]+)\)")
POSITIONSTOTAL_RE = re.compile(r"PositionsTotal\s*\(\s*\)")

# Built-in position/order accessor mappings.
ORDER_ACCESSOR_MAP: Dict[str, str] = {
    "OrderTicket": "order.ticket",
    "OrderSymbol": "order.symbol",
    "OrderType": "order.type",
    "OrderVolume": "order.volume",
    "OrderOpenPrice": "order.open_price",
    "OrderStopLoss": "order.sl",
    "OrderTakeProfit": "order.tp",
    "OrderMagicNumber": "order.magic",
    "OrderComment": "order.comment",
    "OrderOpenTime": "order.open_time_ms",
    "OrderProfit": "order.profit",
    "OrderSwap": "order.swap",
    "OrderClosePrice": "order.close_price",
    "OrderCloseTime": "order.close_time_ms",
    "OrderLots": "order.volume",
}

POSITION_ACCESSOR_MAP: Dict[str, str] = {
    "PositionTicket": "pos.ticket",
    "PositionSymbol": "pos.symbol",
    "PositionType": "pos.side_value",
    "PositionVolume": "pos.volume",
    "PositionOpenPrice": "pos.open_price",
    "PositionStopLoss": "pos.sl",
    "PositionTakeProfit": "pos.tp",
    "PositionMagic": "pos.magic",
    "PositionComment": "pos.comment",
    "PositionProfit": "pos.profit",
    "PositionSwap": "pos.swap",
}

# ── Common MQL functions → Python equivalents ──────────────────────────

COMMON_FUNC_MAP: Dict[str, str] = {
    "Print": "print",
    "Comment": "print",  # approximate
    "Alert": "print",
    "MessageBox": "print",
    "MathAbs": "abs",
    "MathMax": "max",
    "MathMin": "min",
    "MathRound": "round",
    "MathFloor": "int",
    "MathCeil": "lambda x: int(x) + 1 if x > int(x) else int(x)",
    "MathSqrt": "lambda x: x ** 0.5",
    "MathPow": "pow",
    "MathLog": "lambda x: __import__('math').log(x)",
    "MathExp": "lambda x: __import__('math').exp(x)",
    "MathSin": "lambda x: __import__('math').sin(x)",
    "MathCos": "lambda x: __import__('math').cos(x)",
    "MathTan": "lambda x: __import__('math').tan(x)",
    "MathArccos": "lambda x: __import__('math').acos(x)",
    "MathArcsin": "lambda x: __import__('math').asin(x)",
    "MathArctan": "lambda x: __import__('math').atan(x)",
    "MathMod": "lambda a,b: a % b",
    "MathRand": "lambda: __import__('random').randint(0, 32767)",
    "MathSrand": "lambda s: __import__('random').seed(s)",
    "StringConcatenate": "+",
    "StringFind": ".find",
    "StringLen": "len",
    "StringSubstr": "lambda s, start, length: s[start:start+length]",
    "StringReplace": ".replace",
    "DoubleToString": "lambda d, digits: str(round(float(d), digits))",
    "StringToDouble": "float",
    "IntegerToString": "str",
    "TimeCurrent": "self.broker.server_time()",
    "TimeLocal": "self.broker.server_time()",
    "AccountBalance": "self.broker.account().balance",
    "AccountEquity": "self.broker.account().equity",
    "AccountFreeMargin": "self.broker.account().free_margin",
    "AccountMargin": "self.broker.account().margin",
    "AccountLeverage": "self.broker.account().leverage",
    "AccountCurrency": "self.broker.account().currency",
    "AccountCredit": "self.broker.account().credit",
    "AccountCompany": "self.broker.account().company",
    "AccountName": "self.broker.account().name",
    "AccountNumber": "self.broker.account().number",
    "AccountProfit": "self.broker.account().profit",
    "AccountServer": "self.broker.account().server",
    "AccountStopoutLevel": "self.broker.account().stopout_level",
    "AccountStopoutMode": "self.broker.account().stopout_mode",
    "AccountFreeMarginCheck": "self.broker.account().free_margin_check",
    "AccountFreeMarginMode": "self.broker.account().free_margin_mode",
    "Symbol": "self.ctx.symbol",
    "Period": "self.ctx.timeframe",
    "Digits": "self.broker.symbol_info(self.ctx.symbol).digits",
    "Point": "self.broker.symbol_info(self.ctx.symbol).point",
    "MarketInfo": "self.broker.symbol_info",
    "Bid": "series_data_bid",
    "Ask": "series_data_ask",
    "Spread": "series_data_spread",
    # Order accessors not yet in builtin refs
    "OrderCloseTime": "order.close_time_ms",
    "OrderOpenTime": "order.open_time_ms",
    "OrderExpiration": "order.expiration",
    "OrderPrint": "TRANSPILER-GAP: OrderPrint (use self.ctx.log)",
    "OrdersHistoryTotal": "len(self.broker.history_orders())",
    "Open": "bars.open",
    "High": "bars.high",
    "Low": "bars.low",
    "Close": "bars.close",
    "Volume": "bars.volume",
    "Time": "bars.time",
    "iOpen": "bars.open",
    "iHigh": "bars.high",
    "iLow": "bars.low",
    "iClose": "bars.close",
    "iVolume": "bars.volume",
    "iTime": "bars.time",
    "iBars": "bars.total()",
    "Bars": "bars.total",
    "ArrayInitialize": "TRANSPILER-GAP: ArrayInitialize",
    "ArrayResize": "TRANSPILER-GAP: ArrayResize",
    "ArrayCopy": "TRANSPILER-GAP: ArrayCopy",
    "ArraySize": "len",
    "ArrayGetAsSeries": "TRANSPILER-GAP: ArrayGetAsSeries",
    "ArraySetAsSeries": "TRANSPILER-GAP: ArraySetAsSeries",
    "EventSetTimer": "self.ctx.set_timer",
    "EventKillTimer": "self.ctx.kill_timer",
    "Sleep": "TRANSPILER-GAP: Sleep",
    "GetLastError": "TRANSPILER-GAP: GetLastError",
    "ResetLastError": "TRANSPILER-GAP: ResetLastError",
    "FileOpen": "TRANSPILER-GAP: FileIO",
    "FileClose": "TRANSPILER-GAP: FileIO",
    "FileWrite": "TRANSPILER-GAP: FileIO",
    "FileRead": "TRANSPILER-GAP: FileIO",
    "WebRequest": "TRANSPILER-GAP: WebRequest",
    "SendNotification": "TRANSPILER-GAP: SendNotification",
    # MQL5 Position/History getters (stateful — manual translation needed)
    "PositionSelect": "TRANSPILER-GAP: PositionSelect — use self.broker.positions()",
    "PositionSelectByTicket": "TRANSPILER-GAP: PositionSelectByTicket",
    "PositionGetDouble": "TRANSPILER-GAP: PositionGetDouble — use pos.open_price etc.",
    "PositionGetInteger": "TRANSPILER-GAP: PositionGetInteger — use pos.ticket etc.",
    "PositionGetString": "TRANSPILER-GAP: PositionGetString — use pos.symbol etc.",
    "PositionGetSymbol": "TRANSPILER-GAP: PositionGetSymbol — use pos.symbol",
    "PositionGetTicket": "TRANSPILER-GAP: PositionGetTicket — use pos.ticket",
    "HistorySelect": "TRANSPILER-GAP: HistorySelect — use self.broker.deals(from, to)",
    "HistorySelectByPosition": "TRANSPILER-GAP: HistorySelectByPosition",
    "HistoryDealsTotal": "TRANSPILER-GAP: HistoryDealsTotal",
    "HistoryDealSelect": "TRANSPILER-GAP: HistoryDealSelect",
    "HistoryDealGetTicket": "TRANSPILER-GAP: HistoryDealGetTicket",
    "HistoryDealGetDouble": "TRANSPILER-GAP: HistoryDealGetDouble",
    "HistoryDealGetInteger": "TRANSPILER-GAP: HistoryDealGetInteger",
    "HistoryDealGetString": "TRANSPILER-GAP: HistoryDealGetString",
    "HistoryOrdersTotal": "TRANSPILER-GAP: HistoryOrdersTotal — use self.broker.history_orders()",
    "HistoryOrderGetTicket": "TRANSPILER-GAP: HistoryOrderGetTicket",
    "HistoryOrderGetDouble": "TRANSPILER-GAP: HistoryOrderGetDouble",
    "HistoryOrderGetInteger": "TRANSPILER-GAP: HistoryOrderGetInteger",
    "HistoryOrderGetString": "TRANSPILER-GAP: HistoryOrderGetString",
    "OrderSendAsync": "TRANSPILER-GAP: OrderSendAsync — use self.broker.order_send (sync)",
    "OrderCalcMargin": "TRANSPILER-GAP: OrderCalcMargin — broker-side calc, not available",
    "OrderCalcProfit": "TRANSPILER-GAP: OrderCalcProfit — broker-side calc, not available",
    "OrderCheck": "TRANSPILER-GAP: OrderCheck — broker-side check, not available",
    "OrderGetDouble": "TRANSPILER-GAP: OrderGetDouble — use order.open_price etc.",
    "OrderGetInteger": "TRANSPILER-GAP: OrderGetInteger — use order.ticket etc.",
    "OrderGetString": "TRANSPILER-GAP: OrderGetString — use order.symbol etc.",
    # SymbolInfo getters
    "SymbolInfoDouble": "TRANSPILER-GAP: SymbolInfoDouble — use self.broker.symbol_info(sym)",
    "SymbolInfoInteger": "TRANSPILER-GAP: SymbolInfoInteger — use self.broker.symbol_info(sym)",
    "SymbolInfoString": "TRANSPILER-GAP: SymbolInfoString — use self.broker.symbol_info(sym)",
    "SymbolSelect": "TRANSPILER-GAP: SymbolSelect — symbols auto-available in SDK",
    "SendMail": "TRANSPILER-GAP: SendMail",
    "SendFTP": "TRANSPILER-GAP: SendFTP",
    "IndicatorCreate": "TRANSPILER-GAP: DLL/IndicatorCreate",
    "IndicatorRelease": "TRANSPILER-GAP: DLL/IndicatorRelease",
    "ObjectsTotal": "TRANSPILER-GAP: GUI",
    "ObjectCreate": "TRANSPILER-GAP: GUI",
    "ObjectDelete": "TRANSPILER-GAP: GUI",
    "ObjectSetDouble": "TRANSPILER-GAP: GUI",
    "ObjectSetInteger": "TRANSPILER-GAP: GUI",
    "ObjectSetString": "TRANSPILER-GAP: GUI",
    "ObjectGetDouble": "TRANSPILER-GAP: GUI",
    "ObjectGetInteger": "TRANSPILER-GAP: GUI",
    "ObjectGetString": "TRANSPILER-GAP: GUI",
    "ChartOpen": "TRANSPILER-GAP: GUI",
    "ChartClose": "TRANSPILER-GAP: GUI",
}


@dataclass
class MappingResult:
    """Result of applying a mapping rule."""
    success: bool
    output: str
    gap_reason: str = ""


GapReason = str  # human-readable reason for TRANSPILER-GAP
