// Golden EA: T1 MA Crossover — uses iMA, OrderSend, clrGreen/clrRed.
// Must compile with 0 fatal blind spots and produce trades.
// Adversarial: delete MODE_EMA from constants.go → iMA mode=0 (wrong) →
// behavior changes or blind spot appears → CI red.
extern int MagicNumber = 10001;
extern double LotSize = 0.1;
extern int MAPeriod = 14;
int OnInit() { return 0; }
void OnBar()
{
    double ma = iMA(Symbol(), 0, MAPeriod, 0, MODE_EMA, PRICE_CLOSE, 1);
    double prevClose = Close[1];
    if (ma > prevClose)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "MA", MagicNumber, 0, clrGreen);
    if (ma < prevClose)
        OrderSend(Symbol(), OP_SELL, LotSize, Bid, 5, 0, 0, "MA", MagicNumber, 0, clrRed);
}
