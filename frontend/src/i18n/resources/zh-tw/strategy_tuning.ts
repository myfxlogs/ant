// Auto-generated from proto/ant/v1/i18n/strategy_tuning_zh-tw.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyTuning = {
  "strategy": {
    "tuning": {
      "optimizer": {
        "ags": "退火高斯",
        "agsDesc": "帶 sigma 退火的高斯擾動。輕量級 TPE 替代方案。",
        "ai": "AI 最佳化器",
        "aiDesc": "LLM 多輪提案。從歷史結果中學習，共 3 輪。",
        "de": "差分進化",
        "deDesc": "rand/1/bin 變異。在平滑曲面上收斂快。",
        "grid": "網格搜尋",
        "gridDesc": "窮舉笛卡爾乘積。適合 ≤3 個引數。",
        "random": "隨機搜尋",
        "randomDesc": "均勻隨機取樣。適合探索。",
        "tpe": "TPE (核密度估計)",
        "tpeDesc": "樹結構 Parzen 估計器。KDE 建模好/壞分佈。"
      },
      "apply": "套用",
      "degradation": "衰減",
      "enabledCombinations": "{{enabled}} 個啟用 · {{combos}} 個組合",
      "grade": "評級",
      "gridWarning": "網格搜尋將測試 <b>{{count}}</b> 個組合 (限制: 48)。建議切換到 <b>差分進化</b> 以高效處理大引數空間。",
      "hide": "隱藏",
      "oosFootnote": "對前5個候選 (按樣本內評分) 進行樣本外驗證。綠色衰減 <20%, 橙色 20-40%, 紅色 >40%。",
      "oosScore": "樣本外評分",
      "optimizerMethod": "最佳化方法",
      "overfit": "過度擬合",
      "overfitWarning": "⚠ 過度擬合",
      "parameterDimensions": "引數維度",
      "parameters": "引數",
      "preview": "預覽訊號",
      "previewTitle": "預覽 ({{shown}} / {{total}})",
      "rank": "#",
      "requiresAI": "需要設定 AI 服務商",
      "results": "結果 ({{count}})",
      "run": "執行 ({{count}})",
      "score": "評分",
      "started": "智慧調校已啟動",
      "summary": "摘要",
      "switchToDE": "切換到差分進化",
      "truncated": "已截斷",
      "tuning": "調校中…",
      "waiting": "等待實驗... (SSE 自動重新整理)"
    }
  }
} as const;
export default StrategyTuning;
