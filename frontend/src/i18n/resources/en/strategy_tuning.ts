// Auto-generated from proto/ant/v1/i18n/strategy_tuning_en.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Tuning = {
  "strategy": {
    "tuning": {
      "apply": "Apply",
      "degradation": "Degradation",
      "enabledCombinations": "{{enabled}} enabled · {{combos}} combinations",
      "grade": "Grade",
      "gridWarning": "Grid Search would test <b>{{count}}</b> combinations (budget: 48). Consider switching to <b>Differential Evolution</b> which handles large parameter spaces efficiently.",
      "hide": "Hide",
      "oosFootnote": "OOS validation run on top-5 candidates (by IS score). Green degradation <20%, orange 20-40%, red >40%.",
      "oosScore": "OOS Score",
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
      "optimizerMethod": "Optimizer method",
      "overfit": "Overfit",
      "overfitWarning": "⚠ OVERFIT",
      "parameterDimensions": "Parameter dimensions",
      "parameters": "Parameters",
      "preview": "Preview",
      "previewTitle": "Preview ({{shown}} of {{total}})",
      "rank": "#",
      "requiresAI": "Requires AI provider configured",
      "results": "Results ({{count}})",
      "run": "Run ({{count}})",
      "score": "Score",
      "started": "Smart Tuning started",
      "summary": "Summary",
      "switchToDE": "Switch to DE",
      "truncated": "TRUNCATED",
      "tuning": "Tuning…",
      "waiting": "Waiting for experiment... (SSE auto-refresh)"
    }
  }
} as const;
export default Tuning;
