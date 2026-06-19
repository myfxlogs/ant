// Auto-generated from proto/ant/v1/i18n/strategy_backtest_run_ja.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyBacktestRun = {
  "strategy": {
    "backtestRun": {
      "trades": {
        "reasons": {
          "end_of_test": "テスト終了",
          "expired": "期限切れ",
          "margin_call": "追証",
          "signal": "シグナル（発注用）",
          "sl": "損切り",
          "tp": "利確"
        },
        "closePrice": "決済値",
        "closeTime": "終了時刻",
        "commission": "手数料",
        "empty": "取引記録なし",
        "loadFailed": "注文詳細の読込失敗",
        "openPrice": "建値",
        "openTime": "開始時刻",
        "pnl": "損益",
        "reason": "決済理由",
        "side": "方向",
        "sideBuy": "買い",
        "sideSell": "売り",
        "summary": "{{count}}取引 · {{wins}}勝 / {{losses}}敗 · 純損益 {{pnl}}",
        "ticket": "チケット",
        "title": "注文詳細",
        "volume": "数量"
      },
      "actions": {
        "cancel": "キャンセル"
      },
      "fields": {
        "error": "エラー",
        "maxDrawdown": "最大ドローダウン",
        "sharpe": "シャープレシオ",
        "status": "状態"
      },
      "hints": {
        "canceling": "バックテストキャンセル中",
        "queued": "バックテスト待機中",
        "running": "バックテスト実行中"
      },
      "metrics": {
        "annualReturn": "年率リターン",
        "equityCurvePoints": "資産曲線ポイント",
        "maxDrawdown": "最大ドローダウン",
        "sharpe": "シャープレシオ",
        "totalReturn": "総リターン",
        "totalTrades": "取引回数",
        "winRate": "勝率"
      },
      "status": {
        "canceled": "キャンセル済",
        "canceling": "キャンセル中",
        "completed": "完了",
        "ended": "終了",
        "failed": "失败",
        "queued": "待機中",
        "running": "実行中"
      },
      "title": "バックテスト実行"
    }
  }
} as const;
export default StrategyBacktestRun;
