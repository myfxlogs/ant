"""盲区指纹计算 + 知识库管理。

每个盲区生成基于 AST 结构的指纹（不基于源码文本），
同一逻辑模式不同变量名 → 相同指纹 → 系统识别为同一盲区。
"""

from __future__ import annotations

import hashlib
from dataclasses import dataclass, field
from typing import Dict, List, Optional, Set

from tools.mql_migration.intent_ir import (
    BlindSpot,
    BlindSpotCategory,
    BlindSpotFingerprint,
    HandlingStrategy,
    Severity,
)


# ── Fingerprint computation ────────────────────────────────────────────


def compute_fingerprint(ast_node, mql_functions: Set[str] = None,
                        context: str = "") -> BlindSpotFingerprint:
    """从 AST 节点计算盲区指纹。

    Args:
        ast_node: 未识别的 AST 子树
        mql_functions: 该子树中涉及的 MQL 函数名集合
        context: 上下文（"entry_logic", "exit_logic", "sizing"）

    Returns:
        BlindSpotFingerprint with pattern signature hash.
    """
    sig_parts = [_ast_shape(ast_node)]

    if mql_functions:
        # 排序保证稳定性
        sig_parts.append("|".join(sorted(mql_functions)))

    sig_parts.append(context or "unknown")
    raw = "::".join(sig_parts)
    hash_val = hashlib.sha256(raw.encode()).hexdigest()[:16]

    return BlindSpotFingerprint(
        pattern_signature=hash_val,
        mql_functions=set(mql_functions or []),
        context=context,
    )


def _ast_shape(node) -> str:
    """Generate a structural shape string for an AST node.

    Captures the node type and its children's types, NOT the actual values.
    """
    if node is None:
        return "None"

    # Lazy imports to avoid circular deps.
    from tools.mql_transpiler.ast_nodes import (
        BinaryOp, CallExpr, CompoundStmt, ExpressionStmt,
        ForStmt, Identifier, IfStmt, NumberLiteral,
        StringLiteral, SubscriptExpr, VarDecl, AssignmentExpr,
    )

    cls = type(node).__name__

    if isinstance(node, CallExpr):
        return f"CallExpr[{node.name}]"
    elif isinstance(node, IfStmt):
        cond_shape = _ast_shape(node.condition)
        then_shape = _ast_shape(node.then_branch)
        return f"IfStmt{{{cond_shape},{then_shape}}}"
    elif isinstance(node, BinaryOp):
        return f"BinaryOp[{_ast_shape(node.left)}{node.op}{_ast_shape(node.right)}]"
    elif isinstance(node, CompoundStmt):
        inner = ",".join(_ast_shape(s) for s in (node.statements or []))
        return f"CompoundStmt{{{inner}}}"
    elif isinstance(node, ForStmt):
        return f"ForStmt{{{_ast_shape(node.body)}}}"
    elif isinstance(node, ExpressionStmt):
        return f"ExprStmt{{{_ast_shape(node.expr)}}}"
    elif isinstance(node, VarDecl):
        return f"VarDecl[{node.name}]"
    elif isinstance(node, AssignmentExpr):
        return f"Assign[{node.lhs}]"
    elif isinstance(node, (Identifier, NumberLiteral, StringLiteral)):
        return cls
    elif isinstance(node, SubscriptExpr):
        return f"Subscript[{node.name}]"
    else:
        return cls


# ── Blind spot knowledge base ──────────────────────────────────────────


class BlindSpotStatus:
    KNOWN_GAP = "known_gap"
    IN_PROGRESS = "in_progress"
    RESOLVED = "resolved"


@dataclass
class BlindSpotEntry:
    """知识库中的一条盲区记录。"""
    signature: str
    pattern_shape: str = ""
    category: BlindSpotCategory = BlindSpotCategory.CUSTOM_LOGIC
    first_seen: str = ""
    occurrence_count: int = 1
    severity: Severity = Severity.WARNING
    status: str = BlindSpotStatus.KNOWN_GAP
    resolved_by: str = ""              # e.g. "recognizers/custom_entry.py"
    sample_files: List[str] = field(default_factory=list)


@dataclass
class BlindSpotStats:
    total_fingerprints: int = 0
    resolved: int = 0
    known_gaps: int = 0
    in_progress: int = 0
    most_common_unresolved: str = ""


@dataclass
class BlindSpotKB:
    """盲区知识库 — 持久化，版本控制。"""
    fingerprints: Dict[str, BlindSpotEntry] = field(default_factory=dict)
    stats: BlindSpotStats = field(default_factory=BlindSpotStats)

    def record(self, blind_spot: BlindSpot,
               fingerprint: BlindSpotFingerprint,
               source_file: str = "") -> BlindSpotEntry:
        """记录一个盲区。已存在则增加计数。"""
        sig = fingerprint.pattern_signature
        if sig in self.fingerprints:
            entry = self.fingerprints[sig]
            entry.occurrence_count += 1
            if source_file and source_file not in entry.sample_files:
                entry.sample_files.append(source_file)
        else:
            entry = BlindSpotEntry(
                signature=sig,
                pattern_shape=fingerprint.pattern_signature,
                category=blind_spot.category,
                first_seen="",  # caller should set
                occurrence_count=1,
                severity=blind_spot.severity,
                status=BlindSpotStatus.KNOWN_GAP,
                sample_files=[source_file] if source_file else [],
            )
            self.fingerprints[sig] = entry

        self._recalc_stats()
        return entry

    def mark_resolved(self, signature: str, resolved_by: str) -> None:
        """标记盲区为已解决。"""
        if signature in self.fingerprints:
            self.fingerprints[signature].status = BlindSpotStatus.RESOLVED
            self.fingerprints[signature].resolved_by = resolved_by
            self._recalc_stats()

    def get_unresolved_by_frequency(self, min_count: int = 0) -> List[BlindSpotEntry]:
        """按出现频次排序的未解决盲区。"""
        unresolved = [
            e for e in self.fingerprints.values()
            if e.status != BlindSpotStatus.RESOLVED
            and e.occurrence_count >= min_count
        ]
        return sorted(unresolved, key=lambda e: -e.occurrence_count)

    def should_trigger_development(self, signature: str, threshold: int = 3) -> bool:
        """盲区出现次数超过阈值 → 应触发积木开发。"""
        entry = self.fingerprints.get(signature)
        if not entry:
            return False
        return (entry.status == BlindSpotStatus.KNOWN_GAP
                and entry.occurrence_count >= threshold)

    def _recalc_stats(self) -> None:
        resolved = sum(1 for e in self.fingerprints.values()
                       if e.status == BlindSpotStatus.RESOLVED)
        known = sum(1 for e in self.fingerprints.values()
                    if e.status == BlindSpotStatus.KNOWN_GAP)
        in_prog = sum(1 for e in self.fingerprints.values()
                      if e.status == BlindSpotStatus.IN_PROGRESS)
        most_common = ""
        unresolved = self.get_unresolved_by_frequency(min_count=1)
        if unresolved:
            most_common = unresolved[0].signature

        self.stats = BlindSpotStats(
            total_fingerprints=len(self.fingerprints),
            resolved=resolved,
            known_gaps=known,
            in_progress=in_prog,
            most_common_unresolved=most_common,
        )
