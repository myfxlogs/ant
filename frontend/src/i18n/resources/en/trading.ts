const trading = {
  trading: {
    risk: {
      errors: {
        RISK_ACCOUNT_TRADE_DISABLED: {
          title: 'Trading is disabled for this account.',
          action: 'Check account status and permissions, then try again.',
        },
        RISK_SYMBOL_TRADE_DISABLED: {
          title: 'This symbol is currently not tradable.',
          action: 'Switch to a tradable symbol or try later.',
        },
        RISK_MARKET_SESSION_CLOSED: {
          title: 'Market is closed for this symbol.',
          action: 'Wait for the next trading session and retry.',
        },
        RISK_VOLUME_INVALID: {
          title: 'Order volume is invalid.',
          action: 'Adjust volume to match min/max/step requirements.',
        },
        RISK_ORDER_TYPE_UNSUPPORTED: {
          title: 'This order type is not supported for the symbol.',
          action: 'Choose a supported order type and retry.',
        },
        RISK_STOP_DISTANCE_TOO_CLOSE: {
          title: 'Stop-loss or take-profit is too close to market price.',
          action: 'Increase SL/TP distance and retry.',
        },
        RISK_ORDER_FROZEN_ZONE: {
          title: 'Order cannot be modified in the freeze zone.',
          action: 'Wait until price moves away from freeze distance, then retry.',
        },
        RISK_MARGIN_INSUFFICIENT: {
          title: 'Insufficient free margin to place this order.',
          action: 'Reduce volume, close positions, or add funds.',
        },
        RISK_MAX_OPEN_POSITIONS_EXCEEDED: {
          title: 'Maximum open positions limit reached.',
          action: 'Close existing positions or raise the limit.',
        },
        RISK_MAX_PENDING_ORDERS_EXCEEDED: {
          title: 'Maximum pending orders limit reached.',
          action: 'Cancel existing pending orders or raise the limit.',
        },
        RISK_INTERNAL_RULE_UNAVAILABLE: {
          title: 'Risk rules are temporarily unavailable.',
          action: 'Retry later; contact support if the issue persists.',
        },
        unknown: {
          title: 'Trade request was rejected.',
          action: 'Please review order parameters and try again.',
        },
      },
    },
    messages: {
      fetchPositionsFailed: 'Failed to load positions',
      orderSendSuccess: 'Order placed successfully',
      orderSendFailed: 'Failed to place order',
      orderModifySuccess: 'Order updated successfully',
      orderModifyFailed: 'Failed to update order',
      orderCloseSuccess: 'Position closed successfully',
      orderCloseFailed: 'Failed to close position',
      fetchPendingOrdersFailed: 'Failed to load pending orders',
      fetchOrderHistoryFailed: 'Failed to load order history',
    },
    riskConfig: {
      fields: {
        maxRiskPercent: 'Max Risk per Trade',
        maxDailyLoss: 'Max Daily Loss',
        maxDrawdownPercent: 'Max Drawdown Limit',
        maxPositions: 'Max Open Positions',
        maxLotSize: 'Max Lot Size',
        trailingStopEnabled: 'Trailing Stop',
        trailingStopPips: 'Trailing Stop (pips)',
      },
      confirm: {
        title: 'Confirm Risk Settings',
        confirmText: 'Save',
        description: 'Please confirm the following risk settings:',
        info: 'After saving, all auto trading will follow the new risk limits.',
      },
    },
    strategyExecute: {
      confirm: {
        title: 'Confirm Trade Execution',
        confirmText: 'Execute',
        warningTitle: 'Trade execution confirmation',
        warningDescription: 'This action will place a real trade immediately. Please verify all parameters.',
        strategyName: 'Strategy',
        symbol: 'Symbol',
        action: 'Side',
        buy: 'Buy',
        sell: 'Sell',
        volume: 'Volume',
      },
    },
    autoTrade: {
      confirm: {
        enableTitle: 'Enable auto trading',
        disableTitle: 'Disable auto trading',
        enableConfirm: 'Enable',
        disableConfirm: 'Disable',
        enableRiskTitle: 'Risk notice',
        enableRiskDescription: 'Auto trading will place orders automatically. Please ensure you understand the risks.',
        enableQuestion: 'Are you sure you want to enable auto trading?',
        enableBullet1: 'Orders will be executed automatically.',
        enableBullet2: 'Market volatility can cause losses.',
        enableBullet3: 'You can disable auto trading at any time.',
        disableInfoTitle: 'Disable auto trading',
        disableInfoDescription: 'Auto trading will stop placing new orders.',
        disableQuestion: 'Are you sure you want to disable auto trading?',
      },
    },
  },
} as const;

export default trading;
