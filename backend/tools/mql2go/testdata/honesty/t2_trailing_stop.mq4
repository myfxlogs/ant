// Golden EA: T2 Trailing Stop — uses OrderSelect, OrderType, OrderModify.
// Must compile with 0 fatal blind spots.
// Adversarial: break OrderSelect builtin → fatal blind spot → CI red.
extern int MagicNumber = 20002;
extern double LotSize = 0.1;
extern int TrailingStop = 50;
extern int MAPeriod = 14;
int OnInit() { return 0; }
void OnBar()
{
    double ma = iMA(Symbol(), 0, MAPeriod, 0, MODE_EMA, PRICE_CLOSE, 1);
    if (ma > Close[1] && OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "TS", MagicNumber, 0, clrGreen);
    for (int i = 0; i < OrdersTotal(); i++)
    {
        if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES))
        {
            if (OrderType() == OP_BUY)
            {
                double sl = OrderStopLoss();
                double newSL = Bid - TrailingStop * Point;
                if (sl == 0 || newSL > sl)
                    OrderModify(OrderTicket(), OrderOpenPrice(), newSL, OrderTakeProfit(), 0, clrGreen);
            }
        }
    }
}
