"""ExpressionGen — 原子表达式翻译器，从 ast_transpiler 抽取。

负责将 MQL AST 表达式节点翻译为 Python SDK 代码片段。
独立可用，不依赖 ASTTranspiler 的代码组织逻辑。

Usage::

    from tools.mql_migration.expression_gen import ExpressionGen
    gen = ExpressionGen()
    py = gen.translate(expression_node)  # "self.indicators.ema(period=14, shift=1)"
"""

from __future__ import annotations

from typing import Dict, List, Optional, Set, Tuple

from tools.mql_transpiler.ast_nodes import (
    ArrayInitExpr,
    AssignmentExpr,
    BinaryOp,
    CallExpr,
    Expression,
    Identifier,
    NumberLiteral,
    StringLiteral,
    SubscriptExpr,
    TernaryExpr,
    UnaryOp,
)


# ── MQL constants → Python ─────────────────────────────────────────────

_MQL_CONSTANTS: Dict[str, str] = {
    "INIT_SUCCEEDED": "0",
    "OP_BUY": "OrderType.BUY",
    "OP_SELL": "OrderType.SELL",
    "OP_BUYLIMIT": "OrderType.BUY_LIMIT",
    "OP_SELLLIMIT": "OrderType.SELL_LIMIT",
    "OP_BUYSTOP": "OrderType.BUY_STOP",
    "OP_SELLSTOP": "OrderType.SELL_STOP",
    "MODE_EMA": "'ema'",
    "MODE_SMA": "'sma'",
    "MODE_SMMA": "'smma'",
    "MODE_LWMA": "'lwma'",
    "PRICE_CLOSE": "1",
    "PRICE_OPEN": "2",
    "SELECT_BY_POS": "",
    "SELECT_BY_TICKET": "",
    "MODE_TRADES": "",
    "clrNONE": "None",
    "true": "True",
    "false": "False",
    "NULL": "None",
}


class ExpressionGen:
    """Generate Python SDK code from MQL AST expression nodes."""

    def __init__(self, mappings: Optional[Dict[str, str]] = None,
                 rpc_params: Optional[Dict] = None,
                 known_vars: Optional[Set[str]] = None):
        self._known_vars = known_vars or set()
        self._local_vars: Set[str] = set()
        self._inside_orderselect_loop = False
        self._order_loop_var = "order"

        self._mappings = mappings
        self._rpc_params = rpc_params
        self._mql_to_sdk = None

    # ── Public API ──────────────────────────────────────────────────

    def translate(self, expr: Optional[Expression]) -> str:
        """Translate a single AST expression to Python code."""
        return self._expr_to_py(expr)

    def set_known_vars(self, vars_: Set[str]) -> None:
        self._known_vars = vars_

    def set_loop_context(self, inside: bool, var_name: str = "order") -> None:
        self._inside_orderselect_loop = inside
        self._order_loop_var = var_name

    def add_local(self, name: str) -> None:
        self._local_vars.add(name)

    def clear_locals(self) -> None:
        self._local_vars.clear()

    # ── Local variable scoping ──────────────────────────────────────

    class _LocalScope:
        """Context manager for temporary local variable scope.

        Usage::

            with expr_gen.local_scope(local_vars):
                py = expr_gen.translate(condition)
        """
        def __init__(self, gen, extra_vars: Set[str]):
            self._gen = gen
            self._extra = extra_vars
            self._saved = set(gen._local_vars)

        def __enter__(self):
            self._gen._local_vars |= self._extra
            return self

        def __exit__(self, *args):
            self._gen._local_vars = self._saved

    def local_scope(self, extra_vars: Set[str]):
        """Return a context manager that temporarily adds local variables.

        Automatically restores the previous state on exit.  Replaces the
        manual save/restore pattern used across all recognizers.
        """
        return self._LocalScope(self, extra_vars)

    # ── Expression → Python ─────────────────────────────────────────

    def _expr_to_py(self, expr: Optional[Expression]) -> str:
        if expr is None:
            return ""
        if isinstance(expr, NumberLiteral):
            return expr.value
        if isinstance(expr, StringLiteral):
            return f"'{expr.value}'"
        if isinstance(expr, Identifier):
            return self._map_ident(expr.name)
        if isinstance(expr, CallExpr):
            return self._map_call(expr)
        if isinstance(expr, SubscriptExpr):
            return self._translate_subscript(expr)
        if isinstance(expr, BinaryOp):
            return self._translate_binary(expr)
        if isinstance(expr, UnaryOp):
            return self._translate_unary(expr)
        if isinstance(expr, AssignmentExpr):
            rhs = self._expr_to_py(expr.rhs)
            return f"self.{expr.lhs} = {rhs}"
        if isinstance(expr, TernaryExpr):
            cond = self._expr_to_py(expr.condition)
            true_v = self._expr_to_py(expr.true_val)
            false_v = self._expr_to_py(expr.false_val)
            return f"({true_v} if {cond} else {false_v})"
        if isinstance(expr, ArrayInitExpr):
            elems = ", ".join(self._expr_to_py(e) for e in expr.elements)
            return f"[{elems}]"
        return str(expr)

    def _translate_subscript(self, expr: SubscriptExpr) -> str:
        base = self._map_ident(expr.name)
        idx = self._expr_to_py(expr.index) if expr.index else "0"
        if base in ("open", "high", "low", "close", "volume", "time"):
            return f"self.ctx.bars().{base}[{idx}]"
        return f"{base}[{idx}]"

    def _translate_binary(self, expr: BinaryOp) -> str:
        left = self._expr_to_py(expr.left)
        right = self._expr_to_py(expr.right)
        op = expr.op.replace("&&", "and").replace("||", "or")
        if op == "/":
            if "Decimal(" not in left:
                left = f"Decimal(str({left}))"
            if "Decimal(" not in right:
                right = f"Decimal(str({right}))"
        return f"{left} {op} {right}"

    def _translate_unary(self, expr: UnaryOp) -> str:
        operand = self._expr_to_py(expr.operand)
        if expr.op == "!":
            return f"not {operand}"
        return f"{expr.op}{operand}"

    # ── Identifier → Python ─────────────────────────────────────────

    # Builtin identifiers (Symbol, Ask, Bid, etc.)
    _MQL_BUILTIN_IDENTS: Dict[str, str] = {
        "Symbol": "self.ctx.symbol",
        "Ask": "self.ctx.ask",
        "Bid": "self.ctx.bid",
        "Point": "self.ctx.point",
        "Digits": "self.broker.symbol_info(self.ctx.symbol).digits",
        "Bars": "self.ctx.bars().total()",
        "Close": "close",
        "Open": "open",
        "High": "high",
        "Low": "low",
        "Volume": "volume",
        "Time": "time",
        "PERIOD_M1": "None",
        "PERIOD_M5": "None",
        "PERIOD_M15": "None",
        "PERIOD_M30": "None",
        "PERIOD_H1": "None",
        "PERIOD_H4": "None",
        "PERIOD_D1": "None",
        "PERIOD_W1": "None",
        "PERIOD_MN1": "None",
        "PERIOD_CURRENT": "None",
        "PRICE_CLOSE": "1",
        "PRICE_OPEN": "2",
        "PRICE_HIGH": "3",
        "PRICE_LOW": "4",
        "PRICE_MEDIAN": "5",
        "PRICE_TYPICAL": "6",
        "PRICE_WEIGHTED": "7",
    }

    # Order accessor → SDK field name
    _ORDER_ACCESSOR_SDK: Dict[str, str] = {
        "OrderMagicNumber": "order.magic",
        "OrderTicket": "order.ticket",
        "OrderLots": "order.volume",
        "OrderSymbol": "order.symbol",
        "OrderType": "order.type",
        "OrderOpenPrice": "order.open_price",
        "OrderClosePrice": "order.close_price",
        "OrderStopLoss": "order.sl",
        "OrderTakeProfit": "order.tp",
        "OrderComment": "order.comment",
        "OrderCommission": "order.commission",
        "OrderSwap": "order.swap",
        "OrderProfit": "order.profit",
        "OrderOpenTime": "order.open_time_ms",
        "OrderCloseTime": "order.close_time_ms",
        "OrderVolume": "order.volume",
    }

    def _map_ident(self, name: str) -> str:
        if name.startswith("__raw__"):
            return name[7:]

        if name in _MQL_CONSTANTS:
            return _MQL_CONSTANTS[name]

        if name in self._MQL_BUILTIN_IDENTS:
            return self._MQL_BUILTIN_IDENTS[name]

        if self._inside_orderselect_loop and name in self._ORDER_ACCESSOR_SDK:
            sdk = self._ORDER_ACCESSOR_SDK[name]
            lv = self._order_loop_var
            if sdk.startswith("order."):
                return sdk.replace("order.", f"{lv}.", 1)
            return sdk

        mappings = self._get_mappings()
        if name in mappings:
            py = mappings[name]
            if not py.startswith("TRANSPILER-GAP") and not py.startswith("lambda") and "(" not in py:
                return py

        if name in self._local_vars:
            return name

        if name in self._known_vars:
            return f"self.{name}"

        return f"self.{name}"

    # ── Call → Python ───────────────────────────────────────────────

    # MQL indicator signatures
    _INDICATOR_SIGNATURES: Dict[str, Tuple[str, List[Tuple[int, str]]]] = {
        "iMA":         ("ma",   [(2, "period"), (3, "shift"), (4, "method")]),
        "iRSI":        ("rsi",  [(2, "period"), (4, "shift")]),
        "iBands":      ("bands", [(2, "period"), (3, "deviation"), (7, "shift")]),
        "iMACD":       ("macd", [(2, "fast"), (3, "slow"), (4, "signal"), (7, "shift")]),
        "iATR":        ("atr",  [(2, "period"), (3, "shift")]),
        "iStochastic": ("stochastic", [(2, "k_period"), (3, "d_period"), (7, "shift")]),
        "iCCI":        ("cci",  [(2, "period"), (4, "shift")]),
        "iADX":        ("adx",  [(2, "period"), (4, "shift")]),
        "iMomentum":   ("momentum", [(2, "period"), (5, "shift")]),
        "iMFI":        ("mfi",  [(2, "period"), (4, "shift")]),
        "iOBV":        ("obv",  [(4, "shift")]),
        "iSAR":        ("sar",  [(2, "step"), (3, "maximum"), (4, "shift")]),
        "iStdDev":     ("stddev", [(2, "period"), (6, "shift")]),
        "iWPR":        ("wpr",  [(2, "period"), (4, "shift")]),
        "iEnvelopes":  ("envelopes", [(2, "period"), (6, "deviation"), (7, "shift")]),
        "iForce":      ("force", [(2, "period"), (5, "shift")]),
        "iDeMarker":   ("demarker", [(2, "period"), (4, "shift")]),
        "iOsMA":       ("osma", [(2, "fast"), (3, "slow"), (4, "signal"), (7, "shift")]),
    }

    # No-arg time functions
    _TIME_FUNCTIONS: Dict[str, str] = {
        "Day": "__import__('datetime').datetime.now().day",
        "Month": "__import__('datetime').datetime.now().month",
        "Year": "__import__('datetime').datetime.now().year",
        "Hour": "__import__('datetime').datetime.now().hour",
        "Minute": "__import__('datetime').datetime.now().minute",
        "Seconds": "__import__('datetime').datetime.now().second",
    }

    def _map_call(self, call: CallExpr) -> str:
        name = call.name
        args_str = ", ".join(self._expr_to_py(a) for a in call.args)

        if name == "Symbol":
            return "self.ctx.symbol"

        if name == "MarketInfo":
            return self._map_market_info(call)

        if self._inside_orderselect_loop and name in self._ORDER_ACCESSOR_SDK:
            sdk = self._ORDER_ACCESSOR_SDK[name]
            lv = self._order_loop_var
            if sdk.startswith("order."):
                return sdk.replace("order.", f"{lv}.", 1)
            return sdk

        mql_to_sdk = self._get_mql_to_sdk()
        lookup = f"{name}()"
        if lookup in mql_to_sdk:
            sdk = mql_to_sdk[lookup]
            if self._inside_orderselect_loop:
                _FIXUP = {
                    "order.magic_number": "order.magic",
                    "order.lots": "order.volume",
                    "order.stop_loss": "order.sl",
                    "order.take_profit": "order.tp",
                    "order.order_type": "order.type",
                }
                sdk = _FIXUP.get(sdk, sdk)
                lv = self._order_loop_var
                if sdk.startswith("order."):
                    sdk = sdk.replace("order.", f"{lv}.", 1)
            return sdk

        # Trade functions
        if name == "OrderSend":
            return self._map_rpc_call_to_sdk(call, "OrderSend", "order_send")
        if name == "OrderClose":
            return self._map_orderclose_to_sdk(call)
        if name == "OrderModify":
            return self._map_rpc_call_to_sdk(call, "OrderModify", "position_modify")
        if name == "OrderDelete":
            return self._map_rpc_call_to_sdk(call, "OrderDelete", "order_delete")
        if name == "OrderSelect":
            return "True"

        # Indicators
        indicator_map = self._map_indicator_call(name, call)
        if indicator_map is not None:
            return indicator_map

        # Time functions
        if name in self._TIME_FUNCTIONS:
            return self._TIME_FUNCTIONS[name]

        # Common functions from mappings
        mappings = self._get_mappings()
        if name in mappings:
            py = mappings[name]
            if py.startswith("TRANSPILER-GAP") or py == "":
                return f"# TRANSPILER-GAP: {name}"
            if py.startswith("lambda"):
                return _expand_lambda(py, args_str)
            if "(" in py and py.endswith(")"):
                return py.replace("()", f"({args_str})")
            elif py.endswith(")"):
                return f"{py[:-1]}{args_str})"
            else:
                return f"{py}({args_str})"

        return f"self.{name}({args_str})"

    def _map_indicator_call(self, name: str, call: CallExpr) -> Optional[str]:
        sig = self._INDICATOR_SIGNATURES.get(name)
        if sig is None:
            if name == "iCustom":
                return self._map_icustom_call(call)
            return None

        sdk_name, arg_map = sig
        py_args = [self._expr_to_py(a) for a in call.args]

        kwargs = []
        for mql_idx, sdk_param in arg_map:
            if mql_idx >= len(py_args):
                continue
            val = py_args[mql_idx]
            val_str = str(val)
            if val_str in ("0", "None", "''", '""') and sdk_param in ("shift",):
                if val_str == "0":
                    continue
            if val_str in ("0", "0.0") and sdk_param in ("deviation",):
                continue
            if sdk_param == "method":
                val = self._map_indicator_method(val_str)
            kwargs.append(f"{sdk_param}={val}")

        return f"self.indicators.{sdk_name}({', '.join(kwargs)})"

    @staticmethod
    def _map_indicator_method(raw: str) -> str:
        return {
            "0": "'sma'", "'ema'": "'ema'", "MODE_SMA": "'sma'",
            "MODE_EMA": "'ema'", "MODE_SMMA": "'smma'", "MODE_LWMA": "'lwma'",
        }.get(raw.strip("'\""), raw)

    def _map_icustom_call(self, call: CallExpr) -> str:
        py_args = [self._expr_to_py(a) for a in call.args]
        if len(py_args) < 3:
            return "self.indicators.i_custom(name='unknown', params=[], buffer=0, shift=0)"

        name_raw = str(py_args[2])
        if name_raw.startswith("'") and name_raw.endswith("'"):
            name_expr = f"'{name_raw[1:-1]}'"
        elif name_raw.startswith('"') and name_raw.endswith('"'):
            name_expr = f"'{name_raw[1:-1]}'"
        else:
            name_expr = name_raw

        if len(py_args) >= 5:
            buffer_val = py_args[-2]
            shift_val = py_args[-1]
            param_vals = py_args[3:-2] if len(py_args) > 5 else []
        else:
            buffer_val = "0"
            shift_val = "0"
            param_vals = py_args[3:] if len(py_args) > 3 else []

        params_str = ", ".join(str(p) for p in param_vals)
        return f"self.indicators.i_custom(name={name_expr}, params=[{params_str}], buffer={buffer_val}, shift={shift_val})"

    def _map_orderclose_to_sdk(self, call: CallExpr) -> str:
        py_args = [self._expr_to_py(a) for a in call.args]
        ticket = py_args[0] if len(py_args) > 0 else "0"
        volume = py_args[1] if len(py_args) > 1 else "Decimal(str(0))"
        if "OrderLots" in str(volume):
            return f"self.broker.position_close({ticket})"
        return f"self.broker.position_close({ticket}, volume=Decimal(str({volume})))"

    def _map_rpc_call_to_sdk(self, call: CallExpr, rpc_name: str, sdk_method: str) -> str:
        args = call.args
        params = self._get_rpc_params().get(rpc_name, [])
        parts = []
        for i, (sdk_name, _proto_field, _fnum) in enumerate(params):
            if i >= len(args):
                break
            val_raw = self._expr_to_py(args[i])
            val = val_raw
            if val in ("0", "0.0", "None", "''", '""') and sdk_name in ("sl", "tp", "price", "expiration"):
                continue
            if sdk_name in ("volume", "price") and val not in ("None", "0", "0.0", ""):
                val = f"Decimal(str({val}))"
            if sdk_name in ("sl", "tp") and val not in ("None", "0", "0.0"):
                val = f"Decimal(str({val}))"
            if sdk_name == "magic":
                if val in ("0", "None", "0.0", ""):
                    continue
                val = f"int({val})"
            if sdk_name == "deviation":
                val = f"int({val})" if val not in ("0", "None", "") else "3"
            if sdk_name == "comment":
                if val in ("''", '""', ""):
                    continue
            parts.append(f"{sdk_name}={val}")

        if sdk_method == "order_delete":
            ticket_val = parts[0].split("=", 1)[1] if parts else "0"
            return f"self.broker.{sdk_method}({ticket_val})"
        return f"self.broker.{sdk_method}(OrderRequest({', '.join(parts)}))"

    def _map_market_info(self, call: CallExpr) -> str:
        py_args = [self._expr_to_py(a) for a in call.args]
        symbol = py_args[0] if len(py_args) > 0 else "self.ctx.symbol"
        mode = str(py_args[1]) if len(py_args) > 1 else ""
        mode_clean = mode.replace("self.", "").strip("'\"")
        attr = {
            "MODE_POINT": "point", "POINT": "point",
            "MODE_DIGITS": "digits", "DIGITS": "digits",
            "MODE_SPREAD": "spread", "SPREAD": "spread",
            "MODE_STOPLEVEL": "stops_level",
            "MODE_LOTSIZE": "contract_size",
            "MODE_TICKVALUE": "tick_value",
            "MODE_TICKSIZE": "tick_size",
            "MODE_SWAPLONG": "swap_long",
            "MODE_SWAPSHORT": "swap_short",
            "0": "point", "1": "digits",
        }.get(mode_clean)
        if attr:
            return f"self.broker.symbol_info({symbol}).{attr}"
        return f"self.broker.symbol_info({symbol})"

    # ── Lazy dependency loading ──────────────────────────────────────

    def _get_mappings(self) -> Dict[str, str]:
        if self._mappings is None:
            try:
                from tools.mql_transpiler.mappings import COMMON_FUNC_MAP
                self._mappings = COMMON_FUNC_MAP
            except ImportError:
                self._mappings = {}
        return self._mappings

    def _get_rpc_params(self) -> Dict:
        if self._rpc_params is None:
            try:
                from tools.mql_transpiler.rpc_params_generated import RPC_PARAM_ORDER
                self._rpc_params = RPC_PARAM_ORDER
            except ImportError:
                self._rpc_params = {}
        return self._rpc_params

    def _get_mql_to_sdk(self) -> Dict:
        if self._mql_to_sdk is None:
            try:
                from tools.mql_transpiler.mappings_generated import MQL_TO_SDK_ACCESSOR
                self._mql_to_sdk = MQL_TO_SDK_ACCESSOR
            except ImportError:
                self._mql_to_sdk = {}
        return self._mql_to_sdk


# ── Lambda expansion (module-level, also used by ast_transpiler) ──────


def _expand_lambda(lambda_expr: str, args_str: str) -> str:
    """Expand a lambda mapping with the given argument string."""
    import re

    m = re.match(r'lambda\s*([^:]*?)\s*:\s*(.*)', lambda_expr, re.DOTALL)
    if not m:
        return lambda_expr

    params_part = m.group(1).strip()
    body = m.group(2).strip()

    if not params_part:
        return body

    param_names = [p.strip() for p in params_part.split(",")]
    arg_values = [a.strip() for a in args_str.split(",")] if args_str else []

    result = body
    for i, pname in enumerate(param_names):
        val = arg_values[i] if i < len(arg_values) else "None"
        result = re.sub(rf'\b{re.escape(pname)}\b', val, result)
    return result
