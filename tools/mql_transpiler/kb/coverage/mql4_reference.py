#!/usr/bin/env python3
"""Cross-reference MQL4 official reference vs transpiler coverage.

Usage:   PYTHONPATH=. python3 tools/mql_transpiler/kb/coverage/mql4_reference.py
Output:  Gap report for functions in MQL4 docs not yet handled by transpiler.
"""

from __future__ import annotations

# ── MQL4 TRADING FUNCTIONS ──
# Source: https://docs.mql4.com/trading (25 functions)

MQL4_TRADE_FUNCTIONS = {
    "OrderClose": "Closes opened order",
    "OrderCloseBy": "Close by opposite order (hedging)",
    "OrderClosePrice": "Returns close price of selected order",
    "OrderCloseTime": "Returns close time of selected order",
    "OrderComment": "Returns comment of selected order",
    "OrderCommission": "Returns commission of selected order",
    "OrderDelete": "Deletes pending order",
    "OrderExpiration": "Returns expiration of pending order",
    "OrderLots": "Returns lots of selected order",
    "OrderMagicNumber": "Returns magic number of selected order",
    "OrderModify": "Modify order characteristics",
    "OrderOpenPrice": "Returns open price of selected order",
    "OrderOpenTime": "Returns open time of selected order",
    "OrderPrint": "Print order info to log",
    "OrderProfit": "Returns profit of selected order",
    "OrderSelect": "Select order for processing",
    "OrderSend": "Open order / place pending",
    "OrdersHistoryTotal": "Number of closed orders in history",
    "OrderStopLoss": "Returns SL of selected order",
    "OrdersTotal": "Number of market + pending orders",
    "OrderSwap": "Returns swap of selected order",
    "OrderSymbol": "Returns symbol of selected order",
    "OrderTakeProfit": "Returns TP of selected order",
    "OrderTicket": "Returns ticket of selected order",
    "OrderType": "Returns operation type of selected order",
}

# ── MQL4 ACCOUNT FUNCTIONS ──
# Source: https://docs.mql4.com/account

MQL4_ACCOUNT_FUNCTIONS = {
    "AccountBalance": "Returns balance",
    "AccountCredit": "Returns credit",
    "AccountCompany": "Returns broker company name",
    "AccountCurrency": "Returns account currency",
    "AccountEquity": "Returns equity",
    "AccountFreeMargin": "Returns free margin",
    "AccountFreeMarginCheck": "Check free margin for order",
    "AccountFreeMarginMode": "Returns margin calculation mode",
    "AccountLeverage": "Returns leverage",
    "AccountMargin": "Returns used margin",
    "AccountName": "Returns account name",
    "AccountNumber": "Returns account number",
    "AccountProfit": "Returns floating profit",
    "AccountServer": "Returns server name",
    "AccountStopoutLevel": "Returns stop-out level",
    "AccountStopoutMode": "Returns stop-out mode",
}

# ── MQL4 MARKET INFO FUNCTIONS ──
# Source: https://docs.mql4.com/marketinfo

MQL4_MARKET_FUNCTIONS = {
    "MarketInfo": "Returns market info (MODE_* constants)",
    "Symbol": "Returns current symbol name",
    "SymbolInfoDouble": "Returns double property",
    "SymbolInfoInteger": "Returns integer property",
    "SymbolInfoString": "Returns string property",
    "SymbolSelect": "Select symbol in Market Watch",
}

# ── MQL4 COMMON FUNCTIONS ──
# Source: https://docs.mql4.com/common

MQL4_COMMON_FUNCTIONS = {
    "Alert": "Show alert dialog",
    "Comment": "Output comment to chart",
    "GetTickCount": "Returns ms since system start",
    "MarketInfo": "Returns market info (duplicate)",
    "PlaySound": "Play sound file",
    "Print": "Print message to log",
    "SendFTP": "Send file via FTP (UNSUPPORTED)",
    "SendMail": "Send email (UNSUPPORTED)",
    "SendNotification": "Send push notification (UNSUPPORTED)",
    "Sleep": "Pause execution (ms)",
}

# ── MQL4 MATH FUNCTIONS ──

MQL4_MATH_FUNCTIONS = {
    "MathAbs": "Returns absolute value",
    "MathArccos": "Returns arccosine",
    "MathArcsin": "Returns arcsine",
    "MathArctan": "Returns arctangent",
    "MathCeil": "Returns nearest upper integer",
    "MathCos": "Returns cosine",
    "MathExp": "Returns exponent",
    "MathFloor": "Returns nearest lower integer",
    "MathLog": "Returns natural logarithm",
    "MathMax": "Returns max of two",
    "MathMin": "Returns min of two",
    "MathMod": "Returns remainder",
    "MathPow": "Returns power",
    "MathRand": "Returns random int",
    "MathRound": "Returns rounded value",
    "MathSin": "Returns sine",
    "MathSqrt": "Returns square root",
    "MathSrand": "Seed random generator",
    "MathTan": "Returns tangent",
}

# ── MQL4 TECHNICAL INDICATORS ──

MQL4_INDICATORS = {
    "iAC": "Accelerator Oscillator",
    "iAD": "Accumulation/Distribution",
    "iADX": "Average Directional Index",
    "iAlligator": "Alligator",
    "iAO": "Awesome Oscillator",
    "iATR": "Average True Range",
    "iBands": "Bollinger Bands",
    "iBearsPower": "Bears Power",
    "iBullsPower": "Bulls Power",
    "iBWMFI": "Market Facilitation Index",
    "iCCI": "Commodity Channel Index",
    "iCustom": "Custom indicator",
    "iDeMarker": "DeMarker",
    "iEnvelopes": "Envelopes",
    "iForce": "Force Index",
    "iFractals": "Fractals",
    "iGator": "Gator Oscillator",
    "iIchimoku": "Ichimoku Kinko Hyo",
    "iMA": "Moving Average",
    "iMACD": "MACD",
    "iMFI": "Money Flow Index",
    "iMomentum": "Momentum",
    "iOBV": "On Balance Volume",
    "iOsMA": "Moving Average of Oscillator",
    "iRSI": "Relative Strength Index",
    "iRVI": "Relative Vigor Index",
    "iSAR": "Parabolic SAR",
    "iStdDev": "Standard Deviation",
    "iStochastic": "Stochastic Oscillator",
    "iWPR": "Williams Percent Range",
}

# ── Transpiler coverage (from kb/README.md) ──

TRANSPILER_COVERED_FUNCTIONS = {
    # Trade
    "OrderClose", "OrderClosePrice", "OrderComment", "OrderCommission",
    "OrderDelete", "OrderLots", "OrderMagicNumber", "OrderModify",
    "OrderOpenPrice", "OrderProfit", "OrderSelect", "OrderSend",
    "OrdersTotal", "OrderStopLoss", "OrderSwap", "OrderSymbol",
    "OrderTakeProfit", "OrderTicket", "OrderType",
    # Account
    "AccountBalance", "AccountEquity", "AccountFreeMargin",
    "AccountLeverage",
    # Market
    "Symbol", "Ask", "Bid", "Point", "Digits",
    # Common
    "Print", "GetTickCount", "MathAbs", "MathMax", "MathMin",
    "MathRound", "EventSetTimer", "EventKillTimer",
    # Indicators
    "iMA", "iRSI", "iBands", "iMACD", "iATR", "iStochastic",
    "iCCI", "iCustom",
}


def compare(category: str, reference: dict) -> dict:
    """Compare MQL4 reference vs transpiler coverage."""
    covered = []
    uncovered = []
    for name, desc in reference.items():
        if name in TRANSPILER_COVERED_FUNCTIONS:
            covered.append((name, desc))
        else:
            uncovered.append((name, desc))
    return {"category": category, "covered": covered, "uncovered": uncovered}


def main():
    categories = [
        ("Trading", MQL4_TRADE_FUNCTIONS),
        ("Account", MQL4_ACCOUNT_FUNCTIONS),
        ("Market Info", MQL4_MARKET_FUNCTIONS),
        ("Common", MQL4_COMMON_FUNCTIONS),
        ("Math", MQL4_MATH_FUNCTIONS),
        ("Technical Indicators", MQL4_INDICATORS),
    ]

    total_covered = 0
    total_uncovered = 0

    for cat_name, ref in categories:
        result = compare(cat_name, ref)
        c, u = len(result["covered"]), len(result["uncovered"])
        total_covered += c
        total_uncovered += u
        pct = (c / (c + u) * 100) if (c + u) > 0 else 0
        print(f"\n{'='*60}")
        print(f"  {cat_name}: {c}/{c+u} covered ({pct:.0f}%)")
        print(f"{'='*60}")
        if result["uncovered"]:
            print("  🔴 GAPS:")
            for name, desc in result["uncovered"]:
                print(f"    {name}() — {desc}")

    total = total_covered + total_uncovered
    print(f"\n{'='*60}")
    print(f"  OVERALL: {total_covered}/{total} ({total_covered/total*100:.0f}%)")
    print(f"  Gaps remaining: {total_uncovered}")
    print(f"{'='*60}")


if __name__ == "__main__":
    main()
