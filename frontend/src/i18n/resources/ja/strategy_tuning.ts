// Auto-generated from proto/ant/v1/i18n/strategy_tuning_ja.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyTuning = {
  "strategy": {
    "tuning": {
      "optimizer": {
        "ags": "焼鈍ガウス",
        "agsDesc": "シグマ焼鈍付きガウス摂動。TPEの軽量代替。",
        "ai": "AIオプティマイザ",
        "aiDesc": "LLM複数ラウンド提案。過去結果から3ラウンド学習。",
        "de": "差分進化",
        "deDesc": "rand/1/bin突然変異。滑らかな地形で高速収束。",
        "grid": "グリッドサーチ",
        "gridDesc": "網羅的デカルト積。≤3パラメータに最適。",
        "random": "ランダムサーチ",
        "randomDesc": "一様ランダムサンプリング。探索向き。",
        "tpe": "TPE (核密度估计)",
        "tpeDesc": "ツリー構造Parzen推定量。KDEで良/不良分布をモデル化。"
      },
      "apply": "適用",
      "degradation": "劣化",
      "enabledCombinations": "{{enabled}} 有効 · {{combos}} 組合せ",
      "grade": "グレード",
      "gridWarning": "グリッドサーチは<b>{{count}}</b>組み合わせをテストします（上限48）。<b>差分進化</b>への切替を推奨。",
      "hide": "非表示",
      "oosFootnote": "上位5候補をOOS検証（ISスコア基準）。緑の劣化<20%、橙20-40%、赤>40%。",
      "oosScore": "OOSスコア",
      "optimizerMethod": "最適化手法",
      "overfit": "過学習",
      "overfitWarning": "⚠ 過学習",
      "parameterDimensions": "パラメータ次元",
      "parameters": "パラメータ",
      "preview": "シグナルプレビュー",
      "previewTitle": "プレビュー ({{shown}} / {{total}})",
      "rank": "#",
      "requiresAI": "AIプロバイダーの設定が必要です",
      "results": "結果 ({{count}})",
      "run": "実行 ({{count}})",
      "score": "スコア",
      "started": "スマートチューニング開始",
      "summary": "サマリー",
      "switchToDE": "DEに切替",
      "truncated": "切捨",
      "tuning": "チューニング中…",
      "waiting": "実験待機中... (SSE自動更新)"
    }
  }
} as const;
export default StrategyTuning;
