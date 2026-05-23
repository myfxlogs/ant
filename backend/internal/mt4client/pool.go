package mt4client

import (
	"context"
	"fmt"
	"sync"

	"anttrader/internal/config"
)

type ConnectionPool struct {
	cfg     *config.MT4Config
	maxSize int
	poolMu  sync.RWMutex
	pool    []*MT4Connection
	created int
	closed  bool
	client  *MT4Client
}

func NewConnectionPool(cfg *config.MT4Config, maxSize int) *ConnectionPool {
	return &ConnectionPool{
		cfg:     cfg,
		maxSize: maxSize,
		pool:    make([]*MT4Connection, 0, maxSize),
		client:  NewMT4Client(cfg),
	}
}

func (p *ConnectionPool) Get(ctx context.Context, user int32, password, host string, port int32) (*MT4Connection, error) {
	p.poolMu.Lock()
	if p.closed {
		p.poolMu.Unlock()
		return nil, fmt.Errorf("pool is closed")
	}

	for i, conn := range p.pool {
		if conn != nil && conn.IsConnected() {
			p.pool = append(p.pool[:i], p.pool[i+1:]...)
			p.poolMu.Unlock()
			return conn, nil
		}
	}
	p.poolMu.Unlock()

	conn, err := p.createConnection(ctx, user, password, host, port)
	if err != nil {
		return nil, err
	}

	p.poolMu.Lock()
	if !p.closed && len(p.pool) < p.maxSize {
		p.pool = append(p.pool, conn)
		p.created++
	}
	p.poolMu.Unlock()

	return conn, nil
}

func (p *ConnectionPool) createConnection(ctx context.Context, user int32, password, host string, port int32) (*MT4Connection, error) {
	return p.client.Connect(ctx, user, password, host, port)
}

func (p *ConnectionPool) Return(conn *MT4Connection) {
	if conn == nil {
		return
	}

	p.poolMu.Lock()
	if p.closed {
		p.poolMu.Unlock()
		p.client.Disconnect(context.Background(), conn.GetAccountID())
		return
	}

	if len(p.pool) >= p.maxSize {
		p.poolMu.Unlock()
		p.client.Disconnect(context.Background(), conn.GetAccountID())
		return
	}

	p.pool = append(p.pool, conn)
	p.poolMu.Unlock()

}

func (p *ConnectionPool) Close() {
	p.poolMu.Lock()
	defer p.poolMu.Unlock()

	if p.closed {
		return
	}
	p.closed = true

	for _, conn := range p.pool {
		if conn != nil && conn.IsConnected() {
			p.client.Disconnect(context.Background(), conn.GetAccountID())
		}
	}
	p.pool = p.pool[:0]

}

func (p *ConnectionPool) Size() int {
	p.poolMu.RLock()
	defer p.poolMu.RUnlock()
	return len(p.pool)
}

func (p *ConnectionPool) ActiveCount() int {
	p.poolMu.RLock()
	defer p.poolMu.RUnlock()
	count := 0
	for _, conn := range p.pool {
		if conn != nil && conn.IsConnected() {
			count++
		}
	}
	return count
}

func (p *ConnectionPool) Stats() map[string]interface{} {
	p.poolMu.RLock()
	defer p.poolMu.RUnlock()

	active := 0
	for _, conn := range p.pool {
		if conn != nil && conn.IsConnected() {
			active++
		}
	}

	return map[string]interface{}{
		"total_created": p.created,
		"pool_size":     len(p.pool),
		"active":        active,
		"max_size":      p.maxSize,
		"closed":        p.closed,
	}
}

type PoolManager struct {
	cfg         *config.MT4Config
	pools       map[string]*ConnectionPool
	mu          sync.RWMutex
	defaultPool *ConnectionPool
}

func NewPoolManager(cfg *config.MT4Config) *PoolManager {
	pm := &PoolManager{
		cfg:   cfg,
		pools: make(map[string]*ConnectionPool),
	}
	pm.defaultPool = NewConnectionPool(cfg, 10)
	return pm
}

func (pm *PoolManager) GetPool(broker string) *ConnectionPool {
	pm.mu.RLock()
	pool, exists := pm.pools[broker]
	pm.mu.RUnlock()

	if exists {
		return pool
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pool, exists = pm.pools[broker]; exists {
		return pool
	}

	pool = NewConnectionPool(pm.cfg, 10)
	pm.pools[broker] = pool
	return pool
}

func (pm *PoolManager) GetConnection(ctx context.Context, broker string, user int32, password, host string, port int32) (*MT4Connection, error) {
	pool := pm.GetPool(broker)
	return pool.Get(ctx, user, password, host, port)
}

func (pm *PoolManager) ReturnConnection(broker string, conn *MT4Connection) {
	if conn == nil {
		return
	}
	pool := pm.GetPool(broker)
	pool.Return(conn)
}

func (pm *PoolManager) Close() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for broker, pool := range pm.pools {
		pool.Close()
		delete(pm.pools, broker)
	}
}

func (pm *PoolManager) Stats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := make(map[string]interface{})
	totalActive := 0
	totalSize := 0

	for broker, pool := range pm.pools {
		poolStats := pool.Stats()
		stats[broker] = poolStats
		totalActive += poolStats["active"].(int)
		totalSize += poolStats["pool_size"].(int)
	}

	stats["total_brokers"] = len(pm.pools)
	stats["total_active"] = totalActive
	stats["total_size"] = totalSize

	return stats
}
