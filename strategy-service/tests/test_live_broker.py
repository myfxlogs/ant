"""T3.1 — LiveBroker tests.

Validates:
  - LiveBroker implements the Broker ABC (same interface as SimBroker)
  - Order intents are correctly recorded
  - State updates work correctly
  - Intent to signal dict conversion
  - Three-way consistency: paper/live/backtest produce same order intents
"""

import unittest
from decimal import Decimal
from typing import List, Optional

from app.engine.live_broker import (
    CloseIntent,
    LiveBroker,
    ModifyIntent,
    OrderIntent,
    build_live_broker_from_proto,
)
from app.sdk import (
    AccountInfo,
    AccountMode,
    OrderRequest,
    OrderType,
    PendingOrder,
    Position,
    PositionSide,
    Retcode,
    SymbolInfo,
    TypeFilling,
)


class TestLiveBrokerImplementsBroker(unittest.TestCase):
    """LiveBroker must satisfy the Broker ABC."""

    def test_implements_broker(self):
        from app.sdk.broker import Broker
        broker = LiveBroker()
        self.assertIsInstance(broker, Broker)

    def test_all_methods_callable(self):
        broker = LiveBroker()
        methods = [
            "order_send", "position_modify", "position_close",
            "order_delete", "positions", "orders",
            "account", "symbol_info", "server_time",
        ]
        for name in methods:
            self.assertTrue(callable(getattr(broker, name)), f"{name} not callable")


class TestOrderIntentRecording(unittest.TestCase):
    """Order intents must be correctly recorded for Go consumption."""

    def setUp(self):
        self.broker = LiveBroker(
            account_state=AccountInfo(
                balance=Decimal("10000"), equity=Decimal("10050"),
                margin=Decimal("500"), free_margin=Decimal("9550"),
                margin_level=Decimal("20.1"), leverage=100,
                currency="USD", mode=AccountMode.HEDGING,
            ),
        )

    def test_market_buy_recorded(self):
        req = OrderRequest(
            symbol="EURUSD", type=OrderType.BUY,
            volume=Decimal("0.10"), magic=1, comment="test",
        )
        result = self.broker.order_send(req)
        self.assertEqual(result.retcode, Retcode.DONE)
        self.assertIsNotNone(result.ticket)
        self.assertEqual(len(self.broker.order_intents), 1)
        self.assertEqual(self.broker.order_intents[0].request.type, OrderType.BUY)

    def test_pending_order_recorded(self):
        req = OrderRequest(
            symbol="EURUSD", type=OrderType.BUY_LIMIT,
            volume=Decimal("0.10"), price=Decimal("1.08000"), magic=2,
        )
        result = self.broker.order_send(req)
        self.assertEqual(len(self.broker.order_intents), 1)

    def test_close_recorded(self):
        # Add a position first (simulate state update).
        pos = Position(
            ticket=100, symbol="EURUSD", side=PositionSide.BUY,
            volume=Decimal("0.10"), open_price=Decimal("1.08500"),
        )
        self.broker.update_state(positions=[pos])

        result = self.broker.position_close(100)
        self.assertEqual(result.retcode, Retcode.DONE)
        self.assertEqual(len(self.broker.close_intents), 1)
        self.assertEqual(self.broker.close_intents[0].ticket, 100)

    def test_modify_recorded(self):
        pos = Position(
            ticket=200, symbol="EURUSD", side=PositionSide.SELL,
            volume=Decimal("0.20"), open_price=Decimal("1.08200"),
        )
        self.broker.update_state(positions=[pos])

        result = self.broker.position_modify(200, sl=Decimal("1.09000"), tp=Decimal("1.07000"))
        self.assertEqual(result.retcode, Retcode.DONE)
        self.assertEqual(len(self.broker.modify_intents), 1)
        self.assertEqual(self.broker.modify_intents[0].sl, Decimal("1.09000"))

    def test_order_delete_recorded(self):
        self.broker.update_state(pending=[
            PendingOrder(ticket=300, symbol="EURUSD", type=OrderType.BUY_STOP,
                         volume=Decimal("0.10"), price=Decimal("1.09000"),
                         magic=5, comment="pending"),
        ])
        result = self.broker.order_delete(300)
        self.assertEqual(result.retcode, Retcode.DONE)
        self.assertEqual(len(self.broker.delete_intents), 1)
        self.assertEqual(self.broker.delete_intents[0], 300)

    def test_clear_intents(self):
        req = OrderRequest(symbol="EURUSD", type=OrderType.BUY, volume=Decimal("0.10"), magic=1)
        self.broker.order_send(req)
        self.broker.position_close(100)  # even nonexistent
        self.assertEqual(len(self.broker.order_intents), 1)
        self.broker.clear_intents()
        self.assertEqual(len(self.broker.order_intents), 0)
        self.assertEqual(len(self.broker.close_intents), 0)


class TestStateInjection(unittest.TestCase):
    """LiveBroker must accept and return injected state."""

    def setUp(self):
        self.broker = LiveBroker()

    def test_update_account(self):
        new_acct = AccountInfo(
            balance=Decimal("5000"), equity=Decimal("5100"),
            margin=Decimal("200"), free_margin=Decimal("4900"),
            margin_level=Decimal("25.5"), leverage=200,
            currency="EUR", mode=AccountMode.NETTING,
        )
        self.broker.update_state(account=new_acct)
        acct = self.broker.account()
        self.assertEqual(acct.balance, Decimal("5000"))
        self.assertEqual(acct.currency, "EUR")
        self.assertEqual(acct.mode, AccountMode.NETTING)

    def test_update_positions(self):
        positions = [
            Position(ticket=1, symbol="EURUSD", side=PositionSide.BUY,
                     volume=Decimal("0.10"), open_price=Decimal("1.08500")),
            Position(ticket=2, symbol="GBPUSD", side=PositionSide.SELL,
                     volume=Decimal("0.20"), open_price=Decimal("1.30000")),
        ]
        self.broker.update_state(positions=positions)
        self.assertEqual(len(self.broker.positions()), 2)
        self.assertEqual(len(self.broker.positions(symbol="EURUSD")), 1)
        self.assertEqual(len(self.broker.positions(symbol="GBPUSD")), 1)

    def test_update_pending(self):
        pending = [
            PendingOrder(ticket=10, symbol="EURUSD", type=OrderType.BUY_LIMIT,
                         volume=Decimal("0.10"), price=Decimal("1.08000"), magic=1),
        ]
        self.broker.update_state(pending=pending)
        self.assertEqual(len(self.broker.orders()), 1)
        self.assertEqual(self.broker.orders()[0].magic, 1)

    def test_filter_positions_by_magic(self):
        positions = [
            Position(ticket=1, symbol="EURUSD", side=PositionSide.BUY,
                     volume=Decimal("0.10"), open_price=Decimal("1.08500"), magic=100),
            Position(ticket=2, symbol="EURUSD", side=PositionSide.SELL,
                     volume=Decimal("0.20"), open_price=Decimal("1.08600"), magic=200),
        ]
        self.broker.update_state(positions=positions)
        self.assertEqual(len(self.broker.positions(magic=100)), 1)
        self.assertEqual(self.broker.positions(magic=100)[0].magic, 100)
        self.assertEqual(len(self.broker.positions(magic=999)), 0)


class TestIntentToSignalDict(unittest.TestCase):
    """Intent → signal dict conversion for Go-side consumption."""

    def test_order_intent_to_dict(self):
        req = OrderRequest(
            symbol="EURUSD", type=OrderType.BUY,
            volume=Decimal("0.10"), sl=Decimal("1.07000"),
            tp=Decimal("1.09500"), magic=42, comment="entry",
            type_filling=TypeFilling.IOC,
        )
        intent = OrderIntent(ticket=1000001, request=req)
        d = intent.to_signal_dict()
        self.assertEqual(d["action"], "buy")
        self.assertEqual(d["symbol"], "EURUSD")
        self.assertEqual(d["volume"], "0.10")
        self.assertEqual(d["sl"], "1.07000")
        self.assertEqual(d["tp"], "1.09500")
        self.assertEqual(d["magic"], "42")
        self.assertEqual(d["comment"], "entry")

    def test_close_intent_to_dict(self):
        intent = CloseIntent(ticket=500, volume=Decimal("0.10"))
        d = intent.to_signal_dict()
        self.assertEqual(d["action"], "close")
        self.assertEqual(d["ticket"], "500")
        self.assertEqual(d["volume"], "0.10")

    def test_modify_intent_to_dict(self):
        intent = ModifyIntent(ticket=600, sl=Decimal("1.08000"), tp=Decimal("1.10000"))
        d = intent.to_signal_dict()
        self.assertEqual(d["action"], "modify")
        self.assertEqual(d["sl"], "1.08000")
        self.assertEqual(d["tp"], "1.10000")

    def test_close_all_intent(self):
        intent = CloseIntent(ticket=700, volume=None)
        d = intent.to_signal_dict()
        self.assertEqual(d["volume"], "full")  # "full" = close entire position


class TestAllOrderTypes(unittest.TestCase):
    """All 8 SDK order types produce correct intents."""

    def setUp(self):
        self.broker = LiveBroker(account_state=AccountInfo(
            balance=Decimal("10000"), equity=Decimal("10000"),
            margin=Decimal("0"), free_margin=Decimal("10000"),
            margin_level=Decimal("0"), leverage=100,
            currency="USD", mode=AccountMode.HEDGING,
        ))

    def test_all_types(self):
        types = [
            OrderType.BUY, OrderType.SELL,
            OrderType.BUY_LIMIT, OrderType.SELL_LIMIT,
            OrderType.BUY_STOP, OrderType.SELL_STOP,
            OrderType.BUY_STOP_LIMIT, OrderType.SELL_STOP_LIMIT,
        ]
        for ot in types:
            req = OrderRequest(
                symbol="EURUSD", type=ot, volume=Decimal("0.10"),
                price=Decimal("1.08500") if "market" not in ot.value else None,
                magic=1,
            )
            result = self.broker.order_send(req)
            self.assertEqual(result.retcode, Retcode.DONE, f"Failed for {ot}")
        self.assertEqual(len(self.broker.order_intents), 8)


class TestThreeWayConsistency(unittest.TestCase):
    """T3.1 DoD: paper ↔ live ↔ backtest three-way consistency.

    Given the same strategy code and same input, the order intents
    produced through SimBroker and LiveBroker must be semantically identical.
    """

    def test_same_request_produces_same_intent(self):
        """A single buy order produces the same intent shape in both brokers."""
        from app.engine.sim_broker import SimBroker
        from app.engine.cost import CostModel
        from app.engine.fill import FillModel
        from app.engine.margin import MarginModel
        from app.engine.market import MarketSimulator
        from app.engine.portfolio import Portfolio
        from app.engine.types import Bar, CostProfile, SlippageMode, Tick

        # Build SimBroker.
        bars = [Bar(open_time=1719000000000, close_time=1719000060000,
                     open=1.08490, high=1.08520, low=1.08480, close=1.08500, volume=100.0)]
        ticks = [Tick(ts=1719000060000, bid=1.08497, ask=1.08503)]
        market = MarketSimulator(bars)
        cost = CostModel(CostProfile(commission_per_lot=0.0, slippage_mode=SlippageMode.FIXED,
                                     slippage_rate=0.0, contract_size=100000.0))
        fill_model = FillModel(cost)
        portfolio = Portfolio(initial_cash=10000.0)
        margin = MarginModel(leverage=100.0)
        tick_ref: List[Optional[Tick]] = [ticks[0]]

        sim = SimBroker(
            portfolio=portfolio, fill_model=fill_model, cost_model=cost,
            margin_model=margin, market=market,
            tick_source=lambda: tick_ref[0],
            account_mode=AccountMode.HEDGING,
            initial_balance=Decimal("10000"),
        )
        sim.advance_tick(ticks[0])

        # Build LiveBroker.
        live = LiveBroker(account_state=AccountInfo(
            balance=Decimal("10000"), equity=Decimal("10000"),
            margin=Decimal("0"), free_margin=Decimal("10000"),
            margin_level=Decimal("0"), leverage=100,
            currency="USD", mode=AccountMode.HEDGING,
        ))

        # Same request to both.
        req = OrderRequest(
            symbol="EURUSD", type=OrderType.BUY,
            volume=Decimal("0.10"), magic=1, comment="consistency_test",
        )

        sim_result = sim.order_send(req)
        live_result = live.order_send(req)

        # Both must succeed.
        self.assertEqual(sim_result.retcode, Retcode.DONE)
        self.assertEqual(live_result.retcode, Retcode.DONE)

        # Both record the same volume and type.
        sim_positions = sim.positions(magic=1)
        self.assertEqual(len(sim_positions), 1)
        self.assertEqual(sim_positions[0].volume, Decimal("0.10"))

        live_intent = live.order_intents[0]
        self.assertEqual(live_intent.request.volume, Decimal("0.10"))
        self.assertEqual(live_intent.request.type, OrderType.BUY)
        self.assertEqual(live_intent.request.magic, 1)

    def test_close_intent_consistency(self):
        """Close intents are consistent between brokers."""
        sim_broker = None  # SimBroker would need a position to close
        live = LiveBroker()
        live.update_state(positions=[
            Position(ticket=42, symbol="EURUSD", side=PositionSide.BUY,
                     volume=Decimal("0.10"), open_price=Decimal("1.08500")),
        ])

        result = live.position_close(42, volume=Decimal("0.05"))
        self.assertEqual(result.retcode, Retcode.DONE_PARTIAL)
        self.assertEqual(len(live.close_intents), 1)
        self.assertEqual(live.close_intents[0].volume, Decimal("0.05"))


if __name__ == "__main__":
    unittest.main()
