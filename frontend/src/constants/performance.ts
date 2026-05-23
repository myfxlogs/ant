// 性能优化相关常量
export const PERFORMANCE_CONFIG = {
  // SSE批量处理间隔 (毫秒)
  BATCH_INTERVAL_MS: 500,
  
  // 单个报价节流间隔 (毫秒)
  QUOTE_THROTTLE_MS: 100,
  
  // 节流缓存清理间隔 (毫秒)
  THROTTLE_CLEANUP_INTERVAL_MS: 10000,
  
  // 节流缓存过期时间 (毫秒)
  THROTTLE_CACHE_EXPIRY_MS: 5000,
  
  // 控制台日志采样率 (0-1)
  CONSOLE_LOG_SAMPLE_RATE: 0.01,
  
  // 价格显示精度
  PRICE_DECIMAL_PLACES: 5,
  
  // 心跳超时时间 (毫秒)
  HEARTBEAT_TIMEOUT_MS: 30000,
  
  // SSE重连延迟基数 (毫秒)
  RECONNECT_DELAY_BASE_MS: 3000,
  
  // 最大重连次数
  MAX_RECONNECT_ATTEMPTS: 10,
} as const;

// 图表颜色配置
export const CHART_COLORS = [
  '#00A651', // 绿色
  '#E53935', // 红色
  '#D4AF37', // 金色
  '#2196F3', // 蓝色
  '#9C27B0', // 紫色
  '#FF9800', // 橙色
  '#00BCD4', // 青色
] as const;
