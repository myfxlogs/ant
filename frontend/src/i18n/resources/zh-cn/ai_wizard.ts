// Auto-generated from proto/ant/v1/i18n/ai_wizard_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const AiWizard = {
  "ai": {
    "wizard": {
      "generate": {
        "modals": {
          "final": {
            "title": "代码已生成，建议点击\"验证代码\"确认通过校验"
          }
        },
        "status": {
          "running": {
            "code": "代码生成中",
            "generic": "{{title}}中",
            "risk": "风控与执行约束中",
            "signals": "信号与指标设计中",
            "style": "市场状态/风格推荐中"
          },
          "done": "完成",
          "error": "失败",
          "idle": "等待中",
          "inProgress": "进行中"
        },
        "actions": {
          "abort": "中止",
          "goValidate": "去校验",
          "hide": "隐藏",
          "regenerateSummary": "重新生成总结",
          "rerun": "重新生成",
          "runAgents": "多个专家分析 + 代码生成"
        },
        "cards": {
          "resultsTitle": "Multiple experts\\\\\\\\\\\\\\\\"
        },
        "hints": {
          "afterGenerated": "生成完成后进入下一步进行验证/回测/上线"
        },
        "labels": {
          "elapsed": "耗时"
        },
        "sections": {
          "output": "模型返回（Output）",
          "prompt": "发送给模型（Prompt）",
          "spec": "规格（Spec）"
        }
      },
      "publishBacktest": {
        "modals": {
          "score": {
            "title": "评分确认"
          },
          "status": {
            "title": "回测进行中"
          }
        },
        "actions": {
          "close": "关闭",
          "confirm": "确认",
          "inProgress": "进行中",
          "retry": "重试",
          "runInBackground": "后台运行",
          "startBacktest": "开始回测",
          "succeeded": "成功"
        },
        "cards": {
          "backtestTitle": "回测",
          "scoreCardTitle": "评分卡"
        },
        "labels": {
          "confirmed": "已确认",
          "elapsed": "耗时",
          "overallScore": "综合评分",
          "scoringProgress": "评分进度",
          "status": "状态"
        },
        "draftName": "回测 {{datetime}} {{symbol}} {{timeframe}}",
        "draftNameShort": "回测 {{symbol}} {{timeframe}}"
      },
      "setup": {
        "modals": {
          "deleteDataset": {
            "content": "确定删除当前选中的冻结数据集吗？",
            "ok": "删除",
            "title": "删除数据集"
          }
        },
        "actions": {
          "deleteCurrentDataset": "删除当前数据集",
          "freezeFromCurrentRange": "从当前范围冻结",
          "refreshDataset": "刷新"
        },
        "cards": {
          "constraintsAndGoalTitle": "约束与目标",
          "hardConstraintsTitle": "硬约束",
          "hintsTitle": "提示",
          "tradeAndDataTitle": "交易与数据"
        },
        "dataModes": {
          "dataset": "冻结数据集",
          "klineRange": "历史K线范围"
        },
        "hints": {
          "nextWillGenerateCode": "下一步将开始生成策略代码。",
          "tradeDataNextStep": "填写完成后点击\"下一步\"，进入约束与目标设置。"
        },
        "labels": {
          "account": "账号",
          "backtestRange": "回测范围",
          "dataset": "冻结数据集",
          "historicalData": "历史数据",
          "intent": "策略目标/想法",
          "macroEvents": "宏观事件",
          "macroModule": "宏观模块",
          "maxDrawdownPct": "最大回撤(%)",
          "maxTradesPerDay": "日内最多交易次数",
          "riskPerTradePct": "单笔风险(%)",
          "symbol": "品种",
          "timeframe": "周期"
        },
        "macro": {
          "off": "关闭",
          "on": "开"
        },
        "messages": {
          "datasetDeleted": "数据集已删除"
        },
        "placeholders": {
          "intentExample": "示例：突破趋势跟随；避开高波动；偏好更高胜率...",
          "macroExample": "Example:\\\\\\\\\\\\\\\\n2024-01-03 21:15 FOMC minutes\\\\\\\\\\\\\\\\n2024-01-05 20:30 NFP",
          "selectAccount": "选择账号",
          "selectFrozenDataset": "选择冻结数据集",
          "selectSymbol": "选择品种",
          "selectTimeframe": "选择周期"
        },
        "validations": {
          "enterIntent": "请输入策略目标/想法",
          "selectAccount": "请选择账号",
          "selectDataset": "请选择数据集",
          "selectSymbol": "请选择品种",
          "selectTimeframe": "请选择周期"
        }
      },
      "prompts": {
        "base": {
          "account": "账号: {{accountId}}",
          "constraints": "约束: 最大回撤={{maxDrawdownPct}}% 单笔风险={{riskPerTradePct}}% 日内最多交易={{maxTradesPerDay}} 次",
          "data": "数据: {{dataSpec}}",
          "empty": "(空)",
          "macroDisabled": "宏观事件: 不使用",
          "macroEnabled": "Macro events (user-provided):\\\\\\\\\\\\\\\\n{{text}}",
          "params": "Parameters (defs+current values; injected into context[\"params\"] at runtime):\\\\\\\\\\\\\\\\n{{params}}",
          "symbol": "品种: {{symbol}}",
          "timeframe": "周期: {{timeframe}}",
          "userIntent": "User strategy goal (natural language):\\\\\\\\\\\\\\\\n{{intent}}"
        },
        "dataSpec": {
          "dataset": "使用冻结数据集 datasetId={{datasetId}}",
          "klineRange": "使用历史K线范围 from={{from}} to={{to}}"
        },
        "summary": {
          "codeTitle": "代码如下：",
          "intro": "你是量化策略解释助手。请用简洁中文（要点形式，最多 12 行）解释下面这段 AlphaForge Python 策略代码的核心思路，帮助用户判断是否符合预期。",
          "mustInclude1": "1) 策略类型/范式（趋势/均值/突破/动量/网格等，若无法判断则写\"无法确定\"）",
          "mustInclude2": "2) 主要入场条件（用 2~4 条要点）",
          "mustInclude3": "3) 主要出场/止损止盈/风控约束（用 2~4 条要点）",
          "mustInclude4": "4) 适用/不适用场景各 1 条",
          "mustIncludeTitle": "必须包含：",
          "userIntent": "User expectation (natural language):\\\\\\\\\\\\\\\\n{{intent}}"
        },
        "upstream": {
          "risk": "【Risk control conclusion】\\\\\\\\\\\\\\\\n{{text}}",
          "sectionTitle": "【上游 Agent 结论（原样提供）】",
          "signals": "【Signal design conclusion】\\\\\\\\\\\\\\\\n{{text}}",
          "style": "【Market condition/style conclusion】\\\\\\\\\\\\\\\\n{{text}}"
        }
      },
      "publish": {
        "actions": {
          "publishTemplate": "发布模板",
          "startBacktest": "回测（异步任务）",
          "validateCode": "验证代码"
        },
        "cards": {
          "codeTitle": "1) 策略代码（可编辑）",
          "launchTitle": "3) 上线调度",
          "scoreCardTitle": "2) 回测评分卡"
        },
        "messages": {
          "validateFailed": "validate 未通过",
          "validateOk": "validate 通过"
        },
        "placeholders": {
          "codeEditable": "这里会自动填入 AI 生成的代码，你也可以手动修改。"
        }
      },
      "strategyParams": {
        "actions": {
          "addParam": "新增参数",
          "delete": "删除",
          "exportJson": "导出 JSON",
          "importJson": "导入 JSON"
        },
        "hints": {
          "intro": "这些参数会：",
          "line1": "1) 保存到模板 parameters",
          "line2": "2) 创建调度时写入 schedule.parameters（map<string,string>）",
          "line3Prefix": "3) 运行时系统会把参数注入到 Python 策略的"
        },
        "labels": {
          "default": "默认值",
          "description": "描述",
          "label": "标签",
          "max": "最大值",
          "min": "最小值",
          "name": "名称",
          "options": "options（select 可用，逗号分隔）",
          "step": "步长",
          "type": "类型",
          "value": "value（调度当前值）"
        },
        "messages": {
          "copied": "已复制",
          "copyFailed": "复制失败",
          "importFormatInvalid": "导入格式错误：需要是数组，或 { \"paramDefs\": [...] }",
          "importMissingName": "导入失败：存在缺少 name 的参数",
          "imported": "已导入 {{count}} 个参数",
          "jsonParseFailed": "JSON 解析失败"
        },
        "modals": {
          "copyAndClose": "复制并关闭",
          "exportTitle": "导出参数 JSON",
          "importOk": "导入",
          "importTitle": "导入参数 JSON"
        },
        "placeholders": {
          "defaultExample": "例如：10",
          "description": "说明",
          "importJson": "粘贴参数 JSON（数组 或 {\"paramDefs\": [...]}）",
          "label": "展示名",
          "nameExample": "例如：fast",
          "optionsExample": "例如：low,medium,high",
          "value": "留空则使用 default"
        },
        "types": {
          "bool": "布尔",
          "number": "数字",
          "select": "选择",
          "string": "字符串"
        },
        "validations": {
          "nameRequired": "name 必填",
          "typeRequired": "type 必填"
        },
        "empty": "暂无参数。你可以添加如 fast/slow/risk_per_trade 等参数，让策略更模板化。",
        "paramCardTitle": "参数 #{{index}}",
        "title": "策略参数（可选）"
      },
      "actions": {
        "cancel": "取消",
        "next": "下一步",
        "prev": "上一步"
      },
      "agents": {
        "codeTitle": "代码生成",
        "riskTitle": "风控与执行约束",
        "signalsTitle": "信号与指标设计",
        "styleTitle": "市场状态/风格推荐"
      },
      "messages": {
        "agentFailed": "{{title}} 失败",
        "aiRequestTimeout": "AI 请求超时（>{{seconds}}s）",
        "backtestCreated": "回测任务已创建",
        "backtestNotDoneWait": "回测尚未完成，请等待评分卡状态变为\"成功/失败/已取消\"后再继续",
        "chatAborted": "已中止与模型对话",
        "codeInvalidFixAndContinue": "代码验证未通过，请修复后再继续",
        "confirmScoreFirst": "请先在评分弹窗中确认评分结果",
        "createBacktestFailed": "创建回测失败",
        "createDraftFailed": "创建草稿失败",
        "createScheduleFailed": "创建调度失败",
        "datasetFrozenCreated": "已冻结创建 dataset",
        "draftNotCreated": "草稿未创建",
        "draftSaved": "草稿已保存",
        "fillRequired": "请先补全必填项",
        "fillRequiredWithFields": "请先补全必填项：{{fields}}",
        "freezeDatasetFailed": "冻结 dataset 失败",
        "generateCodeFirst": "请先生成策略代码",
        "inputIntentFirst": "请先输入策略目标/想法",
        "loadAccountsFailed": "加载账号失败",
        "loadDatasetFailed": "加载 dataset 失败",
        "loadSymbolsFailed": "加载品种失败",
        "modelReturnedEmpty": "模型返回为空",
        "noCodeToBacktest": "暂无代码可回测",
        "noCodeToValidate": "暂无代码可验证",
        "noPythonCodeBlock": "代码 Agent 未输出 ```python 代码块```，请在结果中检查",
        "publishFailed": "发布失败",
        "publishTemplateFirst": "请先发布模板",
        "publishedNoId": "已发布，但未拿到返回 id（请在策略管理中确认）",
        "saveFailed": "保存失败",
        "scheduleAlreadyExists": "该账号下已存在相同策略调度（模板+品种+周期相同），请勿重复创建。",
        "scheduleCreated": "调度已创建",
        "scheduleCreatedAndEnabled": "调度已创建并启用",
        "startBacktestFirst": "请先点击\"回测（异步任务）\"启动回测",
        "templatePublished": "模板已发布",
        "userAborted": "用户已中止",
        "validateCodeFirst": "请先点击\"验证代码\"",
        "validateError": "验证失败",
        "validateFailed": "验证未通过",
        "validateOk": "验证通过",
        "watchBacktestRunFailed": "watchBacktestRun 失败"
      },
      "schedule": {
        "defaultName": "AI 调度 {{symbol}} {{timeframe}}"
      },
      "steps": {
        "generate": "生成策略",
        "publishBacktest": "回测上线-回测",
        "publishCode": "回测上线-代码",
        "publishLaunch": "回测上线-上线",
        "setup": "基础信息"
      },
      "template": {
        "defaultDescription": "AI 向导生成",
        "defaultName": "AI 策略 {{title}}"
      },
      "currentModel": "当前模型：{{model}}",
      "subtitle": "每步一个页面，可前进/后退",
      "title": "AI 策略向导"
    }
  }
} as const;
export default AiWizard;
