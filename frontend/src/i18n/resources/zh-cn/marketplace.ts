const Marketplace = {
  "marketplace": {
    "title": "策略市场",
    "subtitle": "发现和订阅交易策略",
    "tabs": { "marketplace": "市场", "purchases": "我的购买", "author": "作者中心" },
    "searchPlaceholder": "搜索策略...",
    "empty": "暂无策略",
    "filter": { "all": "全部", "free": "免费", "paid": "付费" },
    "sort": { "score": "评分", "newest": "最新", "popular": "最热", "rating": "评价", "priceAsc": "价格从低到高", "priceDesc": "价格从高到低" },
    "card": { "free": "免费", "owned": "已拥有", "winRate": "胜率", "pnl": "总盈亏", "users": "订阅者", "rent": "¥{{amount}}/月", "buy": "¥{{amount}}" },
    "messages": { "loginFirst": "请先登录", "subscribed": "订阅成功", "subscribeFailed": "订阅失败", "published": "策略已发布到市场", "publishFailed": "发布失败", "rated": "已评分！" },
    "detail": { "author": "作者", "price": "价格", "assetClass": "资产类别", "riskLevel": "风险等级", "subscribers": "订阅者", "avgRating": "评分", "description": "描述", "tags": "标签", "yourRating": "你的评分", "comments": "评论", "noComments": "暂无评论", "commentPlaceholder": "写评论...", "getFree": "免费获取", "buyNow": "立即购买", "owned": "已拥有", "freePrice": "免费", "rentPrice": "¥{{amount}} / 月", "buyPrice": "¥{{amount}} 买断", "runBacktest": "运行回测" },
    "purchases": { "empty": "暂无购买", "strategy": "策略", "date": "购买时间", "status": "状态", "actions": "操作", "runBacktest": "运行回测" },
    "author": { "empty": "暂无已发布策略", "noPublished": "暂无已发布策略", "published": "已发布", "subscribers": "订阅者", "avgRating": "平均评分", "myStrategies": "我的已发布策略", "publishNew": "发布新策略", "goToLibrary": "前往策略库" },
    "payment": { "purchaseSuccess": "购买成功！策略已添加到你的库中。", "purchaseFailed": "购买失败，请重试。", "insufficientBalance": "余额不足", "alreadyPurchased": "你已经拥有这个策略。", "title": "确认购买", "balance": "钱包余额", "confirm": "确认支付", "cancel": "取消" },
    "publish": {
      "title": "发布到市场",
      "titleLabel": "标题",
      "titlePlaceholder": "如：金叉策略",
      "descriptionLabel": "描述",
      "descriptionPlaceholder": "描述你的策略逻辑、入场/出场规则...",
      "assetClass": { "label": "资产类别", "forex": "外汇", "crypto": "加密货币", "commodity": "大宗商品", "index": "指数", "stock": "股票" },
      "riskLevel": { "label": "风险等级", "low": "低", "medium": "中", "high": "高" },
      "priceModel": { "label": "定价", "free": "免费", "subscription": "按月订阅", "once": "一次性购买" },
      "priceAmount": "金额",
      "tags": "标签",
      "tagsPlaceholder": "输入并按回车添加标签",
      "codeSnippet": "策略预览（公开）",
      "codeSnippetPlaceholder": "可选：分享策略的高层思路或代码片段（所有人可见）",
      "includeBacktestSnapshot": "附带最新回测成绩"
    },
    "backtest": { "title": "策略回测", "protected": "策略代码受保护，回测在服务器端执行。", "run": "运行回测", "idle": "设置参数并运行回测" },
    "autogen": {
      "title": "AI 策略生成",
      "subtitle": "用自然语言描述策略需求 — AI 自动生成、编译、回测并上架。",
      "modes": {
        "freeform": "自由描述",
        "template": "模板"
      },
      "templates": {
        "title": "策略模板",
        "subtitle": "选择模板快速生成策略，参数可自定义。",
        "empty": "暂无可用模板。",
        "loadError": "加载模板失败",
        "backToList": "返回模板列表",
        "parameters": "参数",
        "generateFromTemplate": "从模板生成"
      },
      "description": "描述你的策略",
      "placeholder": "如：EURUSD H1 趋势跟踪，EMA 交叉信号，止损 50 点，止盈 100 点...",
      "assetClass": "资产类别",
      "symbol": "品种",
      "timeframe": "时间周期",
      "risk": "风险等级",
      "type": "策略类型",
      "start": "开始生成",
      "cancel": "取消",
      "autoPublishOn": "自动上架：开",
      "autoPublishOff": "自动上架：关",
      "needDescription": "请描述你的策略",
      "failedAt": "失败于",
      "retry": "重试",
      "modify": "修改需求",
      "qualityFailed": "策略已生成但未通过质量门槛",
      "success": "策略生成并上架成功！",
      "actual": "实际值",
      "threshold": "阈值",
      "viewDetail": "查看策略",
      "generateAnother": "再生成一个",
      "stages": {
        "generating": "生成中",
        "compiling": "编译中",
        "backtesting": "回测中",
        "evaluating": "质量评估",
        "publishing": "上架中",
        "completed": "已完成"
      }
    }
  }
};
export default Marketplace;
