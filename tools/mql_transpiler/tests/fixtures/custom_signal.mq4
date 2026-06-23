// Sample 5: Custom Signal (iCustom + multi-timeframe confirmation)
// Corresponds to T0.4: custom_signal.py

extern string CustomName = "SuperTrend";
extern int CustomPeriod = 10;
extern double CustomMultiplier = 3.0;
extern double LotSize = 0.10;
extern int DeviationPts = 30;
extern int MagicNumber = 5001;

double prevSignal = 0.0;

int OnInit() {
    prevSignal = 0.0;
    EventSetTimer(300);
    return INIT_SUCCEEDED;
}

void OnTick() {
    double main = iCustom(Symbol(), 0, CustomName, CustomPeriod, CustomMultiplier, 0, 1);
    double stopLevel = iCustom(Symbol(), 0, CustomName, CustomPeriod, CustomMultiplier, 1, 1);

    if (main == 0.0) {
        return;
    }

    double h4Ema = iMA(Symbol(), PERIOD_H4, 50, 0, MODE_EMA, PRICE_CLOSE);
    if (main > 0.0 && prevSignal <= 0.0 && Close[0] > h4Ema) {
        OrderSend(Symbol(), OP_BUYSTOP, LotSize, stopLevel, DeviationPts,
                  stopLevel - 200 * Point, stopLevel + 200 * Point,
                  "custom_signal", MagicNumber, 0, clrNONE);
    } else if (main < 0.0 && prevSignal >= 0.0) {
        OrderSend(Symbol(), OP_SELLSTOP, LotSize, stopLevel, DeviationPts,
                  stopLevel + 200 * Point, stopLevel - 200 * Point,
                  "custom_signal", MagicNumber, 0, clrNONE);
    }

    prevSignal = main;
}

void OnTimer() {
    if (AccountFreeMargin() < AccountBalance() * 0.5) {
        closeAllByMagic(MagicNumber);
    }
}

void closeAllByMagic(int magic) {
    for (int i = 0; i < OrdersTotal(); i++) {
        if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) {
            if (OrderMagicNumber() == magic) {
                OrderClose(OrderTicket(), OrderLots(), Bid, 3);
            }
        }
    }
}

void OnDeinit(const int reason) {
    EventKillTimer();
    closeAllByMagic(MagicNumber);
}
