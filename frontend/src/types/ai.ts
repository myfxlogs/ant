// AI消息类型
export interface Message {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: Date;
  isLoading?: boolean;
}

// AI配置类型
export interface AIConfig {
  id?: string;
  provider: 'zhipu' | 'deepseek';
  api_key: string;
  model?: string;
  is_active: boolean;
  created_at?: string;
  updated_at?: string;
}

// 策略条件
export interface StrategyCondition {
  type: 'price_above' | 'price_below' | 'indicator_cross' | 'time_based' | 'custom';
  symbol?: string;
  value?: number;
  indicator?: string;
  operator?: string;
  description: string;
}

// 策略动作
export interface StrategyAction {
  type: 'buy' | 'sell' | 'close_long' | 'close_short' | 'alert';
  symbol?: string;
  volume?: number;
  stop_loss?: number;
  take_profit?: number;
  description: string;
}

// AI生成的策略
export interface Strategy {
  id: string;
  name: string;
  description: string;
  symbol: string;
  conditions: StrategyCondition[];
  actions: StrategyAction[];
  status: 'active' | 'inactive' | 'paused';
  created_at: string;
  updated_at: string;
  triggered_count?: number;
  last_triggered_at?: string;
}

// AI生成的信号
export interface Signal {
  id: string;
  type: 'buy' | 'sell';
  symbol: string;
  price: number;
  volume: number;
  stop_loss?: number;
  take_profit?: number;
  reason: string;
  confidence: number; // 0-100
  status: 'pending' | 'confirmed' | 'executed' | 'cancelled';
  account_id?: string;
  strategy_id?: string;
  created_at: string;
  executed_at?: string;
}

// AI建议
export interface AIAdvice {
  symbol: string;
  action: 'buy' | 'sell' | 'hold' | 'close';
  confidence: number;
  reason: string;
  suggested_entry?: number;
  suggested_stop_loss?: number;
  suggested_take_profit?: number;
  risk_level: 'low' | 'medium' | 'high';
  timestamp: string;
}

// AI报告
export interface AIReport {
  account_id: string;
  period: string;
  summary: string;
  performance_analysis: string;
  risk_assessment: string;
  recommendations: string[];
  generated_at: string;
}

// 流式响应事件
export interface StreamEvent {
  type: 'start' | 'delta' | 'end' | 'error';
  content?: string;
  message?: string;
}

// 聊天请求
export interface ChatRequest {
  message: string;
  conversation_id?: string;
  context?: {
    account_id?: string;
    symbol?: string;
    strategy_id?: string;
  };
}

// 策略生成请求
export interface GenerateStrategyRequest {
  input: string;
  symbol?: string;
  account_id?: string;
}

// 建议请求
export interface GetAdviceRequest {
  account_id: string;
  symbol: string;
}

// 报告生成请求
export interface GenerateReportRequest {
  account_id: string;
  period?: 'day' | 'week' | 'month';
}
