"""执行模型 + 手数规则 + 参数/状态提取识别器。"""

from __future__ import annotations

from typing import List

from tools.mql_transpiler.ast_nodes import (
    AssignmentExpr,
    CallExpr,
    ClassDef,
    CompoundStmt,
    ExpressionStmt,
    ForStmt,
    FuncDef,
    Identifier,
    IfStmt,
    NumberLiteral,
    SourceFile,
    VarDecl,
)

from tools.mql_migration.intent_ir import (
    ExecutionKind,
    ExecutionModel,
    ParamGroup,
    ParamRange,
    ParamSpec,
    ParamType,
    SizingKind,
    SizingRule,
    StateVar,
    StrategyMeta,
    TimerRule,
)
from tools.mql_migration.recognizers.base import (
    extract_local_vars,
    find_calls,
    find_functions,
)


def recognize_meta(ast: SourceFile, source_name: str = "") -> StrategyMeta:
    """提取策略元信息。

    Args:
        ast: 已解析的 MQL AST
        source_name: 源文件名（如 "simple_ma_cross.mq4"），用于推断类名
    """
    from tools.mql_migration.intent_ir import MQLVersion, StrategyMeta

    # Detect MQL5 class-based
    has_mql5 = any(isinstance(d, ClassDef) for d in (ast.declarations or []))
    version = MQLVersion.MQL5 if has_mql5 else MQLVersion.MQL4

    # Infer class name from filename
    name = ""
    if source_name:
        base = source_name.rsplit(".", 1)[0]  # strip .mq4/.mq5
        name = base.replace("_", " ").title().replace(" ", "")

    return StrategyMeta(name=name, mql_version=version)


def recognize_params(ast: SourceFile) -> List[ParamSpec]:
    """提取 extern/input 参数声明。"""
    params: List[ParamSpec] = []

    for decl in (ast.declarations or []):
        if isinstance(decl, VarDecl) and (decl.is_extern or decl.is_input):
            param_type = _map_mql_type(decl.var_type)
            default = _extract_default_value(decl)

            # Guess parameter group from name
            group = _guess_param_group(decl.name)

            params.append(ParamSpec(
                name=decl.name,
                label=decl.name,  # Can be enhanced with i18n
                param_type=param_type,
                default=default,
                group=group,
            ))

    return params


def recognize_state_vars(ast: SourceFile) -> List[StateVar]:
    """提取全局状态变量（非 extern 的 global 声明）。"""
    state: List[StateVar] = []

    for decl in (ast.declarations or []):
        if isinstance(decl, VarDecl) and not decl.is_extern and not decl.is_input:
            state.append(StateVar(
                name=decl.name,
                var_type=decl.var_type,
                initial_value=_extract_default_value(decl),
            ))

    return state


def recognize_execution_model(ast: SourceFile) -> ExecutionModel:
    """推断执行模型。

    Heuristics:
      - OnTick + OrderSend(OP_BUYLIMIT/SELLLIMIT) in a for-loop + gridPlaced flag
        → ON_INIT_GRID
      - OnTick + OrderSend(OP_BUY/SELL) + no grid flag
        → ON_BAR (SDK惯用模式推荐)
      - Otherwise → ON_TICK
    """
    # Check for grid-initialization pattern: bool gridPlaced = false + OnTick checks it
    has_grid_flag = False
    has_pending_in_for = False

    for func in find_functions(ast):
        if func.name == "OnTick" and func.body:
            # Check for grid flag check at top of OnTick
            if isinstance(func.body, CompoundStmt):
                for stmt in (func.body.statements or []):
                    if isinstance(stmt, IfStmt):
                        cond = stmt.condition
                        if isinstance(cond, Identifier) and cond.name in ("gridPlaced", "_grid_placed"):
                            has_grid_flag = True
            # Check for pending orders in for-loop
            if isinstance(func.body, CompoundStmt):
                for stmt in (func.body.statements or []):
                    if isinstance(stmt, ForStmt):
                        pending = find_calls(stmt.body, "OrderSend")
                        if pending:
                            has_pending_in_for = True

    if has_grid_flag:
        return ExecutionModel(kind=ExecutionKind.ON_INIT_GRID)
    elif has_pending_in_for:
        return ExecutionModel(kind=ExecutionKind.ON_INIT_GRID)
    else:
        # Market order EAs → recommend on_bar
        has_market_orders = False
        for func in find_functions(ast):
            for call in find_calls(func.body if func.body else [], "OrderSend"):
                type_arg = _get_order_type(call)
                if type_arg in ("OP_BUY", "OP_SELL"):
                    has_market_orders = True

        if has_market_orders:
            return ExecutionModel(
                kind=ExecutionKind.ON_BAR,
                require_account_check=False,
            )
        return ExecutionModel(kind=ExecutionKind.ON_TICK)


def recognize_sizing(ast: SourceFile) -> SizingRule:
    """推断手数规则。

    Heuristics:
      - 固定值 extern double LotSize → FIXED
      - BaseLot + currentLot * 2 pattern → MARTINGALE
    """
    # Check for martingale pattern: currentLot = currentLot * 2
    for func in find_functions(ast):
        for stmt in (func.body.statements if isinstance(func.body, CompoundStmt) else []):
            if isinstance(stmt, ExpressionStmt) and isinstance(stmt.expr, AssignmentExpr):
                rhs = stmt.expr.rhs
                if _is_multiplication(rhs, stmt.expr.lhs, 2):
                    return SizingRule(
                        kind=SizingKind.MARTINGALE,
                        expression="self.base_lot",
                    )

    # Default: fixed lot size from extern param
    lot_param = None
    for decl in (ast.declarations or []):
        if isinstance(decl, VarDecl) and decl.is_extern:
            if "lot" in decl.name.lower():
                lot_param = decl.name

    if lot_param:
        return SizingRule(
            kind=SizingKind.FIXED,
            expression=f"self.ctx.param('{lot_param}', 0.10)",
        )

    # No lot param found — generate a configurable param with safe default
    return SizingRule(
        kind=SizingKind.FIXED,
        expression="Decimal(str(self.ctx.param('LotSize', 0.10)))",
    )


def recognize_timer(ast: SourceFile) -> TimerRule | None:
    """提取定时器规则。"""
    for func in find_functions(ast):
        if func.body:
            for call in find_calls(func.body, "EventSetTimer"):
                if call.args and len(call.args) > 0:
                    interval = _extract_number(call.args[0])
                    return TimerRule(interval_seconds=interval or 300)
    return None


# ── Helpers ─────────────────────────────────────────────────────


def _map_mql_type(mql_type: str) -> ParamType:
    return {
        "int": ParamType.INT, "long": ParamType.INT,
        "uint": ParamType.INT, "ulong": ParamType.INT,
        "double": ParamType.DOUBLE, "float": ParamType.DOUBLE,
        "string": ParamType.STRING,
        "bool": ParamType.BOOL,
    }.get(mql_type, ParamType.DOUBLE)


def _extract_default_value(decl: VarDecl) -> str:
    if decl.value is None:
        if decl.var_type == "bool":
            return "False"
        if decl.var_type in ("int", "long", "uint", "ulong"):
            return "0"
        return "0.0"
    if isinstance(decl.value, NumberLiteral):
        return decl.value.value
    if isinstance(decl.value, Identifier):
        from tools.mql_transpiler.ast_transpiler import _MQL_CONSTANTS
        return _MQL_CONSTANTS.get(decl.value.name, decl.value.name)
    return str(getattr(decl.value, 'value', '0'))


def _guess_param_group(name: str) -> ParamGroup:
    lower = name.lower()
    if any(k in lower for k in ("ma", "rsi", "macd", "period", "cci", "atr", "band")):
        return ParamGroup.ENTRY
    if any(k in lower for k in ("tp", "sl", "stoploss", "takeprofit", "trail")):
        return ParamGroup.EXIT
    if any(k in lower for k in ("lot", "volume", "size")):
        return ParamGroup.SIZING
    if any(k in lower for k in ("risk", "margin", "max", "limit")):
        return ParamGroup.RISK
    if any(k in lower for k in ("magic", "comment", "deviation")):
        return ParamGroup.SYSTEM
    return ParamGroup.SYSTEM


def _extract_number(node) -> int | None:
    if isinstance(node, NumberLiteral):
        try:
            return int(node.value)
        except ValueError:
            return None
    return None


def _get_order_type(call: CallExpr) -> str | None:
    if call.args and len(call.args) > 1:
        arg = call.args[1]
        if isinstance(arg, Identifier):
            return arg.name
    return None


def _is_multiplication(node, var_name: str, factor: int) -> bool:
    """Check if node is 'var_name * factor' or 'factor * var_name'."""
    from tools.mql_transpiler.ast_nodes import BinaryOp
    if isinstance(node, BinaryOp) and node.op == "*":
        if isinstance(node.left, Identifier) and node.left.name == var_name:
            if isinstance(node.right, NumberLiteral):
                return int(float(node.right.value)) == factor
        if isinstance(node.right, Identifier) and node.right.name == var_name:
            if isinstance(node.left, NumberLiteral):
                return int(float(node.left.value)) == factor
    return False
