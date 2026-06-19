// Auto-generated from proto/ant/v1/i18n/strategy_code_editor_ja.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const CodeEditor = {
  "strategy": {
    "codeEditor": {
      "actions": {
        "copy": "コピー",
        "preview": "シグナルプレビュー",
        "saveAsTemplate": "テンプレートとして保存",
        "sendToAI": "AI に修正を依頼",
        "sendToAIFixTitlePreview": "プレビュー問題修正",
        "sendToAIFixTitleValidate": "検証失敗 / 警告あり",
        "validate": "コード検証"
      },
      "aiPrompt": {
        "currentCodeTitle": "【現在のコード】",
        "fenceEnd": "```",
        "intro": "以下の情報に基づいて戦略コードを修正し、検証に通り、プレビュー実行が成功するようにしてください。",
        "outputTitle": "【出力】",
        "outro": "```python```で囲まれた修正コードのみ返してください。",
        "problem": "【問題】{{title}}",
        "pythonFenceStart": "```python"
      },
      "cards": {
        "previewResult": "プレビュー結果",
        "validationResult": "検証結果"
      },
      "hints": {
        "previewInfo": "サンプルデータで実行プレビューします。"
      },
      "labels": {
        "account": "口座",
        "code": "戦略コード",
        "disabledSuffix": "（無効）",
        "symbol": "銘柄",
        "timeframe": "時間足"
      },
      "messages": {
        "copied": "コピーしました",
        "copyFailed": "コピーに失敗しました。手動でコピーしてください",
        "enterCode": "戦略コードを入力してください",
        "execFailed": "実行に失敗しました",
        "previewFailed": "プレビューに失敗しました",
        "previewOk": "プレビューが完了しました",
        "previewSuccess": "プレビューに成功しました",
        "savedAsTemplate": "テンプレートとして保存しました",
        "selectAccount": "口座を選択してください",
        "validateError": "検証エラー",
        "validateFailed": "検証に失敗しました",
        "validateOk": "検証に成功しました"
      },
      "placeholders": {
        "code": "Python 戦略コードを入力...",
        "loadingSymbols": "銘柄を読み込み中...",
        "noSymbols": "利用可能な銘柄なし",
        "selectAccount": "口座を選択",
        "selectAccountFirst": "先に口座を選択",
        "selectSymbol": "銘柄を選択"
      },
      "title": "戦略エディタ"
    }
  }
} as const;
export default CodeEditor;
