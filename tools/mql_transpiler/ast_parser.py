"""Recursive-descent MQL parser → AST nodes (T3 AST-level transpiler).

Zero external dependencies. Based on the tree-sitter grammar rules
defined in tree_sitter_parser.py, translated to recursive descent.

Generates a simple AST that the AST-level transpiler walks to produce
Python SDK code.

Coverage targets (what line-by-line can't handle):
  - Stateful PositionSelect tracking
  - Switch statements
  - Nested ternary expressions
  - Compound conditions with MQL function calls
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field
from typing import List, Optional


# ── AST node types ───────────────────────────────────────────────────

class ASTNode:
    """Base AST node."""
    def __init__(self, **kwargs):
        for k, v in kwargs.items():
            setattr(self, k, v)


@dataclass
class SourceFile(ASTNode):
    declarations: list = field(default_factory=list)


@dataclass
class VarDecl(ASTNode):
    name: str = ""
    var_type: str = ""
    value: Optional[Expression] = None
    is_extern: bool = False
    is_input: bool = False


@dataclass
class FuncDef(ASTNode):
    name: str = ""
    return_type: str = "void"
    params: list = field(default_factory=list)
    body: CompoundStmt = None


@dataclass
class CompoundStmt(ASTNode):
    statements: list = field(default_factory=list)


@dataclass
class ExpressionStmt(ASTNode):
    expr: Expression = None


@dataclass
class IfStmt(ASTNode):
    condition: Expression = None
    then_branch: ASTNode = None
    else_branch: Optional[ASTNode] = None


@dataclass
class ForStmt(ASTNode):
    init: Expression = None
    condition: Expression = None
    update: Expression = None
    body: ASTNode = None


@dataclass
class WhileStmt(ASTNode):
    condition: Expression = None
    body: ASTNode = None


@dataclass
class SwitchStmt(ASTNode):
    expr: Expression = None
    cases: list = field(default_factory=list)  # [(value, body)]
    default: Optional[ASTNode] = None


class ReturnStmt(ASTNode):
    value: Optional[Expression] = None


@dataclass
class Expression(ASTNode):
    """Base expression node."""


@dataclass
class BinaryOp(Expression):
    left: Expression = None
    op: str = ""
    right: Expression = None


@dataclass
class UnaryOp(Expression):
    op: str = ""
    operand: Expression = None


@dataclass
class CallExpr(Expression):
    name: str = ""
    args: list = field(default_factory=list)


@dataclass
class SubscriptExpr(Expression):
    name: str = ""
    index: Expression = None


@dataclass
class Identifier(Expression):
    name: str = ""


@dataclass
class NumberLiteral(Expression):
    value: str = ""


@dataclass
class StringLiteral(Expression):
    value: str = ""


@dataclass
class AssignmentExpr(Expression):
    lhs: str = ""
    rhs: Expression = None


# ── Tokenizer ────────────────────────────────────────────────────────

_TOKEN_RE = re.compile(r"""
    \s*(?:
        (//[^\n]*|/\*.*?\*/)          |  # comments
        (\+\+|--|==|!=|<=|>=|&&|\|\||[+\-*/<>=!%]) |  # operators (++ and -- first for longest match)
        ([{}()\[\];,:])                |  # punctuation
        (\d+\.?\d*)                    |  # numbers
        ("[^"]*"|'[^']*')             |  # strings
        ([a-zA-Z_]\w*)                   # identifiers
    )
""", re.VERBOSE | re.DOTALL)


def tokenize(source: str) -> list[tuple[str, str]]:
    """Tokenize MQL source into (type, value) pairs."""
    tokens = []
    for m in _TOKEN_RE.finditer(source):
        if m.group(1):  # comment
            continue
        elif m.group(2):  # operator
            tokens.append(("OP", m.group(2)))
        elif m.group(3):  # punctuation
            tokens.append((m.group(3), m.group(3)))
        elif m.group(4):  # number
            tokens.append(("NUM", m.group(4)))
        elif m.group(5):  # string
            tokens.append(("STR", m.group(5)[1:-1]))
        elif m.group(6):  # identifier
            tokens.append(("ID", m.group(6)))
    return tokens


# ── Recursive descent parser ──────────────────────────────────────────

class Parser:
    """Recursive descent MQL parser."""

    def __init__(self, source: str):
        self.tokens = tokenize(source)
        self.pos = 0
        self._parse_depth = 0
        self._max_iter = len(self.tokens) * 3  # safety limit
        self._iters = 0
        self._type_keywords = {
            "int", "double", "bool", "string", "color", "datetime",
            "uint", "long", "ulong", "float", "short", "ushort", "void", "char",
        }

    def peek(self) -> Optional[tuple[str, str]]:
        if self.pos < len(self.tokens):
            return self.tokens[self.pos]
        return None

    def advance(self) -> tuple[str, str]:
        t = self.tokens[self.pos]
        self.pos += 1
        return t

    def expect(self, value: str) -> tuple[str, str]:
        t = self.advance()
        if t[1] != value:
            raise SyntaxError(f"Expected '{value}', got '{t[1]}' at pos {self.pos}")
        return t

    def match(self, ttype: str = None, value: str = None) -> bool:
        t = self.peek()
        if t is None:
            return False
        if ttype is not None and t[0] != ttype:
            return False
        if value is not None and t[1] != value:
            return False
        return True

    # ── Top level ─────────────────────────────────────────────────────

    def parse(self) -> SourceFile:
        """Parse a complete MQL source file."""
        declarations = []
        while self.peek() is not None and self._iters < self._max_iter:
            self._iters += 1
            decl = self._parse_top_level()
            if decl:
                declarations.append(decl)
            else:
                self.advance()  # skip unrecognized
        return SourceFile(declarations=declarations)

    def _parse_top_level(self) -> Optional[ASTNode]:
        """Parse a top-level declaration: extern, function, or variable."""
        if self.match(value="extern") or self.match(value="input"):
            return self._parse_extern()
        if self.match(value="#"):
            self._skip_line()
            return None
        if self.match(value="//"):
            self._skip_line()
            return None
        # Try type + identifier + ( = function def
        saved = self.pos
        if self.match("ID") and self.tokens[self.pos][1] in self._type_keywords:
            tok_type = self.advance()
            if self.match("ID"):
                tok_name = self.advance()
                if self.match(value="("):
                    self.pos = saved
                    return self._parse_function()
        self.pos = saved
        # Try variable declaration
        if self.match("ID") and self.tokens[self.pos][1] in self._type_keywords:
            return self._parse_var_decl()
        return None

    def _parse_extern(self) -> VarDecl:
        keyword = self.advance()  # extern or input
        is_input = keyword[1] == "input"
        decl = self._parse_var_decl()
        decl.is_extern = not is_input
        decl.is_input = is_input
        return decl

    def _parse_function(self) -> FuncDef:
        ret_type = self.advance()[1]
        name = self.advance()[1]
        self.expect("(")
        params = self._parse_param_list()
        self.expect(")")
        body = self._parse_compound()
        return FuncDef(name=name, return_type=ret_type, params=params, body=body)

    def _parse_param_list(self) -> list[str]:
        params = []
        if self.match(value=")"):
            return params
        # Skip type qualifiers (const, &, [], type keywords)
        self._skip_type_qualifiers()
        if self.match("ID"):
            params.append(self.advance()[1])
        while self.match(value=","):
            self.advance()
            self._skip_type_qualifiers()
            if self.match("ID"):
                params.append(self.advance()[1])
        return params

    def _skip_type_qualifiers(self) -> None:
        """Skip type keywords, const, &, [] before a parameter name."""
        while self.match("ID") and self.pos + 1 < len(self.tokens):
            if self.tokens[self.pos][1] in ("const", "int", "double", "bool", "string", "long",
                                              "uint", "ulong", "float", "short", "ushort", "char",
                                              "datetime", "color", "void"):
                self.advance()
            else:
                break
        while self.match("OP") and self.peek()[1] == "&":
            self.advance()

    def _parse_var_decl(self) -> VarDecl:
        vtype = self.advance()[1]
        name = self.advance()[1]
        value = None
        if self.match(value="="):
            self.advance()
            value = self._parse_expression()
        if self.match(value=";"):
            self.advance()
        return VarDecl(name=name, var_type=vtype, value=value)

    # ── Statements ────────────────────────────────────────────────────

    def _parse_compound(self) -> CompoundStmt:
        self.expect("{")
        stmts = []
        while not self.match(value="}") and self._iters < self._max_iter:
            self._iters += 1
            stmt = self._parse_statement()
            if stmt:
                stmts.append(stmt)
            elif self.peek() and self.peek()[1] != "}":
                self.advance()  # skip unrecognized token
            else:
                break
        if self.match(value="}"):
            self.advance()
        return CompoundStmt(statements=stmts)

    def _parse_statement(self) -> Optional[ASTNode]:
        t = self.peek()
        if t is None:
            return None
        if t[1] == "if":
            return self._parse_if()
        if t[1] == "for":
            return self._parse_for()
        if t[1] == "while":
            return self._parse_while()
        if t[1] == "return":
            return self._parse_return()
        if t[1] == "{":
            return self._parse_compound()
        if t[1] == "switch":
            return self._parse_switch()
        if t[1] == "}":
            return None
        if t[1] in self._type_keywords:
            return self._parse_var_decl()
        return self._parse_expr_stmt()

    def _parse_if(self) -> IfStmt:
        self.advance()  # if
        self.expect("(")
        cond = self._parse_expression()
        self.expect(")")
        then_branch = self._parse_statement()
        else_branch = None
        # Handle } else { and } else if {
        while self.match(value="}"):
            self.advance()
        if self.match(value="else"):
            self.advance()
            if self.match(value="if"):
                # else if → recursive if
                else_branch = self._parse_if()
            else:
                else_branch = self._parse_statement()
        return IfStmt(condition=cond, then_branch=then_branch, else_branch=else_branch)

    def _parse_for(self) -> ForStmt:
        self.advance()
        self.expect("(")
        # Init: may include type declaration (int i=0), handle separately.
        init_expr = self._parse_for_clause()
        cond_expr = self._parse_for_clause()
        update_expr = self._parse_for_clause()
        self.expect(")")
        body = self._parse_statement()
        return ForStmt(init=init_expr, condition=cond_expr, update=update_expr, body=body)

    def _parse_for_clause(self) -> Optional[Expression]:
        """Parse one clause of a for-loop (init/condition/update), stopping at ;."""
        tokens_buf = []
        while self.peek() and self.peek()[1] not in (";", ")"):
            tokens_buf.append(self.advance())
        if self.match(value=";"):
            self.advance()
        if not tokens_buf:
            return None
        # Simple case: single identifier or expression
        if len(tokens_buf) == 1:
            t = tokens_buf[0]
            if t[0] == "ID":
                return Identifier(name=t[1])
            if t[0] == "NUM":
                return NumberLiteral(value=t[1])
        # Reconstruct as a raw expression string for line-by-line fallback
        raw = " ".join(t[1] for t in tokens_buf)
        return Identifier(name=f"__raw__{raw}")

    def _parse_while(self) -> WhileStmt:
        self.advance()
        self.expect("(")
        cond = self._parse_expression()
        self.expect(")")
        body = self._parse_statement()
        return WhileStmt(condition=cond, body=body)

    def _parse_switch(self) -> SwitchStmt:
        self.advance()  # switch
        self.expect("(")
        expr = self._parse_expression()
        self.expect(")")
        self.expect("{")
        cases = []
        default = None
        while not self.match(value="}") and self._iters < self._max_iter:
            self._iters += 1
            if self.match(value="case"):
                self.advance()
                val = self._parse_expression()
                self.expect(":")
                stmts = []
                while self.peek() and self.peek()[1] not in ("case", "default", "}"):
                    s = self._parse_statement()
                    if s: stmts.append(s)
                cases.append((val, stmts))
            elif self.match(value="default"):
                self.advance()
                self.expect(":")
                stmts = []
                while self.peek() and self.peek()[1] not in ("case", "default", "}"):
                    s = self._parse_statement()
                    if s: stmts.append(s)
                default = stmts
            elif self.peek() and self.peek()[1] == "break":
                self.advance()
                if self.match(value=";"):
                    self.advance()
            else:
                self.advance()
        if self.match(value="}"):
            self.advance()
        return SwitchStmt(expr=expr, cases=cases, default=default)

    def _parse_return(self) -> ReturnStmt:
        self.advance()
        value = None
        if not self.match(value=";"):
            value = self._parse_expression()
        if self.match(value=";"):
            self.advance()
        return ReturnStmt(value=value)

    def _parse_expr_stmt(self) -> ExpressionStmt:
        expr = self._parse_expression()
        if self.match(value=";"):
            self.advance()
        return ExpressionStmt(expr=expr)

    # ── Expressions ───────────────────────────────────────────────────

    def _parse_expression(self) -> Optional[Expression]:
        """Parse a binary expression (lowest precedence first). Stops at ; ) ] , :"""
        self._parse_depth += 1
        if self._parse_depth > 200:
            self._parse_depth -= 1
            return None
        left = self._parse_primary()
        if left is None:
            return None
        while self.match("OP") or self.match(value=":"):
            op_tok = self.peek()
            if op_tok and op_tok[1] in (";", ")", "]", ",", "{", ":"):
                break
            op = self.advance()[1]
            if op == "=":
                if isinstance(left, Identifier):
                    rhs = self._parse_expression()
                    return AssignmentExpr(lhs=left.name, rhs=rhs)
                self.pos -= 1
                return left
            right = self._parse_primary()
            if right is None:
                break
            left = BinaryOp(left=left, op=op, right=right)
        self._parse_depth -= 1
        return left

    def _parse_primary(self) -> Optional[Expression]:
        t = self.peek()
        if t is None:
            return None
        if t[0] == "NUM":
            self.advance()
            return NumberLiteral(value=t[1])
        if t[0] == "STR":
            self.advance()
            return StringLiteral(value=t[1])
        if t[0] == "ID":
            name = self.advance()[1]
            # Function call?
            if self.match(value="("):
                return self._parse_call(name)
            # Subscript?
            if self.match(value="["):
                self.advance()
                idx = self._parse_expression()
                if self.match(value="]"):
                    self.advance()
                return SubscriptExpr(name=name, index=idx)
            return Identifier(name=name)
        if t[1] == "(":
            self.advance()
            expr = self._parse_expression()
            if self.match(value=")"):
                self.advance()
            return expr
        return None

    def _parse_call(self, name: str) -> CallExpr:
        self.expect("(")
        args = []
        if not self.match(value=")"):
            args.append(self._parse_expression())
            while self.match(value=","):
                self.advance()
                args.append(self._parse_expression())
        self.expect(")")
        return CallExpr(name=name, args=args)

    def _skip_line(self) -> None:
        while self.peek() and self.peek()[1] not in (";", "{", "}"):
            self.advance()
        if self.match(value=";"):
            self.advance()


def parse_mql_ast(source: str) -> SourceFile:
    """Parse MQL source and return an AST."""
    parser = Parser(source)
    return parser.parse()
