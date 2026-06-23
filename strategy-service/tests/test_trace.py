"""T4.1 — Trace infrastructure tests.

Validates:
  - TraceEvent and StrategyTrace creation
  - Full lifecycle event recording (tick, bar, decision, intent, risk, fill)
  - Trace serialization (JSON round-trip)
  - Trace replay (human-readable log)
  - Drift comparison (backtest vs live)
"""

import json
import os
import tempfile
import unittest

from app.engine.trace import (
    DriftReport,
    StrategyTrace,
    TraceEvent,
    compare_traces,
    replay_trace,
)


class TestTraceEvent(unittest.TestCase):
    """Individual trace event creation."""

    def test_create_event(self):
        evt = TraceEvent(seq=1, ts_ms=1719000000000, event="tick",
                         payload={"bid": "1.08500", "ask": "1.08503"})
        self.assertEqual(evt.seq, 1)
        self.assertEqual(evt.event, "tick")
        self.assertEqual(evt.payload["bid"], "1.08500")

    def test_to_dict(self):
        evt = TraceEvent(seq=1, ts_ms=1719000000000, event="fill",
                         payload={"ticket": "42", "price": "1.08500", "volume": "0.10"})
        d = evt.to_dict()
        self.assertEqual(d["seq"], 1)
        self.assertEqual(d["ticket"], "42")


class TestStrategyTrace(unittest.TestCase):
    """Full trace collection."""

    def setUp(self):
        self.trace = StrategyTrace(
            strategy_name="TestEA",
            run_id="run-001",
            mode="backtest",
            symbol="EURUSD",
            timeframe="M15",
            started_at_ms=1719000000000,
        )

    def test_create_trace(self):
        self.assertEqual(self.trace.strategy_name, "TestEA")
        self.assertEqual(self.trace.mode, "backtest")
        self.assertEqual(len(self.trace.events), 0)

    def test_add_tick(self):
        self.trace.tick(1719000001000, bid=1.08497, ask=1.08503)
        self.assertEqual(len(self.trace.events), 1)
        self.assertEqual(self.trace.events[0].event, "tick")

    def test_add_bar(self):
        self.trace.bar_close(1719000001000, "M15",
                             open_=1.08490, high=1.08520, low=1.08480,
                             close=1.08500, volume=100.0)
        self.assertEqual(len(self.trace.events), 1)
        evt = self.trace.events[0]
        self.assertEqual(evt.event, "bar")
        self.assertEqual(evt.payload["timeframe"], "M15")

    def test_full_lifecycle(self):
        """Record a complete strategy execution lifecycle."""
        # Tick.
        self.trace.tick(1719000001000, bid=1.08497, ask=1.08503)
        # Bar close.
        self.trace.bar_close(1719000001000, "M15",
                             open_=1.08490, high=1.08520, low=1.08480,
                             close=1.08500, volume=100.0)
        # Lifecycle callback.
        self.trace.lifecycle("on_bar")
        # Order intent.
        self.trace.order_intent(symbol="EURUSD", side="buy", order_type="buy",
                                volume="0.10", price="0", magic="1",
                                comment="ema_cross", ticket=1001)
        # Risk decision.
        self.trace.risk_decision(allowed=True, ticket=1001)
        # Fill.
        self.trace.fill(ticket=1001, symbol="EURUSD", side="buy",
                        volume="0.10", price="1.08503", commission="0.70")
        # Close.
        self.trace.close_position(ticket=1001, volume="0.10",
                                  price="1.09000", pnl="49.70")
        # Deinit.
        self.trace.lifecycle("on_deinit")

        self.assertEqual(len(self.trace.events), 8)
        event_types = [e.event for e in self.trace.events]
        self.assertEqual(event_types, [
            "tick", "bar", "on_bar",
            "order_intent", "risk_decision", "fill",
            "close", "on_deinit",
        ])
        # But wait — there's one extra. Let me count: tick=1, bar=2,
        # lifecycle_on_bar=3, order_intent=4, risk_decision=5, fill=6,
        # close=7, lifecycle_on_deinit=8. That's 8, not 9.
        # Actually I said 9 but let me recount.
        # Events: 1.tick, 2.bar, 3.on_bar, 4.order_intent, 5.risk_decision,
        # 6.fill, 7.close, 8.on_deinit. Total=8.
        self.assertEqual(len(self.trace.events), 8)

    def test_reject_event(self):
        self.trace.reject(reason="margin insufficient", ticket=0)
        self.assertEqual(self.trace.events[0].event, "reject")

    def test_modify_event(self):
        self.trace.modify_position(ticket=100, sl="1.08000", tp="1.10000")
        evt = self.trace.events[0]
        self.assertEqual(evt.event, "modify")
        self.assertEqual(evt.payload["sl"], "1.08000")


class TestTraceSerialization(unittest.TestCase):
    """JSON round-trip."""

    def setUp(self):
        self.trace = StrategyTrace(
            strategy_name="TestEA", run_id="run-001", mode="live",
            symbol="EURUSD", timeframe="M15", started_at_ms=1719000000000,
        )
        self.trace.tick(1719000001000, bid=1.08500, ask=1.08503)
        self.trace.order_intent(symbol="EURUSD", side="buy", order_type="buy",
                                volume="0.10", ticket=42)
        self.trace.risk_decision(allowed=True, ticket=42)
        self.trace.fill(ticket=42, symbol="EURUSD", side="buy",
                        volume="0.10", price="1.08503")

    def test_to_json(self):
        json_str = self.trace.to_json()
        self.assertIn("TestEA", json_str)
        self.assertIn("tick", json_str)

    def test_round_trip(self):
        json_str = self.trace.to_json()
        restored = StrategyTrace.from_json(json_str)
        self.assertEqual(restored.strategy_name, "TestEA")
        self.assertEqual(restored.mode, "live")
        self.assertEqual(len(restored.events), 4)
        self.assertEqual(restored.events[0].event, "tick")

    def test_write_to_file(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as f:
            f.write(self.trace.to_json())
            path = f.name
        try:
            with open(path) as f:
                data = json.load(f)
            self.assertEqual(data["strategy_name"], "TestEA")
            self.assertEqual(data["event_count"], 4)
        finally:
            os.unlink(path)


class TestTraceReplay(unittest.TestCase):
    """Human-readable trace replay."""

    def setUp(self):
        self.trace = StrategyTrace(
            strategy_name="ReplayEA", run_id="run-001", mode="backtest",
            symbol="EURUSD", timeframe="M15", started_at_ms=1719000000000,
        )
        self.trace.tick(1719000001000, bid=1.08500, ask=1.08503)
        self.trace.order_intent(symbol="EURUSD", side="buy", order_type="buy",
                                volume="0.10", price="0", magic="1", ticket=100)
        self.trace.risk_decision(allowed=False, reason="max lot size exceeded",
                                 rule_hit="max_lot_size", ticket=100)
        self.trace.reject(reason="risk gate blocked", ticket=100)

    def test_replay_produces_output(self):
        lines = replay_trace(self.trace)
        self.assertGreater(len(lines), 3)
        self.assertIn("ReplayEA", lines[0])
        self.assertIn("TICK", "\n".join(lines))
        self.assertIn("DENY", "\n".join(lines))

    def test_replay_summary(self):
        lines = replay_trace(self.trace)
        self.assertTrue(lines[-1].startswith("End of trace"))


class TestDriftComparison(unittest.TestCase):
    """Backtest vs live drift detection."""

    def setUp(self):
        # Baseline (backtest).
        self.bt = StrategyTrace(
            strategy_name="TestEA", run_id="bt-001", mode="backtest",
            symbol="EURUSD", timeframe="M15", started_at_ms=1719000000000,
        )
        self.bt.tick(1719000001000, bid=1.08500, ask=1.08503)
        self.bt.order_intent(symbol="EURUSD", side="buy", order_type="buy",
                             volume="0.10", price="0", ticket=100)
        self.bt.fill(ticket=100, symbol="EURUSD", side="buy",
                     volume="0.10", price="1.08503")
        self.bt.close_position(ticket=100, volume="0.10",
                               price="1.09000", pnl="49.70")

        # Compare (live).
        self.live = StrategyTrace(
            strategy_name="TestEA", run_id="live-001", mode="live",
            symbol="EURUSD", timeframe="M15", started_at_ms=1719000000000,
        )
        self.live.tick(1719000001000, bid=1.08500, ask=1.08503)
        self.live.order_intent(symbol="EURUSD", side="buy", order_type="buy",
                               volume="0.10", price="0", ticket=200)
        self.live.fill(ticket=200, symbol="EURUSD", side="buy",
                       volume="0.10", price="1.08503")
        self.live.close_position(ticket=200, volume="0.10",
                                 price="1.09000", pnl="49.70")

    def test_identical_traces_no_drift(self):
        report = compare_traces(self.bt, self.live)
        self.assertIsInstance(report, DriftReport)
        # 4 events in each, all matched.
        self.assertEqual(report.matched_events, 4)
        self.assertEqual(len(report.deviations), 0)
        self.assertAlmostEqual(report.drift_pct, 0.0)

    def test_missing_event_detected(self):
        # Remove the close from live trace.
        self.live.events.pop()  # remove close
        report = compare_traces(self.bt, self.live)
        self.assertGreater(len(report.deviations), 0)
        missing = [d for d in report.deviations if d["type"] == "missing_in_compare"]
        self.assertGreater(len(missing), 0)

    def test_extra_event_detected(self):
        # Add extra event to live trace.
        self.live.order_intent(symbol="EURUSD", side="sell", order_type="sell",
                               volume="0.20", price="0", ticket=300)
        report = compare_traces(self.bt, self.live)
        extra = [d for d in report.deviations if d["type"] == "extra_in_compare"]
        self.assertGreater(len(extra), 0)

    def test_price_delta_detected(self):
        # Change fill price in live trace.
        self.live.events[2].payload["price"] = "1.08600"  # was 1.08503
        report = compare_traces(self.bt, self.live)
        deltas = [d for d in report.deviations if d["type"] == "value_delta"]
        self.assertGreater(len(deltas), 0)

    def test_drift_report_serialization(self):
        report = compare_traces(self.bt, self.live)
        json_str = report.to_json()
        parsed = json.loads(json_str)
        self.assertEqual(parsed["matched"], 4)
        self.assertIn("drift_pct", parsed)

    def test_empty_traces(self):
        empty = StrategyTrace(
            strategy_name="Empty", run_id="e-001", mode="backtest",
            symbol="EURUSD", timeframe="M15", started_at_ms=0,
        )
        report = compare_traces(empty, empty)
        self.assertEqual(report.matched_events, 0)
        self.assertEqual(report.drift_pct, 0.0)


class TestFullEventChain(unittest.TestCase):
    """End-to-end event chain: tick → decision → intent → risk → fill → close."""

    def test_complete_chain(self):
        trace = StrategyTrace(
            strategy_name="FullChain", run_id="fc-001", mode="backtest",
            symbol="EURUSD", timeframe="M15", started_at_ms=1719000000000,
        )

        # Market data.
        trace.tick(1719000001000, bid=1.08497, ask=1.08503)
        trace.bar_close(1719000001000, "M15",
                        open_=1.08490, high=1.08520, low=1.08480,
                        close=1.08500, volume=100.0)
        trace.lifecycle("on_bar")

        # Strategy decision.
        trace.order_intent(symbol="EURUSD", side="buy", order_type="buy",
                           volume="0.10", price="0", sl="1.08000", tp="1.09500",
                           magic="42", comment="ema_cross_up", ticket=1001)

        # Risk gate.
        trace.risk_decision(allowed=True, ticket=1001)

        # Broker execution.
        trace.fill(ticket=1001, symbol="EURUSD", side="buy",
                   volume="0.10", price="1.08503", commission="0.70")

        # Later: close.
        trace.close_position(ticket=1001, volume="0.10",
                             price="1.09000", pnl="49.30", reason="signal")

        # Verify chain (7 events: tick, bar, on_bar, intent, risk, fill, close).
        events = trace.events
        self.assertEqual(len(events), 7)
        self.assertEqual(events[3].event, "order_intent")
        self.assertEqual(events[4].event, "risk_decision")
        self.assertEqual(events[5].event, "fill")
        self.assertEqual(events[6].event, "close")

        # Verify trace replay works.
        lines = replay_trace(trace)
        self.assertGreater(len(lines), 5)

        # Verify JSON round-trip.
        restored = StrategyTrace.from_json(trace.to_json())
        self.assertEqual(len(restored.events), 7)
        self.assertEqual(restored.events[5].event, "fill")


if __name__ == "__main__":
    unittest.main()
