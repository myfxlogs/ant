const strategyImport = {
  importEA: {
    // Additions to strategy.importEA namespace (existing keys stay in proto-generated i18n)
    migration: "Strategy Import",
    aiTranslate: "AI Translate",
    analyze: "Analyze Structure",
    confirmImport: "Confirm Import",
    analyzeFailed: "Analysis failed. Please try again.",
    importFailed: "Import failed. Please try again.",
    analyzing: "Analyzing strategy structure...",
    exportPreview: "Exported Go Source (read-only / secondary development, not execution path)",
    exportPreviewDesc: "Execution uses the original MQL via the IR interpreter. This Go code is for reference only.",
  },
};

export default { strategy: strategyImport };
