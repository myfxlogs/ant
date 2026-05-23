package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"anttrader/internal/config"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

type AlertLevel string

const (
	AlertLevelInfo    AlertLevel = "info"
	AlertLevelWarning AlertLevel = "warning"
	AlertLevelError   AlertLevel = "error"
	AlertLevelCritical AlertLevel = "critical"
)

type Alert struct {
	ID          string                 `json:"id"`
	Level       AlertLevel            `json:"level"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Timestamp   time.Time              `json:"timestamp"`
	Source      string                 `json:"source"`
	Metadata    map[string]interface{} `json:"metadata"`
	Resolved    bool                   `json:"resolved"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
}

type Metric struct {
	Name      string                 `json:"name"`
	Value     float64                `json:"value"`
	Timestamp time.Time              `json:"timestamp"`
	Tags      map[string]string      `json:"tags"`
	Type      string                 `json:"type"` // "counter", "gauge", "histogram"
}

type AlertRule struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	MetricName  string            `json:"metric_name"`
	Condition   string            `json:"condition"` // ">", "<", ">=", "<=", "=="
	Threshold   float64           `json:"threshold"`
	Duration    time.Duration     `json:"duration"`
	Level       AlertLevel        `json:"level"`
	Message     string            `json:"message"`
	Enabled     bool              `json:"enabled"`
	Tags        map[string]string `json:"tags"`
}

type MonitoringService struct {
	redisClient    *redis.Client
	config         *config.MonitoringConfig
	alertRules     map[string]*AlertRule
	activeAlerts   map[string]*Alert
	metricsBuffer  []Metric
	mu             sync.RWMutex
	alertChannels  []chan Alert
	stopCh         chan struct{}
}

func NewMonitoringService(redisClient *redis.Client, cfg *config.MonitoringConfig) *MonitoringService {
	ms := &MonitoringService{
		redisClient:   redisClient,
		config:        cfg,
		alertRules:    make(map[string]*AlertRule),
		activeAlerts:  make(map[string]*Alert),
		metricsBuffer: make([]Metric, 0, 1000),
		alertChannels: make([]chan Alert, 0),
		stopCh:        make(chan struct{}),
	}

	// 加载默认告警规则
	ms.loadDefaultAlertRules()

	// 启动后台处理
	go ms.startBackgroundProcessor()

	return ms
}

func (ms *MonitoringService) RecordMetric(name string, value float64, tags map[string]string, metricType string) {
	metric := Metric{
		Name:      name,
		Value:     value,
		Timestamp: time.Now(),
		Tags:      tags,
		Type:      metricType,
	}

	ms.mu.Lock()
	ms.metricsBuffer = append(ms.metricsBuffer, metric)
	
	// 如果缓冲区满了，刷新到Redis
	if len(ms.metricsBuffer) >= 1000 {
		go ms.flushMetrics()
	}
	ms.mu.Unlock()

	// 检查告警规则
	ms.checkAlertRules(metric)
}

func (ms *MonitoringService) IncrementCounter(name string, tags map[string]string) {
	ms.RecordMetric(name, 1, tags, "counter")
}

func (ms *MonitoringService) SetGauge(name string, value float64, tags map[string]string) {
	ms.RecordMetric(name, value, tags, "gauge")
}

func (ms *MonitoringService) RecordHistogram(name string, value float64, tags map[string]string) {
	ms.RecordMetric(name, value, tags, "histogram")
}

func (ms *MonitoringService) SendAlert(level AlertLevel, title, message, source string, metadata map[string]interface{}) {
	alert := Alert{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Level:     level,
		Title:     title,
		Message:   message,
		Timestamp: time.Now(),
		Source:    source,
		Metadata:  metadata,
		Resolved:  false,
	}

	// 存储告警
	ms.storeAlert(alert)

	// 发送到通知渠道
	ms.distributeAlert(alert)

	// 记录日志
	ms.logAlert(alert)
}

func (ms *MonitoringService) ResolveAlert(alertID string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if alert, exists := ms.activeAlerts[alertID]; exists {
		now := time.Now()
		alert.Resolved = true
		alert.ResolvedAt = &now
		
		ms.storeAlert(*alert)
		delete(ms.activeAlerts, alertID)

	}
}

func (ms *MonitoringService) GetActiveAlerts() []Alert {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	alerts := make([]Alert, 0, len(ms.activeAlerts))
	for _, alert := range ms.activeAlerts {
		alerts = append(alerts, *alert)
	}
	return alerts
}

func (ms *MonitoringService) GetMetrics(name string, startTime, endTime time.Time) ([]Metric, error) {
	ctx := context.Background()
	pattern := fmt.Sprintf("antrader:metrics:%s:*", name)
	
	keys, err := ms.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	var metrics []Metric
	for _, key := range keys {
		data, err := ms.redisClient.LRange(ctx, key, 0, -1).Result()
		if err != nil {
			continue
		}

		for _, item := range data {
			var metric Metric
			if err := json.Unmarshal([]byte(item), &metric); err == nil {
				if metric.Timestamp.After(startTime) && metric.Timestamp.Before(endTime) {
					metrics = append(metrics, metric)
				}
			}
		}
	}

	return metrics, nil
}

func (ms *MonitoringService) AddAlertChannel(ch chan Alert) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.alertChannels = append(ms.alertChannels, ch)
}

func (ms *MonitoringService) loadDefaultAlertRules() {
	defaultRules := []*AlertRule{
		{
			ID:         "high_error_rate",
			Name:       "High Error Rate",
			MetricName: "http_requests_total",
			Condition:  ">",
			Threshold:  0.05, // 5% error rate
			Duration:   time.Minute * 5,
			Level:      AlertLevelWarning,
			Message:    "Error rate is above 5%",
			Enabled:    true,
			Tags:       map[string]string{"status": "500"},
		},
		{
			ID:         "high_response_time",
			Name:       "High Response Time",
			MetricName: "http_request_duration_seconds",
			Condition:  ">",
			Threshold:  2.0, // 2 seconds
			Duration:   time.Minute * 5,
			Level:      AlertLevelWarning,
			Message:    "Response time is above 2 seconds",
			Enabled:    true,
		},
		{
			ID:         "redis_connection_failure",
			Name:       "Redis Connection Failure",
			MetricName: "redis_connection_errors",
			Condition:  ">",
			Threshold:  0,
			Duration:   time.Minute * 1,
			Level:      AlertLevelCritical,
			Message:    "Redis connection failures detected",
			Enabled:    true,
		},
		{
			ID:         "database_connection_failure",
			Name:       "Database Connection Failure",
			MetricName: "database_connection_errors",
			Condition:  ">",
			Threshold:  0,
			Duration:   time.Minute * 1,
			Level:      AlertLevelCritical,
			Message:    "Database connection failures detected",
			Enabled:    true,
		},
		{
			ID:         "high_memory_usage",
			Name:       "High Memory Usage",
			MetricName: "memory_usage_percent",
			Condition:  ">",
			Threshold:  85.0, // 85%
			Duration:   time.Minute * 5,
			Level:      AlertLevelWarning,
			Message:    "Memory usage is above 85%",
			Enabled:    true,
		},
		{
			ID:         "high_cpu_usage",
			Name:       "High CPU Usage",
			MetricName: "cpu_usage_percent",
			Condition:  ">",
			Threshold:  80.0, // 80%
			Duration:   time.Minute * 5,
			Level:      AlertLevelWarning,
			Message:    "CPU usage is above 80%",
			Enabled:    true,
		},
	}

	for _, rule := range defaultRules {
		ms.alertRules[rule.ID] = rule
	}
}

func (ms *MonitoringService) checkAlertRules(metric Metric) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	for _, rule := range ms.alertRules {
		if !rule.Enabled || rule.MetricName != metric.Name {
			continue
		}

		if ms.evaluateCondition(metric.Value, rule.Condition, rule.Threshold) {
			// 检查是否已经有活跃的告警
			alertID := fmt.Sprintf("%s_%s", rule.ID, metric.Name)
			if _, exists := ms.activeAlerts[alertID]; !exists {
				alert := Alert{
					ID:        alertID,
					Level:     rule.Level,
					Title:     rule.Name,
					Message:   rule.Message,
					Timestamp: time.Now(),
					Source:    "monitoring_system",
					Metadata: map[string]interface{}{
						"metric_name": metric.Name,
						"metric_value": metric.Value,
						"threshold": rule.Threshold,
						"rule_id": rule.ID,
					},
					Resolved: false,
				}

				ms.activeAlerts[alertID] = &alert
				ms.storeAlert(alert)
				ms.distributeAlert(alert)
				ms.logAlert(alert)
			}
		}
	}
}

func (ms *MonitoringService) evaluateCondition(value float64, condition string, threshold float64) bool {
	switch condition {
	case ">":
		return value > threshold
	case "<":
		return value < threshold
	case ">=":
		return value >= threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	default:
		return false
	}
}

func (ms *MonitoringService) storeAlert(alert Alert) {
	ctx := context.Background()
	data, _ := json.Marshal(alert)
	
	key := fmt.Sprintf("antrader:alerts:%s", alert.ID)
	ms.redisClient.Set(ctx, key, data, time.Hour*24*7) // 保存7天
	
	// 添加到时间序列列表
	listKey := fmt.Sprintf("antrader:alerts:timeline")
	ms.redisClient.LPush(ctx, listKey, data)
	ms.redisClient.LTrim(ctx, listKey, 0, 999) // 保留最新1000条
}

func (ms *MonitoringService) distributeAlert(alert Alert) {
	ms.mu.RLock()
	channels := make([]chan Alert, len(ms.alertChannels))
	copy(channels, ms.alertChannels)
	ms.mu.RUnlock()

	for _, ch := range channels {
		select {
		case ch <- alert:
		default:
			// 如果通道满了，跳过
		}
	}
}

func (ms *MonitoringService) logAlert(alert Alert) {
	fields := []zap.Field{
		zap.String("alert_id", alert.ID),
		zap.String("level", string(alert.Level)),
		zap.String("title", alert.Title),
		zap.String("message", alert.Message),
		zap.String("source", alert.Source),
		zap.Time("timestamp", alert.Timestamp),
	}

	switch alert.Level {
	case AlertLevelCritical:
		logger.Error("CRITICAL ALERT", fields...)
	case AlertLevelError:
		logger.Error("ERROR ALERT", fields...)
	case AlertLevelWarning:
		logger.Warn("WARNING ALERT", fields...)
	case AlertLevelInfo:
		return
	}
}

func (ms *MonitoringService) flushMetrics() {
	ms.mu.Lock()
	if len(ms.metricsBuffer) == 0 {
		ms.mu.Unlock()
		return
	}

	metrics := make([]Metric, len(ms.metricsBuffer))
	copy(metrics, ms.metricsBuffer)
	ms.metricsBuffer = ms.metricsBuffer[:0]
	ms.mu.Unlock()

	ctx := context.Background()
	pipe := ms.redisClient.Pipeline()

	for _, metric := range metrics {
		data, _ := json.Marshal(metric)
		key := fmt.Sprintf("antrader:metrics:%s:%d", metric.Name, metric.Timestamp.Unix())
		pipe.LPush(ctx, key, data)
		pipe.Expire(ctx, key, time.Hour*24) // 保存24小时
	}

	pipe.Exec(ctx)
}

func (ms *MonitoringService) startBackgroundProcessor() {
	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 刷新指标
			ms.flushMetrics()
			
			// 清理过期的告警
			ms.cleanupExpiredAlerts()
			
		case <-ms.stopCh:
			return
		}
	}
}

func (ms *MonitoringService) cleanupExpiredAlerts() {
	ctx := context.Background()

	pattern := "antrader:alerts:*"
	iter := ms.redisClient.Scan(ctx, 0, pattern, 100).Iterator()
	
	for iter.Next(ctx) {
		key := iter.Val()
		ttl := ms.redisClient.TTL(ctx, key).Val()
		if ttl == -1 { // 没有过期时间
			ms.redisClient.Expire(ctx, key, time.Hour*24)
		}
	}
}

func (ms *MonitoringService) Stop() {
	close(ms.stopCh)
	ms.flushMetrics()
}

func (ms *MonitoringService) GetSystemMetrics() (map[string]interface{}, error) {
	ctx := context.Background()
	
	// 获取Redis指标
	redisInfo := ms.redisClient.Info(ctx).Val()
	
	// 获取系统指标
	metrics := map[string]interface{}{
		"redis_info": redisInfo,
		"active_alerts_count": len(ms.activeAlerts),
		"buffered_metrics_count": len(ms.metricsBuffer),
		"alert_rules_count": len(ms.alertRules),
		"timestamp": time.Now(),
	}
	
	return metrics, nil
}
