"""AST-level MQL → Python codegen.

Walks ``ast_nodes`` types produced by ``ast_bridge`` (tree-sitter CST→AST)
and generates Python SDK code.  Single parser, single codegen — no fallback.

Supports MQL4 (procedural) and MQL5 (class-based, flattened).
"""

from __future__ import annotations
from dataclasses import dataclass, field
from typing import Dict, List, Optional, Set, Tuple

from tools.mql_transpiler.ast_nodes import (
    ArrayInitExpr,
    AssignmentExpr,
    ASTNode,
    BinaryOp,
    CallExpr,
    ClassDef,
    CompoundStmt,
    Expression,
    ExpressionStmt,
    FieldDecl,
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
    UnaryOp,
    VarDecl,
    WhileStmt,
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

# ── C1 (ADR-0020 D8): regex fallback RETIRED ─────────────────────────
# The line-by-line transpiler (transpiler.py) and all hybrid fallback
# paths (_get_line_transpiler, _needs_line_fallback) are REMOVED.
# Expressions the AST cannot translate become hard GAP markers → LLM fills
# them (T4), passing through the same quality gates as everything else.

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
        self._known_vars: Set[str] = set()     # Global variables → self. prefix
        self._local_vars: Set[str] = set()     # Function-local variables → NO self. prefix
        self._extern_params: List[Tuple[str, str, str]] = []  # (name, type, default)
        self._global_vars: List[Tuple[str, str]] = []  # (name, value)
        self._inside_orderselect_loop = False
        self._order_loop_var = "order"

    def transpile(self, source: str) -> ASTTranspileResult:
        """Parse MQL source with tree-sitter and transpile to Python SDK code.

        No fallback — tree-sitter is the single parser (ADR-0020 D8, C1).
        """
        from tools.mql_transpiler.ast_bridge import parse_mql
        ast = parse_mql(source)
        return self._transpile_ast(ast)

    def _transpile_ast(self, ast: SourceFile) -> ASTTranspileResult:
        # Collect externs and globals — flatten ClassDef bodies too.
        declarations = list(ast.declarations) if ast.declarations else []
        for decl in ast.declarations or []:
            if isinstance(decl, VarDecl):
                if decl.is_extern or decl.is_input:
                    val = self._expr_to_py(decl.value) if decl.value else "''"
                    self._extern_params.append((decl.name, decl.var_type, val))
                else:
                    if decl.var_type == "bool":
                        val = self._expr_to_py(decl.value) if decl.value else "False"
                    else:
                        val = self._expr_to_py(decl.value) if decl.value else "0"
                    self._global_vars.append((decl.name, val))
                    self._known_vars.add(decl.name)
            elif isinstance(decl, ClassDef):
                # Use class name if no explicit class_name set.
                if self._class_name == "TranslatedStrategy" and decl.name:
                    self._class_name = decl.name
                # Optional base class.
                if decl.base_class:
                    self._class_name += f"({decl.base_class})"
                # Flatten class body: collect member fields + methods.
                for member in (decl.body or []):
                    if isinstance(member, FieldDecl):
                        if member.var_type == "bool":
                            val = self._expr_to_py(member.value) if member.value else "False"
                        else:
                            val = self._expr_to_py(member.value) if member.value else "0"
                        self._global_vars.append((member.name, val))
                        self._known_vars.add(member.name)
                    elif isinstance(member, FuncDef):
                        declarations.append(member)
                    elif isinstance(member, VarDecl):
                        if member.var_type == "bool":
                            val = self._expr_to_py(member.value) if member.value else "False"
                        else:
                            val = self._expr_to_py(member.value) if member.value else "0"
                        self._global_vars.append((member.name, val))
                        self._known_vars.add(member.name)

        # Emit header.
        self._emit(f'"""Translated from MQL by AST transpiler (T3)."""')
        self._emit()
        for line in _SDK_IMPORTS.strip().split("\n"):
            self._emit(line)
        self._emit()
        base = self._class_name if "(" in self._class_name else f"{self._class_name}(StrategyBase)"
        self._emit(f"class {base}:")
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
        for decl in declarations:
            if isinstance(decl, FuncDef):
                self._transpile_function(decl)

        self._indent -= 1
        # Confidence is now external (quality_gate.py — C2).
        # The old ``matched/(matched+gaps)`` algorithm is RETIRED.
        # This method collects gap info only; confidence is determined
        # by the parser-independent quality gates (ast.parse + SDK import + lint).
        output = "\n".join(self._lines)
        gap_count = output.count("# TRANSPILER-GAP")
        return ASTTranspileResult(
            output=output,
            externs=self._extern_params,
            globals_=self._global_vars,
            stats={
                "externs": len(self._extern_params),
                "globals": len(self._global_vars),
                "gaps": gap_count,
                "gap_samples": [l.strip()[len("# TRANSPILER-GAP: "):] for l in self._lines if "# TRANSPILER-GAP:" in l][:10],
            },
        )

    def _transpile_function(self, func: FuncDef) -> None:
        # Clear local variable tracking on function entry.
        self._local_vars.clear()

        # Track function parameters as local variables.
        if func.params:
            for p in func.params:
                self._local_vars.add(p)

        sdk_name = LIFECYCLE_MAP.get(func.name, func.name.lower())
        if func.name == "OnInit":
            self._emit()
            self._emit(f"def on_init(self) -> None:")
            self._indent += 1
            if self._extern_params:
                self._emit("self._init_params()")
            # Emit global var init (skip those already handled in body).
            emitted_globals = set()
            for name, val in self._global_vars:
                self._emit(f"self.{name} = {val}")
                emitted_globals.add(name)
            # Process body, but skip duplicate global init statements.
            self._transpile_compound_skip_globals(func.body, emitted_globals)
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
            # Non-lifecycle user function → strip _ prefix, emit as Python method.
            py_name = func.name.lstrip("_")
            # Preserve MQL parameters (func.params is List[str] of param names).
            param_strs = []
            if func.params:
                for p in func.params:
                    param_strs.append(f"{p}")
                    self._local_vars.add(p)  # So references don't get self. prefix
            params_sig = ", " + ", ".join(param_strs) if param_strs else ""
            self._emit()
            self._emit(f"def {py_name}(self{params_sig}) -> None:")
            self._indent += 1
            self._transpile_compound(func.body)
            self._indent -= 1

        self._local_vars.clear()

    def _transpile_compound(self, body: CompoundStmt) -> None:
        line_before = len(self._lines)
        for stmt in body.statements:
            self._transpile_stmt(stmt, skip_order_select=True)
        # If no actual Python statements were emitted (only GAP comments or blank
        # lines), emit a 'pass' to keep the function syntactically valid.
        if body.statements and len(self._lines) > line_before:
            new_lines = self._lines[line_before:]
            has_code = any(
                l.strip() and not l.strip().startswith("#")
                for l in new_lines
            )
            if not has_code:
                self._emit("pass")

    def _transpile_compound_skip_globals(self, body: CompoundStmt, skip_vars: set) -> None:
        """Like _transpile_compound but skips VarDecl/assignments for already-emitted globals
        only when the value is a trivial re-init (0, 0.0, False) matching the global default."""
        _TRIVIAL_INITS = {"0", "0.0", "False", "0.00", "''", '""'}
        for stmt in body.statements:
            # Skip VarDecl that re-initialize globals already emitted.
            if isinstance(stmt, VarDecl) and stmt.name in skip_vars:
                continue
            # Skip AssignmentExpr only if it sets the same trivial value as global default.
            if isinstance(stmt, ExpressionStmt) and stmt.expr:
                if isinstance(stmt.expr, AssignmentExpr) and stmt.expr.lhs in skip_vars:
                    rhs_val = self._expr_to_py(stmt.expr.rhs) if stmt.expr.rhs else ""
                    if rhs_val in _TRIVIAL_INITS:
                        continue  # skip trivial re-init (already emitted)
                    # Non-trivial value → emit (it overrides the global default).
            self._transpile_stmt(stmt, skip_order_select=True)

    def _transpile_stmt(self, stmt: ASTNode, skip_order_select: bool = False) -> None:
        # In an OrderSelect loop, skip the redundant "if (OrderSelect(...))" check
        # since every iteration of broker.orders() IS a valid order.
        if skip_order_select and self._inside_orderselect_loop and isinstance(stmt, IfStmt):
            if self._is_orderselect_check(stmt.condition):
                # Skip the if wrapper — emit body directly.
                self._transpile_branch(stmt.then_branch)
                return

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
            self._transpile_expr_stmt(stmt)
        elif isinstance(stmt, VarDecl):
            # Local variable (inside function body) → bare name, no self. prefix.
            self._local_vars.add(stmt.name)
            if stmt.value:
                val = self._expr_to_py(stmt.value)
                self._emit(f"{stmt.name} = {val}")
            elif stmt.var_type == "bool":
                self._emit(f"{stmt.name} = False")
            else:
                self._emit(f"{stmt.name} = 0")

    def _transpile_if(self, stmt: IfStmt) -> None:
        cond = self._expr_to_py(stmt.condition)
        self._emit(f"if {cond}:")
        self._indent += 1
        self._transpile_branch(stmt.then_branch)
        self._indent -= 1
        if stmt.else_branch:
            # Check if else branch is another IfStmt → elif
            if isinstance(stmt.else_branch, IfStmt):
                cond2 = self._expr_to_py(stmt.else_branch.condition)
                self._emit(f"elif {cond2}:")
                self._indent += 1
                self._transpile_branch(stmt.else_branch.then_branch)
                self._indent -= 1
                if stmt.else_branch.else_branch:
                    self._emit("else:")
                    self._indent += 1
                    self._transpile_branch(stmt.else_branch.else_branch)
                    self._indent -= 1
            else:
                self._emit("else:")
                self._indent += 1
                self._transpile_branch(stmt.else_branch)
                self._indent -= 1

    @staticmethod
    def _is_orderselect_check(condition) -> bool:
        """Check if an if-condition is just an OrderSelect(...) call."""
        if isinstance(condition, CallExpr):
            return condition.name == "OrderSelect"
        return False

    def _transpile_branch(self, stmt) -> None:
        """Emit a branch body — unwraps CompoundStmt to avoid double indent."""
        if isinstance(stmt, CompoundStmt):
            self._transpile_compound(stmt)
        else:
            self._transpile_stmt(stmt)

    def _transpile_for(self, stmt: ForStmt) -> None:
        # Detect order/position/history loop patterns.
        # Check the init expression AST for function calls
        # (can't use translated text — the mapping replaces the function name).
        has_orders_total = self._ast_contains_call(stmt.init, "OrdersTotal") or \
                           self._ast_contains_call(stmt.condition, "OrdersTotal")
        has_positions_total = self._ast_contains_call(stmt.init, "PositionsTotal") or \
                              self._ast_contains_call(stmt.condition, "PositionsTotal")
        has_history_deals = self._ast_contains_call(stmt.init, "HistoryDealsTotal") or \
                            self._ast_contains_call(stmt.condition, "HistoryDealsTotal")
        has_history_orders = self._ast_contains_call(stmt.init, "HistoryOrdersTotal") or \
                             self._ast_contains_call(stmt.condition, "HistoryOrdersTotal")

        if has_orders_total:
            # Detect intent: OrderClose → positions(), OrderDelete → orders().
            body_has_close = self._ast_contains_call(stmt.body, "OrderClose")
            body_has_delete = self._ast_contains_call(stmt.body, "OrderDelete")
            if body_has_close and not body_has_delete:
                self._emit("# OrderSelect loop → Python iteration")
                self._emit("for pos in self.broker.positions():")
                self._inside_orderselect_loop = True
                self._order_loop_var = "pos"
            else:
                self._emit("# OrderSelect loop → Python iteration")
                self._emit("for order in self.broker.orders():")
                self._inside_orderselect_loop = True
                self._order_loop_var = "order"
            self._indent += 1
            self._transpile_branch(stmt.body)
            self._indent -= 1
            self._inside_orderselect_loop = False
        elif has_positions_total:
            self._emit("# PositionSelect loop → Python iteration")
            self._emit("for pos in self.broker.positions():")
            self._inside_orderselect_loop = True
            self._order_loop_var = "pos"
            self._indent += 1
            self._transpile_branch(stmt.body)
            self._indent -= 1
            self._inside_orderselect_loop = False
        elif has_history_deals:
            self._emit("# HistorySelect loop → Python iteration")
            self._emit("for deal in self.broker.deals():")
            self._indent += 1
            self._transpile_branch(stmt.body)
            self._indent -= 1
        elif has_history_orders:
            self._emit("# HistoryOrderSelect loop → Python iteration")
            self._emit("for order in self.broker.history_orders():")
            self._indent += 1
            self._transpile_branch(stmt.body)
            self._indent -= 1
        else:
            # Generic for-loop: for (int i = 1; i <= N; i++) → for i in range(1, N+1)
            self._transpile_generic_for(stmt)

    @staticmethod
    def _ast_contains_call(node, func_name: str) -> bool:
        """Check if an AST node tree contains a CallExpr with the given name."""
        if node is None:
            return False
        if isinstance(node, CallExpr) and node.name == func_name:
            return True
        if isinstance(node, BinaryOp):
            return (ASTTranspiler._ast_contains_call(node.left, func_name) or
                    ASTTranspiler._ast_contains_call(node.right, func_name))
        if isinstance(node, UnaryOp):
            return ASTTranspiler._ast_contains_call(node.operand, func_name)
        if isinstance(node, VarDecl):
            return ASTTranspiler._ast_contains_call(node.value, func_name)
        if isinstance(node, AssignmentExpr):
            return ASTTranspiler._ast_contains_call(node.rhs, func_name)
        if isinstance(node, CompoundStmt):
            for stmt in (node.statements or []):
                if ASTTranspiler._ast_contains_call(stmt, func_name):
                    return True
            return False
        if isinstance(node, IfStmt):
            return (ASTTranspiler._ast_contains_call(node.condition, func_name) or
                    ASTTranspiler._ast_contains_call(node.then_branch, func_name) or
                    ASTTranspiler._ast_contains_call(node.else_branch, func_name))
        if isinstance(node, ExpressionStmt):
            return ASTTranspiler._ast_contains_call(node.expr, func_name)
        if isinstance(node, ForStmt):
            return ASTTranspiler._ast_contains_call(node.body, func_name)
        return False

    def _transpile_generic_for(self, stmt: ForStmt) -> None:
        """Translate generic MQL for-loop: for (init; cond; update) → Python for/while."""
        # Extract variable name and start value from init.
        var_name = ""
        start_val = "0"
        if isinstance(stmt.init, VarDecl):
            var_name = stmt.init.name
            start_val = self._expr_to_py(stmt.init.value) if stmt.init.value else "0"
        elif isinstance(stmt.init, AssignmentExpr):
            var_name = stmt.init.lhs
            start_val = self._expr_to_py(stmt.init.rhs) if stmt.init.rhs else "0"

        # Extract end value from condition: i <= N → N+1, i < N → N
        end_val = "10"
        step_val = None  # Only set for descending loops
        if isinstance(stmt.condition, BinaryOp):
            cond_str = self._expr_to_py(stmt.condition)
            if "<=" in cond_str:
                # Extract the right-hand side after <=
                rhs = self._expr_to_py(stmt.condition.right)
                end_val = f"{rhs} + 1"
            elif "<" in cond_str:
                rhs = self._expr_to_py(stmt.condition.right)
                end_val = str(rhs)
            elif ">=" in cond_str:
                # i >= N → range(start, N-1, -1)
                rhs = self._expr_to_py(stmt.condition.right)
                end_val = f"{rhs} - 1"
                step_val = "-1"
            else:
                end_val = "10"

        if var_name:
            self._local_vars.add(var_name)
            range_args = f"{start_val}, {end_val}"
            if step_val:
                range_args += f", {step_val}"
            self._emit(f"for {var_name} in range({range_args}):")
            self._indent += 1
            self._transpile_branch(stmt.body)
            self._indent -= 1
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
        self._transpile_branch(stmt.body)
        self._indent -= 1

    def _transpile_return(self, stmt: ReturnStmt) -> None:
        if stmt.value:
            val = self._expr_to_py(stmt.value)
            self._emit(f"return {val}")
        else:
            self._emit("return")

    def _transpile_expr_stmt(self, stmt: ExpressionStmt) -> None:
        """Translate an expression statement. Hard GAP on unrecognized patterns."""
        if stmt.expr is None:
            return
        py = self._expr_to_py(stmt.expr)
        # Check for obviously untranslated MQL artifacts.
        if self._looks_untranslated(py):
            mql_text = self._expr_to_mql_text(stmt.expr)
            self._emit(f"# TRANSPILER-GAP: expression — {mql_text[:80] if mql_text else 'unknown'}")
            return
        self._emit(py)

    @staticmethod
    def _looks_untranslated(py: str) -> bool:
        """Detect Python output that still contains MQL artifacts (C1: no regex fallback).

        These become hard GAP markers → LLM fills them in T4, going through
        the same quality gates as everything else.
        """
        if not py:
            return False
        # Bare MQL function names in the output.
        mql_indicators = {"iMA", "iRSI", "iATR", "iBands", "iMACD", "iStochastic",
                          "iCCI", "iCustom", "iADX", "iMomentum", "iMFI", "iOBV",
                          "iSAR", "iStdDev", "iWPR", "iEnvelopes", "iForce",
                          "iDeMarker", "iOsMA"}
        for fn in mql_indicators:
            if fn + "(" in py:
                return True
        # MQL relational operators not translated.
        if "&&" in py or "||" in py:
            return True
        return False

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
            if base in ("open", "high", "low", "close", "volume", "time"):
                return f"self.ctx.bars().{base}[{idx}]"
            return f"{base}[{idx}]"
        if isinstance(expr, BinaryOp):
            left = self._expr_to_py(expr.left)
            right = self._expr_to_py(expr.right)
            op = expr.op.replace("&&", "and").replace("||", "or")
            # Wrap division operands in Decimal to avoid float() validation errors.
            if op == "/":
                # Don't double-wrap if already Decimal
                if "Decimal(" not in left:
                    left = f"Decimal(str({left}))"
                if "Decimal(" not in right:
                    right = f"Decimal(str({right}))"
            return f"{left} {op} {right}"
        if isinstance(expr, UnaryOp):
            operand = self._expr_to_py(expr.operand)
            op = expr.op
            # MQL '!' → Python 'not '
            if op == "!":
                return f"not {operand}"
            return f"{op}{operand}"
        if isinstance(expr, AssignmentExpr):
            rhs = self._expr_to_py(expr.rhs)
            return f"self.{expr.lhs} = {rhs}"
        if isinstance(expr, TernaryExpr):
            cond = self._expr_to_py(expr.condition)
            true_v = self._expr_to_py(expr.true_val)
            false_v = self._expr_to_py(expr.false_val)
            return f"({true_v} if {cond} else {false_v})"
        if isinstance(expr, ArrayInitExpr):
            elems = ", ".join(self._expr_to_py(e) for e in expr.elements)
            return f"[{elems}]"
        return str(expr)

    # Order accessor → SDK field name (correct for OrderSelect loop context).
    _ORDER_ACCESSOR_SDK = {
        "OrderMagicNumber": "order.magic",
        "OrderTicket": "order.ticket",
        "OrderLots": "order.volume",
        "OrderSymbol": "order.symbol",
        "OrderType": "order.type",
        "OrderOpenPrice": "order.open_price",
        "OrderClosePrice": "order.close_price",
        "OrderStopLoss": "order.sl",
        "OrderTakeProfit": "order.tp",
        "OrderComment": "order.comment",
        "OrderCommission": "order.commission",
        "OrderSwap": "order.swap",
        "OrderProfit": "order.profit",
        "OrderOpenTime": "order.open_time_ms",
        "OrderCloseTime": "order.close_time_ms",
        "OrderVolume": "order.volume",
    }

    # MQL identifiers that should NOT get self. prefix (builtins/symbols)
    _MQL_BUILTIN_IDENTS = {
        "Symbol": "self.ctx.symbol",
        "Ask": "self.ctx.ask",
        "Bid": "self.ctx.bid",
        "Point": "self.ctx.point",
        "Digits": "self.broker.symbol_info(self.ctx.symbol).digits",
        "Bars": "self.ctx.bars().total()",
        "Close": "close",
        "Open": "open",
        "High": "high",
        "Low": "low",
        "Volume": "volume",
        "Time": "time",
        # MQL timeframe constants → None (follow backtest config)
        "PERIOD_M1": "None",
        "PERIOD_M5": "None",
        "PERIOD_M15": "None",
        "PERIOD_M30": "None",
        "PERIOD_H1": "None",
        "PERIOD_H4": "None",
        "PERIOD_D1": "None",
        "PERIOD_W1": "None",
        "PERIOD_MN1": "None",
        "PERIOD_CURRENT": "None",
        # MQL price constants → proper Python values
        "PRICE_CLOSE": "1",
        "PRICE_OPEN": "2",
        "PRICE_HIGH": "3",
        "PRICE_LOW": "4",
        "PRICE_MEDIAN": "5",
        "PRICE_TYPICAL": "6",
        "PRICE_WEIGHTED": "7",
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

        # Order accessors in OrderSelect loop → map to SDK field names,
        # using the current loop variable (order / pos / deal).
        if self._inside_orderselect_loop and name in self._ORDER_ACCESSOR_SDK:
            sdk = self._ORDER_ACCESSOR_SDK[name]
            # Replace "order." prefix with the current loop variable.
            lv = self._order_loop_var
            if sdk.startswith("order."):
                return sdk.replace("order.", f"{lv}.", 1)
            return sdk

        # Common builtin functions (not called, just referenced)
        if name in COMMON_FUNC_MAP:
            py = COMMON_FUNC_MAP[name]
            if not py.startswith("TRANSPILER-GAP") and not py.startswith("lambda") and "(" not in py:
                return py

        # Local variables (declared inside current function) → bare name.
        if name in self._local_vars:
            return name

        # Known global variables → self. prefix
        if name in self._known_vars:
            return f"self.{name}"

        # User-defined functions → self. prefix (they're methods on the class).
        # We check if this name appears as a FuncDef name in the AST — but
        # at this level we don't have access to that.  Instead, any unknown
        # identifier inside a function body that looks like a call target
        # gets self. prefix.  For bare references, assume self. too.
        return f"self.{name}"

    def _map_call(self, call: CallExpr) -> str:
        """Map MQL function call to Python/SDK."""
        name = call.name
        args_str = ", ".join(self._expr_to_py(a) for a in call.args)

        # Symbol() → self.ctx.symbol (bare getter, not a call)
        if name == "Symbol":
            return "self.ctx.symbol"

        # MarketInfo(symbol, MODE_XXX) → symbol_info(symbol).attr
        if name == "MarketInfo":
            return self._map_market_info(call)

        # Account/Order accessor functions — proto mappings (authoritative for non-loop).
        # In an OrderSelect loop, override with SDK-correct field names.
        if self._inside_orderselect_loop and name in self._ORDER_ACCESSOR_SDK:
            sdk = self._ORDER_ACCESSOR_SDK[name]
            lv = self._order_loop_var
            if sdk.startswith("order."):
                return sdk.replace("order.", f"{lv}.", 1)
            return sdk

        lookup = f"{name}()"
        if lookup in MQL_TO_SDK_ACCESSOR:
            sdk = MQL_TO_SDK_ACCESSOR[lookup]
            # Fix proto-generated names that don't match SDK (only in loop context).
            if self._inside_orderselect_loop:
                _LOOP_FIELD_FIXUP = {
                    "order.magic_number": "order.magic",
                    "order.lots": "order.volume",
                    "order.stop_loss": "order.sl",
                    "order.take_profit": "order.tp",
                    "order.order_type": "order.type",
                }
                sdk = _LOOP_FIELD_FIXUP.get(sdk, sdk)
                lv = self._order_loop_var
                if sdk.startswith("order."):
                    sdk = sdk.replace("order.", f"{lv}.", 1)
            return sdk

        # Trade functions — use proto-generated parameter order
        if name == "OrderSend":
            return self._map_rpc_call_to_sdk(call, "OrderSend", "order_send")
        if name == "OrderClose":
            return self._map_orderclose_to_sdk(call)
        if name == "OrderModify":
            return self._map_rpc_call_to_sdk(call, "OrderModify", "position_modify")
        if name == "OrderDelete":
            return self._map_rpc_call_to_sdk(call, "OrderDelete", "order_delete")
        if name == "OrderSelect":
            # OrderSelect → always True in broker.orders() iteration.
            return "True"

        # Indicator calls — map MQL positional args to SDK named params.
        indicator_map = self._map_indicator_call(name, call)
        if indicator_map is not None:
            return indicator_map

        # MQL time functions (no-arg calls).
        _TIME_FUNCTIONS = {
            "Day": "__import__('datetime').datetime.now().day",
            "Month": "__import__('datetime').datetime.now().month",
            "Year": "__import__('datetime').datetime.now().year",
            "Hour": "__import__('datetime').datetime.now().hour",
            "Minute": "__import__('datetime').datetime.now().minute",
            "Seconds": "__import__('datetime').datetime.now().second",
        }
        if name in _TIME_FUNCTIONS:
            return _TIME_FUNCTIONS[name]

        # Common functions (manual + proto-generated mappings)
        if name in COMMON_FUNC_MAP:
            py = COMMON_FUNC_MAP[name]
            if py.startswith("TRANSPILER-GAP") or py == "":
                return f"# TRANSPILER-GAP: {py[len('TRANSPILER-GAP: '):] if py.startswith('TRANSPILER-GAP: ') else name}"

            # Lambda entries: lambda a,b: expr → substitute args into expr.
            if py.startswith("lambda"):
                return _expand_lambda(py, args_str)

            # Direct replacement: MathAbs → abs, StringLen → len
            if "(" in py and py.endswith(")"):
                return py.replace("()", f"({args_str})")
            elif py.endswith(")"):
                return f"{py[:-1]}{args_str})"
            else:
                return f"{py}({args_str})"

        # Generic fallback: unknown functions are user-defined → self. prefix.
        # This covers cases like closeExisting(), closeByMagic(), CheckTime(), etc.
        # These MQL functions become Python methods on the strategy class.
        return f"self.{name}({args_str})"

    # ── Indicator arg mapping (MQL positional → SDK named) ───────────

    # MQL indicator signature → (sdk_method_name, [(mql_arg_index, sdk_param_name), ...])
    _INDICATOR_SIGNATURES: Dict[str, Tuple[str, List[Tuple[int, str]]]] = {
        "iMA":         ("ma",   [(2, "period"), (3, "shift"), (4, "method")]),
        "iRSI":        ("rsi",  [(2, "period"), (4, "shift")]),
        "iBands":      ("bands", [(2, "period"), (3, "deviation"), (7, "shift")]),
        "iMACD":       ("macd", [(2, "fast"), (3, "slow"), (4, "signal"), (7, "shift")]),
        "iATR":        ("atr",  [(2, "period"), (3, "shift")]),
        "iStochastic": ("stochastic", [(2, "k_period"), (3, "d_period"), (7, "shift")]),
        "iCCI":        ("cci",  [(2, "period"), (4, "shift")]),
        "iADX":        ("adx",  [(2, "period"), (4, "shift")]),
        "iMomentum":   ("momentum", [(2, "period"), (5, "shift")]),
        "iMFI":        ("mfi",  [(2, "period"), (4, "shift")]),
        "iOBV":        ("obv",  [(4, "shift")]),
        "iSAR":        ("sar",  [(2, "step"), (3, "maximum"), (4, "shift")]),
        "iStdDev":     ("stddev", [(2, "period"), (6, "shift")]),
        "iWPR":        ("wpr",  [(2, "period"), (4, "shift")]),
        "iEnvelopes":  ("envelopes", [(2, "period"), (6, "deviation"), (7, "shift")]),
        "iForce":      ("force", [(2, "period"), (5, "shift")]),
        "iDeMarker":   ("demarker", [(2, "period"), (4, "shift")]),
        "iOsMA":       ("osma", [(2, "fast"), (3, "slow"), (4, "signal"), (7, "shift")]),
    }

    def _map_indicator_call(self, name: str, call: CallExpr) -> Optional[str]:
        """Map MQL indicator call to SDK named-param call.

        Uses _INDICATOR_SIGNATURES to map MQL positional args to SDK keyword
        args.  Symbol (arg 0) and timeframe (arg 1) are always dropped as the
        SDK resolves them from context.  Applied_price and mode are dropped.
        """
        sig = self._INDICATOR_SIGNATURES.get(name)
        if sig is None:
            # Handle iCustom specially.
            if name == "iCustom":
                return self._map_icustom_call(call)
            return None

        sdk_name, arg_map = sig
        # Evaluate all MQL args to Python strings.
        py_args = [self._expr_to_py(a) for a in call.args]

        kwargs = []
        for mql_idx, sdk_param in arg_map:
            if mql_idx < len(py_args):
                val = py_args[mql_idx]
            else:
                continue  # Not enough args — skip.

            # Clean up MQL-specific values.
            val_str = str(val)
            if val_str in ("0", "None", "''", '""') and sdk_param in ("shift",):
                # 0 shift → default (0), skip emitting.
                if val_str == "0":
                    continue
            if val_str in ("0", "0.0") and sdk_param in ("deviation",):
                continue  # Use default.

            # Map MQL method constants (MODE_EMA, MODE_SMA, etc.)
            if sdk_param == "method":
                val = self._map_indicator_method(val_str)

            kwargs.append(f"{sdk_param}={val}")

        return f"self.indicators.{sdk_name}({', '.join(kwargs)})"

    @staticmethod
    def _map_indicator_method(raw: str) -> str:
        """Map MODE_EMA → 'ema', MODE_SMA → 'sma', etc."""
        method_map = {
            "0": "'sma'", "'ema'": "'ema'", "MODE_SMA": "'sma'",
            "MODE_EMA": "'ema'", "MODE_SMMA": "'smma'", "MODE_LWMA": "'lwma'",
        }
        return method_map.get(raw.strip("'\""), raw)

    def _map_icustom_call(self, call: CallExpr) -> str:
        """iCustom(symbol, tf, name, ..., mode, buffer, shift) → i_custom(name, params, buffer, shift)."""
        py_args = [self._expr_to_py(a) for a in call.args]
        if len(py_args) < 3:
            return "self.indicators.i_custom(name='unknown', params=[], buffer=0, shift=0)"

        # arg[2] = indicator name (may be string literal or variable ref).
        name_raw = str(py_args[2])
        if name_raw.startswith("'") and name_raw.endswith("'"):
            # String literal → strip quotes, re-quote cleanly.
            name_expr = f"'{name_raw[1:-1]}'"
        elif name_raw.startswith('"') and name_raw.endswith('"'):
            name_expr = f"'{name_raw[1:-1]}'"
        else:
            # Variable reference (e.g. self.CustomName) → pass directly.
            name_expr = name_raw
        # Last 3 args: mode, buffer, shift (for most iCustom calls)
        if len(py_args) >= 5:
            buffer_val = py_args[-2]
            shift_val = py_args[-1]
            # Middle args (between name and last 3) are the custom params
            param_vals = py_args[3:-2] if len(py_args) > 5 else []
        else:
            buffer_val = "0"
            shift_val = "0"
            param_vals = py_args[3:] if len(py_args) > 3 else []

        params_str = ", ".join(str(p) for p in param_vals)
        return f"self.indicators.i_custom(name={name_expr}, params=[{params_str}], buffer={buffer_val}, shift={shift_val})"

    def _map_orderclose_to_sdk(self, call: CallExpr) -> str:
        """OrderClose(ticket, volume, price, slippage) → self.broker.position_close(ticket, volume=...)."""
        py_args = [self._expr_to_py(a) for a in call.args]
        ticket = py_args[0] if len(py_args) > 0 else "0"
        volume = py_args[1] if len(py_args) > 1 else "Decimal(str(0))"
        # volume=OrderLots() → close all (no volume arg)
        if "OrderLots" in str(volume):
            return f"self.broker.position_close({ticket})"
        return f"self.broker.position_close({ticket}, volume=Decimal(str({volume})))"

    # ── RPC call mapping ──────────────────────────────────────────────

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
        # order_delete takes bare ticket (int), not OrderRequest.
        if sdk_method == "order_delete":
            ticket_val = parts[0].split("=", 1)[1] if parts else "0"
            return f"self.broker.{sdk_method}({ticket_val})"
        return f"self.broker.{sdk_method}(OrderRequest({', '.join(parts)}))"

    def _map_market_info(self, call: CallExpr) -> str:
        """MarketInfo(symbol, MODE_XXX) → symbol_info(symbol).attr."""
        py_args = [self._expr_to_py(a) for a in call.args]
        symbol = py_args[0] if len(py_args) > 0 else "self.ctx.symbol"
        mode = str(py_args[1]) if len(py_args) > 1 else ""

        # Map MODE_ constants to SymbolInfo attributes.
        # Strip self. prefix if present (from identifier mapping).
        mode_clean = mode.replace("self.", "").strip("'\"")
        _MODE_ATTR = {
            "MODE_POINT": "point", "POINT": "point",
            "MODE_DIGITS": "digits", "DIGITS": "digits",
            "MODE_SPREAD": "spread", "SPREAD": "spread",
            "MODE_STOPLEVEL": "stops_level",
            "MODE_LOTSIZE": "contract_size",
            "MODE_TICKVALUE": "tick_value",
            "MODE_TICKSIZE": "tick_size",
            "MODE_SWAPLONG": "swap_long",
            "MODE_SWAPSHORT": "swap_short",
            "0": "point",
            "1": "digits",
        }
        attr = _MODE_ATTR.get(mode_clean)
        if attr:
            return f"self.broker.symbol_info({symbol}).{attr}"
        return f"self.broker.symbol_info({symbol})"

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


def _expand_lambda(lambda_expr: str, args_str: str) -> str:
    """Expand a lambda mapping with the given argument string.

    Examples:
        lambda x: __import__('math').sin(x)  with args "0.5" → __import__('math').sin(0.5)
        lambda a,b: a % b                    with args "x, y" → x % y
        lambda: randint(0, 32767)            with args ""    → randint(0, 32767)
        lambda s, start, length: s[start:start+length] with args "s, 0, 5" → s[0:0+5]
    """
    import re

    # Parse: "lambda [params]: body"
    m = re.match(r'lambda\s*([^:]*?)\s*:\s*(.*)', lambda_expr, re.DOTALL)
    if not m:
        return lambda_expr  # Fallback — shouldn't happen.
    params_part = m.group(1).strip()
    body = m.group(2).strip()

    if not params_part:
        # No-arg lambda → body as-is.
        return body

    param_names = [p.strip() for p in params_part.split(",")]

    # Split args_str naively by comma (args are simple expressions, not nested).
    arg_values = [a.strip() for a in args_str.split(",")] if args_str else []

    # Substitute each param with its argument value.
    result = body
    for i, pname in enumerate(param_names):
        if i < len(arg_values):
            val = arg_values[i]
        else:
            val = "None"  # Under-apply → None
        # Replace the parameter name as a word.
        result = re.sub(rf'\b{re.escape(pname)}\b', val, result)
    return result


def transpile_ast(source: str, class_name: str = "TranslatedStrategy") -> ASTTranspileResult:
    """Convenience function: parse + transpile MQL to Python."""
    tp = ASTTranspiler(class_name=class_name)
    return tp.transpile(source)
