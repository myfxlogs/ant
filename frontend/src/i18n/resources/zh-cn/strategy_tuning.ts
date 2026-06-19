// Auto-generated from proto/ant/v1/i18n/strategy_tuning_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Tuning = {
  "strategy": {
    "tuning": {
      "apply": "应用",
      "degradation": "衰减",
      "enabledCombinations": "{{enabled}} 个启用 · {{combos}} 个组合",
      "grade": "评级",
      "gridWarning": "网格搜索将测试 <b>{{count}}</b> 个组合 (限制: 48)。建议切换到 <b>差分进化</b> 以高效处理大参数空间。",
      "hide": "隐藏",
      "oosFootnote": "对前5个候选 (按样本内评分) 进行样本外验证。绿色衰减 <20%, 橙色 20-40%, 红色 >40%。",
      "oosScore": "样本外评分",
      "optimizer": {
        "ags": "退火高斯",
        "agsDesc": "带 sigma 退火的高斯扰动。轻量级 TPE 替代方案。",
        "ai": "AI 优化器",
        "aiDesc": "LLM 多轮提案。从历史结果中学习，共 3 轮。",
        "de": "差分进化",
        "deDesc": "rand/1/bin 变异。在平滑曲面上收敛快。",
        "grid": "网格搜索",
        "gridDesc": "穷举笛卡尔积。适合 ≤3 个参数。",
        "random": "随机搜索",
        "randomDesc": "均匀随机采样。适合探索。",
        "tpe": "TPE (核密度估计)",
        "tpeDesc": "树结构 Parzen 估计器。KDE 建模好/坏分布。"
      },
      "optimizerMethod": "优化方法",
      "overfit": "过拟合",
      "overfitWarning": "⚠ 过拟合",
      "parameterDimensions": "参数维度",
      "parameters": "参数",
      "preview": "预览信号",
      "previewTitle": "预览 ({{shown}} / {{total}})",
      "rank": "#",
      "requiresAI": "需要配置 AI 服务商",
      "results": "结果 ({{count}})",
      "run": "运行 ({{count}})",
      "score": "评分",
      "started": "智能调优已启动",
      "summary": "摘要",
      "switchToDE": "切换到差分进化",
      "truncated": "已截断",
      "tuning": "调优中…",
      "waiting": "等待实验... (SSE 自动刷新)"
    }
  }
} as const;
export default Tuning;
