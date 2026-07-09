// Auto-generated from proto/ant/v1/i18n/base_en.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Base = {
  "admin": {
    "dashboard": {
      "logs": {
        "moduleMap": {
          "accountManagement": "Account Management",
          "systemConfig": "System Config",
          "trading": "Trading",
          "userManagement": "User Management"
        },
        "actionType": "Action Type",
        "failed": "Failed",
        "module": "Module",
        "status": "Status",
        "success": "Success",
        "target": "Target",
        "time": "Time"
      },
      "riskMetrics": {
        "orderCloseFailed": "Order Closed Failed",
        "orderCloseSuccess": "Order Closed Success",
        "orderSendFailed": "Order Sent Failed",
        "orderSendSuccess": "Order Sent Success",
        "riskValidateError": "Risk Validated Error",
        "riskValidatePass": "Risk Validated Pass",
        "riskValidateReject": "Risk Validated Reject",
        "riskValidateTotal": "Risk Validated Total",
        "title": "Risk Control Metrics (Real-time)"
      },
      "riskWindow": {
        "noData": "No window metrics data",
        "noRejectData": "No reject data for current window",
        "orderCloseFailed": "{{window}} Close Failed",
        "orderCloseSuccess": "{{window}} Close Success",
        "orderSendFailed": "{{window}} Order Failed",
        "orderSendSuccess": "{{window}} Order Sent",
        "rejectCount": "Reject Count",
        "rejectRiskCodesHeader": "Top N Reject Risk Codes ({{window}})",
        "title": "Risk Control Window Metrics (1h / 24h / 72h)",
        "validateError": "{{window}} Error",
        "validatePass": "{{window}} Pass",
        "validateReject": "{{window}} Reject",
        "validateTotal": "{{window}} Validated Total"
      },
      "activeUsers": "Active Users",
      "loadFailed": "Failed to load dashboard data",
      "mtAccounts": "MT Accounts",
      "onlineAccounts": "Online Accounts",
      "recentLogs": "Recent Operation Logs",
      "title": "Admin Dashboard",
      "todayProfit": "Today P&L",
      "todayTrades": "Today Trades",
      "totalUsers": "Total Users"
    },
    "userManagement": {
      "drawer": {
        "labels": {
          "createdAt": "Created At",
          "email": "Email",
          "id": "ID",
          "lastLogin": "Last Login",
          "mtAccountCount": "MT Accounts",
          "nickname": "Nickname",
          "role": "Role",
          "status": "Status"
        },
        "title": "User Details"
      },
      "form": {
        "placeholders": {
          "email": "Enter email",
          "nickname": "Enter nickname",
          "password": "Enter password"
        },
        "accountNumber": "Account Number",
        "accountNumberInvalid": "5-6 digits, no leading 0, no 4 or 7",
        "email": "Email",
        "nickname": "Nickname",
        "password": "Password",
        "role": "Role",
        "status": "Status"
      },
      "passwordForm": {
        "placeholders": {
          "confirmPassword": "Re-enter new password",
          "newPassword": "Enter new password"
        },
        "validation": {
          "confirmPasswordRequired": "Please confirm the new password",
          "newPasswordRequired": "New password is required",
          "passwordMin8": "Password must be at least 8 characters",
          "passwordMismatch": "Passwords do not match",
          "passwordMustContainLettersAndNumbers": "Password must contain both letters and numbers"
        },
        "confirmPassword": "Confirm Password",
        "newPassword": "New Password",
        "submit": "Update Password"
      },
      "actions": {
        "changePassword": "Change Password",
        "details": "Details",
        "disable": "Disable",
        "enable": "Enable"
      },
      "deleteConfirm": {
        "batchDeleteConfirm": "Delete {{count}} user(s)? This action cannot be undone.",
        "batchDeletePartial": "{{deleted}} deleted, {{failed}} failed",
        "batchDeleteSuccess": "{{count}} user(s) deleted",
        "title": "Delete this user? This action cannot be undone."
      },
      "filters": {
        "rolePlaceholder": "Filter by role",
        "searchPlaceholder": "Search by email or nickname",
        "statusPlaceholder": "Filter by status"
      },
      "messages": {
        "newPasswordIs": "New password is: {{password}}",
        "passwordUpdateFailed": "Failed to update password",
        "passwordUpdatedSuccess": "Password updated successfully",
        "userCreateFailed": "Failed to create user",
        "userCreatedSuccess": "User created successfully",
        "userDeleteFailed": "Failed to delete user",
        "userDeletedSuccess": "User deleted successfully",
        "userDisabled": "User disabled",
        "userEnabled": "User enabled",
        "userUpdateFailed": "Failed to update user",
        "userUpdatedSuccess": "User updated successfully"
      },
      "modals": {
        "createTitle": "Create User",
        "editTitle": "Edit User",
        "passwordTitle": "Change Password"
      },
      "pagination": {
        "total": "Total {{total}} users"
      },
      "roles": {
        "audit": "Audit",
        "customerService": "Customer Service",
        "operation": "Operation",
        "superAdmin": "Super Admin",
        "user": "User"
      },
      "status": {
        "active": "Active",
        "suspended": "Suspended"
      },
      "table": {
        "actions": "Actions",
        "createdAt": "Created At",
        "email": "Email",
        "id": "ID",
        "mtAccountCount": "MT Accounts",
        "nickname": "Nickname",
        "role": "Role",
        "status": "Status"
      },
      "addUser": "Add User",
      "title": "User Management"
    },
    "config": {
      "messages": {
        "disabled": "Config disabled",
        "enabled": "Config enabled",
        "loadFailed": "Failed to load configs",
        "operationFailed": "Operation failed",
        "updateFailed": "Update failed",
        "updated": "Config updated"
      },
      "placeholders": {
        "apiKey": "Enter API Key",
        "baseUrl": "e.g. https://api.openai.com or self-hosted gateway",
        "configValue": "Enter config value",
        "description": "Enter description",
        "json": "Enter JSON",
        "model": "e.g. glm-4-flash / deepseek-chat / gpt-4o-mini"
      },
      "providerOptions": {
        "custom": "Custom / OpenAI Compatible",
        "deepseek": "DeepSeek",
        "zhipu": "Zhipu"
      },
      "validation": {
        "apiKeyRequired": "API Key cannot be empty",
        "greenMaxFailedRunsNonNegative": "green_max_failed_runs must be >= 0",
        "greenSuccessRateRange": "green_success_rate must be between 0 and 100",
        "jsonEmpty": "JSON cannot be empty",
        "jsonInvalid": "Invalid JSON format",
        "minSampleSizeNonNegative": "min_sample_size must be >= 0",
        "modelRequired": "Model name cannot be empty",
        "yellowNotGreaterThanGreen": "yellow_success_rate cannot be greater than green_success_rate",
        "yellowSuccessRateRange": "yellow_success_rate must be between 0 and 100"
      },
      "aiProviderCatalog": "AI Model Provider Catalog",
      "baseUrlLabel": "Base URL (optional, custom/OpenAI compatible only)",
      "configItem": "Config Item",
      "description": "Description",
      "econAIConfig": "Economic Calendar Translation AI Config",
      "editConfig": "Edit Config: {{key}}",
      "enableToggle": "Enable",
      "fillTemplate": "Fill Example",
      "formatJson": "Format JSON",
      "maxAccountsPerUser": "Max Accounts Per User",
      "modelName": "Model Name",
      "off": "Off",
      "on": "On",
      "provider": "Provider",
      "status": "Status",
      "strategyHealthConfig": "Strategy Health Grading Config",
      "thresholdDesc": "green_success_rate: green success rate threshold; green_max_failed_runs: max failed runs for green; yellow_success_rate: yellow success rate threshold; min_sample_size: minimum sample size.",
      "thresholdInfo": "Threshold Field Description",
      "title": "System Configuration",
      "toggle": "Toggle",
      "updatedAt": "Updated At",
      "value": "Value"
    },
    "jurisdiction": {
      "messages": {
        "countryAddFailed": "Failed to add sanctioned country",
        "countryAdded": "Sanctioned country added",
        "countryRemoveFailed": "Failed to remove sanctioned country",
        "countryRemoved": "Sanctioned country removed",
        "kycUpdateFailed": "Failed to update KYC status",
        "kycUpdated": "KYC status updated",
        "overrideUpdateFailed": "Failed to update sanctioned override",
        "overrideUpdated": "Sanctioned override updated"
      },
      "actions": "Actions",
      "addCountry": "Add Country",
      "addSanctionedCountry": "Add Sanctioned Country",
      "addedBy": "Added By",
      "confirmGrantOverride": "Grant override access to this user?",
      "confirmRevokeOverride": "Revoke override access from this user?",
      "country": "Country",
      "countryCode": "Country Code",
      "countryLabel": "Label",
      "disclaimer": "Disclaimer",
      "emptyKYC": "No users match the selected KYC filter",
      "emptySanctions": "No sanctioned countries configured",
      "filterByKYCStatus": "Filter by KYC status",
      "grantOverride": "Grant Override",
      "kycStatus": "KYC Status",
      "kycStatusTab": "User KYC Status",
      "override": "Override",
      "overrideWarning": "This user is from a sanctioned country. Granting override will allow trading.",
      "pending": "Pending",
      "questionnaire": "Questionnaire",
      "rejected": "Rejected",
      "revokeOverride": "Revoke Override",
      "sanctioned": "Sanctioned",
      "sanctionedCountries": "Sanctioned Countries",
      "sanctionedCountriesTab": "Sanctioned Countries",
      "setKYC": "Set KYC",
      "setKYCStatus": "Set KYC Status",
      "title": "Jurisdiction Gate",
      "unverified": "Unverified",
      "userEmail": "Email",
      "userKYCStatus": "User KYC Status",
      "verified": "Verified"
    },
    "header": {
      "admin": "Admin",
      "adminMode": "Admin Mode",
      "adminPanel": "Admin Panel",
      "backToUser": "Back to User",
      "logout": "Logout"
    },
    "sidebar": {
      "accountManagement": "Account Management",
      "dashboard": "Dashboard",
      "jurisdiction": "Jurisdiction Gate",
      "operationLogs": "Operation Logs",
      "shareManagement": "Share Analytics",
      "systemConfig": "System Config",
      "tradingMonitor": "Trading Monitor",
      "userManagement": "User Management",
      "walletManagement": "Wallets"
    },
    "trading": {
      "accounts": "Accounts",
      "activeUsers": "Active Users",
      "byPlatform": "By Platform",
      "closedOrders": "Closed Orders",
      "connectedAccounts": "Connected Accounts",
      "loadFailed": "Failed to load trading statistics",
      "netProfit": "Net P&L",
      "orders": "Orders",
      "pendingOrders": "Pending Orders",
      "platform": "Platform",
      "profitStats": "P&L Statistics",
      "title": "Trading Monitor",
      "totalAccounts": "Total Accounts",
      "totalLoss": "Total Loss",
      "totalOrders": "Total Orders",
      "totalProfit": "Total Profit",
      "totalUsers": "Total Users",
      "totalVolume": "Total Volume",
      "volume": "Volume"
    },
    "wallet": {
      "accountNumber": "Account",
      "add": "Add",
      "adjustBalance": "Adjust Balance",
      "adjustFailed": "Adjustment failed",
      "adjustSuccess": "Balance adjusted",
      "deduct": "Deduct",
      "noUsers": "No users found",
      "reason": "Reason for adjustment...",
      "searchPlaceholder": "Search by email or account number...",
      "title": "Wallet Management",
      "walletFor": "Wallet for"
    }
  },
  "autoTrading": {
    "logs": {
      "columns": {
        "action": "Action",
        "price": "Price",
        "profit": "P&L",
        "symbol": "Symbol",
        "ticket": "Ticket",
        "time": "Time",
        "volume": "Volume"
      },
      "empty": "No trading logs yet",
      "title": "Recent Trading Logs"
    },
    "messages": {
      "loadFailed": "Failed to load auto trading data",
      "toggleFailed": "Failed to toggle auto trading"
    },
    "settings": {
      "maxDailyLoss": "Max Daily Loss",
      "maxDailyLossHint": "Auto-disable trading if daily loss exceeds this",
      "maxDrawdownPercent": "Max Drawdown %",
      "maxDrawdownPercentHint": "Auto-disable trading if drawdown exceeds this",
      "maxLotSize": "Max Lot Size",
      "maxLotSizeHint": "Maximum volume per trade (lots)",
      "maxPositions": "Max Positions",
      "maxPositionsHint": "Maximum concurrent open positions",
      "maxRiskPercent": "Max Risk %",
      "maxRiskPercentHint": "Percentage of balance to risk per trade",
      "saveFailed": "Failed to save settings",
      "saveSuccess": "Settings saved",
      "title": "Global Risk Settings"
    },
    "status": {
      "activeStrategies": "Active Strategies",
      "disabled": "Auto Trading Disabled",
      "enabled": "Auto Trading Enabled",
      "todayExecutions": "Today's Executions",
      "todayProfit": "Today's Profit"
    },
    "title": "Auto Trading"
  },
  "notifications": {
    "stream": {
      "autoTrading": {
        "fallback": "Auto trading event triggered",
        "title": "Auto Trading"
      },
      "riskAlert": {
        "fallback": "Alert type: {{alertType}}",
        "title": "Risk Alert"
      },
      "strategyExecution": {
        "completed": "{{symbol}} {{action}} completed",
        "failed": "Execution failed: {{error}}",
        "title": "Strategy Execution"
      },
      "strategySignal": {
        "message": "{{symbol}} triggered {{signalType}}",
        "title": "Strategy Signal"
      }
    },
    "actions": {
      "clearAll": "Clear all",
      "clearAllConfirm": "Clear all notifications?",
      "markAllAsRead": "Mark all as read"
    },
    "tabs": {
      "all": "All ({{count}})",
      "unread": "Unread ({{count}})"
    },
    "types": {
      "risk_alert": "Risk Alert",
      "signal": "Signal",
      "strategy_execution": "Execution",
      "system": "System",
      "trade": "Trade"
    },
    "all": "All",
    "clearAll": "Clear all",
    "confirmClearAll": "Clear all notifications?",
    "empty": "No notifications",
    "markAllRead": "Mark all as read",
    "title": "Notifications",
    "unread": "Unread"
  },
  "auth": {
    "fields": {
      "confirmPassword": "Confirm password",
      "email": "Email",
      "password": "Password"
    },
    "forgotPassword": {
      "backToLogin": "Back to Login",
      "hint": "Please contact your administrator or support to reset your password.",
      "title": "Reset Password"
    },
    "login": {
      "forgotPassword": "Forgot password?",
      "login": "Sign in",
      "noAccount": "Don't have an account?",
      "registerNow": "Register now",
      "rememberMe": "Remember me",
      "signingIn": "Signing in...",
      "subtitle": "Sign in to continue"
    },
    "messages": {
      "fetchMeFailed": "Failed to load user profile",
      "loginFailed": "Sign-in failed. Please check your email and password.",
      "loginSuccess": "Signed in",
      "logoutSuccess": "Signed out",
      "registerFailed": "Registration failed. Please try again later.",
      "registerSuccess": "Registered successfully. Please sign in."
    },
    "register": {
      "haveAccount": "Already have an account?",
      "loginNow": "Sign in",
      "register": "Register",
      "signingUp": "Signing up...",
      "subtitle": "Create an account to get started"
    },
    "validation": {
      "confirmPasswordRequired": "Please confirm your password",
      "emailInvalid": "Invalid email address",
      "emailRequired": "Email is required",
      "passwordMin8": "Password must be at least 8 characters",
      "passwordMismatch": "Passwords do not match",
      "passwordRequired": "Password is required"
    }
  },
  "common": {
    "months": {
      "jan": "Jan",
      "jul": "Jul"
    },
    "time": {
      "day": "{{n}}d",
      "hour": "{{n}}h",
      "lessThanMinute": "<1m",
      "minute": "{{n}}m"
    },
    "active": "Active",
    "back": "Back",
    "cancel": "Cancel",
    "clear": "Clear",
    "close": "Close",
    "comingSoon": "Coming Soon",
    "confirm": "Confirm",
    "copied": "Copied",
    "copy": "Copy",
    "copyFailed": "Copy failed",
    "create": "Create",
    "created": "Created",
    "currentPosition": "📊 Current Position",
    "delete": "Delete",
    "deleteFailed": "Delete failed",
    "deleteSelected": "Delete selected ({{count}})",
    "deleted": "Deleted",
    "disable": "Disable",
    "disabled": "Disabled",
    "edit": "Edit",
    "enable": "Enable",
    "enabled": "Enabled",
    "error": "Error",
    "gotIt": "Got it",
    "hideDetails": "Hide details",
    "inactive": "Inactive",
    "indicatorSettings": "{{name}} Settings",
    "lineColor": "Line Color",
    "loading": "Loading...",
    "loadingFailed": "Loading failed",
    "next": "Next",
    "no": "No",
    "noData": "No data",
    "noOpenPositionsForSymbol": "No open positions for {{symbol}}",
    "none": "None",
    "ok": "OK",
    "operationFailed": "Operation failed",
    "pageError": "Page Error",
    "pageUnderDevelopment": "This page is under development",
    "pleaseWait": "Please wait...",
    "previous": "Previous",
    "refresh": "Refresh",
    "remove": "Remove",
    "required": "Required",
    "retry": "Retry",
    "save": "Save",
    "saveFailed": "Save failed",
    "saveSuccess": "Saved successfully",
    "searching": "Searching...",
    "selectSymbolToViewChart": "Select a symbol to view chart",
    "send": "Send",
    "showDetails": "Show details",
    "totalItems": "Total {{count}} items",
    "translate": "Translate",
    "unexpectedError": "An unexpected error occurred",
    "unknown": "Unknown",
    "updated": "Updated",
    "viewOriginal": "View original",
    "viewTranslation": "View translation",
    "yes": "Yes",
    "you": "You",
    "unsaved": "Unsaved",
    "saved": "Saved"
  },
  "errors": {
    "ai": {
      "api_key_required": "API Key is required",
      "base_url_required": "Base URL is required",
      "base_url_scheme_invalid": "Base URL must start with http:// or https://",
      "base_url_should_not_end_with_chat_completions": "Base URL should not end with /chat/completions",
      "config_service_not_initialized": "AI config service has not been initialized",
      "config_valid": "AI config is valid",
      "failed_to_create_request": "Failed to create request",
      "forbidden_quota": "Quota exceeded",
      "free_tier_exhausted": "Free tier exhausted",
      "invalid_base_url": "Invalid Base URL",
      "invalid_provider": "Invalid provider",
      "no_trade_data_available": "No trade data available",
      "not_configured": "AI is not configured. Please enable and configure it in AI Settings first.",
      "probe_ok": "OK",
      "probe_ok_no_models": "OK (no models returned)",
      "provider_required": "Please select a provider first",
      "provider_returned_empty_message": "AI provider returned an empty response",
      "rate_limited": "Rate limited. Please try again later.",
      "request_failed": "API request failed"
    },
    "connection_failed": {
      "content": "Unable to connect to the server. Please check your network and try again.",
      "title": "Connection failed"
    },
    "access_denied": "Access denied",
    "account_connected": "Connected to trading server",
    "account_connection_failed": "Could not connect to the trading server",
    "account_not_found": "Account not found",
    "auto_trading_disabled": "Auto trading disabled",
    "auto_trading_enabled": "Auto trading enabled",
    "email_already_registered": "This email is already registered",
    "invalid_credentials": "Invalid email or password",
    "not_authenticated": "Not signed in",
    "schedule_service_not_available": "Schedule service is unavailable",
    "translate_failed": "Translation failed",
    "user_not_found": "User not found"
  },
  "marketplace": {
    "author": {
      "avgRating": "Avg Rating",
      "empty": "No strategies published yet. Go to Strategy Library to publish one.",
      "published": "Published"
    },
    "card": {
      "by": "by",
      "free": "Free",
      "owned": "Purchased",
      "subscribers": "Subscribers",
      "winRate": "Win Rate"
    },
    "detail": {
      "assetClass": "Asset Class",
      "author": "Author",
      "commentPlaceholder": "Write a comment...",
      "comments": "Comments",
      "description": "Description",
      "getFree": "Get Free",
      "rentPrice": "¥{{amount}} / month",
      "subscribers": "Subscribers",
      "yourRating": "Your Rating"
    },
    "messages": {
      "commentFailed": "Comment failed",
      "commentPosted": "Comment posted",
      "loginFirst": "Please log in first",
      "paymentComingSoon": "Payment coming soon",
      "rateFailed": "Rating failed",
      "rated": "Rating submitted",
      "subscribeFailed": "Failed",
      "subscribed": "Added to your purchases"
    },
    "payment": {
      "alreadyPurchased": "You already own this strategy.",
      "balanceAfter": "Balance after purchase",
      "cancel": "Cancel",
      "confirm": "Confirm Purchase",
      "depositPrompt": "Please deposit funds to continue.",
      "goToDeposit": "Deposit",
      "insufficientBalance": "Insufficient balance",
      "oneTimePurchase": "¥{{amount}} one-time",
      "price": "Price",
      "purchaseFailed": "Purchase failed. Please try again.",
      "purchaseSuccess": "Purchase successful! Strategy added to your library.",
      "purchasing": "Processing...",
      "strategyName": "Strategy",
      "title": "Confirm Purchase",
      "walletBalance": "Your Balance"
    },
    "purchases": {
      "empty": "No purchases yet. Browse the market to find strategies.",
      "status": "Status",
      "strategy": "Strategy"
    },
    "sort": {
      "newest": "Newest",
      "performance": "Best Performance",
      "popular": "Most Popular",
      "priceAsc": "Price: Low to High",
      "priceDesc": "Price: High to Low",
      "rating": "Highest Rated",
      "score": "Composite Score"
    },
    "tabs": {
      "author": "Author Center",
      "marketplace": "Market",
      "purchases": "My Purchases",
      "subscriptions": "My Subscriptions"
    },
    "empty": "No strategies published yet",
    "filterByClass": "Filter by asset class",
    "noSubscriptions": "No subscriptions yet",
    "publish": "Publish Strategy",
    "searchPlaceholder": "Search strategies...",
    "subtitle": "Discover, buy, and use community strategies",
    "title": "Strategy Marketplace"
  },
  "symbolDetection": {
    "tradeMode": {
      "disabled": "Disabled",
      "longOnly": "Long Only",
      "longShort": "Long & Short",
      "shortOnly": "Short Only",
      "unknown": "Unknown"
    },
    "label": "Detected Symbols",
    "loading": "Parsing…",
    "noSymbols": "No trading symbols detected. Try including specific symbol names (e.g. \"Bitcoin\", \"EURUSD\", \"Gold\").",
    "resolvedTooltip": "broker: {{broker}} | mode: {{mode}}",
    "unresolvedTooltip": "No trading account bound yet, unable to resolve"
  },
  "wallet": {
    "table": {
      "amount": "Amount",
      "balanceAfter": "Balance After",
      "description": "Description",
      "time": "Time",
      "type": "Type"
    },
    "txType": {
      "adjustment": "Adjustment",
      "deposit": "Deposit",
      "fee": "Fee",
      "reversal": "Reversal",
      "withdrawal": "Withdrawal"
    },
    "accountNumber": "Account",
    "balance": "Balance",
    "currency": "Currency",
    "deposit": "Deposit",
    "frozen": "Frozen",
    "frozenBalance": "Frozen",
    "history": "History",
    "title": "My Wallet",
    "transactions": "Transactions",
    "withdraw": "Withdraw"
  },
  "app": {
    "name": "AntTrader"
  },
  "language": {
    "english": "English",
    "japanese": "日本語",
    "simplifiedChinese": "简体中文",
    "traditionalChinese": "繁體中文",
    "vietnamese": "Tiếng Việt"
  },
  "market": {
    "allSymbols": "All Symbols",
    "ask": "Ask",
    "bid": "Bid",
    "common": "Common",
    "emptyWatchlist": "No symbols in watchlist",
    "loadingSymbols": "Loading...",
    "mid": "Mid",
    "noSymbolSelected": "Select a symbol to view market data",
    "noSymbolsFound": "No symbols found",
    "popularSymbols": "Popular Symbols",
    "searchPlaceholder": "Search symbol (e.g. EURUSD, XAUUSD)",
    "searchSymbol": "Search symbol...",
    "selectAccount": "Select trading account",
    "selectSymbol": "Select symbol",
    "spread": "Spread",
    "watchlist": "Watchlist"
  },
  "menu": {
    "accounts": "Accounts",
    "aiAssistant": "AI Assistant",
    "algoDashboard": "Algo Dashboard",
    "analytics": "Analytics",
    "assetAnalysis": "AI Analysis",
    "assets": "Assets",
    "autoTrading": "Auto Trading",
    "dashboard": "Dashboard",
    "devGroup": "Development",
    "experiments": "Experiments",
    "indicatorCatalog": "Indicator Catalog",
    "logs": "System Logs",
    "market": "Market",
    "marketRegime": "Market Regime",
    "marketTools": "Market Tools",
    "marketplace": "Marketplace",
    "opsGroup": "Operations",
    "schedules": "Schedules",
    "strategies": "Strategies",
    "strategy": "Strategy",
    "strategyLibrary": "Strategy Library",
    "strategyWorkspace": "Strategy Workspace",
    "trading": "Trading",
    "wallet": "Wallet"
  },
  "profile": {
    "lastLogin": "Last Login",
    "nickname": "Nickname",
    "registered": "Registered",
    "role": "Role",
    "status": "Status",
    "title": "Profile"
  },
  "share": {
    "actions": "Actions",
    "createNew": "Create New Share Link",
    "createdAt": "Created",
    "deleteConfirm": "Delete this share link?",
    "empty": "No share links yet",
    "expires": "Expires",
    "positions": "Positions",
    "showPositions": "Show positions on new link",
    "title": "Share Management",
    "token": "Share Link",
    "userId": "User",
    "views": "Views"
  },
  "sharePage": {
    "avgHolding": "Avg Holding",
    "avgLoss": "Avg Loss",
    "avgWin": "Avg Win",
    "bestTrade": "Best Trade",
    "bySymbol": "Performance by Symbol",
    "closeTime": "Close",
    "count": "Trades",
    "disclaimer": "Past performance is not indicative of future results.",
    "equityCurve": "Equity Curve",
    "expired": "This share link has expired",
    "footer": "Generated by AntTrader",
    "language": "Language",
    "loadFailed": "Failed to load shared performance",
    "losingTrades": "Losing Trades",
    "maxDrawdown": "Max Drawdown",
    "netProfit": "Net Profit",
    "noPositions": "No open positions",
    "noTrades": "No trade records yet",
    "notFound": "Not found",
    "openPrice": "Open",
    "positions": "Open Positions",
    "positionsLocked": "Positions hidden by creator",
    "profit": "Profit",
    "profitFactor": "Profit Factor",
    "sharpeRatio": "Sharpe Ratio",
    "side": "Side",
    "subtitle": "Verified trading results",
    "symbol": "Symbol",
    "title": "Trading Performance",
    "totalReturn": "Net Profit",
    "totalTrades": "Total Trades",
    "totalVolume": "Total Volume",
    "tradeRecords": "Trade Records",
    "volume": "Volume",
    "winRate": "Win Rate",
    "winningTrades": "Winning Trades",
    "worstTrade": "Worst Trade"
  },
  "topbar": {
    "logout": "Logout",
    "profile": "Profile",
    "settings": "Settings",
    "switchToAdmin": "Switch to Admin",
    "systemOk": "System running normally",
    "user": "User"
  }
} as const;
export default Base;
