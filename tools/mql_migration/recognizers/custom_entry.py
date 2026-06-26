"""自定义指标入场识别器。

识别模式:
  if (iCustom(..., buffer=0, shift=1) > 0 && other_conditions) {
      OrderSend(OP_BUYSTOP/SELLSTOP/OP_BUY/OP_SELL, ...)
  }

覆盖 custom_signal 等使用 iCustom 的 EA。
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
    find_calls,
    find_functions,
    ast_contains_call,
    find_ordersend_in_branch,
    ORDER_TYPE_MAP,
    str_to_order_action,
)


def recognize_custom_entries(ast: SourceFile,
                             expr_gen: ExpressionGen) -> List[EntryRule]:
    """识别所有 iCustom 驱动的入场规则。

    只捕获那些条件中包含 iCustom 调用的入场。已被市场入场识别器
    捕获的普通入场（条件不含 iCustom）不重复返回。

    Args:
        ast: 已解析的 MQL AST
        expr_gen: 表达式生成器

    Returns:
        iCustom 驱动的 EntryRule 列表
    """
    entries: List[EntryRule] = []

    for func in find_functions(ast, "OnTick", "OnCalculate"):
        if not func.body:
            continue

        local_vars = extract_local_vars(func)
        with expr_gen.local_scope(local_vars):
            # Find all iCustom calls in this function
            icustom_calls = find_calls(func.body, "iCustom")
            if not icustom_calls:
                continue

            # Scan if-statements whose condition references iCustom results
            _scan_icustom_entries(func.body, entries, expr_gen,
                                  func.name, icustom_calls)

    return entries


def _scan_icustom_entries(body, entries: List[EntryRule],
                          expr_gen: ExpressionGen,
                          func_name: str,
                          icustom_calls: List[CallExpr]) -> None:
    """扫描包含 iCustom 引用的 if 语句，匹配入场模式。"""
    if isinstance(body, CompoundStmt):
        stmts = body.statements or []
    elif isinstance(body, list):
        stmts = body
    else:
        return

    # Pass 1: build map of local variables that hold iCustom results
    # e.g. "main = iCustom(...)" → icustom_vars["main"] = the_custom_call
    icustom_var_map: dict[str, CallExpr] = {}
    for stmt in stmts:
        _collect_icustom_vars(stmt, icustom_var_map)

    # Pass 2: scan if-statements whose condition references icustom vars
    # Walk the if/else-if chain for each matching if-statement.
    icustom_var_names = set(icustom_var_map.keys())
    for stmt in stmts:
        if isinstance(stmt, IfStmt):
            # Walk the entire if/else-if chain
            current = stmt
            while current is not None:
                cond_refs = _ast_var_refs(current.condition)
                hits = cond_refs & icustom_var_names
                has_icustom_direct = ast_contains_call(current.condition, "iCustom")
                if has_icustom_direct or hits:
                    entry = _match_icustom_entry(current, expr_gen, func_name)
                    if entry:
                        entries.append(entry)
                # Advance to next else-if
                current = current.else_branch if isinstance(current.else_branch, IfStmt) else None

        if isinstance(stmt, CompoundStmt):
            _scan_icustom_entries(stmt, entries, expr_gen,
                                  func_name, icustom_calls)


def _collect_icustom_vars(stmt, var_map: dict) -> None:
    """收集持有 iCustom 返回值的局部变量名。"""
    from tools.mql_transpiler.ast_nodes import VarDecl, AssignmentExpr, ExpressionStmt

    if isinstance(stmt, VarDecl):
        # "double main = iCustom(...)" → main → call
        if stmt.value and isinstance(stmt.value, CallExpr) and stmt.value.name == "iCustom":
            var_map[stmt.name] = stmt.value
    elif isinstance(stmt, ExpressionStmt) and isinstance(stmt.expr, AssignmentExpr):
        # "main = iCustom(...)" (less common but valid)
        if isinstance(stmt.expr.rhs, CallExpr) and stmt.expr.rhs.name == "iCustom":
            var_map[stmt.expr.lhs] = stmt.expr.rhs


def _ast_var_refs(node) -> set[str]:
    """收集 AST 子树中所有 Identifier 引用名。"""
    from tools.mql_transpiler.ast_nodes import Identifier, CallExpr
    refs = set()

    def _walk(n):
        if n is None:
            return
        if isinstance(n, Identifier):
            refs.add(n.name)
        if isinstance(n, CallExpr):
            for a in (n.args or []):
                _walk(a)
        from tools.mql_transpiler.ast_nodes import BinaryOp, UnaryOp, IfStmt, CompoundStmt, ExpressionStmt
        if isinstance(n, BinaryOp):
            _walk(n.left)
            _walk(n.right)
        if isinstance(n, UnaryOp):
            _walk(n.operand)
        if isinstance(n, IfStmt):
            _walk(n.condition)
            _walk(n.then_branch)
            _walk(n.else_branch)
        if isinstance(n, CompoundStmt):
            for s in (n.statements or []):
                _walk(s)
        if isinstance(n, ExpressionStmt):
            _walk(n.expr)

    _walk(node)
    return refs


def _match_icustom_entry(if_stmt: IfStmt,
                         expr_gen: ExpressionGen,
                         func_name: str) -> Optional[EntryRule]:
    """匹配 iCustom if → OrderSend 入场模式。"""
    # Find OrderSend in the then-branch
    order_send = find_ordersend_in_branch(if_stmt.then_branch)
    if order_send is None:
        return None

    type_str = _extract_order_type_str(order_send)
    if type_str is None:
        return None
    order_type = str_to_order_action(type_str)

    cond_expr = expr_gen.translate(if_stmt.condition)

    # Extract custom indicator config from iCustom calls in condition
    icustom_config = _extract_icustom_config(if_stmt, expr_gen)

    params = OrderParams(**_extract_order_params_base(order_send, expr_gen))

    # Add custom indicator metadata to the comment
    if icustom_config:
        params.comment = icustom_config.get("name", params.comment)

    return EntryRule(
        conditions=[Condition(
            expr=cond_expr,
            comment=f"iCustom: {icustom_config.get('name', '?')}" if icustom_config else ""
        )],
        action=order_type,
        order_params=params,
        source_location=f"{func_name}:icustom_entry",
    )


def _extract_icustom_config(if_stmt: IfStmt,
                            expr_gen: ExpressionGen) -> dict | None:
    """从 if 条件中提取 iCustom 调用配置。"""
    icustom_calls = find_calls(if_stmt.condition, "iCustom")
    if not icustom_calls:
        return None

    call = icustom_calls[0]  # Take the first iCustom call
    args = call.args or []

    if len(args) < 3:
        return None

    name_arg = expr_gen.translate(args[2]) if len(args) > 2 else "?"

    config = {
        "name": name_arg,
        "buffer": expr_gen.translate(args[-2]) if len(args) >= 5 else "0",
        "shift": expr_gen.translate(args[-1]) if len(args) >= 5 else "1",
    }

    # Extract custom params (middle args between name and last 3)
    if len(args) > 5:
        param_vals = [expr_gen.translate(a) for a in args[3:-2]]
        config["params"] = param_vals

    return config


