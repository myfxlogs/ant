"""ConnectRPC PythonStrategyService — pure protobuf. Uses generated DESCRIPTOR."""

import logging
from connectrpc.server import ConnectASGIApplication, Endpoint
from connectrpc.method import MethodInfo, IdempotencyLevel

from app import python_strategy_pb2 as _pb
from app.python_strategy_pb2 import (
    BacktestStrategyRequest, BacktestStrategyResponse,
    ExecuteLiveRequest, ExecuteLiveResponse,
    ExecuteStrategyRequest, ExecuteStrategyResponse,
    TranspileCodeRequest, TranspileCodeResponse,
    ValidateStrategyRequest, ValidateStrategyResponse,
    GetPythonTemplatesResponse,
)
from app.backtest_run_start_pb2 import StartBacktestRunRequest, StartBacktestRunResponse
from app.backtest_run_query_pb2 import (
    GetBacktestRunRequest, GetBacktestRunResponse,
    ListBacktestRunsRequest, ListBacktestRunsResponse,
    WatchBacktestRunRequest,
)
from app.backtest_run_control_pb2 import (
    CancelBacktestRunRequest, CancelBacktestRunResponse,
    DeleteBacktestRunRequest, DeleteBacktestRunResponse,
    DeleteBacktestRunsRequest, DeleteBacktestRunsResponse,
)
from google.protobuf.empty_pb2 import Empty

_log = logging.getLogger("ConnectRPC")

_SVC = _pb.DESCRIPTOR.services_by_name["PythonStrategyService"]


# ── Transpile handler ────────────────────────────────────────────────

async def _transpile(req: TranspileCodeRequest, _ctx) -> TranspileCodeResponse:
    """Transpile MQL → Python using the deterministic AST transpiler.

    C1 (ADR-0020 D8): NO line-by-line regex fallback.  If the AST parser
    cannot parse a construct, it produces a hard GAP node → LLM fills it
    (passing through the same quality gates).  The regex transpiler is
    RETIRED — not kept as backup.

    Confidence is REDEFINED (C2): HIGH (all gates pass) = 1.0,
    LOW (any gate fails) = 0.0.  No more ``matched/(matched+gaps)``.
    """
    source = req.source_code or ""
    class_name = req.class_name or "TranslatedStrategy"
    if not source:
        return TranspileCodeResponse(is_deterministic=False)

    # ── AST transpiler (ONLY path) ──────────────────────────────────
    try:
        from tools.mql_transpiler.ast_transpiler import transpile_ast
        r = transpile_ast(source, class_name)

        # Gate-based confidence (C2): compiles + SDK imports + lint.
        from tools.mql_transpiler.quality_gate import QualityGate, QualityVerdict
        gate_report = QualityGate.assess(r.output)
        confidence = 1.0 if gate_report.verdict == QualityVerdict.HIGH else 0.0

        # Collect gap info for LLM fill (T4).
        gap_count = r.stats.get("gaps", 0)
        gap_samples = r.stats.get("gap_samples", [])[:10]

        # Scan for unmapped MQL symbols → update knowledge base.
        from tools.mql_transpiler.gap_kb import scan_unmapped, record_gaps
        unmapped = scan_unmapped(r.output)
        if unmapped:
            record_gaps(unmapped, source_mql=source[:200])
            gap_count += len(unmapped)
            for u in unmapped[:5]:
                gap_samples.append(f"UNMAPPED:{u.category}:{u.symbol}")

        # Also include gate failures as gap info.
        if gate_report.failures:
            for f in gate_report.failures[:5]:
                gap_samples.append(f"GATE:{f.gate}: {f.message}")

        return TranspileCodeResponse(
            target_code=r.output,
            confidence=confidence,
            total_patterns=len(r.output.split("\n")),
            gaps=gap_count + len(gate_report.failures),
            gap_samples=gap_samples,
            is_deterministic=True,
        )
    except Exception as e:
        _log.warning("AST transpiler failed: %s", e)
        return TranspileCodeResponse(
            is_deterministic=False,
            gap_samples=[f"transpiler error: {e}"],
        )


# ── Validate handler ──────────────────────────────────────────────────

async def _validate(req: ValidateStrategyRequest, _ctx) -> ValidateStrategyResponse:
    import json as _json
    from app.engine.sandbox import validate_strategy_code
    from app.engine.params_extractor import extract_required_params
    result = validate_strategy_code(req.code or "")
    params = extract_required_params(req.code or "") if result.valid else []
    quality_hints = [
        _pb.CodeQualityHint(category=h.category, severity=h.severity,
                            message=h.message, line=h.line, snippet=h.snippet)
        for h in result.quality_hints
    ]
    from app.routes.strategy_connect import _extract_sweep_dimensions, _extract_strategy_directives
    sweep_dims = _extract_sweep_dimensions(req.code or "")
    strategy_dirs = _extract_strategy_directives(req.code or "")
    from app.engine.vectorized_runner import detect_strategy_type
    strategy_type = detect_strategy_type(req.code or "")
    return ValidateStrategyResponse(
        valid=result.valid,
        errors=list(result.errors),
        warnings=list(result.warnings),
        quality_hints=quality_hints,
        sweep_dimensions=sweep_dims,
        strategy_directives=strategy_dirs,
        strategy_type=strategy_type,
        parameters_json=_json.dumps(result.parameters or [], ensure_ascii=False),
    )


# ── Execute handler ───────────────────────────────────────────────────

async def _execute(req: ExecuteStrategyRequest, _ctx) -> ExecuteStrategyResponse:
    from app.routes.strategy_connect import _execute_inline
    from app.engine.types import StrategyRuntimeError
    context = {"symbol": req.symbol or "", "timeframe": req.timeframe or "1h"}
    try:
        result = _execute_inline(req.code or "", context)
        signal = {
            "signal": str(result.get("signal", "hold")),
            "symbol": req.symbol or "",
            "price": float(result.get("price", 0)),
            "volume": float(result.get("volume", 0)),
            "confidence": float(result.get("confidence", 0)),
            "reason": str(result.get("reason", "")),
            "risk_level": str(result.get("risk_level", "medium")),
        }
        return ExecuteStrategyResponse(success=True, signal=signal)
    except StrategyRuntimeError as e:
        return ExecuteStrategyResponse(success=False, error=str(e))
    except Exception as e:
        return ExecuteStrategyResponse(success=False, error=str(e))


# ── Backtest handlers ─────────────────────────────────────────────────

async def _backtest(req: BacktestStrategyRequest, _ctx) -> BacktestStrategyResponse:
    from app.routes.backtest_connect import _run_backtest
    return await _run_backtest(req)


async def _start_backtest(req: StartBacktestRunRequest, _ctx) -> StartBacktestRunResponse:
    from app.routes.backtest_connect import _handle_start
    return await _handle_start(req)


async def _get_backtest(req: GetBacktestRunRequest, _ctx) -> GetBacktestRunResponse:
    from app.routes.backtest_connect import _handle_get
    return await _handle_get(req)


async def _list_backtests(req: ListBacktestRunsRequest, _ctx) -> ListBacktestRunsResponse:
    from app.routes.backtest_connect import _handle_list
    return await _handle_list(req)


async def _delete_backtest(req: DeleteBacktestRunRequest, _ctx) -> DeleteBacktestRunResponse:
    from app.routes.backtest_connect import _handle_delete
    return await _handle_delete(req)


async def _delete_backtests(req: DeleteBacktestRunsRequest, _ctx) -> DeleteBacktestRunsResponse:
    from app.routes.backtest_connect import _handle_delete_many
    return await _handle_delete_many(req)


async def _get_templates(_req: Empty, _ctx) -> GetPythonTemplatesResponse:
    from app.routes.strategy_connect import _list_templates
    return await _list_templates()


async def _execute_live(req: ExecuteLiveRequest, _ctx) -> ExecuteLiveResponse:
    from app.routes.live_execute_connect import _run_live
    return await _run_live(req)






try:
    from app.routes.strategy_import_connect import (
    )
except ImportError:


# ── Endpoint registry ─────────────────────────────────────────────────

_HANDLERS = {
    "Execute": (_execute, ExecuteStrategyRequest, ExecuteStrategyResponse),
    "Validate": (_validate, ValidateStrategyRequest, ValidateStrategyResponse),
    "Backtest": (_backtest, BacktestStrategyRequest, BacktestStrategyResponse),
    "StartBacktestRun": (_start_backtest, StartBacktestRunRequest, StartBacktestRunResponse),
    "GetBacktestRun": (_get_backtest, GetBacktestRunRequest, GetBacktestRunResponse),
    "ListBacktestRuns": (_list_backtests, ListBacktestRunsRequest, ListBacktestRunsResponse),
    "DeleteBacktestRun": (_delete_backtest, DeleteBacktestRunRequest, DeleteBacktestRunResponse),
    "DeleteBacktestRuns": (_delete_backtests, DeleteBacktestRunsRequest, DeleteBacktestRunsResponse),
    "GetTemplates": (_get_templates, Empty, GetPythonTemplatesResponse),
    "ExecuteLive": (_execute_live, ExecuteLiveRequest, ExecuteLiveResponse),
    "TranspileCode": (_transpile, TranspileCodeRequest, TranspileCodeResponse),
}


def _build_endpoints(_svc) -> dict:
    endpoints = {}

    # PythonStrategyService handlers
    for name, (fn, inp, out) in _HANDLERS.items():
        method_desc = _SVC.methods_by_name.get(name)
        if method_desc is None:
            continue
        mi = MethodInfo(
            name=name,
            service_name="ant.v1.PythonStrategyService",
            input=inp,
            output=out,
            idempotency_level=IdempotencyLevel.NO_SIDE_EFFECTS,
        )
        path = f"/ant.v1.PythonStrategyService/{name}"
        endpoints[path] = Endpoint.unary(mi, fn)

    return endpoints


class _App(ConnectASGIApplication):
    @property
    def path(self) -> str:
        return "/"


def create_app():
    return _App(
        service=_SVC,
        endpoints=_build_endpoints,
    )
