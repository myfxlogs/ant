"""积木识别器单元测试。

用最小 MQL 片段验证每个识别器的匹配逻辑。
"""

import unittest

from tools.mql_transpiler.ast_bridge import parse_mql
from tools.mql_migration.expression_gen import ExpressionGen
from tools.mql_migration.recognizers.market_entry import recognize_market_entries
from tools.mql_migration.recognizers.pending_entry import recognize_pending_entries
from tools.mql_migration.recognizers.magic_exit import recognize_exit_rules
from tools.mql_migration.recognizers.exec_model import (
    recognize_execution_model,
    recognize_params,
    recognize_sizing,
    recognize_state_vars,
)
from tools.mql_migration.intent_ir import (
    ExecutionKind,
    ExitTrigger,
    OrderAction,
    SizingKind,
)


class TestMarketEntryRecognizer(unittest.TestCase):
    """市价单入场识别器。"""

    def setUp(self):
        self._expr_gen = ExpressionGen()

    def test_recognizes_buy_entry(self):
        mql = """
        extern double LotSize = 0.10;
        int OnInit() { return 0; }
        void OnTick() {
            double rsi = iRSI(Symbol(), 0, 14, PRICE_CLOSE, 1);
            if (rsi < 30) {
                OrderSend(Symbol(), OP_BUY, LotSize, Ask, 3, 0, 0, "buy", 1001, 0, clrNONE);
            }
        }
        """
        ast = parse_mql(mql)
        entries = recognize_market_entries(ast, self._expr_gen)
        self.assertEqual(len(entries), 1)
        self.assertEqual(entries[0].action, OrderAction.MARKET_BUY)

    def test_recognizes_sell_entry(self):
        mql = """
        extern double LotSize = 0.10;
        int OnInit() { return 0; }
        void OnTick() {
            double rsi = iRSI(Symbol(), 0, 14, PRICE_CLOSE, 1);
            if (rsi > 70) {
                OrderSend(Symbol(), OP_SELL, LotSize, Bid, 3, 0, 0, "sell", 1002, 0, clrNONE);
            }
        }
        """
        ast = parse_mql(mql)
        entries = recognize_market_entries(ast, self._expr_gen)
        self.assertEqual(len(entries), 1)
        self.assertEqual(entries[0].action, OrderAction.MARKET_SELL)

    def test_recognizes_else_if_chain(self):
        mql = """
        extern double LotSize = 0.10;
        int OnInit() { return 0; }
        void OnTick() {
            if (fast > slow) {
                OrderSend(Symbol(), OP_BUY, LotSize, Ask, 3, 0, 0, "buy", 1001, 0, clrNONE);
            } else if (fast < slow) {
                OrderSend(Symbol(), OP_SELL, LotSize, Bid, 3, 0, 0, "sell", 1001, 0, clrNONE);
            }
        }
        """
        ast = parse_mql(mql)
        entries = recognize_market_entries(ast, self._expr_gen)
        self.assertEqual(len(entries), 2)
        actions = {e.action for e in entries}
        self.assertEqual(actions, {OrderAction.MARKET_BUY, OrderAction.MARKET_SELL})

    def test_skips_non_entry_if(self):
        mql = """
        int OnInit() { return 0; }
        void OnTick() {
            if (fast <= 0.0) { return; }
        }
        """
        ast = parse_mql(mql)
        entries = recognize_market_entries(ast, self._expr_gen)
        self.assertEqual(len(entries), 0)

    def test_local_var_not_self_prefixed(self):
        mql = """
        extern double LotSize = 0.10;
        int OnInit() { return 0; }
        void OnTick() {
            double rsi = iRSI(Symbol(), 0, 14, PRICE_CLOSE, 1);
            bool shouldBuy = (rsi < 30);
            if (shouldBuy) {
                OrderSend(Symbol(), OP_BUY, LotSize, Ask, 3, 0, 0, "buy", 1001, 0, clrNONE);
            }
        }
        """
        ast = parse_mql(mql)
        entries = recognize_market_entries(ast, self._expr_gen)
        self.assertEqual(len(entries), 1)
        cond = entries[0].conditions[0].expr if entries[0].conditions else ""
        self.assertNotIn("self.shouldBuy", cond,
                         "Local var shouldBuy should NOT have self. prefix")


class TestPendingEntryRecognizer(unittest.TestCase):
    """挂单入场识别器。"""

    def setUp(self):
        self._expr_gen = ExpressionGen()

    def test_recognizes_grid_limit_orders(self):
        mql = """
        extern int GridLevels = 5;
        extern double LotSize = 0.10;
        int OnInit() { return 0; }
        void OnTick() {
            for (int i = 1; i <= GridLevels; i++) {
                OrderSend(Symbol(), OP_BUYLIMIT, LotSize, 1.1000, 3, 0, 1.1050, "grid", 2001+i, 0, clrNONE);
                OrderSend(Symbol(), OP_SELLLIMIT, LotSize, 1.1100, 3, 0, 1.1050, "grid", 2101+i, 0, clrNONE);
            }
        }
        """
        ast = parse_mql(mql)
        entries = recognize_pending_entries(ast, self._expr_gen)
        self.assertEqual(len(entries), 2)
        actions = {e.action for e in entries}
        self.assertEqual(actions, {OrderAction.BUY_LIMIT, OrderAction.SELL_LIMIT})

    def test_recognizes_breakout_stop_order(self):
        mql = """
        extern double LotSize = 0.10;
        int OnInit() { return 0; }
        void OnTick() {
            if (Close[0] > ema50) {
                OrderSend(Symbol(), OP_BUYSTOP, LotSize, 1.1050, 5, 0, 0, "breakout", 3001, 0, clrNONE);
            }
        }
        """
        ast = parse_mql(mql)
        entries = recognize_pending_entries(ast, self._expr_gen)
        self.assertEqual(len(entries), 1)
        self.assertEqual(entries[0].action, OrderAction.BUY_STOP)


class TestExitRecognizer(unittest.TestCase):
    """出场规则识别器。"""

    def setUp(self):
        self._expr_gen = ExpressionGen()

    def test_recognizes_magic_close(self):
        mql = """
        extern int MagicNumber = 1001;
        int OnInit() { return 0; }
        void OnTick() {}
        void OnDeinit(const int reason) {
            for (int i = 0; i < OrdersTotal(); i++) {
                if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) {
                    if (OrderMagicNumber() == MagicNumber) {
                        OrderClose(OrderTicket(), OrderLots(), Bid, 3);
                    }
                }
            }
        }
        """
        ast = parse_mql(mql)
        exits = recognize_exit_rules(ast, self._expr_gen)
        self.assertGreaterEqual(len(exits), 1)
        close_exits = [e for e in exits if e.trigger == ExitTrigger.MAGIC_CLOSE]
        self.assertGreaterEqual(len(close_exits), 1)

    def test_recognizes_magic_delete(self):
        mql = """
        extern int MagicBase = 2001;
        int OnInit() { return 0; }
        void OnTick() {}
        void OnDeinit(const int reason) {
            for (int i = 0; i < OrdersTotal(); i++) {
                if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) {
                    int magic = OrderMagicNumber();
                    if (magic >= MagicBase && magic <= MagicBase + 200) {
                        OrderDelete(OrderTicket());
                    }
                }
            }
        }
        """
        ast = parse_mql(mql)
        exits = recognize_exit_rules(ast, self._expr_gen)
        self.assertGreaterEqual(len(exits), 1)
        delete_exits = [e for e in exits if e.trigger == ExitTrigger.MAGIC_DELETE]
        self.assertGreaterEqual(len(delete_exits), 1)


class TestExecModelRecognizer(unittest.TestCase):
    """执行模型识别器。"""

    def test_grid_flag_detected(self):
        mql = """
        extern int GridLevels = 5;
        bool gridPlaced = false;
        int OnInit() { return 0; }
        void OnTick() {
            if (gridPlaced) { return; }
            for (int i = 1; i <= GridLevels; i++) {
                OrderSend(Symbol(), OP_BUYLIMIT, 0.1, 1.1000, 3, 0, 0, "", 2001, 0, clrNONE);
            }
            gridPlaced = true;
        }
        """
        ast = parse_mql(mql)
        exec_model = recognize_execution_model(ast)
        self.assertEqual(exec_model.kind, ExecutionKind.ON_INIT_GRID)

    def test_market_order_detected(self):
        mql = """
        extern double LotSize = 0.10;
        int OnInit() { return 0; }
        void OnTick() {
            if (Close[0] > Close[1]) {
                OrderSend(Symbol(), OP_BUY, LotSize, Ask, 3, 0, 0, "", 1001, 0, clrNONE);
            }
        }
        """
        ast = parse_mql(mql)
        exec_model = recognize_execution_model(ast)
        self.assertEqual(exec_model.kind, ExecutionKind.ON_BAR)


class TestSizingRecognizer(unittest.TestCase):
    """手数规则识别器。"""

    def test_fixed_sizing(self):
        mql = """
        extern double LotSize = 0.10;
        int OnInit() { return 0; }
        void OnTick() {}
        """
        ast = parse_mql(mql)
        sizing = recognize_sizing(ast)
        self.assertEqual(sizing.kind, SizingKind.FIXED)


class TestParamRecognition(unittest.TestCase):
    """参数提取。"""

    def test_extracts_extern_params(self):
        mql = """
        extern double LotSize = 0.10;
        extern int FastMA = 12;
        extern int SlowMA = 26;
        extern int MagicNumber = 1001;
        int OnInit() { return 0; }
        void OnTick() {}
        """
        ast = parse_mql(mql)
        params = recognize_params(ast)
        self.assertEqual(len(params), 4)


if __name__ == "__main__":
    unittest.main()
