// Auto-generated supplementary keys for admin
// TODO: Translate to vi
const AdminExtra = {
  "admin": {
    "aiGateway": {
      "errors": {
        "loadProviders": "Failed to load providers",
        "toggleFailed": "Toggle failed",
        "loadModels": "Failed to load models"
      },
      "addProviderPending": "Add provider feature pending backend support",
      "title": "AI Gateway Management",
      "description": "Manage AI providers, models, and pricing. Users select from available models, billed by token from wallet.",
      "addProvider": "Add Provider",
      "columns": {
        "baseUrl": "Base URL",
        "apiKey": "API Key"
      },
      "configured": "Not configured",
      "editProvider": "Add Provider",
      "providerId": "Please enter provider ID",
      "displayName": "Display Name",
      "baseUrl": "Please enter Base URL",
      "apiKeyLabel": "API key, encrypted at rest",
      "apiKeyEditPlaceholder": "Leave empty to keep",
      "editModel": "Add Model",
      "modelName": "Model Name",
      "priceInput": "Input Price ($/1M)",
      "priceOutput": "Output Price ($/1M)",
      "addModel": "Add Model",
      "confirmDeleteModel": "Delete this model?",
      "noModels": "No models"
    },
    "account": {
      "errors": {
        "loadFailed": "Failed to load accounts",
        "freezeFailed": "Freeze failed",
        "unfreezeFailed": "Unfreeze failed"
      },
      "frozen": "Account frozen",
      "unfrozen": "Account unfrozen",
      "columns": {
        "createdAt": "Created At"
      },
      "confirmFreeze": "Freeze this account?",
      "title": "Account Management",
      "searchPlaceholder": "Search accounts",
      "detail": "Account Detail",
      "auditLogs": "Audit Logs"
    },
    "settings": {
      "saveSuccess": "Saved successfully",
      "saveFailed": "Save failed",
      "deleteFailed": "Delete failed",
      "actionFailed": "Action failed",
      "columns": {
        "key": "Setting Key"
      },
      "confirmDelete": "Confirm delete?",
      "title": "Agent Management Settings",
      "addSetting": "Add Setting",
      "permissionRules": "Permission Rules (permission.rule.N)",
      "permissionFormat": "Format:",
      "permissionExample": "Example:",
      "permissionAddRule": "Add rule: create setting with key",
      "addManagedSetting": "Add Managed Setting",
      "settingKey": "Setting Key",
      "keyPlaceholder": "e.g.: allowed_models, disable_live_trading, permission.rule.1",
      "valuePlaceholder": "e.g.: claude-sonnet-5,deepseek-v4"
    },
    "autogen": {
      "approved": "Task approved and published",
      "rejected": "Task rejected",
      "enqueued": "{{count}} tasks enqueued",
      "confirmApprove": "Approve and publish?",
      "confirmReject": "Reject this task?",
      "title": "AI Strategy Generation Tasks",
      "allStatus": "All Status",
      "triggerBatch": "Trigger Batch Generation",
      "symbols": "Symbols (comma-separated)",
      "timeframes": "Timeframes (comma-separated)",
      "strategyTypes": "Strategy Types (comma-separated)"
    },
    "billing": {
      "columns": {
        "autoRenew": "Auto Renew",
        "periodStart": "Period Start",
        "periodEnd": "Period End",
        "createdAt": "Created At",
        "balanceBefore": "Balance Before",
        "balanceAfter": "Balance After"
      },
      "title": "Billing Management",
      "monthlyRevenue": "Monthly Revenue",
      "totalRevenue": "Total Revenue",
      "activeSubs": "Active Subscriptions",
      "planRevenue": "Plan Revenue Details",
      "filterByPlan": "Filter by plan",
      "filterByStatus": "Filter by status",
      "walletTransactions": "Wallet Transactions",
      "filterByType": "Filter by type",
      "txPlatformFee": "Platform Fee"
    },
    "coupon": {
      "loadFailed": "Failed to load coupons",
      "fillRequired": "Please fill required fields",
      "created": "Coupon created",
      "createFailed": "Failed to create coupon",
      "disabled": "Coupon disabled",
      "disableFailed": "Failed to disable coupon",
      "colMinPurchase": "Min Purchase",
      "create": "Create Coupon",
      "createTitle": "Create Coupon",
      "codePlaceholder": "Coupon code (e.g. SUMMER20)",
      "valuePlaceholder": "Discount value (e.g. 20 for 20% or 50 for ¥50)",
      "minPurchasePlaceholder": "Minimum purchase amount (0 = none)",
      "maxUsesPlaceholder": "Max uses (0 = unlimited)",
      "expiresPlaceholder": "Expires at (ISO 8601, empty = never)"
    },
    "dashboard": {
      "errors": {
        "loadFailed": "Failed to load dashboard data"
      },
      "title": "Admin Dashboard",
      "totalUsers": "Total Users",
      "activeUsers": "Active Users",
      "verifiedUsers": "Verified Users",
      "mtAccounts": "MT Accounts",
      "onlineAccounts": "Online Accounts",
      "todayTrades": "Today Trades",
      "todayProfit": "Today P&L",
      "activeSubs": "Active Subs",
      "monthlyRevenue": "Monthly Revenue",
      "totalRevenue": "Total Revenue",
      "marketStrategies": "Market Strategies",
      "marketSales": "Market Sales",
      "marketRevenue": "Market Revenue",
      "recentLogs": "Recent Logs"
    },
    "logs": {
      "modules": {
        "userManagement": "User Management",
        "accountManagement": "Account Management",
        "systemConfig": "System Config"
      },
      "columns": {
        "actionType": "Action Type",
        "ip": "IP Address"
      },
      "errors": {
        "loadFailed": "Failed to load logs"
      },
      "title": "Operation Logs",
      "filterModule": "Filter by module",
      "filterAction": "Filter by action"
    },
    "depositAddresses": {
      "importFailed": "Import failed",
      "user": "User ID",
      "received": "Received USDT",
      "assignedAt": "Assigned At",
      "importHint": "Use hdgen tool on an offline machine to generate deposit_addresses.bin, then upload it here.",
      "all": "All Status",
      "import": "Import Addresses",
      "availablePool": "Available in Pool",
      "total": "Total Addresses"
    },
    "deposit": {
      "table": {
        "user": "User ID",
        "amount": "USDT Amount",
        "txHash": "Tx Hash"
      },
      "title": "Deposit Management"
    },
    "analytics": {
      "platformRev": "Platform Rev",
      "providerRev": "Provider Rev",
      "activeBuyers": "Active Buyers",
      "refundRate": "Refund Rate",
      "newSubs": "New Subscribers",
      "totalStrategies": "Total Strategies",
      "newStrategies": "New Strategies",
      "topByRevenue": "Top Strategies by Revenue",
      "topBySubs": "Top Strategies by Subscribers",
      "topProvidersRev": "Top Providers by Revenue",
      "topProvidersStrat": "Top Providers by Strategies"
    },
    "marketplace": {
      "loadFailed": "Failed to load strategies",
      "featureSuccess": "Strategy featured",
      "featureFailed": "Failed to feature strategy",
      "unfeatureSuccess": "Removed featured",
      "unfeatureFailed": "Failed to unfeature",
      "unfeature": "Remove featured",
      "filterStatus": "All statuses",
      "searchPlaceholder": "Search by title...",
      "featureTitle": "Feature Strategy",
      "featureDesc": "Set priority for featured placement. Higher = more prominent."
    },
    "refund": {
      "loadFailed": "Failed to load refund requests",
      "approved": "Refund approved and executed",
      "rejected": "Refund request rejected",
      "processFailed": "Failed to process refund",
      "approve": "Approve & Execute",
      "filterStatus": "All statuses",
      "approveTitle": "Approve Refund",
      "rejectTitle": "Reject Refund",
      "reviewNotePlaceholder": "Review note (optional for reject, recommended for approve)..."
    },
    "sidebar": {
      "shareManagement": "Share Analytics"
    },
    "walletCalculator": {
      "title": "Token ↔ USD Calculator",
      "selectModel": "Select model (pricing basis)",
      "usdAmount": "USD Amount",
      "fillResult": "Fill Result"
    },
    "wallet": {
      "errors": {
        "noUserSelected": "No user selected"
      },
      "messages": {
        "adjustSuccess": "Balance adjusted successfully",
        "adjustFailed": "Adjustment failed"
      },
      "columns": {
        "walletNumber": "Wallet No.",
        "balanceAfter": "Balance After"
      },
      "title": "Wallet Management",
      "tabWallets": "User Wallets",
      "userList": "User List",
      "searchPlaceholder": "Search wallet / email / nickname",
      "noMatch": "No users",
      "walletDetail": "Wallet Detail",
      "adjustBalance": "Adjust Balance",
      "tabDepositAddresses": "Deposit Addresses"
    },
    "config": {
      "apiKey": "API Key"
    },
    "userManagement": {
      "form": {
        "accountNumber": "Account Number",
        "accountNumberInvalid": "5-6 digits, no leading 0, no 4 or 7"
      },
      "messages": {
        "loadUsersFailed": "Failed to load users"
      }
    }
  }
} as const;
export default AdminExtra;
