package connection

import (
	"time"
)

func (m *ConnectionManager) cleanupLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanupIdleConnections()
		case <-m.ctx.Done():
			return
		}
	}
}

func (m *ConnectionManager) cleanupIdleConnections() {
	if m == nil {
		return
	}

	cutoff := time.Now().Add(-3 * time.Minute)

	if err := m.connectionsMu.Lock(); err != nil {
		return
	}
	removed := 0
	for id, conn := range m.connections {
		if conn == nil {
			delete(m.connections, id)
			removed++
			continue
		}
		if conn.TotalSubscriptions() != 0 {
			continue
		}
		if conn.IsConnected() {
			continue
		}
		conn.mu.RLock()
		last := conn.LastActive
		conn.mu.RUnlock()
		if last.IsZero() || last.After(cutoff) {
			continue
		}
		delete(m.connections, id)
		removed++
	}
	m.connectionsMu.Unlock()

}
