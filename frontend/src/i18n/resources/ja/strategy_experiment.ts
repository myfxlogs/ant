// Auto-generated from proto/ant/v1/i18n/strategy_experiment_ja.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Experiment = {
  "strategy": {
    "experiment": {
      "candidates": {
        "column": {
          "actions": "操作",
          "generateDraft": "下書き生成",
          "grade": "グレード",
          "parameters": "パラメータ",
          "rank": "順位",
          "recommendation": "推奨",
          "score": "スコア",
          "summary": "サマリー",
          "viewCandidates": "候補表示"
        },
        "title": "候補",
        "titleWithId": "候補: {{id}}"
      },
      "jobEventStream": "ジョブイベントストリーム",
      "list": {
        "column": {
          "actions": "操作",
          "maxCandidates": "最大候補数",
          "objective": "目的",
          "searchMethod": "探索方法",
          "status": "状態",
          "viewCandidates": "候補表示"
        },
        "title": "実験一覧"
      },
      "messages": {
        "candidatesGenerated": "ストラテジー実験候補生成完了",
        "draftGenerated": "下書きテンプレート生成: {{templateId}}",
        "loadCandidatesFailed": "候補読込失敗",
        "loadExperimentsFailed": "実験一覧読込失敗",
        "loadTemplatesFailed": "ストラテジーテンプレート読込失敗",
        "promoteFailed": "候補の下書き昇格失敗",
        "submitFailed": "実験提出失敗。パラメータ空間が有効なJSONか確認してください。",
        "subscribeJobFailed": "実験ジョブイベントの購読失敗"
      },
      "noEvents": "イベントなし",
      "ruleVersionAlert": "現在の最小ループ：決定論的パラメータ実験。候補は下書き生成のみで自動公開・スケジュール・取引は行われません。",
      "selectJobToView": "ジョブ付き実験を選択してイベント表示。",
      "submitForm": {
        "baseTemplate": "ベースストラテジーテンプレート",
        "baseTemplatePlaceholder": "テンプレート選択",
        "baseTemplateRequired": "ベースストラテジーテンプレートを選択してください",
        "maxCandidates": "最大候補数",
        "objective": "目的",
        "parameterSpace": "パラメータ空間JSON",
        "parameterSpaceRequired": "パラメータ空間JSONを入力してください",
        "searchMethod": "探索方法",
        "submit": "実験提出",
        "title": "実験提出"
      },
      "subtitle": "パラメータ組合せを提出して自動実験、候補スコアリング、下書き生成。",
      "title": "ストラテジー実験"
    }
  }
} as const;
export default Experiment;
