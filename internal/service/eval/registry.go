package eval

import (
	"fmt"
	"sync"
)

// Registry maintains a global registry of EvalPlugins.
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]EvalPlugin
}

var globalRegistry = &Registry{
	plugins: make(map[string]EvalPlugin),
}

// Register registers an EvalPlugin.
func Register(plugin EvalPlugin) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.plugins[plugin.Name()] = plugin
}

// Get retrieves a registered plugin by name.
func Get(name string) (EvalPlugin, error) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	plugin, ok := globalRegistry.plugins[name]
	if !ok {
		return nil, fmt.Errorf("eval: plugin not found: %s", name)
	}
	return plugin, nil
}

// List returns all registered plugin names.
func List() []string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	names := make([]string, 0, len(globalRegistry.plugins))
	for name := range globalRegistry.plugins {
		names = append(names, name)
	}
	return names
}

// MustGet retrieves a plugin by name, panics if not found.
func MustGet(name string) EvalPlugin {
	plugin, err := Get(name)
	if err != nil {
		panic(err)
	}
	return plugin
}
