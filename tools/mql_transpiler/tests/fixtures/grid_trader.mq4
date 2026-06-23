// Sample 2: Grid Trader (pending-order grid with magic-number tracking)
// Corresponds to T0.4: grid_trader.py

extern int GridSpacingPips = 20;
extern int GridLevels = 5;
extern double LotSize = 0.10;
extern int TPPips = 20;
extern int MagicBase = 2001;

bool gridPlaced = false;

int OnInit() {
    gridPlaced = false;
    return INIT_SUCCEEDED;
}

void OnTick() {
    if (gridPlaced) {
        return;
    }

    double point = MarketInfo(Symbol(), MODE_POINT);
    double pip = point * 10;
    double gridStep = pip * GridSpacingPips;
    double tpDistance = pip * TPPips;
    double currentBid = Bid;

    for (int i = 1; i <= GridLevels; i++) {
        double buyPrice = currentBid - gridStep * i;
        double sellPrice = currentBid + gridStep * i;

        OrderSend(Symbol(), OP_BUYLIMIT, LotSize, buyPrice, 3,
                  0, buyPrice + tpDistance, "grid_buy", MagicBase + i, 0, clrNONE);

        OrderSend(Symbol(), OP_SELLLIMIT, LotSize, sellPrice, 3,
                  0, sellPrice - tpDistance, "grid_sell", MagicBase + 100 + i, 0, clrNONE);
    }

    gridPlaced = true;
}

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
