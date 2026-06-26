"""Python SDK 代码生成器 — 从意图 IR 生成原生 Python 策略代码。

遵循 SDK 惯用模式：on_bar 事件驱动、positions(magic=) 服务端过滤、
type_filling、AccountMode 检查、on_trade 回调等。
"""

from __future__ import annotations

from typing import List

from tools.mql_migration.intent_ir import (
    CloseAction,
    EntryRule,
    ExecutionKind,
    ExecutionModel,
    ExitRule,
    ExitTrigger,
    OrderAction,
    ParamSpec,
    RiskCheck,
    SizingKind,
    SizingRule,
    StrategyIntent,
    TimerRule,
)


class PythonCodeGenerator:
    """从意图 IR 生成 Python SDK 策略代码。

    使用 ExpressionGen 将语言无关的 AST 节点翻译为 Python。
    """

    def __init__(self, expr_gen=None):
        self._lines: List[str] = []
        self._indent = 0
        # ExpressionGen for AST→Python translation
        from tools.mql_migration.expression_gen import ExpressionGen
        self._expr_gen = expr_gen or ExpressionGen()

    def generate(self, intent: StrategyIntent) -> str:
        """生成完整的 Python 策略文件。"""
        self._lines = []
        self._indent = 0

        self._gen_header(intent)
        self._gen_class_def(intent)
        self._indent += 1
        self._gen_on_init(intent)
        self._gen_main_loop(intent)
        if intent.execution.require_on_trade:
            self._gen_on_trade(intent)
        if intent.timer:
            self._gen_on_timer(intent.timer, intent.risk)
        self._gen_on_deinit(intent)
        self._gen_helper_methods(intent)
        self._indent -= 1

        return "\n".join(self._lines)

    # ── Header ─────────────────────────────────────────────────────

    def _gen_header(self, intent: StrategyIntent) -> None:
        self._emit(f'"""Translated from MQL by ant migration engine.')
        if intent.blind_spots:
            for bs in intent.blind_spots:
                self._emit(f"  BLINDSPOT[{bs.severity.value}]: {bs.description}")
        self._emit('"""')
        self._emit()
        self._emit("from decimal import Decimal, ROUND_HALF_UP")
        self._emit("from app.sdk import (")
        self._emit("    AccountMode,")
        self._emit("    OrderRequest,")
        self._emit("    OrderType,")
        self._emit("    PositionSide,")
        self._emit("    Retcode,")
        self._emit("    StrategyBase,")
        self._emit("    TypeFilling,")
        self._emit(")")
        self._emit()

    # ── Class definition ───────────────────────────────────────────

    def _gen_class_def(self, intent: StrategyIntent) -> None:
        name = intent.meta.name or "ImportedStrategy"
        self._emit(f"class {name}(StrategyBase):")
        desc = intent.meta.description or f"Imported strategy — {intent.meta.mql_version.value}."
        self._emit(f'    """{desc}"""')

    # ── on_init ────────────────────────────────────────────────────

    def _gen_on_init(self, intent: StrategyIntent) -> None:
        self._emit()
        self._emit("def on_init(self) -> None:")

        self._indent += 1

        # Parameters — every value is ctx.param() so user can configure
        for p in intent.params:
            self._emit(f"self.{p.name} = self.ctx.param('{p.name}', {_py_default(p)})")

        # Ensure a convenience alias for lot size exists for code generation
        lot_params = [p for p in intent.params if "lot" in p.name.lower()]
        if lot_params:
            self._emit(f"self.lot_size = Decimal(str(self.{lot_params[0].name}))")

        # State variables
        for sv in intent.state:
            self._emit(f"self.{sv.name} = {sv.initial_value}")

        # Timer
        if intent.timer:
            self._emit(f"self.ctx.set_timer({intent.timer.interval_seconds})")

        self._emit()
        self._emit("# Verify account mode.")
        self._emit("account = self.broker.account()")
        self._emit("self._hedging = account.mode == AccountMode.HEDGING")

        self._indent -= 1

    # ── Main loop ──────────────────────────────────────────────────

    def _gen_main_loop(self, intent: StrategyIntent) -> None:
        exec_model = intent.execution

        if exec_model.kind == ExecutionKind.ON_BAR:
            self._gen_on_bar(intent)
        elif exec_model.kind == ExecutionKind.ON_INIT_GRID:
            self._gen_on_init_grid(intent)
        else:
            self._gen_on_tick(intent)

    def _gen_on_bar(self, intent: StrategyIntent) -> None:
        self._emit()
        self._emit("def on_bar(self, timeframe: str) -> None:")
        self._indent += 1
        self._emit("if timeframe != self.ctx.bars().timeframe:")
        self._emit("    return")
        self._emit()

        # Emit indicator computations referenced in entry conditions
        self._gen_indicator_vars(intent)

        # Magic-close exits: close existing positions BEFORE opening new ones
        magic_exits = [e for e in intent.exit
                      if e.trigger == ExitTrigger.MAGIC_CLOSE]
        # Reverse-signal exits: close on opposite signal before opening
        reverse_exits = [e for e in intent.exit
                        if e.trigger == ExitTrigger.REVERSE_SIGNAL]

        # Emit close-existing blocks (magic_close, reverse_signal) before entries
        for exit_rule in magic_exits + reverse_exits:
            self._gen_exit_block(exit_rule)

        # Entries
        for entry in intent.entry:
            self._gen_entry_block(entry)

        # Other exits
        for exit_rule in intent.exit:
            if exit_rule.trigger not in (ExitTrigger.MAGIC_CLOSE, ExitTrigger.REVERSE_SIGNAL):
                self._gen_exit_block(exit_rule)

        for risk in intent.risk:
            if risk.trigger == "on_bar":
                self._gen_risk_block(risk)

        self._indent -= 1

    def _gen_indicator_vars(self, intent: StrategyIntent) -> None:
        """Emit indicator computations from structured IndicatorSpec list.

        Uses intent.indicators (language-agnostic IR) instead of guessing
        from variable names. Each IndicatorSpec is translated to the
        appropriate SDK call with params.
        """
        emitted = set()
        for spec in intent.indicators:
            if not spec.result_var:
                continue
            if spec.result_var in emitted:
                continue
            code = self._indicator_code(spec, intent)
            if code:
                self._emit(code)
                emitted.add(spec.result_var)

    def _indicator_code(self, spec, intent: StrategyIntent) -> str | None:
        """Generate SDK indicator call from IndicatorSpec.

        Maps sdk_method to the appropriate SDK call with params inferred
        from the intent's ParamSpec list.
        """
        method = spec.sdk_method
        result_var = spec.result_var

        # Infer period param from intent params matching this indicator
        if method in ("ma", "ema"):
            period = self._find_param_for(intent, ["ma", "period", "fast", "slow"], 14)
            return f"{result_var} = self.indicators.ema(period={period}, shift=1)"
        elif method == "rsi":
            period = self._find_param_for(intent, ["rsi", "period"], 14)
            return f"{result_var} = self.indicators.rsi(period={period}, shift=1)"
        elif method == "atr":
            period = self._find_param_for(intent, ["atr", "period"], 14)
            return f"{result_var} = self.indicators.atr(period={period}, shift=1)"
        elif method == "i_custom":
            name = self._find_param_for(intent, ["custom", "name", "indicator"], "CustomIndicator")
            return f"{result_var} = self.indicators.i_custom(name={name}, params=[], buffer=0, shift=1)"
        elif method == "macd":
            fast = self._find_param_for(intent, ["fast", "macd"], 12)
            slow = self._find_param_for(intent, ["slow", "signal"], 26)
            return f"{result_var} = self.indicators.macd(fast={fast}, slow={slow}, shift=1)"
        elif method == "bands":
            period = self._find_param_for(intent, ["band", "period"], 20)
            return f"{result_var} = self.indicators.bands(period={period}, shift=1)"
        return None

    @staticmethod
    def _find_param_for(intent, keywords: list[str], default: int) -> str:
        """Find a param matching any of the given keywords, fall back to default."""
        for p in intent.params:
            lower = p.name.lower()
            if any(kw in lower for kw in keywords):
                return f"self.{p.name}"
        return str(default)

    def _gen_on_tick(self, intent: StrategyIntent) -> None:
        self._emit()
        self._emit("def on_tick(self) -> None:")
        self._indent += 1

        # Exits before entries (close existing, then open new)
        for exit_rule in intent.exit:
            self._gen_exit_block(exit_rule)
        for entry in intent.entry:
            self._gen_entry_block(entry)
        for risk in intent.risk:
            if risk.trigger in ("on_tick",):
                self._gen_risk_block(risk)

        self._indent -= 1

    def _gen_on_init_grid(self, intent: StrategyIntent) -> None:
        """Grid EAs: place all pending orders in on_init, mark as done."""
        self._emit()
        self._emit("def on_tick(self) -> None:")
        self._indent += 1
        self._emit("if self._grid_placed:")
        self._emit("    return")
        self._emit()

        for entry in intent.entry:
            self._gen_entry_block(entry)

        self._emit()
        self._emit("self._grid_placed = True")
        self._indent -= 1

    # ── Entry block ────────────────────────────────────────────────

    def _gen_entry_block(self, entry: EntryRule) -> None:
        # Translate conditions from AST nodes (language-agnostic) to Python
        cond_parts = []
        for c in entry.conditions:
            if c.ast_node is not None:
                py = self._expr_gen.translate(c.ast_node)
            else:
                py = c.expr  # fallback to pre-computed cache
            if py:
                cond_parts.append(py)
        cond_str = " and ".join(cond_parts)
        if cond_str:
            self._emit(f"if {cond_str}:")
            self._indent += 1

        ot = _order_type_enum(entry.action)
        symbol = entry.order_params.symbol or "self.ctx.symbol"
        volume = entry.order_params.volume or "self.lot_size"
        magic = entry.order_params.magic or "self.magic"

        tf = _type_filling(entry.action)

        req_parts = [f"symbol={symbol}", f"type={ot}",
                     f"volume=Decimal(str({volume}))",
                     f"magic=int({magic})"]
        if entry.order_params.sl and entry.order_params.sl not in ("0", "0.0", ""):
            req_parts.append(f"sl=Decimal(str({entry.order_params.sl}))")
        if entry.order_params.tp and entry.order_params.tp not in ("0", "0.0", ""):
            req_parts.append(f"tp=Decimal(str({entry.order_params.tp}))")
        if entry.order_params.deviation and entry.order_params.deviation not in ("0", ""):
            req_parts.append(f"deviation={entry.order_params.deviation}")
        if entry.order_params.comment and entry.order_params.comment not in ("''", '""', ""):
            req_parts.append(f"comment={entry.order_params.comment}")
        if tf:
            req_parts.append(f"type_filling={tf}")

        self._emit("req = OrderRequest(")
        self._indent += 1
        self._emit(",\n".join(req_parts) + ",")
        self._indent -= 1
        self._emit(")")
        self._emit("result = self.broker.order_send(req)")
        self._emit('if result.retcode.value in ("done", "done_partial"):')
        self._emit("    pass  # Entry acknowledged")

        if cond_str:
            self._indent -= 1

    # ── Exit block ─────────────────────────────────────────────────

    def _gen_exit_block(self, exit_rule: ExitRule) -> None:
        if exit_rule.trigger == ExitTrigger.MAGIC_CLOSE:
            self._gen_magic_close(exit_rule)
        elif exit_rule.trigger == ExitTrigger.MAGIC_DELETE:
            self._gen_magic_delete(exit_rule)
        elif exit_rule.trigger == ExitTrigger.REVERSE_SIGNAL:
            self._gen_reverse_signal_exit(exit_rule)
        elif exit_rule.trigger == ExitTrigger.CLOSE_ALL:
            self._gen_close_all(exit_rule)

    def _gen_magic_close(self, exit_rule: ExitRule) -> None:
        target = exit_rule.target
        cond_str = self._build_condition(exit_rule)
        magic_expr = target.magic_value or "self.magic"

        if cond_str:
            self._emit(f"if {cond_str}:")
            self._indent += 1

        if target.kind == "range":
            self._emit(f"for pos in self.broker.positions():")
            self._indent += 1
            self._emit(f"if {target.magic_min} <= pos.magic <= {target.magic_max}:")
            self._emit("    self.broker.position_close(pos.ticket)")
            self._indent -= 1
        else:
            self._emit(f"for pos in self.broker.positions(magic=int({magic_expr})):")
            self._emit("    self.broker.position_close(pos.ticket)")

        if cond_str:
            self._indent -= 1

    def _gen_magic_delete(self, exit_rule: ExitRule) -> None:
        target = exit_rule.target

        if target.kind == "range":
            self._emit("for order in self.broker.orders():")
            self._indent += 1
            self._emit(f"magic = order.magic")
            self._emit(f"if {target.magic_min} <= magic <= {target.magic_max}:")
            self._emit("    self.broker.order_delete(order.ticket)")
            self._indent -= 1
        else:
            magic_expr = target.magic_value or "self.magic"
            self._emit(f"for order in self.broker.orders(magic=int({magic_expr})):")
            self._emit("    self.broker.order_delete(order.ticket)")

    def _gen_reverse_signal_exit(self, exit_rule: ExitRule) -> None:
        cond_str = self._build_condition(exit_rule)
        if not cond_str:
            return

        self._emit(f"if {cond_str}:")
        self._indent += 1
        self._emit(f"for pos in self.broker.positions(magic=int(self.magic)):")
        self._emit("    self.broker.position_close(pos.ticket)")
        self._indent -= 1

    def _gen_close_all(self, exit_rule: ExitRule) -> None:
        self._emit("for pos in self.broker.positions():")
        self._emit("    self.broker.position_close(pos.ticket)")
        self._emit("for order in self.broker.orders():")
        self._emit("    self.broker.order_delete(order.ticket)")

    @staticmethod
    def _build_condition(exit_rule: ExitRule) -> str:
        if exit_rule.conditions:
            return " and ".join(c.expr for c in exit_rule.conditions)
        return ""

    # ── on_trade ───────────────────────────────────────────────────

    def _gen_on_trade(self, intent: StrategyIntent) -> None:
        self._emit()
        self._emit("def on_trade(self) -> None:")
        self._indent += 1

        # Martingale sizing adjustment after trade close
        if intent.sizing and intent.sizing.kind == SizingKind.MARTINGALE:
            self._emit("positions = self.broker.positions(magic=int(self.magic))")
            self._emit("if not positions and self._in_trade:")
            self._indent += 1
            self._emit("self._in_trade = False")
            self._emit("current_equity = self.broker.account().equity")
            self._emit("# Evaluate PnL and adjust lot size")
            self._emit("pnl = current_equity - self._trade_entry_equity")
            self._emit("if pnl > Decimal('0'):")
            self._emit("    self._current_lot = self.base_lot")
            self._emit("elif pnl < Decimal('0'):")
            self._emit("    multiplier = Decimal(str(self.ctx.param('MartingaleMultiplier', 2.0)))")
            self._emit("    doubled = (self._current_lot * multiplier).quantize(")
            self._emit("        Decimal('0.01'), rounding=ROUND_HALF_UP")
            self._emit("    )")
            self._emit("    max_lot = Decimal(str(self.ctx.param('MaxLot', 5.0)))")
            self._emit("    self._current_lot = min(doubled, max_lot)")
            self._indent -= 1
            self._emit("elif positions and not self._in_trade:")
            self._indent += 1
            self._emit("self._in_trade = True")
            self._emit("self._trade_entry_equity = self.broker.account().equity")
            self._indent -= 1

        self._indent -= 1

    # ── on_timer ───────────────────────────────────────────────────

    def _gen_on_timer(self, timer: TimerRule, risks: list = None) -> None:
        self._emit()
        self._emit("def on_timer(self) -> None:")
        self._indent += 1

        # Use risk IR conditions if available, otherwise fall back to template
        timer_risks = [r for r in (risks or []) if r.trigger == "on_timer"]
        if timer_risks:
            for risk in timer_risks:
                self._gen_risk_block(risk)
        else:
            # Safe fallback — uses configurable param
            self._emit("margin_threshold = "
                       "Decimal(str(self.ctx.param('MarginThreshold', 50)))")
            self._emit("if self.broker.account().margin_level < margin_threshold:")
            self._emit("    for pos in self.broker.positions(magic=int(self.magic)):")
            self._emit("        self.broker.position_close(pos.ticket)")

        self._indent -= 1

    # ── on_deinit ──────────────────────────────────────────────────

    def _gen_on_deinit(self, intent: StrategyIntent) -> None:
        self._emit()
        self._emit("def on_deinit(self, reason: str) -> None:")
        self._indent += 1

        if intent.timer:
            self._emit("self.ctx.kill_timer()")

        # Close all positions and pending orders for our magic numbers
        magic_values = set()
        for entry in intent.entry:
            if entry.order_params.magic:
                magic_values.add(entry.order_params.magic)
        for exit_rule in intent.exit:
            if exit_rule.target.magic_value:
                magic_values.add(exit_rule.target.magic_value)

        if magic_values:
            for mv in magic_values:
                self._emit(f"for pos in self.broker.positions(magic=int({mv})):")
                self._emit("    self.broker.position_close(pos.ticket)")
                self._emit(f"for order in self.broker.orders(magic=int({mv})):")
                self._emit("    self.broker.order_delete(order.ticket)")
        else:
            self._emit("if reason in ('user_stop', 'kill_switch'):")
            self._indent += 1
            self._emit("for pos in self.broker.positions():")
            self._emit("    self.broker.position_close(pos.ticket)")
            self._emit("for order in self.broker.orders():")
            self._emit("    self.broker.order_delete(order.ticket)")
            self._indent -= 1

        self._indent -= 1

    # ── Helper methods ─────────────────────────────────────────────

    def _gen_helper_methods(self, intent: StrategyIntent) -> None:
        # Generate close-by-magic helpers referenced by exit rules
        has_magic_close = any(
            e.trigger == ExitTrigger.MAGIC_CLOSE for e in intent.exit
        )

        if has_magic_close:
            self._emit()
            self._emit("def _close_by_magic(self, magic: int) -> None:")
            self._indent += 1
            self._emit("for pos in self.broker.positions(magic=magic):")
            self._emit("    self.broker.position_close(pos.ticket)")
            self._indent -= 1

    # ── Misc ───────────────────────────────────────────────────────

    def _emit(self, line: str = "") -> None:
        if line:
            self._lines.append("    " * self._indent + line)
        else:
            self._lines.append("")

    # ── Risk block ─────────────────────────────────────────────────

    def _gen_risk_block(self, risk: RiskCheck) -> None:
        if risk.kind == "margin_check":
            self._emit(f"if {risk.condition}:")
            self._emit("    for pos in self.broker.positions(magic=int(self.magic)):")
            self._emit("        self.broker.position_close(pos.ticket)")


# ── Helpers ────────────────────────────────────────────────────────────


def _extract_refs(expr: str) -> list[str]:
    """Extract variable names from a Python expression string."""
    import re
    # Match identifiers that aren't keywords, attributes, or function calls
    names = re.findall(r'\b([a-zA-Z_][a-zA-Z0-9_]*)\b', expr)
    keywords = {'and', 'or', 'not', 'if', 'else', 'True', 'False', 'None',
                'self', 'Decimal', 'int', 'float', 'str', 'len', 'abs',
                'min', 'max', 'round', 'pow', 'type'}
    # Filter out dotted refs and keywords
    result = []
    for n in names:
        if n not in keywords and not n.startswith('_'):
            # Check it's not part of a dotted reference like self.xxx or bars.xxx
            parts = expr.split()
            for part in parts:
                if n in part and '.' not in part.split(n)[0][-1:] if len(part.split(n)) > 1 else True:
                    result.append(n)
                    break
    return list(set(result))


def _order_type_enum(action: OrderAction) -> str:
    """OrderAction → OrderType enum string."""
    return {
        OrderAction.MARKET_BUY: "OrderType.BUY",
        OrderAction.MARKET_SELL: "OrderType.SELL",
        OrderAction.BUY_LIMIT: "OrderType.BUY_LIMIT",
        OrderAction.SELL_LIMIT: "OrderType.SELL_LIMIT",
        OrderAction.BUY_STOP: "OrderType.BUY_STOP",
        OrderAction.SELL_STOP: "OrderType.SELL_STOP",
    }.get(action, "OrderType.BUY")


def _type_filling(action: OrderAction) -> str:
    """Market orders → IOC, Pending orders → RETURN."""
    if action in (OrderAction.MARKET_BUY, OrderAction.MARKET_SELL):
        return "TypeFilling.IOC"
    return "TypeFilling.RETURN"


def _py_default(param: ParamSpec) -> str:
    """Convert ParamSpec default to Python literal (self-contained, no L1 dep)."""
    val = str(param.default) if param.default is not None else "0"
    val = val.strip().rstrip(";")
    pt = param.param_type.value
    if pt in ("int", "double", "float", "long", "uint", "ulong"):
        try:
            float(val)
            return val
        except ValueError:
            return f"'{val}'"
    if pt == "bool":
        return "True" if val.lower() in ("true", "1") else "False"
    if pt == "string":
        if val.startswith('"') or val.startswith("'"):
            return val
        return f"'{val}'"
    return val
