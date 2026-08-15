// Auto-generated from proto/ant/v1/i18n/strategy_schedules_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategySchedules = {
  "strategy": {
    "schedules": {
      "editModal": {
        "advanced": {
          "triggerModeOptions": {
            "hf": "高频信号流",
            "stable": "稳定（K线/周期）"
          },
          "fixedIntervalSeconds": "固定间隔(秒)",
          "fixedIntervalSecondsExtra": "可选。填写后将按固定间隔执行（不再自动跟随周期）。例如：60 表示每 60 秒执行一次",
          "hfCooldownMs": "高频模式：最小触发间隔(ms)",
          "hfCooldownMsExtra": "用于去抖：两次评估/下单之间的最小间隔",
          "parametersJson": "参数(JSON对象)",
          "parametersJsonExtra": "策略的 JSON 参数",
          "stableOverrideIntervalSeconds": "稳定模式高级：间隔(秒)",
          "stableOverrideIntervalSecondsExtra": "可选。默认绑定周期(timeframe)。填写后将覆盖稳定模式的触发间隔",
          "timeframe": "周期",
          "timeframeExtra": "默认即可。仅用于K线与指标计算",
          "title": "高级设置",
          "triggerMode": "触发模式",
          "triggerModeExtra": "稳定：按K线/周期触发（更稳但有延迟）；高频：报价流触发（更快但噪声大，需要去抖）"
        },
        "autoName": {
          "strategy": "策略"
        },
        "fields": {
          "account": "账号",
          "cronExpression": "Cron 表达式",
          "cronExtra": "标准 5 段：分钟 小时 日 月 周。例如：*/5 * * * * 每5分钟；0 9 * * 1-5 工作日9点",
          "enableExtra": "创建后启用调度",
          "intervalSeconds": "间隔(秒)",
          "intervalSecondsExtra": "自动跟随周期(timeframe)，无需修改",
          "lot": "手数(Lot)",
          "lotExtra": "下单手数，建议从 0.01 开始",
          "name": "名称",
          "runFrequency": "运行频率",
          "symbol": "品种",
          "template": "模板",
          "templateExtra": "来自「策略管理」中保存的模板"
        },
        "placeholders": {
          "name": "例如：EURUSD M5 早盘策略",
          "selectAccountFirst": "先选账号",
          "symbol": "选择品种"
        },
        "runFrequencyExtra": {
          "byTimeframe": "按时间周期运行",
          "cron": "高级：使用 Cron 精确控制执行时间"
        },
        "runFrequencyOptions": {
          "byTimeframe": "按周期触发（推荐）",
          "cron": "Cron 表达式"
        },
        "title": {
          "create": "新建调度任务",
          "edit": "编辑调度任务"
        },
        "validation": {
          "accountRequired": "请选择账号",
          "cronRequired": "请输入 cron",
          "lotRequired": "请输入手数",
          "nameRequired": "请输入名称",
          "runFrequencyRequired": "请选择运行频率",
          "symbolRequired": "请选择品种",
          "templateRequired": "请选择模板",
          "timeframeRequired": "请选择周期",
          "triggerModeRequired": "请选择触发模式"
        }
      },
      "health": {
        "runLogs": {
          "status": {
            "failed": "失败",
            "running": "运行中",
            "stopped": "已停止",
            "success": "成功"
          },
          "signalType": "信号(用于下单)"
        },
        "fields": {
          "configKey": "配置键",
          "failedRuns": "执行失败次数",
          "grade": "健康级别",
          "lastRunAt": "最后运行时间",
          "latestError": "最近错误",
          "latestProfit": "最近成交盈亏",
          "latestTicket": "最近成交订单号",
          "rule": "判定依据",
          "successOverTotal": "执行成功/总次数",
          "thresholds": "当前阈值"
        },
        "grade": {
          "alert": "警报",
          "healthy": "健康",
          "noSample": "无样本",
          "pending": "待检测",
          "watch": "关注"
        },
        "messages": {
          "clickRefresh": "点击刷新加载健康数据",
          "loadFailed": "加载健康检查数据失败"
        },
        "notes": {
          "alert": "成功率低。请立即检查策略/账户状况。",
          "healthy": "成功率高且失败次数可控。",
          "noSample": "样本不足，至少需要 {{minSampleSize}} 条运行记录。",
          "pending": "请先执行健康检查。",
          "watch": "成功率达到关注阈值（>= {{yellowSuccessRate}}%），建议持续观察。"
        },
        "sections": {
          "orders": "最近订单记录",
          "runLogs": "最近执行日志"
        },
        "value": {
          "buy": "买入",
          "hold": "持有",
          "sell": "卖出"
        },
        "summaryBanner": "健康分级：{{grade}}；最近样本 {{totalRuns}} 次，成功率 {{successRate}}%",
        "thresholdsSummary": "min_sample_size={{minSampleSize}}；绿色：成功率>={{greenSuccessRate}}% 且失败次数<={{greenMaxFailedRuns}}；黄色：成功率>={{yellowSuccessRate}}%",
        "title": "策略健康检查 {{name}}"
      },
      "triggerModal": {
        "actions": {
          "confirmOrder": "确认下单",
          "rerun": "重新运行"
        },
        "cards": {
          "logs": "执行日志",
          "signal": "信号(用于下单)"
        },
        "confirmOrder": {
          "ok": "确认",
          "title": "确认下单"
        },
        "messages": {
          "signalNotOrderable": "信号不可下单"
        },
        "summary": {
          "account": "账号",
          "scheduleName": "调度名称",
          "symbol": "品种",
          "timeframe": "周期"
        },
        "emptyLogs": "(无日志)",
        "emptySignal": "无信号",
        "title": "立即执行(直接下单)"
      },
      "actions": {
        "create": "新建调度",
        "healthCheck": "健康检查",
        "logs": "执行日志",
        "runNow": "立即运行"
      },
      "deleteConfirm": {
        "title": "删除此调度？"
      },
      "format": {
        "cron": "定时: {{expr}}",
        "interval": "每 {{s}}秒"
      },
      "messages": {
        "defaultTemplateNotFound": "默认模板不存在，请刷新页面重试",
        "executeFailed": "执行失败",
        "importDefaultTemplateFailedNoId": "导入默认模板失败：未返回模板ID",
        "noOrderableSignal": "没有可下单的信号",
        "orderFailed": "下单失败",
        "orderSubmitted": "已提交下单",
        "parametersParseFailed": "参数解析失败",
        "signalHoldCannotOrder": "当前信号为 hold/无交易动作，不能下单",
        "strategyExecuteFailed": "策略执行失败",
        "templateCodeEmptyCannotExecute": "模板 code 为空，无法执行",
        "volumeInvalid": "下单手数无效（volume 必须 > 0）"
      },
      "status": {
        "disabled": "已禁用",
        "running": "运行中"
      },
      "table": {
        "account": "账号",
        "actions": "操作",
        "lastRun": "最后运行时间",
        "name": "名称",
        "schedule": "计划",
        "status": "状态",
        "template": "模板",
        "tradeParams": "交易参数"
      },
      "templateVisibility": {
        "private": "私有",
        "public": "公开"
      },
      "validation": {
        "parametersMustBeJsonObject": "参数必须为 JSON 对象"
      },
      "createSchedule": "创建调度",
      "enableCount": "启用次数",
      "nextRunAt": "下次运行",
      "title": "策略调度"
    }
  }
} as const;
export default StrategySchedules;
