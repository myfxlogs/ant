"""MQL → Python SDK transpiler engine (T2.1).

Deterministic, statement-level pattern-matching translator.  No AST, no LLM.
Each MQL construct is matched against mapping tables; matching produces
SDK code; non-matching gets ``// TRANSPILER-GAP:``.

Design: line-by-line with multi-line context for function bodies.
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field
from typing import Dict, List, Optional, Set, Tuple

from tools.mql_transpiler.mappings import (
    CMD_MAP,
    COMMON_FUNC_MAP,
    EXTERN_RE,
    INDICATOR_MAP,
    INPUT_RE,
    LIFECYCLE_MAP,
    METHOD_MAP,
    MQL_TYPE_MAP,
    ORDER_ACCESSOR_MAP,
    ORDERCLOSE_RE,
    ORDERDELETE_RE,
    ORDERMODIFY_RE,
    ORDERSELECT_RE,
    ORDERSEND_RE,
    ORDERSTOTAL_RE,
    POSITION_ACCESSOR_MAP,
    POSITIONSTOTAL_RE,
    GapReason,
    MappingResult,
    method_name,
)

# ── Data types ──────────────────────────────────────────────────────────


@dataclass
class TranspilerStats:
    lines_in: int = 0
    lines_out: int = 0
    patterns_matched: int = 0
    gaps: int = 0
    gap_reasons: Dict[str, int] = field(default_factory=dict)

    def record_gap(self, reason: str) -> None:
        self.gaps += 1
        self.gap_reasons[reason] = self.gap_reasons.get(reason, 0) + 1


@dataclass
class TranspileResult:
    source: str
    output: str
    stats: TranspilerStats


# ── Pattern matching helpers ────────────────────────────────────────────

# Match MQL variable declarations:  type name = value;
_VAR_DECL_RE = re.compile(
    r"(?P<type>int|double|bool|string|color|datetime|uint|long|ulong|float|short|ushort)\s+"
    r"(?P<name>[a-zA-Z_]\w*)\s*=\s*(?P<value>[^;]+)\s*;"
)

# Match MQL function definitions.
_FUNC_DEF_RE = re.compile(
    r"(?P<ret>\w+(?:\s+\w+)*)\s+"
    r"(?P<name>[a-zA-Z_]\w*)\s*"
    r"\(\s*(?P<params>[^)]*)\s*\)"
)

# Match for/while/if blocks — balanced-paren extraction helpers.


def _mql_default_to_py(value: str, mql_type: str = "") -> str:
    """Convert an MQL default value literal to a Python expression.

    >>> _mql_default_to_py("0.01", "double")
    '0.01'
    >>> _mql_default_to_py("true", "bool")
    'True'
    >>> _mql_default_to_py('"Venus"', "string")
    '"Venus"'
    """
    v = value.strip().rstrip(";")
    # MQL boolean / null constants.
    if v.lower() == "true":
        return "True"
    if v.lower() == "false":
        return "False"
    if v.lower() == "null":
        return "None"
    # Already a quoted string.
    if (v.startswith('"') and v.endswith('"')) or (v.startswith("'") and v.endswith("'")):
        return v
    # Enum value (not a number) → wrap in quotes.
    if mql_type.lower() in ("int", "uint", "long", "ulong", "short", "ushort"):
        try:
            int(v)
            return v
        except ValueError:
            return f"'{v}'  # enum"
    if mql_type.lower() in ("double", "float"):
        try:
            float(v)
            return v
        except ValueError:
            return f"'{v}'  # enum"
    # String type but not quoted → quote it.
    if mql_type.lower() in ("string", "color"):
        return f'"{v}"'
    # Default: pass through as-is (could be expression).
    return v


def _extract_parens(text: str, start_pos: int) -> str:
    """Extract content inside balanced parentheses starting at start_pos."""
    depth = 1
    i = start_pos
    while i < len(text) and depth > 0:
        if text[i] == "(":
            depth += 1
        elif text[i] == ")":
            depth -= 1
        i += 1
    return text[start_pos:i - 1]


# MQL type keywords that can prefix variable declarations inside function bodies.
_MQL_TYPE_KEYWORDS: set[str] = {
    "int", "double", "bool", "string", "datetime", "color",
    "void", "uint", "long", "ulong", "float", "short", "ushort",
    "char", "uchar",
}


def _strip_mql_type_prefix(stripped: str) -> str:
    """Remove a leading MQL type keyword from a statement.

    ``"int ticket = OrderSend(...)"`` → ``"ticket = OrderSend(...)"``
    ``"double volume = 0.01;"`` → ``"volume = 0.01"``
    """
    for kw in sorted(_MQL_TYPE_KEYWORDS, key=len, reverse=True):
        if stripped.startswith(kw + " "):
            return stripped[len(kw) + 1 :]
    return stripped


def _match_control_flow(line: str, keyword: str) -> Optional[str]:
    """Extract condition from if/while/for statement with balanced parens."""
    stripped = line.strip()
    prefix = keyword + " ("
    if stripped.startswith(keyword + "("):
        return _extract_parens(stripped, len(keyword) + 1)
    if stripped.startswith(prefix):
        return _extract_parens(stripped, len(keyword) + 2)
    return None

# Match OrderSelect loop pattern.
_ORDERSELECT_POS_RE = re.compile(
    r"OrderSelect\s*\(\s*(?P<index>[^,]+)\s*,\s*SELECT_BY_POS\s*,\s*MODE_TRADES\s*\)"
)
_ORDERSELECT_TICKET_RE = re.compile(
    r"OrderSelect\s*\(\s*(?P<ticket>[^,]+)\s*,\s*SELECT_BY_TICKET\s*\)"
)

# Match PositionSelect loop.
_POSITIONSELECT_RE = re.compile(
    r"PositionSelect\s*\(\s*(?P<symbol>[^)]+)\s*\)"
)

# Match OrderSend return value.
_ORDERSEND_TICKET_RE = re.compile(r"(\w+)\s*=\s*OrderSend\(")
_ORDERRESULT_RE = re.compile(r"(\w+)\s*=\s*OrderSend\(")

# MQL comparison operators (== → ==, != stays !=).
_MQL_COMPARE = {
    "==": "==", "!=": "!=", ">": ">", "<": "<", ">=": ">=", "<=": "<=",
    "&&": "and", "||": "or", "!": "not ",
}

# MQL constants.
_MQL_CONSTANTS: Dict[str, str] = {
    "NULL": "None",
    "true": "True",
    "false": "False",
    "EMPTY_VALUE": "0.0",
    "EMPTY": '""',
    "WHOLE_ARRAY": "None",
    "MODE_ASCEND": "0",
    "MODE_DESCEND": "1",
    "CHARTS_MAX": "100",
    "CLR_NONE": '""',
    "INVALID_HANDLE": "None",
}


# ── Transpiler ──────────────────────────────────────────────────────────


class MQLTranspiler:
    """Statement-level MQL → Python SDK transpiler.

    Usage::

        transpiler = MQLTranspiler()
        result = transpiler.transpile(mql_source)
        print(result.output)
    """

    def __init__(self, class_name: str = "TranslatedStrategy") -> None:
        self._class_name = class_name
        self._stats = TranspilerStats()
        self._output_lines: List[str] = []
        self._indent = 0
        self._inside_function = False
        self._inside_orderselect_loop = False
        self._order_loop_var = "order"
        self._params_found: List[Tuple[str, str, str]] = []  # (name, type, value)
        self._extern_found: List[Tuple[str, str, str]] = []

    # ── Public API ─────────────────────────────────────────────────────

    def transpile(self, source: str, filename: str = "strategy.mq4") -> TranspileResult:
        """Transpile MQL source to Python SDK code."""
        self._stats = TranspilerStats()
        self._output_lines = []
        self._indent = 0
        self._inside_function = False
        self._params_found = []
        self._extern_found = []
        self._has_on_init_func = False  # True if MQL source has OnInit()

        raw_lines = source.split("\n")

        # Pre-process: join multi-line function call arguments.
        lines = self._join_continuation_lines(raw_lines)

        self._stats.lines_in = len(raw_lines)

        # Phase 1: discover extern/input declarations AND check for OnInit.
        for line in lines:
            self._discover_params(line)
            if not self._has_on_init_func:
                m = _FUNC_DEF_RE.match(line.strip())
                if m and m.group("name") == "OnInit":
                    self._has_on_init_func = True

        # Phase 2: emit header.
        self._emit_header(filename)

        # Phase 2.5: if params exist, emit _init_params() helper.
        # When no MQL OnInit exists, also emit on_init() calling it.
        all_params = self._extern_found + self._params_found
        if all_params:
            self._emit_init_params(all_params)
            if not self._has_on_init_func:
                self._emit_on_init_stub()

        # Phase 3: process each line.
        i = 0
        while i < len(lines):
            line = lines[i]
            stripped = line.strip()

            # Skip empty lines and preprocessor.
            if not stripped or stripped.startswith("#"):
                i += 1
                continue

            # Comments pass through.
            if stripped.startswith("//") or stripped.startswith("/*"):
                self._emit(line)
                i += 1
                continue

            # Function definition.
            func_match = _FUNC_DEF_RE.match(stripped)
            # A function definition has a signature like "type Name(...)" followed by
            # either "{" on the same line, "{" on the next line, or a full inline body.
            is_func_def = bool(func_match) and (
                stripped.rstrip().endswith("{")
                or (i + 1 < len(lines) and lines[i + 1].strip() == "{")
                or ("{" in stripped.split(")", 1)[-1] if ")" in stripped else False)
            )
            if is_func_def:
                name = func_match.group("name")
                self._emit_function_def(name, func_match, stripped, lines, i)
                # Process function body statements.
                i = self._process_function_body(lines, i)
                self._inside_function = False
                self._emit()  # blank line after function
                continue

            # Variable declarations.
            var_match = _VAR_DECL_RE.match(stripped)
            if var_match:
                self._emit_variable(var_match)
                i += 1
                continue

            # Global declarations (extern/input handled in on_init).
            if stripped.startswith("extern ") or stripped.startswith("input "):
                self._emit(f"# {stripped}")
                i += 1
                continue

            # Single-line statements.
            self._emit_statement(stripped)
            i += 1

        # Phase 4: emit footer.
        self._emit_footer()

        self._stats.lines_out = len(self._output_lines)
        return TranspileResult(
            source=source,
            output="\n".join(self._output_lines),
            stats=self._stats,
        )

    # ── Emitters ────────────────────────────────────────────────────────

    def _emit(self, line: str = "", indent_delta: int = 0) -> None:
        self._indent += indent_delta
        if line:
            self._output_lines.append("    " * self._indent + line)
        else:
            self._output_lines.append("")

    def _emit_gap(self, reason: str, original: str = "") -> None:
        self._stats.record_gap(reason)
        self._emit(f"# TRANSPILER-GAP: {reason}")
        if original:
            self._emit(f"#   original: {original}")

    def _emit_header(self, filename: str) -> None:
        self._emit(f'"""Translated from: {filename}')
        self._emit(f"Tool: tools/mql_transpiler (T2.1)")
        self._emit(f"Generated by deterministic transpiler — zero LLM, auditable.")
        self._emit(f'"""')
        self._emit()
        self._emit("from decimal import Decimal")
        self._emit("from typing import List, Optional")
        self._emit()
        self._emit("from app.sdk import (")
        self._emit("    AccountMode,", 1)
        self._emit("    OrderRequest,")
        self._emit("    OrderResult,")
        self._emit("    OrderType,")
        self._emit("    PendingOrder,")
        self._emit("    Position,")
        self._emit("    PositionSide,")
        self._emit("    Retcode,")
        self._emit("    StrategyBase,")
        self._emit("    TypeFilling,")
        self._emit(")", -1)
        self._emit()
        self._emit()
        self._emit(f"class {self._class_name}(StrategyBase):")
        self._indent = 1

    def get_confidence(self) -> float:
        """Return 0.0–1.0 confidence score based on matched vs total patterns."""
        total = self._stats.patterns_matched + self._stats.gaps
        if total == 0: return 1.0  # empty source = no patterns needed
        return self._stats.patterns_matched / total

    def get_gap_samples(self, max_samples: int = 10) -> list:
        """Return unique gap reasons for display."""
        return list(self._stats.gap_reasons.keys())[:max_samples]

    def _emit_footer(self) -> None:
        self._indent = 0

    def _emit_init_params(self, all_params: list) -> None:
        """Emit ``_init_params()`` helper with all extern/input param reads."""
        self._emit()
        self._emit("# ——— Auto-generated: read extern/input parameters ———")
        self._emit("def _init_params(self) -> None:")
        self._indent += 1
        for pname, ptype, pvalue in all_params:
            py_val = _mql_default_to_py(pvalue, ptype)
            self._emit(f"self.{pname} = self.ctx.param('{pname}', {py_val})")
        self._indent -= 1
        self._emit()  # blank line after method

    def _emit_on_init_stub(self) -> None:
        """Emit ``on_init()`` that delegates to ``_init_params()``.
        Used when the MQL source has no OnInit function."""
        self._emit("def on_init(self) -> None:")
        self._indent += 1
        self._emit("self._init_params()")
        self._emit_init_globals()
        self._indent -= 1
        self._emit()

    def _emit_init_globals(self) -> None:
        """Emit global variable assignments inside on_init."""
        if not hasattr(self, '_global_vars') or not self._global_vars:
            return
        for vname, py_value in self._global_vars:
            self._emit(f"self.{vname} = {py_value}")
        self._global_vars = []  # clear for next translation

    def _emit_function_def(
        self, name: str, match, line: str, lines: List[str], line_idx: int
    ) -> None:
        if name in LIFECYCLE_MAP:
            sdk_name = LIFECYCLE_MAP[name]
            if sdk_name == "on_bar":
                self._emit(f"def {sdk_name}(self, timeframe: str) -> None:")
            elif sdk_name == "on_deinit":
                self._emit(f"def {sdk_name}(self, reason: str = 'user_stop') -> None:")
            else:
                self._emit(f"def {sdk_name}(self) -> None:")
            self._indent += 1  # function body indent
            self._inside_function = True
            self._stats.patterns_matched += 1

            # For on_init: call _init_params() first, then user's OnInit body.
            all_params = self._extern_found + self._params_found
            if sdk_name == "on_init" and all_params:
                self._emit("self._init_params()")
        else:
            self._emit_gap(f"Unknown function: {name}", line)
            self._emit(f"def {name.lower()}(self) -> None:")
            self._indent += 1  # function body indent
            self._inside_function = True

    def _emit_variable(self, match) -> None:
        vtype = match.group("type")
        vname = match.group("name")
        vvalue = match.group("value").strip().rstrip(";")
        py_value = _mql_default_to_py(vvalue, vtype)
        # Global MQL variables → self. attributes (not class attributes).
        # Deferred to a _init_globals() method called from on_init.
        if not hasattr(self, '_global_vars'):
            self._global_vars = []
        self._global_vars.append((vname, py_value))
        # Emit as comment for audit trail.
        self._emit(f"# global {vtype} {vname} → self.{vname} = {py_value}")

    def _emit_extern_comment(self, line: str) -> None:
        m = EXTERN_RE.match(line.strip())
        if m:
            self._emit(f"# extern {m.group('name')} (type={m.group('type')}) = {m.group('value')}")
            self._emit(f"#   → in on_init: self.{m.group('name')} = self.ctx.param('{m.group('name')}', {m.group('value')})")
        else:
            self._emit(f"// TRANSPILER-GAP: extern parse failed: {line.strip()}")

    def _emit_input_comment(self, line: str) -> None:
        m = INPUT_RE.match(line.strip())
        if m:
            self._emit(f"# input {m.group('name')} (type={m.group('type')}) = {m.group('value')}")
            self._emit(f"#   → in on_init: self.{m.group('name')} = self.ctx.param('{m.group('name')}', {m.group('value')})")
        else:
            self._emit(f"// TRANSPILER-GAP: input parse failed: {line.strip()}")

    def _emit_statement(self, stripped: str) -> None:
        """Process a single MQL statement into Python."""
        # Strip MQL type prefix (e.g. "int ticket = OrderSend(...)")
        # so downstream handlers see clean Python-compatible code.
        stripped = _strip_mql_type_prefix(stripped)
        if self._try_control_flow(stripped):
            return
        if self._try_order_operations(stripped):
            return
        if self._try_indicator_call(stripped):
            return
        if self._try_common_function(stripped):
            return
        if self._try_order_select_loop(stripped):
            return
        if self._try_return_statement(stripped):
            return
        if self._try_simple_assignment(stripped):
            return
        self._emit_gap("Unrecognized statement", stripped)

    # MQL built-in function translations for variable references.
    _MQL_BUILTIN_REFS = {
        "Symbol()": "self.ctx.symbol",
        "Ask": "self.ctx.ask", "Bid": "self.ctx.bid",
        "Point": "self.ctx.point",
        "Digits": "self.sym_info.digits",
        "AccountBalance()": "self.broker.account().balance",
        "AccountEquity()": "self.broker.account().equity",
        "AccountFreeMargin()": "self.broker.account().free_margin",
        "OP_BUY": "OrderType.BUY", "OP_SELL": "OrderType.SELL",
        "OP_BUYLIMIT": "OrderType.BUY_LIMIT", "OP_SELLLIMIT": "OrderType.SELL_LIMIT",
        "OP_BUYSTOP": "OrderType.BUY_STOP", "OP_SELLSTOP": "OrderType.SELL_STOP",
        "SELECT_BY_POS": "", "SELECT_BY_TICKET": "",
        "MODE_TRADES": "", "MODE_HISTORY": "",
        "INIT_SUCCEEDED": "0", "clrNONE": "None",
        # OrderSelect accessors → order attribute
        "OrderMagicNumber()": "order.magic",
        "OrderTicket()": "order.ticket",
        "OrderLots()": "order.volume",
        "OrderProfit()": "order.profit",
        "OrderSymbol()": "order.symbol",
        "OrderType()": "order.type",
        "OrderOpenPrice()": "order.open_price",
        "OrderClosePrice()": "order.close_price",
        "OrderStopLoss()": "order.sl",
        "OrderTakeProfit()": "order.tp",
        "OrderComment()": "order.comment",
        "OrderCommission()": "order.commission",
        "OrderSwap()": "order.swap",
    }

    def _try_simple_assignment(self, line: str) -> bool:
        """Match simple MQL assignment: var = expr;"""
        m = re.match(r"(\w[\w.]*)\s*=\s*(.+)", line)
        if not m: return False
        lhs, rhs = m.group(1), m.group(2)
        # Translate RHS builtin references.
        py_rhs = rhs
        for mql_ref, py_ref in self._MQL_BUILTIN_REFS.items():
            if py_ref == "": continue  # skip empty mappings
            py_rhs = py_rhs.replace(mql_ref, py_ref)
        # Self-reference: if lhs is a known variable, prefix with self.
        self._emit(f"self.{lhs} = {py_rhs}")
        self._stats.patterns_matched += 1
        return True

    def _try_order_operations(self, line: str) -> bool:
        """Match OrderSend, OrderClose, OrderModify, OrderDelete."""
        if "OrderSend(" in line:
            return self._map_ordersend_v2(line)
        if "OrderClose(" in line:
            return self._map_orderclose_v2(line)
        if "OrderModify(" in line:
            return self._map_ordermodify_v2(line)
        if "OrderDelete(" in line:
            return self._map_orderdelete_v2(line)
        return False

    def _try_indicator_call(self, line: str) -> bool:
        """Match iMA, iRSI, iBands, etc. using positional arg extraction."""
        indicator_map = {
            "iMA": ("ma", [None, None, "period", "shift", "method"]),
            "iRSI": ("rsi", [None, None, "period", None, "shift"]),
            "iBands": ("bands", [None, None, "period", "deviation", None, None, None, "shift"]),
            "iMACD": ("macd", [None, None, "fast", "slow", "signal", None, None, "shift"]),
            "iATR": ("atr", [None, None, "period", "shift"]),
            "iStochastic": ("stochastic", [None, None, "k_period", "d_period", None, None, None, "shift"]),
            "iCCI": ("cci", [None, None, "period", None, "shift"]),
            "iCustom": ("i_custom", [None, None, "name", "params_raw", None, "buffer", "shift"]),
            "iADX": ("adx", [None, None, "period", None, "shift"]),
            "iMomentum": ("momentum", [None, None, "period", None, None, "shift"]),
            "iMFI": ("mfi", [None, None, "period", None, "shift"]),
            "iOBV": ("obv", [None, None, None, None, "shift"]),
            "iSAR": ("sar", [None, None, "step", "maximum", "shift"]),
            "iStdDev": ("stddev", [None, None, "period", None, None, None, "shift"]),
            "iWPR": ("wpr", [None, None, "period", None, "shift"]),
            "iEnvelopes": ("envelopes", [None, None, "period", None, None, None, "deviation", "shift"]),
            "iAlligator": ("alligator", [None, None, "jaw_period", "jaw_shift", "teeth_period", "teeth_shift", "lips_period", "lips_shift", None, None, "shift"]),
            "iForce": ("force", [None, None, "period", None, None, "shift"]),
            "iDeMarker": ("demarker", [None, None, "period", None, "shift"]),
            "iOsMA": ("osma", [None, None, "fast", "slow", "signal", None, None, "shift"]),
            "iBearsPower": ("bears_power", [None, None, "period", None, "shift"]),
            "iBullsPower": ("bulls_power", [None, None, "period", None, "shift"]),
            "iBWMFI": ("bw_mfi", [None, None, None, "shift"]),
            "iRVI": ("rvi", [None, None, "period", None, "shift"]),
            "iGator": ("gator", [None, None, "jaw_period", "jaw_shift", "teeth_period", "teeth_shift", "lips_period", "lips_shift", None, None, "shift"]),
            "iIchimoku": ("ichimoku", [None, None, "tenkan", "kijun", "senkou", "shift"]),
        }

        for mql_name, (sdk_name, arg_map) in indicator_map.items():
            prefix = mql_name + "("
            if prefix not in line:
                continue
            try:
                args = self._extract_args(line, mql_name)
            except (ValueError, IndexError):
                continue

            kwargs: List[str] = []
            for i, param_name in enumerate(arg_map):
                if param_name is None:
                    continue
                if i < len(args):
                    val = args[i].strip()
                else:
                    val = "0"
                if param_name == "method":
                    val = f"'{method_name(val)}'"
                elif param_name == "name":
                    val = val.strip('"')
                    val = f"'{val}'"
                elif param_name == "params_raw":
                    # iCustom params: comma-separated values after the name
                    raw_params = [a.strip() for a in args[3:]]
                    # Remove mode, buffer, shift from end
                    val = ", ".join(raw_params[:-2]) if len(raw_params) > 2 else ""
                    param_name = "params"
                    kwargs.append(f"{param_name}=[{val}]")
                    continue
                kwargs.append(f"{param_name}={val}")

            call_expr = f"self.indicators.{sdk_name}({', '.join(kwargs)})"
            # Capture result if there's a left-hand side assignment.
            lhs_match = re.match(r"(\w[\w.]*)\s*=\s*", line)
            if lhs_match:
                self._emit(f"self.{lhs_match.group(1)} = {call_expr}")
            else:
                self._emit(call_expr)
            self._stats.patterns_matched += 1
            return True

        return False

    def _try_common_function(self, line: str) -> bool:
        """Match common MQL built-in functions."""
        for mql_fn, py_fn in COMMON_FUNC_MAP.items():
            if py_fn.startswith("TRANSPILER-GAP"):
                if mql_fn + "(" in line:
                    self._emit_gap(py_fn[len("TRANSPILER-GAP: "):], line)
                    return True
                continue

            # Simple replacement: iClose(Symbol(), ...) → bars.close[...]
            if mql_fn in ("iOpen", "iHigh", "iLow", "iClose", "iVolume", "iTime"):
                pat = re.compile(rf"{mql_fn}\s*\(([^)]+)\)")
                m = pat.search(line)
                if m:
                    sdk_series = py_fn.replace("bars.", "")
                    self._emit(f"# i{mql_fn[1:]} access via {py_fn}")
                    self._stats.patterns_matched += 1
                    return True

            if mql_fn in ("Open", "High", "Low", "Close", "Volume", "Time"):
                pat = re.compile(rf"\b{mql_fn}\s*\[\s*(\d+)\s*\]")
                m = pat.search(line)
                if m:
                    shift = m.group(1)
                    self._emit(f"bars.{mql_fn.lower()}[{shift}]")
                    self._stats.patterns_matched += 1
                    return True

        # Generic function replacements.
        for mql_fn, py_fn in COMMON_FUNC_MAP.items():
            if py_fn.startswith("TRANSPILER-GAP") or py_fn.startswith("lambda"):
                continue
            if mql_fn + "(" in line:
                line = line.replace(mql_fn + "(", py_fn + "(")
                self._emit(line)
                self._stats.patterns_matched += 1
                return True

        return False

    def _try_order_select_loop(self, line: str) -> bool:
        """Match OrderSelect / PositionsTotal patterns."""
        # OrdersTotal() → len(self.broker.orders())
        if "OrdersTotal()" in line:
            self._emit(f"# for loop over orders →")
            self._emit(f"for order in self.broker.orders():")
            self._inside_orderselect_loop = True
            self._order_loop_var = "order"
            self._stats.patterns_matched += 1
            return True

        # PositionsTotal() → len(self.broker.positions())
        if "PositionsTotal()" in line:
            self._emit(f"# for loop over positions →")
            self._emit(f"for pos in self.broker.positions():")
            self._inside_orderselect_loop = True
            self._order_loop_var = "pos"
            self._stats.patterns_matched += 1
            return True

        # OrderSelect(...)
        if "OrderSelect(" in line:
            self._emit(f"# OrderSelect → using {self._order_loop_var} from broker.orders() iteration")
            self._stats.patterns_matched += 1
            return True

        # Order accessor mappings.
        for mql_acc, py_acc in ORDER_ACCESSOR_MAP.items():
            if mql_acc + "(" in line or mql_acc + "()" in line:
                self._emit(f"{self._order_loop_var}_{mql_acc.lower()} = {py_acc}")
                self._stats.patterns_matched += 1
                return True

        # Position accessor mappings.
        for mql_acc, py_acc in POSITION_ACCESSOR_MAP.items():
            if mql_acc + "(" in line or mql_acc + "()" in line:
                self._emit(f"{self._order_loop_var}_{mql_acc.lower()} = {py_acc}")
                self._stats.patterns_matched += 1
                return True

        return False

    def _try_control_flow(self, line: str) -> bool:
        """Match for, while, if statements."""
        stripped = line.strip()

        # for (int i=0; i<OrdersTotal(); i++) → order iteration
        if stripped.startswith("for(") or stripped.startswith("for ("):
            body = _match_control_flow(stripped, "for")
            if body is not None:
                if "OrdersTotal()" in body:
                    self._emit("# OrderSelect loop → Python iteration")
                    self._emit("for order in self.broker.orders():")
                    self._inside_orderselect_loop = True
                    self._order_loop_var = "order"
                    self._indent += 1
                    self._stats.patterns_matched += 1
                    return True
                elif "PositionsTotal()" in body:
                    self._emit("# PositionSelect loop → Python iteration")
                    self._emit("for pos in self.broker.positions():")
                    self._inside_orderselect_loop = True
                    self._order_loop_var = "pos"
                    self._indent += 1
                    self._stats.patterns_matched += 1
                    return True
                else:
                    self._emit_gap(f"For loop (manual conversion needed): {body}", line)
                    return True

        # while (...)
        if stripped.startswith("while(") or stripped.startswith("while ("):
            cond = _match_control_flow(stripped, "while")
            if cond is not None:
                py_cond = self._mql_to_py_condition(cond)
                self._emit(f"while {py_cond}:")
                self._indent += 1
                self._stats.patterns_matched += 1
                return True

        # if (...)
        if stripped.startswith("if(") or stripped.startswith("if ("):
            cond = _match_control_flow(stripped, "if")
            if cond is not None:
                py_cond = self._mql_to_py_condition(cond)
                py_cond = re.sub(r"OrderSelect\([^)]+\)", "True  # OrderSelect check", py_cond)
                if "iMA(" in py_cond or "iRSI(" in py_cond or "iCustom(" in py_cond:
                    py_cond += "  # TRANSPILER-GAP: indicator in condition"
                self._emit(f"if {py_cond}:")
                self._indent += 1
                self._stats.patterns_matched += 1
                return True

        # else if → elif
        m_elif = re.match(r"else\s+if\s*\((.+)\)", stripped)
        if m_elif:
            py_cond = self._mql_to_py_condition(m_elif.group(1))
            self._emit(f"elif {py_cond}:")
            self._stats.patterns_matched += 1
            return True

        # else
        if stripped == "else" or stripped.startswith("else "):
            self._emit("else:")
            return True

        # Closing brace → dedent.
        if stripped in ("}", "};"):
            self._indent = max(0, self._indent - 1)
            self._inside_orderselect_loop = False
            return True

        return False

    def _try_return_statement(self, line: str) -> bool:
        if line.startswith("return ") or line == "return" or line == "return;":
            rest = line[7:].strip().rstrip(";") if line.startswith("return ") else ""
            for mql_c, py_c in _MQL_CONSTANTS.items():
                rest = rest.replace(mql_c, py_c)
            if rest:
                self._emit(f"return {rest}")
            else:
                self._emit("return")
            self._stats.patterns_matched += 1
            return True
        return False

    # ── Complex mappers ─────────────────────────────────────────────────

    def _map_ordersend(self, m, line: str) -> bool:
        """Map OrderSend to broker.order_send with OrderRequest."""
        symbol = m.group("symbol").strip()
        cmd = m.group("cmd").strip()
        volume = m.group("volume").strip()
        price = m.group("price").strip()
        slippage = m.group("slippage").strip()
        sl_str = m.group("sl").strip()
        tp_str = m.group("tp").strip()
        comment = (m.group("comment") or "").strip()
        magic = (m.group("magic") or "0").strip()
        # expiration = (m.group("expiration") or "0").strip()

        sdk_cmd = CMD_MAP.get(cmd, cmd)
        sl_val = "Decimal(str(" + sl_str + "))" if sl_str != "0" else "None"
        tp_val = "Decimal(str(" + tp_str + "))" if tp_str != "0" else "None"

        # Detect assignment: ticket = OrderSend(...)
        ticket_var = ""
        assign_match = _ORDERSEND_TICKET_RE.search(line)
        if assign_match:
            ticket_var = assign_match.group(1) + " = "

        self._emit(f"{ticket_var}self.broker.order_send(OrderRequest(")
        self._emit(f"    symbol={symbol},", 1)
        self._emit(f"    type={sdk_cmd},")
        self._emit(f"    volume=Decimal(str({volume})),")
        self._emit(f"    price=Decimal(str({price})),")
        self._emit(f"    sl={sl_val},")
        self._emit(f"    tp={tp_val},")
        self._emit(f"    deviation=int({slippage}),")
        self._emit(f"    magic=int({magic}),")
        self._emit(f'    comment="{comment}",' if comment else f'    comment="",')
        self._emit(f"))", -1)
        self._stats.patterns_matched += 1
        return True

    def _map_orderclose(self, m, line: str) -> bool:
        ticket = m.group("ticket").strip()
        volume = m.group("volume").strip()
        # price = m.group("price").strip()
        vol_arg = ""
        if volume not in ("0", "OrderLots()"):
            vol_arg = f", volume=Decimal(str({volume}))"
        self._emit(f"self.broker.position_close({ticket}{vol_arg})")
        self._stats.patterns_matched += 1
        return True

    def _map_ordermodify(self, m, line: str) -> bool:
        ticket = m.group("ticket").strip()
        sl = m.group("sl").strip()
        tp = m.group("tp").strip()
        sl_val = f"Decimal(str({sl}))" if sl != "0" else "None"
        tp_val = f"Decimal(str({tp}))" if tp != "0" else "None"
        self._emit(f"self.broker.position_modify({ticket}, sl={sl_val}, tp={tp_val})")
        self._stats.patterns_matched += 1
        return True

    # ── Simplified v2 mappers (handle variable expressions) ────────────

    @staticmethod
    def _extract_args(line: str, fn_name: str) -> List[str]:
        """Extract comma-separated arguments from a function call like fn_name(a, b, c)."""
        start = line.index(fn_name + "(") + len(fn_name) + 1
        depth = 1
        args: List[str] = []
        current: List[str] = []
        i = start
        while i < len(line) and depth > 0:
            ch = line[i]
            if ch == "(":
                depth += 1
                current.append(ch)
            elif ch == ")":
                depth -= 1
                if depth == 0:
                    args.append("".join(current).strip())
                    break
                current.append(ch)
            elif ch == "," and depth == 1:
                args.append("".join(current).strip())
                current = []
            else:
                current.append(ch)
            i += 1
        return args

    def _map_ordersend_v2(self, line: str) -> bool:
        """Map OrderSend using positional arg extraction."""
        try:
            args = self._extract_args(line, "OrderSend")
        except (ValueError, IndexError):
            self._emit_gap("OrderSend parse failed", line)
            return True

        # Assign result to a variable if present.
        ticket_var = ""
        assign_part = line.split("OrderSend")[0].strip()
        if assign_part.endswith("="):
            ticket_var = assign_part.rstrip("= ").strip()

        if len(args) < 8:
            self._emit_gap(f"OrderSend: expected >=8 args, got {len(args)}", line)
            return True

        symbol = args[0].strip()
        cmd = args[1].strip()
        volume = args[2].strip()
        price = args[3].strip()
        slippage = args[4].strip()
        sl_val = args[5].strip()
        tp_val = args[6].strip()
        comment = args[7].strip() if len(args) > 7 else '""'
        magic = args[8].strip() if len(args) > 8 else "0"

        sdk_cmd = CMD_MAP.get(cmd, cmd)
        sl_py = "None" if sl_val in ("0", "0.0") else f"Decimal(str({sl_val}))"
        tp_py = "None" if tp_val in ("0", "0.0") else f"Decimal(str({tp_val}))"
        comment_py = comment if comment.startswith('"') else f"str({comment})"

        prefix = f"{ticket_var} = " if ticket_var else ""
        self._emit(f"{prefix}self.broker.order_send(OrderRequest(")
        self._indent += 1
        self._emit(f"symbol={symbol},")
        self._emit(f"type={sdk_cmd},")
        self._emit(f"volume=Decimal(str({volume})),")
        self._emit(f"price=Decimal(str({price})),")
        self._emit(f"sl={sl_py},")
        self._emit(f"tp={tp_py},")
        self._emit(f"deviation=int({slippage}),")
        self._emit(f"magic=int({magic}),")
        self._emit(f"comment={comment_py},")
        self._emit(f"))")
        self._indent -= 1
        self._stats.patterns_matched += 1
        return True

    def _map_orderclose_v2(self, line: str) -> bool:
        try:
            args = self._extract_args(line, "OrderClose")
        except (ValueError, IndexError):
            self._emit_gap("OrderClose parse failed", line)
            return True
        ticket = args[0].strip()
        volume = args[1].strip() if len(args) > 1 else "OrderLots()"
        vol_arg = f", volume=Decimal(str({volume}))" if volume not in ("0",) else ""
        self._emit(f"self.broker.position_close({ticket}{vol_arg})")
        self._stats.patterns_matched += 1
        return True

    def _map_ordermodify_v2(self, line: str) -> bool:
        try:
            args = self._extract_args(line, "OrderModify")
        except (ValueError, IndexError):
            self._emit_gap("OrderModify parse failed", line)
            return True
        ticket = args[0].strip()
        # price = args[1]; sl = args[2]; tp = args[3]
        sl = args[2].strip() if len(args) > 2 else "0"
        tp = args[3].strip() if len(args) > 3 else "0"
        sl_py = "None" if sl in ("0", "0.0") else f"Decimal(str({sl}))"
        tp_py = "None" if tp in ("0", "0.0") else f"Decimal(str({tp}))"
        self._emit(f"self.broker.position_modify({ticket}, sl={sl_py}, tp={tp_py})")
        self._stats.patterns_matched += 1
        return True

    def _map_orderdelete_v2(self, line: str) -> bool:
        try:
            args = self._extract_args(line, "OrderDelete")
        except (ValueError, IndexError):
            self._emit_gap("OrderDelete parse failed", line)
            return True
        ticket = args[0].strip()
        self._emit(f"self.broker.order_delete({ticket})")
        self._stats.patterns_matched += 1
        return True

    def _mql_to_py_condition(self, condition: str) -> str:
        """Convert MQL condition expression to Python, including indicator calls."""
        return self._translate_expression(condition.strip())

    def _translate_expression(self, expr: str) -> str:
        """Recursively translate an MQL expression to Python.

        Handles indicator calls, OHLCV arrays, builtins, constants, operators.
        """
        result = expr.strip()

        # 1. OHLCV array access: Close[0] → bars.close[0]
        for mql_name in ("Open", "High", "Low", "Close", "Volume", "Time"):
            py_name = mql_name.lower()
            result = re.sub(
                rf"\b{mql_name}\s*\[",
                f"bars.{py_name}[",
                result,
            )

        # 2. Indicator calls in expressions: iMA(...) → self.indicators.ma(...)
        for mql_fn, sdk_fn in [("iMA", "ma"), ("iRSI", "rsi"), ("iATR", "atr"),
                                ("iCCI", "cci"), ("iBands", "bands"), ("iMACD", "macd"),
                                ("iStochastic", "stochastic"), ("iCustom", "i_custom")]:
            prefix = mql_fn + "("
            while prefix in result:
                idx = result.index(prefix)
                args_str = self._extract_parens(result, idx + len(prefix))
                args = [a.strip() for a in args_str.split(",")]
                if mql_fn == "iMA":
                    period = args[2].strip() if len(args) > 2 else "14"
                    shift = args[3].strip() if len(args) > 3 else "0"
                    method = method_name(args[4].strip()) if len(args) > 4 else "sma"
                    replacement = f"self.indicators.{sdk_fn}(period={period}, shift={shift}, method='{method}')"
                elif mql_fn == "iRSI":
                    period = args[2].strip() if len(args) > 2 else "14"
                    shift = args[4].strip() if len(args) > 4 else "0"
                    replacement = f"self.indicators.{sdk_fn}(period={period}, shift={shift})"
                elif mql_fn == "iATR":
                    period = args[2].strip() if len(args) > 2 else "14"
                    shift = args[3].strip() if len(args) > 3 else "0"
                    replacement = f"self.indicators.{sdk_fn}(period={period}, shift={shift})"
                elif mql_fn == "iCCI":
                    period = args[2].strip() if len(args) > 2 else "14"
                    shift = args[4].strip() if len(args) > 4 else "0"
                    replacement = f"self.indicators.{sdk_fn}(period={period}, shift={shift})"
                elif mql_fn == "iCustom":
                    name = args[2].strip().strip('"') if len(args) > 2 else "unknown"
                    buffer = args[-2].strip() if len(args) > 2 else "0"
                    shift = args[-1].strip() if len(args) > 2 else "0"
                    replacement = f"self.indicators.{sdk_fn}(name='{name}', params=[], buffer={buffer}, shift={shift})"
                else:
                    replacement = f"self.indicators.{sdk_fn}({', '.join(args[2:5]) if len(args) > 2 else ''})"
                result = result[:idx] + replacement + result[idx + len(prefix) + len(args_str) + 1:]
                break

        # 3. Common function replacements.
        for mql_fn, py_fn in COMMON_FUNC_MAP.items():
            if py_fn.startswith("TRANSPILER-GAP") or py_fn.startswith("lambda"):
                continue
            if mql_fn + "(" in result:
                result = result.replace(mql_fn + "(", py_fn + "(")

        # 4. Constants.
        for mql_c, py_c in _MQL_CONSTANTS.items():
            result = re.sub(rf"\b{re.escape(mql_c)}\b", py_c, result)

        # 5. Operators.
        for mql_op, py_op in _MQL_COMPARE.items():
            result = result.replace(mql_op, py_op)

        return result

    @staticmethod
    def _join_continuation_lines(raw_lines: List[str]) -> List[str]:
        """Join lines where a function call spans multiple lines.
        A continuation is detected when a line ends with ',' or '(' and
        the next line is indented or starts with an argument continuation.
        """
        joined: List[str] = []
        buf: List[str] = []
        for line in raw_lines:
            stripped = line.strip()
            if buf:
                buf.append(line)
                # Check if this line closes the pending function call.
                if ")" in stripped and not stripped.startswith("//"):
                    # Count parens to see if we've closed the outermost call.
                    full = " ".join(l.strip() for l in buf)
                    if full.count("(") <= full.count(")"):
                        joined.append(" ".join(l.strip() for l in buf))
                        buf = []
                continue
            # Start buffering if line contains a function call that isn't closed.
            if ("(" in stripped and ")" not in stripped.split("(")[-1]) or (
                stripped.rstrip().endswith(",")
            ):
                buf.append(line)
                continue
            joined.append(line)
        # Flush remaining.
        if buf:
            joined.append(" ".join(l.strip() for l in buf))
        return joined

    # ── Param discovery ─────────────────────────────────────────────────

    def _discover_params(self, line: str) -> None:
        m = EXTERN_RE.match(line.strip())
        if m:
            self._extern_found.append((m.group("name"), m.group("type"), m.group("value").strip()))
        m = INPUT_RE.match(line.strip())
        if m:
            self._params_found.append((m.group("name"), m.group("type"), m.group("value").strip()))

    # ── Body skipping ───────────────────────────────────────────────────

    def _process_function_body(self, lines: List[str], start: int) -> int:
        """Process lines inside a function body until the closing brace."""
        i = start
        if "{" not in lines[i]:
            i += 1

        brace_line = lines[i] if i < len(lines) else ""
        if "{" in brace_line:
            before, after_brace = brace_line.split("{", 1)
            if before.strip():
                # Process any statement before the brace (e.g. function return type already handled).
                pass
            i += 1
            if after_brace.strip() and not after_brace.strip().startswith("//"):
                # Check if the closing brace is also on this line.
                open_count = after_brace.count("{")
                close_count = after_brace.count("}")
                self._process_inline_block(after_brace)
                # If the inline block contained the function's closing brace, we're done.
                if close_count > open_count:
                    self._indent = max(0, self._indent - 1)
                    return i
        else:
            i += 1

        depth = 1
        while i < len(lines) and depth > 0:
            stripped = lines[i].strip()

            if not stripped or stripped.startswith("//") or stripped.startswith("/*"):
                if stripped.startswith("//"):
                    self._emit(stripped)
                i += 1
                continue

            # Handle braces in this line.
            has_close = "}" in stripped
            has_open = "{" in stripped

            if has_close:
                depth -= stripped.count("}")
                if depth <= 0:
                    self._indent = max(0, self._indent - 1)
                    return i + 1

            if has_open:
                depth += stripped.count("{")
                before = stripped.split("{", 1)[0].strip()
                # Strip leading closing brace from compound } else if / } else patterns
                before_clean = before.lstrip("}").strip()
                if before_clean:
                    self._emit_statement(before_clean)
                self._indent += 1
                after = stripped.split("{", 1)[1]
                if after.strip():
                    self._process_inline_block(after)
                i += 1
                continue

            # Simple statement.
            stripped_clean = stripped.rstrip(";")
            self._emit_statement(stripped_clean)
            i += 1

        self._indent = max(0, self._indent - 1)
        return i

    def _process_inline_block(self, text: str) -> None:
        """Process code that appears after an opening brace on the same line.
        May contain nested braces and multiple statements."""
        text = text.strip()
        while text:
            # Check for nested opening brace.
            if text.startswith("{"):
                self._indent += 1
                text = text[1:].strip()
                continue

            # Check for closing brace.
            if text.startswith("}"):
                self._indent = max(0, self._indent - 1)
                text = text[1:].strip()
                continue

            # Extract the next statement up to ; or { or }.
            end_semi = text.find(";")
            end_open = text.find("{")
            end_close = text.find("}")

            candidates = [i for i in [end_semi, end_open, end_close] if i >= 0]
            if not candidates:
                if text.strip():
                    self._emit_statement(text.strip())
                break

            end = min(candidates)
            if end == 0:
                if text[0] == "{":
                    self._indent += 1
                    text = text[1:].strip()
                    continue
                elif text[0] == "}":
                    self._indent = max(0, self._indent - 1)
                    text = text[1:].strip()
                    continue
                else:
                    text = text[1:].strip()
                    continue

            stmt = text[:end].strip()
            if stmt:
                self._emit_statement(stmt)
            text = text[end:].strip()
            if text.startswith(";"):
                text = text[1:].strip()

    # Legacy method — replaced by _process_function_body.
    def _skip_function_body(self, lines: List[str], start: int) -> int:
        """Skip from function start to after its closing brace.
        No longer used for transpilation — kept for reference."""
        i = start + 1
        depth = 0
        if "{" in lines[start]:
            depth += lines[start].count("{") - lines[start].count("}")

        while i < len(lines) and depth > 0:
            stripped = lines[i].strip()
            if stripped.startswith("//"):
                i += 1
                continue
            depth += stripped.count("{") - stripped.count("}")
            i += 1
        self._inside_function = False
        self._emit()
        return i
