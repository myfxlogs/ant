// Sample 1: Single MA Cross (EMA crossover trend-follower)
// Corresponds to T0.4: single_ma_cross.py

extern double LotSize = 0.10;
extern int FastMAPeriod = 12;
extern int SlowMAPeriod = 26;
extern int MagicNumber = 1001;

double prevFastMA = 0.0;
double prevSlowMA = 0.0;
bool hasPosition = false;

int OnInit() {
    prevFastMA = 0.0;
    prevSlowMA = 0.0;
    hasPosition = false;
    return INIT_SUCCEEDED;
}

void OnTick() {
    double fast = iMA(Symbol(), 0, FastMAPeriod, 0, MODE_EMA, PRICE_CLOSE);
    double slow = iMA(Symbol(), 0, SlowMAPeriod, 0, MODE_EMA, PRICE_CLOSE);

    if (fast <= 0.0 || slow <= 0.0) {
        return;
    }

    bool crossUp = (prevFastMA <= prevSlowMA && fast > slow);
    bool crossDown = (prevFastMA >= prevSlowMA && fast < slow);

    prevFastMA = fast;
    prevSlowMA = slow;

    if (crossUp) {
        closeExisting();
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 3, 0, 0, "ema_cross", MagicNumber, 0, clrNONE);
    } else if (crossDown) {
        closeExisting();
        OrderSend(Symbol(), OP_SELL, LotSize, Bid, 3, 0, 0, "ema_cross", MagicNumber, 0, clrNONE);
    }
}

void closeExisting() {
    for (int i = 0; i < OrdersTotal(); i++) {
        if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) {
            if (OrderMagicNumber() == MagicNumber) {
                OrderClose(OrderTicket(), OrderLots(), Bid, 3);
            }
        }
    }
}

void OnDeinit(const int reason) {
    closeExisting();
}
