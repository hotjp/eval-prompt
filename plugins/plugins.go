// Package plugins contains plugin implementations that follow dependency inversion.
// Interfaces are defined in internal/service, implementations live here.
// Plugins are optional - use noop implementations when not enabled.
package plugins

// Plugin is the base interface for all plugins.
type Plugin interface {
	// Initialize initializes the plugin.
	Initialize() error
	// Shutdown gracefully shuts down the plugin.
	Shutdown() error
	// Enabled returns whether the plugin is enabled.
	Enabled() bool
}

// NoopPlugin is a no-operation plugin implementation.
type NoopPlugin struct{}

// Initialize implements Plugin.
func (p *NoopPlugin) Initialize() error {
	return nil
}

// Shutdown implements Plugin.
func (p *NoopPlugin) Shutdown() error {
	return nil
}

// Enabled implements Plugin.
func (p *NoopPlugin) Enabled() bool {
	return false
}

// Ensure NoopPlugin implements Plugin interface.
var _ Plugin = (*NoopPlugin)(nil)
