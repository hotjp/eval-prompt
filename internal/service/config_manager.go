// Package service implements L4-Service layer: configuration management with change notification.
package service

import (
	"context"
	"sync"
)

// ConfigChangeHandler receives notifications when a configuration domain changes.
type ConfigChangeHandler func(ctx context.Context, domain string, changed []string)

// ConfigManager provides configuration change registration and notification mechanisms.
type ConfigManager interface {
	// Register registers a handler for a configuration domain.
	// When Notify is called for that domain, all registered handlers will be invoked.
	Register(domain string, h ConfigChangeHandler)

	// Notify notifies all registered handlers that configuration in the given domain has changed.
	// The changed parameter indicates which specific config keys were modified.
	Notify(ctx context.Context, domain string, changed []string)
}

// InMemoryConfigManager is a simple in-memory implementation of ConfigManager.
type InMemoryConfigManager struct {
	mu       sync.RWMutex
	handlers map[string][]ConfigChangeHandler
}

// NewInMemoryConfigManager creates a new InMemoryConfigManager.
func NewInMemoryConfigManager() *InMemoryConfigManager {
	return &InMemoryConfigManager{
		handlers: make(map[string][]ConfigChangeHandler),
	}
}

// Ensure InMemoryConfigManager implements ConfigManager.
var _ ConfigManager = (*InMemoryConfigManager)(nil)

// Register implements ConfigManager.
func (m *InMemoryConfigManager) Register(domain string, h ConfigChangeHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[domain] = append(m.handlers[domain], h)
}

// Notify implements ConfigManager.
func (m *InMemoryConfigManager) Notify(ctx context.Context, domain string, changed []string) {
	m.mu.RLock()
	handlers := m.handlers[domain]
	m.mu.RUnlock()

	for _, h := range handlers {
		h(ctx, domain, changed)
	}
}
