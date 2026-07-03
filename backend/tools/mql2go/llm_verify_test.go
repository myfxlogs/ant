package mql2go

import "testing"

func TestLLMGeneratedRSI_EA(t *testing.T) {
	code := `#property strict

extern int RSIPeriod = 14;
extern double Lot = 0.1;

int Slippage = 3;
int MagicNumber = 123456;

int OnInit(){return(INIT_SUCCEEDED);}

void OnTick(){
   double rsi = iRSI(Symbol(), 0, RSIPeriod, PRICE_CLOSE, 0);
   if (CountOpenPositions() > 0) return;
   if (rsi < 30.0)
      OrderSend(Symbol(), OP_BUY, Lot, Ask, Slippage, 0, 0, "RSI Buy", MagicNumber, 0, clrBlue);
   else if (rsi > 70.0)
      OrderSend(Symbol(), OP_SELL, Lot, Bid, Slippage, 0, 0, "RSI Sell", MagicNumber, 0, clrRed);
}

int CountOpenPositions(){
   int count = 0;
   for (int i = OrdersTotal() - 1; i >= 0; i--){
      if (!OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) continue;
      if (OrderSymbol() != Symbol()) continue;
      if (OrderMagicNumber() != MagicNumber) continue;
      if (OrderType() == OP_BUY || OrderType() == OP_SELL) count++;
   }
   return count;
}`

	runner, err := CompileMQL(code)
	if err != nil {
		t.Fatalf("FAIL: %v", err)
	}
	bc := runner.Bytecode()
	t.Logf("PASS — %d instructions, OnInit=%d OnTick=%d",
		len(bc.Code), bc.OnInit, bc.OnTick)
	params := ExtractParamInfos(bc)
	t.Logf("Params: %d", len(params))
	for _, p := range params {
		t.Logf("  %s (%s) = %s", p.Name, p.Type, p.Default)
	}
}
