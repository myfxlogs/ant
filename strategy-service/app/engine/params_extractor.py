"""Static extraction of strategy parameter requirements from user code.

Goal: tell the frontend which `params['xxx']` keys must be filled by the user
before the code can be safely backtested / executed. We only do AST analysis
here — no code execution — so it stays cheap and side-effect free.

Heuristics:
- `params.get('key')`            -> required (no default)
- `params.get('key', default)`   -> optional, capture default + type
- `params['key']`                -> required (no default)
- `params['key'] if 'key' in params else default` is treated as optional
  (best-effort; we just look for `params[Constant]` outside such guards).

`params` is whatever name the user assigns from `context.get('params') or {}`.
We track those aliases by walking simple `Assign` patterns.
"""

from __future__ import annotations

import ast
from dataclasses import dataclass, asdict
from typing import Any, Dict, List, Optional, Set


@dataclass
class ParamSpec:
    key: str
    required: bool
    default: Optional[Any] = None
    type: Optional[str] = None  # "int" | "float" | "str" | "bool" | None
    suggested: Optional[Any] = None  # heuristic default for required keys

    def to_dict(self) -> Dict[str, Any]:
        return {k: v for k, v in asdict(self).items() if v is not None or k in ("key", "required")}


# Heuristic suggestions for "required but no default" keys. We match the key
# name (case-insensitive, after stripping common prefixes) against a small
# library of trader-friendly defaults so the user has somewhere to start.
# Order matters — first match wins.
# Keys the runtime ALWAYS injects into ``context`` / ``params`` before
# ``run()`` is called: these are bound at schedule/backtest launch time
# (account + symbol + timeframe). Strategies do not need to declare or
# require them — surfacing them as "required" in the UI would force users
# to fill values that get overwritten anyway. We drop them silently from
# the extracted parameter list.
_INJECTED_KEYS: Set[str] = {
    "symbol",
    "timeframe",
    "account",
    "account_id",
    "account_login",
}


_SUGGESTION_RULES: List[Dict[str, Any]] = [
    {"contains": ["timeframe", "interval"], "value": "H1", "type": "str"},
    {"contains": ["symbol", "pair", "ticker"], "value": "EURUSD", "type": "str"},
    {"contains": ["risk_level", "risklevel"], "value": "low", "type": "str"},
    {"contains": ["lot", "volume", "size"], "value": 0.1, "type": "float"},
    {"contains": ["confidence"], "value": 0.7, "type": "float"},
    {"contains": ["take_profit", "tp_pct", "tp_ratio"], "value": 2.0, "type": "float"},
    {"contains": ["stop_loss", "sl_pct", "sl_ratio"], "value": 1.0, "type": "float"},
    {"contains": ["max_loss"], "value": 0.01, "type": "float"},
    {"contains": ["threshold"], "value": 0.5, "type": "float"},
    {"contains": ["pct", "ratio", "percent"], "value": 1.0, "type": "float"},
    {"contains": ["fast"], "value": 12, "type": "int"},
    {"contains": ["slow"], "value": 26, "type": "int"},
    {"contains": ["signal"], "value": 9, "type": "int"},
    {"contains": ["rsi"], "value": 14, "type": "int"},
    {"contains": ["ema"], "value": 50, "type": "int"},
    {"contains": ["sma", "ma_"], "value": 20, "type": "int"},
    {"contains": ["period", "length", "window"], "value": 14, "type": "int"},
]


def _suggest_default(key: str) -> Optional[Dict[str, Any]]:
    k = key.lower()
    for rule in _SUGGESTION_RULES:
        for needle in rule["contains"]:
            if needle in k:
                return {"value": rule["value"], "type": rule["type"]}
    return None


def _literal_value(node: ast.AST) -> Any:
    if isinstance(node, ast.Constant):
        return node.value
    if isinstance(node, ast.UnaryOp) and isinstance(node.op, ast.USub) and isinstance(node.operand, ast.Constant):
        v = node.operand.value
        return -v if isinstance(v, (int, float)) else None
    return None


def _type_of(value: Any) -> Optional[str]:
    if isinstance(value, bool):
        return "bool"
    if isinstance(value, int):
        return "int"
    if isinstance(value, float):
        return "float"
    if isinstance(value, str):
        return "str"
    return None


def _is_params_alias(value: ast.AST) -> bool:
    """Detect `context.get('params') or {}` / `context['params']` patterns."""
    # context.get('params') ...
    if isinstance(value, ast.BoolOp) and isinstance(value.op, ast.Or):
        return any(_is_params_alias(v) for v in value.values)
    if isinstance(value, ast.Call) and isinstance(value.func, ast.Attribute):
        if value.func.attr == "get" and value.args:
            arg0 = _literal_value(value.args[0])
            if arg0 == "params":
                return True
    if isinstance(value, ast.Subscript) and isinstance(value.value, ast.Name):
        idx = value.slice
        if isinstance(idx, ast.Constant) and idx.value == "params":
            return True
    return False


def _collect_alias_names(tree: ast.AST) -> Set[str]:
    aliases: Set[str] = {"params"}
    for node in ast.walk(tree):
        if isinstance(node, ast.Assign):
            if _is_params_alias(node.value):
                for target in node.targets:
                    if isinstance(target, ast.Name):
                        aliases.add(target.id)
    return aliases


def extract_required_params(code: str) -> List[Dict[str, Any]]:
    """Return list of param specs sorted by first appearance, required first."""
    try:
        tree = ast.parse(code)
    except SyntaxError:
        return []

    aliases = _collect_alias_names(tree)
    found: Dict[str, ParamSpec] = {}
    order: List[str] = []

    def remember(spec: ParamSpec) -> None:
        existing = found.get(spec.key)
        if existing is None:
            found[spec.key] = spec
            order.append(spec.key)
            return
        # Promote required > optional only if newly-discovered spec is required
        # without default; keep optional default if previously discovered.
        if existing.required and not spec.required:
            found[spec.key] = spec
        elif not existing.required and spec.required:
            # Keep the optional one (it has a default).
            return

    for node in ast.walk(tree):
        # params.get('key', default)
        if isinstance(node, ast.Call) and isinstance(node.func, ast.Attribute):
            if node.func.attr == "get" and isinstance(node.func.value, ast.Name) and node.func.value.id in aliases:
                if not node.args:
                    continue
                key = _literal_value(node.args[0])
                if not isinstance(key, str):
                    continue
                if len(node.args) >= 2:
                    default = _literal_value(node.args[1])
                    remember(ParamSpec(key=key, required=False, default=default, type=_type_of(default)))
                else:
                    remember(ParamSpec(key=key, required=True))
        # params['key']
        elif isinstance(node, ast.Subscript) and isinstance(node.value, ast.Name) and node.value.id in aliases:
            idx = node.slice
            key = _literal_value(idx)
            if isinstance(key, str):
                remember(ParamSpec(key=key, required=True))

    # Attach heuristic suggestions to required entries (only when the user
    # didn't already infer a type from a sibling default in the same code).
    for spec in found.values():
        if spec.required and spec.suggested is None:
            hint = _suggest_default(spec.key)
            if hint is not None:
                spec.suggested = hint["value"]
                if spec.type is None:
                    spec.type = hint["type"]

    # Stable order: required first (by appearance), then optional. Keys that
    # the runtime guarantees to inject (symbol/timeframe/account…) are
    # filtered out so users are not asked to fill values the engine will
    # overwrite at schedule/backtest launch time.
    visible_order = [k for k in order if k not in _INJECTED_KEYS]
    required = [found[k].to_dict() for k in visible_order if found[k].required]
    optional = [found[k].to_dict() for k in visible_order if not found[k].required]
    return required + optional
