"""市价单入场识别器。

识别模式: if (条件) { OrderSend(OP_BUY/SELL, volume, price, ...) }
"""

from __future__ import annotations

from typing import List, Optional

from tools.mql_transpiler.ast_nodes import (
    CallExpr,
    CompoundStmt,
    ExpressionStmt,
    FuncDef,
    IfStmt,
    NumberLiteral,
    SourceFile,
    StringLiteral,
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
)


def _order_action_from_str(name: str) -> OrderAction:
    """Convert MQL order type name to OrderAction enum."""
    action_map = {
        "market_buy": OrderAction.MARKET_BUY,
        "market_sell": OrderAction.MARKET_SELL,
        "buy_limit": OrderAction.BUY_LIMIT,
        "sell_limit": OrderAction.SELL_LIMIT,
        "buy_stop": OrderAction.BUY_STOP,
        "sell_stop": OrderAction.SELL_STOP,
    }
    return action_map.get(name, OrderAction.MARKET_BUY)


def recognize_market_entries(ast: SourceFile,
                             expr_gen: ExpressionGen) -> List[EntryRule]:
    """从 MQL AST 中识别所有市价单入场规则。

    Args:
        ast: 已解析的 MQL AST
        expr_gen: 表达式生成器（用于将条件表达式转为 Python 代码）

    Returns:
        识别到的 EntryRule 列表
    """
    entries: List[EntryRule] = []

    for func in find_functions(ast):
        if not func.body:
            continue
        local_vars = extract_local_vars(func)
        with expr_gen.local_scope(local_vars):
            _scan_statements(func.body, entries, expr_gen, func.name)

    return entries


def _scan_statements(body, entries: List[EntryRule],
                     expr_gen: ExpressionGen, func_name: str) -> None:
    """递归扫描语句树，匹配 if/else-if → OrderSend 模式。"""
    if isinstance(body, CompoundStmt):
        stmts = body.statements or []
    elif isinstance(body, list):
        stmts = body
    else:
        return

    for stmt in stmts:
        if isinstance(stmt, IfStmt):
            # Walk the entire if/else-if chain.
            current = stmt
            while current is not None:
                entry = _match_if_ordersend(current, expr_gen, func_name)
                if entry is not None:
                    entries.append(entry)
                current = current.else_branch if isinstance(current.else_branch, IfStmt) else None
                if not isinstance(current, IfStmt):
                    # Non-IfStmt else branch (final else)
                    if current is not None:
                        _scan_statements(getattr(stmt, 'else_branch', None), entries, expr_gen, func_name)
                    break

        # 递归进入 CompoundStmt
        if isinstance(stmt, CompoundStmt):
            _scan_statements(stmt, entries, expr_gen, func_name)
        elif isinstance(stmt, IfStmt) and stmt.else_branch and not isinstance(stmt.else_branch, IfStmt):
            _scan_statements(stmt.else_branch, entries, expr_gen, func_name)


def _match_if_ordersend(if_stmt: IfStmt,
                        expr_gen: ExpressionGen,
                        func_name: str) -> Optional[EntryRule]:
    """检查 if 语句是否为 'if (条件) { OrderSend(...) }' 入场模式。"""
    order_send = find_ordersend_in_branch(if_stmt.then_branch)
    if order_send is None:
        return None

    order_type = _extract_order_type_str(order_send)
    if order_type is None:
        return None
    order_type = _order_action_from_str(order_type)
    if order_type is None:
        return None

    # Store AST node in Condition (language-agnostic IR).
    # Expression string is a pre-computed Python cache.
    cond_expr = expr_gen.translate(if_stmt.condition)
    raw_params = _extract_order_params_base(order_send, expr_gen)

    return EntryRule(
        conditions=[Condition(
            ast_node=if_stmt.condition,
            expr=cond_expr,
        )],
        action=order_type,
        order_params=OrderParams(**raw_params),
        source_location=f"{func_name}:if_stmt",
    )
