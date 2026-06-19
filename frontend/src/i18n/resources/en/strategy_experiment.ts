// Auto-generated from proto/ant/v1/i18n/strategy_experiment_en.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyExperiment = {
  "strategy": {
    "experiment": {
      "candidates": {
        "column": {
          "actions": "Actions",
          "generateDraft": "Generate Draft",
          "grade": "Grade",
          "parameters": "Parameters",
          "rank": "Rank",
          "recommendation": "Recommendation",
          "score": "Score",
          "summary": "Summary",
          "viewCandidates": "View Candidates"
        },
        "title": "Candidates",
        "titleWithId": "Candidates: {{id}}"
      },
      "jobEventStream": "Job Event Stream",
      "list": {
        "column": {
          "actions": "Actions",
          "maxCandidates": "Max Candidates",
          "objective": "Objective",
          "searchMethod": "Search Method",
          "status": "Status",
          "viewCandidates": "View Candidates"
        },
        "title": "Experiment List"
      },
      "messages": {
        "candidatesGenerated": "Strategy experiment candidates generated",
        "draftGenerated": "Draft template generated: {{templateId}}",
        "loadCandidatesFailed": "Failed to load candidates",
        "loadExperimentsFailed": "Failed to load experiment list",
        "loadTemplatesFailed": "Failed to load strategy templates",
        "promoteFailed": "Failed to promote candidate to draft",
        "submitFailed": "Failed to submit experiment. Please verify the parameter space is valid JSON.",
        "subscribeJobFailed": "Failed to subscribe to experiment Job events"
      },
      "noEvents": "No events",
      "ruleVersionAlert": "Current minimal loop: deterministic parameter experiment. Candidates only generate drafts and will not auto-publish, schedule, or trade.",
      "selectJobToView": "Select an experiment with a Job to view events.",
      "submitForm": {
        "baseTemplate": "Base Strategy Template",
        "baseTemplatePlaceholder": "Select template",
        "baseTemplateRequired": "Please select a base strategy template",
        "maxCandidates": "Max Candidates",
        "objective": "Objective",
        "parameterSpace": "Parameter Space JSON",
        "parameterSpaceRequired": "Please enter parameter space JSON",
        "searchMethod": "Search Method",
        "submit": "Submit Experiment",
        "title": "Submit Experiment"
      },
      "subtitle": "Submit parameter combinations to automatically run experiments, score candidate strategies, and generate drafts.",
      "title": "Strategy Experiment"
    }
  }
} as const;
export default StrategyExperiment;
