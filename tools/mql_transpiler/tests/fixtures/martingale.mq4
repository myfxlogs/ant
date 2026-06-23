// Sample 3: Martingale (RSI-based entry with martingale position sizing)
// Corresponds to T0.4: martingale.py

extern int RSIPeriod = 14;
extern double RSIOverSold = 30.0;
extern double RSIOverBought = 70.0;
extern double BaseLot = 0.10;
extern double MaxLot = 5.0;
extern int MagicNumber = 3001;

double currentLot = 0.0;
int consecutiveLosses = 0;

int OnInit() {
    currentLot = BaseLot;
    consecutiveLosses = 0;
    return INIT_SUCCEEDED;
}

void OnTick() {
    double rsi = iRSI(Symbol(), 0, RSIPeriod, PRICE_CLOSE, 1);
    if (rsi <= 0.0) {
        return;
    }

    if (rsi < RSIOverSold) {
        OrderSend(Symbol(), OP_BUY, currentLot, Ask, 3, 0, 0, "martingale", MagicNumber, 0, clrNONE);
    } else if (rsi > RSIOverBought) {
        OrderSend(Symbol(), OP_SELL, currentLot, Bid, 3, 0, 0, "martingale", MagicNumber, 0, clrNONE);
    }
}

void OnDeinit(const int reason) {
    for (int i = 0; i < OrdersTotal(); i++) {
        if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) {
            if (OrderMagicNumber() == MagicNumber) {
                OrderClose(OrderTicket(), OrderLots(), Bid, 3);
            }
        }
    }
}
