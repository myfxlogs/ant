// Golden EA: T1 MACD Signal — uses iMACD with MODE_MAIN/MODE_SIGNAL.
// Must compile with 0 fatal blind spots and produce trades.
// Adversarial: delete MODE_SIGNAL from constants.go → signal=0=MODE_MAIN →
// macd==signal → no trades → CI red.
extern int MagicNumber = 10003;
extern double LotSize = 0.1;
int OnInit() { return 0; }
void OnBar()
{
    double macd = iMACD(Symbol(), 0, 12, 26, 9, PRICE_CLOSE, MODE_MAIN, 0);
    double signal = iMACD(Symbol(), 0, 12, 26, 9, PRICE_CLOSE, MODE_SIGNAL, 0);
    if (macd > signal)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "MACD", MagicNumber, 0, clrGreen);
    if (macd < signal)
        OrderSend(Symbol(), OP_SELL, LotSize, Bid, 5, 0, 0, "MACD", MagicNumber, 0, clrRed);
}
