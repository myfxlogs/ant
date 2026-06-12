const strategy = {
  strategy: {
    validation: {
      passed: '代碼驗證通過',
      notPassed: '代碼驗證未通過',
      riskEval: {
        title: '風險評估',
        passed: '風險評估通過',
        notPassed: '風險評估未通過',
        highRisk: '風險評估：高風險',
        unreliable: '風險評估：不可靠',
        riskHigh: '風險等級：高',
        riskUnreliable: '風險評估：不可靠（isReliable=false）',
        riskLoading: '後端風險評估仍在計算中'
      }
    },
    codeEditor: {
      title: '策略編輯器',
      labels: {
        account: '帳戶',
        symbol: '品種',
        timeframe: '時間週期',
        code: '策略代碼',
        disabledSuffix: '（已禁用）'
      },
      actions: {
        copy: '複製',
        validate: '驗證代碼',
        preview: '預覽訊號',
        saveAsTemplate: '保存為模板',
        sendToAI: '發給AI修改',
        sendToAIFixTitleValidate: '驗證未通過/有警告',
        sendToAIFixTitlePreview: '預覽訊號執行失敗/需要優化'
      },
      placeholders: {
        selectAccount: '選擇帳號',
        selectAccountFirst: '請先選擇帳號',
        loadingSymbols: '可用品種載入中…',
        selectSymbol: '選擇品種',
        noSymbols: '未取得到品種列表',
        code: '輸入Python策略代碼...'
      },
      cards: {
        validationResult: '驗證結果',
        previewResult: '預覽結果'
      },
      hints: {
        previewInfo: 'Preview 取最近 N 根K線（默認 500，配置：strategy.preview_bars）；回測取最近 N 個月（默認 3，配置：strategy.backtest_window_months）。'
      },
      messages: {
        enterCode: '請輸入策略代碼',
        selectAccount: '請選擇帳號',
        validateOk: '代碼驗證通過',
        validateFailed: '代碼驗證失敗',
        validateError: '驗證失敗',
        previewOk: '預覽完成',
        previewSuccess: '預覽成功',
        previewFailed: '預覽失敗',
        execFailed: '執行失敗',
        savedAsTemplate: '已保存為模板',
        copied: '代碼已複製',
        copyFailed: '複製失敗，請手動複製'
      },
      aiPrompt: {
        intro: '請根據以下資訊修改策略代碼，使其通過驗證並且預覽訊號執行成功。',
        problem: '【問題】{{title}}',
        currentCodeTitle: '【當前代碼】',
        outputTitle: '【輸出資訊】',
        outro: '請直接給出修改後的完整代碼（用 \`\`\`python 包裹），並說明修改點。',
        pythonFenceStart: '\`\`\`python',
        fenceEnd: '\`\`\`'
      }
    },
    schedules: {
      title: '策略調度',
      createSchedule: '建立調度',
      format: {
        interval: '每 {{s}}秒',
        cron: '定時: {{expr}}',
      },
      actions: {
        create: '新建調度',
        logs: '日誌',
        healthCheck: '健康檢查',
        runNow: '立即執行'
      },
      health: {
        title: '策略健康檢查 {{name}}',
        summaryBanner: '健康分級：{{grade}}；最近樣本 {{totalRuns}} 次，成功率 {{successRate}}%',
        grade: {
          pending: '待檢測',
          noSample: '無樣本',
          healthy: '健康',
          watch: '關注',
          alert: '告警'
        },
        notes: {
          pending: '請先執行健康檢查。',
          noSample: '樣本不足，至少需要 {{minSampleSize}} 筆運行記錄。',
          healthy: '成功率高且失敗次數可控。',
          watch: '成功率達到關注門檻（>= {{yellowSuccessRate}}%），建議持續觀察。',
          alert: '成功率偏低，建議立即排查策略與帳戶狀態。'
        },
        fields: {
          grade: '健康級別',
          rule: '判定依據',
          thresholds: '當前門檻',
          configKey: '設定鍵',
          lastRunAt: '最後運行時間',
          latestTicket: '最近成交 Ticket',
          successOverTotal: '執行成功/總次數',
          failedRuns: '執行失敗次數',
          latestProfit: '最近成交盈虧',
          latestError: '最近錯誤訊息'
        },
        thresholdsSummary: 'min_sample_size={{minSampleSize}}；綠色：成功率>={{greenSuccessRate}}% 且失敗次數<={{greenMaxFailedRuns}}；黃色：成功率>={{yellowSuccessRate}}%',
        sections: {
          runLogs: '最近執行日誌',
          orders: '最近成交記錄'
        },
        runLogs: {
          signalType: '信號'
        },
        messages: {
          loadFailed: '載入健康檢查資料失敗',
          clickRefresh: '點擊重新整理以載入健康資料'
        }
      },
      editModal: {
        title: {
          create: '新建調度任務',
          edit: '編輯調度任務'
        },
        autoName: {
          strategy: '策略'
        },
        fields: {
          template: '模板',
          templateExtra: '來自「策略管理」中保存的模板',
          account: '帳號',
          name: '名稱',
          symbol: '品種',
          lot: '手數(Lot)',
          lotExtra: '下單手數，建議從 0.01 開始',
          runFrequency: '運行頻率',
          cronExpression: 'Cron 表達式',
          cronExtra: '標準 5 段：分鐘 小時 日 月 週。例如：*/5 * * * * 每5分鐘；0 9 * * 1-5 工作日9點',
          intervalSeconds: '間隔(秒)',
          intervalSecondsExtra: '自動跟隨週期(timeframe)，無需修改',
          enableExtra: 'EA 體驗：啟用後會持續運行，直到你手動停用'
        },
        placeholders: {
          name: '例如：EURUSD M5 早盤策略',
          selectAccountFirst: '先選帳號',
          symbol: '選擇品種'
        },
        validation: {
          templateRequired: '請選擇模板',
          accountRequired: '請選擇帳號',
          nameRequired: '請輸入名稱',
          symbolRequired: '請輸入品種',
          lotRequired: '請輸入手數',
          runFrequencyRequired: '請選擇運行頻率',
          cronRequired: '請輸入 cron',
          timeframeRequired: '請選擇週期',
          triggerModeRequired: '請選擇觸發模式'
        },
        runFrequencyExtra: {
          cron: '高級：使用 Cron 精確控制執行時間',
          byTimeframe: '預設：跟隨週期(timeframe)觸發（最像EA的OnTick/OnTimer體驗）'
        },
        runFrequencyOptions: {
          byTimeframe: '按週期觸發（推薦）',
          cron: 'Cron（高級）'
        },
        advanced: {
          title: '高級設定',
          fixedIntervalSeconds: '固定間隔(秒)',
          fixedIntervalSecondsExtra: '可選。填寫後將按固定間隔執行（不會再自動跟隨週期）。例如：60 表示每 60 秒執行一次',
          timeframe: '週期',
          timeframeExtra: '預設即可。僅用於K線與指標計算，不影響EA本質（策略驅動交易）',
          triggerMode: '觸發模式',
          triggerModeExtra: '穩定：按K線/週期觸發（信號更穩但有延遲）；高頻：報價流觸發（更快但噪聲大，需要去抖）',
          triggerModeOptions: {
            stable: '穩定（K線/週期）',
            hf: '高頻（報價/tick）'
          },
          stableOverrideIntervalSeconds: '穩定模式高級：間隔(秒)',
          stableOverrideIntervalSecondsExtra: '可選。預設綁定週期(timeframe)。填寫後將覆蓋穩定模式的觸發間隔',
          hfCooldownMs: '高頻模式：最小觸發間隔(ms)',
          hfCooldownMsExtra: '用於去抖：兩次評估/下單之間的最小間隔',
          parametersJson: '參數(JSON對象)',
          parametersJsonExtra: '用於給策略代碼傳參，形如 key->value。值會以字串形式傳給後端執行（必要時可在策略裡自行轉換）。示例：{ "fast": 10, "slow": 20, "risk": "low" }'
        }
      },
      triggerModal: {
        title: '立即執行(直接下單)',
        actions: {
          rerun: '重新執行',
          confirmOrder: '確認下單'
        },
        confirmOrder: {
          title: '確認要下單嗎？',
          ok: '確認下單'
        },
        summary: {
          scheduleName: '調度名稱',
          account: '帳號',
          symbol: '品種',
          timeframe: '週期'
        },
        messages: {
          signalNotOrderable: '當前信號不可下單：需要 buy/sell 且 volume > 0'
        },
        cards: {
          logs: '執行日誌',
          signal: '信號(用於下單)'
        },
        emptyLogs: '(無日誌)',
        emptySignal: '(無信號)'
      },
      table: {
        name: '名稱',
        template: '模板',
        account: '帳號',
        tradeParams: '交易參數',
        schedule: '計劃',
        status: '狀態',
        lastRun: '最近運行',
        actions: '操作'
      },
      templateVisibility: {
        public: '公開',
        private: '私有'
      },
      status: {
        running: '運行中',
        disabled: '已停用'
      },
      nextRunAt: '下次運行',
      enableCount: '啟用次數',
      deleteConfirm: {
        title: '確認刪除該調度任務？'
      },
      validation: {
        parametersMustBeJsonObject: '參數必須是 JSON 對象'
      },
      messages: {
        parametersParseFailed: '參數解析失敗',
        defaultTemplateNotFound: '預設模板不存在，請刷新頁面重試',
        importDefaultTemplateFailedNoId: '匯入預設模板失敗：未返回模板ID',
        templateCodeEmptyCannotExecute: '模板 code 為空，無法執行',
        executeFailed: '執行失敗',
        strategyExecuteFailed: '策略執行失敗',
        noOrderableSignal: '沒有可下單的信號',
        signalHoldCannotOrder: '當前信號為 hold/無交易動作，不能下單',
        volumeInvalid: '下單手數無效（volume 必須 > 0）',
        orderSubmitted: '已提交下單',
        orderFailed: '下單失敗'
      },
      createSchedule: '建立计划'
    },
    scheduleLogs: {
      title: '記錄',
      titleWithName: '記錄 - {{name}}',
      tabs: {
        exec: '運行記錄',
        orders: '交易記錄',
        execLogs: '执行日誌',
        orderLogs: '訂單日誌'
      },
      messages: {
        missingScheduleId: '缺少 scheduleId'
      },
      summary: {
        name: '名稱',
        status: '狀態',
        trade: '交易',
        enableCount: '啟用次數',
        lastRun: '最近運行',
        lastError: '最近錯誤'
      },
      execStatus: {
        pending: '待執行',
        running: '運行中',
        completed: '已完成',
        failed: '失敗',
        skipped: '已跳過'
      },
      operationStatus: {
        success: '成功',
        failed: '失敗',
        running: '執行中'
      },
      execTable: {
        time: '時間',
        action: '操作',
        status: '狀態',
        durationMs: '耗時(ms)',
        error: '錯誤',
        execute: '執行'
      },
      ordersTable: {
        time: '時間',
        side: '方向',
        symbol: '品種',
        lots: '手數',
        openPrice: '開倉價',
        closePrice: '平倉價',
        profit: '盈虧',
        ticket: '訂單號'
      },
      orderSide: {
        buy: '市價買入',
        sell: '市價賣出',
        close: '平倉',
        buyLimit: '限價買入',
        sellLimit: '限價賣出',
        buyStop: '突破買入',
        sellStop: '突破賣出',
        buyStopLimit: '限價突破買',
        sellStopLimit: '限價突破賣'
      },
      scheduleIdLabel: '排程ID:',
      status: {
        success: '成功',
        failed: '失敗'
      },
      action: {
        start: '啟動',
        stop: '停止',
        restart: '重启'
      }
    },
    templates: {
      title: '策略模板',
      tabs: {
        system: '系統模板',
        user: '自建模板'
      },
      copySuffix: ' (副本)',
      scheduleName: '{{symbol}} {{timeframe}} {{name}}',
      badges: {
        preset: '預設'
      },
      visibility: {
        public: '公開',
        private: '私有'
      },
      codeModal: {
        title: '策略代碼',
        actions: {
          copy: '複製'
        }
      },
      status: {
        draft: '草稿',
        published: '已發布'
      },
      actions: {
        create: '新建模板',
        createTemplate: '新建模板',
        edit: '編輯',
        delete: '刪除',
        copy: '複製',
        viewCode: '查看代碼',
        backtest: '回測',
        launchSchedule: '上線到調度'
      },
      table: {
        name: '名稱',
        description: '描述',
        tags: '標籤',
        visibility: '可見性',
        status: '狀態',
        useCount: '使用次數',
        createdAt: '建立時間',
        updatedAt: '更新時間',
        actions: '操作',
        loadingDefault: '正在載入預設模板...',
        defaultHint: '預設',
        emptyUser: '尚無使用者模板。請點擊上方「建立模板」開始使用。'
      },
      scheduleLaunch: {
        title: '調度上線',
        noRun: '暫無回測運行',
        backtestRunningHint: '回測正在運行，請稍候。',
        score: '評分',
        keyMetrics: '關鍵指標',
        launchSection: '上線調度',
        actions: {
          publishTemplate: '發布模板',
          createScheduleNoEnable: '建立調度',
          createAndEnable: '建立並啟用',
          create: '建立计划',
          addAccount: '新增账戶',
          updateTradingPassword: '更新交易密碼'
        },
        metrics: {
          totalReturn: '總收益',
          annualReturn: '年化收益',
          maxDrawdown: '最大回撤',
          sharpe: '夏普比率',
          winRate: '勝率',
          totalTrades: '交易次數'
        },
        form: {
          account: '账戶',
          accountPlaceholder: '選擇账戶',
          scheduleName: '计划名稱',
          scheduleNamePlaceholder: '輸入计划名稱',
          scheduleNameMax: '最多64字元',
          scheduleType: '排程類型',
          scheduleTypes: {
            interval: '定時執行',
            hfQuote: '高頻報價',
            klineClose: 'K線收盤'
          },
          intervalMs: '間隔(毫秒)',
          intervalMsTip: '非高頻模式最小1000ms',
          hfCooldownMs: '高頻冷卻(毫秒)',
          hfCooldownMsTip: '报价驱动执行间的冷却時間',
          symbol: '商品',
          symbolPlaceholder: '選擇商品',
          symbolPlaceholderEmpty: '未配置商品',
          timeframe: '時間週期',
          defaultVolume: '默认手數',
          defaultVolumeTip: '每個信号的默认下單量',
          enableAfterCreate: '建立后立即啟用',
          riskSection: '風控設定',
          maxDrawdownPct: '最大回撤%',
          maxDrawdownPctTip: '回撤超過此閾值自动停止',
          maxPositions: '最大持仓數',
          maxPositionsTip: '同時持有的最大仓位元數量',
          stopLossOffset: '止損偏移',
          stopLossOffsetTip: '距入場價的止損距離(點)',
          takeProfitOffset: '止盈偏移',
          takeProfitOffsetTip: '距入場價的止盈距離(點)',
          strategyParamsSection: '策略参數',
          investorTag: '投資者(唯讀)'
        },
        noAccountTitle: '无账戶',
        noAccountBody: '啟動计划前需要先绑定MT账戶。',
        investorWarningTitle: '投資者账戶',
        investorWarningBody: '此账戶为投資者(唯讀)模式，需要交易權限才能啟動计划。',
        errorInvestorAccount: '无法使用投資者账戶啟動计划。请更新交易密碼以啟用交易。',
        verifyingPermission: '驗證交易權限中...',
        tradePermissionOk: '交易權限驗證通過',
        updatePasswordTitle: '更新交易密碼',
        updatePasswordHint: '輸入此账戶的交易密碼以啟用交易。',
        updatePasswordOk: '交易密碼已更新',
        updatePasswordFailed: '更新交易密碼失敗',
        updatePasswordStillInvestor: '密碼更新成功但账戶仍为投資者模式，請聯絡客服。',
        newPasswordPlaceholder: '輸入新交易密碼'
      },
      editTemplateModal: {
        title: {
          create: '新建模板',
          edit: '編輯模板'
        },
        fields: {
          name: '名稱',
          description: '描述',
          code: '策略代碼',
          publicShare: '公開分享'
        },
        placeholders: {
          name: '例如：均線交叉策略',
          description: '可選：策略說明',
          codeSample: `# 策略代碼示例
# 可用的變量: close, open, high, low, volume, symbol
# 返回: signal字典

import numpy as np

# 計算指標
maFast = np.mean(close[-10:])
maSlow = np.mean(close[-20:])

# 生成信號
if maFast > maSlow:
    signal = 'buy'
elif maFast < maSlow:
    signal = 'sell'
else:
    signal = 'hold'

# 返回結果
signal = {
    'signal': signal,
    'symbol': symbol,
    'price': close[-1],
    'confidence': 0.7,
    'reason': f'MA快線={maFast:.5f}, MA慢線={maSlow:.5f}'
}`
        },
        validation: {
          nameRequired: '請輸入名稱',
          codeRequired: '請輸入策略代碼'
        },
        actions: {
          validateCode: '驗證代碼'
        }
      },
      backtest: {
        title: '回測',
        fields: {
          title: '標題',
          account: '帳戶',
          symbol: '品種',
          timeframe: '週期',
          initialCapital: '初始資金',
          range: '回測區間',
          extraSymbols: '附加品種（多選）'
        },
        placeholders: {
          account: '選擇帳戶',
          symbol: '選擇品種',
          range: '請選擇回測區間',
          extraSymbols: '可選，適用於配對/輪動策略'
        },
        validation: {
          accountRequired: '請選擇帳戶',
          symbolRequired: '請選擇品種',
          timeframeRequired: '請選擇週期',
          initialCapitalRequired: '請輸入初始資金',
          rangeRequired: '請選擇回測區間'
        },
        quickRange: {
          '1d': '1天',
          '3d': '3天',
          '1w': '1週',
          '1y': '1年',
          custom: '自訂'
        },
        accountDisabledSuffix: ' (已停用)',
        modalTitleWithName: '回測：{{name}}',
        parameters: {
          title: '策略參數'
        },
        tooltips: {
          extraSymbols: '額外獲取K線的品種（同一帳戶、同一時間框架）。策略可透過context["closes_by_symbol"]存取。'
        }
      },
      messages: {
        fetchTemplateListFailed: '獲取模板列表失敗',
        deepLinkNavigate: '已為你定位模板與回測詳情',
        enterStrategyCode: '請輸入策略代碼',
        codeValidationPassed: '代碼驗證通過',
        codeValidationNotPassed: '代碼未通過驗證',
        codeValidationFailed: '代碼驗證失敗',
        templateUpdated: '模板已更新',
        templateCreated: '模板已建立',
        templateDeleted: '模板已刪除',
        readStrategyCodeFailed: '讀取策略代碼失敗',
        strategyCodeEmptyCannotBacktest: '策略代碼為空，無法回測',
        selectBacktestRange: '請選擇回測區間',
        backtestRangeInvalid: '回測區間無效',
        backtestSubmitted: '已提交回測',
        backtestSubmitFailed: '提交回測失敗',
        backtestCancelRequested: '已請求取消回測',
        backtestCancelFailed: '取消回測失敗',
        backtestReportDeleted: '回測報告已刪除',
        backtestReportNotFound: '回測報告不存在',
        codeCopied: '代碼已複製',
        copyFailed: '複製失敗',
        missingScheduleInfo: '缺少調度必要資訊',
        templateNotPublishedCannotCreateSchedule: '模板未發布，無法建立調度',
        readTemplateStatusFailed: '讀取模板狀態失敗',
        scheduleCreated: '調度已建立',
        scheduleCreatedAndEnabled: '調度已建立並啟用',
        createScheduleFailed: '建立調度失敗',
        templatePublished: '模板已發布',
        cannotPublishAndCreateDraftFailed: '無法發布。草稿建立失敗。',
        republishedButNoTemplateId: '已重新發布，但缺少模板ID。',
        backtestRunningCannotPublish: '回測正在執行。目前無法發布。',
        missingDraftIdCannotPublish: '缺少草稿ID。無法發布。',
        publishedButNoTemplateId: '已發布，但缺少模板ID。',
        templateRepublished: '模板已重新發布',
        templateAlreadyPublished: '模板已發布',
        templateNotDraftUnknownPublishStatus: '模板不是草稿。發布狀態未知。',
        publishFailed: '發布失敗',
        backtestRunNoPublishedTemplate: '回測執行沒有已發布的模板',
        strategyCodeEmptyCannotPublish: '策略代碼為空白，请先保存代碼再發布。',
        systemTemplateReadOnly: '系統範本為唯讀，请克隆后再編輯。'
      },
      backtestRuns: {
        title: '回測報告',
        empty: '暫無回測記錄',
        deleteConfirm: '確定要刪除這條回測報告嗎？',
        batchDeleteConfirm: '確定要刪除 {{count}} 條回測報告嗎？',
        batchDeleteSuccess: '已刪除 {{count}} 條回測報告',
        status: {
          queued: '排隊中',
          running: '運行中',
          completed: '已完成',
          failed: '失敗',
          canceling: '取消中',
          canceled: '已取消'
        },
        table: {
          title: '標題',
          status: '狀態',
          symbol: '品種',
          timeframe: '週期',
          createdAt: '建立時間',
          actions: '操作'
        },
        actions: {
          view: '查看',
          launchSchedule: '調度上線',
          createSchedule: '建立調度'
        }
      },
      deleteConfirm: '確定刪除此模板？',
      defaultDraftName: '草稿模板'
    },
    backtestRun: {
      title: '回測運行',
      status: {
        queued: '排隊中',
        running: '運行中',
        completed: '已完成',
        failed: '失敗',
        canceling: '取消中',
        canceled: '已取消',
        ended: '已結束'
      },
      actions: {
        cancel: '取消'
      },
      hints: {
        queued: '回測排隊中',
        running: '回測運行中',
        canceling: '正在取消回測'
      },
      fields: {
        status: '狀態',
        error: '錯誤',
        maxDrawdown: '最大回撤',
        sharpe: '夏普比率'
      },
      metrics: {
        totalReturn: '總收益',
        annualReturn: '年化收益',
        maxDrawdown: '最大回撤',
        sharpe: '夏普比率',
        winRate: '勝率',
        totalTrades: '交易次數',
        equityCurvePoints: '資金曲線點數'
      },
      trades: {
        title: '訂單明細',
        empty: '暫無成交記錄',
        loadFailed: '加載訂單明細失敗',
        ticket: '編號',
        side: '方向',
        sideBuy: '買入',
        sideSell: '賣出',
        volume: '手數',
        openTime: '開倉時間',
        openPrice: '開倉價',
        closeTime: '平倉時間',
        closePrice: '平倉價',
        pnl: '盈虧',
        commission: '手續費',
        reason: '平倉原因',
        reasons: {
          signal: '訊號',
          sl: '停損',
          tp: '停利',
          margin_call: '強平',
          expired: '到期',
          end_of_test: '回測結束'
        },
        summary: '共 {{count}} 筆，勝 {{wins}} / 負 {{losses}}，淨盈虧 {{pnl}}'
      }
    },
    defaultTemplates: {
      maCross: {
        name: '雙均線交叉策略',
        description: '當快均線上穿慢均線時買入，下穿時賣出'
      },
      forceBuy: {
        name: '測試下單（強制BUY）',
        description: '用於驗證下單鏈路：每次執行都返回 buy，並從 context/params 讀取 lot 作為 volume'
      },
      rsi: {
        name: 'RSI 超買超賣策略',
        description: 'RSI 低於30超賣區買入，高於70超買區賣出'
      },
      macd: {
        name: 'MACD 策略',
        description: 'MACD 金叉買入，死叉賣出'
      }
    },
    codeAssist: {
      tabAI: 'AI 修改',
      tabExplain: '程式碼解釋',
      explain: '解釋程式碼',
      requiredParamsTitle: '必填參數',
      requiredParamsDesc: '策略程式碼讀取了下列參數但沒有給出預設值，請在儲存前填寫。',
      optionalParamsTitle: '可選參數',
      optionalParamsDesc: '這些參數在程式碼裡已有預設值。留空表示使用預設值；填入新值僅作用於本次執行，不會修改已儲存的策略。',
      defaultLabel: '預設',
      paramDescriptions: {
        riskLevel: '風險等級，常用 low / medium / high，影響倉位與停損停利幅度。',
        takeProfit: '停利幅度（%），價格相對開倉價達到該百分比時平倉獲利。',
        stopLoss: '停損幅度（%），價格反向波動該百分比時強制平倉停損。',
        maxLoss: '單筆最大可承受虧損（占帳戶的比例，0.01 = 1%）。',
        confidence: '訊號置信度閾值（0~1），低於此值的訊號會被忽略。',
        threshold: '觸發訊號的閾值，實際含義依程式碼判斷條件而定。',
        lotSize: '下單手數 / 交易量，越大風險越高。',
        fastPeriod: '快線週期（K 線根數），常用於 MACD/雙均線，越小越敏感。',
        slowPeriod: '慢線週期（K 線根數），常用於 MACD/雙均線，越大越平滑。',
        signalPeriod: '訊號線週期（K 線根數），MACD DIF 與 DEA 的平滑週期。',
        rsiPeriod: 'RSI 計算週期（K 線根數），常用 14。',
        emaPeriod: 'EMA 指數移動平均的週期（K 線根數）。',
        smaPeriod: 'SMA 簡單移動平均的週期（K 線根數）。',
        genericPeriod: '回看週期（K 線根數），用於指標計算的視窗長度。',
        genericPercent: '百分比/比率類參數，單位通常為 %（如 1 表示 1%）。'
      },
      required: '必填',
      suggested: '建議',
      applyAllSuggestions: '一鍵填入建議值',
      fillRequiredParams: '請先填寫必填參數：{{keys}}',
      aiReviseTitle: 'AI 助手 — 修改程式碼',
      reviseInputPlaceholder: '例如：把 SMA(20) 換成 EMA(50)，並加 1% 停損。',
      reviseSend: '發送給 AI',
      enterInstruction: '請描述你想做的修改。',
      codeEmpty: '目前沒有可修改的程式碼。',
      codeUpdated: '程式碼已更新，請重新進行程式碼驗證後再儲存。',
      noPython: 'AI 沒有回傳 Python 程式碼區塊，請換個說法再試一次。',
      saveBlockedNotValidated: '請先點擊「驗證程式碼」，驗證通過後才能儲存。',
      generatePlaceholder: '描述你的策略需求...'
    },
    templateModal: {
      title: '另存為模板',
      fields: {
        name: '名稱',
        description: '說明'
      },
      placeholders: {
        name: '請輸入模板名稱',
        description: '請輸入說明'
      }
    },
    asset: {
      title: '策略資產',
      subtitle: '資產發布、審查狀態與複製由後端維護。複製結果為獨立的使用者模板。',
      submitAsset: '提交資產',
      assetList: '資產列表',
      name: '名稱',
      visibility: '可見性',
      reviewStatus: '審查狀態',
      cloneCount: '複製次數',
      version: '版本',
      description: '說明',
      actions: '操作',
      cloneAsDraft: '複製為草稿',
      sourceTemplate: '來源模板',
      assetName: '資產名稱',
      submit: '提交',
      messages: {
        loadFailed: '載入策略資產失敗',
        submitSuccess: '策略資產已提交',
        submitFailed: '提交策略資產失敗',
        cloneSuccess: '已複製為模板：{{templateId}}',
        cloneFailed: '複製策略資產失敗'
      },
      validation: {
        selectTemplate: '請選擇來源模板',
        enterName: '請輸入資產名稱'
      },
      empty: '暫無策略资产'
    },
    gen: {
      title: '策略生成',
      send: '生成策略',
      regenerate: '重新生成',
      reset: '重新開始',
      template: '模板',
      generating: '生成中...',
      validating: '合規檢查',
      backtestStarted: '回測已啟動',
      done: '完成',
      backtestMsg: '回測任務已建立',
      clarifyTitle: '需確認幾個細節：',
      useDefaults: '使用預設值繼續',
      placeholder: '描述您想建立的交易策略，例如：「為EURUSD建立1小時布林帶均值回歸策略」',
      chat: {
        generate: '⚡ 生成',
        revise: '✏️ 修改',
        repair: '🔧 修復',
        discuss: '💬 討論'
      },
      feedback: {
        heading: '📊 回測結果',
        placeholder: '提供反饋以進行迭代（例如「過於激進」、「加入止損」）'
      },
      metrics: {
        sharpe: '夏普',
        maxDrawdown: '最大回撤',
        winRate: '勝率',
        trades: '交易',
        return: '收益'
      }
    },
    marketRegime: {
      title: '市場狀態偵測',
      subtitle: '後端從K線計算趨勢、波動性和效率特徵。前端僅顯示結果。',
      ruleVersionAlert: '目前使用基於規則的偵測模型rule-v1。K線權威來源仍為後端Market/Kline服務。',
      detectSuccess: '市場狀態偵測完成',
      detectFailed: '市場狀態偵測失敗',
      form: {
        title: '偵測參數',
        accountId: '帳戶ID',
        accountIdRequired: '請輸入帳戶ID',
        accountIdPlaceholder: 'MT帳戶UUID',
        symbol: '品種',
        symbolRequired: '請輸入品種',
        symbolPlaceholder: 'EURUSD',
        timeframe: '時間框架',
        klineCount: 'K線數量',
        submit: '開始偵測'
      },
      result: {
        title: '偵測結果',
        status: '狀態',
        confidence: '信心度',
        modelVersion: '模型版本',
        strategyFamilies: '策略家族',
        features: '特徵',
        recordId: '記錄ID'
      }
    },
    experiment: {
      title: '策略實驗',
      subtitle: '參數實驗、候選評分與草稿生成由後端處理。前端僅提交和顯示。',
      ruleVersionAlert: '當前最小化迴圈：確定性參數實驗。候選僅生成草稿，不會自動發布、排程或交易。',
      jobEventStream: '任務事件流',
      noEvents: '暫無事件',
      selectJobToView: '請選擇一個有任務的實驗來檢視事件。',
      submitForm: {
        title: '提交實驗',
        baseTemplate: '基礎策略模板',
        baseTemplateRequired: '請選擇基礎策略模板',
        baseTemplatePlaceholder: '選擇模板',
        parameterSpace: '參數空間JSON',
        parameterSpaceRequired: '請輸入參數空間JSON',
        searchMethod: '搜尋方法',
        maxCandidates: '最大候選數',
        objective: '目標',
        submit: '提交實驗'
      },
      list: {
        title: '實驗列表',
        column: {
          status: '狀態',
          searchMethod: '搜尋方法',
          maxCandidates: '最大候選數',
          objective: '目標',
          actions: '操作',
          viewCandidates: '檢視候選'
        }
      },
      candidates: {
        title: '候選',
        titleWithId: '候選：{{id}}',
        column: {
          rank: '排名',
          grade: '評級',
          score: '分數',
          parameters: '參數',
          summary: '摘要',
          recommendation: '建議',
          actions: '操作',
          viewCandidates: '檢視候選',
          generateDraft: '生成草稿'
        }
      },
      messages: {
        loadTemplatesFailed: '載入策略模板失敗',
        loadExperimentsFailed: '載入實驗列表失敗',
        loadCandidatesFailed: '載入候選失敗',
        subscribeJobFailed: '訂閱實驗任務事件失敗',
        candidatesGenerated: '策略實驗候選已生成',
        submitFailed: '提交實驗失敗。請確認參數空間為有效的JSON格式。',
        draftGenerated: '草稿模板已生成：{{templateId}}',
        promoteFailed: '將候選提升為草稿失敗'
      }
    },
    workspace: {
      title: '策略工作區',
      account: '帳戶',
      accountPlaceholder: '帳戶ID',
      chartWindow: '圖表',
      hideCode: '隱藏程式碼',
      showCode: '顯示程式碼',
      quickTrade: '快速交易',
      quickTradeHint: '請先選擇品種',
      tradePanelPlaceholder: '交易面板——即將推出',
      selectSymbolHint: '請選擇交易帳戶和品種以檢視圖表',
      noAccounts: '暫無可用帳戶',
      selectSymbol: '品種',
      code: '策略程式碼',
      codePlaceholder: `# Python策略程式碼...
def run(context):
    return {"signal": "hold"}`,
      validate: '驗證',
      validatePass: '驗證通過',
      validateFailed: '驗證失敗',
      validateBeforeSave: '請在儲存前驗證程式碼',
      runBacktest: '執行回測',
      save: '儲存',
      copy: '複製',
      copySuccess: '已複製',
      copyFailed: '複製失敗',
      saveSuccess: '已儲存',
      chart: 'K線',
      backtest: '回測',
      backtestRunning: '回測執行中...',
      backtestCompleted: '已完成',
      backtestError: '回測失敗',
      backtestEmpty: '執行回測以檢視結果',
      backtestTab: '回測結果',
      tuningTab: '智慧調參',
      execAssumptions: 'ℹ 執行假設',
      execAssumptionsFields: {
        mode: '模式',
        timing: '時機',
        fillRule: '成交規則',
        direction: '方向',
        commission: '手續費',
        slippage: '滑點',
        leverage: '槓桿',
        mtfFallback: 'MTF備用'
      },
      aiAssist: 'AI助手',
      ai: 'AI',
      runtimeMode: '執行模式',
      saveFailed: '儲存失敗',
      autoFix: {
        fixing: '修復中...',
        button: '自動修復',
        askAI: '詢問AI',
        dismiss: '關閉',
        passed: '自動修復通過（{{iterations}} 次迭代）{{plural}}',
        failed: '自動修復：{{iterations}}次迭代後仍有{{remaining}}個問題',
        fixed: '已修復（{{count}}）',
        remaining: '剩餘（{{count}}）',
        newRegression: '新增回歸（{{count}}）',
        lineInfo: '第{{line}}行'
      },
      template: {
        title: '模板',
        selectPlaceholder: '選擇一個模板...',
        load: '載入',
        saveAs: '另存為新模板',
        loaded: '已載入'
      },
      watchlist: '觀察清單',
      selectAccount: '選擇帳戶',
      openPositions: '持倉 ({{count}})',
      noOpenPositions: '該帳戶無持倉',
      chartError: '圖表錯誤 — 請重新整理',
      smartTuning: '智能調校',
      quickTradeSection: {
        selectSymbol: '請先選擇品種',
        validVolume: '請輸入有效手數',
        priceRequired: '限價/停損單需要指定價格',
        orderPlaced: '{{side}} 訂單已提交',
        orderFailed: '訂單失敗',
        amountLots: '數量（手）',
        marginMode: '保證金模式',
        cross: '全倉',
        isolated: '逐倉',
        mt4CrossOnly: 'MT4 僅支援全倉模式'
      },
      chartTools: {
        streamActive: '即時K線串流運作中',
        streamUnavailable: '串流不可用',
        hide: '隱藏',
        show: '顯示',
        settings: '設定',
        remove: '移除',
        clearDrawings: '清除所有繪圖',
        candle: '蠟燭圖',
        ohlc: 'OHLC',
        area: '面積圖',
        live: '即時',
        error: '錯誤',
        static: '靜態'
      },
      backtestRunIdLabel: '選擇回測運行...',
      investorReadOnly: '投資者（唯讀）',
      masterTrading: '主帳戶（交易）',
      riskControls: '程式碼中的風控規則',
      jumpToCode: '跳轉到程式碼',
      runningStatus: '執行中...',
      completedStatus: '已完成',
      backtestResultsLabel: '回測結果',
      gateTab: 'Gate'
    },
    codeQuality: {
      category: {
        FUTURE_DATA_LEAK: '未來資料洩漏',
        MISSING_PARAM: '缺少參數',
        UNREAD_PARAM: '未讀取參數',
        NDARRAY_PANDAS_MISUSE: 'ndarray/pandas誤用',
        NO_STOP_AND_TAKE_PROFIT: '缺少止損/止盈',
        NO_ENTRY_PCT: '缺少進場比例%'
      }
    },
    backtestParams: {
      title: '回測',
      currentDraft: '📝 目前草稿',
      dateRange: '日期範圍',
      execution: '執行',
      capital: '資金',
      leverage: '槓桿',
      commission: '手續費',
      slippage: '滑點',
      trade: '交易',
      direction: '方向',
      long: '↑ 做多',
      short: '↓ 做空',
      both: '雙向',
      strictMode: '嚴格模式',
      strictModeOn: '開啟',
      strictModeOff: '關閉',
      strictModeOnDesc: '下一根K線開盤價。標準，保守。',
      strictModeOffDesc: '同一根K線收盤價 + MTF 1分鐘。更高精度。',
      strictModeOnTooltip: '開啟：訊號在K線收盤時確認，下一根K線開盤時執行',
      strictModeOffTooltip: '關閉：同一根K線收盤執行，搭配1分鐘子解析度',
      vectorizedMode: '向量化',
      eventDrivenMode: 'Run(context)',
      runtimeMode: '執行模式',
      history: '回測歷史',
      run: '▶ 執行',
      settingsSave: '儲存為我的預設',
      settingsLoad: '載入我的預設',
      settingsReset: '重設為出廠設定',
      defaultsSaved: '預設值已儲存',
      defaultsLoaded: '預設值已載入',
      defaultsReset: '已重設為出廠預設值',
      presets: {
        liveAligned: '與真實交易對齊',
        exploration: '探索模式'
      },
      enterCodeAndSymbol: '請輸入策略程式碼並選擇品種',
      backtestFailed: '回測失敗'
    },
    tuning: {
      optimizerMethod: '最佳化方法',
      parameterDimensions: '參數維度',
      enabledCombinations: '已啟用{{enabled}}個 · {{combos}}種組合',
      hide: '隱藏',
      preview: '預覽',
      previewTitle: '預覽（顯示{{shown}}個，共{{total}}個）',
      truncated: '已截斷',
      results: '結果（{{count}}）',
      rank: '#',
      grade: '評級',
      score: '分數',
      parameters: '參數',
      summary: '摘要',
      oosScore: '樣本外評分',
      degradation: '衰減',
      overfit: '過度擬合',
      overfitWarning: '⚠ 過度擬合',
      apply: '套用',
      run: '執行（{{count}}）',
      tuning: '調參中...',
      requiresAI: '需要配置AI供應商',
      switchToDE: '切換至差分進化',
      waiting: '等待實驗中...（SSE自動重新整理）',
      gridWarning: '網格搜尋將測試<b>{{count}}</b>種組合（預算：48）。建議切換至<b>差分進化</b>，後者可有效處理大型參數空間。',
      oosFootnote: '對前5名候選（按樣本內分數）執行樣本外驗證。綠色衰減<20%，橘色20-40%，紅色>40%。',
      optimizer: {
        grid: '網格搜尋',
        random: '隨機搜尋',
        de: '差分進化',
        tpe: 'TPE（KDE）',
        ags: '退火高斯',
        ai: 'AI最佳化器',
        gridDesc: '窮舉笛卡爾積。最適合≤3個參數。',
        randomDesc: '均勻隨機取樣。適合探索。',
        deDesc: 'rand/1/bin變異。在平滑地形上快速收斂。',
        tpeDesc: '樹狀結構Parzen估計器。KDE建模好/差分布。',
        agsDesc: '帶sigma退火的高斯抖動。TPE的輕量級替代方案。',
        aiDesc: 'LLM多輪提案。在3輪中從先前結果學習。'
      },
      started: '智慧調參已啟動'
    },
    paper: {
      title: '📊 模擬交易',
      createAccount: '建立模擬帳戶',
      accountName: '帳戶名稱',
      create: '建立',
      noAccounts: '暫無模擬帳戶。建立一個開始模擬交易。',
      running: '執行中 {{symbol}} {{timeframe}}',
      start: '啟動',
      stop: '停止',
      watch: '監控',
      paper: '模擬',
      startStrategy: '啟動模擬策略',
      symbol: '品種',
      timeframe: '週期',
      strategyCode: '策略程式碼 (Python)',
      messages: {
        enterName: '請輸入名稱',
        created: '模擬帳戶已建立',
        createFailed: '建立失敗',
        pasteCode: '貼上您的策略程式碼',
        strategyStarted: '模擬策略已啟動',
        startFailed: '啟動失敗',
        strategyStopped: '模擬策略已停止',
        stopFailed: '停止失敗'
      }
    },
    aiChat: {
      title: 'AI 對話',
      you: '你',
      ai: 'AI',
      revise: '修改',
      feedback: '🔄 回饋',
      streaming: '生成中',
      analyzing: '分析中',
      reset: '重置',
      applyCode: '套用程式碼',
      dismiss: '關閉',
      reviewCode: 'AI 已生成程式碼 — 請在套用前查看上方的對話。'
    },
    assetAnalysis: {
      title: 'AI 資產分析',
      subtitle: '多週期趨勢展望、支撐阻力位檢測、波動率分類及 AI 策略推薦',
      symbolPlaceholder: '輸入品種 (例如 EURUSD, XAUUSD, BTCUSD)',
      analyze: '分析',
      fetchingData: '正在取得市場資料...',
      phase: '階段: {{phase}}',
      mtfOutlook: '多週期展望',
      srLevels: '支撐 / 阻力位',
      volatility: '波動率',
      state: '狀態',
      atrPct: 'ATR %',
      aiRecommendation: 'AI 策略推薦',
      aiUnavailable: 'AI 推薦不可用。請在設定中配置 AI 提供商。',
      noLevels: '未檢測到顯著價位',
      noResults: '未返回分析結果。請嘗試其他品種。',
      volLow: '低波動率 — 可考慮突破或均值回歸策略，配合緊湊止損。',
      volNormal: '正常波動率 — 適合大多數策略類型。',
      volHigh: '高波動率 — 建議擴大止損；趨勢跟蹤和突破策略更有利。',
      volExtreme: '極端波動率 — 請大幅降低倉位；需要寬止損。'
    },
    ai: {
      checkSettings: '檢查AI設定',
      refreshFailed: '重新整理失敗',
      settings: 'AI設定'
    },
    backtest: {
      annualReturn: '年化報酬',
      equityCurve: '權益曲線',
      maxDrawdown: '最大回撤',
      sharpe: '夏普比率',
      totalReturn: '總收益',
      totalTrades: '總交易',
      winRate: '勝率',
      tradeLog: '交易日誌',
      tradeTime: '時間',
      tradeSide: '方向',
      tradePrice: '價格',
      tradeVolume: '數量'
    },
    chartTools: {
      clearDrawings: '清除所有繪圖',
      hide: '隱藏',
      show: '顯示',
      settings: '設定',
      remove: '移除'
    },
    quickTradeSection: {
      amountLots: '數量(手)',
      marginMode: '保證金模式',
      cross: '跨式',
      isolated: '逐倉',
      mt4CrossOnly: 'MT4 僅支援跨式保证金',
      selectSymbol: '请選擇交易商品',
      validVolume: '交易量需 ≥ 0.01 手',
      priceRequired: '请輸入價格',
      orderPlaced: '下單成功',
      orderFailed: '下單失敗'
    }
  },
  indicatorCatalog: {
    title: '指標目錄',
    description: '策略沙箱中可用的技術指標和風險參數。請在策略代碼中僅使用這些輔助函數和參數鍵。',
    indicatorsTitle: '技術指標',
    riskSectionTitle: '風險管理參數',
    riskParamsTitle: '通用風險參數',
    riskParamsDesc: '無論選擇哪些指標，每個策略都應遵循這些風險管理參數。',
    paramKey: '鍵',
    paramLabel: '標籤',
    paramType: '類型',
    paramDefault: '預設值',
    paramRange: '範圍',
    paramDescription: '描述',
    workspace: {
      title: '策略工作台',
      account: '賬戶',
      accountPlaceholder: '賬戶 ID',
      chartWindow: '圖表',
      hideCode: '隱藏程式碼',
      showCode: '顯示程式碼',
      quickTrade: '快速交易',
      quickTradeHint: '請先選擇品種',
      tradePanelPlaceholder: '交易面板 — 即將推出',
      selectSymbolHint: '選擇交易賬戶和品種以檢視圖表',
      noAccounts: '暫無可用賬戶',
      selectSymbol: '品種',
      code: '策略程式碼',
      codePlaceholder: `# Python 策略程式碼...
def run(context):
    return {"signal": "hold"}`,
      validate: '驗證',
      validatePass: '驗證通過',
      validateFailed: '驗證失敗',
      validateBeforeSave: '請先驗證程式碼再儲存',
      runBacktest: '執行回測',
      save: '儲存',
      copy: '複製',
      copySuccess: '已複製',
      copyFailed: '複製失敗',
      saveSuccess: '已儲存',
      chart: 'K線',
      backtest: '回測',
      backtestRunning: '回測執行中...',
      backtestCompleted: '已完成',
      backtestError: '回測失敗',
      backtestEmpty: '執行回測查看結果',
      aiAssist: 'AI 助手',
      ai: 'AI',
      template: {
        title: '模板',
        selectPlaceholder: '選擇一個模板...',
        load: '載入',
        saveAs: '另存為',
        loaded: '已載入'
      }
    }
  },

    library: {
      title: '策略庫',
      myStrategies: '我的策略',
      create: '新建',
      filterAll: '全部',
      filterMine: '我的',
      filterSystem: '預置',
      searchPlaceholder: '搜尋策略...',
      empty: '暫無策略',
      published: '已發佈',
      draft: '草稿',
      unpublish: '下架',
      unpublishShort: '下架',
      publish: '發佈到市場',
      publishSuccess: '已發佈',
      unpublishSuccess: '已下架',
      publishStatus: '市場狀態',
      selectHint: '選擇左側策略查看詳情',
      overview: '概覽',
      schedules: '執行',
      backtestHistory: '回測歷史',
      scheduleCount: '{{count}} 個執行中',
      scheduleRunningCount: '{{count}} 個執行中',
      noSchedules: '未執行',
      openInWorkspace: '在 Workspace 中開啟',
      createSchedule: '建立執行',
      codePreview: '程式碼預覽',
      viewCode: '檢視策略程式碼',
    }
} as const;

export default strategy;
