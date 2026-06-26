"""识别器基类 + AST 遍历工具。"""

from __future__ import annotations

from typing import List, Optional, Set

from tools.mql_transpiler.ast_nodes import (
    AssignmentExpr,
    ASTNode,
    BinaryOp,
    CallExpr,
    CompoundStmt,
    ExpressionStmt,
    ForStmt,
    FuncDef,
    IfStmt,
    SourceFile,
    UnaryOp,
    VarDecl,
)


def find_functions(ast: SourceFile, *names: str) -> List[FuncDef]:
    """Find functions by name in the AST."""
    result = []
    for decl in (ast.declarations or []):
        if isinstance(decl, FuncDef) and (not names or decl.name in names):
            result.append(decl)
    return result


def extract_local_vars(func: FuncDef) -> Set[str]:
    """Extract local variable names from a function body.

    Scans VarDecl nodes inside the function body (including nested compound
    statements).  These are variables declared with a type inside the function
    — NOT global/extern variables.
    """
    locals_: Set[str] = set()

    def _scan(node):
        if node is None:
            return
        if isinstance(node, VarDecl):
            if node.name:
                locals_.add(node.name)
        if isinstance(node, CompoundStmt):
            for s in (node.statements or []):
                _scan(s)
        if isinstance(node, IfStmt):
            _scan(node.condition)
            _scan(node.then_branch)
            _scan(node.else_branch)
        if isinstance(node, ForStmt):
            _scan(node.init)  # for (int i = ...)
            _scan(node.body)

    _scan(func.body)
    return locals_


def walk_statements(body, *target_types):
    """Walk a body (CompoundStmt or Function body) and yield statements of target types."""
    items = body.statements if isinstance(body, CompoundStmt) else [body]
    for stmt in items:
        if isinstance(stmt, target_types):
            yield stmt
        if isinstance(stmt, CompoundStmt):
            yield from walk_statements(stmt, *target_types)


def find_calls(body, *call_names: str) -> List[CallExpr]:
    """Find all CallExpr nodes with matching names in a subtree."""
    found = []

    def _walk(node):
        if node is None:
            return
        if isinstance(node, CallExpr) and (not call_names or node.name in call_names):
            found.append(node)
        if isinstance(node, CompoundStmt):
            for s in (node.statements or []):
                _walk(s)
        elif isinstance(node, IfStmt):
            _walk(node.condition)
            _walk(node.then_branch)
            _walk(node.else_branch)
        elif isinstance(node, ForStmt):
            _walk(node.init)
            _walk(node.condition)
            _walk(node.update)
            _walk(node.body)
        elif isinstance(node, ExpressionStmt):
            _walk(node.expr)
        elif isinstance(node, BinaryOp):
            _walk(node.left)
            _walk(node.right)
        elif isinstance(node, UnaryOp):
            _walk(node.operand)
        elif isinstance(node, VarDecl):
            _walk(node.value)
        elif isinstance(node, AssignmentExpr):
            _walk(node.rhs)

    _walk(body)
    return found


def flatten_compound(body) -> List:
    """Unwrap CompoundStmt into a flat list, recursing into nested compounds."""
    if isinstance(body, CompoundStmt):
        result = []
        for s in (body.statements or []):
            if isinstance(s, CompoundStmt):
                result.extend(flatten_compound(s))
            else:
                result.append(s)
        return result
    return [body]


def get_arg(call: CallExpr, idx: int) -> Optional[ASTNode]:
    """Get the Nth argument of a CallExpr, or None."""
    if call.args and idx < len(call.args):
        return call.args[idx]
    return None


# ── Shared order-entry helpers (used by market_entry, pending_entry,
#    custom_entry recognizers) ────────────────────────────────────────

# MQL order type → Python OrderAction (single source of truth)
ORDER_TYPE_MAP = {
    "OP_BUY": "market_buy",
    "OP_SELL": "market_sell",
    "OP_BUYLIMIT": "buy_limit",
    "OP_SELLLIMIT": "sell_limit",
    "OP_BUYSTOP": "buy_stop",
    "OP_SELLSTOP": "sell_stop",
}


def find_ordersend_in_branch(branch) -> Optional[CallExpr]:
    """Find an OrderSend call in a branch (CompoundStmt or ExpressionStmt)."""
    if branch is None:
        return None
    if isinstance(branch, ExpressionStmt):
        if isinstance(branch.expr, CallExpr) and branch.expr.name == "OrderSend":
            return branch.expr
    if isinstance(branch, CompoundStmt):
        for stmt in (branch.statements or []):
            result = find_ordersend_in_branch(stmt)
            if result:
                return result
    return None


def extract_order_type(call: CallExpr) -> Optional[str]:
    """Extract order type from OrderSend — arg[1] is cmd."""
    type_arg = get_arg(call, 1)
    if type_arg is None:
        return None
    type_name = getattr(type_arg, 'name', None)
    if type_name:
        return ORDER_TYPE_MAP.get(type_name)
    return None


def extract_order_params(call: CallExpr, expr_gen) -> dict:
    """Extract OrderSend params: symbol, volume, price, deviation, sl, tp, magic, comment.

    Returns a dict compatible with OrderParams constructor.
    Reuses ExpressionGen for AST→Python translation.
    """
    args = call.args or []

    def arg(idx, default=""):
        if idx < len(args):
            return expr_gen.translate(args[idx])
        return default

    return {
        "symbol": arg(0),
        "volume": arg(2),
        "price": arg(3),
        "deviation": arg(4),
        "sl": arg(5),
        "tp": arg(6),
        "magic": arg(8),
        "comment": arg(7),
    }


def ast_contains_call(node, func_name: str) -> bool:
    """Check if an AST subtree contains a CallExpr with the given name.

    Recurses through all relevant AST node types.
    """
    if node is None:
        return False
    if isinstance(node, CallExpr) and node.name == func_name:
        return True
    if isinstance(node, BinaryOp):
        return (ast_contains_call(node.left, func_name) or
                ast_contains_call(node.right, func_name))
    if isinstance(node, UnaryOp):
        return ast_contains_call(node.operand, func_name)
    if isinstance(node, IfStmt):
        return (ast_contains_call(node.condition, func_name) or
                ast_contains_call(node.then_branch, func_name) or
                ast_contains_call(node.else_branch, func_name))
    if isinstance(node, CompoundStmt):
        for s in (node.statements or []):
            if ast_contains_call(s, func_name):
                return True
        return False
    if isinstance(node, ExpressionStmt):
        return ast_contains_call(node.expr, func_name)
    if isinstance(node, ForStmt):
        return ast_contains_call(node.body, func_name)
    if isinstance(node, VarDecl):
        return ast_contains_call(node.value, func_name)
    if isinstance(node, AssignmentExpr):
        return ast_contains_call(node.rhs, func_name)
    return False
