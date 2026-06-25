#!/usr/bin/env python3
"""Parse MT4/MT5 proto files → authoritative MQL function catalog.

Extracts RPC names, enum definitions, and message fields from
reference/grpc/mt4.proto and reference/grpc/mt5.proto, then
cross-references with the transpiler's current mappings to
produce an authoritative gap report.

Usage:   python3 tools/mql_transpiler/kb/coverage/proto_parser.py
"""

from __future__ import annotations

import re
from collections import defaultdict
from pathlib import Path

PROTO_DIR = Path(__file__).resolve().parent.parent.parent.parent.parent / "reference" / "grpc"
MT4_PROTO = PROTO_DIR / "mt4.proto"
MT5_PROTO = PROTO_DIR / "mt5.proto"

# ── Proto parser (regex-based, no protoc needed) ──


def parse_rpcs(path: Path) -> list[tuple[str, str]]:
    """Extract (rpc_name, request_type) pairs from a proto file."""
    results = []
    with open(path) as f:
        for line in f:
            m = re.match(r"\s*rpc\s+(\w+)\s*\(\s*(\w+)\s*\)", line)
            if m:
                results.append((m.group(1), m.group(2)))
    return results


def parse_enums(path: Path) -> dict[str, list[tuple[str, int]]]:
    """Extract enum definitions with their values."""
    results = {}
    current_enum = None
    with open(path) as f:
        for line in f:
            m_enum = re.match(r"\s*enum\s+(\w+)\s*\{", line)
            m_close = re.match(r"\s*\}", line)
            m_value = re.match(r"\s*(\w+)\s*=\s*(\d+)\s*;", line)

            if m_enum:
                current_enum = m_enum.group(1)
                results[current_enum] = []
            elif m_close and current_enum:
                current_enum = None
            elif m_value and current_enum:
                results[current_enum].append((m_value.group(1), int(m_value.group(2))))
    return results


def parse_message_fields(path: Path) -> dict[str, list[str]]:
    """Extract message field names."""
    results = {}
    current_msg = None
    with open(path) as f:
        for line in f:
            m_msg = re.match(r"\s*message\s+(\w+)\s*\{", line)
            m_close = re.match(r"\s*\}", line)
            m_field = re.match(r"\s*(?:repeated\s+)?(?:optional\s+)?\w+\s+(\w+)\s*=", line)

            if m_msg:
                current_msg = m_msg.group(1)
                results[current_msg] = []
            elif m_close and current_msg:
                current_msg = None
            elif m_field and current_msg:
                results[current_msg].append(m_field.group(1))
    return results


# ── Cross-reference with transpiler ──

# Inline the transpiler's covered functions (import would complicate PYTHONPATH)
COVERED = {
    # Trade
    "OrderSend", "OrderClose", "OrderCloseBy", "OrderClosePrice", "OrderCloseTime",
    "OrderComment", "OrderCommission", "OrderDelete", "OrderExpiration", "OrderLots",
    "OrderMagicNumber", "OrderModify", "OrderOpenPrice", "OrderOpenTime",
    "OrderProfit", "OrderSelect", "OrderStopLoss", "OrdersTotal", "OrderSwap",
    "OrderSymbol", "OrderTakeProfit", "OrderTicket", "OrderType",
    # MQL5 trade
    "PositionSelect", "PositionsTotal", "PositionGetSymbol", "PositionGetTicket",
    # Account
    "AccountBalance", "AccountEquity", "AccountFreeMargin",
    "AccountLeverage",
    # Market
    "Symbol", "MarketInfo",
    # Common
    "Print", "Comment", "Alert",
    # Math
    "MathAbs", "MathMax", "MathMin", "MathRound",
    "MathFloor", "MathCeil", "MathSqrt", "MathPow",
    "MathLog", "MathExp", "MathSin", "MathCos",
    # Timers
    "EventSetTimer", "EventKillTimer",
}


def main():
    if not MT4_PROTO.exists() or not MT5_PROTO.exists():
        print(f"Proto files not found in {PROTO_DIR}")
        return

    # Parse both proto files.
    all_rpcs = {}
    all_enums = {}
    all_fields = {}

    for label, path in [("MT4", MT4_PROTO), ("MT5", MT5_PROTO)]:
        all_rpcs[label] = parse_rpcs(path)
        all_enums[label] = parse_enums(path)
        all_fields[label] = parse_message_fields(path)

    # Report.
    total_rpcs = len(all_rpcs["MT4"]) + len(all_rpcs["MT5"])
    total_enums = sum(len(v) for v in all_enums["MT4"].values()) + sum(len(v) for v in all_enums["MT5"].values())
    total_field_count = sum(len(v) for v in all_fields["MT4"].values()) + sum(len(v) for v in all_fields["MT5"].values())
    total_messages = len(all_fields["MT4"]) + len(all_fields["MT5"])

    # Map RPC names → MQL function equivalents.
    mql_from_rpcs = set()
    for label in ["MT4", "MT5"]:
        for rpc_name, _ in all_rpcs[label]:
            # Common RPC → MQL function name mappings
            mql_name = rpc_name
            # Strip common suffixes
            for suffix in ["Pagination", "Many", "Ex", "Month", "Today", "HighLow"]:
                if mql_name.endswith(suffix):
                    mql_name = mql_name[:-len(suffix)]
            mql_from_rpcs.add(mql_name)

    covered_by_rpc = {name for name in mql_from_rpcs if name in COVERED}
    uncovered_by_rpc = mql_from_rpcs - COVERED

    print(f"Proto API Surface")
    print(f"{'='*60}")
    print(f"  MT4: {len(all_rpcs['MT4'])} RPCs, {len(all_enums['MT4'])} enums, {len(all_fields['MT4'])} messages")
    print(f"  MT5: {len(all_rpcs['MT5'])} RPCs, {len(all_enums['MT5'])} enums, {len(all_fields['MT5'])} messages")
    print(f"  Total RPCs (MQL functions): {total_rpcs}")
    print(f"  Total enum values:          {total_enums}")
    print(f"  Total message fields:       {total_field_count}")
    print(f"  Total message types:        {total_messages}")
    print()
    print(f"Transpiler Coverage vs Proto API")
    print(f"{'='*60}")
    print(f"  Unique MQL functions from RPCs:  {len(mql_from_rpcs)}")
    print(f"  Covered by transpiler:           {len(covered_by_rpc)} ({len(covered_by_rpc)/max(len(mql_from_rpcs),1)*100:.0f}%)")
    print(f"  Not yet covered (from proto):    {len(uncovered_by_rpc)}")
    print()

    if uncovered_by_rpc:
        print(f"  Top uncovered RPC-derived functions:")
        for name in sorted(uncovered_by_rpc)[:30]:
            print(f"    - {name}")

    # Key enums for MQL5 property mappings.
    print()
    print(f"Key MQL5 Enum Definitions (from proto)")
    print(f"{'='*60}")
    for label in ["MT4", "MT5"]:
        for enum_name, values in all_enums[label].items():
            if any(kw in enum_name for kw in ["Type", "State", "Mode", "Property", "Reason"]):
                print(f"  [{label}] {enum_name}: {len(values)} values")
                for name, val in values[:5]:
                    print(f"    {name} = {val}")
                if len(values) > 5:
                    print(f"    ... +{len(values)-5} more")


if __name__ == "__main__":
    main()
