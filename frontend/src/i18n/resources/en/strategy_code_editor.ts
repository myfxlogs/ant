// Auto-generated from proto/ant/v1/i18n/strategy_code_editor_en.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyCodeEditor = {
  "strategy": {
    "codeEditor": {
      "actions": {
        "copy": "Copy",
        "preview": "Preview",
        "saveAsTemplate": "Save as template",
        "sendToAI": "Send to AI",
        "sendToAIFixTitlePreview": "Fix preview issues",
        "sendToAIFixTitleValidate": "Fix validation issues",
        "validate": "Validate"
      },
      "aiPrompt": {
        "currentCodeTitle": "Current code",
        "fenceEnd": "```",
        "intro": "Please help fix the strategy based on the following issues:",
        "outputTitle": "Output fixed code",
        "outro": "Return only the fixed code wrapped in ```python```.",
        "problem": "Problem",
        "pythonFenceStart": "```python"
      },
      "cards": {
        "previewResult": "Preview result",
        "validationResult": "Validation result"
      },
      "hints": {
        "previewInfo": "Preview will execute with sample market data."
      },
      "labels": {
        "account": "Account",
        "code": "Strategy code",
        "disabledSuffix": " (disabled)",
        "symbol": "Symbol",
        "timeframe": "Timeframe"
      },
      "messages": {
        "copied": "Copied",
        "copyFailed": "Copy failed",
        "enterCode": "Please enter strategy code",
        "execFailed": "Execution failed",
        "previewFailed": "Preview failed",
        "previewOk": "Preview completed",
        "previewSuccess": "Preview succeeded",
        "savedAsTemplate": "Saved as template",
        "selectAccount": "Please select an account",
        "validateError": "Validation error",
        "validateFailed": "Validation failed",
        "validateOk": "Validation passed"
      },
      "placeholders": {
        "code": "Paste strategy code here",
        "loadingSymbols": "Loading symbols...",
        "noSymbols": "No symbols available",
        "selectAccount": "Select an account",
        "selectAccountFirst": "Select an account first",
        "selectSymbol": "Select a symbol"
      },
      "title": "Strategy editor"
    }
  }
} as const;
export default StrategyCodeEditor;
