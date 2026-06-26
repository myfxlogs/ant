"""参数面板 schema 生成器。

将 ParamSpec[] 转为前端可消费的参数规格（通过 gRPC API 传递，proto 序列化）。
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional

from tools.mql_migration.intent_ir import (
    ParamGroup,
    ParamRange,
    ParamSpec,
    ParamType,
)


# ── Frontend schema types ──────────────────────────────────────────────


@dataclass
class FieldSchema:
    """单个参数的 UI 字段规格。"""
    name: str                           # 参数名（MQL 原始）
    label: str                          # 显示标签
    field_type: str                     # "number" | "text" | "switch" | "select"
    default: Any = None
    group: str = "系统参数"               # 分组标签
    group_order: int = 90               # 分组排序权重
    description: str = ""
    # Number fields
    min: Optional[float] = None
    max: Optional[float] = None
    step: Optional[float] = None
    # Select fields
    options: List[Dict[str, str]] = field(default_factory=list)


@dataclass
class PanelSchema:
    """完整的参数面板 schema。"""
    strategy_name: str = ""
    groups: List[Dict[str, Any]] = field(default_factory=list)
    fields: List[FieldSchema] = field(default_factory=list)


# 分组排序权重（决定前端面板展示顺序）
_GROUP_ORDER = {
    ParamGroup.ENTRY: 10,
    ParamGroup.EXIT: 20,
    ParamGroup.SIZING: 30,
    ParamGroup.RISK: 40,
    ParamGroup.SYSTEM: 90,
}


def generate_panel_schema(params: List[ParamSpec],
                          strategy_name: str = "") -> PanelSchema:
    """从 ParamSpec 列表生成前端参数面板 schema。

    Args:
        params: 策略参数列表
        strategy_name: 策略名称

    Returns:
        PanelSchema — 前端可直接消费
    """
    fields = []
    groups_seen: Dict[str, List[FieldSchema]] = {}

    for p in params:
        field = _to_field(p)
        fields.append(field)

        group_name = field.group
        if group_name not in groups_seen:
            groups_seen[group_name] = []
        groups_seen[group_name].append(field)

    # 构建分组列表（按权重排序）
    groups = []
    for group_name, group_fields in sorted(
        groups_seen.items(),
        key=lambda x: _GROUP_ORDER.get(
            _param_group_from_label(x[0]), 99
        )
    ):
        groups.append({
            "name": group_name,
            "order": _GROUP_ORDER.get(_param_group_from_label(group_name), 99),
            "field_count": len(group_fields),
            "field_names": [f.name for f in group_fields],
        })

    return PanelSchema(
        strategy_name=strategy_name,
        groups=groups,
        fields=fields,
    )


def _to_field(param: ParamSpec) -> FieldSchema:
    """ParamSpec → FieldSchema。"""
    field_type = _map_field_type(param.param_type)

    field = FieldSchema(
        name=param.name,
        label=param.label or param.name,
        field_type=field_type,
        default=param.default,
        group=param.group.value if isinstance(param.group, ParamGroup) else str(param.group),
        group_order=_GROUP_ORDER.get(param.group, 90),
        description=param.description,
    )

    if param.range:
        field.min = param.range.min
        field.max = param.range.max
        field.step = param.range.step

    return field


def _map_field_type(pt: ParamType) -> str:
    return {
        ParamType.INT: "number",
        ParamType.DOUBLE: "number",
        ParamType.BOOL: "switch",
        ParamType.STRING: "text",
        ParamType.ENUM: "select",
    }.get(pt, "text")


def _param_group_from_label(label: str) -> ParamGroup | None:
    for g in ParamGroup:
        if g.value == label:
            return g
    return None


# ── Proto-compatible serialization ─────────────────────────────────────


def panel_schema_to_dict(schema: PanelSchema) -> dict:
    """将 PanelSchema 转为字典（可直接序列化为 proto JSON）。"""
    return {
        "strategy_name": schema.strategy_name,
        "groups": schema.groups,
        "fields": [
            {
                "name": f.name,
                "label": f.label,
                "type": f.field_type,
                "default": f.default,
                "group": f.group,
                "groupOrder": f.group_order,
                "description": f.description,
                "min": f.min,
                "max": f.max,
                "step": f.step,
                "options": f.options,
            }
            for f in schema.fields
        ],
    }
