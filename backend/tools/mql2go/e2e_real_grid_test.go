package mql2go

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// Real-world MQL4 grid EA source — stripped of Object*/Button* UI functions
// that are permanent blind spots (MT client-side only). Core trading logic intact.
const realGridEA = `
extern string 加载货币M1 = "EURUSD GBPUSD";
enum BuyOrSell0 {只做多 = 0,只做空 = 1,多空都做 = 2};
input BuyOrSell0 多空方向 = 2;
extern bool 启用区间控制 = false;
extern bool 启用顺势加码 = false;
extern bool 启用虚拟下单 = false;
extern int 虚拟下单单数 = 10;
extern int 单向最大单数 = 25;
extern int 顺势最大单数 = 50;
extern double 资金5W单笔最大下单量 = 3;
extern double 起始下单量 = 0.01;
extern double 翻倍 = 1.5;
extern int 区间K线数 = 8;
extern double 逆势回调点数 = 1;
extern double 逆势加码间隔 = 5;
extern double 逆势加码间隔递减 = 0.1;
extern double 逆势加码间隔最小 = 1;
extern double 顺势加码间隔 = 9;
extern double 逆势盈利点数 = 3;
extern double 逆势盈利点数递减 = 0.1;
extern double 逆势盈利点数最小 = 0.5;
extern double 顺势盈利点数 = 10;
extern double 整体移动止损点数 = 0;
extern double 总亏损平仓 = 0.0;
extern double 总盈利平仓 = 0.0;
extern int 滑点 = 3;
extern int 定单识别码 = 10533;
extern string 定单注释 = "ok";
extern bool 启用时间控制 = false;
extern int 开始小时 = 8;
extern int 开始分钟 = 0;
extern int 结束小时 = 19;
extern int 结束分钟 = 0;
extern bool 启用iBands控制 = false;
extern int iBands_平均周期 = 20;
extern double iBands_偏差 = 2;
extern int iBands_平移 = 0;
extern bool 启用iRSI控制 = false;
extern int iRSI_平均周期 = 14;
extern double iRSI_上线 = 70;
extern double iRSI_下线 = 30;
extern bool 启用iStochastic控制 = false;
extern int iStoc_K周期 = 5;
extern int iStoc_D周期 = 3;
extern int iStoc_慢速 = 3;
extern double iStoc_上线 = 80;
extern double iStoc_下线 = 20;
extern bool 启用iCCI控制 = false;
extern int iCCI_平均周期 = 14;
extern double iCCI_上线 = 100;
extern double iCCI_下线 = -100;

int Gi_268 = 0;
int Gi_272 = 0;
double Gd_168, Gd_176, Gd_216, Gd_218;
int G_magic_340 = 10533;
string G_comment_344 = "";
int Gi_276 = 1;
double Gd_284;
int Gi_336 = 3;
double LowPrice = 0;
double HighPrice = 10000;
double STTPPrice0 = 0;
double STTPPrice1 = 100000;
bool OpenNew = true;
bool OpenAdd0, OpenAdd1;
bool BoolClose;
double G_lots_290 = 0.02;
double G_lots_292 = 0.02;
double Gd_300 = 1.5;
int Gi_308 = 25;
int Gi_307 = 50;
int Gi_309 = 10;
double Gi_311 = 1;
double Gi_312 = 5;
double Gi_320 = 0.1;
double Gi_321 = 1;
double Gi_313 = 9;
double Gi_316 = 3;
double Gi_317 = 10;
double Gi_318 = 0.1;
double Gi_319 = 0.5;
double Gi_322 = 0;
double Gd_320 = 0;
double Gd_328 = 0;
int Gi_306 = 7;
bool Gi_310 = false;
bool Gi_315 = true;
double STOPLEVEL;
double MinLot, MaxLot;
int Gi_280;
double icustom0, icustom1, icustom2, icustom3, icustom4;
int Bands_period = 20;
double deviation = 2;
int bands_shift = 0;
int rsi_period = 14;
double RSI_High = 70, RSI_Low = 30;
int Kperiod = 5, Dperiod = 3, kd_slowing = 3;
double KD_High = 80, KD_Low = 20;
int cci_period = 14;
double CCI_High = 100, CCI_Low = -100;
bool booliBands, booliRSI, booliStochastic, booliCCI;
bool Use_Time, Use_Zone, Use_Msg;
int StartHour, StartMinit, EndHour, EndMinit;
int Gi_269 = 2, Gi_265 = 8;
double Gd_217, Gd_219;
double G_order_open_price_150, G_order_open_price_152, G_order_open_price_153;
double G_order_open_price_158, G_order_open_price_160, G_order_open_price_161;
double Gd_184, Gd_192, Gd_200, Gd_208;
int Gi_260, Gi_261;
bool TrackZoer0 = true, TrackZoer1 = true;
bool TrackZoerXuNi0, TrackZoerXuNi1;
double Gd_720[30], Gd_730[30], Gd_721[30], Gd_731[30];
int Gi_720 = 0, Gi_721 = 0, Gi_323 = 0, Gi_324 = 0;
int G_pos_260, G_ticket_264;
bool Gi_256;
double G_price_232;
double LotsAll, LotsC0, Lots0;
int l_pod_110;
double HighPrice0, LowPrice0;
int Day0 = -1;
bool TrackXuNi = false;

int OnInit() {
   G_comment_344 = "ok_";
   Gd_284 = MathPow(0.1, Digits);
   if (Digits == 5 || Digits == 3) Gi_276 = 10;
   EventSetMillisecondTimer(300);
   LowPrice = Ask;
   HighPrice = Bid;
   Day0 = Day();
   STOPLEVEL = MarketInfo(Symbol(),MODE_STOPLEVEL);
   MinLot = MarketInfo(Symbol(),MODE_MINLOT);
   MaxLot = MarketInfo(Symbol(),MODE_MAXLOT);
   Gi_280 = MathRound((-MathLog(MarketInfo(Symbol(), MODE_LOTSTEP))) / 2.302585093);
   return(0);
}

void OnDeinit(const int reason) {
   Comment("");
   EventKillTimer();
}

void OnTick() {
   if(!IsTesting() && MathAbs(Day() - Day0) > 5) { Day0 = Day(); }
   CountOrders();
   if(Use_Zone && Gi_268 == 0) {
     LowPrice0 = Low[iLowest(NULL,0,MODE_LOW,Gi_265,2)];
     if(Bid < LowPrice0) { if (Ask < LowPrice) LowPrice = Ask; } else LowPrice = Ask;
   } else if (Ask < LowPrice) LowPrice = Ask;
   if(Use_Zone && Gi_272 == 0) {
     HighPrice0 = High[iHighest(NULL,0,MODE_HIGH,Gi_265,2)];
     if(Bid > HighPrice0) { if(Bid > HighPrice) HighPrice = Bid; } else HighPrice = Bid;
   } else if (Bid > HighPrice) HighPrice = Bid;

   icustom0 = iBands(NULL,0,Bands_period, deviation,bands_shift,PRICE_CLOSE,1,0);
   icustom1 = iBands(NULL,0,Bands_period, deviation,bands_shift,PRICE_CLOSE,2,0);
   icustom2 = iRSI(NULL,0,rsi_period,PRICE_CLOSE,0);
   icustom3 = iStochastic(NULL,0,Kperiod,Dperiod,kd_slowing,MODE_SMA,STO_LOWHIGH,0,0);
   icustom4 = iCCI(NULL,0,cci_period,PRICE_CLOSE,0);
   HideTestIndicators(true);

   if(Gi_322 > 0) {
     if (Gi_268 == 0) STTPPrice0 = 0;
     if (Gi_268 > 0 && Bid > Gd_168 + Gi_322 * Gi_276 * Gd_284) {
        STTPPrice0 = MathMax(STTPPrice0,Bid - Gi_322 * Gi_276 * Gd_284);
        O_Modify1(0,Bid - Gi_322 * Gi_276 * Gd_284);
     }
     if (Gi_272 == 0) STTPPrice1 = 100000;
     if (Gi_272 > 0 && Ask < Gd_176 - Gi_322 * Gi_276 * Gd_284) {
        STTPPrice1 = MathMin(STTPPrice1,Ask + Gi_322 * Gi_276 * Gd_284);
        O_Modify1(1,Ask + Gi_322 * Gi_276 * Gd_284);
     }
     if (Gi_268 > 0 && Bid < STTPPrice0) CloseOrders(OP_BUY);
     if (Gi_272 > 0 && Ask > STTPPrice1) CloseOrders(OP_SELL);
   }
   BoolClose = false;
   if ((Gd_328 > 0.0 && Gd_216 >= Gd_328) || (Gd_320 > 0.0 && Gd_216 <= (-Gd_320))) CloseOrders(OP_BUY);
   if ((Gd_328 > 0.0 && Gd_218 >= Gd_328) || (Gd_320 > 0.0 && Gd_218 <= (-Gd_320))) CloseOrders(OP_SELL);
   if (Gi_268 > 0 && Bid > Gd_168 + MathMax(Gi_319,Gi_316 - Gi_318*(Gi_268 - 1)) * Gi_276 * Gd_284) CloseOrders(OP_BUY);
   if (Gi_272 > 0 && Ask < Gd_176 - MathMax(Gi_319,Gi_316 - Gi_318*(Gi_272 - 1)) * Gi_276 * Gd_284) CloseOrders(OP_SELL);
   if (BoolClose) return;

   G_lots_290 = NormalizeDouble(AccountBalance() / 50000 * 3, Gi_280);
   if (Gi_268 == 0 && (Gi_269 == 0 || Gi_269 == 2) && (OpenNew && CheckTime() && Ask - LowPrice >= Gi_311 * Gi_276 * Gd_284)) {
      G_ticket_264 = OrderSend(Symbol(), OP_BUY, G_lots_292, Ask, Gi_336 * Gi_276, 0, 0, G_comment_344, G_magic_340, 0, Blue);
   }
   if (Gi_272 == 0 && (Gi_269 == 1 || Gi_269 == 2) && (OpenNew && CheckTime() && HighPrice - Bid >= Gi_311 * Gi_276 * Gd_284)) {
      G_ticket_264 = OrderSend(Symbol(), OP_SELL, G_lots_292, Bid, Gi_336 * Gi_276, 0, 0, G_comment_344, G_magic_340, 0, Red);
   }
   if (Gi_268 > 0 && Gi_268 < Gi_308 && Ask - LowPrice >= Gi_311 * Gi_276 * Gd_284 && G_order_open_price_152 - Ask >= MathMax(Gi_321,Gi_312 - Gi_320 * (Gi_268 - 1)) * Gi_276 * Gd_284) {
      LotsAll = MathMin(G_lots_290,NormalizeDouble(G_lots_292 * MathPow(Gd_300, Gi_268), 4));
      LotsAll = NormalizeDouble(LotsAll, Gi_280);
      Lots0 = MathMin(LotsAll,MaxLot);
      G_ticket_264 = OrderSend(Symbol(), OP_BUY, Lots0, Ask, Gi_336 * Gi_276, 0, 0, G_comment_344, G_magic_340, 0, Blue);
   }
   if (Gi_272 > 0 && Gi_272 < Gi_308 && HighPrice - Bid >= Gi_311 * Gi_276 * Gd_284 && Bid - G_order_open_price_160 >= MathMax(Gi_321,Gi_312 - Gi_320 * (Gi_272 - 1)) * Gi_276 * Gd_284) {
      LotsAll = MathMin(G_lots_290,NormalizeDouble(G_lots_292 * MathPow(Gd_300, Gi_272), 4));
      LotsAll = NormalizeDouble(LotsAll, Gi_280);
      Lots0 = MathMin(LotsAll,MaxLot);
      G_ticket_264 = OrderSend(Symbol(), OP_SELL, Lots0, Bid, Gi_336 * Gi_276, 0, 0, G_comment_344, G_magic_340, 0, Red);
   }
}

void O_Modify(int Ai_0) {
   if (Ai_0 == 0) {
      if(Gi_268 > 1) G_price_232 = Gd_168 + MathMax(Gi_319,Gi_316 - Gi_318*(Gi_268 - 1)) * Gi_276 * Gd_284;
      if (G_price_232 - Bid <= STOPLEVEL * Point) return;
      for (G_pos_260 = OrdersTotal() - 1; G_pos_260 >= 0; G_pos_260--) {
         if (OrderSelect(G_pos_260, SELECT_BY_POS, MODE_TRADES)) {
            if (OrderSymbol() == Symbol() && OrderMagicNumber() == G_magic_340)
               if (OrderType() == OP_BUY) Gi_256 = OrderModify(OrderTicket(), OrderOpenPrice(), 0, G_price_232, 0, Blue);
         }
      }
   }
}

void O_Modify1(int Ai_0, double Ad_0) {
   if (Ai_0 == 0) {
      for (G_pos_260 = OrdersTotal() - 1; G_pos_260 >= 0; G_pos_260--) {
         if (OrderSelect(G_pos_260, SELECT_BY_POS, MODE_TRADES)) {
            if (OrderSymbol() == Symbol() && OrderMagicNumber() == G_magic_340)
               if (OrderType() == OP_BUY && OrderStopLoss() < Ad_0) Gi_256 = OrderModify(OrderTicket(), OrderOpenPrice(), Ad_0, OrderTakeProfit(), 0, Blue);
         }
      }
   }
}

bool CheckTime() {
   if(!Use_Time) return(1);
   return(0);
}

void CloseOrders(int A_cmd_0) {
   BoolClose = true;
   for (int pos_4 = OrdersTotal() - 1; pos_4 >= 0; pos_4--) {
      if (OrderSelect(pos_4, SELECT_BY_POS, MODE_TRADES)) {
         if (OrderSymbol() == Symbol() && OrderMagicNumber() == G_magic_340)
            if (OrderType() == A_cmd_0) Gi_256 = OrderClose(OrderTicket(), OrderLots(), OrderClosePrice(), Gi_336 * Gi_276, Yellow);
      }
   }
}

void CountOrders() {
   Gi_268 = Gi_720;
   Gi_272 = Gi_721;
   Gd_184 = 0; Gd_192 = 0; Gd_200 = 0; Gd_208 = 0;
   Gd_216 = 0; Gd_218 = 0;
   Gi_260 = 0; Gi_261 = 0;
   G_order_open_price_152 = 10000;
   G_order_open_price_153 = 0;
   G_order_open_price_160 = 0;
   G_order_open_price_161 = 10000;
   for (G_pos_260 = 0; G_pos_260 < OrdersTotal(); G_pos_260++) {
      if (OrderSelect(G_pos_260, SELECT_BY_POS, MODE_TRADES)) {
         if (OrderSymbol() == Symbol() && OrderMagicNumber() == G_magic_340) {
            if (OrderType() == OP_BUY) {
               Gi_268++;
               Gi_260++;
               Gd_184 += OrderLots();
               Gd_200 += OrderOpenPrice() * OrderLots();
               Gd_216 += OrderProfit() + OrderSwap() + OrderCommission();
               G_order_open_price_150 = OrderOpenPrice();
               G_order_open_price_152 = MathMin(G_order_open_price_152,OrderOpenPrice());
               G_order_open_price_153 = MathMax(G_order_open_price_153,OrderOpenPrice());
            }
            if (OrderType() == OP_SELL) {
               Gi_272++;
               Gi_261++;
               Gd_192 += OrderLots();
               Gd_208 += OrderOpenPrice() * OrderLots();
               Gd_218 += OrderProfit() + OrderSwap() + OrderCommission();
               G_order_open_price_158 = OrderOpenPrice();
               G_order_open_price_160 = MathMax(G_order_open_price_160,OrderOpenPrice());
               G_order_open_price_161 = MathMin(G_order_open_price_161,OrderOpenPrice());
            }
         }
      }
   }
   if (Gd_184 > 0.0) Gd_168 = NormalizeDouble(Gd_200 / Gd_184, Digits);
   if (Gd_192 > 0.0) Gd_176 = NormalizeDouble(Gd_208 / Gd_192, Digits);
}
`

func TestRealGridEA_Analyze(t *testing.T) {
	ir, err := CompileToIR(realGridEA)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	rep := interp.Analyze(ir)

	fmt.Printf("\n══════════════════════════════════════════════════\n")
	fmt.Printf("  真实网格 EA 分析报告\n")
	fmt.Printf("══════════════════════════════════════════════════\n")
	fmt.Printf("版本:       %s\n", rep.Version)
	fmt.Printf("执行模式:   %s\n", rep.ExecKind)
	fmt.Printf("覆盖率:     %.1f%%\n", rep.Coverage*100)
	fmt.Printf("总调用:     %d\n", rep.TotalCalls)
	fmt.Printf("已支持:     %d\n", rep.SupportedCalls)
	fmt.Printf("参数:       %d\n", len(rep.Params))
	fmt.Printf("指标:       %v\n", rep.Indicators)
	fmt.Printf("入场规则:   %d\n", rep.EntryRules)

	fmt.Printf("\n── Blind Spots ──\n")
	for _, bs := range rep.BlindSpots {
		fmt.Printf("  %-40s [%s] count=%d\n", bs.Builtin, bs.Severity, bs.Count)
	}
	if len(rep.BlindSpots) == 0 {
		fmt.Printf("  (无)\n")
	}

	fmt.Printf("\n── 用户函数 ──\n")
	for name := range ir.Funcs {
		fmt.Printf("  %s()\n", name)
	}
	fmt.Printf("══════════════════════════════════════════════════\n\n")

	if rep.Version != "mql4" {
		t.Errorf("expected mql4, got %s", rep.Version)
	}
}

func TestRealGridEA_Backtest(t *testing.T) {
	runner, err := CompileMQL(realGridEA)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	bars := makeRealGridBars(100)

	cfg := backtest.Config{
		Symbol:         "EURUSD",
		Timeframe:      "M15",
		InitialCapital: decimal.NewFromInt(50000),
		Leverage:       100,
		Params: map[string]string{
			"定单识别码":           "10533",
			"起始下单量":            "0.01",
			"翻倍":              "1.5",
			"逆势加码间隔":           "5",
			"逆势盈利点数":           "3",
			"顺势盈利点数":           "10",
			"单向最大单数":           "25",
		},
	}

	engine := backtest.New(cfg, runner, bars)
	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("backtest failed: %v", err)
	}

	fmt.Printf("\n══════════════════════════════════════════════════\n")
	fmt.Printf("  真实网格 EA 回测结果\n")
	fmt.Printf("══════════════════════════════════════════════════\n")
	fmt.Printf("K线数:       %d\n", len(bars))
	fmt.Printf("Equity点数:  %d\n", len(result.Equity))
	fmt.Printf("交易数:      %d\n", len(result.Trades))

	if len(result.Equity) > 0 {
		first := result.Equity[0].Equity
		last := result.Equity[len(result.Equity)-1].Equity
		fmt.Printf("初始资金:    %s\n", first.String())
		fmt.Printf("最终资金:    %s\n", last.String())
	}

	// Check broker internal state
	broker := engine.Broker()
	openPos := broker.Positions(0)
	history := broker.HistoryOrders(0, 0)
	fmt.Printf("未平仓:      %d\n", len(openPos))
	fmt.Printf("已平仓:      %d\n", len(history))
	for _, p := range openPos {
		fmt.Printf("  open: ticket=%d side=%d vol=%s price=%s\n", p.Ticket, p.Side, p.Volume.String(), p.OpenPrice.String())
	}
	for _, h := range history {
		fmt.Printf("  closed: ticket=%d side=%d vol=%s profit=%s\n", h.Ticket, h.Side, h.Volume.String(), h.Profit.String())
	}

	blinds := runner.GetRuntimeBlindSpots()
	fmt.Printf("\n运行时 Blind Spots: %d\n", len(blinds))
	for _, bs := range blinds {
		fmt.Printf("  %s (count=%d)\n", bs.Builtin, bs.Count)
	}
	fmt.Printf("══════════════════════════════════════════════════\n\n")

	if len(result.Equity) == 0 {
		t.Fatal("expected equity points from backtest")
	}

	for _, bs := range blinds {
		if bs.Severity == interp.SeverityFatal {
			t.Errorf("fatal blind spot in backtest: %s (count=%d)", bs.Builtin, bs.Count)
		}
	}
}

func makeRealGridBars(n int) []sdk.Bar {
	bars := make([]sdk.Bar, n)
	price := 1.1000
	for i := 0; i < n; i++ {
		if (i/15)%2 == 0 {
			price += 0.0030
		} else {
			price -= 0.0025
		}
		bars[i] = sdk.Bar{
			Open:      decimal.NewFromFloat(price - 0.0005),
			High:      decimal.NewFromFloat(price + 0.0015),
			Low:       decimal.NewFromFloat(price - 0.0015),
			Close:     decimal.NewFromFloat(price),
			Volume:    1000,
			Timestamp: time.Now().Add(time.Duration(i) * 15 * time.Minute).UnixMilli(),
		}
	}
	return bars
}
