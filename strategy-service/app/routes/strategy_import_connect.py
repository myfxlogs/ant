"""StrategyImportService ConnectRPC handlers.

MQL 策略导入管线: MQL 源码 → 分析报告 → 用户确认 → Python 代码生成。

Protocol: POST /ant.v1.StrategyImportService/{AnalyzeCode,GenerateCode,ImportStrategy}
"""

from __future__ import annotations

import logging
from dataclasses import dataclass, field
from typing import List, Optional

from tools.mql_migration.pipeline import MigrationPipeline
from tools.mql_migration.generators.param_schema_gen import (
    generate_panel_schema,
)

logger = logging.getLogger(__name__)

# Singleton pipeline — initialized once, reused across requests.
_pipe: Optional[MigrationPipeline] = None


def _get_pipe() -> MigrationPipeline:
    global _pipe
    if _pipe is None:
        _pipe = MigrationPipeline()
    return _pipe


# ── Response dataclasses (mirror proto messages) ───────────────────────


@dataclass
class ParamOption:
    value: str = ""
    label: str = ""


@dataclass
class ParamField:
    name: str = ""
    label: str = ""
    param_type: str = "number"
    default_value: str = ""
    group: str = ""
    group_order: int = 90
    min: Optional[float] = None
    max: Optional[float] = None
    step: Optional[float] = None
    options: List[ParamOption] = field(default_factory=list)


@dataclass
class ParamGroupInfo:
    name: str = ""
    label: str = ""
    order: int = 0
    field_count: int = 0


@dataclass
class BlindSpotItem:
    id: str = ""
    location: str = ""
    category: str = ""
    severity: str = ""
    description: str = ""
    handling: str = ""
    user_action_required: bool = False


@dataclass
class AnalyzeCodeResponse:
    strategy_name: str = ""
    mql_version: str = "mql4"
    coverage_score: float = 0.0
    total_blocks: int = 0
    recognized_blocks: int = 0
    params: List[ParamField] = field(default_factory=list)
    groups: List[ParamGroupInfo] = field(default_factory=list)
    execution_kind: str = "on_tick"
    blind_spots: List[BlindSpotItem] = field(default_factory=list)
    entry_rules_count: int = 0
    exit_rules_count: int = 0
    sizing_kind: str = "fixed"
    risk_checks_count: int = 0
    indicator_names: List[str] = field(default_factory=list)


@dataclass
class GenerateCodeResponse:
    python_code: str = ""
    code_lines: int = 0
    compiles: bool = False
    quality_gate_failures: List[str] = field(default_factory=list)


@dataclass
class ImportStrategyResponse:
    strategy_id: str = ""
    strategy_name: str = ""
    python_code: str = ""
    coverage_score: float = 0.0
    blind_spots: List[BlindSpotItem] = field(default_factory=list)


# ── Handlers ──────────────────────────────────────────────────────────


async def _analyze_code(req, _ctx) -> AnalyzeCodeResponse:
    """分析 MQL 源码 → 策略意图摘要 + 参数面板 + 盲区报告。"""
    pipe = _get_pipe()
    source = req.source_code or ""
    name = req.source_name or "untitled.mq4"

    if not source:
        return AnalyzeCodeResponse()

    try:
        result = pipe.run(source, source_name=name)
    except Exception as e:
        logger.warning("AnalyzeCode failed: %s", e)
        return AnalyzeCodeResponse(
            blind_spots=[BlindSpotItem(
                severity="致命",
                description=f"分析失败: {e}",
                handling="请检查 MQL 源码是否有效",
                user_action_required=True,
            )]
        )

    intent = result.intent

    # Build param panel schema
    schema = generate_panel_schema(intent.params, intent.meta.name)

    params = []
    for f in schema.fields:
        pf = ParamField(
            name=f.name,
            label=f.label,
            param_type=f.field_type,
            default_value=str(f.default) if f.default is not None else "",
            group=f.group,
            group_order=f.group_order,
            min=f.min,
            max=f.max,
            step=f.step,
        )
        params.append(pf)

    groups = [
        ParamGroupInfo(
            name=g["name"],
            label=g["name"],
            order=g["order"],
            field_count=g["field_count"],
        )
        for g in schema.groups
    ]

    blind_spots = [
        BlindSpotItem(
            id=bs.id,
            location=bs.location,
            category=bs.category.value if hasattr(bs.category, 'value') else str(bs.category),
            severity=bs.severity.value if hasattr(bs.severity, 'value') else str(bs.severity),
            description=bs.description,
            handling=bs.handling.value if hasattr(bs.handling, 'value') else str(bs.handling),
            user_action_required=bs.user_action_required,
        )
        for bs in intent.blind_spots
    ]

    indicator_names = [ind.sdk_method for ind in intent.indicators]

    return AnalyzeCodeResponse(
        strategy_name=intent.meta.name or "ImportedStrategy",
        mql_version=intent.meta.mql_version.value,
        coverage_score=intent.coverage_score,
        total_blocks=intent.total_blocks,
        recognized_blocks=intent.recognized_blocks,
        params=params,
        groups=groups,
        execution_kind=intent.execution.kind.value,
        blind_spots=blind_spots,
        entry_rules_count=len(intent.entry),
        exit_rules_count=len(intent.exit),
        sizing_kind=intent.sizing.kind.value if intent.sizing else "fixed",
        risk_checks_count=len(intent.risk),
        indicator_names=indicator_names,
    )


async def _generate_code(req, _ctx) -> GenerateCodeResponse:
    """生成 Python 策略代码（应用用户参数覆盖）。"""
    pipe = _get_pipe()
    source = req.source_code or ""
    name = req.source_name or "untitled.mq4"

    if not source:
        return GenerateCodeResponse(
            quality_gate_failures=["源码为空"]
        )

    try:
        result = pipe.run(source, source_name=name)
    except Exception as e:
        logger.warning("GenerateCode failed: %s", e)
        return GenerateCodeResponse(
            quality_gate_failures=[f"生成失败: {e}"]
        )

    code = result.python_code

    # Quality gate check
    import ast as py_ast
    compiles = True
    failures = []
    try:
        py_ast.parse(code)
    except SyntaxError as e:
        compiles = False
        failures.append(f"SyntaxError L{e.lineno}: {e.msg}")

    from tools.mql_transpiler.quality_gate import QualityGate, QualityVerdict
    gate = QualityGate.assess(code)
    if gate.verdict != QualityVerdict.HIGH:
        for f in gate.failures:
            failures.append(f"[{f.gate}] {f.message}")

    return GenerateCodeResponse(
        python_code=code,
        code_lines=len(code.splitlines()),
        compiles=compiles,
        quality_gate_failures=failures,
    )


async def _import_strategy(req, _ctx) -> ImportStrategyResponse:
    """完整导入: 分析 → 生成 → 返回策略代码。"""
    pipe = _get_pipe()
    source = req.source_code or ""
    name = req.source_name or "untitled.mq4"

    if not source:
        return ImportStrategyResponse()

    try:
        result = pipe.run(source, source_name=name)
    except Exception as e:
        logger.warning("ImportStrategy failed: %s", e)
        return ImportStrategyResponse(
            blind_spots=[BlindSpotItem(
                severity="致命",
                description=f"导入失败: {e}",
                user_action_required=True,
            )]
        )

    intent = result.intent

    blind_spots = [
        BlindSpotItem(
            id=bs.id,
            location=bs.location,
            category=bs.category.value if hasattr(bs.category, 'value') else str(bs.category),
            severity=bs.severity.value if hasattr(bs.severity, 'value') else str(bs.severity),
            description=bs.description,
            handling=bs.handling.value if hasattr(bs.handling, 'value') else str(bs.handling),
            user_action_required=bs.user_action_required,
        )
        for bs in intent.blind_spots
    ]

    return ImportStrategyResponse(
        strategy_name=intent.meta.name or "ImportedStrategy",
        python_code=result.python_code,
        coverage_score=intent.coverage_score,
        blind_spots=blind_spots,
    )
