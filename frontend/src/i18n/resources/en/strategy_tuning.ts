// Auto-generated from proto/ant/v1/i18n/strategy_tuning_en.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyTuning = {
  "strategy": {
    "tuning": {
      "optimizer": {
        "ags": "Annealed Gaussian",
        "agsDesc": "Gaussian jitter with sigma annealing. Lightweight alternative to TPE.",
        "ai": "AI Optimizer",
        "aiDesc": "LLM multi-round proposal. Learns from previous results over 3 rounds.",
        "de": "Differential Evolution",
        "deDesc": "rand/1/bin mutation. Converges fast on smooth landscapes.",
        "grid": "Grid Search",
        "gridDesc": "Exhaustive Cartesian product. Best for ≤3 params.",
        "random": "Random Search",
        "randomDesc": "Uniform random sampling. Good for exploration.",
        "tpe": "TPE (KDE)",
        "tpeDesc": "Tree-structured Parzen Estimator. KDE models good/bad distributions."
      },
      "apply": "Apply",
      "degradation": "Degradation",
      "degradationTip": "How much the returns drop when tested on unseen data vs. training data. Lower is better.",
      "enabledCombinations": "{{enabled}} enabled · {{combos}} combinations",
      "grade": "Grade",
      "gradeTip": "Overall rating from A (best) to E (worst), based on composite score.",
      "gridWarning": "Grid Search would test <b>{{count}}</b> combinations (budget: 48). Consider switching to <b>Differential Evolution</b> which handles large parameter spaces efficiently.",
      "hide": "Hide",
      "oosFootnote": "OOS validation run on top-5 candidates (by IS score). Green degradation <20%, orange 20-40%, red >40%.",
      "oosScore": "OOS Score",
      "oosScoreTip": "Score from backtesting on data not used in training. Measures real-world robustness.",
      "optimizerMethod": "Optimizer method",
      "overfit": "Overfit",
      "overfitTip": "Strategy performs well on training data but poorly on new data — parameters just memorized history rather than finding real edges.",
      "overfitWarning": "⚠ OVERFIT",
      "parameterDimensions": "Parameter dimensions",
      "parameters": "Parameters",
      "parametersTip": "The parameter values tried in this candidate.",
      "preview": "Preview",
      "previewTitle": "Preview ({{shown}} of {{total}})",
      "rank": "#",
      "requiresAI": "Requires AI provider configured",
      "results": "Results ({{count}})",
      "run": "Run ({{count}} backtests)",
      "score": "Score",
      "scoreTip": "Weighted rating (0-100) combining returns, Sharpe ratio, win rate, drawdown, etc.",
      "started": "Smart Tuning started",
      "summary": "Summary",
      "summaryTip": "Brief description of the candidate result.",
      "switchToDE": "Switch to DE",
      "truncated": "TRUNCATED",
      "tuning": "Tuning…",
      "waiting": "Waiting for experiment... (SSE auto-refresh)"
    }
  }
} as const;
export default StrategyTuning;
