"""挂单入场识别器。

识别模式:
  1. for (i=1; i<=N; i++) { OrderSend(OP_BUYLIMIT/SELLLIMIT/...) } — 网格挂单
  2. if (cond) { OrderSend(OP_BUYSTOP/SELLSTOP/...) } — 突破单
"""

from __future__ import annotations

from typing import List, Optional

from tools.mql_transpiler.ast_nodes import (
    CallExpr,
    CompoundStmt,
    ExpressionStmt,
    ForStmt,
    FuncDef,
    IfStmt,
    SourceFile,
)

from tools.mql_migration.expression_gen import ExpressionGen
from tools.mql_migration.intent_ir import (
    Condition,
    EntryRule,
    OrderAction,
    OrderParams,
)
from tools.mql_migration.recognizers.base import (
    extract_local_vars,
    extract_order_params as _extract_order_params_base,
    extract_order_type as _extract_order_type_str,
    find_functions,
    find_ordersend_in_branch,
    ORDER_TYPE_MAP,
    str_to_order_action,
)

# Pending order types (not market) — MQL names
_PENDING_TYPES = {"OP_BUYLIMIT", "OP_SELLLIMIT", "OP_BUYSTOP", "OP_SELLSTOP"}


def recognize_pending_entries(ast: SourceFile,
                             expr_gen: ExpressionGen) -> List[EntryRule]:
    """从 MQL AST 中识别所有挂单入场规则。

    Covers:
      - for-loop grid orders (BUY_LIMIT / SELL_LIMIT)
      - breakout orders (BUY_STOP / SELL_STOP)

    Args:
        ast: 已解析的 MQL AST
        expr_gen: 表达式生成器

    Returns:
        识别到的 EntryRule 列表
    """
    entries: List[EntryRule] = []

    for func in find_functions(ast):
        if not func.body:
            continue
        local_vars = extract_local_vars(func)
        with expr_gen.local_scope(local_vars):
            _scan_function_body(func.body, entries, expr_gen, func.name)

    return entries


def _scan_function_body(body, entries: List[EntryRule],
                        expr_gen: ExpressionGen, func_name: str) -> None:
    """Recursively scan function body for pending order entries."""
    if isinstance(body, CompoundStmt):
        stmts = body.statements or []
    elif isinstance(body, list):
        stmts = body
    else:
        return

    for stmt in stmts:
        # Pattern 1: for-loop with pending orders
        if isinstance(stmt, ForStmt):
            pending_entries = _match_for_pending(stmt, expr_gen, func_name)
            entries.extend(pending_entries)

        # Pattern 2: if-statement with breakout orders
        if isinstance(stmt, IfStmt):
            entry = _match_if_pending(stmt, expr_gen, func_name)
            if entry:
                entries.append(entry)
            # Recurse into else branch
            if stmt.else_branch:
                _scan_function_body(stmt.else_branch, entries, expr_gen, func_name)

        # Recurse into nested compounds
        if isinstance(stmt, CompoundStmt):
            _scan_function_body(stmt, entries, expr_gen, func_name)


def _match_for_pending(for_stmt: ForStmt,
                       expr_gen: ExpressionGen,
                       func_name: str) -> List[EntryRule]:
    """匹配 for 循环中的挂单入场。

    模式: for (i=1; i<=N; i++) {
              OrderSend(OP_BUYLIMIT, ...)
              OrderSend(OP_SELLLIMIT, ...)
          }
    """
    entries = []
    # Look for OrderSend calls in the for-loop body
    if isinstance(for_stmt.body, CompoundStmt):
        for stmt in (for_stmt.body.statements or []):
            if isinstance(stmt, ExpressionStmt) and isinstance(stmt.expr, CallExpr):
                if stmt.expr.name == "OrderSend":
                    entry = _build_pending_entry(stmt.expr, None, expr_gen, func_name)
                    if entry:
                        entries.append(entry)
    elif isinstance(for_stmt.body, IfStmt):
        # for-body is an if with OrderSend inside
        entry = _match_if_pending(for_stmt.body, expr_gen, func_name)
        if entry:
            entries.append(entry)

    return entries


def _match_if_pending(if_stmt: IfStmt,
                      expr_gen: ExpressionGen,
                      func_name: str) -> Optional[EntryRule]:
    """匹配 if 语句中的挂单入场。"""
    order_send = find_ordersend_in_branch(if_stmt.then_branch)
    if order_send is None:
        return None

    order_type = _extract_pending_type(order_send)
    if order_type is None:
        return None

    cond_expr = expr_gen.translate(if_stmt.condition)

    return _build_pending_entry(order_send, cond_expr, expr_gen, func_name)


def _extract_pending_type(call: CallExpr) -> Optional[OrderAction]:
    """Check if OrderSend is a pending order type (not market)."""
    from tools.mql_migration.recognizers.base import get_arg
    type_arg = get_arg(call, 1)
    if type_arg is None:
        return None
    mql_name = getattr(type_arg, 'name', None)
    if mql_name and mql_name in _PENDING_TYPES:
        action_str = ORDER_TYPE_MAP.get(mql_name)
        if action_str:
            return str_to_order_action(action_str, default=OrderAction.BUY_LIMIT)
    return None


def _build_pending_entry(call: CallExpr, cond_expr: Optional[str],
                         expr_gen: ExpressionGen, func_name: str) -> EntryRule:
    """Build EntryRule from a pending OrderSend call (uses shared param extractor)."""
    order_type = _extract_pending_type(call) or OrderAction.BUY_LIMIT
    raw_params = _extract_order_params_base(call, expr_gen)

    return EntryRule(
        conditions=[Condition(expr=cond_expr)] if cond_expr else [],
        action=order_type,
        order_params=OrderParams(**raw_params),
        source_location=f"{func_name}:pending",
    )
