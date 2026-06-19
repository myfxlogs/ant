// Auto-generated from proto/ant/v1/i18n/strategy_code_editor_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyCodeEditor = {
  "strategy": {
    "codeEditor": {
      "actions": {
        "copy": "复制",
        "preview": "预览信号",
        "saveAsTemplate": "保存为模板",
        "sendToAI": "发给AI修改",
        "sendToAIFixTitlePreview": "修复预览问题",
        "sendToAIFixTitleValidate": "验证未通过/有警告",
        "validate": "验证代码"
      },
      "aiPrompt": {
        "currentCodeTitle": "【当前代码】",
        "fenceEnd": "```",
        "intro": "请根据以下信息修改策略代码，使其通过验证并且预览信号执行成功。",
        "outputTitle": "【输出信息】",
        "outro": "仅返回用 ```python``` 包裹的修复后代码。",
        "problem": "【问题】{{title}}",
        "pythonFenceStart": "```python"
      },
      "cards": {
        "previewResult": "预览结果",
        "validationResult": "验证结果"
      },
      "hints": {
        "previewInfo": "预览将使用示例市场数据执行。"
      },
      "labels": {
        "account": "账号",
        "code": "策略代码",
        "disabledSuffix": "（已禁用）",
        "symbol": "品种",
        "timeframe": "周期"
      },
      "messages": {
        "copied": "代码已复制",
        "copyFailed": "复制失败，请手动复制",
        "enterCode": "请输入策略代码",
        "execFailed": "执行失败",
        "previewFailed": "预览失败",
        "previewOk": "预览完成",
        "previewSuccess": "预览成功",
        "savedAsTemplate": "已保存为模板",
        "selectAccount": "请选择账号",
        "validateError": "验证失败",
        "validateFailed": "代码验证失败",
        "validateOk": "代码验证通过"
      },
      "placeholders": {
        "code": "输入Python策略代码...",
        "loadingSymbols": "可用品种加载中…",
        "noSymbols": "暂无可用品种",
        "selectAccount": "选择账号",
        "selectAccountFirst": "先选账号",
        "selectSymbol": "选择品种"
      },
      "title": "策略编辑器"
    }
  }
} as const;
export default StrategyCodeEditor;
