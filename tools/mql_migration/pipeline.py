"""迁移流水线 — 整合 Layer 1/2/3。

Usage::

    from tools.mql_migration.pipeline import MigrationPipeline
    pipe = MigrationPipeline()
    intent = pipe.analyze(mql_source)
    code = pipe.generate(intent)
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import List, Optional

from tools.mql_transpiler.ast_bridge import parse_mql
from tools.mql_transpiler.ast_nodes import SourceFile

from tools.mql_migration.expression_gen import ExpressionGen
from tools.mql_migration.intent_ir import (
    BlindSpot,
    BlindSpotCategory,
    HandlingStrategy,
    Severity,
    StrategyIntent,
)
from tools.mql_migration.generators.python_gen import PythonCodeGenerator


@dataclass
class PipelineResult:
    """迁移流水线完整结果。"""
    intent: StrategyIntent
    python_code: str = ""
    coverage_score: float = 0.0
    errors: List[str] = field(default_factory=list)
    warnings: List[str] = field(default_factory=list)


class MigrationPipeline:
    """完整的 MQL → Python 迁移流水线。"""

    def __init__(self):
        self._expr_gen = ExpressionGen()
        self._code_gen = PythonCodeGenerator()

    def analyze(self, source: str, source_name: str = "") -> StrategyIntent:
        """分析 MQL 源码，提取意图 IR。

        Args:
            source: MQL4/MQL5 源码文本
            source_name: 源文件名（用于推断策略名）

        Returns:
            提取的 StrategyIntent
        """
        ast = parse_mql(source)
        # Normalize AST before recognition — same intent, different code styles
        from tools.mql_migration.ast_normalizer import normalize
        normalize(ast)
        return self._analyze_ast(ast, source_name)

    def generate(self, intent: StrategyIntent) -> str:
        """从意图 IR 生成 Python SDK 代码。

        Args:
            intent: StrategyIntent from analyze()

        Returns:
            Python 策略源代码
        """
        return self._code_gen.generate(intent)

    def run(self, source: str, source_name: str = "") -> PipelineResult:
        """完整流水线: MQL → Intent IR → Python 代码。

        Args:
            source: MQL4/MQL5 源码文本
            source_name: 源文件名

        Returns:
            PipelineResult with intent and generated code
        """
        intent = self.analyze(source, source_name)
        code = self.generate(intent)
        return PipelineResult(
            intent=intent,
            python_code=code,
            coverage_score=intent.coverage_score,
        )

    def _analyze_ast(self, ast: SourceFile, source_name: str = "") -> StrategyIntent:
        """从 AST 提取完整策略意图。"""
        from tools.mql_migration.intent_ir import ParamSpec, ParamType, ParamGroup, SizingKind
        from tools.mql_migration.recognizers.exec_model import (
            recognize_execution_model,
            recognize_meta,
            recognize_params,
            recognize_sizing,
            recognize_state_vars,
            recognize_timer,
        )
        from tools.mql_migration.recognizers.market_entry import recognize_market_entries
        from tools.mql_migration.recognizers.magic_exit import recognize_exit_rules
        from tools.mql_migration.recognizers.pending_entry import recognize_pending_entries
        from tools.mql_migration.recognizers.custom_entry import recognize_custom_entries
        from tools.mql_migration.recognizers.margin_check import recognize_margin_checks

        # ---- Actual recognition tracking (only count real hits) ----
        recognized = 0
        total = 0
        blocks: dict[str, int] = {}  # block_name → count of items found

        meta = recognize_meta(ast, source_name)
        blocks["meta"] = 1

        params = recognize_params(ast)
        state = recognize_state_vars(ast)
        has_lot = any("lot" in p.name.lower() for p in params)
        if not has_lot:
            params.insert(0, ParamSpec("LotSize", "交易手数", ParamType.DOUBLE, 0.10,
                          group=ParamGroup.SIZING))
        blocks["params"] = len(params)
        blocks["state"] = len(state)

        execution = recognize_execution_model(ast)
        blocks["execution"] = 1

        sizing = recognize_sizing(ast)
        blocks["sizing"] = 1

        if sizing and sizing.kind == SizingKind.MARTINGALE:
            martingale_params = [
                ParamSpec("MartingaleMultiplier", "马丁倍数", ParamType.DOUBLE, 2.0,
                          group=ParamGroup.SIZING),
                ParamSpec("MaxLot", "最大手数", ParamType.DOUBLE, 5.0,
                          group=ParamGroup.SIZING),
            ]
            existing_names = {p.name for p in params}
            for mp in martingale_params:
                if mp.name not in existing_names:
                    params.append(mp)

        timer = recognize_timer(ast)
        blocks["timer"] = 1 if timer else 0

        indicators = self._extract_indicators(ast)
        blocks["indicators"] = len(indicators)

        market_entries = recognize_market_entries(ast, self._expr_gen)
        pending_entries = recognize_pending_entries(ast, self._expr_gen)
        custom_entries = recognize_custom_entries(ast, self._expr_gen)
        all_entries = market_entries + pending_entries + custom_entries
        blocks["entries"] = len(all_entries)

        exit_rules = recognize_exit_rules(ast, self._expr_gen)
        blocks["exits"] = len(exit_rules)

        risk = recognize_margin_checks(ast, self._expr_gen) or self._recognize_risk(ast)
        blocks["risk"] = len(risk)

        if risk:
            risk_params = [
                ParamSpec("MarginThreshold", "保证金水平阈值(%)", ParamType.DOUBLE, 50.0,
                          group=ParamGroup.RISK),
            ]
            existing_names = {p.name for p in params}
            for rp in risk_params:
                if rp.name not in existing_names:
                    params.append(rp)

        blind_spots = self._detect_blind_spots(ast, all_entries, exit_rules)

        # Coverage: only count TRADING categories. GUI noise is excluded.
        gui_noise_count = sum(1 for bs in blind_spots if bs.severity == Severity.INFO)
        real_blind_count = len(blind_spots) - gui_noise_count

        category_weights = {"meta": 1, "params": 1, "state": 1, "execution": 1,
                          "sizing": 1, "timer": 1, "entries": 2, "exits": 2,
                          "risk": 1, "indicators": 1}
        for cat, weight in category_weights.items():
            total += weight
            if blocks.get(cat, 0) > 0:
                recognized += weight
        coverage = recognized / max(total, 1)

        return StrategyIntent(
            meta=meta,
            params=params,
            state=state,
            entry=all_entries,
            exit=exit_rules,
            indicators=indicators,
            sizing=sizing,
            risk=risk,
            execution=execution,
            timer=timer,
            blind_spots=blind_spots,
            coverage_score=coverage,
            total_blocks=total,
            recognized_blocks=recognized,
        )

    def _extract_indicators(self, ast: SourceFile) -> list:
        """Extract all indicator calls from the AST as IndicatorSpec."""
        from tools.mql_migration.intent_ir import IndicatorSpec
        from tools.mql_migration.recognizers.base import ast_contains_call, find_functions, find_calls
        from tools.mql_transpiler.ast_nodes import (
            AssignmentExpr, CallExpr, CompoundStmt, ExpressionStmt, VarDecl,
        )

        indicator_names = {"iMA", "iRSI", "iATR", "iBands", "iMACD", "iStochastic",
                          "iCCI", "iADX", "iMomentum", "iMFI", "iOBV", "iSAR",
                          "iStdDev", "iWPR", "iEnvelopes", "iForce", "iDeMarker",
                          "iOsMA", "iCustom"}

        specs = []
        seen = set()

        for func in find_functions(ast):
            if not func.body:
                continue

            # Find all indicator calls
            calls = []
            for ind_name in indicator_names:
                calls.extend(find_calls(func.body, ind_name))

            for call in calls:
                # Deduplicate by name + args
                arg_sig = ",".join(str(a) for a in (call.args or []))
                key = f"{call.name}:{arg_sig}"
                if key in seen:
                    continue
                seen.add(key)

                # Determine result variable name
                result_var = ""
                source_expr = call

                # Check if this call is inside a VarDecl
                if isinstance(func.body, CompoundStmt):
                    for stmt in func.body.statements:
                        if isinstance(stmt, VarDecl) and stmt.value is call:
                            result_var = stmt.name
                            break
                        elif isinstance(stmt, ExpressionStmt) and isinstance(stmt.expr, AssignmentExpr):
                            if stmt.expr.rhs is call:
                                result_var = stmt.expr.lhs
                                break

                # Map to SDK method
                method_map = {
                    "iMA": "ma", "iRSI": "rsi", "iATR": "atr", "iBands": "bands",
                    "iMACD": "macd", "iStochastic": "stochastic", "iCCI": "cci",
                    "iADX": "adx", "iMomentum": "momentum", "iMFI": "mfi",
                    "iOBV": "obv", "iSAR": "sar", "iStdDev": "stddev",
                    "iWPR": "wpr", "iEnvelopes": "envelopes", "iForce": "force",
                    "iDeMarker": "demarker", "iOsMA": "osma", "iCustom": "i_custom",
                }
                sdk_method = method_map.get(call.name, call.name.lower())

                specs.append(IndicatorSpec(
                    sdk_method=sdk_method,
                    result_var=result_var,
                    comment=f"{call.name}(...) in {func.name}",
                ))

        return specs

    def _recognize_risk(self, ast: SourceFile) -> list:
        """Recognize risk checks (margin checks via OnTimer).

        Translates the actual MQL condition from the OnTimer body.
        Extracts the threshold as a ParamSpec so it's user-configurable.
        """
        from tools.mql_migration.intent_ir import RiskCheck
        from tools.mql_migration.recognizers.base import ast_contains_call, find_calls, find_functions
        from tools.mql_transpiler.ast_nodes import IfStmt, CompoundStmt

        risk = []
        for func in find_functions(ast):
            if func.name != "OnTimer" or not func.body:
                continue

            has_margin = bool(
                find_calls(func.body, "AccountFreeMargin") or
                find_calls(func.body, "AccountBalance")
            )
            if not has_margin:
                continue

            # Translate the actual MQL condition from the if-statement
            condition_expr = ""
            if isinstance(func.body, CompoundStmt):
                for stmt in (func.body.statements or []):
                    if isinstance(stmt, IfStmt):
                        cond = stmt.condition
                        if (ast_contains_call(cond, "AccountFreeMargin") or
                            ast_contains_call(cond, "AccountBalance")):
                            # Register local vars for this function
                            from tools.mql_migration.recognizers.base import extract_local_vars
                            locals_ = extract_local_vars(func)
                            prev = set(self._expr_gen._local_vars)
                            self._expr_gen._local_vars |= locals_
                            condition_expr = self._expr_gen.translate(cond)
                            self._expr_gen._local_vars = prev
                            break

            # Replace hardcoded numeric ratio with ctx.param if present
            # e.g. "balance * 0.5" → "balance * Decimal(str(self.ctx.param('RiskMarginRatio', 0.5)))"
            import re
            if condition_expr:
                # Replace trailing numeric multipliers with param reference
                condition_expr = re.sub(
                    r'\*\s*([0-9]+\.?[0-9]*)\s*$',
                    r'* Decimal(str(self.ctx.param("RiskMarginRatio", \1)))',
                    condition_expr
                )

            # Fallback: use a generic param-based condition
            if not condition_expr:
                condition_expr = (
                    "self.broker.account().free_margin < "
                    "self.broker.account().balance "
                    '* Decimal(str(self.ctx.param("RiskMarginRatio", 0.5)))'
                )

            risk.append(RiskCheck(
                kind="margin_check",
                condition=condition_expr,
                action="close_all",
                trigger="on_timer",
            ))

        return risk

    def _detect_blind_spots(self, ast, entries, exits) -> List[BlindSpot]:
        """Detect areas not covered by recognizers."""
        from tools.mql_migration.recognizers.base import find_calls, find_functions
        from tools.mql_migration.blind_spot import compute_fingerprint

        spots = []

        # ── GUI noise — silently skipped, not counted as blind spots ──
        _GUI_NOISE = {"ObjectCreate", "ObjectDelete", "ObjectSetDouble", "ObjectSetInteger",
                      "ObjectSetString", "ObjectGetDouble", "ObjectGetInteger", "ObjectGetString",
                      "ObjectsDeleteAll", "ObjectFind", "ObjectSet", "ObjectSetText",
                      "ObjectGet", "ObjectSetInteger", "ObjectsTotal", "ObjectName",
                      "ChartSetInteger", "ChartOpen", "ChartClose", "Comment",
                      "ButtonCreate", "RectLabelCreate"}
        for func in find_functions(ast):
            for call in find_calls(func.body if func.body else [], *_GUI_NOISE):
                fp = compute_fingerprint(call, {call.name}, "gui_noise")
                spots.append(BlindSpot(
                    id=fp.pattern_signature,
                    location=f"{func.name}",
                    category=BlindSpotCategory.UNSUPPORTED_API,
                    severity=Severity.INFO,
                    description=f"已跳过（图表显示功能）: {call.name}",
                    handling=HandlingStrategy.SKIP_WITH_COMMENT,
                    user_action_required=False,
                ))

        # ── Genuinely unsupported — hard boundary ──
        _HARD_UNSUPPORTED = {"FileOpen", "FileWrite", "WebRequest",
                             "SendFTP", "SendMail", "SendNotification"}
        for func in find_functions(ast):
            for call in find_calls(func.body if func.body else [], *_HARD_UNSUPPORTED):
                fp = compute_fingerprint(call, {call.name}, "unsupported_api")
                spots.append(BlindSpot(
                    id=fp.pattern_signature,
                    location=f"{func.name}",
                    category=BlindSpotCategory.UNSUPPORTED_API,
                    severity=Severity.WARNING,
                    description=f"不支持的API调用: {call.name}",
                    handling=HandlingStrategy.HARD_BLOCK,
                    user_action_required=True,
                ))

        # Check if any lifecycle function body is empty
        main_funcs = find_functions(ast, "OnTick", "OnCalculate", "start")
        if not main_funcs:
            spots.append(BlindSpot(
                location="<top-level>",
                category=BlindSpotCategory.CUSTOM_LOGIC,
                severity=Severity.CRITICAL,
                description="未找到 OnTick/OnCalculate/start 入口函数",
                handling=HandlingStrategy.HARD_BLOCK,
                user_action_required=True,
            ))

        return spots
