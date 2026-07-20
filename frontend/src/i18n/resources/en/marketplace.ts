const Marketplace = {
  "marketplace": {
    "title": "Marketplace",
    "subtitle": "Discover and subscribe to trading strategies",
    "tabs": { "marketplace": "Market", "purchases": "My Purchases", "author": "Author Center" },
    "searchPlaceholder": "Search strategies...",
    "empty": "No strategies found",
    "filter": { "all": "All", "free": "Free", "paid": "Paid" },
    "sort": {
      "score": "Score", "newest": "Newest", "popular": "Popular",
      "rating": "Rating", "priceAsc": "Price: Low to High", "priceDesc": "Price: High to Low"
    },
    "card": {
      "free": "Free", "owned": "Owned", "winRate": "Win Rate",
      "pnl": "Total PnL", "users": "Subscribers", "rent": "¥{{amount}}/mo", "buy": "¥{{amount}}"
    },
    "messages": {
      "loginFirst": "Please login first", "subscribed": "Subscribed successfully",
      "subscribeFailed": "Subscribe failed", "published": "Strategy published to marketplace",
      "publishFailed": "Failed to publish strategy", "rated": "Rated!"
    },
    "detail": {
      "author": "Author", "price": "Price", "assetClass": "Asset Class",
      "riskLevel": "Risk Level", "subscribers": "Subscribers", "avgRating": "Rating",
      "description": "Description", "tags": "Tags", "yourRating": "Your Rating",
      "comments": "Comments", "noComments": "No comments yet",
      "commentPlaceholder": "Write a comment...", "getFree": "Get Free", "buyNow": "Buy Now",
      "owned": "Owned", "freePrice": "Free", "rentPrice": "¥{{amount}} / month",
      "buyPrice": "¥{{amount}} one-time", "runBacktest": "Run Backtest"
    },
    "purchases": {
      "empty": "No purchases yet", "strategy": "Strategy", "date": "Purchased",
      "status": "Status", "actions": "Actions", "runBacktest": "Run Backtest"
    },
    "author": {
      "empty": "No published strategies yet", "noPublished": "No published strategies",
      "published": "Published", "subscribers": "Subscribers", "avgRating": "Avg Rating",
      "myStrategies": "My Published Strategies", "publishNew": "Publish New Strategy",
      "goToLibrary": "Go to Strategy Library"
    },
    "payment": {
      "purchaseSuccess": "Purchase successful! Strategy added to your library.",
      "purchaseFailed": "Purchase failed. Please try again.",
      "insufficientBalance": "Insufficient balance",
      "alreadyPurchased": "You already own this strategy.",
      "title": "Confirm Purchase", "balance": "Wallet Balance",
      "confirm": "Confirm Payment", "cancel": "Cancel"
    },
    "publish": {
      "title": "Publish to Marketplace",
      "titleLabel": "Title",
      "titlePlaceholder": "e.g. Golden Cross Strategy",
      "descriptionLabel": "Description",
      "descriptionPlaceholder": "Describe your strategy logic, entry/exit rules...",
      "assetClass": {
        "label": "Asset Class",
        "forex": "Forex",
        "crypto": "Crypto",
        "commodity": "Commodity",
        "index": "Index",
        "stock": "Stock"
      },
      "riskLevel": {
        "label": "Risk Level",
        "low": "Low",
        "medium": "Medium",
        "high": "High"
      },
      "priceModel": {
        "label": "Pricing",
        "free": "Free",
        "monthly": "Monthly Subscription",
        "once": "One-Time Purchase"
      },
      "priceAmount": "Amount",
      "tags": "Tags",
      "tagsPlaceholder": "Type and press enter to add tags",
      "codeSnippet": "Strategy Preview (public)",
      "codeSnippetPlaceholder": "Optional: share a snippet or high-level idea of your strategy (visible to all)",
      "includeBacktestSnapshot": "Include latest backtest results"
    },
    "backtest": {
      "title": "Strategy Backtest",
      "protected": "Strategy code is protected. Backtest runs on our servers.",
      "run": "Run Backtest",
      "idle": "Set parameters and run a backtest"
    },
    "autogen": {
      "title": "AI Strategy Generation",
      "subtitle": "Describe your strategy in natural language — AI will generate, compile, backtest, and publish it.",
      "description": "Describe your strategy",
      "placeholder": "e.g. Trend following on EURUSD H1 using EMA crossover, 50 pip stop loss, 100 pip take profit...",
      "assetClass": "Asset Class",
      "symbol": "Symbol",
      "timeframe": "Timeframe",
      "risk": "Risk Level",
      "type": "Strategy Type",
      "start": "Start Generation",
      "cancel": "Cancel",
      "autoPublishOn": "Auto-publish: ON",
      "autoPublishOff": "Auto-publish: OFF",
      "needDescription": "Please describe your strategy",
      "failedAt": "Failed at",
      "retry": "Retry",
      "modify": "Modify Request",
      "qualityFailed": "Strategy generated but did not pass quality gates",
      "success": "Strategy generated and published successfully!",
      "actual": "Actual",
      "threshold": "Threshold",
      "viewDetail": "View Strategy",
      "generateAnother": "Generate Another",
      "stages": {
        "generating": "Generating",
        "compiling": "Compiling",
        "backtesting": "Backtesting",
        "evaluating": "Quality Check",
        "publishing": "Publishing",
        "completed": "Completed"
      }
    }
  }
};
export default Marketplace;
