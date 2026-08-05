//+------------------------------------------------------------------+
//|                                                  MACD Sample.mq4 |
//|                   Copyright 2005-2014, MetaQuotes Software Corp. |
//|                                              http://www.metaquotes.net |
//+------------------------------------------------------------------+
#property copyright "2005-2014, MetaQuotes Software Corp."
#property link      "http://www.metaquotes.net"
#property version   "1.00"
#property strict

extern double Lots = 0.1;
extern double MaximumRisk = 0.02;
extern double DecreaseFactor = 3;
extern int    MovingPeriod = 12;
extern int    MovingAverage = 26;
extern int    MACDOpenLevel = 3;
extern int    MACDCloseLevel = 2;
extern int    MATrendPeriod = 26;

//+------------------------------------------------------------------+
//| Calculate open positions                                         |
//+------------------------------------------------------------------+
int CalculateCurrentOrders(string symbol)
{
   int buys = 0, sells = 0;
   for(int i = 0; i < OrdersTotal(); i++)
   {
      if(OrderSelect(i, SELECT_BY_POS, MODE_TRADES) == false) break;
      if(OrderSymbol() == symbol && OrderMagicNumber() == 16384)
      {
         if(OrderType() == OP_BUY)  buys++;
         if(OrderType() == OP_SELL) sells++;
      }
   }
   if(buys > 0) return(buys);
   else         return(-sells);
}

//+------------------------------------------------------------------+
//| Calculate optimal lot size                                       |
//+------------------------------------------------------------------+
double LotsOptimized()
{
   double lot = Lots;
   int    orders = OrdersTotal();
   int    losses = 0;

   lot = NormalizeDouble(lot, 1);
   if(lot < 0.1) lot = 0.1;
   return(lot);
}

//+------------------------------------------------------------------+
//| Check for open order conditions                                  |
//+------------------------------------------------------------------+
void CheckForOpen()
{
   double macdCurrent, macdPrevious, signalCurrent;
   double maCurrent, maPrevious;
   int    res;
   int    ticket;

   macdCurrent  = iMACD(NULL, 0, 12, 26, 9, PRICE_CLOSE, MODE_MAIN, 0);
   signalCurrent = iMACD(NULL, 0, 12, 26, 9, PRICE_CLOSE, MODE_SIGNAL, 0);
   macdPrevious = iMACD(NULL, 0, 12, 26, 9, PRICE_CLOSE, MODE_MAIN, 1);
   maCurrent    = iMA(NULL, 0, MATrendPeriod, 0, MODE_EMA, PRICE_CLOSE, 0);
   maPrevious   = iMA(NULL, 0, MATrendPeriod, 0, MODE_EMA, PRICE_CLOSE, 1);

   if(macdCurrent < 0 && macdCurrent > signalCurrent &&
      macdPrevious < signalCurrent &&
      MathAbs(macdCurrent) > MACDOpenLevel * Point &&
      maCurrent > maPrevious)
   {
      res = OrderSend(Symbol(), OP_BUY, LotsOptimized(), Ask, 3, 0, 0,
                      "MACD Sample", 16384, 0, clrGreen);
      return;
   }
   if(macdCurrent > 0 && macdCurrent < signalCurrent &&
      macdPrevious > signalCurrent &&
      MathAbs(macdCurrent) > MACDOpenLevel * Point &&
      maCurrent < maPrevious)
   {
      res = OrderSend(Symbol(), OP_SELL, LotsOptimized(), Bid, 3, 0, 0,
                      "MACD Sample", 16384, 0, clrRed);
   }
}

//+------------------------------------------------------------------+
//| Check for close order conditions                                 |
//+------------------------------------------------------------------+
void CheckForClose()
{
   double macdCurrent, macdPrevious, signalCurrent;
   int    total;

   macdCurrent  = iMACD(NULL, 0, 12, 26, 9, PRICE_CLOSE, MODE_MAIN, 0);
   signalCurrent = iMACD(NULL, 0, 12, 26, 9, PRICE_CLOSE, MODE_SIGNAL, 0);
   macdPrevious = iMACD(NULL, 0, 12, 26, 9, PRICE_CLOSE, MODE_MAIN, 1);

   for(int i = 0; i < OrdersTotal(); i++)
   {
      if(OrderSelect(i, SELECT_BY_POS, MODE_TRADES) == false) break;
      if(OrderMagicNumber() != 16384 || OrderSymbol() != Symbol()) continue;

      if(OrderType() == OP_BUY)
      {
         if(macdCurrent > 0 && macdCurrent < signalCurrent &&
            macdPrevious > signalCurrent &&
            MathAbs(macdCurrent) > MACDCloseLevel * Point)
         {
            OrderClose(OrderTicket(), OrderLots(), Bid, 3, clrWhite);
         }
      }
      if(OrderType() == OP_SELL)
      {
         if(macdCurrent < 0 && macdCurrent > signalCurrent &&
            macdPrevious < signalCurrent &&
            MathAbs(macdCurrent) > MACDCloseLevel * Point)
         {
            OrderClose(OrderTicket(), OrderLots(), Ask, 3, clrWhite);
         }
      }
   }
}

//+------------------------------------------------------------------+
//| OnTick function                                                  |
//+------------------------------------------------------------------+
void OnTick()
{
   if(Bars < 35) return;
   if(CalculateCurrentOrders(Symbol()) == 0) CheckForOpen();
   else                                      CheckForClose();
}
