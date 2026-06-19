// Auto-generated from proto/ant/v1/i18n/strategy_experiment_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Experiment = {
  "strategy": {
    "experiment": {
      "candidates": {
        "column": {
          "actions": "操作",
          "generateDraft": "生成草稿",
          "grade": "评级",
          "parameters": "参数",
          "rank": "排名",
          "recommendation": "推荐",
          "score": "评分",
          "summary": "摘要",
          "viewCandidates": "查看候选"
        },
        "title": "候选策略",
        "titleWithId": "候选: {{id}}"
      },
      "jobEventStream": "任务事件流",
      "list": {
        "column": {
          "actions": "操作",
          "maxCandidates": "最大候选数",
          "objective": "优化目标",
          "searchMethod": "搜索方法",
          "status": "状态",
          "viewCandidates": "查看候选"
        },
        "title": "实验列表"
      },
      "messages": {
        "candidatesGenerated": "策略实验候选已生成",
        "draftGenerated": "草稿模板已生成: {{templateId}}",
        "loadCandidatesFailed": "加载候选列表失败",
        "loadExperimentsFailed": "加载实验列表失败",
        "loadTemplatesFailed": "加载策略模板失败",
        "promoteFailed": "候选升级为草稿失败",
        "submitFailed": "提交实验失败，请确认参数空间为有效 JSON。",
        "subscribeJobFailed": "订阅实验任务事件失败"
      },
      "noEvents": "暂无事件",
      "ruleVersionAlert": "当前最小回路：确定性参数实验。候选仅生成草稿，不会自动发布、调度或交易。",
      "selectJobToView": "选择带有任务的实验以查看事件。",
      "submitForm": {
        "baseTemplate": "基础策略模板",
        "baseTemplatePlaceholder": "选择模板",
        "baseTemplateRequired": "请选择基础策略模板",
        "maxCandidates": "最大候选数",
        "objective": "优化目标",
        "parameterSpace": "参数空间 JSON",
        "parameterSpaceRequired": "请输入参数空间 JSON",
        "searchMethod": "搜索方法",
        "submit": "提交实验",
        "title": "提交实验"
      },
      "subtitle": "提交参数组合以自动运行实验、评分候选策略并生成草稿。",
      "title": "策略实验"
    }
  }
} as const;
export default Experiment;
