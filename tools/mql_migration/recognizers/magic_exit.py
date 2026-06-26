"""Magic-based exit recognizers.

识别模式:
  1. OrderClose in for-loop with OrderMagicNumber check → magic_close
  2. OrderDelete in for-loop with magic range check → magic_delete
"""

from __future__ import annotations

from typing import List, Optional

from tools.mql_transpiler.ast_nodes import (
    BinaryOp,
    CallExpr,
    CompoundStmt,
    ExpressionStmt,
    ForStmt,
    FuncDef,
    Identifier,
    IfStmt,
    NumberLiteral,
    SourceFile,
)

from tools.mql_migration.expression_gen import ExpressionGen
from tools.mql_migration.intent_ir import (
    CloseAction,
    Condition,
    ExitRule,
    ExitTrigger,
    MagicFilter,
)
from tools.mql_migration.recognizers.base import (
    extract_local_vars,
    find_calls,
    find_functions,
)


def recognize_exit_rules(ast: SourceFile,
                         expr_gen: ExpressionGen) -> List[ExitRule]:
    """从 MQL AST 识别所有出场规则。

    Scrubs all functions (OnTick, OnDeinit, and user helpers like closeByMagic)
    for OrderClose/OrderDelete loops with magic filtering.

    Args:
        ast: 已解析的 MQL AST
        expr_gen: 表达式生成器

    Returns:
        识别到的 ExitRule 列表
    """
    exits: List[ExitRule] = []

    for func in find_functions(ast):
        if not func.body:
            continue
        local_vars = extract_local_vars(func)
        with expr_gen.local_scope(local_vars):
            _scan_for_exits(func.body, exits, expr_gen, func.name)

    return exits


def _scan_for_exits(body, exits: List[ExitRule],
                    expr_gen: ExpressionGen, func_name: str) -> None:
    """Recursively scan for OrderClose/OrderDelete loop patterns."""
    if isinstance(body, CompoundStmt):
        stmts = body.statements or []
    elif isinstance(body, list):
        stmts = body
    else:
        return

    for stmt in stmts:
        if isinstance(stmt, ForStmt):
            # Check if for-loop body has OrderClose or OrderDelete
            close_calls = find_calls(stmt.body, "OrderClose")
            delete_calls = find_calls(stmt.body, "OrderDelete")

            if close_calls:
                exit_rule = _extract_close_rule(stmt, expr_gen, func_name)
                if exit_rule:
                    exits.append(exit_rule)

            if delete_calls:
                exit_rule = _extract_delete_rule(stmt, expr_gen, func_name)
                if exit_rule:
                    exits.append(exit_rule)

        if isinstance(stmt, IfStmt):
            # Direct OrderClose in if-body (reverse signal exit)
            close_calls = find_calls(stmt.then_branch, "OrderClose")
            if close_calls:
                cond_expr = expr_gen.translate(stmt.condition)
                exits.append(ExitRule(
                    trigger=ExitTrigger.REVERSE_SIGNAL,
                    conditions=[Condition(expr=cond_expr)],
                    action=CloseAction.POSITION_CLOSE,
                    target=MagicFilter(kind="all"),
                    source_location=f"{func_name}:if_close",
                ))

        if isinstance(stmt, CompoundStmt):
            _scan_for_exits(stmt, exits, expr_gen, func_name)


def _extract_close_rule(for_stmt: ForStmt,
                        expr_gen: ExpressionGen,
                        func_name: str) -> Optional[ExitRule]:
    """Extract magic_close ExitRule from an OrderClose for-loop.

    Pattern: for (i=0; i<OrdersTotal(); i++) {
                 if (OrderSelect(...)) {
                     if (OrderMagicNumber() == MAGIC) {
                         OrderClose(OrderTicket(), OrderLots(), ...)
                     }
                 }
             }
    """
    # Find the magic comparison inside the loop
    if_chain = _find_nested_if(for_stmt.body)
    magic_filter = _detect_magic_filter(if_chain, expr_gen)

    return ExitRule(
        trigger=ExitTrigger.MAGIC_CLOSE,
        action=CloseAction.POSITION_CLOSE,
        target=magic_filter or MagicFilter(kind="all"),
        source_location=f"{func_name}:magic_close",
    )


def _extract_delete_rule(for_stmt: ForStmt,
                         expr_gen: ExpressionGen,
                         func_name: str) -> Optional[ExitRule]:
    """Extract magic_delete ExitRule from an OrderDelete for-loop.

    Pattern: for (i=0; i<OrdersTotal(); i++) {
                 if (OrderSelect(...)) {
                     int magic = OrderMagicNumber();
                     if (magic >= BASE && magic <= BASE+200) {
                         OrderDelete(OrderTicket());
                     }
                 }
             }
    """
    if_chain = _find_nested_if(for_stmt.body)
    magic_filter = _detect_magic_range(if_chain, expr_gen)

    return ExitRule(
        trigger=ExitTrigger.MAGIC_DELETE,
        action=CloseAction.ORDER_DELETE,
        target=magic_filter or MagicFilter(kind="all"),
        source_location=f"{func_name}:magic_delete",
    )


def _find_nested_if(body) -> Optional[IfStmt]:
    """Find the innermost IfStmt in a for-loop body (skipping OrderSelect wrapper)."""
    if isinstance(body, CompoundStmt):
        for stmt in (body.statements or []):
            if isinstance(stmt, IfStmt):
                # If this is an OrderSelect check, go deeper
                if _is_orderselect_check(stmt):
                    return _find_nested_if(stmt.then_branch)
                return stmt
    if isinstance(body, IfStmt):
        if _is_orderselect_check(body):
            return _find_nested_if(body.then_branch)
        return body
    return None


def _is_orderselect_check(stmt: IfStmt) -> bool:
    """Check if the if-condition is an OrderSelect(...) call."""
    if isinstance(stmt.condition, CallExpr):
        return stmt.condition.name == "OrderSelect"
    return False


def _detect_magic_filter(if_stmt: Optional[IfStmt],
                         expr_gen: ExpressionGen) -> Optional[MagicFilter]:
    """Detect magic filter from a magic comparison if-statement.

    Pattern: if (OrderMagicNumber() == MAGIC) { ... }
    """
    if if_stmt is None:
        return None

    cond = if_stmt.condition
    if isinstance(cond, BinaryOp) and cond.op == "==":
        # Left is OrderMagicNumber(), right is the magic value
        if isinstance(cond.left, CallExpr) and cond.left.name == "OrderMagicNumber":
            magic_val = expr_gen.translate(cond.right)
            return MagicFilter(kind="exact", magic_value=magic_val)

    return None


def _detect_magic_range(if_stmt: Optional[IfStmt],
                        expr_gen: ExpressionGen) -> Optional[MagicFilter]:
    """Detect magic range filter from a range comparison.

    Pattern: if (magic >= BASE && magic <= BASE+200) { ... }
    """
    if if_stmt is None:
        return None

    cond = if_stmt.condition
    if isinstance(cond, BinaryOp) and cond.op == "&&":
        left = cond.left
        right = cond.right
        if (isinstance(left, BinaryOp) and left.op == ">=" and
            isinstance(right, BinaryOp) and right.op == "<="):
            magic_min = expr_gen.translate(left.right)
            magic_max = expr_gen.translate(right.right)
            return MagicFilter(kind="range",
                             magic_min=magic_min,
                             magic_max=magic_max)

    return None
