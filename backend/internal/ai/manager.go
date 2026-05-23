package ai

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"anttrader/pkg/logger"
)

// Manager AI模型管理器，管理多个AI提供商实例
type Manager struct {
	providers map[ProviderType]AIProvider
	current   ProviderType
	mu        sync.RWMutex
}

// NewManager 创建新的AI管理器
func NewManager() *Manager {
	return &Manager{
		providers: make(map[ProviderType]AIProvider),
		current:   "",
	}
}

// RegisterProvider 注册AI提供商
func (m *Manager) RegisterProvider(providerType ProviderType, provider AIProvider) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.providers[providerType] = provider

	// 如果是第一个注册的提供商，自动设为当前提供商
	if m.current == "" {
		m.current = providerType
	}
}

// SetCurrentProvider 设置当前使用的提供商
func (m *Manager) SetCurrentProvider(providerType ProviderType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.providers[providerType]; !exists {
		return fmt.Errorf("%w: %s", ErrProviderNotFound, providerType)
	}

	m.current = providerType
	return nil
}

// GetCurrentProvider 获取当前提供商
func (m *Manager) GetCurrentProvider() (AIProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.current == "" {
		return nil, fmt.Errorf("no current provider set")
	}

	provider, exists := m.providers[m.current]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, m.current)
	}

	return provider, nil
}

// GetProvider 获取指定提供商
func (m *Manager) GetProvider(providerType ProviderType) (AIProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, exists := m.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, providerType)
	}

	return provider, nil
}

// Chat 使用当前提供商进行同步对话
func (m *Manager) Chat(ctx context.Context, messages []Message) (*Response, error) {
	provider, err := m.GetCurrentProvider()
	if err != nil {
		logger.Error("Failed to get current provider for chat",
			zap.Error(err))
		return nil, err
	}

	response, err := provider.Chat(ctx, messages)
	if err != nil {
		logger.Error("AI chat failed",
			zap.String("provider", provider.GetProviderName()),
			zap.Error(err))
		return nil, err
	}

	return response, nil
}

// StreamChat 使用当前提供商进行流式对话
func (m *Manager) StreamChat(ctx context.Context, messages []Message) (<-chan StreamChunk, error) {
	provider, err := m.GetCurrentProvider()
	if err != nil {
		logger.Error("Failed to get current provider for stream chat",
			zap.Error(err))
		return nil, err
	}

	return provider.StreamChat(ctx, messages)
}

// ValidateProviderConfig 验证提供商配置
func (m *Manager) ValidateProviderConfig(providerType ProviderType, apiKey string) error {
	m.mu.RLock()
	provider, exists := m.providers[providerType]
	m.mu.RUnlock()

	if !exists {
		logger.Error("Provider not found for config validation",
			zap.String("provider", providerType.String()))
		return fmt.Errorf("%w: %s", ErrProviderNotFound, providerType)
	}

	if err := provider.ValidateConfig(); err != nil {
		logger.Error("Provider config validation failed",
			zap.String("provider", providerType.String()),
			zap.Error(err))
		return err
	}

	return nil
}

// GetCurrentProviderType 获取当前提供商类型
func (m *Manager) GetCurrentProviderType() ProviderType {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// GetRegisteredProviders 获取所有已注册的提供商类型
func (m *Manager) GetRegisteredProviders() []ProviderType {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providers := make([]ProviderType, 0, len(m.providers))
	for p := range m.providers {
		providers = append(providers, p)
	}
	return providers
}

// HasProvider 检查是否已注册指定提供商
func (m *Manager) HasProvider(providerType ProviderType) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.providers[providerType]
	return exists
}

// UnregisterProvider 注销提供商
func (m *Manager) UnregisterProvider(providerType ProviderType) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.providers[providerType]; exists {
		delete(m.providers, providerType)

		// 如果注销的是当前提供商，需要重置
		if m.current == providerType {
			m.current = ""
			logger.Warn("Current provider unregistered, no current provider set",
				zap.String("provider", providerType.String()))
		}
	}
}

// GetProviderInfo 获取提供商信息
func (m *Manager) GetProviderInfo(providerType ProviderType) (map[string]interface{}, error) {
	provider, err := m.GetProvider(providerType)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"provider": provider.GetProviderName(),
		"model":    provider.GetModelName(),
		"type":     providerType.String(),
	}, nil
}

// GetAllProvidersInfo 获取所有提供商信息
func (m *Manager) GetAllProvidersInfo() []map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]map[string]interface{}, 0, len(m.providers))
	for providerType, provider := range m.providers {
		infos = append(infos, map[string]interface{}{
			"provider": provider.GetProviderName(),
			"model":    provider.GetModelName(),
			"type":     providerType.String(),
			"current":  providerType == m.current,
		})
	}
	return infos
}

// RegisterProviderWithConfig 使用配置注册AI提供商
func (m *Manager) RegisterProviderWithConfig(providerType ProviderType, provider AIProvider) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.providers[providerType] = provider

	if m.current == "" {
		m.current = providerType
	}

	

	return nil
}
