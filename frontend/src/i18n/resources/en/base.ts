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
      "errors": {
        "loadFailed": "Failed to load dashboard data"
      },
      "activeUsers": "Active Users",
      "loadFailed": "Failed to load dashboard data",
      "mtAccounts": "MT Accounts",
      "onlineAccounts": "Online Accounts",
      "recentLogs": "Recent Operation Logs",
      "title": "Admin Dashboard",
      "todayProfit": "Today P&L",
      "todayTrades": "Today Trades",
      "totalUsers": "Total Users",
      "verifiedUsers": "Verified Users",
      "activeSubs": "Active Subs",
      "monthlyRevenue": "Monthly Revenue",
      "totalRevenue": "Total Revenue",
      "marketStrategies": "Market Strategies",
      "marketSales": "Market Sales",
      "marketRevenue": "Market Revenue",
      "validateTotal": "Validate Total",
      "validatePass": "Validate Pass",
      "validateReject": "Validate Reject",
      "validateError": "Validate Error",
      "orderSendSuccess": "Order Send Success",
      "orderSendFailed": "Order Send Failed",
      "orderCloseSuccess": "Order Close Success",
      "orderCloseFailed": "Order Close Failed",
      "rejectCount": "Reject Count"
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
        "userUpdatedSuccess": "User updated successfully",
        "loadUsersFailed": "Failed to load users"
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
      "value": "Value",
      "apiKey": "API Key"
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
    "aiGateway": {
      "errors": {
        "loadProviders": "Failed to load providers",
        "toggleFailed": "Toggle failed",
        "loadModels": "Failed to load models"
      },
      "columns": {
        "baseUrl": "Base URL",
        "apiKey": "API Key"
      },
      "addProviderPending": "Add provider feature pending backend support",
      "title": "AI Gateway Management",
      "description": "Manage AI providers, models, and pricing. Users select from available models, billed by token from wallet.",
      "addProvider": "Add Provider",
      "provider": "Provider",
      "configured": "Configured",
      "notConfigured": "Not configured",
      "models": "Models",
      "editProvider": "Edit Provider",
      "providerId": "Provider ID",
      "providerIdRequired": "Please enter provider ID",
      "displayName": "Display Name",
      "displayNameRequired": "Please enter display name",
      "baseUrl": "Base URL",
      "baseUrlRequired": "Please enter Base URL",
      "apiKeyLabel": "API Key",
      "apiKeyEditHint": "Leave empty to keep existing key",
      "apiKeyHint": "API key, encrypted at rest",
      "apiKeyEditPlaceholder": "Leave empty to keep",
      "editModel": "Edit Model",
      "addModel": "Add Model",
      "modelName": "Model Name",
      "modelNameRequired": "Please enter model name",
      "priceInput": "Input Price ($/1M)",
      "priceOutput": "Output Price ($/1M)",
      "confirmDeleteModel": "Delete this model?",
      "noModels": "No models"
    },
    "account": {
      "errors": {
        "loadFailed": "Failed to load accounts",
        "freezeFailed": "Freeze failed",
        "unfreezeFailed": "Unfreeze failed"
      },
      "columns": {
        "id": "ID",
        "user": "User",
        "login": "Login",
        "type": "Type",
        "broker": "Broker",
        "status": "Status",
        "balance": "Balance",
        "createdAt": "Created At",
        "action": "Action",
        "server": "Server",
        "equity": "Equity",
        "margin": "Margin",
        "time": "Time",
        "detail": "Detail"
      },
      "frozen": "Account frozen",
      "unfrozen": "Account unfrozen",
      "detail": "Detail",
      "unfreeze": "Unfreeze",
      "confirmFreeze": "Freeze this account?",
      "freeze": "Freeze",
      "title": "Account Management",
      "searchPlaceholder": "Search accounts",
      "status": "Status",
      "online": "Online",
      "offline": "Offline",
      "auditLogs": "Audit Logs"
    },
    "settings": {
      "columns": {
        "key": "Setting Key",
        "value": "Value",
        "action": "Action"
      },
      "saveSuccess": "Saved successfully",
      "saveFailed": "Save failed",
      "deleted": "Deleted",
      "deleteFailed": "Delete failed",
      "actionFailed": "Action failed",
      "confirmDelete": "Confirm delete?",
      "title": "Agent Management Settings",
      "addSetting": "Add Setting",
      "permissionRules": "Permission Rules (permission.rule.N)",
      "permissionFormat": "Format: ",
      "permissionExample": "Example: ",
      "permissionAddRule": "Add rule: create setting with key ",
      "addManagedSetting": "Add Managed Setting",
      "settingKey": "Setting Key",
      "keyPlaceholder": "e.g.: allowed_models, disable_live_trading, permission.rule.1",
      "valuePlaceholder": "e.g.: claude-sonnet-5,deepseek-v4"
    },
    "billing": {
      "columns": {
        "user": "User",
        "plan": "Plan",
        "status": "Status",
        "cycle": "Cycle",
        "price": "Price",
        "autoRenew": "Auto Renew",
        "periodStart": "Period Start",
        "periodEnd": "Period End",
        "createdAt": "Created At",
        "type": "Type",
        "amount": "Amount",
        "balanceBefore": "Balance Before",
        "balanceAfter": "Balance After",
        "description": "Description",
        "time": "Time"
      },
      "title": "Billing Management",
      "monthlyRevenue": "Monthly Revenue",
      "totalRevenue": "Total Revenue",
      "activeSubs": "Active Subscriptions",
      "txRecords": "Transactions",
      "planRevenue": "Plan Revenue Details",
      "activeCount": "Active",
      "subscriptions": "Subscriptions",
      "filterByPlan": "Filter by plan",
      "planFree": "Free",
      "planPro": "Pro",
      "planEnterprise": "Enterprise",
      "filterByStatus": "Filter by status",
      "statusActive": "Active",
      "statusCancelled": "Cancelled",
      "statusExpired": "Expired",
      "walletTransactions": "Wallet Transactions",
      "filterByType": "Filter by type",
      "txPurchase": "Purchase",
      "txSale": "Sale",
      "txPlatformFee": "Platform Fee",
      "txDeposit": "Deposit",
      "txWithdrawal": "Withdrawal"
    },
    "logs": {
      "columns": {
        "time": "Time",
        "module": "Module",
        "actionType": "Action Type",
        "target": "Target",
        "status": "Status",
        "ip": "IP Address",
        "action": "Action",
        "details": "Details"
      },
      "modules": {
        "userManagement": "User Management",
        "accountManagement": "Account Management",
        "trading": "Trading",
        "systemConfig": "System Config"
      },
      "errors": {
        "loadFailed": "Failed to load logs"
      },
      "actions": {
        "create": "Create",
        "update": "Update",
        "delete": "Delete",
        "disable": "Disable",
        "enable": "Enable",
        "freeze": "Freeze",
        "unfreeze": "Unfreeze"
      },
      "title": "Operation Logs",
      "filterModule": "Filter by module",
      "filterAction": "Filter by action"
    },
    "deposit": {
      "table": {
        "user": "User",
        "amount": "USDT Amount",
        "amountUsd": "USD Credit",
        "txHash": "Tx Hash",
        "status": "Status",
        "reviewNote": "Review Note",
        "time": "Time",
        "action": "Action"
      },
      "approved": "Deposit approved and wallet credited.",
      "approveFailed": "Failed to approve deposit.",
      "rejected": "Deposit rejected.",
      "rejectFailed": "Failed to reject deposit.",
      "approve": "Approve",
      "reject": "Reject",
      "title": "Deposit Management",
      "allStatuses": "All Statuses",
      "statusPending": "Pending",
      "statusApproved": "Approved",
      "statusRejected": "Rejected",
      "approveTitle": "Approve Deposit",
      "rejectTitle": "Reject Deposit",
      "reviewNoteLabel": "Review Note (optional)",
      "reviewNotePlaceholder": "Add a note for this review...",
      "approveWarning": "Approving will credit the user wallet immediately."
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
        "email": "Email",
        "nickname": "Nickname",
        "type": "Type",
        "amount": "Amount",
        "balanceAfter": "Balance After",
        "description": "Description",
        "time": "Time",
        "balance": "Balance",
        "frozen": "Frozen",
        "currency": "Currency"
      },
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
      "walletFor": "Wallet for",
      "unassigned": "Unassigned",
      "userList": "User List",
      "noMatch": "No matching users",
      "walletDetail": "Wallet Detail",
      "transactions": "Transactions",
      "adjustReason": "Reason"
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
      "agentSettings": "Agent Settings",
      "aiGateway": "AI Gateway",
      "billing": "Billing",
      "dashboard": "Dashboard",
      "deposits": "Deposits",
      "jurisdiction": "Jurisdiction Gate",
      "monitoring": "Monitoring & Alerts",
      "operationLogs": "Operation Logs",
      "shareManagement": "Share Analytics",
      "sre": "SRE Controls",
      "strategies": "Strategies",
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
    "walletCalculator": {
      "title": "Token ↔ USD Calculator",
      "selectModel": "Select model (pricing basis)",
      "usdAmount": "USD Amount",
      "tokenAmount": "Token Amount",
      "fillResult": "Fill Result"
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
  "wallet": {
    "deposit": {
      "table": {
        "amount": "USDT Amount",
        "amountUsd": "USD Credit",
        "status": "Status",
        "time": "Time",
        "txHash": "Tx Hash"
      },
      "address": "Receiving Address",
      "addressCopied": "Address copied to clipboard",
      "amountLabel": "USDT Amount",
      "button": "New Deposit",
      "copy": "Copy",
      "exchangeRate": "Exchange Rate",
      "failed": "Failed to submit deposit request.",
      "history": "Deposit History",
      "modalTitle": "Submit Deposit Request",
      "network": "Network",
      "notConfigured": "USDT deposit is not yet configured. Please contact support.",
      "notice": "Only send USDT via the specified network. Sending other tokens or using a different network may result in permanent loss. After sending, submit a deposit request with the amount and optional tx hash for admin review.",
      "submit": "Submit",
      "success": "Deposit request submitted. Please wait for admin review.",
      "title": "Deposit",
      "txHashLabel": "Transaction Hash (optional)",
      "willCredit": "Will credit"
    },
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
    "frozen": "Frozen",
    "frozenBalance": "Frozen",
    "history": "History",
    "title": "My Wallet",
    "transactions": "Transactions",
    "withdraw": "Withdraw"
  },
  "strategy": {
    "workspace": {
      "chartIndicators": {
        "overlay": "Overlay (main chart)",
        "subPane": "Sub-pane indicators"
      }
    },
    "tuning": {
      "searchMethod": {
        "grid": "Grid",
        "random": "Random"
      }
    },
    "backtest": {
      "canceled": "Backtest canceled",
      "lotSize": "Lot Size",
      "strategyParameters": "Strategy Parameters"
    },
    "chat": {
      "executionPlan": "Execution Plan",
      "codeGenerated": "Code generated. Use the buttons below to run strategy review and backtest."
    },
    "aiChat": {
      "historyTab": "History",
      "strategiesTab": "Strategies"
    },
    "templates": {
      "title": "Strategy Templates",
      "saveCurrent": "Save Current Strategy",
      "lines": "lines",
      "chatEdit": "Chat Edit",
      "source": "Source",
      "rename": "Rename",
      "confirmDelete": "Delete this strategy?",
      "noTemplates": "No saved strategy templates",
      "sourceCode": "Strategy Source",
      "copyAll": "Copy All"
    },
    "live": {
      "stopSuccess": "Strategy stopped",
      "stopFailed": "Failed to stop",
      "runId": "Run ID",
      "account": "Account",
      "symbol": "Symbol",
      "timeframe": "TF",
      "mode": "Mode",
      "signals": "Signals",
      "errors": "Errors",
      "startedAt": "Started",
      "watchSignals": "Watch Signals",
      "confirmStop": "Stop this strategy?",
      "status": "Status",
      "totalSignals": "Total Signals",
      "stoppedAt": "Stopped",
      "error": "Error",
      "title": "Live Strategy Monitor",
      "activeTab": "Active Runs",
      "noActive": "No active strategies",
      "historyTab": "Run History",
      "noRuns": "No strategy runs",
      "schedulesTab": "Schedules",
      "time": "Time",
      "signalType": "Type",
      "volume": "Volume",
      "price": "Price",
      "sl": "SL",
      "tp": "TP",
      "reason": "Reason",
      "signalLog": "Signal Log",
      "waitingSignals": "Waiting for signals..."
    },
    "schedule": {
      "maxPositionsPlaceholder": "Unlimited"
    },
    "ai": {
      "reviseHint": "Write code first, then ask AI to improve it.",
      "explainHint": "Write code to see AI explanation.",
      "settingsHint": "Configure AI provider and model"
    },
    "validate": {
      "running": "Running validation...",
      "errors": "Errors",
      "warnings": "Warnings",
      "fixWithAI": "Send errors to AI Revise",
      "parameters": "parameters",
      "hints": "Suggestions",
      "allClear": "All checks passed — no issues found.",
      "passed": "Validation passed — Save is now unlocked."
    },
    "importEA": {
      "writeTab": "Strategy Code",
      "importTab": "Import EA",
      "codeTooShort": "Please paste complete EA/indicator source code.",
      "pastePlaceholder": "Paste MQL4/MQL5 EA code...",
      "migration": "策略导入",
      "aiTranslate": "AI 翻译",
      "bridge": "盲区桥接",
      "analyze": "分析策略结构",
      "confirmImport": "确认导入",
      "tryAI": "AI 翻译补充",
      "apply": "Apply to Editor",
      "importSuccess": "MQL 源码已导入，点击「Apply to Editor」写入编辑器",
      "hint": "Paste MQL4/MQL5 code and click Analyze",
      "translate": "Translate to Go",
      "translating": "AI translating...",
      "bridgeBtn": "盲区桥接翻译",
      "bridgeSuccess": "桥接成功",
      "bridgeFailedTag": "桥接失败",
      "bridging": "AI bridging blind spots...",
      "bridgeFailedMsg": "Agent 无法自动桥接所有盲区",
      "noBridgeNeeded": "覆盖率 100%，无需桥接",
      "bridgeHint": "粘贴 MQL4/MQL5 EA 代码，AI 将自动翻译盲区为 Python 子集"
    },
    "version": {
      "loadFailed": "Failed to load versions",
      "rollbackFailed": "Rollback failed",
      "loadVersionFailed": "Failed to load version",
      "loadDiffFailed": "Failed to load diff",
      "colVersion": "Version",
      "colSummary": "Change Summary",
      "colLang": "Lang",
      "colHash": "Hash",
      "colDate": "Date",
      "colActions": "Actions",
      "title": "Version History",
      "diff": "Diff",
      "empty": "No version history yet",
      "history": "Version History"
    }
  },
  "accounts": {
    "bind": {
      "fields": {
        "alias": "Account Alias"
      },
      "placeholders": {
        "alias": "Optional custom name"
      },
      "messages": {
        "changeCredentials": "Change credentials"
      }
    },
    "messages": {
      "shareLinkCopied": "Share link copied to clipboard",
      "shareLinkFailed": "Failed to create share link"
    }
  },
  "sre": {
    "breakers": {
      "columns": {
        "strategyId": "Strategy ID",
        "state": "State",
        "totalPnl": "Total P&L",
        "lossPercent": "Loss %",
        "tradeCount": "Trades",
        "trippedAt": "Tripped At",
        "tripReason": "Trip Reason"
      },
      "title": "Strategy Breakers",
      "stateClosed": "Normal",
      "stateOpen": "Tripped",
      "stateHalfOpen": "Half-Open (probing)",
      "confirmReset": "Reset this breaker?",
      "description": "Strategy breaker status overview — auto-detects abnormal losses and trips",
      "noBreakers": "No registered breakers"
    },
    "canary": {
      "columns": {
        "strategyId": "Strategy ID",
        "versionTag": "Version Tag",
        "accounts": "Canary Accounts",
        "startAt": "Start At",
        "days": "Days",
        "status": "Status"
      },
      "promoted": "Promoted",
      "canarying": "Canary",
      "confirmDelete": "Delete this canary config?",
      "title": "Canary Configuration",
      "description": "New strategy versions run on a few accounts for N days before promotion to all",
      "newCanary": "New Canary",
      "noCanaries": "No canary configs",
      "newCanaryTitle": "New Canary",
      "accountIdsLabel": "Canary Account IDs (comma or newline separated)",
      "durationDays": "Canary Days"
    },
    "killSwitch": {
      "description": "One-click stop all trading — requires KILL confirmation; undo within 5 minutes",
      "engaged": "Kill Switch engaged — all trading stopped",
      "disarmed": "Kill Switch disarmed — trading normal",
      "status": "Status",
      "reason": "Reason",
      "operator": "Operator",
      "engagedAt": "Engaged At",
      "undo": "Undo Kill Switch",
      "disengage": "Disengage Kill Switch",
      "engage": "Engage Kill Switch",
      "confirmTitle": "Engage Kill Switch — Confirmation",
      "confirmEngage": "Confirm Engage",
      "confirmWarning": "This will immediately stop all trading activity for all accounts, including pending and submitted orders. Enter a reason and type KILL to confirm.",
      "reasonLabel": "Reason (required)",
      "reasonPlaceholder": "e.g.: Detected abnormal market volatility, emergency stop all trading",
      "typeKill": "Type KILL to confirm",
      "typeKillPlaceholder": "Type KILL (uppercase)"
    }
  },
  "marketplace": {
    "publish": {
      "priceModel": {
        "free": "Free",
        "subscription": "Monthly Subscription",
        "once": "One-Time Purchase",
        "label": "Pricing"
      },
      "assetClass": {
        "label": "Asset Class"
      },
      "riskLevel": {
        "label": "Risk Level"
      },
      "return": "Return",
      "winRate": "Win Rate",
      "trades": "Trades",
      "title": "Publish to Marketplace",
      "titleLabel": "Title",
      "titlePlaceholder": "e.g. Golden Cross Strategy",
      "descriptionLabel": "Description",
      "descriptionPlaceholder": "Describe your strategy logic, entry/exit rules...",
      "priceAmount": "Amount",
      "tags": "Tags",
      "tagsPlaceholder": "Type and press enter to add tags",
      "codeSnippet": "Strategy Preview (public)",
      "codeSnippetPlaceholder": "Optional: share a snippet or high-level idea of your strategy (visible to all)",
      "includeBacktestSnapshot": "Include latest backtest results"
    },
    "author": {
      "avgRating": "Avg Rating",
      "empty": "No strategies published yet. Go to Strategy Library to publish one.",
      "published": "Published",
      "myStrategies": "My Published Strategies",
      "publishNew": "Publish New Strategy",
      "monthlyRevenue": "Monthly Revenue",
      "totalRevenue": "Total Revenue",
      "goToLibrary": "Go to Strategy Library"
    },
    "card": {
      "by": "by",
      "free": "Free",
      "owned": "Purchased",
      "subscribers": "Subscribers",
      "winRate": "Win Rate",
      "yourStrategy": "Your Strategy"
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
      "yourRating": "Your Rating",
      "runBacktest": "Run Backtest"
    },
    "messages": {
      "commentFailed": "Comment failed",
      "commentPosted": "Comment posted",
      "loginFirst": "Please log in first",
      "paymentComingSoon": "Payment coming soon",
      "rateFailed": "Rating failed",
      "rated": "Rating submitted",
      "subscribeFailed": "Failed",
      "subscribed": "Added to your purchases",
      "published": "Strategy published to marketplace!",
      "publishFailed": "Failed to publish strategy"
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
      "strategy": "Strategy",
      "runBacktest": "Run Backtest"
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
    "backtest": {
      "title": "Strategy Backtest",
      "capital": "Capital",
      "commission": "Commission",
      "leverage": "Leverage",
      "completed": "Completed",
      "totalReturn": "Total Return",
      "maxDrawdown": "Max Drawdown",
      "sharpe": "Sharpe",
      "winRate": "Win Rate",
      "totalTrades": "Total Trades",
      "equityCurve": "Equity Curve",
      "protected": "Strategy code is protected. Backtest runs on our servers.",
      "run": "Run Backtest",
      "idle": "Set parameters and run a backtest"
    },
    "empty": "No strategies published yet",
    "filterByClass": "Filter by asset class",
    "noSubscriptions": "No subscriptions yet",
    "searchPlaceholder": "Search strategies...",
    "subtitle": "Discover, buy, and use community strategies",
    "title": "Strategy Marketplace"
  },
  "onboarding": {
    "step1": {
      "title": "Connect Your Account",
      "desc": "Link your MT4/MT5 trading account to start.",
      "action": "Bind Account"
    },
    "step2": {
      "title": "Create Your First Strategy",
      "desc": "Use AI to generate a trading strategy from natural language.",
      "action": "Open Workspace"
    },
    "step3": {
      "title": "Upgrade Your Plan",
      "desc": "Unlock more AI tokens, strategies, and live trading with Pro.",
      "action": "View Plans"
    },
    "subtitle": "Get started in 3 simple steps",
    "dismiss": "Got it, dismiss"
  },
  "auth": {
    "fields": {
      "confirmPassword": "Confirm password",
      "email": "Email",
      "password": "Password",
      "login": "邮箱/账号"
    },
    "forgotPassword": {
      "backToLogin": "Back to Login",
      "title": "Reset Password",
      "emailTab": "Email",
      "mtTab": "MT Verify",
      "adminTab": "Admin",
      "emailSent": "If the email exists, a reset link has been sent.",
      "sendResetLink": "Send Reset Link",
      "platform": "Platform",
      "brokerName": "Broker Name",
      "brokerPlaceholder": "Enter broker name to search",
      "company": "Company",
      "selectCompany": "Select company",
      "server": "Server",
      "selectServer": "Select server",
      "noBrokers": "No brokers found.",
      "searchFailed": "Broker search failed.",
      "mtLogin": "MT Account Number",
      "mtLoginPlaceholder": "e.g. 12345678",
      "mtPassword": "MT Password",
      "mtPasswordPlaceholder": "MT trading password",
      "mtHint": "Enter your bound MT account credentials to verify your identity.",
      "verifyAndReset": "Verify & Reset Password",
      "mtVerified": "Identity verified. Redirecting to password reset.",
      "mtFailed": "MT credential verification failed.",
      "adminHint": "Please contact your administrator or support to reset your password."
    },
    "resetPassword": {
      "title": "Set New Password",
      "newPassword": "New Password",
      "confirmPassword": "Confirm Password",
      "confirmRequired": "Please confirm your password",
      "submit": "Reset Password",
      "success": "Password has been reset. Please log in with your new password.",
      "failed": "Failed to reset password.",
      "mismatch": "Passwords do not match.",
      "invalidToken": "Invalid or missing reset token."
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
      "passwordRequired": "Password is required",
      "loginRequired": "Please enter your email or account number"
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
    "saved": "Saved",
    "unknownError": "Unknown error",
    "duplicateName": "Name already exists",
    "step1Label": "Broker",
    "step2Label": "Credentials",
    "step3Label": "Confirm",
    "unit": "units",
    "action": "Action",
    "on": "On",
    "off": "Off",
    "true": "true",
    "false": "false",
    "success": "Success",
    "failed": "Failed",
    "reset": "Reset",
    "saving": "saving..."
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
      "request_failed": "API request failed",
      "insufficient_balance_title": "Insufficient Balance",
      "insufficient_balance": "Your AI wallet balance is insufficient. Please top up before continuing."
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
  "subscription": {
    "feature": {
      "aiTokens": "{{count}} AI tokens/mo",
      "strategies": "{{count}} strategies",
      "backtests": "{{count}} backtests/day",
      "liveStrategies": "{{count}} live strategies",
      "symbols": "{{count}} symbols/strategy"
    },
    "title": "Subscription Plans",
    "subscribeSuccess": "Subscription activated successfully!",
    "charged": "Charged: {{amount}}, Balance: {{balance}}",
    "insufficientBalance": "Insufficient wallet balance. Please top up your wallet first.",
    "subscribeFailed": "Subscription failed. Please try again.",
    "cancelSuccess": "Auto-renewal cancelled. Your subscription remains active until the period ends.",
    "cancelFailed": "Failed to cancel. Please try again.",
    "changeSuccess": "Plan changed successfully!",
    "changeFailed": "Plan change failed. Please try again.",
    "billingCycle": "Billing",
    "autoRenew": "Auto-renew",
    "period": "Current period",
    "cancelAutoRenew": "Cancel Auto-renew",
    "usageTitle": "Current Month Usage",
    "aiTokens": "AI Tokens",
    "activeStrategies": "Active Strategies",
    "runtimeMinutes": "Runtime (min)",
    "walletBalance": "Wallet Balance",
    "month": "mo",
    "year": "yr",
    "freeForever": "Free forever",
    "currentPlan": "Current Plan",
    "choosePlan": "Choose Plan",
    "noPlans": "No plans available",
    "changePlanTitle": "Change Plan",
    "subscribeTitle": "Subscribe to Plan",
    "selectBillingCycle": "Billing Cycle",
    "monthly": "Monthly",
    "yearly": "Yearly",
    "chargeNotice": "Your wallet will be charged for paid plans. Free plans have no charge."
  },
  "agent": {
    "analysis": {
      "title": "Backtest Analysis",
      "sharpe": "Sharpe",
      "drawdown": "DD",
      "winrate": "Win Rate",
      "consistency": "Consistency",
      "risk_adj": "Risk-Adj Return",
      "overfitting": "Overfitting Risk",
      "observations": "Key Observations",
      "suggestions": "Improvement Suggestions",
      "detailed": "Detailed Analysis"
    },
    "semantic_diff": {
      "title": "Strategy Changes",
      "effect": "Effect"
    },
    "profile": {
      "title": "Strategy Profile",
      "timeframe": "Timeframe",
      "regime": "Market Regime",
      "indicators": "Indicators",
      "entry": "Entry",
      "exit": "Exit",
      "risk": "Risk Management",
      "coverage": "Coverage",
      "strengths": "Strengths",
      "weaknesses": "Weaknesses",
      "blind_spots": "Blind Spots"
    }
  },
  "importAnalysis": {
    "execution": {
      "onBar": "Bar close event-driven",
      "onTick": "Tick-driven",
      "onInitGrid": "Init grid"
    },
    "sizing": {
      "fixed": "Fixed lots",
      "martingale": "Martingale",
      "percentBalance": "Percent of balance"
    },
    "analyzing": "Analyzing strategy structure...",
    "tradeLogicComplete": "Trading logic fully recognized",
    "guiNoiseDesc": "The following blind spots are chart display/button features that are skipped during server-side execution and do not affect trading results. Safe to import.",
    "cannotImport": "Cannot auto-import",
    "incompleteCoverage": "Trading logic coverage incomplete",
    "goodCoverage": "Import coverage is good",
    "goodCoverageDesc": "Strategy main logic recognized. Safe to import. Check parameter list before use.",
    "coverageTitle": "Import Coverage",
    "location": "Location",
    "handling": "Handling",
    "userActionRequired": "Your action required",
    "noBlindSpots": "No logic needs confirmation",
    "noBlindSpotsDesc": "All strategy logic auto-recognized. Safe to import."
  },
  "dashboard": {
    "quickActions": {
      "aiStrategy": "AI Strategy"
    }
  },
  "logs": {
    "triggerSource": {
      "manual": "Manual",
      "strategy": "Strategy",
      "recovery": "Recovery"
    },
    "result": {
      "pass": "PASS",
      "reject": "REJECT"
    }
  },
  "app": {
    "name": "AlphaForge"
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
    "mtSessionLost": "⚠ MT session lost — reconnecting…",
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
    "strategyLive": "Live Monitor",
    "strategyWorkspace": "Strategy Workspace",
    "subscription": "Subscription",
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
    "footer": "Generated by AlphaForge",
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
    "worstTrade": "Worst Trade",
    "countUnit": "笔"
  },
  "topbar": {
    "logout": "Logout",
    "profile": "Profile",
    "settings": "Settings",
    "switchToAdmin": "Switch to Admin",
    "systemOk": "System running normally",
    "user": "User"
  },
  "theme": {
    "switchToDark": "Switch to dark mode",
    "switchToLight": "Switch to light mode"
  },
  "monitoring": {
    "unknown": "Unknown",
    "healthy": "OK",
    "title": "System Monitoring",
    "sseConnected": "SSE Connected",
    "disconnected": "Disconnected",
    "streamError": "Stream Error",
    "waitingData": "Waiting for data...",
    "serviceHealth": "Service Health",
    "uptime": "Uptime",
    "database": "Database",
    "diskUsage": "Disk Usage",
    "goRuntime": "Go Runtime",
    "goroutines": "Goroutines",
    "gcCount": "GC Count",
    "gcPauseAvg": "GC Pause Avg",
    "stackUsage": "Stack Usage",
    "heapMemory": "Heap Memory",
    "dbPool": "DB Connection Pool",
    "totalConns": "Total",
    "idle": "Idle",
    "acquired": "Acquired",
    "mdGateway": "MD Gateway",
    "spillFiles": "Spill Files",
    "droppedBars": "Dropped Bars",
    "droppedSignals": "Dropped Signals",
    "consumerLag": "Consumer Lag",
    "staleAccounts": "Stale Accounts",
    "deadAccounts": "Dead Accounts",
    "avgGapSec": "Avg Gap (s)",
    "maxGapSec": "Max Gap (s)",
    "dlq": "Dead Letter Queue (DLQ)",
    "parseErrors": "Parse Errors",
    "bidGtAsk": "Bid>Ask",
    "nonPositive": "Non-Positive",
    "pushInterval": "Push interval: 5s",
    "lastUpdate": "Last update"
  }
} as const;
export default Base;
