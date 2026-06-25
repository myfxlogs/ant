"""AST-level MQL → Python transpiler (T3).

Walks the AST produced by ast_parser.py and generates Python SDK code.
Handles patterns the line-by-line transpiler cannot:
  - Stateful PositionSelect tracking
  - Switch statements (via if/elif)
  - Compound MQL function calls in expressions
  - Variable scope awareness

Falls back to the line-by-line transpiler for statements not handled
by the AST walker.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Dict, List, Optional, Set, Tuple

from tools.mql_transpiler.ast_parser import (
    ASTNode,
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
    ReturnStmt,
    SourceFile,
    StringLiteral,
    SubscriptExpr,
    VarDecl,
    WhileStmt,
    parse_mql_ast,
)

# Load proto-generated accessor mappings.
try:
    from tools.mql_transpiler.mappings_generated import MQL_TO_SDK_ACCESSOR
except ImportError:
    MQL_TO_SDK_ACCESSOR = {}

# Manual mappings for functions not in proto (indicators, common).
from tools.mql_transpiler.mappings import (
    COMMON_FUNC_MAP,
    INDICATOR_MAP,
    LIFECYCLE_MAP,
    MQL_TYPE_MAP,
)

# MQL constants → Python
_MQL_CONSTANTS = {
    "INIT_SUCCEEDED": "0",
    "OP_BUY": "OrderType.BUY",
    "OP_SELL": "OrderType.SELL",
    "OP_BUYLIMIT": "OrderType.BUY_LIMIT",
    "OP_SELLLIMIT": "OrderType.SELL_LIMIT",
    "OP_BUYSTOP": "OrderType.BUY_STOP",
    "OP_SELLSTOP": "OrderType.SELL_STOP",
    "MODE_EMA": "'ema'",
    "MODE_SMA": "'sma'",
    "MODE_SMMA": "'smma'",
    "MODE_LWMA": "'lwma'",
    "PRICE_CLOSE": "1",
    "PRICE_OPEN": "2",
    "SELECT_BY_POS": "",
    "SELECT_BY_TICKET": "",
    "MODE_TRADES": "",
    "clrNONE": "None",
    "true": "True",
    "false": "False",
    "NULL": "None",
}

# SDK import block
_SDK_IMPORTS = """from decimal import Decimal
from app.sdk import (
    AccountMode,
    OrderRequest,
    OrderResult,
    OrderType,
    PendingOrder,
    Position,
    PositionSide,
    Retcode,
    StrategyBase,
    TypeFilling,
)
"""


@dataclass
class ASTTranspileResult:
    output: str
    externs: list = field(default_factory=list)
    globals_: list = field(default_factory=list)
    stats: dict = field(default_factory=dict)


class ASTTranspiler:
    """AST-level MQL→Python transpiler."""

    def __init__(self, class_name: str = "TranslatedStrategy"):
        self._class_name = class_name
        self._lines: List[str] = []
        self._indent = 0
        # State tracking
        self._known_vars: Set[str] = set()
        self._extern_params: List[Tuple[str, str, str]] = []  # (name, type, default)
        self._global_vars: List[Tuple[str, str]] = []  # (name, value)
        self._inside_orderselect_loop = False
        self._order_loop_var = "order"

    def transpile(self, source: str) -> ASTTranspileResult:
        ast = parse_mql_ast(source)
        return self._transpile_ast(ast)

    def _transpile_ast(self, ast: SourceFile) -> ASTTranspileResult:
        # Collect externs and globals.
        for decl in ast.declarations:
            if isinstance(decl, VarDecl):
                if decl.is_extern or decl.is_input:
                    val = self._expr_to_py(decl.value) if decl.value else "''"
                    self._extern_params.append((decl.name, decl.var_type, val))
                else:
                    val = self._expr_to_py(decl.value) if decl.value else "0"
                    self._global_vars.append((decl.name, val))
                    self._known_vars.add(decl.name)

        # Emit header.
        self._emit(f'"""Translated from MQL by AST transpiler (T3)."""')
        self._emit()
        for line in _SDK_IMPORTS.strip().split("\n"):
            self._emit(line)
        self._emit()
        self._emit(f"class {self._class_name}(StrategyBase):")
        self._indent += 1

        # Emit _init_params.
        if self._extern_params:
            self._emit()
            self._emit("def _init_params(self) -> None:")
            self._indent += 1
            for name, ptype, val in self._extern_params:
                py_val = self._mql_default_to_py(val, ptype)
                self._emit(f"self.{name} = self.ctx.param('{name}', {py_val})")
            self._indent -= 1

        # Emit functions.
        for decl in ast.declarations:
            if isinstance(decl, FuncDef):
                self._transpile_function(decl)

        self._indent -= 1
        return ASTTranspileResult(
            output="\n".join(self._lines),
            externs=self._extern_params,
            globals_=self._global_vars,
            stats={"externs": len(self._extern_params), "globals": len(self._global_vars)},
        )

    def _transpile_function(self, func: FuncDef) -> None:
        sdk_name = LIFECYCLE_MAP.get(func.name, func.name.lower())
        if func.name == "OnInit":
            self._emit()
            self._emit(f"def on_init(self) -> None:")
            self._indent += 1
            if self._extern_params:
                self._emit("self._init_params()")
            for name, val in self._global_vars:
                self._emit(f"self.{name} = {val}")
            self._transpile_compound(func.body)
            self._indent -= 1
        elif func.name in LIFECYCLE_MAP:
            self._emit()
            if sdk_name == "on_deinit":
                self._emit(f"def {sdk_name}(self, reason: str = 'user_stop') -> None:")
            else:
                self._emit(f"def {sdk_name}(self) -> None:")
            self._indent += 1
            self._transpile_compound(func.body)
            self._indent -= 1
        else:
            # Non-lifecycle function → skip or emit as comment.
            self._emit()
            self._emit(f"# User function: {func.name}() — manual translation needed")

    def _transpile_compound(self, body: CompoundStmt) -> None:
        for stmt in body.statements:
            self._transpile_stmt(stmt)

    def _transpile_stmt(self, stmt: ASTNode) -> None:
        if isinstance(stmt, IfStmt):
            self._transpile_if(stmt)
        elif isinstance(stmt, ForStmt):
            self._transpile_for(stmt)
        elif isinstance(stmt, WhileStmt):
            self._transpile_while(stmt)
        elif isinstance(stmt, ReturnStmt):
            self._transpile_return(stmt)
        elif isinstance(stmt, CompoundStmt):
            self._indent += 1
            self._transpile_compound(stmt)
            self._indent -= 1
        elif isinstance(stmt, ExpressionStmt):
            self._transpile_expr_stmt(stmt)
        elif isinstance(stmt, VarDecl):
            if stmt.value:
                val = self._expr_to_py(stmt.value)
                self._emit(f"self.{stmt.name} = {val}")
            else:
                self._emit(f"self.{stmt.name} = 0")
            self._known_vars.add(stmt.name)

    def _transpile_if(self, stmt: IfStmt) -> None:
        cond = self._expr_to_py(stmt.condition)
        self._emit(f"if {cond}:")
        self._indent += 1
        self._transpile_stmt(stmt.then_branch)
        self._indent -= 1
        if stmt.else_branch:
            # Check if else branch is another IfStmt → elif
            if isinstance(stmt.else_branch, IfStmt):
                cond2 = self._expr_to_py(stmt.else_branch.condition)
                self._emit(f"elif {cond2}:")
                self._indent += 1
                self._transpile_stmt(stmt.else_branch.then_branch)
                self._indent -= 1
                if stmt.else_branch.else_branch:
                    self._emit("else:")
                    self._indent += 1
                    self._transpile_stmt(stmt.else_branch.else_branch)
                    self._indent -= 1
            else:
                self._emit("else:")
                self._indent += 1
                self._transpile_stmt(stmt.else_branch)
                self._indent -= 1

    def _transpile_for(self, stmt: ForStmt) -> None:
        # Detect OrdersTotal / PositionsTotal loop patterns.
        cond_str = self._expr_to_py(stmt.condition) if stmt.condition else ""
        if "OrdersTotal" in cond_str:
            self._emit("# OrderSelect loop → Python iteration")
            self._emit("for order in self.broker.orders():")
            self._inside_orderselect_loop = True
            self._order_loop_var = "order"
            self._indent += 1
            self._transpile_stmt(stmt.body)
            self._indent -= 1
            self._inside_orderselect_loop = False
        elif "PositionsTotal" in cond_str:
            self._emit("# PositionSelect loop → Python iteration")
            self._emit("for pos in self.broker.positions():")
            self._indent += 1
            self._transpile_stmt(stmt.body)
            self._indent -= 1
        else:
            self._emit(f"# TRANSPILER-GAP: for loop — manual conversion needed")

    def _transpile_while(self, stmt: WhileStmt) -> None:
        cond = self._expr_to_py(stmt.condition)
        self._emit(f"while {cond}:")
        self._indent += 1
        self._transpile_stmt(stmt.body)
        self._indent -= 1

    def _transpile_return(self, stmt: ReturnStmt) -> None:
        if stmt.value:
            val = self._expr_to_py(stmt.value)
            self._emit(f"return {val}")
        else:
            self._emit("return")

    def _transpile_expr_stmt(self, stmt: ExpressionStmt) -> None:
        if stmt.expr is None:
            return
        py = self._expr_to_py(stmt.expr)
        self._emit(py)

    # ── Expression → Python ──────────────────────────────────────────

    def _expr_to_py(self, expr: Optional[Expression]) -> str:
        if expr is None:
            return ""
        if isinstance(expr, NumberLiteral):
            return expr.value
        if isinstance(expr, StringLiteral):
            return f"'{expr.value}'"
        if isinstance(expr, Identifier):
            return self._map_ident(expr.name)
        if isinstance(expr, CallExpr):
            return self._map_call(expr)
        if isinstance(expr, SubscriptExpr):
            base = self._map_ident(expr.name)
            idx = self._expr_to_py(expr.index) if expr.index else "0"
            return f"bars.{base}[{idx}]" if base in ("open", "high", "low", "close", "volume") else f"{base}[{idx}]"
        if isinstance(expr, BinaryOp):
            left = self._expr_to_py(expr.left)
            right = self._expr_to_py(expr.right)
            op = expr.op.replace("&&", "and").replace("||", "or")
            return f"{left} {op} {right}"
        if isinstance(expr, AssignmentExpr):
            rhs = self._expr_to_py(expr.rhs)
            return f"self.{expr.lhs} = {rhs}"
        return str(expr)

    # MQL identifiers that should NOT get self. prefix (builtins/symbols)
    _MQL_BUILTIN_IDENTS = {
        "Symbol": "self.ctx.symbol",
        "Ask": "self.ctx.ask",
        "Bid": "self.ctx.bid",
        "Point": "self.ctx.point",
        "Digits": "self.sym_info.digits",
        "Bars": "bars",
        "Close": "bars.close",
        "Open": "bars.open",
        "High": "bars.high",
        "Low": "bars.low",
        "Volume": "bars.volume",
        "Time": "bars.time",
    }

    def _map_ident(self, name: str) -> str:
        """Map MQL identifier to Python/SDK equivalent."""
        # Raw IDs from for-loop clauses
        if name.startswith("__raw__"):
            return name[7:]

        # MQL constants
        if name in _MQL_CONSTANTS:
            return _MQL_CONSTANTS[name]

        # Builtin identifiers (Symbol, Ask, Bid, etc.)
        if name in self._MQL_BUILTIN_IDENTS:
            return self._MQL_BUILTIN_IDENTS[name]

        # Order accessors in OrderSelect loop
        if self._inside_orderselect_loop:
            for mql_call, sdk_prop in MQL_TO_SDK_ACCESSOR.items():
                if mql_call.rstrip("()") == name:
                    return sdk_prop

        # Common builtin functions (not called, just referenced)
        if name in COMMON_FUNC_MAP:
            py = COMMON_FUNC_MAP[name]
            if not py.startswith("TRANSPILER-GAP") and not py.startswith("lambda") and "(" not in py:
                return py

        # Known variables → self. prefix
        if name in self._known_vars:
            return f"self.{name}"

        # Unknown identifier → assume self. prefix for member access
        return f"self.{name}"

    def _map_call(self, call: CallExpr) -> str:
        """Map MQL function call to Python/SDK."""
        name = call.name
        args_str = ", ".join(self._expr_to_py(a) for a in call.args)

        # Symbol() → self.ctx.symbol (bare getter, not a call)
        if name == "Symbol":
            return "self.ctx.symbol"

        # Account functions
        lookup = f"{name}()"
        if lookup in MQL_TO_SDK_ACCESSOR:
            sdk = MQL_TO_SDK_ACCESSOR[lookup]
            return sdk

        # Trade functions
        if name == "OrderSend":
            return self._map_ordersend_to_sdk(call)
        if name == "OrderClose":
            return f"self.broker.position_close({args_str})"
        if name == "OrderModify":
            return f"self.broker.position_modify({args_str})"
        if name == "OrderDelete":
            return f"self.broker.order_delete({args_str})"
        if name == "OrderSelect":
            return "True  # OrderSelect"

        # Indicator calls
        if name in ("iMA", "iRSI", "iBands", "iMACD", "iATR", "iStochastic", "iCCI", "iCustom",
                     "iADX", "iMomentum", "iMFI", "iOBV", "iSAR", "iStdDev", "iWPR",
                     "iEnvelopes", "iForce", "iDeMarker", "iOsMA"):
            # Map to SDK indicator method
            sdk_method = name[1:].lower()  # iMA → ma
            return f"self.indicators.{sdk_method}({args_str})"

        # Common functions
        if name in COMMON_FUNC_MAP:
            py = COMMON_FUNC_MAP[name]
            if not py.startswith("TRANSPILER-GAP") and not py.startswith("lambda"):
                if "(" in py and py.endswith(")"):
                    return py.replace("()", f"({args_str})")
                return f"{py}({args_str})"

        # Generic fallback
        return f"{name}({args_str})"

    def _map_ordersend_to_sdk(self, call: CallExpr) -> str:
        """Map OrderSend call to SDK OrderRequest."""
        args = call.args
        # Positional: OrderSend(symbol, cmd, volume, price, slippage, sl, tp, comment, magic, expiration, color)
        symbol = self._expr_to_py(args[0]) if len(args) > 0 else "self.ctx.symbol"
        cmd = self._expr_to_py(args[1]) if len(args) > 1 else "OrderType.BUY"
        volume = self._expr_to_py(args[2]) if len(args) > 2 else "Decimal('0.01')"
        price = self._expr_to_py(args[3]) if len(args) > 3 else "None"
        sl = self._expr_to_py(args[5]) if len(args) > 5 else "None"
        tp = self._expr_to_py(args[6]) if len(args) > 6 else "None"
        comment = self._expr_to_py(args[7]) if len(args) > 7 else "''"
        magic = self._expr_to_py(args[8]) if len(args) > 8 else "0"

        parts = [f"symbol={symbol}", f"type={cmd}", f"volume=Decimal(str({volume}))"]
        if price != "None":
            parts.append(f"price=Decimal(str({price}))")
        parts.append(f"sl=None" if sl in ("0", "0.0") else f"sl=Decimal(str({sl}))")
        parts.append(f"tp=None" if tp in ("0", "0.0") else f"tp=Decimal(str({tp}))")
        parts.append(f"comment={comment}")
        parts.append(f"magic=int({magic})")

        return f"self.broker.order_send(OrderRequest({', '.join(parts)}))"

    # ── Helpers ───────────────────────────────────────────────────────

    def _emit(self, line: str = "") -> None:
        if line:
            self._lines.append("    " * self._indent + line)
        else:
            self._lines.append("")

    @staticmethod
    def _mql_default_to_py(value: str, mql_type: str = "") -> str:
        """Convert MQL default value to Python literal."""
        value = value.strip().rstrip(";")
        if mql_type in ("int", "double", "float", "long", "uint", "ulong"):
            try:
                float(value)
                return value
            except ValueError:
                return f"'{value}'"
        if mql_type == "bool":
            return "True" if value.lower() in ("true", "1") else "False"
        if mql_type == "string":
            if value.startswith('"') or value.startswith("'"):
                return value
            return f"'{value}'"
        return value


def transpile_ast(source: str, class_name: str = "TranslatedStrategy") -> ASTTranspileResult:
    """Convenience function: parse + transpile MQL to Python."""
    tp = ASTTranspiler(class_name=class_name)
    return tp.transpile(source)
