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

import re
from dataclasses import dataclass, field
from typing import Dict, List, Optional, Set, Tuple

from tools.mql_transpiler.ast_parser import (
    ASTNode,
    ArrayInitExpr,
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
    SwitchStmt,
    TernaryExpr,
    VarDecl,
    WhileStmt,
    parse_mql_ast,
)

# Load proto-generated accessor mappings.
try:
    from tools.mql_transpiler.mappings_generated import MQL_TO_SDK_ACCESSOR
except ImportError:
    MQL_TO_SDK_ACCESSOR = {}

# Load proto-generated RPC parameter order mappings.
try:
    from tools.mql_transpiler.rpc_params_generated import RPC_PARAM_ORDER
except ImportError:
    RPC_PARAM_ORDER = {}

# Manual mappings for functions not in proto (indicators, common).
from tools.mql_transpiler.mappings import (
    COMMON_FUNC_MAP,
    INDICATOR_MAP,
    LIFECYCLE_MAP,
    MQL_TYPE_MAP,
)

# Lazy import for hybrid mode — line-by-line transpiler as fallback.
_LINE_TRANSPILER = None

def _get_line_transpiler():
    global _LINE_TRANSPILER
    if _LINE_TRANSPILER is None:
        try:
            from tools.mql_transpiler.transpiler import MQLTranspiler
            _LINE_TRANSPILER = MQLTranspiler("_hybrid")
        except ImportError:
            pass
    return _LINE_TRANSPILER

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
        from tools.mql_transpiler.ast_parser import Parser
        parser = Parser(source)
        ast = parser.parse()
        # Merge parser globals (from #define) into our own.
        for name, val in parser._global_vars:
            self._global_vars.append((name, val))
            self._known_vars.add(name)
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
        elif isinstance(stmt, SwitchStmt):
            self._transpile_switch(stmt)
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
            expr_str = self._expr_to_py(stmt.expr)
            if expr_str and expr_str != "break":
                self._emit(expr_str)
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
            # Generic for-loop: for (int i = 1; i <= N; i++) → for i in range(1, N+1)
            init_str = self._expr_to_mql_text(stmt.init) if stmt.init else ""
            cond_str = self._expr_to_mql_text(stmt.condition) if stmt.condition else ""
            # Extract variable name and start value from init: "int i = 1" or "i = 1"
            var_match = re.match(r"(?:\w+\s+)?(\w+)\s*=\s*(.+)", init_str)
            if var_match:
                var_name = var_match.group(1)
                start_val = var_match.group(2).strip()
                # Extract end value from condition: "i <= N" or "i < N"
                if "<=" in cond_str:
                    end_val = cond_str.split("<=")[-1].strip() + " + 1"
                elif "<" in cond_str:
                    end_val = cond_str.split("<")[-1].strip()
                else:
                    end_val = "10"
                self._emit(f"for self.{var_name} in range({start_val}, {end_val}):")
                self._indent += 1
                self._transpile_stmt(stmt.body)
                self._indent -= 1
                self._known_vars.add(var_name)
            else:
                self._emit(f"# TRANSPILER-GAP: for loop — manual conversion needed")

    def _transpile_switch(self, stmt: SwitchStmt) -> None:
        """Translate MQL switch → Python if/elif/else."""
        sw_expr = self._expr_to_py(stmt.expr)
        for i, (val, body) in enumerate(stmt.cases):
            val_py = self._expr_to_py(val)
            if i == 0:
                self._emit(f"if {sw_expr} == {val_py}:")
            else:
                self._emit(f"elif {sw_expr} == {val_py}:")
            self._indent += 1
            for s in body:
                self._transpile_stmt(s)
            self._indent -= 1
        if stmt.default:
            if isinstance(stmt.default, list):
                self._emit("else:")
                self._indent += 1
                for s in stmt.default:
                    self._transpile_stmt(s)
                self._indent -= 1

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
        # If output looks incomplete (contains raw MQL function names), try line-by-line.
        if self._needs_line_fallback(py):
            mql_text = self._expr_to_mql_text(stmt.expr)
            if mql_text:
                lb = _get_line_transpiler()
                if lb:
                    try:
                        # Run line-by-line on this single statement, capture output.
                        lb_result = lb.transpile(f"void _dummy() {{ {mql_text} }}", "_hybrid.mq4")
                        # Extract the statement line from the output.
                        for line in lb_result.output.split("\n"):
                            stripped = line.strip()
                            if stripped and not stripped.startswith(("class", "def", "from", "import", '"""', "#")):
                                if "TRANSPILER-GAP" not in stripped:
                                    self._emit(stripped)
                                    return
                    except Exception:
                        pass
        self._emit(py)

    def _needs_line_fallback(self, py: str) -> bool:
        """Check if the Python output still contains untranslated MQL artifacts."""
        return bool(py) and ("self." not in py or "(" in py) and not py.startswith("self.broker")

    def _expr_to_mql_text(self, expr) -> str:
        """Serialize an AST expression back to MQL-like text for line-by-line fallback."""
        if isinstance(expr, CallExpr):
            args = ", ".join(self._expr_to_mql_text(a) for a in expr.args)
            return f"{expr.name}({args})"
        if isinstance(expr, AssignmentExpr):
            rhs = self._expr_to_mql_text(expr.rhs) if expr.rhs else ""
            return f"{expr.lhs} = {rhs}"
        if isinstance(expr, Identifier):
            name = expr.name
            return name[7:] if name.startswith("__raw__") else name  # strip __raw__ prefix
        if isinstance(expr, NumberLiteral):
            return expr.value
        if isinstance(expr, StringLiteral):
            return f'"{expr.value}"'
        if isinstance(expr, BinaryOp):
            left = self._expr_to_mql_text(expr.left) if expr.left else ""
            right = self._expr_to_mql_text(expr.right) if expr.right else ""
            return f"{left} {expr.op} {right}"
        if isinstance(expr, SubscriptExpr):
            idx = self._expr_to_mql_text(expr.index) if expr.index else "0"
            return f"{expr.name}[{idx}]"
        return ""

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
        if isinstance(expr, TernaryExpr):
            cond = self._expr_to_py(expr.condition)
            true_v = self._expr_to_py(expr.true_val)
            false_v = self._expr_to_py(expr.false_val)
            return f"{true_v} if {cond} else {false_v}"
        if isinstance(expr, ArrayInitExpr):
            elems = ", ".join(self._expr_to_py(e) for e in expr.elements)
            return f"[{elems}]"
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

        # Trade functions — use proto-generated parameter order
        if name == "OrderSend":
            return self._map_rpc_call_to_sdk(call, "OrderSend", "order_send")
        if name == "OrderClose":
            return self._map_rpc_call_to_sdk(call, "OrderClose", "position_close")
        if name == "OrderModify":
            return self._map_rpc_call_to_sdk(call, "OrderModify", "position_modify")
        if name == "OrderDelete":
            return self._map_rpc_call_to_sdk(call, "OrderDelete", "order_delete")
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

    def _map_rpc_call_to_sdk(self, call: CallExpr, rpc_name: str, sdk_method: str) -> str:
        """Map an MQL RPC call to SDK method using proto-generated parameter order."""
        args = call.args
        params = RPC_PARAM_ORDER.get(rpc_name, [])
        parts = []
        for i, (sdk_name, _proto_field, _fnum) in enumerate(params):
            if i >= len(args):
                break
            val_raw = self._expr_to_py(args[i])
            val = val_raw
            # Handle zero/empty → None for optional fields
            if val in ("0", "0.0", "None", "''", '""') and sdk_name in ("sl", "tp", "price", "expiration"):
                continue  # skip optional zero-valued fields
            if sdk_name in ("volume", "price") and val not in ("None", "0", "0.0", ""):
                val = f"Decimal(str({val}))"
            if sdk_name in ("sl", "tp") and val not in ("None", "0", "0.0"):
                val = f"Decimal(str({val}))"
            if sdk_name == "magic":
                if val in ("0", "None", "0.0", ""):
                    continue  # skip if zero
                val = f"int({val})"
            if sdk_name == "deviation":
                val = f"int({val})" if val not in ("0", "None", "") else "3"
            if sdk_name == "comment":
                if val in ("''", '""', ""):
                    continue  # skip empty comment
            parts.append(f"{sdk_name}={val}")
        return f"self.broker.{sdk_method}(OrderRequest({', '.join(parts)}))"

    def _map_ordersend_to_sdk(self, call: CallExpr) -> str:
        return self._map_rpc_call_to_sdk(call, "OrderSend", "order_send")

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
