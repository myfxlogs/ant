"""ConnectRPC PythonStrategyService — pure protobuf. Uses generated DESCRIPTOR."""

import logging
from connectrpc.server import ConnectASGIApplication, Endpoint
import connectrpc.method

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

async def _transpile(req: TranspileCodeRequest) -> TranspileCodeResponse:
    source = req.source_code or ""
    class_name = req.class_name or "TranslatedStrategy"
    if not source:
        return TranspileCodeResponse(is_deterministic=False)

    # AST transpiler first.
    try:
        from tools.mql_transpiler.ast_transpiler import transpile_ast
        r = transpile_ast(source, class_name)
        return TranspileCodeResponse(
            target_code=r.output,
            confidence=r.stats.get("confidence", 0),
            total_patterns=r.stats.get("matched", 0) + r.stats.get("gaps", 0),
            gaps=r.stats.get("gaps", 0),
            gap_samples=r.stats.get("gap_samples", [])[:10],
            is_deterministic=True,
        )
    except Exception:
        pass

    # Line-by-line fallback.
    try:
        from tools.mql_transpiler.transpiler import MQLTranspiler
        tp = MQLTranspiler(class_name or "TranslatedStrategy")
        r = tp.transpile(source, "_t.mq4")
        return TranspileCodeResponse(
            target_code=r.output,
            confidence=tp.get_confidence(),
            total_patterns=r.stats.patterns_matched + r.stats.gaps,
            gaps=r.stats.gaps,
            gap_samples=list(r.stats.gap_reasons.keys())[:10],
            is_deterministic=True,
        )
    except Exception:
        pass

    return TranspileCodeResponse(is_deterministic=False, gap_samples=["unavailable"])


# ── Validate handler ──────────────────────────────────────────────────

async def _validate(req: ValidateStrategyRequest) -> ValidateStrategyResponse:
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

async def _execute(req: ExecuteStrategyRequest) -> ExecuteStrategyResponse:
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

async def _backtest(req: BacktestStrategyRequest) -> BacktestStrategyResponse:
    from app.routes.backtest_connect import _run_backtest
    return await _run_backtest(req)


async def _start_backtest(req: StartBacktestRunRequest) -> StartBacktestRunResponse:
    from app.routes.backtest_connect import _handle_start
    return await _handle_start(req)


async def _get_backtest(req: GetBacktestRunRequest) -> GetBacktestRunResponse:
    from app.routes.backtest_connect import _handle_get
    return await _handle_get(req)


async def _list_backtests(req: ListBacktestRunsRequest) -> ListBacktestRunsResponse:
    from app.routes.backtest_connect import _handle_list
    return await _handle_list(req)


async def _delete_backtest(req: DeleteBacktestRunRequest) -> DeleteBacktestRunResponse:
    from app.routes.backtest_connect import _handle_delete
    return await _handle_delete(req)


async def _delete_backtests(req: DeleteBacktestRunsRequest) -> DeleteBacktestRunsResponse:
    from app.routes.backtest_connect import _handle_delete_many
    return await _handle_delete_many(req)


async def _get_templates(_req: Empty) -> GetPythonTemplatesResponse:
    from app.routes.strategy_connect import _list_templates
    return await _list_templates()


async def _execute_live(req: ExecuteLiveRequest) -> ExecuteLiveResponse:
    from app.routes.live_execute_connect import _run_live
    return await _run_live(req)


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
    for name, (fn, inp, out) in _HANDLERS.items():
        method_desc = _SVC.methods_by_name.get(name)
        if method_desc is None:
            continue
        mi = connectrpc.method.MethodInfo(
            name=name,
            service_name="ant.v1.PythonStrategyService",
            input=inp,
            output=out,
        )
        endpoints[name] = Endpoint(method=mi)(fn)
    return endpoints


class _App(ConnectASGIApplication):
    @property
    def path(self) -> str:
        return "/ant.v1.PythonStrategyService/"


def create_app():
    return _App(
        service=_SVC,
        endpoints=_build_endpoints,
    )
