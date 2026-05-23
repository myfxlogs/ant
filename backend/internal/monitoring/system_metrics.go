package monitoring

import (
	"context"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type SystemMetricsCollector struct {
	redisClient       *redis.Client
	monitoringService *MonitoringService
	stopCh            chan struct{}
}

func NewSystemMetricsCollector(redisClient *redis.Client, monitoringService *MonitoringService) *SystemMetricsCollector {
	return &SystemMetricsCollector{
		redisClient:       redisClient,
		monitoringService: monitoringService,
		stopCh:            make(chan struct{}),
	}
}

func (smc *SystemMetricsCollector) Start() {
	go smc.collectSystemMetrics()
	go smc.collectRedisMetrics()
	go smc.collectApplicationMetrics()
}

func (smc *SystemMetricsCollector) Stop() {
	close(smc.stopCh)
}

func (smc *SystemMetricsCollector) collectSystemMetrics() {
	ticker := time.NewTicker(time.Minute * 1)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			smc.collectMemoryMetrics()
			smc.collectCPUMetrics()
			smc.collectGCMetrics()
			smc.collectGoroutineMetrics()

		case <-smc.stopCh:
			return
		}
	}
}

func (smc *SystemMetricsCollector) collectMemoryMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 内存使用量（字节）
	smc.monitoringService.SetGauge("memory_alloc_bytes", float64(m.Alloc), map[string]string{
		"type": "heap",
	})

	// 总分配内存
	smc.monitoringService.SetGauge("memory_total_alloc_bytes", float64(m.TotalAlloc), map[string]string{
		"type": "total",
	})

	// 系统内存
	smc.monitoringService.SetGauge("memory_sys_bytes", float64(m.Sys), map[string]string{
		"type": "system",
	})

	// 堆内存使用量
	smc.monitoringService.SetGauge("memory_heap_alloc_bytes", float64(m.HeapAlloc), map[string]string{
		"type": "heap_alloc",
	})

	// 堆内存总量
	smc.monitoringService.SetGauge("memory_heap_sys_bytes", float64(m.HeapSys), map[string]string{
		"type": "heap_sys",
	})

	// 堆内存对象数量
	smc.monitoringService.SetGauge("memory_heap_objects", float64(m.HeapObjects), map[string]string{
		"type": "heap_objects",
	})

	// GC次数
	smc.monitoringService.SetGauge("gc_cycles_total", float64(m.NumGC), map[string]string{
		"type": "gc_cycles",
	})

	// 计算内存使用百分比（假设系统总内存为8GB，实际应该从系统获取）
	totalMemory := float64(8 * 1024 * 1024 * 1024) // 8GB
	memoryUsagePercent := (float64(m.Sys) / totalMemory) * 100
	smc.monitoringService.SetGauge("memory_usage_percent", memoryUsagePercent, map[string]string{
		"type": "percentage",
	})

	// 内存使用告警
	if memoryUsagePercent > 85 {
		smc.monitoringService.SendAlert(
			AlertLevelWarning,
			"High Memory Usage",
			"Memory usage is above 85%",
			"system",
			map[string]interface{}{
				"memory_usage_percent": memoryUsagePercent,
				"alloc_bytes": m.Alloc,
				"sys_bytes": m.Sys,
			},
		)
	}
}

func (smc *SystemMetricsCollector) collectCPUMetrics() {
	// Go运行时CPU指标
	numGoroutines := float64(runtime.NumGoroutine())
	smc.monitoringService.SetGauge("goroutines_count", numGoroutines, map[string]string{
		"type": "current",
	})

	numCPU := float64(runtime.NumCPU())
	smc.monitoringService.SetGauge("cpu_count", numCPU, map[string]string{
		"type": "available",
	})

	// Goroutine数量告警
	if numGoroutines > 1000 {
		smc.monitoringService.SendAlert(
			AlertLevelWarning,
			"High Goroutine Count",
			"Goroutine count is above 1000",
			"system",
			map[string]interface{}{
				"goroutines_count": numGoroutines,
			},
		)
	}
}

func (smc *SystemMetricsCollector) collectGCMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// GC暂停时间（纳秒转换为毫秒）
	gcPauseTotal := float64(m.PauseTotalNs) / 1e6
	smc.monitoringService.SetGauge("gc_pause_total_ms", gcPauseTotal, map[string]string{
		"type": "total",
	})

	// 上次GC暂停时间
	if len(m.PauseNs) > 0 {
		lastGCPause := float64(m.PauseNs[0]) / 1e6
		smc.monitoringService.RecordHistogram("gc_pause_duration_ms", lastGCPause, map[string]string{
			"type": "last",
		})
	}

	// GC频率
	smc.monitoringService.SetGauge("gc_cycles_total", float64(m.NumGC), map[string]string{
		"type": "total",
	})

	// 强制GC次数
	smc.monitoringService.SetGauge("gc_forced_total", float64(m.NumForcedGC), map[string]string{
		"type": "forced",
	})
}

func (smc *SystemMetricsCollector) collectGoroutineMetrics() {
	count := float64(runtime.NumGoroutine())
	smc.monitoringService.SetGauge("goroutines_count", count, map[string]string{
		"type": "current",
	})

	// 记录goroutine数量历史
	smc.monitoringService.RecordHistogram("goroutines_count", count, map[string]string{
		"type": "histogram",
	})
}

func (smc *SystemMetricsCollector) collectRedisMetrics() {
	ticker := time.NewTicker(time.Minute * 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			
			// 获取Redis信息
			info, err := smc.redisClient.Info(ctx).Result()
			if err != nil {
				smc.monitoringService.IncrementCounter("redis_connection_errors", map[string]string{
					"operation": "info",
				})
				
				smc.monitoringService.SendAlert(
					AlertLevelCritical,
					"Redis Connection Error",
					"Failed to get Redis info",
					"redis",
					map[string]interface{}{
						"error": err.Error(),
					},
				)
				cancel()
				continue
			}

			// 解析Redis信息
			smc.parseRedisInfo(info)
			cancel()

		case <-smc.stopCh:
			return
		}
	}
}

func (smc *SystemMetricsCollector) parseRedisInfo(info string) {
	lines := make(map[string]string)
	
	// 简单解析Redis INFO输出
	for _, line := range strings.Split(info, "\n") {
		if line == "" || line[0] == '#' {
			continue
		}
		
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			lines[parts[0]] = parts[1]
		}
	}

	// 提取关键指标
	if connectedClients, ok := lines["connected_clients"]; ok {
		if value, err := strconv.ParseFloat(connectedClients, 64); err == nil {
			smc.monitoringService.SetGauge("redis_connected_clients", value, map[string]string{
				"type": "current",
			})
		}
	}

	if usedMemory, ok := lines["used_memory"]; ok {
		if value, err := strconv.ParseFloat(usedMemory, 64); err == nil {
			smc.monitoringService.SetGauge("redis_used_memory_bytes", value, map[string]string{
				"type": "used",
			})
		}
	}

	if maxMemory, ok := lines["maxmemory"]; ok && maxMemory != "0" {
		if used, ok := lines["used_memory"]; ok {
			if usedVal, err1 := strconv.ParseFloat(used, 64); err1 == nil {
				if maxVal, err2 := strconv.ParseFloat(maxMemory, 64); err2 == nil {
					usagePercent := (usedVal / maxVal) * 100
					smc.monitoringService.SetGauge("redis_memory_usage_percent", usagePercent, map[string]string{
						"type": "percentage",
					})

					// Redis内存使用告警
					if usagePercent > 80 {
						smc.monitoringService.SendAlert(
							AlertLevelWarning,
							"High Redis Memory Usage",
							"Redis memory usage is above 80%",
							"redis",
							map[string]interface{}{
								"usage_percent": usagePercent,
								"used_memory": usedVal,
								"max_memory": maxVal,
							},
						)
					}
				}
			}
		}
	}

	if keyspaceHits, ok := lines["keyspace_hits"]; ok {
		if hits, err1 := strconv.ParseFloat(keyspaceHits, 64); err1 == nil {
			if misses, ok := lines["keyspace_misses"]; ok {
				if missesVal, err2 := strconv.ParseFloat(misses, 64); err2 == nil {
					total := hits + missesVal
					if total > 0 {
						hitRate := (hits / total) * 100
						smc.monitoringService.SetGauge("redis_hit_rate_percent", hitRate, map[string]string{
							"type": "percentage",
						})
					}
				}
			}
		}
	}
}

func (smc *SystemMetricsCollector) collectApplicationMetrics() {
	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 收集应用级别的自定义指标
			smc.collectCustomMetrics()

		case <-smc.stopCh:
			return
		}
	}
}

func (smc *SystemMetricsCollector) collectCustomMetrics() {
	// 应用启动时间
	smc.monitoringService.SetGauge("application_uptime_seconds", 
		float64(time.Since(startTime).Seconds()), 
		map[string]string{"type": "uptime"})

	// 版本信息
	smc.monitoringService.SetGauge("application_version", 1.0, map[string]string{
		"version": "1.0.0",
		"build":   "latest",
	})

	// 健康检查状态
	smc.monitoringService.SetGauge("application_health_status", 1.0, map[string]string{
		"status": "healthy",
	})
}

// 全局变量记录应用启动时间
var startTime = time.Now()
