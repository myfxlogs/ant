"""MQL AST node types — pure data, zero logic, zero external dependencies.

These node types represent the parsed MQL source structure.  They are produced
by the CST→AST bridge (``ast_bridge.py``) and consumed by the codegen
(``ast_transpiler.py``).

No parser logic lives here.  No imports beyond the standard library.
"""

from __future__ import annotations


class ASTNode:
    """Base AST node."""


class SourceFile(ASTNode):
    def __init__(self, declarations=None):
        self.declarations = declarations


class VarDecl(ASTNode):
    def __init__(self, name="", var_type="", value=None, is_extern=False, is_input=False):
        self.name = name
        self.var_type = var_type
        self.value = value
        self.is_extern = is_extern
        self.is_input = is_input


class FuncDef(ASTNode):
    def __init__(self, name="", return_type="void", params=None, body=None):
        self.name = name
        self.return_type = return_type
        self.params = params
        self.body = body


class CompoundStmt(ASTNode):
    def __init__(self, statements=None):
        self.statements = statements


class ExpressionStmt(ASTNode):
    def __init__(self, expr=None):
        self.expr = expr


class IfStmt(ASTNode):
    def __init__(self, condition=None, then_branch=None, else_branch=None):
        self.condition = condition
        self.then_branch = then_branch
        self.else_branch = else_branch


class ForStmt(ASTNode):
    def __init__(self, init=None, condition=None, update=None, body=None):
        self.init = init
        self.condition = condition
        self.update = update
        self.body = body


class WhileStmt(ASTNode):
    def __init__(self, condition=None, body=None):
        self.condition = condition
        self.body = body


class SwitchStmt(ASTNode):
    def __init__(self, expr=None, cases=None, default=None):
        self.expr = expr
        self.cases = cases or []
        self.default = default


class ReturnStmt(ASTNode):
    def __init__(self, value=None):
        self.value = value


class Expression(ASTNode):
    """Base expression node."""


class TernaryExpr(Expression):
    """condition ? true_val : false_val"""
    def __init__(self, condition=None, true_val=None, false_val=None):
        self.condition = condition
        self.true_val = true_val
        self.false_val = false_val


class ArrayInitExpr(Expression):
    """{1.0, 2.0, 3.0} or array literal"""
    def __init__(self, elements=None):
        self.elements = elements or []


class BinaryOp(Expression):
    def __init__(self, left=None, op="", right=None):
        self.left = left
        self.op = op
        self.right = right


class UnaryOp(Expression):
    def __init__(self, op="", operand=None):
        self.op = op
        self.operand = operand


class CallExpr(Expression):
    def __init__(self, name="", args=None):
        self.name = name
        self.args = args


class SubscriptExpr(Expression):
    def __init__(self, name="", index=None):
        self.name = name
        self.index = index


class Identifier(Expression):
    def __init__(self, name=""):
        self.name = name


class NumberLiteral(Expression):
    def __init__(self, value=""):
        self.value = value


class StringLiteral(Expression):
    def __init__(self, value=""):
        self.value = value


class AssignmentExpr(Expression):
    def __init__(self, lhs="", rhs=None):
        self.lhs = lhs
        self.rhs = rhs
