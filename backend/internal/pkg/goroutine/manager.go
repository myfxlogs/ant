package goroutine

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"
	"anttrader/pkg/logger"
)

// Manager goroutine 管理器
type Manager struct {
	mu          sync.RWMutex
	goroutines  map[string]*GoroutineInfo
	maxGoroutines int
	stats       *Stats
}

// GoroutineInfo goroutine 信息
type GoroutineInfo struct {
	ID          string
	Name        string
	StartTime   time.Time
	Context     context.Context
	Cancel      context.CancelFunc
	Status      string
	Error       error
}

// Stats 统计信息
type Stats struct {
	TotalSpawned  int64
	TotalFinished int64
	ActiveCount   int64
	MaxReached    int64
	ErrorCount    int64
	mu            sync.RWMutex
}

// Option 配置选项
type Option func(*Manager)

// WithMaxGoroutines 设置最大 goroutine 数量
func WithMaxGoroutines(max int) Option {
	return func(m *Manager) {
		m.maxGoroutines = max
	}
}

// NewManager 创建新的 goroutine 管理器
func NewManager(opts ...Option) *Manager {
	manager := &Manager{
		goroutines:    make(map[string]*GoroutineInfo),
		maxGoroutines: 10000, // 默认最大 10000 个
		stats:         &Stats{},
	}

	for _, opt := range opts {
		opt(manager)
	}

	// 启动监控 goroutine
	go manager.monitorLoop()

	return manager
}

// Spawn 启动受管 goroutine
func (m *Manager) Spawn(name string, fn func(context.Context) error) (string, error) {
	m.stats.mu.Lock()
	if m.stats.ActiveCount >= int64(m.maxGoroutines) {
		m.stats.mu.Unlock()
		return "", fmt.Errorf("maximum goroutines reached: %d", m.maxGoroutines)
	}
	m.stats.TotalSpawned++
	m.stats.ActiveCount++
	if m.stats.ActiveCount > m.stats.MaxReached {
		m.stats.MaxReached = m.stats.ActiveCount
	}
	m.stats.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	id := fmt.Sprintf("%s-%d", name, time.Now().UnixNano())

	info := &GoroutineInfo{
		ID:        id,
		Name:      name,
		StartTime: time.Now(),
		Context:   ctx,
		Cancel:    cancel,
		Status:    "running",
	}

	m.mu.Lock()
	m.goroutines[id] = info
	m.mu.Unlock()

	go func() {
		defer func() {
			m.cleanupGoroutine(id)
			if r := recover(); r != nil {
				logger.Error("Goroutine panicked",
					zap.String("id", id),
					zap.String("name", name),
					zap.Any("panic", r))
				
				m.stats.mu.Lock()
				m.stats.ErrorCount++
				m.stats.mu.Unlock()
			}
		}()

		err := fn(ctx)
		if err != nil {
			// Expected cancellations should not be treated as failures.
			if err == context.Canceled || err == context.DeadlineExceeded {
				info.Status = "completed"
				return
			}
			info.Error = err
			info.Status = "error"
			logger.Error("Goroutine failed",
				zap.String("id", id),
				zap.String("name", name),
				zap.Error(err))
			
			m.stats.mu.Lock()
			m.stats.ErrorCount++
			m.stats.mu.Unlock()
		} else {
			info.Status = "completed"
		}
	}()

	return id, nil
}

// Cancel 取消指定的 goroutine
func (m *Manager) Cancel(id string) bool {
	m.mu.RLock()
	info, exists := m.goroutines[id]
	m.mu.RUnlock()

	if !exists {
		return false
	}

	info.Cancel()
	return true
}

// GetInfo 获取 goroutine 信息
func (m *Manager) GetInfo(id string) (*GoroutineInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	info, exists := m.goroutines[id]
	return info, exists
}

// ListActive 获取所有活跃的 goroutine
func (m *Manager) ListActive() []*GoroutineInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	active := make([]*GoroutineInfo, 0)
	for _, info := range m.goroutines {
		if info.Status == "running" {
			active = append(active, info)
		}
	}
	return active
}

// GetStats 获取统计信息
func (m *Manager) GetStats() *Stats {
	m.stats.mu.RLock()
	defer m.stats.mu.RUnlock()
	
	// 返回副本以避免竞态
	return &Stats{
		TotalSpawned:  m.stats.TotalSpawned,
		TotalFinished: m.stats.TotalFinished,
		ActiveCount:   m.stats.ActiveCount,
		MaxReached:    m.stats.MaxReached,
		ErrorCount:    m.stats.ErrorCount,
	}
}

// ForceCleanup 强制清理所有已完成的 goroutine
func (m *Manager) ForceCleanup() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	cleaned := 0
	for id, info := range m.goroutines {
		if info.Status != "running" {
			delete(m.goroutines, id)
			cleaned++
		}
	}

	m.stats.mu.Lock()
	m.stats.TotalFinished += int64(cleaned)
	m.stats.ActiveCount -= int64(cleaned)
	m.stats.mu.Unlock()

	return cleaned
}

// cleanupGoroutine 清理完成的 goroutine
func (m *Manager) cleanupGoroutine(id string) {
	m.mu.Lock()
	delete(m.goroutines, id)
	m.mu.Unlock()

	m.stats.mu.Lock()
	m.stats.TotalFinished++
	m.stats.ActiveCount--
	m.stats.mu.Unlock()
}

// monitorLoop 监控循环
func (m *Manager) monitorLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 检查 goroutine 数量
		activeCount := len(m.ListActive())
		
		// 如果活跃 goroutine 过多，强制清理
		if activeCount > int(float64(m.maxGoroutines)*0.8) {
			cleaned := m.ForceCleanup()
			if cleaned > 0 {
				logger.Warn("Force cleaned goroutines due to high count",
					zap.Int("cleaned", cleaned),
					zap.Int("active", activeCount))
			}
		}

		// 检查系统 goroutine 数量
		sysGoroutines := runtime.NumGoroutine()
		if sysGoroutines > m.maxGoroutines*2 {
			logger.Error("System goroutine count is extremely high",
				zap.Int("system_goroutines", sysGoroutines),
				zap.Int("managed_active", activeCount))
		}
	}
}

// SafeGo 安全地启动 goroutine（带超时和取消）
func SafeGo(ctx context.Context, name string, fn func(context.Context) error) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)
	
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("SafeGo goroutine panicked",
					zap.String("name", name),
					zap.Any("panic", r))
			}
		}()

		err := fn(ctx)
		if err != nil && err != context.Canceled {
			logger.Error("SafeGo goroutine failed",
				zap.String("name", name),
				zap.Error(err))
		}
	}()

	return cancel
}

// WithTimeout 启动带超时的 goroutine
func WithTimeout(ctx context.Context, name string, timeout time.Duration, fn func(context.Context) error) context.CancelFunc {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	
	go func() {
		defer func() {
			cancel() // 确保超时后取消 context
			if r := recover(); r != nil {
				logger.Error("WithTimeout goroutine panicked",
					zap.String("name", name),
					zap.Any("panic", r))
			}
		}()

		err := fn(ctx)
		if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			logger.Error("WithTimeout goroutine failed",
				zap.String("name", name),
				zap.Error(err))
		}
	}()

	return cancel
}