"""Strategy import ConnectRPC handlers — MQL → Python pipeline.

Protocol: ant.v1.PythonStrategyService/{AnalyzeImportCode,GenerateImportCode,ImportStrategy}
"""

from __future__ import annotations

import logging
from dataclasses import dataclass, field
from typing import Any, List, Optional

from tools.mql_migration.pipeline import MigrationPipeline
from tools.mql_migration.generators.param_schema_gen import generate_panel_schema

logger = logging.getLogger(__name__)

_pipe: Optional[MigrationPipeline] = None


def _get_pipe() -> MigrationPipeline:
    global _pipe
    if _pipe is None:
        _pipe = MigrationPipeline()
    return _pipe


def _to_json(obj: Any) -> str:
    """Serialize dataclass or dict to JSON string for proto string fields."""
    import json as _json
    import dataclasses as _dc

    def _convert(o):
        if _dc.is_dataclass(o):
            return {f.name: _convert(getattr(o, f.name)) for f in _dc.fields(o)}
        if isinstance(o, dict):
            return {k: _convert(v) for k, v in o.items()}
        if isinstance(o, list):
            return [_convert(v) for v in o]
        return o

    return _json.dumps(_convert(obj), ensure_ascii=False)


# ── Handlers ──────────────────────────────────────────────────────────


async def _analyze_code(req, _ctx):
    """MQL 源码 → 策略意图摘要 + 参数面板 + 盲区报告。"""
    from app.python_strategy_pb2 import AnalyzeImportCodeResponse

    pipe = _get_pipe()
    source = getattr(req, 'source_code', '') or getattr(req, 'sourceCode', '')
    name = getattr(req, 'source_name', '') or getattr(req, 'sourceName', 'untitled.mq4')

    if not source:
        return AnalyzeImportCodeResponse()

    try:
        result = pipe.run(source, source_name=name)
    except Exception as e:
        logger.warning("AnalyzeCode failed: %s", e)
        return AnalyzeImportCodeResponse()

    intent = result.intent

    # Param schema
    schema = generate_panel_schema(intent.params, intent.meta.name)
    params_list = [
        dict(name=f.name, label=f.label, type=f.field_type,
             default=str(f.default) if f.default is not None else "",
             group=f.group, groupOrder=f.group_order)
        for f in schema.fields
    ]

    blind_spot_list = [
        dict(id=bs.id, location=bs.location,
             category=bs.category.value if hasattr(bs.category, 'value') else str(bs.category),
             severity=bs.severity.value if hasattr(bs.severity, 'value') else str(bs.severity),
             description=bs.description,
             handling=bs.handling.value if hasattr(bs.handling, 'value') else str(bs.handling),
             userActionRequired=bs.user_action_required)
        for bs in intent.blind_spots
    ]

    return AnalyzeImportCodeResponse(
        strategy_name=intent.meta.name or "ImportedStrategy",
        mql_version=intent.meta.mql_version.value,
        coverage_score=intent.coverage_score,
        total_blocks=intent.total_blocks,
        recognized_blocks=intent.recognized_blocks,
        execution_kind=intent.execution.kind.value,
        entry_rules_count=len(intent.entry),
        exit_rules_count=len(intent.exit),
        sizing_kind=intent.sizing.kind.value if intent.sizing else "fixed",
        risk_checks_count=len(intent.risk),
        params_json=_to_json(params_list),
        groups_json=_to_json(schema.groups),
        blind_spots_json=_to_json(blind_spot_list),
        indicator_names=[ind.sdk_method for ind in intent.indicators],
    )


async def _generate_code(req, _ctx):
    """生成 Python 策略代码。"""
    from app.python_strategy_pb2 import GenerateImportCodeResponse

    pipe = _get_pipe()
    source = getattr(req, 'source_code', '') or getattr(req, 'sourceCode', '')
    name = getattr(req, 'source_name', '') or getattr(req, 'sourceName', 'untitled.mq4')

    if not source:
        return GenerateImportCodeResponse(quality_gate_failures=["源码为空"])

    try:
        result = pipe.run(source, source_name=name)
    except Exception as e:
        logger.warning("GenerateCode failed: %s", e)
        return GenerateImportCodeResponse(quality_gate_failures=[f"生成失败: {e}"])

    code = result.python_code
    failures = []

    import ast as py_ast
    try:
        py_ast.parse(code)
    except SyntaxError as e:
        failures.append(f"SyntaxError L{e.lineno}: {e.msg}")

    from tools.mql_transpiler.quality_gate import QualityGate, QualityVerdict
    gate = QualityGate.assess(code)
    if gate.verdict != QualityVerdict.HIGH:
        for f in gate.failures:
            failures.append(f"[{f.gate}] {f.message}")

    return GenerateImportCodeResponse(
        python_code=code,
        code_lines=len(code.splitlines()),
        compiles=len(failures) == 0,
        quality_gate_failures=failures,
    )


async def _import_strategy(req, _ctx):
    """完整导入: 分析 → 生成 → 返回策略代码。"""
    from app.python_strategy_pb2 import ImportStrategyResponse

    pipe = _get_pipe()
    source = getattr(req, 'source_code', '') or getattr(req, 'sourceCode', '')
    name = getattr(req, 'source_name', '') or getattr(req, 'sourceName', 'untitled.mq4')

    if not source:
        return ImportStrategyResponse()

    try:
        result = pipe.run(source, source_name=name)
    except Exception as e:
        logger.warning("ImportStrategy failed: %s", e)
        return ImportStrategyResponse()

    intent = result.intent
    blind_spot_list = [
        dict(id=bs.id, location=bs.location,
             category=bs.category.value if hasattr(bs.category, 'value') else str(bs.category),
             severity=bs.severity.value if hasattr(bs.severity, 'value') else str(bs.severity),
             description=bs.description,
             handling=bs.handling.value if hasattr(bs.handling, 'value') else str(bs.handling),
             userActionRequired=bs.user_action_required)
        for bs in intent.blind_spots
    ]

    return ImportStrategyResponse(
        strategy_id="",
        strategy_name=intent.meta.name or "ImportedStrategy",
        python_code=result.python_code,
        coverage_score=intent.coverage_score,
        blind_spots_json=_to_json(blind_spot_list),
    )
