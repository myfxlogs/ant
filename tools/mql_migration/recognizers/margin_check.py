"""保证金 / 风控检查识别器。

识别模式:
  1. OnTimer: AccountFreeMargin < AccountBalance * ratio → closeByMagic
  2. OnTimer: AccountFreeMargin < AccountEquity * ratio → closeAll
  3. 任何定时器驱动的风控检查
"""

from __future__ import annotations

from typing import List, Optional

from tools.mql_transpiler.ast_nodes import (
    BinaryOp, CallExpr, CompoundStmt, ExpressionStmt,
    ForStmt, IfStmt, SourceFile,
)
from tools.mql_migration.recognizers.base import (
    ast_contains_call,
    extract_local_vars,
    find_calls,
    find_functions,
)

from tools.mql_migration.expression_gen import ExpressionGen
from tools.mql_migration.intent_ir import RiskCheck


def recognize_margin_checks(ast: SourceFile,
                            expr_gen: ExpressionGen) -> List[RiskCheck]:
    """识别所有保证金/风控检查规则。

    扫描 OnTimer 函数，查找 AccountFreeMargin / AccountBalance 等调用，
    提取风控条件和动作。

    Args:
        ast: 已解析的 MQL AST
        expr_gen: 表达式生成器

    Returns:
        识别的 RiskCheck 列表
    """
    checks: List[RiskCheck] = []

    for func in find_functions(ast, "OnTimer"):
        if not func.body:
            continue

        local_vars = extract_local_vars(func)
        with expr_gen.local_scope(local_vars):
            _scan_for_margin_checks(func.body, checks, expr_gen)

    return checks


def _scan_for_margin_checks(body, checks: List[RiskCheck],
                            expr_gen: ExpressionGen) -> None:
    """扫描语句树，查找 margin 检查 if 语句。"""
    if isinstance(body, CompoundStmt):
        stmts = body.statements or []
    elif isinstance(body, list):
        stmts = body
    else:
        return

    for stmt in stmts:
        if isinstance(stmt, IfStmt):
            check = _match_margin_check(stmt, expr_gen)
            if check:
                checks.append(check)

        if isinstance(stmt, CompoundStmt):
            _scan_for_margin_checks(stmt, checks, expr_gen)


def _match_margin_check(if_stmt: IfStmt,
                        expr_gen: ExpressionGen) -> Optional[RiskCheck]:
    """匹配保证金检查 if 语句。

    模式:
      if (AccountFreeMargin() < AccountBalance() * 0.5) {
          closeAllByMagic(MagicNumber);
      }
    """
    # Check condition for margin-related calls
    has_margin = (ast_contains_call(if_stmt.condition, "AccountFreeMargin") or
                  ast_contains_call(if_stmt.condition, "AccountMargin"))
    if not has_margin:
        return None

    # Generate the condition expression
    cond_expr = expr_gen.translate(if_stmt.condition)

    # Determine action from then-branch
    action = _detect_action(if_stmt.then_branch)

    return RiskCheck(
        kind="margin_check",
        condition=cond_expr,
        action=action,
        trigger="on_timer",
    )


def _detect_action(branch) -> str:
    """从 then 分支检测风控动作类型。"""
    if branch is None:
        return "close_all"

    # Check for closeAllByMagic → close by magic
    close_calls = find_calls(branch, "closeAllByMagic", "closeByMagic")
    if close_calls:
        return "close_by_magic"

    # Check for OrderClose → close positions
    order_close = find_calls(branch, "OrderClose")
    if order_close:
        return "close_all"

    # Check for explicit close loop (for → OrderClose)
    if isinstance(branch, CompoundStmt):
        for stmt in (branch.statements or []):
            if isinstance(stmt, ForStmt):
                inner_close = find_calls(stmt.body, "OrderClose")
                if inner_close:
                    return "close_all"

    return "close_all"


