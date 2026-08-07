package mdgateway

// Health returns health status for all registered gateways.
func (m *Manager) Health() []AccountHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()
	now := Clk.Now().UnixMilli()
	var result []AccountHealth
	for _, gw := range m.gateways {
		lastAt := m.lastTickAt[gw.AccountID()]
		state := "healthy"
		if lastAt == 0 {
			state = "no_data"
		} else if now-lastAt > 15*60*1000 {
			state = "dead"
		} else if now-lastAt > 5*60*1000 {
			state = "stale"
		}
		result = append(result, AccountHealth{
			AccountID:  gw.AccountID(),
			Platform:   gw.Platform(),
			State:      state,
			LastTickAt: lastAt,
		})
	}
	return result
}

type AccountHealth struct {
	AccountID    string
	Broker       string
	Platform     string
	State        string
	LastTickAt   int64
	CircuitState string
	TickRate1m   float64
}
