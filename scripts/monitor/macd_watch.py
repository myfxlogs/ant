#!/usr/bin/env python3
"""
MACD Sample 策略开仓条件监控器
策略: 599ddaa5-6a19-4889-ad73-503a800bcd39
账户: 904d14e6-8d67-4541-80f9-f3b7f9587a00
品种: BTCUSDm 1m

开仓条件 (MACD Sample.mq4):
  BUY:  MacdCurrent<0 && MacdCurrent>Signal && MacdPrev<SignalPrev
        && |MacdCurrent| > MACDOpenLevel*Point(=3*0.01=0.03)
        && MaCurrent > MaPrevious (EMA26 上升)
  SELL: MacdCurrent>0 && MacdCurrent<Signal && MacdPrev>SignalPrev
        && MacdCurrent > 3*Point
        && MaCurrent < MaPrevious (EMA26 下降)

用法:
  python3 macd_watch.py            # 单次快照
  python3 macd_watch.py --watch    # 持续监控，每 30s 刷新
  python3 macd_watch.py --watch --interval 60
"""
import argparse
import os
import sys
import time
import math
from datetime import datetime, timezone

try:
    import psycopg2
except ImportError:
    print("ERROR: pip install psycopg2-binary", file=sys.stderr)
    sys.exit(1)

# ── 配置 ──
PG_HOST = os.environ.get("PG_HOST", "localhost")
PG_PORT = os.environ.get("PG_PORT", "5432")
PG_DB   = os.environ.get("PG_DB", "ant")
PG_USER = os.environ.get("PG_USER", "ant")
PG_PASS = os.environ.get("PG_PASS", os.environ.get("POSTGRES_PASSWORD", ""))

ACCOUNT_ID = "904d14e6-8d67-4541-80f9-f3b7f9587a00"
SYMBOL     = "BTCUSDm"
PERIOD     = "1m"
# 策略参数 (MACD Sample 默认)
MACD_FAST = 12
MACD_SLOW = 26
MACD_SIGNAL = 9
MA_PERIOD = 26  # MATrendPeriod
MACD_OPEN_LEVEL = 3.0
POINT = 0.01  # BTCUSDm point (digits=2)
LOTS = 0.1
FREE_MARGIN_THRESHOLD = 1000 * LOTS  # 100

# ── 指标计算 ──

def ema(values, period):
    """计算 EMA，返回与 values 等长的数组。"""
    if len(values) == 0:
        return []
    k = 2.0 / (period + 1)
    out = [0.0] * len(values)
    out[0] = values[0]
    for i in range(1, len(values)):
        out[i] = values[i] * k + out[i-1] * (1 - k)
    return out

def macd(closes, fast=12, slow=26, signal=9):
    """返回 (macd_line, signal_line) 数组。"""
    if len(closes) < slow:
        return [], []
    ema_fast = ema(closes, fast)
    ema_slow = ema(closes, slow)
    macd_line = [f - s for f, s in zip(ema_fast, ema_slow)]
    signal_line = ema(macd_line, signal)
    return macd_line, signal_line

# ── 数据获取 ──

def fetch_bars(conn, limit=500):
    sql = """
        SELECT open_ts_unix_ms, open, high, low, close
        FROM md_bars
        WHERE account_id = %s AND canonical = %s AND period = %s
        ORDER BY open_ts_unix_ms DESC
        LIMIT %s
    """
    with conn.cursor() as cur:
        cur.execute(sql, (ACCOUNT_ID, SYMBOL, PERIOD, limit))
        rows = cur.fetchall()
    # 反转为升序 ( oldest first )
    rows.reverse()
    return rows

def fetch_account_snapshot(conn):
    """读最新账户快照 (account_balance_history)。"""
    sql = """
        SELECT balance, equity, margin, free_margin, recorded_at
        FROM account_balance_history
        WHERE account_id = %s
        ORDER BY recorded_at DESC
        LIMIT 1
    """
    with conn.cursor() as cur:
        cur.execute(sql, (ACCOUNT_ID,))
        row = cur.fetchone()
    return row

# ── 条件评估 ──

def evaluate(macd_cur, macd_prev, sig_cur, sig_prev, ma_cur, ma_prev, free_margin):
    """返回 (action, conditions) — action: 'BUY'/'SELL'/'HOLD'/'BLOCKED'。"""
    conditions = {
        "macd_current": macd_cur,
        "macd_previous": macd_prev,
        "signal_current": sig_cur,
        "signal_previous": sig_prev,
        "ma_current": ma_cur,
        "ma_previous": ma_prev,
        "macd_abs": abs(macd_cur),
        "macd_above_threshold": abs(macd_cur) > MACD_OPEN_LEVEL * POINT,
        "free_margin": free_margin,
        "margin_ok": free_margin > FREE_MARGIN_THRESHOLD,
    }

    # BUY 条件
    buy = (
        macd_cur < 0
        and macd_cur > sig_cur
        and macd_prev < sig_prev
        and abs(macd_cur) > MACD_OPEN_LEVEL * POINT
        and ma_cur > ma_prev
    )
    # SELL 条件
    sell = (
        macd_cur > 0
        and macd_cur < sig_cur
        and macd_prev > sig_prev
        and macd_cur > MACD_OPEN_LEVEL * POINT
        and ma_cur < ma_prev
    )

    if not conditions["margin_ok"]:
        return "BLOCKED_NO_MONEY", conditions
    if buy:
        return "BUY", conditions
    if sell:
        return "SELL", conditions
    return "HOLD", conditions

def proximity(macd_cur, sig_cur, macd_prev, sig_prev):
    """估算距离交叉的"接近度"——0=刚交叉，越大越远。"""
    # 当前差值 ( 正 = macd 在 signal 上方 )
    cur_diff = macd_cur - sig_cur
    # 上一根差值
    prev_diff = macd_prev - sig_prev
    # 差值变化方向
    trending = cur_diff - prev_diff
    return cur_diff, trending

# ── 渲染 ──

def fmt(v):
    if v is None:
        return "N/A"
    return f"{v:.4f}"

def render(action, conds, bar_time, cur_diff, trending, acct):
    ts = datetime.now(timezone.utc).strftime("%H:%M:%S UTC")
    bt = datetime.fromtimestamp(bar_time/1000, tz=timezone.utc).strftime("%H:%M UTC")
    print(f"\n{'='*68}")
    print(f"  MACD Sample Watch  {ts}  |  bar={bt}  BTCUSDm 1m")
    print(f"{'='*68}")

    # 账户
    if acct:
        bal, eq, marg, fm, rec = acct
        fm_str = f"{float(fm):.2f}" if fm is not None else "N/A"
        print(f"  Account: balance={float(bal):.2f}  equity={float(eq):.2f}  "
              f"margin={float(marg):.2f}  free_margin={fm_str}")
        fm_ok = "OK" if (fm is not None and float(fm) > FREE_MARGIN_THRESHOLD) else "BLOCKED"
        print(f"  Free Margin Gate: {fm_ok}  (threshold={FREE_MARGIN_THRESHOLD:.0f}, "
              f"actual={float(fm):.2f})" if fm is not None else f"  Free Margin Gate: N/A")
    else:
        print(f"  Account: no snapshot found")

    # 指标
    print(f"  ---")
    print(f"  MACD(12,26,9):")
    print(f"    current  = {fmt(conds['macd_current'])}   signal = {fmt(conds['signal_current'])}")
    print(f"    previous = {fmt(conds['macd_previous'])}   signal = {fmt(conds['signal_previous'])}")
    print(f"    |macd|   = {fmt(conds['macd_abs'])}   threshold = {MACD_OPEN_LEVEL*POINT:.4f}  "
          f"({'PASS' if conds['macd_above_threshold'] else 'FAIL'})")
    print(f"  EMA26:")
    print(f"    current  = {fmt(conds['ma_current'])}   previous = {fmt(conds['ma_previous'])}  "
          f"({'UP' if conds['ma_current'] > conds['ma_previous'] else 'DOWN'})")
    print(f"  ---")
    print(f"  MACD-Signal spread: current={cur_diff:+.4f}  delta={trending:+.4f}/bar")

    # 接近度提示
    # BUY 需要负→正交叉 ( macd 从下穿 signal 上来 )
    # SELL 需要正→负交叉 ( macd 从上穿 signal 下来 )
    if cur_diff < 0 and trending > 0:
        est = "approaching BUY cross" if abs(cur_diff) < 0.5 else "far from BUY cross"
        print(f"  Trend: MACD below signal, rising  -> {est}  (gap={abs(cur_diff):.4f})")
    elif cur_diff > 0 and trending < 0:
        est = "approaching SELL cross" if abs(cur_diff) < 0.5 else "far from SELL cross"
        print(f"  Trend: MACD above signal, falling -> {est}  (gap={abs(cur_diff):.4f})")
    elif cur_diff > 0 and trending > 0:
        print(f"  Trend: MACD above signal, widening (no cross imminent)")
    elif cur_diff < 0 and trending < 0:
        print(f"  Trend: MACD below signal, widening (no cross imminent)")
    else:
        print(f"  Trend: stable")

    # 动作
    color = {
        "BUY": "\033[32m",   # green
        "SELL": "\033[31m",  # red
        "BLOCKED_NO_MONEY": "\033[33m",  # yellow
        "HOLD": "\033[90m",  # gray
    }.get(action, "")
    reset = "\033[0m" if color else ""
    print(f"  ---")
    print(f"  Action: {color}{action}{reset}")

    if action == "BUY":
        print(f"  >>> BUY signal! OrderSend(OP_BUY, {LOTS}, Ask, ...)")
    elif action == "SELL":
        print(f"  >>> SELL signal! OrderSend(OP_SELL, {LOTS}, Bid, ...)")
    elif action == "BLOCKED_NO_MONEY":
        print(f"  >>> blocked: free_margin < {FREE_MARGIN_THRESHOLD:.0f} (We have no money)")
    else:
        # 列出哪些条件不满足
        failing = []
        if not conds["margin_ok"]:
            failing.append("free_margin")
        if not conds["macd_above_threshold"]:
            failing.append(f"|macd|>{MACD_OPEN_LEVEL*POINT:.2f}")
        # BUY 条件: MacdCurrent<0 && MacdCurrent>Signal && MacdPrev<SignalPrev && EMA26 UP
        buy_conds = {
            "macd<0": conds["macd_current"] < 0,
            "macd>signal": conds["macd_current"] > conds["signal_current"],
            "prev<signal_prev": conds["macd_previous"] < conds["signal_previous"],
            "EMA26_up": conds["ma_current"] > conds["ma_previous"],
        }
        # SELL 条件: MacdCurrent>0 && MacdCurrent<Signal && MacdPrev>SignalPrev && EMA26 DOWN
        sell_conds = {
            "macd>0": conds["macd_current"] > 0,
            "macd<signal": conds["macd_current"] < conds["signal_current"],
            "prev>signal_prev": conds["macd_previous"] > conds["signal_previous"],
            "EMA26_down": conds["ma_current"] < conds["ma_previous"],
        }
        # 判断哪个方向更接近
        buy_score = sum(buy_conds.values())
        sell_score = sum(sell_conds.values())
        if buy_score >= sell_score:
            for name, ok in buy_conds.items():
                if not ok:
                    failing.append(f"BUY: {name}")
        else:
            for name, ok in sell_conds.items():
                if not ok:
                    failing.append(f"SELL: {name}")
        if failing:
            print(f"  Failing: {', '.join(failing)}")

# ── 主循环 ──

def connect():
    return psycopg2.connect(
        host=PG_HOST, port=PG_PORT, dbname=PG_DB,
        user=PG_USER, password=PG_PASS,
    )

def tick(conn):
    rows = fetch_bars(conn, 500)
    if len(rows) < 30:
        print(f"  not enough bars: {len(rows)}")
        return
    closes = [float(r[4]) for r in rows]
    macd_line, signal_line = macd(closes, MACD_FAST, MACD_SLOW, MACD_SIGNAL)
    ma = ema(closes, MA_PERIOD)
    if len(macd_line) < 2 or len(signal_line) < 2 or len(ma) < 2:
        print("  insufficient data for indicators")
        return
    macd_cur = macd_line[-1]
    macd_prev = macd_line[-2]
    sig_cur = signal_line[-1]
    sig_prev = signal_line[-2]
    ma_cur = ma[-1]
    ma_prev = ma[-2]
    acct = fetch_account_snapshot(conn)
    fm = float(acct[3]) if acct and acct[3] is not None else 0.0
    action, conds = evaluate(macd_cur, macd_prev, sig_cur, sig_prev, ma_cur, ma_prev, fm)
    cur_diff, trending = proximity(macd_cur, sig_cur, macd_prev, sig_prev)
    bar_time = rows[-1][0]
    render(action, conds, bar_time, cur_diff, trending, acct)

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--watch", action="store_true", help="持续监控")
    ap.add_argument("--interval", type=int, default=30, help="刷新间隔秒数")
    args = ap.parse_args()
    conn = connect()
    if args.watch:
        try:
            while True:
                tick(conn)
                time.sleep(args.interval)
        except KeyboardInterrupt:
            print("\nstopped")
    else:
        tick(conn)
    conn.close()

if __name__ == "__main__":
    main()
