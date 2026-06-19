// Auto-generated from proto/ant/v1/i18n/strategy_experiment_zh-tw.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Experiment = {
  "strategy": {
    "experiment": {
      "candidates": {
        "column": {
          "actions": "操作",
          "generateDraft": "生成草稿",
          "grade": "評級",
          "parameters": "參數",
          "rank": "排名",
          "recommendation": "推薦",
          "score": "評分",
          "summary": "摘要",
          "viewCandidates": "查看候選"
        },
        "title": "候選策略",
        "titleWithId": "候選: {{id}}"
      },
      "jobEventStream": "任务事件流",
      "list": {
        "column": {
          "actions": "操作",
          "maxCandidates": "最大候選数",
          "objective": "最佳化目标",
          "searchMethod": "搜索方法",
          "status": "狀態",
          "viewCandidates": "查看候選"
        },
        "title": "實驗列表"
      },
      "messages": {
        "candidatesGenerated": "策略實驗候選已生成",
        "draftGenerated": "草稿範本已生成: {{templateId}}",
        "loadCandidatesFailed": "載入候選列表失敗",
        "loadExperimentsFailed": "載入實驗列表失敗",
        "loadTemplatesFailed": "載入策略範本失敗",
        "promoteFailed": "候選升级为草稿失敗",
        "submitFailed": "提交實驗失敗，請確認參數空间为有效 JSON。",
        "subscribeJobFailed": "订阅實驗任务事件失敗"
      },
      "noEvents": "暫無事件",
      "ruleVersionAlert": "当前最小回路：确定性參數實驗。候選仅生成草稿，不会自动发布、排程或交易。",
      "selectJobToView": "選擇带有任务的實驗以查看事件。",
      "submitForm": {
        "baseTemplate": "基础策略範本",
        "baseTemplatePlaceholder": "選擇範本",
        "baseTemplateRequired": "請選擇基础策略範本",
        "maxCandidates": "最大候選数",
        "objective": "最佳化目标",
        "parameterSpace": "參數空间 JSON",
        "parameterSpaceRequired": "請輸入參數空间 JSON",
        "searchMethod": "搜索方法",
        "submit": "提交實驗",
        "title": "提交實驗"
      },
      "subtitle": "提交參數组合以自动執行實驗、評分候選策略并生成草稿。",
      "title": "策略實驗"
    }
  }
} as const;
export default Experiment;
