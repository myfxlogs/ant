// Sample 4: Hedge Twins (simultaneous long+short in hedging mode)
// Corresponds to T0.4: hedge_twins.py

extern int TrendMA = 50;
extern double TrendLot = 0.20;
extern int MagicTrend = 4001;
extern int MRRSIPeriod = 7;
extern double MRLot = 0.10;
extern int MagicMR = 4002;

int OnInit() {
    return INIT_SUCCEEDED;
}

void OnTick() {
    double ema = iMA(Symbol(), 0, TrendMA, 0, MODE_EMA, PRICE_CLOSE);
    if (ema > 0.0 && Close[0] > ema) {
        OrderSend(Symbol(), OP_BUY, TrendLot, Ask, 3, 0, 0, "trend_long", MagicTrend, 0, clrNONE);
    } else if (ema > 0.0 && Close[0] <= ema) {
        closeByMagic(MagicTrend);
    }

    double rsi = iRSI(Symbol(), 0, MRRSIPeriod, PRICE_CLOSE, 1);
    if (rsi > 70.0) {
        OrderSend(Symbol(), OP_SELL, MRLot, Bid, 3, 0, 0, "mr_short", MagicMR, 0, clrNONE);
    } else if (rsi < 50.0) {
        closeByMagic(MagicMR);
    }
}

void closeByMagic(int magic) {
    for (int i = 0; i < OrdersTotal(); i++) {
        if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) {
            if (OrderMagicNumber() == magic) {
                OrderClose(OrderTicket(), OrderLots(), Bid, 3);
            }
        }
    }
}

void OnDeinit(const int reason) {
    closeByMagic(MagicTrend);
    closeByMagic(MagicMR);
}
