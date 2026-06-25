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
from typing import List, Optional


# ── AST node types ───────────────────────────────────────────────────

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


# ── Tokenizer ────────────────────────────────────────────────────────

_TOKEN_RE = re.compile(r"""
    \s*(?:
        (//[^\n]*|/\*.*?\*/)          |  # comments
        (\#\w+)                        |  # preprocessor (#define, #property...)
        (\+\+|--|==|!=|<=|>=|&&|\|\||[+\-*/<>=!%]) |  # operators
        ([{}()\[\];,:?])               |  # punctuation
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
        elif m.group(2):  # preprocessor
            tokens.append(("PP", m.group(2)))
        elif m.group(3):  # operator
            tokens.append(("OP", m.group(3)))
        elif m.group(4):  # punctuation
            tokens.append((m.group(4), m.group(4)))
        elif m.group(5):  # number
            tokens.append(("NUM", m.group(5)))
        elif m.group(6):  # string
            tokens.append(("STR", m.group(6)[1:-1]))
        elif m.group(7):  # identifier
            tokens.append(("ID", m.group(7)))
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
        self._global_vars: list = []  # from #define macros
        self._known_vars: set = set()  # known variable names

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
        if self.match("PP"):
            self._parse_preprocessor()
            return ASTNode()  # marker — prevents double-advance
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
        # Check for ternary: condition ? true_val : false_val
        if self.match(value="?"):
            self.advance()
            true_val = self._parse_expression()
            if self.match(value=":"):
                self.advance()
                false_val = self._parse_expression()
                left = TernaryExpr(condition=left, true_val=true_val, false_val=false_val)
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
        if t[1] == "{":
            return self._parse_array_init()
        return None

    def _parse_array_init(self) -> ArrayInitExpr:
        self.advance()  # {
        elements = []
        while self.peek() and self.peek()[1] != "}":
            self._iters += 1
            elem = self._parse_expression()
            if elem:
                elements.append(elem)
            if self.match(value=","):
                self.advance()
            elif self.peek()[1] == "}":
                break
        if self.match(value="}"):
            self.advance()
        return ArrayInitExpr(elements=elements)

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

    def _parse_preprocessor(self) -> None:
        """Handle #define, #property, #include — store macros as globals."""
        pp = self.advance()[1]  # e.g. "#define"
        if pp == "#define" and self.match("ID"):
            name = self.advance()[1]
            value_parts = []
            while self.peek() and self.peek()[1] not in (";", "{", "}", "#") and self.peek()[0] != "PP":
                # Stop before the next type keyword or function def
                if self.peek()[0] == "ID" and self.peek()[1] in self._type_keywords:
                    break
                value_parts.append(self.advance()[1])
            value = " ".join(value_parts).strip()
            if value:
                self._global_vars.append((name, value))
                self._known_vars.add(name)
        else:
            while self.peek() and self.peek()[1] not in (";", "{", "}", "#"):
                self.advance()

    def _skip_line(self) -> None:
        while self.peek() and self.peek()[1] not in (";", "{", "}"):
            self.advance()
        if self.match(value=";"):
            self.advance()


def parse_mql_ast(source: str) -> SourceFile:
    """Parse MQL source and return an AST."""
    parser = Parser(source)
    return parser.parse()
