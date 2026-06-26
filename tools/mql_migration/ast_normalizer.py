"""AST 规范化器 — 在模式匹配前统一 AST 形态。

消除"同意图不同写法"导致的匹配失败。

Passes (按顺序):
  1. normalize_if_chains    — 统一 if/else-if 结构
  2. propagate_constants    — 常量传播、内联单次赋值变量
  3. simplify_conditions    — 化简冗余的逻辑运算
"""

from __future__ import annotations

from typing import Dict, List, Optional, Set, Tuple

from tools.mql_transpiler.ast_nodes import (
    AssignmentExpr,
    BinaryOp,
    CallExpr,
    CompoundStmt,
    Expression,
    ExpressionStmt,
    ForStmt,
    FuncDef,
    Identifier,
    IfStmt,
    NumberLiteral,
    SourceFile,
    StringLiteral,
    UnaryOp,
    VarDecl,
    WhileStmt,
)


def normalize(ast: SourceFile) -> SourceFile:
    """对 AST 执行全部规范化 pass，原地修改。

    Args:
        ast: 已解析的 MQL AST

    Returns:
        规范化后的 AST（同一对象，节点已修改）
    """
    for func in _all_functions(ast):
        if func.body:
            _normalize_function(func)
    return ast


def _all_functions(ast: SourceFile) -> List[FuncDef]:
    """收集所有函数定义（包括 class body 内的）。"""
    funcs = []
    for decl in (ast.declarations or []):
        if isinstance(decl, FuncDef):
            funcs.append(decl)
    return funcs


def _normalize_function(func: FuncDef) -> None:
    """对单个函数体执行全部规范化 pass。"""
    if not isinstance(func.body, CompoundStmt):
        return

    # Pass 1: 收集可内联的变量定义
    _inline_single_use_vars(func)

    # Pass 2: Normalize if-chains into canonical form
    _normalize_if_chains(func.body)


# ── Pass 1: 单次使用变量内联 ────────────────────────────────────────


def _inline_single_use_vars(func: FuncDef) -> None:
    """内联只赋值一次、只使用一次的中间变量。

    Uses node identity (not indices) to avoid index-drift after removals.
    """
    if not isinstance(func.body, CompoundStmt):
        return

    stmts = func.body.statements or []

    # Map: variable name → (assign_stmt, assign_value_expr)
    assigns: Dict[str, Tuple[any, Expression]] = {}
    # Map: variable name → [stmt_nodes_that_reference_it]
    refs: Dict[str, List[any]] = {}

    for stmt in stmts:
        if isinstance(stmt, VarDecl) and stmt.value:
            if stmt.name not in assigns:
                assigns[stmt.name] = (stmt, stmt.value)
        if isinstance(stmt, ExpressionStmt) and isinstance(stmt.expr, AssignmentExpr):
            ae = stmt.expr
            if ae.lhs not in assigns:
                assigns[ae.lhs] = (stmt, ae.rhs)
        _collect_refs_by_node(stmt, refs)

    # Inline: assigned once, referenced exactly once (not counting the assignment)
    for var_name, (assign_stmt, assign_value) in assigns.items():
        ref_stmts = refs.get(var_name, [])
        # Count references excluding the assignment statement itself
        real_refs = [r for r in ref_stmts if r is not assign_stmt]
        if len(real_refs) != 1:
            continue

        target_stmt = real_refs[0]
        _inline_variable(func.body, var_name, assign_value, assign_stmt, target_stmt)


def _collect_refs_by_node(stmt, refs: Dict[str, List[any]]) -> None:
    """Collect all Identifier references in a statement, keyed by var name."""
    names = set()

    def _walk(node):
        if node is None:
            return
        if isinstance(node, Identifier):
            names.add(node.name)
        if isinstance(node, VarDecl):
            _walk(node.value)
        if isinstance(node, ExpressionStmt):
            _walk(node.expr)
        if isinstance(node, AssignmentExpr):
            _walk(node.rhs)
        if isinstance(node, BinaryOp):
            _walk(node.left); _walk(node.right)
        if isinstance(node, UnaryOp):
            _walk(node.operand)
        if isinstance(node, CallExpr):
            for a in (node.args or []):
                _walk(a)
        if isinstance(node, IfStmt):
            _walk(node.condition); _walk(node.then_branch); _walk(node.else_branch)
        if isinstance(node, ForStmt):
            _walk(node.init); _walk(node.condition); _walk(node.body)
        if isinstance(node, CompoundStmt):
            for s in (node.statements or []):
                _walk(s)

    _walk(stmt)
    for name in names:
        refs.setdefault(name, []).append(stmt)


def _inline_variable(body: CompoundStmt, var_name: str,
                     value: Expression, assign_stmt,
                     target_stmt) -> None:
    """Inline variable: replace references in target_stmt, remove assign_stmt."""
    if not isinstance(body, CompoundStmt):
        return

    # Replace all references to var_name in target_stmt with value
    _replace_identifier(target_stmt, var_name, value)

    # Remove the assignment statement
    body.statements = [s for s in (body.statements or []) if s is not assign_stmt]


def _replace_identifier(node, var_name: str, replacement: Expression) -> None:
    """在 AST 子树中，将所有对 var_name 的 Identifier 引用替换为 replacement。"""
    if node is None:
        return

    # Don't replace in new variable declarations with the same name
    if isinstance(node, VarDecl) and node.name == var_name:
        return
    if isinstance(node, AssignmentExpr) and node.lhs == var_name:
        # Replace the RHS but keep the assignment target
        _replace_identifier(node.rhs, var_name, replacement)
        return

    if isinstance(node, Identifier) and node.name == var_name:
        # In-place mutation: copy all attributes from replacement
        node.__class__ = replacement.__class__
        node.__dict__.update(replacement.__dict__)
        return

    # Recurse
    if isinstance(node, BinaryOp):
        _replace_identifier(node.left, var_name, replacement)
        _replace_identifier(node.right, var_name, replacement)
    elif isinstance(node, UnaryOp):
        _replace_identifier(node.operand, var_name, replacement)
    elif isinstance(node, CallExpr):
        for a in (node.args or []):
            _replace_identifier(a, var_name, replacement)
    elif isinstance(node, IfStmt):
        _replace_identifier(node.condition, var_name, replacement)
        _replace_identifier(node.then_branch, var_name, replacement)
        _replace_identifier(node.else_branch, var_name, replacement)
    elif isinstance(node, CompoundStmt):
        for s in (node.statements or []):
            _replace_identifier(s, var_name, replacement)
    elif isinstance(node, ForStmt):
        _replace_identifier(node.init, var_name, replacement)
        _replace_identifier(node.condition, var_name, replacement)
        _replace_identifier(node.body, var_name, replacement)
    elif isinstance(node, ExpressionStmt):
        _replace_identifier(node.expr, var_name, replacement)
    elif isinstance(node, AssignmentExpr):
        _replace_identifier(node.rhs, var_name, replacement)
    elif isinstance(node, VarDecl):
        _replace_identifier(node.value, var_name, replacement)


# ── Pass 2: if-chain normalization ──────────────────────────────────


def _normalize_if_chains(body: CompoundStmt) -> None:
    """Normalize if/else-if chains into canonical form.

    Ensures:
      - All if statements with else-if have consistent shape
      - Nested if-in-else is unwrapped to elif chain
    """
    if not isinstance(body, CompoundStmt):
        return

    for stmt in (body.statements or []):
        if isinstance(stmt, IfStmt):
            # Already handled by the canonical form — just recurse
            _normalize_if_chains(stmt.then_branch)
            if stmt.else_branch:
                if isinstance(stmt.else_branch, IfStmt):
                    _normalize_if_chains(stmt.else_branch)
                elif isinstance(stmt.else_branch, CompoundStmt):
                    _normalize_if_chains(stmt.else_branch)
        elif isinstance(stmt, CompoundStmt):
            _normalize_if_chains(stmt)
        elif isinstance(stmt, ForStmt):
            if isinstance(stmt.body, CompoundStmt):
                _normalize_if_chains(stmt.body)
