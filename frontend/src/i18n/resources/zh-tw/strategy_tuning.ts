// Auto-generated from proto/ant/v1/i18n/strategy_tuning_zh-tw.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyTuning = {
  "strategy": {
    "tuning": {
      "optimizer": {
        "ags": "退火高斯",
        "agsDesc": "带 sigma 退火的高斯擾動。輕量級 TPE 替代方案。",
        "ai": "AI 最佳化器",
        "aiDesc": "LLM 多輪提案。從歷史結果中學習，共 3 輪。",
        "de": "差分進化",
        "deDesc": "rand/1/bin 变异。在平滑曲面上收斂快。",
        "grid": "網格搜尋",
        "gridDesc": "窮舉笛卡爾乘積。适合 ≤3 个參數。",
        "random": "隨機搜尋",
        "randomDesc": "均勻隨機取樣。適合探索。",
        "tpe": "TPE (核密度估计)",
        "tpeDesc": "树结构 Parzen 估计器。KDE 建模好/坏分布。"
      },
      "apply": "套用",
      "degradation": "衰減",
      "degradationTip": "用樣本外數據測試時，收益相比樣本內的下降比例。越低越好。",
      "enabledCombinations": "{{enabled}} 個啟用 · {{combos}} 個組合",
      "grade": "評級",
      "gradeTip": "綜合評分等級，A 最佳，E 最差。",
      "gridWarning": "網格搜尋将测试 <b>{{count}}</b> 個組合 (限制: 48)。建議切换到 <b>差分進化</b> 以高效处理大參數空间。",
      "hide": "隱藏",
      "oosFootnote": "对前5个候選 (按样本内評分) 进行样本外驗證。绿色衰減 <20%, 橙色 20-40%, 红色 >40%。",
      "oosScore": "樣本外評分",
      "oosScoreTip": "用未參與訓練的歷史數據重新回測後的得分，衡量策略在未見數據上的表現。",
      "optimizerMethod": "最佳化方法",
      "overfit": "過度擬合",
      "overfitTip": "策略在訓練數據上表現好，但在新數據上表現差，說明參數只是「記住」了歷史而非真正有效。",
      "overfitWarning": "⚠ 過度擬合",
      "parameterDimensions": "參數維度",
      "parameters": "參數",
      "parametersTip": "本次調優嘗試的參數組合。",
      "preview": "預覽訊號",
      "previewTitle": "預覽 ({{shown}} / {{total}})",
      "rank": "#",
      "requiresAI": "需要設定 AI 服務商",
      "results": "结果 ({{count}})",
      "run": "執行 ({{count}} 次回測)",
      "score": "評分",
      "scoreTip": "綜合收益率、夏普比率、勝率、回撤等指標的加權評分，滿分 100。",
      "started": "智慧調校已啟動",
      "summary": "摘要",
      "summaryTip": "對候選結果的簡要描述。",
      "switchToDE": "切換到差分進化",
      "truncated": "已截斷",
      "tuning": "調校中…",
      "waiting": "等待實驗... (SSE 自動刷新)"
    }
  }
} as const;
export default StrategyTuning;
