// Auto-generated from proto/ant/v1/i18n/strategy_experiment_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyExperiment = {
  "strategy": {
    "experiment": {
      "candidates": {
        "column": {
          "actions": "操作",
          "generateDraft": "生成草稿",
          "grade": "評級",
          "parameters": "引數",
          "rank": "排名",
          "recommendation": "推薦",
          "score": "評分",
          "summary": "摘要",
          "viewCandidates": "檢視候選"
        },
        "title": "候選策略",
        "titleWithId": "候選: {{id}}"
      },
      "list": {
        "column": {
          "actions": "操作",
          "maxCandidates": "最大候選數",
          "objective": "最佳化目標",
          "searchMethod": "搜尋方法",
          "status": "狀態",
          "viewCandidates": "檢視候選"
        },
        "title": "實驗列表"
      },
      "messages": {
        "candidatesGenerated": "策略實驗候選已生成",
        "draftGenerated": "草稿模板已生成: {{templateId}}",
        "loadCandidatesFailed": "載入候選列表失敗",
        "loadExperimentsFailed": "載入實驗列表失敗",
        "loadTemplatesFailed": "載入策略模板失敗",
        "promoteFailed": "候選升級為草稿失敗",
        "submitFailed": "提交實驗失敗，請確認引數空間為有效 JSON。",
        "subscribeJobFailed": "訂閱實驗任務事件失敗"
      },
      "submitForm": {
        "baseTemplate": "基礎策略模板",
        "baseTemplatePlaceholder": "選擇模板",
        "baseTemplateRequired": "請選擇基礎策略模板",
        "maxCandidates": "最大候選數",
        "objective": "最佳化目標",
        "parameterSpace": "引數空間 JSON",
        "parameterSpaceRequired": "請輸入引數空間 JSON",
        "searchMethod": "搜尋方法",
        "submit": "提交實驗",
        "title": "提交實驗"
      },
      "jobEventStream": "任務事件流",
      "noEvents": "暫無事件",
      "ruleVersionAlert": "當前最小回路：確定性引數實驗。候選僅生成草稿，不會自動釋出、排程或交易。",
      "selectJobToView": "選擇帶有任務的實驗以檢視事件。",
      "subtitle": "提交引數組合以自動執行實驗、評分候選策略並生成草稿。",
      "title": "策略實驗"
    }
  }
} as const;
export default StrategyExperiment;
